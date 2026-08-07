# Progress Ledger

Tracks implementation progress against `docs/PLAN.md`. Update this file (check
off items, add notes) whenever a work session ends, so a future session can
pick up without re-deriving state.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

## 1. Init Go module + scaffold directory structure
- [x] `go mod init github.com/johncegom/go-youtube-mcp-cli`
- [x] Created `cmd/youtube-cli`, `cmd/youtube-mcp`, `internal/core`, `internal/cli`, `internal/mcpserver`
- [x] Added deps: `modelcontextprotocol/go-sdk/mcp`, `lrstanley/go-ytdlp`, `spf13/cobra`, `golang.org/x/term` (`golang.org/x/sync` came in transitively, available for errgroup use)
- Notes: confirmed API shapes via `go doc` before coding — `ytdlp.Command` builder methods (`SkipDownload`, `WriteAutoSubs`, `WriteSubs`, `SubLangs`, `SubFormat`, `FFmpegLocation`, `Run`), `ytdlp.Install`/`InstallFFmpeg`/`ResolvedInstall`, and `mcp.NewServer`/`mcp.AddTool`/`ToolHandlerFor`/`CallToolResult`/`TextContent`/`StdioTransport`.

## 2. Port internal/core: videoid, format, paths
- [x] `internal/core/videoid.go` — `ExtractVideoID`
- [x] `internal/core/format.go` — `FormatTimestamp`, `FormatDuration`, `parseISODuration`, `SanitizeTitle`
- [x] `internal/core/paths.go` — `GetDownloadsDir`, `ResolveOutputDir`, `LogDownloadError` (uses `os.UserCacheDir()` instead of hardcoded `~/.cache` — see `docs/DECISIONS.md` DECISION-002)
- [x] `internal/core/videoid_test.go`, `internal/core/format_test.go` — passing (verified `SanitizeTitle` edge cases against the real TS regex logic run through Node to get ground truth)

