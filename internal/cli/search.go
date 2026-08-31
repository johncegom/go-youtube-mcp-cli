package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

// printSearchResult writes result to out, unless it's a "no matches found"
// message, in which case it's written to errOut and the process exits 1
// (via exitFunc, seamed for tests).
func printSearchResult(out, errOut io.Writer, result string) {
	if strings.HasPrefix(result, "No matches found") {
		fmt.Fprintln(errOut, result)
		exitFunc(1)
		return
	}
	fmt.Fprintln(out, result)
}

func newSearchCommand() *cobra.Command {
	var language string
	var contextSecs float64

	cmd := &cobra.Command{
		Use:   "search <url> <query>",
		Short: "Search for a keyword in the transcript with timestamps and context",
		Args:  cobra.ExactArgs(2),
		Example: `  youtube-cli search dQw4w9WgXcQ "never gonna"
  youtube-cli search dQw4w9WgXcQ "chorus" --language en
  youtube-cli search dQw4w9WgXcQ "chorus" --context 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			videoID := core.ExtractVideoID(args[0])
			if videoID == "" {
				return fatal("invalid YouTube URL or video ID: %q", args[0])
			}
			query := args[1]

			stop := spinner(fmt.Sprintf("Searching for %q", query), quietFlag)
			result, err := core.SearchInTranscript(cmd.Context(), videoID, query, language, contextSecs)
			stop()
			if err != nil {
				return fatal("%s", core.TranscriptErrorText(videoID, err))
			}

			printSearchResult(cmd.OutOrStdout(), os.Stderr, result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "en", "language code")
	cmd.Flags().Float64Var(&contextSecs, "context", 15, "seconds of context around each match (0 = matched segments only)")

	return cmd
}
