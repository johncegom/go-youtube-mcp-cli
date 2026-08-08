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
| — | Out-of-band: CI/CD (GitHub Actions; branch protection blocked, see DECISION-007) | [x] done | [docs/tasks/ci-cd/TASK.md](tasks/ci-cd/TASK.md) |
| 10 | Packaging: GoReleaser + GitHub Releases, `go install`, Docker image, README | [x] done | [docs/tasks/10-packaging/TASK.md](tasks/10-packaging/TASK.md) |

## Current status

**All 9 numbered tasks plus task 10 are done.** Phase 1 (the faithful Go
port) is complete, and the project is now packaged/distributable three
ways: cross-compiled GitHub Release binaries (via `.goreleaser.yaml` +
`.github/workflows/release.yml`, triggered on `v*` tags), `go install`, and
a Docker image (`Dockerfile`, both binaries on `PATH`, `CMD ["youtube-mcp"]`
default) — see `README.md` for install instructions and
`docs/tasks/10-packaging/TASK.md` for full verification detail. All
out-of-band items (bug fixes, fuzz tests, the anti-drift process work,
CI/CD) are also complete. Remaining known gaps, both deliberate and
documented: BUG-001/002 fixes cover only the platforms verified
(`docs/DECISIONS.md` DECISION-006), branch protection isn't enabled
(DECISION-007), and a pre-existing (not task-10-introduced) CRLF/gofmt
working-tree issue on 3 files is flagged in task 10's `TASK.md` 10.6,
pending a human decision on whether to run `git add --renormalize .`.

**Deliberately deferred out of task 10, and not yet scoped as a future
task:** an HTTP/SSE-based MCP transport for genuinely hosted/remote,
multi-client use (the SDK already supports it; the blocker is unaddressed
design questions — shared download directory, per-session isolation, auth
— not a missing library feature, see DECISION-011). Revisit if/when
there's a concrete need for remote (not just locally-spawned-via-Docker)
MCP access.

There is no task 11 yet. Any further work needs its own scoping pass
(PLAN.md update or a new out-of-band entry with its own `TASK.md` +
Definition of Done + Test Plan) before starting, per the project's
standing process.

## Resume checklist for next session

1. Read this index first (not the individual task files, unless you need one).
2. Run `go build ./... && go vet ./... && go test ./...` to confirm the current state still holds.
3. Skim `docs/RETRO.md` for any still-relevant advice before starting new work.
4. Tasks 1–10 are done — there's no pre-scoped next task. Check with the human for direction before starting new work; scope it (DoD + Test Plan, per `CLAUDE.md`) before writing code.
5. After finishing a task: update **that task's `TASK.md`** with full detail first, then update this index's status column/checkbox for it, then pause and ask the human before starting the next task. If the task involved a deliberate design/scope tradeoff, log it in `docs/DECISIONS.md` too; if it surfaced a way-of-working lesson that generalizes beyond that one task, log it in `docs/RETRO.md`.
