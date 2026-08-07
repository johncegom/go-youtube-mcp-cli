# Out-of-band: fix BUG-001 and BUG-002

**Status:** fixed, merged into `main` (branch `fix/bug-001-002-subtitle-and-ffmpeg`)

Manual smoke-testing `cmd/youtube-cli` against a real video (`dQw4w9WgXcQ`)
after task 6 surfaced two real bugs, tracked and fully written up in
`docs/BUGS.md`:

- **BUG-001**: transcript/search could 429 or silently return a
  mistranslated caption track, due to an overly broad `--sub-langs` wildcard
  inherited verbatim from the upstream TS project, compounded by naive
  "first .vtt file" selection.
- **BUG-002**: ffmpeg auto-install failures were silently swallowed, and
  `go-ytdlp`'s internal 30s download timeout is provably too short for the
  ~170MB ffmpeg archive on this environment's network (~43s at measured
  throughput) — not transient flakiness, a deterministic mismatch.

Both were root-caused (see `docs/BUGS.md` for full detail incl. exact
`go-ytdlp` source references) and fixed with TDD on a **separate branch**,
per explicit instruction: the repo was git-initialized and an initial commit
made on `main` capturing the state through task 6, then
`fix/bug-001-002-subtitle-and-ffmpeg` was branched off for the fix work.
The fix branch was build/vet/test clean (41 tests passing) and both fixes
were verified end-to-end against the real video before being merged into
`main` (merge commit, `--no-ff`, per explicit instruction after the human
confirmed the branch passed).

New files added on the fix branch: `internal/core/ffmpeg_prewarm.go` (+
test), `internal/core/ytdlp_test.go`. Modified: `internal/core/transcript.go`
(+ test), `internal/core/ytdlp.go`, `internal/core/download.go`.

## Fix details

### BUG-001
1. `SubLangs(language)` — exact match, no `.*` wildcard
   (`internal/core/transcript.go`, `fetchSegments`). Verified directly
   against `yt-dlp` outside our code that this pulls only the plain-language
   track and avoids the multi-variant request burst.
2. `pickVttFile(files, language)` (`internal/core/transcript.go`) — new pure
   function, prefers a file ending exactly in `.<language>.vtt`, falls back
   to the first `.vtt` found only if no exact match exists. Unit-tested
   (`internal/core/transcript_test.go`, `TestPickVttFile`, 5 cases) — ground
   truth is the fix's own specification since there's no upstream
   equivalent to derive from.
3. Verified end-to-end against `dQw4w9WgXcQ`: `transcript --timestamps` and
   `search` both now return the correct English transcript with no 429,
   where they previously failed reliably.

### BUG-002
Root cause confirmed precisely by reading `go-ytdlp` v1.3.5's own source
(`install.go`): `downloadTimeout = 30 * time.Second` is a single hardcoded
`http.Client{Timeout: ...}` used by the same `downloadFile` function for
both the yt-dlp binary (small, succeeds) and the ffmpeg archive (large,
times out) — no size-aware adjustment. Measured directly: the archive is
170MB; a raw `curl` in this environment pulled it in ~43s at ~4MB/s — a
sustained throughput limit, not a transient blip, so simple retries alone
cannot fix it.

1. **Bounded retry + visible failure** (`internal/core/ytdlp.go`):
   `installFFmpegWithRetry` (3 attempts, injectable install function for
   testability — `internal/core/ytdlp_test.go`, 3 cases covering
   transient-then-success, first-try-success, and exhausted-retries).
   Failures are now logged to stderr and `LogDownloadError` instead of
   silently swallowed.
2. **Cache pre-warm workaround** (`internal/core/ffmpeg_prewarm.go`): before
   falling back to go-ytdlp's own (timeout-limited) downloader, download the
   ffmpeg archive ourselves with a 5-minute timeout and extract just the
   `ffmpeg`/`ffmpeg.exe` binary directly into go-ytdlp's own expected cache
   path (`ytdlp.GetCacheDir()`). `go-ytdlp`'s `InstallFFmpeg` checks that
   exact location *before* attempting any network call (confirmed by
   reading `resolveExecutable` in its source), so once pre-warmed, it finds
   our file and never hits its own 30s wall. Implemented for `windows/amd64`
   (zip) and `linux/amd64` (tar.xz) — the platforms verifiable/most likely
   to matter; other platforms fall through to the unchanged retry path.
   Unit-tested: `ffmpegPrewarmConfig` (platform→URL/archive-type table, 5
   cases), `entryMatchesBinary` (archive-entry-name matching, 6 cases),
   `extractFromZip` (real in-memory zip fixture, round-tripped through
   actual `archive/zip`, 2 cases) — all pure/offline. The network download +
   tar.xz path are I/O and not unit-tested, consistent with project
   convention, but the zip path (this environment's platform) was verified
   for real end-to-end below.
3. **`requireFFmpeg(ffmpegPath)`** (`internal/core/ytdlp.go`, unit-tested):
   audio downloads (`StartAudioDownload`/`DownloadAudioBlocking`) now fail
   fast with a clear, actionable message when ffmpeg truly isn't available,
   instead of a confusing deep yt-dlp postprocessing error. Video downloads
   print a clear upfront warning (not a hard failure) when ffmpeg is
   unavailable, since merging is graceful-degradable (matches upstream's
   "ffmpeg optional" intent) but the degraded output should no longer be a
   surprise.
4. **Friendly manual-install suggestion**, per explicit request: printed
   once, before the download starts (`"ffmpeg not found locally;
   downloading now (one-time, ~170MB, can take a while on slower
   connections). For faster/more reliable startup, consider installing
   ffmpeg yourself and ensuring it's on PATH."`), and repeated in both the
   final failure warning and `requireFFmpeg`'s error message.
5. **Verified end-to-end** against `dQw4w9WgXcQ` after clearing the ffmpeg
   cache: `download --audio --format mp3` now actually runs `[ExtractAudio]`
   and produces a real, correctly-sized `.mp3` (previously: hard failure);
   `download --quality sd360` now runs `[Merger] Merging formats into
   "...mp4"` and produces one playable `.mp4` (previously: silently left two
   unmerged files on disk).

See `docs/BUGS.md` BUG-001 and BUG-002 for the full original symptom/root
cause writeups (this file is process/task-tracking history; `docs/BUGS.md`
is the canonical bug record).
