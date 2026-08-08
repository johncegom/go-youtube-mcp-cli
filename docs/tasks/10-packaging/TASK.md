# Task 10: Package the project into runnable distributable artifacts (CLI + local-MCP)

**Status:** done, with one pre-existing, out-of-scope issue flagged for a
human decision (10.6's gofmt/CRLF noise — not introduced by this task, see
below).

## Context

Phase 1 (the faithful Go port) is done and merged (`docs/LEDGER.md`). Until
now the only way to run either binary was `go build ./...` / `go run
./cmd/...` from a full source checkout with the Go toolchain installed — no
README, no cross-compiled release binaries, no container image, `go
install` never verified end-to-end. Requested: package the source so it's
runnable as a **CLI** and as an **MCP server**, distributable as more than
"clone the repo and build it yourself."

Two scope-defining questions were put to the human via `AskUserQuestion`
before implementation started:

1. **Distribution mechanism** — chosen: all three of GoReleaser + GitHub
   Releases (cross-compiled binaries), `go install` support, and a Docker
   image.
2. **Hosted MCP transport (HTTP/SSE)** — chosen: **defer entirely**. The
   MCP server stays stdio-only. The current tool design
   (`ResolveOutputDir`, `LogDownloadError`, `EnsureYtDlp`'s shared
   lazy-install cache) assumes a single local user on a single machine;
   exposing this over HTTP to remote/multiple clients raises real
   unaddressed questions (shared download directory across callers, no
   per-session isolation, no auth) that deserve their own scoping pass.
   The SDK (`github.com/modelcontextprotocol/go-sdk v1.7.0`, already in
   `go.mod`) already ships `mcp.NewStreamableHTTPHandler`/`mcp.NewSSEHandler`
   for whenever that follow-up task happens.

So "runnable as MCP, usable as ... hosted server" in this task means: a
downloadable/installable binary or container that runs the existing stdio
server the same way Claude Desktop already spawns it — not a new
hosted-transport feature, not multi-tenant remote access.

**Critical-thinking catch, stated explicitly:** the repo is **private**
(`docs/DECISIONS.md` DECISION-007). A GitHub Release binary,
`go install .../cmd/youtube-cli@<tag>`, and `docker build` from a clone all
only work for accounts with read access to the private repo. This is not
public distribution — it's "anyone already invited as a collaborator can
install this without building from source." The README states this
plainly.

## Design decisions

- **GoReleaser** for cross-compiled release binaries: `{linux,darwin,windows}
  x {amd64,arm64}`, skipping windows/arm64 (low value). Version injected via
  `-X main.version={{.Version}}` into the existing `var version = "dev"` in
  both `cmd/*/main.go` — no source change needed.
- **Debian-slim runtime image**, not `scratch`/Alpine: `go-ytdlp`'s lazy
  installer downloads and runs a real `yt-dlp`/`ffmpeg` binary at container
  runtime, so libc + `ca-certificates` must actually be present.
- **One combined image** with both binaries on `PATH`, `CMD ["youtube-mcp"]`
  default (overridable to `youtube-cli ...`), rather than two separate
  images — avoids duplicating the build stage for two nearly-identical
  images when both binaries are small and share all dependencies.
- **`release.yml` separate from `ci.yml`**, triggered only on `v*` tag push,
  using the auto-provided `GITHUB_TOKEN` — no new secret needed.

## Out of scope (explicit)

- HTTP/SSE MCP transport, auth, multi-tenant file storage.
- Publishing/pushing the Docker image to any registry.
- Making the repo public.

## Definition of Done (tracked as sub-tasks)

- [x] **10.1** `.goreleaser.yaml` written (root, v2 schema): builds
  `youtube-cli`/`youtube-mcp` for `{linux,darwin,windows} x {amd64,arm64}`
  (windows/arm64 excluded), ldflags version injection, archives, checksums,
  changelog, GitHub release config.
  - [x] **10.1a** `goreleaser check` — confirmed clean (`.goreleaser.yaml`
    validated, 1 config file). Took a long time on first run purely because
    `go run github.com/goreleaser/goreleaser/v2@latest` compiles
    goreleaser's entire CLI, including cloud-publisher SDKs (Azure, GCS,
    sigstore, etc.) this repo's config never touches — not a sign of a
    config problem, just a slow one-time module download (see
    `docs/RETRO.md` RETRO-001).
  - [x] **10.1b** `goreleaser release --snapshot --clean --skip=publish` —
    succeeded (`release succeeded after 6m48s`), produced all 5 expected
    archives in `dist/` (`{linux,darwin}_{amd64,arm64}` + `windows_amd64`),
    each containing `youtube-cli`, `youtube-mcp`, and `README.md`; version
    string correctly resolved to `0.0.0-SNAPSHOT-5d1be65` in snapshot mode
    (no tag exists yet, expected). `dist/` deleted after inspection and
    added to `.gitignore` (it's goreleaser's own build-output directory,
    not something to commit).
- [x] **10.2** `.github/workflows/release.yml` written, triggered on `v*`
  tag push, `goreleaser-action` + auto `GITHUB_TOKEN`. Mirrors 10.1's
  config; not fired against GitHub in this task (no tag pushed).
