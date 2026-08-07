package core

import (
	"net/url"
	"regexp"
	"strings"
)

var videoIDRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// ExtractVideoID pulls an 11-character YouTube video ID out of a bare ID,
// a youtu.be short link, or a youtube.com watch/shorts/embed/v URL.
func ExtractVideoID(input string) string {
	if videoIDRe.MatchString(input) {
		return input
	}

	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	var id string
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "youtu.be":
		id = strings.SplitN(strings.TrimPrefix(u.Path, "/"), "?", 2)[0]
	case strings.Contains(host, "youtube.com"):
		parts := strings.Split(u.Path, "/")
		if len(parts) > 2 && (parts[1] == "shorts" || parts[1] == "embed" || parts[1] == "v") {
			id = parts[2]
		} else {
			id = u.Query().Get("v")
		}
	}

	if id != "" && videoIDRe.MatchString(id) {
		return id
	}
	return ""
}
