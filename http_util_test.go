package main

import "testing"

func TestBuildStreamURLEscapesSourceAndPath(t *testing.T) {
	t.Parallel()

	got := buildStreamURL("100% #动漫", "Season 1/Ep #1.mp4")
	want := "/stream/100%25%20%23%E5%8A%A8%E6%BC%AB/Season%201/Ep%20%231.mp4"
	if got != want {
		t.Fatalf("buildStreamURL() = %q, want %q", got, want)
	}
}
