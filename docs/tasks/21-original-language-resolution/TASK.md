# Task 21: Resolve the default language from the video's original language (BUG-013 fix)

**Status:** DONE — merged in PR #38 (2026-09-27), Grade passed. The human approved this
Definition of Done + Test Plan as drafted (rev 2). Fixes `docs/BUGS.md` BUG-013 (option 1, decided by the human 2026-09-27).
Rev 2 incorporates an Advise call (logged in `docs/eagd-log.md`). Written on the
`docs/bug-012-measurement-evidence` branch beside the evidence it rests on
(`docs/evidence/bug-013/`).

**What the approval covers, and what it does not.** Approved: the scope as
written — the precedence applies only where the original language is known from
the `.4` id — and the reading that "the English" is the *uploaded* English track.
**Not decided:** the "Open question for the human" below (extending the rule to
videos without the dubbing structure); it stays out of this task and needs its own
decision. If implementation reaches a case this file does not settle, stop and ask.

## User need

Task 18's `resolveDefaultLanguage` returns `en` as soon as **any** caption track
has code `en` or `en-*`. YouTube now adds an auto-generated (dubbed) `en` track
to non-English videos, so the default resolves the wrong language and the fetch
fails with the BUG-011 429. Measured on 41 videos: the rule is wrong on **7 of
33** videos that yt-dlp gives a language for — every non-English video that has
the auto-dubbing structure (4 Vietnamese, 3 German) — and right on the other 26
(`docs/evidence/bug-013/README.md`). `--language vi` still works, so the
workaround exists; the point of task 18 was that an agent should not need it.

## Human decision (2026-09-27), and how it is read here

> "if a non-English video also has uploaded English subtitles -> if it has
> uploaded subtitles of their original language, use it first. If not, use the
> English."

Read as this precedence, applied when the video's **original language `O`** is
known (see next section):

1. an **uploaded** track in `O` exists → use that track;
2. else an **uploaded English** track exists → `en`;
3. else the **auto-generated** track in `O` → that track's code (the BUG-013 fix
   proper);
4. else today's remaining rules, unchanged.

Interpretation to confirm at review: "the English" means the **uploaded**
English track. Falling back to *auto* English for a non-English video would be
the machine-translated request that 429s (BUG-011), so it is not used unless
step 3 has nothing.

## What `O` (the original language) is — and where the rule applies

**Only the `.4` audio-track id.** `captions.playerCaptionsTracklistRenderer.
audioTracks[].audioTrackId`, on videos with YouTube's auto-dubbing structure, has
exactly one id ending `.4` (`vi.4`, `de-DE.4`, `en-US.4`) — the original audio —
and the rest end `.10`. Its language equalled yt-dlp's `language` on **14 / 14**
such videos and the `.4` id was identical across four viewer locales on the 3
videos tested. Not usable: the `streamingData` `audioIsDefault` track,
`hostLanguage`, `requestLanguage` — they depend on the viewer's locale.

**If there is not exactly one `.4` id (no `audioTracks`, several `.4` ids,
malformed ids), `O` is unknown and the whole precedence is skipped: today's
rules run, unchanged.** So every video without the dubbing structure — 19 of 19
sampled where today's rule was right — behaves exactly as now. If `O` is known
but no track matches it, the result is likewise today's rules: **the failure
mode is today's behaviour, never a new one.**

*Not in this task (Advise, and see "Open question"):* using the single auto
track as `O` on videos **without** the structure, which would extend step 1 to
them. It is a behaviour change on videos that are not in BUG-013.

**Matching a language to a track code.** Codes differ in kind (`.4` says `de-DE`,
the auto-caption track is `de`, an uploaded track may be `de-DE`; Vietnamese is
`vi` everywhere). Preference order, within the tracks of the wanted kind:
(1) exact code; (2) same base language (the part before `-`) with a
region-shaped suffix (`de-DE`, `pt-BR`, `es-419`); (3) any other same-base track
(e.g. a custom-named `vi-eEY6…` variant — the shape `UF8uR6Z6KLc` uses). The
code returned is the **track's own code**, because yt-dlp's `--sub-langs` is a
full match. **Exception: English stays the literal `en`.** A matched track whose
base language is `en` resolves to `"en"`, exactly as today, so no English video
changes its resolved language, saved filename (`<title>_en.md`), `**Language:**`
header (absent for `en`) or `LanguageNote` (absent for `en`). Checked live
(2026-09-27, yt-dlp 2026.07.04): plain `de` fetches a real transcript on all
three German videos (and `de-orig` returned the identical file for
`cZSgL76ddDs`), plain `vi` on all three Vietnamese ones — the codes the new rule
resolves are fetchable.

## Definition of Done

`[x]` done · `[ ]` not started. **Only 21.0b blocked coding.**

