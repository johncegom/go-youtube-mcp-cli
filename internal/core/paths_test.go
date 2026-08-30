package core

// paths.go has no upstream TS equivalent (DECISION-002: LogDownloadError
// deliberately diverges from the TS original's hardcoded ~/.cache path to
// use os.UserCacheDir() instead), so ground truth here is the function's
// own documented contract, not a ported TS behavior.
//
// Writing these tests surfaced BUG-004 (docs/BUGS.md): pathStartsWith does
// a string-prefix compare, not a path-segment compare, so a sibling
// directory whose name happens to extend the allowed root's last path
// component (e.g. an allowed root of ".../Temp" and a candidate of
// ".../TempXYZ/evil") is incorrectly treated as inside the allowed root.
// TestPathStartsWith_SegmentBoundaryBug pins down this *current* (buggy)
// behavior rather than silently fixing it, per the project's bug-tracking
// process — flip its assertion once BUG-004 is fixed.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDownloadsDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available in this environment: %v", err)
	}

	if runtime.GOOS == "linux" {
		t.Run("xdg_absolute_used", func(t *testing.T) {
			t.Setenv("XDG_DOWNLOAD_DIR", "/custom/downloads")
			if got := GetDownloadsDir(); got != "/custom/downloads" {
				t.Errorf("GetDownloadsDir() = %q, want %q", got, "/custom/downloads")
			}
		})
		t.Run("xdg_relative_ignored", func(t *testing.T) {
			t.Setenv("XDG_DOWNLOAD_DIR", "relative/downloads")
			want := filepath.Join(home, "Downloads")
			if got := GetDownloadsDir(); got != want {
				t.Errorf("GetDownloadsDir() = %q, want %q (relative XDG_DOWNLOAD_DIR should be ignored)", got, want)
			}
		})
		t.Run("xdg_unset_falls_back_to_home", func(t *testing.T) {
			t.Setenv("XDG_DOWNLOAD_DIR", "")
			want := filepath.Join(home, "Downloads")
			if got := GetDownloadsDir(); got != want {
				t.Errorf("GetDownloadsDir() = %q, want %q", got, want)
			}
		})
		return
	}

	// Non-Linux (including this dev environment's Windows and CI's
	// windows-latest runner): XDG_DOWNLOAD_DIR must be ignored entirely.
	t.Setenv("XDG_DOWNLOAD_DIR", "/should/be/ignored")
	want := filepath.Join(home, "Downloads")
	if got := GetDownloadsDir(); got != want {
		t.Errorf("GetDownloadsDir() = %q, want %q (XDG_DOWNLOAD_DIR must be ignored on %s)", got, want, runtime.GOOS)
	}
}

func TestResolveOutputDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available in this environment: %v", err)
	}

	t.Run("empty_defaults_to_downloads", func(t *testing.T) {
		want := GetDownloadsDir()
		if got := ResolveOutputDir(""); got != want {
			t.Errorf("ResolveOutputDir(\"\") = %q, want %q", got, want)
		}
	})

	t.Run("subdir_of_temp_allowed", func(t *testing.T) {
		dir := filepath.Join(os.TempDir(), "youtube-mcp-test-subdir")
		if got := ResolveOutputDir(dir); got != dir {
			t.Errorf("ResolveOutputDir(%q) = %q, want %q", dir, got, dir)
		}
	})

	t.Run("subdir_of_home_allowed", func(t *testing.T) {
		dir := filepath.Join(home, "youtube-mcp-test-subdir")
		if got := ResolveOutputDir(dir); got != dir {
			t.Errorf("ResolveOutputDir(%q) = %q, want %q", dir, got, dir)
		}
	})

	t.Run("relative_path_resolved_and_checked", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd: %v", err)
		}
		if !pathStartsWith(filepath.Clean(cwd), filepath.Clean(os.TempDir())) &&
			!pathStartsWith(filepath.Clean(cwd), filepath.Clean(home)) {
			t.Skip("working directory isn't under an allowed root in this environment")
		}
		want, err := filepath.Abs("relative-subdir")
		if err != nil {
			t.Fatalf("filepath.Abs: %v", err)
		}
		if got := ResolveOutputDir("relative-subdir"); got != want {
			t.Errorf("ResolveOutputDir(%q) = %q, want %q", "relative-subdir", got, want)
		}
	})

	t.Run("outside_all_roots_rejected", func(t *testing.T) {
		var outside string
		switch runtime.GOOS {
		case "windows":
			outside = `C:\definitely-unrelated-root-xyz\evil`
		default:
			outside = "/definitely-unrelated-root-xyz/evil"
		}
		if pathStartsWith(filepath.Clean(outside), filepath.Clean(os.TempDir())) ||
			pathStartsWith(filepath.Clean(outside), filepath.Clean(home)) {
			t.Skipf("chosen probe path %q unexpectedly falls under an allowed root in this environment", outside)
		}
		if got := ResolveOutputDir(outside); got != "" {
			t.Errorf("ResolveOutputDir(%q) = %q, want \"\" (outside all allowed roots)", outside, got)
		}
	})
}

func TestPathStartsWith(t *testing.T) {
	t.Run("exact_match", func(t *testing.T) {
		if !pathStartsWith(os.TempDir(), os.TempDir()) {
			t.Error("expected exact match to start with itself")
		}
	})

	t.Run("real_subdir", func(t *testing.T) {
		child := filepath.Join(os.TempDir(), "sub", "dir")
		if !pathStartsWith(child, os.TempDir()) {
			t.Errorf("expected %q to start with %q", child, os.TempDir())
		}
	})

	if runtime.GOOS == "windows" {
		t.Run("windows_case_insensitive", func(t *testing.T) {
			parent := os.TempDir()
			child := filepath.Join(strings.ToUpper(parent), "sub")
			if !pathStartsWith(child, parent) {
				t.Errorf("expected case-insensitive match on windows: %q vs %q", child, parent)
			}
		})
	}
}

// TestPathStartsWith_SegmentBoundaryBug documents BUG-004 (docs/BUGS.md):
// pathStartsWith does a raw string-prefix compare, so a sibling directory
// whose name merely extends the allowed root's final path segment is
// incorrectly treated as a child of that root. This test asserts the
// *current* (buggy) behavior — flip it to want=false once BUG-004 is fixed.
func TestPathStartsWith_SegmentBoundaryBug(t *testing.T) {
	parent := filepath.Clean(os.TempDir())
	sibling := parent + "XYZ" + string(filepath.Separator) + "evil"

	got := pathStartsWith(sibling, parent)
	if !got {
		t.Fatal("expected BUG-004's string-prefix bug to currently misclassify a sibling directory as a child; " +
			"if this now fails, BUG-004 has been fixed and this assertion should be flipped to false")
	}
}

func TestLogDownloadError(t *testing.T) {
	tmp := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", tmp)
	case "darwin":
		t.Setenv("HOME", tmp)
	default:
		t.Setenv("XDG_CACHE_HOME", tmp)
	}

	LogDownloadError("test-context", "first message")
	LogDownloadError("test-context", "second message")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("os.UserCacheDir: %v", err)
	}
	logFile := filepath.Join(cacheDir, "youtube-mcp", "errors.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading log file %q: %v", logFile, err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2: %q", len(lines), string(data))
	}
	for i, want := range []string{"first message", "second message"} {
		if !strings.Contains(lines[i], "test-context: "+want) {
			t.Errorf("line %d = %q, want it to contain %q", i, lines[i], "test-context: "+want)
		}
		if !strings.HasPrefix(lines[i], "[") {
			t.Errorf("line %d = %q, want a leading [timestamp]", i, lines[i])
		}
	}
}
