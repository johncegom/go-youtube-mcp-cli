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
| 7 | MCP server with official go-sdk (`youtube-mcp`) | [ ] not started (DoD + Test Plan approved) | [docs/tasks/07-mcpserver/TASK.md](tasks/07-mcpserver/TASK.md) |
| 8 | Unit tests for core pure functions | [x] done (folded into other tasks) | [docs/tasks/08-unit-tests/TASK.md](tasks/08-unit-tests/TASK.md) |
| 9 | Build + smoke test both binaries | [~] in progress | [docs/tasks/09-smoke-test/TASK.md](tasks/09-smoke-test/TASK.md) |
| — | Out-of-band: fix BUG-001 + BUG-002 | [x] fixed, merged to `main` | [docs/tasks/bugfix-001-002/TASK.md](tasks/bugfix-001-002/TASK.md) |
| — | Out-of-band: fuzz tests for untrusted-input parsers | [x] done | [docs/tasks/fuzz-tests/TASK.md](tasks/fuzz-tests/TASK.md) |
| — | Out-of-band: task-level Definition of Done/Test Plan + decision log | [x] done | see `CLAUDE.md` process rules + `docs/DECISIONS.md` |
| — | Out-of-band: CI/CD (GitHub Actions + branch protection) | [~] in progress | [docs/tasks/ci-cd/TASK.md](tasks/ci-cd/TASK.md) |

## Current status

Tasks 1–6, 8 done; task 9 partially done (CLI half only). Task 7 (MCP
server) is next — it's the only remaining blocker for finishing task 9, and
already has an approved Definition of Done + Test Plan (see its `TASK.md`).
Both out-of-band items (bug fixes, fuzz tests) are complete, as is the
anti-drift process work (per-task DoD/Test Plan requirement + decision log).

This restructure (splitting the ledger into per-task files under
`docs/tasks/`, plus the DoD/Test Plan and decision-log additions) was done
across two branches — `docs/ledger-task-split` and
`docs/task-dod-and-decision-log` — reconciled together on
`docs/reconcile-anti-drift`.

## Resume checklist for next session

1. Read this index first (not the individual task files, unless you need one).
2. Run `go build ./... && go vet ./... && go test ./...` to confirm the current state holds.
3. Open `docs/tasks/07-mcpserver/TASK.md` and follow its Definition of Done + Test Plan — that's the next task, already scoped and approved.
4. After finishing a task: update **that task's `TASK.md`** with full detail first, then update this index's status column/checkbox for it, then pause and ask the human before starting the next task. If the task involved a deliberate design/scope tradeoff, log it in `docs/DECISIONS.md` too.
