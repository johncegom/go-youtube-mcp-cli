# Task 19: Apply spoken-language resolution to the tools task 18 left out

**Status:** in progress. Definition of Done + Test Plan approved by the
human 2026-09-20, after an Advise call (logged in `docs/eagd-log.md`) and
the human's review decisions (filename gets `_<lang>` for every save;
playlist gets the minimum improvement only). Exists so the scope cut
recorded in `docs/DECISIONS.md` DECISION-022 has a tracked owner instead of
being a note someone has to remember.

## User need

Task 18 made an omitted `language` resolve to the video's spoken language
for `get_transcript`, `get_transcript_timed`, `get_transcript_range`,
`search_transcript` and the CLI `transcript`/`search`. Three surfaces still
default to plain `en`, so the same non-English auto-caption video works in
one tool and fails in another with the BUG-011 error:

| Surface | Entry point | Today with a Vietnamese video, `language` omitted |
|---|---|---|
| `download_transcript`, `download_transcript_timed`, CLI `transcript --save` | `core.SaveTranscriptFile` (`normalizeLanguage` inside) | BUG-011 error |
| `get_video_brief` | `core.FetchVideoBrief` (`normalizeLanguage` inside) | metadata + chapters fine; transcript section fails |
| `search_playlist` | `core.SearchPlaylist` → `searchPlaylistEntries` (`normalizeLanguage` inside) | every non-English video is reported as skipped/failed |

Workaround today: pass `language` explicitly. The goal is that an agent never
needs to know that.

## Approach

Reuse task 18's `core.ResolveLanguage` and `core.LanguageNote`; no second
resolution mechanism. An explicit `language` is always honored and skips
resolution. The note is shown only when the omitted language resolved to
something other than `en`. Each surface has a different cost profile, so the
mechanism differs; the reason is recorded per surface.

- **Save (`SaveTranscriptFile`).** Handlers and the CLI `--save` path call
  `ResolveLanguage` first (task 18's shape) and pass the result down. The
  note goes in the tool/CLI result text (`Saved to: …`), never inside the
  file body. Additionally the saved file's metadata header gets a
  `**Language:** <code>` line **only when the language used is not `en`**
  (auto-detected or explicit), so English file *contents* stay byte-identical (only the filename changes,
  below). Reason the language belongs in the file: it is a durable artifact re-read later
  with no memory of the request.
  **Filename (human decision):** every save is named
  `<title>_<lang>[_timed].md` — including `en` (`title_en.md`), so two
  languages of one video never overwrite each other. This renames files
  for existing English users; deliberate. The `<lang>` component comes from
  an MCP-controlled argument, so it is restricted to `[A-Za-z0-9_-]` (any
  other character becomes `_`) before it is joined to `outputDir`.
- **Brief (`FetchVideoBrief`).** Call `ResolveLanguage` **inside the
  transcript goroutine**, before `fetchTranscript`, not before the two
  goroutines start. The metadata/chapters goroutine stays fully concurrent,
  so latency is `max(metadata, resolve + transcript)` and the ~1 s resolve is
  usually hidden behind the yt-dlp fetch. The goroutines still write disjoint
  `VideoBrief` fields; no new plumbing or failure path. `VideoBrief` gains a
  `Language` field, rendered as a `Language:` line next to `Caption kind`,
  **only when not `en`**. Explicit `language` behaves exactly as today.
- **Playlist (`SearchPlaylist`) — minimum improvement only (human
  decision).** No resolution and no retry: playlists stay on the plain `en`
  default. The per-video skip line already carries the BUG-011 language
  hint (`TranscriptErrorText`'s rate-limited text says to set the language
  option to the spoken language), so the only change is that the
  `search_playlist` `language` description says plainly that the default is
  `en` and is **not** auto-detected for playlists, and to pass `language`
  for a non-English playlist. The lazy retry-on-failure design Advise
  recommended is parked under "Future improvements" below, with its
  constraints, so it is not lost.

## Definition of Done

- [ ] 19.1 `SaveTranscriptFile` path: an omitted `language` resolves via
  `ResolveLanguage` in the handlers and CLI `--save`; explicit `language` is
  untouched; the note (non-`en` only) appears in the MCP result and CLI
  `--save` output; the saved header has `**Language:** <code>` only when the
  language used is not `en`; the body of an `en` save is byte-identical to today. Saved files are named
  `<title>_<lang>[_timed].md` for every language including `en`; the `<lang>`
  component is sanitized to `[A-Za-z0-9_-]` — a test proves a language
  argument like `../../x` cannot produce a path outside `outputDir`.
- [ ] 19.2 `FetchVideoBrief`: `ResolveLanguage` runs inside the transcript
  goroutine; metadata/chapters remain concurrent; `VideoBrief.Language` set
  and a `Language:` line rendered only when not `en`; explicit `language`
  and resolution failure behave exactly as today; the per-section
  partial-failure contract is unchanged (`isError` iff the transcript
  section failed).
- [ ] 19.3 `search_playlist`: no behavior change. Its `language` description
  states that the default is `en` and is not auto-detected for playlists;
  a test (or the existing skip-line test extended) shows a per-video 429
  failure in the skipped list carries the language hint text.
- [ ] 19.4 Unit tests for each path with the stubbed `lookupSpokenLanguage`
  seam (no network):
  omitted+non-en → resolved language used, note/`Language:` line present;
  omitted+`en` → none; explicit language → lookup never called; lookup
  error → `en`, no note; brief: metadata section unaffected when
  the transcript goroutine's resolve fails.
- [ ] 19.5 `language` descriptions: `download_transcript*`, `get_video_brief`
  and CLI `--save` say the default is the video's spoken language (task 18's
  wording); `search_playlist` says it is `en` and not auto-detected. No
  description misstates its tool's actual default.
