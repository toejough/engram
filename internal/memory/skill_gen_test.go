//go:build sqlite_fts5

package memory_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Dynamic Skill Generation (TASK-1)
// ============================================================================

// TestGeneratedSkillsTableCreated verifies that initEmbeddingsDB creates
// the generated_skills table with all required columns.
func TestGeneratedSkillsTableCreated(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Verify table exists and has correct schema
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='generated_skills'").Scan(&tableName)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(tableName).To(Equal("generated_skills"))

	// Verify all required columns exist by trying to select them
	row := db.QueryRow(`SELECT
		id, slug, theme, description, content, source_memory_ids,
		alpha, beta, utility, retrieval_count, last_retrieved,
		created_at, updated_at, pruned, embedding_id
		FROM generated_skills LIMIT 0`)

	var id int64
	var slug, theme, description, content, sourceMemoryIDs string
	var alpha, beta, utility float64
	var retrievalCount int
	var lastRetrieved, createdAt, updatedAt string
	var pruned int
	var embeddingID int64

	err = row.Scan(&id, &slug, &theme, &description, &content, &sourceMemoryIDs,
		&alpha, &beta, &utility, &retrievalCount, &lastRetrieved,
		&createdAt, &updatedAt, &pruned, &embeddingID)
	// Expecting sql.ErrNoRows since we LIMIT 0, but that means the columns exist
	g.Expect(err).To(MatchError(ContainSubstring("no rows")))
}

// TestInsertSkill verifies that insertSkill inserts a GeneratedSkill
// and returns its auto-generated ID.
func TestInsertSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "Test Theme",
		Description:     "A test skill",
		Content:         "# Test Skill\n\nContent here.",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           2.0,
		Beta:            1.0,
		Utility:         0.75,
		RetrievalCount:  0,
		LastRetrieved:   "",
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
		EmbeddingID:     0,
	}

	id, err := memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(BeNumerically(">", 0))
}

// TestInsertSkillUniqueSlug verifies that inserting a skill with a duplicate
// slug fails with a unique constraint violation.
func TestInsertSkillUniqueSlug(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:        "duplicate-slug",
		Theme:       "Test",
		Description: "First",
		Content:     "Content",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Try to insert again with same slug
	skill2 := &memory.GeneratedSkill{
		Slug:        "duplicate-slug",
		Theme:       "Test",
		Description: "Second",
		Content:     "Different content",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = memory.InsertSkillForTest(db, skill2)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("UNIQUE"))
}

// TestGetSkillBySlug verifies that getSkillBySlug retrieves a skill by its slug.
func TestGetSkillBySlug(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "Test Theme",
		Description:     "A test skill",
		Content:         "# Test Skill\n\nContent here.",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           2.0,
		Beta:            1.0,
		Utility:         0.75,
		RetrievalCount:  5,
		LastRetrieved:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
		EmbeddingID:     100,
	}

	id, err := memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	retrieved, err := memory.GetSkillBySlugForTest(db, "test-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(retrieved).ToNot(BeNil())
	g.Expect(retrieved.ID).To(Equal(id))
	g.Expect(retrieved.Slug).To(Equal("test-skill"))
	g.Expect(retrieved.Theme).To(Equal("Test Theme"))
	g.Expect(retrieved.Description).To(Equal("A test skill"))
	g.Expect(retrieved.Content).To(Equal("# Test Skill\n\nContent here."))
	g.Expect(retrieved.SourceMemoryIDs).To(Equal("[1,2,3]"))
	g.Expect(retrieved.Alpha).To(BeNumerically("~", 2.0, 0.001))
	g.Expect(retrieved.Beta).To(BeNumerically("~", 1.0, 0.001))
	g.Expect(retrieved.Utility).To(BeNumerically("~", 0.75, 0.001))
	g.Expect(retrieved.RetrievalCount).To(Equal(5))
	g.Expect(retrieved.LastRetrieved).To(Equal(now))
	g.Expect(retrieved.CreatedAt).To(Equal(now))
	g.Expect(retrieved.UpdatedAt).To(Equal(now))
	g.Expect(retrieved.Pruned).To(BeFalse())
	g.Expect(retrieved.EmbeddingID).To(Equal(int64(100)))
}

