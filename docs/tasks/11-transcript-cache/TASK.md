# Task 11: Transcript cache + windowed retrieval (`get_transcript_range`)

**Status:** not started (Definition of Done + Test Plan approved 2026-08-28,
before implementation, per the task-approval process in `CLAUDE.md`)

## User need

"Ask questions about a long video without re-downloading the transcript on
every tool call, getting 429'd, or blowing my context window."

## Problem

- Every `get_transcript` / `get_transcript_timed` / `search_transcript` /
  `download_transcript(_timed)` call independently re-runs yt-dlp for the
  same video and throws the parsed segments away
  (`internal/core/transcript.go`, `fetchSegments`). A typical agent
  workflow is multi-call on one video, so this is 3–4× slower than
  necessary and multiplies exposure to YouTube's 429 rate limiting (the
  same class of problem as BUG-001).
- `get_transcript` returns the entire transcript as one text blob — tens of
  thousands of tokens for a long video — with no way to retrieve just a
  window.

## Plan

- New `internal/core/transcache.go`:
  - In-memory cache `map[cacheKey][]transcriptSegment` behind a `sync.Mutex`
    (or `RWMutex`), keyed by `videoID + "|" + language`.
  - TTL per entry (default 15 minutes) and an entry-count cap (default 32
    videos, evict oldest) to bound memory. Clock injectable for tests.
  - Negative results (fetch errors) are **not** cached — a transient 429
    shouldn't poison the cache for its TTL.
  - `fetchSegments` checks the cache first; on miss it fetches, stores, and
    returns. All existing public entrypoints get caching transparently.
  - Disk-backed cache is explicitly **out of scope** (memory-only covers
    the within-session agent workflow; revisit only with a concrete need).
- New pure function `filterSegmentsByRange(segments, startMs, endMs)` and a
  timestamp parser accepting `M:SS`, `MM:SS`, `H:MM:SS` (reuse/extend
  existing format helpers where possible).
- New MCP tool `get_transcript_range(url, start, end, language)` in
  `internal/mcpserver/tools.go`:
  - `start`/`end` are timestamps (`"12:30"`); either may be omitted
    (open-ended range). Returns timed-format lines for segments whose
    offset falls inside `[start, end]`.
  - Invalid timestamps, `start > end`, or an empty window return an
    `isError` text result (matching the server's existing error style),
    never a panic or protocol error.
- CLI: no new subcommand in this task (MCP is where the pain is); noted as
  a possible follow-up, not silently added.

## Definition of Done

- [ ] 11.1 `transcache.go` exists; a second `fetchSegments` call for the same
  `videoID+language` within the TTL does **not** invoke yt-dlp again
  (verified in tests via an injectable fetch function + call counter).
- [ ] 11.2 TTL expiry causes a re-fetch; the entry cap evicts and stays
  bounded (tested with an injected clock).
- [ ] 11.3 Fetch errors are returned to the caller but not cached; the next
  call retries.
- [ ] 11.4 Concurrent calls for the same key are safe (`go test -race`
  clean); duplicate concurrent fetches are acceptable, corruption is not.
- [ ] 11.5 `filterSegmentsByRange` + timestamp parsing handle: `M:SS` /
  `MM:SS` / `H:MM:SS`, open-ended start/end, out-of-bounds windows (empty
  result, not error), and reject malformed input.
- [ ] 11.6 `get_transcript_range` registered and callable; happy path
  returns only the windowed segments in timed format; bad input returns
  `isError: true` with a clear message.
- [ ] 11.7 All existing transcript tools/CLI commands behave identically
  from the outside (same output for the same input) — caching is invisible
  except for speed.
- [ ] 11.8 `go build ./... && go vet ./... && go test ./...` clean;
  `gofmt` clean; no regressions.

## Test Plan

- **Unit tests (TDD, pure logic):**
  - Timestamp parser: table-driven, ground truth from YouTube's own
    timestamp display conventions + the existing `FormatTimestamp`
    round-trip.
  - `filterSegmentsByRange`: hand-built segment fixtures; boundary
    inclusion at exact `start`/`end` offsets defined in the test first.
  - Cache: injectable fetcher + clock; assert call counts for hit, miss,
    TTL expiry, error-not-cached, eviction at cap. Run with `-race`.
- **Unit tests (mcpserver):** range-tool input validation paths (invalid
  timestamp, start > end) via direct handler calls with a canned core —
  if handler seams make this awkward, validation logic is factored into a
  pure helper and tested there instead.
- **Manual smoke test (commands):**
  1. `go run ./cmd/youtube-mcp` via a throwaway MCP client (same technique
     as task 7): call `get_transcript` then `search_transcript` then
     `get_transcript_timed` on the same video (`dQw4w9WgXcQ`); confirm via
     temporary debug logging (removed before commit) that yt-dlp ran once.
  2. `get_transcript_range` with `start: "0:30", end: "1:00"` on the same
     video; confirm only that window's lines come back.
  3. `get_transcript_range` with `start: "abc"` → `isError: true`.

## Notes / deviations

(fill in during/after implementation)

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
11, then pause and ask the human before starting task 12.
