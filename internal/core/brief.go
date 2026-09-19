package core

import (
	"context"
	"math"
	"regexp"
	"strings"
	"sync"
)

// TranscriptStats is the transcript-level summary get_video_brief attaches
// to a timed transcript so an agent doesn't have to estimate these by
// reading the whole thing (docs/tasks/17-video-brief/TASK.md).
type TranscriptStats struct {
	Words              int            // whitespace-split words over speech segments only
	SpeakingRateWPM    int            // Words per minute of covered time (last segment end); 0 if no time covered
	NonSpeechCues      int            // segments whose whole text is one "[...]" token, e.g. [Music]
	NonSpeechBreakdown map[string]int // per lowercased token, e.g. "[music]": 10
	LongestGapSecs     float64        // largest silence between consecutive segments (negatives clamp to 0)
	LongestGapAtSecs   float64        // where that gap starts (end of the segment before it)
}

// nonSpeechCueRe matches a segment whose entire text is a single bracketed
// caption cue such as "[Music]", "[Applause]" or "[inaudible]". A bracket
// token inline with speech ("[inaudible] hello") is not a cue.
var nonSpeechCueRe = regexp.MustCompile(`^\[[^\]]+\]$`)

// computeTranscriptStats derives TranscriptStats from parsed segments.
// Pure; the rules are spelled out in TASK.md 17.3 and pinned by
// TestComputeTranscriptStats.
func computeTranscriptStats(segs []transcriptSegment) TranscriptStats {
	st := TranscriptStats{NonSpeechBreakdown: map[string]int{}}
	var lastEndMs float64
	for i, s := range segs {
		if nonSpeechCueRe.MatchString(s.Text) {
			st.NonSpeechCues++
			st.NonSpeechBreakdown[strings.ToLower(s.Text)]++
		} else {
			st.Words += len(strings.Fields(s.Text))
		}
		end := s.Offset + s.Duration
		if end > lastEndMs {
			lastEndMs = end
		}
		if i+1 < len(segs) {
			// Offsets come from VTT timestamps parsed as float ms and carry
			// float noise (163771.00000000003); round to whole ms before
			// converting so the rendered value is "6.771", not
			// "6.771000000000029".
			if gap := math.Round(segs[i+1].Offset - end); gap > st.LongestGapSecs*1000 {
				st.LongestGapSecs = gap / 1000
				st.LongestGapAtSecs = math.Round(end) / 1000
			}
		}
	}
	if lastEndMs > 0 {
		st.SpeakingRateWPM = int(math.Round(float64(st.Words) / (lastEndMs / 60000)))
	}
	return st
}

// VideoBrief is get_video_brief's result: every section an evaluation
// needs, fetched in one call, with each section's failure reported in its
// own field so the sections that did succeed are still usable. Chapters
// come from the same HTTP fetch as Metadata, so MetadataErr covers both.
type VideoBrief struct {
	Metadata    map[string]string
	Chapters    []Chapter
	MetadataErr error

	TranscriptTimed string
	CaptionKind     CaptionKind
	Stats           TranscriptStats
	TranscriptErr   error

	// Language is the transcript language actually fetched. LanguageAutoDetected
	// is true only when the caller omitted the language and it resolved to
	// something other than "en" (task 19); the MCP output states the language
	// only then.
	Language             string
	LanguageAutoDetected bool
}

// The two network legs of FetchVideoBrief, as variables so tests can check its
// sections fail independently without the network.
var (
	fetchMetadataForBrief   = fetchVideoMetadataAndChapters
	fetchTranscriptForBrief = fetchTranscript
)

// FetchVideoBrief fetches metadata+chapters (one watch-page fetch) and the
// transcript (yt-dlp, cache-backed) concurrently — the same shape as
// SaveTranscriptFile — and never returns a Go error: each section's error
// lands in the corresponding VideoBrief field.
func FetchVideoBrief(ctx context.Context, videoID, language string) VideoBrief {
	var b VideoBrief

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		meta, prChapters, err := fetchMetadataForBrief(ctx, videoID)
		if err != nil {
			b.MetadataErr = err
			return
		}
		b.Metadata = meta
		b.Chapters = chaptersFrom(meta, prChapters)
	}()
	go func() {
		defer wg.Done()
		tr, lang, auto, err := fetchBriefTranscript(ctx, videoID, language)
		if err != nil {
			b.TranscriptErr = err
			return
		}
		b.Language, b.LanguageAutoDetected = lang, auto
		b.TranscriptTimed = transcriptTimed(tr.Segments)
		b.CaptionKind = tr.CaptionKind
		b.Stats = computeTranscriptStats(tr.Segments)
	}()
	wg.Wait()
	return b
}

// fetchBriefTranscript resolves the transcript language and fetches the
// transcript. The resolution (a memoized watch-page lookup when language is
// omitted) deliberately runs here, inside the brief's transcript goroutine,
// not before it: the metadata/chapters goroutine stays fully concurrent, so
// the ~1 s lookup usually hides behind the yt-dlp fetch (task 19). An explicit
// language skips the lookup; a failed lookup keeps "en", i.e. today's behavior.
func fetchBriefTranscript(ctx context.Context, videoID, requested string) (tr parsedTranscript, language string, autoDetected bool, err error) {
	language = ResolveLanguage(ctx, videoID, requested)
	autoDetected = requested == "" && language != "en"
	tr, err = fetchTranscriptForBrief(ctx, videoID, language)
	return tr, language, autoDetected, err
}
