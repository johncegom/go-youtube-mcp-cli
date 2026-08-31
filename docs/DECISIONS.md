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

**Superseded by DECISION-012** — the repo went public and branch
protection is now enabled. Kept below for the historical record of why it
was blocked in the first place.

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

## DECISION-009: Debian-slim (not scratch/Alpine) for the Docker runtime image

- **Where:** `Dockerfile`, second (`FROM debian:bookworm-slim`) stage
- **Context:** task 10 (packaging) needed a container image running
  `youtube-mcp`/`youtube-cli`. The obvious minimal-image choices for a
  statically-built Go binary are `scratch` (no OS at all) or an Alpine base
  (musl libc, ~5MB). Neither works cleanly here: `internal/core/ytdlp.go`'s
  `EnsureYtDlp` downloads and executes a real, separately-built `yt-dlp`/
  `ffmpeg` binary *at container runtime*, not at image-build time — that
  binary needs a working libc and TLS root certificates to run and to
  download itself in the first place, neither of which `scratch` provides,
  and Alpine's musl libc has a real history of subtle incompatibilities
  with binaries built expecting glibc (the downloaded `yt-dlp`/`ffmpeg`
  releases target glibc).
- **Decision:** use `debian:bookworm-slim` for the final stage, with
  `ca-certificates` explicitly installed.
- **Alternatives considered:** `scratch` (smallest, but breaks the lazy
  install entirely — no shell, no libc, no CA bundle); Alpine (smaller
  than Debian-slim, but risks glibc/musl binary-compatibility failures in
  the very dependency this task exists to package cleanly).
- **Consequences:** a noticeably larger image than `scratch`/Alpine would
  produce, in exchange for the runtime `yt-dlp`/`ffmpeg` auto-install path
  (the project's core value proposition, per DECISION-001) actually working
  inside the container instead of failing on first real tool call.

## DECISION-010: one combined Docker image for both binaries, not two

- **Where:** `Dockerfile`
- **Context:** the project produces two independent binaries
  (`youtube-cli`, `youtube-mcp`) that share 100% of their dependencies and
  build environment.
- **Decision:** build both into a single image (`CMD ["youtube-mcp"]` as
  the default, `youtube-cli` also on `PATH` and reachable by overriding the
  command), rather than publishing two separate images.
- **Alternatives considered:** two images (`youtube-mcp-cli-server`,
  `youtube-mcp-cli-tool`) built from the same Dockerfile via build args/
  targets — doubles the number of images to build, tag, and (eventually)
  publish for two binaries that are a few MB each and share every layer
  anyway; not worth it at this scale.
- **Consequences:** anyone using the CLI via Docker must remember to
  override the default command (`docker run --rm youtube-mcp-cli
  youtube-cli ...`) rather than getting CLI behavior for free — documented
  explicitly in `README.md`.

## DECISION-011: HTTP/SSE MCP transport (hosted, multi-client) explicitly deferred out of task 10

- **Where:** `cmd/youtube-mcp/main.go` (unchanged — still hardcodes
  `&mcp.StdioTransport{}`); `docs/tasks/10-packaging/TASK.md`
- **Context:** task 10 was scoped via an explicit `AskUserQuestion` to the
  human, offering (a) add an unauthenticated `--transport=http` flag now,
  (b) add it with basic token auth, or (c) defer entirely. The human chose
  to defer. The MCP SDK already in `go.mod`
  (`github.com/modelcontextprotocol/go-sdk v1.7.0`) ships
  `mcp.NewStreamableHTTPHandler`/`mcp.NewSSEHandler`, so the library
  capability already exists — the gap is design, not tooling.
- **Decision:** ship task 10 with the MCP server still stdio-only.
  "Runnable ... as a hosted server" is satisfied only in the sense of
  "a container someone can run wherever they run containers, spawned
  locally by its MCP client via `docker run -i`" — not a server multiple
  remote clients connect to concurrently over a network.
- **Alternatives considered:** adding the HTTP transport now, since the
  SDK support already exists. Rejected because several real design
  questions have no answer yet: `internal/core/paths.go`'s
  `ResolveOutputDir`/downloads directory and `LogDownloadError` both assume
  a single local filesystem for a single user — under concurrent remote
  clients there is no per-session isolation, so two callers' downloads (or
  error logs) could collide; there is also no authentication story at all,
  meaning shipping the flag today would make it trivial to accidentally
  expose an open, unauthenticated file-writing service to a network.