- [x] 21.0 **Research**:
  - [x] 21.0a *Resolved codes are fetchable* — done 2026-09-27 (above).
  - [x] 21.0b **(blocks coding)** *Real fixtures.* Capture trimmed real
    `captionTracks` + `audioTracks` payloads (long `baseUrl`s stripped, as task 18
    did) for `r8CppXSqVDU` (auto `en-US` then `vi`; audio `vi.4` + `en-US.10`),
    `cZSgL76ddDs` (auto `en-US`, `de`; `de-DE.4` + one `.10`) and `vyIgAO8aCbA`
    (21 tracks; `en-US.4` + 20 `.10`; its auto track is exactly `en`,
    `pagecodes-results.txt`), from current pages, **keeping the real track order and
    the `.4` id's real position** (it is not first in `audioTracks`; a
    "first match" bug must fail a test). The fixtures' comment states the date and
    that expected values are confirmed by the yt-dlp runs in `docs/evidence/bug-013/`.
  - [x] 21.0c *(informative)* Find a real non-English video with the dubbing
    structure **and** uploaded English or original-language subtitles; record what
    the page says and what `--sub-langs <O>` returns, or say none was found. If
    step 1 or 2 cannot be shown on a real video, they are covered by unit tests
    only and 21.7 says so.
  - [~] 21.0d *(informative — not searched)* Creator-uploaded extra audio tracks (other
    `audioTrackId` suffixes).
- [x] 21.1 Pure functions, test-first: `originalAudioLanguage(ids)`, `findTrack`
  (the preference order above; English → `"en"`) and the new resolution over a
  parsed `captionInfo`. Table cases: the three real shapes from 21.0b; **`.4` =
  `vi`, uploaded `en` only, auto `vi` → `en` (step 2 beats step 3)**; **`.4` present
  with uploaded `en` + auto `O` — this case must carry a `.4` id, never reach `O`
  any other way**; uploaded `O` + uploaded `en` → `O`; neither uploaded → `O`'s
  auto code; `.4` present, no matching track → today's rules; two `.4` ids →
  today's rules; **no `.4` and a single auto track + uploaded `O` + uploaded `en` →
  today's rules (`en`) — pins the scope decision**; `de-DE.4` with auto `de` →
  `de`; uploaded `de-DE` for `.4` `de-DE` → `de-DE`; a custom-named uploaded
  `vi-abc123` loses to an exact/region-shaped match; **English stays `en`:**
  uploaded `en-US` + auto `en`, and `.4` `en-US` + auto `en` → `"en"`; malformed
  ids (`""`, `"x"`, `".4"`) → unknown.
- [x] 21.2 **No regression:** every existing `TestResolveDefaultLanguage` case
  passes with its expected value **unchanged; the diff to the existing table is
  additions only** (no edited or deleted cases), and
  `FuzzResolveDefaultLanguage` still never returns `""` and gets new seeds (an
  `audioTracks` payload, a truncated one). Note: no existing case distinguishes
  this task's behaviour from today's — the new cases in 21.1 must.
- [x] 21.3 **Parse/resolve split:** a parsed type `captionInfo{Tracks
  []captionTrack; AudioIDs []string}`, `parseCaptions(playerResponse)` and a
  pure `resolveFromCaptions(info)`; `resolveDefaultLanguage([]byte)` keeps its
  signature (= parse, then resolve), as do `ResolveLanguage`, the language memo,
  `LanguageNote` and the `lookupSpokenLanguage` seam, so the rest of
  `language_test.go` is untouched. (Task 20 will memoize `captionInfo`.)
