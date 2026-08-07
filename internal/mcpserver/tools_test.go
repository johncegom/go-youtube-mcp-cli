package mcpserver

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult/invalidURLResult have no upstream TS equivalent to derive
// ground truth from — they're the MCP response-building layer, not a port
// of specific TS logic. The invariants below are the fix's/function's own
// specification: text content flowing through this layer can originate
// from scraped HTML, foreign-language transcripts, or directly from an MCP
// client's tool-call arguments, so it must never panic and must never
// silently corrupt arbitrary content (including invalid UTF-8, control
// characters, or literal fmt-verb-shaped strings like "%s"/"%n" — textResult
// and invalidURLResult only ever pass such content as an *argument* to
// fmt.Sprintf, never as the format string itself, so it should be inert
// regardless of content).

func contentText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil {
		t.Fatal("result is nil")
	}
	if len(res.Content) != 1 {
		t.Fatalf("Content has %d entries, want exactly 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

func TestTextResult(t *testing.T) {
	cases := []struct {
		text    string
		isError bool
	}{
		{"hello", false},
		{"something went wrong", true},
		{"", false},
	}
	for _, tc := range cases {
		res := textResult(tc.text, tc.isError)
		if got := contentText(t, res); got != tc.text {
			t.Errorf("textResult(%q, %v) text = %q, want %q", tc.text, tc.isError, got, tc.text)
		}
		if res.IsError != tc.isError {
			t.Errorf("textResult(%q, %v) IsError = %v, want %v", tc.text, tc.isError, res.IsError, tc.isError)
		}
	}
}

func TestInvalidURLResult(t *testing.T) {
	res := invalidURLResult("not-a-url")
	if !res.IsError {
		t.Error("invalidURLResult().IsError = false, want true")
	}
	text := contentText(t, res)
	if !strings.Contains(text, "not-a-url") {
		t.Errorf("invalidURLResult() text = %q, want it to contain the original url", text)
	}
}

// FuzzTextResult checks textResult never panics and always round-trips the
// exact text it was given (no truncation, no mangling) regardless of
// content — including invalid UTF-8 and fmt-verb-shaped strings.
func FuzzTextResult(f *testing.F) {
	seeds := []string{
		"normal text",
		"",
		"%s %d %n %v",
		"unicode: 你好 🎬 ñ",
		"line1\nline2\r\nline3",
		"\x00\x01\x02 control chars",
		strings.Repeat("a", 10000),
	}
	for _, s := range seeds {
		f.Add(s, false)
		f.Add(s, true)
	}

	f.Fuzz(func(t *testing.T, text string, isError bool) {
		res := textResult(text, isError)
		if res == nil {
			t.Fatal("textResult returned nil")
		}
		if len(res.Content) != 1 {
			t.Fatalf("Content has %d entries, want exactly 1", len(res.Content))
		}
		tc, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("Content[0] is %T, want *mcp.TextContent", res.Content[0])
		}
		if tc.Text != text {
			t.Errorf("textResult(%q) round-tripped to %q", text, tc.Text)
		}
		if res.IsError != isError {
			t.Errorf("textResult(_, %v).IsError = %v", isError, res.IsError)
		}
	})
}

// FuzzInvalidURLResult checks invalidURLResult never panics on arbitrary
// input and always produces a well-formed isError result. It does NOT
// assert the result text contains url as a literal substring: url is
// passed through fmt.Sprintf's %q verb, which deliberately escapes
// quotes/backslashes/control characters for a readable error message, so
// for inputs containing those characters the quoted representation differs
// from the raw string by design — that's correct behavior, not something
// to flag as a bug.
func FuzzInvalidURLResult(f *testing.F) {
	seeds := []string{
		"https://example.com",
		"",
		"%s%d%n",
		"unicode: 你好",
		strings.Repeat("x", 5000),
		"\x00null byte",
		`has "quotes" and \backslashes`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, url string) {
		res := invalidURLResult(url)
		if res == nil {
			t.Fatal("invalidURLResult returned nil")
		}
		if !res.IsError {
			t.Error("invalidURLResult().IsError = false, want true")
		}
		if len(res.Content) != 1 {
			t.Fatalf("Content has %d entries, want exactly 1", len(res.Content))
		}
		tc, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("Content[0] is %T, want *mcp.TextContent", res.Content[0])
		}
		if tc.Text == "" {
			t.Error("invalidURLResult() produced empty text")
		}
	})
}
