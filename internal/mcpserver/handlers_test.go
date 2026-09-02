package mcpserver

// These tests cover the input-validation branches of the handlers that
// return before reaching any network/yt-dlp call (invalid URL, empty
// query, a rejected outputDir), mirroring the pattern already used for
// getTranscriptRangeHandler in tools_test.go. Happy paths remain covered
// only by the manual smoke test (docs/tasks/07-mcpserver/TASK.md), per the
// project's convention for I/O-bound entrypoints.

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// outsideAllowedRoot is a directory outside both allowed output roots
// (os.TempDir() and the user's home dir), used to exercise
// core.ResolveOutputDir's rejection branch in the download handlers.
func outsideAllowedRoot() string {
	if runtime.GOOS == "windows" {
		return `C:\definitely-unrelated-root-xyz\evil`
	}
	return "/definitely-unrelated-root-xyz/evil"
}

func TestGetTranscriptHandler_InvalidURL(t *testing.T) {
	res, _, err := getTranscriptHandler(context.Background(), nil, urlLangInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestGetTranscriptTimedHandler_InvalidURL(t *testing.T) {
	res, _, err := getTranscriptTimedHandler(context.Background(), nil, urlLangInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestGetMetadataHandler_InvalidURL(t *testing.T) {
	res, _, err := getMetadataHandler(context.Background(), nil, metadataInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestGetChaptersHandler_InvalidURL(t *testing.T) {
	res, _, err := getChaptersHandler(context.Background(), nil, metadataInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestSearchTranscriptHandler_InvalidURL(t *testing.T) {
	res, _, err := searchTranscriptHandler(context.Background(), nil, searchInput{URL: "not a url", Query: "hi"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestSearchTranscriptHandler_EmptyQuery(t *testing.T) {
	res, _, err := searchTranscriptHandler(context.Background(), nil, searchInput{URL: "dQw4w9WgXcQ", Query: "   "})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for a blank query")
	}
	if !strings.Contains(contentText(t, res), "non-empty search query") {
		t.Errorf("text = %q, want it to mention the empty-query requirement", contentText(t, res))
	}
}

func TestListPlaylistHandler_InvalidURL(t *testing.T) {
	res, _, err := listPlaylistHandler(context.Background(), nil, playlistInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid playlist URL")
	}
}

func TestListPlaylistHandler_VideoURLNotPlaylist(t *testing.T) {
	res, _, err := listPlaylistHandler(context.Background(), nil, playlistInput{URL: "dQw4w9WgXcQ"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for a bare video ID (not a playlist ID)")
	}
}

func TestSearchPlaylistHandler_InvalidURL(t *testing.T) {
	res, _, err := searchPlaylistHandler(context.Background(), nil, playlistSearchInput{URL: "not a url", Query: "hi"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid playlist URL")
	}
}

func TestSearchPlaylistHandler_EmptyQuery(t *testing.T) {
	res, _, err := searchPlaylistHandler(context.Background(), nil, playlistSearchInput{URL: "PLBCF2DAC6FFB574DE", Query: "   "})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for a blank query")
	}
	if !strings.Contains(contentText(t, res), "non-empty search query") {
		t.Errorf("text = %q, want it to mention the empty-query requirement", contentText(t, res))
	}
}

func TestDownloadVideoHandler_InvalidURL(t *testing.T) {
	res, _, err := downloadVideoHandler(context.Background(), nil, downloadVideoInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestDownloadVideoHandler_InvalidOutputDir(t *testing.T) {
	res, _, err := downloadVideoHandler(context.Background(), nil, downloadVideoInput{
		URL: "dQw4w9WgXcQ", OutputDir: outsideAllowedRoot(),
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an outputDir outside the allowed roots")
	}
	if !strings.Contains(contentText(t, res), "Invalid outputDir") {
		t.Errorf("text = %q, want it to mention the invalid outputDir", contentText(t, res))
	}
}

func TestDownloadAudioHandler_InvalidURL(t *testing.T) {
	res, _, err := downloadAudioHandler(context.Background(), nil, downloadAudioInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestDownloadAudioHandler_InvalidOutputDir(t *testing.T) {
	res, _, err := downloadAudioHandler(context.Background(), nil, downloadAudioInput{
		URL: "dQw4w9WgXcQ", OutputDir: outsideAllowedRoot(),
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an outputDir outside the allowed roots")
	}
}

func TestGetDownloadStatusHandler_UnknownJobID(t *testing.T) {
	res, _, err := getDownloadStatusHandler(context.Background(), nil, jobStatusInput{JobID: "nope"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an unknown job ID")
	}
	if !strings.Contains(contentText(t, res), "Unknown job ID") {
		t.Errorf("text = %q, want it to mention the unknown job ID", contentText(t, res))
	}
}

func TestListDownloadsHandler_Empty(t *testing.T) {
	res, _, err := listDownloadsHandler(context.Background(), nil, listDownloadsInput{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Error("IsError = true, want false for list_downloads")
	}
}

func TestDownloadTranscriptHandler_InvalidURL(t *testing.T) {
	res, _, err := downloadTranscriptHandler(context.Background(), nil, downloadTranscriptInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestDownloadTranscriptHandler_InvalidOutputDir(t *testing.T) {
	res, _, err := downloadTranscriptHandler(context.Background(), nil, downloadTranscriptInput{
		URL: "dQw4w9WgXcQ", OutputDir: outsideAllowedRoot(),
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an outputDir outside the allowed roots")
	}
}

func TestDownloadTranscriptTimedHandler_InvalidURL(t *testing.T) {
	res, _, err := downloadTranscriptTimedHandler(context.Background(), nil, downloadTranscriptInput{URL: "not a url"})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an invalid URL")
	}
}

func TestDownloadTranscriptTimedHandler_InvalidOutputDir(t *testing.T) {
	res, _, err := downloadTranscriptTimedHandler(context.Background(), nil, downloadTranscriptInput{
		URL: "dQw4w9WgXcQ", OutputDir: outsideAllowedRoot(),
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Error("IsError = false, want true for an outputDir outside the allowed roots")
	}
}

// connectedClient spins up NewServer's real MCP server in-process, wired to
// a real client over an in-memory transport pair (the same technique as
// task 7's throwaway smoke-test client, kept permanently here instead of
// thrown away). Returns the connected client session; t.Cleanup handles
// disconnecting both ends.
func connectedClient(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := NewServer("test")
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestNewServer_ToolCount(t *testing.T) {
	cs := connectedClient(t)
	// 8 canonical tools + 3 aliases (get_transcript_timestamps,
	// get_video_metadata, search_in_transcript) + get_transcript_range,
	// download_transcript_timed, get_download_status, list_downloads,
	// get_chapters, list_playlist, and search_playlist = 17 — see
	// tools.go's NewServer doc comment.
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if got, want := len(res.Tools), 17; got != want {
		names := make([]string, len(res.Tools))
		for i, tl := range res.Tools {
			names[i] = tl.Name
		}
		t.Fatalf("registered tool count = %d, want %d (tools: %v)", got, want, names)
	}
}

// TestNewServer_AliasesMatchCanonical drives each alias/canonical pair
// through a real MCP round trip with the same (invalid-URL, no network
// needed) input, and asserts byte-identical output — verifying, by actual
// behavior rather than a source comment, that each alias is wired to the
// same handler as its canonical tool.
func TestNewServer_AliasesMatchCanonical(t *testing.T) {
	cs := connectedClient(t)
	ctx := context.Background()

	pairs := []struct {
		canonical, alias string
		args             map[string]any
	}{
		{"get_transcript_timed", "get_transcript_timestamps", map[string]any{"url": "not a url"}},
		{"get_metadata", "get_video_metadata", map[string]any{"url": "not a url"}},
		{"search_transcript", "search_in_transcript", map[string]any{"url": "not a url", "query": "x"}},
	}

	for _, p := range pairs {
		t.Run(p.canonical, func(t *testing.T) {
			canonRes, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: p.canonical, Arguments: p.args})
			if err != nil {
				t.Fatalf("CallTool(%q): %v", p.canonical, err)
			}
			aliasRes, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: p.alias, Arguments: p.args})
			if err != nil {
				t.Fatalf("CallTool(%q): %v", p.alias, err)
			}

			if canonRes.IsError != aliasRes.IsError {
				t.Errorf("IsError: canonical=%v alias=%v, want equal", canonRes.IsError, aliasRes.IsError)
			}
			canonText, aliasText := contentText(t, canonRes), contentText(t, aliasRes)
			if canonText != aliasText {
				t.Errorf("output differs:\ncanonical (%s) = %q\nalias (%s)     = %q", p.canonical, canonText, p.alias, aliasText)
			}
		})
	}
}
