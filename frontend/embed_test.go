package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesIndexAndSPAFallback(t *testing.T) {
	t.Parallel()

	handler := Handler()
	for _, path := range []string{"/", "/library/folder/"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, recorder.Code)
		}
		if !strings.Contains(strings.ToLower(recorder.Body.String()), "<!doctype html>") {
			t.Fatalf("%s did not return index.html", path)
		}
	}
}

func TestHandlerDoesNotSwallowAPIRoutes(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("API status = %d, want 404", recorder.Code)
	}
}

func TestErrorHandlerEscapesMessage(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	ErrorHandler("<script>alert(1)</script>").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	body := recorder.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") || !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("error page did not escape message: %q", body)
	}
}
