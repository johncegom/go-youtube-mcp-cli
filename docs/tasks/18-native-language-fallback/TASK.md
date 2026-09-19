# Task 18: Resolve the default transcript language from the video's captionTracks

**Status:** done (2026-09-20). Definition of Done + Test Plan approved by the
human 2026-09-20 (language line shown only when the resolved language is
not `en`). Follow-up to `docs/BUGS.md` BUG-011, whose message-only fix
deliberately left this feature out. Approach revised after an Advise call
(`docs/eagd-log.md`, 2026-09-20): originally "fall back on a 429", now
"resolve before fetching".

## User need

An agent calls `get_transcript` on a Vietnamese video without passing
`language`. Today it gets an error telling it to retry with `language`
set to the spoken language. The agent has no cheap way to know that
language, so it guesses or gives up. It should just get the transcript.

## Problem

`normalizeLanguage` defaults an omitted `language` to `"en"`. For an
auto-caption video not spoken in English, `en` selects a machine-translated
track and YouTube answers HTTP 429 (BUG-011, reproduced with plain yt-dlp:
`en` 429s / `vi` works on `r8CppXSqVDU`; `vi` 429s / `en` works on the
English video `BqRhBq-_kgE`). The spoken language is knowable for free
from the watch page: verified 2026-09-20 on three real pages,
`ytInitialPlayerResponse.captions.playerCaptionsTracklistRenderer.
captionTracks` lists `{"languageCode":"vi","kind":"asr"}` (Vietnamese-only
video), a single `{"languageCode":"en","kind":"asr"}` (English ASR-only
video), and for a video with uploaded subs the manual tracks (no `kind`)
plus the `asr` one. The `kind:"asr"` track's `languageCode` is the spoken
language.

## Approach (chosen after Advise): resolve first, when `language` is omitted

Why not "fall back after a 429": `classifyTranscriptError == "rate_limited"`
is a substring match on `429`/`Too Many Requests`, so it cannot tell a
rejected translated track from a genuine throttle. A fallback keyed on it
could silently serve a different language than requested during a real
throttle, and it burns a doomed yt-dlp call per call per video. Resolving
first uses a positive signal (`captionTracks`), never an ambiguous error.

