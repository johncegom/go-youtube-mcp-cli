package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// transcriptCache has no upstream TS equivalent (new in task 11). Ground
// truth is the cache's own specification from
// docs/tasks/11-transcript-cache/TASK.md's Definition of Done: a hit within
// TTL must not re-invoke the fetch function; TTL expiry must trigger a
// re-fetch; the cap must evict the oldest entry and stay bounded; fetch
// errors must never be cached; concurrent access must be race-free.

func countingFetch(segs []transcriptSegment, err error) (func() ([]transcriptSegment, error), *int32) {
	var calls int32
	fetch := func() ([]transcriptSegment, error) {
		atomic.AddInt32(&calls, 1)
		return segs, err
	}
	return fetch, &calls
}

func TestTranscriptCache_HitWithinTTL(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, 32, func() time.Time { return now })
	key := cacheKey{videoID: "abc", language: "en"}
	segs := []transcriptSegment{{Text: "hi", Offset: 0, Duration: 100}}
	fetch, calls := countingFetch(segs, nil)

	for i := 0; i < 3; i++ {
		got, err := c.getOrFetch(key, fetch)
		if err != nil {
			t.Fatalf("getOrFetch() error = %v", err)
		}
		if len(got) != 1 || got[0].Text != "hi" {
			t.Errorf("getOrFetch() = %+v, want %+v", got, segs)
		}
	}
	if n := atomic.LoadInt32(calls); n != 1 {
		t.Errorf("fetch called %d times, want exactly 1 (cache hit should skip it)", n)
	}
}

func TestTranscriptCache_TTLExpiryRefetches(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(1*time.Minute, 32, func() time.Time { return now })
	key := cacheKey{videoID: "abc", language: "en"}
	fetch, calls := countingFetch([]transcriptSegment{{Text: "hi"}}, nil)

	if _, err := c.getOrFetch(key, fetch); err != nil {
		t.Fatalf("getOrFetch() error = %v", err)
	}
	now = now.Add(2 * time.Minute) // past TTL
	if _, err := c.getOrFetch(key, fetch); err != nil {
		t.Fatalf("getOrFetch() error = %v", err)
	}
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("fetch called %d times, want exactly 2 (TTL expiry should trigger re-fetch)", n)
	}
}

func TestTranscriptCache_ErrorNotCached(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, 32, func() time.Time { return now })
	key := cacheKey{videoID: "abc", language: "en"}
	wantErr := errors.New("429 rate limited")
	fetch, calls := countingFetch(nil, wantErr)

	if _, err := c.getOrFetch(key, fetch); !errors.Is(err, wantErr) {
		t.Fatalf("getOrFetch() error = %v, want %v", err, wantErr)
	}
	if _, err := c.getOrFetch(key, fetch); !errors.Is(err, wantErr) {
		t.Fatalf("getOrFetch() error = %v, want %v", err, wantErr)
	}
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("fetch called %d times, want exactly 2 (an error result must never be cached)", n)
	}
}

func TestTranscriptCache_CapEvictsOldest(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, 2, func() time.Time { return now })
	fetchFor := func(id string) func() ([]transcriptSegment, error) {
		return func() ([]transcriptSegment, error) {
			return []transcriptSegment{{Text: id}}, nil
		}
	}

	if _, err := c.getOrFetch(cacheKey{videoID: "v1", language: "en"}, fetchFor("v1")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.getOrFetch(cacheKey{videoID: "v2", language: "en"}, fetchFor("v2")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.getOrFetch(cacheKey{videoID: "v3", language: "en"}, fetchFor("v3")); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	n := len(c.entries)
	_, v1Present := c.entries[cacheKey{videoID: "v1", language: "en"}]
	_, v3Present := c.entries[cacheKey{videoID: "v3", language: "en"}]
	c.mu.Unlock()

	if n != 2 {
		t.Errorf("cache has %d entries, want exactly 2 (cap must stay bounded)", n)
	}
	if v1Present {
		t.Error("oldest entry v1 is still present, want it evicted")
	}
	if !v3Present {
		t.Error("newest entry v3 is missing, want it present")
	}
}

func TestTranscriptCache_ConcurrentAccessSafe(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, 32, func() time.Time { return now })
	key := cacheKey{videoID: "abc", language: "en"}
	fetch, _ := countingFetch([]transcriptSegment{{Text: "hi"}}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.getOrFetch(key, fetch); err != nil {
				t.Errorf("getOrFetch() error = %v", err)
			}
		}()
	}
	wg.Wait()
}
