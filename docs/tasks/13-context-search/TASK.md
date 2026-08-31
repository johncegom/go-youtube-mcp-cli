# Task 13: Context-aware transcript search

**Status:** done (2026-09-01). Definition of Done + Test Plan approved
2026-08-28, before implementation, per the task-approval process in
`CLAUDE.md`.

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

- [x] 13.1 A query phrase spanning two adjacent segments is found
  (demonstrated by a test that fails against the old `searchSegments`
  behavior — red first).
- [x] 13.2 Existing single-segment matches still found; case-insensitivity
  preserved; "no matches" message preserved for zero hits.
- [x] 13.3 Context windows correct at transcript start/end boundaries (no
  panic, no phantom segments).
- [x] 13.4 Overlapping/adjacent match windows merge into one block; match
  *count* still reflects distinct matches, not blocks.
- [x] 13.5 `context: 0` returns matched segments only (old-style behavior,
  minus the false negatives).
- [x] 13.6 `search_transcript` + alias accept the optional `context` field;
  alias output remains byte-identical to canonical for identical input.
- [x] 13.7 CLI `search --context` works and defaults sensibly.
- [x] 13.8 Output format documented (in the tool description and this
  file); deviation from upstream's search output logged in
  `docs/DECISIONS.md`.
- [x] 13.9 `go build ./... && go vet ./... && go test ./...` clean;
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

- **New file:** `internal/core/search.go` holds `buildMergedStream`,
  `segmentForCharIndex`, `searchSegmentsWithContext`,
  `formatSearchResultWithContext` — kept separate from
  `transcript.go` rather than added inline, since it's a distinct
  self-contained algorithm with its own tests.
- **Dead code removed:** the old `searchSegments`/`formatSearchResult`
  (`internal/core/transcript.go`) and their direct tests
  (`TestFormatSearchResult_Found`/`_NoMatches`,
  `TestSearchSegments_CaseInsensitive` in `transcript_test.go`) were
  deleted once `SearchInTranscript` was rewired to the new functions —
  per this repo's no-orphaned-dead-code norm. Superseded one-for-one by
  the new tests in `internal/core/search_test.go`.
- **Reused `filterSegmentsByRange` (task 11) as-is** for populating each
  merged block's segments — no changes needed, its existing
  offset-only-inclusive-bound contract (already relied on by
  `get_transcript_range`) is exactly right for this too.
- **Real-world observation (not a bug):** the manual smoke test showed
  that even with `context: 0`, a block can include one extra segment
  beyond the two the match spans. Root cause: YouTube's real
  auto-generated captions have overlapping display durations — segment
  N's `Offset+Duration` commonly extends past segment N+1's `Offset` —
  so `filterSegmentsByRange`'s offset-inclusion rule legitimately pulls
  in that next segment too. This is the same behavior `get_transcript_range`
  already has on the same data (task 11), not something task 13
  introduced; not a DoD violation, since the block still only contains
  segments genuinely "within" the computed window per the established
  (offset-based) definition.
- **Test infrastructure decisions:** used a hand-built `searchFixture`
  (in `search_test.go`) with a real cross-segment phrase, since the
  existing `sampleSegments` fixture (`transcript_test.go`) has no
  boundary-spanning phrase and is still used by other, unrelated tests
  (`TestTranscriptText`/`TestTranscriptTimed`) so was left untouched.
- **Manual smoke test** (2026-09-01): `rfscVS0vtbw` (freeCodeCamp Python
  course, same video used for task 14's smoke test), query
  `"programming in Python"`. Old per-segment search would have missed
  the very first match entirely — segment 0 ends "...get started
  programming", segment 1 starts "in Python. Now, ..." — the phrase only
  exists once the two are joined. New search found this and 5 further
  matches, `context: 0` correctly narrowed each block to (approximately)
  just the matched segment(s). Verified via both `core.SearchInTranscript`
  called directly and `go run ./cmd/youtube-cli search rfscVS0vtbw
  "programming in Python" --context 0`, output identical in shape.

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
13, log the output-format deviation in `docs/DECISIONS.md`, then pause and
ask the human before starting task 14.
