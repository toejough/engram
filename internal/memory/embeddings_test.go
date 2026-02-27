package memory

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestGetActivationStatsEffectivenessDefaultAlpha verifies default alpha_weight = 0.5 when not set.
func TestGetActivationStatsEffectivenessDefaultAlpha(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert a memory — alpha_weight will be the default 0.5 inserted by initEmbeddingsDB
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, importance_score, impact_score)
		VALUES ('default alpha test memory', 'test', 0.4, 0.6)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	_ = db.Close()

	stats, err := GetActivationStats(ActivationStatsOpts{
		MemoryRoot: tempDir,
		Content:    "default alpha test memory",
	})
	g.Expect(err).ToNot(HaveOccurred())

	if stats == nil {
		t.Fatal("GetActivationStats returned nil stats")
	}

	// effectiveness = 0.4 + 0.5 × 0.6 = 0.4 + 0.3 = 0.7
	g.Expect(stats.Effectiveness).To(BeNumerically("~", 0.7, 0.001))
}

// TestGetActivationStatsIncludesEffectiveness verifies that effectiveness = importance_score + α × impact_score
// per FR-016 and research R5.
func TestGetActivationStatsIncludesEffectiveness(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert a memory with known importance_score (B_i) and impact_score
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, importance_score, impact_score)
		VALUES ('test alpha impact effectiveness', 'test', 0.6, 0.4)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	// Set alpha_weight = 0.5 in metadata
	err = setMetadata(db, "alpha_weight", "0.5")
	g.Expect(err).ToNot(HaveOccurred())

	_ = db.Close()

	// Get activation stats
	stats, err := GetActivationStats(ActivationStatsOpts{
		MemoryRoot: tempDir,
		Content:    "test alpha impact effectiveness",
	})
	g.Expect(err).ToNot(HaveOccurred())

	if stats == nil {
		t.Fatal("GetActivationStats returned nil stats")
	}

	// effectiveness = importance_score + α × impact_score
	// = 0.6 + 0.5 × 0.4 = 0.8
	g.Expect(stats.Effectiveness).To(BeNumerically("~", 0.8, 0.001))
}
