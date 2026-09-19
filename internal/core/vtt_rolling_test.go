package core

import (
	"math"
	"os"
	"testing"
)

// BUG-010: YouTube's auto-generated (ASR) captions are "rolling" two-line
// cues. Each spoken line appears in three blocks — the typing block that
// introduces it (line 1 = the previous line as context, or blank; line 2 =
// the new line, with inline word timings), a ~10 ms carry block repeating
// it, and again as the context line of the next typing block — so the old
// parseVtt (join lines, dedupe on offset|text) returned every line ~3 times.
//
// Ground truth: testdata/asr_rolling_{en,vi}_head.vtt are the first 12
// blocks, byte-for-byte, of real files captured 2026-09-20 with
//
//	yt-dlp --skip-download --write-auto-subs --sub-langs {en|vi} --sub-format vtt
//
// from BqRhBq-_kgE (English speech) and r8CppXSqVDU (Vietnamese speech), both
// ASR-only videos. Each head holds 6 typing blocks and 6 carry blocks, i.e.
// 6 spoken lines. The expected segments below are the spec of docs/BUGS.md
// BUG-010 option 1 applied to those blocks — one segment per typing block,
// text = that block's last non-empty line (tags stripped), offset/duration
// from the block's own timing line — checked against the raw lines, not
// against whatever parseVtt currently prints.

func mustRead(t testing.TB, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// approx compares millisecond offsets/durations that come from float parsing.
func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func assertSegments(t *testing.T, got, want []transcriptSegment) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d:\n got: %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i].Text != want[i].Text || !approx(got[i].Offset, want[i].Offset) || !approx(got[i].Duration, want[i].Duration) {
			t.Errorf("segment %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseVtt_RollingCues_English(t *testing.T) {
	got := parseVtt(mustRead(t, "testdata/asr_rolling_en_head.vtt"))
	assertSegments(t, got, []transcriptSegment{
		{Text: "MCP is blowing up right now. Agent's", Offset: 0, Duration: 3390},
		{Text: "usage of MCP is increasing exponentially", Offset: 3400, Duration: 3150},
		{Text: "and it's not showing any signs of", Offset: 6560, Duration: 2030},
		{Text: "slowing down.", Offset: 8600, Duration: 1310},
		{Text: "But at the same time, many agents are", Offset: 9920, Duration: 1950},
		{Text: "now changing the way that they handle", Offset: 11880, Duration: 2150},
	})
}

func TestParseVtt_RollingCues_Vietnamese(t *testing.T) {
	got := parseVtt(mustRead(t, "testdata/asr_rolling_vi_head.vtt"))
	assertSegments(t, got, []transcriptSegment{
		{Text: "100 tuổi là con số bạn phải đạt được nếu", Offset: 280, Duration: 2869},
		{Text: "muốn được rút bảo hiểm. Trong khi số", Offset: 3159, Duration: 2071},
		{Text: "người trên 100 tuổi ở Việt Nam thì chỉ", Offset: 5240, Duration: 2589},
		{Text: "khoảng 0,7%. Có nghĩa là nếu may mắn thì", Offset: 7839, Duration: 2910},
		{Text: "bạn sẽ là người thứ bảy trong số 100", Offset: 10759, Duration: 2151},
		{Text: "người. Tưởng như đây là một trường hợp", Offset: 12920, Duration: 1510},
	})
}

// The real head captured in task 17 (dQw4w9WgXcQ, --write-auto-subs): a
// long "[Music]" block with a blank context line, an empty 10 ms block, a
// typing block after the silence, and its carry block.
func TestParseVtt_RollingCues_MusicThenLine(t *testing.T) {
	got := parseVtt(vttAutoHeadFixture)
	assertSegments(t, got, []transcriptSegment{
		{Text: "[Music]", Offset: 320, Duration: 18470},
		{Text: "We're no strangers to", Offset: 18800, Duration: 2990},
	})
}

// The consequence that motivated the bug: get_video_brief's word count is
// derived from the parsed segments. The English head's six distinct lines
// hold 7+6+7+2+8+7 = 37 words; the old parser reported about three times
// that (a 3x-inflated speaking rate).
func TestParseVtt_RollingCues_BriefWordCount(t *testing.T) {
	segs := parseVtt(mustRead(t, "testdata/asr_rolling_en_head.vtt"))
	if got := computeTranscriptStats(segs).Words; got != 37 {
		t.Errorf("Words = %d, want 37 (one count per spoken line)", got)
	}
}

// No two consecutive segments of a rolling parse may carry the same text
// (that is exactly the carry-block duplication BUG-010 describes).
func TestParseVtt_RollingCues_NoConsecutiveDuplicates(t *testing.T) {
	for _, path := range []string{"testdata/asr_rolling_en_head.vtt", "testdata/asr_rolling_vi_head.vtt"} {
		segs := parseVtt(mustRead(t, path))
		for i := 1; i < len(segs); i++ {
			if segs[i].Text == segs[i-1].Text {
				t.Errorf("%s: segments %d and %d both %q", path, i-1, i, segs[i].Text)
			}
		}
	}
}

// Characterization: uploaded (non-rolling) captions must parse exactly as
// before the BUG-010 fix — multi-line cues joined into one segment, no
// last-line rule. These pass on the pre-fix parser and must keep passing.
func TestParseVtt_UploadedCaptionsUnchanged(t *testing.T) {
	assertSegments(t, parseVtt(vttUploadedHeadFixture), []transcriptSegment{
		{Text: "[♪♪♪]", Offset: 1360, Duration: 1680},
		{Text: "♪ We're no strangers to love ♪", Offset: 18640, Duration: 3240},
		{Text: "♪ You know the rules and so do I ♪", Offset: 22640, Duration: 4320},
	})
	assertSegments(t, parseVtt(vttUploadedHeadFixture2), []transcriptSegment{
		{Text: "In this course, I'm going to teach you everything you need to know to get started programming", Offset: 0, Duration: 4080},
		{Text: "in Python. Now, Python is one of the most popular programming languages out there. And it's by far", Offset: 4080, Duration: 6480},
	})
}
