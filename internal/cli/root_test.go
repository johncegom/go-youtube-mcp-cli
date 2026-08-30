package cli

// internal/cli has no upstream TS test equivalent (it's cobra command
// wiring around already-tested internal/core functions), so ground truth
// here is the command tree's own documented contract (flag names/defaults
// visible in each newXCommand()), not a ported TS behavior.

import (
	"testing"

	"github.com/spf13/cobra"
)

func checkFlagDefault(t *testing.T, cmd *cobra.Command, name, want string) {
	t.Helper()
	f := cmd.Flags().Lookup(name)
	if f == nil {
		t.Fatalf("expected flag %q to be registered on %q", name, cmd.Use)
	}
	if f.DefValue != want {
		t.Errorf("--%s default = %q, want %q", name, f.DefValue, want)
	}
}

func TestNewRootCommand_Subcommands(t *testing.T) {
	root := NewRootCommand("test")

	want := map[string]bool{"transcript": false, "search": false, "metadata": false, "download": false}
	for _, cmd := range root.Commands() {
		name := cmd.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}

func TestNewRootCommand_PersistentQuietFlag(t *testing.T) {
	root := NewRootCommand("test")
	f := root.PersistentFlags().Lookup("quiet")
	if f == nil {
		t.Fatal("expected a persistent --quiet flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--quiet default = %q, want %q", f.DefValue, "false")
	}
}

func TestTranscriptCommand_FlagDefaults(t *testing.T) {
	cmd := newTranscriptCommand()
	checkFlagDefault(t, cmd, "language", "en")
	checkFlagDefault(t, cmd, "timestamps", "false")
	checkFlagDefault(t, cmd, "save", "false")
}

func TestSearchCommand_FlagDefaults(t *testing.T) {
	cmd := newSearchCommand()
	checkFlagDefault(t, cmd, "language", "en")
}

func TestMetadataCommand_FlagDefaults(t *testing.T) {
	cmd := newMetadataCommand()
	checkFlagDefault(t, cmd, "json", "false")
}

func TestDownloadCommand_FlagDefaults(t *testing.T) {
	cmd := newDownloadCommand()
	checkFlagDefault(t, cmd, "audio", "false")
	checkFlagDefault(t, cmd, "quality", "hd720")
	checkFlagDefault(t, cmd, "format", "mp3")
}

func TestFatal_CallsExitFuncWithStatus1(t *testing.T) {
	orig := exitFunc
	defer func() { exitFunc = orig }()

	var gotCode int
	called := false
	exitFunc = func(code int) { called = true; gotCode = code }

	_ = fatal("boom: %s", "details")

	if !called {
		t.Fatal("expected fatal() to call exitFunc")
	}
	if gotCode != 1 {
		t.Errorf("exit code = %d, want 1", gotCode)
	}
}
