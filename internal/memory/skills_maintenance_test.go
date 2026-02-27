//go:build integration && sqlite_fts5

package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"github.com/toejough/projctl/internal/memory"
)

// Custom Gomega matcher for MaintenanceProposal
func MatchProposal(tier, action, target string) types.GomegaMatcher {
	return &proposalMatcher{
		expectedTier:   tier,
		expectedAction: action,
		expectedTarget: target,
	}
}

func TestApplySkillsProposal_Decay(t *testing.T) {
	g := NewWithT(t)

	// Given: A low-utility skill
	memoryRoot := t.TempDir()
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "decay-skill", "Decay Skill", "Desc", "# Content", "[]",
		5.0, 5.0, 0.35, 2, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	skillID, _ := result.LastInsertId()

	proposal := memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "decay-skill",
		Reason: "Low utility",
	}

	// When: Applying decay
	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Beta should increase (decreases confidence/utility)
	var alpha, beta, utility float64
	err = db.QueryRow("SELECT alpha, beta, utility FROM generated_skills WHERE id = ?", skillID).Scan(&alpha, &beta, &utility)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(beta).To(BeNumerically(">", 5.0))     // Beta increased
	g.Expect(utility).To(BeNumerically("<", 0.35)) // Utility decreased
}

func TestApplySkillsProposal_Promote(t *testing.T) {
	g := NewWithT(t)

	// Given: A high-utility skill
	memoryRoot := t.TempDir()
	claudeMDPath := filepath.Join(memoryRoot, "CLAUDE.md")
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Create initial CLAUDE.md with Promoted Learnings section
	initialContent := `# Working With Joe

## Promoted Learnings

- Existing learning
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(initialContent), 0644)).To(Succeed())

	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "promote-skill", "Always use TDD", "TDD pattern", "# Content\nAlways write tests first", "[1,2,3]",
		9.0, 1.0, 0.88, 20, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	skillID, _ := result.LastInsertId()

	proposal := memory.MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Target:  "promote-skill",
		Reason:  "High utility across 4 projects",
		Preview: "Always write tests first before implementation",
	}

	// When: Applying promotion
	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{
		ClaudeMDPath: claudeMDPath,
	})
	err = applier.Apply(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Should be added to CLAUDE.md
	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("Always write tests first"))

	// And: Should be marked as promoted in DB
	var claudeMDPromoted int
	err = db.QueryRow("SELECT claude_md_promoted FROM generated_skills WHERE id = ?", skillID).Scan(&claudeMDPromoted)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(claudeMDPromoted).To(Equal(1))
}

func TestApplySkillsProposal_Prune(t *testing.T) {
	g := NewWithT(t)

	// Given: A skill and a prune proposal
	memoryRoot := t.TempDir()
	skillsDir := filepath.Join(memoryRoot, "skills")
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "old-skill", "Old Skill", "Desc", "# Content", "[]",
		1.0, 1.0, 0.2, 0, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	skillID, _ := result.LastInsertId()

	// Create skill file
	skillDir := filepath.Join(skillsDir, "mem-old-skill")
	g.Expect(os.MkdirAll(skillDir, 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Old Skill"), 0644)).To(Succeed())

	proposal := memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: "old-skill",
		Reason: "Unused for 60 days",
	}

	// When: Applying the prune proposal
	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{
		SkillsDir: skillsDir,
	})
	err = applier.Apply(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Skill should be soft-deleted in DB
	var pruned int
	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", skillID).Scan(&pruned)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(Equal(1))

	// And: Skill directory should be removed
	_, err = os.Stat(skillDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestScanSkills_DetectsHighUtilityMultiProjectSkills(t *testing.T) {
	g := NewWithT(t)

	// Given: A high-utility skill used across 3+ projects
	memoryRoot := t.TempDir()
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, last_retrieved, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "universal-pattern", "Universal Pattern", "Applies everywhere", "# Content", "[]",
		9.0, 1.0, 0.85, 15, now, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	skillID, _ := result.LastInsertId()

	// Create skill_usage table and record usage across 4 projects
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (
		skill_id INTEGER,
		project TEXT,
		timestamp TEXT
	)`)
	g.Expect(err).ToNot(HaveOccurred())

	for _, proj := range []string{"proj-a", "proj-b", "proj-c", "proj-d"} {
		_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)",
			skillID, proj, now)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// When: Scanning
	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		PromoteUtilityThreshold: 0.7,
		PromoteMinProjects:      3,
	})
	proposals, err := scanner.Scan()
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Should propose promotion to CLAUDE.md
	g.Expect(proposals).To(ContainElement(MatchProposal("skills", "promote", "universal-pattern")))

	promoteProposal := findProposalByAction(proposals, "promote")
	g.Expect(promoteProposal.Reason).To(ContainSubstring("4 projects"))
	g.Expect(promoteProposal.Reason).To(ContainSubstring("0.85"))
}

