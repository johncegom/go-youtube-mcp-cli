# Task 6: Build internal/cli with cobra (youtube-cli)

**Status:** done

- [x] `internal/cli/root.go` — `NewRootCommand(version)`, persistent `--quiet` flag, `fatal()` helper (prints `error: ...` to stderr + `os.Exit(1)`, matching the TS CLI's `fatal()` exactly — `SilenceErrors`/`SilenceUsage` set so cobra never double-prints)
- [x] `internal/cli/spinner.go` — braille-frame spinner gated on `--quiet` and `golang.org/x/term.IsTerminal`, matching TS `process.stderr.isTTY` check
- [x] `internal/cli/transcript.go`, `search.go`, `metadata.go`, `download.go` — 1:1 port of the TS subcommands/flags/shorthands, calling the now-complete `internal/core` functions
- [x] `cmd/youtube-cli/main.go` — wires it up, `version` var overridable via `-ldflags`
- [x] `completions`: used cobra's **built-in** `completion` subcommand (bash/zsh/fish/powershell) instead of hand-porting the TS's literal bash/zsh script strings — the deliberate simplification called out in `docs/PLAN.md`. No `internal/cli/completions.go` file exists; nothing needed to be written for it.
- [x] Smoke-tested manually: no-args help output, `--version`, invalid-URL error path (`error: invalid YouTube URL or video ID: "..."`, exit 1, matches TS format), `completion bash` produces a real completion script
- [x] `go build ./... && go vet ./... && gofmt -l .` — clean; `go test ./...` — clean (30 core tests still passing; no new unit tests added here since this layer is I/O/CLI-wiring, consistent with the task 6 kickoff note)

## Known minor divergences from the TS CLI

Logged as deliberate decisions rather than bugs — see `docs/DECISIONS.md`
DECISION-003 (cobra completion vs hand-ported scripts), DECISION-004
(`metadata --json` key order), DECISION-005 (unknown-command message
wording).
