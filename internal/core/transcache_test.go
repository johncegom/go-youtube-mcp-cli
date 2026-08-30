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

func TestTranscriptCache_ZeroCapNeverCaches(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, 0, func() time.Time { return now })
	key := cacheKey{videoID: "abc", language: "en"}
	fetch, calls := countingFetch([]transcriptSegment{{Text: "hi"}}, nil)

	if _, err := c.getOrFetch(key, fetch); err != nil {
		t.Fatalf("getOrFetch() error = %v", err)
	}
	if _, err := c.getOrFetch(key, fetch); err != nil {
		t.Fatalf("getOrFetch() error = %v", err)
	}
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("fetch called %d times, want exactly 2 (cap=0 means every entry is evicted immediately, so nothing is ever cached)", n)
	}
	c.mu.Lock()
	n := len(c.entries)
	c.mu.Unlock()
	if n != 0 {
		t.Errorf("cache has %d entries, want 0 with cap=0", n)
	}
}

// TestTranscriptCache_NegativeCapPanics documents docs/BUGS.md BUG-005: with
// a negative cap, set()'s eviction loop condition (len(c.entries) > c.cap)
// stays true even at zero entries, so it indexes an empty c.order slice and
// panics. cap is only ever 32 via defaultCache today, so this isn't
// reachable through any current public code path — this test pins the gap
// rather than leaving it silently undiscovered. Flip this assertion once
// BUG-005 is fixed.
func TestTranscriptCache_NegativeCapPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected set() to panic with a negative cap (BUG-005); if this no longer panics, BUG-005 has been fixed and this test should be updated")
		}
	}()
	now := time.Now()
	c := newTranscriptCache(15*time.Minute, -1, func() time.Time { return now })
	fetch, _ := countingFetch([]transcriptSegment{{Text: "hi"}}, nil)
	_, _ = c.getOrFetch(cacheKey{videoID: "v1", language: "en"}, fetch)
}

// TestTranscriptCache_ReAddingExistingKeyDoesNotReorder pins down that
// set()'s eviction order (c.order) is by first-insertion time, not
// last-write time: re-setting an already-present key does not move it to
// the end of c.order, so a freshly-refreshed entry can still be the next
// one evicted. This is FIFO-by-first-insertion, not LRU — documented
// behavior worth pinning explicitly since it's easy to assume otherwise.
func TestTranscriptCache_ReAddingExistingKeyDoesNotReorder(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(1*time.Minute, 2, func() time.Time { return now })
	v1, v2, v3 := cacheKey{videoID: "v1"}, cacheKey{videoID: "v2"}, cacheKey{videoID: "v3"}
	fetchFor := func(id string) func() ([]transcriptSegment, error) {
		return func() ([]transcriptSegment, error) { return []transcriptSegment{{Text: id}}, nil }
	}

	if _, err := c.getOrFetch(v1, fetchFor("v1")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.getOrFetch(v2, fetchFor("v2")); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute) // expire both, but don't evict yet
	if _, err := c.getOrFetch(v1, fetchFor("v1-refreshed")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.getOrFetch(v3, fetchFor("v3")); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	_, v1Present := c.entries[v1]
	_, v2Present := c.entries[v2]
	_, v3Present := c.entries[v3]
	total := len(c.entries)
	c.mu.Unlock()

	// v1 was refreshed most recently but never moved within c.order, so it's
	// still the "oldest" entry by insertion position and is the one evicted
	// when v3 pushes the cache over cap=2 — v2 (never touched again) survives.
	if v1Present {
		t.Error("v1 is present, want it evicted despite being the most recently refreshed entry (FIFO-by-first-insertion, not LRU)")
	}
	if !v2Present {
		t.Error("v2 is missing, want it present (it was never re-set, so it's not the oldest by insertion order)")
	}
	if !v3Present {
		t.Error("v3 is missing, want it present")
	}
	if total != 2 {
		t.Errorf("cache has %d entries, want exactly 2 (cap=2)", total)
	}
}

// TestTranscriptCache_ExpiredEntryStillCountsTowardCap pins down that TTL
// expiry and the cap are independent: an expired-but-not-yet-evicted entry
// still occupies a slot and counts toward c.cap until something actually
// triggers eviction (a cache miss for a *different* key).
func TestTranscriptCache_ExpiredEntryStillCountsTowardCap(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache(1*time.Minute, 1, func() time.Time { return now })
	v1, v2 := cacheKey{videoID: "v1"}, cacheKey{videoID: "v2"}
	fetchFor := func(id string) func() ([]transcriptSegment, error) {
		return func() ([]transcriptSegment, error) { return []transcriptSegment{{Text: id}}, nil }
	}

	if _, err := c.getOrFetch(v1, fetchFor("v1")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute) // v1 now expired, but still present in c.entries

	c.mu.Lock()
	n := len(c.entries)
	c.mu.Unlock()
	if n != 1 {
		t.Fatalf("expired entry was already removed before any new fetch; got %d entries, want 1", n)
	}

	if _, err := c.getOrFetch(v2, fetchFor("v2")); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	_, v1Present := c.entries[v1]
	_, v2Present := c.entries[v2]
	total := len(c.entries)
	c.mu.Unlock()

	if v1Present {
		t.Error("v1 is present, want it evicted once v2's insertion pushed the cache over cap")
	}
	if !v2Present {
		t.Error("v2 is missing, want it present")
	}
	if total != 1 {
		t.Errorf("cache has %d entries, want exactly 1 (cap=1)", total)
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
