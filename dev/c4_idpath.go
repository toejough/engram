//go:build targ

package dev

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IDPath is a hierarchical C4 element identifier (e.g. "S2-N3-M5").
type IDPath struct {
	Segments []string // raw segments, e.g. ["S2","N3","M5"]
	Level    int      // depth: 1..4 (number of segments)
}

// Append adds a new segment at the next level. The letter must match the next
// position in the S→N→M→P sequence, and number must be positive.
func (path IDPath) Append(letter string, number int) (IDPath, error) {
	if number <= 0 {
		return IDPath{}, fmt.Errorf("appending to %q: number must be positive, got %d", path.String(), number)
	}
	nextLevel := path.Level + 1
	if nextLevel > maxIDPathDepth {
		return IDPath{}, fmt.Errorf(
			"appending to %q: cannot extend beyond max depth %d",
			path.String(), maxIDPathDepth,
		)
	}
	expectedLetter := levelLetters[nextLevel-1]
	if letter != expectedLetter {
		return IDPath{}, fmt.Errorf(
			"appending to %q: level %d expects letter %q, got %q",
			path.String(), nextLevel, expectedLetter, letter,
		)
	}
	segment := letter + strconv.Itoa(number)
	newSegments := make([]string, 0, nextLevel)
	newSegments = append(newSegments, path.Segments...)
	newSegments = append(newSegments, segment)
	return IDPath{Segments: newSegments, Level: nextLevel}, nil
}

// IsAncestorOf reports whether this path is a strict prefix of other. It is
// not reflexive: a path is not an ancestor of itself.
func (path IDPath) IsAncestorOf(other IDPath) bool {
	if len(path.Segments) >= len(other.Segments) {
		return false
	}
	for index, segment := range path.Segments {
		if other.Segments[index] != segment {
			return false
		}
	}
	return true
}

// String returns the canonical hyphen-separated form (e.g. "S2-N3-M5").
func (path IDPath) String() string {
	return strings.Join(path.Segments, "-")
}

// ParseIDPath parses a hyphen-separated hierarchical path string. The first
// segment must start with S; each subsequent segment uses the next letter in
// the fixed sequence S, N, M, P. Numbers must be positive integers without
// letter suffixes. Rejects empty input, too-deep paths, wrong case, skipped
// letters, missing S, and legacy flat IDs (e.g. "E27").
func ParseIDPath(input string) (IDPath, error) {
	if input == "" {
		return IDPath{}, fmt.Errorf("parsing id path: empty input")
	}
	segments := strings.Split(input, "-")
	if len(segments) > maxIDPathDepth {
		return IDPath{}, fmt.Errorf(
			"parsing id path %q: %d segments exceeds max depth %d",
			input, len(segments), maxIDPathDepth,
		)
	}
	for index, segment := range segments {
		match := segmentPattern.FindStringSubmatch(segment)
		if match == nil {
			return IDPath{}, fmt.Errorf(
				"parsing id path %q: segment %q does not match <Letter><PositiveInt>",
				input, segment,
			)
		}
		expectedLetter := levelLetters[index]
		if match[1] != expectedLetter {
			return IDPath{}, fmt.Errorf(
				"parsing id path %q: segment %q has letter %q but expected %q at depth %d",
				input, segment, match[1], expectedLetter, index+1,
			)
		}
	}
	return IDPath{Segments: segments, Level: len(segments)}, nil
}

// unexported constants.
const (
	maxIDPathDepth = 4
)

// unexported variables.
var (
	levelLetters   = []string{"S", "N", "M", "P"}
	segmentPattern = regexp.MustCompile(`^([SNMP])([1-9][0-9]*)$`)
)
