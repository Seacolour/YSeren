package main

import "testing"

func TestCompareVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.3", "0.1.2", 1},
		{"0.1.2", "0.1.2", 0},
		{"0.1.2", "0.1.3", -1},
		{"v1.0.0", "0.9.9", 1},
		{"1.0.0", "v1.0.1", -1},
	}

	for _, tc := range cases {
		got := compareVersion(tc.a, tc.b)
		if got != tc.want {
			t.Fatalf("compareVersion(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsComparableVersion(t *testing.T) {
	t.Parallel()

	if isComparableVersion("dev") {
		t.Fatal("dev should not be comparable")
	}
	if !isComparableVersion("0.1.2") {
		t.Fatal("0.1.2 should be comparable")
	}
	if isComparableVersion("0.1") {
		t.Fatal("0.1 should not be comparable")
	}
}

func TestIsNewerVersion(t *testing.T) {
	t.Parallel()

	if !isNewerVersion("0.1.3", "0.1.2") {
		t.Fatal("expected 0.1.3 to be newer than 0.1.2")
	}
	if isNewerVersion("0.1.2", "0.1.2") {
		t.Fatal("same version should not be newer")
	}
}
