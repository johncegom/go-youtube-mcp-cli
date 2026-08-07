package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var allowedOutputRoots = func() []string {
	roots := []string{os.TempDir()}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, home)
	}
	return roots
}()

// GetDownloadsDir returns the user's Downloads folder, honoring
// $XDG_DOWNLOAD_DIR on Linux when set to an absolute path.
func GetDownloadsDir() string {
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_DOWNLOAD_DIR"); xdg != "" && filepath.IsAbs(xdg) {
			return xdg
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "Downloads")
	}
	return filepath.Join(home, "Downloads")
}

func pathStartsWith(child, parent string) bool {
	normalised := filepath.Clean(parent)
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(child), strings.ToLower(normalised))
	}
	return strings.HasPrefix(child, normalised)
}

// ResolveOutputDir resolves rawDir to an absolute path (defaulting to the
// Downloads folder when empty) and verifies it falls within the user's home
// or temp directory. Returns "" if the resolved path is not allowed.
func ResolveOutputDir(rawDir string) string {
	dir := rawDir
	if dir == "" {
		dir = GetDownloadsDir()
	} else if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	for _, root := range allowedOutputRoots {
		if pathStartsWith(dir, root) {
			return dir
		}
	}
	return ""
}

// LogDownloadError appends a timestamped error line to the shared
// youtube-mcp error log in the OS cache directory.
func LogDownloadError(context, msg string) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	logFile := filepath.Join(cacheDir, "youtube-mcp", "errors.log")
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s: %s\n", time.Now().UTC().Format(time.RFC3339), context, msg)
}
