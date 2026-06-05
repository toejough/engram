package vaultgraph_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

// TestParseWikilinks_FenceMembershipProperty asserts the fence contract over a
// generated body: every wikilink placed OUTSIDE any fenced code block appears
// in the result, and every wikilink placed INSIDE a ```/~~~-fenced block never
// does. Inside and outside targets are drawn from disjoint namespaces (in%d vs
// out%d) so dedup cannot make an inside target "appear" via an outside twin,
// and every opened fence is closed with an identical marker so no fence leaks
// past its block to swallow later outside links.
func TestParseWikilinks_FenceMembershipProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		body, outside, inside := genFencedBody(rt)

		got := vaultgraph.ParseWikilinks([]byte(body))
		result := make(map[string]struct{}, len(got))

		for _, target := range got {
			result[target] = struct{}{}
		}

		for _, target := range outside {
			_, found := result[target]
			g.Expect(found).To(BeTrue(), "outside target %q missing from result; body:\n%s", target, body)
		}

		for _, target := range inside {
			_, found := result[target]
			g.Expect(found).To(BeFalse(), "inside target %q leaked into result; body:\n%s", target, body)
		}

		// No extras: every result target is an outside target (set equality).
		g.Expect(got).To(HaveLen(len(outside)))
	})
}

// genFencedBody builds a block-structured markdown body and returns it alongside
// the outside (must-appear) and inside (must-not-appear) wikilink targets. Each
// block is either a plain outside line carrying one out%d link, or a fenced
// block whose opener and closer are the same marker run and whose interior lines
// each carry one in%d link. Targets use disjoint index namespaces so no inside
// target can collide with an outside one; filler is space/letters only so no
// stray fence character opens a phantom block.
func genFencedBody(rt *rapid.T) (body string, outside, inside []string) {
	const (
		maxBlocks   = 6
		maxInside   = 4
		minFenceLen = 3
		maxFenceLen = 5
	)

	blockCount := rapid.IntRange(0, maxBlocks).Draw(rt, "blocks")

	var (
		lines      []string
		outsideIdx int
		insideIdx  int
	)

	for range blockCount {
		isFence := rapid.Bool().Draw(rt, "isFence")
		if !isFence {
			target := genTarget(rt, "out", &outsideIdx)
			outside = append(outside, target)
			lines = append(lines, genFiller(rt)+"[["+target+"]]"+genFiller(rt))

			continue
		}

		fenceChar := rapid.SampledFrom([]string{"`", "~"}).Draw(rt, "fenceChar")
		fenceLen := rapid.IntRange(minFenceLen, maxFenceLen).Draw(rt, "fenceLen")
		marker := strings.Repeat(fenceChar, fenceLen)

		lines = append(lines, marker)

		insideCount := rapid.IntRange(0, maxInside).Draw(rt, "insideCount")
		for range insideCount {
			target := genTarget(rt, "in", &insideIdx)
			inside = append(inside, target)
			lines = append(lines, genFiller(rt)+"[["+target+"]]"+genFiller(rt))
		}

		lines = append(lines, marker)
	}

	return strings.Join(lines, "\n"), outside, inside
}

// genFiller draws space/letter-only padding that cannot contain a fence
// character (backtick or tilde) and so cannot open or close a code block.
func genFiller(rt *rapid.T) string {
	return rapid.StringMatching(`[a-z ]{0,5}`).Draw(rt, "filler")
}

// genTarget returns a unique wikilink target by combining a namespace prefix
// with a monotonically increasing index, guaranteeing inside and outside
// targets never collide (which would defeat the dedup-aware membership check).
func genTarget(rt *rapid.T, namespace string, idx *int) string {
	suffix := rapid.StringMatching(`[a-zA-Z0-9]{0,4}`).Draw(rt, namespace+"Suffix")
	target := namespace + intToDecimal(*idx) + suffix
	*idx++

	return target
}

// intToDecimal renders a non-negative int as decimal digits; the value is small
// (bounded by block/inside counts) and kept local to avoid importing strconv
// just for the generator.
func intToDecimal(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte

	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