- [x] **10.3** `Dockerfile` written (multi-stage, `golang:1.26-bookworm` →
  `debian:bookworm-slim`, both binaries on `PATH`, `CMD ["youtube-mcp"]`).
  - [x] **10.3a** `docker build -t youtube-mcp-cli .` — succeeds (verified).
  - [x] **10.3b** `docker run --rm youtube-mcp-cli youtube-cli --help` —
    prints usage correctly (verified).
  - [x] **10.3c** MCP round-trip over stdio in the container — **confirmed
    working.** The earlier `printf | docker run -i` attempt was a
    test-harness artifact, not a packaging defect (see the now-superseded
    note originally here, kept in git history). Re-verified properly with a
    throwaway Go client using the same SDK
    (`mcp.NewClient(...).Connect(ctx, &mcp.CommandTransport{Command:
    exec.CommandContext(ctx, "docker", "run", "-i", "--rm",
    "youtube-mcp-cli")}, nil)`, written to a temporary `.smoketest/` dir
    inside the module so it could resolve `go.mod`'s dependencies, deleted
    immediately after — never committed): connects cleanly and
    `ListTools` returns all 11 registered tools. Confirms the containerized
    server is a fully working drop-in replacement for the local
    `go run ./cmd/youtube-mcp` form.
- [x] **10.4** `go install ./cmd/youtube-cli` and `./cmd/youtube-mcp`
  (local path form) — both succeed, verified runnable from a scratch
  `GOBIN` outside the repo's working directory.
- [x] **10.5** `README.md` written: what this is, all three install paths
  with the private-repo caveat stated explicitly, CLI examples, MCP config
  examples (local `go run` and Docker forms), explicit "local only, not
  hosted/remote" scope note.
- [~] **10.6** `go build ./...`, `go vet ./...`, `go test ./...` all pass
  (verified). `gofmt -l .` is **not** clean, but the 3 files it flags
  (`cmd/youtube-mcp/main.go`, `internal/mcpserver/tools.go`,
  `internal/mcpserver/tools_test.go`) are untouched by this task (confirmed
  via `git status` — no working-tree changes to them) and flagged only
  because they're checked out with CRLF line endings on this machine (the
  known DECISION-008 phenomenon — `.gitattributes` normalizes newly
  re-checked-out files but doesn't retroactively rewrite files already
  sitting in the working tree with CRLF). Not this task's to fix
  silently — no packaging file this task added is a `.go` file, so this
  task introduces zero new gofmt violations. Flagging for a human decision
  rather than touching unrelated files: worth a `git add --renormalize .`
  as a separate, explicit action outside this task's diff.
- [x] **10.7** `docs/LEDGER.md` updated (task 10 row added, "Current
  status" updated with the deferred-HTTP-transport note).
- [x] **10.8** `docs/DECISIONS.md` updated: DECISION-009 (Debian-slim base),
  DECISION-010 (one combined image), DECISION-011 (HTTP transport deferral).

**All sub-tasks complete.** The one open item, 10.6's gofmt/CRLF flag, is
deliberately left unresolved here rather than silently fixed — it predates
this task, touches files this task never modified, and per this project's
own scope-drift rule a fix belongs in its own explicit action
(`git add --renormalize .`), not folded into this task's diff.

## Test Plan

- **Unit-testable:** none — no new Go logic, only build/packaging tooling.
- **Manual smoke tests (all run, all passed):**
  - [x] `goreleaser check` — 1 config file validated.
  - [x] `goreleaser release --snapshot --clean --skip=publish` — 5 archives
    produced, `youtube-cli`/`youtube-mcp`/`README.md` each present inside.
  - [x] `docker build -t youtube-mcp-cli .` — succeeds.
  - [x] `docker run --rm youtube-mcp-cli youtube-cli --help` — prints usage.
  - [x] MCP round-trip against the container via a throwaway Go SDK client
    (`mcp.NewClient` + `mcp.CommandTransport` wrapping `docker run -i --rm
    youtube-mcp-cli`) — connects, `ListTools` returns all 11 tools.
  - [x] `go install ./cmd/youtube-cli && go install ./cmd/youtube-mcp` to a
    scratch `GOBIN`, ran both installed binaries — work correctly outside
    the repo's working directory.
  - [x] `go build ./...`, `go vet ./...`, `go test ./...` — all pass.
  - [~] `gofmt -l .` — **not** clean, but the 3 flagged files are untouched
    by this task (pre-existing CRLF working-tree state, see 10.6). Zero new
    gofmt violations introduced by any file this task added.
- **Not tested in this task (as scoped):** an actual tag-triggered run of
  `release.yml` against GitHub (no tag pushed); no image registry
  push/pull.

## Notes / deviations

- No deviations from the approved plan. One scope clarification surfaced
  mid-task and resolved via `docs/CLAUDE.md` process updates rather than
  task-10 file changes: sub-task-level progress tracking in `TASK.md` (now
  a standing rule) and a new `docs/RETRO.md` continuous-improvement log
  (way-of-working lessons, not product features) — both prompted directly
  by friction encountered while running this task (a slow goreleaser
  validation step, needing to resume cleanly after a compaction).
- The first attempt at verifying 10.3c (MCP stdio round-trip in the
  container) used a bare `printf ... | docker run -i --rm ...` pipe and
  produced a misleading `Fatal error: server is closing: EOF`. Confirmed
  this was a test-harness artifact (stdin closing before a response could
  be observed, reproduced identically against the plain local binary
  outside Docker) rather than a real defect, then re-verified properly with
  a throwaway MCP SDK client that keeps the connection open — see
  `docs/RETRO.md` for whether this is worth its own generalized entry.
