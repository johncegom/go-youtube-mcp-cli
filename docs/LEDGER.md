# Progress Ledger (Index)

Entry point for tracking progress against `docs/PLAN.md`. Each task's full
detail (checklist, notes, deviations, root-cause writeups) lives in its own
file under `docs/tasks/<slug>/TASK.md` — **read this index first, then open
only the task file(s) you actually need.** This split exists specifically to
save context: a fresh session no longer has to load the full history of
every finished task just to find out what's next.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

## Standing process rules

Full detail lives in `CLAUDE.md` ("Required workflow: strict TDD" and the
pause-and-ask rule) and `docs/BUGS.md` (bug-tracking process) — not
duplicated here. In short: strict TDD for pure functions (ground truth
before the test, test before the implementation), pause and ask before
starting the next task, and update the relevant `TASK.md` (not just this
index) before pausing.

## Tasks

| # | Task | Status | Detail |
|---|------|--------|--------|
| 1 | Init Go module + scaffold directory structure | [x] done | [docs/tasks/01-scaffold/TASK.md](tasks/01-scaffold/TASK.md) |
| 2 | Core: videoid, format, paths | [x] done | [docs/tasks/02-core-basics/TASK.md](tasks/02-core-basics/TASK.md) |
| 3 | Core: ytdlp wrapper + metadata scraping | [x] done | [docs/tasks/03-core-ytdlp-metadata/TASK.md](tasks/03-core-ytdlp-metadata/TASK.md) |
| 4 | Core: transcript (VTT fetch/parse/search/save) | [x] done | [docs/tasks/04-core-transcript/TASK.md](tasks/04-core-transcript/TASK.md) |
| 5 | Core: download (video/audio, blocking + fire-and-forget) | [x] done | [docs/tasks/05-core-download/TASK.md](tasks/05-core-download/TASK.md) |
| 6 | CLI with cobra (`youtube-cli`) | [x] done | [docs/tasks/06-cli-cobra/TASK.md](tasks/06-cli-cobra/TASK.md) |
| 7 | MCP server with official go-sdk (`youtube-mcp`) | [ ] not started | [docs/tasks/07-mcpserver/TASK.md](tasks/07-mcpserver/TASK.md) |
| 8 | Unit tests for core pure functions | [x] done (folded into other tasks) | [docs/tasks/08-unit-tests/TASK.md](tasks/08-unit-tests/TASK.md) |
| 9 | Build + smoke test both binaries | [~] in progress | [docs/tasks/09-smoke-test/TASK.md](tasks/09-smoke-test/TASK.md) |
| — | Out-of-band: fix BUG-001 + BUG-002 | [x] fixed, merged to `main` | [docs/tasks/bugfix-001-002/TASK.md](tasks/bugfix-001-002/TASK.md) |
| — | Out-of-band: fuzz tests for untrusted-input parsers | [x] done | [docs/tasks/fuzz-tests/TASK.md](tasks/fuzz-tests/TASK.md) |

## Current status

Tasks 1–6, 8 done; task 9 partially done (CLI half only). Task 7 (MCP
server) is next — it's the only remaining blocker for finishing task 9.
Both out-of-band items (bug fixes, fuzz tests) are complete.

This restructure (splitting the ledger into per-task files under
`docs/tasks/`) was itself done on branch `docs/ledger-task-split`.

## Resume checklist for next session

1. Read this index first (not the individual task files, unless you need one).
2. Run `go build ./... && go vet ./... && go test ./...` to confirm the current state holds.
3. Open `docs/tasks/07-mcpserver/TASK.md` and follow its kickoff plan — that's the next task.
4. After finishing a task: update **that task's `TASK.md`** with full detail first, then update this index's status column/checkbox for it, then pause and ask the human before starting the next task.
