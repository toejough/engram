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

// IsAncestorOrEqual reports whether this path is a prefix of other or equal
// to other. It is reflexive: a path is an ancestor-or-equal of itself.
func (path IDPath) IsAncestorOrEqual(other IDPath) bool {
	if len(path.Segments) > len(other.Segments) {
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

// Anchor returns the canonical HTML anchor for an element: lowercase id, a
// hyphen, then the slug of name (e.g. Anchor("S2-N3-M5", "Recall") →
// "s2-n3-m5-recall").
func Anchor(id, name string) string {
	return strings.ToLower(id) + "-" + slug(name)
}

// LocalLetter returns the letter that a spec at the given level allocates for
// new elements: level 1→S, 2→N, 3→M, 4→P. Returns an error for out-of-range
// levels.
func LocalLetter(level int) (string, error) {
	if level < 1 || level > maxIDPathDepth {
		return "", fmt.Errorf("LocalLetter: level %d out of range [1..%d]", level, maxIDPathDepth)
	}
	return levelLetters[level-1], nil
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

// ValidateDiagramNodeID returns nil iff id is acceptable as a node in a
// diagram whose focus is focus. Valid shapes:
//   - identical to focus
//   - any path shallower than focus (carried-over peers from parent diagrams)
//   - sibling at focus's depth (same depth, shares focus's parent prefix)
//
// Descendants of focus are rejected.
func ValidateDiagramNodeID(focus IDPath, id string) error {
	path, err := ParseIDPath(id)
	if err != nil {
		return fmt.Errorf("id %q: %w", id, err)
	}
	if path.String() == focus.String() {
		return nil
	}
	// Any path shallower than focus: carried-over peer (any system, container).
	if path.Level < focus.Level {
		return nil
	}
	// Sibling: same depth, shares all but last segment with focus.
	if path.Level == focus.Level && sharesParentPath(path, focus) {
		return nil
	}
	return fmt.Errorf(
		"id %q is not valid as a diagram node for focus %q: must be focus, ancestor, or sibling",
		id, focus.String(),
	)
}

// ValidateElementID returns nil iff id is acceptable for an element in a spec
// at the given level whose focus is focus. Valid shapes:
//   - any path shallower than focus's depth (carry-over from parent diagrams)
//   - identical to focus (the focus element itself)
//   - a new local: exactly one level deeper than focus, under focus, using
//     LocalLetter(level) (e.g. focus "S2-N3" at level 3 → "S2-N3-M<n>")
//
// For level 1 the focus is the empty IDPath (depth 0); accepted shapes are
// "S<n>" (depth 1 = focus.Level+1).
func ValidateElementID(level int, focus IDPath, id string) error {
	letter, err := LocalLetter(level)
	if err != nil {
		return err
	}
	path, err := ParseIDPath(id)
	if err != nil {
		return fmt.Errorf("id %q: %w", id, err)
	}
	// Any path at or shallower than the focus depth is a carried-over peer —
	// always OK (other systems, containers, etc. from parent diagrams).
	if path.Level <= focus.Level {
		return nil
	}
	// New local element: exactly one level deeper than focus, under focus,
	// using the expected letter for this spec level.
	if path.Level == focus.Level+1 && focus.IsAncestorOrEqual(path) &&
		path.Segments[path.Level-1][0:1] == letter {
		return nil
	}
	return fmt.Errorf(
		"id %q is not valid at level %d (focus %q): must be a carry-over (depth ≤ %d), "+
			"or %s<n> directly under focus",
		id, level, focus.String(), focus.Level, letter,
	)
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
