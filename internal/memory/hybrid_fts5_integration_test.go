//go:build integration && sqlite_fts5

package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: FTS5 table created on init (requires sqlite_fts5 build tag)
func TestFTS5TableCreatedOnInit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn something to trigger DB init
	err = memory.Learn(memory.LearnOpts{
		Message:    "test fts5 init",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Open the DB directly and check sqlite_master for embeddings_fts
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='embeddings_fts'").Scan(&tableName)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(tableName).To(Equal("embeddings_fts"))
}

// TEST: learnToEmbeddings populates FTS5 (requires sqlite_fts5 build tag)
func TestLearnPopulatesFTS5(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn a unique word
	err = memory.Learn(memory.LearnOpts{
		Message:    "xylophoneUniqueWord testing FTS5 population",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query FTS5 directly for the unique word
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var content string
	err = db.QueryRow("SELECT content FROM embeddings_fts WHERE embeddings_fts MATCH 'xylophoneUniqueWord'").Scan(&content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("xylophoneUniqueWord"))
}

// TEST: BM25Enabled is true when FTS5 is available
func TestBM25EnabledWithFTS5(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "test bm25 enabled flag",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	results, err := memory.Query(memory.QueryOpts{
		Text:       "bm25 enabled",
		Limit:      3,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.BM25Enabled).To(BeTrue())
	g.Expect(results.UsedHybridSearch).To(BeTrue())
}