- [ ] 19.6 Live smoke against `r8CppXSqVDU` (Vietnamese ASR only): the
  download tool saves `<title>_vi.md` (Vietnamese) with the `**Language:** vi`
  header, and the brief's transcript section succeeds with a `Language:` line.
  `BqRhBq-_kgE` (English) saves `<title>_en.md` with no `**Language:**` header
  line and behaves as before on the brief (no `Language:` line).
  `search_playlist` is unchanged (its skip line for a non-English video
  carries the language hint).
- [ ] 19.7 `go build/vet/test ./...` + `gofmt -l` clean; DECISION-022's
  "inconsistency by design" consequence updated to say task 19 closed it (or
  names what remains); BUG-011's follow-up pointer updated; ledger + this
  file updated.

## Test Plan

- **Unit:** 19.4 through `withStubbedLookup` (already in
  `internal/core/language_test.go`); handler-level composition tests alongside the
  existing `withLanguageNote` test.
- **Manual smoke (live):** 19.6. If YouTube stops rejecting translated
  tracks the smoke test proves nothing and the unit tests are the coverage.
- Commands: `go test ./internal/core/... ./internal/mcpserver/... ./internal/cli/... -v`,
  `go build ./... && go vet ./... && go test ./... && gofmt -l internal/`.

## Notes for the implementer

- **Cache footprint.** The transcript cache key is `{videoID, language}`.
  Once resolution returns `vi` for a video, an entry cached earlier under
  `en` is dead weight and the first post-change call is a guaranteed miss.
  That is expected, not a cache bug.
- **`ResolveLanguage` does not memoize its error path**, so a video whose
  watch page keeps failing re-pays the ~1 s lookup on every call. Harmless for
  the single-video surfaces in this task; it matters only if the parked
  playlist design (see "Future improvements") is revived, where it must be
  weighed again.

## Out of scope / open

- **Filename suffix — decided by the human (2026-09-20):** `_<lang>` for every
  save, including `en` (see Approach). Advise had recommended it for
  non-`en` only; the human chose all languages for consistency.
- Naming the spoken language inside the BUG-011 error for an *explicit*
  `en` on a non-English video (needs a lookup on the error path and
  `TranscriptErrorText` has no `ctx`). Separate candidate follow-up.
- A cheaper `captionTracks` extraction than the existing
  `ytInitialPlayerRe` scan (~1 s on a real page); worth doing only if the
  lookup cost starts to matter.

## Future improvements / research (not scheduled)

Logged so they are not lost; also indexed in `docs/LEDGER.md` "Backlog".

- **Playlist lazy retry-on-failure (Advise-recommended full fix).** Per
  entry, fetch with the caller's language; only if `language` was omitted and
  the error classifies as `rate_limited`, call `ResolveLanguage` and, if it
  returns something other than `en`, retry once in that language. Costs
  nothing for all-English playlists (no 25 x ~1 s of extra youtube.com page
  fetches, which is the real 429 exposure); fixes mixed-language ones.
  Constraints if revived: the retry is a second yt-dlp run, so count it as a
  cache miss for the 500 ms pacing (sleep before the retry as well as before
  the next entry); inject the resolver into `searchPlaylistEntries` next to
  the `fetch`/`sleep` seams; never retry `missing_captions` or an explicit
  `language`; keep the `<title> [MM:SS] <text>` line shape and report the
  language in a trailing `Searched in a non-English language:` section.
  Why it does not repeat Advise's task-18 objection to "fall back after a
  429": the failure only decides *when to look*; the language still comes
  from `captionTracks` and is reported.
- **Research: is the translated-track 429 permanent or does it vary?** The
  finding (n=4 in task 18's 18.0, plus BUG-011) is from one network on one
  day. Re-check occasionally (other networks, other times); if translated
  tracks sometimes succeed, `en`-first fallbacks regain value.
- **Research: resolve from `yt-dlp` instead of the page scrape?** yt-dlp
  lists a `<lang>-orig` native track; a resolver based on it would avoid our
  own `ytInitialPlayerRe` scan but costs a subprocess. Compare only if the
  page scan (~1 s) starts to matter.
