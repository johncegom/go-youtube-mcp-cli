# BUG-012 evidence: plain `--sub-langs en` vs `en-orig`

Raw data, scripts and findings behind `docs/BUGS.md` BUG-012 and the follow-up it motivated (guarded
orig-first). Collected 2026-09-26 / 2026-09-27 on one Windows 11 machine, one network, by hand-driven
sessions. **Read the caveats before drawing conclusions from any number here.**

## TL;DR

- On today's YouTube, a plain `en` request for an **auto-caption-only** video can be turned by yt-dlp into a
  *translation* of another track (`lang=ar&tlang=en`, or earlier `lang=uk&tlang=en`), which YouTube rejects
  with HTTP 429. The genuine `en-orig` track downloads fine.
- In a 17-video sample, `tlang=` in the auto-caption `en` URL predicted the 429 with 0 mismatches
  (9/9 rejected, 7/7 accepted, 1 video without captions).
- Videos **with an uploaded English track** were never affected, and `en-orig` there is the *auto* track:
  a blanket orig-first would replace the uploaded transcript with speech recognition.
- The watch page's `captionTracks` (which the app already scrapes) marked "uploaded English track exists"
  identically to yt-dlp's own list on all 17 videos.
- One further failure mode was seen only intermittently and is **not** fixed by the shipped retry: plain `en`
  *succeeding* with a machine back-translation (see "Silent back-translation").

## Environment

| | |
|---|---|
| yt-dlp used for the measurement | `2026.07.04` (`%LOCALAPPDATA%\go-ytdlp\yt-dlp-2026.07.04.exe`). Also present: `2026.03.17`. |
| yt-dlp the app itself ran on 2026-09-26 | `stable@2026.08.19` per its own `errors.log` verbose block; that binary is no longer on disk, so **nothing here was measured on it**. The app installs with `AllowVersionMismatch: true` (`internal/core/ytdlp.go`), i.e. it runs whichever yt-dlp it resolves first, not the pinned one. |
| Flags (as the app) | `--no-check-formats --skip-download --force-ipv4 --sub-format vtt --write-auto-subs --write-subs --sub-langs <X> --output …` |
| yt-dlp JS runtime | none found (`WARNING: No supported JavaScript runtime`) |
| Time zone | local = UTC+7; `errors.log` is UTC |
| Requests made | ~15 plain-`en` fetches from one IP during investigation before the sample run, then 36 (18 videos × 2) paced 15 s apart, then 17 `-J` runs paced 6 s. **A temporary per-IP throttle caused by this volume cannot be excluded.** |

## The sample and its bias

18 videos, 17 usable (`osLrm8nve_A` is "Video unavailable"; kept in `manifest.json`, excluded from analysis).

- **15 IDs came from the app's `errors.log`, which records only failures.** They are biased toward videos that
  already failed once, so **the rate of 429s here says nothing about how common the problem is across YouTube.**
- 3 controls were added: `qp0HIF3SfI4`, `iG9CE55wbtY`, `UF8uR6Z6KLc` (talks, uploaded subtitles). All three
  returned a transcript on plain `en`. Three is too few to estimate a prevalence.
- `errors.log` entry counts per video ID at the time (not broken down by category, so a high count is not proof
  of 429s): `5oer61Xyi4c` 21, `X0UI0O8YzJM` 16, `kjoQPn--F7A` 14, `r8CppXSqVDU` 11, `BqRhBq-_kgE` 9,
  `I8XaYkRW1tA` 5, `vyIgAO8aCbA` 4, `qN6OM1IzjIE` 3, `vsGwx28z4jk` 2, `LXb3EKWsInQ` 2, and one each for
  `QcdpeFbuy8o`, `osLrm8nve_A`, `KcVkq5L-0f0`, `dQw4w9WgXcQ`, `9EUTRL_4Cj8`.
- Fetch order alternated per video (even index: `en` first, odd: `en-orig` first) to avoid an order bias.
  Each video was fetched **once** per language; there are no repeat runs in the sample.

## Results, per video

`manual_en` = uploaded English subtitle track exists (yt-dlp `subtitles`); `en-orig listed` = key present in
`automatic_captions`; `tlang` = the auto-caption `en` URL carries `tlang=`. Fetch columns are the outcome of a
real `--sub-langs` run. `similarity` = word-level LCS ratio of the two parsed transcripts, only when both
downloaded (`similarity-results.txt`).

