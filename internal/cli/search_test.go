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
