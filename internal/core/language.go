package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

// The default transcript language is "en", but for an auto-caption video
// spoken in another language "en" selects a machine-translated track, which
// YouTube rejects with HTTP 429 (docs/BUGS.md BUG-011). When the caller
// omits `language`, ResolveLanguage picks the language from the watch
// page's captionTracks instead — see docs/tasks/18-native-language-fallback.

// captionTracksResponse mirrors the paths of ytInitialPlayerResponse this file
// reads: captions.playerCaptionsTracklistRenderer.{captionTracks,audioTracks}.
type captionTracksResponse struct {
	Captions struct {
		Renderer struct {
			CaptionTracks []struct {
				LanguageCode string `json:"languageCode"`
				Kind         string `json:"kind"`
			} `json:"captionTracks"`
			AudioTracks []struct {
				AudioTrackID string `json:"audioTrackId"`
			} `json:"audioTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
}

// captionTrack is one caption track; Kind "asr" is auto-generated, "" uploaded.
type captionTrack struct{ Language, Kind string }

// captionInfo is what the resolution reads from a watch page.
type captionInfo struct {
	Tracks   []captionTrack
	AudioIDs []string // audioTracks[].audioTrackId, in page order
}

// parseCaptions extracts captionInfo from a ytInitialPlayerResponse payload;
// nil or garbage yields the zero value.
func parseCaptions(playerResponse []byte) captionInfo {
	var pr captionTracksResponse
	if err := json.Unmarshal(playerResponse, &pr); err != nil {
		return captionInfo{}
	}
	var info captionInfo
	for _, t := range pr.Captions.Renderer.CaptionTracks {
		info.Tracks = append(info.Tracks, captionTrack{Language: t.LanguageCode, Kind: t.Kind})
	}
	for _, a := range pr.Captions.Renderer.AudioTracks {
		info.AudioIDs = append(info.AudioIDs, a.AudioTrackID)
	}
	return info
}

// originalAudioLanguage returns the video's original audio language from the
// audioTrackId list: on videos with YouTube's auto-dubbing structure exactly
// one id ends ".4" ("vi.4", "de-DE.4") and the rest end ".10" (dubbed). Any
// other shape — none, several, or an id with no language — is unknown, and the
// caller falls back to the track-only rules (docs/BUGS.md BUG-013).
func originalAudioLanguage(ids []string) (string, bool) {
	lang, n := "", 0
	for _, id := range ids {
		if strings.HasSuffix(id, ".4") {
			lang = strings.TrimSuffix(id, ".4")
			n++
		}
	}
	if n != 1 || lang == "" {
		return "", false
	}
	return lang, true
}

func baseLanguage(code string) string {
	base, _, _ := strings.Cut(code, "-")
	return base
}

func isEnglishCode(code string) bool {
	return code == "en" || strings.HasPrefix(code, "en-")
}

// isRegionSuffix reports whether s looks like a region ("DE", "BR", "419"),
// as opposed to a custom variant id ("eEY6OEpapPo").
func isRegionSuffix(s string) bool {
	if len(s) == 3 {
		return s[0] >= '0' && s[0] <= '9' && s[1] >= '0' && s[1] <= '9' && s[2] >= '0' && s[2] <= '9'
	}
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

// findTrack finds the track of the wanted kind (asr = auto-generated) for
// language lang, preferring an exact code, then the bare base language or a
// region-shaped variant of it ("de" / "de-DE" / "es-419"), then any other
// same-base track. It returns the track's own code — yt-dlp's --sub-langs is a
// full match — except that English is always the literal "en", so an English
// video's resolved language, filenames and notes stay as they were.
func findTrack(tracks []captionTrack, lang string, asr bool) (string, bool) {
	base := baseLanguage(lang)
	best, bestRank := "", 4
	for _, t := range tracks {
		if (t.Kind == "asr") != asr || t.Language == "" || baseLanguage(t.Language) != base {
			continue
		}
		rank := 3
		switch _, suffix, has := strings.Cut(t.Language, "-"); {
		case t.Language == lang:
			rank = 1
		case !has || isRegionSuffix(suffix):
			rank = 2
		}
		if rank < bestRank {
			best, bestRank = t.Language, rank
		}
	}
	if bestRank == 4 {
		return "", false
	}
	if base == "en" {
		return "en", true
	}
	return best, true
}

// resolveFromCaptions picks the language to request when the caller omitted
// one. When the original audio language O is known (originalAudioLanguage):
//  1. an uploaded track in O -> that track;
//  2. else an uploaded English track -> "en" (the machine-translated auto
//     English of a non-English video is what 429s, BUG-011);
//  3. else the auto-generated track in O (BUG-013: YouTube adds a dubbed auto
//     "en" track to non-English videos, so "any en track" no longer means
//     English speech).
//
// Then, always, and the only path when O is unknown or has no track:
//  4. any English track (manual or ASR) -> "en", today's behavior;
//  5. else the first auto-generated track's language;
//  6. else "en" (no captions at all), i.e. today's path and today's error.
//
// It never returns "".
func resolveFromCaptions(info captionInfo) string {
	if o, ok := originalAudioLanguage(info.AudioIDs); ok {
		if lang, ok := findTrack(info.Tracks, o, false); ok {
			return lang
		}
		for _, t := range info.Tracks {
			if t.Kind != "asr" && isEnglishCode(t.Language) {
				return "en"
			}
		}
		if lang, ok := findTrack(info.Tracks, o, true); ok {
			return lang
		}
	}
	for _, t := range info.Tracks {
		if isEnglishCode(t.Language) {
			return "en"
		}
	}
	for _, t := range info.Tracks {
		if t.Kind == "asr" && t.Language != "" {
			return t.Language
		}
	}
	return "en"
}

// resolveDefaultLanguage is parseCaptions followed by resolveFromCaptions,
// from a ytInitialPlayerResponse JSON payload. It never returns "".
func resolveDefaultLanguage(playerResponse []byte) string {
	return resolveFromCaptions(parseCaptions(playerResponse))
}

// languageMemo remembers the resolved language per video so that only the
// first omitted-language call for a video pays for a watch-page fetch. It is
// bounded by clearing itself when full — a language is stable for a video
// and a re-lookup is one cheap page fetch, so nothing fancier (TTL, LRU) is
// worth its moving parts.
type languageMemo struct {
	mu  sync.Mutex
	cap int
	m   map[string]string
}

func newLanguageMemo(cap int) *languageMemo {
	return &languageMemo{cap: cap, m: make(map[string]string)}
}

func (l *languageMemo) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.m)
}

// resolve returns the memoized language for videoID, calling lookup on a
// miss. A lookup error yields "en" and is not memoized, so a transient
// failure neither blocks a later retry nor surfaces as anything but today's
// behavior. Two concurrent misses may both call lookup; the write is
// idempotent, so that is accepted.
func (l *languageMemo) resolve(videoID string, lookup func() (string, error)) string {
	l.mu.Lock()
	if lang, ok := l.m[videoID]; ok {
		l.mu.Unlock()
		return lang
	}
	l.mu.Unlock()

	lang, err := lookup()
	if err != nil || lang == "" {
		return "en"
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.m) >= l.cap {
		l.m = make(map[string]string)
	}
	l.m[videoID] = lang
	return lang
}

var defaultLanguageMemo = newLanguageMemo(256)

// lookupSpokenLanguage fetches the watch page and resolves the default
// language from it. It is a variable so tests can run ResolveLanguage
// without the network.
var lookupSpokenLanguage = func(ctx context.Context, videoID string) (string, error) {
	html, err := fetchWatchPageHTML(ctx, videoID)
	if err != nil {
		return "", err
	}
	m := ytInitialPlayerRe.FindStringSubmatch(html)
	if m == nil {
		// Not memoized: a page without a player response is more likely a
		// transient oddity than a stable property of the video.
		return "", errors.New("no ytInitialPlayerResponse in watch page")
	}
	raw := m[1]
	if raw == "" {
		raw = m[2]
	}
	return resolveDefaultLanguage([]byte(raw)), nil
}

// ResolveLanguage returns the language to fetch a transcript in. An explicit
// requested language (including "en") is returned untouched with no network
// call — asking for a translation on purpose is honored. When requested is
// empty it returns the video's resolved default (see resolveDefaultLanguage),
// or "en" if the watch page can't be read. Callers that want to tell the
// user the language was auto-detected compare the result to "en".
func ResolveLanguage(ctx context.Context, videoID, requested string) string {
	if requested != "" {
		return requested
	}
	return defaultLanguageMemo.resolve(videoID, func() (string, error) {
		return lookupSpokenLanguage(ctx, videoID)
	})
}

// LanguageNote returns the one-line notice telling the user which language
// was picked for them, or "" when there is nothing to say: the note appears
// only when the caller omitted the language and the auto-detected result
// isn't "en" (the default they would have expected anyway). It is shown
// outside the transcript body by the MCP handlers and CLI.
func LanguageNote(requested, resolved string) string {
	if requested != "" || resolved == "en" {
		return ""
	}
	return "language: " + resolved + " (auto-detected spoken language)"
}

// sanitizeLanguageForFilename makes a language code safe to embed in a
// filename: anything outside [A-Za-z0-9_-] becomes "_". The language is an
// MCP-controlled argument, so without this a value like "../../x" would let
// SaveTranscriptFile write outside the output directory.
func sanitizeLanguageForFilename(language string) string {
	var b strings.Builder
	for _, r := range language {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// transcriptFilename is the saved transcript's file name:
// <title>_<lang>[_timed].md, for every language including "en", so two
// languages of one video never overwrite each other.
func transcriptFilename(safeTitle, language string, timed bool) string {
	name := safeTitle + "_" + sanitizeLanguageForFilename(language)
	if timed {
		name += "_timed"
	}
	return name + ".md"
}

// languageMetaLine is the saved file's header line naming its language, or
// "" for "en" so an English file's body is unchanged.
func languageMetaLine(language string) string {
	if language == "en" {
		return ""
	}
	return "**Language:** " + language
}

// SaveTranscriptFileResolved is the save path with the language handled: it
// resolves an omitted requested language (see ResolveLanguage), saves the
// transcript in that language, and returns the auto-detected-language note
// (see LanguageNote; "" when there is nothing to say) for the caller to show
// outside the file. The MCP download tools and the CLI's `transcript --save`
// both call it, so the resolve/save/note composition lives in one place.
func SaveTranscriptFileResolved(ctx context.Context, videoID, requested, outputDir string, withTimestamps bool) (path, note string, err error) {
	language := ResolveLanguage(ctx, videoID, requested)
	path, err = SaveTranscriptFile(ctx, videoID, language, outputDir, withTimestamps)
	if err != nil {
		return "", "", err
	}
	return path, LanguageNote(requested, language), nil
}
