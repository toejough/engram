package memory

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestApplyEmbeddingsProposal_Decay(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert embedding to decay
	result, err := db.Exec(`
		INSERT INTO embeddings (content, source, confidence, promoted)
		VALUES ('Entry to decay', 'test', 0.6, 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("db.Exec returned nil result")
	}

	entryID, _ := result.LastInsertId()

	// Create decay proposal
	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "decay",
		Target:  strconv.FormatInt(entryID, 10),
		Reason:  "Stale (90+ days)",
		Preview: "Confidence: 0.6 → 0.3",
	}

	// Apply proposal
	err = applyEmbeddingsProposal(db, tempDir, "", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify confidence was reduced
	var newConfidence float64

	err = db.QueryRow("SELECT confidence FROM embeddings WHERE id = ?", entryID).Scan(&newConfidence)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(newConfidence).To(BeNumerically("<", 0.6))
	g.Expect(newConfidence).To(BeNumerically("~", 0.3, 0.1))
}

func TestApplyEmbeddingsProposal_Promote(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	err := os.MkdirAll(skillsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert high-value embedding
	result, err := db.Exec(`
		INSERT INTO embeddings (content, source, confidence, retrieval_count, projects_retrieved, principle, promoted)
		VALUES ('High value pattern', 'test', 0.9, 15, 'proj1,proj2,proj3', 'Always use TDD', 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("db.Exec returned nil result")
	}

	entryID, _ := result.LastInsertId()

	// Create promote proposal
	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "promote",
		Target:  strconv.FormatInt(entryID, 10),
		Reason:  "High retrieval (15x), confidence 0.9, 3 projects",
		Preview: "Generate skill: always-use-tdd",
	}

	// Apply proposal
	err = applyEmbeddingsProposal(db, tempDir, skillsDir, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify a skill was created
	var skillCount int

	err = db.QueryRow("SELECT COUNT(*) FROM generated_skills WHERE pruned = 0").Scan(&skillCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skillCount).To(BeNumerically(">=", 1))
}

func TestApplyEmbeddingsProposal_Prune(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert embedding to prune
	result, err := db.Exec(`
		INSERT INTO embeddings (content, source, confidence, promoted)
		VALUES ('Entry to prune', 'test', 0.15, 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("db.Exec returned nil result")
	}

	entryID, _ := result.LastInsertId()

	// Create prune proposal
	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "prune",
		Target: strconv.FormatInt(entryID, 10),
		Reason: "Low confidence (0.15)",
	}

	// Apply proposal
	err = applyEmbeddingsProposal(db, tempDir, "", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify entry was deleted
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE id = ?", entryID).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

func TestScanEmbeddings_HighValuePromotion(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create skills directory
	skillsDir := filepath.Join(tempDir, "skills")
	err = os.MkdirAll(skillsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert high-value embedding (retrieval 10+, confidence 0.8+, multi-project)
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, retrieval_count, projects_retrieved, principle, promoted)
		VALUES ('High value pattern', 'test', 0.9, 15, 'proj1,proj2,proj3', 'Always use TDD', 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert normal embedding (doesn't meet criteria)
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, retrieval_count, projects_retrieved, principle, promoted)
		VALUES ('Normal entry', 'test', 0.5, 2, 'proj1', 'Some principle', 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	// Scan for proposals
	proposals, err := scanEmbeddings(db, tempDir, skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Expect promote proposal for high-value entry
	promoteProposals := filterProposals(proposals, "promote")
	g.Expect(promoteProposals).ToNot(BeEmpty())

	if len(promoteProposals) < 1 {
		t.Fatal("expected at least 1 promote proposal")
	}

	g.Expect(promoteProposals[0].Tier).To(Equal("embeddings"))
	g.Expect(promoteProposals[0].Action).To(Equal("promote"))
	g.Expect(promoteProposals[0].Reason).To(ContainSubstring("High retrieval"))
}

func TestScanEmbeddings_LowConfidence(t *testing.T) {
	g := NewWithT(t)

	// Setup test DB
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert low-confidence embedding (confidence < 0.3)
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, promoted)
		VALUES ('Low confidence entry', 'test', 0.15, 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert normal confidence embedding
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, promoted)
		VALUES ('Normal confidence entry', 'test', 0.8, 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	// Low-confidence entries are now handled automatically by AutoMaintenance
	// scanEmbeddings should NOT return prune proposals for them
	proposals, err := scanEmbeddings(db, tempDir, "")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify no prune proposals are returned (handled automatically now)
	pruneProposals := filterProposals(proposals, "prune")
	g.Expect(pruneProposals).To(BeEmpty())
}

func TestScanEmbeddings_StaleEntries(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert stale embedding (>90 days old, low retrieval)
	oldTime := time.Now().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, last_retrieved, retrieval_count, promoted)
		VALUES ('Stale entry', 'test', 0.5, ?, 0, 0)
	`, oldTime)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert recent embedding
	recentTime := time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, confidence, last_retrieved, retrieval_count, promoted)
		VALUES ('Recent entry', 'test', 0.5, ?, 5, 0)
	`, recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	// Stale entries are now handled automatically by AutoMaintenance
	// scanEmbeddings should NOT return decay proposals for them
	proposals, err := scanEmbeddings(db, tempDir, "")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify no decay proposals are returned (handled automatically now)
	decayProposals := filterProposals(proposals, "decay")
	g.Expect(decayProposals).To(BeEmpty())
}

// Helper function to filter proposals by action
func filterProposals(proposals []MaintenanceProposal, action string) []MaintenanceProposal {
	var filtered []MaintenanceProposal

	for _, p := range proposals {
		if p.Action == action {
			filtered = append(filtered, p)
		}
	}

	return filtered
}
