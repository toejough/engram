package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// --- mergeClusterReps ---

// TestMergeClusterReps_AddsClusterRepProvenance verifies that a cluster
// representative receives the cluster_rep provenance.
func TestMergeClusterReps_AddsClusterRepProvenance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	members := []string{"notes/a.md"}
	scores := []float32{0.8}
	contents := []string{"content A"}
	reps := map[int]int{0: 0}

	result := cli.ExportMergeClusterReps(members, scores, contents, reps)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Provenances).To(ContainElement("cluster_rep"))
}

// TestMergeClusterReps_MultipleReps verifies correct handling of multiple
// cluster representatives.
func TestMergeClusterReps_MultipleReps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	members := []string{"notes/a.md", "notes/b.md", "notes/c.md"}
	scores := []float32{0.9, 0.7, 0.5}
	contents := []string{"A", "B", "C"}
	// Clusters 0 and 1 each have a representative.
	reps := map[int]int{0: 0, 1: 2}

	result := cli.ExportMergeClusterReps(members, scores, contents, reps)

	g.Expect(result).To(HaveLen(2))

	paths := make([]string, 0, len(result))
	for _, entry := range result {
		paths = append(paths, entry.NotePath)
	}

	g.Expect(paths).To(ConsistOf("notes/a.md", "notes/c.md"))
}

// TestMergeClusterReps_SetsClusterID verifies that the representative has its
// clusterID set.
func TestMergeClusterReps_SetsClusterID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	members := []string{"notes/a.md"}
	scores := []float32{0.8}
	contents := []string{"content A"}
	reps := map[int]int{0: 0}

	result := cli.ExportMergeClusterReps(members, scores, contents, reps)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].ClusterID).NotTo(BeNil())

	if result[0].ClusterID == nil {
		return
	}

	g.Expect(*result[0].ClusterID).To(Equal(0))
}

// TestMergeClusterReps_SkipsOutOfBoundsMemberIdx verifies that an invalid
// member index (negative or >= len) is silently skipped.
func TestMergeClusterReps_SkipsOutOfBoundsMemberIdx(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	members := []string{"notes/a.md"}
	scores := []float32{0.8}
	contents := []string{"content A"}
	// memberIdx -1 and 99 are both out of range.
	reps := map[int]int{0: -1}

	result := cli.ExportMergeClusterReps(members, scores, contents, reps)

	g.Expect(result).To(BeEmpty(), "out-of-bounds memberIdx must be skipped")
}

// TestProvenanceRankFor_ClusterRep verifies the rank for the "cluster_rep" role.
func TestProvenanceRankFor_ClusterRep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	rank := cli.ExportProvenanceRankFor("cluster_rep")
	g.Expect(rank).To(Equal(2))
}

// --- provenanceRankFor ---

// TestProvenanceRankFor_Direct verifies the rank for the "direct" role.
func TestProvenanceRankFor_Direct(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	rank := cli.ExportProvenanceRankFor("direct")
	g.Expect(rank).To(Equal(3))
}

// TestProvenanceRankFor_Unknown verifies that an unknown role returns 0.
func TestProvenanceRankFor_Unknown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	rank := cli.ExportProvenanceRankFor("hub")
	g.Expect(rank).To(Equal(0))
}

// --- resolvedItemLess ---

// TestResolvedItemLess_MoreProvenancesWins verifies that more provenances
// beats fewer provenances regardless of score.
func TestResolvedItemLess_MoreProvenancesWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// 2 provenances + low score beats 1 provenance + high score.
	multi := cli.ExportNewNoteResolvedItemWithProvenances("a.md", 0.5, []string{"direct", "cluster_rep"})
	single := cli.ExportNewNoteResolvedItemWithProvenances("b.md", 0.99, []string{"direct"})

	result := cli.ExportResolvedItemLess(multi, single)
	g.Expect(result).To(BeTrue(), "more provenances wins over higher score")
}

// TestResolvedItemLess_SameCountHigherRankWins verifies that when provenance
// count ties, the higher-rank provenance wins.
func TestResolvedItemLess_SameCountHigherRankWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// direct (rank 3) beats cluster_rep (rank 2), even when cluster_rep has higher score.
	directItem := cli.ExportNewNoteResolvedItemWithProvenances("a.md", 0.5, []string{"direct"})
	clusterRepItem := cli.ExportNewNoteResolvedItemWithProvenances("b.md", 0.9, []string{"cluster_rep"})

	// Both have 1 provenance. direct (rank 3) beats cluster_rep (rank 2).
	result := cli.ExportResolvedItemLess(directItem, clusterRepItem)
	g.Expect(result).To(BeTrue(), "direct (rank 3) beats cluster_rep (rank 2) despite lower score")
}

// TestResolvedItemLess_ScoreBreaksTie verifies that higher score wins when
// provenance count and max-rank both tie.
func TestResolvedItemLess_ScoreBreaksTie(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	high := cli.ExportNewNoteResolvedItemWithScore("a.md", 0.9, 0.9)
	low := cli.ExportNewNoteResolvedItemWithScore("b.md", 0.1, 0.1)

	// Both have 0 provenances and 0 rank (same), so score decides.
	g.Expect(cli.ExportResolvedItemLess(high, low)).To(BeTrue())
	g.Expect(cli.ExportResolvedItemLess(low, high)).To(BeFalse())
}
