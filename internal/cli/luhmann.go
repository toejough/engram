package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// unexported constants.
const (
	relationContinuation = "continuation"
	relationSibling      = "sibling"
	relationTop          = "top"
)

// unexported variables.
var (
	errLuhmannEmpty                    = errors.New("luhmann: empty ID")
	errLuhmannLeadingLetter            = errors.New("luhmann: ID must start with a digit")
	errLuhmannRelation                 = errors.New("luhmann: relation must be top, continuation, or sibling")
	errLuhmannSiblingTopLevelMustBeTop = errors.New(
		"luhmann: sibling of top-level requires relation=top",
	)
	errLuhmannTargetEmpty = errors.New("luhmann: target required for continuation/sibling")
)

// directChildSegments returns the trailing seg of every direct child of parent
// (existing IDs that begin with parent and have exactly depth+1 segments).
func directChildSegments(existing []string, parent string, depth int) []string {
	out := make([]string, 0, len(existing))

	for _, id := range existing {
		if !strings.HasPrefix(id, parent) || id == parent {
			continue
		}

		segs, parseErr := parseLuhmannID(id)
		if parseErr != nil || len(segs) != depth+1 {
			continue
		}

		out = append(out, segs[depth])
	}

	return out
}

func luhmannLess(a, b string) bool {
	aSegs, _ := parseLuhmannID(a)
	bSegs, _ := parseLuhmannID(b)

	for idx := 0; idx < len(aSegs) && idx < len(bSegs); idx++ {
		if aSegs[idx] == bSegs[idx] {
			continue
		}

		aIsDigit := unicode.IsDigit(rune(aSegs[idx][0]))
		if aIsDigit {
			aNum, _ := strconv.Atoi(aSegs[idx])
			bNum, _ := strconv.Atoi(bSegs[idx])

			return aNum < bNum
		}

		return aSegs[idx] < bSegs[idx]
	}

	return len(aSegs) < len(bSegs)
}

// maxDigitSeg returns the largest integer value among segs (0 if empty).
// Caller must guarantee every seg is non-empty all-digit (parseLuhmannID-derived).
func maxDigitSeg(segs []string) int {
	maxN := 0

	for _, seg := range segs {
		n, _ := strconv.Atoi(seg) // safe: segs from parseLuhmannID at digit position

		if n > maxN {
			maxN = n
		}
	}

	return maxN
}

// maxLetterSeg returns the lexically-largest letter segment ("" if none).
func maxLetterSeg(segs []string) string {
	maxL := ""

	for _, seg := range segs {
		if seg > maxL {
			maxL = seg
		}
	}

	return maxL
}

func nextChild(existing []string, parent string) (string, error) {
	parentSegs, err := parseLuhmannID(parent)
	if err != nil {
		return "", err
	}

	depth := len(parentSegs)
	// Parent depth 1 (top, e.g. "1") → letter child ("1a"). Depth 2 ("1a") → digit child ("1a1").
	const evenDepthMod = 2

	childSegments := directChildSegments(existing, parent, depth)

	if depth%evenDepthMod == 0 {
		return parent + strconv.Itoa(maxDigitSeg(childSegments)+1), nil
	}

	return parent + nextLetter(maxLetterSeg(childSegments)), nil
}

// nextLetter returns "a" if cur is empty, else the next letter ("a"→"b", "z"→"aa", "az"→"ba").
func nextLetter(cur string) string {
	if cur == "" {
		return "a"
	}

	runes := []rune(cur)
	for idx := len(runes) - 1; idx >= 0; idx-- {
		if runes[idx] < 'z' {
			runes[idx]++

			return string(runes)
		}

		runes[idx] = 'a'
	}

	return "a" + string(runes)
}

// nextLuhmannID computes the next available Luhmann ID given existing IDs and a (target, relation).
// relation=top  → next available top-level (ignores target)
// relation=continuation → next child of target
// relation=sibling → next sibling of target (target must have a parent; for top-level use relation=top)
func nextLuhmannID(existing []string, target, relation string) (string, error) {
	switch relation {
	case relationTop:
		return nextTopLevel(existing), nil
	case relationContinuation:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}

		return nextChild(existing, target)
	case relationSibling:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}

		return nextSibling(existing, target)
	default:
		return "", fmt.Errorf("%w: got %q", errLuhmannRelation, relation)
	}
}

func nextSibling(existing []string, target string) (string, error) {
	targetSegs, err := parseLuhmannID(target)
	if err != nil {
		return "", err
	}

	if len(targetSegs) == 1 {
		return "", fmt.Errorf("%w: %q", errLuhmannSiblingTopLevelMustBeTop, target)
	}

	parent := strings.Join(targetSegs[:len(targetSegs)-1], "")

	return nextChild(existing, parent)
}

func nextTopLevel(existing []string) string {
	maxN := 0

	for _, id := range existing {
		segs, err := parseLuhmannID(id)
		if err != nil || len(segs) != 1 {
			continue
		}

		n, atoiErr := strconv.Atoi(segs[0])
		if atoiErr == nil && n > maxN {
			maxN = n
		}
	}

	return strconv.Itoa(maxN + 1)
}

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

// sortLuhmannIDs sorts in tree order: parent before children, numeric segments
// compared numerically, alphabetic segments compared lexically. Mutates the input.
func sortLuhmannIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return luhmannLess(ids[i], ids[j])
	})
}
