//go:build sqlite_fts5

package memory_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Skill Retrieval & Query Integration (TASK-4)
// ============================================================================

// TestSearchSkillsFindsRelevantSkill verifies that SearchSkills performs
// vector similarity search on skill description embeddings and returns
// matching skills.
func TestSearchSkillsFindsRelevantSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Insert a skill with embedding
	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "git-workflow",
		Theme:           "Version Control",
		Description:     "Best practices for git commit workflow",
		Content:         "# Git Workflow\n\nAlways use semantic commits.",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           8.0,
		Beta:            2.0, // confidence = 0.8 > 0.3
		Utility:         0.85,
		RetrievalCount:  5,
		LastRetrieved:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
		EmbeddingID:     0, // Will be set after embedding insert
	}

	// Create and insert embedding for the skill description
	fakeEmb := make([]float32, 384)
	for i := range fakeEmb {
		fakeEmb[i] = 0.1
	}
	blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())
	embID, err := result.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	skill.EmbeddingID = embID

	skillID, err := memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skillID).To(BeNumerically(">", 0))

	// Search with similar embedding
	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.1 // Same as skill embedding for high similarity
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeEmpty())
	g.Expect(results[0].Slug).To(Equal("git-workflow"))
	g.Expect(results[0].Theme).To(Equal("Version Control"))
	g.Expect(results[0].Description).To(Equal("Best practices for git commit workflow"))
	g.Expect(results[0].Confidence).To(BeNumerically("~", 0.8, 0.01))
	g.Expect(results[0].Utility).To(BeNumerically("~", 0.85, 0.01))
	g.Expect(results[0].Similarity).To(BeNumerically(">", 0.9)) // High similarity
}

// TestSearchSkillsExcludesPruned verifies that SearchSkills excludes
// skills marked as pruned.
func TestSearchSkillsExcludesPruned(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert pruned skill with embedding
	prunedSkill := &memory.GeneratedSkill{
		Slug:            "pruned-skill",
		Theme:           "Obsolete",
		Description:     "This skill is pruned",
		Content:         "# Pruned\n\nContent",
		SourceMemoryIDs: "[1]",
		Alpha:           5.0,
		Beta:            1.0, // confidence = 0.83 > 0.3
		Utility:         0.75,
		RetrievalCount:  0,
		LastRetrieved:   "",
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          true, // Pruned!
		EmbeddingID:     0,
	}

	fakeEmb := make([]float32, 384)
	for i := range fakeEmb {
		fakeEmb[i] = 0.2
	}
	blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())
	embID, err := result.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	prunedSkill.EmbeddingID = embID

	_, err = memory.InsertSkillForTest(db, prunedSkill)
	g.Expect(err).ToNot(HaveOccurred())

	// Search with similar embedding
	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.2
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty()) // Pruned skill should not appear
}

// TestSearchSkillsExcludesLowConfidence verifies that SearchSkills excludes
// skills with confidence <= 0.3 (alpha/(alpha+beta) <= 0.3).
func TestSearchSkillsExcludesLowConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert low-confidence skill (alpha=1, beta=4 => confidence=0.2 < 0.3)
	lowConfSkill := &memory.GeneratedSkill{
		Slug:            "low-conf-skill",
		Theme:           "Uncertain",
		Description:     "Low confidence skill",
		Content:         "# Low Conf\n\nContent",
		SourceMemoryIDs: "[1]",
		Alpha:           1.0,
		Beta:            4.0, // confidence = 1/(1+4) = 0.2 < 0.3
		Utility:         0.3,
		RetrievalCount:  0,
		LastRetrieved:   "",
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
		EmbeddingID:     0,
	}

	fakeEmb := make([]float32, 384)
	for i := range fakeEmb {
		fakeEmb[i] = 0.3
	}
	blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())
	embID, err := result.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	lowConfSkill.EmbeddingID = embID

	_, err = memory.InsertSkillForTest(db, lowConfSkill)
	g.Expect(err).ToNot(HaveOccurred())

	// Search with similar embedding
	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.3
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty()) // Low confidence skill should not appear
}

