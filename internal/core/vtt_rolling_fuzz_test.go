package core

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Fuzz targets for the BUG-010 rolling-cue handling in parseVtt. The input is
// network-sourced (yt-dlp output for arbitrary videos), effectively
// untrusted, so besides "never panics" (FuzzParseVtt) these pin the
// properties the fix promises: clean output, no carry-block duplicates, and
// — the strongest guarantee — that anything that is not a rolling-cue file
// parses exactly as it did before the fix.

// fuzzSeeds are the real captured heads (testdata/, task 17's dQw4w9WgXcQ
// head) plus the uploaded-caption fixtures and a few degenerate shapes.
func fuzzSeeds(t testing.TB) []string {
	t.Helper()
	return []string{
		mustRead(t, "testdata/asr_rolling_en_head.vtt"),
		mustRead(t, "testdata/asr_rolling_vi_head.vtt"),
		vttAutoHeadFixture,
		vttUploadedHeadFixture,
		vttUploadedHeadFixture2,
		vttFixture,
		"",
		"WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHello<00:00:00.500><c> world</c>",
		"00:00:00.000 --> 00:00:01.000\n<00:00:00.500>\n\n00:00:01.000 --> 00:00:01.010\n \n",
		"00:00:00.000 --> 00:00:01.000\r\nA<00:00:00.500>\r\nB\r\n\r\n00:00:01.000 --> 00:00:01.010\r\nB\r\n",
		"--> --> -->\n\n\n\n",
	}
}

// legacyParseVtt is a verbatim copy of parseVtt as it was before the
// BUG-010 fix (commit c482bd8, internal/core/transcript.go). It is the
// oracle for "uploaded captions are unchanged".
func legacyParseVtt(content string) []transcriptSegment {
	var segments []transcriptSegment
	seen := map[string]struct{}{}

	blocks := blankLineRe.Split(content, -1)
	for _, block := range blocks {
		lines := newlineRe.Split(block, -1)

		timingIdx := -1
		for i, l := range lines {
			if strings.Contains(l, " --> ") {
				timingIdx = i
				break
			}
		}
		if timingIdx == -1 {
			continue
		}

		m := vttTimingRe.FindStringSubmatch(lines[timingIdx])
		if m == nil {
			continue
		}
		offset := parseVttTime(m[1])
		end := parseVttTime(m[2])

		text := strings.Join(lines[timingIdx+1:], " ")
		text = vttTagRe.ReplaceAllString(text, "")
		text = vttEntityReplacer.Replace(text)
		text = whitespaceRe.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}
		key := strconv.FormatFloat(offset, 'f', -1, 64) + "|" + text
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		segments = append(segments, transcriptSegment{Text: text, Offset: offset, Duration: end - offset})
	}

	return segments
}

// Whatever the input, the output is well-formed: cue text is non-empty,
// trimmed, with no control-whitespace or doubled spaces left behind by the
// cleaning; offsets and durations are finite; there can't be more segments
// than timing lines; and a rolling parse never has two consecutive segments
// with the same text (the carry-block duplication BUG-010 describes).
func FuzzParseVtt_Invariants(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, content string) {
		segs := parseVtt(content)
		if max := strings.Count(content, " --> "); len(segs) > max {
			t.Fatalf("%d segments from %d timing lines", len(segs), max)
		}
		for i, s := range segs {
			if s.Text == "" || s.Text != strings.TrimSpace(s.Text) {
				t.Fatalf("segment %d text %q is empty or untrimmed", i, s.Text)
			}
			if strings.ContainsAny(s.Text, "\r\n\t") || strings.Contains(s.Text, "  ") {
				t.Fatalf("segment %d text %q has leftover whitespace", i, s.Text)
			}
			if math.IsNaN(s.Offset) || math.IsInf(s.Offset, 0) || math.IsNaN(s.Duration) || math.IsInf(s.Duration, 0) {
				t.Fatalf("segment %d has a non-finite time: %+v", i, s)
			}
		}
		if detectCaptionKind(content) == CaptionAuto {
			for i := 1; i < len(segs); i++ {
				if segs[i].Text == segs[i-1].Text {
					t.Fatalf("rolling parse: segments %d and %d both %q", i-1, i, segs[i].Text)
				}
			}
		}
	})
}

// Differential: for any input that is NOT a rolling-cue file
// (detectCaptionKind != CaptionAuto) parseVtt must equal the pre-fix parser
// byte for byte. This is the guarantee that uploaded captions and the
// upstream-ported ground truth are untouched by BUG-010's fix.
func FuzzParseVtt_NonRollingMatchesLegacy(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, content string) {
		if detectCaptionKind(content) == CaptionAuto {
			return
		}
		if got, want := parseVtt(content), legacyParseVtt(content); !reflect.DeepEqual(got, want) {
			t.Fatalf("non-rolling input parsed differently from the pre-fix parser:\n got: %+v\nwant: %+v", got, want)
		}
	})
}

// Metamorphic: a carry block — a tiny timing line whose text repeats the
// last kept line — must not add a segment to a rolling file. Lines that
// contain '<', '>' or '&' are skipped because the cleaning (tag strip,
// entity decode) would not round-trip them through a second parse.
func FuzzParseVtt_CarryBlockAddsNothing(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, content string) {
		if detectCaptionKind(content) != CaptionAuto {
			return
		}
		segs := parseVtt(content)
		if len(segs) == 0 {
			return
		}
		last := segs[len(segs)-1].Text
		if strings.ContainsAny(last, "<>&") {
			return
		}
		extended := content + "\n\n99:00:00.000 --> 99:00:00.010\n" + last + "\n \n"
		if got := parseVtt(extended); len(got) != len(segs) {
			t.Fatalf("appending a carry block of %q changed the segment count from %d to %d", last, len(segs), len(got))
		}
	})
}
