package core

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/ulikunitz/xz"
)

// ffmpegPrewarmTimeout bounds our own ffmpeg archive download, well above
// go-ytdlp's internal hardcoded 30s timeout (see docs/BUGS.md BUG-002) —
// the archive is ~170MB, which comfortably exceeds 30s on anything but a
// fast connection.
const ffmpegPrewarmTimeout = 5 * time.Minute

type ffmpegPrewarmCfg struct {
	url         string
	archiveType string // "zip" or "tar.xz"
	binaryName  string // filename go-ytdlp's resolveExecutable looks for in its cache dir
}

// ffmpegPrewarmConfig returns the download URL and archive/binary details
// for pre-warming go-ytdlp's ffmpeg cache on the given platform, mirroring
// go-ytdlp v1.3.5's own (unexported) ffmpegBinConfigs table for the
// entries we support. ok is false for platforms we don't implement
// pre-warming for — callers fall back to go-ytdlp's own (retried) install
// path in that case.
func ffmpegPrewarmConfig(goos, goarch string) (cfg ffmpegPrewarmCfg, ok bool) {
	switch {
	case goos == "windows" && goarch == "amd64":
		return ffmpegPrewarmCfg{
			url:         "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip",
			archiveType: "zip",
			binaryName:  "ffmpeg.exe",
		}, true
	case goos == "linux" && goarch == "amd64":
		return ffmpegPrewarmCfg{
			url:         "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz",
			archiveType: "tar.xz",
			binaryName:  "ffmpeg",
		}, true
	default:
		return ffmpegPrewarmCfg{}, false
	}
}

// entryMatchesBinary reports whether an archive entry name refers to the
// target binary, matching go-ytdlp's own extraction logic: an exact name
// match, or the target appearing as the final path segment after "/" or
// "\" (archives commonly nest binaries under e.g. "<build>/bin/ffmpeg.exe").
func entryMatchesBinary(entryName, target string) bool {
	return entryName == target ||
		strings.HasSuffix(entryName, "/"+target) ||
		strings.HasSuffix(entryName, "\\"+target)
}

// prewarmFFmpegCache downloads and extracts ffmpeg into go-ytdlp's own
// expected cache location (GetCacheDir()/<binaryName>) using a longer
// timeout than go-ytdlp's internal downloader allows, so that a subsequent
// ytdlp.InstallFFmpeg call finds it via its cache-directory check and never
// attempts (and times out on) its own download. Returns an error — never
// fatal to the caller — if the platform isn't supported or the download/
// extraction fails; EnsureYtDlp treats this as best-effort.
func prewarmFFmpegCache(ctx context.Context, goos, goarch string) error {
	cfg, ok := ffmpegPrewarmConfig(goos, goarch)
	if !ok {
		return fmt.Errorf("ffmpeg cache pre-warm not implemented for %s/%s", goos, goarch)
	}

	cacheDir, err := ytdlp.GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	destPath := filepath.Join(cacheDir, cfg.binaryName)
	if info, err := os.Stat(destPath); err == nil && !info.IsDir() {
		return nil // already cached, nothing to do
	}

	dlCtx, cancel := context.WithTimeout(ctx, ffmpegPrewarmTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, cfg.url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading ffmpeg archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ffmpeg archive download failed: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "go-ytdlp-ffmpeg-archive-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("saving ffmpeg archive: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	switch cfg.archiveType {
	case "zip":
		return extractFromZip(tmpPath, cfg.binaryName, destPath)
	case "tar.xz":
		return extractFromTarXz(tmpPath, cfg.binaryName, destPath)
	default:
		return fmt.Errorf("unsupported archive type %q", cfg.archiveType)
	}
}

func extractFromZip(archivePath, binaryName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if !entryMatchesBinary(f.Name, binaryName) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, rc); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("no entry matching %q found in zip archive", binaryName)
}

func extractFromTarXz(archivePath, binaryName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	xr, err := xz.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(xr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || !entryMatchesBinary(hdr.Name, binaryName) {
			continue
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, tr); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("no entry matching %q found in tar.xz archive", binaryName)
}
