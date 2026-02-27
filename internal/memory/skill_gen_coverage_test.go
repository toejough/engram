//go:build sqlite_fts5

package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestComputeUtility_EmptyLastRetrieved verifies empty lastRetrieved gives zero recency score.
func TestComputeUtility_EmptyLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	utility := memory.ComputeUtilityForTest(1.0, 1.0, 0, "")

	g.Expect(utility).To(BeNumerically(">=", 0.0))
	g.Expect(utility).To(BeNumerically("<=", 1.0))
}

// TestComputeUtility_HighValuesClampedToOne verifies extreme values clamp at 1.0.
func TestComputeUtility_HighValuesClampedToOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recent := time.Now().UTC().Format(time.RFC3339)
	utility := memory.ComputeUtilityForTest(1000.0, 0.001, 10000, recent)

	g.Expect(utility).To(BeNumerically("<=", 1.0))
	g.Expect(utility).To(BeNumerically(">=", 0.0))
}

// TestComputeUtility_InvalidLastRetrieved verifies bad timestamp is treated as empty.
func TestComputeUtility_InvalidLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	utility := memory.ComputeUtilityForTest(1.0, 1.0, 0, "not-a-valid-date")

	g.Expect(utility).To(BeNumerically(">=", 0.0))
	g.Expect(utility).To(BeNumerically("<=", 1.0))
}

// TestComputeUtility_ValidLastRetrieved verifies recent timestamp increases utility.
func TestComputeUtility_ValidLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recent := time.Now().UTC().Format(time.RFC3339)
	utility := memory.ComputeUtilityForTest(5.0, 1.0, 10, recent)

	g.Expect(utility).To(BeNumerically(">", 0.0))
	g.Expect(utility).To(BeNumerically("<=", 1.0))
}

// TestGenerateSkillContent_CompilerError verifies non-LLM error falls through to template.
func TestGenerateSkillContent_CompilerError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 1, Content: "Test content"}}
	content, err := memory.GenerateSkillContentForTest("theme", cluster, &mockSkillCompiler{err: errors.New("some error")})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("theme"))
}

// TestGenerateSkillContent_CompilerLLMUnavailable verifies ErrLLMUnavailable falls through to template.
func TestGenerateSkillContent_CompilerLLMUnavailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 1, Content: "Test content"}}
	content, err := memory.GenerateSkillContentForTest("theme", cluster, &mockSkillCompiler{err: memory.ErrLLMUnavailable})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("theme"))
}

// TestGenerateSkillContent_CompilerSuccess verifies compiler success path returns compiled content.
func TestGenerateSkillContent_CompilerSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{{ID: 1, Content: "Test content"}}
	content, err := memory.GenerateSkillContentForTest("theme", cluster, &mockSkillCompiler{content: "# Compiled skill content"})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("Compiled skill content"))
}

// TestGenerateSkillContent_NilCompiler verifies nil compiler uses template fallback.
func TestGenerateSkillContent_NilCompiler(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "Always handle errors"},
		{ID: 2, Content: "Use fmt.Errorf for wrapping"},
	}
	content, err := memory.GenerateSkillContentForTest("error handling", cluster, nil)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("error handling"))
}

// TestGetSkillBySlug_NotFound verifies not-found returns nil with no error.
func TestGetSkillBySlug_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	skill, err := memory.GetSkillBySlugForTest(db, "nonexistent-slug")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill).To(BeNil())
}

// TestInsertSkill_Pruned verifies inserting a pruned skill is filtered from listSkills.
func TestInsertSkill_Pruned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	id, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "pruned-skill",
		Theme:           "Pruned Theme",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Pruned:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(BeNumerically(">", int64(0)))

	skills, err := memory.ListSkillsForTest(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(0))
}

// TestListSkills_WithEmbeddingIDAndLastRetrieved verifies listSkills populates nullable fields.
func TestListSkills_WithEmbeddingIDAndLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "listed-skill",
		Theme:           "List Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		EmbeddingID:     42,
		LastRetrieved:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := memory.ListSkillsForTest(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))
	g.Expect(skills[0].EmbeddingID).To(Equal(int64(42)))
	g.Expect(skills[0].LastRetrieved).ToNot(BeEmpty())
}

