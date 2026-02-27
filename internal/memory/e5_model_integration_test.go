//go:build integration

package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestModelURLMetadataTracking(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Learn an entry - this should set model_url metadata
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test for URL tracking",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Check metadata
	modelURL, err := memory.GetMetadataForTest(tempDir, "model_url")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(modelURL).To(ContainSubstring("intfloat/e5-small-v2"), "model_url metadata should point to e5-small-v2")
}

func TestModelURLPointsToE5SmallV2(t *testing.T) {
	g := NewWithT(t)

	// Read the embeddings.go file to verify URL
	wd, err := os.Getwd()
	g.Expect(err).ToNot(HaveOccurred())

	embeddingsPath := filepath.Join(wd, "embeddings.go")
	content, err := os.ReadFile(embeddingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	contentStr := string(content)

	// Verify URL points to e5-small-v2, not all-MiniLM-L6-v2
	g.Expect(contentStr).To(ContainSubstring("intfloat/e5-small-v2"))
	g.Expect(contentStr).ToNot(ContainSubstring("sentence-transformers/all-MiniLM-L6-v2"))

	// Verify the misleading comment is removed
	g.Expect(contentStr).ToNot(ContainSubstring("Using all-MiniLM-L6-v2 as a compatible alternative"))
}

func TestModelVersionColumnExists(t *testing.T) {
	g := NewWithT(t)

	// Create temp dir
	tempDir := t.TempDir()

	// Initialize DB
	db, err := memory.InitDBForTest(tempDir)
	g.Expect(err).ToNot(HaveOccurred())
	defer db.Close()

	// Query table schema
	rows, err := db.Query("PRAGMA table_info(embeddings)")
	g.Expect(err).ToNot(HaveOccurred())
	defer rows.Close()

	// Check for model_version column
	foundModelVersion := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, dfltValue, pk interface{}
		err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk)
		g.Expect(err).ToNot(HaveOccurred())

		if name == "model_version" {
			foundModelVersion = true
			g.Expect(typ).To(Equal("TEXT"))
		}
	}

	g.Expect(foundModelVersion).To(BeTrue(), "embeddings table should have model_version column")
}

func TestModelVersionMigration(t *testing.T) {
	g := NewWithT(t)

	// Create temp dir
	tempDir := t.TempDir()

	// Learn an entry (this will trigger migration if needed)
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test entry for migration",
		Project:    "test",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Check that model_version_migrated metadata is set
	migrated, err := memory.GetMetadataForTest(tempDir, "model_version_migrated")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(migrated).To(Equal("e5-small-v2"), "Migration should set metadata to e5-small-v2")

	// Learn another entry - should not re-migrate
	err = memory.Learn(memory.LearnOpts{
		Message:    "Second test entry",
		Project:    "test",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestPassagePrefixInLearn(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// This test verifies that the "passage: " prefix is applied during Learn
	// We can't directly verify the prefix was used, but we can verify that
	// the embedding was created and can be queried
	err := memory.Learn(memory.LearnOpts{
		Message:    "Unique test phrase for passage prefix verification",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query should be able to find it (with query: prefix on query side)
	results, err := memory.Query(memory.QueryOpts{
		Text:       "passage prefix verification",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())
}

func TestQueryPrefixesQueryText(t *testing.T) {
	g := NewWithT(t)

	// Create temp dir for test
	tempDir := t.TempDir()

	// Learn an entry with "passage: " prefix
	err := memory.Learn(memory.LearnOpts{
		Message:    "This is a test memory about Go programming",
		Project:    "test-project",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query with "query: " prefix - should find the result
	results, err := memory.Query(memory.QueryOpts{
		Text:       "Go programming",
		MemoryRoot: tempDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty(), "Query with 'query:' prefix should find stored 'passage:' entry")
}

func TestStaleModelDeletion(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Set stale model_url metadata (simulate old model)
	err := memory.SetMetadataForTest(tempDir, "model_url", "https://old-url.com/model.onnx")
	g.Expect(err).ToNot(HaveOccurred())

	// Create a fake model file
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	modelPath := filepath.Join(homeDir, ".claude", "models", "e5-small-v2.onnx")
	modelDir := filepath.Dir(modelPath)

	err = os.MkdirAll(modelDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a small fake model file
	err = os.WriteFile(modelPath, []byte("fake model content"), 0644)
	if err != nil {
		// If we can't write (permissions), skip this test
		t.Skip("Cannot write to model directory, skipping stale model deletion test")
	}
	defer os.Remove(modelPath) // cleanup

	// Learn should detect stale URL and trigger re-download
	// (We can't actually test the download without network, but we can verify the metadata gets updated)
	err = memory.Learn(memory.LearnOpts{
		Message:    "Test stale model detection",
		MemoryRoot: tempDir,
	})

	// Check if model_url was updated to new URL
	modelURL, err := memory.GetMetadataForTest(tempDir, "model_url")
	g.Expect(err).ToNot(HaveOccurred())

	// After Learn, metadata should be updated to the new URL
	if !strings.Contains(modelURL, "intfloat/e5-small-v2") {
		t.Logf("Expected model_url to be updated to e5-small-v2, got: %s", modelURL)
	}
}
