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

- **Status:** fixed (branch `fix/bug-003-transcript-error-classification`, pending merge to `main`)
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

Fix now (human decision, 2026-08-28). Investigation found two distinct
error sources reach `TranscriptErrorText`, only one of which preserves a
real Go error chain — so the fix classifies each differently rather than
using one substring match for both:

1. **`EnsureYtDlp`'s own network failure** (`internal/core/ytdlp.go`) is
   `%w`-wrapped down to `go-ytdlp`'s HTTP client error, which is
   ultimately a `*net.DNSError`/`*net.OpError` — both satisfy `net.Error`.
   `TranscriptErrorText` (`internal/core/transcript.go`) now classifies
   this case with `errors.As(err, &netErr)` against `net.Error`, by type
   rather than OS-specific text — correct identically on Windows and
   Linux CI. Verified directly: binding a loopback listener, closing it,
   then dialing the closed address produces a real `*net.OpError`
   satisfying this check (deterministic, offline-safe, no network
   dependency).
2. **yt-dlp's own process/network failure** (`fetchSegments`) only
   survives as a flattened string (`"yt-dlp failed: <stderr>"`), so no
   type chain is available there. `TranscriptErrorText` matches yt-dlp's
   own real, OS-independent wrapper phrasing. Manual end-to-end testing
   (see step 5) found that yt-dlp uses *different* wrapper phrasing
   depending on which pipeline stage hits the network failure, so two
   real phrasings are matched, both captured directly (not guessed):
   - `"Unable to download webpage"` — captured by running yt-dlp directly
     (outside this Go code) against a URL on the RFC 2606 reserved
     `.invalid` TLD, guaranteed non-resolving with no need to cut real
     internet access:
     ```
     ERROR: [generic] Unable to download webpage: HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed) (caused by TransportError("HTTPSConnection(host='nonexistent.invalid', port=443): Failed to resolve 'nonexistent.invalid' ([Errno 11001] getaddrinfo failed)"))
     ```
   - `"Unable to download API page"` — captured by running the CLI
     (`go run ./cmd/youtube-cli transcript <real-youtube-url>`) with
     `HTTPS_PROXY`/`HTTP_PROXY` pointed at an unreachable local port
     (`http://127.0.0.1:1`), a deterministic connection-refused failure
     with no DNS dependency and no network state left behind:
     ```
     ERROR: [youtube] dQw4w9WgXcQ: Unable to download API page: ('Unable to connect to proxy', NewConnectionError("HTTPSConnection(host='127.0.0.1', port=1): Failed to establish a new connection: [WinError 10061] No connection could be made because the target machine actively refused it")) (caused by ProxyError(...))
     ```
3. The old `"ENOTFOUND"`/`"ECONNREFUSED"` substrings are removed entirely
   — the ground-truth capture above confirmed neither Go's `net` package
   nor yt-dlp's own stderr ever produce them.
4. Unit-tested (`internal/core/transcript_test.go`, `TestTranscriptErrorText`)
   — the two Node-shaped hand-constructed cases replaced with a real
   dial-refused `net.Error` case, the same error wrapped like
   `EnsureYtDlp` actually wraps it (proving `errors.As` survives realistic
   wrap depth), and the two literal captured yt-dlp stderr lines above.
   `FuzzTranscriptErrorText` added (744K executions, zero failures) for
   no-panic coverage on arbitrary error text.