| video | origin | manual_en | en-orig listed | tlang | plain `en` | `en-orig` | similarity (kind en / orig) |
|---|---|---|---|---|---|---|---|
| `5oer61Xyi4c` | log | no | yes | yes | **429** | ok | – |
| `X0UI0O8YzJM` | log | yes | yes | no | ok | ok | 0.954 (uploaded / auto) |
| `kjoQPn--F7A` | log | yes | yes | no | ok | ok | 0.929 (uploaded / auto) |
| `r8CppXSqVDU` (Vietnamese) | log | no | no (`vi-orig`, `en-US-orig`) | yes | **429** | none | – |
| `BqRhBq-_kgE` | log | no | yes | yes | **429** | ok | – |
| `I8XaYkRW1tA` | log | yes | no | no | ok | none | – |
| `vyIgAO8aCbA` | log | no | yes | yes | **429** | ok | – |
| `qN6OM1IzjIE` | log | no | yes | yes | **429** | ok | – |
| `vsGwx28z4jk` | log | no | yes | yes | **429** | ok | – |
| `LXb3EKWsInQ` | log | no | no | – (no captions) | none | none | – |
| `QcdpeFbuy8o` | log | no | yes | yes | **429** | ok | – |
| `KcVkq5L-0f0` | log | no | yes | yes | **429** | ok | – |
| `dQw4w9WgXcQ` | log | yes | yes | no | ok | ok | 0.442 (uploaded / auto) |
| `9EUTRL_4Cj8` | log | no | no | yes | **429** | none | – |
| `qp0HIF3SfI4` | control | yes | no | no | ok | none | – |
| `iG9CE55wbtY` | control | yes | yes | no | ok | ok | 0.754 (uploaded / auto) |
| `UF8uR6Z6KLc` | control | yes | yes | no | ok | ok | 1.000 (auto / auto) |

Reading the table:

- **H1 (tlang predicts the 429): 0 mismatches / 17** (`analyze-results.txt`). Caveat: for videos that *have* an
  uploaded English track yt-dlp requests the uploaded track for `en`, so the auto-caption `en` URL is not what it
  downloads; "tlang: no" on those rows is therefore weak evidence for H1, and the informative rows are the
  auto-only ones (all `tlang: yes` → 429, apart from the caption-less video).
- **H2 (rule: uploaded → plain `en`; else `en-orig` if listed; else plain `en`) chose a request that returned a
  transcript on 14 / 17.** The 3 that did not (`r8CppXSqVDU`, `9EUTRL_4Cj8`, `LXb3EKWsInQ`) have no English
  original or no captions; the rule falls back to today's behaviour there, so it is no worse than today.
- **Uploaded vs auto text really differs.** Where an uploaded track exists and `en-orig` also downloads, the texts
  are 0.93 / 0.95 / 0.75 / 0.44 similar (0.44 is `dQw4w9WgXcQ`: lyrics vs speech recognition). Serving `en-orig`
  there would silently downgrade the transcript and mislabel `caption kind` in `get_video_brief`.
- **When plain `en` succeeded on an auto track (`UF8uR6Z6KLc`, n = 1) it was identical to `en-orig`.**
- **Every 429 in the sample was on an auto-caption-only video; every video with an uploaded English track
  succeeded on plain `en`.** With 15/18 IDs chosen from failures this is an observation about the sample, not an
  established rule. **Correction (2026-09-27):** the `manual_en` column above uses a loose `en` / `en-*` test.
  For `UF8uR6Z6KLc` the uploaded key is `en-eEY6OEpapPo`, not `en`, and plain `en` returned its **auto** track
  (hence "auto / auto", similarity 1.000). So plain `en` succeeded on all 7 videos with an uploaded English track,
  but it is confirmed to have returned the *uploaded* track on 4 (`X0UI0O8YzJM`, `kjoQPn--F7A`, `dQw4w9WgXcQ`,
  `iG9CE55wbtY`); on `I8XaYkRW1tA` and `qp0HIF3SfI4` the kind was not scored, and on `UF8uR6Z6KLc` it was auto.
- **Watch page vs yt-dlp agree on "uploaded English track exists": 17 / 17** (`pagecheck-results.txt`,
  non-`asr` track with `languageCode` `en`/`en-*`). So the guard needs no extra yt-dlp run.

## Mechanism observed on `vyIgAO8aCbA` (investigated first, 2026-09-26)

- App `errors.log`, yt-dlp `2026.08.19`: failing timedtext URL `…&lang=uk&tlang=en&variant=timing-optimized&fmt=vtt`
  → `HTTP Error 429: Too Many Requests`, category `rate_limited`, 5–11 s per failed attempt.
