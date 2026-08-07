# Decision Log

Deliberate design/scope decisions made during implementation — choices where
a reasonable reviewer might want a say, or might want to override them
later. This is **not** a bug tracker: if something is *wrong*, it goes in
`docs/BUGS.md` instead (symptom, root cause, fix). A decision here is a
tradeoff that was made on purpose, not a defect.

**Bar for logging:** not every micro-choice belongs here — only decisions
where the alternative was genuinely plausible and a reviewer might
reasonably pick differently. When in doubt, log it; a short unused entry
costs little, a missed real decision costs a confused reviewer later.

Each entry: context, decision, alternatives considered, consequences.

---

## DECISION-001: Lazy (first-use) yt-dlp/ffmpeg auto-install instead of npm-postinstall-time

- **Where:** `internal/core/ytdlp.go`, `EnsureYtDlp`
- **Context:** the upstream TS project auto-downloads `yt-dlp`/`ffmpeg` via
  an npm `postinstall` hook, which runs once at `npm install` time, visibly,
  with the user watching. Go binaries have no equivalent hook.
- **Decision:** install lazily, on first real use of a command that needs
  the binary, via `github.com/lrstanley/go-ytdlp`'s `Install`/`InstallFFmpeg`.
- **Alternatives considered:** require the user to pre-install `yt-dlp`/
  `ffmpeg` themselves and just shell out via `os/exec` (simpler code, but
  breaks the upstream's "works out of the box" experience) — this was
  explicitly put to the human via `AskUserQuestion` during initial planning
  (`docs/PLAN.md`), who chose the auto-download option.
- **Consequences:** turned an install-time failure (loud, visible, easy to
  retry) into a possible first-command-runtime failure. This consequence
  materialized concretely as BUG-002 (`docs/BUGS.md`) — the lazy install
  path swallowed a real `go-ytdlp` timeout limitation silently until fixed.

## DECISION-002: `LogDownloadError` writes to `os.UserCacheDir()` instead of a hardcoded `~/.cache` path

- **Where:** `internal/core/paths.go`, `LogDownloadError`
- **Context:** the upstream TS project hardcodes `~/.cache/youtube-mcp/errors.log`, which is a Unix-only convention and doesn't fit Windows.
- **Decision:** use Go's `os.UserCacheDir()`, which resolves correctly per-OS (`%LOCALAPPDATA%` on Windows, `~/Library/Caches` on macOS, `~/.cache` on Linux).
- **Alternatives considered:** matching the TS behavior exactly (hardcoded `~/.cache`) for strict parity.
- **Consequences:** deliberate, minor, cross-platform-correctness improvement over the upstream; not a faithful-parity break in any way a user would notice (it's an internal error log path).

## DECISION-003: `completions` uses cobra's built-in `completion` subcommand instead of porting the TS's hand-written bash/zsh scripts

- **Where:** `internal/cli/root.go` (no `internal/cli/completions.go` exists)
- **Context:** the TS CLI hand-writes literal bash and zsh completion scripts as string constants.
- **Decision:** rely on cobra's built-in `completion` subcommand (bash/zsh/fish/powershell), rather than hand-porting the TS scripts.
- **Alternatives considered:** hand-port the two TS scripts verbatim for exact output-format parity.
- **Consequences:** functionally equivalent (tab-completion works), broader shell support (fish/powershell for free), but the *generated script text* differs from upstream's if anyone diffs them directly. Called out in `docs/PLAN.md` as a deliberate simplification enabled by choosing cobra.

## DECISION-004: `metadata --json` output has alphabetically-sorted keys, not the TS object's insertion order

- **Where:** `internal/cli/metadata.go`
- **Context:** Go's `encoding/json` always sorts `map[string]string` keys alphabetically when marshaling; the TS version's plain object literal preserves insertion order.
- **Decision:** accept the difference rather than switching to an ordered structure (e.g. a slice of key-value pairs) just to match key order.
- **Alternatives considered:** use an ordered map/slice-based JSON encoder to preserve insertion order.
- **Consequences:** both are valid, equivalent JSON; key order is not semantically meaningful to consumers like `jq`. Judged not worth the added complexity.

## DECISION-005: cobra's built-in unknown-command message differs in wording from the TS custom handler

- **Where:** `internal/cli/root.go`
- **Context:** the TS CLI has a custom `program.on("command:*", ...)` handler with specific wording; cobra has its own built-in unknown-command message.
- **Decision:** accept cobra's default wording rather than overriding it to match the TS text exactly.
- **Alternatives considered:** override cobra's error handling to reproduce the exact TS message.
- **Consequences:** cosmetic difference only in an edge-case error message; not worth the added code to special-case.

## DECISION-006: ffmpeg cache pre-warm (BUG-002's fix) implemented only for `windows/amd64` and `linux/amd64`

- **Where:** `internal/core/ffmpeg_prewarm.go`, `ffmpegPrewarmConfig`
- **Context:** working around `go-ytdlp`'s too-short ffmpeg-download timeout (see `docs/BUGS.md` BUG-002) requires per-platform download URL + archive-format (zip vs tar.xz vs plain-binary) handling.
- **Decision:** implement pre-warming only for the two platforms verifiable/most likely to matter (`windows/amd64`, this environment; `linux/amd64`, the most common CI/server platform). Other platforms (macOS, ARM) silently fall through to `go-ytdlp`'s own retried install path — which may still fail for the same underlying timeout reason on those platforms.
- **Alternatives considered:** implement pre-warming for all platforms `go-ytdlp` itself supports (adds macOS's different download shape — plain binary, not an archive — and ARM variants).
- **Consequences:** BUG-002 is fully fixed on Windows/Linux amd64; on unimplemented platforms, the underlying timeout risk (and BUG-002's original symptom) can still occur, mitigated only by the bounded-retry + clear-error-message part of the fix, not the pre-warm workaround.