5. **Verified end-to-end**: `go build ./... && go vet ./... && go test ./...`
   clean, `gofmt` clean. Two real end-to-end checks against the actual
   CLI: (a) a throwaway manual test (removed before commit, same
   convention as task 7's throwaway MCP client) ran a genuine
   dial-refused `net.Error` and the `.invalid`-TLD yt-dlp stderr directly
   through `TranscriptErrorText`; (b) `go run ./cmd/youtube-cli transcript`
   against a real YouTube URL with `HTTPS_PROXY`/`HTTP_PROXY` set to an
   unreachable port reproduced the second phrasing live through the full
   CLI path and confirmed it initially fell through to the generic
   fallback — which is what prompted adding the second substring above.
   Both phrasings now correctly produce the friendly "Network error...
   check your internet connection" message.

---

## BUG-004: `pathStartsWith` does a string-prefix compare, not a path-segment compare

- **Status:** fixed
- **Discovered:** test-coverage-hardening review of `internal/core/paths.go` (2026-08-30), while writing the first unit tests that file has ever had. Not triggered by a runtime failure.
- **Inherited from upstream:** no — this is a Go-only file (`docs/DECISIONS.md` DECISION-002); no TS equivalent to compare against.

### Symptom

`ResolveOutputDir` is the allowlist gate deciding whether a caller-supplied
output directory is allowed to sit under the user's home or temp directory
before `yt-dlp` is told to write files there. Its boundary check,
`pathStartsWith` (`internal/core/paths.go:35-41`), does:

```go
func pathStartsWith(child, parent string) bool {
	normalised := filepath.Clean(parent)
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(child), strings.ToLower(normalised))
	}
	return strings.HasPrefix(child, normalised)
}
```

This is a **string** prefix check, not a **path-segment** check. If the
allowed root is `/home/user`, a resolved directory of `/home/userXYZ/evil`
(no path separator between `user` and `XYZ`) also satisfies
`strings.HasPrefix`, and so is treated as inside the allowed root and passes
`ResolveOutputDir`, even though it's a sibling directory outside it.

### Root cause

`strings.HasPrefix` compares raw characters; it has no concept of a path
separator boundary. The function needs to check that `child` equals
`parent` or that `child` starts with `parent + string(filepath.Separator)`
(with the usual case-insensitive treatment kept for Windows).

### Options

- **Fix now:** change `pathStartsWith` to require an exact match or a
  separator-bounded prefix, e.g. `child == normalised ||
  strings.HasPrefix(child, normalised+string(filepath.Separator))` (case-folded
  on Windows as today). Low risk, small, isolated change; the new
  `paths_test.go` test added under the test-coverage-hardening task already
  demonstrates the gap and can be flipped to assert the fixed behavior.
- **Leave as-is:** in practice, `allowedOutputRoots` is `os.TempDir()` and
  the user's home directory — both fairly deep, "sibling directory with a
  same-prefixed name" is a narrow attack surface on a single-user local tool
  (this isn't a multi-tenant server; see DECISION-011 on why remote/
  multi-client MCP access is deliberately not shipped yet). Still worth
  fixing since the cost is trivial.

### Decision

Fix now (human decision, this session). Implemented as:
1. `pathStartsWith` (`internal/core/paths.go`) now requires an exact match
   or a separator-bounded prefix (`c == p || strings.HasPrefix(c, p +
   string(filepath.Separator))`), with the existing case-folding on Windows
   kept as-is.
2. `internal/core/paths_test.go`'s `TestPathStartsWith_SegmentBoundaryBug`
   (which pinned the buggy behavior) was flipped to
   `TestPathStartsWith_SegmentBoundary`, now asserting the sibling directory
   is correctly rejected while a true subdirectory is still accepted.
3. Verified: `go build ./...`, `go vet ./...`, `go test ./...`, and `gofmt`
   all clean.

---

## BUG-005: `transcriptCache.set` panics on a negative cap

- **Status:** fixed
- **Discovered:** test-coverage-hardening review of `internal/core/transcache.go` (2026-08-30), while adding cap-boundary unit tests. Not triggered by a runtime failure — `cap` is currently only ever `32` via the single process-wide `defaultCache` (`internal/core/transcache.go:46`), so this isn't reachable through any current code path.
- **Inherited from upstream:** no — this cache is new in task 11, no TS equivalent.

### Symptom

`set`'s eviction loop is:

```go
for len(c.entries) > c.cap {
	oldest := c.order[0]
	...
}
```

If `c.cap` is negative, this condition (`len(c.entries) > c.cap`) is true even
when `c.entries` is empty (`0 > -1`), so `c.order[0]` indexes an empty slice
and panics with an index-out-of-range runtime error.

### Root cause

The loop condition assumes `cap >= 0`; there's no validation on the value
passed to `newTranscriptCache`.

### Options

- **Fix now:** clamp `cap` to a minimum of 0 in `newTranscriptCache` (or guard the loop with `len(c.order) > 0`), and decide what a `cap <= 0` cache should mean in practice (most likely: "never actually caches anything," matching the already-tested `cap == 0` behavior).
- **Leave as-is:** `cap` is never externally configurable today (`defaultCache` is a fixed literal), so there is no live attack surface. Worth fixing opportunistically since it's a one-line guard, but not urgent.

### Decision

Fix now (human decision, this session). Implemented as:
1. `newTranscriptCache` (`internal/core/transcache.go`) now clamps a
   negative `cap` to `0` before storing it, so `set`'s eviction loop
   (`len(c.entries) > c.cap`) never runs against an empty `c.order` slice.
   A negative cap now behaves identically to an explicit cap of `0`
   ("never actually caches anything"), matching the already-tested
   `cap == 0` behavior — no change to `set` itself was needed.
2. `internal/core/transcache_test.go`'s `TestTranscriptCache_NegativeCapPanics`
   (which pinned the panic) was flipped to
   `TestTranscriptCache_NegativeCapClampedToZero`, asserting a negative cap
   no longer panics and behaves like `cap == 0`.
3. Verified: `go build ./...`, `go vet ./...`, `go test ./...`, and `gofmt`
   all clean.

---

## BUG-006: pinned `go-ytdlp` yt-dlp version (2026.03.17) can no longer download `dQw4w9WgXcQ` — every real download fails with `HTTP Error 403: Forbidden`

- **Status:** fixed
- **Discovered:** task 12 manual smoke test (2026-08-30), running the real MCP server (`bin/youtube-mcp.exe`) end-to-end against `download_audio`/`download_video` for the first time since `EnsureYtDlp`'s `go-ytdlp` dependency was pinned.
- **Reachability: yes** — this is the real call path (`download_audio`/`download_video` MCP tools → `core.StartAudioDownload`/`StartVideoDownload` → `NewYtDlpCommand().Run(...)`), not a test-only branch. Every real download attempt in this environment today hits it.
- **Inherited from upstream:** no — this is a Go-port-specific dependency-pinning consequence; the TS original resolves whatever `yt-dlp` version is installed on the system (or via its own npm-postinstall download) rather than a version baked into a Go library at build time.

### Symptom

`download_audio`/`download_video` (and the underlying blocking CLI commands)
fail immediately with:
```
ERROR: unable to download video data: HTTP Error 403: Forbidden
```
for a well-known, definitely-still-public video (`dQw4w9WgXcQ`), reproducible both through the MCP server and by invoking the resolved `yt-dlp` binary directly with the same flags outside any of our Go code.

### Root cause

`github.com/lrstanley/go-ytdlp@v1.3.5` (this project's pinned dependency
version, `go.mod`) hardcodes the yt-dlp release it installs/verifies:
`constants.gen.go: Version = "2026.03.17"`. `EnsureYtDlp` calls
`ytdlp.Install(ctx, nil)` unconditionally on every process start, which
re-verifies (and, if mismatched, re-downloads) exactly that pinned version
— confirmed experimentally this session: manually running `yt-dlp -U` to
upgrade the cached binary in place to `2026.08.19` fixed the 403 when
invoked directly, but the *next* `EnsureYtDlp` call (a fresh
`youtube-mcp.exe` process) silently reverted the binary back to
`2026.03.17`, and the 403 came back. yt-dlp `2026.03.17` is now more than
90 days old (yt-dlp itself warns about this) and can no longer solve
YouTube's current player-signature/format-serving challenge for at least
some formats — confirmed the failure persists even with
`--js-runtimes node` explicitly enabled, so it isn't just the "no JS
runtime" warning yt-dlp also prints.

### Options

- **Fix now:** bump the `github.com/lrstanley/go-ytdlp` dependency to a
  version that pins a current yt-dlp release (`go get -u
  github.com/lrstanley/go-ytdlp && go mod tidy`), verify against a real
  download end-to-end, and treat future staleness as an ordinary dependency
  update rather than a code bug — same category of risk as BUG-002.
- **Workaround only, no fix:** document that operators experiencing 403s
  should replace the cached binary and additionally patch/fork
  `go-ytdlp`'s pinned version locally (fragile — `EnsureYtDlp` reverts it
  on every fresh process per the root cause above).
- **Leave as-is:** accept that downloads may intermittently/eventually stop
  working until the dependency is bumped in a future maintenance pass; out
  of scope for task 12 specifically, whose job is observability
  (`get_download_status`/`list_downloads`) of whatever outcome yt-dlp
  produces, not yt-dlp's own success rate.

### Decision

Fix now (human decision, this session). Implemented as:
1. Bumped `github.com/lrstanley/go-ytdlp` v1.3.5 → v1.3.6 (`go get` +
   `go mod tidy`) — moves the pinned version forward
   (`2026.03.17` → `2026.07.04`), but retesting confirmed this alone is
   **not** a durable fix: `2026.07.04` was *already* stale (still 403;
   `yt-dlp -U` on that exact resolved binary to `2026.08.19` fixed it
   immediately). Kept the bump anyway (newer baseline is still better),
   but it doesn't address the actual mechanism.
2. `internal/core/ytdlp.go`, `EnsureYtDlp`: changed
   `ytdlp.Install(ctx, nil)` to
   `ytdlp.Install(ctx, &ytdlp.InstallOptions{AllowVersionMismatch: true})`.
   Read through `go-ytdlp`'s `install_ytdlp.go` to confirm the exact
   semantics: with `AllowVersionMismatch` unset, `Install` re-downloads
   and overwrites any already-resolved binary (cache or PATH) that
   doesn't match the pinned `Version` const — this is what was silently
   reverting a manually-`yt-dlp -U`-updated binary back to the stale
   pinned version on every process start. With it set, a resolved binary
   is accepted as-is regardless of version, so an operator's own
   `yt-dlp -U` (or a newer manual binary drop-in) now sticks across
   server restarts. A fresh install with no binary present at all still
   downloads and checksum-verifies the pinned version as before —
   unaffected.
3. Verified end-to-end via the same throwaway MCP smoke-test client used
   for task 12 (`cmd/smoketest`, rebuilt temporarily, deleted after — not
   committed): with a `2026.08.19` binary already cached (mismatching the
   pinned `2026.07.04`), `download_audio` on `dQw4w9WgXcQ` now reaches
   **`done`** via `get_download_status`, with `ActualPath` resolving to a
   real file confirmed on disk (`ls` after the run) — the one step this
   bug previously blocked. Re-confirmed the rest of task 12's smoke test
   still passes unchanged (job IDs present, `download_video` on a
   nonexistent ID still reaches `failed` with captured error text,
   unknown job ID still `isError: true`, `list_downloads` still correct).
