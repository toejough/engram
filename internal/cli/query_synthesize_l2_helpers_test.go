package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// --- applyCombinedRecencyBand ---

// TestApplyCombinedRecencyBand_ChunksInactive verifies that when chunksActive
// is false, items are returned unchanged.
func TestApplyCombinedRecencyBand_ChunksInactive(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "", ""),
		cli.ExportNewNoteResolvedItem("b.md", "", ""),
	}

	out := cli.ExportApplyCombinedRecencyBand(items, nil, nil, 1, false)
	g.Expect(out).To(HaveLen(2), "chunksInactive: items returned unchanged")
}

// TestApplyCombinedRecencyBand_NilNowFn_ExceedsLimit verifies that when
// nowFn is nil and items exceed limit, a plain cap is applied.
func TestApplyCombinedRecencyBand_NilNowFn_ExceedsLimit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "", ""),
		cli.ExportNewNoteResolvedItem("b.md", "", ""),
		cli.ExportNewNoteResolvedItem("c.md", "", ""),
	}

	out := cli.ExportApplyCombinedRecencyBand(items, nil, nil, 2, true)
	g.Expect(out).To(HaveLen(2))
}

// TestApplyCombinedRecencyBand_NilNowFn_WithinLimit verifies that when nowFn
// is nil and items fit within limit, all items are returned.
func TestApplyCombinedRecencyBand_NilNowFn_WithinLimit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "", ""),
		cli.ExportNewNoteResolvedItem("b.md", "", ""),
	}

	out := cli.ExportApplyCombinedRecencyBand(items, nil, nil, 10, true)
	g.Expect(out).To(HaveLen(2))
}

// TestApplyCombinedRecencyBand_NowFnSet_NoEviction verifies that when nowFn
// is set and items fit within limit, the item is returned.
func TestApplyCombinedRecencyBand_NowFnSet_NoEviction(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	nowFn := func() time.Time { return time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC) }

	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "2026-06-20", ""),
	}

	out := cli.ExportApplyCombinedRecencyBand(items, nil, nowFn, 10, true)
	g.Expect(out).To(HaveLen(1))
	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.md"))
}

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

// TestMergeIntoExisting_CopiesCreatedWhenEmpty verifies created is copied
// from src when existing.created is empty.
func TestMergeIntoExisting_CopiesCreatedWhenEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.5, "", "")
	src := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.3, "", "2026-01-01")

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemCreated(existing)).To(Equal("2026-01-01"))
}

// TestMergeIntoExisting_CopiesInDegreeWhenNil verifies that inDegree is
// copied from src when existing.inDegree is nil.
func TestMergeIntoExisting_CopiesInDegreeWhenNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.5, "", "")
	src := cli.ExportNewNoteResolvedItemWithInDegree("note.md", 7)

	cli.ExportMergeIntoExisting(&existing, &src)

	inDegree := cli.ExportResolvedItemInDegree(existing)
	g.Expect(inDegree).NotTo(BeNil())

	if inDegree == nil {
		return
	}

	g.Expect(*inDegree).To(Equal(7))
}

// TestMergeIntoExisting_DoesNotOverwriteCreated verifies that an already-set
// created field is not overwritten by src.
func TestMergeIntoExisting_DoesNotOverwriteCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.5, "", "2025-01-01")
	src := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.3, "", "2026-01-01")

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemCreated(existing)).To(Equal("2025-01-01"))
}

// TestMergeIntoExisting_DoesNotOverwriteInDegree verifies that an already-set
// inDegree is not overwritten.
func TestMergeIntoExisting_DoesNotOverwriteInDegree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithInDegree("note.md", 3)
	src := cli.ExportNewNoteResolvedItemWithInDegree("note.md", 99)

	cli.ExportMergeIntoExisting(&existing, &src)

	inDegree := cli.ExportResolvedItemInDegree(existing)
	g.Expect(inDegree).NotTo(BeNil())

	if inDegree == nil {
		return
	}

	g.Expect(*inDegree).To(Equal(3))
}

// TestMergeIntoExisting_KeepsHigherExistingScore verifies score is not
// downgraded when existing.score > src.score.
func TestMergeIntoExisting_KeepsHigherExistingScore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithScore("note.md", 0.9, 0.9)
	src := cli.ExportNewNoteResolvedItemWithScore("note.md", 0.3, 0.3)

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemScore(existing)).To(Equal(float32(0.9)))
}

// --- mergeIntoExisting ---

// TestMergeIntoExisting_TakesHigherScore verifies score/content are updated
// when src.score > existing.score.
func TestMergeIntoExisting_TakesHigherScore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithScore("note.md", 0.5, 0.5)
	src := cli.ExportNewNoteResolvedItemWithScore("note.md", 0.9, 0.9)

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemScore(existing)).To(Equal(float32(0.9)))
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
