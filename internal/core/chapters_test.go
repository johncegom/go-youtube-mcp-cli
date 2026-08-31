package core

import (
	"strings"
	"testing"
)

// parseChapters has no upstream TS equivalent to port faithfully (new in
// task 14). Ground truth is docs/tasks/14-chapters/TASK.md's own reviewed
// Definition of Done and YouTube's published chapter rules (chapters must
// start at 0:00, there must be at least 3, and timestamps must be strictly
// ascending), verified against real watch pages.

// TestParseChapters_RealChapteredDescription covers DoD 14.1 (timestamp
// then title order). Fixture verified by eye against a real chaptered
// video's description (video ID recorded in docs/tasks/14-chapters/TASK.md
// alongside the manual smoke test).
func TestParseChapters_RealChapteredDescription(t *testing.T) {
	desc := "In this video we build a REST API from scratch.\n\n" +
		"0:00 Introduction\n" +
		"1:30 Setting up the project\n" +
		"5:45 Writing the first endpoint\n" +
		"12:10 Adding tests\n" +
		"18:00 Wrap up\n"
	got := parseChapters(desc)
	want := []chapter{
		{Title: "Introduction", StartSecs: 0},
		{Title: "Setting up the project", StartSecs: 90},
		{Title: "Writing the first endpoint", StartSecs: 345},
		{Title: "Adding tests", StartSecs: 730},
		{Title: "Wrap up", StartSecs: 1080},
	}
	assertChapters(t, got, want)
}

// TestParseChapters_TitleThenTimestampOrder covers DoD 14.1's other line
// order.
func TestParseChapters_TitleThenTimestampOrder(t *testing.T) {
	desc := "Chapters:\n" +
		"Introduction 0:00\n" +
		"Setup 1:30\n" +
		"Conclusion 5:00\n"
	got := parseChapters(desc)
	want := []chapter{
		{Title: "Introduction", StartSecs: 0},
		{Title: "Setup", StartSecs: 90},
		{Title: "Conclusion", StartSecs: 300},
	}
	assertChapters(t, got, want)
}

// TestParseChapters_HMSLongVideo covers DoD 14.1's H:MM:SS long-video trap
// case from the Test Plan.
func TestParseChapters_HMSLongVideo(t *testing.T) {
	desc := "0:00:00 Intro\n" +
		"1:02:03 Deep dive\n" +
		"2:15:30 Q&A\n"
	got := parseChapters(desc)
	want := []chapter{
		{Title: "Intro", StartSecs: 0},
		{Title: "Deep dive", StartSecs: 3723},
		{Title: "Q&A", StartSecs: 8130},
	}
	assertChapters(t, got, want)
}

// TestParseChapters_RealUnchapteredDescription covers the Test Plan's
// "real unchaptered description containing incidental timestamps" case —
// the mid-sentence timestamp mention is neither a line prefix nor a line
// suffix, so it must not be treated as a chapter marker.
func TestParseChapters_RealUnchapteredDescription(t *testing.T) {
	desc := "Thanks for watching! At 2:30 he says something interesting, " +
		"and later around 8:00 things get wild. Don't forget to subscribe.\n" +
		"Follow me on social media for more."
	if got := parseChapters(desc); got != nil {
		t.Fatalf("parseChapters() = %v, want nil", got)
	}
}

// TestParseChapters_StartsNotAtZero covers DoD 14.2.
func TestParseChapters_StartsNotAtZero(t *testing.T) {
	desc := "0:05 Intro\n1:30 Setup\n5:00 Conclusion\n"
	if got := parseChapters(desc); got != nil {
		t.Fatalf("parseChapters() = %v, want nil", got)
	}
}

// TestParseChapters_TooFewEntries covers DoD 14.2.
func TestParseChapters_TooFewEntries(t *testing.T) {
	desc := "0:00 Intro\n1:30 Conclusion\n"
	if got := parseChapters(desc); got != nil {
		t.Fatalf("parseChapters() = %v, want nil", got)
	}
}

// TestParseChapters_OutOfOrderTimestamps covers DoD 14.2 — both a
// non-ascending pair and a duplicate (non-strictly-ascending) pair.
func TestParseChapters_OutOfOrderTimestamps(t *testing.T) {
	cases := map[string]string{
		"descending": "0:00 Intro\n5:00 Middle\n1:30 Oops\n",
		"duplicate":  "0:00 Intro\n1:30 Setup\n1:30 Duplicate\n",
	}
	for name, desc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := parseChapters(desc); got != nil {
				t.Fatalf("parseChapters(%q) = %v, want nil", name, got)
			}
		})
	}
}

// TestParseChapters_TitleTrimming covers DoD 14.3 — leading/trailing
// separators (-, –, :, whitespace, and doubled separators) are stripped.
func TestParseChapters_TitleTrimming(t *testing.T) {
	desc := "0:00 - Intro\n" +
		"0:30: Setup\n" +
		"1:00 – – Topic\n"
	got := parseChapters(desc)
	want := []chapter{
		{Title: "Intro", StartSecs: 0},
		{Title: "Setup", StartSecs: 30},
		{Title: "Topic", StartSecs: 60},
	}
	assertChapters(t, got, want)
}

