# Task 16: Transcript-fetch observability logging

**Status:** done

## Plan

Triggered by a real support case: `get_transcript_timed` returned
`"Transcript fetch timed out for video kjoQPn--F7A. Please try again."`, but
the very next call succeeded, and `errors.log` had no record of the failure
at all. Investigation found that `LogDownloadError`
(`internal/core/paths.go`, writes to `os.UserCacheDir()/youtube-mcp/errors.log`)
is only ever called from the download path (`StartVideoDownload`/
`StartAudioDownload` in `internal/core/download.go`) and the ffmpeg-install
retry path (`internal/core/ytdlp.go`, BUG-002) — never from anywhere in
`internal/core/transcript.go`. Every transcript-fetch failure is currently
visible to the caller only as one of `TranscriptErrorText`'s four canned
sentences and then disappears; there is no way to reconstruct what actually
happened after the fact. Framed by the human as a **feature**
(observability/diagnostics), not a `docs/BUGS.md` entry.

Design (see the approved plan under `.claude/plans` for full context — not
duplicated here beyond what's needed to execute):

- Reuse `LogDownloadError`/`errors.log` as-is (already described as "the
  shared youtube-mcp error log," plain-text, append-only) — no new logging
  framework, no JSON, no rotation. No reachable requirement today for a
  second consumer to parse this file programmatically.
- Single call site: `fetchSegmentsFromYtDlp` in `internal/core/transcript.go`
  — the one choke point every transcript entrypoint (`GetTranscriptText`,
  `GetTranscriptTimed`, `GetTranscriptRange`, `SearchInTranscript`,
  `SaveTranscriptFile`) funnels through via `fetchSegments`. It's in
  `internal/core`, so this covers **both** `cmd/youtube-cli` and
  `cmd/youtube-mcp`, not just the MCP server.
- Extract `TranscriptErrorText`'s classification `switch` into a pure
  `classifyTranscriptError(err error) string` helper (`"timeout"` |
  `"missing_captions"` | `"network"` | `"generic"`), reused by both the
  user-facing message and the new log line. `TranscriptErrorText`'s output
  must not change — existing tests are the regression guard.
- Name the existing inline `30*time.Second` as `transcriptFetchTimeout`; add
  `transcriptFetchSlowThreshold = 20*time.Second`.
- Give `fetchSegmentsFromYtDlp` named returns, track `start := time.Now()`,
  and log via one `defer` on any non-nil error — covers every existing
  failure return (EnsureYtDlp, temp-dir, yt-dlp timeout/process failure,
  no-.vtt-file, empty-parse) without touching each `return` site
  individually. Log-line text built by a pure, separately-tested
  `formatTranscriptFailureLog(language string, elapsed time.Duration,
  category string, err error) string`.
- On success, if `time.Since(start) > transcriptFetchSlowThreshold`, log a
  `transcript_fetch_slow` line (approved scope addition) — gives a trend
  signal for fetches creeping toward the timeout before they start failing
  outright, the kind of data that would have shortened the BUG-007
  investigation.

## Definition of Done

- [x] `classifyTranscriptError` extracted, unit-tested against the same
      ground-truth error fixtures already used for `TranscriptErrorText`
      (all 4 categories covered).
- [x] `TranscriptErrorText` refactored to call it; all existing
      `TranscriptErrorText` tests pass unchanged (proves no output drift).
- [x] `transcriptFetchTimeout`/`transcriptFetchSlowThreshold` constants
      added; the `context.WithTimeout` call site uses the named constant.
- [x] `fetchSegmentsFromYtDlp` logs every failure path via
      `LogDownloadError`, including elapsed duration and classified
      category, using a pure `formatTranscriptFailureLog` helper that is
      unit-tested independently of any file I/O.
- [x] A successful fetch exceeding `transcriptFetchSlowThreshold` logs a
      `transcript_fetch_slow` line via the same mechanism.
- [x] `go build ./... && go vet ./... && go test ./... && gofmt -l .` all
      clean, no regressions elsewhere in the suite.
- [x] Manual smoke test confirms real log lines land in
      `errors.log` for (a) a forced deterministic failure and (b) a real
      transcript fetch (slow-path line only if duration happens to exceed
      the threshold).
- [x] This file, `docs/LEDGER.md`, and `docs/DECISIONS.md` updated after
      implementation, per `CLAUDE.md`'s process.

## Test Plan

- **Unit tests** (`internal/core/transcript_test.go`):
  - `classifyTranscriptError`: table-driven over the existing
    `TranscriptErrorText` ground-truth fixtures, asserting the category tag.
  - `formatTranscriptFailureLog`: pure string-formatting test, a handful of
    representative `(language, elapsed, category, err)` inputs → exact
    expected output string. No file I/O involved.
  - Regression: existing `TranscriptErrorText` tests must pass unmodified.
- **Manual smoke test** (I/O path — per `CLAUDE.md`, yt-dlp subprocess code
  is intentionally left to smoke tests, not unit tests):
  1. Force a deterministic failure without relying on network flakiness —
     request a language with no captions (reproduces the existing
     "missing captions" branch reliably) and confirm a
     `transcript_fetch <videoID>` line appears in `errors.log` with
     `category=missing_captions`, a duration, and the raw error text.
  2. Run a normal successful fetch (e.g. `kjoQPn--F7A`, the video from the
     original report) and confirm no failure line is written; if duration
     happens to exceed the slow threshold, confirm a `transcript_fetch_slow`
     line appears instead.
  3. If a real timeout can be reproduced (e.g. an unreachable proxy, the
     same technique already used to capture the BUG-003 fixtures), confirm
     `category=timeout` and a duration close to `transcriptFetchTimeout`.

This Definition of Done + Test Plan was written and reviewed *before*
starting implementation, per the project's task-approval process (see
`CLAUDE.md`).

## Before starting

Run `go build ./... && go vet ./... && go test ./...` to confirm the current
state holds clean.

## After finishing

Update this file's status/checklist, then update `docs/LEDGER.md`'s index
row for task 16, add the `docs/DECISIONS.md` entry covering the
reuse-vs-new-logging-framework and single-call-site choices, then pause and
ask the human before starting any further task.

## Notes / deviations

- Implementation followed the plan exactly — `classifyTranscriptError` and
  `formatTranscriptFailureLog` extracted as pure functions, `defer`-based
  logging in `fetchSegmentsFromYtDlp` via named returns, near-timeout
  success logging before the final return. No deviations.
- Unit tests written first (red: `undefined: classifyTranscriptError` /
  `undefined: formatTranscriptFailureLog`, compile failure) before the
  implementation (green). One test fixture had to be corrected after
  writing it — `formatTranscriptFailureLog`'s expected output for an
  812ms duration was written as `"0.812s"` but Go's `time.Duration.String()`
  actually renders sub-second durations as `"812ms"`; fixed the test
  fixture, not the implementation, since the implementation's behavior was
  correct and the fixture was simply wrong.
- Manual smoke test used the repo's own CLI build (`go run ./cmd/youtube-cli
  transcript ...`), not the `minh-toolkit` Claude Code plugin's installed
  `youtube-mcp` tool — the plugin runs a separately-installed binary, not
  this repo's current source, so testing through it would not have
  exercised the new code at all. Confirmed:
  - Forced failure (`--language zzz-nonexistent` against `kjoQPn--F7A`, the
    video from the original report) produced:
    `[2026-09-08T19:09:35Z] transcript_fetch kjoQPn--F7A: lang=zzz-nonexistent duration=6.825s category=missing_captions err=no transcript available for video kjoQPn--F7A. The video may not have captions in language "zzz-nonexistent"`
  - A real successful fetch of the same video produced no new log line
    (correct — well under the 20s slow threshold).
  - Timeout-path and slow-path-on-success were not separately reproduced
    live (no reliable deterministic way to force either without real
    network manipulation, matching the same constraint noted for the
    original `TranscriptErrorText` network-error fixtures in BUG-003); the
    logging code path is identical for every failure branch (single
    `defer` in `fetchSegmentsFromYtDlp`), so the missing-captions
    verification above exercises the same mechanism a timeout would.
