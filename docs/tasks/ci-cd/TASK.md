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

- [x] `.github/workflows/ci.yml` exists with `build-test` (2-OS matrix) and
  `gofmt` (Ubuntu-only) jobs, triggered on push-to-`main` and
  PRs-targeting-`main`.
- [x] Both jobs actually run and pass on GitHub (confirmed via a real run,
  not just local dry-run) — both on the `ci/github-actions` PR (#1) and
  again on the push-to-`main` merge commit.
- [ ] ~~Branch protection on `main` requires all `build-test` matrix legs
  and `gofmt` to pass before merge.~~ **Blocked, not done** — GitHub
  rejected this: branch protection on a private repo requires GitHub Pro on
  the free tier (`403: Upgrade to GitHub Pro or make this repository public`).
  Put to the human; decided to keep the repo private and skip enforcement
  for now. See `docs/DECISIONS.md` DECISION-007 for the full tradeoff and
  exactly how to re-enable this later.
- [x] Negative-path proof (scaled down from the original plan, since there's
  no branch protection to prove blocks a merge): a deliberately-broken
  change on a scratch branch/PR produces a real red CI run on GitHub, then
  the break is discarded without merging.

## Test Plan

- [x] Local dry run of `go build ./...`, `go vet ./...`, `go test ./...`,
  `gofmt -l .` against current `main` first (clean baseline confirmed).
- [x] Push the workflow branch, open a PR, confirm both jobs go green on
  GitHub for all matrix legs.
- [x] Apply branch protection via `gh api` PUT — **failed with 403** (see
  DoD above); confirmed this is a real platform constraint, not a mistake
  in the request, by checking `gh repo view --json isPrivate` (`true`).
- [x] Scratch branch with an intentional test failure → PR → confirm CI red
  → close/delete without merging (no "merge blocked" assertion, since
  enforcement isn't enabled).
