package core

import "testing"

// Ground truth for qualityFormat and the message templates below was
// derived by running the upstream TS QUALITY_FORMAT_MAP lookup (including
// its `??` fallback-to-hd720 behavior) and message template literals
// directly through Node.

func TestQualityFormat(t *testing.T) {
	hd720 := "bestvideo[vcodec^=avc1][height<=720]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=720]+bestaudio/best[height<=720][ext=mp4]/best[height<=720]"

	cases := []struct {
		quality string
		want    string
	}{
		{"best", "bestvideo[vcodec^=avc1]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1]+bestaudio/best[ext=mp4]/best"},
		{"hd1080", "bestvideo[vcodec^=avc1][height<=1080]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=1080]+bestaudio/best[height<=1080][ext=mp4]/best[height<=1080]"},
		{"hd720", hd720},
		{"sd480", "bestvideo[vcodec^=avc1][height<=480]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=480]+bestaudio/best[height<=480][ext=mp4]/best[height<=480]"},
		{"sd360", "bestvideo[vcodec^=avc1][height<=360]+bestaudio[acodec^=mp4a]/bestvideo[vcodec^=avc1][height<=360]+bestaudio/best[height<=360][ext=mp4]/best[height<=360]"},
		{"bogus", hd720},
		{"", hd720},
	}
	for _, tc := range cases {
		t.Run(tc.quality, func(t *testing.T) {
			if got := qualityFormat(tc.quality); got != tc.want {
				t.Errorf("qualityFormat(%q) = %q, want %q", tc.quality, got, tc.want)
			}
		})
	}
}

func TestFormatVideoDownloadStarted(t *testing.T) {
	want := "Download started:\nJob ID: dl-1\nTitle: My Video\nThe file will appear at: /home/user/Downloads/My_Video.mp4 (extension may differ if H.264 is unavailable)\nIt may take a while for long videos."
	if got := formatVideoDownloadStarted("dl-1", "My Video", "/home/user/Downloads/My_Video.mp4"); got != want {
		t.Errorf("formatVideoDownloadStarted() = %q, want %q", got, want)
	}
}

func TestFormatAudioDownloadStarted(t *testing.T) {
	want := "Download started:\nJob ID: dl-1\nTitle: My Video\nThe file will appear at: /home/user/Downloads/My_Video.mp3\nIt may take a while for long videos."
	if got := formatAudioDownloadStarted("dl-1", "My Video", "/home/user/Downloads/My_Video.mp3"); got != want {
		t.Errorf("formatAudioDownloadStarted() = %q, want %q", got, want)
	}
}
