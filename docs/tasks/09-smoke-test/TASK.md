# Task 9: Build + smoke test both binaries

**Status:** in progress

- [x] `cmd/youtube-cli` builds and was manually smoke-tested against a real
      video (see `docs/tasks/06-cli-cobra/TASK.md` and
      `docs/tasks/bugfix-001-002/TASK.md` for the full smoke-test history,
      including the two bugs it surfaced and their fixes).
- [ ] `cmd/youtube-mcp` — blocked on task 7 (not started).
- [ ] Full combined smoke test (both binaries, plus an MCP client exercising
      all 10 tools) — blocked on task 7.

## What's left when task 7 is done

Per `docs/PLAN.md`'s verification section:
- `go build ./...` for both binaries.
- CLI smoke test: `go run ./cmd/youtube-cli transcript <id> --timestamps`,
  `... metadata <id> --json`, `... download <id> --audio` against a real
  public video (already done for the CLI — see task 6/bugfix TASK.md).
- MCP smoke test: run `go run ./cmd/youtube-mcp` and drive it with an MCP
  client (e.g. an inspector, or Claude Code itself via a temporary
  `.mcp.json` entry) to call each of the 10 tools once, confirming alias
  tools return identical results to their canonical counterpart.
