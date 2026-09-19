package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
)

// formatVideoBrief has no upstream equivalent (task 17). Ground truth is
// the section contract in docs/tasks/17-video-brief/TASK.md (17.5): every
// section that succeeded is rendered, a failed section is replaced by its
// failure line, chapters are omitted when metadata failed (same fetch),
// and isError is true iff the transcript section failed.

func okBrief() core.VideoBrief {
	return core.VideoBrief{
		Metadata: map[string]string{
			"title": "T", "channel": "C", "publishDate": "2026-01-01",
			"viewCount": "42", "duration": "1:00", "description": "0:00 Intro\n0:20 Middle\n0:40 End",
		},
		Chapters: []core.Chapter{
			{Title: "Intro", StartSecs: 0}, {Title: "Middle", StartSecs: 20}, {Title: "End", StartSecs: 40},
		},
		TranscriptTimed: "[0:00] hello world\n[0:30] [Music]",
		CaptionKind:     core.CaptionAuto,
		Stats: core.TranscriptStats{
			Words: 2, SpeakingRateWPM: 2, NonSpeechCues: 3,
			NonSpeechBreakdown: map[string]int{"[music]": 2, "[applause]": 1},
			LongestGapSecs:     12.5, LongestGapAtSecs: 90,
		},
	}
}

func TestFormatVideoBrief_AllOK(t *testing.T) {
	text, isErr := formatVideoBrief("vid123", okBrief())
	if isErr {
		t.Fatal("isError = true, want false when every section succeeded")
	}
	want := strings.Join([]string{
		"Video: vid123",
		"Title: T",
		"Channel: C",
		"Published: 2026-01-01",
		"Views: 42",
		"Duration: 1:00",
		"Description: 0:00 Intro\n0:20 Middle\n0:40 End",
		"",
		"Chapters:",
		"[00:00] Intro",
		"[00:20] Middle",
		"[00:40] End",
		"",
		"Transcript stats:",
		"Captions: likely auto-generated",
		"Words: 2",
		"Speaking rate: 2 words/min",
		"Non-speech cues: 3 ([applause] x1, [music] x2)",
		"Longest gap: 12.5s at [01:30]",
		"",
		"Transcript (timed):",
		"[0:00] hello world",
		"[0:30] [Music]",
	}, "\n")
	if text != want {
		t.Errorf("formatVideoBrief() =\n%s\n\nwant\n%s", text, want)
	}
}

func TestFormatVideoBrief_CaptionKindWording(t *testing.T) {
	for kind, want := range map[core.CaptionKind]string{
		core.CaptionAuto:     "Captions: likely auto-generated",
		core.CaptionUploaded: "Captions: likely uploaded",
		core.CaptionUnknown:  "Captions: unknown",
	} {
		b := okBrief()
		b.CaptionKind = kind
		text, _ := formatVideoBrief("v", b)
		if !strings.Contains(text, want+"\n") {
			t.Errorf("kind %q: output lacks %q", kind, want)
		}
	}
}

func TestFormatVideoBrief_MetadataFailedOnly(t *testing.T) {
	b := okBrief()
	b.Metadata, b.Chapters, b.MetadataErr = nil, nil, errors.New("boom")
	text, isErr := formatVideoBrief("v", b)
	if isErr {
		t.Error("isError = true, want false: the transcript section succeeded")
	}
	if !strings.Contains(text, "Video: v\nMetadata: Failed to fetch metadata: boom\n\nTranscript stats:\n") {
		t.Errorf("metadata failure line / section order wrong:\n%s", text)
	}
	if strings.Contains(text, "Chapters") || strings.Contains(text, "Title:") {
		t.Errorf("chapters/metadata should be omitted entirely when metadata failed:\n%s", text)
	}
	if !strings.Contains(text, "Transcript (timed):\n[0:00] hello world") {
		t.Errorf("transcript should still be present:\n%s", text)
	}
}