4. Verified: `go build ./...`, `go vet ./...`, `go test ./... -race`, and
   `gofmt` all clean.

---

## BUG-007: transcript/download `yt-dlp` calls hang past the 30s timeout when the machine's IPv6 connectivity is down

- **Status:** fixed
- **Discovered:** investigating a user-reported `"Transcript fetch timed
  out for video t1xKo4spo3s. Please try again."` error (2026-08-30/31).
- **Reachability: yes** — the real call path (`get_transcript` and every
  other transcript tool → `core.GetTranscriptText`/etc. →
  `fetchSegmentsFromYtDlp`, `internal/core/transcript.go:231-273`), not a
  test-only branch. On the affected machine this reproduces
  **deterministically, for every video**, not just the reported one.
- **Inherited from upstream:** no — this is about this machine's/network's
  routing to YouTube, not project logic. But the *timeout* itself
  (`context.WithTimeout(ctx, 30*time.Second)`, `transcript.go:242`) and the
  resulting user-facing message are this project's code.

### Symptom

`get_transcript`/`get_transcript_timed`/`search_transcript`/
`download_transcript`/etc. reliably return:
```
Transcript fetch timed out for video <id>. Please try again.
```
"Please try again" is misleading here: retrying fails identically every
time on an affected machine, because the underlying cause isn't transient.

### Root cause