- **Consequences:** genuinely remote/multi-client MCP access is not
  possible yet. This is tracked as an open (not yet scoped) future item in
  `docs/LEDGER.md`'s "Current status" rather than a task; scoping it
  properly means answering the isolation/auth questions above first, not
  just flipping on the SDK's existing HTTP handler.

## DECISION-012: repo made public and licensed MIT, superseding DECISION-007

- **Where:** GitHub repo visibility (`johncegom/go-youtube-mcp-cli`);
  `LICENSE` (new file); `README.md` (private-repo language removed)
- **Context:** the project is a Go port of `umbertotancorre/youtube-mcp-cli`,
  which is itself MIT-licensed (confirmed via the GitHub API, not assumed).
  Before this decision, the repo was private with no `LICENSE` file
  (DECISION-007), which also meant branch protection on `main` was blocked
  by GitHub's free-tier restriction (private repos need a paid plan for
  branch protection).
- **Decision:** made the repo public and added a root `LICENSE` (MIT,
  copyright Minh Duong). The copyright line names the port's own author,
  not Umberto Tancorre — verified first that no verbatim upstream
  TypeScript source was ever copied into this repo (ground-truth test
  fixtures were derived by *running* the TS code externally and recording
  its output, e.g. `internal/core/transcript_test.go:9-12`, not by copying
  its source text), so this Go codebase is an independent implementation,
  not a derivative work requiring upstream's copyright notice. Upstream is
  still credited by name in `README.md`. Branch protection on `main` was
  enabled at the same time, now that going public unblocked it for free —
  closing the gap DECISION-007 left open.
- **Alternatives considered:** staying private indefinitely (keeps the
  status quo but leaves DECISION-007's branch-protection gap permanently
  unresolved on the free tier); upgrading to GitHub Pro instead of going
  public (also unblocks branch protection while staying private, but is a
  real billing action, and doesn't serve any purpose for a project already
  built as an open-source port of an open-source project).
- **Consequences:** before flipping visibility, tracked files were checked
  for secrets, tokens, and personal machine paths — none found (the only
  match for common secret-pattern searches was the legitimate
  `${{ secrets.GITHUB_TOKEN }}` reference in `release.yml`). Git history
  was not exhaustively scanned commit-by-commit beyond that; the repo is
  small (21 commits at the time of this decision) and entirely
  project-authored, so residual risk was judged low, but a full
  history/secret scan (e.g. `gitleaks`) remains a reasonable one-time
  follow-up if ever in doubt. Exposure from going public is not fully
  reversible — anything already cloned or cached during the public window
  stays out even if visibility is later reverted.

## DECISION-014: `internal/cli` gets an injectable `exitFunc` seam for testability

