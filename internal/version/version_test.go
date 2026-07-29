package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevelopmentHandlerDoesNotRequireReleaseLookup(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	New("dev", nil).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/version", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var response Response
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Version != "dev" || response.ReleaseURL != ReleasePageURL || response.Update != nil {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	New("dev", nil).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/version", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	tests := []struct {
		first  string
		second string
		want   int
	}{
		{first: "0.1.3", second: "0.1.2", want: 1},
		{first: "0.1.2", second: "0.1.2", want: 0},
		{first: "0.1.2", second: "0.1.3", want: -1},
		{first: "v1.0.0", second: "0.9.9", want: 1},
		{first: "1.0.0", second: "v1.0.1", want: -1},
	}
	for _, test := range tests {
		if got := Compare(test.first, test.second); got != test.want {
			t.Fatalf("Compare(%q, %q) = %d, want %d", test.first, test.second, got, test.want)
		}
	}
}

func TestComparableAndNewer(t *testing.T) {
	t.Parallel()
	if IsComparable("dev") || IsComparable("0.1") || !IsComparable("0.1.2") {
		t.Fatal("unexpected comparable-version classification")
	}
	if !IsNewer("0.1.3", "0.1.2") || IsNewer("0.1.2", "0.1.2") {
		t.Fatal("unexpected newer-version comparison")
	}
}