// TestSearchSkillsEmpty verifies that SearchSkills returns an empty slice
// when no skills match the query.
func TestSearchSkillsEmpty(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// No skills inserted

	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.5
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// TestFormatSkillContext verifies that FormatSkillContext renders markdown
// with skills in the expected format.
func TestFormatSkillContext(t *testing.T) {
	g := NewWithT(t)

	skills := []memory.SkillSearchResult{
		{
			Slug:        "git-workflow",
			Theme:       "Version Control",
			Description: "Best practices for git commit workflow",
			Confidence:  0.85,
			Utility:     0.90,
			Similarity:  0.95,
		},
		{
			Slug:        "tdd-testing",
			Theme:       "Testing",
			Description: "Test-driven development patterns",
			Confidence:  0.75,
			Utility:     0.80,
			Similarity:  0.88,
		},
	}

	output := memory.FormatSkillContextForTest(skills)
	g.Expect(output).To(ContainSubstring("## Relevant Skills"))
	g.Expect(output).To(ContainSubstring("Version Control"))
	g.Expect(output).To(ContainSubstring("Best practices for git commit workflow"))
	g.Expect(output).To(ContainSubstring("85%")) // Confidence as percentage
	g.Expect(output).To(ContainSubstring("Testing"))
	g.Expect(output).To(ContainSubstring("Test-driven development patterns"))
	g.Expect(output).To(ContainSubstring("75%"))
}

// TestFormatSkillContextEmpty verifies that FormatSkillContext returns
// an empty string when given an empty slice.
func TestFormatSkillContextEmpty(t *testing.T) {
	g := NewWithT(t)

	output := memory.FormatSkillContextForTest([]memory.SkillSearchResult{})
	g.Expect(output).To(Equal(""))
}

// TestFormatMarkdownWithSkills verifies that FormatMarkdown includes
// a skills section when FormatMarkdownOpts.Skills is populated.
func TestFormatMarkdownWithSkills(t *testing.T) {
	g := NewWithT(t)

	skills := []memory.SkillSearchResult{
		{
			Slug:        "git-workflow",
			Theme:       "Version Control",
			Description: "Best practices for git commit workflow",
			Confidence:  0.85,
			Utility:     0.90,
			Similarity:  0.95,
		},
	}

	memories := []memory.QueryResult{
		{
			Content:    "Always write clear commit messages",
			Confidence: 0.8,
			Score:      0.9,
			Source:     "memory",
			SourceType: "internal",
		},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Skills:        skills,
		Results:       memories,
		MinConfidence: 0.0,
		MaxEntries:    10,
		MaxTokens:     2000,
		Tier:          memory.TierCompact,
	})

	g.Expect(output).To(ContainSubstring("## Relevant Skills"))
	g.Expect(output).To(ContainSubstring("Version Control"))
	g.Expect(output).To(ContainSubstring("## Recent Context from Memory"))
}

// TestFormatMarkdownSkillsBeforeMemories verifies that the skills section
// appears BEFORE the memories section in FormatMarkdown output.
func TestFormatMarkdownSkillsBeforeMemories(t *testing.T) {
	g := NewWithT(t)

	skills := []memory.SkillSearchResult{
		{
			Slug:        "testing-skill",
			Theme:       "Testing",
			Description: "TDD patterns",
			Confidence:  0.80,
			Utility:     0.85,
			Similarity:  0.90,
		},
	}

	memories := []memory.QueryResult{
		{
			Content:    "Write tests first",
			Confidence: 0.75,
			Score:      0.85,
			Source:     "memory",
			SourceType: "internal",
		},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Skills:        skills,
		Results:       memories,
		MinConfidence: 0.0,
		MaxEntries:    10,
		MaxTokens:     2000,
		Tier:          memory.TierCompact,
	})

	// Find positions of skills and memories sections
	skillsPos := findSubstring(output, "## Relevant Skills")
	memoriesPos := findSubstring(output, "## Recent Context from Memory")

	g.Expect(skillsPos).To(BeNumerically(">=", 0))
	g.Expect(memoriesPos).To(BeNumerically(">=", 0))
	g.Expect(skillsPos).To(BeNumerically("<", memoriesPos)) // Skills before memories
}

