# Task 13: Context-aware transcript search

**Status:** not started (Definition of Done + Test Plan approved 2026-08-28,
before implementation, per the task-approval process in `CLAUDE.md`)

## User need

"Find where they discuss X and show me enough around it to understand" —
and have the search actually *find* it even when the phrase straddles a
caption-segment boundary.

## Problem

`searchSegments` (`internal/core/transcript.go`) is per-segment,
case-insensitive substring matching:

1. VTT segments split mid-sentence, so a query phrase spanning two segments
   ("machine learning" split as "...machine" / "learning...") never
   matches at all — silent false negatives.
2. A match returns only its own fragmented segment line, with no
   surrounding context, which is often unintelligible alone and forces the
   agent into follow-up calls.

## Plan

Pure-logic task (no new I/O) — the most TDD-friendly of Phase 2.

- New search core in `internal/core` (extend `transcript.go` or a new
  `search.go`):
  - Build a merged lowercase text stream from all segments joined with
    single spaces, plus an offset map from merged-stream position back to
    segment index.
  - Match the query against the merged stream (still plain substring,
    case-insensitive — regex support is explicitly out of scope for this
    task), so cross-segment phrases hit.
  - For each match, expand to a context window: all segments whose offset
    lies within ±`contextSeconds` (default 15) of the matched segment(s)'
    time range.
  - Merge overlapping/adjacent windows into one block; render blocks as
    timed lines separated by `---`, with the total match count in the
    header (evolving the existing `formatSearchResult` shape).
- MCP surface: upgrade `search_transcript` (and its `search_in_transcript`
  alias) **in place** — new optional `context` input field (seconds,
  default 15; `0` = matched segments only). The response format change vs.
  upstream is a deliberate deviation → log in `docs/DECISIONS.md` at
  implementation time. No new tool name.
- CLI: `search` subcommand gains the same optional `--context` flag wired
  to the same core function (kept trivially thin).
- Uses task 11's cache automatically (search goes through `fetchSegments`);
  no direct dependency on task 11's code beyond that.

## Definition of Done

- [ ] 13.1 A query phrase spanning two adjacent segments is found
  (demonstrated by a test that fails against the old `searchSegments`
  behavior — red first).
- [ ] 13.2 Existing single-segment matches still found; case-insensitivity
  preserved; "no matches" message preserved for zero hits.
- [ ] 13.3 Context windows correct at transcript start/end boundaries (no
  panic, no phantom segments).
- [ ] 13.4 Overlapping/adjacent match windows merge into one block; match
  *count* still reflects distinct matches, not blocks.
- [ ] 13.5 `context: 0` returns matched segments only (old-style behavior,
  minus the false negatives).
- [ ] 13.6 `search_transcript` + alias accept the optional `context` field;
  alias output remains byte-identical to canonical for identical input.
- [ ] 13.7 CLI `search --context` works and defaults sensibly.
- [ ] 13.8 Output format documented (in the tool description and this
  file); deviation from upstream's search output logged in
  `docs/DECISIONS.md`.
- [ ] 13.9 `go build ./... && go vet ./... && go test ./...` clean;
  `gofmt` clean; existing search tests updated only where the *spec*
  changed (each such change justified here, not silently rewritten).

## Test Plan

- **Unit tests (TDD):** hand-built segment fixtures with known
  boundary-spanning phrases. Ground truth is derived by specification, not
  from upstream — there is no upstream behavior to match; this task is
  deliberately *better* than upstream, which is exactly what
  `docs/DECISIONS.md` exists to record. Cases: cross-boundary phrase;
  phrase entirely inside one segment; match at segment 0 and at the last
  segment with full context requested; two matches whose windows overlap;
  two matches far apart; `context: 0`; query with regex metacharacters
  treated literally; empty/whitespace query rejected (existing behavior
  preserved).
- **Fuzz test:** extend the existing fuzz-test convention — fuzz the
  merged-stream builder + window expansion with arbitrary segment text
  (including invalid UTF-8) asserting no panic and window indices always
  in bounds.
- **Manual smoke test (commands):** via throwaway MCP client against a
  real video: search a phrase confirmed (by reading the timed transcript)
  to straddle a segment boundary; confirm it now matches and the context
  block reads coherently. Repeat via `go run ./cmd/youtube-cli search`.

## Notes / deviations

(fill in during/after implementation)

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
13, log the output-format deviation in `docs/DECISIONS.md`, then pause and
ask the human before starting task 14.