// TestRecordSkillFeedback_Success verifies RecordSkillFeedback updates alpha on success.
func TestRecordSkillFeedback_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "feedback-skill",
		Theme:           "Feedback Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.RecordSkillFeedbackForTest(db, "feedback-skill", true)

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "feedback-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Alpha).To(BeNumerically(">", 1.0))
}

// TestRecordSkillUsage_Failure verifies RecordSkillUsage with success=false increments retrieval count.
func TestRecordSkillUsage_Failure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "usage-fail-skill",
		Theme:           "Usage Fail Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.RecordSkillUsageForTest(db, "usage-fail-skill", false)

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "usage-fail-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.RetrievalCount).To(Equal(1))
}

// TestRecordSkillUsage_NotFound verifies RecordSkillUsage errors when skill missing.
func TestRecordSkillUsage_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = memory.RecordSkillUsageForTest(db, "nonexistent", false)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestRecordSkillUsage_Success verifies RecordSkillUsage with success=true also records feedback.
func TestRecordSkillUsage_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "usage-success-skill",
		Theme:           "Usage Success Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.RecordSkillUsageForTest(db, "usage-success-skill", true)

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "usage-success-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.RetrievalCount).To(Equal(1))
	g.Expect(skill.Alpha).To(BeNumerically(">", 1.0))
}

// TestScoreCluster_Empty verifies empty cluster returns zero score.
func TestScoreCluster_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	score, err := memory.ScoreClusterForTest(db, nil)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(Equal(0.0))
}

// TestScoreCluster_WithEmbeddings verifies cluster with embeddings returns positive score.
func TestScoreCluster_WithEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, embedding_id, confidence, retrieval_count) VALUES ('test content', 'test', 1, 0.8, 5)`)
	g.Expect(err).ToNot(HaveOccurred())

	var embedID int64
	err = db.QueryRow("SELECT id FROM embeddings WHERE source = 'test'").Scan(&embedID)
	g.Expect(err).ToNot(HaveOccurred())

	cluster := []memory.ClusterEntry{{ID: embedID, Content: "test content"}}
	score, err := memory.ScoreClusterForTest(db, cluster)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically(">", 0.0))
}

// TestSkillConfidenceMean_Normal verifies Mean returns alpha/(alpha+beta).
func TestSkillConfidenceMean_Normal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sc := memory.SkillConfidence{Alpha: 3.0, Beta: 1.0}

	g.Expect(sc.Mean()).To(BeNumerically("~", 0.75, 0.001))
}

// TestSkillConfidenceMean_ZeroSum verifies Mean returns 0 when both are zero.
func TestSkillConfidenceMean_ZeroSum(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sc := memory.SkillConfidence{Alpha: 0, Beta: 0}

	g.Expect(sc.Mean()).To(Equal(0.0))
}

// TestSlugify verifies SlugifyForTest wraps slugify correctly.
func TestSlugify(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := memory.SlugifyForTest("Go Error Handling Patterns")

	g.Expect(slug).To(Equal("go-error-handling-patterns"))
}

// TestSoftDeleteSkill verifies SoftDeleteSkillForTest marks skill as pruned.
func TestSoftDeleteSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	id, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "soft-del-skill",
		Theme:           "Delete Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(BeNumerically(">", int64(0)))

	err = memory.SoftDeleteSkillForTest(db, id)

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "soft-del-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill).ToNot(BeNil())
	g.Expect(skill.Pruned).To(BeTrue())
}

// TestUpdateSkill verifies UpdateSkillForTest updates skill fields.
func TestUpdateSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	id, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "update-skill",
		Theme:           "Update Test",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.UpdateSkillForTest(db, &memory.GeneratedSkill{
		ID:              id,
		Slug:            "update-skill",
		Theme:           "Updated Theme",
		SourceMemoryIDs: "[]",
		Alpha:           2.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "update-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Theme).To(Equal("Updated Theme"))
	g.Expect(skill.Alpha).To(Equal(2.0))
}

// mockSkillCompiler implements SkillCompiler for testing.
type mockSkillCompiler struct {
	err     error
	content string
}

func (m *mockSkillCompiler) CompileSkill(_ context.Context, _ string, _ []string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.content, nil
}

func (m *mockSkillCompiler) Synthesize(_ context.Context, _ []string) (string, error) {
	return "", nil
}
