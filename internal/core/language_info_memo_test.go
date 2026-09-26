package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// Task 20.2: the captionInfo memo beside the language memo. The property that
// matters is "one page fetch": ResolveLanguage and the fetch funnel's plan step
// share one lookup, and the funnel never triggers one itself.

// withStubbedCaptionInfo swaps the page lookup and both memos so the default
// lookupSpokenLanguage runs end to end with no network, counting lookups.
func withStubbedCaptionInfo(t *testing.T, info captionInfo, err error) *int32 {
	t.Helper()
	var calls int32
	oldLookup, oldInfo, oldLang := lookupCaptionInfo, defaultInfoMemo, defaultLanguageMemo
	lookupCaptionInfo = func(ctx context.Context, videoID string) (captionInfo, error) {
		atomic.AddInt32(&calls, 1)
		return info, err
	}
	defaultInfoMemo, defaultLanguageMemo = newInfoMemo(8), newLanguageMemo(8)
	t.Cleanup(func() { lookupCaptionInfo, defaultInfoMemo, defaultLanguageMemo = oldLookup, oldInfo, oldLang })
	return &calls
}

func TestCaptionInfoMemo_OneLookupForResolveAndPlan(t *testing.T) {
	info := parseCaptions(playerResponseWithAudio(fxTracksVi, fxAudioVi))
	calls := withStubbedCaptionInfo(t, info, nil)

	if got := ResolveLanguage(context.Background(), "vid1", ""); got != "vi" {
		t.Fatalf("ResolveLanguage() = %q, want vi", got)
	}
	got, ok := peekCaptionInfo("vid1")
	if !ok || len(got.Tracks) != 2 || len(got.AudioIDs) != 2 {
		t.Fatalf("peekCaptionInfo() = (%+v, %v), want the warmed info", got, ok)
	}
	if again, ok := captionInfoFor(context.Background(), "vid1"); !ok || len(again.Tracks) != 2 {
		t.Fatalf("captionInfoFor() = (%+v, %v)", again, ok)
	}
	if n := atomic.LoadInt32(calls); n != 1 {
		t.Errorf("page lookups = %d, want exactly 1 across ResolveLanguage + peek + captionInfoFor", n)
	}
}

func TestPeekCaptionInfo_ColdMakesNoLookup(t *testing.T) {
	calls := withStubbedCaptionInfo(t, captionInfo{}, nil)
	if _, ok := peekCaptionInfo("cold"); ok {
		t.Error("peekCaptionInfo on a cold memo reported ok")
	}
	if n := atomic.LoadInt32(calls); n != 0 {
		t.Errorf("peek made %d lookups, want 0 (memo read only, never the network)", n)
	}
}

func TestCaptionInfoFor_LookupErrorIsNotMemoizedAndLanguageStaysEn(t *testing.T) {
	calls := withStubbedCaptionInfo(t, captionInfo{}, errors.New("page fetch failed"))
	if _, ok := captionInfoFor(context.Background(), "vid1"); ok {
		t.Error("captionInfoFor reported ok on a lookup error")
	}
	if got := ResolveLanguage(context.Background(), "vid1", ""); got != "en" {
		t.Errorf("ResolveLanguage() on a lookup error = %q, want en", got)
	}
	if _, ok := peekCaptionInfo("vid1"); ok {
		t.Error("an error was memoized")
	}
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("lookups = %d, want 2 (errors are retried, not memoized)", n)
	}
}

func TestInfoMemo_IsBounded(t *testing.T) {
	m := newInfoMemo(2)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		m.put(id, captionInfo{})
	}
	if got := m.size(); got > 2 {
		t.Errorf("info memo size = %d, want <= 2", got)
	}
}
