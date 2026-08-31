# go-youtube-mcp-cli: from Go port to standalone product — Phase 1 + Phase 2

## Context

`johncegom/youtube-mcp-cli` is a small TypeScript npm monorepo that gives AI agents and CLI users transcript/metadata/download access to YouTube videos, without needing an API key. It works by scraping YouTube's watch-page HTML for metadata and shelling out to `yt-dlp` for captions/downloads. It ships two consumable binaries — an MCP stdio server and a `youtube-cli` command — built on a shared `core` package.

Phase 1's goal was a faithful **Go port** of that same three-binary/one-core shape, living in this (`go-youtube-mcp-cli`) directory — no functionality beyond what the TS version already did. Phase 1 is complete (see `docs/LEDGER.md`). Phase 2 (below) deliberately lifted that constraint: this project is no longer scoped to upstream parity, and is evolving as its own standalone product with agent-oriented features that have no TS equivalent. Phase 1's plan is kept below as the historical record of what was ported and why.

Confirmed technology choices (per user):
- **yt-dlp/ffmpeg binaries**: auto-download on first run via `github.com/lrstanley/go-ytdlp`, matching the original npm package's `postinstall` auto-download behavior.
- **MCP SDK**: official `github.com/modelcontextprotocol/go-sdk`.
- **CLI framework**: `github.com/spf13/cobra` (the de-facto Go analog to `commander`, used by the TS CLI).

## Source repo analysis (reference)

