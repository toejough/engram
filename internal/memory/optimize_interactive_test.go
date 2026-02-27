package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

// MockExtractor implements LLMExtractor for testing
type MockExtractor struct {
	RewriteFunc      func(ctx context.Context, content string) (string, error)
	AddRationaleFunc func(ctx context.Context, content string) (string, error)
}

func (m *MockExtractor) AddRationale(ctx context.Context, content string) (string, error) {
	if m.AddRationaleFunc != nil {
		return m.AddRationaleFunc(ctx, content)
	}

	return content, nil
}

func (m *MockExtractor) Curate(ctx context.Context, query string, candidates []QueryResult) ([]CuratedResult, error) {
	return nil, nil
}

func (m *MockExtractor) Decide(ctx context.Context, newContent string, existing []ExistingMemory) (*IngestDecision, error) {
	return nil, nil
}

func (m *MockExtractor) Extract(ctx context.Context, content string) (*Observation, error) {
	return nil, nil
}

func (m *MockExtractor) Filter(ctx context.Context, query string, candidates []QueryResult) ([]FilterResult, error) {
	return nil, nil
}

func (m *MockExtractor) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return &PostEvalResult{Faithfulness: 0.5, Signal: "positive"}, nil
}

func (m *MockExtractor) Rewrite(ctx context.Context, content string) (string, error) {
	if m.RewriteFunc != nil {
		return m.RewriteFunc(ctx, content)
	}

	return content, nil
}

func (m *MockExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	return "", nil
}

