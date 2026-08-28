package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func toHMS(seconds float64) (h, m, s int) {
	total := int(seconds)
	h = total / 3600
	m = (total % 3600) / 60
	s = total % 60
	return
}

// FormatTimestamp renders a padded H:MM:SS or MM:SS timestamp, used for
// transcript line prefixes.
func FormatTimestamp(seconds float64) string {
	h, m, s := toHMS(seconds)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// FormatDuration renders a natural H:MM:SS or M:SS duration, used for
// video-length display.
func FormatDuration(seconds float64) string {
	h, m, s := toHMS(seconds)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// ParseTimestamp parses a "M:SS", "MM:SS", or "H:MM:SS" timestamp string
// into a total number of seconds. Seconds must be in [0, 60), and minutes
// must be in [0, 60) when an hours component is present; minutes are
// unbounded in the 2-part form (e.g. "90:00" is 5400 seconds).
func ParseTimestamp(s string) (float64, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")

	parseComponent := func(s string, max int) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || (max > 0 && n >= max) {
			return 0, fmt.Errorf("invalid timestamp component %q", s)
		}
		return n, nil
	}

	switch len(parts) {
	case 2:
		m, err := parseComponent(parts[0], 0)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		sec, err := parseComponent(parts[1], 60)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		return float64(m*60 + sec), nil
	case 3:
		h, err := parseComponent(parts[0], 0)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		m, err := parseComponent(parts[1], 60)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		sec, err := parseComponent(parts[2], 60)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		return float64(h*3600 + m*60 + sec), nil
	default:
		return 0, fmt.Errorf("invalid timestamp %q: expected M:SS or H:MM:SS", s)
	}
}

var isoDurationRe = regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)

func parseISODuration(iso string) int {
	m := isoDurationRe.FindStringSubmatch(iso)
	if m == nil {
		return 0
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	return toInt(m[1])*3600 + toInt(m[2])*60 + toInt(m[3])
}

var (
	illegalFilenameCharsRe  = regexp.MustCompile(`[\\/:*?"<>|]`)
	spaceAroundUnderscoreRe = regexp.MustCompile(`\s*_\s*`)
	repeatedUnderscoreRe    = regexp.MustCompile(`_+`)
)

// SanitizeTitle turns an arbitrary video title into a safe filename stem.
func SanitizeTitle(title string) string {
	result := illegalFilenameCharsRe.ReplaceAllString(title, "_")
	result = spaceAroundUnderscoreRe.ReplaceAllString(result, "_")
	result = repeatedUnderscoreRe.ReplaceAllString(result, "_")
	result = strings.Trim(result, "_")
	if result == "" {
		return "video"
	}
	return result
}
