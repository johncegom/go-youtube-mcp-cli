# Task 19: Apply spoken-language resolution to the tools task 18 left out

**Status:** draft (2026-09-20) — not started. Definition of Done + Test Plan
below are a first draft and need human review before any coding; per
`CLAUDE.md`, an Advise call runs once the draft is settled (the two open
design questions below are exactly its kind of question). Exists so the
scope cut recorded in `docs/DECISIONS.md` DECISION-022 has a tracked owner
instead of being a note someone has to remember.

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

## Approach (drafted; two questions open for Advise)

Reuse task 18's `core.ResolveLanguage` and `core.LanguageNote`; do not add a
second resolution mechanism. An explicit `language` is still always honored.
The note is still shown only when the omitted language resolved to something
other than `en`. The open parts are where resolution happens and how each
surface shows the note.

- **`SaveTranscriptFile`** — handlers and CLI call `ResolveLanguage` first
  (same shape as task 18) and pass the result down. Note goes in the tool's
  result text (`Saved to: …`), not inside the saved Markdown body. *Open:*
  should the saved file's metadata header also gain a `**Language:**` line
  (a saved file that doesn't say which language it holds is arguably a real
  gap, and unlike the transcript body a header line breaks nothing)?
- **`get_video_brief`** — *Open question 1.* `FetchVideoBrief` fetches the
  watch page (metadata + chapters) and the transcript concurrently. Resolving
  first would fetch the page twice (once for the language, once for
  metadata) and serialize the two sections; alternatively the language can be
  taken from the page fetch the brief already does, at the price of starting
  the transcript only after it. The brief's existing "Caption kind" line
  suggests a natural `Language` line for the output.
- **`search_playlist`** — *Open question 2.* Resolution is per video, and a
  playlist search walks up to 25 videos sequentially with delays already; one
  extra page fetch per uncached video (~1 s each, memoized afterwards) adds up
  to tens of seconds worst case. Options: resolve per video lazily only on a
  transcript-cache miss; resolve once from the first video and apply to all
  (wrong for mixed-language playlists); or leave playlists on `en` and just
  make the per-video skip message name the language hint. The output would
  also need to say which language each video was searched in.

## Definition of Done (draft)

- [ ] 19.0 Settle the two open questions (Advise, then human review), and
  update this file with the chosen designs before coding.
- [ ] 19.1 `SaveTranscriptFile` path: an omitted `language` resolves via
  `ResolveLanguage`; explicit `language` is untouched; the auto-detected note
  (non-`en` only) appears in the MCP tool result and the CLI `--save` output,
  and (if 19.0 says so) as a `**Language:**` line in the saved file header.
- [ ] 19.2 `get_video_brief`: an omitted `language` resolves via the design
  chosen in 19.0; the brief states the language in the same non-`en`-only
  way; a resolution failure falls back to `en` exactly as today; the
  brief's per-section partial-failure behavior is unchanged.
- [ ] 19.3 `search_playlist`: behavior per the design chosen in 19.0,
  including how each video's language is reported, without making the
  worst-case runtime materially worse than today's playlist search.
- [ ] 19.4 Unit tests for each path with the stubbed `lookupSpokenLanguage`
  seam (no network): omitted+non-en resolved → resolved language used and
  note present; omitted+`en` → no note; explicit `language` → lookup never
  called; lookup error → `en`, no note, today's behavior.
- [ ] 19.5 `language` tool/flag descriptions for the three surfaces say the
  default is the video's spoken language (mirroring task 18's wording), and
  no description still claims a plain `en` default.
- [ ] 19.6 Live smoke against `r8CppXSqVDU` (Vietnamese ASR only): the
  download tool saves a Vietnamese transcript, the brief's transcript
  section succeeds, and a playlist search over a video set that includes
  it finds matches; an English video (`BqRhBq-_kgE`) behaves exactly as
  before with no language note.
- [ ] 19.7 `go build/vet/test ./...` + `gofmt -l` clean;
  `docs/DECISIONS.md` DECISION-022's "inconsistency by design" consequence
  is updated to say it was closed by task 19 (or names what remains);
  `docs/BUGS.md` BUG-011's follow-up pointer updated; ledger + this file
  updated.

## Test Plan (draft)

- **Unit:** 19.4 through `withStubbedLookup` (already in
  `internal/core/language_test.go`); handler-level composition tests
  alongside the existing `withLanguageNote` test; brief and playlist logic
  tested with their existing injectable fetch seams (`searchPlaylistEntries`
  already takes a `fetch` func).
- **Manual smoke (live):** 19.6. If YouTube stops rejecting translated
  tracks the smoke test proves nothing and the unit tests are the coverage.
- Commands: `go test ./internal/core/... ./internal/mcpserver/... ./internal/cli/... -v`,
  `go build ./... && go vet ./... && go test ./... && gofmt -l internal/`.

## Out of scope / open

- Naming the spoken language inside the BUG-011 error for an *explicit*
  `en` on a non-English video (Advise suggested it during task 18; needs a
  lookup on the error path and `TranscriptErrorText` has no `ctx`).
  Separate candidate follow-up.
- A cheaper `captionTracks` extraction than the existing
  `ytInitialPlayerRe` scan (~1 s on a real page); worth doing if 19's
  per-video playlist resolution makes it hurt.