- [x] 21.4 **Live smoke** (CLI built from the branch; record the yt-dlp version;
  paced ≥ 15 s; **record the resolved language from the note or `errors.log` for
  every video**, not only the transcript's language): `r8CppXSqVDU`,
  `B9MBdB1Ih6Q`, `Za_PoC0D3CQ`, `fdkYE4uxL0A`, language omitted → Vietnamese
  transcript, note `language: vi (auto-detected spoken language)`; `cZSgL76ddDs`,
  `0n5AYXkXP3Y`, `iLnTZhrkUpA` → `de` with the note; `vyIgAO8aCbA` → `en`, no
  note; unchanged: `9EUTRL_4Cj8` (vi), `z0hwcaKiHvQ` (ja), `dQw4w9WgXcQ` (`en`),
  `X0UI0O8YzJM`; **negative check:** `r8CppXSqVDU --language en` still asks for
  `en` (explicit language is honoured, BUG-011 message as before). If YouTube
  changes the structure, the smoke proves less and the unit tests are the coverage;
  say so.
- [x] 21.5 Docs: `docs/BUGS.md` BUG-013 decision + status; `docs/DECISIONS.md`
  DECISION-022 updated (the resolution order; why the `.4` id and not the
  viewer-dependent fields; why uploaded original-language subtitles now precede
  uploaded English on structured videos — the human's decision quoted); the
  ledger; task 20's Program design note; this file.
- [x] 21.6 `go build ./... && go vet ./... && go test ./... && gofmt -l internal/ cmd/`
  clean — **`gofmt -l` must print nothing** (it exits 0 even when it lists files);
  Grade run against these items.
- [x] 21.7 A note in this file records which of steps 1–3 were exercised on a real
  video and which only by unit tests.

## Test Plan

- **Unit:** 21.1–21.3 in `language_test.go`: real-shape fixtures (21.0b), an
  `audioTracks` builder beside the existing payload helpers; pure functions only,
  no network, no seam changes.
- **Research (live, scripted):** 21.0b–d, using `docs/evidence/bug-013/`
  (`research.py`, `locale_check.py`).
- **Live smoke:** 21.4.
- Commands: `go test ./internal/core/ -run 'ResolveDefaultLanguage|OriginalAudio|FindTrack' -v`;
  `go build ./... && go vet ./... && go test ./... && gofmt -l internal/ cmd/`.

## Program design

One file, `internal/core/language.go`; no new package, no `mcpserver`/`cli`
changes (every omitted-language caller already goes through `ResolveLanguage`).

```go
type captionTrack struct{ Language, Kind string }  // Kind "asr" = auto-generated, "" = uploaded
type captionInfo  struct {
    Tracks   []captionTrack
    AudioIDs []string // captions.playerCaptionsTracklistRenderer.audioTracks[].audioTrackId
}

func parseCaptions(playerResponse []byte) captionInfo            // nil/garbage -> zero value
func originalAudioLanguage(ids []string) (string, bool)          // exactly one "<lang>.4" -> lang
func findTrack(tracks []captionTrack, lang string, asr bool) (string, bool) // exact > region-shaped > other same-base; English -> "en"
func resolveFromCaptions(info captionInfo) string                // steps 1-4; never ""
func resolveDefaultLanguage(playerResponse []byte) string        // SIGNATURE KEPT = resolveFromCaptions(parseCaptions(pr))
```

`captionTracksResponse` gains the `audioTracks[].audioTrackId` path. Call graph
unchanged: `ResolveLanguage → language memo → lookupSpokenLanguage →
resolveDefaultLanguage`.

## Notes for the implementer

- Every existing `TestResolveDefaultLanguage` case has no `.4` id, so under this
  scope it takes today's rules and keeps its value (checked case by case against
  `language_test.go`, incl. the four empty/no-captions/not-JSON/nil cases → `en`).
- Downstream effects of returning a track's own code (`de-DE`, `pt-BR`) were
  checked by reading the callers: `sanitizeLanguageForFilename` keeps `-`;
  `LanguageNote` / `languageMetaLine` work for any code but `en`; the cache key
  is `{videoID, language}`, so an auto-resolved `de-DE` and an explicit `de` are
  two cache entries (acceptable). That yt-dlp prefers an uploaded subtitle to an
  auto one for the same code (needed for step 1) is supported by
  `dQw4w9WgXcQ` returning its uploaded track for `--sub-langs en` but is **not
  independently verified for step 1's case** (21.0c).
- Step 2 keeps today's "any `en` / `en-*` uploaded track counts as English",
  including the pre-existing quirk that a variant-coded uploaded track
  (`UF8uR6Z6KLc`'s `en-eEY6OEpapPo`) resolves `en` although `--sub-langs en`
  would not fetch it. Out of scope.
- The memo (`defaultLanguageMemo`) caches the resolved language per process, so
  no migration is needed.
- `search_playlist` does not resolve a language and is unchanged.

## Risks and open questions

- **Open question for the human — the general version.** The rule you gave is
  applied here only to videos where `O` is known from the `.4` id. Extending it to
  videos **without** the structure (using their single auto track as `O`) would
  reach every video, but changes today's `en` to the original language for videos
  that have uploaded subtitles in both — a behaviour change on videos not in
  BUG-013, on a population where today's rule was right 19/19. Advise recommended a
  separate follow-up; it is not part of this task unless you say so. (The
  tested Japanese case, `BIAUJLWoN4k` → `en`, holds either way: no `.4`, or no
  uploaded `ja`.)
- **The dangerous failure mode:** exactly one `.4` id that is *not* the original
  (e.g. a creator-dubbed track). Step 3 cannot catch it, because dubbed auto tracks
  exist anyway (`r8CppXSqVDU` lists an auto `en-US`); the result would be a
  confidently wrong non-English language on an English video. The defence is the
  evidence — 14/14 agreement with yt-dlp, four-locale stability — and nothing else.
  21.0d looks for such videos.
- **`.4` / `.10` is inferred, not documented** (localized "original" display names
  on the `.4` tracks; 14/14 agreement). If YouTube changes the suffixes the source
  stops matching and resolution falls back to today's rules — today's behaviour.
- **Evidence limits:** 41 videos, one region/day, one yt-dlp version; the
  structure appeared only on Vietnamese and German search results.
- **Steps 1–2 may not be demonstrable on a real video** (21.0c); then they are
  unit-tested only, stated in 21.7.
- **Interaction with task 20 (BUG-012 follow-up).** Task 20's plan assumes the
  language it is handed is right; this task makes it more often right. Land this
  first; **task 20's Program design (it memoizes `[]captionTrack` and moves
  "today's rules" into `languageFromTracks`) must be revised to memoize
  `captionInfo` and reuse `resolveFromCaptions`** before it is implemented.

## Out of scope

- The general version above; fixing the variant-coded uploaded-English quirk;
  rewording the BUG-011 message; anything about *fetching* (BUG-012 / task 20).
- Reading yt-dlp's `-J` `language` at runtime (rejected in BUG-013: an extra ~5 s
  per video when the page already carries the signal).
