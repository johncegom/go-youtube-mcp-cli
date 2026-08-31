package core

import (
	"strings"
	"testing"
)

// searchSegmentsWithContext has no upstream TS equivalent (new in task 13;
// the old searchSegments/formatSearchResult it replaces did have a TS
// origin, but per docs/tasks/13-context-search/TASK.md this is a
// deliberate improvement, not a port — ground truth is the task's own
// reviewed Definition of Done, same convention as task 11/12/14).

var searchFixture = []transcriptSegment{
	{Text: "we are learning about machine", Offset: 0, Duration: 3000},
	{Text: "learning models today", Offset: 3000, Duration: 3000},
	{Text: "in the middle of the video", Offset: 20000, Duration: 3000},
	{Text: "near the very end now", Offset: 96000, Duration: 2000},
}

// TestSearchSegmentsWithContext_CrossBoundaryMatch covers DoD 13.1: a
// query phrase spanning two adjacent segments is found. Neither segment's
// own Text contains the full phrase — proof the old per-segment approach
// would have missed this.
func TestSearchSegmentsWithContext_CrossBoundaryMatch(t *testing.T) {
	if strings.Contains(strings.ToLower(searchFixture[0].Text), "machine learning") ||
		strings.Contains(strings.ToLower(searchFixture[1].Text), "machine learning") {
		t.Fatal("fixture invalid: \"machine learning\" must not appear within a single segment")
	}
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "machine learning", 15)
	if matchCount != 1 {
		t.Fatalf("matchCount = %d, want 1", matchCount)
	}
	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1", len(blocks))
	}
}

// TestSearchSegmentsWithContext_SingleSegmentMatch covers DoD 13.2:
// existing single-segment matches still found.
func TestSearchSegmentsWithContext_SingleSegmentMatch(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "middle", 15)
	if matchCount != 1 {
		t.Fatalf("matchCount = %d, want 1", matchCount)
	}
	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1", len(blocks))
	}
}

// TestSearchSegmentsWithContext_CaseInsensitive covers DoD 13.2.
func TestSearchSegmentsWithContext_CaseInsensitive(t *testing.T) {
	matchCount, _ := searchSegmentsWithContext(searchFixture, "MACHINE", 15)
	if matchCount != 1 {
		t.Fatalf("matchCount = %d, want 1", matchCount)
	}
}

// TestSearchSegmentsWithContext_NoMatches covers DoD 13.2: "no matches"
// preserved for zero hits.
func TestSearchSegmentsWithContext_NoMatches(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "zzz", 15)
	if matchCount != 0 {
		t.Fatalf("matchCount = %d, want 0", matchCount)
	}
	if blocks != nil {
		t.Fatalf("blocks = %v, want nil", blocks)
	}
}

// TestFormatSearchResultWithContext_NoMatches pins the exact "no matches"
// wording — internal/cli/search.go's printSearchResult detects it via
// strings.HasPrefix(result, "No matches found"), so this string must stay
// byte-compatible with that check.
func TestFormatSearchResultWithContext_NoMatches(t *testing.T) {
	got := formatSearchResultWithContext("abc123", "zzz", 0, nil)
	want := `No matches found for "zzz" in video abc123.`
	if got != want {
		t.Errorf("formatSearchResultWithContext() = %q, want %q", got, want)
	}
}

// TestSearchSegmentsWithContext_StartBoundary covers DoD 13.3: a match at
// the very first segment with a large context window must not panic or
// produce phantom entries, even though the window's start goes negative.
func TestSearchSegmentsWithContext_StartBoundary(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "we are", 9999)
	if matchCount != 1 || len(blocks) != 1 {
		t.Fatalf("matchCount=%d len(blocks)=%d, want 1,1", matchCount, len(blocks))
	}
	if blocks[0][0] != searchFixture[0] {
		t.Fatalf("blocks[0][0] = %+v, want %+v", blocks[0][0], searchFixture[0])
	}
}

// TestSearchSegmentsWithContext_EndBoundary covers DoD 13.3: a match at
// the last segment with a large context window must not panic or run past
// the end of the transcript.
func TestSearchSegmentsWithContext_EndBoundary(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "end now", 9999)
	if matchCount != 1 || len(blocks) != 1 {
		t.Fatalf("matchCount=%d len(blocks)=%d, want 1,1", matchCount, len(blocks))
	}
	last := blocks[0][len(blocks[0])-1]
	if last != searchFixture[len(searchFixture)-1] {
		t.Fatalf("last segment in block = %+v, want %+v", last, searchFixture[len(searchFixture)-1])
	}
}

// TestSearchSegmentsWithContext_OverlappingWindowsMerge covers DoD 13.4:
// two matches whose context windows overlap must merge into a single
// block, while the match count still reflects 2 distinct matches.
func TestSearchSegmentsWithContext_OverlappingWindowsMerge(t *testing.T) {
	// "learning" appears in segment 0 (offset 0) and segment 1 (offset
	// 3000); with 15s of context each window easily overlaps the other.
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "learning", 15)
	if matchCount != 2 {
		t.Fatalf("matchCount = %d, want 2", matchCount)
	}
	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1 (overlapping windows should merge)", len(blocks))
	}
}

// TestSearchSegmentsWithContext_FarApartMatchesStaySeparate covers DoD
// 13.4's other half: matches whose windows don't overlap stay as separate
// blocks.
func TestSearchSegmentsWithContext_FarApartMatchesStaySeparate(t *testing.T) {
	fixture := []transcriptSegment{
		{Text: "alpha marker one", Offset: 0, Duration: 1000},
		{Text: "far away marker two", Offset: 100000, Duration: 1000},
	}
	matchCount, blocks := searchSegmentsWithContext(fixture, "marker", 5)
	if matchCount != 2 {
		t.Fatalf("matchCount = %d, want 2", matchCount)
	}
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2 (far-apart windows should not merge)", len(blocks))
	}
}

