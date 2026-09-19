package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// resolveDefaultLanguage has no upstream TS equivalent (task 18). Ground
// truth for its fixtures is the real `captionTracks` array from each
// video's live watch page (fetched 2026-09-20; the long signed "baseUrl"
// fields are stripped, everything else is verbatim):
//
//	r8CppXSqVDU  Vietnamese speech, auto-captions only
//	BqRhBq-_kgE  English speech, auto-captions only
//	dQw4w9WgXcQ  uploaded en/de-DE/ja/pt-BR/es-419 tracks + en ASR
//	BIAUJLWoN4k  Japanese speech, uploaded en track + ja ASR
//
// The expected values for those four are confirmed by live yt-dlp runs
// (docs/tasks/18-native-language-fallback/TASK.md 18.0): `en` on the
// Vietnamese/Korean ASR-only videos 429s while the native language works,
// and `en` on BIAUJLWoN4k returns the real uploaded English subtitles.
// The "no captions key" / "empty captionTracks" payloads are structural
// (no such live page was captured) and only pin the fall-through to "en".

const (
	tracksViASROnly     = `[{"name":{"simpleText":"Vietnamese (auto-generated)"},"vssId":"a.vi","languageCode":"vi","kind":"asr","isTranslatable":true,"trackName":""}]`
	tracksEnASROnly     = `[{"name":{"simpleText":"English (auto-generated)"},"vssId":"a.en","languageCode":"en","kind":"asr","isTranslatable":true,"trackName":""}]`
	tracksManualEn      = `[{"name":{"simpleText":"English"},"vssId":".en","languageCode":"en","isTranslatable":true,"trackName":""},{"name":{"simpleText":"English (auto-generated)"},"vssId":"a.en","languageCode":"en","kind":"asr","isTranslatable":true,"trackName":""},{"name":{"simpleText":"German (Germany)"},"vssId":".de-DE","languageCode":"de-DE","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Japanese"},"vssId":".ja","languageCode":"ja","isTranslatable":true,"trackName":""}]`
	tracksManualEnJaASR = `[{"name":{"simpleText":"English"},"vssId":".en","languageCode":"en","isTranslatable":true,"trackName":""},{"name":{"simpleText":"Japanese (auto-generated)"},"vssId":"a.ja","languageCode":"ja","kind":"asr","isTranslatable":true,"trackName":""}]`
)

func playerResponseWithTracks(tracksJSON string) []byte {
	return []byte(`{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":` + tracksJSON + `}}}`)
}

