# EAGD log

Append-only. One row per event, one line per row, no line breaks inside a cell, a literal `|` written as `\|`, `—` for a field that doesn't apply. Rationale: [execute-advise-grade-dream.md](execute-advise-grade-dream.md). Mechanism: `CLAUDE.md`, "Execute / Advise / Grade".

## Advise calls

| Date | Branch | Question | Prior leaning | Answer | Taken | Tool | Requested | Reported | Status |
|------|--------|----------|---------------|--------|-------|------|-----------|----------|--------|
| 2026-09-20 | feat/task-18-native-language-fallback | Fall back to the native caption track after an en 429, or resolve the default language from captionTracks before fetching? | Fallback on 429 (smallest diff, no change on working paths) | Resolve first + per-video memo: the 429 category cannot tell a translated-track rejection from a real throttle, so a fallback could silently serve the wrong language; also drop parsedTranscript.Language, keep the language note out of the transcript body, add a DECISIONS entry for the Save/brief scope cut | Answer taken, except naming the spoken language in the BUG-011 message (deferred as a follow-up) | Agent | opus | claude-opus-5 | ok |
| 2026-09-20 | feat/task-18-native-language-fallback | Task 19 mechanism per surface: how should SaveTranscriptFile, FetchVideoBrief and SearchPlaylist apply ResolveLanguage (resolve first vs lazily; where the note goes)? | Save: resolve in handlers + conditional file-header line; Brief: resolve before the goroutines; Playlist: resolve per video before the cache check | Save: as leaning, plus add a _lang filename suffix; Brief: resolve inside the transcript goroutine; Playlist: resolve lazily only on failure, retry once, pace the retry, gate on rate_limited | At the time: save header line taken, filename suffix not taken (collision judged narrow, left as an open question), Brief and Playlist answers taken (playlist retry gated on omitted language + rate_limited + resolved != en). After human review the same day: filename suffix taken for EVERY language including en (human decision), and the playlist recommendation parked as a future improvement in favor of a minimum change (description + existing skip-line hint only) | Agent | opus | claude-opus-5 | ok |
| 2026-09-26 | main | Video vyIgAO8aCbA: `--sub-langs en` 429s (yt-dlp maps `en` to lang=uk&tlang=en) while `en-orig` works. Fall back to `<lang>-orig` on a 429, go orig-first, request both, or decide up front from captionTracks? | Keep plain `<lang>` primary; on a 429 only, retry once with `<lang>-orig`; extend pickVttFile | Take the leaning with changes: trigger from the existing rate_limited category; skip the retry if the language already ends in -orig; cache a successful retry under the caller's language; on a failed retry return the ORIGINAL 429 error (not "no captions"); pin pickVttFile's first-.vtt fallback with a test rather than adding an -orig branch; reject orig-first (regresses uploaded subs), `en-orig,en` and up-front detection; log as new BUG-012, not a BUG-011 follow-up; list six live checks first | Not yet applied: BUG-012 entry and human decision pending; live checks in progress | Agent | opus | claude-opus-5-5 | ok |

## Binding changes

| Date | Role | Tool | Old | New | Reason |
|------|------|------|-----|-----|--------|
| 2026-09-19 | advise | Agent | — | opus | initial install, probe reported claude-opus-5 |
| 2026-09-19 | grade | Agent | — | haiku | initial install, probe reported claude-haiku-4-5-20251001 |
| 2026-09-26 | advise | Agent | opus (reported claude-opus-5) | opus (reported claude-opus-5-5) | the `opus` alias now reports claude-opus-5-5; the reported id no longer matched the row, so the row was updated on human instruction rather than left stale |

## Grade fallbacks

| Date | Branch | Tool | Reason |
|------|--------|------|--------|
