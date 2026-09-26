package core

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

// BUG-012: a plain "<lang>" request can 429 (yt-dlp turns it into a
// translation of another track) while the genuine "<lang>-orig" track
// downloads fine. Ground truth is the bug's own specification (docs/BUGS.md
// BUG-012, "Options" 1), not an upstream oracle — there is no TS equivalent.

func TestOrigRetryLanguage(t *testing.T) {
	cases := []struct {
		name     string
		category string
		language string
		want     string
	}{
		{"rate limited plain language retries with -orig", "rate_limited", "en", "en-orig"},
		{"rate limited regional code retries with -orig", "rate_limited", "pt-BR", "pt-BR-orig"},
		{"already -orig is never retried", "rate_limited", "en-orig", ""},
		{"timeout is not retried", "timeout", "en", ""},
		{"missing captions is not retried", "missing_captions", "en", ""},
		{"network error is not retried", "network", "en", ""},
		{"generic error is not retried", "generic", "en", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := origRetryLanguage(tc.category, tc.language); got != tc.want {
				t.Errorf("origRetryLanguage(%q, %q) = %q, want %q", tc.category, tc.language, got, tc.want)
			}
		})
	}
}

var errRateLimited = errors.New("yt-dlp failed: ERROR: Unable to download video subtitles for 'en': HTTP Error 429: Too Many Requests")

func TestFetchWithOrigRetry(t *testing.T) {
	want := parsedTranscript{Segments: []transcriptSegment{{Text: "hello"}}}

	t.Run("success on the first attempt makes no retry", func(t *testing.T) {
		var calls []string
		got, err := fetchWithOrigRetry("en", func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			return want, nil
		})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got (%+v, %v), want (%+v, nil)", got, err, want)
		}
		if !reflect.DeepEqual(calls, []string{"en"}) {
			t.Errorf("calls = %v, want [en]", calls)
		}
	})

	t.Run("429 then successful -orig retry returns the retry's transcript", func(t *testing.T) {
		var calls []string
		got, err := fetchWithOrigRetry("en", func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			if lang == "en" {
				return parsedTranscript{}, errRateLimited
			}
			return want, nil
		})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got (%+v, %v), want (%+v, nil)", got, err, want)
		}
		if !reflect.DeepEqual(calls, []string{"en", "en-orig"}) {
			t.Errorf("calls = %v, want [en en-orig]", calls)
		}
	})

	t.Run("429 then failed retry returns the ORIGINAL 429 error", func(t *testing.T) {
		// e.g. an explicit "en" on a non-English video: en-orig has no track,
		// so the retry's own error ("no transcript ... captions in language")
		// must not replace BUG-011's rate-limited diagnosis.
		retryErr := errors.New(`no transcript available for video x. The video may not have captions in language "en-orig"`)
		_, err := fetchWithOrigRetry("en", func(lang string) (parsedTranscript, error) {
			if lang == "en" {
				return parsedTranscript{}, errRateLimited
			}
			return parsedTranscript{}, retryErr
		})
		if err != errRateLimited {
			t.Errorf("err = %v, want the original rate-limited error", err)
		}
	})

	t.Run("429 on an explicit -orig language is not retried", func(t *testing.T) {
		var calls []string
		_, err := fetchWithOrigRetry("en-orig", func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			return parsedTranscript{}, errRateLimited
		})
		if err != errRateLimited {
			t.Errorf("err = %v, want the original rate-limited error", err)
		}
		if !reflect.DeepEqual(calls, []string{"en-orig"}) {
			t.Errorf("calls = %v, want [en-orig]", calls)
		}
	})

	t.Run("non-429 failure is returned without a retry", func(t *testing.T) {
		timeoutErr := errors.New("transcript fetch timed out")
		var calls []string
		_, err := fetchWithOrigRetry("en", func(lang string) (parsedTranscript, error) {
			calls = append(calls, lang)
			return parsedTranscript{}, timeoutErr
		})
		if err != timeoutErr {
			t.Errorf("err = %v, want the timeout error", err)
		}
		if !reflect.DeepEqual(calls, []string{"en"}) {
			t.Errorf("calls = %v, want [en]", calls)
		}
	})
}

// The retry's outcome gets its own log line: without it a recovered retry
// leaves only the first attempt's rate_limited failure in the log, and a
// failed retry looks like a second, unrelated failure. Same style as
// TestFormatTranscriptFailureLog: the format is the specification.
func TestFormatOrigRetryLog(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		elapsed time.Duration
		want    string
	}{
		{
			name:    "recovered",
			err:     nil,
			elapsed: 3212345678 * time.Nanosecond,
			want:    "lang=en retry=en-orig outcome=recovered duration=3.212s",
		},
		{
			name:    "failed with no captions is classified",
			err:     errors.New(`no transcript available for video x. The video may not have captions in language "en-orig"`),
			elapsed: 3 * time.Second,
			want:    "lang=en retry=en-orig outcome=failed category=missing_captions duration=3s",
		},
		{
			name:    "failed with another 429 is classified",
			err:     errRateLimited,
			elapsed: 6500 * time.Millisecond,
			want:    "lang=en retry=en-orig outcome=failed category=rate_limited duration=6.5s",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatOrigRetryLog("en", "en-orig", tc.elapsed, tc.err); got != tc.want {
				t.Errorf("formatOrigRetryLog() = %q, want %q", got, tc.want)
			}
		})
	}
}
