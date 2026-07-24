package media

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestScanKeepsSafeFileSymlinksAndSkipsEscapingOnes(t *testing.T) {
	root := t.TempDir()
	insidePath := writeTestFile(t, root, "Music/target.mp3", "audio-target")
	outsidePath := writeTestFile(t, t.TempDir(), "outside.mp4", "outside")
	insideLink := filepath.Join(root, "inside-link.mp4")
	outsideLink := filepath.Join(root, "outside-link.mp4")
	if err := os.Symlink(insidePath, insideLink); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}
	if err := os.Symlink(outsidePath, outsideLink); err != nil {
		t.Fatalf("create escaping symlink: %v", err)
	}

	entries, err := Scan(root, testClassifier{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	byPath := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		byPath[entry.RelPath] = entry
	}
	if got := byPath["inside-link.mp4"]; got.Size != int64(len("audio-target")) || got.MediaType != "audio" {
		t.Fatalf("safe symlink entry = %#v", got)
	}
	if _, ok := byPath["outside-link.mp4"]; ok {
		t.Fatalf("escaping symlink was indexed: %#v", byPath["outside-link.mp4"])
	}
}

func TestScanSkipsUnreadableFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file ACLs are not represented by chmod mode bits")
	}

	root := t.TempDir()
	target := writeTestFile(t, root, "unreadable.mp4", "private")
	if err := os.Chmod(target, 0); err != nil {
		t.Fatalf("chmod unreadable fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o600) })
	if file, err := os.Open(target); err == nil {
		_ = file.Close()
		t.Skip("current user can still read a mode-000 file")
	}

	entries, err := Scan(root, testClassifier{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("unreadable entries = %#v, want none", entries)
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

func writeTestFile(t *testing.T, root, relPath, content string) string {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", relPath, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", relPath, err)
	}
	return target
}
