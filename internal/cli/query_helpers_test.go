package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestAppendUniqueProvenance_AddsNewRole exercises the append branch.
func TestAppendUniqueProvenance_AddsNewRole(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance(nil, "direct")
	g.Expect(got).To(Equal([]string{"direct"}))
}

// TestAppendUniqueProvenance_AppendsDistinctRoles exercises both branches.
func TestAppendUniqueProvenance_AppendsDistinctRoles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance(nil, "direct", "cluster_rep", "hub", "direct")
	g.Expect(got).To(Equal([]string{"direct", "cluster_rep", "hub"}))
}

// TestAppendUniqueProvenance_DedupsExisting exercises the dedup branch.
func TestAppendUniqueProvenance_DedupsExisting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance([]string{"direct"}, "direct", "direct")
	g.Expect(got).To(Equal([]string{"direct"}))
}

// TestBreakRepresentativeTie_HigherScoreWins exercises the score-tiebreak branch.
func TestBreakRepresentativeTie_HigherScoreWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	winner := cli.ExportBreakRepresentativeTie(0.9, "A.md", 0.5, "B.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_LexicographicPath exercises the final path tiebreak.
func TestBreakRepresentativeTie_LexicographicPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Same score → "A.md" < "B.md" → A wins.
	winner := cli.ExportBreakRepresentativeTie(0.5, "A.md", 0.5, "B.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_LexicographicPathReverse exercises the default branch.
func TestBreakRepresentativeTie_LexicographicPathReverse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Same score, "Z" > "A" → B wins.
	winner := cli.ExportBreakRepresentativeTie(0.5, "Z.md", 0.5, "A.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_SecondHigherWins exercises the reverse score-tiebreak branch.
func TestBreakRepresentativeTie_SecondHigherWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	winner := cli.ExportBreakRepresentativeTie(0.5, "A.md", 0.9, "B.md")
	g.Expect(winner).To(Equal("B.md"))
}