func TestScanSkills_DetectsLowUtilitySkills(t *testing.T) {
	g := NewWithT(t)

	// Given: A skill with low utility (but some recent usage)
	memoryRoot := t.TempDir()
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, last_retrieved, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "low-util", "Low Utility", "Pattern with low utility", "# Content", "[]",
		2.0, 8.0, 0.28, 3, now, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	// When: Scanning
	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		LowUtilityThreshold: 0.4,
	})
	proposals, err := scanner.Scan()
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Should propose decay
	g.Expect(proposals).To(ContainElement(MatchProposal("skills", "decay", "low-util")))
}

func TestScanSkills_DetectsRedundantSkills(t *testing.T) {
	g := NewWithT(t)

	// Given: Two skills with high semantic similarity
	memoryRoot := t.TempDir()
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().Format(time.RFC3339)

	// Create two similar skills with embeddings
	skill1ID := insertSkillWithEmbedding(t, db, "pattern-a", "Test Pattern Alpha", 0.8, now)
	skill2ID := insertSkillWithEmbedding(t, db, "pattern-b", "Test Pattern Beta", 0.75, now)

	// When: Scanning with similarity detector
	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		SimilarityThreshold: 0.85,
		SimilarityFunc: func(db *sql.DB, emb1, emb2 int64) (float64, error) {
			// Mock high similarity
			return 0.92, nil
		},
	})
	proposals, err := scanner.Scan()
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Should propose consolidation
	g.Expect(proposals).To(ContainElement(MatchProposal("skills", "consolidate", "")))

	// Verify both skills are mentioned in consolidation proposal
	consolidateProposal := findProposalByAction(proposals, "consolidate")
	g.Expect(consolidateProposal).ToNot(BeNil())
	g.Expect(consolidateProposal.Target).To(Or(
		Equal("pattern-a"),
		Equal("pattern-b"),
	))

	_, _ = skill1ID, skill2ID
}

func TestScanSkills_DetectsUnusedSkills(t *testing.T) {
	g := NewWithT(t)

	// Given: A skill with no retrievals in 45 days and low utility
	memoryRoot := t.TempDir()
	db, err := memory.OpenSkillDB(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Create unused skill (45 days old, no retrievals, low utility)
	oldDate := time.Now().AddDate(0, 0, -45).Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, last_retrieved, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "unused-pattern", "Unused Pattern", "Old unused pattern", "# Content", "[]",
		1.0, 1.0, 0.25, 0, nil, oldDate, oldDate, 0)
	g.Expect(err).ToNot(HaveOccurred())

	// When: Scanning for maintenance proposals
	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		UnusedDaysThreshold: 30,
		LowUtilityThreshold: 0.4,
	})
	proposals, err := scanner.Scan()
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Should propose pruning
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0]).To(MatchProposal("skills", "prune", "unused-pattern"))
	g.Expect(proposals[0].Reason).To(ContainSubstring("no retrievals"))
	g.Expect(proposals[0].Reason).To(ContainSubstring("45 days"))
}

type proposalMatcher struct {
	expectedTier   string
	expectedAction string
	expectedTarget string
}

func (m *proposalMatcher) FailureMessage(actual interface{}) (message string) {
	return "Expected proposal to match criteria"
}

func (m *proposalMatcher) Match(actual interface{}) (success bool, err error) {
	proposal, ok := actual.(memory.MaintenanceProposal)
	if !ok {
		return false, nil
	}

	if m.expectedTier != "" && proposal.Tier != m.expectedTier {
		return false, nil
	}
	if m.expectedAction != "" && proposal.Action != m.expectedAction {
		return false, nil
	}
	if m.expectedTarget != "" && proposal.Target != m.expectedTarget {
		return false, nil
	}

	return true, nil
}

func (m *proposalMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "Expected proposal NOT to match criteria"
}

func findProposalByAction(proposals []memory.MaintenanceProposal, action string) *memory.MaintenanceProposal {
	for i := range proposals {
		if proposals[i].Action == action {
			return &proposals[i]
		}
	}
	return nil
}

// Helper functions

func insertSkillWithEmbedding(t *testing.T, db *sql.DB, slug, theme string, utility float64, timestamp string) int64 {
	g := NewWithT(t)

	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, created_at, updated_at, pruned, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, slug, theme, "Description", "# Content", "[]",
		8.0, 2.0, utility, 5, timestamp, timestamp, 0, 100) // Mock embedding_id = 100

	g.Expect(err).ToNot(HaveOccurred())
	id, _ := result.LastInsertId()
	return id
}