- Languages beyond what 21.0c/21.4 happen to include.

## Notes / deviations (implementation, 2026-09-27)

- **21.0b:** fixtures are in `internal/core/language_fixtures_test.go`, captured with
  `docs/evidence/bug-013/capture_fixtures.py` (all fields kept except `baseUrl`; real order;
  `vyIgAO8aCbA`'s `.4` is 11th of 21).
- **21.0c (real-video coverage):** `0n5AYXkXP3Y` and `iLnTZhrkUpA` have an uploaded `de` subtitle
  (yt-dlp `--list-subs` for the first) and `de-DE.4`, so **step 1 ran on real videos** — but the
  resolved code is `de` either way, so the smoke cannot tell an uploaded fetch from an auto one
  (not checked). No video with uploaded English + original-language auto was found.
- **21.0d:** not searched beyond the 41-video data, which shows only `.4` and `.10` suffixes.
- **Tie-break not settled by the task:** in `findTrack` a bare-base track (`de` for wanted
  `de-DE`) ranks with the region-shaped ones (rank 2), above custom-named variants; otherwise
  `de` and `de-abc` would tie. No specified case changes.
- **21.2:** the only edit to `language_test.go` is two added fuzz seeds; new cases live in
  `language_original_test.go`. Fuzz 10 s: 76k execs, no failure.
- **21.4 result:** 13/13 as specified, yt-dlp 2026.07.04, 15 s pacing
  (`docs/evidence/bug-013/smoke-task21-results.txt`); resolved language read from the note
  (`vi`/`de`/`ja`) or its absence (`en`); negative check `r8CppXSqVDU --language en` still gives the
  BUG-011 429 message.
- **21.7 — which steps were exercised where:** step 3 (auto original language) on real videos: the
  4 Vietnamese and `cZSgL76ddDs` (no uploaded tracks). Step 1 (uploaded original) on real videos:
  `0n5AYXkXP3Y`, `iLnTZhrkUpA` (see 21.0c caveat). **Step 2 (uploaded English beats auto original)
  is unit-tested only**; so are the tie-break tiers and the malformed-id cases.
- **Grade:** run on `haiku` against 21.0–21.7, no failing items (21.0c/d, 21.7 and the ledger/TASK.md
  parts of 21.5 were pending by construction and are done here). The grader did not print its model id.
- Also noted: a stray no-op subagent was spawned by mistake during grading; it did nothing.

## Review log

- 2026-09-27: drafted by Execute (rev 1). Human decision on precedence received
  during drafting and folded in (quoted above).
- 2026-09-27: **Advise** called (`opus`, reported `claude-opus-5-5`); row in
  `docs/eagd-log.md`. Prior leanings: (1) include the single-auto-track extension
  (b); (2) English stays the literal `en`; (3) no new logging; (4) land before
  task 20, as separate tasks. Answer: **drop (b)** (its only effect is a behaviour
  change on videos that are not in BUG-013, and my "it keeps the Japanese case"
  argument was wrong: that case is `en` without it), keep 2, keep 3 (not
  challenged), keep 4 with a shared `captionInfo` type; only 21.0b blocks; extra
  test cases (`.4`+uploaded `en` only, `.4` position, real track order, scope pin);
  additions-only test diff; `gofmt -l` must print nothing; negative smoke check;
  tie-break order for variant-coded tracks; name the "single wrong `.4`" mode.
  Verified before adopting: (b)'s steps 2 and 3 equal today's rules 1 and 2, so
  only step 1 differs; my derivation of the existing cases matched Advise's.
  Taken: all of the above.
- 2026-09-27: **Human review: approved as drafted** ("approve task 21"). The two
  confirmation questions put to the human with the draft (the "uploaded English"
  reading, and the general version) were not answered separately; approval is
  recorded as covering the first (accepted by approving the text) and **not** the
  second (still an open, separate decision).
