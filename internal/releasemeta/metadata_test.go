package releasemeta

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	metadata, err := Parse("v0.2.0")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if metadata.Tag != "v0.2.0" || metadata.Version != "0.2.0" || metadata.AndroidVersionCode != 2000 {
		t.Fatalf("Parse() = %#v", metadata)
	}
}

func TestParseRejectsInvalidTags(t *testing.T) {
	t.Parallel()

	for _, tag := range []string{
		"0.2.0",
		"v0.2",
		"v0.2.0-beta.1",
		"v00.2.0",
		"v0.02.0",
		"v0.0.0",
		"v1000.0.0",
		"latest",
	} {
		tag := tag
		t.Run(tag, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(tag); err == nil {
				t.Fatalf("Parse(%q) unexpectedly succeeded", tag)
			}
		})
	}
}
