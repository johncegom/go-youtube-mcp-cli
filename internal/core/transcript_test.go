package core

import (
	"errors"
	"fmt"
	"net"
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

// FuzzParseVtt checks that parseVtt never panics on arbitrary input. This
// content comes straight from a yt-dlp-downloaded subtitle file (i.e.
// network-sourced, effectively untrusted), so it must degrade gracefully
// (empty/partial results) on malformed or adversarial VTT rather than
// crash the caller.
func FuzzParseVtt(f *testing.F) {
	f.Add(vttFixture)
	f.Add("")
	f.Add("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHello")
	f.Add("not vtt at all, just plain text\nwith some\nnewlines")
	f.Add("00:00:00.000 --> \n<garbled")
	f.Add("--> --> -->\n\n\n\n")

	f.Fuzz(func(t *testing.T, content string) {
		_ = parseVtt(content) // must not panic
	})
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

// f64 returns a pointer to v, for building *float64 range bounds in tests.
func f64(v float64) *float64 { return &v }

// filterSegmentsByRange has no upstream TS equivalent (new in task 11).
// Ground truth is the function's own specification from
// docs/tasks/11-transcript-cache/TASK.md: inclusive of segments whose
// Offset falls exactly on start or end, a nil bound is open-ended, and an
// out-of-bounds window yields an empty (not nil-panicking) result.
func TestFilterSegmentsByRange(t *testing.T) {
	segs := []transcriptSegment{
		{Text: "a", Offset: 0, Duration: 1000},
		{Text: "b", Offset: 2500, Duration: 2500},
		{Text: "c", Offset: 65200, Duration: 1800},
	}
	cases := []struct {
		name           string
		startMs, endMs *float64
		want           []transcriptSegment
	}{
		{"open-ended both", nil, nil, segs},
		{"open start", nil, f64(2500), segs[:2]},
		{"open end", f64(2500), nil, segs[1:]},
		{"exact boundary inclusive on start", f64(2500), f64(2500), segs[1:2]},
		{"exact boundary inclusive on end", f64(0), f64(0), segs[0:1]},
		{"window covering only middle", f64(1000), f64(3000), segs[1:2]},
		{"out of bounds window", f64(1_000_000), f64(2_000_000), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterSegmentsByRange(segs, tc.startMs, tc.endMs)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("filterSegmentsByRange() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// pickVttFile has no upstream TS equivalent (the TS original just takes
// vttFiles[0] blindly, which is BUG-001's second contributing cause — see
// docs/BUGS.md). Ground truth here is the fix's own specification: prefer a
// file whose name exactly ends in ".<language>.vtt", falling back to the
// first ".vtt" file only if no exact match exists.
func TestPickVttFile(t *testing.T) {
	cases := []struct {
		name     string
		files    []string
		language string
		want     string
	}{
		{
			name:     "single exact match",
			files:    []string{"sub.en.vtt"},
			language: "en",
			want:     "sub.en.vtt",
		},
		{
			name:     "prefers exact match among multiple candidates",
			files:    []string{"sub.en-de-DE.vtt", "sub.en.vtt", "sub.en-ja.vtt"},
			language: "en",
			want:     "sub.en.vtt",
		},
		{
			name:     "falls back to first .vtt when no exact match",
			files:    []string{"sub.en-de-DE.vtt", "sub.en-ja.vtt"},
			language: "en",
			want:     "sub.en-de-DE.vtt",
		},
		{
			name:     "no vtt files at all",
			files:    []string{"sub.info.json"},
			language: "en",
			want:     "",
		},
		{
			name:     "empty file list",
			files:    []string{},
			language: "en",
			want:     "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickVttFile(tc.files, tc.language); got != tc.want {
				t.Errorf("pickVttFile(%v, %q) = %q, want %q", tc.files, tc.language, got, tc.want)
			}
		})
	}
}

// dialRefused returns a real *net.OpError (satisfying net.Error) by
// binding a loopback listener, closing it immediately, then dialing the
// now-closed address — deterministic, offline-safe, and identical on
// Windows/Linux CI, unlike hand-written OS-specific errno text.
func dialRefused(t *testing.T) error {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("dialRefused: listen: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("dialRefused: close: %v", err)
	}
	_, dialErr := net.Dial("tcp", addr)
	if dialErr == nil {
		t.Fatal("dialRefused: expected dial to a closed port to fail")
	}
	return dialErr
}

// The "network error" cases' ground truth comes from three real, distinct
// sources (see docs/BUGS.md BUG-003 and TranscriptErrorText's doc
// comment): a genuine Go net.Error (dialRefused, above — a real dial to a
// just-closed loopback listener) for EnsureYtDlp's own failure path, and
// two literal captured yt-dlp stderr lines for yt-dlp's own process-failure
// path — one from running yt-dlp directly against a URL on the RFC 2606
// reserved ".invalid" TLD, and one from running the CLI against a real
// YouTube URL with HTTPS_PROXY/HTTP_PROXY pointed at an unreachable local
// port (yt-dlp fails at a different pipeline stage — YouTube's API-page
// fetch — with different wrapper phrasing). None of these three is
// hand-reasoned or ported from the upstream TS project's Node.js error
// vocabulary.
func TestTranscriptErrorText(t *testing.T) {
	const wantNetworkMsg = `Network error while fetching transcript for video abc123. Please check your internet connection.`
	const capturedYtDlpStderr = `ERROR: [generic] Unable to download webpage: HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed) (caused by TransportError("HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed)"))`
	const capturedYtDlpProxyStderr = `ERROR: [youtube] dQw4w9WgXcQ: Unable to download API page: ('Unable to connect to proxy', NewConnectionError("HTTPSConnection(host='127.0.0.1', port=1): Failed to establish a new connection: [WinError 10061] No connection could be made because the target machine actively refused it")) (caused by ProxyError('(\'Unable to connect to proxy\', NewConnectionError("HTTPSConnection(host=\'127.0.0.1\', port=1): Failed to establish a new connection: [WinError 10061] No connection could be made because the target machine actively refused it"))'))`

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"timed out", errors.New("Transcript fetch timed out"), `Transcript fetch timed out for video abc123. Please try again.`},
		{"no transcript", errors.New("No transcript available"), `No transcript available for video abc123. The video may not have captions.`},
		{"captions", errors.New("the video may not have captions"), `No transcript available for video abc123. The video may not have captions.`},
		{"real net.Error (dial refused)", dialRefused(t), wantNetworkMsg},
		{"real net.Error, wrapped like EnsureYtDlp", fmt.Errorf("yt-dlp binary not found and auto-install failed: %w", dialRefused(t)), wantNetworkMsg},
		{"captured yt-dlp network stderr (generic webpage fetch)", errors.New(capturedYtDlpStderr), wantNetworkMsg},
		{"captured yt-dlp network stderr (API page fetch via proxy)", errors.New(capturedYtDlpProxyStderr), wantNetworkMsg},
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

// FuzzTranscriptErrorText checks that TranscriptErrorText never panics on
// arbitrary error text, including text shaped like the two real
// network-error sources it now classifies.
func FuzzTranscriptErrorText(f *testing.F) {
	f.Add("")
	f.Add("Transcript fetch timed out")
	f.Add(`ERROR: [generic] Unable to download webpage: HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed)`)
	f.Add(`ERROR: [youtube] dQw4w9WgXcQ: Unable to download API page: ('Unable to connect to proxy', NewConnectionError(...))`)
	f.Add("dial tcp: lookup youtube.com: ENOTFOUND")
	f.Add("boom")

	f.Fuzz(func(t *testing.T, message string) {
		_ = TranscriptErrorText("abc123", errors.New(message)) // must not panic
	})
}
