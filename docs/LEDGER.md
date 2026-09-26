# Progress Ledger (Index)

Entry point for tracking progress against `docs/PLAN.md`. Each task's full
detail (checklist, notes, deviations, root-cause writeups) lives in its own
file under `docs/tasks/<slug>/TASK.md` — **read this index first, then open
only the task file(s) you actually need.** This split exists specifically to
save context: a fresh session no longer has to load the full history of
every finished task just to find out what's next.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done

## Standing process rules

Full detail lives in `CLAUDE.md` ("Required workflow: strict TDD", the
task-approval Definition-of-Done/Test-Plan rule, and the pause-and-ask
rule), `docs/BUGS.md` (bug-tracking process), and `docs/DECISIONS.md`
(deliberate design/scope tradeoffs, not bugs) — not duplicated here. In
short: strict TDD for pure functions (ground truth before the test, test
before the implementation); every task needs a Definition of Done + Test
Plan written and reviewed in its `TASK.md` *before* coding starts; pause and
ask before starting the next task; update the relevant `TASK.md` (not just
this index) before pausing.

## Tasks

| # | Task | Status | Detail |
|---|------|--------|--------|
| 1 | Init Go module + scaffold directory structure | [x] done | [docs/tasks/01-scaffold/TASK.md](tasks/01-scaffold/TASK.md) |
| 2 | Core: videoid, format, paths | [x] done | [docs/tasks/02-core-basics/TASK.md](tasks/02-core-basics/TASK.md) |
| 3 | Core: ytdlp wrapper + metadata scraping | [x] done | [docs/tasks/03-core-ytdlp-metadata/TASK.md](tasks/03-core-ytdlp-metadata/TASK.md) |
| 4 | Core: transcript (VTT fetch/parse/search/save) | [x] done | [docs/tasks/04-core-transcript/TASK.md](tasks/04-core-transcript/TASK.md) |
| 5 | Core: download (video/audio, blocking + fire-and-forget) | [x] done | [docs/tasks/05-core-download/TASK.md](tasks/05-core-download/TASK.md) |
| 6 | CLI with cobra (`youtube-cli`) | [x] done | [docs/tasks/06-cli-cobra/TASK.md](tasks/06-cli-cobra/TASK.md) |
| 7 | MCP server with official go-sdk (`youtube-mcp`) | [x] done | [docs/tasks/07-mcpserver/TASK.md](tasks/07-mcpserver/TASK.md) |
| 8 | Unit tests for core pure functions | [x] done (folded into other tasks) | [docs/tasks/08-unit-tests/TASK.md](tasks/08-unit-tests/TASK.md) |
| 9 | Build + smoke test both binaries | [x] done | [docs/tasks/09-smoke-test/TASK.md](tasks/09-smoke-test/TASK.md) |
| — | Out-of-band: fix BUG-001 + BUG-002 | [x] fixed, merged to `main` | [docs/tasks/bugfix-001-002/TASK.md](tasks/bugfix-001-002/TASK.md) |
| — | Out-of-band: fuzz tests for untrusted-input parsers | [x] done | [docs/tasks/fuzz-tests/TASK.md](tasks/fuzz-tests/TASK.md) |
| — | Out-of-band: task-level Definition of Done/Test Plan + decision log | [x] done | see `CLAUDE.md` process rules + `docs/DECISIONS.md` |
| — | Out-of-band: CI/CD (GitHub Actions; branch protection enabled, see DECISION-012) | [x] done | [docs/tasks/ci-cd/TASK.md](tasks/ci-cd/TASK.md) |
| 10 | Packaging: GoReleaser + GitHub Releases, `go install`, Docker image, README | [x] done | [docs/tasks/10-packaging/TASK.md](tasks/10-packaging/TASK.md) |
| — | Out-of-band: repo made public, MIT-licensed, branch protection enabled | [x] done | see `docs/DECISIONS.md` DECISION-012 |
| 11 | Phase 2: transcript cache + `get_transcript_range` | [x] done | [docs/tasks/11-transcript-cache/TASK.md](tasks/11-transcript-cache/TASK.md) |
| 12 | Phase 2: download job tracking (`get_download_status`, `list_downloads`) | [x] done | [docs/tasks/12-download-jobs/TASK.md](tasks/12-download-jobs/TASK.md) |
| 13 | Phase 2: context-aware transcript search | [x] done | [docs/tasks/13-context-search/TASK.md](tasks/13-context-search/TASK.md) |
| 14 | Phase 2: chapters (`get_chapters`) | [x] done (out of order, before 13) | [docs/tasks/14-chapters/TASK.md](tasks/14-chapters/TASK.md) |
| 15 | Phase 2: playlist listing + cross-video search (depends on 11) | [x] done | [docs/tasks/15-playlist-search/TASK.md](tasks/15-playlist-search/TASK.md) |
| — | Out-of-band: test-coverage hardening (tiers 1-4) | [x] done | [docs/tasks/test-coverage-hardening/TASK.md](tasks/test-coverage-hardening/TASK.md) |
| — | Out-of-band: `Taskfile.yml` dev-tooling wrapper | [x] done | [docs/tasks/taskfile/TASK.md](tasks/taskfile/TASK.md) |
| 16 | Transcript-fetch observability logging | [x] done | [docs/tasks/16-transcript-observability/TASK.md](tasks/16-transcript-observability/TASK.md) |
| 17 | Composite `get_video_brief` tool (metadata + chapters + timed transcript + stats, per-section failure) | [x] done | [docs/tasks/17-video-brief/TASK.md](tasks/17-video-brief/TASK.md) |
| 18 | Resolve the default transcript language from the video's `captionTracks` (fixes the BUG-011 default-`en` trigger for `get_transcript*`/`search_transcript`/CLI) | [x] done | [docs/tasks/18-native-language-fallback/TASK.md](tasks/18-native-language-fallback/TASK.md) |
| 19 | Apply spoken-language resolution to `download_transcript*` and `get_video_brief` (`search_playlist` intentionally left on `en`; closes most of DECISION-022's scope cut) | [x] done | [docs/tasks/19-language-resolution-remaining-tools/TASK.md](tasks/19-language-resolution-remaining-tools/TASK.md) |
| 20 | Guarded, peek-only orig-first transcript fetch (BUG-012 follow-up; land task 21 first) | [ ] DRAFT — DoD + Test Plan not approved | [docs/tasks/20-guarded-orig-first/TASK.md](tasks/20-guarded-orig-first/TASK.md) |
| 21 | Resolve the default language from the video's original language — the `.4` audio-track id (BUG-013 fix; land before task 20) | [ ] approved 2026-09-27, not started | [docs/tasks/21-original-language-resolution/TASK.md](tasks/21-original-language-resolution/TASK.md) |

## Current status

**All 9 numbered tasks plus task 10 are done.** Phase 1 (the faithful Go
port) is complete, and the project is now packaged/distributable three
ways: cross-compiled GitHub Release binaries (via `.goreleaser.yaml` +
`.github/workflows/release.yml`, triggered on `v*` tags), `go install`, and
a Docker image (`Dockerfile`, both binaries on `PATH`, `CMD ["youtube-mcp"]`
default) — see `README.md` for install instructions and
`docs/tasks/10-packaging/TASK.md` for full verification detail. The repo
is now **public** and **MIT-licensed** (`LICENSE`, `docs/DECISIONS.md`
DECISION-012), and branch protection on `main` is enabled (requires
`build-test (ubuntu-latest)`, `build-test (windows-latest)`, and `gofmt`
to pass before merge — this actually enforces CI now, not just reports
it, closing the gap DECISION-007 originally left open). All out-of-band
items (bug fixes, fuzz tests, the anti-drift process work, CI/CD) are also
complete. Remaining known gap: BUG-001/002 fixes cover only the platforms
verified (`docs/DECISIONS.md` DECISION-006). A pre-existing (not
task-10-introduced) CRLF/gofmt working-tree issue on 3 files, previously
flagged in task 10's `TASK.md`
10.6, has since been fixed on the `task-10-packaging` branch.

**Deliberately deferred out of task 10, and not yet scoped as a future
task:** an HTTP/SSE-based MCP transport for genuinely hosted/remote,
multi-client use (the SDK already supports it; the blocker is unaddressed
design questions — shared download directory, per-session isolation, auth
— not a missing library feature, see DECISION-011). Revisit if/when
there's a concrete need for remote (not just locally-spawned-via-Docker)
MCP access.

**Phase 2 is scoped and approved (2026-08-28):** tasks 11–15, driven by a
critical evaluation of the MCP tools from the LLM-agent-consumer
perspective — see `docs/PLAN.md`'s "Phase 2" section for the rationale and
`docs/DECISIONS.md` DECISION-013 for the scope-expansion decision. Each
task has a reviewed Definition of Done + Test Plan in its own `TASK.md`.
Implementation order is 11 → 12 → 13 → 14 → 15 (11 is the foundation the
others lean on; 15 hard-depends on 11). Task 11 is done (2026-08-29) — see
`docs/tasks/11-transcript-cache/TASK.md` for full detail: an in-memory
TTL/cap-bounded transcript cache (`internal/core/transcache.go`) transparent
to all existing `fetchSegments` callers, plus the new `get_transcript_range`
MCP tool and its `core.ParseTimestamp`/`filterSegmentsByRange` building
blocks. Task 12 is done (2026-08-30) — see
`docs/tasks/12-download-jobs/TASK.md` for full detail: a process-lifetime,
capped `jobRegistry` (`internal/core/jobs.go`) tracking download outcomes,
wired into `StartVideoDownload`/`StartAudioDownload`, plus the new
`get_download_status`/`list_downloads` MCP tools and honesty fixes to the
`download_video`/`download_audio` descriptions (`docs/DECISIONS.md`
DECISION-015). Its manual smoke test surfaced and led to fixing BUG-006 (a
`go-ytdlp` pinned-version staleness bug reverting locally-updated yt-dlp
binaries — see `docs/BUGS.md`); the smoke test has since been fully
re-run end-to-end including the success path, all green — see its
`TASK.md` "Notes / deviations" for the full transcript. Task 14 is done
(2026-08-31) — see `docs/tasks/14-chapters/TASK.md` for full detail: a
pure `parseChapters` (`internal/core/chapters.go`) extracting chapters from
a video description, a player-response JSON fallback tier
(`chaptersFromPlayerResponseJSON`), and the new `get_chapters` MCP tool.
Implemented **out of the documented order** — before task 13, with
explicit human approval, since 14 has no code dependency on 13. Manual
smoke test against a real 35-chapter video and a real unchaptered video,
both correct (see `TASK.md` "Notes / deviations"). Task 13 is done
(2026-09-01) — see `docs/tasks/13-context-search/TASK.md` for full detail:
search now matches the merged transcript text as one stream
(`internal/core/search.go`'s `searchSegmentsWithContext`), so phrases
spanning two VTT segments are found (previously a silent false negative),
expands each match to a ±`context`-second window (default 15; `0` =
matched segments only), merges overlapping windows into blocks separated
by `---`. Upgrades `search_transcript`/`search_in_transcript` and CLI
`search --context` in place — no new tool. Output-format deviation from
upstream logged as `docs/DECISIONS.md` DECISION-018, per DECISION-013's
anticipation. Manual smoke test against a real video found a genuine
cross-boundary match the old search would have missed (see `TASK.md`
"Notes / deviations"). Task 15 is done (2026-09-02) — see
`docs/tasks/15-playlist-search/TASK.md` for full detail: a new
`internal/core/playlist.go` adding `ExtractPlaylistID` (pure, reads the
`list` query param, mirrors `ExtractVideoID`'s shape), flat-playlist
listing via yt-dlp's `--flat-playlist --print` mode
(`ListPlaylistVideos`, capped to 25 entries with a "showing first 25 of
N" note), and sequential, cache-backed, per-video-failure-tolerant
cross-video search (`SearchPlaylist`/`searchPlaylistEntries`, reusing
task 13's `searchSegmentsWithContext` per video and task 11's transcript
cache transparently — no new cache plumbing). New MCP tools
`list_playlist`/`search_playlist` registered in
`internal/mcpserver/tools.go`. This is the first Phase-2 feature with no
upstream counterpart at all — see `docs/DECISIONS.md` DECISION-019. Manual smoke test against a real
239-video public playlist confirmed correct listing/cap, correct
cross-video search matches with timestamps, and a ~28x cache speedup on
a repeat search (2m13s → 4.7s) — see `TASK.md` "Notes / deviations" for
the full transcript. **This completes the currently-approved Phase 2
scope (tasks 11-15).** The pause-and-ask rule still applies before any
further scoping.
Related: BUG-003 (dead Node-ism branches in
`TranscriptErrorText`) is fixed and merged to `main` (PR #10, commit
`be6c93d`) — see `docs/BUGS.md` for the full writeup.

**Out-of-band test-coverage hardening is done (2026-08-30)** — see
`docs/tasks/test-coverage-hardening/TASK.md`. Closed the highest-risk gaps
found by a coverage audit: `internal/core/paths.go` (previously zero tests)
now has a full test file; `internal/cli` (previously zero tests) now has
cobra wiring/flag/output-format tests plus a new `exitFunc` seam
(`docs/DECISIONS.md` DECISION-014) so exit-1 paths are testable;
`internal/mcpserver` now covers all 12 tools' validation branches plus a
real in-process round-trip test proving alias tools are wired to their
canonical handler; `transcache.go` gained cap/TTL boundary tests; CI's
`go test` now runs with `-race`. Surfaced two new bugs along the way
(neither fixed yet, both open pending decision): **BUG-004** (a
path-traversal-allowlist boundary bug in `pathStartsWith`) and **BUG-005**
(a negative-cap panic in `transcriptCache.set`, not reachable via any
current code path). Tier 5 (httptest/fake-binary I/O-boundary tests for
`FetchVideoMetadata`/`fetchSegmentsFromYtDlp`) was explicitly deferred to a
future task.

**Task 16 (out-of-band, done, 2026-09-08)** — see
`docs/tasks/16-transcript-observability/TASK.md` for full detail. Prompted
by a real support case: a `get_transcript_timed` call on video `kjoQPn--F7A`
returned a timeout error, but `errors.log` had no record of it — the
existing `LogDownloadError` (`internal/core/paths.go`) was only ever wired
into the download path, never the transcript path. Fixed by extracting
`TranscriptErrorText`'s classification into a pure, separately-tested
`classifyTranscriptError` helper (also unit-tested), then logging every
transcript-fetch failure from the single choke point all transcript
entrypoints share (`fetchSegmentsFromYtDlp` in `internal/core/transcript.go`
— covers both `cmd/youtube-cli` and `cmd/youtube-mcp`, not just the MCP
server) via a `defer` on named returns, with elapsed duration, language, and
classified category alongside the raw error text. Also logs a
`transcript_fetch_slow` line for successful fetches exceeding 20s (2/3 of
the 30s timeout budget), as an early-warning trend signal for the kind of
intermittent timeout BUG-007 took a live investigation to root-cause. No new
logging framework — reused the existing plain-text `errors.log`. See
`docs/DECISIONS.md` DECISION-020.

**Task 18 (2026-09-20)** — see `docs/tasks/18-native-language-fallback/TASK.md`: an omitted `language` is now resolved from the watch page's `captionTracks` (any `en*` track → `en`, else the ASR track's language, else `en`) by `core.ResolveLanguage`, called by the MCP `get_transcript`/`get_transcript_timed`/`get_transcript_range`/`search_transcript` handlers and the CLI `transcript`/`search`; an explicit `language` is always honored. A `language: <code> (auto-detected spoken language)` note is shown outside the transcript body only when the result isn't `en`. `download_transcript*`, `get_video_brief` and `search_playlist` keep the plain `en` default (DECISION-022). Follow-up to BUG-011; approach chosen after an Advise call (`docs/eagd-log.md`).

**Task 19 (2026-09-20)** — see `docs/tasks/19-language-resolution-remaining-tools/TASK.md`: applies task 18's spoken-language resolution to `download_transcript*`/CLI `transcript --save` (via `core.SaveTranscriptFileResolved`) and `get_video_brief` (resolved inside its transcript goroutine; `Language:` line only when auto-detected and not `en`). Saved transcripts are now named `<title>_<lang>[_timed].md` for every language including `en` (human decision; language component sanitized), with a `**Language:**` header line only for non-`en`. `search_playlist` deliberately stays on the plain `en` default (human decision, minimum improvement: description only); the lazy-retry design is parked in the Backlog below. DECISION-022 updated. Grade needed two targeted fixes before 19.4 passed (save-path and brief-independence tests).

**Task 17 (`get_video_brief`, 2026-09-15)** — see
`docs/tasks/17-video-brief/TASK.md` for full detail. Scoped from a
brainstorm of "use-case-shaped" composite tools (one call for a whole
agent workflow, rather than one call per primitive), filtered by whether
the server does something between the calls the agent does badly. The
new tool returns metadata + chapters + full timed transcript +
server-computed transcript stats (caption kind auto/uploaded, words,
speaking rate, non-speech cues, longest gap) in one call, with
per-section partial failure (`isError` iff the transcript failed) —
`docs/DECISIONS.md` DECISION-021. Thin orchestration over existing core
functions (`internal/core/brief.go`); primitives unchanged; tool count 18.
Caption kind is sniffed from VTT content (inline word timings) and cached
with the segments. The task's ground-truth capture surfaced **BUG-010**
(`parseVtt` returned each auto-caption line ~3×, reachable on every
auto-caption-only video — fixed 2026-09-20, see `docs/BUGS.md`), which had
also inflated the brief's word stats. BUG-008's pending diagnostic work was
committed first on its own branch (PR #29) so both changes to
`fetchSegmentsFromYtDlp` stack cleanly.

## Backlog (not scheduled)

Ideas and research items that were deliberately parked, so a fresh session can
find them without re-deriving them. Nothing here is approved work; anything
promoted to a task gets a row in the table above and a `TASK.md`.

| Item | Kind | Detail |
|------|------|--------|
| Playlist search: lazy retry-on-failure language resolution (Advise-recommended full fix for `search_playlist`) | improvement | [task 19, "Future improvements"](tasks/19-language-resolution-remaining-tools/TASK.md) |
| Re-check whether translated-caption 429s are permanent or vary by network/time (n=4 on one network, one day) | research | [task 19, "Future improvements"](tasks/19-language-resolution-remaining-tools/TASK.md) |
| Resolve the spoken language via yt-dlp's `<lang>-orig` track instead of the page scan | research | [task 19, "Future improvements"](tasks/19-language-resolution-remaining-tools/TASK.md) |
| Name the spoken language inside the BUG-011 error for an explicit `en` on a non-English video | improvement | [task 19, "Out of scope"](tasks/19-language-resolution-remaining-tools/TASK.md) |
| Cheaper `captionTracks` extraction than the `ytInitialPlayerRe` scan (~1 s on a real page) | improvement | [task 18 notes](tasks/18-native-language-fallback/TASK.md) |

## Resume checklist for next session

1. Read this index first (not the individual task files, unless you need one).
2. Run `go build ./... && go vet ./... && go test ./...` to confirm the current state still holds.
3. Skim `docs/RETRO.md` for any still-relevant advice before starting new work.
4. **Phase 2 (tasks 11-15), task 16 (observability logging), task 17
   (`get_video_brief`) and task 18 (default language resolved from
   `captionTracks`, DECISION-022) and task 19 (same resolution for the
   download tools and `get_video_brief`; `search_playlist` intentionally left
   on `en`) are done.** No task is currently approved to start
   next — the next step is a new scoping pass with the human. One open
   item is waiting: BUG-008's next live repro (PR #29's `Verbose()` +
   PID logging needs a Claude Desktop timeout to confirm it's useful). BUG-010 (auto-caption
   rolling-cue duplication in `parseVtt`) is fixed as of 2026-09-20.
   BUG-012 (plain `en` 429s; retry with `<lang>-orig`) is fixed by PR #36 but stays
   open for one limitation (silent back-translation); its evidence is in
   `docs/evidence/bug-012/`. **BUG-013 is open and awaiting a human decision**
   (`ResolveLanguage` returns `en` for every non-English video with YouTube's
   auto-dubbing structure — 7 of 7 sampled — so task 18's fix regressed on real
   videos). Its research is done (`docs/evidence/bug-013/`; the signal is the `.4` id in
   `captions.audioTracks`). The human chose that option on 2026-09-27; its fix is
   **task 21 (approved 2026-09-27, not started)**. Task 20 is a DRAFT that should follow it.
5. After finishing a task: update **that task's `TASK.md`** with full detail first, then update this index's status column/checkbox for it, then pause and ask the human before starting the next task. If the task involved a deliberate design/scope tradeoff, log it in `docs/DECISIONS.md` too; if it surfaced a way-of-working lesson that generalizes beyond that one task, log it in `docs/RETRO.md`.