// TestSearchSegmentsWithContext_ZeroContext covers DoD 13.5: context: 0
// returns matched segments only.
func TestSearchSegmentsWithContext_ZeroContext(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "middle", 0)
	if matchCount != 1 || len(blocks) != 1 {
		t.Fatalf("matchCount=%d len(blocks)=%d, want 1,1", matchCount, len(blocks))
	}
	want := []transcriptSegment{searchFixture[2]}
	if len(blocks[0]) != len(want) || blocks[0][0] != want[0] {
		t.Fatalf("blocks[0] = %+v, want %+v", blocks[0], want)
	}
}

// TestSearchSegmentsWithContext_RegexMetacharactersLiteral covers the Test
// Plan's "query with regex metacharacters treated literally" case.
func TestSearchSegmentsWithContext_RegexMetacharactersLiteral(t *testing.T) {
	fixture := []transcriptSegment{
		{Text: "a.b test literal dot", Offset: 0, Duration: 1000},
	}
	if mc, _ := searchSegmentsWithContext(fixture, "a.b", 15); mc != 1 {
		t.Fatalf("literal query a.b: matchCount = %d, want 1", mc)
	}
	// If "." were treated as a regex wildcard, "axb" would also match —
	// it must not.
	if mc, _ := searchSegmentsWithContext(fixture, "axb", 15); mc != 0 {
		t.Fatalf("wildcard-style query axb: matchCount = %d, want 0 (literal substring only)", mc)
	}
}

// TestSearchSegmentsWithContext_EmptyQueryNoHang is a direct regression
// test for the infinite-loop risk: strings.Index(s, "") always returns 0,
// so without an explicit empty-query guard the match loop would never
// advance. This test completing at all is the assertion. A whitespace-only
// query like " " is NOT dangerous the same way — it's a real substring
// (the space joining segments) that advances normally, so it's excluded
// here (rejecting it is the MCP handler's existing job, upstream of this
// function, not this function's).
func TestSearchSegmentsWithContext_EmptyQueryNoHang(t *testing.T) {
	matchCount, blocks := searchSegmentsWithContext(searchFixture, "", 15)
	if matchCount != 0 || blocks != nil {
		t.Fatalf("query \"\": matchCount=%d blocks=%v, want 0,nil", matchCount, blocks)
	}
}

// TestFormatSearchResultWithContext_BlocksSeparatedByDashes covers DoD
// 13.4's rendering: multiple blocks are separated by "---".
func TestFormatSearchResultWithContext_BlocksSeparatedByDashes(t *testing.T) {
	fixture := []transcriptSegment{
		{Text: "alpha marker one", Offset: 0, Duration: 1000},
		{Text: "far away marker two", Offset: 100000, Duration: 1000},
	}
	matchCount, blocks := searchSegmentsWithContext(fixture, "marker", 5)
	got := formatSearchResultWithContext("abc123", "marker", matchCount, blocks)
	if !strings.Contains(got, "\n---\n") {
		t.Fatalf("formatSearchResultWithContext() = %q, want a \\n---\\n block separator", got)
	}
	if strings.Count(got, "\n---\n") != 1 {
		t.Fatalf("formatSearchResultWithContext() separator count = %d, want 1", strings.Count(got, "\n---\n"))
	}
}

// FuzzSegmentForCharIndex checks segmentForCharIndex never returns an
// out-of-bounds index for any pos, against a fixed representative
// segStarts scaffold (docs/tasks/13-context-search/TASK.md Test Plan).
func FuzzSegmentForCharIndex(f *testing.F) {
	f.Add(0)
	f.Add(-100)
	f.Add(6)
	f.Add(1_000_000)
	segStarts := []int{0, 8, 20, 45}

	f.Fuzz(func(t *testing.T, pos int) {
		idx := segmentForCharIndex(segStarts, pos)
		if idx < 0 || idx >= len(segStarts) {
			t.Fatalf("segmentForCharIndex(%d) = %d, out of bounds [0,%d)", pos, idx, len(segStarts))
		}
	})
}

// FuzzSearchSegmentsWithContext feeds arbitrary segment text (including
// invalid UTF-8), an arbitrary query, and an arbitrary context through the
// full pipeline, asserting no panic and output consistency.
func FuzzSearchSegmentsWithContext(f *testing.F) {
	f.Add("we are learning about machine", "learning models today", "machine learning", 15.0)
	f.Add("", "", "x", 0.0)
	f.Add("\xff\xfe invalid utf8", "more \x80 bytes", "invalid", -5.0)
	f.Add("a.b (test) [x]", "plain text", "(test)", 999999.0)

	f.Fuzz(func(t *testing.T, text0, text1, query string, contextSecs float64) {
		segs := []transcriptSegment{
			{Text: text0, Offset: 0, Duration: 3000},
			{Text: text1, Offset: 3000, Duration: 3000},
		}
		if contextSecs < 0 {
			contextSecs = 0 // mirrors SearchInTranscript's own clamp
		}
		matchCount, blocks := searchSegmentsWithContext(segs, query, contextSecs)
		if matchCount == 0 && blocks != nil {
			t.Fatalf("matchCount 0 but blocks = %+v, want nil", blocks)
		}
		if len(blocks) > matchCount {
			t.Fatalf("block count %d exceeds match count %d", len(blocks), matchCount)
		}
		for _, b := range blocks {
			if len(b) == 0 {
				t.Fatal("empty block returned")
			}
		}
		_ = formatSearchResultWithContext("abc123", query, matchCount, blocks) // must not panic
	})
}
