# Out-of-band: test-coverage hardening (tiers 1-4)

**Status:** done (T1-T3 complete; T4 complete except the deliberately-skipped
`EnsureYtDlp` concurrency item, see its checklist note)

## Background

Two Explore audits of `internal/core`, `internal/cli`, `internal/mcpserver`,
and `.github/workflows/ci.yml` (2026-08-29/30) found coverage that's uneven
in a way that matters, not just incomplete — see the approved plan for full
detail. Highlights:

- `internal/core/paths.go` (the path-traversal allowlist gating where
  `yt-dlp` writes files) has **zero tests**, and reading it while planning
  turned up a real correctness gap in `pathStartsWith` (string-prefix
  compare, not path-segment compare) — tracked separately as a `docs/BUGS.md`
  entry, not fixed here.
- `internal/cli` has **no test file at all**.
- `internal/mcpserver` registers 12 tools; only 2 have any test coverage.
- Several Go-only files (`transcache.go`, `ytdlp.go`) stop at the happy path.
- CI runs `go test ./...` with no `-race`.

Scope is tiers 1-4 of the approved plan. **Tier 5** (httptest-backed
`FetchVideoMetadata` tests, fake-binary `fetchSegmentsFromYtDlp` tests) is
explicitly deferred to a separately-scoped future task — it needs
production-code seams beyond what tiers 1-4 touch.

Two process side-effects, decided up front (not silent drive-bys):
1. `pathStartsWith`'s prefix-vs-segment gap → new `docs/BUGS.md` entry,
   human decides the fix; the new test pins current behavior.
2. `internal/cli/root.go`'s `fatal()` gets an injectable exit-function seam
   so `os.Exit(1)` paths become testable → logged in `docs/DECISIONS.md`.

## Definition of Done

- [x] **T1 — `internal/core/paths_test.go` (new file)**
  - [x] `GetDownloadsDir`: `XDG_DOWNLOAD_DIR` set-absolute / set-relative-ignored / unset, non-Linux fallback.
  - [x] `pathStartsWith`/`ResolveOutputDir`: same-dir, subdir, path outside all roots → `""`, empty `rawDir` defaults to Downloads, relative-path resolution, and an explicit named test documenting the `/home/user` vs `/home/userXYZ` substring-boundary behavior (`TestPathStartsWith_SegmentBoundaryBug`).
  - [x] `LogDownloadError`: happy path (two appended lines, correct format) via a redirected cache dir.
  - [x] `docs/BUGS.md` gets the new `pathStartsWith` entry (status `open`) — filed as **BUG-004**.
- [x] **T2 — `internal/cli` (new test files)**
  - [x] `fatal()` gets an injectable exit-function seam (`exitFunc`); logged as `docs/DECISIONS.md` DECISION-014.
  - [x] Cobra wiring test: 4 subcommands present, flag names/defaults correct (`--audio`, `--quality=hd720`, `--format=mp3`, `--json`/`-j`, `--timestamps`/`-t`, `--save`/`-s`), plus the persistent `--quiet` flag.
  - [x] `metadata.go`'s JSON vs plain-text output formatting tested against fixed maps, via new pure helpers `formatMetadataJSON`/`formatMetadataPlain` extracted from `RunE`.
  - [x] `search.go`'s "no matches → exit 1" branch tested via a new `printSearchResult` helper + the `exitFunc` seam.
