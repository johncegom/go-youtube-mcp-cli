# Task 3: Port internal/core: ytdlp wrapper + metadata scraping

**Status:** done

- [x] `internal/core/ytdlp.go` — `EnsureYtDlp` (lazy `sync.Once` install of yt-dlp + best-effort ffmpeg), `NewYtDlpCommand`
- [x] `internal/core/metadata.go` — `FetchVideoMetadata` (ytInitialPlayerResponse tier, JSON-LD tier, `<meta>` tag tier)
- [x] `go build ./... && go vet ./...` — clean
- [x] `internal/core/metadata_test.go` — tests for the pure/testable helpers (`extractMetaTag`, `keywordsToString`, `atoiSafe`, `fillFromJSONLD` incl. `@graph` and no-overwrite behavior). `FetchVideoMetadata` itself is a live network call and intentionally untested at the unit level.

## Process note

Starting task 4, switched to strict TDD — write the failing test first, then
the minimal implementation to pass it, per user instruction. See
`CLAUDE.md`'s "Required workflow: strict TDD" section for the rule as it
applies for the rest of the project.
