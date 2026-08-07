# Task 1: Init Go module + scaffold directory structure

**Status:** done

- [x] `go mod init github.com/johncegom/go-youtube-mcp-cli`
- [x] Created `cmd/youtube-cli`, `cmd/youtube-mcp`, `internal/core`, `internal/cli`, `internal/mcpserver`
- [x] Added deps: `modelcontextprotocol/go-sdk/mcp`, `lrstanley/go-ytdlp`, `spf13/cobra`, `golang.org/x/term` (`golang.org/x/sync` came in transitively, available for errgroup use)

## Notes

Confirmed API shapes via `go doc` before coding — `ytdlp.Command` builder
methods (`SkipDownload`, `WriteAutoSubs`, `WriteSubs`, `SubLangs`,
`SubFormat`, `FFmpegLocation`, `Run`), `ytdlp.Install`/`InstallFFmpeg`/
`ResolvedInstall`, and `mcp.NewServer`/`mcp.AddTool`/`ToolHandlerFor`/
`CallToolResult`/`TextContent`/`StdioTransport`.
