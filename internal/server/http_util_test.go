package server

import "testing"

func TestBuildStreamURLEscapesSourceAndPath(t *testing.T) {
	t.Parallel()
	got := BuildStreamURL("100% #动漫", "Season 1/Ep #1.mp4")
	want := "/stream/100%25%20%23%E5%8A%A8%E6%BC%AB/Season%201/Ep%20%231.mp4"
	if got != want {
		t.Fatalf("BuildStreamURL() = %q, want %q", got, want)
	}
}

func TestStreamRoutePatternEscapesSource(t *testing.T) {
	t.Parallel()
	got := StreamRoutePattern("100% #动漫")
	want := "/stream/100%25%20%23%E5%8A%A8%E6%BC%AB/"
	if got != want {
		t.Fatalf("StreamRoutePattern() = %q, want %q", got, want)
	}
}
