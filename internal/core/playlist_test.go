package core

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Playlist ID shapes below are real YouTube playlist ID conventions:
//   - "PL..." — user-created playlists (e.g. from a playlist share link).
//   - "UU..." — a channel's uploads playlist (UC<channelID> with UU swapped in).
//   - "OLAK5uy_..." — auto-generated "album" playlists (e.g. from a music release).
//   - "RD...", "FL...", "LL..." — YouTube-generated mix/favorites/liked-videos playlists.
func TestExtractPlaylistID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bare PL id", "PLBCF2DAC6FFB574DE", "PLBCF2DAC6FFB574DE"},
		{"bare UU id", "UU_x5XG1OV2P6uZZ5FSM9Ttw", "UU_x5XG1OV2P6uZZ5FSM9Ttw"},
		{"bare OLAK5uy id", "OLAK5uy_lZYnO6nDCd9paxOwPRQZuvfN5CmT9uk", "OLAK5uy_lZYnO6nDCd9paxOwPRQZuvfN5CmT9uk"},
		{"playlist url", "https://www.youtube.com/playlist?list=PLBCF2DAC6FFB574DE", "PLBCF2DAC6FFB574DE"},
		{"playlist url no www", "https://youtube.com/playlist?list=PLBCF2DAC6FFB574DE", "PLBCF2DAC6FFB574DE"},
		{"watch url with list", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLBCF2DAC6FFB574DE", "PLBCF2DAC6FFB574DE"},
		{"watch url with list and index", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLBCF2DAC6FFB574DE&index=3", "PLBCF2DAC6FFB574DE"},
		{"video-only watch url", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ""},
		{"bare video id", "dQw4w9WgXcQ", ""},
		{"invalid id", "not-a-valid-playlist-id!", ""},
		{"unrelated url", "https://example.com/playlist?list=PLBCF2DAC6FFB574DE", ""},
		{"empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractPlaylistID(tc.input)
			if got != tc.want {
				t.Errorf("ExtractPlaylistID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// parseFlatPlaylistOutput fixture: captured verbatim (id/title text
// replaced with representative stand-ins, tab-separation and line shape
// preserved) from a real
// `yt-dlp --flat-playlist --print "%(id)s\t%(title)s"` run against a small
// public YouTube playlist. Includes a unicode title and a trailing blank
// line (yt-dlp's real output ends with one), plus injected malformed lines
// (no tab, empty id) that must be skipped rather than crashing the parser.
const flatPlaylistFixture = "dQw4w9WgXcQ\tRick Astley - Never Gonna Give You Up\n" +
	"9bZkp7q19f0\tPSY - GANGNAM STYLE(강남스타일) M/V\n" +
	"\tTitle with no id\n" +
	"justanid-no-title\n" +
	"jNQXAC9IVRw\tMe at the zoo\n" +
	"\n"

func TestParseFlatPlaylistOutput(t *testing.T) {
	got := parseFlatPlaylistOutput(flatPlaylistFixture)
	want := []playlistEntry{
		{VideoID: "dQw4w9WgXcQ", Title: "Rick Astley - Never Gonna Give You Up"},
		{VideoID: "9bZkp7q19f0", Title: "PSY - GANGNAM STYLE(강남스타일) M/V"},
		{VideoID: "jNQXAC9IVRw", Title: "Me at the zoo"},
	}
	if len(got) != len(want) {
		t.Fatalf("parseFlatPlaylistOutput() returned %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseFlatPlaylistOutput_Empty(t *testing.T) {
	if got := parseFlatPlaylistOutput(""); len(got) != 0 {
		t.Errorf("parseFlatPlaylistOutput(\"\") = %+v, want empty", got)
	}
}

func seg(text string, offsetSecs, durSecs float64) transcriptSegment {
	return transcriptSegment{Text: text, Offset: offsetSecs * 1000, Duration: durSecs * 1000}
}

// TestSearchPlaylistEntries_PartialFailure drives searchPlaylistEntries
// with a fake fetch where the second of three videos errors, and asserts
// the loop still processes the third video (DoD 15.5: one bad video never
// aborts the whole search) and records the failure with a
// TranscriptErrorText-classified reason.
func TestSearchPlaylistEntries_PartialFailure(t *testing.T) {
	entries := []playlistEntry{
		{VideoID: "v1", Title: "Video One"},
		{VideoID: "v2", Title: "Video Two"},
		{VideoID: "v3", Title: "Video Three"},
	}
	fetch := func(_ context.Context, videoID, _ string) ([]transcriptSegment, bool, error) {
		if videoID == "v2" {
			return nil, true, errors.New("no transcript available: captions disabled")
		}
		return []transcriptSegment{seg("hello world", 0, 2)}, true, nil
	}

	results, skipped := searchPlaylistEntries(context.Background(), entries, "hello", "en", fetch, func(time.Duration) {})

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (v1, v3): %+v", len(results), results)
	}
	if results[0].VideoID != "v1" || results[1].VideoID != "v3" {
		t.Errorf("results in wrong order/videos: %+v", results)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "Video Two") {
		t.Errorf("skipped = %+v, want one entry mentioning Video Two", skipped)
	}
}

// TestSearchPlaylistEntries_DelayOnlyOnMiss verifies sequential ordering
// and that sleep is called once per cache-miss video except never before
// the first fetch (DoD 15.6) — no real timing, just counting sleep calls
// against an injected clock.
func TestSearchPlaylistEntries_DelayOnlyOnMiss(t *testing.T) {
	entries := []playlistEntry{
		{VideoID: "v1", Title: "One"},
		{VideoID: "v2", Title: "Two"},
		{VideoID: "v3", Title: "Three"},
	}
	var fetchOrder []string
	misses := map[string]bool{"v1": true, "v2": false, "v3": true}
	fetch := func(_ context.Context, videoID, _ string) ([]transcriptSegment, bool, error) {
		fetchOrder = append(fetchOrder, videoID)
		return []transcriptSegment{seg("nothing matches", 0, 2)}, misses[videoID], nil
	}
	sleepCalls := 0
	sleep := func(d time.Duration) {
		sleepCalls++
		if d != playlistSearchDelay {
			t.Errorf("sleep called with %v, want %v", d, playlistSearchDelay)
		}
	}

	searchPlaylistEntries(context.Background(), entries, "xyz", "en", fetch, sleep)

	wantOrder := []string{"v1", "v2", "v3"}
	if len(fetchOrder) != len(wantOrder) {
		t.Fatalf("fetchOrder = %v, want %v", fetchOrder, wantOrder)
	}
	for i := range wantOrder {
		if fetchOrder[i] != wantOrder[i] {
			t.Errorf("fetchOrder[%d] = %q, want %q", i, fetchOrder[i], wantOrder[i])
		}
	}
	// v1 is a miss -> sleep before v2's fetch. v2 is a hit -> no sleep
	// before v3's fetch. So exactly 1 sleep call total.
	if sleepCalls != 1 {
		t.Errorf("sleepCalls = %d, want 1", sleepCalls)
	}
}

func TestFormatPlaylistSearchResult_NoMatches(t *testing.T) {
	got := formatPlaylistSearchResult("nothing", nil, nil, 3, 3)
	want := `No matches found for "nothing" in this playlist.`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatPlaylistSearchResult_MatchesGroupedByVideo(t *testing.T) {
	results := []videoSearchResult{
		{
			VideoID:    "v1",
			Title:      "Intro to Go",
			MatchCount: 1,
			Blocks:     [][]transcriptSegment{{seg("go is great", 5, 2)}},
		},
		{
			VideoID:    "v2",
			Title:      "Advanced Go",
			MatchCount: 1,
			Blocks:     [][]transcriptSegment{{seg("go generics", 65, 2)}},
		},
	}
	got := formatPlaylistSearchResult("go", results, nil, 2, 2)

	want := "Found 2 match(es) for \"go\" across 2 video(s):\n" +
		"\nIntro to Go [" + FormatTimestamp(5) + "] go is great\n" +
		"\nAdvanced Go [" + FormatTimestamp(65) + "] go generics"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatPlaylistSearchResult_SkippedAndCapNote(t *testing.T) {
	got := formatPlaylistSearchResult("go", nil, []string{"Bad Video (v9): No transcript available for video v9. The video may not have captions."}, 25, 40)
	if !strings.Contains(got, "(showing first 25 of 40 videos)") {
		t.Errorf("missing cap note: %q", got)
	}
	if !strings.Contains(got, "Skipped 1 video(s):") || !strings.Contains(got, "Bad Video (v9)") {
		t.Errorf("missing skipped section: %q", got)
	}
}

// FuzzExtractPlaylistID mirrors FuzzExtractVideoID's invariant: never
// panics, and any non-empty result always matches the playlist-ID regex
// used internally — never a partial match or unvalidated URL syntax
// leaking through, since the result gets embedded directly into a
// yt-dlp subprocess argument elsewhere in the package.
func FuzzExtractPlaylistID(f *testing.F) {
	seeds := []string{
		"PLBCF2DAC6FFB574DE",
		"UU_x5XG1OV2P6uZZ5FSM9Ttw",
		"OLAK5uy_lZYnO6nDCd9paxOwPRQZuvfN5CmT9uk",
		"https://www.youtube.com/playlist?list=PLBCF2DAC6FFB574DE",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLBCF2DAC6FFB574DE",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"dQw4w9WgXcQ",
		"not-a-valid-playlist-id!",
		"",
		"://not a url at all",
		"youtube.com",
		"https://youtube.com/",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := ExtractPlaylistID(input)
		if got != "" && !playlistIDRe.MatchString(got) {
			t.Errorf("ExtractPlaylistID(%q) = %q, which is not a valid playlist ID", input, got)
		}
	})
}
