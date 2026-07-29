// Package releasemeta owns the strict version contract shared by release jobs.
package releasemeta

import (
	"fmt"
	"regexp"
	"strconv"
)

const maxVersionPart = 999

var releaseTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// Metadata is the normalized information derived from one release tag.
type Metadata struct {
	Tag                string
	Version            string
	AndroidVersionCode int
}

// Parse validates a strict vMAJOR.MINOR.PATCH tag and derives values consumed
// by Headless, Desktop and Android release builds.
func Parse(tag string) (Metadata, error) {
	matches := releaseTagPattern.FindStringSubmatch(tag)
	if matches == nil {
		return Metadata{}, fmt.Errorf("release tag %q must match vMAJOR.MINOR.PATCH without leading zeroes", tag)
	}

	parts := make([]int, 3)
	for index, raw := range matches[1:] {
		value, err := strconv.Atoi(raw)
		if err != nil || value > maxVersionPart {
			return Metadata{}, fmt.Errorf("release tag %q contains a version component outside 0..%d", tag, maxVersionPart)
		}
		parts[index] = value
	}

	androidVersionCode := parts[0]*1_000_000 + parts[1]*1_000 + parts[2]
	if androidVersionCode < 1 {
		return Metadata{}, fmt.Errorf("release tag %q produces an invalid Android versionCode", tag)
	}

	return Metadata{
		Tag:                tag,
		Version:            tag[1:],
		AndroidVersionCode: androidVersionCode,
	}, nil
}
