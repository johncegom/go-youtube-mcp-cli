package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

var qualityFormatMap = map[string]string{
	"best":   "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1]+bestaudio/best[ext=mp4]/best",
	"hd1080": "bestvideo[vcodec^=avc1][height<=1080]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=1080]+bestaudio/best[height<=1080][ext=mp4]/best[height<=1080]",
	"hd720":  "bestvideo[vcodec^=avc1][height<=720]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=720]+bestaudio/best[height<=720][ext=mp4]/best[height<=720]",
	"sd480":  "bestvideo[vcodec^=avc1][height<=480]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=480]+bestaudio/best[height<=480][ext=mp4]/best[height<=480]",
	"sd360":  "bestvideo[vcodec^=avc1][height<=360]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=360]+bestaudio/best[height<=360][ext=mp4]/best[height<=360]",
}

// qualityFormat resolves a quality preset name to a yt-dlp format-selector
// string, falling back to the hd720 preset for unknown/empty names (mirrors
// the TS `QUALITY_FORMAT_MAP[quality] ?? QUALITY_FORMAT_MAP.hd720`).
func qualityFormat(quality string) string {
	if f, ok := qualityFormatMap[quality]; ok {
		return f
	}
	return qualityFormatMap["hd720"]
}

func formatVideoDownloadStarted(title, predictedPath string) string {
	return fmt.Sprintf("Download started:\nTitle: %s\nThe file will appear at: %s (extension may differ if H.264 is unavailable)\nIt may take a while for long videos.", title, predictedPath)
}

func formatAudioDownloadStarted(title, predictedPath string) string {
	return fmt.Sprintf("Download started:\nTitle: %s\nThe file will appear at: %s\nIt may take a while for long videos.", title, predictedPath)
}

// resolveTitle fetches metadata for a display title and its filename-safe
// counterpart, falling back to the bare video ID if metadata fetch fails.
func resolveTitle(ctx context.Context, videoID string) (title, safeTitle string) {
	meta, err := FetchVideoMetadata(ctx, videoID)
	title = videoID
	if err == nil && meta["title"] != "" {
		title = meta["title"]
	}
	return title, SanitizeTitle(title)
}

func videoURL(videoID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
}

// StartVideoDownload kicks off a video download in the background and
// returns immediately with the predicted output path (fire-and-forget,
// used by the MCP server — mirrors the TS execFile(...).unref() pattern).
// The download itself runs detached from ctx so it survives past the
// request that started it; only the title-resolution step honors ctx.
func StartVideoDownload(ctx context.Context, videoID, quality, outputDir string) (string, error) {
	if err := EnsureYtDlp(ctx); err != nil {
		return "", err
	}
	if ffmpegPath == "" {
		fmt.Fprintln(os.Stderr, "warning: ffmpeg is not available; video and audio streams will not be merged into a single file")
	}

	title, safeTitle := resolveTitle(ctx, videoID)
	predictedPath := filepath.Join(outputDir, safeTitle+".mp4")

	cmd := NewYtDlpCommand().
		Output(filepath.Join(outputDir, safeTitle+".%(ext)s")).
		NoWarnings().
		MergeOutputFormat("mp4").
		Format(qualityFormat(quality))

	go func() {
		if _, err := cmd.Run(context.Background(), videoURL(videoID)); err != nil {
			msg := err.Error()
			fmt.Fprintf(os.Stderr, "yt-dlp error: %s\n", msg)
			LogDownloadError(fmt.Sprintf("download_video %s", videoID), msg)
		}
	}()

	return formatVideoDownloadStarted(title, predictedPath), nil
}

// StartAudioDownload is the audio counterpart to StartVideoDownload.
func StartAudioDownload(ctx context.Context, videoID, audioFormat, outputDir string) (string, error) {
	if err := EnsureYtDlp(ctx); err != nil {
		return "", err
	}
	if err := requireFFmpeg(ffmpegPath); err != nil {
		return "", err
	}

	title, safeTitle := resolveTitle(ctx, videoID)
	predictedPath := filepath.Join(outputDir, safeTitle+"."+audioFormat)

	cmd := NewYtDlpCommand().
		Output(filepath.Join(outputDir, safeTitle+".%(ext)s")).
		NoWarnings().
		ExtractAudio().
		AudioFormat(audioFormat).
		Format("bestaudio/best")

	go func() {
		if _, err := cmd.Run(context.Background(), videoURL(videoID)); err != nil {
			msg := err.Error()
			fmt.Fprintf(os.Stderr, "yt-dlp error: %s\n", msg)
			LogDownloadError(fmt.Sprintf("download_audio %s", videoID), msg)
		}
	}()

	return formatAudioDownloadStarted(title, predictedPath), nil
}

// DownloadVideoBlocking downloads a video and blocks until it finishes,
// streaming yt-dlp's own output to stdout/stderr (used by the CLI).
func DownloadVideoBlocking(ctx context.Context, videoID, quality, outputDir string) error {
	if err := EnsureYtDlp(ctx); err != nil {
		return err
	}
	if ffmpegPath == "" {
		fmt.Fprintln(os.Stderr, "warning: ffmpeg is not available; video and audio streams will not be merged into a single file")
	}

	builder := NewYtDlpCommand().
		Output(filepath.Join(outputDir, "%(title)s.%(ext)s")).
		MergeOutputFormat("mp4").
		Format(qualityFormat(quality))

	cmd := builder.BuildCommand(ctx, videoURL(videoID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DownloadAudioBlocking is the audio counterpart to DownloadVideoBlocking.
func DownloadAudioBlocking(ctx context.Context, videoID, audioFormat, outputDir string) error {
	if err := EnsureYtDlp(ctx); err != nil {
		return err
	}
	if err := requireFFmpeg(ffmpegPath); err != nil {
		return err
	}

	builder := NewYtDlpCommand().
		Output(filepath.Join(outputDir, "%(title)s.%(ext)s")).
		ExtractAudio().
		AudioFormat(audioFormat).
		Format("bestaudio/best")

	cmd := builder.BuildCommand(ctx, videoURL(videoID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