// TestGetSkillBySlugNotFound verifies that getSkillBySlug returns nil
// for a non-existent slug.
func TestGetSkillBySlugNotFound(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	retrieved, err := memory.GetSkillBySlugForTest(db, "nonexistent-slug")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(retrieved).To(BeNil())
}

// TestListSkills verifies that listSkills returns all non-pruned skills.
func TestListSkills(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert 3 skills: 2 active, 1 pruned
	skill1 := &memory.GeneratedSkill{
		Slug:        "skill-1",
		Theme:       "Theme 1",
		Description: "First skill",
		Content:     "Content 1",
		CreatedAt:   now,
		UpdatedAt:   now,
		Pruned:      false,
	}
	skill2 := &memory.GeneratedSkill{
		Slug:        "skill-2",
		Theme:       "Theme 2",
		Description: "Second skill",
		Content:     "Content 2",
		CreatedAt:   now,
		UpdatedAt:   now,
		Pruned:      false,
	}
	skill3 := &memory.GeneratedSkill{
		Slug:        "skill-3",
		Theme:       "Theme 3",
		Description: "Third skill (pruned)",
		Content:     "Content 3",
		CreatedAt:   now,
		UpdatedAt:   now,
		Pruned:      true,
	}

	_, err = memory.InsertSkillForTest(db, skill1)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = memory.InsertSkillForTest(db, skill2)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = memory.InsertSkillForTest(db, skill3)
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := memory.ListSkillsForTest(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(2))

	slugs := []string{skills[0].Slug, skills[1].Slug}
	g.Expect(slugs).To(ContainElements("skill-1", "skill-2"))
}

