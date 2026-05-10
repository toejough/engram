package cli

import (
	"errors"
	"fmt"
	"unicode"
)

// unexported variables.
var (
	errLuhmannEmpty         = errors.New("luhmann: empty ID")
	errLuhmannLeadingLetter = errors.New("luhmann: ID must start with a digit")
)

// parseLuhmannID splits a Luhmann ID into alternating digit/letter segments.
// "1a3b" → ["1", "a", "3", "b"]. "12ab3" → ["12", "ab", "3"]. Top-level segment
// must be digits.
func parseLuhmannID(id string) ([]string, error) {
	if id == "" {
		return nil, errLuhmannEmpty
	}

	if !unicode.IsDigit(rune(id[0])) {
		return nil, fmt.Errorf("%w: %q", errLuhmannLeadingLetter, id)
	}

	segments := make([]string, 0, 4) //nolint:mnd // initial capacity hint
	current := []rune{rune(id[0])}
	currentIsDigit := unicode.IsDigit(rune(id[0]))

	for _, r := range id[1:] {
		isDigit := unicode.IsDigit(r)
		if isDigit == currentIsDigit {
			current = append(current, r)
			continue
		}

		segments = append(segments, string(current))
		current = []rune{r}
		currentIsDigit = isDigit
	}

	segments = append(segments, string(current))

	return segments, nil
}
