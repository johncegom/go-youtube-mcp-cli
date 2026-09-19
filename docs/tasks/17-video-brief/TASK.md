# Task 17: Composite `get_video_brief` MCP tool

**Status:** done (2026-09-15). Definition of Done + Test Plan
approved 2026-09-15 (plan-mode review), before implementation, per the
task-approval process in `CLAUDE.md`. Scoped as a new use-case-shaped
tool after the Phase-2 (11–15) scope closed — see `docs/DECISIONS.md`
DECISION-021 and `docs/PLAN.md`'s Phase-2 addendum.

## User need

"One call for the whole evaluation" — the `youtube-video-critic` skill's
full-evaluation path always runs `get_metadata` → `get_transcript_timed`
(and could use `get_chapters`), one agent turn each, and then has to
*estimate* from raw captions things it does badly: how much actual talking
there is, whether the captions are auto-generated (which decides how much
to trust them), how many `[Music]`/`[inaudible]` cues there are, and where
the long silent gaps fall.

## Problem

Every primitive tool answers one question. There is no tool that returns
the full raw material for a video evaluation in one shot, and no code path
computes transcript-level facts (word count, speaking rate, caption kind,
non-speech cues, gaps) that an agent otherwise re-derives by reading the
entire transcript. Caption kind isn't detectable at all today: the fetch
passes `WriteAutoSubs()` + `WriteSubs()` together and both land in the
same `sub.<lang>.vtt` filename, and nothing records which one it was.

## Plan

Thin orchestration over existing core functions — no new scraping, no new
HTTP fetches, no new parsing beyond a caption-kind sniff and a stats
calculator. The primitive tools are unchanged.

- `internal/core/transcript.go`:
  - `CaptionKind` exported typed string (`CaptionAuto`, `CaptionUploaded`,
    `CaptionUnknown`) + pure `detectCaptionKind(rawVTT string)`. YouTube
    auto (ASR) captions as written by yt-dlp carry inline word timings
    `<00:00:19.039>`; uploaded ones don't. Match **only** that token, not
    `<c>` — uploaded captions can carry `<c.colorXXX>` styling (the
    TS-derived fixture in `transcript_test.go` shows one). Empty input →
    unknown. Ground truth: real yt-dlp VTT heads for `dQw4w9WgXcQ` (has
    both an uploaded and an auto track) and `rfscVS0vtbw` (uploaded),
    recorded in the test file with provenance.
  - `parsedTranscript{Segments, CaptionKind}` becomes the transcript
    cache's value (`transcache.go`) so a cache hit still knows the kind
    (the raw VTT is gone by then). New `fetchTranscript` returns it;
    `fetchSegments` becomes a wrapper returning `.Segments`, so all
    existing callers (`transcript.go` ×5, `playlist.go`) are untouched.
- `internal/core/chapters.go`: extract `chaptersFrom(meta, prChapters)`
  from `FetchChapters` (two callers).
- New `internal/core/brief.go`:
  - `TranscriptStats{Words, SpeakingRateWPM, NonSpeechCues,
    NonSpeechBreakdown map[string]int, LongestGapSecs, LongestGapAtSecs}`
    and pure `computeTranscriptStats(segs)`: non-speech cue = a segment
    whose whole text is `[...]` (keys lowercased, excluded from `Words`);
    `Words` = whitespace-split over speech segments; WPM = round(Words /
    (lastEnd/60000)), 0 when lastEnd is 0; gap = `next.Offset −
    (cur.Offset+cur.Duration)`, negatives clamped to 0 (auto captions
    overlap); record the max gap and where it starts.
  - `VideoBrief{Metadata, Chapters, MetadataErr, TranscriptTimed,
    CaptionKind, Stats, TranscriptErr}` and `FetchVideoBrief(ctx, id,
    lang) VideoBrief` — never a Go error; each section's error is a field.
    Metadata+chapters (one HTTP fetch via
    `fetchVideoMetadataAndChapters`) and the transcript run concurrently
    (WaitGroup, same shape as `SaveTranscriptFile`). Wall time is
    max(15s, 30s), same as today.
- `internal/mcpserver/tools.go`:
  - `metadataLines(meta)` extracted from `getMetadataHandler` (same keys,
    same order, description last), shared by both.
  - Pure `formatVideoBrief(videoID, brief) (text, isError)` → single
    text block, sections in this order: `Video: <id>`; metadata lines (or
    `Metadata: Failed to fetch metadata: <err>`); `Chapters:` block or
    `No chapters found.` (omitted entirely when metadata failed — same
    fetch); `Transcript stats:` block (`Captions: likely auto-generated |
    likely uploaded | unknown`, `Words`, `Speaking rate`, `Non-speech
    cues` with a sorted breakdown, `Longest gap: Ns at [M:SS]`) or
    `Transcript: <TranscriptErrorText>`; `Transcript (timed):` block.
    **`isError` is true iff the transcript section failed.** Full
    description included (noise-level next to the transcript).
  - `getVideoBriefHandler` on `urlLangInput`; registered as
    `get_video_brief` with a description that tells the agent when to use
    it versus the primitives (it returns the full timed transcript; for
    long videos or targeted questions prefer `get_chapters` +
    `get_transcript_range`). Tool count 17 → 18.
