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

## DECISION-007: CI runs and reports status, but branch protection is not enabled — repo stays private

- **Where:** `.github/workflows/ci.yml`; GitHub repo settings (not a file in this repo)
- **Context:** the CI/CD plan (`docs/tasks/ci-cd/TASK.md`) called for branch protection on `main` requiring the CI checks to pass before merge, so a red run actually *blocks* bad code, not just reports it. GitHub's API rejected this (`403: Upgrade to GitHub Pro or make this repository public to enable this feature`) — branch protection on a private repo requires a paid plan on the free tier.
- **Decision:** keep the repo private and skip branch protection for now. CI still runs and reports pass/fail on every push and PR (a real, visible signal), it just can't technically prevent a merge.
- **Alternatives considered:** make the repo public (enables branch protection immediately, free, but exposes the source publicly); upgrade to GitHub Pro (keeps it private, requires an actual billing action the user would have to do themselves, not something doable via the API/CLI). Both put to the human explicitly; both declined in favor of staying private without enforcement, for now.
- **Consequences:** "everything currently will not break" is only *detected* (visible red CI), not *enforced* (a human could still click merge on a failing PR). Revisit if/when the repo goes public or the plan is upgraded — at that point, re-run `gh api repos/johncegom/go-youtube-mcp-cli/branches/main/protection` (PUT) with `required_status_checks.contexts: ["build-test (ubuntu-latest)", "build-test (windows-latest)", "gofmt"]`, the exact check names already confirmed working in `docs/tasks/ci-cd/TASK.md`.

## DECISION-008: `.gitattributes` forces `eol=lf` repo-wide instead of touching `core.autocrlf`

- **Where:** `.gitattributes` (new file, repo root)
- **Context:** throughout development on Windows (`core.autocrlf=true` locally), `git status` repeatedly flagged files as modified immediately after a tool wrote to them — even though `git diff` showed zero real content differences and the committed blobs were confirmed pure LF. Root cause: tool-driven file writes don't go through git's own checkout "smudge" filter (which is what sets mtimes/line-endings in a way git's index cache expects under `autocrlf=true`), so the index's cached stat info repeatedly went stale, and `git status`'s dirty-check path disagreed with `git diff`'s content-comparison path on how to reconcile it.
- **Decision:** add `.gitattributes` with `* text=auto eol=lf`. An explicit `eol` attribute in `.gitattributes` takes precedence over `core.autocrlf` for any path it covers, so this fixes the problem without touching git config at all (respecting the project's "never update git config" rule) — verified directly: rewrote a tracked file via the same tool that caused the churn all session, `git status` stayed clean afterward.
- **Alternatives considered:** setting `core.autocrlf=false` locally/globally (works, but requires a git config change, and only fixes it on machines where that config is set — `.gitattributes` fixes it for anyone who clones the repo, regardless of their local config); doing nothing and periodically running `git add` to refresh the index (works as a one-off per BUG-triage in this session, but doesn't address the root cause and recurs indefinitely).
- **Consequences:** `git add --renormalize .` was run once to apply the new rule to all currently-tracked files — it found nothing to change (every tracked blob was already pure LF), so this was a purely additive fix with zero content rewrite.