- [x] **T3 — `internal/mcpserver` (new `handlers_test.go`)**
  - [x] Validation-only-branch test added for each of the 7 currently-untested handlers (invalid URL; empty query for `searchTranscriptHandler`; rejected `outputDir` for the download handlers) — no network required.
  - [x] `TestNewServer_ToolCount` asserts 12 registered tools; `TestNewServer_AliasesMatchCanonical` drives each alias/canonical pair through a real in-process MCP round trip (`mcp.NewInMemoryTransports`, same technique as task 7's throwaway smoke client, kept permanently this time) with an invalid-URL input and asserts byte-identical output — verifying alias wiring by actual behavior, not just a source comment.
- [x] **T4 — concurrency + fuzz-completeness**
  - [x] `transcache_test.go`: cap=0 (`TestTranscriptCache_ZeroCapNeverCaches`), negative cap (`TestTranscriptCache_NegativeCapPanics` — surfaced **BUG-005**, a real panic, filed and left open), re-adding an existing key (`TestTranscriptCache_ReAddingExistingKeyDoesNotReorder` — pins FIFO-by-first-insertion, not LRU), TTL-expired-and-over-cap interaction (`TestTranscriptCache_ExpiredEntryStillCountsTowardCap`).
  - [~] `ytdlp_test.go`: concurrent `EnsureYtDlp` — **not done, deliberately.** `EnsureYtDlp` calls the real `ytdlp.Install`/`ytdlp.InstallFFmpeg` network calls directly (no injectable seam, unlike `installFFmpegWithRetry` in the same file), and its `sync.Once` state is a package-level global — exercising it concurrently in a unit test would either hit real network (contradicts the project's own "I/O-bound entrypoints are smoke-tested, not unit-tested" convention, `CLAUDE.md`) or require a new seam that wasn't part of the approved plan. Flagged to the human rather than silently adding a seam or silently skipping; see chat for the go/no-go if this is wanted later.
  - [x] `docs/tasks/fuzz-tests/TASK.md` updated to list `FuzzTranscriptErrorText` (already existed in code, just undocumented).
  - [x] `.github/workflows/ci.yml`'s `go test ./...` → `go test -race ./...`; confirmed clean locally (`go test -race ./...`, all 3 packages pass) before relying on the CI matrix to confirm both `ubuntu-latest`/`windows-latest`.
- [x] `go build ./... && go vet ./... && gofmt -l .` clean and `go test ./...` / `go test -race ./...` both green after each tier.
- [x] `docs/LEDGER.md` updated to list this out-of-band task; pause-and-ask between each tier.

## Summary of new bugs surfaced (not fixed here, per project process)

- **BUG-004** (`docs/BUGS.md`): `pathStartsWith` string-prefix vs path-segment compare — status `open`, human decided "leave open for later."
- **BUG-005** (`docs/BUGS.md`): `transcriptCache.set` panics on a negative cap — status `open`, pending decision.

## Notes / deviations

- Tier 5 (httptest-backed `FetchVideoMetadata`, fake-`yt-dlp` `fetchSegmentsFromYtDlp` tests) was explicitly scoped out before starting (see the approved plan) — not part of this task.
- The `ytdlp_test.go` concurrent-`EnsureYtDlp` item from T4 was not implemented — see its checklist entry above for why (no injectable seam for the real install calls, and adding one wasn't part of the approved scope).
- Two small, pre-approved production-code changes were made in service of testability (not scope creep — both were put to the human via `AskUserQuestion` before implementation): `internal/cli`'s `exitFunc` seam (DECISION-014) and `internal/cli/metadata.go`'s `formatMetadataJSON`/`formatMetadataPlain` extraction (also DECISION-014), plus `internal/cli/search.go`'s `printSearchResult` extraction.

## Test Plan

- All new tests are unit tests (table-driven where the input space is
  small/enumerable) plus targeted concurrency tests (`t.Run` in parallel
  goroutines + `-race`) for T4's `transcache`/`ytdlp` cases — no new
  network-dependent or `yt-dlp`-subprocess-dependent tests are added in this
  task (that's Tier 5, deferred).
- Env-var-dependent tests (`GetDownloadsDir`, `LogDownloadError`) use
  `t.Setenv`/`t.TempDir` for isolation and platform-conditional assertions
  where behavior is `runtime.GOOS`-dependent, matching the existing repo
  convention (e.g. `ffmpeg_prewarm_test.go`'s per-GOOS table).
- Verification commands: `go build ./...`, `go vet ./...`, `gofmt -l .`,
  `go test ./...`, `go test -race ./...` — all must be clean/green before a
  tier is considered done.
- No manual/smoke testing needed — everything in scope is deterministic,
  offline unit-test territory.
