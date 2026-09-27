package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// Task 20.1: the fetch plan. Ground truth is the BUG-012 evidence
// (docs/evidence/bug-012/): for auto-caption-only English videos plain `en`
// 429s while `en-orig` works; with an uploaded track, `en-orig` is the AUTO
// track and must not replace it; pagecodes-results.txt: an `asr` track with
// exactly code L on the page <=> yt-dlp lists `L-orig` (0 mismatches / 17).
// Restricted to English `L` after 20.0b (docs/evidence/bug-012/
// orig_nonenglish-results.txt): plain vi/de never 429 and equal `-orig`.

func TestHasUploadedTrack(t *testing.T) {
	cases := []struct {
		name   string
		tracks []captionTrack
		lang   string
		want   bool
	}{
		{"exact uploaded", upTrack("en"), "en", true},
		{"variant-coded uploaded counts (UF8uR6Z6KLc)", upTrack("en-eEY6OEpapPo"), "en", true},
		{"region uploaded counts", upTrack("en-US"), "en", true},
		{"auto only does not count", autoTrack("en"), "en", false},
		{"other language uploaded does not count", upTrack("de"), "en", false},
		{"a code that merely starts with the language does not count", upTrack("enm"), "en", false},
		{"no tracks", nil, "en", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasUploadedTrack(tc.tracks, tc.lang); got != tc.want {
				t.Errorf("hasUploadedTrack(%v, %q) = %v, want %v", tc.tracks, tc.lang, got, tc.want)
			}
		})
	}
}

func TestHasASRTrack(t *testing.T) {
	cases := []struct {
		name   string
		tracks []captionTrack
		lang   string
		want   bool
	}{
		{"exact auto", autoTrack("en"), "en", true},
		{"en-US auto is not exactly en (r8CppXSqVDU shape)", autoTrack("en-US", "vi"), "en", false},
		{"uploaded does not count", upTrack("en"), "en", false},
		{"no tracks", nil, "en", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasASRTrack(tc.tracks, tc.lang); got != tc.want {
				t.Errorf("hasASRTrack(%v, %q) = %v, want %v", tc.tracks, tc.lang, got, tc.want)
			}
		})
	}
}

func TestFetchPlan(t *testing.T) {
	info := func(ts ...[]captionTrack) captionInfo { return captionInfo{Tracks: concatTracks(ts...)} }
	cases := []struct {
		name string
		info captionInfo
		warm bool
		lang string
		want []string
	}{
		{"cold memo -> [L] even with an asr en track", info(autoTrack("en")), false, "en", []string{"en"}},
		{"known, zero tracks", captionInfo{}, true, "en", []string{"en"}},
		{"only auto en-US (r8CppXSqVDU shape)", info(autoTrack("en-US", "vi")), true, "en", []string{"en"}},
		{"uploaded exact en (dQw4w9WgXcQ shape)", info(upTrack("en"), autoTrack("en")), true, "en", []string{"en"}},
		{"uploaded only under a variant code (UF8uR6Z6KLc shape)", info(upTrack("en-eEY6OEpapPo"), autoTrack("en")), true, "en", []string{"en"}},
		{"auto exact en, no uploaded -> orig first", info(autoTrack("en")), true, "en", []string{"en-orig", "en"}},
		{"auto exact en AND uploaded -> [L]", info(autoTrack("en"), upTrack("en")), true, "en", []string{"en"}},
		{"L already -orig", info(autoTrack("en")), true, "en-orig", []string{"en-orig"}},
		// restricted to English L (20.0b)
		{"vi with an exact auto vi track -> [L] (restricted)", info(autoTrack("en-US", "vi")), true, "vi", []string{"vi"}},
		{"de with an exact auto de track -> [L] (restricted)", info(autoTrack("en-US", "de")), true, "de", []string{"de"}},
		{"de-DE with only auto de -> [L]", info(autoTrack("de")), true, "de-DE", []string{"de-DE"}},
		{"de with uploaded de + auto de -> [L]", info(upTrack("de"), autoTrack("de")), true, "de", []string{"de"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fetchPlan(tc.info, tc.warm, tc.lang); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fetchPlan(warm=%v, %q) = %v, want %v", tc.warm, tc.lang, got, tc.want)
			}
		})
	}
}

