package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	f.Add(playerResponseWithAudio(fxTracksVi, fxAudioVi))
	f.Add([]byte(`{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[{"languageCode":"vi","kind":"asr"}],"audioTracks":[{"audioTrackId":"vi.4"},{"audioTrackId":`))
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

// ── task 19: save filename/header, brief transcript ──────────────────────

// The filename rule is the human's decision (docs/tasks/19-language-
// resolution-remaining-tools/TASK.md): every save is named
// <title>_<lang>[_timed].md, including "en". The language comes from an
// MCP-controlled argument, so it is restricted to [A-Za-z0-9_-]. There is no
// upstream equivalent; ground truth is that spec.
func TestSanitizeLanguageForFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"en", "en"},
		{"vi", "vi"},
		{"zh-Hans", "zh-Hans"},
		{"pt_BR", "pt_BR"},
		{"../../x", "______x"},
		{`a/b\c`, "a_b_c"},
		{"en US", "en_US"},
		{"vi\x00", "vi_"},
	}
	for _, tc := range cases {
		if got := sanitizeLanguageForFilename(tc.in); got != tc.want {
			t.Errorf("sanitizeLanguageForFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTranscriptFilename(t *testing.T) {
	cases := []struct {
		safeTitle, language string
		timed               bool
		want                string
	}{
		{"My_Video", "en", false, "My_Video_en.md"},
		{"My_Video", "en", true, "My_Video_en_timed.md"},
		{"My_Video", "vi", false, "My_Video_vi.md"},
		{"My_Video", "vi", true, "My_Video_vi_timed.md"},
		{"My_Video", "../../x", false, "My_Video_______x.md"},
	}
	for _, tc := range cases {
		got := transcriptFilename(tc.safeTitle, tc.language, tc.timed)
		if got != tc.want {
			t.Errorf("transcriptFilename(%q, %q, %v) = %q, want %q", tc.safeTitle, tc.language, tc.timed, got, tc.want)
		}
		if strings.ContainsAny(got, `/\`) {
			t.Errorf("transcriptFilename(%q, %q, %v) = %q contains a path separator", tc.safeTitle, tc.language, tc.timed, got)
		}
	}
}

// A saved file gets a "**Language:**" header line only when the language
// used is not "en" (auto-detected or explicit), so an English file's body is
// byte-identical to before task 19.
func TestLanguageMetaLine(t *testing.T) {
	if got := languageMetaLine("en"); got != "" {
		t.Errorf("languageMetaLine(en) = %q, want empty", got)
	}
	if got, want := languageMetaLine("vi"), "**Language:** vi"; got != want {
		t.Errorf("languageMetaLine(vi) = %q, want %q", got, want)
	}
}

// fetchBriefTranscript resolves the language itself (it runs inside the
// brief's transcript goroutine, task 19): an omitted language is resolved,
// an explicit one is untouched with no lookup, a failed lookup keeps "en".
// The transcript cache is seeded so no yt-dlp run happens.
func seedTranscript(t *testing.T, videoID, language, text string) {
	t.Helper()
	key := cacheKey{videoID: videoID, language: language}
	defaultCache.mu.Lock()
	defaultCache.set(key, parsedTranscript{Segments: []transcriptSegment{{Text: text, Offset: 0, Duration: 1000}}})
	defaultCache.mu.Unlock()
	t.Cleanup(func() {
		defaultCache.mu.Lock()
		delete(defaultCache.entries, key)
		defaultCache.mu.Unlock()
	})
}

func TestFetchBriefTranscript(t *testing.T) {
	var calls int32
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "vi", nil
	})
	const videoID = "task19brief"
	seedTranscript(t, videoID, "vi", "xin chào")
	seedTranscript(t, videoID, "en", "hello")
	ctx := context.Background()

	tr, lang, auto, err := fetchBriefTranscript(ctx, videoID, "")
	if err != nil || lang != "vi" || !auto || tr.Segments[0].Text != "xin chào" {
		t.Errorf("omitted: got (%q, lang=%q, auto=%v, err=%v), want (xin chào, vi, true, nil)", tr.Segments[0].Text, lang, auto, err)
	}

	before := atomic.LoadInt32(&calls)
	tr, lang, auto, err = fetchBriefTranscript(ctx, videoID, "en")
	if err != nil || lang != "en" || auto || tr.Segments[0].Text != "hello" {
		t.Errorf("explicit en: got (%q, lang=%q, auto=%v, err=%v), want (hello, en, false, nil)", tr.Segments[0].Text, lang, auto, err)
	}
	tr, lang, auto, err = fetchBriefTranscript(ctx, videoID, "vi")
	if err != nil || lang != "vi" || auto || tr.Segments[0].Text != "xin chào" {
		t.Errorf("explicit vi: got (%q, lang=%q, auto=%v, err=%v), want (xin chào, vi, false, nil)", tr.Segments[0].Text, lang, auto, err)
	}
	if n := atomic.LoadInt32(&calls); n != before {
		t.Errorf("lookup called %d more times for explicit languages, want 0", n-before)
	}
}

func TestFetchBriefTranscript_LookupFailureKeepsEn(t *testing.T) {
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
		return "", errors.New("page fetch failed")
	})
	const videoID = "task19brieffail"
	seedTranscript(t, videoID, "en", "hello")
	tr, lang, auto, err := fetchBriefTranscript(context.Background(), videoID, "")
	if err != nil || lang != "en" || auto || tr.Segments[0].Text != "hello" {
		t.Errorf("got (%q, lang=%q, auto=%v, err=%v), want (hello, en, false, nil)", tr.Segments[0].Text, lang, auto, err)
	}
}

// ── task 19: the save path end to end, no network ────────────────────────
//
// SaveTranscriptFileResolved is resolve -> save -> note in one place (the
// MCP handler and the CLI both call it). The transcript cache is seeded, the
// language lookup stubbed, and the metadata fetch stubbed, so the whole path
// runs hermetically against a temp directory.

func withStubbedMetadata(t *testing.T) {
	t.Helper()
	old := fetchMetadataForSave
	fetchMetadataForSave = func(ctx context.Context, videoID string) (map[string]string, error) {
		return map[string]string{"title": "My Video", "channel": "C"}, nil
	}
	t.Cleanup(func() { fetchMetadataForSave = old })
}

func TestSaveTranscriptFileResolved(t *testing.T) {
	cases := []struct {
		name         string
		requested    string
		lookupLang   string
		lookupErr    error
		wantFile     string
		wantNote     string
		wantLangLine bool
		wantLookups  int32
	}{
		{"omitted, resolves to vi", "", "vi", nil, "My Video_vi.md", "language: vi (auto-detected spoken language)", true, 1},
		{"omitted, resolves to en", "", "en", nil, "My Video_en.md", "", false, 1},
		{"explicit vi never looks up", "vi", "ja", nil, "My Video_vi.md", "", true, 0},
		{"explicit en never looks up", "en", "vi", nil, "My Video_en.md", "", false, 0},
		{"lookup error keeps en, no note", "", "", errors.New("page fetch failed"), "My Video_en.md", "", false, 1},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var lookups int32
			withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
				atomic.AddInt32(&lookups, 1)
				return tc.lookupLang, tc.lookupErr
			})
			withStubbedMetadata(t)
			videoID := "task19save" + string(rune('a'+i))
			for _, lang := range []string{"en", "vi", "ja"} {
				seedTranscript(t, videoID, lang, "hello "+lang)
			}
			dir := t.TempDir()

			path, note, err := SaveTranscriptFileResolved(context.Background(), videoID, tc.requested, dir, false)
			if err != nil {
				t.Fatalf("SaveTranscriptFileResolved() error = %v", err)
			}
			if got := filepath.Base(path); got != tc.wantFile {
				t.Errorf("file = %q, want %q", got, tc.wantFile)
			}
			if note != tc.wantNote {
				t.Errorf("note = %q, want %q", note, tc.wantNote)
			}
			if n := atomic.LoadInt32(&lookups); n != tc.wantLookups {
				t.Errorf("lookups = %d, want %d", n, tc.wantLookups)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(body), "**Language:**"); got != tc.wantLangLine {
				t.Errorf("header has **Language:** line = %v, want %v\n%s", got, tc.wantLangLine, body)
			}
		})
	}
}

// A hostile language argument must never place the file outside outputDir.
func TestSaveTranscriptFileResolved_HostileLanguageStaysInsideOutputDir(t *testing.T) {
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) { return "en", nil })
	withStubbedMetadata(t)
	const videoID = "task19hostile"
	seedTranscript(t, videoID, "../../x", "hello")
	dir := t.TempDir()

	path, _, err := SaveTranscriptFileResolved(context.Background(), videoID, "../../x", dir, true)
	if err != nil {
		t.Fatalf("SaveTranscriptFileResolved() error = %v", err)
	}
	if got := filepath.Dir(path); got != dir {
		t.Errorf("file written in %q, want exactly %q", got, dir)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("saved file missing: %v", err)
	}
}

// Brief, omitted language whose lookup succeeds and returns "en": nothing to
// announce (task 19, clause "omitted+en -> none").
func TestFetchBriefTranscript_OmittedResolvesToEnIsSilent(t *testing.T) {
	withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) { return "en", nil })
	const videoID = "task19briefen"
	seedTranscript(t, videoID, "en", "hello")
	tr, lang, auto, err := fetchBriefTranscript(context.Background(), videoID, "")
	if err != nil || lang != "en" || auto || tr.Segments[0].Text != "hello" {
		t.Errorf("got (%q, lang=%q, auto=%v, err=%v), want (hello, en, false, nil)", tr.Segments[0].Text, lang, auto, err)
	}
}

// FetchVideoBrief's sections fail independently (docs/tasks/17-video-brief):
// the metadata/chapters goroutine and the transcript goroutine write
// disjoint fields, and the language resolution now runs inside the
// transcript goroutine (task 19), so neither a failed resolve nor a failed
// transcript may disturb the metadata section, and vice versa. Both network
// legs are stubbed through fetchMetadataForBrief / fetchTranscriptForBrief.
func withStubbedBrief(t *testing.T, meta func() (map[string]string, []Chapter, error), tr func() (parsedTranscript, error)) {
	t.Helper()
	oldMeta, oldTr := fetchMetadataForBrief, fetchTranscriptForBrief
	fetchMetadataForBrief = func(ctx context.Context, videoID string) (map[string]string, []Chapter, error) { return meta() }
	fetchTranscriptForBrief = func(ctx context.Context, videoID, language string) (parsedTranscript, error) { return tr() }
	t.Cleanup(func() { fetchMetadataForBrief, fetchTranscriptForBrief = oldMeta, oldTr })
}

func TestFetchVideoBrief_SectionsFailIndependently(t *testing.T) {
	okMeta := func() (map[string]string, []Chapter, error) {
		return map[string]string{"title": "T", "description": "0:00 Intro\n0:20 Middle\n0:40 End"}, nil, nil
	}
	okTranscript := func() (parsedTranscript, error) {
		return parsedTranscript{Segments: []transcriptSegment{{Text: "hello world", Offset: 0, Duration: 1000}}}, nil
	}

	t.Run("resolve fails and transcript fails: metadata intact", func(t *testing.T) {
		withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) { return "", errors.New("page fetch failed") })
		withStubbedBrief(t, okMeta, func() (parsedTranscript, error) { return parsedTranscript{}, errors.New("yt-dlp failed") })
		b := FetchVideoBrief(context.Background(), "vid", "")
		if b.TranscriptErr == nil {
			t.Error("TranscriptErr = nil, want the transcript failure")
		}
		if b.MetadataErr != nil || b.Metadata["title"] != "T" || len(b.Chapters) != 3 {
			t.Errorf("metadata section disturbed: err=%v meta=%v chapters=%d", b.MetadataErr, b.Metadata, len(b.Chapters))
		}
		if b.Language != "" || b.LanguageAutoDetected {
			t.Errorf("language fields set on a failed transcript: %q, %v", b.Language, b.LanguageAutoDetected)
		}
	})

	t.Run("metadata fails: transcript and language intact", func(t *testing.T) {
		withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) { return "vi", nil })
		withStubbedBrief(t, func() (map[string]string, []Chapter, error) { return nil, nil, errors.New("page fetch failed") }, okTranscript)
		b := FetchVideoBrief(context.Background(), "vid", "")
		if b.MetadataErr == nil {
			t.Error("MetadataErr = nil, want the metadata failure")
		}
		if b.TranscriptErr != nil || b.TranscriptTimed == "" {
			t.Errorf("transcript section disturbed: err=%v timed=%q", b.TranscriptErr, b.TranscriptTimed)
		}
		if b.Language != "vi" || !b.LanguageAutoDetected {
			t.Errorf("language = %q auto=%v, want vi/true", b.Language, b.LanguageAutoDetected)
		}
	})

	t.Run("explicit language: not auto-detected, lookup never called", func(t *testing.T) {
		var lookups int32
		withStubbedLookup(t, func(ctx context.Context, videoID string) (string, error) {
			atomic.AddInt32(&lookups, 1)
			return "ja", nil
		})
		withStubbedBrief(t, okMeta, okTranscript)
		b := FetchVideoBrief(context.Background(), "vid", "vi")
		if b.Language != "vi" || b.LanguageAutoDetected || atomic.LoadInt32(&lookups) != 0 {
			t.Errorf("language=%q auto=%v lookups=%d, want vi/false/0", b.Language, b.LanguageAutoDetected, lookups)
		}
	})
}
