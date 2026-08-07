package cli

import (
	"fmt"
	"os"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

func newTranscriptCommand() *cobra.Command {
	var language string
	var timestamps bool
	var save bool

	cmd := &cobra.Command{
		Use:   "transcript <url>",
		Short: "Print transcript of a YouTube video",
		Args:  cobra.ExactArgs(1),
		Example: `  youtube-cli transcript dQw4w9WgXcQ
  youtube-cli transcript dQw4w9WgXcQ --timestamps
  youtube-cli transcript dQw4w9WgXcQ --save
  youtube-cli transcript dQw4w9WgXcQ --language it`,
		RunE: func(cmd *cobra.Command, args []string) error {
			videoID := core.ExtractVideoID(args[0])
			if videoID == "" {
				return fatal("invalid YouTube URL or video ID: %q", args[0])
			}

			if save {
				dir := core.GetDownloadsDir()
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fatal("%s", err.Error())
				}
				stop := spinner("Saving transcript", quietFlag)
				filePath, err := core.SaveTranscriptFile(cmd.Context(), videoID, language, dir, timestamps)
				stop()
				if err != nil {
					return fatal("%s", core.TranscriptErrorText(videoID, err))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Saved to: %s\n", filePath)
				return nil
			}

			stop := spinner("Fetching transcript", quietFlag)
			var text string
			var err error
			if timestamps {
				text, err = core.GetTranscriptTimed(cmd.Context(), videoID, language)
			} else {
				text, err = core.GetTranscriptText(cmd.Context(), videoID, language)
			}
			stop()
			if err != nil {
				return fatal("%s", core.TranscriptErrorText(videoID, err))
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return nil
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "en", "language code")
	cmd.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "include [MM:SS] timestamps")
	cmd.Flags().BoolVarP(&save, "save", "s", false, "save as .md file to Downloads instead of printing")

	return cmd
}
