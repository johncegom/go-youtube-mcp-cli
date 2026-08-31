# Out-of-band: add `Taskfile.yml` dev-tooling wrapper

**Status:** done

## Context

Requested directly by the human ("I want a taskfile to speed up when I want
to run such a command") — the local toolchain (`go build`/`go vet`/`go
test`/`gofmt`) documented in `CLAUDE.md`'s Commands section had to be
retyped by hand every time, including the CI-matching sequence (build, vet,
test -race, gofmt check) used to verify a change before pushing.

This `TASK.md` was written **after** the work was already merged (PR #19,
then #20 for the doc follow-up) — the Definition-of-Done/Test-Plan ceremony
was initially skipped on the (incorrect) assumption that it only applied to
the Phase 1/Phase 2 porting-plan tasks tracked in `docs/LEDGER.md`. Human
correction: the ceremony is the general anti-drift mechanism for any new
thing added to this repo, not scoped to porting. This file exists to close
that gap retroactively; going forward, dev-tooling additions like this get
their `TASK.md` *before* implementation, per the normal rule.

## Design decisions

- **Tool**: [`go-task`](https://taskfile.dev) (`Taskfile.yml`), not a
  Makefile — no existing precedent in this repo, and `go-task`'s YAML
  syntax stays easy to keep 1:1 with the command table already in
  `CLAUDE.md`. See `docs/DECISIONS.md` DECISION-016 for the full tradeoff.
- **Subordinate, not authoritative**: `Taskfile.yml` wraps the documented
  raw commands; it does not replace them as the source of truth in
  `CLAUDE.md`, and nothing in CI or the required TDD workflow depends on
  `task` being installed.
- **`deps:update` chains into `check`**: a dependency bump that breaks the
  build/vet/tests/formatting must fail loudly in the same task run, not
  land silently in `go.mod`/`go.sum` for a later `task check` to catch.
- **`go:check-update` is read-only**: updating the Go toolchain itself is
  an admin-level, install-method-dependent action (no winget/scoop on the
  dev machine, and Go wasn't installed via the Chocolatey that is present)
  — the task reports installed-vs-latest version only, it never attempts
  to install anything.

## Definition of Done

- [x] `Taskfile.yml` exists at repo root with tasks covering the full local
  toolchain: `build`, `vet`, `test`, `test:race`, `fmt`, `fmt:check`.
- [x] `check` task runs the same four checks CI runs, in the same shape
  (build, vet, test -race, gofmt check).
- [x] `install` task (`go install` both `cmd/` binaries) and `cli`/`mcp` run
  helpers for day-to-day use.
- [x] `deps:update` task updates dependencies and self-verifies via `check`
  before leaving mutated `go.mod`/`go.sum` unverified.
- [x] `go:check-update` task reports Go-toolchain update availability
  without performing a system-level install.
- [x] `CLAUDE.md`'s Commands section updated to mention `Taskfile.yml`/
  `task` (was previously stale, claiming no Makefile/task runner existed
  at all — fixed in PR #20).
- [x] Tradeoff logged in `docs/DECISIONS.md` (DECISION-016).
- [x] This `TASK.md` + `docs/LEDGER.md` row added retroactively, closing
  the process gap the human flagged.

## Test Plan

- [x] `task check` run locally, confirmed all four checks pass clean.
- [x] `task install` run locally, confirmed `youtube-cli.exe`/
  `youtube-mcp.exe` land in `$GOPATH/bin` (`C:\Users\Admin\go\bin`).
- [x] `task go:check-update` run locally, confirmed it correctly reported
  installed (`go1.26.3`) vs. latest (`go1.27.0`) without touching the
  system Go install.
- No unit tests apply — this is dev tooling (a YAML task-runner config),
  not a `internal/core` pure function with a ground-truth behavior to pin
  down via TDD.

## Notes / deviations

- Retroactive approval: DoD/Test-Plan were written and satisfied *after*
  merge, not reviewed beforehand as the process normally requires. No
  further code changes resulted from writing this file — it's a paper-trail
  fix, not a rework. Future dev-tooling requests will get this file
  written and reviewed before implementation starts.
