package core

import (
	"strings"
	"testing"
)

// Task 21 (BUG-013): resolve the default language from the video's original
// audio language, taken from the `.4` audioTrackId. Real-page fixtures are in
// language_fixtures_test.go; the synthetic cases below pin the precedence
// (docs/tasks/21-original-language-resolution/TASK.md).

func playerResponseWithAudio(tracksJSON string, audioIDs []string) []byte {
	var ids []string
	for _, id := range audioIDs {
		ids = append(ids, `{"audioTrackId":"`+id+`"}`)
	}
	return []byte(`{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":` + tracksJSON +
		`,"audioTracks":[` + strings.Join(ids, ",") + `]}}}`)
}

func upTrack(codes ...string) []captionTrack {
	var ts []captionTrack
	for _, c := range codes {
		ts = append(ts, captionTrack{Language: c})
	}
	return ts
}

func autoTrack(codes ...string) []captionTrack {
	var ts []captionTrack
	for _, c := range codes {
		ts = append(ts, captionTrack{Language: c, Kind: "asr"})
	}
	return ts
}

func TestResolveDefaultLanguage_RealStructuredPages(t *testing.T) {
	cases := []struct {
		name   string
		tracks string
		audio  []string
		want   string
	}{
		{"vietnamese video with dubbed auto en-US -> vi (r8CppXSqVDU)", fxTracksVi, fxAudioVi, "vi"},
		{"german video with dubbed auto en-US -> de, not de-DE (cZSgL76ddDs)", fxTracksDe, fxAudioDe, "de"},
		{"english video, 21 auto tracks, .4 is not first -> en (vyIgAO8aCbA)", fxTracksEn21, fxAudioEn21, "en"},
		{"r8CppXSqVDU tracks without audioTracks -> en (today's rule)", fxTracksVi, nil, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveDefaultLanguage(playerResponseWithAudio(tc.tracks, tc.audio)); got != tc.want {
				t.Errorf("resolveDefaultLanguage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOriginalAudioLanguage(t *testing.T) {
	cases := []struct {
		name   string
		ids    []string
		want   string
		wantOK bool
	}{
		{"real vi", fxAudioVi, "vi", true},
		{"real de-DE", fxAudioDe, "de-DE", true},
		{".4 in the middle, not first", fxAudioEn21, "en-US", true},
		{"none", nil, "", false},
		{"only dubbed", []string{"vi.10", "en-US.10"}, "", false},
		{"two .4 ids", []string{"vi.4", "de-DE.4", "en-US.10"}, "", false},
		{"empty id", []string{""}, "", false},
		{"x", []string{"x"}, "", false},
		{"bare .4", []string{".4"}, "", false},
		{"bare .4 beside a real one is still ambiguous", []string{".4", "vi.4"}, "", false},
		{".40 is not .4", []string{"vi.40"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := originalAudioLanguage(tc.ids)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("originalAudioLanguage(%q) = (%q, %v), want (%q, %v)", tc.ids, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestFindTrack(t *testing.T) {
	cases := []struct {
		name   string
		tracks []captionTrack
		lang   string
		asr    bool
		want   string
		wantOK bool
	}{
		{"exact vi", autoTrack("en-US", "vi"), "vi", true, "vi", true},
		{"de-DE audio -> auto de", autoTrack("en-US", "de"), "de-DE", true, "de", true},
		{"uploaded de-DE for de-DE", upTrack("de-DE"), "de-DE", false, "de-DE", true},
		{"exact beats region-shaped", upTrack("pt-BR", "pt-PT", "pt"), "pt-PT", false, "pt-PT", true},
		{"region-shaped beats custom-named", upTrack("vi-abc123", "vi-VN"), "vi", false, "vi-VN", true},
		{"bare base beats custom-named", upTrack("vi-abc123", "vi"), "vi-VN", false, "vi", true},
		{"numeric region es-419 is region-shaped", upTrack("es-abc123", "es-419"), "es", false, "es-419", true},
		{"custom-named as a last resort", upTrack("vi-abc123"), "vi", false, "vi-abc123", true},
		{"kind is respected: uploaded wanted, only auto exists", autoTrack("vi"), "vi", false, "", false},
		{"kind is respected: auto wanted, only uploaded exists", upTrack("vi"), "vi", true, "", false},
		{"different base does not match", autoTrack("vie"), "vi", true, "", false},
		{"english stays en: uploaded en-US", upTrack("en-US"), "en-US", false, "en", true},
		{"english stays en: auto en for en-US", autoTrack("en"), "en-US", true, "en", true},
		{"english stays en: auto en-GB for en", autoTrack("en-GB"), "en", true, "en", true},
		{"no tracks", nil, "vi", true, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := findTrack(tc.tracks, tc.lang, tc.asr)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("findTrack(%v, %q, asr=%v) = (%q, %v), want (%q, %v)", tc.tracks, tc.lang, tc.asr, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func concatTracks(groups ...[]captionTrack) []captionTrack {
	var out []captionTrack
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func TestResolveFromCaptions(t *testing.T) {
	cases := []struct {
		name string
		info captionInfo
		want string
	}{
		// step 2 beats step 3
		{".4=vi, uploaded en only, auto vi -> en",
			captionInfo{Tracks: concatTracks(autoTrack("en-US"), upTrack("en"), autoTrack("vi")), AudioIDs: []string{"vi.4", "en-US.10"}}, "en"},
		// step 1 beats step 2 (carries a .4 id: O is never reached any other way)
		{".4=vi, uploaded vi + uploaded en -> vi",
			captionInfo{Tracks: concatTracks(upTrack("en", "vi"), autoTrack("vi")), AudioIDs: []string{"en-US.10", "vi.4"}}, "vi"},
		// step 3
		{".4=vi, neither uploaded -> auto vi (BUG-013)",
			captionInfo{Tracks: autoTrack("en-US", "vi"), AudioIDs: []string{"vi.4", "en-US.10"}}, "vi"},
		{".4=de-DE, auto de -> de",
			captionInfo{Tracks: autoTrack("en-US", "de"), AudioIDs: []string{"de-DE.4", "en-US.10"}}, "de"},
		{".4=de-DE, uploaded de-DE + uploaded en -> de-DE",
			captionInfo{Tracks: concatTracks(upTrack("en", "de-DE"), autoTrack("de")), AudioIDs: []string{"de-DE.4"}}, "de-DE"},
		{"custom-named uploaded vi-abc123 loses to region-shaped uploaded vi-VN",
			captionInfo{Tracks: upTrack("vi-abc123", "vi-VN"), AudioIDs: []string{"vi.4"}}, "vi-VN"},
		// step 4: today's rules
		{".4=vi, no vi track at all -> today's rule (en)",
			captionInfo{Tracks: autoTrack("en-US"), AudioIDs: []string{"vi.4"}}, "en"},
		{".4=vi, no vi track, auto ko -> today's rule (ko)",
			captionInfo{Tracks: autoTrack("ko"), AudioIDs: []string{"vi.4"}}, "ko"},
		{"two .4 ids -> today's rule (en)",
			captionInfo{Tracks: autoTrack("en-US", "vi"), AudioIDs: []string{"vi.4", "de-DE.4"}}, "en"},
		{"no audio ids -> today's rule (en)",
			captionInfo{Tracks: autoTrack("en-US", "vi")}, "en"},
		// scope pin: no .4 + single auto track + uploaded O + uploaded en -> en, as today
		{"no .4, uploaded vi + uploaded en + auto vi -> en (scope pin)",
			captionInfo{Tracks: concatTracks(upTrack("vi", "en"), autoTrack("vi"))}, "en"},
		{"malformed ids only, same tracks -> en",
			captionInfo{Tracks: concatTracks(upTrack("vi", "en"), autoTrack("vi")), AudioIDs: []string{"", "x", ".4"}}, "en"},
		// English stays the literal "en"
		{"uploaded en-US + auto en, .4=en-US -> en",
			captionInfo{Tracks: concatTracks(upTrack("en-US"), autoTrack("en")), AudioIDs: []string{"en-US.4", "vi.10"}}, "en"},
		{".4=en-US, auto en only -> en",
			captionInfo{Tracks: autoTrack("en"), AudioIDs: []string{"en-US.4"}}, "en"},
		{".4=en-US, auto en-US only -> en",
			captionInfo{Tracks: autoTrack("en-US"), AudioIDs: []string{"en-US.4"}}, "en"},
		{"zero value -> en", captionInfo{}, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveFromCaptions(tc.info); got != tc.want {
				t.Errorf("resolveFromCaptions() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCaptions(t *testing.T) {
	got := parseCaptions(playerResponseWithAudio(fxTracksVi, fxAudioVi))
	wantTracks := autoTrack("en-US", "vi")
	if len(got.Tracks) != 2 || got.Tracks[0] != wantTracks[0] || got.Tracks[1] != wantTracks[1] ||
		strings.Join(got.AudioIDs, ",") != strings.Join(fxAudioVi, ",") {
		t.Errorf("parseCaptions() = %+v, want tracks %v and audio %v", got, wantTracks, fxAudioVi)
	}
	for _, bad := range [][]byte{nil, []byte(`not json`), []byte(`{}`)} {
		if got := parseCaptions(bad); len(got.Tracks) != 0 || len(got.AudioIDs) != 0 {
			t.Errorf("parseCaptions(%q) = %+v, want zero value", bad, got)
		}
	}
}