// TestListSkillsEmpty verifies that listSkills returns empty slice
// when no skills exist.
func TestListSkillsEmpty(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	skills, err := memory.ListSkillsForTest(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

// TestUpdateSkill verifies that updateSkill modifies an existing skill.
func TestUpdateSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:           "test-skill",
		Theme:          "Original Theme",
		Description:    "Original description",
		Content:        "Original content",
		Alpha:          1.0,
		Beta:           1.0,
		Utility:        0.5,
		RetrievalCount: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	id, err := memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Update the skill
	later := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	skill.ID = id
	skill.Theme = "Updated Theme"
	skill.Description = "Updated description"
	skill.Alpha = 3.0
	skill.Beta = 2.0
	skill.Utility = 0.8
	skill.RetrievalCount = 10
	skill.LastRetrieved = later
	skill.UpdatedAt = later

	err = memory.UpdateSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the update
	retrieved, err := memory.GetSkillBySlugForTest(db, "test-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(retrieved.Theme).To(Equal("Updated Theme"))
	g.Expect(retrieved.Description).To(Equal("Updated description"))
	g.Expect(retrieved.Alpha).To(BeNumerically("~", 3.0, 0.001))
	g.Expect(retrieved.Beta).To(BeNumerically("~", 2.0, 0.001))
	g.Expect(retrieved.Utility).To(BeNumerically("~", 0.8, 0.001))
	g.Expect(retrieved.RetrievalCount).To(Equal(10))
	g.Expect(retrieved.LastRetrieved).To(Equal(later))
	g.Expect(retrieved.UpdatedAt).To(Equal(later))
}

// TestSoftDeleteSkill verifies that softDeleteSkill sets pruned=1.
func TestSoftDeleteSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:        "test-skill",
		Theme:       "Test",
		Description: "To be deleted",
		Content:     "Content",
		CreatedAt:   now,
		UpdatedAt:   now,
		Pruned:      false,
	}

	id, err := memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Soft delete
	err = memory.SoftDeleteSkillForTest(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify it's marked as pruned
	retrieved, err := memory.GetSkillBySlugForTest(db, "test-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(retrieved).ToNot(BeNil())
	g.Expect(retrieved.Pruned).To(BeTrue())

	// Verify it doesn't appear in listSkills
	skills, err := memory.ListSkillsForTest(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

// TestSkillConfidenceMean verifies that SkillConfidence.Mean() computes
// alpha/(alpha+beta) correctly.
func TestSkillConfidenceMean(t *testing.T) {
	tests := []struct {
		name     string
		alpha    float64
		beta     float64
		expected float64
	}{
		{"Equal weights", 1.0, 1.0, 0.5},
		{"More success", 3.0, 1.0, 0.75},
		{"More failure", 1.0, 3.0, 0.25},
		{"High confidence success", 10.0, 2.0, 10.0 / 12.0},
		{"Zero start", 0.0, 0.0, 0.0}, // Edge case: 0/0 should handle gracefully
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			conf := memory.SkillConfidence{
				Alpha: tt.alpha,
				Beta:  tt.beta,
			}

			mean := conf.Mean()
			if tt.alpha == 0.0 && tt.beta == 0.0 {
				// Special case: should return 0 or handle gracefully
				g.Expect(mean).To(BeNumerically(">=", 0.0))
				g.Expect(mean).To(BeNumerically("<=", 1.0))
			} else {
				g.Expect(mean).To(BeNumerically("~", tt.expected, 0.001))
			}
		})
	}
}

// TestComputeUtility verifies that computeUtility implements the MACLA formula:
// utility = 0.5*(alpha/(alpha+beta)) + 0.3*min(1, ln(1+retrievals)/5) + 0.2*exp(-days_since_last/30)
func TestComputeUtility(t *testing.T) {
	now := time.Now().UTC()
	lastWeek := now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	lastMonth := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name          string
		alpha         float64
		beta          float64
		retrievals    int
		lastRetrieved string
		description   string
	}{
		{
			name:          "Balanced skill with recent use",
			alpha:         2.0,
			beta:          2.0,
			retrievals:    10,
			lastRetrieved: lastWeek,
			description:   "Confidence 0.5, decent retrievals, recent use",
		},
		{
			name:          "High confidence, many retrievals",
			alpha:         10.0,
			beta:          2.0,
			retrievals:    100,
			lastRetrieved: lastWeek,
			description:   "Should have high utility",
		},
		{
			name:          "New skill, no retrievals",
			alpha:         1.0,
			beta:          1.0,
			retrievals:    0,
			lastRetrieved: "",
			description:   "No retrieval history",
		},
		{
			name:          "Old skill, not used recently",
			alpha:         5.0,
			beta:          1.0,
			retrievals:    50,
			lastRetrieved: lastMonth,
			description:   "High confidence but stale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			utility := memory.ComputeUtilityForTest(tt.alpha, tt.beta, tt.retrievals, tt.lastRetrieved)

			// Verify utility is in valid range [0, 1]
			g.Expect(utility).To(BeNumerically(">=", 0.0))
			g.Expect(utility).To(BeNumerically("<=", 1.0))

			// Compute expected value manually for specific cases
			if tt.name == "New skill, no retrievals" {
				confidence := tt.alpha / (tt.alpha + tt.beta)
				// Allow some flexibility in implementation for recency when empty
				g.Expect(utility).To(BeNumerically(">=", 0.5*confidence-0.2))
				g.Expect(utility).To(BeNumerically("<=", 0.5*confidence+0.2))
			}
		})
	}
}

// TestComputeUtilityMACLAFormula verifies the exact MACLA formula implementation.
func TestComputeUtilityMACLAFormula(t *testing.T) {
	g := NewWithT(t)

	// Test a specific case where we can compute the expected value exactly
	alpha := 3.0
	beta := 1.0
	retrievals := 20
	now := time.Now().UTC()
	lastRetrieved := now.Add(-15 * 24 * time.Hour).Format(time.RFC3339) // 15 days ago

	utility := memory.ComputeUtilityForTest(alpha, beta, retrievals, lastRetrieved)

	// Expected computation:
	confidence := alpha / (alpha + beta)                     // 0.75
	retrievalScore := math.Min(1.0, math.Log(1+20)/5)        // min(1, ln(21)/5) = min(1, 0.6072) = 0.6072
	recencyScore := math.Exp(-15.0 / 30.0)                   // exp(-0.5) = 0.6065
	expected := 0.5*confidence + 0.3*retrievalScore + 0.2*recencyScore

	g.Expect(utility).To(BeNumerically("~", expected, 0.01))
}

