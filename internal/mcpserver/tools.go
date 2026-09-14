// Package mcpserver implements the youtube-mcp MCP stdio server: a Go
// port of the upstream project's server (packages/mcp/src/index.ts) built
// on the official github.com/modelcontextprotocol/go-sdk, on top of
// internal/core.
package mcpserver

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/johncegom/go-youtube-mcp-cli/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Input shapes ──────────────────────────────────────────────────────────

type urlLangInput struct {
	URL      string `json:"url" jsonschema:"The full YouTube URL or video ID (e.g. https://youtube.com/watch?v=abc123 or just abc123)"`
	Language string `json:"language,omitempty" jsonschema:"Optional. Language code for the transcript (e.g. 'en', 'it'). Defaults to 'en'."`
}

type metadataInput struct {
	URL string `json:"url" jsonschema:"The full YouTube URL or video ID"`
}

type searchInput struct {
	URL      string   `json:"url" jsonschema:"The full YouTube URL or video ID"`
	Query    string   `json:"query" jsonschema:"The keyword or phrase to search for"`
	Language string   `json:"language,omitempty" jsonschema:"Optional. Language code. Defaults to 'en'."`
	Context  *float64 `json:"context,omitempty" jsonschema:"Optional. Seconds of context to include around each match, default 15. Pass 0 to return only the matched segment(s), no surrounding context."`
}

type rangeInput struct {
	URL      string `json:"url" jsonschema:"The full YouTube URL or video ID"`
	Start    string `json:"start,omitempty" jsonschema:"Optional start timestamp (e.g. '1:30' or '1:02:03'). Omit for the beginning of the video."`
	End      string `json:"end,omitempty" jsonschema:"Optional end timestamp (e.g. '3:45' or '1:05:00'). Omit for the end of the video."`
	Language string `json:"language,omitempty" jsonschema:"Optional. Language code for the transcript (e.g. 'en', 'it'). Defaults to 'en'."`
}

type downloadVideoInput struct {
	URL       string `json:"url" jsonschema:"The full YouTube URL or video ID"`
	Quality   string `json:"quality,omitempty" jsonschema:"Optional. 'hd720' (default), 'best', 'hd1080', 'sd480', 'sd360'."`
	OutputDir string `json:"outputDir,omitempty" jsonschema:"Optional. Directory to save. Defaults to ~/Downloads."`
}

type downloadAudioInput struct {
	URL       string `json:"url" jsonschema:"The full YouTube URL or video ID"`
	Format    string `json:"format,omitempty" jsonschema:"Optional. Audio format: 'mp3' (default), 'm4a', 'aac', 'flac', 'opus', 'wav', 'vorbis'."`
	OutputDir string `json:"outputDir,omitempty" jsonschema:"Optional. Directory to save. Defaults to ~/Downloads."`
}

type jobStatusInput struct {
	JobID string `json:"jobId" jsonschema:"The job ID returned by download_video or download_audio (e.g. 'dl-1')."`
}

type listDownloadsInput struct{}

type downloadTranscriptInput struct {
	URL       string `json:"url" jsonschema:"The full YouTube URL or video ID"`
	Language  string `json:"language,omitempty" jsonschema:"Optional. Language code. Defaults to 'en'."`
	OutputDir string `json:"outputDir,omitempty" jsonschema:"Optional. Directory to save. Defaults to ~/Downloads."`
}

type playlistInput struct {
	URL string `json:"url" jsonschema:"The full YouTube playlist URL or playlist ID"`
}

type playlistSearchInput struct {
	URL      string `json:"url" jsonschema:"The full YouTube playlist URL or playlist ID"`
	Query    string `json:"query" jsonschema:"The keyword or phrase to search for"`
	Language string `json:"language,omitempty" jsonschema:"Optional. Language code. Defaults to 'en'."`
}

// ── Result helpers ────────────────────────────────────────────────────────

func textResult(text string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: isError,
	}
}

func invalidURLResult(url string) *mcp.CallToolResult {
	return textResult(fmt.Sprintf(
		"Invalid YouTube URL or video ID: %q. Please provide a valid YouTube URL (e.g. https://youtube.com/watch?v=abc123) or a bare video ID.",
		url), true)
}

