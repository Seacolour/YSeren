package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListTreeHandlerReturnsMediaTreeAndSearches(t *testing.T) {
	root := t.TempDir()
	writeTreeTestFile(t, root, "Season 1/Episode 01.mp4", "video")
	writeTreeTestFile(t, root, "Music/theme song.mp3", "audio")
	writeTreeTestFile(t, root, "private.txt", "not shared")
	writeTreeTestFile(t, root, "archive.zip", "not shared")

	conf := streamTestConfig("tree", root)
	handler := New(conf, Options{}).TreeHandler()
	full := getTreeResponse(t, handler, "/api/tree")
	video := findTreeNodeByName(full.Root, "Episode 01.mp4")
	if video == nil || video.MediaType != "video" {
		t.Fatalf("video node = %#v", video)
	}
	if video.URL != BuildStreamURL("tree", "Season 1/Episode 01.mp4") || video.Size != int64(len("video")) || video.ModTime <= 0 {
		t.Fatalf("video metadata = %#v", video)
	}
	audio := findTreeNodeByName(full.Root, "theme song.mp3")
	if audio == nil || audio.MediaType != "audio" {
		t.Fatalf("audio node = %#v", audio)
	}
	if findTreeNodeByName(full.Root, "private.txt") != nil || findTreeNodeByName(full.Root, "archive.zip") != nil {
		t.Fatal("non-media file appeared in tree")
	}

	filtered := getTreeResponse(t, handler, "/api/tree?q=theme")
	if findTreeNodeByName(filtered.Root, "theme song.mp3") == nil || findTreeNodeByName(filtered.Root, "Episode 01.mp4") != nil {
		t.Fatalf("filtered tree = %#v", filtered.Root)
	}
}

func TestListTreeHandlerRefreshInvalidatesCache(t *testing.T) {
	root := t.TempDir()
	writeTreeTestFile(t, root, "first.mp4", "first")
	handler := New(streamTestConfig("refresh", root), Options{}).TreeHandler()

	initial := getTreeResponse(t, handler, "/api/tree")
	if findTreeNodeByName(initial.Root, "first.mp4") == nil {
		t.Fatal("initial file missing")
	}
	writeTreeTestFile(t, root, "second.mp4", "second")
	if findTreeNodeByName(getTreeResponse(t, handler, "/api/tree").Root, "second.mp4") != nil {
		t.Fatal("cached response unexpectedly included new file")
	}
	if findTreeNodeByName(getTreeResponse(t, handler, "/api/tree?refresh=1").Root, "second.mp4") == nil {
		t.Fatal("refresh did not include new file")
	}
}

func TestListTreeHandlerRejectsUnsupportedRequests(t *testing.T) {
	handler := New(streamTestConfig("videos", t.TempDir()), Options{}).TreeHandler()
	postRecorder := httptest.NewRecorder()
	handler.ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/api/tree", nil))
	if postRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", postRecorder.Code)
	}
	unknownRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unknownRecorder, httptest.NewRequest(http.MethodGet, "/api/tree?source=missing", nil))
	if unknownRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown source status = %d, want 400", unknownRecorder.Code)
	}
}

func TestVideosHandlerPreservesLegacySearchAndPagination(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTreeTestFile(t, root, "A/episode-01.mp4", "first")
	writeTreeTestFile(t, root, "B/episode-02.mp4", "second")
	writeTreeTestFile(t, root, "B/theme.mp3", "audio")
	writeTreeTestFile(t, root, "private.txt", "private")
	application := New(streamTestConfig("media", root), Options{})

	full := getVideosResponse(t, application.VideosHandler(), "/api/videos")
	if full.Total != 3 || len(full.Items) != 3 {
		t.Fatalf("full response = %#v", full)
	}
	if full.Items[2].MediaType != "audio" || full.Items[2].Name != "theme.mp3" {
		t.Fatalf("audio item = %#v", full.Items[2])
	}

	page := getVideosResponse(t, application.VideosHandler(), "/api/videos?q=episode&offset=1&limit=1")
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Name != "episode-02.mp4" {
		t.Fatalf("paged response = %#v", page)
	}
}

func getTreeResponse(t *testing.T, handler http.Handler, target string) TreeResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want 200; body=%q", target, recorder.Code, recorder.Body.String())
	}
	var response TreeResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode %s response: %v", target, err)
	}
	if response.Root == nil {
		t.Fatalf("%s returned nil root", target)
	}
	return response
}

func getVideosResponse(t *testing.T, handler http.Handler, target string) VideosResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want 200; body=%q", target, recorder.Code, recorder.Body.String())
	}
	var response VideosResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode %s response: %v", target, err)
	}
	return response
}

func findTreeNodeByName(node *TreeNode, name string) *TreeNode {
	if node == nil {
		return nil
	}
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if found := findTreeNodeByName(child, name); found != nil {
			return found
		}
	}
	return nil
}

func writeTreeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir parent for %q: %v", relPath, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", relPath, err)
	}
}
