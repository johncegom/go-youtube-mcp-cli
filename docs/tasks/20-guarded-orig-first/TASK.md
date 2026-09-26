# Task 20: Guarded orig-first transcript fetch (BUG-012 follow-up)

**Status:** IMPLEMENTED on branch `feat/task-20-guarded-orig-first` (2026-09-27), Grade run, not merged. APPROVED rev 3 (2026-09-27) — the human approved this Definition of Done + Test Plan
(rev 3: rev 2 + Program design revised after task 21). Restricted to English `L` (human decision after 20.0b). All sub-tasks done except 20.0c (not done, informative). Rev 2 incorporates an
Advise call (logged in `docs/eagd-log.md`) and four checks run against its
claims (see "Review log"). Written on the `docs/bug-012-measurement-evidence`
branch next to the evidence it rests on.

**Sequencing note — task 21 has landed (PR #38, 2026-09-27); Program design
revised the same day (rev 3).** This task assumes the language `L` it is handed
is right; BUG-013 (`ResolveLanguage` returning `en` for the Vietnamese
`r8CppXSqVDU`) was fixed by task 21, which added
`captionInfo{Tracks []captionTrack; AudioIDs []string}`, `parseCaptions` and
`resolveFromCaptions` to `internal/core/language.go`. This task builds on them:
the memo holds `captionInfo`, `parseCaptionTracks` and `languageFromTracks` are
**not** added (they are `parseCaptions` and `resolveFromCaptions`), and
`captionTrack` is not redefined. **What task 21 changes for this task's
evidence:** `L` may now be a *non-English* code (`vi`, `de`, `de-DE`) on the
default path, so the plan `[L-orig, L]` will fire for auto-caption-only
non-English videos too. Whether `L-orig` is genuine there is exactly 20.0b —
it is no longer "informative", see 20.0b and 20.7.

## User need

BUG-012 (PR #36) made a rate-limited fetch retry once with `<lang>-orig`. That
turns a hard failure into the real transcript, but three costs remain:

1. **A doomed request every time.** For an auto-caption-only video plain `en`
   is expected to 429 (5–11 s per failed attempt in `errors.log`) before the
   retry succeeds (3–6 s). Every uncached fetch pays both.
2. **A silent-degradation gap.** The retry only acts on an error. On two CLI
   runs of `vyIgAO8aCbA` plain `en` did *not* 429 and returned a machine
   back-translation (uk → en). No error, so no retry. Frequency **unmeasured**
   (n = 2, one video). This is the only thing task 20 can fix that PR #36
   cannot, and it is a correctness problem, not a latency one.
3. **Extra requests to an endpoint that is throttling us.**

Evidence (`docs/evidence/bug-012/README.md`; 17 videos, 15 drawn from a
failures-only log, single pass, yt-dlp 2026.07.04, **not** the 2026.08.19 the
app ran):

| Video has… | Plain `en` | `en-orig` | Wanted |
|---|---|---|---|
| an **uploaded** track under exactly `en` (6 videos) | ok; the uploaded track was returned on 4 confirmed, 2 not scored | is the *auto* track (0.44–0.95 similar to the uploaded text) | plain `en` |
| an uploaded track only under a **variant code** (`UF8uR6Z6KLc`: `en-eEY6OEpapPo`) | ok, but returned the **auto** track (identical to `en-orig`) | ok, auto | either; plain `en` is today's |
| only **auto** captions, an `asr` track with code exactly `en` (7 videos, ~20 tracks each) | 429 (7/7) | ok (7/7) | `en-orig` |
| no `asr` track coded exactly `en` (`r8CppXSqVDU`: `en-US`+`vi`; `9EUTRL_4Cj8`; caption-less) | 429 or none | none | plain `en` (today's behaviour) |

Unguarded orig-first is rejected: it would replace an uploaded transcript with
speech recognition and mislabel `caption kind`. Two page signals separate the
rows, both already in the `captionTracks` the app scrapes:

- *an uploaded (non-`asr`) track exists for `L`* — matched yt-dlp's own list on
  17/17 videos (`pagecheck-results.txt`);
- *an `asr` track whose `languageCode` is exactly `L` exists* ⇔ *yt-dlp lists
  `L-orig`* — 0 mismatches on 17 videos (`pagecodes-results.txt`; this
  is what restores the "en-orig **if listed**" condition).

## Approach

Decide the attempt order **before** the first yt-dlp call, from the track list
**only if it is already in memory**; keep the BUG-012 retry as the safety net.

1. **Peek, do not fetch.** The fetch funnel reads the per-video track memo
   without any network call (`peekCaptionInfo`). Omitted-language callers
   have it warm (`ResolveLanguage` ran first); explicit-language callers and
   `search_playlist` do not, and get today's `[L]` + BUG-012 retry. So this task
   adds **no request and no latency**, and leaves DECISION-022's playlist scope
   cut intact.
2. **Plan** for requested language `L` (pure):
   - memo cold, `L` already ends in `-orig`, `L`'s base language is not `en`, or an
     uploaded track exists for
     `L` (`L` or `L-*`, i.e. **prefix**, see below) → `[L]`;
   - else, **only when `L`'s base language is `en`** (human decision after 20.0b), an
     `asr` track with `languageCode == L` exists → `[L-orig, L]`;
   - else → `[L]`.
3. **Run** the plan in order; the first success wins. If every attempt fails,
   return the error of the attempt for the **requested** language `L` (a
   non-English video asked for `en` keeps its 429 + BUG-011 hint, not an
   `-orig` "no captions" error). After a plain `L` failure classified
   `rate_limited`, `L-orig` is appended **only if not already attempted** (this
   is the shipped BUG-012 retry, folded in — one mechanism, no repeated request).
4. Cache key stays `{videoID, L}`: a success from `L-orig` is cached under what
   the caller asked for.

**Why prefix, not exact, for "uploaded exists".** The two mistakes are not
symmetric. Treating a variant-coded uploaded track as "exists" errs toward
`[L]` — today's behaviour plus the retry, safe. Requiring an exact match errs
toward `[L-orig, L]`, which is right only if the page's code equals yt-dlp's
key, and `UF8uR6Z6KLc` shows it does not (`en-eEY6OEpapPo`). Pin that shape in
a test.

Explicitly **not** doing:

- Reading `tlang=` from a yt-dlp `-J` run to decide (predicted the 429 on 17/17
  but costs an extra ~5 s extraction on every uncached fetch — more than the
  doomed request it saves on the common uploaded path).
- Fetching the watch page from the funnel when cold (rejected: it would widen
  DECISION-022 and add ~1 s + a request to every explicit-language call).
- Changing yt-dlp version, flags or the BUG-011 message text; deciding what
  language `L` should be (BUG-013).

## Definition of Done

`[x]` done · `[ ]` not started. Numbering follows task 10's sub-task pattern.

- [x] 20.0 **Research (20.0b now gates the plan for non-English `L`; the rest are informative)** — a negative
  or "could not find one" result is a valid outcome, recorded in
  `docs/evidence/bug-012/`:
  - [x] 20.0a *Page track codes vs yt-dlp `-orig` keys* — done 2026-09-27:
    `pagecodes.py` / `pagecodes-results.txt`, 0 mismatches / 17. (Replaces the
    earlier "exact vs prefix" gate: the artifacts already settle it, see
    Approach.)
  - [x] 20.0b **(blocks coding the plan for non-English `L`)** — **done 2026-09-27, result below; a plan change is pending the human's decision.** *Is `<L>-orig`
    genuine for non-English speech?* Task 21 made the default `L` a track's own
    code (`vi`, `de`); task 21's live check saw `de-orig` return the identical
    file as `de` on `cZSgL76ddDs`, and plain `vi`/`de` fetch real transcripts,
    so the plan `[vi-orig, vi]` could be a wasted attempt or harmless. Measure
    on `r8CppXSqVDU`, `B9MBdB1Ih6Q`, `Za_PoC0D3CQ`, `fdkYE4uxL0A`, `cZSgL76ddDs`,
    `0n5AYXkXP3Y`, `iLnTZhrkUpA`: does `<L>-orig` exist, does it 429, does it
    differ from plain `<L>`? If it adds nothing, restrict the plan to
    `L` whose base is `en`. (The BUG-013 half of the old question is closed by
    task 21.)
    **Result** (`docs/evidence/bug-012/orig_nonenglish.py`, `orig_nonenglish-results.txt`;
    yt-dlp 2026.07.04, 7 videos, one pass, alternating order, 15 s pace): **`<L>-orig`
    exists on all 7 and never 429ed, and plain `<L>` never 429ed either** (`vi` x4, `de` x3).
    - Auto-only videos (5): `-orig` text **identical** to plain on 4 (`r8CppXSqVDU`,
      `B9MBdB1Ih6Q`, `Za_PoC0D3CQ`, `cZSgL76ddDs`); `fdkYE4uxL0A` 0.87 similar
      (72581 vs 72587 chars, unexplained, one run).
    - `0n5AYXkXP3Y`, `iLnTZhrkUpA` (uploaded `de` exists): plain `de` is the **uploaded**
      track (9077 / 12830 chars), `de-orig` the auto one (22841 / 32140 chars, 0.51
      similar) — the guard case: orig-first unguarded would replace the uploaded text,
      and `hasUploadedTrack` correctly yields `[L]` there.
    **Reading:** for non-English `L` the plan `[L-orig, L]` buys nothing measurable — plain
    `L` already returns the original-language track and does not 429 — so it would be
    an extra attempt-order rule on a path with no observed failure. Per this item's own
    rule, "if it adds nothing, restrict the plan to `L` whose base is `en`". **This changes
    the plan (`fetchPlan` returns `[L]` unless base(L) is `en`) so it goes to the human
    before 20.1 is written**; caveats: n = 7, only vi/de, one day, no non-English
    video that 429s on plain `L` was seen.
  - [~] 20.0c **not done** — *A sample not drawn from `errors.log`* (≥ 10 videos, mixed
    languages, with and without uploaded subtitles) with the saved scripts.
  - [x] 20.0d *Is the timedtext URL printed on a **successful** fetch by the
    yt-dlp the app actually resolves?* Verified on 2026.07.04 (yes,
    `[debug] Invoking http downloader on "…timedtext…"`); the app's
    2026.08.19 was only seen printing it on failure. If it does not, 20.5's
    translation log line is dropped and the plan line is kept.
- [x] 20.1 Pure functions, test-first (**plan restricted to base-`en` `L`**; add cases: real `vi`/`de`
  auto-only tracks → `[L]`, `de` and `de-DE` with an exact/other auto track → `[L]`): `hasUploadedTrack` (prefix),
  `hasASRTrack` (exact), `fetchPlan` (over `captionInfo`; parsing is task 21's
  `parseCaptions`, already tested). The
  `fetchPlan` table includes: cold memo → `[L]`; known, zero tracks → `[L]`;
  only `asr` `en-US` (the `r8CppXSqVDU` shape) → `[L]`; uploaded exact `en` →
  `[L]`; uploaded only under a variant code (the `UF8uR6Z6KLc` shape) → `[L]`;
  `asr` exact `en`, no uploaded → `[L-orig, L]`; `asr` exact `en` **and**
  uploaded → `[L]`; `L` ending in `-orig` → `[L]`.
- [x] 20.2 `captionInfo` memo, additive: a per-video memo beside the language memo
  (same cap/clear-when-full shape; the language memo, `lookupSpokenLanguage`
  and `resolveDefaultLanguage([]byte)` keep their signatures, so every
  existing `language_test.go` case passes **without edits**); the default
  `lookupSpokenLanguage` is rebuilt on the `captionInfo` memo
  (`resolveFromCaptions(info)`). Tests: (a) stub
  `lookupCaptionInfo`, call `ResolveLanguage` and then the funnel's plan
  step, assert **exactly one** lookup (this is what "one page fetch" means, and
  a stub of `lookupSpokenLanguage` alone cannot prove it); (b) on a cold memo
  the funnel makes **zero** lookups; (c) a lookup error is not memoized and
  still yields `en`.
- [x] 20.3 `fetchWithPlan(plan, fetch)` replaces `fetchWithOrigRetry` /
  `origRetryLanguage`. The assertions of the ported `TestFetchWithOrigRetry`
  and `TestOrigRetryLanguage` stay **unedited** (only the call target
  changes), so PR #36's behaviour stays pinned. New cases: orig fails, plain
  succeeds; both fail → the requested language's error; `L-orig` never
  attempted twice.
- [x] 20.4 Wired in `fetchSegmentsFromYtDlp`; `fetchTranscript`'s cache key
  unchanged. A `fetchTranscript`-level test with an injected fetcher proves a
  success from `L-orig` is cached under `{videoID, L}` (Grade flagged this as
  untested on PR #36).
- [x] 20.5 **Observability**, two lines, both format-pinned by unit tests and
  both extending the existing style (`transcript_fetch_orig_retry` is kept):
  - `transcript_fetch_plan <id>: lang=L plan=<a,b> outcome=<attempt that succeeded | failed> duration=…`
    for every fetch whose plan is not just `[L]`;
  - `transcript_fetch_translated <id>: lang=L tlang=<src>` when a **successful**
    attempt's yt-dlp verbose output shows a timedtext URL with `tlang=`
    (plain `en` "succeeding" on a translated track — the silent back-translation).
    Only when it occurs, not on every fetch, so `errors.log` stays an error log.
    Subject to 20.0d.
  Together they make "how often does orig-first fire" and "how often does plain
  `en` silently succeed on a translation" countable from `errors.log`.
- [x] 20.6 Unit tests for 20.1–20.5 (no network; seams stubbed as
  `withStubbedLookup` does in `language_test.go`).
- [x] 20.7 **Live smoke** (CLI built from the branch; record the yt-dlp version
  actually used; paced ≥ 15 s between videos):
  - one Vietnamese and one German video from task 21's smoke (e.g.
    `r8CppXSqVDU`, `cZSgL76ddDs`), language omitted → transcript in the same
    language as before this task, plan/log per the 20.0b outcome;
  - `vyIgAO8aCbA`, `BqRhBq-_kgE`, language omitted → genuine `en-orig` text,
    log shows `plan=en-orig,en` succeeding and **no** `lang=en` 429 line;
  - `X0UI0O8YzJM`, `dQw4w9WgXcQ` (uploaded) → plan `[en]`, no `-orig`
    attempt, transcript unchanged; `get_video_brief` on `dQw4w9WgXcQ` still
    reports `caption kind` uploaded;
  - `UF8uR6Z6KLc`, `iG9CE55wbtY` unchanged;
  - an explicit `--language en` on `vyIgAO8aCbA` (cold memo) → today's
    behaviour: plain `en`, retry on 429, no lookup;
  - `r8CppXSqVDU --language en` → still the BUG-011 message.
  Worst case per fetch is two attempts, each bounded by
  `transcriptFetchTimeout` (30 s) — the same 60 s bound the shipped retry has.
  If YouTube stops rejecting plain `en`, the smoke proves less and the unit
  tests are the coverage; say so in the note.
- [x] 20.8 Docs: BUG-012 decision/status updated; its option 4 ("decide up
  front from `captionTracks`", rejected) marked **superseded**, with the reason
  (the objection "never runs for an explicit `en`" is accepted here: explicit
  callers keep today's path, and the plan uses the track list only when already
  known); a `docs/DECISIONS.md` entry (guarded, peek-only orig-first; `-J`/`tlang`
  and unguarded orig-first rejected, with evidence links); `docs/LEDGER.md` row
  and backlog cross-references; this file.
- [x] 20.9 `go build ./... && go vet ./... && go test ./... && gofmt -l internal/ cmd/`
  clean; Grade run against these items.

## Test Plan

- **Unit:** 20.1–20.6, pure functions and the composition with an injected
  fetcher; seams stubbed like `withStubbedLookup`.
- **Research (live, scripted, before or alongside coding):** 20.0b–d with the
  scripts in `docs/evidence/bug-012/`.
- **Live smoke:** 20.7, paced — the earlier measurement may itself have
  contributed to throttling.
- Commands: `go test ./internal/core/... -v`;
  `go build ./... && go vet ./... && go test ./... && gofmt -l internal/ cmd/`.
- If the MCP path is smoke-tested, hold stdin open (RETRO-002); never
  `printf | server`.

## Program design

Files: `internal/core/language.go` (tracks, memo), `internal/core/transcript.go`
(plan + wiring). No new package; no `mcpserver`/`cli` changes (every
omitted-language caller already calls `ResolveLanguage`, then funnels into
`fetchTranscript`).

```go
// language.go — captionTrack, captionInfo, parseCaptions, originalAudioLanguage,
// findTrack, resolveFromCaptions and resolveDefaultLanguage already exist (task 21).
func hasUploadedTrack(tracks []captionTrack, lang string) bool     // pure; prefix: lang or lang-*
func hasASRTrack(tracks []captionTrack, lang string) bool          // pure; exact languageCode

// ADDITIVE: a second small memo (video -> captionInfo) and its own stub seam.
var lookupCaptionInfo = func(ctx context.Context, videoID string) (captionInfo, error)
var defaultInfoMemo = newInfoMemo(256)
func captionInfoFor(ctx context.Context, videoID string) (captionInfo, bool) // fetches on miss; !ok on error, not memoized
func peekCaptionInfo(videoID string) (captionInfo, bool)                      // memo read only, NEVER the network
// lookupSpokenLanguage's DEFAULT body becomes: info, ok := captionInfoFor(...); return resolveFromCaptions(info)
func ResolveLanguage(ctx, videoID, requested string) string       // contract unchanged

// transcript.go
func fetchPlan(info captionInfo, warm bool, lang string) []string                // pure; the "Approach" step 2, reads info.Tracks
func fetchWithPlan(plan []string, fetch func(lang string) (parsedTranscript, error)) (parsedTranscript, error)
func formatPlanLog(lang string, plan []string, succeeded string, elapsed time.Duration, err error) string
func formatTranslatedLog(lang, tlang string) string
```

Call graph (language omitted, cache miss):

```
handler/CLI ─▶ ResolveLanguage ─▶ language memo ─▶ lookupSpokenLanguage ─▶ captionInfoFor ─▶ [watch page GET, once]
     │                                                                          │ warms captionInfo memo
     └─▶ fetchTranscript(videoID, L) ─▶ defaultCache.getOrFetch({videoID, L})
                                          └▶ fetchSegmentsFromYtDlp
                                               ├▶ peekCaptionInfo   (memo read; cold ⇒ plan [L])
                                               ├▶ fetchPlan ─▶ [L] | [L-orig, L]
                                               └▶ fetchWithPlan ─▶ fetchSegmentsOnce(L') per attempt
                                                     └▶ per-attempt failure lines + plan line (+ translated line)
```

`fetchWithOrigRetry` / `origRetryLanguage` (BUG-012) fold into `fetchWithPlan`;
their tests are ported, not deleted.

## Risks and open questions

- **Small effect where the memo is cold.** Explicit-language callers and
  `search_playlist` are unchanged (that is the point of peek-only), so they keep
  the doomed request where it happens. Acceptable: they are not the default
  path, and the retry recovers them.
- **The benefit that only this task delivers rests on n = 2** (the silent
  back-translation). If the human does not accept that as the reason, "keep
  PR #36 and stop" is a defensible outcome (Advise). The latency and throttling
  savings depend on how common the 429 is, which the failure-biased sample
  cannot say.
- **Evidence is thin and dated.** "Uploaded exact `en` ⇒ plain `en` returns the
  uploaded track" is confirmed on 4 videos (2 more not scored; the 7th
  succeeded but returned the auto track). One yt-dlp version, one day, one
  network; the ~20-track structure appeared between 2026-09-20 and 09-26, so
  the ground can move again. The plan degrades to today's behaviour when its
  assumption fails, and the retry stays as the safety net.
- **`en-orig` genuineness on non-English speech is unmeasured** (20.0b).
- **`L` is now a track's own code.** After task 21, `L` may be `vi`, `de` or
  `de-DE`. `hasASRTrack` is an exact match on `L`, so `[L-orig, L]` is planned
  only when the page lists an auto track with exactly that code; 20.1 should pin a
  `de` / `de-orig` case and a `de-DE`-with-only-auto-`de` case (→ `[L]`).
- **Two memos for one page.** Chosen over replacing the language memo because
  that would edit ~24 references in `language_test.go` and change the task
  18/19 seam for no user-visible gain; the cost is ~25 duplicated lines of memo
  code (RETRO-003: not worth a generic yet). Advise agreed.
- **Track memo is capped by count, not content** (256 videos, cleared when full,
  like the language memo); a track list is ~20 small structs. Stated so nobody
  adds an LRU.
- **Peek can go stale.** A video whose memo was cleared between
  `ResolveLanguage` and the fetch just gets `[L]` + retry; no correctness
  issue.

## Out of scope

- What language `L` should be (BUG-013); naming the spoken language in the
  BUG-011 error or rewording it (its "set the language" advice is wrong for the
  videos this task helps, but that is its own decision).
- Upgrading or pinning yt-dlp; flag differences between the CLI and a bare
  yt-dlp (unexplained in the evidence).
- Languages other than what 20.0c happens to include (only `en` was swept).
- A `tlang`-based or `-J`-based decision (rejected above).

## Notes / deviations (implementation, 2026-09-27)

- **Plan restricted to English `L`** (human decision after 20.0b): `fetchPlan` returns `[L]` unless
  base(`L`) is `en`. Non-English videos keep plain `L` plus the shipped retry.
- **20.0c not done.** No sample of >= 10 mixed-language videos beyond the 41 already in
  `docs/evidence/`; informative only, the plan degrades to today's behaviour when its assumption fails.
- **20.0d verified on both yt-dlp binaries:** a *successful* `-v` fetch prints
  `[debug] Invoking http downloader on "...timedtext..."` on 2026.07.04 (`dQw4w9WgXcQ`, `vyIgAO8aCbA`)
  and on 2026.08.19 (`en-orig` on `vyIgAO8aCbA`; that binary is at `%TEMP%\ytpath-bug012\yt-dlp.exe`,
  the one the app resolved during the smoke). So the `transcript_fetch_translated` line is kept.
- **Third observation of the silent back-translation:** plain `en` on `vyIgAO8aCbA` with 2026.07.04
  *succeeded* via `lang=uk&tlang=en` (no error). The smoke run with 2026.08.19 429ed instead and the
  retry recovered. The real URL is the fixture in `TestTimedtextTranslation`. No live run of *this
  branch* produced a `transcript_fetch_translated` line (no translated success occurred); that line
  is covered by unit tests on the real URL shape only.
- **`transcript_fetch_translated` format:** `lang=<requested> source=<lang param> tlang=<tlang param>`
  (the draft's `tlang=<src>` was ambiguous). `formatPlanLog` takes the succeeded language instead of an
  `err` (`outcome=<lang>` or `failed`).
- **`origRetryLanguage` kept** (with its unedited test) and reused by `fetchWithPlan`; only
  `fetchWithOrigRetry` is replaced. A `fetchOnce` package variable is the new test seam.
- **The `transcript_fetch_orig_retry` line** is now logged only for attempts the plan did not include
  (the rate-limit retry), so a planned `-orig` first attempt is not mislabelled a retry.
- **20.7 (live smoke, `docs/evidence/bug-012/smoke-task20-results.txt`; the app ran yt-dlp 2026.08.19):**
  `r8CppXSqVDU` -> vi, `cZSgL76ddDs` -> de, no plan line (restricted); `vyIgAO8aCbA`, `BqRhBq-_kgE` ->
  `plan=en-orig,en outcome=en-orig` (7.5 s), no `lang=en` failure line; `X0UI0O8YzJM`, `dQw4w9WgXcQ`,
  `UF8uR6Z6KLc`, `iG9CE55wbtY` -> no plan line, unchanged; explicit `--language en` on `vyIgAO8aCbA`
  (cold memo) -> today's path: plain `en` 429, `en-orig` retry recovered; `r8CppXSqVDU --language en` ->
  the BUG-011 message. `get_video_brief` on `dQw4w9WgXcQ` via the MCP binary (stdin held open):
  "Captions: likely uploaded".
- **Grade** (`haiku`): 9 PASS, 3 PARTIAL, 2 FAIL on the first pass. FAIL 20.0c: accepted (not done, above).
  FAIL 20.0d and PARTIAL 20.5/20.7 (`get_video_brief`) were real gaps, closed above with live checks.
  PARTIAL 20.8: the ledger row and these checkboxes were pending by construction, done here.

## Review log

- 2026-09-27: drafted by Execute (rev 1).
- 2026-09-27: **Advise** called (`opus`, reported `claude-opus-5-5`); full row in
  `docs/eagd-log.md`. Prior leanings: (1) additive second memo, (2) fold the
  retry into one mechanism, (3) plan in the shared funnel. Answer: agree on 1
  and 2, **flip 3** to peek-don't-fetch; put `L-orig` first only when an `asr`
  track with the exact code exists (the draft had dropped "if listed"); use
  prefix for "uploaded exists"; drop the exact-vs-prefix research as a gate;
  don't gate on 20.0b; more test cases; log `tlang` on success. Checked before
  adopting: `UF8uR6Z6KLc`'s uploaded key is `en-eEY6OEpapPo` and plain `en`
  returned the auto track (**Advise right; the draft's "7/7" was wrong**); exact
  asr-`en` on the page ⇔ yt-dlp `en-orig`, 0 mismatches / 17; the timedtext URL
  is printed on a successful fetch on 2026.07.04 (not checked on 2026.08.19).
  Not taken as written: Advise's "log `tlang` on **every** uncached fetch" —
  narrowed to when it occurs, to keep `errors.log` an error log. Advise's side
  observation that `resolveDefaultLanguage` returns `en` whenever any `en-*`
  track exists was **confirmed live** and is now BUG-013.
- 2026-09-27: **rev 3** after task 21 merged (PR #38): memo type and function
  names moved to `captionInfo`; 20.0b promoted to a gate for non-English `L`; no
  change to the approach or scope. Not re-Advised (mechanical revision, no new
  judgement call). Human re-approval of rev 3 was requested.
- 2026-09-27: **Human review: rev 3 approved** ("approve task 20 rev 3"), after PR #39 merged. The approval covers the text as written; 20.0b remains a gate for the non-English part of the plan, and a result that changes the plan goes back to the human.
- 2026-09-27: **Human decision on 20.0b:** restrict to en - fetchPlan returns [L] unless L base language is en; 20.1 started. (A restriction, not new scope; reversible if a non-English video ever 429s on plain L.)