func invalidPlaylistURLResult(url string) *mcp.CallToolResult {
	return textResult(fmt.Sprintf(
		"Invalid YouTube playlist URL or playlist ID: %q. Please provide a valid playlist URL (e.g. https://youtube.com/playlist?list=PL...) or a bare playlist ID.",
		url), true)
}

// ── Handlers ──────────────────────────────────────────────────────────────
//
// Out is always `any`: these tools return free-text content built by hand,
// matching the upstream TS server's `{ content: [...], isError }` pattern
// exactly, not the SDK's structured-output auto-marshaling (which would
// kick in for any concrete Out type).

func getTranscriptHandler(ctx context.Context, _ *mcp.CallToolRequest, in urlLangInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	text, err := core.GetTranscriptText(ctx, videoID, in.Language)
	if err != nil {
		return textResult(core.TranscriptErrorText(videoID, err), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func getTranscriptTimedHandler(ctx context.Context, _ *mcp.CallToolRequest, in urlLangInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	text, err := core.GetTranscriptTimed(ctx, videoID, in.Language)
	if err != nil {
		return textResult(core.TranscriptErrorText(videoID, err), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

// getTranscriptRangeHandler returns only the transcript segments within
// [start, end] (either may be omitted for an open-ended range), formatted
// as timed lines.
func getTranscriptRangeHandler(ctx context.Context, _ *mcp.CallToolRequest, in rangeInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}

	var startSec, endSec *float64
	if strings.TrimSpace(in.Start) != "" {
		s, err := core.ParseTimestamp(in.Start)
		if err != nil {
			return textResult(fmt.Sprintf("Invalid start timestamp %q: %s", in.Start, err), true), nil, nil
		}
		startSec = &s
	}
	if strings.TrimSpace(in.End) != "" {
		e, err := core.ParseTimestamp(in.End)
		if err != nil {
			return textResult(fmt.Sprintf("Invalid end timestamp %q: %s", in.End, err), true), nil, nil
		}
		endSec = &e
	}
	if startSec != nil && endSec != nil && *startSec > *endSec {
		return textResult("Invalid range: start must not be after end.", true), nil, nil
	}

	text, err := core.GetTranscriptRange(ctx, videoID, in.Language, startSec, endSec)
	if err != nil {
		return textResult(core.TranscriptErrorText(videoID, err), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func getMetadataHandler(ctx context.Context, _ *mcp.CallToolRequest, in metadataInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	meta, err := core.FetchVideoMetadata(ctx, videoID)
	if err != nil {
		return textResult(fmt.Sprintf("Failed to fetch metadata: %s", err.Error()), true), nil, nil
	}

	lines := metadataLines(meta)
	text := "No metadata found."
	if len(lines) > 0 {
		text = strings.Join(lines, "\n")
	}
	return textResult(text, false), nil, nil
}

// metadataLines renders the non-empty metadata fields as "Label: value"
// lines in get_metadata's fixed order (description last, since it's the
// longest). Shared by get_metadata and get_video_brief.
func metadataLines(meta map[string]string) []string {
	var lines []string
	if meta["title"] != "" {
		lines = append(lines, "Title: "+meta["title"])
	}
	if meta["channel"] != "" {
		lines = append(lines, "Channel: "+meta["channel"])
	}
	if meta["publishDate"] != "" {
		lines = append(lines, "Published: "+meta["publishDate"])
	}
	if meta["viewCount"] != "" {
		lines = append(lines, "Views: "+meta["viewCount"])
	}
	if meta["duration"] != "" {
		lines = append(lines, "Duration: "+meta["duration"])
	}
	if meta["channelUrl"] != "" {
		lines = append(lines, "Channel URL: "+meta["channelUrl"])
	}
	if meta["channelId"] != "" {
		lines = append(lines, "Channel ID: "+meta["channelId"])
	}
	if meta["description"] != "" {
		lines = append(lines, "Description: "+meta["description"])
	}
	return lines
}

// chapterLines renders chapters one "[H:MM:SS] Title" line each — the
// get_chapters output format, reused verbatim inside get_video_brief.
func chapterLines(chapters []core.Chapter) []string {
	lines := make([]string, len(chapters))
	for i, c := range chapters {
		lines[i] = fmt.Sprintf("[%s] %s", core.FormatTimestamp(c.StartSecs), c.Title)
	}
	return lines
}

// formatVideoBrief renders a core.VideoBrief as one text block, sections
// separated by a blank line and ordered so a client that truncates still
// shows the header: video ID, metadata, chapters, transcript stats, timed
// transcript. A failed section is replaced by a single failure line;
// chapters are omitted entirely when metadata failed (same fetch). The
// returned isError is true iff the transcript section failed — see
// docs/tasks/17-video-brief/TASK.md and docs/DECISIONS.md DECISION-021.
func formatVideoBrief(videoID string, b core.VideoBrief) (string, bool) {
	sections := []string{"Video: " + videoID}

	if b.MetadataErr != nil {
		sections[0] += "\nMetadata: Failed to fetch metadata: " + b.MetadataErr.Error()
	} else {
		if lines := metadataLines(b.Metadata); len(lines) > 0 {
			sections[0] += "\n" + strings.Join(lines, "\n")
		}
		if len(b.Chapters) == 0 {
			sections = append(sections, "No chapters found.")
		} else {
			sections = append(sections, "Chapters:\n"+strings.Join(chapterLines(b.Chapters), "\n"))
		}
	}

	if b.TranscriptErr != nil {
		sections = append(sections, "Transcript: "+core.TranscriptErrorText(videoID, b.TranscriptErr))
		return strings.Join(sections, "\n\n"), true
	}

	sections = append(sections, "Transcript stats:\n"+strings.Join(statsLines(b.CaptionKind, b.Stats), "\n"))
	sections = append(sections, "Transcript (timed):\n"+b.TranscriptTimed)
	return strings.Join(sections, "\n\n"), false
}

func statsLines(kind core.CaptionKind, st core.TranscriptStats) []string {
	captions := "unknown"
	switch kind {
	case core.CaptionAuto:
		captions = "likely auto-generated"
	case core.CaptionUploaded:
		captions = "likely uploaded"
	}

	cues := fmt.Sprintf("Non-speech cues: %d", st.NonSpeechCues)
	if len(st.NonSpeechBreakdown) > 0 {
		keys := make([]string, 0, len(st.NonSpeechBreakdown))
		for k := range st.NonSpeechBreakdown {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = fmt.Sprintf("%s x%d", k, st.NonSpeechBreakdown[k])
		}
		cues += " (" + strings.Join(parts, ", ") + ")"
	}

	return []string{
		"Captions: " + captions,
		fmt.Sprintf("Words: %d", st.Words),
		fmt.Sprintf("Speaking rate: %d words/min", st.SpeakingRateWPM),
		cues,
		fmt.Sprintf("Longest gap: %ss at [%s]", strconv.FormatFloat(st.LongestGapSecs, 'f', -1, 64), core.FormatTimestamp(st.LongestGapAtSecs)),
	}
}

// getVideoBriefHandler is the composite "everything an evaluation needs"
// tool: metadata + chapters + timed transcript + stats in one call, with
// per-section failure reporting (see formatVideoBrief).
func getVideoBriefHandler(ctx context.Context, _ *mcp.CallToolRequest, in urlLangInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	text, isErr := formatVideoBrief(videoID, core.FetchVideoBrief(ctx, videoID, in.Language))
	return textResult(text, isErr), nil, nil
}

func getChaptersHandler(ctx context.Context, _ *mcp.CallToolRequest, in metadataInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	chapters, err := core.FetchChapters(ctx, videoID)
	if err != nil {
		return textResult(fmt.Sprintf("Failed to fetch chapters: %s", err.Error()), true), nil, nil
	}
	if len(chapters) == 0 {
		return textResult(fmt.Sprintf("No chapters found for video %s.", videoID), false), nil, nil
	}
	return textResult(strings.Join(chapterLines(chapters), "\n"), false), nil, nil
}

func searchTranscriptHandler(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	if strings.TrimSpace(in.Query) == "" {
		return textResult("Please provide a non-empty search query.", true), nil, nil
	}
	contextSecs := 15.0
	if in.Context != nil {
		contextSecs = *in.Context
	}
	text, err := core.SearchInTranscript(ctx, videoID, in.Query, in.Language, contextSecs)
	if err != nil {
		return textResult(core.TranscriptErrorText(videoID, err), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func downloadVideoHandler(ctx context.Context, _ *mcp.CallToolRequest, in downloadVideoInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	outputDir := core.ResolveOutputDir(in.OutputDir)
	if outputDir == "" {
		return textResult("Invalid outputDir: must be within the home or temp directory.", true), nil, nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return textResult(err.Error(), true), nil, nil
	}

	quality := in.Quality
	if quality == "" {
		quality = "hd720"
	}
	text, err := core.StartVideoDownload(ctx, videoID, quality, outputDir)
	if err != nil {
		return textResult(err.Error(), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func downloadAudioHandler(ctx context.Context, _ *mcp.CallToolRequest, in downloadAudioInput) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	outputDir := core.ResolveOutputDir(in.OutputDir)
	if outputDir == "" {
		return textResult("Invalid outputDir: must be within the home or temp directory.", true), nil, nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return textResult(err.Error(), true), nil, nil
	}

	format := in.Format
	if format == "" {
		format = "mp3"
	}
	text, err := core.StartAudioDownload(ctx, videoID, format, outputDir)
	if err != nil {
		return textResult(err.Error(), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func getDownloadStatusHandler(_ context.Context, _ *mcp.CallToolRequest, in jobStatusInput) (*mcp.CallToolResult, any, error) {
	text, ok := core.FormatDownloadStatus(in.JobID)
	if !ok {
		return textResult(fmt.Sprintf("Unknown job ID: %q", in.JobID), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

func listDownloadsHandler(_ context.Context, _ *mcp.CallToolRequest, _ listDownloadsInput) (*mcp.CallToolResult, any, error) {
	return textResult(core.FormatDownloadsList(), false), nil, nil
}

// downloadTranscript is shared by the download_transcript and
// download_transcript_timed tools, which differ only in withTimestamps —
// mirrors the TS server's switch statement branching on request.params.name
// for these two cases.
func downloadTranscript(ctx context.Context, in downloadTranscriptInput, withTimestamps bool) (*mcp.CallToolResult, any, error) {
	videoID := core.ExtractVideoID(in.URL)
	if videoID == "" {
		return invalidURLResult(in.URL), nil, nil
	}
	outputDir := core.ResolveOutputDir(in.OutputDir)
	if outputDir == "" {
		return textResult("Invalid outputDir: must be within the home or temp directory.", true), nil, nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return textResult(err.Error(), true), nil, nil
	}

	filePath, err := core.SaveTranscriptFile(ctx, videoID, in.Language, outputDir, withTimestamps)
	if err != nil {
		return textResult(core.TranscriptErrorText(videoID, err), true), nil, nil
	}
	return textResult(fmt.Sprintf("Transcript saved to: %s", filePath), false), nil, nil
}

func downloadTranscriptHandler(ctx context.Context, _ *mcp.CallToolRequest, in downloadTranscriptInput) (*mcp.CallToolResult, any, error) {
	return downloadTranscript(ctx, in, false)
}

func downloadTranscriptTimedHandler(ctx context.Context, _ *mcp.CallToolRequest, in downloadTranscriptInput) (*mcp.CallToolResult, any, error) {
	return downloadTranscript(ctx, in, true)
}

func listPlaylistHandler(ctx context.Context, _ *mcp.CallToolRequest, in playlistInput) (*mcp.CallToolResult, any, error) {
	playlistID := core.ExtractPlaylistID(in.URL)
	if playlistID == "" {
		return invalidPlaylistURLResult(in.URL), nil, nil
	}
	entries, total, err := core.ListPlaylistVideos(ctx, playlistID)
	if err != nil {
		return textResult(err.Error(), true), nil, nil
	}
	if len(entries) == 0 {
		return textResult("This playlist has no videos.", false), nil, nil
	}
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = fmt.Sprintf("%d. %s (%s)", i+1, e.Title, e.VideoID)
	}
	text := strings.Join(lines, "\n")
	if len(entries) < total {
		text += fmt.Sprintf("\n\n(showing first %d of %d videos)", len(entries), total)
	} else {
		text += fmt.Sprintf("\n\n%d video(s) total.", total)
	}
	return textResult(text, false), nil, nil
}

func searchPlaylistHandler(ctx context.Context, _ *mcp.CallToolRequest, in playlistSearchInput) (*mcp.CallToolResult, any, error) {
	playlistID := core.ExtractPlaylistID(in.URL)
	if playlistID == "" {
		return invalidPlaylistURLResult(in.URL), nil, nil
	}
	if strings.TrimSpace(in.Query) == "" {
		return textResult("Please provide a non-empty search query.", true), nil, nil
	}
	text, err := core.SearchPlaylist(ctx, playlistID, in.Query, in.Language)
	if err != nil {
		return textResult(err.Error(), true), nil, nil
	}
	return textResult(text, false), nil, nil
}

// ── Server construction ──────────────────────────────────────────────────

// NewServer builds the youtube-mcp-cli MCP server with all 18 tools
// registered (including the 3 alias pairs, which point at the same
// handler function as their canonical tool).
func NewServer(version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "youtube-mcp-cli", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript",
		Description: "Fetches the transcript of a YouTube video given its URL or video ID.",
	}, getTranscriptHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript_timed",
		Description: "Fetches the transcript with timestamps for each segment.",
	}, getTranscriptTimedHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript_timestamps",
		Description: "Fetches the transcript with timestamps for each segment. (Alias for get_transcript_timed)",
	}, getTranscriptTimedHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript_range",
		Description: "Fetches a windowed portion of the transcript between start and end timestamps (e.g. '1:30' to '3:45'). Either may be omitted for an open-ended range.",
	}, getTranscriptRangeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_metadata",
		Description: "Fetches video metadata: title, channel, description, publish date, views, duration.",
	}, getMetadataHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_video_metadata",
		Description: "Fetches video metadata: title, channel, description, publish date, views, duration. (Alias for get_metadata)",
	}, getMetadataHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_chapters",
		Description: "Fetches the video's chapters (timestamped table of contents), one '[H:MM:SS] Title' line per chapter, or a message if the video has no chapters.",
	}, getChaptersHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_video_brief",
		Description: "One call for a full video evaluation: metadata, chapters, the full timed transcript, and transcript stats (caption kind: auto-generated vs uploaded; word count; speaking rate; non-speech cues like [Music]; longest silent gap). Use this instead of calling get_metadata + get_chapters + get_transcript_timed separately. For long videos or targeted questions prefer get_chapters + get_transcript_range instead. Sections fail independently — a failed section is replaced by a failure line and the rest is still returned; isError is set only when the transcript could not be fetched.",
	}, getVideoBriefHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_transcript",
		Description: "Searches for a keyword or phrase in the transcript, matching across segment boundaries. Returns each match's surrounding context (± 'context' seconds, default 15; pass 0 for matched segments only) as timed-line blocks separated by '---', with the total distinct match count.",
	}, searchTranscriptHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_in_transcript",
		Description: "Searches for a keyword or phrase in the transcript, matching across segment boundaries. Returns each match's surrounding context (± 'context' seconds, default 15; pass 0 for matched segments only) as timed-line blocks separated by '---', with the total distinct match count. (Alias for search_transcript)",
	}, searchTranscriptHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "download_video",
		Description: "Starts a background download of a YouTube video (video+audio) to the local filesystem and returns a job ID; use get_download_status to check completion and the final file path.",
	}, downloadVideoHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "download_audio",
		Description: "Starts a background download of audio from a YouTube video and returns a job ID; use get_download_status to check completion and the final file path.",
	}, downloadAudioHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_download_status",
		Description: "Checks the status of a background download started by download_video or download_audio: running, done (with the actual final file path), or failed (with the captured error).",
	}, getDownloadStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_downloads",
		Description: "Lists all download jobs known to this server process (ID, kind, video, state).",
	}, listDownloadsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "download_transcript",
		Description: "Downloads the transcript of a YouTube video as a markdown file (.md). Returns the file path.",
	}, downloadTranscriptHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "download_transcript_timed",
		Description: "Downloads the transcript of a YouTube video as a markdown file (.md) with timestamps. Returns the file path.",
	}, downloadTranscriptTimedHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_playlist",
		Description: "Lists the videos in a YouTube playlist (title + video ID, numbered, in playlist order). Capped to the first 25 videos, with a note if the playlist has more.",
	}, listPlaylistHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_playlist",
		Description: "Searches for a keyword or phrase across every video's transcript in a YouTube playlist (capped to the first 25 videos), fetched sequentially to avoid rate limiting. Returns matches grouped by video as '<title> [MM:SS] <text>' lines, plus a list of any videos skipped due to fetch errors (e.g. no captions).",
	}, searchPlaylistHandler)

	return server
}
