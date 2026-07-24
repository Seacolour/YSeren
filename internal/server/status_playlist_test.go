package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusHandlerReturnsSanitizedSourceInformation(t *testing.T) {
	root := t.TempDir()
	conf := streamTestConfig("media", root)
	conf.Server.Port = 1479
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.Host = "phone.local:1479"
	recorder := httptest.NewRecorder()
	New(conf, Options{}).Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.Bytes()
	var response StatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if response.State != "running" || response.Name != "YSeren" || response.Source != "media" || response.RootName != "media" || response.Port != 1479 {
		t.Fatalf("response = %#v", response)
	}
	if len(response.URLs) != 1 || response.URLs[0] != "http://phone.local:1479/" {
		t.Fatalf("urls = %#v", response.URLs)
	}
	if strings.Contains(string(body), root) {
		t.Fatalf("status response exposed source path %q", root)
	}
}

func TestStatusHandlerRejectsUnsupportedMethods(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(streamTestConfig("media", t.TempDir()), Options{}).Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/status", nil),
	)
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET" {
		t.Fatalf("response = status %d Allow %q", recorder.Code, recorder.Header().Get("Allow"))
	}
}

func TestPlaylistHandlerReturnsAbsoluteEncodedMediaURLs(t *testing.T) {
	root := t.TempDir()
	writeTreeTestFile(t, root, "Season 1/正片 #1.mp4", "video")
	writeTreeTestFile(t, root, "Music/theme.mp3", "audio")
	writeTreeTestFile(t, root, "archive.zip", "private")
	application := New(streamTestConfig("100% #media", root), Options{})

	request := httptest.NewRequest(http.MethodGet, "/playlist.m3u", nil)
	request.Host = "living-room.local:1479"
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "audio/x-mpegurl; charset=utf-8" {
		t.Fatalf("response = status %d Content-Type %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"#EXTM3U\n",
		"#EXTINF:-1,theme.mp3\nhttp://living-room.local:1479/stream/100%25%20%23media/Music/theme.mp3\n",
		"#EXTINF:-1,正片 #1.mp4\nhttp://living-room.local:1479/stream/100%25%20%23media/Season%201/%E6%AD%A3%E7%89%87%20%231.mp4\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("playlist missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "archive.zip") {
		t.Fatalf("playlist exposed non-media file:\n%s", body)
	}

	headRequest := httptest.NewRequest(http.MethodHead, "/playlist.m3u", nil)
	headRequest.Host = request.Host
	headRecorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(headRecorder, headRequest)
	if headRecorder.Code != http.StatusOK || headRecorder.Body.Len() != 0 || headRecorder.Header().Get("Content-Length") != recorder.Header().Get("Content-Length") {
		t.Fatalf("HEAD response = status %d length %q body %q", headRecorder.Code, headRecorder.Header().Get("Content-Length"), headRecorder.Body.String())
	}
}

func TestPlaylistHandlerRejectsUnsupportedMethods(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(streamTestConfig("media", t.TempDir()), Options{}).Handler().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/playlist.m3u", nil),
	)
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("response = status %d Allow %q", recorder.Code, recorder.Header().Get("Allow"))
	}
}