// TestPropertyUtilityAlwaysInRange verifies via property test that
// computeUtility always returns a value in [0, 1].
func TestPropertyUtilityAlwaysInRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		alpha := rapid.Float64Range(0.1, 100.0).Draw(t, "alpha")
		beta := rapid.Float64Range(0.1, 100.0).Draw(t, "beta")
		retrievals := rapid.IntRange(0, 10000).Draw(t, "retrievals")

		// Generate a random timestamp within the last year or empty
		useTimestamp := rapid.Bool().Draw(t, "useTimestamp")
		var lastRetrieved string
		if useTimestamp {
			daysAgo := rapid.IntRange(0, 365).Draw(t, "daysAgo")
			lastRetrieved = time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour).Format(time.RFC3339)
		}

		utility := memory.ComputeUtilityForTest(alpha, beta, retrievals, lastRetrieved)

		g.Expect(utility).To(BeNumerically(">=", 0.0))
		g.Expect(utility).To(BeNumerically("<=", 1.0))
	})
}

// TestPropertySkillConfidenceAlwaysInRange verifies via property test that
// SkillConfidence.Mean() always returns a value in [0, 1].
func TestPropertySkillConfidenceAlwaysInRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		alpha := rapid.Float64Range(0.1, 100.0).Draw(t, "alpha")
		beta := rapid.Float64Range(0.1, 100.0).Draw(t, "beta")

		conf := memory.SkillConfidence{
			Alpha: alpha,
			Beta:  beta,
		}

		mean := conf.Mean()

		g.Expect(mean).To(BeNumerically(">=", 0.0))
		g.Expect(mean).To(BeNumerically("<=", 1.0))
	})
}

// TestPropertyConfidenceMeanIsWeightedAverage verifies that Mean() correctly
// computes alpha/(alpha+beta).
func TestPropertyConfidenceMeanIsWeightedAverage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		alpha := rapid.Float64Range(0.1, 100.0).Draw(t, "alpha")
		beta := rapid.Float64Range(0.1, 100.0).Draw(t, "beta")

		conf := memory.SkillConfidence{
			Alpha: alpha,
			Beta:  beta,
		}

		mean := conf.Mean()
		expected := alpha / (alpha + beta)

		g.Expect(mean).To(BeNumerically("~", expected, 0.0001))
	})
}

// TestPropertyUtilityMonotonicWithConfidence verifies that utility increases
// with confidence when other factors are held constant.
func TestPropertyUtilityMonotonicWithConfidence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		retrievals := rapid.IntRange(0, 100).Draw(t, "retrievals")
		daysAgo := rapid.IntRange(0, 60).Draw(t, "daysAgo")
		lastRetrieved := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour).Format(time.RFC3339)

		// Low confidence skill
		alpha1 := 1.0
		beta1 := 4.0
		utility1 := memory.ComputeUtilityForTest(alpha1, beta1, retrievals, lastRetrieved)

		// High confidence skill
		alpha2 := 4.0
		beta2 := 1.0
		utility2 := memory.ComputeUtilityForTest(alpha2, beta2, retrievals, lastRetrieved)

		// Higher confidence should yield higher utility (all else equal)
		g.Expect(utility2).To(BeNumerically(">", utility1))
	})
}

// ============================================================================
// Unit tests for Cluster Scoring & Skill Content Generation (TASK-2)
// ============================================================================

// mockSkillCompilerGen is a test double for SkillCompiler interface.
type mockSkillCompilerGen struct {
	CompileFunc    func(theme string, memories []string) (string, error)
	SynthesizeFunc func(memories []string) (string, error)
}

func (m *mockSkillCompilerGen) CompileSkill(theme string, memories []string) (string, error) {
	if m.CompileFunc != nil {
		return m.CompileFunc(theme, memories)
	}
	return "# Mock Skill\n\nGenerated from mock compiler.", nil
}

func (m *mockSkillCompilerGen) Extract(message string) (*memory.Observation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSkillCompilerGen) Decide(newMessage string, existing []memory.ExistingMemory) (*memory.IngestDecision, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSkillCompilerGen) Synthesize(memories []string) (string, error) {
	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(memories)
	}
	return "Mock synthesized principle.", nil
}