- No CLI subcommand (task-14 precedent; agent-driven use case).

## Definition of Done

- [x] 17.1 `detectCaptionKind` returns auto / uploaded / unknown against
  the real yt-dlp VTT samples; `<c.xxx>`-only content is **not** auto;
  `FuzzDetectCaptionKind` never panics.
- [x] 17.2 Cache stores `parsedTranscript`; every existing transcript and
  playlist caller is unchanged and green; a cache hit preserves
  `CaptionKind`.
- [x] 17.3 `computeTranscriptStats` matches hand-derived fixtures: word
  count excludes non-speech cues, WPM 0 on empty / zero-length input,
  negative gaps clamp to 0, breakdown keys lowercased.
- [x] 17.4 `FetchVideoBrief` fetches metadata+chapters and transcript
  concurrently; a metadata failure and a transcript failure land in their
  own fields independently; `chaptersFrom` is shared with `FetchChapters`
  (existing chapter tests green).
- [x] 17.5 `formatVideoBrief` test matrix: all ok; metadata failed only
  (isError=false, failure line present, no Chapters block); transcript
  failed only (isError=true, stats omitted, metadata present); both
  failed; no chapters; zero stats; breakdown ordering deterministic.
- [x] 17.6 `get_video_brief` registered; invalid URL → the standard
  invalid-URL error; `TestNewServer_ToolCount` = 18 (number and comment).
- [x] 17.7 Manual smoke test recorded below: one auto-caption video, one
  uploaded-caption video, one no-captions video (IDs recorded), via a
  direct `core.FetchVideoBrief` call or the in-process client — never
  `printf | run` (RETRO-002).
- [x] 17.8 `go build ./... && go vet ./... && go test ./... -race` clean,
  `gofmt -l .` empty; docs updated: this file, `docs/LEDGER.md` (row 17,
  status, resume checklist), `docs/DECISIONS.md` DECISION-021,
  `docs/PLAN.md` Phase-2 addendum, `README.md` tool-table row,
  `CLAUDE.md` `internal/core` bullet for `brief.go`.

## Test Plan

- **Unit tests (TDD):**
  - `detectCaptionKind`: ground truth = trimmed heads of real yt-dlp
    output (`--write-auto-subs` vs `--write-subs`, `--sub-langs en
    --sub-format vtt`) for `dQw4w9WgXcQ` (auto head with inline
    `<00:00:19.039><c>` timings; uploaded head with `♪` lyric lines and no
    inline timings) and `rfscVS0vtbw` (uploaded), recorded verbatim in
    `transcript_test.go` with video IDs. Trap: the existing TS-derived
    `vttFixture` (has `<c.colorE5E5E5>` but no inline timing) → uploaded.
  - `computeTranscriptStats`: hand-derived table fixtures for each rule in
    17.3, plus `parseVtt`'s real output on the captured auto-caption
    sample to sanity-check WPM.
  - `transcache_test.go`: mechanical closure-type update; one new test
    that a cache hit returns the stored `CaptionKind`.
  - `formatVideoBrief`: the 17.5 matrix on hand-built `core.VideoBrief`
    values (no network).
  - `getVideoBriefHandler`: invalid-URL branch (the only branch that
    returns before network, matching the existing handler-test
    convention); tool count.
- **Fuzz test:** `FuzzDetectCaptionKind` (untrusted yt-dlp output; no
  panic, result always one of the three constants).
- **Manual smoke test (commands):** throwaway `_test.go` calling
  `core.FetchVideoBrief` on: an auto-caption-only video, an
  uploaded-caption video (`dQw4w9WgXcQ` or `rfscVS0vtbw`), and a
  no-captions video; eyeball caption kind, WPM plausibility, and the
  transcript-failure section with the rest still present. IDs and output
  recorded under "Notes / deviations".

```sh
go build ./... && go vet ./... && go test ./... -race && gofmt -l .
go test ./internal/core/... -run 'TestDetectCaptionKind|TestComputeTranscriptStats|TestTranscriptCache' -v
go test ./internal/mcpserver/... -run 'TestFormatVideoBrief|TestGetVideoBriefHandler|TestNewServer_ToolCount' -v
go test ./internal/core/... -run FuzzDetectCaptionKind -fuzz=FuzzDetectCaptionKind -fuzztime=10s
```

## Notes / deviations

