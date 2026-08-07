# Task 7: Build internal/mcpserver with official go-sdk (youtube-mcp)

**Status:** not started

## Plan (per `docs/PLAN.md`'s "Package-by-package porting notes")

Uses the official `github.com/modelcontextprotocol/go-sdk`.

- Typed input structs per distinct input shape (`urlLangInput{URL, Language}`,
  `searchInput{URL, Query, Language}`, `downloadVideoInput{URL, Quality,
  OutputDir}`, `downloadAudioInput{URL, Format, OutputDir}`,
  `downloadTranscriptInput{URL, Language, OutputDir}`), with `json`/
  `jsonschema` tags mirroring the TS `inputSchema` objects.
- Register all 10 tools via `mcp.AddTool(server, &mcp.Tool{Name: ...,
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

## Before starting

Run `go build ./... && go vet ./... && go test ./...` to confirm the current
state holds (should be clean; see `docs/LEDGER.md` for the current test
count).

## After finishing

Update this file's status/checklist, then update `docs/LEDGER.md`'s index
row for task 7, then pause and ask the human before continuing to task 9
(final combined smoke test — task 8 is already fully covered by tests
written during tasks 2, 4, 5, and the fuzz-tests task).
