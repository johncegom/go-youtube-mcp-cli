package core

import "testing"

func TestExtractVideoID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bare id", "dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"youtu.be", "https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"youtu.be with query", "https://youtu.be/dQw4w9WgXcQ?t=30", "dQw4w9WgXcQ"},
		{"watch url", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"watch url extra params", "https://youtube.com/watch?v=dQw4w9WgXcQ&list=abc", "dQw4w9WgXcQ"},
		{"shorts", "https://youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"embed", "https://youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"v path", "https://youtube.com/v/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"invalid id", "not-a-valid-id", ""},
		{"unrelated url", "https://example.com/watch?v=dQw4w9WgXcQx", ""},
		{"empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractVideoID(tc.input)
			if got != tc.want {
				t.Errorf("ExtractVideoID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// FuzzExtractVideoID checks the function's core invariant against arbitrary
// input: it never panics, and any non-empty result is always exactly an
// 11-character [a-zA-Z0-9_-] video ID — never a partial match, never
// something carrying URL syntax through unvalidated. This matters because
// the result gets embedded directly into a watch-page URL and a yt-dlp
// subprocess argument elsewhere in the package.
func FuzzExtractVideoID(f *testing.F) {
	seeds := []string{
		"dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?t=30",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/watch?v=dQw4w9WgXcQ&list=abc",
		"https://youtube.com/shorts/dQw4w9WgXcQ",
		"https://youtube.com/embed/dQw4w9WgXcQ",
		"https://youtube.com/v/dQw4w9WgXcQ",
		"not-a-valid-id",
		"https://example.com/watch?v=dQw4w9WgXcQx",
		"",
		"://not a url at all",
		"youtube.com",
		"https://youtube.com/",
		"https://youtube.com/shorts/",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := ExtractVideoID(input)
		if got != "" && !videoIDRe.MatchString(got) {
			t.Errorf("ExtractVideoID(%q) = %q, which is not a valid 11-char video ID", input, got)
		}
	})
}