- **Where:** `internal/cli/root.go` (`var exitFunc = os.Exit`, used by `fatal()`), `internal/cli/search.go` (`printSearchResult`'s "no matches" branch)
- **Context:** `internal/cli` had zero test coverage — flagged by the test-coverage-hardening audit. The main structural blocker was `fatal()` calling `os.Exit(1)` directly (and `search.go`'s "no matches" branch doing the same inline), which would terminate the test binary itself if exercised in-process.
- **Decision:** introduce a package-level `var exitFunc = os.Exit`, called by `fatal()` and by a new `printSearchResult` helper (extracted from `search.go`'s `RunE`) instead of calling `os.Exit` directly. Tests swap `exitFunc` for a recording stub and restore it via `defer`. Also extracted `formatMetadataJSON`/`formatMetadataPlain` out of `metadata.go`'s `RunE` into pure, directly-testable functions, for the same reason (output-format assertions without needing a live `FetchVideoMetadata` call).
- **Alternatives considered:** leave `fatal`/`os.Exit` untouched and only test what's reachable without hitting an exit path (flag defaults only) — rejected because it was explicitly put to the human via `AskUserQuestion` and the seam was approved as the better tradeoff; running CLI subcommands as a real subprocess (`os/exec`) and asserting on the exit code — works but is slower and heavier per-test than an in-process stub, and doesn't help unit-test the pure formatting logic anyway.
- **Consequences:** `fatal()` and `printSearchResult`'s exit-1 path, and `metadata.go`'s two output formats, are now directly unit-tested (`internal/cli/root_test.go`, `search_test.go`, `metadata_test.go`). Behavior is unchanged in the real binary (`exitFunc` is never reassigned outside tests). `search.go` and `metadata.go` gained two small extracted helper functions as a result — no behavior change, same output as before the refactor.

## DECISION-013: Phase 2 scoped — the "faithful port only" constraint lifted for five approved tasks

- **Where:** `docs/PLAN.md` ("Phase 2" section); `docs/LEDGER.md` tasks
  11–15; `docs/tasks/{11-transcript-cache,12-download-jobs,13-context-search,14-chapters,15-playlist-search}/TASK.md`
- **Context:** Phase 1's governing rule was "no new features beyond what
  the TS version has." With Phase 1 complete, a critical evaluation of the
  MCP tools from the perspective of their actual consumer (an LLM agent)
  identified five structural weaknesses (no caching, unobservable
  fire-and-forget downloads with a misleading tool description, whole-blob
  transcript retrieval, naive per-segment search, single-video-only
  scope) — all present upstream too, so a faithful port can't fix them.
- **Decision:** human approved (2026-08-28) five Phase-2 tasks (11–15)
  that deliberately go beyond upstream. Each got a full Definition of Done
  + Test Plan reviewed *before* any implementation, per the standing
  task-approval process. Ground truth for TDD shifts from "run the
  upstream TS code" to real-world fixtures / published rules / the
  reviewed spec itself, as documented in `docs/PLAN.md`'s "Phase 2
  ground-truth note" — the rest of the TDD discipline is unchanged.
- **Alternatives considered:** staying a pure mirror of upstream
  indefinitely (keeps parity as the single invariant, but freezes in
  upstream's agent-hostile weaknesses); contributing the fixes upstream to
  the TS project instead (doesn't help this Go artifact's users, and the
  two codebases have already deliberately diverged via BUG-001/002 fixes).
- **Consequences:** this repo is no longer a faithful mirror of upstream's
  feature surface — README/tool lists will diverge as tasks 11–15 land,
  and per-task deviations (e.g. task 12's honest download descriptions,
  task 13's changed search output format) each get their own DECISIONS
  entry at implementation time. Only these five tasks are approved;
  anything further still needs its own scoping pass.

## DECISION-015: `download_video`/`download_audio` tool descriptions rewritten to stop claiming they return the file path

- **Where:** `internal/mcpserver/tools.go`, `NewServer`'s tool registration for `download_video`/`download_audio`
- **Context:** both descriptions said "Returns the file path," but the handlers only ever returned a *predicted* path before the detached goroutine's download had even started — a real failure, or an extension differing from the prediction (e.g. mkv instead of mp4 when H.264 isn't available), left an LLM agent confidently reporting a file that doesn't exist or isn't where it said. Task 12 added `get_download_status`/`list_downloads` and a `jobRegistry` (`internal/core/jobs.go`) specifically to make the real outcome observable.
- **Decision:** reworded both descriptions to "Starts a background download ... returns a job ID; use get_download_status to check completion and the final file path," and appended a `Job ID: <id>` line to the existing "Download started" response text (`formatVideoDownloadStarted`/`formatAudioDownloadStarted` in `internal/core/download.go`), rather than restructuring the response into a different shape.
- **Alternatives considered:** make `download_video`/`download_audio` block until completion so "returns the file path" becomes literally true — rejected because it changes the tool's fire-and-forget contract (large videos would hold an MCP request open for however long yt-dlp takes) and was explicitly out of scope for task 12 (see its `TASK.md` "Plan").
- **Consequences:** the tool descriptions and response text are now honest about what's known synchronously vs. what requires a follow-up `get_download_status` call. This is a deliberate divergence from upstream's wording, per task 12's `TASK.md`.

## DECISION-016: `Taskfile.yml` added as an optional convenience wrapper, subordinate to the documented raw commands

- **Where:** `Taskfile.yml` (new file); `CLAUDE.md` "Commands" section
- **Context:** `CLAUDE.md` documents `go build`/`go vet`/`go test`/`gofmt` as "the whole local toolchain," with no Makefile or task runner. Retyping the same command combinations (especially the CI-matching sequence: build, vet, test -race, gofmt check) by hand is repetitive. **Correction (see `docs/tasks/taskfile/TASK.md`):** this addition was initially treated as exempt from the `docs/tasks/<slug>/TASK.md` ceremony on the assumption it only applied to the porting plan — that assumption was wrong (the ceremony is the general anti-drift mechanism for any addition to this repo, already evidenced by prior "Out-of-band:" `docs/LEDGER.md` rows), and a `TASK.md` was added retroactively.
- **Decision:** add `Taskfile.yml` (run via `go-task`, `task <name>`) as a thin wrapper: `build`, `vet`, `test`, `test:race`, `fmt`, `fmt:check`, `check` (mirrors CI's four checks), `cli`/`mcp` (run helpers), `install` (`go install` both binaries), `deps:update` (`go get -u ./... && go mod tidy`, chained into `check` so a breaking update fails loudly instead of landing silently in `go.mod`/`go.sum`), and `go:check-update` (reports installed vs. latest stable Go version — read-only, doesn't touch the system install since that's an admin-level action whose method varies by machine). `CLAUDE.md`'s Commands section was updated to mention it without replacing the raw commands as the source of truth.
- **Alternatives considered:** a Makefile — rejected, no existing precedent in this repo and `go-task`'s YAML syntax is easier to keep consistent with the command table already in `CLAUDE.md`; making `task check` the canonical way to verify changes (replacing the four raw commands in `CLAUDE.md`) — rejected, since not every environment running this repo will have `go-task` installed, so the raw commands must remain authoritative and CI-equivalent on their own.
- **Consequences:** `task` is optional — nothing in CI, hooks, or the required TDD workflow depends on it. `Taskfile.yml` needs to be kept in sync by hand if the raw commands in `CLAUDE.md`/CI ever change (no automated check enforces that).

## DECISION-017: project identity reframed from "Go port" to "standalone product with a porting origin"

- **Where:** `CLAUDE.md` ("What this is"); `docs/PLAN.md` (title, "Context" section)
- **Context:** `CLAUDE.md`'s "What this is" still described the project as "A Go port ... Phase 1 (in progress) is a faithful port ... no new features beyond what the TS version has are in scope yet," and `docs/PLAN.md`'s title was "Port youtube-mcp-cli (TypeScript) to Go." Both were stale: Phase 1 finished long ago and Phase 2 (DECISION-013) already lifted the "faithful port only" constraint and added agent-oriented features with no upstream equivalent (transcript cache, windowed retrieval, download job tracking). `README.md`'s Credits section already reflected this ("this repo has grown beyond a port"); the two most upstream-facing docs (`CLAUDE.md`, `docs/PLAN.md`) had not caught up, and human direction was explicit that the project should now be positioned as its own standalone product rather than framed primarily around the port.
- **Decision:** rewrote `CLAUDE.md`'s "What this is" to lead with what the tool does today, then note its porting origin and Phase 1→2 evolution as history rather than as the primary identity. Retitled `docs/PLAN.md` to "go-youtube-mcp-cli: from Go port to standalone product" and reworded its "Context" section the same way. Left the rest of `docs/PLAN.md` (the Phase 1 porting notes, package-by-package breakdown, Phase 2 task list) untouched — it remains accurate as a historical/design record, just no longer framed as the project's sole reason for existing.
- **Alternatives considered:** rewriting `docs/PLAN.md` end-to-end into a forward-looking product roadmap — rejected as unnecessary scope: the file's *content* (what was built and why) is still correct, only the framing was stale, so a full rewrite would have thrown away an accurate historical record for no benefit. Leaving `CLAUDE.md`/`PLAN.md` as-is and treating this as cosmetic — rejected per explicit human direction that the docs should reflect the current vision, not the Phase 1 starting point.
- **Consequences:** `johncegom/youtube-mcp-cli` vs. `umbertotancorre/youtube-mcp-cli` — `CLAUDE.md` linked the former as the TS source, `README.md`'s Credits section named the latter as the actual upstream original. Flagged to the human and confirmed: `johncegom/youtube-mcp-cli` is a fork of `umbertotancorre/youtube-mcp-cli`, not the original — `CLAUDE.md`'s "What this is" and `docs/PLAN.md`'s "Context" now both link `umbertotancorre/youtube-mcp-cli`, consistent with `README.md`'s Credits section.

## DECISION-018: search output changed from one line per match to context blocks separated by `---`

- **Where:** `internal/core/search.go` (new); `internal/core/transcript.go` (`SearchInTranscript` gains `contextSecs`, old `searchSegments`/`formatSearchResult` deleted); `internal/mcpserver/tools.go` (`searchInput.Context`); `internal/cli/search.go` (`--context` flag)
- **Context:** the prior per-segment `searchSegments`/`formatSearchResult` had two structural weaknesses task 13 exists to fix (per DECISION-013): (1) a query phrase spanning two VTT segment boundaries never matched at all — a silent false negative, since captions split mid-sentence; (2) a matched line, isolated from its surroundings, is often unintelligible without a follow-up call.
- **Decision:** search the merged transcript text as one lowercase stream (so cross-segment phrases match), expand each match to a ±`context`-second window of surrounding segments (default 15s; `context: 0` reproduces the old matched-segment-only shape, now with the false negatives fixed too), merge overlapping/adjacent windows into blocks, and render blocks separated by `---`, with the header's match count reflecting distinct matches found, not the (generally smaller) number of rendered blocks. `search_transcript`/`search_in_transcript` are upgraded in place — no new tool name, same handler for both (alias output stays byte-identical, same pattern as the other alias pairs). CLI `search` gains a matching `--context` flag, default 15.
- **Alternatives considered:** fix only the cross-boundary matching gap and keep the old one-line-per-match shape — rejected, leaves the no-context weakness unaddressed; count context windows in segments instead of seconds — rejected, seconds are pacing-independent and consistent with task 11's `get_transcript_range` timestamp vocabulary; add a separate `search_transcript_v2` tool instead of upgrading in place — rejected, TASK.md's plan was explicit about upgrading in place, and there is no reason an agent would ever prefer the strictly-worse old behavior.
- **Consequences:** the response format for `search_transcript`/`search_in_transcript`/CLI `search` is no longer a faithful port of upstream's shape — the deliberate, pre-flagged deviation DECISION-013 anticipated. The "no matches" message text is unchanged byte-for-byte (`No matches found for %q in video %s.`), so `internal/cli/search.go`'s `printSearchResult` exit-1 detection (a `strings.HasPrefix` check) required no change. The MCP `context` field is `*float64` (a pointer), not a bare `float64` — required because this project's `jsonschema-go` dependency has no `default=` mini-language in its struct-tag format, so "omitted → default 15" vs. "explicit 0 → matched segments only" can only be distinguished via nil-vs-non-nil, not by relying on Go's zero value.
