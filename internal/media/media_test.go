package media

import (
	"os"
	"path/filepath"
	"testing"
)

type testClassifier struct{}

func (testClassifier) IsMediaFile(filename string) bool {
	return HasExtension(filename, []string{".mp4", ".mp3"})
}

func (testClassifier) IsAudioFile(filename string) bool {
	return HasExtension(filename, []string{".mp3"})
}

func TestScanFiltersAndClassifiesMedia(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "Season 1/Episode.mp4", "video")
	writeTestFile(t, root, "Music/Theme.mp3", "audio")
	writeTestFile(t, root, "private.txt", "private")

	entries, err := Scan(root, testClassifier{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2: %#v", len(entries), entries)
	}

	byPath := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		byPath[entry.RelPath] = entry
	}
	if got := byPath["Season 1/Episode.mp4"]; got.MediaType != "video" || got.Size != int64(len("video")) {
		t.Fatalf("video entry = %#v", got)
	}
	if got := byPath["Music/Theme.mp3"]; got.MediaType != "audio" || got.Size != int64(len("audio")) {
		t.Fatalf("audio entry = %#v", got)
	}
}

func TestContentTypeUsesStableMediaMappings(t *testing.T) {
	t.Parallel()

	if got := ContentType("movie.mkv"); got != "video/x-matroska" {
		t.Fatalf("ContentType(mkv) = %q", got)
	}
	if got := ContentType("song.opus"); got != "audio/ogg" {
		t.Fatalf("ContentType(opus) = %q", got)
	}
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", relPath, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", relPath, err)
	}
}
