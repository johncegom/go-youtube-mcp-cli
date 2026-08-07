# Out-of-band: fuzz tests for the highest-value untrusted-input parsers

**Status:** done

Added Go native fuzz tests (`go test -fuzz`) for the three pure functions
that parse the most externally-influenced/adversarial-ish input:

- **`FuzzExtractVideoID`** (`internal/core/videoid_test.go`) — invariant: any
  non-empty result is always exactly an 11-char `[a-zA-Z0-9_-]` ID (it gets
  embedded directly into a URL and a `yt-dlp` subprocess argument
  elsewhere).
- **`FuzzSanitizeTitle`** (`internal/core/format_test.go`) — invariant:
  result is never empty, and never contains a path-separator or other
  filesystem-illegal character. Flagged as security-relevant, not cosmetic:
  video titles are attacker-influenced (set by the video's uploader) and
  this is the *only* sanitization before the result is used to build a real
  filesystem path in `SaveTranscriptFile`/`StartVideoDownload`/
  `StartAudioDownload`.
- **`FuzzParseVtt`** (`internal/core/transcript_test.go`) — invariant: must
  not panic. VTT content comes straight from a `yt-dlp`-downloaded subtitle
  file (network-sourced, effectively untrusted).

Each was fuzzed for 10s locally (`go test ./internal/core/... -run '^$'
-fuzz FuzzX -fuzztime 10s`) — 150K–380K executions each, zero failures,
confirming the invariants hold. No crashers were found, so nothing was
added to `testdata/fuzz/` (Go only persists regression corpus entries there
on an actual failure). Seed corpora run as ordinary test cases under plain
`go test ./...`, so this is fully CI-safe with no added runtime cost beyond
the seeds.

Also added `.claude/` to `.gitignore` (a Claude Code tool-generated lock
file had shown up as untracked; not project source).
