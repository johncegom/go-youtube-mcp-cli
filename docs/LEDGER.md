# Progress Ledger (Index)

Entry point for tracking progress against `docs/PLAN.md`. Each task's full
detail (checklist, notes, deviations, root-cause writeups) lives in its own
file under `docs/tasks/<slug>/TASK.md` — **read this index first, then open
only the task file(s) you actually need.** This split exists specifically to
save context: a fresh session no longer has to load the full history of
every finished task just to find out what's next.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

## Standing process rules

Full detail lives in `CLAUDE.md` ("Required workflow: strict TDD", the
task-approval Definition-of-Done/Test-Plan rule, and the pause-and-ask
rule), `docs/BUGS.md` (bug-tracking process), and `docs/DECISIONS.md`
(deliberate design/scope tradeoffs, not bugs) — not duplicated here. In
short: strict TDD for pure functions (ground truth before the test, test
before the implementation); every task needs a Definition of Done + Test
Plan written and reviewed in its `TASK.md` *before* coding starts; pause and
ask before starting the next task; update the relevant `TASK.md` (not just
this index) before pausing.

## Tasks

| # | Task | Status | Detail |
|---|------|--------|--------|
| 1 | Init Go module + scaffold directory structure | [x] done | [docs/tasks/01-scaffold/TASK.md](tasks/01-scaffold/TASK.md) |
| 2 | Core: videoid, format, paths | [x] done | [docs/tasks/02-core-basics/TASK.md](tasks/02-core-basics/TASK.md) |
| 3 | Core: ytdlp wrapper + metadata scraping | [x] done | [docs/tasks/03-core-ytdlp-metadata/TASK.md](tasks/03-core-ytdlp-metadata/TASK.md) |
| 4 | Core: transcript (VTT fetch/parse/search/save) | [x] done | [docs/tasks/04-core-transcript/TASK.md](tasks/04-core-transcript/TASK.md) |
| 5 | Core: download (video/audio, blocking + fire-and-forget) | [x] done | [docs/tasks/05-core-download/TASK.md](tasks/05-core-download/TASK.md) |
| 6 | CLI with cobra (`youtube-cli`) | [x] done | [docs/tasks/06-cli-cobra/TASK.md](tasks/06-cli-cobra/TASK.md) |
| 7 | MCP server with official go-sdk (`youtube-mcp`) | [x] done | [docs/tasks/07-mcpserver/TASK.md](tasks/07-mcpserver/TASK.md) |
| 8 | Unit tests for core pure functions | [x] done (folded into other tasks) | [docs/tasks/08-unit-tests/TASK.md](tasks/08-unit-tests/TASK.md) |
| 9 | Build + smoke test both binaries | [x] done | [docs/tasks/09-smoke-test/TASK.md](tasks/09-smoke-test/TASK.md) |
| — | Out-of-band: fix BUG-001 + BUG-002 | [x] fixed, merged to `main` | [docs/tasks/bugfix-001-002/TASK.md](tasks/bugfix-001-002/TASK.md) |
| — | Out-of-band: fuzz tests for untrusted-input parsers | [x] done | [docs/tasks/fuzz-tests/TASK.md](tasks/fuzz-tests/TASK.md) |
| — | Out-of-band: task-level Definition of Done/Test Plan + decision log | [x] done | see `CLAUDE.md` process rules + `docs/DECISIONS.md` |
| — | Out-of-band: CI/CD (GitHub Actions; branch protection enabled, see DECISION-012) | [x] done | [docs/tasks/ci-cd/TASK.md](tasks/ci-cd/TASK.md) |
| 10 | Packaging: GoReleaser + GitHub Releases, `go install`, Docker image, README | [x] done | [docs/tasks/10-packaging/TASK.md](tasks/10-packaging/TASK.md) |
| — | Out-of-band: repo made public, MIT-licensed, branch protection enabled | [x] done | see `docs/DECISIONS.md` DECISION-012 |
| 11 | Phase 2: transcript cache + `get_transcript_range` | [x] done | [docs/tasks/11-transcript-cache/TASK.md](tasks/11-transcript-cache/TASK.md) |
| 12 | Phase 2: download job tracking (`get_download_status`, `list_downloads`) | [x] done | [docs/tasks/12-download-jobs/TASK.md](tasks/12-download-jobs/TASK.md) |
| 13 | Phase 2: context-aware transcript search | [ ] not started | [docs/tasks/13-context-search/TASK.md](tasks/13-context-search/TASK.md) |
| 14 | Phase 2: chapters (`get_chapters`) | [x] done (out of order, before 13) | [docs/tasks/14-chapters/TASK.md](tasks/14-chapters/TASK.md) |
| 15 | Phase 2: playlist listing + cross-video search (depends on 11) | [ ] not started | [docs/tasks/15-playlist-search/TASK.md](tasks/15-playlist-search/TASK.md) |
| — | Out-of-band: test-coverage hardening (tiers 1-4) | [x] done | [docs/tasks/test-coverage-hardening/TASK.md](tasks/test-coverage-hardening/TASK.md) |
| — | Out-of-band: `Taskfile.yml` dev-tooling wrapper | [x] done | [docs/tasks/taskfile/TASK.md](tasks/taskfile/TASK.md) |

## Current status