// Real pages (language_fixtures_test.go), through parseCaptions.
func TestFetchPlan_RealPages(t *testing.T) {
	cases := []struct {
		name   string
		tracks string
		audio  []string
		lang   string
		want   []string
	}{
		{"vyIgAO8aCbA: 21 auto tracks incl. exact en -> orig first", fxTracksEn21, fxAudioEn21, "en", []string{"en-orig", "en"}},
		{"r8CppXSqVDU: en-US + vi, L=vi -> [vi]", fxTracksVi, fxAudioVi, "vi", []string{"vi"}},
		{"r8CppXSqVDU: explicit en, no exact auto en -> [en]", fxTracksVi, fxAudioVi, "en", []string{"en"}},
		{"cZSgL76ddDs: L=de -> [de]", fxTracksDe, fxAudioDe, "de", []string{"de"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := parseCaptions(playerResponseWithAudio(tc.tracks, tc.audio))
			if got := fetchPlan(info, true, tc.lang); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fetchPlan(%q) = %v, want %v", tc.lang, got, tc.want)
			}
		})
	}
}

// fetchWithPlan replaces fetchWithOrigRetry; the ported TestFetchWithOrigRetry
// (transcript_orig_retry_test.go) pins PR #36's behaviour with a one-element plan.
func TestFetchWithPlan(t *testing.T) {
	want := parsedTranscript{Segments: []transcriptSegment{{Text: "hello"}}}
	other := parsedTranscript{Segments: []transcriptSegment{{Text: "other"}}}

	t.Run("orig succeeds first: one call", func(t *testing.T) {
		var calls []string
		got, err := fetchWithPlan([]string{"en-orig", "en"}, func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			return want, nil
		})
		if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(calls, []string{"en-orig"}) {
			t.Fatalf("got (%+v, %v) calls %v", got, err, calls)
		}
	})

	t.Run("orig fails, plain succeeds", func(t *testing.T) {
		var calls []string
		got, err := fetchWithPlan([]string{"en-orig", "en"}, func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			if lang == "en-orig" {
				return parsedTranscript{}, errors.New(`no transcript available for video x. The video may not have captions in language "en-orig"`)
			}
			return other, nil
		})
		if err != nil || !reflect.DeepEqual(got, other) || !reflect.DeepEqual(calls, []string{"en-orig", "en"}) {
			t.Fatalf("got (%+v, %v) calls %v", got, err, calls)
		}
	})

	t.Run("both fail: the requested language's error, and -orig is never attempted twice", func(t *testing.T) {
		origErr := errors.New(`no transcript available for video x. The video may not have captions in language "en-orig"`)
		var calls []string
		_, err := fetchWithPlan([]string{"en-orig", "en"}, func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			if lang == "en-orig" {
				return parsedTranscript{}, origErr
			}
			return parsedTranscript{}, errRateLimited
		})
		if err != errRateLimited {
			t.Errorf("err = %v, want the requested language's rate-limited error", err)
		}
		if !reflect.DeepEqual(calls, []string{"en-orig", "en"}) {
			t.Errorf("calls = %v, want [en-orig en] (no repeat of en-orig after the 429)", calls)
		}
	})

	t.Run("single-language plan keeps the shipped retry", func(t *testing.T) {
		var calls []string
		got, err := fetchWithPlan([]string{"en"}, func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			if lang == "en" {
				return parsedTranscript{}, errRateLimited
			}
			return want, nil
		})
		if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(calls, []string{"en", "en-orig"}) {
			t.Fatalf("got (%+v, %v) calls %v", got, err, calls)
		}
	})
}

// 20.4: the wiring. A success from "L-orig" is cached under what the caller
// asked for ({videoID, L}); a cold memo keeps today's [L] + retry.
func withStubbedFetchOnce(t *testing.T, videoID string, fn func(lang string) (parsedTranscript, error)) *[]string {
	t.Helper()
	var calls []string
	oldFetch, oldInfo := fetchOnce, defaultInfoMemo
	fetchOnce = func(ctx context.Context, id, lang string) (parsedTranscript, error) {
		calls = append(calls, lang)
		return fn(lang)
	}
	defaultInfoMemo = newInfoMemo(8)
	key := cacheKey{videoID: videoID, language: "en"}
	t.Cleanup(func() {
		fetchOnce, defaultInfoMemo = oldFetch, oldInfo
		defaultCache.mu.Lock()
		delete(defaultCache.entries, key)
		defaultCache.mu.Unlock()
	})
	return &calls
}

func TestFetchTranscript_OrigSuccessIsCachedUnderRequestedLanguage(t *testing.T) {
	const vid = "planWarmVid"
	want := parsedTranscript{Segments: []transcriptSegment{{Text: "genuine"}}}
	calls := withStubbedFetchOnce(t, vid, func(lang string) (parsedTranscript, error) {
		if lang == "en-orig" {
			return want, nil
		}
		return parsedTranscript{}, errRateLimited
	})
	defaultInfoMemo.put(vid, captionInfo{Tracks: autoTrack("en")})

	for i := 0; i < 2; i++ {
		got, err := fetchTranscript(context.Background(), vid, "en")
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("call %d: got (%+v, %v), want (%+v, nil)", i, got, err, want)
		}
	}
	if !reflect.DeepEqual(*calls, []string{"en-orig"}) {
		t.Errorf("fetch attempts = %v, want exactly [en-orig] (planned first, second call served from cache under {vid, en})", *calls)
	}
}

