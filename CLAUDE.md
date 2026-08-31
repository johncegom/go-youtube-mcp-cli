# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go port of [`johncegom/youtube-mcp-cli`](https://github.com/johncegom/youtube-mcp-cli) (a TypeScript project): a YouTube transcript/metadata/download tool exposed both as an MCP stdio server and a standalone CLI, with no YouTube API key — metadata comes from scraping the watch-page HTML and transcripts/downloads come from shelling out to `yt-dlp`. Phase 1 (in progress) is a faithful port of the existing TS functionality; no new features beyond what the TS version has are in scope yet.

**Before doing any work here, read `docs/PLAN.md` (the full design/porting plan), `docs/LEDGER.md` (a lightweight index — current status + a table linking to each task's full detail under `docs/tasks/<slug>/TASK.md`), `docs/BUGS.md` (tracked bugs — including inherited-from-upstream ones — pending a decision), `docs/DECISIONS.md` (deliberate design/scope tradeoffs made along the way), and `docs/RETRO.md` (continuous-improvement retrospectives — process/product-quality lessons, not bugs or decisions) — the ledger index is the source of truth for progress, not this file. Read `docs/LEDGER.md` first, then open only the specific `docs/tasks/*/TASK.md` file(s) you actually need — this split exists specifically to avoid loading every finished task's history into context.**

### Task approval: Definition of Done + Test Plan (anti-drift control)

Before starting implementation on any task, its `docs/tasks/<slug>/TASK.md`
must have a **Definition of Done** (concrete, testable exit criteria) and a
**Test Plan** (what gets unit-tested, what gets manually smoke-tested, and
the specific commands to run) written out and reviewed — see
`docs/tasks/07-mcpserver/TASK.md` for the pattern. This is deliberately
**committed to the repo** (the task's own `TASK.md`, not a Plan Mode
session's ephemeral plan file under `~/.claude/plans/`, which nobody else
and no future session can see) — that's the whole point: a future session or
a human reviewer should be able to see exactly what a task was scoped and
approved to do, without reconstructing it from a plan file that isn't part
of this repository's history at all.

Do not silently expand scope mid-task — e.g. fixing unrelated stale doc
content "while you're in there" — even when the fix is correct. If you
notice something out of scope while working, either ask first or log it
separately (as its own decision/task), rather than folding it into the
current task's diff unannounced.

### Long-running tasks: track progress as sub-tasks in TASK.md

If a task's Definition of Done has multiple independently-verifiable parts
(e.g. several files to write, several smoke tests to run), break it into
numbered sub-tasks (`10.1`, `10.2`, `10.2a`, ...) directly in that task's
`TASK.md` Definition of Done, and check them off (`[x]` done, `[~]`
partial/blocked, `[ ]` not started) as you go — don't wait until the whole
task is finished to write anything down. This matters most when a single
sub-task turns out to be slow or gets stuck (a slow first-time tool
download, a flaky smoke test, waiting on external verification): record
which sub-tasks are actually done and exactly which one is still open and
why, rather than re-deriving the whole task's state from scratch or
burning further effort re-verifying already-confirmed parts, if the session
is interrupted, compacted, or handed off. See
`docs/tasks/10-packaging/TASK.md` for the pattern (`10.1a`/`10.3c` marked
`[~]` with a specific resume note each, while the rest of that task's
sub-tasks are `[x]`).

### Bug tracking

When a bug is found (including latent bugs inherited from the upstream TS project that a faithful port reproduces), it goes into `docs/BUGS.md`, not a silent fix. Add an entry with symptom and root cause (or say explicitly that root cause is unknown), propose options if there's an obvious fix, and **wait for a human decision before applying it** — the same pause-and-ask discipline that applies to `docs/LEDGER.md`'s numbered tasks applies to bugs.

Every entry must state **Reachability** (real call path today, vs. only a
test built to hit it) — see "Proportionality" below.

### Proportionality: avoid overengineering, ceremony sized to risk

Overengineering = **code or design that solves problems you don't have.**
Rationale/history: `docs/RETRO.md` RETRO-003.

- Write idiomatic Go, not a TS-shaped port. Add abstraction, config knobs,
  or defensive branches only for a real, currently-reachable call path in
  this codebase — not to mirror TS's structure or "just in case."
- Before adding a guard, name the caller. No real caller (only a test
  built to hit the branch) → fix inline, same change, no `docs/BUGS.md`
  entry/branch/PR needed.
- KISS: among designs that equally satisfy a real reachable requirement,
  pick the one with the fewest moving parts.
- Full `docs/BUGS.md` ceremony (entry, human decision gate, dedicated
  branch/PR) is reserved for bugs marked **Reachable: yes**.
- Extra defensiveness/branching/abstraction beyond the obvious minimal fix
  needs a stated reason (a reachable input) in the commit/PR/TASK.md —
  "just in case" isn't one.
- Exception: strict-TDD ground-truth rigor for *ported* functions stays
  (parity drift is real risk); this only trims ceremony/complexity that
  isn't earning its keep.

### Continuous improvement retrospective

This is about the **way of working** — process, tooling, collaboration
between the human and the assistant — not the product's feature set;
product-scope ideas still go through the normal task-approval process
above, not this log.

**Bar for logging (generalization test):** log an entry only if the lesson
would change how a *different, future* task gets approached. If it's just
"here's what happened in this task," that belongs in that task's own
`TASK.md` "Notes / deviations" section, not here — `docs/RETRO.md` is for
patterns that outlive a single task, not a per-task narrative duplicating
what `TASK.md` already records.

When an entry clears that bar, answer three questions in `docs/RETRO.md`:
**what went well** (a process/tooling/collaboration pattern worth
deliberately repeating), **what could be improved about how the work got
done** (a concrete friction point — a slow tool choice, a missing check, an
ambiguous instruction — not "polish the product more"), and **advice for
next time** (one or two actionable takeaways). This is distinct from
`docs/BUGS.md` (something in the *product* is wrong) and `docs/DECISIONS.md`
(a deliberate product/design tradeoff was *made*).

**Before starting a new task**, skim `docs/RETRO.md` for outstanding advice
relevant to what you're about to do — an entry that's written once and
never rereads is not continuous improvement, it's a journal. See
`docs/RETRO.md` RETRO-001 for the pattern.

### Decision log

When you make a deliberate design/scope tradeoff during implementation —
not a bug, a genuine choice where a reasonable reviewer might pick
differently (e.g. "use library X's built-in Y instead of porting upstream's
hand-written Z") — log it in `docs/DECISIONS.md`: context, decision,
alternatives considered, consequences. Don't log every micro-choice; log
ones worth a reviewer knowing about. See `docs/DECISIONS.md`'s own header
for the precise boundary against `docs/BUGS.md`.

## Commands

```sh
go build ./...                          # build everything
go vet ./...                            # static checks — always run before/after changes
go test ./...                           # run all tests
go test ./internal/core/... -v          # verbose test output for the core package
go test ./internal/core/... -run TestParseVtt -v   # run a single test by name
gofmt -w <files>                        # format — run before every commit/verification pass
```

There is no linter config in this repo — `go build`, `go vet`, `go test`, and `gofmt` are the whole local toolchain. `Taskfile.yml` (run via [`go-task`](https://taskfile.dev), `task <name>`) is a thin optional wrapper around these same commands — `task check` runs build/vet/test -race/gofmt-check in one shot to mirror CI locally; `task --list` shows every task. It's a convenience layer only, not a build system — the commands above remain the source of truth. CI (`.github/workflows/ci.yml`) runs the same four checks on GitHub Actions for every push/PR to `main` (Ubuntu + Windows matrix for build/vet/test, Ubuntu-only for gofmt — see `docs/tasks/ci-cd/TASK.md`); branch protection on `main` requires all of those checks to pass before merge (see `docs/DECISIONS.md` DECISION-007 and DECISION-012 — it was blocked while the repo was private on the free tier, and enabled once the repo went public).

Two binaries live under `cmd/`, both built and working: `cmd/youtube-cli` (CLI — `go run ./cmd/youtube-cli <args>`) and `cmd/youtube-mcp` (MCP stdio server — `go run ./cmd/youtube-mcp`).

## Required workflow: strict TDD

This project follows TDD strictly, and **the implementation must adapt to the test, never the other way around**:

1. For every pure function being ported, first derive **ground truth** for expected behavior by running the actual upstream TypeScript logic (not by hand-reasoning what it "should" do) — e.g. via `node -e '...'` with the real TS snippet, as done for `SanitizeTitle`, `parseVtt`, `transcriptText`/`transcriptTimed`/`formatSearchResult`, and `TranscriptErrorText`. See `internal/core/*_test.go` for the pattern and the comments explaining where each fixture came from.
2. Write the Go test against that ground truth.
3. Confirm the test fails (red) — it should fail to compile if the function doesn't exist yet.
4. Write the minimal implementation to make it pass (green).
5. If a test later turns out wrong, fix the test by re-deriving ground truth — never adjust it just to match whatever the code currently outputs.

I/O-bound code that can't be unit-tested this way (live HTTP scraping in `FetchVideoMetadata`, subprocess calls to `yt-dlp` in `fetchSegments`) is intentionally left uncovered by unit tests and is instead exercised by the end-to-end smoke test (final task in the plan). Pure logic embedded in otherwise I/O-heavy functions should still be factored out and tested (see how `transcript.go` splits pure presentation/search/error-classification functions from the I/O `fetchSegments`/`SaveTranscriptFile`).

**After completing each task, update that task's `docs/tasks/<slug>/TASK.md` first** — check off what was done, note any deviations — **then update `docs/LEDGER.md`'s index** (status column/checkbox, "Current status", "Resume checklist" pointer to the next task) — **then pause and ask the human whether to continue** before starting the next task. Do not chain multiple tasks together in one uninterrupted run, and do not ask before both files reflect the work just finished. Before starting the *next* task, its Definition of Done + Test Plan must exist and be reviewed first (see "Task approval" above).

## Architecture

One Go module, two `cmd/` binaries, one shared `internal/core` package — the idiomatic-Go equivalent of the original TS monorepo's three npm packages (`packages/core`, `packages/cli`, `packages/mcp`).

```
cmd/youtube-cli/       cobra-based CLI entrypoint      (built)
cmd/youtube-mcp/       MCP stdio server entrypoint     (built)
internal/core/         all shared logic — see below
internal/cli/          cobra command wiring            (built)
internal/mcpserver/    MCP tool registration/handlers  (built)
docs/PLAN.md           full design plan + source-repo analysis + porting notes
docs/LEDGER.md         index: current status + links to per-task detail — READ FIRST
docs/tasks/<slug>/TASK.md   full detail for one task (checklist, notes, deviations, DoD, Test Plan)
docs/BUGS.md           tracked bugs (symptom, root cause, options, decision)
docs/DECISIONS.md      deliberate design/scope tradeoffs (not bugs)
```

`internal/` (not `pkg/`) is deliberate: nothing here is meant to be imported by other modules.

### `internal/core` — current shape

- **`videoid.go`** — `ExtractVideoID`: parses bare IDs, `youtu.be`, and `youtube.com/{watch,shorts,embed,v}` URLs.
- **`format.go`** — `FormatTimestamp`/`FormatDuration` (H:MM:SS vs M:SS rendering), `parseISODuration`, `SanitizeTitle` (filename-safe title).
- **`paths.go`** — `GetDownloadsDir`, `ResolveOutputDir` (allowlists output to home-dir/temp-dir subtrees, case-insensitive on Windows), `LogDownloadError` (writes to `os.UserCacheDir()/youtube-mcp/errors.log` — a deliberate cross-platform fix vs. the TS original's hardcoded `~/.cache`).
- **`ytdlp.go`** — `EnsureYtDlp` (lazy, `sync.Once`-guarded install/resolve of the `yt-dlp` binary via `github.com/lrstanley/go-ytdlp`, with best-effort `ffmpeg` install), `NewYtDlpCommand` (returns a command builder with `--ffmpeg-location` pre-set when available). Every code path that shells out to `yt-dlp` must go through these, not construct its own `exec.Cmd`.
- **`metadata.go`** — `FetchVideoMetadata`: scrapes the watch-page HTML with three fallback tiers, most to least reliable — embedded `ytInitialPlayerResponse` JSON, then JSON-LD `<script type="application/ld+json">` blocks, then individual `<meta>` tags — each tier only filling fields the previous left empty. Note: the `ytInitialPlayerResponse` tier intentionally mirrors an upstream quirk where `playerMicroformatRenderer` is read from the top level rather than nested under `microformat` (matching the real TS behavior, not "fixing" it), so in practice `videoDetails` + the `<meta>` tier do most of the real work.
- **`transcript.go`** — split into pure and I/O halves:
  - *Pure* (unit-tested against ground truth): `parseVtt` (VTT → segments, with tag-stripping/entity-decoding/whitespace-collapse/dedupe), `transcriptText`, `transcriptTimed`, `searchSegments`, `formatSearchResult`, `TranscriptErrorText` (classifies a raw error into a user-facing message by substring match — timeout / missing captions / network / generic).
  - *I/O*: `fetchSegments` (runs `yt-dlp` into a temp dir to pull `.vtt` subtitles, parses the result), and the public entrypoints `GetTranscriptText`, `GetTranscriptTimed`, `SearchInTranscript`, `SaveTranscriptFile` (the last fetches transcript + metadata concurrently via two goroutines + a `sync.WaitGroup`, mirroring the TS `Promise.all`, then writes a Markdown file with a metadata header).
- **`download.go`** — `qualityFormatMap`/`qualityFormat` (five quality presets, unknown-quality falls back to hd720), `resolveTitle` (metadata-based display+safe title, falls back to video ID on error), `StartVideoDownload`/`StartAudioDownload` (fire-and-forget goroutine detached from request `ctx`, used by the MCP path, mirrors TS `execFile(...).unref()`), `DownloadVideoBlocking`/`DownloadAudioBlocking` (blocking, uses `Command.BuildCommand` to get the raw `*exec.Cmd` and wires `Stdout`/`Stderr` directly to the process's own — used by the CLI path, mirrors TS `spawn(..., {stdio: "inherit"})`).
- **`ffmpeg_prewarm.go`** — works around a `go-ytdlp` limitation (its ffmpeg download uses a hardcoded 30s HTTP timeout too short for the ~170MB archive on slower connections — see `docs/BUGS.md` BUG-002): downloads the archive ourselves with a longer timeout and drops the extracted binary at `go-ytdlp`'s own expected cache path, so its downloader finds it and never attempts its own (too-short) download. Implemented for `windows/amd64` and `linux/amd64`.

### `internal/cli` — cobra CLI, built (`cmd/youtube-cli`)

`spf13/cobra`-based CLI mirroring the TS `commander` commands (`transcript`,
`search`, `metadata`, `download`) — see `docs/tasks/06-cli-cobra/TASK.md` for
the full breakdown. Uses cobra's built-in `completion` subcommand rather
than hand-porting the original's bash/zsh completion scripts.

### `internal/mcpserver` — MCP stdio server, built (`cmd/youtube-mcp`)

Built on the official `github.com/modelcontextprotocol/go-sdk`, registering
11 tools via `mcp.AddTool` — 8 canonical tools plus 3 aliases
(`get_transcript_timestamps`, `get_video_metadata`, `search_in_transcript`)
that point at the same Go handler function as their canonical counterpart.
Every handler's `Out` type parameter is `any`, so responses are built by
hand as `*mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{...}},
IsError: bool}`, matching the upstream TS server's `{ content, isError }`
shape exactly rather than the SDK's structured-output auto-marshaling path.
See `docs/tasks/07-mcpserver/TASK.md` for the full breakdown and smoke-test
results.

### Key external dependencies and why

- `github.com/lrstanley/go-ytdlp` — wraps the `yt-dlp` CLI with a typed builder and can auto-download `yt-dlp`/`ffmpeg` on first use, matching the original npm package's `postinstall` auto-download behavior (done lazily here since Go binaries have no postinstall hook).
- `github.com/modelcontextprotocol/go-sdk` — official Go MCP SDK.
- `github.com/spf13/cobra` — CLI framework, Go's de facto analog to `commander`.
