# go-youtube-mcp-cli

This is a Go version of [`umbertotancorre/youtube-mcp-cli`](https://github.com/umbertotancorre/youtube-mcp-cli).
It gets YouTube transcripts, metadata, and downloads. You do not need a
YouTube API key. It reads the video page HTML to get metadata. It uses
`yt-dlp` to get transcripts and downloads (see `internal/core/ytdlp.go`).
The tool installs `yt-dlp` automatically the first time you need it.

You can use this tool two ways:

- As a command-line tool (CLI).
- As an MCP server. An MCP server lets AI assistants like Claude use this
  tool directly.

For more details on the design, read `docs/PLAN.md`, `docs/LEDGER.md`,
`docs/BUGS.md`, and `docs/DECISIONS.md`.

> **This repository is private.** All install methods below need an
> account with read access to this repo. This is not a public release.

## Install

### Option 1: download a release binary

Go to this repo's [Releases](../../releases) page. Download the file for
your system. Unzip it. Put `youtube-cli` and `youtube-mcp` on your `PATH`.

### Option 2: `go install`

You need the Go toolchain. You also need git access to this private repo
(for example, an SSH key or a `GOPRIVATE` setting):

```sh
go install github.com/johncegom/go-youtube-mcp-cli/cmd/youtube-cli@latest
go install github.com/johncegom/go-youtube-mcp-cli/cmd/youtube-mcp@latest
```

You can also install a specific version instead of `@latest`.

### Option 3: Docker

First, clone this repo. Then build the image:

```sh
docker build -t youtube-mcp-cli .
```

This repo does not publish the image anywhere. You must build it yourself.

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

For all options, run `youtube-cli --help`. For help on one command, run
`youtube-cli <command> --help`.

With Docker, run:

```sh
docker run --rm youtube-mcp-cli youtube-cli --help
```

The Docker image runs `youtube-mcp` by default. You must name `youtube-cli`
to use it instead.

## MCP server usage

The MCP server (`youtube-mcp`) talks to its client over stdio (standard
input and output). An MCP client, such as Claude Desktop, starts this
server itself. You do not run it by hand.

**From source**, add this to `claude_desktop_config.json`:

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

The `-C <dir>` flag tells `go` to run from that folder. This works no
matter where the client starts the process from. On Windows, use the full
path to `go.exe`. Apps with a graphical interface do not always see your
shell's `PATH`.

**With Docker**, you do not need Go or a source checkout on the machine
that runs Claude Desktop. You only need Docker and the built image:

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

**With an installed or downloaded binary** (the simplest option — no Go,
no Docker, and no source checkout needed at all), set `command` to the
full path of the `youtube-mcp` file. Leave `args` empty:

```json
{
  "mcpServers": {
    "youtube": {
      "command": "/absolute/path/to/youtube-mcp"
    }
  }
}
```

On Windows, use a full path with forward slashes and the `.exe` ending,
for example `"C:/Users/you/bin/youtube-mcp.exe"`. If you used
`go install`, the binary is in your Go bin folder — run `go env GOPATH`
and look inside `bin` there, or run `go env GOBIN` if you set that
instead.

After you change the config, you must fully quit Claude Desktop and open
it again. Closing the window is not enough.

**Tip:** if you have a coding agent with file access (for example, Claude
Code), you can ask it to make this change for you. Tell it the path to
`youtube-mcp` on your computer, and ask it to add an entry to your
`claude_desktop_config.json` file, using the JSON example above. Ask it to
add the entry, not replace the whole file — you may already have other
MCP servers set up, and you do not want to lose them.

### This server runs locally only

`youtube-mcp` only supports stdio. It always runs as a local process
started by its client. It never opens a network port. This repo does not
support running it as a shared server that many remote clients connect to
over HTTP.

Here is why: the tool assumes one user on one machine. It uses one shared
cache for `yt-dlp`, one shared downloads folder, and one shared error log.
To serve many remote users safely, the tool would need separate storage
per user and a way to check who is allowed to connect. Neither exists yet.

The MCP SDK this project uses
(`github.com/modelcontextprotocol/go-sdk`) does support HTTP-based
servers. A future task could add that support once these open questions
are answered. See `docs/LEDGER.md` for details.
