# Task 14: Chapters (`get_chapters`)

**Status:** not started (Definition of Done + Test Plan approved 2026-08-28,
before implementation, per the task-approval process in `CLAUDE.md`)

## User need

"Give the agent the video's own structure instead of making it infer one" —
chapters are the video's native table of contents, and pair directly with
task 11's `get_transcript_range` (chapters supply the timestamps to ask
for).

## Problem

`FetchVideoMetadata` (`internal/core/metadata.go`) already scrapes the
description, but chapter information is never parsed, so agents can't
navigate a video by topic without reading the whole transcript.

## Plan

- New pure function `parseChapters(description string) []chapter` in
  `internal/core` (new `chapters.go`), where `chapter{Title string,
  StartSecs float64}`:
  - Recognizes the well-known description format: lines containing a
    timestamp (`M:SS`, `MM:SS`, `H:MM:SS`) followed (or preceded) by a
    title.
  - Applies YouTube's own published validity rules to reject non-chapter
    timestamp mentions: the first chapter must start at `0:00`, there must
    be **≥ 3** entries, and timestamps must be strictly ascending. If the
    candidate list fails any rule, the video has no chapters — return nil,
    never a partial guess.
  - Timestamp parsing reuses task 11's parser (shared helper, not a
    duplicate).
- Fallback tier: if the description yields no chapters, try the `chapters`
  array in the already-fetched `ytInitialPlayerResponse` JSON if present
  (best-effort; same "only fill what's empty" tiering philosophy as
  `FetchVideoMetadata`). If neither yields chapters, that's a clean
  "no chapters" result, not an error.
- New MCP tool `get_chapters(url)` in `internal/mcpserver/tools.go`:
  returns one `[H:MM:SS] Title` line per chapter, or the text
  `"No chapters found for video <id>."` with `isError: false` (absence of
  chapters is a valid answer, not a failure).
- `get_metadata` output unchanged. No new CLI subcommand in this task
  (noted as possible follow-up, not silently added).

## Definition of Done

- [ ] 14.1 `parseChapters` accepts `M:SS`/`MM:SS`/`H:MM:SS`, both
  "timestamp then title" and "title then timestamp" line orders.
- [ ] 14.2 Validity rules enforced: must start at `0:00`, ≥ 3 entries,
  strictly ascending — a description with scattered mid-text timestamp
  mentions (e.g. "at 2:30 he says...") yields **no** chapters.
- [ ] 14.3 Titles trimmed of leading/trailing separators (`-`, `–`, `:`,
  whitespace); empty-title lines skipped.
- [ ] 14.4 Player-response fallback tier used only when the description
  tier yields nothing; malformed JSON in that tier degrades to
  "no chapters", never an error.
- [ ] 14.5 `get_chapters` registered and callable; chaptered video returns
  timed lines; unchaptered video returns the no-chapters message with
  `isError: false`; invalid URL returns the standard invalid-URL
  `isError` result.
- [ ] 14.6 `go build ./... && go vet ./... && go test ./...` clean;
  `gofmt` clean; no regressions.

## Test Plan

- **Unit tests (TDD):** ground truth = fixture descriptions copied from
  real YouTube videos (recorded verbatim in the test file with their video
  IDs, per the Phase-1 fixture-provenance convention) plus YouTube's own
  published chapter rules:
  - A real chaptered description (timestamps parse to the chapters the
    watch page actually shows — verified by eye against the live page and
    recorded in a test comment).
  - A real unchaptered description containing incidental timestamps →
    nil.
  - Trap cases: starts at `0:05` (→ nil), only 2 entries (→ nil),
    out-of-order timestamps (→ nil), `H:MM:SS` long-video chapters.
- **Fuzz test:** `FuzzParseChapters` over arbitrary description text —
  no panic, and any returned chapter list always satisfies the validity
  invariants (starts at 0, ascending, ≥ 3).
- **Manual smoke test (commands):** via throwaway MCP client:
  `get_chapters` against one known chaptered video and one known
  unchaptered video (IDs recorded here at implementation time); confirm
  outputs match the live watch pages.

## Notes / deviations

(fill in during/after implementation)

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
14, then pause and ask the human before starting task 15.