## 3. Port internal/core: ytdlp wrapper + metadata scraping
- [x] `internal/core/ytdlp.go` — `EnsureYtDlp` (lazy `sync.Once` install of yt-dlp + best-effort ffmpeg — see `docs/DECISIONS.md` DECISION-001 for why lazy install was chosen over matching upstream's npm-postinstall-time install), `NewYtDlpCommand`
- [x] `internal/core/metadata.go` — `FetchVideoMetadata` (ytInitialPlayerResponse tier, JSON-LD tier, `<meta>` tag tier)
- [x] `go build ./... && go vet ./...` — clean
- [x] `internal/core/metadata_test.go` — tests for the pure/testable helpers (`extractMetaTag`, `keywordsToString`, `atoiSafe`, `fillFromJSONLD` incl. `@graph` and no-overwrite behavior). `FetchVideoMetadata` itself is a live network call and intentionally untested at the unit level.
- **Status: DONE**

**Process note:** starting task 4, switching to strict TDD — write the failing test first, then the minimal implementation to pass it, per user instruction. Will pause after each numbered task and ask whether to continue.

## 4. Port internal/core: transcript (VTT fetch/parse/search/save)
- [x] `internal/core/transcript.go` — `parseVtt`, `transcriptText`/`transcriptTimed`/`searchSegments`/`formatSearchResult` (pure), `TranscriptErrorText` (pure), `fetchSegments` (I/O: temp dir + yt-dlp subtitle download via `NewYtDlpCommand`), `GetTranscriptText`, `GetTranscriptTimed`, `SearchInTranscript`, `SaveTranscriptFile` (concurrent segments+metadata fetch via goroutines, matching TS `Promise.all`)
- [x] TDD followed strictly: for every pure function, ground truth was derived by running the *actual* upstream TS logic through Node first (not hand-derived expectations), tests written and confirmed red (compile failure — functions didn't exist), then implementation written to make them pass.
- [x] `internal/core/transcript_test.go` — `parseVtt` (incl. dedupe/tag-strip/entity-decode/whitespace-collapse fixture, header-only, empty-text-block edge cases), `transcriptText`, `transcriptTimed`, `formatSearchResult` (found + no-matches), `searchSegments` (case-insensitive), `TranscriptErrorText` (all 4 message-classification branches)
- [x] `go build ./... && go vet ./... && go test ./...` — clean, all tests pass
- **Status: DONE**

## 5. Port internal/core: download (video/audio, blocking + fire-and-forget)
- [x] `internal/core/download.go` — `qualityFormatMap`/`qualityFormat` (pure, five presets + unknown-quality fallback to hd720), `formatVideoDownloadStarted`/`formatAudioDownloadStarted` (pure message templates), `resolveTitle` (fetches metadata for display+safe title, falls back to video ID on error), `StartVideoDownload`/`StartAudioDownload` (fire-and-forget via goroutine detached from request `ctx` so it survives past the request — mirrors TS `execFile(...).unref()` — errors logged via `LogDownloadError` + written to stderr), `DownloadVideoBlocking`/`DownloadAudioBlocking` (blocking, uses `Command.BuildCommand` to get the raw `*exec.Cmd` and wires `Stdout`/`Stderr` directly to the process's own, mirroring TS `spawn(..., {stdio: "inherit"})`)
- [x] TDD followed: ground truth for `qualityFormat` (all 5 presets + `??` fallback behavior on bogus/empty input) and the two message templates derived by running the actual TS `QUALITY_FORMAT_MAP` lookup + template literals through Node; tests written and confirmed red (`undefined: qualityFormat`) before implementation.
- [x] `internal/core/download_test.go` — `TestQualityFormat` (7 cases incl. fallback), `TestFormatVideoDownloadStarted`, `TestFormatAudioDownloadStarted`
- [x] `go build ./... && go vet ./... && go test ./...` — clean, 30 tests passing
- **Status: DONE**
- Note: `resolveTitle`, `StartVideoDownload`/`StartAudioDownload`, `DownloadVideoBlocking`/`DownloadAudioBlocking` are I/O-bound (network + subprocess) and intentionally left uncovered by unit tests, consistent with `FetchVideoMetadata`/`fetchSegments` in earlier tasks — covered instead by the task 9 smoke test.

## 6. Build internal/cli with cobra (youtube-cli)
- [x] `internal/cli/root.go` — `NewRootCommand(version)`, persistent `--quiet` flag, `fatal()` helper (prints `error: ...` to stderr + `os.Exit(1)`, matching the TS CLI's `fatal()` exactly — `SilenceErrors`/`SilenceUsage` set so cobra never double-prints)
- [x] `internal/cli/spinner.go` — braille-frame spinner gated on `--quiet` and `golang.org/x/term.IsTerminal`, matching TS `process.stderr.isTTY` check
- [x] `internal/cli/transcript.go`, `search.go`, `metadata.go`, `download.go` — 1:1 port of the TS subcommands/flags/shorthands, calling the now-complete `internal/core` functions
- [x] `cmd/youtube-cli/main.go` — wires it up, `version` var overridable via `-ldflags`
- [x] `completions`: used cobra's **built-in** `completion` subcommand (bash/zsh/fish/powershell) instead of hand-porting the TS's literal bash/zsh script strings — the deliberate simplification called out in `docs/PLAN.md`. No `internal/cli/completions.go` file exists; nothing needed to be written for it.
- [x] Smoke-tested manually: no-args help output, `--version`, invalid-URL error path (`error: invalid YouTube URL or video ID: "..."`, exit 1, matches TS format), `completion bash` produces a real completion script
- [x] `go build ./... && go vet ./... && gofmt -l .` — clean; `go test ./...` — clean (30 core tests still passing; no new unit tests added here since this layer is I/O/CLI-wiring, consistent with the task 6 kickoff note)
- Known minor divergences from the TS CLI, logged as deliberate decisions rather than bugs — see `docs/DECISIONS.md` DECISION-003 (cobra completion vs hand-ported scripts), DECISION-004 (`metadata --json` key order), DECISION-005 (unknown-command message wording).
- **Status: DONE**

## 7. Build internal/mcpserver with official go-sdk (youtube-mcp)
- [ ] Not started.

### Plan (per `docs/PLAN.md`'s "Package-by-package porting notes")

- Typed input structs per distinct input shape (`urlLangInput{URL, Language}`, `searchInput{URL, Query, Language}`, `downloadVideoInput{URL, Quality, OutputDir}`, `downloadAudioInput{URL, Format, OutputDir}`, `downloadTranscriptInput{URL, Language, OutputDir}`), with `json`/`jsonschema` tags mirroring the TS `inputSchema` objects.
- Register all 10 tools via `mcp.AddTool(server, &mcp.Tool{Name: ..., Description: ...}, handlerFunc)`: `get_transcript`, `get_transcript_timed`, `get_transcript_timestamps` (alias of timed), `get_metadata`, `get_video_metadata` (alias), `search_transcript`, `search_in_transcript` (alias), `download_video`, `download_audio`, `download_transcript`, `download_transcript_timed`. Aliases = two `AddTool` calls pointing at the same Go handler function, mirroring the TS `switch` statement's fallthrough cases.
- Handlers build `*mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: ...}}, IsError: bool}` directly (confirmed API shape in task 1's `go doc` research) and return a zero-value output — matching the TS server's `{ content: [...], isError }` pattern exactly, not the SDK's structured-output auto-marshaling path.
- `download_video`/`download_audio`/`download_transcript(_timed)` handlers must call `core.ResolveOutputDir(args.OutputDir)` and return an `isError` result (not a Go error) when it's empty, matching the TS "Invalid outputDir: must be within the home or temp directory." behavior.
- `cmd/youtube-mcp/main.go`: `mcp.NewServer(&mcp.Implementation{Name: "youtube-mcp-cli", Version: version}, nil)`, register tools, `server.Run(ctx, &mcp.StdioTransport{})`.

### Definition of Done

- All 10 tools (incl. the 3 alias pairs) registered via `mcp.AddTool` and callable.
- Each alias tool (`get_transcript_timestamps`, `get_video_metadata`, `search_in_transcript`) returns output identical to its canonical counterpart for the same input.
- `download_video`/`download_audio`/`download_transcript(_timed)` return an `isError` `CallToolResult` (not a crash, not an unhandled Go error) when `core.ResolveOutputDir` rejects the given `outputDir`.
- Invalid `url`/missing required args produce a clear `isError` result, not a panic or protocol-level error.
- `cmd/youtube-mcp` starts, connects over stdio, and stays running until the client disconnects or the process is killed.
- `go build ./... && go vet ./... && go test ./...` clean; no regressions in the existing `internal/core`/`internal/cli` test suite.

### Test Plan

- **Unit tests**: none anticipated up front — this task is almost entirely I/O/protocol wiring around already-tested `internal/core` functions. If any pure helper emerges (e.g. building `CallToolResult` content, mapping an alias name to its canonical handler), it gets a TDD-style unit test per the project's standard process (`CLAUDE.md`), same as `pickVttFile`/`qualityFormat` did.
- **Manual smoke test** (per task 9): run `go run ./cmd/youtube-mcp`, connect with a real MCP client (an inspector tool, or Claude Code itself via a temporary `.mcp.json` entry), and call each of the 10 tools once against a real video ID (e.g. `dQw4w9WgXcQ`) — confirm the 3 alias pairs return identical output to their canonical tool, and confirm an intentionally-invalid `outputDir` (e.g. `/etc`) returns `isError: true` rather than crashing the server.

This Definition of Done + Test Plan was written and reviewed *before* starting
implementation, per the project's task-approval process (see `CLAUDE.md`) —
so a future session or reviewer can see exactly what this task was scoped and
approved to do, without having to reconstruct it from an ephemeral Plan Mode
session.

## 8. Add unit tests for core pure functions
- [x] videoid + format tests done as part of task 2 (see above)
- [x] `parseVtt` + transcript formatting/error tests done as part of task 4 (see above)
- [x] `qualityFormat` + download message templates done as part of task 5 (see above)
- [ ] Remaining: cobra/mcpserver-level tests if any pure logic emerges there (most of tasks 6–7 is I/O wiring, likely covered by the task 9 smoke test instead)

## 9. Build + smoke test both binaries
- [~] Partially covered already: `cmd/youtube-cli` builds and was manually smoke-tested (see task 6). Full task 9 (both binaries, MCP client smoke test) still blocked on task 7.

---

## Out-of-band work: BUG-001 and BUG-002 fixed on branch `fix/bug-001-002-subtitle-and-ffmpeg`

Manual smoke-testing `cmd/youtube-cli` against a real video (`dQw4w9WgXcQ`) after task 6
surfaced two real bugs, tracked and fully written up in `docs/BUGS.md`:
- **BUG-001**: transcript/search could 429 or silently return a mistranslated
  caption track, due to an overly broad `--sub-langs` wildcard inherited
  verbatim from the upstream TS project, compounded by naive "first .vtt file"
  selection.
- **BUG-002**: ffmpeg auto-install failures were silently swallowed, and
  `go-ytdlp`'s internal 30s download timeout is provably too short for the
  ~170MB ffmpeg archive on this environment's network (~43s at measured
  throughput) — not transient flakiness, a deterministic mismatch.

Both were root-caused (see `docs/BUGS.md` for full detail incl. exact
`go-ytdlp` source references) and fixed with TDD on a **separate branch**,
per explicit instruction: this repo was git-initialized and an initial commit
made on `main` capturing the state through task 6, then
`fix/bug-001-002-subtitle-and-ffmpeg` was branched off for the fix work.
**`main` has not been touched since** — it still reflects the pre-fix,
task-6-only state. The fix branch is fully committed-ready (build/vet/test
clean, 41 tests passing, both fixes verified end-to-end against the real
video) but **not yet merged** — merging is a decision for the human, to be
asked separately, per "we will try to resolve the problem on that branch
first, if it all passed, we might proceed (ask me again)."

New files on the fix branch: `internal/core/ffmpeg_prewarm.go` (+ test),
`internal/core/ytdlp_test.go`. Modified: `internal/core/transcript.go` (+ test),
`internal/core/ytdlp.go`, `internal/core/download.go`.

The ffmpeg pre-warm fix's platform scope (`windows/amd64`, `linux/amd64`
only) is logged as `docs/DECISIONS.md` DECISION-006, since it's a deliberate
scoping choice, not part of the bug itself.

---

## Session status: PAUSED after task 6 on `main` / bug fixes complete on `fix/bug-001-002-subtitle-and-ffmpeg` (awaiting merge decision)

Tasks 1–6 are done: module scaffolding, all of `internal/core` (videoid, format,
paths, ytdlp wrapper, metadata scraping, transcript, download), and the full
`internal/cli` cobra CLI (`cmd/youtube-cli` builds and runs). 30 unit tests
passing (all in `internal/core`), `go build ./... && go vet ./... && gofmt -l .`
clean. Task 7 (MCP server) and the remainder of task 9 (full smoke test incl.
MCP client) have not been started.

**Process rule in effect for the rest of this project:** strict TDD — for every
pure function, derive ground-truth expected output first (e.g. by running the
equivalent upstream TS logic through Node, as done for `SanitizeTitle`,
`parseVtt`, `transcriptText`/`transcriptTimed`/`formatSearchResult`,
`TranscriptErrorText`, `qualityFormat`), write the test against that ground
truth, confirm it fails to compile/red, then write the minimal implementation
to pass — never adjust the test to match whatever the implementation happens
to produce. I/O-bound and CLI-wiring code (most of tasks 6–7) doesn't fit this
pattern and is instead verified by manual smoke testing. Also: pause after each
numbered task and ask the human whether to continue, and update this ledger
*before* pausing (not after).

## Resume checklist for next session
1. Read this file + `docs/PLAN.md` first.
2. Run `cd go-youtube-mcp-cli && go build ./... && go vet ./... && go test ./...` to confirm the current state still holds (should be clean, 30 tests passing).
3. Start task 7 (`internal/mcpserver`, official `github.com/modelcontextprotocol/go-sdk`) — see task 7's section above for the plan, Definition of Done, and Test Plan (already reviewed/approved; do not start coding on a task without its own DoD + Test Plan filled in first — see `CLAUDE.md`).
4. After task 7, update this ledger, then pause and ask before continuing to task 9 (final combined smoke test — task 8 is already fully covered by tests written during tasks 2, 4, and 5).
