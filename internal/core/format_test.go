package core

import (
	"strings"
	"testing"
)

func TestFormatTimestamp(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{0, "00:00"},
		{5, "00:05"},
		{65, "01:05"},
		{3599, "59:59"},
		{3600, "1:00:00"},
		{3665, "1:01:05"},
	}
	for _, tc := range cases {
		if got := FormatTimestamp(tc.seconds); got != tc.want {
			t.Errorf("FormatTimestamp(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{0, "0:00"},
		{65, "1:05"},
		{3665, "1:01:05"},
	}
	for _, tc := range cases {
		if got := FormatDuration(tc.seconds); got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestParseISODuration(t *testing.T) {
	cases := []struct {
		iso  string
		want int
	}{
		{"PT1H2M3S", 3723},
		{"PT5M", 300},
		{"PT45S", 45},
		{"garbage", 0},
	}
	for _, tc := range cases {
		if got := parseISODuration(tc.iso); got != tc.want {
			t.Errorf("parseISODuration(%q) = %d, want %d", tc.iso, got, tc.want)
		}
	}
}

func TestSanitizeTitle(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Normal Title", "Normal Title"},
		{`Weird: "Title" / \ * ? < > |`, "Weird_Title"},
		{"___leading and trailing___", "leading and trailing"},
		{"", "video"},
		{"***", "video"},
	}
	for _, tc := range cases {
		if got := SanitizeTitle(tc.title); got != tc.want {
			t.Errorf("SanitizeTitle(%q) = %q, want %q", tc.title, got, tc.want)
		}
	}
}

// illegalFilenameChars are the characters SanitizeTitle must never let
// through — used by the fuzz test below to check that invariant directly,
// rather than relying on illegalFilenameCharsRe (which would just be
// testing the regex against itself).
var illegalFilenameChars = []string{`\`, `/`, `:`, `*`, `?`, `"`, `<`, `>`, `|`}

// FuzzSanitizeTitle checks SanitizeTitle's core invariants against
// arbitrary input: the result is never empty (always falls back to
// "video"), and never contains a path separator or other filesystem-illegal
// character. This is security-relevant, not just cosmetic — video titles
// are attacker-influenced (set by whoever uploaded the video) and this is
// the only sanitization applied before the result is used to build a real
// filesystem path in transcript.go (SaveTranscriptFile) and download.go
// (StartVideoDownload/StartAudioDownload's predicted paths).
func FuzzSanitizeTitle(f *testing.F) {
	seeds := []string{
		"Normal Title",
		`Weird: "Title" / \ * ? < > |`,
		"___leading and trailing___",
		"",
		"***",
		"../../../etc/passwd",
		"..\\..\\Windows\\System32\\config",
		"con", // reserved device name on Windows
		strings.Repeat("a", 1000),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, title string) {
		got := SanitizeTitle(title)
		if got == "" {
			t.Errorf("SanitizeTitle(%q) = \"\", want a non-empty fallback", title)
		}
		for _, c := range illegalFilenameChars {
			if strings.Contains(got, c) {
				t.Errorf("SanitizeTitle(%q) = %q, still contains illegal filename character %q", title, got, c)
			}
		}
	})
}