- Later `-J` with `2026.07.04`: the same request maps to `lang=ar&tlang=en` (first of 21 `-orig` entries:
  `ar-orig, bn-orig, nl-NL-orig, en-orig, fr-FR-orig, de-DE-orig, iw-orig, hi-orig, id-orig, it-orig, ja-orig,
  ko-orig, ml-orig, pl-orig, pt-BR-orig, pa-orig, ru-orig, es-US-orig, ta-orig, te-orig, uk-orig`). So the
  translation *source* language is not stable across yt-dlp versions or time. Root cause of the mapping: **unknown**.
- Watch page (`ytInitialPlayerResponse`): 21 `captionTracks`, all `kind:"asr"`; the `a.en` entry carries
  `variant=gemini&exp=xpe`, the others `variant=timing-optimized`; `audioTracks` include dubbed ids such as
  `it.10`, `id.10`, each with `defaultCaptionTrackIndex: 1`. The video is English speech (Buffett/Munger), and the
  Ukrainian track (the only non-English one whose text was read) is a machine translation of it; the Arabic
  track was never read. `defaultCaptionTrackIndex` pointed at an
  unrelated track and was not usable as an "original language" marker.
- `yt-dlp 2026.03.17 --list-subs` (older binary) showed only `en, ru, uk` plus `en-orig, ru-orig, uk-orig` for the
  same video, versus 21 `-orig` entries under `2026.07.04`: the track list is **version-dependent**.
- Direct runs, `2026.07.04`: `--sub-langs en` → 429 four times in a row (three spaced ~2.5 min), and again at
  23:29 local on this video and on `BqRhBq-_kgE`; `--sub-langs en-orig` → 88 KB of genuine English;
  `--sub-langs uk` → succeeds but returns a **Ukrainian translation of English speech**, so the BUG-011 message's
  advice ("set the language to the spoken language") is wrong for this video.
- Double-failure shape (`r8CppXSqVDU`, `--sub-langs en-orig`): exit code 0, no file written, stderr
  `[info] There are no subtitles for the requested languages`.
- BUG-011 (2026-09-20) recorded plain `en` as **working** on `BqRhBq-_kgE`, `5oer61Xyi4c` and `dQw4w9WgXcQ`.
  Here it 429s on the first two. Either YouTube/yt-dlp changed in between, or a throttle applies now; not resolved.

## Silent back-translation (not fixed by the shipped retry)

On `vyIgAO8aCbA`, two CLI runs built from the BUG-012 branch (2026-09-26 ≈ 16:27–16:29 UTC) did **not** hit a
429 (no `transcript_fetch` line was logged) and returned a different text from the genuine track:

- CLI, plain `en`: "Well, let me ask a **clarifying** question about this. It's from **Jack Saling**, who asks,
  "What's your approach when you see so many of these companies that have skyrocketed? Not GME stocks or
  **“mem” stocks**…"
- `en-orig` (genuine): "Well, let me ask a **follow-up** question on that then. This comes from **Jack saying**
  who says what's your mindset when you see so many of these high flyers? Not the GME or **meme** stocks…"

The first reads as a uk→en back-translation. A third CLI run (2026-09-26 16:29 UTC) got the 429, retried with
`en-orig`, and returned the genuine text. In the same minute a direct `yt-dlp 2026.07.04 --sub-langs en` still
returned 429, so the CLI and a bare yt-dlp disagreed (flag differences: the app adds `--ffmpeg-location` and
`--verbose`; cause unknown). In the 17-video sample no such silent success on a `tlang` URL occurred (0 of 9), so
the frequency of this failure mode is **unmeasured**: n = 2 runs on one video.

## Findings added 2026-09-27 (while reviewing the draft of task 20)

Run after the main measurement, on the same 17 videos, with the saved scripts (`pagecodes.py` →
`pagecodes-results.txt`); plus two one-off live checks.

- **Page track codes ⇔ yt-dlp `-orig` keys.** "The watch page lists an auto-generated (`asr`) track whose
  `languageCode` is exactly `en`" matched "yt-dlp lists `en-orig`" on **17 / 17** videos, 0 mismatches. The 7
  many-track videos all have `asr` `en`; `r8CppXSqVDU` has `en-US` + `vi` (no exact `en`, and yt-dlp lists
  `en-US-orig`, `vi-orig`, not `en-orig`); `9EUTRL_4Cj8` has one non-English `asr` track. This is the signal
  that separates "`-orig` will exist" from "it will not", using data the app already scrapes.