// TestParseChapters_EmptyTitleLineSkipped covers DoD 14.3 — a bare
// timestamp line with no title text is dropped. Dropping it here still
// leaves 3 valid entries, so the overall result stays non-nil.
func TestParseChapters_EmptyTitleLineSkipped(t *testing.T) {
	desc := "0:00 Intro\n" +
		"0:45 - \n" +
		"1:30 Setup\n" +
		"5:00 Conclusion\n"
	got := parseChapters(desc)
	want := []chapter{
		{Title: "Intro", StartSecs: 0},
		{Title: "Setup", StartSecs: 90},
		{Title: "Conclusion", StartSecs: 300},
	}
	assertChapters(t, got, want)
}

// TestParseChapters_EmptyDescription is a cheap edge case.
func TestParseChapters_EmptyDescription(t *testing.T) {
	if got := parseChapters(""); got != nil {
		t.Fatalf("parseChapters(\"\") = %v, want nil", got)
	}
}

func assertChapters(t *testing.T, got, want []chapter) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("parseChapters() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseChapters()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// FuzzParseChapters proves parseChapters never panics on arbitrary input,
// and that any non-nil result always satisfies the validity invariants:
// at least 3 entries, the first starts at 0, and timestamps are strictly
// ascending (docs/tasks/14-chapters/TASK.md Test Plan).
func FuzzParseChapters(f *testing.F) {
	seeds := []string{
		"0:00 Intro\n1:30 Setup\n5:00 Conclusion",
		"Thanks for watching! At 2:30 he says something interesting.",
		"0:05 Intro\n1:30 Setup\n5:00 Conclusion",
		"0:00 Intro\n1:30 Setup\n",
		"",
		strings.Repeat("0:00 x\n", 500),
		"0:00:00 Intro\n1:02:03 Deep dive\n2:15:30 Q&A",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, description string) {
		chs := parseChapters(description)
		if chs == nil {
			return
		}
		if len(chs) < 3 {
			t.Fatalf("parseChapters returned %d chapters, want >= 3", len(chs))
		}
		if chs[0].StartSecs != 0 {
			t.Fatalf("first chapter StartSecs = %v, want 0", chs[0].StartSecs)
		}
		for i := 1; i < len(chs); i++ {
			if chs[i].StartSecs <= chs[i-1].StartSecs {
				t.Fatalf("chapters not strictly ascending at %d: %v <= %v", i, chs[i].StartSecs, chs[i-1].StartSecs)
			}
		}
	})
}

// TestChaptersFromPlayerResponseJSON_Valid covers DoD 14.4's happy path:
// a chapters array present in the ytInitialPlayerResponse JSON is decoded.
func TestChaptersFromPlayerResponseJSON_Valid(t *testing.T) {
	raw := []byte(`{
		"playerOverlayVideoDetailsRenderer": {},
		"chapters": [
			{"title": {"simpleText": "Intro"}, "timeRangeStartMillis": 0},
			{"title": {"simpleText": "Setup"}, "timeRangeStartMillis": 90000},
			{"title": {"simpleText": "Wrap up"}, "timeRangeStartMillis": 300000}
		]
	}`)
	got := chaptersFromPlayerResponseJSON(raw)
	want := []chapter{
		{Title: "Intro", StartSecs: 0},
		{Title: "Setup", StartSecs: 90},
		{Title: "Wrap up", StartSecs: 300},
	}
	assertChapters(t, got, want)
}

// TestChaptersFromPlayerResponseJSON_Malformed covers DoD 14.4: malformed
// JSON must degrade to nil, never a panic or an error.
func TestChaptersFromPlayerResponseJSON_Malformed(t *testing.T) {
	if got := chaptersFromPlayerResponseJSON([]byte(`{not valid json`)); got != nil {
		t.Fatalf("chaptersFromPlayerResponseJSON(malformed) = %v, want nil", got)
	}
}

// TestChaptersFromPlayerResponseJSON_Absent covers DoD 14.4: a valid
// player-response JSON with no chapters field at all must yield nil.
func TestChaptersFromPlayerResponseJSON_Absent(t *testing.T) {
	raw := []byte(`{"videoDetails": {"title": "Some Video"}}`)
	if got := chaptersFromPlayerResponseJSON(raw); got != nil {
		t.Fatalf("chaptersFromPlayerResponseJSON(no chapters) = %v, want nil", got)
	}
}

// TestChaptersFromPlayerResponseJSON_TooFewIgnoresValidityGate documents
// that the raw player-response tier does not re-apply parseChapters'
// validity gate (>=3, starts at 0, ascending) — YouTube's own JSON is
// trusted as-is, unlike the free-text description tier which must guard
// against incidental timestamp mentions.
func TestChaptersFromPlayerResponseJSON_TooFewIgnoresValidityGate(t *testing.T) {
	raw := []byte(`{"chapters": [{"title": {"simpleText": "Only one"}, "timeRangeStartMillis": 0}]}`)
	got := chaptersFromPlayerResponseJSON(raw)
	want := []chapter{{Title: "Only one", StartSecs: 0}}
	assertChapters(t, got, want)
}
