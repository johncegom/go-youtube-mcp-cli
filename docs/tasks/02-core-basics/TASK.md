# Task 2: Port internal/core: videoid, format, paths

**Status:** done

- [x] `internal/core/videoid.go` — `ExtractVideoID`
- [x] `internal/core/format.go` — `FormatTimestamp`, `FormatDuration`, `parseISODuration`, `SanitizeTitle`
- [x] `internal/core/paths.go` — `GetDownloadsDir`, `ResolveOutputDir`, `LogDownloadError` (uses `os.UserCacheDir()` instead of hardcoded `~/.cache`, per plan's deliberate Windows fix)
- [x] `internal/core/videoid_test.go`, `internal/core/format_test.go` — passing (verified `SanitizeTitle` edge cases against the real TS regex logic run through Node to get ground truth)
