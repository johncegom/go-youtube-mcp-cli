# Continuous Improvement Log

A running retrospective on the **way of working** — process, tooling,
human/assistant collaboration — not a bug tracker or decision log (see
`docs/BUGS.md` / `docs/DECISIONS.md` for those, which are about the
*product*) and not a per-task changelog (that's `TASK.md`'s own
"Notes / deviations" section).

**Bar for logging (generalization test):** an entry belongs here only if
the lesson would change how a *different, future* task gets approached.
"Here's what happened in this task" alone belongs in that task's own
`TASK.md`, not here — this file exists specifically to avoid duplicating
that.

Each entry that clears the bar answers three questions:

- **What went well** — a process/tooling/collaboration pattern worth
  deliberately repeating on future tasks.
- **What could be improved about how the work got done** — a concrete
  friction point (a slow tool choice, a missing check, an ambiguous
  instruction). Name it concretely; "polish more" is not an entry.
- **Advice for next time** — one or two concrete, actionable takeaways: a
  pattern to reuse, a pitfall to avoid, a check to run earlier.

Skim this file before starting a new task for any still-relevant advice —
an entry that's written once and never rereads isn't continuous
improvement.

---

## RETRO-001: Task 10 (packaging/distribution)

- **What went well:** putting the two genuinely ambiguous scope questions
  (distribution mechanism, hosted-MCP ambition) to the human via
  `AskUserQuestion` *before* writing any files paid off directly — the
  answers ("all three" / "defer entirely") shaped the Dockerfile base-image
  choice and kept the task from ballooning into an HTTP-transport redesign
  nobody asked for. Sub-task tracking in `TASK.md` (added mid-task, now a
  standing rule above) also worked exactly as intended: when the goreleaser
  validation stalled on a slow first-time dependency download, the rest of
  the task's state stayed legible instead of needing to be re-derived.
- **What could be improved:** local verification of `.goreleaser.yaml` used
  `go run github.com/goreleaser/goreleaser/v2@latest`, which compiles
  goreleaser's entire CLI — including cloud-publisher SDKs (Azure, GCS,
  etc.) this repo's config never touches — before it can run `check` even
  once. That made the "quick sanity check" step by far the slowest part of
  the whole task, and it was still unresolved when the task paused. A
  faster path exists and wasn't used.
- **Advice for next time:** for CLI tools used only for local
  validation (not becoming a project dependency), prefer installing a
  pinned release binary directly (`go install
  github.com/goreleaser/goreleaser/v2@v2.x.y` still compiles from source
  and hits the same problem — for goreleaser specifically, its published
  GitHub Release binaries or `winget`/`scoop`/`brew` package are prebuilt
  and skip the compile-from-source step entirely) over `go run
  pkg@latest`, especially for tools known to have a large dependency
  surface. Check the tool's own install docs for a prebuilt-binary option
  before defaulting to `go run`/`go install` for anything larger than a
  small single-purpose CLI.

## RETRO-002: bare `printf | run` is not a valid way to smoke-test a stdio server

- **What went well:** when the first stdio smoke-test attempt produced a
  confusing error, the instinct was to isolate the variable (reproduce the
  same test against the plain local binary, outside Docker entirely)
  before concluding the container was broken — that took a few minutes and
  prevented misdiagnosing a test-harness bug as a packaging bug.
- **What could be improved about how the work got done:** the first attempt
  to verify an MCP server's stdio handshake used `printf '<json>' | run`.
  This pipes exactly one line into stdin and then closes it the instant
  `printf` returns — before the server necessarily has time to write its
  response — so it reliably produces a misleading `Fatal error: server is
  closing: EOF` regardless of whether the server actually works. This isn't
  specific to Docker or to this task; it would misfire against *any*
  stdio-based server tested the same way.
- **Advice for next time:** never validate a stdio JSON-RPC/MCP server with
  a one-shot `printf | run` pipe — it tests pipe-closing behavior, not the
  server. Use a real client that keeps the connection open and reads the
  response before closing (for this project: `mcp.NewClient(...).Connect`
  with `mcp.CommandTransport` from `github.com/modelcontextprotocol/go-sdk`,
  wrapping whatever command needs testing — local binary, `docker run -i`,
  or a release binary — as a small throwaway program). Write it once,
  parameterize the command, reuse it for every future stdio smoke test in
  this repo instead of reinventing a pipe test each time.
