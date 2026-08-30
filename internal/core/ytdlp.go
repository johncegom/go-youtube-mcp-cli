package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/lrstanley/go-ytdlp"
)

var (
	installOnce sync.Once
	installErr  error
	ffmpegPath  string
)

// ffmpegInstallAttempts bounds installFFmpegWithRetry's retries. ffmpeg's
// archive (~80-100MB) is downloaded through go-ytdlp's shared downloadFile
// helper, which uses the same hardcoded 30s HTTP client timeout sized for
// the much smaller yt-dlp binary (see docs/BUGS.md BUG-002) — a transient
// slow patch on that budget can fail one attempt without meaning the
// network is actually unusable, so a bounded retry is worth it.
const ffmpegInstallAttempts = 3

// installFFmpegWithRetry retries installFn up to attempts times, returning
// as soon as one succeeds. installFn is injected so this is unit-testable
// without touching the network (see ytdlp_test.go).
func installFFmpegWithRetry(
	ctx context.Context,
	attempts int,
	installFn func(context.Context, *ytdlp.InstallFFmpegOptions) (*ytdlp.ResolvedInstall, error),
) (*ytdlp.ResolvedInstall, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		resolved, err := installFn(ctx, nil)
		if err == nil {
			return resolved, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// requireFFmpeg returns a clear, actionable error if ffmpegPath is empty
// (i.e. ffmpeg could not be resolved/installed), instead of letting the
// caller find out via a confusing failure deep inside yt-dlp's own
// postprocessing step. See docs/BUGS.md BUG-002.
func requireFFmpeg(ffmpegPath string) error {
	if ffmpegPath != "" {
		return nil
	}
	return errors.New("ffmpeg is required for this operation but is not available " +
		"(auto-install may have failed; check the error log, or for faster/more " +
		"reliable startup, install ffmpeg yourself and ensure it's on PATH)")
}

// EnsureYtDlp makes sure a yt-dlp binary is installed and resolvable,
// downloading it (and ffmpeg, best-effort) into the OS cache dir on first
// use if neither is already present on PATH or in go-ytdlp's cache. Safe to
// call repeatedly; the actual install work runs once per process.
func EnsureYtDlp(ctx context.Context) error {
	installOnce.Do(func() {
		// AllowVersionMismatch: accept whatever yt-dlp is already resolved
		// (cache or PATH) even if its version doesn't match go-ytdlp's
		// pinned Version const, instead of silently re-downloading and
		// overwriting it back down to that (often stale) pinned version —
		// see docs/BUGS.md BUG-006.
		if _, err := ytdlp.Install(ctx, &ytdlp.InstallOptions{AllowVersionMismatch: true}); err != nil {
			installErr = fmt.Errorf("yt-dlp binary not found and auto-install failed: %w", err)
			return
		}
		// ffmpeg is optional: downloads/merges of separate video+audio
		// streams need it, but format selection can still succeed without
		// it for some formats, so a failure here is not fatal to
		// EnsureYtDlp itself — callers that specifically require ffmpeg
		// (e.g. audio extraction) check via requireFFmpeg. The failure is
		// still surfaced (not swallowed) so it's visible instead of only
		// showing up later as a confusing downstream error.

		// Check PATH/cache only first (no network) — if ffmpeg is already
		// available, skip straight past all of the below.
		if resolved, err := ytdlp.InstallFFmpeg(ctx, &ytdlp.InstallFFmpegOptions{DisableDownload: true}); err == nil {
			ffmpegPath = resolved.Executable
			return
		}

		fmt.Fprintln(os.Stderr, "ffmpeg not found locally; downloading now (one-time, "+
			"~170MB, can take a while on slower connections). For faster/more reliable "+
			"startup, consider installing ffmpeg yourself and ensuring it's on PATH.")

		// Best-effort: pre-warm go-ytdlp's own expected cache location
		// ourselves, using a longer timeout than go-ytdlp's internal
		// downloader allows (see docs/BUGS.md BUG-002). Only implemented
		// for a couple of platforms (ffmpegPrewarmConfig); if it's not
		// supported here, or it fails, we still fall through to
		// go-ytdlp's own retried install below as a fallback.
		if err := prewarmFFmpegCache(ctx, runtime.GOOS, runtime.GOARCH); err != nil {
			LogDownloadError("ffmpeg cache pre-warm", err.Error())
		}

		resolved, err := installFFmpegWithRetry(ctx, ffmpegInstallAttempts, ytdlp.InstallFFmpeg)
		if err != nil {
			msg := err.Error()
			fmt.Fprintf(os.Stderr, "warning: ffmpeg auto-install failed after %d attempts: %s\n"+
				"You can install ffmpeg manually and ensure it's on PATH to avoid this next time.\n",
				ffmpegInstallAttempts, msg)
			LogDownloadError("ffmpeg auto-install", msg)
			return
		}
		ffmpegPath = resolved.Executable
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
