package core

import (
	"context"
	"errors"
	"testing"

	"github.com/lrstanley/go-ytdlp"
)

// installFFmpegWithRetry and requireFFmpeg have no upstream TS equivalent —
// they're new logic added to fix docs/BUGS.md BUG-002 (go-ytdlp's
// InstallFFmpeg uses a hardcoded 30s HTTP timeout sized for the much
// smaller yt-dlp binary, which can be too short for ffmpeg's ~80-100MB
// archive; failures were also being silently swallowed). Ground truth here
// is the fix's own specification.

func TestInstallFFmpegWithRetry_SucceedsAfterTransientFailures(t *testing.T) {
	attempts := 0
	fake := func(_ context.Context, _ *ytdlp.InstallFFmpegOptions) (*ytdlp.ResolvedInstall, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("transient network error")
		}
		return &ytdlp.ResolvedInstall{Executable: "/fake/ffmpeg"}, nil
	}

	resolved, err := installFFmpegWithRetry(context.Background(), 3, fake)
	if err != nil {
		t.Fatalf("installFFmpegWithRetry() error = %v, want nil", err)
	}
	if resolved == nil || resolved.Executable != "/fake/ffmpeg" {
		t.Errorf("installFFmpegWithRetry() = %+v, want Executable /fake/ffmpeg", resolved)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestInstallFFmpegWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	attempts := 0
	wantErr := errors.New("persistent failure")
	fake := func(_ context.Context, _ *ytdlp.InstallFFmpegOptions) (*ytdlp.ResolvedInstall, error) {
		attempts++
		return nil, wantErr
	}

	_, err := installFFmpegWithRetry(context.Background(), 3, fake)
	if !errors.Is(err, wantErr) {
		t.Errorf("installFFmpegWithRetry() error = %v, want %v", err, wantErr)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want exactly 3 (bounded, no more, no less)", attempts)
	}
}

func TestInstallFFmpegWithRetry_SucceedsFirstTry(t *testing.T) {
	attempts := 0
	fake := func(_ context.Context, _ *ytdlp.InstallFFmpegOptions) (*ytdlp.ResolvedInstall, error) {
		attempts++
		return &ytdlp.ResolvedInstall{Executable: "/fake/ffmpeg"}, nil
	}

	_, err := installFFmpegWithRetry(context.Background(), 3, fake)
	if err != nil {
		t.Fatalf("installFFmpegWithRetry() error = %v, want nil", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry after success)", attempts)
	}
}

func TestRequireFFmpeg(t *testing.T) {
	if err := requireFFmpeg("/path/to/ffmpeg"); err != nil {
		t.Errorf("requireFFmpeg(non-empty) = %v, want nil", err)
	}
	if err := requireFFmpeg(""); err == nil {
		t.Error("requireFFmpeg(\"\") = nil, want an error")
	}
}
