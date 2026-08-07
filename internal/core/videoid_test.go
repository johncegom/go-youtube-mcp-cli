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
