package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type transcriptSegment struct {
	Text     string
	Offset   float64 // milliseconds
	Duration float64
}

var (
	vttTimingRe  = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}\.\d{3})`)
	vttTagRe     = regexp.MustCompile(`<[^>]+>`)
	blankLineRe  = regexp.MustCompile(`\r?\n\r?\n`)
	newlineRe    = regexp.MustCompile(`\r?\n`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func parseVttTime(ts string) float64 {
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.ParseFloat(parts[2], 64)
	return (float64(h)*3600 + float64(m)*60 + s) * 1000
}

var vttEntityReplacer = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&nbsp;", " ",
)

func parseVtt(content string) []transcriptSegment {
	var segments []transcriptSegment
	seen := map[string]struct{}{}

	blocks := blankLineRe.Split(content, -1)
	for _, block := range blocks {
		lines := newlineRe.Split(block, -1)

		timingIdx := -1
		for i, l := range lines {
			if strings.Contains(l, " --> ") {
				timingIdx = i
				break
			}
		}
		if timingIdx == -1 {
			continue
		}

		m := vttTimingRe.FindStringSubmatch(lines[timingIdx])
		if m == nil {
			continue
		}
		offset := parseVttTime(m[1])
		end := parseVttTime(m[2])

		text := strings.Join(lines[timingIdx+1:], " ")
		text = vttTagRe.ReplaceAllString(text, "")
		text = vttEntityReplacer.Replace(text)
		text = whitespaceRe.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}
		key := strconv.FormatFloat(offset, 'f', -1, 64) + "|" + text
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		segments = append(segments, transcriptSegment{Text: text, Offset: offset, Duration: end - offset})
	}

	return segments
}

// ── Presentation (pure) ──────────────────────────────────────────────────

func transcriptText(segments []transcriptSegment) string {
	texts := make([]string, len(segments))
	for i, s := range segments {
		texts[i] = s.Text
	}
	return strings.Join(texts, " ")
}

func transcriptTimed(segments []transcriptSegment) string {
	lines := make([]string, len(segments))
	for i, s := range segments {
		lines[i] = fmt.Sprintf("[%s] %s", FormatTimestamp(s.Offset/1000), s.Text)
	}
	return strings.Join(lines, "\n")
}

// filterSegmentsByRange returns segments whose Offset (ms) falls within
// [startMs, endMs], inclusive of both bounds. A nil bound is open-ended on
// that side.
func filterSegmentsByRange(segments []transcriptSegment, startMs, endMs *float64) []transcriptSegment {
	var out []transcriptSegment
	for _, s := range segments {
		if startMs != nil && s.Offset < *startMs {
			continue
		}
		if endMs != nil && s.Offset > *endMs {
			continue
		}
		out = append(out, s)
	}
	return out
}

// TranscriptErrorText turns a raw fetch error into a user-facing message,
// classifying (timeout / missing captions / network error / generic).
//
// The network-error case has two distinct sources reachable from this
// function's callers (see docs/BUGS.md BUG-003), each classified
// differently because only one preserves a real Go error chain:
//   - EnsureYtDlp's own network failure (internal/core/ytdlp.go) is
//     %w-wrapped down to go-ytdlp's HTTP client error, which is ultimately
//     a *net.DNSError/*net.OpError — both satisfy net.Error, so it's
//     classified by type, correct regardless of OS-specific error text.
//   - yt-dlp's own process failure (fetchSegments below) only survives as
//     a flattened string ("yt-dlp failed: <stderr>"), so it's classified
//     by matching yt-dlp's own real, OS-independent wrapper phrasing.
//     yt-dlp uses different wrapper phrasing depending on which stage of
//     its pipeline hits the network failure, so two real phrasings are
//     matched, both captured directly from real yt-dlp runs (not guessed):
//   - "Unable to download webpage" — captured by running yt-dlp directly
//     against a URL on the RFC 2606 reserved ".invalid" TLD (guaranteed
//     non-resolving, no real internet access needed):
//     ERROR: [generic] Unable to download webpage: HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed) ...
//   - "Unable to download API page" — captured by running the CLI against
//     a real YouTube URL with HTTPS_PROXY/HTTP_PROXY pointed at an
//     unreachable local port (deterministic connection-refused, no DNS
//     dependency), which fails at YouTube's API-page fetch stage instead:
//     ERROR: [youtube] <id>: Unable to download API page: ('Unable to connect to proxy', NewConnectionError(...)) (caused by ProxyError(...))
//     The previous "ENOTFOUND"/"ECONNREFUSED" substrings (Node.js/libuv
//     error codes ported from the upstream TS project) never matched any
//     real source and have been removed.
func TranscriptErrorText(videoID string, err error) string {
	message := err.Error()
	var netErr net.Error
	switch {
	case strings.Contains(message, "timed out"):
		return fmt.Sprintf("Transcript fetch timed out for video %s. Please try again.", videoID)
	case strings.Contains(message, "No transcript available") || strings.Contains(message, "captions"):
		return fmt.Sprintf("No transcript available for video %s. The video may not have captions.", videoID)
	case errors.As(err, &netErr):
		return fmt.Sprintf("Network error while fetching transcript for video %s. Please check your internet connection.", videoID)
	case strings.Contains(message, "Unable to download webpage") || strings.Contains(message, "Unable to download API page"):
		return fmt.Sprintf("Network error while fetching transcript for video %s. Please check your internet connection.", videoID)
	default:
		return fmt.Sprintf("Failed to fetch transcript for video %s: %s", videoID, message)
	}
}

// pickVttFile chooses which .vtt file to parse out of a directory listing.
// It prefers an exact match on "<language>.vtt" and falls back to the first
// .vtt file found if there's no exact match, returning "" if there are no
// .vtt files at all. See docs/BUGS.md BUG-001: the upstream TS
// implementation has no equivalent — it just takes the first .vtt file
// unconditionally, which can silently pick an auto-translated variant.
func pickVttFile(files []string, language string) string {
	exactSuffix := "." + language + ".vtt"
	for _, f := range files {
		if strings.HasSuffix(f, exactSuffix) {
			return f
		}
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".vtt") {
			return f
		}
	}
	return ""
}

// ── Fetching (I/O) ────────────────────────────────────────────────────────

// fetchSegments returns the parsed transcript for videoID+language, serving
// a cached result when available (see transcache.go) instead of re-shelling
// out to yt-dlp on every call.
func fetchSegments(ctx context.Context, videoID, language string) ([]transcriptSegment, error) {
	key := cacheKey{videoID: videoID, language: language}
	return defaultCache.getOrFetch(key, func() ([]transcriptSegment, error) {
		return fetchSegmentsFromYtDlp(ctx, videoID, language)
	})
}

func fetchSegmentsFromYtDlp(ctx context.Context, videoID, language string) ([]transcriptSegment, error) {
	if err := EnsureYtDlp(ctx); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("ytmcp-%s-*", videoID))
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// SubLangs uses an exact language match, not a "language.*" wildcard
	// (which the upstream TS uses) — see docs/BUGS.md BUG-001. The wildcard
	// also matches YouTube's auto-translated caption variants (e.g.
	// "en-de-DE" = English translated from German), which can both trigger
	// 429s from requesting many variants at once and produce the wrong
	// transcript if a translated file gets picked over the real one.
	outputTemplate := filepath.Join(tmpDir, "sub")
	cmd := NewYtDlpCommand().
		SkipDownload().
		WriteAutoSubs().
		WriteSubs().
		SubLangs(language).
		SubFormat("vtt").
		Output(outputTemplate).
		NoWarnings().
		Quiet()

	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	result, err := cmd.Run(runCtx, videoURL)
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("transcript fetch timed out")
		}
		stderr := ""
		if result != nil {
			stderr = result.Stderr
		}
		return nil, fmt.Errorf("yt-dlp failed: %s", firstNonEmpty(stderr, err.Error()))
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	picked := pickVttFile(names, language)
	if picked == "" {
		return nil, fmt.Errorf("no transcript available for video %s. The video may not have captions in language %q", videoID, language)
	}
	vttPath := filepath.Join(tmpDir, picked)

	content, err := os.ReadFile(vttPath)
	if err != nil {
		return nil, err
	}
	segments := parseVtt(string(content))
	if len(segments) == 0 {
		return nil, fmt.Errorf("no transcript found for video %s", videoID)
	}
	return segments, nil
}

func normalizeLanguage(language string) string {
	if language == "" {
		return "en"
	}
	return language
}

// GetTranscriptText fetches the plain (untimed) transcript text for a video.
func GetTranscriptText(ctx context.Context, videoID, language string) (string, error) {
	segments, err := fetchSegments(ctx, videoID, normalizeLanguage(language))
	if err != nil {
		return "", err
	}
	return transcriptText(segments), nil
}

// GetTranscriptTimed fetches the transcript with a [MM:SS] prefix per line.
func GetTranscriptTimed(ctx context.Context, videoID, language string) (string, error) {
	segments, err := fetchSegments(ctx, videoID, normalizeLanguage(language))
	if err != nil {
		return "", err
	}
	return transcriptTimed(segments), nil
}

// secToMsPtr converts a *float64 seconds bound to a *float64 milliseconds
// bound, preserving nil (open-ended).
func secToMsPtr(sec *float64) *float64 {
	if sec == nil {
		return nil
	}
	ms := *sec * 1000
	return &ms
}

// GetTranscriptRange fetches the transcript and returns only the segments
// whose offset falls within [startSec, endSec] (a nil bound is
// open-ended), formatted as timed lines. An out-of-bounds or otherwise
// empty window returns an empty string, not an error.
func GetTranscriptRange(ctx context.Context, videoID, language string, startSec, endSec *float64) (string, error) {
	segments, err := fetchSegments(ctx, videoID, normalizeLanguage(language))
	if err != nil {
		return "", err
	}
	filtered := filterSegmentsByRange(segments, secToMsPtr(startSec), secToMsPtr(endSec))
	return transcriptTimed(filtered), nil
}

// SearchInTranscript searches for a keyword/phrase in the transcript and
// returns matching lines with timestamps, or a "no matches" message.
// SearchInTranscript searches for a keyword or phrase in the transcript
// (matching across segment boundaries) and returns each match's
// ±contextSecs window as a block of timed lines, or a "no matches"
// message. contextSecs < 0 is clamped to 0 (matched segments only).
func SearchInTranscript(ctx context.Context, videoID, query, language string, contextSecs float64) (string, error) {
	if contextSecs < 0 {
		contextSecs = 0
	}
	segments, err := fetchSegments(ctx, videoID, normalizeLanguage(language))
	if err != nil {
		return "", err
	}
	matchCount, blocks := searchSegmentsWithContext(segments, query, contextSecs)
	return formatSearchResultWithContext(videoID, query, matchCount, blocks), nil
}

// SaveTranscriptFile fetches the transcript and metadata concurrently and
// writes a Markdown file (with a metadata header) into outputDir, returning
// the written file's path.
func SaveTranscriptFile(ctx context.Context, videoID, language, outputDir string, withTimestamps bool) (string, error) {
	language = normalizeLanguage(language)

	var (
		segments []transcriptSegment
		segErr   error
		meta     = map[string]string{}
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		segments, segErr = fetchSegments(ctx, videoID, language)
	}()
	go func() {
		defer wg.Done()
		if m, err := FetchVideoMetadata(ctx, videoID); err == nil {
			meta = m
		}
	}()
	wg.Wait()

	if segErr != nil {
		return "", segErr
	}

	titleFromMeta := meta["title"]
	if titleFromMeta == "" {
		titleFromMeta = videoID
	}
	safeTitle := SanitizeTitle(titleFromMeta)

	var lines []string
	for _, s := range segments {
		if withTimestamps {
			lines = append(lines, fmt.Sprintf("[%s] %s", FormatTimestamp(s.Offset/1000), s.Text))
		} else {
			lines = append(lines, s.Text)
		}
	}

	var metaLines []string
	if meta["channel"] != "" {
		metaLines = append(metaLines, "**Channel:** "+meta["channel"])
	}
	if meta["publishDate"] != "" {
		metaLines = append(metaLines, "**Published:** "+meta["publishDate"])
	}
	if meta["viewCount"] != "" {
		metaLines = append(metaLines, "**Views:** "+meta["viewCount"])
	}
	if meta["duration"] != "" {
		metaLines = append(metaLines, "**Duration:** "+meta["duration"])
	}
	metaLines = append(metaLines, "**Video ID:** "+videoID)
	metaLines = append(metaLines, fmt.Sprintf("**URL:** https://www.youtube.com/watch?v=%s", videoID))

	sep := " "
	if withTimestamps {
		sep = "\n"
	}
	md := fmt.Sprintf("# Transcript - %s\n\n%s\n\n---\n\n%s\n", titleFromMeta, strings.Join(metaLines, "\n"), strings.Join(lines, sep))

	filename := safeTitle
	if withTimestamps {
		filename += "_timed"
	}
	filename += ".md"

	fp := filepath.Join(outputDir, filename)
	if err := os.WriteFile(fp, []byte(md), 0o644); err != nil {
		return "", err
	}
	return fp, nil
}
