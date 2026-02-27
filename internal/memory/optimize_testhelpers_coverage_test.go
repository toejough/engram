//go:build sqlite_fts5

package memory_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestCheckContextForTest_CanceledContext verifies CheckContextForTest returns error when context is canceled.
func TestCheckContextForTest_CanceledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := memory.OptimizeOpts{Context: ctx}

	err := memory.CheckContextForTest(opts)

	g.Expect(err).To(HaveOccurred())
}

// TestCheckContextForTest_NilContext verifies CheckContextForTest returns nil when context is nil.
func TestCheckContextForTest_NilContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	opts := memory.OptimizeOpts{}

	err := memory.CheckContextForTest(opts)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestClusterEntriesToEmbeddingsForTest_EmptyCluster verifies empty cluster returns empty slice.
func TestClusterEntriesToEmbeddingsForTest_EmptyCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.ClusterEntriesToEmbeddingsForTest(nil)

	g.Expect(result).To(BeEmpty())
}

// TestClusterEntriesToEmbeddingsForTest_WithEntries verifies cluster entries are converted to embeddings.
func TestClusterEntriesToEmbeddingsForTest_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "always use TDD"},
		{ID: 2, Content: "test before implement"},
	}

	result := memory.ClusterEntriesToEmbeddingsForTest(cluster)

	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].ID).To(Equal(int64(1)))
	g.Expect(result[0].Content).To(Equal("always use TDD"))
}

// TestClusterHasExistingSkillForTest_Match verifies true returned when cluster member is in existingIDs.
func TestClusterHasExistingSkillForTest_Match(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 5, Content: "test"}}
	existing := map[int64]bool{5: true}

	result := memory.ClusterHasExistingSkillForTest(cluster, existing)

	g.Expect(result).To(BeTrue())
}

// TestClusterHasExistingSkillForTest_NoMatch verifies false returned when no cluster member in existingIDs.
func TestClusterHasExistingSkillForTest_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 1, Content: "test"}}
	existing := map[int64]bool{99: true}

	result := memory.ClusterHasExistingSkillForTest(cluster, existing)

	g.Expect(result).To(BeFalse())
}

// TestCosineSimilarityForTest_DifferentLengths verifies 0.0 returned for mismatched vector lengths.
func TestCosineSimilarityForTest_DifferentLengths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.CosineSimilarityForTest([]float32{1.0}, []float32{1.0, 2.0})

	g.Expect(result).To(Equal(0.0))
}

// TestCosineSimilarityForTest_SameVector verifies similarity is 1.0 for identical vectors.
func TestCosineSimilarityForTest_SameVector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vec := []float32{1.0, 0.0, 0.0}

	result := memory.CosineSimilarityForTest(vec, vec)

	g.Expect(result).To(BeNumerically("~", 1.0, 0.001))
}

// TestDefaultSkillTesterForTest_ReturnsNonNil verifies a non-nil SkillTester is returned.
func TestDefaultSkillTesterForTest_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tester := memory.DefaultSkillTesterForTest("fake-key")

	g.Expect(tester).ToNot(BeNil())
}

// TestFormatClusterSourceIDsForTest_EmptyCluster verifies empty cluster returns empty JSON array.
func TestFormatClusterSourceIDsForTest_EmptyCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.FormatClusterSourceIDsForTest(nil)

	g.Expect(result).To(Equal("[]"))
}

// TestFormatClusterSourceIDsForTest_WithEntries verifies cluster IDs are formatted as JSON array.
func TestFormatClusterSourceIDsForTest_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 10}, {ID: 20}}

	result := memory.FormatClusterSourceIDsForTest(cluster)

	g.Expect(result).To(Equal("[10,20]"))
}

// TestGenerateThemeFromClusterForTest_EmptyCluster verifies "Unknown" returned for empty cluster.
func TestGenerateThemeFromClusterForTest_EmptyCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.GenerateThemeFromClusterForTest(nil)

	g.Expect(result).To(Equal("Unknown"))
}

// TestGenerateThemeFromClusterForTest_WithEntry verifies theme is derived from first entry content.
func TestGenerateThemeFromClusterForTest_WithEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 1, Content: "always use TDD when writing code"}}

	result := memory.GenerateThemeFromClusterForTest(cluster)

	g.Expect(result).ToNot(BeEmpty())
	g.Expect(result).To(ContainSubstring("TDD"))
}

// TestGetExistingPatternsForTest_EmptyDB verifies empty slice returned from empty database.
func TestGetExistingPatternsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	patterns := memory.GetExistingPatternsForTest(db)

	g.Expect(patterns).To(BeEmpty())
}

