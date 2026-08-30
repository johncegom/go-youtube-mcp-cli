package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatMetadataJSON(t *testing.T) {
	meta := map[string]string{"title": "A Title", "channel": "A Channel"}
	got, err := formatMetadataJSON(meta)
	if err != nil {
		t.Fatalf("formatMetadataJSON: %v", err)
	}

	var roundTrip map[string]string
	if err := json.Unmarshal([]byte(got), &roundTrip); err != nil {
		t.Fatalf("output isn't valid JSON: %v\noutput: %s", err, got)
	}
	if roundTrip["title"] != "A Title" || roundTrip["channel"] != "A Channel" {
		t.Errorf("round-tripped map = %+v, want title/channel preserved", roundTrip)
	}
	if !strings.Contains(got, "  ") {
		t.Error("expected indented JSON output")
	}
}

func TestFormatMetadataPlain(t *testing.T) {
	cases := []struct {
		name string
		meta map[string]string
		want []string // substrings expected present
		omit []string // substrings expected absent
	}{
		{
			name: "all fields present",
			meta: map[string]string{
				"title": "A Title", "channel": "A Channel", "publishDate": "2020-01-01",
				"viewCount": "42", "duration": "1:00", "channelUrl": "http://x",
				"channelId": "UCxxx", "description": "hello world",
			},
			want: []string{"Title:       A Title", "Channel:     A Channel", "Published:   2020-01-01",
				"Views:       42", "Duration:    1:00", "Channel URL: http://x", "Channel ID:  UCxxx",
				"Description:\nhello world"},
		},
		{
			name: "empty fields omitted",
			meta: map[string]string{"title": "Only Title"},
			want: []string{"Title:       Only Title"},
			omit: []string{"Channel:", "Published:", "Views:", "Duration:", "Channel URL:", "Channel ID:", "Description:"},
		},
		{
			name: "all empty produces no output",
			meta: map[string]string{},
			omit: []string{"Title:", "Channel:"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatMetadataPlain(&buf, tc.meta)
			got := buf.String()
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("output missing %q\nfull output:\n%s", w, got)
				}
			}
			for _, o := range tc.omit {
				if strings.Contains(got, o) {
					t.Errorf("output unexpectedly contains %q\nfull output:\n%s", o, got)
				}
			}
		})
	}
}
