# Task 12: Download job tracking (`get_download_status`, `list_downloads`)

**Status:** not started (Definition of Done + Test Plan approved 2026-08-28,
before implementation, per the task-approval process in `CLAUDE.md`)

## User need

"I asked for a download — did it actually work, and where is the file?"

## Problem

- `StartVideoDownload`/`StartAudioDownload` (`internal/core/download.go`)
  detach a goroutine and immediately return a *predicted* path. Failures go
  to stderr and `LogDownloadError`'s log file — which no MCP client ever
  reads. The agent has no way to learn a download's outcome.
- The `download_video`/`download_audio` tool descriptions say "Returns the
  file path", which is false (it returns a prediction before the download
  runs) — an LLM reading that will confidently tell the user the file
  exists when it may not, or may land at a different extension.

## Plan

- New `internal/core/jobs.go`:
  - `jobRegistry`: mutex-guarded `map[string]*downloadJob`; job IDs are
    short opaque strings (e.g. `dl-1`, `dl-2` from an atomic counter —
    stable, human-readable, no collision risk within one process).
  - `downloadJob` fields: ID, kind (`video`/`audio`), video ID, state
    (`running` / `done` / `failed`), predicted path, actual final path
    (discovered after completion by scanning `outputDir` for
    `safeTitle.*` — handles the "extension may differ" case), captured
    error text on failure, start/finish timestamps.
  - Registry is process-lifetime, in-memory only; a capped history
    (e.g. last 100 jobs) so `list_downloads` stays bounded.
  - Pure/state-machine logic (transitions, final-path resolution from a
    file listing) factored out from the goroutine I/O so it's unit-testable
    with an injectable runner.
- `StartVideoDownload`/`StartAudioDownload` register a job, transition it
  in the detached goroutine's completion/error paths (in addition to, not
  instead of, the existing stderr + `LogDownloadError` behavior), and
  include the job ID in their "Download started" text.
- New MCP tools in `internal/mcpserver/tools.go`:
  - `get_download_status(jobId)` — state, actual path when done, error text
    when failed; unknown job ID → `isError: true` with a clear message.
  - `list_downloads()` — one line per known job (ID, kind, video, state).
- Tool description fixes (honesty): `download_video`/`download_audio`
  descriptions change to "Starts a background download ... returns a job ID;
  use get_download_status to check completion and the final file path."
  This deliberately diverges from upstream's wording — log in
  `docs/DECISIONS.md` at implementation time.
- The known limitation that server exit kills in-flight detached downloads
  is **out of scope** here (unchanged from Phase 1 / upstream behavior),
  but `get_download_status` at least makes the outcome observable while
  the server lives.

## Definition of Done

- [ ] 12.1 Job registry with correct state transitions for success,
  failure, and unknown-ID lookup (unit-tested with a fake runner — no real
  yt-dlp in unit tests).
- [ ] 12.2 Actual-final-path resolution finds the real output file when the
  extension differs from the prediction (pure function over a file listing,
  unit-tested), and reports the predicted path with a "not found" note if
  no matching file exists.
- [ ] 12.3 `download_video`/`download_audio` responses include a job ID;
  existing response text otherwise preserved.
- [ ] 12.4 `get_download_status` and `list_downloads` registered and
  callable; unknown job ID returns `isError: true`, not a crash.
- [ ] 12.5 Registry history is capped; `-race` clean under concurrent
  register/lookup.
- [ ] 12.6 `download_video`/`download_audio` descriptions no longer claim
  to return the file path; deviation logged in `docs/DECISIONS.md`.
- [ ] 12.7 `go build ./... && go vet ./... && go test ./...` clean;
  `gofmt` clean; no regressions.

## Test Plan

- **Unit tests (TDD):** job state machine + final-path resolution are new
  logic with no upstream equivalent — ground truth is this reviewed spec
  (same convention as `pickVttFile`). Table-driven: success flow, failure
  flow (error text captured), unknown ID, extension-differs resolution,
  history cap eviction. Concurrency test with `-race`.
- **Manual smoke test (commands):** via throwaway MCP client against
  `dQw4w9WgXcQ`:
  1. `download_audio` → response contains a job ID; immediately call
     `get_download_status` → `running`.
  2. Poll until `done`; confirm the reported actual path exists on disk.
  3. `download_video` with an unreachable/invalid video ID that passes URL
     validation (e.g. a well-formed but nonexistent ID) → poll until
     `failed`; confirm captured yt-dlp error text is present in the status.
  4. `get_download_status` with `jobId: "nope"` → `isError: true`.
  5. `list_downloads` → shows all jobs from steps 1–3.

## Notes / deviations

(fill in during/after implementation)

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
12, log the description-wording deviation in `docs/DECISIONS.md`, then
pause and ask the human before starting task 13.
