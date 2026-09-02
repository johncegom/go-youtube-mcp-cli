package core

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// playlistIDRe matches real YouTube playlist ID shapes: user-created
// playlists ("PL..."), a channel's uploads playlist ("UU..."),
// auto-generated album playlists ("OLAK5uy_..."), and YouTube-generated
// mix/favorites/liked-videos playlists ("RD...", "FL...", "LL...").
var playlistIDRe = regexp.MustCompile(`^(PL|UU|OL|RD|FL|LL)[A-Za-z0-9_-]{8,}$`)

// ExtractPlaylistID pulls a playlist ID out of a bare ID, a
// youtube.com/playlist?list=... URL, or a youtube.com/watch?...&list=...
// URL. Unlike ExtractVideoID, the playlist ID always lives in the query
// string in every supported URL shape, so there's no path-segment case to
// handle.
func ExtractPlaylistID(input string) string {
	if playlistIDRe.MatchString(input) {
		return input
	}

	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	host := strings.ToLower(u.Hostname())
	if !strings.Contains(host, "youtube.com") {
		return ""
	}

	id := u.Query().Get("list")
	if id != "" && playlistIDRe.MatchString(id) {
		return id
	}
	return ""
}

// playlistEntry is one video entry from a flat-playlist listing.
type playlistEntry struct {
	VideoID string
	Title   string
}

// parseFlatPlaylistOutput parses the stdout of
// `yt-dlp --flat-playlist --print "%(id)s\t%(title)s"`: one "id\ttitle"
// line per video, in playlist order. Lines missing a tab or with an empty
// id are skipped rather than treated as fatal, since a single malformed
// entry shouldn't take down the whole listing.
func parseFlatPlaylistOutput(stdout string) []playlistEntry {
	var entries []playlistEntry
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		id, title := parts[0], parts[1]
		if id == "" {
			continue
		}
		entries = append(entries, playlistEntry{VideoID: id, Title: title})
	}
	return entries
}

// maxPlaylistEntries caps how many videos ListPlaylistVideos and
// SearchPlaylist will process from a single playlist, bounding both
// runtime and 429 exposure from repeatedly hitting YouTube's caption
// endpoint (see docs/BUGS.md BUG-001).
const maxPlaylistEntries = 25

// ListPlaylistVideos fetches a playlist's video listing via yt-dlp's
// flat-playlist mode (no per-video metadata fetch, just id+title), capped
// to the first maxPlaylistEntries entries. Returns the capped entries and
// the total entry count found, so callers can note "showing first N of
// total".
func ListPlaylistVideos(ctx context.Context, playlistID string) ([]playlistEntry, int, error) {
	if err := EnsureYtDlp(ctx); err != nil {
		return nil, 0, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := NewYtDlpCommand().
		FlatPlaylist().
		Print("%(id)s\t%(title)s").
		NoWarnings().
		Quiet()

	playlistURL := fmt.Sprintf("https://www.youtube.com/playlist?list=%s", playlistID)
	result, err := cmd.Run(runCtx, playlistURL)
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, 0, fmt.Errorf("playlist listing timed out")
		}
		stderr := ""
		if result != nil {
			stderr = result.Stderr
		}
		return nil, 0, fmt.Errorf("yt-dlp failed: %s", firstNonEmpty(stderr, err.Error()))
	}

	entries := parseFlatPlaylistOutput(result.Stdout)
	total := len(entries)
	if total > maxPlaylistEntries {
		entries = entries[:maxPlaylistEntries]
	}
	return entries, total, nil
}

// videoSearchResult is one video's search matches within a playlist search.
type videoSearchResult struct {
	VideoID    string
	Title      string
	MatchCount int
	Blocks     [][]transcriptSegment
}

// playlistSearchContextSecs is the fixed context window used by
// SearchPlaylist for each match — no `context` param is exposed at the
// playlist-search level (see docs/tasks/15-playlist-search/TASK.md).
const playlistSearchContextSecs = 15.0

// playlistSearchDelay is the fixed pause between cache-miss transcript
// fetches while searching a playlist — deliberate 429 protection (see
// docs/BUGS.md BUG-001); the transcript cache (transcache.go) absorbs
// repeat calls, so a re-run of the same search over the same playlist
// incurs no delay at all.
const playlistSearchDelay = 500 * time.Millisecond

