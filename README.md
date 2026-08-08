# go-youtube-mcp-cli

A Go port of [`umbertotancorre/youtube-mcp-cli`](https://github.com/umbertotancorre/youtube-mcp-cli):
YouTube transcript/metadata/download tooling exposed both as an MCP stdio
server and a standalone CLI, with no YouTube API key — metadata comes from
scraping the watch-page HTML, transcripts/downloads come from shelling out
to `yt-dlp` (auto-installed on first use, see `internal/core/ytdlp.go`).

See `docs/PLAN.md`, `docs/LEDGER.md`, `docs/BUGS.md`, and
`docs/DECISIONS.md` for the full design/porting history.

> **This repository is private.** All three install methods below require
> an account with read access to this repo (i.e. an invited collaborator) —
> none of this is public distribution.

## Install

### Option 1: download a release binary

Grab the archive for your platform from this repo's
[Releases](../../releases) page (requires private-repo access), extract it,
and put `youtube-cli` / `youtube-mcp` on your `PATH`.

### Option 2: `go install`

Requires the Go toolchain and git access to this private repo (e.g. via SSH
key or a configured `GOPRIVATE`/credential helper):

```sh
go install github.com/johncegom/go-youtube-mcp-cli/cmd/youtube-cli@latest
go install github.com/johncegom/go-youtube-mcp-cli/cmd/youtube-mcp@latest
```

Or a specific tagged version instead of `@latest`.

### Option 3: Docker

Build the image locally (requires cloning this repo):

```sh
docker build -t youtube-mcp-cli .
```

The image is not published to any registry — build it yourself from a
checkout.

### Option 4: run from source

```sh
go run ./cmd/youtube-cli <args>
go run ./cmd/youtube-mcp
```

## CLI usage

```sh
youtube-cli transcript https://youtu.be/dQw4w9WgXcQ
youtube-cli transcript dQw4w9WgXcQ --timestamps
youtube-cli search dQw4w9WgXcQ "never gonna"
youtube-cli metadata https://youtube.com/watch?v=dQw4w9WgXcQ --json
youtube-cli download dQw4w9WgXcQ --audio --format mp3
```

Run `youtube-cli --help` or `youtube-cli <command> --help` for full flag
details. Via Docker: `docker run --rm youtube-mcp-cli youtube-cli --help`
(the image's default command is `youtube-mcp`, so `youtube-cli` must be
named explicitly).

## MCP server usage

The MCP server (`youtube-mcp`) speaks MCP over stdio — it's meant to be
spawned by an MCP client (e.g. Claude Desktop), not run interactively.

**From source**, in `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "youtube": {
      "command": "go",
      "args": ["-C", "/absolute/path/to/go-youtube-mcp-cli", "run", "./cmd/youtube-mcp"]
    }
  }
}
```

(`go -C <dir> run ...` resolves `go.mod` relative to `<dir>`, so this works
regardless of the client's own working directory. On Windows, use an
absolute path to `go.exe` itself — GUI-launched processes don't reliably
inherit a shell's `PATH`.)

**Via Docker** (no Go toolchain or source checkout needed on the machine
running Claude Desktop, only Docker + the already-built image):

```json
{
  "mcpServers": {
    "youtube": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "youtube-mcp-cli"]
    }
  }
}
```

**Via an installed/downloaded binary**, point `command` directly at the
`youtube-mcp` binary's path with no `args`.

Claude Desktop must be fully quit and relaunched (not just the window
closed) to pick up any config change.

### Scope note: local only, not a hosted/remote server

`youtube-mcp` only implements the stdio transport — it is always spawned
as a local subprocess of its MCP client, never listens on a network port.
Nothing here supports running it as a shared/remote server that multiple
clients connect to over HTTP: the tool's design (per-machine `yt-dlp`
cache, download directory allowlisted to the local home/temp dirs, a
single local error log) assumes one local user, and serving multiple
remote clients would need real answers for shared download-directory
collisions, per-session isolation, and authentication that don't exist
yet. The underlying MCP SDK (`github.com/modelcontextprotocol/go-sdk`)
does support HTTP-based transports for a future task that scopes those
questions properly — see `docs/LEDGER.md`.
