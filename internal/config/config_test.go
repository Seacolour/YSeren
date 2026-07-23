package config

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestLoadConfigAutoPrefersCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "yseren.yml")
	if err := os.WriteFile(path, []byte("server:\n  port: 2480\nsources:\n  - path: D:/Media\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Chdir(dir)

	conf, usedPath, err := LoadConfigAuto("")
	if err != nil {
		t.Fatalf("LoadConfigAuto() error = %v", err)
	}
	if usedPath != "yseren.yml" {
		t.Fatalf("used path = %q, want yseren.yml", usedPath)
	}
	if conf.Server.Port != 2480 {
		t.Fatalf("port = %d, want 2480", conf.Server.Port)
	}
}

func TestLoadConfigAutoUsesExplicitPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(path, []byte("sources:\n  - path: D:/Explicit\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, usedPath, err := LoadConfigAuto(path)
	if err != nil {
		t.Fatalf("LoadConfigAuto() error = %v", err)
	}
	if usedPath != path || conf.Sources[0].Name != "Explicit" {
		t.Fatalf("result = path %q config %#v", usedPath, conf)
	}
}

func TestLoadConfigAddsAudioExtensionsToMediaList(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "yseren.yaml")
	content := []byte("sources:\n  - path: D:/Videos\naudio_extensions:\n  - opus\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !slices.Contains(conf.MediaExtensions, ".opus") || !conf.IsAudioFile("demo.opus") {
		t.Fatalf("audio extensions = media %v audio %v", conf.MediaExtensions, conf.AudioExtensions)
	}
}

func TestLoadConfigRecognizesKnownAudioExtensionFromMediaList(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "yseren.yaml")
	content := []byte("sources:\n  - path: D:/Videos\nmedia_extensions:\n  - .opus\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !conf.IsMediaFile("demo.opus") || !conf.IsAudioFile("demo.opus") {
		t.Fatalf(".opus classification = media %t audio %t", conf.IsMediaFile("demo.opus"), conf.IsAudioFile("demo.opus"))
	}
}

func TestLoadConfigAppliesDefaultsAndDerivesSourceName(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "yseren.yaml")
	if err := os.WriteFile(path, []byte("sources:\n  - path: D:/Media/Videos\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	conf, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if conf.Server.Port != DefaultPort {
		t.Fatalf("default port = %d, want %d", conf.Server.Port, DefaultPort)
	}
	if len(conf.Sources) != 1 || conf.Sources[0].Name != "Videos" {
		t.Fatalf("derived sources = %#v", conf.Sources)
	}
	if !conf.IsMediaFile("movie.MP4") || !conf.IsMediaFile("song.mp3") {
		t.Fatalf("default media extensions = %v", conf.MediaExtensions)
	}
}

func TestConfigValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		sources []Source
	}{
		{name: "invalid port", port: 70000},
		{name: "empty path", sources: []Source{{Name: "videos"}}},
		{name: "slash in name", sources: []Source{{Name: "a/b", Path: "D:/Videos"}}},
		{name: "dot traversal name", sources: []Source{{Name: "a..b", Path: "D:/Videos"}}},
		{
			name: "case insensitive duplicate",
			sources: []Source{
				{Name: "Videos", Path: "D:/Videos"},
				{Name: "videos", Path: "D:/Other"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := &Config{Server: ServerConfig{Port: test.port}, Sources: test.sources}
			if err := conf.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
		})
	}
}

func TestDefaultConfigMatchesLoadDefaults(t *testing.T) {
	t.Parallel()

	conf := DefaultConfig()
	if conf.Server.Port != DefaultPort {
		t.Fatalf("default port = %d, want %d", conf.Server.Port, DefaultPort)
	}
	if !conf.IsMediaFile("movie.mp4") || !conf.IsMediaFile("song.mp3") {
		t.Fatalf("default media extensions = %v", conf.MediaExtensions)
	}
	if len(conf.Sources) != 0 {
		t.Fatalf("default sources = %#v, want empty", conf.Sources)
	}
}

func TestSaveConfigCreatesDirectoryAndRoundTrips(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "yseren.yaml")
	conf := DefaultConfig()
	conf.Server.Port = 2480
	conf.Sources = []Source{{Path: "D:/Media/Videos"}}
	if err := SaveConfig(path, conf); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Server.Port != 2480 || len(loaded.Sources) != 1 || loaded.Sources[0].Name != "Videos" {
		t.Fatalf("loaded config = %#v", loaded)
	}
}

func TestSaveConfigReplacesExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "yseren.yaml")
	first := DefaultConfig()
	first.Server.Port = 1479
	second := DefaultConfig()
	second.Server.Port = 2480
	if err := SaveConfig(path, first); err != nil {
		t.Fatalf("first SaveConfig() error = %v", err)
	}
	if err := SaveConfig(path, second); err != nil {
		t.Fatalf("second SaveConfig() error = %v", err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Server.Port != 2480 {
		t.Fatalf("loaded port = %d, want 2480", loaded.Server.Port)
	}
}

func TestSaveConfigRejectsInvalidConfigWithoutCreatingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "yseren.yaml")
	conf := DefaultConfig()
	conf.Sources = []Source{{Name: "invalid/name", Path: "D:/Media"}}
	if err := SaveConfig(path, conf); err == nil {
		t.Fatal("SaveConfig() error = nil, want validation error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("saved invalid config, stat error = %v", err)
	}
}

func TestIsPathSafe(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "media")
	inside := filepath.Join(base, "season", "episode.mp4")
	sibling := filepath.Join(filepath.Dir(base), "media-private", "secret.mp4")
	if !IsPathSafe(base, base) || !IsPathSafe(base, inside) || IsPathSafe(base, sibling) {
		t.Fatalf("unexpected path safety result for base=%q", base)
	}

	caseVariantBase := filepath.Join(t.TempDir(), "Media")
	caseVariantPath := filepath.Join(filepath.Dir(caseVariantBase), "media", "episode.mp4")
	if runtime.GOOS == "windows" && !IsPathSafe(caseVariantBase, caseVariantPath) {
		t.Fatal("case-variant child should be safe on Windows")
	}
	if runtime.GOOS != "windows" && IsPathSafe(caseVariantBase, caseVariantPath) {
		t.Fatal("case-variant sibling should not be safe on a case-sensitive platform")
	}
}