func TestResolveDefaultLanguage(t *testing.T) {
	cases := []struct {
		name string
		pr   []byte
		want string
	}{
		{"vietnamese ASR only -> vi (r8CppXSqVDU)", playerResponseWithTracks(tracksViASROnly), "vi"},
		{"english ASR only -> en (BqRhBq-_kgE)", playerResponseWithTracks(tracksEnASROnly), "en"},
		{"uploaded en + en ASR -> en (dQw4w9WgXcQ)", playerResponseWithTracks(tracksManualEn), "en"},
		{"uploaded en + ja ASR -> en, not ja (BIAUJLWoN4k)", playerResponseWithTracks(tracksManualEnJaASR), "en"},
		{"en-US only counts as English", playerResponseWithTracks(`[{"languageCode":"en-US","vssId":".en-US"},{"languageCode":"ko","kind":"asr"}]`), "en"},
		{"a non-en code that merely starts with 'en' does not count", playerResponseWithTracks(`[{"languageCode":"enm","vssId":".enm"},{"languageCode":"ko","kind":"asr"}]`), "ko"},
		{"first ASR track wins when several", playerResponseWithTracks(`[{"languageCode":"vi","kind":"asr"},{"languageCode":"ko","kind":"asr"}]`), "vi"},
		{"uploaded non-en tracks only, no ASR -> en", playerResponseWithTracks(`[{"languageCode":"de","vssId":".de"}]`), "en"},
		{"empty captionTracks -> en", playerResponseWithTracks(`[]`), "en"},
		{"no captions key at all -> en", []byte(`{"videoDetails":{"videoId":"x"}}`), "en"},
		{"not JSON -> en", []byte(`not json`), "en"},
		{"nil -> en", nil, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveDefaultLanguage(tc.pr); got != tc.want {
				t.Errorf("resolveDefaultLanguage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func FuzzResolveDefaultLanguage(f *testing.F) {
	f.Add(playerResponseWithTracks(tracksViASROnly))
	f.Add([]byte(`{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[{"languageCode":`))
	f.Add([]byte(``))
	f.Fuzz(func(t *testing.T, pr []byte) {
		if got := resolveDefaultLanguage(pr); got == "" {
			t.Errorf("resolveDefaultLanguage(%q) = \"\", want a non-empty language", pr)
		}
	})
}

func countingLookup(lang string, err error) (func() (string, error), *int32) {
	var calls int32
	return func() (string, error) {
		atomic.AddInt32(&calls, 1)
		return lang, err
	}, &calls
}

func TestLanguageMemo_HitSkipsLookup(t *testing.T) {
	m := newLanguageMemo(8)
	lookup, calls := countingLookup("vi", nil)
	for i := 0; i < 3; i++ {
		if got := m.resolve("vid1", lookup); got != "vi" {
			t.Fatalf("resolve() = %q, want vi", got)
		}
	}
	if n := atomic.LoadInt32(calls); n != 1 {
		t.Errorf("lookup called %d times, want exactly 1 (memo hit should skip it)", n)
	}
}

func TestLanguageMemo_LookupErrorFallsBackToEnAndIsNotMemoized(t *testing.T) {
	m := newLanguageMemo(8)
	failing, failCalls := countingLookup("", errors.New("page fetch failed"))
	if got := m.resolve("vid1", failing); got != "en" {
		t.Errorf("resolve() on lookup error = %q, want en", got)
	}
	m.resolve("vid1", failing)
	if n := atomic.LoadInt32(failCalls); n != 2 {
		t.Errorf("failing lookup called %d times, want 2 (errors must not be memoized)", n)
	}
	ok, _ := countingLookup("vi", nil)
	if got := m.resolve("vid1", ok); got != "vi" {
		t.Errorf("resolve() after a recovered lookup = %q, want vi", got)
	}
}

func TestLanguageMemo_IsBounded(t *testing.T) {
	m := newLanguageMemo(2)
	lookup, _ := countingLookup("vi", nil)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		m.resolve(id, lookup)
	}
	if got := m.size(); got > 2 {
		t.Errorf("memo size = %d, want <= 2", got)
	}
}

// withStubbedLookup swaps the package-level page lookup and memo for the
// duration of a test so ResolveLanguage runs with no network.
func withStubbedLookup(t *testing.T, lookup func(ctx context.Context, videoID string) (string, error)) {
	t.Helper()
	oldLookup, oldMemo := lookupSpokenLanguage, defaultLanguageMemo
	lookupSpokenLanguage, defaultLanguageMemo = lookup, newLanguageMemo(8)
	t.Cleanup(func() { lookupSpokenLanguage, defaultLanguageMemo = oldLookup, oldMemo })
}

func TestResolveLanguage(t *testing.T) {
	var calls int32
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "vi", nil
	})

	if got := ResolveLanguage(context.Background(), "vid1", "en"); got != "en" {
		t.Errorf("explicit en = %q, want en (an explicit language is always honored)", got)
	}
	if got := ResolveLanguage(context.Background(), "vid1", "ja"); got != "ja" {
		t.Errorf("explicit ja = %q, want ja", got)
	}
	if n := atomic.LoadInt32(&calls); n != 0 {
		t.Fatalf("lookup called %d times for explicit languages, want 0", n)
	}
	if got := ResolveLanguage(context.Background(), "vid1", ""); got != "vi" {
		t.Errorf("omitted language = %q, want vi", got)
	}
	if got := ResolveLanguage(context.Background(), "vid1", ""); got != "vi" {
		t.Errorf("omitted language, second call = %q, want vi", got)
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Errorf("lookup called %d times for two omitted-language calls, want 1 (memo)", n)
	}
}

func TestResolveLanguage_LookupErrorKeepsEn(t *testing.T) {
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
		return "", errors.New("network down")
	})
	if got := ResolveLanguage(context.Background(), "vid1", ""); got != "en" {
		t.Errorf("omitted language with failing lookup = %q, want en", got)
	}
}

func TestLanguageNote(t *testing.T) {
	cases := []struct {
		name              string
		requested, actual string
		want              string
	}{
		{"auto-detected non-en shows the note", "", "vi", "language: vi (auto-detected spoken language)"},
		{"auto-detected en stays silent", "", "en", ""},
		{"explicit non-en request stays silent", "vi", "vi", ""},
		{"explicit en request stays silent", "en", "en", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LanguageNote(tc.requested, tc.actual); got != tc.want {
				t.Errorf("LanguageNote(%q, %q) = %q, want %q", tc.requested, tc.actual, got, tc.want)
			}
		})
	}
}

// TestResolveLanguage_SharesCacheEntryWithExplicitLanguage: an omitted-
// language call resolves to "vi" and must land on the same transcript-cache
// entry an explicit language:"vi" call uses, so the second call never
// shells out to yt-dlp (task 18, 18.4). A cache miss here would run yt-dlp
// against a fake video ID and fail, so the seeded entry is the only way
// both calls can succeed.
func TestResolveLanguage_SharesCacheEntryWithExplicitLanguage(t *testing.T) {
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) { return "vi", nil })

	const videoID = "task18cache"
	key := cacheKey{videoID: videoID, language: "vi"}
	defaultCache.mu.Lock()
	defaultCache.set(key, parsedTranscript{Segments: []transcriptSegment{{Text: "xin chào", Offset: 0, Duration: 1000}}})
	defaultCache.mu.Unlock()
	t.Cleanup(func() {
		defaultCache.mu.Lock()
		delete(defaultCache.entries, key)
		defaultCache.mu.Unlock()
	})

	ctx := context.Background()
	auto, err := GetTranscriptText(ctx, videoID, ResolveLanguage(ctx, videoID, ""))
	if err != nil {
		t.Fatalf("omitted-language call: %v", err)
	}
	explicit, err := GetTranscriptText(ctx, videoID, ResolveLanguage(ctx, videoID, "vi"))
	if err != nil {
		t.Fatalf("explicit vi call: %v", err)
	}
	if auto != "xin chào" || explicit != auto {
		t.Errorf("auto = %q, explicit = %q, want both %q", auto, explicit, "xin chào")
	}
}