// TestGetExistingSkillSourceIDsForTest_EmptyDB verifies empty map returned from empty database.
func TestGetExistingSkillSourceIDsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	ids, err := memory.GetExistingSkillSourceIDsForTest(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids).To(BeEmpty())
}

// TestHasLearnTimestampPrefixForTest_NoPrefix verifies false returned for regular content.
func TestHasLearnTimestampPrefixForTest_NoPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.HasLearnTimestampPrefixForTest("always use TDD")

	g.Expect(result).To(BeFalse())
}

// TestHasLearnTimestampPrefixForTest_ValidPrefix verifies true returned for timestamp format.
func TestHasLearnTimestampPrefixForTest_ValidPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := memory.HasLearnTimestampPrefixForTest("- 2026-02-10 15:04: do the thing")

	g.Expect(result).To(BeTrue())
}

// TestIsNarrowByKeywordsForTest_NarrowLearning verifies narrow learning is detected.
func TestIsNarrowByKeywordsForTest_NarrowLearning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isNarrow, reason := memory.IsNarrowByKeywordsForTest("fix ISSUE-152 in projctl")

	g.Expect(isNarrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

// TestIsNarrowByKeywordsForTest_UniversalLearning verifies universal learning is not narrow.
func TestIsNarrowByKeywordsForTest_UniversalLearning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isNarrow, _ := memory.IsNarrowByKeywordsForTest("always write tests before implementing")

	g.Expect(isNarrow).To(BeFalse())
}

// TestOptimizeAutoDemoteForTest_EmptyDB verifies no error on empty database.
func TestOptimizeAutoDemoteForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir, AutoDemoteUtility: 0.4}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeAutoDemoteForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeClaudeMDDedupForTest_EmptyDB verifies no error on empty database.
func TestOptimizeClaudeMDDedupForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeClaudeMDDedupForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeCompileSkillsForTest_EmptyDB verifies no error on empty database.
func TestOptimizeCompileSkillsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeCompileSkillsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeContradictionsForTest_EmptyDB verifies no error on empty database.
func TestOptimizeContradictionsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeContradictionsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeDecayForTest_EmptyDB verifies no error on empty database.
func TestOptimizeDecayForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir, DecayBase: 0.9}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDecayForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeDedupForTest_EmptyDB verifies no error on empty database.
func TestOptimizeDedupForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir, DupThreshold: 0.95}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDedupForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeDemoteClaudeMDForTest_EmptyDB verifies no error on empty database.
func TestOptimizeDemoteClaudeMDForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDemoteClaudeMDForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeMergeSkillsForTest_EmptyDB verifies no error on empty database.
func TestOptimizeMergeSkillsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeMergeSkillsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizePromoteForTest_EmptyDB verifies no error on empty database.
func TestOptimizePromoteForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePromoteForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizePromoteSkillsForTest_EmptyDB verifies no error on empty database.
func TestOptimizePromoteSkillsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePromoteSkillsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizePruneForTest_EmptyDB verifies no error on empty database.
func TestOptimizePruneForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir, PruneThreshold: 0.1}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePruneForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizePurgeBoilerplateForTest_EmptyDB verifies no error on empty database.
func TestOptimizePurgeBoilerplateForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePurgeBoilerplateForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizePurgeLegacySessionEmbeddingsForTest_EmptyDB verifies no error on empty database.
func TestOptimizePurgeLegacySessionEmbeddingsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePurgeLegacySessionEmbeddingsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeSplitSkillsForTest_EmptyDB verifies no error on empty database.
func TestOptimizeSplitSkillsForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeSplitSkillsForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeSynthesizeForTest_EmptyDB verifies no error on empty database.
func TestOptimizeSynthesizeForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeSynthesizeForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestPerformSkillReorganizationForTest_EmptyDB verifies no error on empty database.
func TestPerformSkillReorganizationForTest_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	opts := memory.OptimizeOpts{MemoryRoot: tmpDir, ReorgThreshold: 0.8}
	result := &memory.OptimizeResult{}

	err = memory.PerformSkillReorganizationForTest(db, opts, result)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestPruneOrphanedSkillsForTest_NoSkillsDir verifies 0 returned for non-existent skills dir.
func TestPruneOrphanedSkillsForTest_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	count, err := memory.PruneOrphanedSkillsForTest(db, tmpDir+"/nonexistent-skills", map[string]bool{})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

// TestPruneStaleSkillsForTest_NoSkillsDir verifies 0 returned for non-existent skills dir.
func TestPruneStaleSkillsForTest_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	count := memory.PruneStaleSkillsForTest(db, tmpDir+"/nonexistent-skills", 0.4)

	g.Expect(count).To(Equal(0))
}