- **yt-dlp's own `language` field** (in `j.json`, per video): `vi` for `r8CppXSqVDU` and `9EUTRL_4Cj8`, `en-US`
  for the 7 many-track videos, `en` for 5 others, `None` for the 3 with no auto-caption tracks
  (`I8XaYkRW1tA`, `LXb3EKWsInQ`, `qp0HIF3SfI4`). It matches every video's known language (English speech
  confirmed by reading `vyIgAO8aCbA`; Vietnamese for `r8CppXSqVDU` from BUG-011; the other labels are
  yt-dlp's word, not independently checked). Where yt-dlp obtains it, and whether the watch page carries an
  equivalent, is **unverified**. It is the candidate signal for BUG-013.
- **The timedtext URL is printed on a *successful* fetch** with yt-dlp 2026.07.04 and `--verbose`
  (`[debug] Invoking http downloader on "https://www.youtube.com/api/timedtext?…"`, checked on
  `UF8uR6Z6KLc`), so a `tlang=` in it could be logged on success. For the app's 2026.08.19 the line was only seen
  on failure (that binary is gone); **unverified on success**.
- **BUG-013 observed live.** The branch-built CLI (tasks 18 + 19 included), `youtube-cli transcript
  r8CppXSqVDU` with no language, resolves `en` and fails with the BUG-011 429; page tracks are
  `[('en-US','asr'), ('vi','asr')]`. `--language vi` still returns the Vietnamese transcript. Logged as
  `docs/BUGS.md` BUG-013; it is a different mechanism from BUG-012 (wrong language chosen, not a bad request for
  the right one).

## What was NOT measured

- Prevalence across YouTube (the sample is failure-biased) and stability over time (single pass, one day).
- Any yt-dlp version other than `2026.07.04` (and the one `--list-subs` on `2026.03.17`); none on `2026.08.19`.
- Whether `en-orig` is a genuine original for a video **not spoken in English** that has the 17–21-track
  structure: it might then be a machine translation. `r8CppXSqVDU` (Vietnamese) has no `en-orig`, so it says nothing.
- Non-English requested languages (`vi`, `uk`, …); only `en` was swept.
- Repeat runs per video, other networks/IPs, cookies/logged-in state.
- Why yt-dlp picks the translation source it picks; why a bare yt-dlp 429s where the CLI did not.

## Files

| file | what it is |
|---|---|
| `measure.py` | Fetches `en` and `en-orig` for each video (alternating order, 15 s pacing) and writes `manifest.json` + `vtt/`. |
| `manifest.json` | Outcome per video and language (`ok` / `429` / `none` / `other`). `path` fields reduced to file names (they were absolute local paths). The `.vtt` files themselves are not kept. |
| `jcheck.py` | `yt-dlp -J --skip-download` per video → `j.json`: uploaded-track keys, `-orig` keys, `lang/tlang/kind/variant` of the `en`, `en-orig` and uploaded-`en` URLs. Reads `manifest.json`. |
| `j.json` | Its output (only URL *parameters* are stored, never signed URLs). |
| `analyze.py` → `analyze-results.txt` | Joins the two and tests H1 (tlang ⇔ 429) and H2 (the rule). |
| `pagecheck.py` → `pagecheck-results.txt` | Compares the watch page's uploaded-track marking with yt-dlp's (plain GETs, no yt-dlp). |
| `pagecodes.py` → `pagecodes-results.txt` | Per-track page codes (`asr` / uploaded) joined with yt-dlp's `-orig` keys, `language` field and uploaded-`en` keys; tests "exact `asr` `en` on the page ⇔ yt-dlp lists `en-orig`" (0 mismatches / 17). Plain GETs, no yt-dlp. |
| `score_pairs_test.go.txt` → `similarity-results.txt` | The throwaway Go scorer (kept as `.txt`, not compiled) and its output: word-level similarity using the app's own `parseVtt`. |

## Re-running

```sh
cd docs/evidence/bug-012
python measure.py        # ~10 min, 36 yt-dlp runs; writes manifest.json and vtt/
python jcheck.py         # ~3 min, 17 yt-dlp -J runs; writes j.json
python analyze.py
python pagecheck.py
python pagecodes.py
```

`measure.py` and `jcheck.py` expect `%LOCALAPPDATA%\go-ytdlp\yt-dlp-2026.07.04.exe` (edit `YTDLP` to change the
version). Re-running will overwrite `manifest.json` and `j.json` and will very likely return different numbers if
YouTube's behaviour has moved; treat the files in this folder as a dated snapshot.

## Related

`docs/BUGS.md` BUG-012 (decision, shipped retry, known limitation) and BUG-013 (wrong resolved language, found
while reviewing task 20); PR #36 (the retry and the `transcript_fetch_orig_retry` log line);
`docs/tasks/20-guarded-orig-first/TASK.md` (draft, not approved).