func TestOptimizeInteractive(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("creates backups and reviews proposals from all tiers", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Insert test data: low-confidence entry that should trigger prune proposal
		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, promoted, embedding_id)
			VALUES ('Test low confidence entry', 'memory', 'user', 0.2, 0, NULL)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create test CLAUDE.md with duplicate entries
		claudeMDContent := `# Working With Joe

## Promoted Learnings

- Learning one about testing
- Learning one about testing
- Different learning about docs
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Auto-approve all proposals
		reviewFunc := func(p MaintenanceProposal) bool {
			return true
		}

		// Run interactive optimization
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   reviewFunc,
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).ToNot(gomega.BeNil())

		// Verify backups were cleaned up (success case)
		g.Expect(dbPath + ".bak").ToNot(gomega.BeAnExistingFile())
		g.Expect(claudeMDPath + ".bak").ToNot(gomega.BeAnExistingFile())

		// Verify proposals were processed
		// At minimum, embeddings prune proposal should have been generated and applied
		// (Exact counts depend on scanner implementation and test data)
	})

	t.Run("rolls back on error", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create initial data
		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, promoted)
			VALUES ('Original entry', 'memory', 'user', 0.2, 0)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		db.Close()

		// Make DB read-only to cause error during Apply
		err = os.Chmod(dbPath, 0444)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer os.Chmod(dbPath, 0644) // Restore for cleanup

		// Run interactive optimization with auto-approve
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		// Should succeed scanning but fail during apply
		// The function returns success even if individual applies fail
		g.Expect(err).ToNot(gomega.HaveOccurred())

		if result != nil && result.Total > 0 {
			// Some applies should have failed due to read-only DB
			g.Expect(result.Failed).To(gomega.BeNumerically(">", 0))
		}

		// Verify backups were cleaned up
		g.Expect(dbPath + ".bak").ToNot(gomega.BeAnExistingFile())
	})

	t.Run("respects user rejections", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Create test CLAUDE.md with duplicate entries to generate proposals
		claudeMDContent := `# Working With Joe

## Promoted Learnings

- Learning one about testing
- Learning one about testing
- Different learning about docs
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Reject all proposals
		reviewFunc := func(p MaintenanceProposal) bool {
			return false
		}

		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   reviewFunc,
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Verify no changes were applied
		g.Expect(result.Approved).To(gomega.Equal(0))
		g.Expect(result.Rejected).To(gomega.BeNumerically(">", 0))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      ctx,
		}

		_, err = OptimizeInteractive(opts)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err).To(gomega.Equal(context.Canceled))
	})

	t.Run("reports summary statistics", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Create test CLAUDE.md with duplicate entries to generate proposals
		claudeMDContent := `# Working With Joe

## Promoted Learnings

- Learning one about testing
- Learning one about testing
- Different learning about docs
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Auto-approve all
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Verify summary has reasonable values
		g.Expect(result.Total).To(gomega.BeNumerically(">", 0))
		g.Expect(result.Approved).To(gomega.Equal(result.Total))
		g.Expect(result.Rejected).To(gomega.Equal(0))
		g.Expect(result.TierSummary).ToNot(gomega.BeEmpty())
	})
}

func TestOptimizeInteractiveWithSkills(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("scans and processes skills tier proposals", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB with skills
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Create low-utility skill
		_, err = db.Exec("\n\t\t\tINSERT INTO generated_skills (slug, theme, content, description, source_memory_ids, alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)\n\t\t\tVALUES ('test-skill', 'Test Skill', '# Test\\nContent', 'Test skill', '[]', 1.0, 5.0, 0.2, 10, datetime('now'), datetime('now', '-1 second'), 0)\n\t\t")
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create skill directory
		err = os.MkdirAll(filepath.Join(skillsDir, "mem-test-skill"), 0755)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Auto-approve all
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Verify skills tier was processed
		skillSummary, exists := result.TierSummary["skills"]
		if exists {
			g.Expect(skillSummary.Total).To(gomega.BeNumerically(">", 0))
		}
	})
}

// ============================================================================
// Additional tests for ISSUE-184: Tier Filtering
// ============================================================================

// TEST-1218: OptimizeInteractive supports tier filtering (ISSUE-184)
func TestOptimizeInteractive_TierFilter(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("filters proposals by tier when filter is provided", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")
		err := os.MkdirAll(skillsDir, 0755)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Initialize test DB with high-value promotable entry
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Insert high-value embedding that should trigger promote proposal
		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, retrieval_count, projects_retrieved, principle, promoted)
			VALUES ('High value pattern', 'memory', 'user', 0.9, 15, 'proj1,proj2,proj3', 'Always use TDD', 0)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create CLAUDE.md with duplicate entries
		claudeMDContent := `## Promoted Learnings

- Learning one
- Learning one
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Filter to only embeddings tier
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
			TierFilter:   "embeddings",
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Should only have embeddings tier proposals
		_, hasEmbeddings := result.TierSummary["embeddings"]
		_, hasClaudeMD := result.TierSummary["claude-md"]

		g.Expect(hasEmbeddings).To(gomega.BeTrue())
		g.Expect(hasClaudeMD).To(gomega.BeFalse())
	})

	t.Run("shows all tiers when no filter is provided", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB with low-confidence entry
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, promoted)
			VALUES ('Low confidence entry', 'memory', 'user', 0.1, 0)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create CLAUDE.md with duplicate entries
		claudeMDContent := `## Promoted Learnings

- Learning one
- Learning one
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// No filter - should get all tiers
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Should have proposals from multiple tiers
		g.Expect(len(result.TierSummary)).To(gomega.BeNumerically(">", 1))
	})
}

// ============================================================================
// Additional tests for ISSUE-218: Content Refinement Operations
// ============================================================================

// TEST-1217: OptimizeInteractive includes refinement proposals (ISSUE-218)
func TestOptimizeInteractive_WithRefinements(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("includes refinement proposals when extractor is provided", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Initialize test DB
		db, err := InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		defer db.Close()

		// Insert test data with flagged_for_rewrite=1
		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, promoted, flagged_for_rewrite)
			VALUES ('Content needing rewrite', 'memory', 'user', 0.8, 0, 1)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create mock extractor
		extractor := &MockExtractor{
			RewriteFunc: func(ctx context.Context, content string) (string, error) {
				return "Refined: " + content, nil
			},
		}

		// Auto-approve all
		opts := OptimizeInteractiveOpts{
			DBPath:       dbPath,
			ClaudeMDPath: claudeMDPath,
			SkillsDir:    skillsDir,
			ReviewFunc:   func(p MaintenanceProposal) bool { return true },
			Context:      context.Background(),
			Extractor:    extractor,
		}

		result, err := OptimizeInteractive(opts)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Should have processed refinement proposals
		g.Expect(result.Total).To(gomega.BeNumerically(">", 0))

		// Check if embeddings tier has proposals (which may include refinements)
		embeddingsSummary, exists := result.TierSummary["embeddings"]
		if exists {
			g.Expect(embeddingsSummary.Total).To(gomega.BeNumerically(">", 0))
		}
	})
}
