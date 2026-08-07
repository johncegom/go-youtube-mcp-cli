# Task 9: Build + smoke test both binaries

**Status:** done

- [x] `cmd/youtube-cli` builds and was manually smoke-tested against a real
      video (see `docs/tasks/06-cli-cobra/TASK.md` and
      `docs/tasks/bugfix-001-002/TASK.md` for the full smoke-test history,
      including the two bugs it surfaced and their fixes).
- [x] `cmd/youtube-mcp` builds and was manually smoke-tested against a real
      video via a throwaway MCP client (see `docs/tasks/07-mcpserver/TASK.md`
      for the full results: all 11 tools called, all 3 alias pairs verified
      byte-identical to their canonical tool, both error paths — invalid
      `outputDir`, invalid URL — confirmed to return `isError` without
      crashing the server).
- [x] Full combined smoke test (both binaries, plus an MCP client exercising
      all 11 tools) — done as part of task 7.

`go build ./...` builds both binaries cleanly; `go vet ./... && go test ./...`
clean throughout.
