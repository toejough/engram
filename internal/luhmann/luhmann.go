// Package luhmann parses, compares, and sorts Luhmann zettelkasten IDs.
package luhmann

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode"
)

// Exported variables.
var (
	ErrEmpty         = errors.New("luhmann: empty ID")
	ErrLeadingLetter = errors.New("luhmann: ID must start with a digit")
)

// Less reports whether ID a sorts before ID b in tree order: parent before
// children, numeric segments compared numerically, alphabetic segments in
// Luhmann order (a..z, then aa..az, ba..bz, ..., zz, aaa, ...) — i.e.
// shorter letter segments sort before longer ones; within equal length, lex.
// This matches the z→aa rollover convention in nextLetter (internal/cli/luhmann.go).
func Less(a, b string) bool {
	aSegs, _ := ParseID(a)
	bSegs, _ := ParseID(b)

	for idx := 0; idx < len(aSegs) && idx < len(bSegs); idx++ {
		if aSegs[idx] == bSegs[idx] {
			continue
		}

		if unicode.IsDigit(rune(aSegs[idx][0])) {
			aNum, _ := strconv.Atoi(aSegs[idx])
			bNum, _ := strconv.Atoi(bSegs[idx])

			return aNum < bNum
		}

		if len(aSegs[idx]) != len(bSegs[idx]) {
			return len(aSegs[idx]) < len(bSegs[idx])
		}

		return aSegs[idx] < bSegs[idx]
	}

	return len(aSegs) < len(bSegs)
}

// ParseID splits a Luhmann ID into alternating digit/letter segments.
// "1a3b" → ["1", "a", "3", "b"]. "12ab3" → ["12", "ab", "3"]. The top-level
// segment must be digits.
func ParseID(id string) ([]string, error) {
	if id == "" {
		return nil, ErrEmpty
	}

	if !unicode.IsDigit(rune(id[0])) {
		return nil, fmt.Errorf("%w: %q", ErrLeadingLetter, id)
	}

	const initialCap = 4

	segments := make([]string, 0, initialCap)
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

// SortIDs sorts ids in tree order (mutates input).
func SortIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return Less(ids[i], ids[j])
	})
}
