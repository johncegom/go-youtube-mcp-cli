# Task 7: Build internal/mcpserver with official go-sdk (youtube-mcp)

**Status:** done

## Plan (per `docs/PLAN.md`'s "Package-by-package porting notes")

Uses the official `github.com/modelcontextprotocol/go-sdk`.

- Typed input structs per distinct input shape (`urlLangInput{URL, Language}`,
  `searchInput{URL, Query, Language}`, `downloadVideoInput{URL, Quality,
  OutputDir}`, `downloadAudioInput{URL, Format, OutputDir}`,
  `downloadTranscriptInput{URL, Language, OutputDir}`), with `json`/
  `jsonschema` tags mirroring the TS `inputSchema` objects.
- Register all 11 tools via `mcp.AddTool(server, &mcp.Tool{Name: ...,
  Description: ...}, handlerFunc)`: `get_transcript`, `get_transcript_timed`,
  `get_transcript_timestamps` (alias of timed), `get_metadata`,
  `get_video_metadata` (alias), `search_transcript`, `search_in_transcript`
  (alias), `download_video`, `download_audio`, `download_transcript`,
  `download_transcript_timed`. Aliases = two `AddTool` calls pointing at the
  same Go handler function, mirroring the TS `switch` statement's fallthrough
  cases.
- Handlers build `*mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text:
  ...}}, IsError: bool}` directly (confirmed API shape in task 1's `go doc`
  research) and return a zero-value output — matching the TS server's
  `{ content: [...], isError }` pattern exactly, not the SDK's
  structured-output auto-marshaling path.
- `download_video`/`download_audio`/`download_transcript(_timed)` handlers
  must call `core.ResolveOutputDir(args.OutputDir)` and return an `isError`
  result (not a Go error) when it's empty, matching the TS "Invalid
  outputDir: must be within the home or temp directory." behavior.
- `cmd/youtube-mcp/main.go`: `mcp.NewServer(&mcp.Implementation{Name:
  "youtube-mcp-cli", Version: version}, nil)`, register tools,
  `server.Run(ctx, &mcp.StdioTransport{})`.
- This is almost entirely I/O/protocol wiring around already-complete
  `internal/core` functions — no new pure logic expected, so no new unit
  tests anticipated; verify via manual smoke test (e.g. an MCP inspector or
  a temporary client) per task 9.

## Definition of Done

- [x] All 11 tools (8 canonical + 3 aliases) registered via `mcp.AddTool` and callable.
- [x] Each alias tool (`get_transcript_timestamps`, `get_video_metadata`, `search_in_transcript`) returns output identical to its canonical counterpart for the same input.
- [x] `download_video`/`download_audio`/`download_transcript(_timed)` return an `isError` `CallToolResult` (not a crash, not an unhandled Go error) when `core.ResolveOutputDir` rejects the given `outputDir`.
- [x] Invalid `url`/missing required args produce a clear `isError` result, not a panic or protocol-level error.
- [x] `cmd/youtube-mcp` starts, connects over stdio, and stays running until the client disconnects or the process is killed.
- [x] `go build ./... && go vet ./... && go test ./...` clean; no regressions in the existing `internal/core`/`internal/cli` test suite.

## Test Plan

- **Unit tests**: none needed — no new pure logic emerged; this task was, as anticipated, entirely I/O/protocol wiring around already-tested `internal/core` functions.
- **Manual smoke test**: done via a throwaway MCP client program (`github.com/modelcontextprotocol/go-sdk/mcp`'s `NewClient`/`CommandTransport`, spawning `go run ./cmd/youtube-mcp` as a subprocess over stdio — written temporarily as `cmd/mcpsmoke`, deleted before committing, never part of the shipped product). Called all 11 tools against a real video (`dQw4w9WgXcQ`):
  - All 3 alias pairs (`get_transcript_timed`/`get_transcript_timestamps`, `get_metadata`/`get_video_metadata`, `search_transcript`/`search_in_transcript`) returned byte-identical output to their canonical tool.
  - `download_video`/`download_audio` (fire-and-forget) both actually completed — confirmed the resulting `.mp4`/`.mp3` files existed in the temp output dir after the client disconnected, so the detached goroutine survives past session close in practice, not just in theory.
  - `download_transcript`/`download_transcript_timed` produced real `.md` files, confirmed via the returned path.
  - `download_video` with `outputDir: "/etc"` returned `isError: true` with the exact expected message, no crash.
  - `get_transcript` with an invalid URL (`"not-a-valid-url"`) returned `isError: true` with the exact expected message, no crash.
  - `ListTools` confirmed exactly 11 tool entries — 8 canonical tools + 3 aliases (`get_transcript_timestamps`, `get_video_metadata`, `search_in_transcript`); `get_transcript` itself has no alias.
- **Independent re-verification**: re-ran the same throwaway-client technique in a later session against a second, much longer real video (90 min, `Bu0xNDLNORU`) via `get_video_metadata`/`get_transcript_timed` — no errors, no crash, ~4870-line transcript handled cleanly.
- **Fuzz tests** (`internal/mcpserver/tools_test.go`, added after the initial smoke test): the handlers themselves aren't fuzzed — they're I/O-bound (network calls) and fuzzing them in a tight loop would hammer YouTube's endpoints, exactly the anti-pattern flagged in this project's own concurrency analysis. Instead, fuzzed the two pure response-building helpers, since their input can originate from scraped HTML, foreign-language transcripts, or directly from an MCP client's arguments:
  - `FuzzTextResult` — round-trips arbitrary text (including invalid UTF-8, control characters, fmt-verb-shaped strings) through unmodified; 442K executions, zero failures.
  - `FuzzInvalidURLResult` — confirms no panic and a well-formed `isError` result for arbitrary URL input; deliberately does *not* assert literal substring containment, since `%q` intentionally escapes quotes/backslashes for readability (a wrong invariant was caught and fixed before running the fuzzer, not after); 113K executions, zero failures.

This Definition of Done + Test Plan was written and reviewed *before*
starting implementation, per the project's task-approval process (see
`CLAUDE.md`) — so a future session or reviewer can see exactly what this
task was scoped and approved to do, without having to reconstruct it from an
ephemeral Plan Mode session.

## Before starting

Run `go build ./... && go vet ./... && go test ./...` to confirm the current
state holds (should be clean; see `docs/LEDGER.md` for the current test
count).

## After finishing

Update this file's status/checklist, then update `docs/LEDGER.md`'s index
row for task 7, then pause and ask the human before continuing to task 9
(final combined smoke test — task 8 is already fully covered by tests
written during tasks 2, 4, 5, and the fuzz-tests task).
