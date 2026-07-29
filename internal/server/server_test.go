package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	coreconfig "yseren/internal/config"
)

func TestApplicationUsesIndependentServeMux(t *testing.T) {
	originalDefault := http.DefaultServeMux
	isolate := http.NewServeMux()
	http.DefaultServeMux = isolate
	t.Cleanup(func() { http.DefaultServeMux = originalDefault })

	application := New(&coreconfig.Config{}, Options{})
	request := httptest.NewRequest(http.MethodGet, "/api/tree", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("application status = %d, want 200", recorder.Code)
	}

	_, pattern := isolate.Handler(request)
	if pattern != "" {
		t.Fatalf("DefaultServeMux unexpectedly registered pattern %q", pattern)
	}
}

func TestApplicationsDoNotShareSourceCaches(t *testing.T) {
	t.Parallel()

	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	writeServerTestFile(t, firstRoot, "first.mp4")
	writeServerTestFile(t, secondRoot, "second.mp4")

	first := New(serverTestConfig(firstRoot), Options{})
	second := New(serverTestConfig(secondRoot), Options{})
	firstTree := requestTree(t, first)
	secondTree := requestTree(t, second)
	if !treeContains(firstTree.Root, "first.mp4") || treeContains(firstTree.Root, "second.mp4") {
		t.Fatalf("first application tree = %#v", firstTree.Root)
	}
	if !treeContains(secondTree.Root, "second.mp4") || treeContains(secondTree.Root, "first.mp4") {
		t.Fatalf("second application tree = %#v", secondTree.Root)
	}
}

func serverTestConfig(root string) *coreconfig.Config {
	return &coreconfig.Config{
		Sources:         []coreconfig.Source{{Name: "media", Path: root}},
		MediaExtensions: []string{".mp4"},
	}
}

func requestTree(t *testing.T, application *Application) TreeResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/tree", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("tree status = %d", recorder.Code)
	}
	var response TreeResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode tree: %v", err)
	}
	return response
}

func treeContains(node *TreeNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Name == name {
		return true
	}
	for _, child := range node.Children {
		if treeContains(child, name) {
			return true
		}
	}
	return false
}

func writeServerTestFile(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
		t.Fatalf("write %q: %v", name, err)
	}
}
