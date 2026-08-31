package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintSearchResult_NoMatches_ExitsWithStatus1(t *testing.T) {
	orig := exitFunc
	defer func() { exitFunc = orig }()

	var gotCode int
	called := false
	exitFunc = func(code int) { called = true; gotCode = code }

	var out, errOut bytes.Buffer
	printSearchResult(&out, &errOut, "No matches found for \"xyz\"")

	if !called {
		t.Fatal("expected exitFunc to be called for a no-matches result")
	}
	if gotCode != 1 {
		t.Errorf("exit code = %d, want 1", gotCode)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on no-matches, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), "No matches found") {
		t.Errorf("stderr = %q, want it to contain the no-matches message", errOut.String())
	}
}

func TestPrintSearchResult_Match_WritesToStdout(t *testing.T) {
	orig := exitFunc
	defer func() { exitFunc = orig }()

	called := false
	exitFunc = func(int) { called = true }

	var out, errOut bytes.Buffer
	printSearchResult(&out, &errOut, "[00:12] found it")

	if called {
		t.Fatal("did not expect exitFunc to be called for a real match")
	}
	if !strings.Contains(out.String(), "found it") {
		t.Errorf("stdout = %q, want it to contain the result", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr should be empty on a match, got %q", errOut.String())
	}
}

// TestNewSearchCommand_ContextFlagDefault covers DoD 13.7: the --context
// flag defaults to 15, matching core.SearchInTranscript's own default.
func TestNewSearchCommand_ContextFlagDefault(t *testing.T) {
	cmd := newSearchCommand()
	f := cmd.Flags().Lookup("context")
	if f == nil {
		t.Fatal("expected a --context flag to be registered")
	}
	if f.DefValue != "15" {
		t.Errorf("--context default = %q, want %q", f.DefValue, "15")
	}
}

// TestNewSearchCommand_ContextFlagParses covers DoD 13.7: --context 0
// round-trips to exactly 0, not silently falling back to the default (a
// careless implementation using Cobra's Changed() trickery could conflate
// "explicitly set to 0" with "not set").
func TestNewSearchCommand_ContextFlagParses(t *testing.T) {
	cmd := newSearchCommand()
	if err := cmd.Flags().Set("context", "0"); err != nil {
		t.Fatalf("Set(context, 0) failed: %v", err)
	}
	got, err := cmd.Flags().GetFloat64("context")
	if err != nil {
		t.Fatalf("GetFloat64(context) failed: %v", err)
	}
	if got != 0 {
		t.Errorf("--context 0 parsed as %v, want 0", got)
	}
}
