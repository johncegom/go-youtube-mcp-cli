package cli

import (
	"fmt"
	"os"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

func newDownloadCommand() *cobra.Command {
	var audio bool
	var quality string
	var format string

	cmd := &cobra.Command{
		Use:   "download <url>",
		Short: "Download video or audio to Downloads folder",
		Args:  cobra.ExactArgs(1),
		Example: `  youtube-cli download dQw4w9WgXcQ
  youtube-cli download dQw4w9WgXcQ --quality hd1080
  youtube-cli download dQw4w9WgXcQ --audio
  youtube-cli download dQw4w9WgXcQ --audio --format flac`,
		RunE: func(cmd *cobra.Command, args []string) error {
			videoID := core.ExtractVideoID(args[0])
			if videoID == "" {
				return fatal("invalid YouTube URL or video ID: %q", args[0])
			}

			dir := core.GetDownloadsDir()
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fatal("%s", err.Error())
			}
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "Saving to %s\n", dir)
			}

			var err error
			if audio {
				err = core.DownloadAudioBlocking(cmd.Context(), videoID, format, dir)
			} else {
				err = core.DownloadVideoBlocking(cmd.Context(), videoID, quality, dir)
			}
			if err != nil {
				return fatal("%s", err.Error())
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&audio, "audio", "a", false, "download audio only")
	cmd.Flags().StringVarP(&quality, "quality", "q", "hd720", "video quality: best, hd1080, hd720, sd480, sd360")
	cmd.Flags().StringVarP(&format, "format", "f", "mp3", "audio format: mp3, m4a, aac, flac, opus, wav, vorbis")

	return cmd
}