// TestScoreClusterComputesMACLA verifies that scoreCluster computes MACLA
// utility score from cluster member metadata (confidence, retrieval_count, last_retrieved).
func TestScoreClusterComputesMACLA(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Insert 3 memories with known metadata into embeddings table
	lastWeek := time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// High confidence, high retrievals, recent
	_, err = db.Exec(`INSERT INTO embeddings (content, source, confidence, retrieval_count, last_retrieved, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?)`, "Memory 1", "test", 0.9, 50, lastWeek, 1)
	g.Expect(err).ToNot(HaveOccurred())

	// Medium confidence, medium retrievals
	_, err = db.Exec(`INSERT INTO embeddings (content, source, confidence, retrieval_count, last_retrieved, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?)`, "Memory 2", "test", 0.7, 20, lastWeek, 2)
	g.Expect(err).ToNot(HaveOccurred())

	// Low confidence, few retrievals, never retrieved
	_, err = db.Exec(`INSERT INTO embeddings (content, source, confidence, retrieval_count, embedding_id)
		VALUES (?, ?, ?, ?, ?)`, "Memory 3", "test", 0.3, 5, 3)
	g.Expect(err).ToNot(HaveOccurred())

	// Get embedding IDs
	var id1, id2, id3 int64
	g.Expect(db.QueryRow("SELECT id FROM embeddings WHERE embedding_id = 1").Scan(&id1)).To(Succeed())
	g.Expect(db.QueryRow("SELECT id FROM embeddings WHERE embedding_id = 2").Scan(&id2)).To(Succeed())
	g.Expect(db.QueryRow("SELECT id FROM embeddings WHERE embedding_id = 3").Scan(&id3)).To(Succeed())

	cluster := []memory.ClusterEntry{
		{ID: id1, Content: "Memory 1", EmbeddingID: 1},
		{ID: id2, Content: "Memory 2", EmbeddingID: 2},
		{ID: id3, Content: "Memory 3", EmbeddingID: 3},
	}

	score, err := memory.ScoreClusterForTest(db, cluster)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically(">=", 0.0))
	g.Expect(score).To(BeNumerically("<=", 1.0))

	// The score should be > 0 since we have retrievals and confidence
	g.Expect(score).To(BeNumerically(">", 0.0))
}

// TestScoreClusterAveragesMembers verifies that scoreCluster averages
// utility scores across all cluster members.
func TestScoreClusterAveragesMembers(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Insert 2 memories with identical metadata
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = db.Exec(`INSERT INTO embeddings (content, source, confidence, retrieval_count, last_retrieved, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?)`, "Memory 1", "test", 0.8, 30, now, 1)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(`INSERT INTO embeddings (content, source, confidence, retrieval_count, last_retrieved, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?)`, "Memory 2", "test", 0.8, 30, now, 2)
	g.Expect(err).ToNot(HaveOccurred())

	var id1, id2 int64
	g.Expect(db.QueryRow("SELECT id FROM embeddings WHERE embedding_id = 1").Scan(&id1)).To(Succeed())
	g.Expect(db.QueryRow("SELECT id FROM embeddings WHERE embedding_id = 2").Scan(&id2)).To(Succeed())

	cluster := []memory.ClusterEntry{
		{ID: id1, Content: "Memory 1", EmbeddingID: 1},
		{ID: id2, Content: "Memory 2", EmbeddingID: 2},
	}

	score, err := memory.ScoreClusterForTest(db, cluster)
	g.Expect(err).ToNot(HaveOccurred())

	// Since both have identical metadata, their individual utility should be the same
	// So the average should equal the individual utility
	// We can verify this is in a reasonable range
	g.Expect(score).To(BeNumerically(">", 0.5)) // Should be fairly high given good metadata
	g.Expect(score).To(BeNumerically("<", 1.0))
}

// TestGenerateSkillContentUsesCompiler verifies that generateSkillContent
// calls the SkillCompiler when available.
func TestGenerateSkillContentUsesCompiler(t *testing.T) {
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "Always write tests first", EmbeddingID: 1},
		{ID: 2, Content: "TDD red green refactor", EmbeddingID: 2},
		{ID: 3, Content: "Test before implementation", EmbeddingID: 3},
	}

	compilerCalled := false
	compiler := &mockSkillCompilerGen{
		CompileFunc: func(theme string, memories []string) (string, error) {
			compilerCalled = true
			g.Expect(theme).To(Equal("TDD Best Practices"))
			g.Expect(memories).To(HaveLen(3))
			return "# TDD Skill\n\nWrite tests first, then implement.", nil
		},
	}

	content, err := memory.GenerateSkillContentForTest("TDD Best Practices", cluster, compiler)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(compilerCalled).To(BeTrue())
	g.Expect(content).To(ContainSubstring("# TDD Skill"))
	g.Expect(content).To(ContainSubstring("Write tests first"))
}

