package cli

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/toejough/engram/internal/luhmann"
)

// unexported constants.
const (
	positionContinuation = "continuation"
	positionSibling      = "sibling"
	positionTop          = "top"
)

// unexported variables.
var (
	errLuhmannPosition = errors.New(
		"luhmann: position must be top, continuation, or sibling",
	)
	errLuhmannSiblingTopLevelMustBeTop = errors.New(
		"luhmann: sibling of top-level requires position=top",
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

		segs, parseErr := luhmann.ParseID(id)
		if parseErr != nil || len(segs) != depth+1 {
			continue
		}

		out = append(out, segs[depth])
	}

	return out
}

// maxDigitSeg returns the largest integer value among segs (0 if empty).
// Caller must guarantee every seg is non-empty all-digit (luhmann.ParseID-derived).
func maxDigitSeg(segs []string) int {
	maxN := 0

	for _, seg := range segs {
		n, _ := strconv.Atoi(seg) // safe: segs from luhmann.ParseID at digit position

		if n > maxN {
			maxN = n
		}
	}

	return maxN
}

// maxLetterSeg returns the lexically-largest letter segment ("" if none).
// maxLetterSeg returns the largest letter segment in Luhmann order — see
// luhmann.LetterLess for the ordering rule.
func maxLetterSeg(segs []string) string {
	maxL := ""

	for _, seg := range segs {
		if luhmann.LetterLess(maxL, seg) {
			maxL = seg
		}
	}

	return maxL
}

func nextChild(existing []string, parent string) (string, error) {
	parentSegs, err := luhmann.ParseID(parent)
	if err != nil {
		return "", fmt.Errorf("parsing parent: %w", err)
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
	for idx := range slices.Backward(runes) {
		if runes[idx] < 'z' {
			runes[idx]++

			return string(runes)
		}

		runes[idx] = 'a'
	}

	return "a" + string(runes)
}

// nextLuhmannID computes the next available Luhmann ID given existing IDs and a (target, position).
// position=top  → next available top-level (ignores target)
// position=continuation → next child of target
// position=sibling → next sibling of target (target must have a parent; for top-level use position=top)
func nextLuhmannID(existing []string, target, position string) (string, error) {
	switch position {
	case positionTop:
		return nextTopLevel(existing), nil
	case positionContinuation:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}

		return nextChild(existing, target)
	case positionSibling:
		if target == "" {
			return "", errLuhmannTargetEmpty
		}

		return nextSibling(existing, target)
	default:
		return "", fmt.Errorf("%w: got %q", errLuhmannPosition, position)
	}
}

func nextSibling(existing []string, target string) (string, error) {
	targetSegs, err := luhmann.ParseID(target)
	if err != nil {
		return "", fmt.Errorf("parsing target: %w", err)
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
		segs, err := luhmann.ParseID(id)
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