// TestSearchSkillsConfidenceThreshold uses property-based testing to verify
// that only skills with confidence > 0.3 are returned.
func TestSearchSkillsConfidenceThreshold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir, err := os.MkdirTemp("", "skill-conf-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(tempDir) }()
		memoryRoot := filepath.Join(tempDir, ".claude", "memory")
		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

		db, err := memory.InitDBForTest(memoryRoot)
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = db.Close() }()

		now := time.Now().UTC().Format(time.RFC3339)

		// Generate random alpha and beta that result in confidence > 0.3
		alpha := rapid.Float64Range(1.0, 100.0).Draw(t, "alpha")
		beta := rapid.Float64Range(0.1, alpha*2.333).Draw(t, "beta") // Ensure alpha/(alpha+beta) > 0.3

		confidence := alpha / (alpha + beta)
		g.Expect(confidence).To(BeNumerically(">", 0.3))

		skill := &memory.GeneratedSkill{
			Slug:            "test-skill",
			Theme:           "Test",
			Description:     "Test description",
			Content:         "Content",
			SourceMemoryIDs: "[1]",
			Alpha:           alpha,
			Beta:            beta,
			Utility:         0.5,
			RetrievalCount:  0,
			LastRetrieved:   "",
			CreatedAt:       now,
			UpdatedAt:       now,
			Pruned:          false,
			EmbeddingID:     0,
		}

		// Insert embedding
		fakeEmb := make([]float32, 384)
		for i := range fakeEmb {
			fakeEmb[i] = 0.5
		}
		blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		g.Expect(err).ToNot(HaveOccurred())
		embID, err := result.LastInsertId()
		g.Expect(err).ToNot(HaveOccurred())

		skill.EmbeddingID = embID

		_, err = memory.InsertSkillForTest(db, skill)
		g.Expect(err).ToNot(HaveOccurred())

		// Search
		queryEmb := make([]float32, 384)
		for i := range queryEmb {
			queryEmb[i] = 0.5
		}

		results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results).ToNot(BeEmpty())
		g.Expect(results[0].Confidence).To(BeNumerically(">", 0.3))
		g.Expect(results[0].Confidence).To(BeNumerically("~", confidence, 0.01))
	})
}

// Helper function to find substring position
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestSearchSkillsNullEmbeddingID verifies that skills with NULL embedding_id
// are excluded from search results (they have no vector to search against).
func TestSearchSkillsNullEmbeddingID(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert skill with NULL embedding_id
	skill := &memory.GeneratedSkill{
		Slug:            "no-embedding",
		Theme:           "Test",
		Description:     "Skill without embedding",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           5.0,
		Beta:            1.0, // confidence = 0.83 > 0.3
		Utility:         0.75,
		RetrievalCount:  0,
		LastRetrieved:   "",
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
		EmbeddingID:     0, // NULL in database
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Search
	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.5
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty()) // Skill without embedding should not appear
}

// TestSearchSkillsLimit verifies that SearchSkills respects the limit parameter.
func TestSearchSkillsLimit(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert 5 skills
	for i := 0; i < 5; i++ {
		skill := &memory.GeneratedSkill{
			Slug:            fmt.Sprintf("test-skill-%d", i),
			Theme:           "Test",
			Description:     "Test skill",
			Content:         "Content",
			SourceMemoryIDs: "[1]",
			Alpha:           5.0,
			Beta:            1.0,
			Utility:         0.75,
			RetrievalCount:  0,
			LastRetrieved:   "",
			CreatedAt:       now,
			UpdatedAt:       now,
			Pruned:          false,
			EmbeddingID:     0,
		}

		// Insert embedding
		fakeEmb := make([]float32, 384)
		for j := range fakeEmb {
			fakeEmb[j] = 0.5
		}
		blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		g.Expect(err).ToNot(HaveOccurred())
		embID, err := result.LastInsertId()
		g.Expect(err).ToNot(HaveOccurred())

		skill.EmbeddingID = embID

		_, err = memory.InsertSkillForTest(db, skill)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Search with limit=2
	queryEmb := make([]float32, 384)
	for i := range queryEmb {
		queryEmb[i] = 0.5
	}

	results, err := memory.SearchSkillsForTest(db, queryEmb, 2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2)) // Should only return 2 results
}