// TestGenerateSkillContentFallsBackToTemplate verifies that generateSkillContent
// uses a template when the compiler is nil or returns an error.
func TestGenerateSkillContentFallsBackToTemplate(t *testing.T) {
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "Always write tests first", EmbeddingID: 1},
		{ID: 2, Content: "TDD red green refactor", EmbeddingID: 2},
	}

	// Test with nil compiler
	content, err := memory.GenerateSkillContentForTest("TDD Best Practices", cluster, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).ToNot(BeEmpty())
	g.Expect(content).To(ContainSubstring("TDD Best Practices"))
}

// TestGenerateSkillContentFallsBackOnCompilerError verifies fallback behavior
// when the compiler returns an error.
func TestGenerateSkillContentFallsBackOnCompilerError(t *testing.T) {
	g := NewWithT(t)

	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "Memory 1", EmbeddingID: 1},
	}

	compiler := &mockSkillCompilerGen{
		CompileFunc: func(theme string, memories []string) (string, error) {
			return "", memory.ErrLLMUnavailable
		},
	}

	content, err := memory.GenerateSkillContentForTest("Test Theme", cluster, compiler)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).ToNot(BeEmpty())
	g.Expect(content).To(ContainSubstring("Test Theme"))
}

// TestSlugifyProducesURLSafe verifies that slugify produces URL-safe identifiers.
func TestSlugifyProducesURLSafe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "TDD Best Practices",
			expected: "tdd-best-practices",
		},
		{
			name:     "With special chars",
			input:    "Go: Testing & Mocking!",
			expected: "go-testing-mocking",
		},
		{
			name:     "Multiple spaces",
			input:    "Error   Handling    Patterns",
			expected: "error-handling-patterns",
		},
		{
			name:     "Leading/trailing spaces",
			input:    "  Clean Code  ",
			expected: "clean-code",
		},
		{
			name:     "Unicode characters",
			input:    "Documentation 文档",
			expected: "documentation",
		},
		{
			name:     "All special chars",
			input:    "!!!@@@###",
			expected: "", // Should produce empty or some default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			slug := memory.SlugifyForTest(tt.input)

			// Verify URL-safe: only lowercase letters, numbers, and hyphens
			if slug != "" {
				g.Expect(slug).To(MatchRegexp("^[a-z0-9-]+$"))
			}

			// Verify no leading/trailing hyphens
			if slug != "" {
				g.Expect(slug).ToNot(HavePrefix("-"))
				g.Expect(slug).ToNot(HaveSuffix("-"))
			}

			// Verify no consecutive hyphens
			g.Expect(slug).ToNot(ContainSubstring("--"))

			// Check expected output if specified
			if tt.expected != "" {
				g.Expect(slug).To(Equal(tt.expected))
			}
		})
	}
}

// TestPropertySlugifyAlwaysURLSafe verifies via property test that slugify
// always produces URL-safe identifiers.
func TestPropertySlugifyAlwaysURLSafe(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		input := rapid.String().Draw(t, "input")
		slug := memory.SlugifyForTest(input)

		// If slug is non-empty, it must be URL-safe
		if slug != "" {
			g.Expect(slug).To(MatchRegexp("^[a-z0-9-]+$"))
			g.Expect(slug).ToNot(HavePrefix("-"))
			g.Expect(slug).ToNot(HaveSuffix("-"))
			g.Expect(slug).ToNot(ContainSubstring("--"))
		}
	})
}

