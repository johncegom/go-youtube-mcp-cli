package core

import (
	"encoding/json"
	"testing"
)

func TestExtractMetaTag(t *testing.T) {
	cases := []struct {
		name string
		html string
		attr string
		val  string
		want string
	}{
		{
			name: "attr before content",
			html: `<meta property="og:title" content="Hello World">`,
			attr: "property", val: "og:title",
			want: "Hello World",
		},
		{
			name: "content before attr",
			html: `<meta content="Hello World" property="og:title">`,
			attr: "property", val: "og:title",
			want: "Hello World",
		},
		{
			name: "not found",
			html: `<meta property="og:description" content="something">`,
			attr: "property", val: "og:title",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractMetaTag(tc.html, tc.attr, tc.val); got != tc.want {
				t.Errorf("extractMetaTag() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKeywordsToString(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"array", `["a","b","c"]`, "a, b, c"},
		{"string", `"a,b,c"`, "a,b,c"},
		{"empty", ``, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw json.RawMessage
			if tc.raw != "" {
				raw = json.RawMessage(tc.raw)
			}
			if got := keywordsToString(raw); got != tc.want {
				t.Errorf("keywordsToString(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestAtoiSafe(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"123", 123},
		{"0", 0},
		{"12a", 0},
		{"", 0},
	}
	for _, tc := range cases {
		if got := atoiSafe(tc.in); got != tc.want {
			t.Errorf("atoiSafe(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestFillFromJSONLD_TopLevelObject(t *testing.T) {
	html := `<html><head>
<script type="application/ld+json">
{"@type":"VideoObject","name":"Test Title","description":"Test Desc",
 "author":{"name":"Test Channel"},"uploadDate":"2020-01-01",
 "interactionStatistic":{"interactionType":"http://schema.org/WatchAction","userInteractionCount":123},
 "duration":"PT1M30S"}
</script>
</head></html>`

	meta := map[string]string{}
	fillFromJSONLD(html, meta)

	want := map[string]string{
		"title":       "Test Title",
		"description": "Test Desc",
		"channel":     "Test Channel",
		"publishDate": "2020-01-01",
		"viewCount":   "123",
		"duration":    "1:30",
	}
	for k, v := range want {
		if meta[k] != v {
			t.Errorf("meta[%q] = %q, want %q", k, meta[k], v)
		}
	}
}

func TestFillFromJSONLD_Graph(t *testing.T) {
	html := `<script type="application/ld+json">
{"@graph":[{"@type":"WebPage","name":"ignored"},{"@type":"VideoObject","name":"Graph Title","author":"Plain Author String"}]}
</script>`

	meta := map[string]string{}
	fillFromJSONLD(html, meta)

	if meta["title"] != "Graph Title" {
		t.Errorf("meta[title] = %q, want %q", meta["title"], "Graph Title")
	}
	if meta["channel"] != "Plain Author String" {
		t.Errorf("meta[channel] = %q, want %q", meta["channel"], "Plain Author String")
	}
}

func TestFillFromJSONLD_DoesNotOverwriteExisting(t *testing.T) {
	html := `<script type="application/ld+json">
{"@type":"VideoObject","name":"Should Not Win"}
</script>`

	meta := map[string]string{"title": "Already Set"}
	fillFromJSONLD(html, meta)

	if meta["title"] != "Already Set" {
		t.Errorf("meta[title] = %q, want %q (should not be overwritten)", meta["title"], "Already Set")
	}
}
