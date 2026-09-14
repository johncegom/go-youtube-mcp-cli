package core

import (
	"reflect"
	"testing"
)

// computeTranscriptStats has no upstream equivalent (new in task 17).
// Ground truth is the rule set in docs/tasks/17-video-brief/TASK.md (17.3),
// applied by hand to small fixtures: word count over speech segments only,
// WPM = round(words / (last segment end in minutes)) and 0 when that end is
// 0, non-speech cue = a segment whose entire text is one "[...]" token
// (counted per lowercased token), gap = next.Offset - cur end, negatives
// clamped to 0, longest gap reported with the time it starts.

func TestComputeTranscriptStats(t *testing.T) {
	cases := []struct {
		name string
		segs []transcriptSegment
		want TranscriptStats
	}{
		{
			name: "empty",
			segs: nil,
			want: TranscriptStats{},
		},
		{
			// [Music] 0-18470 | "hello world" 18800-21790 | [music] 25000-26000 |
			// "one two three" 30000-60000. Words = 2+3 = 5; last end = 60000 ms =
			// 1 min → 5 wpm. Cues: 2, both "[music]". Gaps: 330, 3210, 4000 ms →
			// longest 4.0 s starting at 26.0 s.
			name: "mixed speech and cues",
			segs: []transcriptSegment{
				{Text: "[Music]", Offset: 0, Duration: 18470},
				{Text: "hello world", Offset: 18800, Duration: 2990},
				{Text: "[music]", Offset: 25000, Duration: 1000},
				{Text: "one two three", Offset: 30000, Duration: 30000},
			},
			want: TranscriptStats{
				Words:              5,
				SpeakingRateWPM:    5,
				NonSpeechCues:      2,
				NonSpeechBreakdown: map[string]int{"[music]": 2},
				LongestGapSecs:     4,
				LongestGapAtSecs:   26,
			},
		},
		{
			// Overlapping cues (auto-caption style): 0-5000 then 1000-2000.
			// Gap = 1000 - 5000 < 0 → clamped to 0. Last end = max end = 5000 ms
			// = 1/12 min; 2 words / (1/12) = 24 wpm.
			name: "overlap clamps gap and uses max end",
			segs: []transcriptSegment{
				{Text: "a", Offset: 0, Duration: 5000},
				{Text: "b", Offset: 1000, Duration: 1000},
			},
			want: TranscriptStats{Words: 2, SpeakingRateWPM: 24, NonSpeechBreakdown: map[string]int{}},
		},
		{
			name: "zero-length transcript has no rate",
			segs: []transcriptSegment{{Text: "a b", Offset: 0, Duration: 0}},
			want: TranscriptStats{Words: 2, SpeakingRateWPM: 0, NonSpeechBreakdown: map[string]int{}},
		},
		{
			// 7 words over 90000 ms = 1.5 min → 4.67 → rounds to 5.
			name: "rate rounds to nearest",
			segs: []transcriptSegment{{Text: "one two three four five six seven", Offset: 0, Duration: 90000}},
			want: TranscriptStats{Words: 7, SpeakingRateWPM: 5, NonSpeechBreakdown: map[string]int{}},
		},
		{
			// A bracket token inline with speech is not a cue — the whole text
			// must be the token. Cue keys are lowercased so [MUSIC]/[Music] merge.
			name: "inline bracket is speech; cue keys lowercased",
			segs: []transcriptSegment{
				{Text: "[inaudible] hello", Offset: 0, Duration: 1000},
				{Text: "[MUSIC]", Offset: 1000, Duration: 1000},
				{Text: "[Applause]", Offset: 2000, Duration: 1000},
				{Text: "[Music]", Offset: 3000, Duration: 57000},
			},
			want: TranscriptStats{
				Words:              2,
				SpeakingRateWPM:    2,
				NonSpeechCues:      3,
				NonSpeechBreakdown: map[string]int{"[music]": 2, "[applause]": 1},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeTranscriptStats(tc.segs)
			if got.NonSpeechBreakdown == nil {
				got.NonSpeechBreakdown = map[string]int{}
			}
			if tc.want.NonSpeechBreakdown == nil {
				tc.want.NonSpeechBreakdown = map[string]int{}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("computeTranscriptStats() =\n%+v\nwant\n%+v", got, tc.want)
			}
		})
	}
}

// TestChaptersFrom pins the shared description-first / player-response-
// fallback selection that FetchChapters and FetchVideoBrief both use.
func TestChaptersFrom(t *testing.T) {
	desc := "0:00 Intro\n1:00 Middle\n2:00 End"
	pr := []Chapter{{Title: "PR only", StartSecs: 0}}

	if got := chaptersFrom(map[string]string{"description": desc}, pr); len(got) != 3 || got[1].Title != "Middle" {
		t.Errorf("description tier should win; got %+v", got)
	}
	if got := chaptersFrom(map[string]string{"description": "no timestamps here"}, pr); !reflect.DeepEqual(got, pr) {
		t.Errorf("player-response fallback should be used; got %+v", got)
	}
	if got := chaptersFrom(map[string]string{}, nil); got != nil {
		t.Errorf("no chapters anywhere should be nil; got %+v", got)
	}
}

// TestComputeTranscriptStats_GapRoundsToMilliseconds pins a rendering bug
// caught by task 17's live smoke test: offsets parsed from VTT timestamps
// (e.g. "00:02:43.771" → 163771.00000000003 ms) leave float noise in the
// gap arithmetic, which then rendered as "6.771000000000029s". Gaps and
// their start times are rounded to whole milliseconds before the /1000.
func TestComputeTranscriptStats_GapRoundsToMilliseconds(t *testing.T) {
	segs := []transcriptSegment{
		{Text: "a", Offset: parseVttTime("00:02:40.000"), Duration: parseVttTime("00:02:43.771") - parseVttTime("00:02:40.000")},
		{Text: "b", Offset: parseVttTime("00:02:50.542"), Duration: 1000},
	}
	got := computeTranscriptStats(segs)
	if got.LongestGapSecs != 6.771 || got.LongestGapAtSecs != 163.771 {
		t.Errorf("gap = %v at %v, want 6.771 at 163.771", got.LongestGapSecs, got.LongestGapAtSecs)
	}
}