Fetched directly from GitHub (not cloned locally — this was a design/analysis pass):
- `packages/core/src/index.ts` (485 lines) — all shared logic:
  - `extractVideoId` — regex + URL parsing for `youtu.be`, `/shorts/`, `/embed/`, `/v/`, `?v=`.
  - `formatTimestamp` / `formatDuration` / `parseIsoDuration` / `sanitizeTitle` — pure string/number formatting.
  - `getDownloadsDir` / `resolveOutputDir` (path allowlist: home dir + temp dir only) / `logDownloadError`.
  - `findBinaryPath` / `buildYtdlArgs` — wraps `ytdlp-nodejs` binary discovery + ffmpeg location flag injection.
  - `fetchVideoMetadata` — fetches the watch page HTML, extracts `ytInitialPlayerResponse` JSON, falls back to JSON-LD `<script type="application/ld+json">`, falls back further to individual `<meta>` tags (`og:title`, `itemprop`, etc.).
  - `parseVtt` / `fetchSegments` — runs `yt-dlp --write-auto-sub --write-sub --sub-format vtt` into a temp dir, parses the resulting `.vtt` file into timed segments, dedupes.
  - `getTranscriptText` / `getTranscriptTimed` / `searchInTranscript` / `saveTranscriptFile` (writes a Markdown file with metadata header).
  - `startVideoDownload` / `startAudioDownload` — fire-and-forget (`execFile` + `unref()`), used by MCP (returns immediately with predicted path).
  - `downloadVideoBlocking` / `downloadAudioBlocking` — blocking (`spawn` with `stdio: inherit`), used by CLI (streams yt-dlp's own progress bar to the terminal).
  - `QUALITY_FORMAT_MAP` — five named quality presets mapped to yt-dlp format-selector strings.
- `packages/cli/src/index.ts` (279 lines) — `commander`-based CLI: `transcript`, `search`, `metadata`, `download`, `completions` subcommands, a spinner for TTY progress, hand-written bash/zsh completion scripts.
- `packages/mcp/src/index.ts` (331 lines) — stdio MCP server exposing 11 tools (3 are name aliases of a canonical tool): `get_transcript`, `get_transcript_timed`, `get_transcript_timestamps` (alias), `get_metadata`, `get_video_metadata` (alias), `search_transcript`, `search_in_transcript` (alias), `download_video`, `download_audio`, `download_transcript`, `download_transcript_timed`.

No test files exist anywhere in the source repo.

## Target Go module layout

```
go-youtube-mcp-cli/
  go.mod                          # module github.com/johncegom/go-youtube-mcp-cli
  cmd/
    youtube-cli/main.go           # cobra root, wires subcommands
    youtube-mcp/main.go           # mcp.NewServer + AddTool registrations + StdioTransport
  internal/
    core/
      videoid.go                  # extractVideoId
      format.go                   # formatTimestamp, formatDuration, parseIsoDuration, sanitizeTitle
      paths.go                    # getDownloadsDir, resolveOutputDir, logDownloadError
      ytdlp.go                    # go-ytdlp binary install/lookup wrapper, buildBaseCommand-equivalent
      metadata.go                 # fetchVideoMetadata (HTML scraping)
      transcript.go               # fetchSegments (vtt via yt-dlp), parseVtt, getTranscriptText/Timed, searchInTranscript, saveTranscriptFile, transcriptErrorText
      download.go                 # QUALITY_FORMAT_MAP, startVideoDownload/startAudioDownload, downloadVideoBlocking/downloadAudioBlocking
    cli/
      transcript.go, search.go, metadata.go, download.go, completions.go, root.go, spinner.go
    mcpserver/
      tools.go                    # tool registration + typed input structs + handlers
  README.md
```

One Go module, two `cmd/` binaries, shared `internal/core` — the idiomatic Go equivalent of the TS monorepo's three npm packages. `internal/` (not `pkg/`) since nothing here is meant for external import.

## Package-by-package porting notes

**`internal/core`** — direct port of `packages/core/src/index.ts`, split into files by concern (Go convention favors many small files over one 485-line file):
- `extractVideoId`: same regex (`^[a-zA-Z0-9_-]{11}$`) + `net/url.Parse` switch on host/path, same as the TS `URL` handling.
- `fetchVideoMetadata`: `net/http` client with `context.WithTimeout(15s)` and the same `User-Agent`; same three-tier fallback (regex-extracted `ytInitialPlayerResponse` JSON → JSON-LD `<script type="application/ld+json">` → individual `<meta>` tag regexes). Go's `regexp` (RE2) supports `(?s)` for dot-matches-newline, equivalent to JS's `s` flag used in the TS source.
- `parseVtt`: direct line-for-line port — split on blank lines, find the `-->` line, strip tags/entities, dedupe via a `map[string]struct{}` keyed on `offset|text`.
- Binary resolution (`findBinaryPath`/`buildYtdlArgs` in TS) becomes a thin `internal/core/ytdlp.go` wrapper around `go-ytdlp`:
  - On first need, call `ytdlp.MustInstall(ctx, nil)` (or the non-panicking `Install` variant with proper error propagation) to auto-download `yt-dlp`/`ffmpeg` into the OS cache dir if not already present/on PATH — mirrors the npm `postinstall` step but performed lazily at first real use instead of at install time (Go binaries have no postinstall hook).
  - Use `go-ytdlp`'s builder (`ytdlp.New()...`) instead of hand-built `[]string` args for subtitle fetch and downloads — e.g. `.SkipDownload()`, `.WriteAutoSub()`, `.WriteSubs()`, `.SubLangs(lang+".*")`, `.SubFormat("vtt")`, `.Output(...)`, `.NoWarnings()`, `.Quiet()` for the subtitle path; `.Format(...)`, `.MergeOutputFormat("mp4")`, `.Output(...)` for video; `.ExtractAudio()`, `.AudioFormat(fmt)`, `.Format("bestaudio/best")` for audio.
- `fetchSegments`: use `os.MkdirTemp` (handles unique naming natively, unlike the TS `Date.now()` suffix) + `dl.Run(ctx, url)` (blocking, captures stdout/stderr) with a `context.WithTimeout(30s)` for the cancellation the TS version does via `setTimeout`+`child.kill()`.
- Fire-and-forget downloads (MCP path): launch via `go dl.RunWithOptions(...)` in a goroutine that logs failures through `logDownloadError`, mirroring `execFile(...).unref()`; return the predicted path string immediately, same as TS.
- Blocking downloads (CLI path): `dl.Run(ctx, url)` with stdout/stderr wired to `os.Stdout`/`os.Stderr` so yt-dlp's own progress bar streams through, mirroring `spawn(..., { stdio: "inherit" })`.
- `resolveOutputDir`'s allowlist (home dir + temp dir, case-insensitive prefix match on Windows) ports directly using `os.UserHomeDir()` / `os.TempDir()` / `filepath`.
- `logDownloadError`: switch the original hardcoded `~/.cache/youtube-mcp/errors.log` to `os.UserCacheDir()` + `"youtube-mcp"` — same intent, but correct on Windows (`%LOCALAPPDATA%`) instead of assuming a Unix-style `.cache` folder. Called out as a deliberate, minor cross-platform fix, not scope creep.
- Concurrent metadata+segments fetch in `saveTranscriptFile` (TS `Promise.all`): use two goroutines + a 2-slot error channel, or `golang.org/x/sync/errgroup`.

**`internal/cli`** — cobra port of `packages/cli/src/index.ts`:
- Root command carries the persistent `--quiet` flag and version (from a `var version = "dev"` overridable via `-ldflags`, replacing `pkg.version` import).
- `transcript`, `search`, `metadata`, `download` subcommands map 1:1 to the TS commands, same flags/shorthands (`-l/--language`, `-t/--timestamps`, `-s/--save`, `-j/--json`, `-a/--audio`, `-q/--quality`, `-f/--format`), same defaults, same examples text via cobra's `Example` field.
- Spinner: small `internal/cli/spinner.go` reproducing the Braille-frame spinner gated on `quiet` and `term.IsTerminal(os.Stderr.Fd())` (via `golang.org/x/term`), matching the TS `process.stderr.isTTY` check.
- `completions`: use **cobra's built-in `completion` subcommand** (bash/zsh/fish/powershell) instead of hand-porting the TS's two literal shell-script strings — this is a deliberate simplification enabled by using cobra, called out explicitly rather than silently dropped. If exact output-format parity with the original scripts matters, flag this and we'll hand-port them instead.
- Unknown-command and no-args-prints-help behavior ports directly (cobra supports both natively).

**`internal/mcpserver`** — port of `packages/mcp/src/index.ts` onto `modelcontextprotocol/go-sdk`:
- One typed input struct per distinct input shape (e.g. `urlLangInput{URL, Language string}`, `searchInput{URL, Query, Language string}`, `downloadVideoInput{URL, Quality, OutputDir string}`, etc.) with `json`+`jsonschema` tags mirroring the TS `inputSchema` objects (including `enum` constraints for `quality`/`format` via jsonschema tags, and defaults noted in `Description`).
- Each tool handler follows the SDK's `func(ctx, *mcp.CallToolRequest, Input) (*mcp.CallToolResult, Output, error)` shape; since all responses are single free-text blocks (no structured output consumed elsewhere), handlers build `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: ...}}, IsError: bool}` directly and return a zero-value `Output{}` — same pattern the TS server uses (`{ content: [...], isError }`).
- Alias tools (`get_transcript_timed`/`get_transcript_timestamps`, `get_metadata`/`get_video_metadata`, `search_transcript`/`search_in_transcript`) are registered via two `mcp.AddTool` calls pointing at the same handler function, exactly mirroring the TS `switch` statement's fallthrough `case` pairs.
- `main.go`: `mcp.NewServer(&mcp.Implementation{Name: "youtube-mcp-cli", Version: version}, nil)`, register all 11 tools, `server.Run(ctx, &mcp.StdioTransport{})`.

## Dependencies (go.mod)

- `github.com/modelcontextprotocol/go-sdk` — MCP server.
- `github.com/lrstanley/go-ytdlp` — yt-dlp binary management + command building.
- `github.com/spf13/cobra` — CLI framework (pulls in `spf13/pflag`).
- `golang.org/x/term` — TTY detection for the spinner.
- (optional) `golang.org/x/sync/errgroup` — concurrent metadata+transcript fetch.

No dependency is needed for YouTube API access — same as the original, everything is unauthenticated scraping + local `yt-dlp`.

## Verification

- **Unit tests** (none exist in the source repo, but Go convention and the pure-function nature of this code make them cheap and valuable — add for the ported logic, not new scope):
  - `internal/core/videoid_test.go` — table-driven tests for `extractVideoId` covering bare ID, `youtu.be`, `watch?v=`, `/shorts/`, `/embed/`, invalid inputs.
  - `internal/core/format_test.go` — `formatTimestamp`, `formatDuration`, `parseIsoDuration`, `sanitizeTitle` edge cases (hours boundary, illegal filename chars, empty title).
  - `internal/core/transcript_test.go` — `parseVtt` against a fixture VTT string, verifying dedupe and tag/entity stripping.
- **Build**: `go build ./...` for both binaries.
- **CLI smoke test**: run `go run ./cmd/youtube-cli transcript dQw4w9WgXcQ --timestamps`, `... metadata dQw4w9WgXcQ --json`, `... download dQw4w9WgXcQ --audio` against a real public video, confirm output/format matches the TS CLI's behavior (this exercises the go-ytdlp auto-install path on a clean machine/cache too).
- **MCP smoke test**: run `go run ./cmd/youtube-mcp` and drive it with an MCP client (e.g. the `mcp` CLI inspector, or Claude Code itself via a temporary `.mcp.json` entry) to call each of the 11 tools once, confirming the 3 alias tools return identical results to their canonical counterpart.

---

# Phase 2 — beyond the faithful port (approved 2026-08-28)

Phase 1 (the faithful port) is complete — see `docs/LEDGER.md`. Phase 2
deliberately goes **beyond** upstream's feature set, driven by a critical
evaluation of the MCP tools from the perspective of their actual consumer
(an LLM agent). The "no new features beyond the TS version" constraint that
governed Phase 1 is hereby lifted for these five scoped tasks only (see
`docs/DECISIONS.md` DECISION-013); anything further still needs its own
scoping pass.

The evaluation found five structural weaknesses:

1. **No caching** — every transcript tool call re-runs yt-dlp for the same
   video; a typical agent workflow (metadata → transcript → search) is
   multi-call, so this multiplies latency and YouTube 429 exposure.
2. **Fire-and-forget downloads are a black hole** — `StartVideoDownload`
   returns a *predicted* path immediately; failures go to a log file no MCP
   client reads, and the tool descriptions falsely say "Returns the file
   path."
3. **Token-cost blindness** — `get_transcript` returns a whole transcript
   as one blob; no windowed/range retrieval for long videos.
4. **Naive search** — per-segment substring matching means phrases spanning
   two VTT segments never match, and matches come back with no surrounding
   context.
5. **Single-video only** — no playlist awareness; cross-video questions
   ("which episode covers X?") are impossible.

## Phase 2 tasks (implementation order = dependency order)

Each task has its own `docs/tasks/<slug>/TASK.md` with a full Definition of
Done + Test Plan, per the standing task-approval process. Summary:

- **Task 11 — Transcript cache + windowed retrieval**
  (`docs/tasks/11-transcript-cache/TASK.md`): in-memory TTL cache of parsed
  segments keyed by `videoID+language`, transparently used by all existing
  transcript tools; new `get_transcript_range` MCP tool for
  timestamp-windowed retrieval. Fixes weaknesses 1 and 3. Foundation for
  tasks 13 and 15.
- **Task 12 — Download job tracking**
  (`docs/tasks/12-download-jobs/TASK.md`): job registry for fire-and-forget
  downloads; `download_video`/`download_audio` return a job ID; new
  `get_download_status` and `list_downloads` MCP tools; honest tool
  descriptions. Fixes weakness 2.
- **Task 13 — Context-aware transcript search**
  (`docs/tasks/13-context-search/TASK.md`): match against a merged text
  stream so cross-segment phrases hit; return each match with a
  configurable context window of surrounding segments; merge overlapping
  windows. Upgrades `search_transcript` in place. Fixes weakness 4.
- **Task 14 — Chapters** (`docs/tasks/14-chapters/TASK.md`): parse
  YouTube chapters from the video description (already scraped); new
  `get_chapters` MCP tool. Complements task 11's `get_transcript_range`
  (chapters give the agent the timestamps to ask for).
- **Task 15 — Playlist listing + cross-video search**
  (`docs/tasks/15-playlist-search/TASK.md`): resolve playlist entries via
  `yt-dlp --flat-playlist`; new `list_playlist` and `search_playlist` MCP
  tools; per-video failures reported inline, never aborting the whole
  search. Depends on task 11's cache. Fixes weakness 5.

## Phase 2 ground-truth note

Phase 1's TDD rule derived ground truth by running the upstream TS code.
Phase 2 features have **no upstream equivalent**, so ground truth comes
from: (a) real-world fixtures (e.g. actual YouTube video descriptions for
chapter parsing, real VTT files for search fixtures), (b) published,
verifiable rules (e.g. YouTube's own documented chapter requirements), and
(c) the task's own reviewed specification (its Definition of Done) for
behavior that is genuinely new — the same convention already used for
`pickVttFile` in BUG-001's fix. The rest of the TDD discipline (test first,
red before green, never bend a test to match the code) is unchanged.
