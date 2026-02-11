package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestMemoryOptimizeReviewFlag(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("runs interactive review workflow with --review flag", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		memoryRoot := filepath.Join(tmpDir, "memory")
		dbPath := filepath.Join(memoryRoot, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
		skillsDir := filepath.Join(tmpDir, "skills")

		// Create memory directory
		err := os.MkdirAll(memoryRoot, 0755)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Initialize test DB
		db, err := memory.InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		defer db.Close()

		// Insert test data
		_, err = db.Exec(`
			INSERT INTO embeddings (content, source, source_type, confidence, promoted)
			VALUES ('Low confidence entry', 'memory', 'user', 0.2, 0)
		`)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Create CLAUDE.md
		claudeMDContent := `# Working With Joe

## Promoted Learnings

- Test learning one
`
		err = os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Run optimize with --review and --yes (auto-approve)
		args := memoryOptimizeArgs{
			Review:     true,
			Yes:        true,
			MemoryRoot: memoryRoot,
			ClaudeMD:   claudeMDPath,
		}

		ctx := context.Background()
		err = runInteractiveOptimize(ctx, memoryRoot, claudeMDPath, skillsDir, args)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Verify backups were cleaned up
		g.Expect(dbPath + ".bak").ToNot(gomega.BeAnExistingFile())
	})

	t.Run("legacy workflow runs without --review flag", func(t *testing.T) {
		// Setup test environment
		tmpDir := t.TempDir()
		memoryRoot := filepath.Join(tmpDir, "memory")
		dbPath := filepath.Join(memoryRoot, "embeddings.db")
		claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

		// Create memory directory
		err := os.MkdirAll(memoryRoot, 0755)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Initialize test DB
		db, err := memory.InitTestDB(dbPath)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		defer db.Close()

		// Create CLAUDE.md
		err = os.WriteFile(claudeMDPath, []byte("# Working With Joe\n"), 0644)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		// Run optimize without --review (legacy mode)
		args := memoryOptimizeArgs{
			Review:     false,
			Yes:        true,
			MemoryRoot: memoryRoot,
			ClaudeMD:   claudeMDPath,
		}

		err = memoryOptimize(args)
		g.Expect(err).ToNot(gomega.HaveOccurred())
	})
}
