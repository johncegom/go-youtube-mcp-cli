# Task 4: Port internal/core: transcript (VTT fetch/parse/search/save)

**Status:** done

- [x] `internal/core/transcript.go` — `parseVtt`, `transcriptText`/`transcriptTimed`/`searchSegments`/`formatSearchResult` (pure), `TranscriptErrorText` (pure), `fetchSegments` (I/O: temp dir + yt-dlp subtitle download via `NewYtDlpCommand`), `GetTranscriptText`, `GetTranscriptTimed`, `SearchInTranscript`, `SaveTranscriptFile` (concurrent segments+metadata fetch via goroutines, matching TS `Promise.all`)
- [x] TDD followed strictly: for every pure function, ground truth was derived by running the *actual* upstream TS logic through Node first (not hand-derived expectations), tests written and confirmed red (compile failure — functions didn't exist), then implementation written to make them pass.
- [x] `internal/core/transcript_test.go` — `parseVtt` (incl. dedupe/tag-strip/entity-decode/whitespace-collapse fixture, header-only, empty-text-block edge cases), `transcriptText`, `transcriptTimed`, `formatSearchResult` (found + no-matches), `searchSegments` (case-insensitive), `TranscriptErrorText` (all 4 message-classification branches)
- [x] `go build ./... && go vet ./... && go test ./...` — clean, all tests pass

Note: this file was later revisited on the `fix/bug-001-002-subtitle-and-ffmpeg`
branch to fix BUG-001 — see `docs/tasks/bugfix-001-002/TASK.md`.
