package core

import (
	"fmt"
	"sort"
	"strings"
)

// buildMergedStream concatenates all segments' lowercased text, joined by
// single spaces, and returns the char offset (in the merged string) where
// each segment's text begins — segStarts[i] is monotonically
// non-decreasing and has the same length as segments.
func buildMergedStream(segments []transcriptSegment) (merged string, segStarts []int) {
	segStarts = make([]int, len(segments))
	parts := make([]string, len(segments))
	pos := 0
	for i, s := range segments {
		segStarts[i] = pos
		lower := strings.ToLower(s.Text)
		parts[i] = lower
		pos += len(lower)
		if i < len(segments)-1 {
			pos++ // for the joining space
		}
	}
	return strings.Join(parts, " "), segStarts
}

// segmentForCharIndex returns the index of the segment whose text spans
// character offset pos in the merged stream built by buildMergedStream,
// via a floor search on segStarts. Always returns a valid index (clamped),
// never panics — pos may be negative, in a joining-space gap, or past the
// end of the stream.
func segmentForCharIndex(segStarts []int, pos int) int {
	i := sort.Search(len(segStarts), func(i int) bool { return segStarts[i] > pos })
	idx := i - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(segStarts) {
		idx = len(segStarts) - 1
	}
	return idx
}

// searchSegmentsWithContext finds every non-overlapping, case-insensitive
// occurrence of query in the merged transcript stream (so phrases
// spanning two segments are found), expands each match to a window of
// ±contextSecs (already clamped to >= 0 by the caller) around its matched
// segment(s), merges overlapping/adjacent windows, and returns the
// distinct match count plus one segment slice per merged block, in
// chronological order. An empty query or empty segments list returns
// (0, nil) immediately — critical to avoid an infinite loop, since
// strings.Index(s, "") always returns 0.
func searchSegmentsWithContext(segments []transcriptSegment, query string, contextSecs float64) (matchCount int, blocks [][]transcriptSegment) {
	lowerQuery := strings.ToLower(query)
	if lowerQuery == "" || len(segments) == 0 {
		return 0, nil
	}

	merged, segStarts := buildMergedStream(segments)

	type window struct{ startMs, endMs float64 }
	var windows []window

	contextMs := contextSecs * 1000
	pos := 0
	for {
		i := strings.Index(merged[pos:], lowerQuery)
		if i < 0 {
			break
		}
		matchStart := pos + i
		matchEnd := matchStart + len(lowerQuery) // exclusive
		matchCount++

		segStart := segmentForCharIndex(segStarts, matchStart)
		segEnd := segmentForCharIndex(segStarts, matchEnd-1)

		windows = append(windows, window{
			startMs: segments[segStart].Offset - contextMs,
			endMs:   segments[segEnd].Offset + segments[segEnd].Duration + contextMs,
		})

		pos = matchEnd // non-overlapping advance, strings.Count semantics
	}

	if matchCount == 0 {
		return 0, nil
	}

	sort.Slice(windows, func(i, j int) bool { return windows[i].startMs < windows[j].startMs })

	mergedWindows := make([]window, 0, len(windows))
	for _, w := range windows {
		if n := len(mergedWindows); n > 0 && w.startMs <= mergedWindows[n-1].endMs {
			if w.endMs > mergedWindows[n-1].endMs {
				mergedWindows[n-1].endMs = w.endMs
			}
			continue
		}
		mergedWindows = append(mergedWindows, w)
	}

	blocks = make([][]transcriptSegment, 0, len(mergedWindows))
	for _, w := range mergedWindows {
		start, end := w.startMs, w.endMs
		blocks = append(blocks, filterSegmentsByRange(segments, &start, &end))
	}
	return matchCount, blocks
}

// formatSearchResultWithContext renders the "No matches" message (byte-
// identical to the old formatSearchResult's, since
// internal/cli/search.go's printSearchResult detects it via
// strings.HasPrefix(result, "No matches found")) or the match count
// header followed by one block of timed lines per merged window, blocks
// separated by "---".
func formatSearchResultWithContext(videoID, query string, matchCount int, blocks [][]transcriptSegment) string {
	if matchCount == 0 {
		return fmt.Sprintf("No matches found for %q in video %s.", query, videoID)
	}
	blockTexts := make([]string, len(blocks))
	for i, b := range blocks {
		blockTexts[i] = transcriptTimed(b)
	}
	return fmt.Sprintf("Found %d match(es) for %q:\n\n%s", matchCount, query, strings.Join(blockTexts, "\n---\n"))
}
