package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "yseren/internal/config"
)

func TestStreamHandlerServesAllowedMedia(t *testing.T) {
	root := t.TempDir()
	relPath := "Season 1/正片 + 100%.mp4"
	content := "0123456789"
	writeStreamTestFile(t, root, relPath, content)

	conf := streamTestConfig("100% #动漫", root)
	recorder := httptest.NewRecorder()
	NewStreamHandler(conf).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, BuildStreamURL(conf.Sources[0].Name, relPath), nil))

	if recorder.Code != http.StatusOK || recorder.Body.String() != content {
		t.Fatalf("response = status %d body %q", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "video/mp4" {
		t.Fatalf("Content-Type = %q, want video/mp4", got)
	}
	if got := recorder.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestStreamHandlerBlocksNonMediaAndUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeStreamTestFile(t, root, "movie.mp4", "media")
	writeStreamTestFile(t, root, "notes.txt", "private")
	if err := os.Mkdir(filepath.Join(root, "folder.mp4"), 0o755); err != nil {
		t.Fatalf("mkdir media-like directory: %v", err)
	}

	handler := NewStreamHandler(streamTestConfig("videos", root))
	tests := []struct {
		name   string
		target string
	}{
		{name: "source root directory", target: "/stream/videos/"},
		{name: "non media file", target: "/stream/videos/notes.txt"},
		{name: "media-like directory", target: "/stream/videos/folder.mp4"},
		{name: "unknown source", target: "/stream/missing/movie.mp4"},
		{name: "dot segment", target: "/stream/videos/../movie.mp4"},
		{name: "encoded dot segment", target: "/stream/videos/%2e%2e/movie.mp4"},
		{name: "encoded backslash", target: "/stream/videos/folder%5Cmovie.mp4"},
		{name: "alternate data stream", target: "/stream/videos/movie.mp4%3Ahidden.mp4"},
		{name: "empty segment", target: "/stream/videos/folder//movie.mp4"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.target, nil))
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestStreamHandlerRejectsEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside.mp4")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside media: %v", err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(root, "linked.mp4")); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}

	recorder := httptest.NewRecorder()
	NewStreamHandler(streamTestConfig("videos", root)).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stream/videos/linked.mp4", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestStreamHandlerAllowsOnlyGetAndHead(t *testing.T) {
	root := t.TempDir()
	writeStreamTestFile(t, root, "movie.mp4", "media")
	recorder := httptest.NewRecorder()
	NewStreamHandler(streamTestConfig("videos", root)).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stream/videos/movie.mp4", strings.NewReader("ignored")))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("response = status %d Allow %q", recorder.Code, recorder.Header().Get("Allow"))
	}
}

func TestMediaContentType(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"movie.mkv":    "video/x-matroska",
		"movie.webm":   "video/webm",
		"song.mp3":     "audio/mpeg",
		"song.flac":    "audio/flac",
		"song.opus":    "audio/ogg",
		"song.mka":     "audio/x-matroska",
		"file.unknown": "application/octet-stream",
	}
	for filename, want := range tests {
		if got := MediaContentType(filename); got != want {
			t.Errorf("MediaContentType(%q) = %q, want %q", filename, got, want)
		}
	}
}

func streamTestConfig(sourceName, root string) *coreconfig.Config {
	return &coreconfig.Config{
		Sources:         []coreconfig.Source{{Name: sourceName, Path: root}},
		MediaExtensions: append([]string(nil), coreconfig.DefaultMediaExtensions...),
	}
}

func writeStreamTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir parent for %q: %v", relPath, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", relPath, err)
	}
}
