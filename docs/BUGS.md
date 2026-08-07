# Bug Tracker

Bugs found during development are tracked here — like tasks in `docs/LEDGER.md`,
a bug needs to be tracked, planned, and resolved deliberately, not fixed
silently in passing. When a bug is found:

1. Add an entry below with the next sequential ID, status `open`.
2. Fill in **Symptom** and **Root cause** as precisely as possible. If the
   root cause isn't known yet, say so explicitly (`Root cause: unknown —
   needs investigation`) rather than guessing.
3. If there's an obvious fix, propose it under **Options**, but do not apply
   it without the human deciding — the same pause-and-ask discipline as the
   ledger's numbered tasks applies to bugs.
4. Once a decision is made, record it under **Decision**, update **Status**,
   and (if fixed) note which commit/session did it and cross-reference the
   ledger task it was fixed under, if any.

**Status values:** `open` (found, undecided) · `tracked` (decision made, not
yet fixed) · `fixed` · `wontfix` (decided to leave as-is, e.g. faithful-parity
tradeoff) · `not-a-bug` (investigated, turned out to be expected behavior).

---

## BUG-001: Transcript fetch can 429 or silently return a translated/wrong-language transcript

- **Status:** open
- **Discovered:** manual CLI smoke test of `internal/core/transcript.go`'s `fetchSegments` (task 6 verification), against `dQw4w9WgXcQ`
- **Inherited from upstream:** yes — reproduced independently by running `yt-dlp` directly with the same flags outside any of our Go code, so this is a latent bug in the original TS project (`packages/core/src/index.ts`), not something the Go port introduced. A faithful Phase-1 port currently reproduces it exactly.

### Symptom
`youtube-cli transcript <id> --timestamps` (and the underlying `fetchSegments`/`GetTranscriptText`/`GetTranscriptTimed`/`SearchInTranscript`/`SaveTranscriptFile`) can fail with:
```
error: Failed to fetch transcript for video dQw4w9WgXcQ: yt-dlp failed: ERROR: Unable to download video subtitles for 'en-de-DE': HTTP Error 429: Too Many Requests
```
and even when it *doesn't* fail outright, it can silently pick an auto-translated caption track (e.g. "English from German") instead of the actual requested-language captions.

### Root cause
Two compounding issues, both present in the original TS logic and carried over verbatim in the Go port:

1. **`--sub-langs "<language>.*"` is too broad.** YouTube now exposes multiple auto-translated caption variants per video for popular content — confirmed via `yt-dlp --list-subs` on the test video: `en`, `en-en`, `en-orig`, `en-de-DE`, `en-ja`, `en-pt-BR`, `en-es-419`. The `.*` wildcard matches all of them, so `--write-auto-sub --write-sub --sub-langs "en.*"` makes yt-dlp attempt to download every variant in one invocation. Enough sequential requests to YouTube's caption endpoint in a short window triggers `HTTP 429 Too Many Requests`, aborting the whole command (see `internal/core/transcript.go`, `fetchSegments`, the `NewYtDlpCommand()...SubLangs(language + ".*")` call — mirrors TS `--sub-langs`, `${language}.*``).
2. **No exact-language file selection.** After the subtitle download step, both the TS original and the Go port just take the *first* `.vtt` file found in the temp directory (`fs.readdirSync(tmpDir).filter(f => f.endsWith(".vtt"))[0]` in TS; `os.ReadDir` + first match in Go — see `fetchSegments`'s vtt-file-selection loop). There's no check that the picked file actually corresponds to the requested language. Because yt-dlp names files like `sub.en-de-DE.vtt` vs `sub.en.vtt`, and `-` (0x2D) sorts before `.` (0x2E) in ASCII, a translated variant can sort ahead of the plain-language file and get silently picked — even on runs where every variant downloads successfully (no 429).

### Options
- **Fix now:** narrow `SubLangs` to an exact match for the requested language (e.g. `language` alone, or a tighter pattern that excludes translated variants), and change the vtt-file-selection to prefer a file whose language segment exactly equals the requested language, falling back to the first available `.vtt` only if no exact match exists.
- **Leave as-is for Phase 1:** faithful parity with upstream; revisit in a later phase.

### Decision
Pending — awaiting human input (a `docs/BUGS.md` tracking process was requested specifically so this doesn't get decided or fixed silently; will be raised for a decision separately from task 7).

---

## BUG-002: `EnsureYtDlp` silently swallows ffmpeg auto-install failure, breaking downloads

- **Status:** open
- **Discovered:** manual CLI smoke test of `internal/core/download.go` (`DownloadVideoBlocking`/`DownloadAudioBlocking`), same session as BUG-001, against `dQw4w9WgXcQ`
- **Not inherited from upstream in the same way as BUG-001:** the *symptom* (ffmpeg missing) can happen to the TS original too if its `ytdlp-nodejs` postinstall download fails, but the TS version at least runs that download at `npm install` time where a failure is loud (non-zero exit, visible in install logs). Our Go port's design choice to install lazily and swallow the error is what turns this into a *silent* failure — see `internal/core/ytdlp.go`, `EnsureYtDlp`.

### Symptom
- `youtube-cli download <id> --audio` fails hard:
  ```
  ERROR: Postprocessing: ffprobe and ffmpeg not found. Please install or provide the path using --ffmpeg-location
  error: exit status 1
  ```
- `youtube-cli download <id>` (video, no `--audio`) does **not** error, but silently produces the *wrong output*: instead of one merged playable `.mp4`, it leaves two separate files on disk — e.g. `....f134.mp4` (video-only) and `....f140.m4a` (audio-only) — with only a yt-dlp `WARNING: ... ffmpeg is not installed. The formats won't be merged` buried in the output. `StartVideoDownload`'s predicted-path message (`"...appear at: <path>.mp4"`) would be actively wrong in this case too.

### Root cause
`EnsureYtDlp` (`internal/core/ytdlp.go`) calls `ytdlp.InstallFFmpeg(ctx, nil)` and, on error, just doesn't set `ffmpegPath` — no log, no propagated error, no user-visible signal of any kind:
```go
if resolved, err := ytdlp.InstallFFmpeg(ctx, nil); err == nil {
    ffmpegPath = resolved.Executable
}
```
In this environment, `InstallFFmpeg` actually fails (confirmed by calling it directly, outside our wrapper):
```
InstallFFmpeg error: failed to download and extract ffmpeg archive: unable to download go-ytdlp
dependent file "...\go-ytdlp\ffmpeg-master-latest-win64-gpl.zip": streaming data: context deadline
exceeded (Client.Timeout or context cancellation while reading body)
```
i.e. the ffmpeg archive (~80–100MB) timed out mid-download. That specific timeout may be environment/network flakiness, but the design bug is independent of *why* the install fails: any `InstallFFmpeg` failure for any reason (network, disk, unsupported platform — go-ytdlp's own docs note ffmpeg auto-install "is only supported on a handful of platforms") is currently invisible until a downstream `yt-dlp` postprocessing step fails confusingly, or — worse — silently produces unmerged output with only a warning line to notice.

### Options
- **Fix now:** at minimum, log the `InstallFFmpeg` error (e.g. via the same stderr+`LogDownloadError` pattern already used for download failures) so it's not silent; consider whether `StartVideoDownload`/`DownloadVideoBlocking` should surface a clear upfront error when ffmpeg isn't available and the request needs it (merging formats / audio extraction), rather than letting yt-dlp fail or degrade downstream. Also consider retrying/extending the download timeout for the ffmpeg archive specifically, since it's much larger than the yt-dlp binary itself.
- **Leave as-is for Phase 1:** faithful-enough parity (TS also depends on an external ffmpeg install succeeding); revisit in a later phase.

### Decision
Pending — awaiting human input, same as BUG-001.
