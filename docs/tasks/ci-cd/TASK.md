# Out-of-band: add CI/CD (GitHub Actions + branch protection)

**Status:** in progress

## Context

Requested to guard against regressions as task 7 (the biggest remaining
chunk of new code) begins: "don't break anything" had only been enforced by
manually running `go build`/`go vet`/`go test`/`gofmt` before each commit —
fragile, since nothing stops broken code landing on `main` if a step is
skipped. GitHub Actions CI runs the same checks automatically on every
push/PR, and branch protection makes a red run actually block merging.

## Design decisions

- **Platform**: GitHub Actions (repo already lives on GitHub).
- **OS matrix**: `ubuntu-latest` + `windows-latest` for `go build`/`go vet`/
  `go test` — the codebase has real Windows-specific branches
  (`internal/core/paths.go`'s `ResolveOutputDir` case-insensitive path-prefix
  check; `internal/core/ffmpeg_prewarm.go`'s `windows/amd64` zip-extraction
  path), so Ubuntu-only would leave those uncovered.
- **`gofmt` check runs Ubuntu-only**, not in the matrix. This session hit
  real CRLF/LF false-positives from `gofmt -l` multiple times, caused by
  Windows checking out files with CRLF (`core.autocrlf=true` locally).
  Verified directly (`git show HEAD:<file> | cat -A`) that the actual
  git-committed blobs are LF — GitHub's Ubuntu runner checks out LF and
  `gofmt` sees the real formatting correctly there; running the check only
  once avoids a spurious red build from line-ending noise unrelated to
  actual code formatting.
- **Enforcement**: branch protection on `main` requires all `build-test`
  matrix legs + `gofmt` to pass before merge — not just a visible badge.

## Definition of Done

- `.github/workflows/ci.yml` exists with `build-test` (2-OS matrix) and
  `gofmt` (Ubuntu-only) jobs, triggered on push-to-`main` and
  PRs-targeting-`main`.
- Both jobs actually run and pass on GitHub (confirmed via a real run, not
  just local dry-run).
- Branch protection on `main` requires all `build-test` matrix legs and
  `gofmt` to pass before merge.
- Negative-path proof: a deliberately-broken change on a scratch branch/PR
  produces a red CI run and GitHub visibly blocks merging it; the break is
  then reverted/discarded.

## Test Plan

- Local dry run of `go build ./...`, `go vet ./...`, `go test ./...`,
  `gofmt -l .` against current `main` first (clean baseline confirmed).
- Push the workflow branch, open a PR, confirm both jobs go green on GitHub
  for all matrix legs.
- Apply branch protection via `gh api` PUT, verify via GET.
- Scratch branch with an intentional test failure → PR → confirm CI red +
  merge blocked → close/delete without merging.
