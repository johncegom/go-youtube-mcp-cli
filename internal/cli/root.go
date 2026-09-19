// Package cli implements the youtube-cli command-line tool: a cobra port
// of the upstream project's commander-based CLI (packages/cli/src/index.ts),
// built on top of internal/core.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

var quietFlag bool

// exitFunc is os.Exit by default; tests swap it for a recorder so exit-1
// paths (fatal, search's "no matches" branch) can be asserted in-process
// instead of actually terminating the test binary. See docs/DECISIONS.md.
var exitFunc = os.Exit

// fatal prints an "error: ..." message to stderr and exits the process with
// status 1, mirroring the TS CLI's fatal() helper. It "returns" an error
// only so it can be used as `return fatal(...)` inside a RunE func; the
// process has already exited by the time that return executes.
func fatal(format string, args ...any) error {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	exitFunc(1)
	return nil
}

// NewRootCommand builds the youtube-cli command tree.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:     "youtube-cli",
		Short:   "YouTube transcripts, metadata, search, and downloads from the command line",
		Version: version,
		Example: `  youtube-cli transcript https://youtu.be/dQw4w9WgXcQ
  youtube-cli transcript dQw4w9WgXcQ --timestamps
  youtube-cli search dQw4w9WgXcQ "never gonna"
  youtube-cli metadata https://youtube.com/watch?v=dQw4w9WgXcQ --json
  youtube-cli download dQw4w9WgXcQ --audio --format mp3`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "suppress progress output")

	root.AddCommand(newTranscriptCommand())
	root.AddCommand(newSearchCommand())
	root.AddCommand(newMetadataCommand())
	root.AddCommand(newDownloadCommand())

	return root
}

// languageFlagUsage is the --language help text for the commands whose
// omitted language is resolved from the video (task 18).
const languageFlagUsage = "language code (default: the video's spoken language, else en)"

// printLanguageNote tells the user, on stderr and outside the transcript on
// stdout, which language was auto-detected for them. It prints nothing when
// the caller passed --language or the detected language is "en".
func printLanguageNote(w io.Writer, requested, resolved string) {
	if note := core.LanguageNote(requested, resolved); note != "" {
		fmt.Fprintln(w, note)
	}
}
