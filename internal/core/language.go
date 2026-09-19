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

// captionTracksResponse mirrors the one path of ytInitialPlayerResponse this
// file reads: captions.playerCaptionsTracklistRenderer.captionTracks.
type captionTracksResponse struct {
	Captions struct {
		Renderer struct {
			CaptionTracks []struct {
				LanguageCode string `json:"languageCode"`
				Kind         string `json:"kind"`
			} `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
}

// resolveDefaultLanguage picks the language to request when the caller
// omitted one, from a ytInitialPlayerResponse JSON payload:
//  1. any English track (manual or ASR) -> "en", today's behavior, so a
//     video with uploaded English subtitles is never switched away from them;
//  2. else the auto-generated (kind "asr") track's language, which is the
//     language actually spoken — the first one if there are several;
//  3. else "en" (no captions at all, unparseable payload), i.e. today's
//     path and today's error.
//
// It never returns "".
func resolveDefaultLanguage(playerResponse []byte) string {
	var pr captionTracksResponse
	if err := json.Unmarshal(playerResponse, &pr); err != nil {
		return "en"
	}
	tracks := pr.Captions.Renderer.CaptionTracks
	for _, t := range tracks {
		if t.LanguageCode == "en" || strings.HasPrefix(t.LanguageCode, "en-") {
			return "en"
		}
	}
	for _, t := range tracks {
		if t.Kind == "asr" && t.LanguageCode != "" {
			return t.LanguageCode
		}
	}
	return "en"
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