// transcriptCacheHas reports whether a fresh, unexpired transcript is
// already cached for videoID+language, without fetching or mutating
// anything.
func transcriptCacheHas(videoID, language string) bool {
	key := cacheKey{videoID: videoID, language: language}
	defaultCache.mu.Lock()
	defer defaultCache.mu.Unlock()
	entry, ok := defaultCache.entries[key]
	return ok && defaultCache.now().Before(entry.expires)
}

// fetchSegmentsReportingMiss wraps fetchSegments, additionally reporting
// whether the call was a cache miss (i.e. actually shelled out to
// yt-dlp) so the caller can pace inter-video delays accordingly.
func fetchSegmentsReportingMiss(ctx context.Context, videoID, language string) (segments []transcriptSegment, wasMiss bool, err error) {
	hit := transcriptCacheHas(videoID, language)
	segments, err = fetchSegments(ctx, videoID, language)
	return segments, !hit, err
}

// searchPlaylistEntries runs a search query against each entry's
// transcript sequentially via fetch, pausing (via sleep) before the next
// fetch whenever the previous one was a cache miss. A per-video fetch
// error is recorded in skipped (with a TranscriptErrorText-classified
// reason) rather than aborting the loop. fetch/sleep are injected so
// tests can exercise ordering, partial-failure tolerance, and delay
// pacing without any real I/O or timing.
func searchPlaylistEntries(
	ctx context.Context,
	entries []playlistEntry,
	query, language string,
	fetch func(ctx context.Context, videoID, language string) (segments []transcriptSegment, wasMiss bool, err error),
	sleep func(time.Duration),
) (results []videoSearchResult, skipped []string) {
	prevWasMiss := false
	for i, e := range entries {
		if i > 0 && prevWasMiss {
			sleep(playlistSearchDelay)
		}

		segments, wasMiss, err := fetch(ctx, e.VideoID, language)
		prevWasMiss = wasMiss
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%s): %s", e.Title, e.VideoID, TranscriptErrorText(e.VideoID, err)))
			continue
		}

		matchCount, blocks := searchSegmentsWithContext(segments, query, playlistSearchContextSecs)
		if matchCount > 0 {
			results = append(results, videoSearchResult{
				VideoID:    e.VideoID,
				Title:      e.Title,
				MatchCount: matchCount,
				Blocks:     blocks,
			})
		}
	}
	return results, skipped
}

// formatPlaylistSearchResult renders a cross-video search result: a
// match-count header, each result's matches as "<title> [MM:SS] <text>"
// lines (blocks within one video separated by "---", matching the
// single-video search convention), a trailing "skipped" section listing
// per-video failures, and a "showing first N of total" note when the
// playlist was capped.
func formatPlaylistSearchResult(query string, results []videoSearchResult, skipped []string, shown, total int) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}

	var b strings.Builder
	if totalMatches == 0 {
		fmt.Fprintf(&b, "No matches found for %q in this playlist.", query)
	} else {
		fmt.Fprintf(&b, "Found %d match(es) for %q across %d video(s):\n", totalMatches, query, len(results))
		for _, r := range results {
			b.WriteString("\n")
			for bi, block := range r.Blocks {
				if bi > 0 {
					b.WriteString("---\n")
				}
				for _, s := range block {
					fmt.Fprintf(&b, "%s [%s] %s\n", r.Title, FormatTimestamp(s.Offset/1000), s.Text)
				}
			}
		}
	}

	if shown < total {
		fmt.Fprintf(&b, "\n(showing first %d of %d videos)", shown, total)
	}

	if len(skipped) > 0 {
		fmt.Fprintf(&b, "\n\nSkipped %d video(s):\n", len(skipped))
		for _, s := range skipped {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// SearchPlaylist lists playlistID's videos (capped to maxPlaylistEntries)
// and searches each one's transcript for query, sequentially and
// cache-backed (see searchPlaylistEntries), returning the aggregated,
// formatted result.
func SearchPlaylist(ctx context.Context, playlistID, query, language string) (string, error) {
	language = normalizeLanguage(language)

	entries, total, err := ListPlaylistVideos(ctx, playlistID)
	if err != nil {
		return "", err
	}

	results, skipped := searchPlaylistEntries(ctx, entries, query, language, fetchSegmentsReportingMiss, time.Sleep)
	return formatPlaylistSearchResult(query, results, skipped, len(entries), total), nil
}