- **Branching:** the working tree had uncommitted BUG-008 diagnostic work
  in `transcript.go` (`Verbose()`, PID/kill logging). Per human decision it
  was committed on `fix/bug-008-verbose-pid-logging` (PR #29) first, and
  this task's branch `feat/task-17-video-brief` was cut from it, since
  both edit `fetchSegmentsFromYtDlp`.
- **Ground-truth capture finding:** for `rfscVS0vtbw`, `--write-auto-subs`
  alone and `--write-subs` alone produced byte-identical clean-text files
  with no inline timings — i.e. yt-dlp's flag does not reliably say which
  kind of track was served. The content sniff (inline word timings) is the
  only reliable signal, which is why detection is content-based.
- **`chapter` exported as `Chapter`:** `VideoBrief.Chapters` would otherwise
  expose an unexported type through an exported field, and the
  `formatVideoBrief` tests in `internal/mcpserver` need to build chapter
  fixtures. Mechanical rename in `internal/core` only; no behavior change.
- **BUG-010 surfaced (not fixed here; fixed later, 2026-09-20 — see `docs/BUGS.md`):** running `parseVtt` on the captured
  auto-caption sample showed each line ~3× (rolling-cue carry blocks
  survive the `offset|text` dedupe). Reachable on every auto-caption-only
  video today, so it went to `docs/BUGS.md` for a decision rather than a
  silent fix. The brief's `Words`/`Speaking rate` inherit the inflation on
  `likely auto-generated` videos until it's fixed (visible in the smoke
  test below: 838 words / 198 wpm for a 4-minute song).
- **Gap rendering fix from the smoke test:** the first live run rendered
  `Longest gap: 6.771000000000029s` — float noise from VTT ms parsing.
  Fixed by rounding the gap and its start to whole ms in
  `computeTranscriptStats` (red test
  `TestComputeTranscriptStats_GapRoundsToMilliseconds` first).
- **Manual smoke test (2026-09-15),** throwaway `_test.go` in
  `internal/mcpserver` calling `getVideoBriefHandler` directly (same code
  path the MCP transport invokes; deleted before commit):
  - `dQw4w9WgXcQ` (Rick Astley): `isError=false`, 11.6s cold. Metadata
    full; `No chapters found.`; `Captions: likely uploaded`; 486 words,
    138 wpm; `Non-speech cues: 1 ([♪♪♪] x1)`; `Longest gap: 15.6s at
    [00:03]` (the intro before the first lyric — correct).
  - `rfscVS0vtbw` (freeCodeCamp Python, 4h27m): `isError=false`, 9.0s.
    35 chapters rendered identically to `get_chapters`; `likely uploaded`;
    50,862 words, 191 wpm; 291 KB total — the tool description's "prefer
    get_chapters + get_transcript_range for long videos" caveat is earned.
  - `X0UI0O8YzJM` and `kjoQPn--F7A` (IBM Technology, the BUG-007/008
    repro videos): both `likely uploaded` (consistent with the user's
    BUG-008 observation that those had human transcripts); `kjoQPn--F7A`
    also yielded 10 chapters from its `MM:SS - Title` description format.
  - `9bZkp7q19f0` (Gangnam Style, auto-only English track per
    `yt-dlp --list-subs`): `isError=false`, `Captions: likely
    auto-generated` — the auto branch verified live. BUG-010's rolling
    duplicates visible in the transcript lines.
  - `LXb3EKWsInQ` (no captions at all per `--list-subs`): `isError=true`;
    metadata and `No chapters found.` still present, last line
    `Transcript: No transcript available for video LXb3EKWsInQ. The video
    may not have captions.` — the partial-failure contract end to end.
  - `dQw4w9WgXcQ` with `language="zz"`: same shape as above
    (`isError=true`, metadata kept), exercising the missing-language path.
- **Fuzz:** `FuzzDetectCaptionKind` 8s clean.
- **Stale-doc follow-ups, done in this task per human direction
  ("documents all first"):** `README.md` tool table gained the missing
  `list_playlist`/`search_playlist` rows; `CLAUDE.md`'s `internal/mcpserver`
  section no longer says "11 tools" (now 18, with the Phase-2 additions
  listed). `docs/PLAN.md`'s three "11 tools" mentions were left alone on
  purpose — they describe the upstream TS server and the original Phase-1
  port plan, which is history, not current state. The
  `youtube-video-critic` `SKILL.md` (outside this repo, in the
  `minh-skills` marketplace checkout) was updated to know about
  `get_video_brief` and `get_chapters`, to stop claiming search returns no
  context, and to drop the "chapters/context search don't exist" line —
  that change lives in the marketplace repo, not here, and the installed
  plugin cache (`0.9.0`) still holds the old copy until the plugin is
  re-installed or its version bumped.

## After finishing

Update this file's status/checklist, then `docs/LEDGER.md`'s row for task
17, then pause and ask the human before starting anything else.
