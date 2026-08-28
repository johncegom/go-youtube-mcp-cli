# Task 15: Playlist listing + cross-video search

**Status:** not started (Definition of Done + Test Plan approved 2026-08-28,
before implementation, per the task-approval process in `CLAUDE.md`).
**Depends on task 11** (transcript cache) — do not start before task 11 is
done.

## User need

"Which video in this series/playlist talks about X?" — cross-video
questions an agent is uniquely suited to answer but currently can't,
because every tool is single-video.

## Problem

- All existing tools take one video URL/ID; playlist URLs aren't recognized
  at all (`ExtractVideoID` has no playlist concept).
- Without task 11's cache, iterating a playlist's transcripts would hammer
  YouTube's caption endpoint — the exact 429 pattern BUG-001 documented.

## Plan

- New `internal/core/playlist.go`:
  - `ExtractPlaylistID(url)` — pure; recognizes `youtube.com/playlist?list=`
    and `watch?v=...&list=...` forms, and bare playlist IDs
    (`PL...`/`UU...`/`OLAK...` prefixes); rejects everything else.
  - `ListPlaylistVideos(ctx, playlistID)` — I/O; one
    `yt-dlp --flat-playlist --print "%(id)s\t%(title)s"`-style invocation
    via `NewYtDlpCommand` (never a hand-built `exec.Cmd`, per the standing
    rule); returns ordered `(videoID, title)` pairs. Output-line parsing is
    a separate pure function (unit-testable on captured fixture output).
  - Hard cap: first **25** entries of larger playlists, with a "showing
    first 25 of N" note in tool output — bounds both runtime and 429
    exposure.
- Cross-video search: `SearchPlaylist(ctx, playlistID, query, language)`:
  - Fetches each video's transcript **sequentially** (not concurrently —
    deliberate 429 protection; the cache from task 11 absorbs repeat
    calls), with a small fixed inter-video delay (e.g. 500ms) between
    cache-miss fetches.
  - Per-video failures (no captions, fetch error) are collected and
    reported inline at the end ("skipped N videos: ...") — one bad video
    never aborts the whole search. Aggregation/formatting is a pure
    function.
  - Match lines formatted as `<video title> [MM:SS] <text>`, grouped by
    video, using task 13's context-aware search if it has landed, else the
    then-current search core (whichever `searchSegments`-successor exists —
    resolve at implementation time and note it here).
- New MCP tools in `internal/mcpserver/tools.go`:
  - `list_playlist(url)` — numbered `title (videoID)` lines + total count.
  - `search_playlist(url, query, language)` — aggregated results as above;
    empty query rejected like `search_transcript`; invalid playlist URL →
    `isError: true`.
- Scope decision to log in `docs/DECISIONS.md` at implementation time:
  first Phase-2 feature with **no upstream counterpart at all** (11–14
  extend existing surfaces; this adds a new input domain).

## Definition of Done

- [ ] 15.1 `ExtractPlaylistID` handles `playlist?list=`, `watch?v=...&list=...`,
  bare IDs; rejects video-only URLs and garbage (table-driven tests).
- [ ] 15.2 Flat-playlist output-line parsing correct on captured real
  yt-dlp output, including titles containing tabs/unicode; malformed lines
  skipped, not fatal.
- [ ] 15.3 25-entry cap enforced with the "showing first 25 of N" note.
- [ ] 15.4 Result aggregation: matches grouped per video in playlist
  order; per-video failures listed at the end; zero-match playlists get a
  clear "no matches" message (pure function, unit-tested).
- [ ] 15.5 One video failing (e.g. no captions) does not abort the search —
  demonstrated in tests with a fake fetcher.
- [ ] 15.6 Sequential fetch with inter-fetch delay on cache misses
  (verifiable via injectable fetcher/clock in tests — no real timing
  asserts).
- [ ] 15.7 `list_playlist` + `search_playlist` registered and callable;
  invalid playlist URL and empty query produce `isError: true`.
- [ ] 15.8 `go build ./... && go vet ./... && go test ./...` clean;
  `gofmt` clean; no regressions.

## Test Plan

- **Unit tests (TDD):**
  - `ExtractPlaylistID`: ground truth = real playlist URL shapes collected
    from live YouTube (recorded in the test file), same convention as the
    Phase-1 `ExtractVideoID` tests.
  - Line parsing: fixture = captured verbatim output of a real
    `yt-dlp --flat-playlist --print` run against a small public playlist
    (provenance comment in the test file).
  - Aggregation/formatting + partial-failure behavior: spec-derived ground
    truth (no upstream equivalent — the reviewed DoD above is the spec, per
    the Phase-2 ground-truth note in `docs/PLAN.md`), driven through a fake
    per-video fetcher.
- **Fuzz test:** `FuzzExtractPlaylistID` (arbitrary URL input, no panic) —
  consistent with the existing untrusted-input-parser fuzz convention.
- **Manual smoke test (commands):** via throwaway MCP client against a
  small (≤10 video) public playlist chosen at implementation time:
  1. `list_playlist` → titles/order match the live playlist page.
  2. `search_playlist` for a term known to appear in exactly one video →
     that video, correct timestamp.
  3. Re-run the same search immediately → visibly faster (cache hits;
     confirmed via temporary debug logging, removed before commit).
  4. A playlist containing at least one captionless video → search
     completes with that video listed as skipped.

## Notes / deviations

(fill in during/after implementation)

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
15, log the new-input-domain scope decision in `docs/DECISIONS.md`, then
pause and ask the human — Phase 2 as approved ends here; anything further
needs a new scoping pass.
