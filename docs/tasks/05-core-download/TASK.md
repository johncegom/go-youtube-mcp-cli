# Task 5: Port internal/core: download (video/audio, blocking + fire-and-forget)

**Status:** done

- [x] `internal/core/download.go` — `qualityFormatMap`/`qualityFormat` (pure, five presets + unknown-quality fallback to hd720), `formatVideoDownloadStarted`/`formatAudioDownloadStarted` (pure message templates), `resolveTitle` (fetches metadata for display+safe title, falls back to video ID on error), `StartVideoDownload`/`StartAudioDownload` (fire-and-forget via goroutine detached from request `ctx` so it survives past the request — mirrors TS `execFile(...).unref()` — errors logged via `LogDownloadError` + written to stderr), `DownloadVideoBlocking`/`DownloadAudioBlocking` (blocking, uses `Command.BuildCommand` to get the raw `*exec.Cmd` and wires `Stdout`/`Stderr` directly to the process's own, mirroring TS `spawn(..., {stdio: "inherit"})`)
- [x] TDD followed: ground truth for `qualityFormat` (all 5 presets + `??` fallback behavior on bogus/empty input) and the two message templates derived by running the actual TS `QUALITY_FORMAT_MAP` lookup + template literals through Node; tests written and confirmed red (`undefined: qualityFormat`) before implementation.
- [x] `internal/core/download_test.go` — `TestQualityFormat` (7 cases incl. fallback), `TestFormatVideoDownloadStarted`, `TestFormatAudioDownloadStarted`
- [x] `go build ./... && go vet ./... && go test ./...` — clean, 30 tests passing

Note: `resolveTitle`, `StartVideoDownload`/`StartAudioDownload`,
`DownloadVideoBlocking`/`DownloadAudioBlocking` are I/O-bound (network +
subprocess) and intentionally left uncovered by unit tests, consistent with
`FetchVideoMetadata`/`fetchSegments` in earlier tasks — covered instead by
the task 9 smoke test.

This file was later revisited on the `fix/bug-001-002-subtitle-and-ffmpeg`
branch to fix BUG-002 — see `docs/tasks/bugfix-001-002/TASK.md`.
