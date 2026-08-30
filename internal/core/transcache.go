package core

import (
	"sync"
	"time"
)

// cacheKey identifies one cached transcript by video + language.
type cacheKey struct {
	videoID  string
	language string
}

type cacheEntry struct {
	segments []transcriptSegment
	expires  time.Time
}

// transcriptCache is an in-memory, TTL/cap-bounded cache of parsed
// transcript segments, keyed by video ID + language. It exists so a
// multi-call agent workflow on one video (get_transcript, then
// search_transcript, then get_transcript_timed, ...) shells out to yt-dlp
// once instead of once per call — see docs/tasks/11-transcript-cache/TASK.md.
// Disk-backed persistence is deliberately out of scope; this only covers
// within-session reuse.
type transcriptCache struct {
	mu      sync.Mutex
	entries map[cacheKey]cacheEntry
	order   []cacheKey // insertion order, oldest first, for eviction
	ttl     time.Duration
	cap     int
	now     func() time.Time
}

func newTranscriptCache(ttl time.Duration, cap int, now func() time.Time) *transcriptCache {
	if cap < 0 {
		cap = 0
	}
	return &transcriptCache{
		entries: make(map[cacheKey]cacheEntry),
		ttl:     ttl,
		cap:     cap,
		now:     now,
	}
}

// defaultCache is the process-wide transcript cache used by fetchSegments:
// 15-minute TTL, 32-video cap.
var defaultCache = newTranscriptCache(15*time.Minute, 32, time.Now)

// getOrFetch returns the cached segments for key if present and unexpired;
// otherwise it calls fetch, stores the result on success, and returns it.
// fetch errors are returned as-is and never cached, so a transient failure
// (e.g. a 429) doesn't poison the cache for the rest of its TTL. Duplicate
// concurrent fetches for the same key on a cache miss are possible but
// harmless — the cache itself is never left in a corrupted state.
func (c *transcriptCache) getOrFetch(key cacheKey, fetch func() ([]transcriptSegment, error)) ([]transcriptSegment, error) {
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok && c.now().Before(entry.expires) {
		c.mu.Unlock()
		return entry.segments, nil
	}
	c.mu.Unlock()

	segments, err := fetch()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.set(key, segments)
	return segments, nil
}

// set stores segments for key, evicting the oldest entry if over capacity.
// Callers must hold c.mu.
func (c *transcriptCache) set(key cacheKey, segments []transcriptSegment) {
	if _, existed := c.entries[key]; !existed {
		c.order = append(c.order, key)
	}
	c.entries[key] = cacheEntry{segments: segments, expires: c.now().Add(c.ttl)}
	for len(c.entries) > c.cap {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}
