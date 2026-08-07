package core

import "testing"

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
