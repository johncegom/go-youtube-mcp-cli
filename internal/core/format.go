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
