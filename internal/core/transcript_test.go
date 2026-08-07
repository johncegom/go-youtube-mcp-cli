package core

import (
	"errors"
	"reflect"
	"testing"
)

// Fixture and expected output were derived by running the upstream TS
// parseVtt implementation directly through Node (not by hand-deriving what
// the Go port "should" produce), so the Go implementation must conform to
// this, not the other way around.
const vttFixture = `WEBVTT
Kind: captions
Language: en

00:00:00.000 --> 00:00:02.500
Hello <b>world</b>

00:00:02.500 --> 00:00:05.000
This &amp; that

00:00:02.500 --> 00:00:05.000
This &amp; that

00:01:05.200 --> 00:01:07.000
Multi
line   text

00:01:07.000 --> 00:01:08.000
<c.colorE5E5E5>tags &lt;stripped&gt; &nbsp; here</c>

00:01:08.000 --> 00:01:09.000
`

func TestParseVtt(t *testing.T) {
	want := []transcriptSegment{
		{Text: "Hello world", Offset: 0, Duration: 2500},
		{Text: "This & that", Offset: 2500, Duration: 2500},
		{Text: "Multi line text", Offset: 65200, Duration: 1800},
		{Text: "tags <stripped> here", Offset: 67000, Duration: 1000},
	}

	got := parseVtt(vttFixture)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseVtt() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseVtt_NoTimingBlocksIgnored(t *testing.T) {
	content := "WEBVTT\nKind: captions\nLanguage: en\n"
	got := parseVtt(content)
	if len(got) != 0 {
		t.Errorf("parseVtt(header-only) = %+v, want empty", got)
	}
}

func TestParseVtt_EmptyTextBlockDropped(t *testing.T) {
	content := "00:00:00.000 --> 00:00:01.000\n\n00:00:01.000 --> 00:00:02.000\nreal text"
	got := parseVtt(content)
	if len(got) != 1 || got[0].Text != "real text" {
		t.Errorf("parseVtt(empty-text block) = %+v, want single segment %q", got, "real text")
	}
}

// Ground truth for the functions below was derived by running the
// equivalent TS logic (getTranscriptText/getTranscriptTimed/
// searchInTranscript/transcriptErrorText) directly through Node.

var sampleSegments = []transcriptSegment{
	{Text: "Hello world", Offset: 0, Duration: 2500},
	{Text: "This & that", Offset: 2500, Duration: 2500},
	{Text: "Multi line text", Offset: 65200, Duration: 1800},
}

func TestTranscriptText(t *testing.T) {
	want := "Hello world This & that Multi line text"
	if got := transcriptText(sampleSegments); got != want {
		t.Errorf("transcriptText() = %q, want %q", got, want)
	}
}

func TestTranscriptTimed(t *testing.T) {
	want := "[00:00] Hello world\n[00:02] This & that\n[01:05] Multi line text"
	if got := transcriptTimed(sampleSegments); got != want {
		t.Errorf("transcriptTimed() = %q, want %q", got, want)
	}
}

func TestFormatSearchResult_Found(t *testing.T) {
	matches := searchSegments(sampleSegments, "text")
	want := "Found 1 match(es) for \"text\":\n\n[01:05] Multi line text"
	if got := formatSearchResult("abc123", "text", matches); got != want {
		t.Errorf("formatSearchResult() = %q, want %q", got, want)
	}
}

func TestFormatSearchResult_NoMatches(t *testing.T) {
	matches := searchSegments(sampleSegments, "zzz")
	want := `No matches found for "zzz" in video abc123.`
	if got := formatSearchResult("abc123", "zzz", matches); got != want {
		t.Errorf("formatSearchResult() = %q, want %q", got, want)
	}
}

func TestSearchSegments_CaseInsensitive(t *testing.T) {
	matches := searchSegments(sampleSegments, "HELLO")
	if len(matches) != 1 || matches[0].Text != "Hello world" {
		t.Errorf("searchSegments(case-insensitive) = %+v", matches)
	}
}

func TestTranscriptErrorText(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"timed out", errors.New("Transcript fetch timed out"), `Transcript fetch timed out for video abc123. Please try again.`},
		{"no transcript", errors.New("No transcript available"), `No transcript available for video abc123. The video may not have captions.`},
		{"captions", errors.New("the video may not have captions"), `No transcript available for video abc123. The video may not have captions.`},
		{"dns error", errors.New("dial tcp: lookup youtube.com: ENOTFOUND"), `Network error while fetching transcript for video abc123. Please check your internet connection.`},
		{"conn refused", errors.New("dial tcp: ECONNREFUSED"), `Network error while fetching transcript for video abc123. Please check your internet connection.`},
		{"other", errors.New("boom"), `Failed to fetch transcript for video abc123: boom`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TranscriptErrorText("abc123", tc.err); got != tc.want {
				t.Errorf("TranscriptErrorText() = %q, want %q", got, tc.want)
			}
		})
	}
}
