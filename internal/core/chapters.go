package core

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// chapter is one entry in a video's chapter list (its native table of
// contents), new in task 14 with no upstream TS equivalent.
type chapter struct {
	Title     string
	StartSecs float64
}

var (
	chapterTimestampRe = regexp.MustCompile(`\b\d{1,2}(?::[0-5]\d){1,2}\b`)
	hasLetterOrDigitRe = regexp.MustCompile(`[\p{L}\p{N}]`)
)

// parseChapters extracts a YouTube-style chapter list from a video
// description. It recognizes lines containing a timestamp (M:SS, MM:SS,
// H:MM:SS) with real title text on exactly one side of it — in either
// "timestamp then title" or "title then timestamp" order, tolerating
// decorative wrapping like bullets, emoji, or brackets around the
// timestamp (e.g. "⌨️ (0:00) Introduction") — and applies YouTube's
// published validity rules: chapters must start at 0:00, there must be at
// least 3, and timestamps must be strictly ascending. If any rule fails,
// returns nil — never a partial guess (docs/tasks/14-chapters/TASK.md,
// 14.2).
func parseChapters(description string) []chapter {
	var candidates []chapter

	for _, line := range strings.Split(description, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		loc := chapterTimestampRe.FindStringIndex(trimmed)
		if loc == nil {
			continue
		}
		token := trimmed[loc[0]:loc[1]]
		prefix, suffix := trimmed[:loc[0]], trimmed[loc[1]:]
		prefixHasTitle := hasLetterOrDigitRe.MatchString(prefix)
		suffixHasTitle := hasLetterOrDigitRe.MatchString(suffix)

		var titlePart string
		switch {
		case !prefixHasTitle && suffixHasTitle:
			titlePart = suffix // timestamp then title
		case prefixHasTitle && !suffixHasTitle:
			titlePart = prefix // title then timestamp
		default:
			// Both sides carry text (mid-sentence mention, e.g.
			// "at 2:30 he says...") or neither does (bare timestamp
			// line) — not a chapter marker, skip.
			continue
		}

		secs, err := ParseTimestamp(token)
		if err != nil {
			continue
		}

		title := trimTitle(titlePart)
		if title == "" {
			continue
		}

		candidates = append(candidates, chapter{Title: title, StartSecs: secs})
	}

	if !validChapters(candidates) {
		return nil
	}
	return candidates
}

func validChapters(chs []chapter) bool {
	if len(chs) < 3 {
		return false
	}
	if chs[0].StartSecs != 0 {
		return false
	}
	for i := 1; i < len(chs); i++ {
		if chs[i].StartSecs <= chs[i-1].StartSecs {
			return false
		}
	}
	return true
}

func trimTitle(s string) string {
	return strings.Trim(strings.TrimSpace(s), " \t-–—:()[]")
}

// chapterRenderer mirrors the shape of one entry in ytInitialPlayerResponse's
// "chapters" array.
type chapterRenderer struct {
	Title struct {
		SimpleText string `json:"simpleText"`
	} `json:"title"`
	TimeRangeStartMillis int64 `json:"timeRangeStartMillis"`
}

type playerResponseChapters struct {
	Chapters []chapterRenderer `json:"chapters"`
}

// chaptersFromPlayerResponseJSON decodes the raw ytInitialPlayerResponse
// JSON's "chapters" array, if present. This is YouTube's own structured
// data, so unlike parseChapters it does not re-apply the free-text
// validity gate — it's a best-effort tier: any decode error or absent
// field yields nil, never a panic or an error
// (docs/tasks/14-chapters/TASK.md, 14.4).
func chaptersFromPlayerResponseJSON(raw []byte) []chapter {
	var prc playerResponseChapters
	if err := json.Unmarshal(raw, &prc); err != nil || len(prc.Chapters) == 0 {
		return nil
	}

	chs := make([]chapter, 0, len(prc.Chapters))
	for _, cr := range prc.Chapters {
		title := trimTitle(cr.Title.SimpleText)
		if title == "" {
			continue
		}
		chs = append(chs, chapter{
			Title:     title,
			StartSecs: float64(cr.TimeRangeStartMillis) / 1000,
		})
	}
	if len(chs) == 0 {
		return nil
	}
	return chs
}

// FetchChapters returns the video's chapters, first by parsing the
// description (parseChapters), falling back to the chapters array embedded
// in ytInitialPlayerResponse only if the description tier yields nothing
// (docs/tasks/14-chapters/TASK.md, 14.4). A video with genuinely no
// chapters yields a nil slice and a nil error — not an error.
func FetchChapters(ctx context.Context, videoID string) ([]chapter, error) {
	meta, prChapters, err := fetchVideoMetadataAndChapters(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if chs := parseChapters(meta["description"]); chs != nil {
		return chs, nil
	}
	return prChapters, nil
}
