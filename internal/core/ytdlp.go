package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/lrstanley/go-ytdlp"
)

var (
	installOnce sync.Once
	installErr  error
	ffmpegPath  string
)

// EnsureYtDlp makes sure a yt-dlp binary is installed and resolvable,
// downloading it (and ffmpeg, best-effort) into the OS cache dir on first
// use if neither is already present on PATH or in go-ytdlp's cache. Safe to
// call repeatedly; the actual install work runs once per process.
func EnsureYtDlp(ctx context.Context) error {
	installOnce.Do(func() {
		if _, err := ytdlp.Install(ctx, nil); err != nil {
			installErr = fmt.Errorf("yt-dlp binary not found and auto-install failed: %w", err)
			return
		}
		// ffmpeg is optional: downloads/merges of separate video+audio
		// streams need it, but format selection can still succeed without
		// it for some formats, so a failure here is not fatal.
		if resolved, err := ytdlp.InstallFFmpeg(ctx, nil); err == nil {
			ffmpegPath = resolved.Executable
		}
	})
	return installErr
}

// NewYtDlpCommand returns a yt-dlp command builder with the ffmpeg location
// pre-configured when available. Callers must call EnsureYtDlp first.
func NewYtDlpCommand() *ytdlp.Command {
	cmd := ytdlp.New()
	if ffmpegPath != "" {
		cmd = cmd.FFmpegLocation(ffmpegPath)
	}
	return cmd
}