Rule, applied only when the caller **omitted** `language`:
1. any track whose `languageCode` starts with `en` (manual or ASR) → `en`
   (today's behavior; no regression for manual-English-subs videos);
2. else the `kind:"asr"` track's `languageCode` (first one if several);
3. else (no `captions` key, live/upcoming, lookup failed) → `en`, i.e.
   today's path and today's error.

An explicit `language` (including explicit `"en"`) skips resolution
entirely; asking for a translation on purpose is honored and, if YouTube
rejects it, gets the BUG-011 message.

## Program design

- `resolveDefaultLanguage(playerResponse []byte) string` — pure, in
  `metadata.go` beside `chaptersFromPlayerResponseJSON`; implements the
  three-step rule above from `captionTracks`.
- `defaultLanguage(ctx, videoID) string` — I/O + memo. Looks up a
  per-video memo (bounded, mutex-guarded, same shape as `defaultCache` in
  `transcache.go`); on miss does one watch-page fetch (reuse the fetch in
  `fetchVideoMetadataAndChapters`, factor out rather than a second HTTP
  path) and stores the result. Any page-fetch error returns `"en"` and is
  not memoized; the caller then surfaces the yt-dlp error, never the page
  error.
- Call graph: `GetTranscriptText/Timed/Range` and `SearchInTranscript`
  (raw `language` in) → `language == "" ? defaultLanguage(...) :
  language` → `fetchSegments` (unchanged). The transcript cache key is the
  resolved language, so a later explicit `language:"vi"` reuses the entry.
- No new field on `parsedTranscript` (Advise: the caller already knows the
  language it resolved; no reachable consumer).
- **Disclosure:** when resolution picked a language other than `en`, the
  MCP tool result carries one leading line *outside* the transcript body
  (its own `mcp.TextContent` block, or a blank-line-separated header in the
  CLI), e.g. `language: vi (auto-detected spoken language)`. It is never
  put inside the transcript text, so offsets, `searchSegments`, and agent
  quoting are unaffected.
- Concurrency: `FetchVideoBrief` already runs transcript + page fetch
  concurrently; it is **out of scope** for this task (keeps today's
  behavior). Two goroutines resolving one video may both fetch the page
  once; the memo write is idempotent, so that is accepted rather than
  singleflighted.

## Definition of Done

- [x] 18.0 Firm up the finding before coding: confirm on at least one more
  non-English auto-caption video that the default `en` 429s and the native
  language works, and on a non-English video with uploaded English subs
  that `en` works. If translated tracks sometimes succeed, revisit the
  approach with the human.
  Done 2026-09-20: `viuFVSIwxUQ` (vi) and `9bZkp7q19f0` (ko) both 429 on `en`
  (`vi` works on the former); `BqRhBq-_kgE` 429s on `vi`; `BIAUJLWoN4k`
  (Japanese speech, uploaded `en` track + `ja` ASR) returns real English subs
  with `en`. n=4 for translated-429, rule step 1 confirmed. `BIAUJLWoN4k`
  added as a fixture for 18.1 (manual `en` + non-en ASR → `en`).
- [x] 18.1 `resolveDefaultLanguage` unit-tested against ground truth
  captured from the real watch pages of `r8CppXSqVDU` (→ `vi`),
  `BqRhBq-_kgE` (→ `en`), `dQw4w9WgXcQ` (manual + ASR → `en`), plus a
  payload with **no `captions` key at all** and one with an empty
  `captionTracks` (→ `en`).
- [x] 18.2 `defaultLanguage` decision logic unit-tested with a stubbed page
  fetcher (no network): omitted → resolved value; second call is a memo hit
  (fetcher called once); page-fetch error → `"en"`, not memoized; explicit
  `language` never calls the fetcher.
- [x] 18.3 The four public text entrypoints use the resolved language only
  when `language` is omitted; existing tests unchanged and green; output
  for `en`-resolving videos byte-identical to today.
- [x] 18.4 Cache reuse: an omitted-language call on the Vietnamese video
  populates the `{videoID,"vi"}` entry, and a later explicit `vi` call does
  not run yt-dlp again.
- [x] 18.5 The MCP handlers and CLI emit the language line outside the
  transcript body only when the resolved language is not `en`; the
  transcript text itself is unchanged.
- [x] 18.6 `language` descriptions (`internal/mcpserver/tools.go`, CLI
  `--language`) say the default is the video's spoken language.
- [x] 18.7 `docs/DECISIONS.md` entry: `SaveTranscriptFile`
  (`download_transcript*`) and `get_video_brief` keep `en`-only behavior
  for now, so the same video can return Vietnamese from `get_transcript`
  and the BUG-011 error from `download_transcript`. Deliberate scope cut,
  with the follow-up task named.
- [x] 18.8 Live smoke: `youtube-cli transcript r8CppXSqVDU` (no flag)
  prints the language line + Vietnamese text; `--language en` still prints
  the BUG-011 message; `dQw4w9WgXcQ` and `BqRhBq-_kgE` outputs unchanged.
- [x] 18.9 `go build/vet/test ./...` + `gofmt -l` clean; BUG-011 gets a
  "follow-up: task 18" pointer; ledger + this file updated.

## Test Plan

- **Unit (ground-truth):** 18.1 fixtures are minimal `captionTracks`
  excerpts copied from the live pages (fetched 2026-09-20), provenance in a
  test comment, per the strict-TDD rule.
- **Unit (logic):** 18.2/18.4 with an injectable page fetcher (same seam
  style as `transcache_test.go`); no yt-dlp, no HTTP.
- **Manual smoke (live):** 18.0 and 18.8. If YouTube stops rejecting
  translated tracks, the smoke test proves nothing and the unit tests are
  the real coverage.
- Commands: `go test ./internal/core/... -run 'DefaultLanguage' -v`,
  `go build ./... && go vet ./... && go test ./... && gofmt -l internal/`.

## Out of scope / open

- `SaveTranscriptFile`, `get_video_brief` (see 18.7).
- Naming the spoken language in the BUG-011 message for an explicit `en`
  (Advise suggested it; needs a lookup on the error path, and
  `TranscriptErrorText` has no ctx). Candidate follow-up.
- Language line on every call vs only when not `en`: decided 2026-09-20, only when not `en`.

## Notes / deviations (2026-09-20)

Implemented as approved, with these differences from the draft's names and
placement (behavior is what the Definition of Done specifies):

- **Placement and names.** All new code is in `internal/core/language.go`
  (not `metadata.go`), except `fetchWatchPageHTML`, factored out of
  `fetchVideoMetadataAndChapters` in `metadata.go` so there is one HTTP path
  to the watch page. `defaultLanguage` became the exported
  `ResolveLanguage(ctx, videoID, requested)`, backed by `languageMemo.resolve`
  and the stubbable package var `lookupSpokenLanguage`. No
  `fetchTranscriptDefault` exists.
- **Resolution lives in the callers, not in the four core entrypoints.**
  The MCP handlers and the CLI call `core.ResolveLanguage` first and pass the
  resolved language to the unchanged `GetTranscriptText/Timed/Range` and
  `SearchInTranscript`. Reason: they also have to show the user the
  auto-detected note, so they need the resolved value anyway. The transcript
  cache key is therefore the resolved language by construction; 18.4 is
  covered by `TestResolveLanguage_SharesCacheEntryWithExplicitLanguage`
  (seeded cache entry, no yt-dlp).
- **Memo is simpler than "same shape as `defaultCache`".** A mutex-guarded
  map that clears itself at 256 entries; no TTL or LRU. A video's language is
  stable and a re-lookup is one page fetch.
- **CLI `--language` default changed from `"en"` to `""`.** Required: with
  `"en"` the CLI always passed an explicit `en` and resolution never ran.
  `internal/cli/root_test.go` was updated for the two flag-default
  assertions (intentional). Help text: `languageFlagUsage`.
- **MCP note is a single-block header**, not a second `TextContent` block:
  `language: vi (auto-detected spoken language)\n\n<transcript>`. A client
  that reads only the first content block would otherwise lose the
  transcript. CLI prints the note to stderr so piped stdout stays a pure
  transcript.
- **`transcriptInput`** is a new input type for `get_transcript` and
  `get_transcript_timed` (their `language` description changed);
  `get_video_brief` keeps `urlLangInput` and its `en` default.
  `handlers_test.go` was updated for the two type names.
- **Other bug-surfaced finding:** the branch was cut from `main` before
  PR #33 (BUG-011 message) merged, so the first `--language en` smoke run
  showed the old wording; after syncing to `origin/main` it shows the
  BUG-011 message (verified live).
- **Cost:** the first omitted-language call per video runs the existing
  `ytInitialPlayerRe` scan over a ~1.3 MB page; on the four real pages it
  took ~1 s each. Accepted (once per video per process); a cheaper
  extraction (`strings.Index` on `"captionTracks"`) is a possible follow-up.
- **`FuzzResolveDefaultLanguage`** was added beyond the Test Plan (the
  function parses untrusted page JSON; matches the fuzz-tests convention).
- **18.0 result** is recorded inline under 18.0 above. 18.1's fixtures are
  the real `captionTracks` arrays of `r8CppXSqVDU`, `BqRhBq-_kgE`,
  `dQw4w9WgXcQ` and `BIAUJLWoN4k`; the "no `captions` key" and "empty
  `captionTracks`" cases are structural (no such live page was captured).
- **Grade:** fresh Haiku call, rubric = the Definition of Done verbatim,
  input = diff + command output only: 10 PASS / 0 FAIL. Its reply did not
  open with the requested `model:` line, so the model identity is not
  confirmed from the reply itself.
- **Follow-ups:** applying the same resolution to `SaveTranscriptFile`,
  `get_video_brief` and `search_playlist` is tracked as task 19
  (`docs/tasks/19-language-resolution-remaining-tools/TASK.md`, DECISION-022);
  not scheduled: naming the spoken language in the BUG-011 message for an explicit `en`.