**All 9 numbered tasks plus task 10 are done.** Phase 1 (the faithful Go
port) is complete, and the project is now packaged/distributable three
ways: cross-compiled GitHub Release binaries (via `.goreleaser.yaml` +
`.github/workflows/release.yml`, triggered on `v*` tags), `go install`, and
a Docker image (`Dockerfile`, both binaries on `PATH`, `CMD ["youtube-mcp"]`
default) — see `README.md` for install instructions and
`docs/tasks/10-packaging/TASK.md` for full verification detail. The repo
is now **public** and **MIT-licensed** (`LICENSE`, `docs/DECISIONS.md`
DECISION-012), and branch protection on `main` is enabled (requires
`build-test (ubuntu-latest)`, `build-test (windows-latest)`, and `gofmt`
to pass before merge — this actually enforces CI now, not just reports
it, closing the gap DECISION-007 originally left open). All out-of-band
items (bug fixes, fuzz tests, the anti-drift process work, CI/CD) are also
complete. Remaining known gap: BUG-001/002 fixes cover only the platforms
verified (`docs/DECISIONS.md` DECISION-006). A pre-existing (not
task-10-introduced) CRLF/gofmt working-tree issue on 3 files, previously
flagged in task 10's `TASK.md`
10.6, has since been fixed on the `task-10-packaging` branch.

**Deliberately deferred out of task 10, and not yet scoped as a future
task:** an HTTP/SSE-based MCP transport for genuinely hosted/remote,
multi-client use (the SDK already supports it; the blocker is unaddressed
design questions — shared download directory, per-session isolation, auth
— not a missing library feature, see DECISION-011). Revisit if/when
there's a concrete need for remote (not just locally-spawned-via-Docker)
MCP access.

**Phase 2 is scoped and approved (2026-08-28):** tasks 11–15, driven by a
critical evaluation of the MCP tools from the LLM-agent-consumer
perspective — see `docs/PLAN.md`'s "Phase 2" section for the rationale and
`docs/DECISIONS.md` DECISION-013 for the scope-expansion decision. Each
task has a reviewed Definition of Done + Test Plan in its own `TASK.md`.
Implementation order is 11 → 12 → 13 → 14 → 15 (11 is the foundation the
others lean on; 15 hard-depends on 11). Task 11 is done (2026-08-29) — see
`docs/tasks/11-transcript-cache/TASK.md` for full detail: an in-memory
TTL/cap-bounded transcript cache (`internal/core/transcache.go`) transparent
to all existing `fetchSegments` callers, plus the new `get_transcript_range`
MCP tool and its `core.ParseTimestamp`/`filterSegmentsByRange` building
blocks. Task 12 is done (2026-08-30) — see
`docs/tasks/12-download-jobs/TASK.md` for full detail: a process-lifetime,
capped `jobRegistry` (`internal/core/jobs.go`) tracking download outcomes,
wired into `StartVideoDownload`/`StartAudioDownload`, plus the new
`get_download_status`/`list_downloads` MCP tools and honesty fixes to the
`download_video`/`download_audio` descriptions (`docs/DECISIONS.md`
DECISION-015). Its manual smoke test surfaced and led to fixing BUG-006 (a
`go-ytdlp` pinned-version staleness bug reverting locally-updated yt-dlp
binaries — see `docs/BUGS.md`); the smoke test has since been fully
re-run end-to-end including the success path, all green — see its
`TASK.md` "Notes / deviations" for the full transcript. Task 14 is done
(2026-08-31) — see `docs/tasks/14-chapters/TASK.md` for full detail: a
pure `parseChapters` (`internal/core/chapters.go`) extracting chapters from
a video description, a player-response JSON fallback tier
(`chaptersFromPlayerResponseJSON`), and the new `get_chapters` MCP tool.
Implemented **out of the documented order** — before task 13, with
explicit human approval, since 14 has no code dependency on 13. Manual
smoke test against a real 35-chapter video and a real unchaptered video,
both correct (see `TASK.md` "Notes / deviations"). Tasks 13 and 15 have
not been started. The pause-and-ask rule still applies between tasks.
Related: BUG-003 (dead Node-ism branches in
`TranscriptErrorText`) is fixed and merged to `main` (PR #10, commit
`be6c93d`) — see `docs/BUGS.md` for the full writeup.

**Out-of-band test-coverage hardening is done (2026-08-30)** — see
`docs/tasks/test-coverage-hardening/TASK.md`. Closed the highest-risk gaps
found by a coverage audit: `internal/core/paths.go` (previously zero tests)
now has a full test file; `internal/cli` (previously zero tests) now has
cobra wiring/flag/output-format tests plus a new `exitFunc` seam
(`docs/DECISIONS.md` DECISION-014) so exit-1 paths are testable;
`internal/mcpserver` now covers all 12 tools' validation branches plus a
real in-process round-trip test proving alias tools are wired to their
canonical handler; `transcache.go` gained cap/TTL boundary tests; CI's
`go test` now runs with `-race`. Surfaced two new bugs along the way
(neither fixed yet, both open pending decision): **BUG-004** (a
path-traversal-allowlist boundary bug in `pathStartsWith`) and **BUG-005**
(a negative-cap panic in `transcriptCache.set`, not reachable via any
current code path). Tier 5 (httptest/fake-binary I/O-boundary tests for
`FetchVideoMetadata`/`fetchSegmentsFromYtDlp`) was explicitly deferred to a
future task.

## Resume checklist for next session

1. Read this index first (not the individual task files, unless you need one).
2. Run `go build ./... && go vet ./... && go test ./...` to confirm the current state still holds.
3. Skim `docs/RETRO.md` for any still-relevant advice before starting new work.
4. Next task: **13** (context-aware transcript search) or **15** (playlist
   listing + cross-video search, depends on 11 which is done) — 14 is now
   done, out of order. Whichever is picked, its DoD + Test Plan should be
   (re-)reviewed before implementation starts.
5. After finishing a task: update **that task's `TASK.md`** with full detail first, then update this index's status column/checkbox for it, then pause and ask the human before starting the next task. If the task involved a deliberate design/scope tradeoff, log it in `docs/DECISIONS.md` too; if it surfaced a way-of-working lesson that generalizes beyond that one task, log it in `docs/RETRO.md`.
