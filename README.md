# go-youtube-mcp-cli

Get YouTube transcripts, metadata, and downloads. You do not need a
YouTube API key. It reads the video page HTML to get metadata. It uses
`yt-dlp` to get transcripts and downloads (see `internal/core/ytdlp.go`).
The tool installs `yt-dlp` automatically the first time you need it.

You can use this tool two ways:

- As a command-line tool (CLI).
- As an MCP server. An MCP server lets AI assistants like Claude use this
  tool directly.

Beyond the basics, it also has: an in-process transcript cache so repeated
tool calls on the same video (get the transcript, then search it, then get
it with timestamps) only shell out to `yt-dlp` once; windowed transcript
retrieval (`get_transcript_range`) instead of always returning the whole
transcript; and observable background downloads —
`download_video`/`download_audio` return a job ID you can poll with
`get_download_status`/`list_downloads`, instead of a blind fire-and-forget
that leaves you guessing whether it worked.

For more details on the design, read `docs/PLAN.md`, `docs/LEDGER.md`,
`docs/BUGS.md`, and `docs/DECISIONS.md`.

This project is open source, under the [MIT License](LICENSE). See
[Credits](#credits) for the project this started from.

## Install

### Option 1: download a release binary

Go to this repo's [Releases](../../releases) page. Download the file for
your system. Unzip it. Put `youtube-cli` and `youtube-mcp` on your `PATH`.

### Option 2: `go install`

You need the Go toolchain:

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

It registers these tools (a few have a name alias, listed alongside the
canonical one — both call the same handler):

| Tool | Purpose |
|------|---------|
| `get_transcript` | Plain transcript text |
| `get_transcript_timed` (alias `get_transcript_timestamps`) | Transcript with a timestamp per segment |
| `get_transcript_range` | Transcript segments between a start/end timestamp only |
| `search_transcript` (alias `search_in_transcript`) | Find a keyword/phrase in the transcript, with timestamps |
| `get_metadata` (alias `get_video_metadata`) | Title, channel, description, publish date, views, duration |
| `get_chapters` | The video's chapters (timestamped table of contents), or a message if it has none |
| `download_video` | Start a background video download; returns a job ID |
| `download_audio` | Start a background audio-only download; returns a job ID |
| `get_download_status` | Check a job ID: running / done (with the real file path) / failed (with the error) |
| `list_downloads` | List every download job known to the running server |
| `download_transcript` | Save the transcript to a Markdown file |
| `download_transcript_timed` | Save the transcript to a Markdown file, with timestamps |

The most reliable way to find `claude_desktop_config.json` is inside
Claude Desktop itself: **Settings → Developer → Edit Config**. Use that
instead of guessing a path by hand, especially on Windows — a
Microsoft-Store-installed copy of an app can be sandboxed (MSIX app
container) so it actually reads from a different, per-package folder even
though the "normal" location below looks right.

For a regular (non-Store) install, the file typically lives at:

- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json` — that's
  `cmd.exe`/Explorer-address-bar syntax; in PowerShell use
  `$env:APPDATA\Claude\claude_desktop_config.json` instead, since `%...%`
  variables aren't expanded there.
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

If the file doesn't exist yet at that path, create it — but check the
in-app menu first if you're not sure which install you have.

**Before editing the config at all, get every `command`/`args` value
working in a plain terminal first.** This is the single biggest
plug-and-play failure point — a wrong or unresolved path fails silently
inside Claude Desktop (the server just never shows up, or errors with no
useful detail), whereas the same mistake in a terminal gives you an
immediate, readable error.

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

Get the exact absolute path first, don't type it from memory:

- **Downloaded/unzipped release binary:** in the folder you unzipped it
  into, run `pwd` (macOS/Linux) or, in PowerShell,
  `(Resolve-Path .\youtube-mcp.exe).Path` (Windows) to print the real
  absolute path — then append the filename if `pwd` alone doesn't include
  it.
- **`go install`:** the binary lands in your Go bin folder, not the repo
  you cloned. Run `go env GOPATH` (or `go env GOBIN` if you've set that
  instead) and append `/bin/youtube-mcp` (`\bin\youtube-mcp.exe` on
  Windows) to whatever it prints.
- On Windows, the path must end in `.exe`. Forward slashes work fine in
  the JSON value (`C:/Users/you/bin/youtube-mcp.exe`) — you do not need to
  escape backslashes, just use forward slashes instead.

**Then verify it directly**, before touching the config: open a terminal
and run that exact path, e.g. `& "C:/Users/you/bin/youtube-mcp.exe"` in
PowerShell or `/absolute/path/to/youtube-mcp` in a POSIX shell. A correctly
resolved binary prints nothing and just hangs, waiting on stdin — that's
correct (it's a stdio server; press Ctrl+C to stop). "command not found"
or "no such file" here means the config's `command` value will fail the
same way, silently, inside Claude Desktop.

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
matter where the client starts the process from. Get that folder's real
absolute path the same way as above (`pwd`, or PowerShell's
`(Resolve-Path .).Path`, run from inside your `go-youtube-mcp-cli`
checkout) rather than typing it from memory — and verify with the same
"run it directly first" check: `go -C <that path> run ./cmd/youtube-mcp`
should hang silently, not error. On Windows, use the full path to `go.exe`
(find it with `(Get-Command go).Source`) — apps with a graphical interface
do not always see your shell's `PATH`.

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

After you change the config, you must fully quit Claude Desktop and open
it again — closing the window is not enough. On Windows, Claude Desktop
can keep running from the system tray after you close its window; right-
click the tray icon and quit from there (or end the process from Task
Manager) if reopening the app doesn't pick up your config change.

**Tip:** if you have a coding agent with file access, you can ask it to
make this Claude Desktop config change for you instead of hand-editing the
JSON yourself. Tell it the path to `youtube-mcp` on your computer, and ask
it to add an entry to your `claude_desktop_config.json` file, using the
JSON example above. Ask it to add the entry, not replace the whole file —
you may already have other MCP servers set up, and you do not want to lose
them.

If the agent you're using is Claude Code itself, though, it's simpler to
register this server directly with Claude Code's own config instead — see
below.

### Claude Code

Claude Code registers MCP servers with its own `claude mcp add` command —
don't hand-edit a config file for this one. Use the same "verify the path
in a plain terminal first" check described above, then run one of:

```sh
# installed or downloaded binary
claude mcp add --scope user youtube -- /absolute/path/to/youtube-mcp

# from source
claude mcp add --scope user youtube -- go -C /absolute/path/to/go-youtube-mcp-cli run ./cmd/youtube-mcp

# Docker
claude mcp add --scope user youtube -- docker run -i --rm youtube-mcp-cli
```

`--scope user` makes it available from every project on your machine
(closest match to how Claude Desktop's config works). Leave it off (the
default is `local`) to scope it to only the project directory you're
currently in, or use `--scope project` to write it to a `.mcp.json` file
checked into that project instead, shared with anyone who clones it.

Confirm it's registered and connects with `claude mcp list` (or
`claude mcp get youtube` for detail on that one server). A session already
running when you add it may not pick it up until you start a new one.

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

## Credits

This project started as a faithful Go port of
[`umbertotancorre/youtube-mcp-cli`](https://github.com/umbertotancorre/youtube-mcp-cli),
a TypeScript project (also MIT-licensed) — thanks to Umberto Tancorre for
the original idea and design. That port (Phase 1) covered the core
transcript/metadata/download functionality one-for-one.

Since then, this repo has grown beyond a port: the transcript cache,
`get_transcript_range`, and download job tracking
(`get_download_status`/`list_downloads`) described above are original work
with no upstream equivalent (see `docs/DECISIONS.md` DECISION-013 for the
full writeup on that scope change).
