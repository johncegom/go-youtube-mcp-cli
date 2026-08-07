package core

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// ffmpegPrewarmConfig, entryMatchesBinary, and extractFromZip have no
// upstream TS equivalent — new logic added to fix docs/BUGS.md BUG-002
// (go-ytdlp's InstallFFmpeg uses a hardcoded 30s HTTP timeout that's too
// short for ffmpeg's ~170MB archive on slower connections; this pre-warms
// go-ytdlp's own expected cache location using our own, more generous
// timeout, so InstallFFmpeg finds it locally and never has to download).
// Ground truth here is the fix's own specification, and — where structural
// facts about go-ytdlp's own cache layout matter (e.g. GetCacheDir, the
// exact binary filenames it looks for) — the go-ytdlp v1.3.5 source itself
// (install.go's resolveExecutable, install_ffmpeg.go's ffmpegBinConfigs).

func TestFfmpegPrewarmConfig(t *testing.T) {
	cases := []struct {
		goos, goarch string
		wantOK       bool
		wantBinary   string
		wantArchive  string
	}{
		{"windows", "amd64", true, "ffmpeg.exe", "zip"},
		{"linux", "amd64", true, "ffmpeg", "tar.xz"},
		{"darwin", "amd64", false, "", ""},
		{"windows", "arm64", false, "", ""},
		{"freebsd", "amd64", false, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.goos+"_"+tc.goarch, func(t *testing.T) {
			cfg, ok := ffmpegPrewarmConfig(tc.goos, tc.goarch)
			if ok != tc.wantOK {
				t.Fatalf("ffmpegPrewarmConfig(%q, %q) ok = %v, want %v", tc.goos, tc.goarch, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if cfg.binaryName != tc.wantBinary {
				t.Errorf("binaryName = %q, want %q", cfg.binaryName, tc.wantBinary)
			}
			if cfg.archiveType != tc.wantArchive {
				t.Errorf("archiveType = %q, want %q", cfg.archiveType, tc.wantArchive)
			}
			if cfg.url == "" {
				t.Error("url is empty")
			}
		})
	}
}

func TestEntryMatchesBinary(t *testing.T) {
	cases := []struct {
		entryName string
		target    string
		want      bool
	}{
		{"ffmpeg-master-latest-win64-gpl/bin/ffmpeg.exe", "ffmpeg.exe", true},
		{"ffmpeg-master-latest-win64-gpl\\bin\\ffmpeg.exe", "ffmpeg.exe", true},
		{"ffmpeg.exe", "ffmpeg.exe", true},
		{"ffmpeg-master-latest-win64-gpl/bin/ffprobe.exe", "ffmpeg.exe", false},
		{"ffmpeg-master-latest-win64-gpl/bin/ffplay.exe", "ffmpeg.exe", false},
		{"some/deep/path/notffmpeg.exe", "ffmpeg.exe", false},
	}
	for _, tc := range cases {
		if got := entryMatchesBinary(tc.entryName, tc.target); got != tc.want {
			t.Errorf("entryMatchesBinary(%q, %q) = %v, want %v", tc.entryName, tc.target, got, tc.want)
		}
	}
}

func TestExtractFromZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "archive.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"ffmpeg-build/bin/ffprobe.exe", "ffmpeg-build/bin/ffmpeg.exe", "ffmpeg-build/README.txt"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		content := "fake content for " + name
		if name == "ffmpeg-build/bin/ffmpeg.exe" {
			content = "this is the real ffmpeg binary"
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	destPath := filepath.Join(dir, "ffmpeg.exe")
	if err := extractFromZip(zipPath, "ffmpeg.exe", destPath); err != nil {
		t.Fatalf("extractFromZip() error = %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(got) != "this is the real ffmpeg binary" {
		t.Errorf("extracted content = %q, want the ffmpeg.exe entry's content", string(got))
	}
}

func TestExtractFromZip_NoMatchingEntry(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "archive.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("some/other/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("irrelevant"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = extractFromZip(zipPath, "ffmpeg.exe", filepath.Join(dir, "ffmpeg.exe"))
	if err == nil {
		t.Error("extractFromZip() with no matching entry: error = nil, want an error")
	}
}
