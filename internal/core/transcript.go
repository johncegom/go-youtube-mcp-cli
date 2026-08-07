package core

import (
	"context"
	"fmt"
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

func searchSegments(segments []transcriptSegment, query string) []transcriptSegment {
	lowerQuery := strings.ToLower(query)
	var matches []transcriptSegment
	for _, s := range segments {
		if strings.Contains(strings.ToLower(s.Text), lowerQuery) {
			matches = append(matches, s)
		}
	}
	return matches
}

func formatSearchResult(videoID, query string, matches []transcriptSegment) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for %q in video %s.", query, videoID)
	}
	lines := make([]string, len(matches))
	for i, s := range matches {
		lines[i] = fmt.Sprintf("[%s] %s", FormatTimestamp(s.Offset/1000), s.Text)
	}
	return fmt.Sprintf("Found %d match(es) for %q:\n\n%s", len(matches), query, strings.Join(lines, "\n"))
}

// TranscriptErrorText turns a raw fetch error into a user-facing message,
// classifying by substring the same way the upstream TS implementation
// does (timeout / missing captions / network error / generic).
func TranscriptErrorText(videoID string, err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "timed out"):
		return fmt.Sprintf("Transcript fetch timed out for video %s. Please try again.", videoID)
	case strings.Contains(message, "No transcript available") || strings.Contains(message, "captions"):
		return fmt.Sprintf("No transcript available for video %s. The video may not have captions.", videoID)
	case strings.Contains(message, "ENOTFOUND") || strings.Contains(message, "ECONNREFUSED"):
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

func fetchSegments(ctx context.Context, videoID, language string) ([]transcriptSegment, error) {
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

// SearchInTranscript searches for a keyword/phrase in the transcript and
// returns matching lines with timestamps, or a "no matches" message.
func SearchInTranscript(ctx context.Context, videoID, query, language string) (string, error) {
	segments, err := fetchSegments(ctx, videoID, normalizeLanguage(language))
	if err != nil {
		return "", err
	}
	return formatSearchResult(videoID, query, searchSegments(segments, query)), nil
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
