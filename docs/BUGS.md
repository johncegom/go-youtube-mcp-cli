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