Confirmed directly, outside any of this project's code, and then narrowed
further after a fair question ("have we ever actually succeeded over
IPv6?") exposed that the first pass at this root cause was too narrow:

1. `yt-dlp --write-auto-sub --write-sub --sub-langs en --sub-format vtt
   <url>` hangs indefinitely at its very first step, `"Downloading
   webpage"` — tested to 90+ seconds with zero progress, for **both** the
   reported video (`t1xKo4spo3s`) and a known-good one (`dQw4w9WgXcQ`), so
   it's not video-specific.
2. A plain, unforced `curl` GET to the same watch-page URL succeeded in
   ~1.2s — but forcing that same `curl` call to use IPv6 explicitly
   (`curl -6`) reproduced the same hang/timeout (`http_code=000`, 8s
   `--max-time` exceeded). Explicitly forcing IPv4 (`curl -4`, `yt-dlp
   -4`) succeeded instantly every time, for both `yt-dlp` and `curl`.
3. **Crucially, this isn't YouTube-specific at all**: `curl -6` to
   `cloudflare.com` and `www.google.com` — two unrelated, definitely-up
   hosts — also both hung to the full 8s `--max-time` with `http_code=000`.
   So this machine's IPv6 path is down for *any* destination right now,
   not something particular to YouTube or `yt-dlp`.
4. The machine does have real global-unicast IPv6 addresses configured
   (`Get-NetIPAddress -AddressFamily IPv6`, RouterAdvertisement/DHCP-
   assigned `2405:...` prefixes, not just link-local) — so IPv6 is
   configured and was presumably reachable at some point, but is not
   actually routing out to the internet right now (an ISP/router-level
   IPv6 issue, upstream of anything this project or `yt-dlp` controls).
5. This also corrects the original framing: earlier `yt-dlp` calls
   *during this same session* (while diagnosing BUG-006) succeeded
   without ever forcing an IP version. Given point 3, there is no actual
   evidence any of those calls used IPv6 — Windows' dual-stack address
   selection can silently prefer/race IPv4 first, so those successes were
   most likely IPv4 the whole time, and IPv6 may well have been down
   throughout. "Present-but-intermittently-dead" was an overclaim; the
   honest statement is: IPv6 is confirmed down right now, and there is no
   confirmed instance of it ever working in this session.
6. This project's own 30s `context.WithTimeout` (`transcript.go:242`) is
   what turns the hang into an error at all instead of hanging the whole
   MCP/CLI call forever — a reasonable safety net on its own, but the
   resulting message ("Please try again") doesn't distinguish "transient,
   try again" from "this machine's IPv6 is down, retrying won't help until
   that's fixed or IPv4 is forced."

### Options

- **Fix now:** add `.ForceIPv4()` to `NewYtDlpCommand()`
  (`internal/core/ytdlp.go`) — confirmed available in the pinned
  `go-ytdlp` version (`Command.ForceIPv4()`, maps to `-4`). This is the
  shared builder used by every `yt-dlp` invocation in the codebase
  (transcript fetch and both download paths), so one change covers all of
  them. Forcing IPv4 is a widely-used, low-risk `yt-dlp` community
  workaround for exactly this class of hang; the main tradeoff is it
  removes IPv6 as an option entirely, including on networks where IPv6 is
  the *working* path and IPv4 is what's degraded (the inverse case) — no
  evidence either way for this project's actual user base.
- **Narrower fix:** don't force IPv4 globally; instead retry once with
  `-4` specifically after a timeout, so IPv6-healthy networks are
  unaffected. More code (a retry path + a way to thread a "retry with
  forced IPv4" flag through `fetchSegmentsFromYtDlp` and the download
  paths) for a benefit that only matters on the subset of networks where
  IPv6 is broken.
- **Message-only fix, no behavior change:** reword the timeout message to
  not imply retrying will likely help, and hint at the IPv6 possibility
  (e.g. mention `-4`/force-IPv4 as a thing to check) — cheaper, but leaves
  the actual hang (and the CLI/MCP call blocking for the full 30s each
  time) unfixed.
- **Leave as-is:** this may be specific to unusual network configurations;
  wait for more reports before changing shared command-building behavior
  for everyone.

### Decision

Fix now — force IPv4 globally (human decision, this session). Implemented
as:
1. `NewYtDlpCommand()` (`internal/core/ytdlp.go`) now chains `.ForceIPv4()`
   unconditionally onto the returned `*ytdlp.Command`, alongside the
   existing `FFmpegLocation` wiring. This is the single shared builder used
   by every `yt-dlp` invocation in the codebase (`fetchSegmentsFromYtDlp`
   in `transcript.go`, and both download paths in `download.go`), so one
   change covers all of them, matching how the fix was scoped in the
   Options above.
2. No retry/threading logic was added (the narrower "retry once with -4"
   option was considered and explicitly not chosen): forcing IPv4
   unconditionally accepts the one-sided tradeoff already noted in
   Options — it removes IPv6 as an option entirely, including on a
   hypothetical network where IPv6 is the healthy path and IPv4 is
   degraded. No such report exists for this project's user base.
3. Verified end-to-end on the affected machine (IPv6 confirmed down per
   Root cause above): `go run ./cmd/youtube-cli transcript t1xKo4spo3s
   --timestamps` (the originally user-reported failing video) now returns
   the transcript immediately instead of hanging to the 30s timeout.
   Re-checked `dQw4w9WgXcQ` (BUG-007's other repro video) — also succeeds
   immediately.
4. Verified: `go build ./...`, `go vet ./...`, `go test ./...`, and
   `gofmt` all clean. No unit test added — this is a one-line CLI-flag
   change on an I/O-bound builder with no existing unit tests for its flag
   wiring, consistent with project convention of leaving `yt-dlp`
   invocation behavior to manual/smoke verification.

---

## BUG-008: `get_transcript` reliably times out at ~30s through Claude Desktop, but succeeds instantly via the CLI for the identical video — fixed (yt-dlp format probe stalling in the Desktop-spawned environment)

- **Status:** open
- **Discovered:** user-reported (Claude Desktop MCP client) session, 2026-09-08/09, against `kjoQPn--F7A`. Investigated live via the actual per-connection MCP log (`%LOCALAPPDATA%\Claude\Logs\mcp-server-youtube-mcp.log`) and this project's own `errors.log` (`os.UserCacheDir()/youtube-mcp/errors.log`).
- **Reachability: yes** — real call path (`get_transcript`/other transcript tools → `core.GetTranscriptText`/etc. → `fetchSegmentsFromYtDlp`, `internal/core/transcript.go:244`), hit repeatedly by a real user in a real Claude Desktop session, not a test-only branch.
- **Inherited from upstream:** no — no evidence tying this to TS-original behavior; looks specific to this machine's process/parent-process environment.

### Symptom

Every `get_transcript` (and presumably other transcript tools) call for video `kjoQPn--F7A` made through Claude Desktop fails with the user-facing message:
```
Transcript fetch timed out for video kjoQPn--F7A. Please try again.
```
"Please try again" is misleading: it fails identically on every retry within the same session, across multiple fresh server process restarts.

The project's own `errors.log` confirms this happened repeatedly, always for the same video/language, always clustered at almost exactly the `transcriptFetchTimeout` value (30s):
```
[2026-09-08T19:24:40Z] transcript_fetch kjoQPn--F7A: lang=en duration=33.937s category=timeout err=transcript fetch timed out
[2026-09-08T19:27:59Z] transcript_fetch kjoQPn--F7A: lang=en duration=33.921s category=timeout err=transcript fetch timed out
[2026-09-08T19:28:31Z] transcript_fetch kjoQPn--F7A: lang=en duration=31.003s category=timeout err=transcript fetch timed out
... (6 more, all 31.00–31.02s)
[2026-09-09T02:48:32Z] transcript_fetch kjoQPn--F7A: lang=en duration=34.571s category=timeout err=transcript fetch timed out
[2026-09-09T02:49:06Z] transcript_fetch kjoQPn--F7A: lang=en duration=31.001s category=timeout err=transcript fetch timed out
[2026-09-09T02:49:48Z] transcript_fetch kjoQPn--F7A: lang=en duration=31.002s category=timeout err=transcript fetch timed out
[2026-09-09T02:50:31Z] transcript_fetch kjoQPn--F7A: lang=en duration=31.016s category=timeout err=transcript fetch timed out
```
Critically: `go run ./cmd/youtube-cli transcript kjoQPn--F7A` on the **same machine, same video, same day**, run directly from a terminal, returned the full transcript **immediately** (well under a second to first output), with no hang at all — reproduced once, cleanly.

### Root cause

**Unknown — needs further investigation.** What was ruled out during this session's investigation:
- Not a stale binary: the `youtube-mcp.exe` Claude Desktop invokes (`C:\Users\Admin\go\bin\youtube-mcp.exe`) was built same-day, after the BUG-007 `ForceIPv4` fix, and that fix is present in the running binary.
- Not IPv6 routing (BUG-007's cause): `ForceIPv4()` is already unconditionally applied in `NewYtDlpCommand()` (`internal/core/ytdlp.go:129`), and a fresh CLI run succeeds instantly, which wouldn't happen if this machine's outbound routing were generally broken.
- Not a user/machine-wide HTTP(S) proxy: `HTTP_PROXY`/`HTTPS_PROXY` (User + Machine scope) are unset, and `netsh winhttp show proxy` reports direct access, no proxy.
- Not an explicit Windows Firewall block rule: no rule targets `yt-dlp.exe` or `youtube-mcp.exe` by name.
- Not simply "orphaned process contention": at one point 5 separate `youtube-mcp.exe` processes were found running simultaneously (PIDs accumulated across repeated Claude Desktop reconnects between 9:42–9:50 AM, since cleaned up — see Decision below), which looked like a plausible cause, but the actual per-connection log (`mcp-server-youtube-mcp.log`) shows the identical ~31s timeout pattern occurring on a single, freshly-started, otherwise-idle server process too (first `tools/call` after `initialize` succeeds in 2s — a metadata call — then every transcript call from that same lone process times out at ~31s). So process duplication is a real, separate problem (see Decision) but not the demonstrated cause of the timeout itself.

One piece of the symptom is now explained, not just observed: the near-exact **31.00-31.02s clustering** (vs. the raw 30s `transcriptFetchTimeout`) is `exec.Cmd.WaitDelay`, which `go-ytdlp` sets to its `cancelMaxWait` field (default 1s, `go-ytdlp@v1.3.6/command.go:29,266`) — the grace period given to the killed process to actually exit before `cmd.Run()` returns. So 30s timeout + ~1s `WaitDelay` = the observed ~31s consistently. This is a confirmed mechanism, not a guess, and it also means the tight clustering itself is *not* evidence of anything unusual by itself — it's just the timeout firing on schedule every time, which still leaves open why the underlying `yt-dlp` call never completes within that window when spawned under Claude Desktop.

Also newly identified: `go-ytdlp`'s `runWithResult` (`go-ytdlp@v1.3.6/command.go:293-331`) always populates `Result.Stdout`/`Result.Stderr` from writers attached to the child process's pipes *while it runs*, so on a timeout `result` already contains whatever `yt-dlp` printed up to the moment it was killed — but `fetchSegmentsFromYtDlp`'s `DeadlineExceeded` branch (`internal/core/transcript.go:285-294`) returns early without ever reading `result.Stdout`/`result.Stderr`. That's the concrete form of the "instrumentation gap" below: the diagnostic data already exists in memory at the point of failure, it's just discarded instead of logged.

What's left unexplained: the one reliable, reproducible distinguishing factor found so far is **process spawned as a child of Claude Desktop vs. spawned from an interactive terminal**, for the literal same `yt-dlp` invocation against the same video. Candidate explanations not yet confirmed or ruled out:
- AV/Defender or other endpoint-security behavioral scanning treating child processes of Claude Desktop differently from terminal-spawned processes (added latency on outbound connections established by such children).
- Some inherited process/security-context restriction specific to how Claude Desktop (an Electron app installed via MSIX-style packaging — its user data lives under `AppData\Local\Packages\Claude_<id>\...`) spawns child processes.
- A genuinely intermittent YouTube-side rate-limit/throttle (same family as BUG-001) that happened to coincide with the Claude Desktop session and had cleared by the time the CLI repro ran minutes later — can't be ruled out with the data collected so far, since only one CLI repro attempt was made.

The investigation was also blocked from going further by an **instrumentation gap**: `fetchSegmentsFromYtDlp`'s timeout logging (`internal/core/transcript.go`, task 16 observability) records only the classified failure string and elapsed duration — it does not capture `yt-dlp`'s own stdout/stderr up to the point of cancellation, nor the subprocess PID, nor whether the subprocess was actually killed on timeout. So there's no way, from the log alone, to tell whether the 30s was spent stuck at DNS resolution, TCP connect, TLS handshake, waiting on YouTube's response, or something else — which is what's actually needed to distinguish the candidate explanations above.

### Options

- **Add diagnostic instrumentation first (recommended before attempting a fix):**
  1. **(cheap, no added latency)** On the `DeadlineExceeded` path in `fetchSegmentsFromYtDlp`, log a tail of the already-captured `result.Stdout`/`result.Stderr` (they're populated by `go-ytdlp`'s `runWithResult` regardless of whether the call timed out — see Root cause above) instead of discarding `result` and logging only the generic "transcript fetch timed out" string.
  2. **(if step 1's captured tail comes back empty/uninformative on the next repro)** `Quiet()`+`NoWarnings()` are always set on this call (`internal/core/ytdlp.go` / `transcript.go:280-281`), which may suppress yt-dlp's phase-by-phase debug output (`[debug] ...`, `[youtube] Extracting URL...`, etc.) that would otherwise show whether it's stuck at DNS resolution, TCP connect, TLS handshake, or waiting on YouTube's response. Drop `Quiet()` (or add verbose output) specifically on this path to get that detail.
  3. **(heavier — only if PID/kill-confirmation is still needed after 1-2)** `go-ytdlp`'s public `Command.Run()` doesn't expose the underlying `*exec.Cmd`, so there's no `cmd.Process.Pid` to log. Getting it means bypassing `Command.Run()` and calling the exported `Command.BuildCommand(runCtx, videoURL)` directly, then wiring `Stdout`/`Stderr` and driving `Start()`/`Wait()` by hand — mirroring the pattern `DownloadVideoBlocking` already uses in `internal/core/download.go`. Log the PID right after `Start()`, and check `cmd.ProcessState` after `Wait()` returns to confirm the kill actually landed, to rule in/out a leaked/zombie subprocess (which would also explain the separate orphaned-`youtube-mcp.exe`-process pileup, if a handler blocked on a leaked subprocess is what's preventing prompt shutdown on reconnect).
  4. With the above in place, wait for the next live repro (in Claude Desktop) and capture the enriched log before proposing a fix.
- **Add orphan self-detection to `youtube-mcp.exe`** (`cmd/youtube-mcp/main.go`), *not* a single-instance/kill-siblings guard: a stdio MCP server is legitimately spawned once per client connection, so multiple simultaneous instances (Claude Desktop plus a terminal-launched one, or Desktop opening more than one connection) are a normal, valid state — a naive "kill any other running instance of myself" guard would tear down another client's live connection, not just true orphans, since the two cases are indistinguishable by process count alone. Instead, each instance should periodically check whether *its own* parent process (the process that spawned it) is still alive, and self-exit if not, and/or force-exit within a bounded grace period after detecting stdin EOF even if a handler is still blocked on a slow `yt-dlp` call, rather than waiting indefinitely for a graceful drain. This only ever acts on the process's own state, so it's safe under any number of concurrent legitimate clients, and is independently justified even if it turns out unrelated to the timeout's root cause.
  - **Must work identically on Windows, Linux, and macOS** — this project ships all three (`.goreleaser.yaml`, `docs/tasks/10-packaging/TASK.md`), even though CI (`docs/DECISIONS.md` DECISION-007) currently only runs the build/vet/test matrix on Ubuntu + Windows, not macOS. The "is my parent still alive" check is not portable by a single mechanism, though: on Unix (Linux/macOS), a process is automatically reparented (typically to PID 1 / launchd) the instant its original parent dies, so the simplest and fully portable-across-Unix signal is just polling `os.Getppid()` (Go stdlib, no platform-specific API needed) for a change from the PID observed at startup — no explicit liveness check required. Windows does **not** reparent orphans this way — `os.Getppid()` can keep returning the original (now-dead) parent PID indefinitely — so the Windows path needs an explicit liveness check on that original PPID (e.g. `OpenProcess`/`GetExitCodeProcess`, or the `github.com/mitchellh/go-ps`-style approach already informally precedented by this codebase's platform-conditional code in `internal/core/ffmpeg_prewarm.go`). The stdin-EOF-triggers-bounded-force-exit half of this option, by contrast, is naturally portable as-is (`os.Stdin` EOF detection is stdlib, not OS-specific).
- **Leave as-is / gather more data passively:** the single clean CLI repro succeeded, so this may be intermittent (YouTube-side throttling, transient AV scan) rather than deterministic; wait for more occurrences (ideally with the enriched logging above already in place) before committing to a fix direction.

### Decision

Pending — awaiting human decision on which option(s) to pursue. The orphaned-process pile-up found during this investigation (5 concurrent `youtube-mcp.exe` instances, accumulated across repeated Claude Desktop reconnects) was manually cleaned up (`Stop-Process`) as an immediate mitigation, but no code change has been made yet for either the process-duplication issue or the timeout itself — both remain open pending the decision above.

### Second occurrence (2026-09-14/15, video `X0UI0O8YzJM`)

User reported the same symptom via Claude Desktop for a different video
(`X0UI0O8YzJM`). Investigated via this project's `errors.log` (`os.UserCacheDir()/youtube-mcp/errors.log`)
and `%LOCALAPPDATA%\Claude\Logs\mcp-server-youtube-mcp.log`, after the
`transcript_fetch_timeout_output` logging from option 1 above (commit
0926a14) was already live.

- `errors.log` shows 11 timeouts clustered 2026-09-14T15:14Z–16:48Z, all
  ~31.0s (one 34.955s outlier), identical signature to the first
  occurrence: `category=timeout err=transcript fetch timed out`.
- The new `transcript_fetch_timeout_output` line is present on every one of
  those, but **both `stdout=` and `stderr=` are empty on every occurrence**
  — option 1's instrumentation is confirmed live and working, but there was
  nothing captured to diagnose with. This points at option 2 (`Quiet()`/
  `NoWarnings()` suppressing yt-dlp's phase output) as the next step, not
  yet tried.
- `mcp-server-youtube-mcp.log` confirms the MCP transport itself is not
  hanging: the server returns a `tools/call` result to Claude Desktop at
  almost exactly the same ~31s mark every time (e.g. request `id=2` at
  16:40:13.740Z, response at 16:40:48.700Z). So the user-visible "timeout"
  is our own `transcriptFetchTimeout` firing and being reported as a normal
  (if unwelcome) tool result, not a dropped MCP connection. Interspersed
  among the slow calls, a couple of `tools/call` requests in the same log
  window returned in under a second (`id=6`, `id=9`) — not confirmed
  whether those were transcript calls that happened to succeed fast or
  calls to a different, cheaper tool (e.g. metadata); worth checking
  `params`/tool name if the log is captured again.
- Two `youtube-mcp.exe` processes (PIDs 16952, 18060, both started
  2026-09-14 ~23:39-23:40 local) were still running at investigation time —
  consistent with the orphan-process pile-up already noted above, not
  independently resolved here (left running, not killed, since it wasn't
  confirmed whether either was still a live Claude Desktop connection).
- **Net new conclusion:** this rules out "video-specific" as an
  explanation — same asymmetry (CLI instant, Claude-Desktop-spawned ~31s
  timeout) now confirmed on two unrelated videos on two different days.
  Root cause is still unconfirmed; option 2 (drop `Quiet()` to see yt-dlp's
  own phase output on this path) is the most promising untried next step.
- **User-reported pattern (unverified):** the user's impression across
  their own repros is that this happens more often when the video has a
  human-uploaded ("manual") transcript, rather than only an
  auto-generated one. Not yet confirmed against the actual video set (both
  known repro videos, `kjoQPn--F7A` and `X0UI0O8YzJM`, would need their
  caption-track types checked to test this), but worth keeping in mind: the
  fetch call always requests `WriteAutoSubs()` **and** `WriteSubs()`
  together (`internal/core/transcript.go`), so a video offering a manual
  track may make yt-dlp do extra track-listing/selection work under the
  hood versus a video with only the auto track — a plausible mechanism for
  this correlation, not confirmed.

### Action taken (2026-09-15): switched to `Verbose()` for the next repro (option 2)

Per human decision, replaced `.NoWarnings().Quiet()` with `.Verbose()` on
the transcript-fetch `yt-dlp` command (`internal/core/transcript.go`,
`fetchSegmentsFromYtDlp`) so the next timeout's captured `stdout`/`stderr`
tail (already logged via the `transcript_fetch_timeout_output` line from
the first occurrence's fix) should show yt-dlp's phase-by-phase progress
instead of nothing. `go build ./...` and `go vet ./...` pass. Not yet
verified against a live repro — needs the next Claude Desktop timeout to
confirm the captured output is actually useful this time. If the extra
verbose noise turns out to be a problem for the non-timeout `yt-dlp
failed: %s` error path (which also reads `result.Stderr`), that's a
follow-up to watch for, not addressed here.

### Action taken (2026-09-15): PID + kill-confirmation logging (option 3)

Per human decision, `fetchSegmentsFromYtDlp` (`internal/core/transcript.go`)
now bypasses `Command.Run()` and instead calls `Command.BuildCommand()` +
`Start()`/`Wait()` directly (mirroring the existing pattern in
`DownloadVideoBlocking`/`DownloadAudioBlocking`, `internal/core/download.go`),
since `go-ytdlp`'s `Run()` doesn't expose the underlying `*exec.Cmd` or its
PID. On a timeout, the `transcript_fetch_timeout_output` log line now also
includes `pid=<n>` (the subprocess PID) and `killed=<bool>` (`true` iff
`exec.Cmd.ProcessState` is non-nil and `Exited()` returns true by the time
`Wait()` returns) — this should confirm or rule out a leaked/zombie
subprocess, which was one of the still-open explanations noted above and
would also account for the separate stale-process pileup already observed.
`go build ./...`, `go vet ./...`, `gofmt -l .` (clean), and `go test ./...`
all pass; also re-verified end-to-end with a live CLI transcript fetch
against `X0UI0O8YzJM`, unaffected. Not yet verified against a live Claude
Desktop timeout — needs the next repro to confirm the new fields are
populated and useful.

### Third occurrence (2026-09-15, video `X0UI0O8YzJM`) — first useful capture, and a lead

User re-ran the failing video through Claude Desktop right after the two
actions above were live (the server binary in `go\bin` had just been
rebuilt). `errors.log` at `2026-09-14T18:47:39Z` (01:47 local):

- `transcript_fetch X0UI0O8YzJM: lang=en duration=34.378s category=timeout`
  — same ~31-35s signature.
- `transcript_fetch_timeout_output X0UI0O8YzJM: lang=en pid=19056
  killed=true stdout=... stderr=...` — **both new fields populated, and
  the verbose capture is no longer empty.** `killed=true` rules out a
  leaked/zombie subprocess on this path.
- The captured stdout ends with:
  ```
  [youtube] X0UI0O8YzJM: Downloading m3u8 information
  [info] X0UI0O8YzJM: Downloading subtitles: en
  [info] Testing format 616
  ```
  i.e. yt-dlp got through the page fetch and the subtitle listing, then
  was killed while **probing format 616** (YouTube's premium HLS video
  format) with a live network request. That probe happens because
  yt-dlp's default format selection still runs on a `--skip-download`
  subtitle fetch and tests any format the extractor flags for testing.
- stderr (verbose) shows `JS runtimes: none` and `Proxy map: {}`.

Comparison from a CLI shell, same yt-dlp binary, same flags, same video:
the identical `Testing format 616` line appears, the probe completes, and
the subtitle file is written in ~7s. So the CLI/Desktop difference is
**not** in what yt-dlp does — both environments also report
`JS runtimes: none`, so a JS-runtime difference is ruled out — it's that
this one googlevideo probe stalls in the Desktop-spawned environment (same
flavour as BUG-007's hanging googlevideo request). Why it stalls there is
still unconfirmed.

Two CLI runs with one extra flag each removed the probe entirely
(`Testing format` line absent, identical `sub.en.vtt` written, 5-6s):
`--no-check-formats`, and `-f ba`. `--no-check-formats` is the safer of
the two (it only disables format probes, which a subtitles-only fetch
never needs; `-f ba` would fail on a video with no audio-only format).

### Action taken (2026-09-15): `--no-check-formats` on the transcript fetch

Per human decision, added `NoCheckFormats()` to the transcript-fetch
command in `fetchSegmentsFromYtDlp` (`internal/core/transcript.go`) —
**only** there, not on the shared `NewYtDlpCommand`, because the download
paths in `download.go` rely on format checks to skip dead formats. Effect
on the transcript path: yt-dlp no longer probes any format, so the step
it was being killed in no longer exists; subtitle output is unchanged
(verified byte-identical from the CLI). If the Desktop-spawned fetch
still times out after this, the verbose capture will show the *next*
stalling step, which would mean the environment stalls on more than this
one request.

### Verified (2026-09-15, Claude Desktop, video `X0UI0O8YzJM`)

User rebuilt the `go\bin` binary from a branch carrying this change, fully
quit and reopened Claude Desktop, and reran the same video that had timed
out 12+ times over two days. Desktop's `mcp-server-youtube-mcp.log`: server
restarted `2026-09-14T19:07:16Z` (02:07 local), `tools/call id=2` sent at
`19:07:52.480Z`, result returned at `19:08:02.162Z` — **9.7s, transcript
fetched.** `errors.log` has no new `transcript_fetch` entry after the
`18:47:39Z` timeout from the run before the fix. User confirmed the
transcript came through.

**Resolution:** the timeout was yt-dlp's default format probe (`Testing
format 616`, a live request to a googlevideo HLS URL) stalling in the
Claude-Desktop-spawned process environment while completing in seconds
from a CLI shell. Removing the probe from the subtitles-only fetch
(`NoCheckFormats()`) removes the stalling step; the transcript path never
needed it. *Why* that one request stalls only under Desktop is still not
explained (same binary, same flags, same `JS runtimes: none`, `Proxy map:
{}` — the remaining suspects are process-environment differences such as
inherited network/proxy settings, in the same family as BUG-007's IPv6
hang), but with the probe gone there is no reachable code path that makes
the request, so this is closed rather than kept open on that question. The
`Verbose()` + `pid=`/`killed=` logging from the earlier actions stays in
place — it is what made this diagnosable and costs nothing on the success
path.

---

## BUG-009: Auto-generated captions 429 on a separate, stricter YouTube quota than manual captions, with no retry/backoff in the tool

- **Status:** fixed
- **Discovered:** user-reported session, 2026-09-09, after `get_transcript` failed repeatedly (6 manual retries with increasing backoff up to 30s) against video `5oer61Xyi4c` (~6.5 hours long) through Claude Desktop, then continued failing ~3 hours later against unrelated videos.
- **Reachability: yes** — real call path (`get_transcript`/etc. → `core.GetTranscriptText`/etc. → `fetchSegmentsFromYtDlp`, `internal/core/transcript.go`), hit repeatedly by a real user in real sessions, not a test-only branch.
- **Inherited from upstream:** no — this is YouTube-side rate limiting, not a code defect the port introduced or carried over; the same 429 reproduces identically calling `yt-dlp` directly, outside any of our Go code.

### Symptom

`get_transcript`/`youtube-cli transcript` fails with:
```
error: Failed to fetch transcript for video <id>: yt-dlp failed: ERROR: Unable to download video subtitles for 'en': HTTP Error 429: Too Many Requests
```
This was first suspected to be either an MCP-specific problem or a general IP-wide ban, but neither held up under investigation:
- The plain CLI (`go run ./cmd/youtube-cli transcript <id>`), run directly in a terminal, reproduces the identical 429 — ruling out anything MCP/Claude-Desktop-specific.
- Metadata scraping (`youtube-cli metadata <id>`) for the same video succeeds fine at the same time — ruling out a full IP ban.
- A video with **manually-uploaded** English subtitles (`dQw4w9WgXcQ`) fetched its transcript successfully, repeatedly, throughout the same window that two other videos with **only auto-generated captions** (`5oer61Xyi4c`, `BqRhBq-_kgE`) both 429'd consistently — including ~3 hours after the original 6-retry failure, and regardless of which yt-dlp player client (`web`/`visionos` default vs. explicit `android` via `--extractor-args`) was used.
- Direct `yt-dlp` calls confirm the split: `--write-sub` (manual captions) on `dQw4w9WgXcQ` downloads instantly; `--write-auto-sub` (auto captions) on either of the other two videos 429s every time.

### Root cause

YouTube appears to enforce a separate, stricter rate-limit bucket for the auto-generated-caption ("ASR"/ `kind=asr` timedtext) download endpoint than for manually-uploaded caption tracks. Once that bucket is exhausted — plausibly by pulling a 6.5-hour video's auto-captions, which likely requires many more underlying timedtext segment requests than a typical short video — every subsequent auto-caption request from this IP fails with 429, for any video, for an extended period (still failing ~3 hours after the triggering pull), while manual-caption downloads and general page/metadata scraping are entirely unaffected.

`fetchSegmentsFromYtDlp` (`internal/core/transcript.go`) has no retry or backoff of its own on a 429 — it fails the request immediately and surfaces the raw yt-dlp error. The BUG-008 investigation separately confirmed the timeout-clustering mechanism for a different symptom (30s deadline exceeded); this bug is a distinct, now-diagnosed root cause specifically for the 429 case, not the BUG-008 timeout.

### Options

- **Do nothing (accept as inherent external limit):** this is YouTube throttling its own service, not something client-side retry can reliably outrun if the quota window is long (observed still active 3+ hours later); document it as expected behavior for heavy auto-caption usage and let users wait it out or switch network/IP.
- **Add bounded retry-with-backoff specifically on 429** in `fetchSegmentsFromYtDlp`: would help for short-lived, low-volume 429s but would not have helped in this session's repro, since the throttle was still active 3 hours and many retries later — risks adding latency/complexity for a quota window client-side retry can't shorten.
- **Surface a clearer, more specific user-facing error** distinguishing "auto-caption quota throttled, try again later or use a different network" from the current generic "Too Many Requests" message, without adding retry logic — cheaper than a retry loop and sets correct user expectations (this is not a transient blip that a few more seconds will fix).

### Decision

Fix now (human decision, this session): **Option 3 only** — a clearer,
429-specific user-facing message, no retry/backoff logic. Implemented as:
1. `classifyTranscriptError` (`internal/core/transcript.go`) gained a new
   `"rate_limited"` category, matched on `"429"` / `"Too Many Requests"` —
   confirmed no collision with the existing `timeout`/`missing_captions`/
   `network` substrings.
2. `TranscriptErrorText` returns a message that explicitly says this is
   YouTube-side throttling, not a transient blip, that immediate retry
   won't help, and to wait or switch networks — deliberately not
   suggesting a short retry, since this session's data showed the throttle
   still active 3+ hours and many retries after the triggering pull.
3. Unit-tested (`internal/core/transcript_test.go`,
   `TestTranscriptErrorText`/`TestClassifyTranscriptError`/
   `TestFormatTranscriptFailureLog`/`FuzzTranscriptErrorText`) against the
   exact raw 429 string captured live during this bug's investigation.
4. Verified end-to-end: `youtube-cli transcript 5oer61Xyi4c` (still
   throttled at verification time) now returns the new message instead of
   the raw yt-dlp error.

No retry/backoff logic was added (Option 2, declined) — this session's own
repro data (still 429ing 3+ hours and many retries later) showed retries

**Amendment (2026-09-20, see BUG-011):** the root cause above (a separate, stricter quota for auto-generated captions) is now **unconfirmed**. A later investigation found that a 429 is also returned for any *translated* caption track (e.g. the default `en` on a non-English video), independent of quota, and that `5oer61Xyi4c`/`BqRhBq-_kgE` (both English) fetch fine with `en` today. Whether a real IP-wide auto-caption throttle exists remains open. The user-facing message written for this bug was reworded under BUG-011 so it no longer asserts one.
don't reliably outrun YouTube's cooldown for this specific throttle.

---

## BUG-010: `parseVtt` returns each auto-caption line ~3 times (rolling-cue carry blocks survive the dedupe)

**Found:** 2026-09-15, during task 17's ground-truth capture
(`docs/tasks/17-video-brief/TASK.md`), by running `parseVtt` on a real
yt-dlp `--write-auto-subs` file for `dQw4w9WgXcQ`.

**Reachability: yes.** Any video whose only English track is YouTube's
auto-generated (ASR) one — the majority of videos — goes through this
path on every `get_transcript`/`get_transcript_timed`/`get_transcript_range`
/`search_transcript`/`download_transcript*` call and the CLI equivalents.
Videos with an uploaded track are unaffected (yt-dlp prefers the uploaded
track when both flags are passed, and uploaded VTTs are not rolling-style).

### Symptom

For an auto-caption video, the parsed transcript contains every line about
three times, shifted by a few seconds each. Real `parseVtt` output on the
captured sample (offset ms, duration ms, text):

```
 18800 + 2990  "We're no strangers to"
 21790 +   10  "We're no strangers to"
 21800 + 4150  "We're no strangers to love. You know the rules and so do"
 25950 +   10  "love. You know the rules and so do"
 25960 + 3149  "love. You know the rules and so do I. I feel commitments from what I'm"
```

95 segments came out of a file whose actual caption content is ~30
distinct lines. Word counts (and anything derived from them, e.g. task
17's speaking-rate stat) are inflated ~3×, transcript text is ~3× longer
than it should be, and `search_transcript` reports the same hit at
several nearby timestamps.

### Root cause

YouTube's ASR captions, as converted to VTT by yt-dlp, are "rolling"
two-line cues: each block repeats the previous line as its first line
(context), then the new line with inline `<00:00:19.039><c> word</c>`
word timings; between blocks there is a ~10 ms carry block whose text is
the previous line alone. `parseVtt` (ported faithfully from the upstream
TS) joins all lines of a block, strips tags, and dedupes on
`offset|text` — but every one of these blocks has a *different* offset,
so nothing is deduped. The upstream TS project has the same behavior;
this is inherited, not introduced by the port.

### Options

1. **Content-aware parse for rolling cues (recommended, minimal):** when
   the file is rolling-style (task 17's `detectCaptionKind` == auto, i.e.
   inline word timings are present), take only the *last* non-empty line
   of each block as the cue text, and drop a block whose resulting text
   equals the previously kept segment's text (that's the 10 ms carry).
   On the sample this yields exactly one segment per real line (`"We're
   no strangers to"`, `"love. You know the rules and so do"`, `"I. I feel
   commitments from what I'm"`, `"thinking"`, ...), with the offset of the
   block that introduced the line. Uploaded VTTs are untouched.
2. Request a non-rolling subtitle format from yt-dlp (`--sub-format
   json3`/`srv3`) and write a second parser — bigger change, new format
   fixtures, drops the TS-parity ground truth for the VTT parser.
3. Leave as-is (upstream parity) and only document — rejected on the
   merits: it makes every auto-caption transcript wrong for agents.

### Decision

Pending — awaiting human decision. Task 17 proceeds without changing
`parseVtt`; its stats are computed on whatever `parseVtt` returns, so a
fix here automatically corrects them. Until then, `get_video_brief`'s
`Words`/`Speaking rate` on a `likely auto-generated` video should be read
as ~3× too high.

---

## BUG-011: Transcript 429 on a *translated* caption track is reported as "network throttled, wait hours" — wrong diagnosis, and it corrects BUG-009's root cause

- **Status:** fixed (message-only; branch `fix/bug-011-translated-track-429`)
- **Discovered:** 2026-09-20, user-reported `get_transcript` failure on `r8CppXSqVDU` (Vietnamese speech, auto-captions only) with the BUG-009 message.
- **Reachability: yes** — real call path (`get_transcript`/etc. → `normalizeLanguage` defaults to `"en"` → `fetchSegmentsFromYtDlp`, `internal/core/transcript.go`), hit by a real user in a real session. Every non-English video fetched without an explicit `language` goes through it.
- **Inherited from upstream:** the `language` default of `"en"` is (TS default is `'en'`); the misleading message is ours (BUG-009's fix).

### Symptom

```
YouTube is throttling auto-caption downloads for this network right now (video r8CppXSqVDU). This isn't a transient blip — it can take hours to clear after heavy usage, and retrying immediately won't help. Wait before trying again, or use a different network.
```

The message tells the user to wait hours or change network. For this video that is wrong: the same network, same minute, succeeds with the right language code.

### Evidence (all 2026-09-20, same machine/network, yt-dlp 2026.07.04)

| Video | Spoken language | Request | Result |
|---|---|---|---|
| `r8CppXSqVDU` | Vietnamese (ASR only) | `en` (the default) | **429** — via our CLI *and* via `yt-dlp --write-auto-subs --sub-langs en` directly |
| `r8CppXSqVDU` | Vietnamese | `vi` | OK — full transcript |
| `BqRhBq-_kgE` | English | `en` | OK |
| `BqRhBq-_kgE` | English | `vi` | **429** (yt-dlp direct) |
| `5oer61Xyi4c`, `dQw4w9WgXcQ` | English | `en` | OK |

`yt-dlp --list-subs r8CppXSqVDU` lists `vi-orig` (the native ASR track) and offers `en` as one of ~100 machine-translated "automatic captions". Requesting a language other than the video's spoken one selects a *translated* track; that download is what 429s. The requested track's language, not the network, is the variable: the 429 flips with the language code on the same network and video.

### Root cause

For an auto-caption video, requesting a language that isn't the spoken one makes YouTube serve a machine-translated track, and that request is rejected with a 429 (at least today, from this network). `normalizeLanguage` defaults to `"en"`, so any non-English auto-caption video fetched with no `language` argument hits this. `classifyTranscriptError` maps every 429 to the "network throttled" text (BUG-009), so the actual remedy (pass the video's own language) is never surfaced.

### Effect on BUG-009 (unresolved — needs a decision, not assumed)

BUG-009 attributed its 429s to a separate, stricter quota for auto-generated captions. This entry shows that translated-track requests 429 independently of any accumulated quota, so that explanation is at best incomplete. It does **not** by itself explain BUG-009's original repro, because `5oer61Xyi4c` and `BqRhBq-_kgE` are English videos and both fetch fine with `en` today. Possible readings, none verified:
- the original throttle was real and has since cleared (11 days later; no way to test retroactively);
- the yt-dlp version changed between then and now (BUG-006 moved the pin from 2026.03.17 to 2026.07.04) and the older one selected a different track for `en`;
- the BUG-009 note calls `5oer61Xyi4c` "~6.5 hours long" but its metadata now reports 20:36, so that note may have described a different video or been misrecorded.

A real IP-wide auto-caption throttle is therefore neither confirmed nor ruled out; the current message asserts it as fact.

### Options

- **Message-only (smallest):** make the 429 text stop asserting a network-wide throttle, and add the language hint — e.g. "YouTube rejected the caption download (HTTP 429). If this video isn't in English, retry with `language` set to its spoken language (a translated track is what gets rejected); otherwise this may be a temporary throttle — wait or try another network." No behavior change.
- **Auto-detect the spoken language when `language` is omitted:** feature, not a bug fix — needs the original-language signal from yt-dlp (`vi-orig`-style track / metadata) or the watch-page scrape. Changes the documented default; goes through the normal task-approval process.
- **Fall back to the native track on a 429 for a translated request:** feature; adds a second yt-dlp call and a "which language did I actually return" disclosure to the result.
- **Amend BUG-009's text** to link here and downgrade its root-cause claim to "unconfirmed" (docs only, could accompany the message fix).

Recommendation: message-only fix plus the BUG-009 amendment now; treat auto-detect / fallback as a separate feature task for the human to schedule.

### Decision

Fix now (human decision, 2026-09-20): **message-only fix + BUG-009 amendment**, as recommended. Auto-detect / native-track fallback are left as a possible future feature task, not scheduled here.

1. `TranscriptErrorText`'s `rate_limited` message (`internal/core/transcript.go`) no longer asserts a network-wide throttle. It now says YouTube rejected the download (HTTP 429), tells the user to set `language` to the video's spoken language if it isn't English (the default `"en"` selects a translated track), and otherwise mentions a possible temporary throttle. No signature or behavior change; `classifyTranscriptError` untouched.
2. `TestTranscriptErrorText` updated first (red, then green) against the same captured yt-dlp 429 stderr; the rate-limited case's expected text is the new message.
3. BUG-009 amended (below its Decision) to link here and downgrade its root cause to unconfirmed.

**Follow-up (2026-09-20): task 18** (`docs/tasks/18-native-language-fallback/TASK.md`, DECISION-022) resolves an omitted `language` from the video's caption tracks, so the default-`en` trigger no longer occurs for `get_transcript`, `get_transcript_timed`, `get_transcript_range`, `search_transcript` and the CLI `transcript`/`search`. It still applies to an explicit `en` on a non-English video, and to `download_transcript*`, `get_video_brief` and `search_playlist`, which keep the plain `en` default. Tracked as task 19 (`docs/tasks/19-language-resolution-remaining-tools/TASK.md`).
