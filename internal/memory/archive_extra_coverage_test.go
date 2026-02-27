//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestPruneArchive_ExecError verifies PruneArchive returns error when table is missing.
func TestPruneArchive_ExecError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec("DROP TABLE embeddings_archive")
	g.Expect(err).ToNot(HaveOccurred())

	count, err := memory.PruneArchive(db, 30)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to prune archive"))
	g.Expect(count).To(Equal(0))
}
