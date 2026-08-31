package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	ytInitialPlayerRe = regexp.MustCompile(`(?s)window\s*\[\s*"ytInitialPlayerResponse"\s*\]\s*=\s*(\{.*?\});|ytInitialPlayerResponse\s*=\s*(\{.*?\});`)
	jsonLDRe          = regexp.MustCompile(`(?is)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)
)

type playerMicroformatText struct {
	SimpleText string `json:"simpleText"`
}

type playerMicroformatRenderer struct {
	Title            *playerMicroformatText `json:"title"`
	Description      *playerMicroformatText `json:"description"`
	OwnerChannelName string                 `json:"ownerChannelName"`
	PublishDate      string                 `json:"publishDate"`
	ViewCount        string                 `json:"viewCount"`
	LengthSeconds    string                 `json:"lengthSeconds"`
	OwnerProfileURL  string                 `json:"ownerProfileUrl"`
}

type videoDetails struct {
	Title            string          `json:"title"`
	ShortDescription string          `json:"shortDescription"`
	Author           string          `json:"author"`
	ViewCount        string          `json:"viewCount"`
	LengthSeconds    string          `json:"lengthSeconds"`
	ChannelID        string          `json:"channelId"`
	Keywords         json.RawMessage `json:"keywords"`
}

// playerResponse mirrors the fields the upstream TS implementation reads
// off the parsed ytInitialPlayerResponse object. Note it reads
// playerMicroformatRenderer from the top level rather than nested under
// "microformat" (matching the source project's actual field access), so in
// practice this tier rarely populates and videoDetails + <meta> tags do the
// real work below.
type playerResponse struct {
	PlayerMicroformatRenderer *playerMicroformatRenderer `json:"playerMicroformatRenderer"`
	VideoDetails              *videoDetails              `json:"videoDetails"`
}

func keywordsToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return strings.Join(list, ", ")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

// FetchVideoMetadata scrapes the YouTube watch page for title, channel,
// description, publish date, view count, and duration, without using any
// official API. It layers three extraction strategies, most to least
// reliable: the embedded ytInitialPlayerResponse JSON, JSON-LD
// <script type="application/ld+json"> blocks, and finally individual <meta>
// tags — each only filling in fields the previous tier left empty.
func FetchVideoMetadata(ctx context.Context, videoID string) (map[string]string, error) {
	meta, _, err := fetchVideoMetadataAndChapters(ctx, videoID)
	return meta, err
}

// fetchVideoMetadataAndChapters does the actual watch-page fetch, shared by
// FetchVideoMetadata and FetchChapters so a get_chapters call only needs one
// HTTP request, not two.
func fetchVideoMetadataAndChapters(ctx context.Context, videoID string) (map[string]string, []chapter, error) {
	pageURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YoutubeMCP/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("YouTube responded with %d %s", resp.StatusCode, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	html := string(body)

	meta := map[string]string{}
	var prChapters []chapter

	if m := ytInitialPlayerRe.FindStringSubmatch(html); m != nil {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		prChapters = chaptersFromPlayerResponseJSON([]byte(raw))
		var pr playerResponse
		if err := json.Unmarshal([]byte(raw), &pr); err == nil {
			if mf := pr.PlayerMicroformatRenderer; mf != nil {
				if mf.Title != nil && mf.Title.SimpleText != "" {
					meta["title"] = mf.Title.SimpleText
				}
				if mf.Description != nil && mf.Description.SimpleText != "" {
					meta["description"] = mf.Description.SimpleText
				}
				if mf.OwnerChannelName != "" {
					meta["channel"] = mf.OwnerChannelName
				}
				if mf.PublishDate != "" {
					meta["publishDate"] = mf.PublishDate
				}
				if mf.ViewCount != "" {
					meta["viewCount"] = mf.ViewCount
				}
				if mf.LengthSeconds != "" {
					if secs := atoiSafe(mf.LengthSeconds); secs > 0 {
						meta["duration"] = FormatDuration(float64(secs))
					}
				}
				if mf.OwnerProfileURL != "" {
					meta["channelUrl"] = mf.OwnerProfileURL
				}
			}
			if vd := pr.VideoDetails; vd != nil {
				setIfEmpty(meta, "title", vd.Title)
				setIfEmpty(meta, "description", vd.ShortDescription)
				setIfEmpty(meta, "channel", vd.Author)
				setIfEmpty(meta, "viewCount", vd.ViewCount)
				if meta["duration"] == "" && vd.LengthSeconds != "" {
					if secs := atoiSafe(vd.LengthSeconds); secs > 0 {
						meta["duration"] = FormatDuration(float64(secs))
					}
				}
				if vd.ChannelID != "" {
					meta["channelId"] = vd.ChannelID
				}
				if kw := keywordsToString(vd.Keywords); kw != "" {
					meta["keywords"] = kw
				}
			}
		}
	}

	if meta["title"] == "" || meta["description"] == "" || meta["channel"] == "" ||
		meta["publishDate"] == "" || meta["viewCount"] == "" || meta["duration"] == "" {
		fillFromJSONLD(html, meta)
	}

	if meta["title"] == "" {
		meta["title"] = firstNonEmpty(extractMetaTag(html, "property", "og:title"), extractMetaTag(html, "name", "twitter:title"), videoID)
	}
	if meta["description"] == "" {
		meta["description"] = firstNonEmpty(extractMetaTag(html, "property", "og:description"), extractMetaTag(html, "name", "twitter:description"))
	}
	if meta["channel"] == "" {
		meta["channel"] = firstNonEmpty(extractMetaTag(html, "itemprop", "author"), extractMetaTag(html, "name", "author"))
	}
	if meta["publishDate"] == "" {
		meta["publishDate"] = extractMetaTag(html, "itemprop", "datePublished")
	}
	if meta["viewCount"] == "" {
		meta["viewCount"] = extractMetaTag(html, "itemprop", "interactionCount")
	}
	if meta["duration"] == "" {
		if durStr := extractMetaTag(html, "itemprop", "duration"); durStr != "" {
			if secs := parseISODuration(durStr); secs > 0 {
				meta["duration"] = FormatDuration(float64(secs))
			}
		}
	}

	return meta, prChapters, nil
}

func setIfEmpty(meta map[string]string, key, value string) {
	if meta[key] == "" && value != "" {
		meta[key] = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// fillFromJSONLD scans <script type="application/ld+json"> blocks for a
// VideoObject/Video entry (either at the top level or nested under
// "@graph") and fills in any metadata fields still missing.
func fillFromJSONLD(html string, meta map[string]string) {
	matches := jsonLDRe.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		var data map[string]any
		if err := json.Unmarshal([]byte(m[1]), &data); err != nil {
			continue
		}

		var items []map[string]any
		if graph, ok := data["@graph"].([]any); ok {
			for _, g := range graph {
				if item, ok := g.(map[string]any); ok {
					items = append(items, item)
				}
			}
		} else {
			items = append(items, data)
		}

		for _, item := range items {
			itemType, _ := item["@type"].(string)
			if itemType != "VideoObject" && itemType != "Video" {
				continue
			}

			if meta["title"] == "" {
				if name, ok := item["name"].(string); ok {
					meta["title"] = name
				}
			}
			if meta["description"] == "" {
				if desc, ok := item["description"].(string); ok {
					meta["description"] = desc
				}
			}
			if meta["channel"] == "" {
				switch author := item["author"].(type) {
				case string:
					meta["channel"] = author
				case map[string]any:
					if name, ok := author["name"].(string); ok {
						meta["channel"] = name
					}
				}
			}
			if meta["publishDate"] == "" {
				if v, ok := item["uploadDate"].(string); ok && v != "" {
					meta["publishDate"] = v
				} else if v, ok := item["datePublished"].(string); ok {
					meta["publishDate"] = v
				}
			}
			if stat, ok := item["interactionStatistic"]; ok {
				var stats []map[string]any
				switch s := stat.(type) {
				case []any:
					for _, e := range s {
						if em, ok := e.(map[string]any); ok {
							stats = append(stats, em)
						}
					}
				case map[string]any:
					stats = append(stats, s)
				}
				for _, s := range stats {
					if meta["viewCount"] != "" {
						break
					}
					if it, ok := s["interactionType"].(string); ok && strings.Contains(it, "WatchAction") {
						if count, ok := s["userInteractionCount"]; ok {
							meta["viewCount"] = fmt.Sprintf("%v", count)
						}
					}
				}
			}
			if meta["duration"] == "" {
				if d, ok := item["duration"].(string); ok && d != "" {
					if secs := parseISODuration(d); secs > 0 {
						meta["duration"] = FormatDuration(float64(secs))
					} else {
						meta["duration"] = d
					}
				}
			}
		}
	}
}

func extractMetaTag(html, attr, value string) string {
	re1 := regexp.MustCompile(fmt.Sprintf(`(?i)<meta\s+%s=["']%s["']\s+content=["']([^"']*)["']`, regexp.QuoteMeta(attr), regexp.QuoteMeta(value)))
	if m := re1.FindStringSubmatch(html); m != nil {
		return m[1]
	}
	re2 := regexp.MustCompile(fmt.Sprintf(`(?i)<meta\s+content=["']([^"']*)["']\s+%s=["']%s["']`, regexp.QuoteMeta(attr), regexp.QuoteMeta(value)))
	if m := re2.FindStringSubmatch(html); m != nil {
		return m[1]
	}
	return ""
}
