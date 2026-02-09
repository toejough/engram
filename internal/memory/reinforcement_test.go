package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	_ "github.com/mattn/go-sqlite3"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for confidence reinforcement (retrieval boost + learn-time dedup)
// traces: ISSUE-184
// ============================================================================

// TEST-1120: Querying a memory boosts its confidence
func TestQueryBoostsConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn something
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "use gomega for assertions",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Decay it first so boost is observable (starts at 1.0, won't increase past 1.0)
	_, err := memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryRoot,
		Factor:     0.8,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Get confidence after decay
	initialConf := getConfidence(g, memoryRoot, "gomega")
	g.Expect(initialConf).To(BeNumerically("<", 1.0))

	// Query to trigger retrieval boost
	_, err = memory.Query(memory.QueryOpts{
		Text:       "gomega assertions",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Confidence should have increased by 0.05
	newConf := getConfidence(g, memoryRoot, "gomega")
	g.Expect(newConf).To(BeNumerically(">", initialConf))
}

// TEST-1121: Confidence caps at 1.0
func TestConfidenceCapsAtOne(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn something (starts at confidence 1.0)
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "confidence cap test",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query multiple times to try to exceed 1.0
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "confidence cap test",
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	conf := getConfidence(g, memoryRoot, "confidence cap")
	g.Expect(conf).To(BeNumerically("<=", 1.0))
}

// TEST-1122: Learning a near-duplicate (>0.9 similarity) boosts existing instead of creating new
func TestLearnDedupBoostsExisting(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn original — the exact same message will have similarity 1.0
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always use gomega matchers for test assertions in Go",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Decay it to test boost
	_, err := memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryRoot,
		Factor:     0.8,
	})
	g.Expect(err).ToNot(HaveOccurred())

	confBefore := getConfidence(g, memoryRoot, "gomega matchers")

	// Learn near-duplicate (identical message triggers similarity 1.0)
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always use gomega matchers for test assertions in Go",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	confAfter := getConfidence(g, memoryRoot, "gomega matchers")

	// Confidence should have been boosted
	g.Expect(confAfter).To(BeNumerically(">", confBefore))

	// Should NOT have created a second embedding entry — count embeddings matching
	count := countEmbeddings(g, memoryRoot, "gomega matchers")
	g.Expect(count).To(Equal(1), "Should not create duplicate embedding entry")
}

// getConfidence reads confidence for an entry matching the content substring.
func getConfidence(g Gomega, memoryRoot string, contentSubstr string) float64 {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var conf float64
	err = db.QueryRow("SELECT confidence FROM embeddings WHERE content LIKE ? LIMIT 1", "%"+contentSubstr+"%").Scan(&conf)
	g.Expect(err).ToNot(HaveOccurred())
	return conf
}

// countEmbeddings counts embeddings matching a content substring.
func countEmbeddings(g Gomega, memoryRoot string, contentSubstr string) int {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE content LIKE ?", "%"+contentSubstr+"%").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	return count
}
