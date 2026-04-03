package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConfigAddsAudioExtensionsToMediaList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "yseren.yaml")
	content := []byte("sources:\n  - path: D:/Videos\naudio_extensions:\n  - opus\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !slices.Contains(conf.MediaExtensions, ".opus") {
		t.Fatalf("expected .opus in MediaExtensions, got %v", conf.MediaExtensions)
	}
	if !conf.IsAudioFile("demo.opus") {
		t.Fatalf("expected .opus to be treated as audio")
	}
}

func TestLoadConfigRecognizesKnownAudioExtensionFromMediaList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "yseren.yaml")
	content := []byte("sources:\n  - path: D:/Videos\nmedia_extensions:\n  - .opus\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !conf.IsMediaFile("demo.opus") {
		t.Fatalf("expected .opus to be treated as media")
	}
	if !conf.IsAudioFile("demo.opus") {
		t.Fatalf("expected known audio extension .opus to use audio player")
	}
}