// TestWriteSkillFileCreatesDirectory verifies that writeSkillFile creates
// the skill directory structure.
func TestWriteSkillFileCreatesDirectory(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:        "test-skill",
		Theme:       "Test Theme",
		Description: "A test skill",
		Content:     "# Test Skill\n\nContent here.",
		Alpha:       2.0,
		Beta:        1.0,
		Utility:     0.75,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := memory.WriteSkillFileForTest(skillsDir, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify directory was created
	skillDir := filepath.Join(skillsDir, "mem-test-skill")
	info, err := os.Stat(skillDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.IsDir()).To(BeTrue())

	// Verify SKILL.md was created
	skillFile := filepath.Join(skillDir, "SKILL.md")
	_, err = os.Stat(skillFile)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestWriteSkillFileContainsFrontmatter verifies that writeSkillFile creates
// SKILL.md with proper YAML frontmatter.
func TestWriteSkillFileContainsFrontmatter(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:        "test-skill",
		Theme:       "Test Theme",
		Description: "A test skill for validation",
		Content:     "# Test Skill\n\nSkill content goes here.",
		Alpha:       3.0,
		Beta:        1.0,
		Utility:     0.82,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := memory.WriteSkillFileForTest(skillsDir, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Read the file
	skillFile := filepath.Join(skillsDir, "mem-test-skill", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	g.Expect(err).ToNot(HaveOccurred())

	contentStr := string(content)

	// Verify frontmatter structure
	g.Expect(contentStr).To(HavePrefix("---\n"))
	g.Expect(contentStr).To(ContainSubstring("name: mem:test-skill"))
	g.Expect(contentStr).To(ContainSubstring("description: A test skill for validation"))
	g.Expect(contentStr).ToNot(ContainSubstring("context: inherit"))
	g.Expect(contentStr).To(ContainSubstring("model: haiku"))
	g.Expect(contentStr).To(ContainSubstring("user-invocable: true"))
	g.Expect(contentStr).To(ContainSubstring("generated: true"))
	g.Expect(contentStr).To(ContainSubstring("confidence: 0.75")) // alpha/(alpha+beta) = 3/4 = 0.75
	g.Expect(contentStr).To(ContainSubstring("source: memory-compilation"))

	// Verify content appears after frontmatter
	g.Expect(contentStr).To(ContainSubstring("# Test Skill"))
	g.Expect(contentStr).To(ContainSubstring("Skill content goes here"))
}

// TestWriteSkillFileOverwritesExisting verifies that writeSkillFile overwrites
// an existing SKILL.md file.
func TestWriteSkillFileOverwritesExisting(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:        "test-skill",
		Theme:       "Test Theme",
		Description: "Original description",
		Content:     "# Original Content",
		Alpha:       1.0,
		Beta:        1.0,
		Utility:     0.5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Write once
	err := memory.WriteSkillFileForTest(skillsDir, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Update and write again
	skill.Description = "Updated description"
	skill.Content = "# Updated Content"
	skill.Alpha = 4.0
	skill.Beta = 1.0

	err = memory.WriteSkillFileForTest(skillsDir, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Read and verify updated content
	skillFile := filepath.Join(skillsDir, "mem-test-skill", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	g.Expect(err).ToNot(HaveOccurred())

	contentStr := string(content)
	g.Expect(contentStr).To(ContainSubstring("Updated description"))
	g.Expect(contentStr).To(ContainSubstring("# Updated Content"))
	g.Expect(contentStr).To(ContainSubstring("confidence: 0.80")) // 4/(4+1) = 0.80
}

// TestClaudeCLIExtractorImplementsSkillCompiler verifies that ClaudeCLIExtractor
// implements the SkillCompiler interface.
func TestClaudeCLIExtractorImplementsSkillCompiler(t *testing.T) {
	g := NewWithT(t)

	// This is a compile-time check that ClaudeCLIExtractor implements SkillCompiler
	var _ memory.SkillCompiler = (*memory.ClaudeCLIExtractor)(nil)

	// Also verify we can create one and call CompileSkill
	extractor := memory.NewClaudeCLIExtractor()
	g.Expect(extractor).ToNot(BeNil())

	// We won't actually call CompileSkill here since it requires the claude CLI,
	// but we verify the method exists by checking it compiles
}

// TestSkillCompilerInterface verifies that SkillCompiler interface is defined
// with the expected method signature.
func TestSkillCompilerInterface(t *testing.T) {
	g := NewWithT(t)

	// Create a mock that implements the interface
	mock := &mockSkillCompilerGen{
		CompileFunc: func(theme string, memories []string) (string, error) {
			return "Test output", nil
		},
	}

	// Verify it implements SkillCompiler
	var compiler memory.SkillCompiler = mock
	g.Expect(compiler).ToNot(BeNil())

	// Verify we can call CompileSkill
	result, err := compiler.CompileSkill("Test Theme", []string{"mem1", "mem2"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("Test output"))
}
