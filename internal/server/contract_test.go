package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTreeResponseV1ContractFixture(t *testing.T) {
	var response TreeResponse
	readContractFixture(t, "tree-response.v1.json", &response)
	if response.GeneratedAt <= 0 || response.Root == nil {
		t.Fatalf("invalid top-level fixture: %#v", response)
	}
	video := findTreeNodeByName(response.Root, "Example Movie.mp4")
	if video == nil || video.Type != "file" || video.Source != "videos" ||
		video.RelPath != "Movies/Example Movie.mp4" ||
		video.URL != "/stream/videos/Movies/Example%20Movie.mp4" ||
		video.Size != 1048576 || video.ModTime != 1699999000 || video.MediaType != "video" {
		t.Fatalf("invalid video fixture node: %#v", video)
	}
	audio := findTreeNodeByName(response.Root, "Theme Song.mp3")
	if audio == nil || audio.MediaType != "audio" || audio.ModTime != 1699998000 {
		t.Fatalf("invalid audio fixture node: %#v", audio)
	}
}

func TestStreamHandlerMatchesRangeV1ContractFixture(t *testing.T) {
	var fixture struct {
		Version  int `json:"version"`
		Resource struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		} `json:"resource"`
		Cases []struct {
			Name          string  `json:"name"`
			Method        string  `json:"method"`
			Range         string  `json:"range"`
			Status        int     `json:"status"`
			Body          *string `json:"body,omitempty"`
			ContentRange  string  `json:"contentRange"`
			ContentLength string  `json:"contentLength,omitempty"`
		} `json:"cases"`
	}
	readContractFixture(t, "range-cases.v1.json", &fixture)
	if fixture.Version != 1 {
		t.Fatalf("fixture version = %d, want 1", fixture.Version)
	}

	root := t.TempDir()
	writeStreamTestFile(t, root, fixture.Resource.Name, fixture.Resource.Content)
	handler := NewStreamHandler(streamTestConfig("contract", root))
	target := BuildStreamURL("contract", fixture.Resource.Name)
	for _, test := range fixture.Cases {
		t.Run(test.Name, func(t *testing.T) {
			request := httptest.NewRequest(test.Method, target, nil)
			if test.Range != "" {
				request.Header.Set("Range", test.Range)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.Status {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, test.Status, recorder.Body.String())
			}
			if test.Body != nil && recorder.Body.String() != *test.Body {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), *test.Body)
			}
			if got := recorder.Header().Get("Content-Range"); got != test.ContentRange {
				t.Fatalf("Content-Range = %q, want %q", got, test.ContentRange)
			}
			if test.ContentLength != "" && recorder.Header().Get("Content-Length") != test.ContentLength {
				t.Fatalf("Content-Length = %q, want %q", recorder.Header().Get("Content-Length"), test.ContentLength)
			}
			if recorder.Code != http.StatusRequestedRangeNotSatisfiable && recorder.Header().Get("Accept-Ranges") != "bytes" {
				t.Fatalf("Accept-Ranges = %q, want bytes", recorder.Header().Get("Accept-Ranges"))
			}
		})
	}
}

func readContractFixture(t *testing.T, name string, target any) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "contracts", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract fixture %q: %v", name, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode contract fixture %q: %v", name, err)
	}
}
