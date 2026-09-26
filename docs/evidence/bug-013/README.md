# BUG-013 evidence: where does the watch page carry a video's original language?

Research behind `docs/BUGS.md` BUG-013 (`resolveDefaultLanguage` returns `en` for a non-English video that
lists an `en-*` auto-caption track). Collected 2026-09-27 on one Windows machine / one network. **Read the
caveats before relying on any number.**

## Question

Task 18's `ResolveLanguage` decides the default transcript language from the watch page's `captionTracks`:
any `en` / `en-*` track ⇒ `en`. On 2026-09-26 `r8CppXSqVDU` (Vietnamese) resolved `en` and failed with the
BUG-011 429. Is there a signal *already in the page the app fetches* that says what language the video was
actually recorded in, and does it agree with yt-dlp's own `language` field?

## TL;DR

- **Today's rule is wrong on 7 of the 33 sampled videos that yt-dlp gives a language for**: every non-English
  video that has YouTube's auto-dubbing structure (4 Vietnamese, 3 German) resolves `en`. It is right on the
  other 26.
- **The signal:** `captions.playerCaptionsTracklistRenderer.audioTracks[].audioTrackId`. On videos with the
  structure there is exactly one id ending `.4` (the original audio: `vi.4`, `de-DE.4`, `en-US.4`); all others
  end `.10` (auto-dubbed). **`.4`'s language equalled yt-dlp's `language` on 14 / 14** such videos, and the
  `.4` id was **identical across four viewer locales** on the 3 videos tested.
- **Do not use** the streamingData `audioIsDefault:true` track, `hostLanguage` or `requestLanguage`: they
  depend on the *viewer's* locale (measured: the default original track disappears for an `en-US` viewer of a
  Vietnamese or German video).
- The audio language is regional (`de-DE`, `en-US`) but the auto-caption track that would be requested uses the
  base code (`de`, `en`); Vietnamese matches exactly. A resolver needs a base-language match.

## Sample and method

41 videos: the 17 usable from `docs/evidence/bug-012/` (15 of them originally drawn from the failure-only
`errors.log`) plus 24 new ones from `yt-dlp "ytsearch4:<query>" --flat-playlist` in six languages (top 4 per
query; the query language is a rough proxy for the spoken language, not ground truth):

| intended | query |
|---|---|
| vi | `tin tức thời sự hôm nay` |
| es | `noticias de hoy en vivo` |
| ja | `ニュース 今日 解説` |
| uk | `новини україна сьогодні` |
| hi | `हिंदी समाचार आज` |
| de | `nachrichten heute deutschland` |

Per video (`research.py` → `research.json`): a plain GET of the watch page with the app's own User-Agent
(`fetchWatchPageHTML`), from which it extracts the caption tracks (`languageCode`, `asr` vs uploaded),
`captions.audioTracks` ids and the streamingData default audio track; today's rule is recomputed offline
(`current_rule` in the script mirrors `resolveDefaultLanguage`); and one `yt-dlp -J --skip-download` (2026.07.04)
for its `language` field. Paced (1.5 s + 6 s per video). yt-dlp's `language` is used as the **reference, not as
independent ground truth**; it agreed with the search's intended language for every non-English video for which it
reports one.

## Results

`analyze_research-results.txt` has the per-video table. Summary:

| Group | Videos | Today's rule | Note |
|---|---|---|---|
| Non-English, has the `.4`/`.10` structure | 7 (4 vi, 3 de) | **wrong on 7 / 7** (all resolve `en`) | `.4` = original, matches yt-dlp on 7 / 7 |
| English, has the structure (2–21 tracks) | 7 | right (`en`) | `.4` = `en-US`, matches yt-dlp on 7 / 7 |
| No structure, yt-dlp reports a language | 19 (en, vi, es, ja, uk, hi, de) | right on 19 / 19 | no `audioTracks` in the page |
| No language from yt-dlp (no auto captions) | 8 | resolves `en` (no data) | not assessable |

- Of the 24 new videos, the structure appeared on 3 / 4 Vietnamese and 3 / 4 German results and on **none** of
  the Spanish, Japanese, Ukrainian or Hindi ones.
- `.4` vs yt-dlp `language`: **14 / 14** videos with the structure had exactly one `.4` id whose language
  equalled yt-dlp's.
- Audio language → caption code: for `vi` the exact code exists (`vi`); for `de-DE` the caption track is `de`; for
  `en-US` it is `en` (`analyze_research-results.txt`, last block).
- **Locale check** (`locale_check.py` → `locale_check-results.txt`): for `r8CppXSqVDU`, `cZSgL76ddDs` and
  `vyIgAO8aCbA`, with `Accept-Language` none / `en-US` / `de-DE` / `ja-JP`, the `.4` id and the number of audio
  tracks are identical every time. The `audioIsDefault:true` track is not (absent for some locales).

## Caveats

- **Small, non-random sample at one moment.** Six search queries, 4 results each, one region, one day. The
  auto-dubbing structure is an ongoing YouTube rollout; its extent can change.
- **Meaning of `.4` / `.10` is inferred, not documented** — from the localized display names on `.4` tracks
  ("gốc", "(Original)", "オリジナル", "original") and from the consistent pattern across 14 videos.
- **Creator-uploaded extra audio tracks** (multi-language audio) were not seen; they may use other suffixes.
- **A non-English video with the structure *and* uploaded English subtitles** was not observed, so whether the
  uploaded English track or the original language should win is a decision the data does not make.
- **Reference is yt-dlp's `language`**, which itself was not verified against the audio for every video. English
  speech was confirmed by reading a transcript for `vyIgAO8aCbA`; Vietnamese for `r8CppXSqVDU` via BUG-011.
- **Single yt-dlp version** (2026.07.04); not the 2026.08.19 the app once ran.
- 8 videos have no yt-dlp language (no auto captions); nothing was learned about them.
- The app sends no `Accept-Language`; the server-side default depends on the requesting IP, so what other users
  see may differ from this machine's view (which is why viewer-dependent fields are unusable).

## Files

| file | what it is |
|---|---|
| `research.py` | Search, page scan and yt-dlp `-J` per video → `research.json`. Needs `%LOCALAPPDATA%\go-ytdlp\yt-dlp-2026.07.04.exe`; ~12 min, ~41 GETs + ~41 yt-dlp runs + 6 searches. |
| `research.json` | Its output: per video the intended language, caption tracks, audio-track ids, default audio track, today's rule and yt-dlp's `language`. Search results (IDs) are as of 2026-09-27. |
| `analyze_research.py` → `analyze_research-results.txt` | Offline analysis of `research.json`: rule vs yt-dlp, `.4` vs yt-dlp, audio language → caption code. |
| `locale_check.py` → `locale_check-results.txt` | The viewer-locale test (plain GETs). |

Re-running `research.py` will overwrite `research.json` and produce different search results.

## Related

`docs/BUGS.md` BUG-013 (options, updated recommendation) and BUG-012; `docs/evidence/bug-012/` (the fetch-side
measurement this builds on); `docs/tasks/20-guarded-orig-first/TASK.md` (draft; assumes the language it is
handed is right).