func TestFetchTranscript_ColdMemoKeepsPlainThenRetry(t *testing.T) {
	const vid = "planColdVid"
	want := parsedTranscript{Segments: []transcriptSegment{{Text: "genuine"}}}
	calls := withStubbedFetchOnce(t, vid, func(lang string) (parsedTranscript, error) {
		if lang == "en" {
			return parsedTranscript{}, errRateLimited
		}
		return want, nil
	})
	got, err := fetchTranscript(context.Background(), vid, "en")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got (%+v, %v), want (%+v, nil)", got, err, want)
	}
	if !reflect.DeepEqual(*calls, []string{"en", "en-orig"}) {
		t.Errorf("fetch attempts = %v, want [en en-orig]", *calls)
	}
	if _, ok := peekCaptionInfo(vid); ok {
		t.Error("the fetch path warmed the memo; it must never make or trigger a page lookup")
	}
}

// 20.5: observability. Formats are the specification (same style as
// TestFormatOrigRetryLog). Fixture URLs are real yt-dlp 2026.07.04 -v lines
// (2026-09-27) with the signed values replaced by X: plain `en` on
// vyIgAO8aCbA SUCCEEDED as a back-translation (lang=uk&tlang=en), and
// `en-orig` is the genuine track.
const (
	debugLineTranslated = `[debug] Invoking http downloader on "https://www.youtube.com/api/timedtext?v=vyIgAO8aCbA&ei=X&caps=asr&opi=X&xoaf=X&xowf=1&xospf=1&hl=en&ip=X&ipbits=X&expire=X&sparams=X&signature=X&key=yt8&kind=asr&lang=uk&variant=timing-optimized&tlang=en&fmt=vtt"`
	debugLineGenuine    = `[debug] Invoking http downloader on "https://www.youtube.com/api/timedtext?v=vyIgAO8aCbA&ei=X&caps=asr&opi=X&hl=en&expire=X&signature=X&key=yt8&kind=asr&lang=en-orig&variant=timing-optimized&fmt=vtt"`
)

func TestTimedtextTranslation(t *testing.T) {
	cases := []struct {
		name       string
		output     string
		wantSource string
		wantTlang  string
	}{
		{"back-translation (real line)", "[info] Writing video subtitles\n" + debugLineTranslated + "\n[download] done", "uk", "en"},
		{"genuine -orig track has no tlang", debugLineGenuine, "", ""},
		{"first translated URL wins among several", debugLineGenuine + "\n" + debugLineTranslated, "uk", "en"},
		{"no timedtext line", "[info] nothing here", "", ""},
		{"empty", "", "", ""},
		{"non-timedtext URL with tlang is ignored", `[debug] Invoking http downloader on "https://example.com/x?lang=uk&tlang=en"`, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src, tl := timedtextTranslation(tc.output)
			if src != tc.wantSource || tl != tc.wantTlang {
				t.Errorf("timedtextTranslation() = (%q, %q), want (%q, %q)", src, tl, tc.wantSource, tc.wantTlang)
			}
		})
	}
}

func TestFormatTranslatedLog(t *testing.T) {
	if got, want := formatTranslatedLog("en", "uk", "en"), "lang=en source=uk tlang=en"; got != want {
		t.Errorf("formatTranslatedLog() = %q, want %q", got, want)
	}
}

func TestFormatPlanLog(t *testing.T) {
	cases := []struct {
		name      string
		plan      []string
		succeeded string
		elapsed   time.Duration
		want      string
	}{
		{"orig first succeeded", []string{"en-orig", "en"}, "en-orig", 3212 * time.Millisecond, "lang=en plan=en-orig,en outcome=en-orig duration=3.212s"},
		{"fell back to plain", []string{"en-orig", "en"}, "en", 9 * time.Second, "lang=en plan=en-orig,en outcome=en duration=9s"},
		{"all failed", []string{"en-orig", "en"}, "", 6500 * time.Millisecond, "lang=en plan=en-orig,en outcome=failed duration=6.5s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatPlanLog("en", tc.plan, tc.succeeded, tc.elapsed); got != tc.want {
				t.Errorf("formatPlanLog() = %q, want %q", got, tc.want)
			}
		})
	}
}
