package state_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

func TestFindRepoRoot(t *testing.T) {
	t.Run("returns git root for directory inside repo", func(t *testing.T) {
		g := NewWithT(t)

		// Create a temp dir with git init
		dir := t.TempDir()
		// Resolve symlinks (macOS /var -> /private/var)
		dir, _ = filepath.EvalSymlinks(dir)

		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		err := cmd.Run()
		g.Expect(err).ToNot(HaveOccurred())

		// Create a subdirectory
		subdir := filepath.Join(dir, "sub", "nested")
		g.Expect(os.MkdirAll(subdir, 0o755)).To(Succeed())

		// FindRepoRoot from subdir should return the git root
		root, err := state.FindRepoRoot(subdir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(root).To(Equal(dir))
	})

	t.Run("returns error for directory outside repo", func(t *testing.T) {
		g := NewWithT(t)

		// Create a temp dir WITHOUT git init
		dir := t.TempDir()

		_, err := state.FindRepoRoot(dir)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not in a git repository"))
	})

	t.Run("returns git root from repo root itself", func(t *testing.T) {
		g := NewWithT(t)

		// Create a temp dir with git init
		dir := t.TempDir()
		// Resolve symlinks (macOS /var -> /private/var)
		dir, _ = filepath.EvalSymlinks(dir)

		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		err := cmd.Run()
		g.Expect(err).ToNot(HaveOccurred())

		// FindRepoRoot from root should return itself
		root, err := state.FindRepoRoot(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(root).To(Equal(dir))
	})
}
