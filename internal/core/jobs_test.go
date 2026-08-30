package core

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Ground truth for these tests is this task's own reviewed spec
// (docs/tasks/12-download-jobs/TASK.md) — job tracking has no upstream TS
// equivalent to derive behavior from.

func TestJobRegistry_SuccessFlow(t *testing.T) {
	r := newJobRegistry(100)
	id := r.register(jobKindVideo, "abc123", "/out/video.mp4", time.Now())

	job, ok := r.get(id)
	if !ok || job.State != jobStateRunning {
		t.Fatalf("after register: job=%+v ok=%v, want running", job, ok)
	}

	r.succeed(id, "/out/video.mkv", time.Now())
	job, ok = r.get(id)
	if !ok || job.State != jobStateDone || job.ActualPath != "/out/video.mkv" {
		t.Fatalf("after succeed: job=%+v ok=%v, want done with actual path", job, ok)
	}
}

func TestJobRegistry_FailureFlow(t *testing.T) {
	r := newJobRegistry(100)
	id := r.register(jobKindAudio, "abc123", "/out/audio.mp3", time.Now())

	r.fail(id, "yt-dlp: network error", time.Now())
	job, ok := r.get(id)
	if !ok || job.State != jobStateFailed || job.Err != "yt-dlp: network error" {
		t.Fatalf("after fail: job=%+v ok=%v, want failed with captured error text", job, ok)
	}
}

func TestJobRegistry_UnknownID(t *testing.T) {
	r := newJobRegistry(100)
	if _, ok := r.get("dl-999"); ok {
		t.Error("get() on unknown ID = ok, want not found")
	}
	if _, ok := FormatDownloadStatus("dl-999-does-not-exist"); ok {
		t.Error("FormatDownloadStatus() on unknown ID = ok, want not found")
	}
	// succeed/fail on an unknown ID must be a no-op, not a panic.
	r.succeed("dl-999", "/x", time.Now())
	r.fail("dl-999", "err", time.Now())
}

func TestJobRegistry_HistoryCapEviction(t *testing.T) {
	r := newJobRegistry(3)
	var ids []string
	for range 5 {
		ids = append(ids, r.register(jobKindVideo, "vid", "/out/x.mp4", time.Now()))
	}

	jobs := r.list()
	if len(jobs) != 3 {
		t.Fatalf("list() length = %d, want 3 (cap)", len(jobs))
	}
	for _, id := range ids[:2] {
		if _, ok := r.get(id); ok {
			t.Errorf("evicted job %s still present", id)
		}
	}
	for _, id := range ids[2:] {
		if _, ok := r.get(id); !ok {
			t.Errorf("surviving job %s missing", id)
		}
	}
}

func TestJobRegistry_ConcurrentRegisterAndLookup(t *testing.T) {
	r := newJobRegistry(50)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := r.register(jobKindVideo, fmt.Sprintf("vid%d", i), "/out/x.mp4", time.Now())
			r.get(id)
			r.list()
			r.succeed(id, "/out/x.mp4", time.Now())
		}(i)
	}
	wg.Wait()
}

func TestFinalPathFromListing(t *testing.T) {
	cases := []struct {
		name      string
		safeTitle string
		filenames []string
		wantName  string
		wantOK    bool
	}{
		{"exact predicted extension", "My_Video", []string{"My_Video.mp4"}, "My_Video.mp4", true},
		{"extension differs from prediction", "My_Video", []string{"other.txt", "My_Video.mkv"}, "My_Video.mkv", true},
		{"no match", "My_Video", []string{"Unrelated.mp4"}, "", false},
		{"empty listing", "My_Video", nil, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := finalPathFromListing(tc.safeTitle, tc.filenames)
			if name != tc.wantName || ok != tc.wantOK {
				t.Errorf("finalPathFromListing(%q, %v) = (%q, %v), want (%q, %v)",
					tc.safeTitle, tc.filenames, name, ok, tc.wantName, tc.wantOK)
			}
		})
	}
}

func TestResolveActualPath_NotFoundFallback(t *testing.T) {
	dir := t.TempDir()
	got := resolveActualPath(dir, "Nonexistent_Title", dir+"/Nonexistent_Title.mp4")
	want := dir + "/Nonexistent_Title.mp4 (not found)"
	if got != want {
		t.Errorf("resolveActualPath() = %q, want %q", got, want)
	}
}

func TestFormatDownloadsList_Empty(t *testing.T) {
	r := newJobRegistry(10)
	orig := defaultJobRegistry
	defaultJobRegistry = r
	defer func() { defaultJobRegistry = orig }()

	if got, want := FormatDownloadsList(), "No downloads yet."; got != want {
		t.Errorf("FormatDownloadsList() = %q, want %q", got, want)
	}
}
