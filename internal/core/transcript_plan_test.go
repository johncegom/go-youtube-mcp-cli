package core

import (
	"reflect"
	"testing"
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
