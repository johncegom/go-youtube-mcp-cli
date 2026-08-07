package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	var language string

	cmd := &cobra.Command{
		Use:   "search <url> <query>",
		Short: "Search for a keyword in the transcript with timestamps",
		Args:  cobra.ExactArgs(2),
		Example: `  youtube-cli search dQw4w9WgXcQ "never gonna"
  youtube-cli search dQw4w9WgXcQ "chorus" --language en`,
		RunE: func(cmd *cobra.Command, args []string) error {
			videoID := core.ExtractVideoID(args[0])
			if videoID == "" {
				return fatal("invalid YouTube URL or video ID: %q", args[0])
			}
			query := args[1]

			stop := spinner(fmt.Sprintf("Searching for %q", query), quietFlag)
			result, err := core.SearchInTranscript(cmd.Context(), videoID, query, language)
			stop()
			if err != nil {
				return fatal("%s", core.TranscriptErrorText(videoID, err))
			}

			if strings.HasPrefix(result, "No matches found") {
				fmt.Fprintln(os.Stderr, result)
				os.Exit(1)
			}
			fmt.Fprintln(cmd.OutOrStdout(), result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "en", "language code")

	return cmd
}