func TestFormatVideoBrief_TranscriptFailedOnly(t *testing.T) {
	b := okBrief()
	b.TranscriptTimed, b.CaptionKind, b.Stats = "", "", core.TranscriptStats{}
	b.TranscriptErr = errors.New("transcript fetch timed out")
	text, isErr := formatVideoBrief("v", b)
	if !isErr {
		t.Error("isError = false, want true when the transcript section failed")
	}
	if !strings.Contains(text, "Title: T\n") || !strings.Contains(text, "Chapters:\n[00:00] Intro") {
		t.Errorf("metadata and chapters should still be present:\n%s", text)
	}
	wantLine := "Transcript: " + core.TranscriptErrorText("v", b.TranscriptErr)
	if !strings.Contains(text, "\n\n"+wantLine) || !strings.HasSuffix(text, wantLine) {
		t.Errorf("transcript failure line missing or not last:\n%s", text)
	}
	for _, absent := range []string{"Transcript stats:", "Words:", "Captions:", "Transcript (timed):"} {
		if strings.Contains(text, absent) {
			t.Errorf("%q should be omitted when the transcript failed:\n%s", absent, text)
		}
	}
}

func TestFormatVideoBrief_BothFailed(t *testing.T) {
	b := core.VideoBrief{MetadataErr: errors.New("m"), TranscriptErr: errors.New("t")}
	text, isErr := formatVideoBrief("v", b)
	if !isErr {
		t.Error("isError = false, want true")
	}
	want := "Video: v\nMetadata: Failed to fetch metadata: m\n\nTranscript: " + core.TranscriptErrorText("v", b.TranscriptErr)
	if text != want {
		t.Errorf("formatVideoBrief() =\n%s\nwant\n%s", text, want)
	}
}

func TestFormatVideoBrief_NoChapters(t *testing.T) {
	b := okBrief()
	b.Chapters = nil
	text, _ := formatVideoBrief("v", b)
	if !strings.Contains(text, "\n\nNo chapters found.\n\nTranscript stats:") {
		t.Errorf("want the no-chapters line in the chapters slot:\n%s", text)
	}
}

func TestFormatVideoBrief_ZeroStats(t *testing.T) {
	b := okBrief()
	b.Stats = core.TranscriptStats{}
	text, _ := formatVideoBrief("v", b)
	want := "Words: 0\nSpeaking rate: 0 words/min\nNon-speech cues: 0\nLongest gap: 0s at [00:00]\n"
	if !strings.Contains(text, want) {
		t.Errorf("zero stats rendering wrong:\n%s", text)
	}
}

func TestFormatVideoBrief_MetadataMissingFieldsUsesSameLinesAsGetMetadata(t *testing.T) {
	b := okBrief()
	b.Metadata = map[string]string{"title": "Only title"}
	text, _ := formatVideoBrief("v", b)
	if !strings.Contains(text, "Video: v\nTitle: Only title\n\nChapters:") {
		t.Errorf("only non-empty metadata keys should render, same as get_metadata:\n%s", text)
	}
}

func TestGetVideoBriefHandler_InvalidURL(t *testing.T) {
	res, _, err := getVideoBriefHandler(context.Background(), nil, urlLangInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
	if !strings.Contains(contentText(t, res), "Invalid YouTube URL") {
		t.Errorf("unexpected text: %s", contentText(t, res))
	}
}

// task 19: an auto-detected non-en language is stated in the stats block; a
// brief without it (explicit language, or en) renders exactly as before —
// TestFormatVideoBrief_AllOK above is that unchanged-output check.
func TestFormatVideoBrief_AutoDetectedLanguageLine(t *testing.T) {
	b := okBrief()
	b.Language, b.LanguageAutoDetected = "vi", true
	text, isErr := formatVideoBrief("vid123", b)
	if isErr {
		t.Fatal("isError = true, want false")
	}
	const want = "Transcript stats:\nLanguage: vi (auto-detected spoken language)\nCaptions: likely auto-generated"
	if !strings.Contains(text, want) {
		t.Errorf("brief missing the language line before Captions:\n%s", text)
	}

	b.LanguageAutoDetected = false // e.g. an explicit language: no line
	text, _ = formatVideoBrief("vid123", b)
	if strings.Contains(text, "Language:") {
		t.Errorf("brief has a Language line although it was not auto-detected:\n%s", text)
	}
}
