package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type jobKind string

const (
	jobKindVideo jobKind = "video"
	jobKindAudio jobKind = "audio"
)

type jobState string

const (
	jobStateRunning jobState = "running"
	jobStateDone    jobState = "done"
	jobStateFailed  jobState = "failed"
)

// downloadJob is a snapshot of one StartVideoDownload/StartAudioDownload
// invocation's lifecycle, tracked so an MCP client can later ask "did it
// work, and where's the file?" (see docs/tasks/12-download-jobs/TASK.md).
type downloadJob struct {
	ID            string
	Kind          jobKind
	VideoID       string
	State         jobState
	PredictedPath string
	ActualPath    string
	Err           string
	StartedAt     time.Time
	FinishedAt    time.Time
}

// jobRegistry is a mutex-guarded, capped, in-memory (process-lifetime only)
// store of download jobs. Oldest jobs are evicted once the cap is exceeded
// so list_downloads/memory stay bounded across a long-lived server process.
type jobRegistry struct {
	mu      sync.Mutex
	jobs    map[string]*downloadJob
	order   []string // insertion order, oldest first, for eviction
	cap     int
	counter atomic.Int64
}

func newJobRegistry(cap int) *jobRegistry {
	if cap < 0 {
		cap = 0
	}
	return &jobRegistry{jobs: make(map[string]*downloadJob), cap: cap}
}

// defaultJobRegistry is the process-wide registry used by
// StartVideoDownload/StartAudioDownload and the get_download_status/
// list_downloads MCP tools. 100-job history cap.
var defaultJobRegistry = newJobRegistry(100)

// register creates and stores a new running job, returning its ID.
func (r *jobRegistry) register(kind jobKind, videoID, predictedPath string, startedAt time.Time) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("dl-%d", r.counter.Add(1))
	r.jobs[id] = &downloadJob{
		ID:            id,
		Kind:          kind,
		VideoID:       videoID,
		State:         jobStateRunning,
		PredictedPath: predictedPath,
		StartedAt:     startedAt,
	}
	r.order = append(r.order, id)
	for len(r.order) > r.cap {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.jobs, oldest)
	}
	return id
}

// succeed and fail are no-ops if id has since been evicted (only possible
// once more than cap jobs have been registered) — there is no one left to
// report the outcome to.
func (r *jobRegistry) succeed(id, actualPath string, finishedAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job, ok := r.jobs[id]; ok {
		job.State = jobStateDone
		job.ActualPath = actualPath
		job.FinishedAt = finishedAt
	}
}

func (r *jobRegistry) fail(id, errText string, finishedAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job, ok := r.jobs[id]; ok {
		job.State = jobStateFailed
		job.Err = errText
		job.FinishedAt = finishedAt
	}
}

func (r *jobRegistry) get(id string) (downloadJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return downloadJob{}, false
	}
	return *job, true
}

func (r *jobRegistry) list() []downloadJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]downloadJob, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, *r.jobs[id])
	}
	return out
}

// finalPathFromListing finds the actual output filename for a completed
// download by matching safeTitle against a directory listing: yt-dlp's
// chosen extension may differ from the prediction (e.g. mkv instead of mp4
// if H.264 wasn't available — see formatVideoDownloadStarted). Returns the
// matched filename (not a full path) and whether a match was found.
func finalPathFromListing(safeTitle string, filenames []string) (string, bool) {
	prefix := safeTitle + "."
	for _, name := range filenames {
		if strings.HasPrefix(name, prefix) {
			return name, true
		}
	}
	return "", false
}

// resolveActualPath resolves the real on-disk output path for a completed
// download, falling back to predictedPath annotated as unconfirmed if no
// matching file is found in outputDir (e.g. the directory listing failed,
// or yt-dlp wrote somewhere unexpected).
func resolveActualPath(outputDir, safeTitle, predictedPath string) string {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return predictedPath + " (not found)"
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	if name, ok := finalPathFromListing(safeTitle, names); ok {
		return filepath.Join(outputDir, name)
	}
	return predictedPath + " (not found)"
}

func formatJobStatus(j downloadJob) string {
	switch j.State {
	case jobStateDone:
		return fmt.Sprintf("Job %s (%s, video %s): done\nFile: %s", j.ID, j.Kind, j.VideoID, j.ActualPath)
	case jobStateFailed:
		return fmt.Sprintf("Job %s (%s, video %s): failed\nError: %s", j.ID, j.Kind, j.VideoID, j.Err)
	default:
		return fmt.Sprintf("Job %s (%s, video %s): running", j.ID, j.Kind, j.VideoID)
	}
}

func formatJobLine(j downloadJob) string {
	return fmt.Sprintf("%s [%s] %s: %s", j.ID, j.Kind, j.VideoID, j.State)
}

// FormatDownloadStatus returns a human-readable status report for jobID, or
// ("", false) if no job with that ID is known (including one evicted by the
// history cap).
func FormatDownloadStatus(jobID string) (string, bool) {
	job, ok := defaultJobRegistry.get(jobID)
	if !ok {
		return "", false
	}
	return formatJobStatus(job), true
}

// FormatDownloadsList returns one line per known job (most recent last).
func FormatDownloadsList() string {
	jobs := defaultJobRegistry.list()
	if len(jobs) == 0 {
		return "No downloads yet."
	}
	lines := make([]string, len(jobs))
	for i, j := range jobs {
		lines[i] = formatJobLine(j)
	}
	return strings.Join(lines, "\n")
}
