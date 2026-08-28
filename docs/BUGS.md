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

- **Status:** fixed (branch `fix/bug-001-002-subtitle-and-ffmpeg`, pending merge to `main`)
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
Fix now (human decision, this session). Implemented as:
1. `SubLangs(language)` — exact match, no `.*` wildcard (`internal/core/transcript.go`, `fetchSegments`). Verified directly against `yt-dlp` outside our code that this pulls only the plain-language track and avoids the multi-variant request burst.
2. `pickVttFile(files, language)` (`internal/core/transcript.go`) — new pure function, prefers a file ending exactly in `.<language>.vtt`, falls back to the first `.vtt` found only if no exact match exists. Unit-tested (`internal/core/transcript_test.go`, `TestPickVttFile`, 5 cases) — ground truth is the fix's own specification since there's no upstream equivalent to derive from.
3. Verified end-to-end against `dQw4w9WgXcQ`: `transcript --timestamps` and `search` both now return the correct English transcript with no 429, where they previously failed reliably.

---

## BUG-002: `EnsureYtDlp` silently swallows ffmpeg auto-install failure, breaking downloads

- **Status:** fixed (branch `fix/bug-001-002-subtitle-and-ffmpeg`, pending merge to `main`)
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

### Root cause, precisely (confirmed before fixing)
Read `go-ytdlp` v1.3.5's own source (`install.go`): `downloadTimeout = 30 * time.Second` is a single hardcoded `http.Client{Timeout: ...}` used by the *same* `downloadFile` function for both the yt-dlp binary (small, succeeds) and the ffmpeg archive (large, times out) — no size-aware adjustment. Measured directly: the archive is **170MB**; a raw `curl` in this environment pulled it in ~43s at ~4MB/s — a sustained throughput limit, not a transient blip, so simple retries alone cannot fix it (confirmed: 3 retries all failed identically before the deeper fix below).

### Decision
Fix now (human decision, this session), plus "kindly suggest the user pre-install ffmpeg for convenience" (explicit follow-up instruction). Implemented as:
1. **Bounded retry + visible failure** (`internal/core/ytdlp.go`): `installFFmpegWithRetry` (3 attempts, injectable install function for testability — `internal/core/ytdlp_test.go`, 3 cases covering transient-then-success, first-try-success, and exhausted-retries). Failures are now logged to stderr and `LogDownloadError` instead of silently swallowed.
2. **Cache pre-warm workaround** (`internal/core/ffmpeg_prewarm.go`): before falling back to go-ytdlp's own (timeout-limited) downloader, we download the ffmpeg archive ourselves with a 5-minute timeout and extract just the `ffmpeg`/`ffmpeg.exe` binary directly into go-ytdlp's own expected cache path (`ytdlp.GetCacheDir()`). `go-ytdlp`'s `InstallFFmpeg` checks that exact location *before* attempting any network call (confirmed by reading `resolveExecutable` in its source), so once pre-warmed, it finds our file and never hits its own 30s wall. Implemented for `windows/amd64` (zip) and `linux/amd64` (tar.xz) — the platforms verifiable/most likely to matter; other platforms fall through to the unchanged retry path. Unit-tested: `ffmpegPrewarmConfig` (platform→URL/archive-type table, 5 cases), `entryMatchesBinary` (archive-entry-name matching, 6 cases), `extractFromZip` (real in-memory zip fixture, round-tripped through actual `archive/zip`, 2 cases) — all pure/offline. The network download + tar.xz path are I/O and not unit-tested, consistent with project convention, but the zip path (this environment's platform) was verified for real end-to-end below.
3. **`requireFFmpeg(ffmpegPath)`** (`internal/core/ytdlp.go`, unit-tested): audio downloads (`StartAudioDownload`/`DownloadAudioBlocking`) now fail fast with a clear, actionable message when ffmpeg truly isn't available, instead of a confusing deep yt-dlp postprocessing error. Video downloads print a clear upfront warning (not a hard failure) when ffmpeg is unavailable, since merging is graceful-degradable (matches upstream's "ffmpeg optional" intent) but the degraded output should no longer be a surprise.
4. **Friendly manual-install suggestion**, per explicit request: printed once, before the download starts (`"ffmpeg not found locally; downloading now (one-time, ~170MB, can take a while on slower connections). For faster/more reliable startup, consider installing ffmpeg yourself and ensuring it's on PATH."`), and repeated in both the final failure warning and `requireFFmpeg`'s error message.
5. **Verified end-to-end** against `dQw4w9WgXcQ` after clearing the ffmpeg cache: `download --audio --format mp3` now actually runs `[ExtractAudio]` and produces a real, correctly-sized `.mp3` (previously: hard failure); `download --quality sd360` now runs `[Merger] Merging formats into "...mp4"` and produces one playable `.mp4` (previously: silently left two unmerged files on disk).

---

## BUG-003: `TranscriptErrorText`'s network-error branch is dead code in Go (ported Node.js error strings)

- **Status:** open
- **Discovered:** critical code review of the MCP server surface (2026-08-28, the evaluation session that scoped Phase 2), by reading `internal/core/transcript.go` — not triggered by a runtime failure.
- **Inherited from upstream:** yes, in the sense that the substrings were ported verbatim from the TS implementation, where they are *correct* — `ENOTFOUND`/`ECONNREFUSED` are Node.js `libuv` error codes that really do appear in Node error messages. The port carried the strings across a runtime boundary where they can never occur.

### Symptom

Network failures during transcript fetch (DNS resolution failure, connection refused) are reported to the user/agent via the generic branch — `"Failed to fetch transcript for video <id>: <raw error>"` — instead of the intended friendly message `"Network error while fetching transcript for video <id>. Please check your internet connection."` No crash, no wrong data; the classification is just unreachable.

### Root cause

`TranscriptErrorText` (`internal/core/transcript.go:144`) classifies by substring match on `"ENOTFOUND"` and `"ECONNREFUSED"`. Go's `net` errors render as e.g. `"dial tcp: lookup example.com: no such host"` or `"connect: connection refused"` — they never contain the Node.js error-code strings, so that `case` can never be true in this codebase. (In practice the error text reaching this function usually comes from yt-dlp's stderr anyway, which has its own phrasing — any fix should classify against *actually observed* Go/yt-dlp error text, derived per the TDD ground-truth rule, not hand-reasoned substrings.)

### Options

- **Fix:** replace/augment the substrings with ones derived from real observed failures (e.g. capture yt-dlp stderr and Go `net` error text under a disconnected/blocked network, then match on those; or match Go error types upstream of the string conversion where possible). Small, self-contained; natural to fold into Phase 2 only if a task already touches this function, otherwise as its own out-of-band fix.
- **Wontfix (faithful parity):** the generic branch's message still contains the raw error, so the information isn't lost — only the friendly classification is. Leave as a documented quirk.

### Decision

Pending — awaiting human decision, per the bug-tracking process above.
