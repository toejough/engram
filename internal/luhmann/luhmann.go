// Package luhmann parses, compares, and sorts Luhmann zettelkasten IDs.
package luhmann

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Exported variables.
var (
	ErrEmpty         = errors.New("luhmann: empty ID")
	ErrLeadingLetter = errors.New("luhmann: ID must start with a digit")
)

// FromBasename extracts the leading Luhmann ID from a basename of the form
// `<luhmann-id>.<YYYY-MM-DD>.<slug>` (no `.md` suffix). Returns ("", false)
// when the leading dot-segment is not a valid Luhmann ID: empty, doesn't
// start with a digit, or contains a non-digit non-letter rune. Letters are
// classified by unicode.IsLetter; this is the canonical character class for
// Luhmann IDs in this codebase. Callers with full filenames should strip
// the `.md` extension before calling.
func FromBasename(basename string) (string, bool) {
	dotIdx := strings.IndexByte(basename, '.')
	if dotIdx <= 0 {
		return "", false
	}

	candidate := basename[:dotIdx]
	if !unicode.IsDigit(rune(candidate[0])) {
		return "", false
	}

	for _, r := range candidate {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return "", false
		}
	}

	return candidate, true
}

// Less reports whether ID a sorts before ID b in tree order: parent before
// children, numeric segments compared numerically, alphabetic segments via
// LetterLess.
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

		return LetterLess(aSegs[idx], bSegs[idx])
	}

	return len(aSegs) < len(bSegs)
}

// LetterLess reports whether letter segment a sorts before b in Luhmann
// order: shorter segments first, then lex within equal length (a..z, then
// aa..az, ba..bz, ..., zz, aaa, ...). Matches the z→aa rollover convention
// in nextLetter (internal/cli/luhmann.go) and is the single source of truth
// for letter-segment ordering, shared by Less and by allocator code.
func LetterLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}

	return a < b
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
