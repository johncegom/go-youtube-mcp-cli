package cli

import (
	"encoding/json"
	"fmt"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/spf13/cobra"
)

func newMetadataCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "metadata <url>",
		Short: "Print title, channel, views, duration, and description",
		Args:  cobra.ExactArgs(1),
		Example: `  youtube-cli metadata dQw4w9WgXcQ
  youtube-cli metadata dQw4w9WgXcQ --json
  youtube-cli metadata dQw4w9WgXcQ --json | jq '.title'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			videoID := core.ExtractVideoID(args[0])
			if videoID == "" {
				return fatal("invalid YouTube URL or video ID: %q", args[0])
			}

			stop := spinner("Fetching metadata", quietFlag)
			meta, err := core.FetchVideoMetadata(cmd.Context(), videoID)
			stop()
			if err != nil {
				return fatal("failed to fetch metadata: %s", err.Error())
			}

			out := cmd.OutOrStdout()
			if jsonOutput {
				b, err := json.MarshalIndent(meta, "", "  ")
				if err != nil {
					return fatal("%s", err.Error())
				}
				fmt.Fprintln(out, string(b))
				return nil
			}

			if meta["title"] != "" {
				fmt.Fprintf(out, "Title:       %s\n", meta["title"])
			}
			if meta["channel"] != "" {
				fmt.Fprintf(out, "Channel:     %s\n", meta["channel"])
			}
			if meta["publishDate"] != "" {
				fmt.Fprintf(out, "Published:   %s\n", meta["publishDate"])
			}
			if meta["viewCount"] != "" {
				fmt.Fprintf(out, "Views:       %s\n", meta["viewCount"])
			}
			if meta["duration"] != "" {
				fmt.Fprintf(out, "Duration:    %s\n", meta["duration"])
			}
			if meta["channelUrl"] != "" {
				fmt.Fprintf(out, "Channel URL: %s\n", meta["channelUrl"])
			}
			if meta["channelId"] != "" {
				fmt.Fprintf(out, "Channel ID:  %s\n", meta["channelId"])
			}
			if meta["description"] != "" {
				fmt.Fprintf(out, "\nDescription:\n%s\n", meta["description"])
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")

	return cmd
}
