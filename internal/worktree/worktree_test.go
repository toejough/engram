package worktree_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/worktree"
)

func TestManager_Path(t *testing.T) {
	t.Run("returns canonical worktree path", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/Users/joe/repos/personal/projctl")
		path := m.Path("TASK-001")

		g.Expect(path).To(Equal("/Users/joe/repos/personal/projctl-worktrees/TASK-001"))
	})

	t.Run("handles repo in nested directory", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/home/user/code/myproject")
		path := m.Path("TASK-042")

		g.Expect(path).To(Equal("/home/user/code/myproject-worktrees/TASK-042"))
	})

	t.Run("sanitizes task ID with dots", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/repos/myproject")
		path := m.Path("feature.login")

		// Dots converted to dashes to avoid path issues
		g.Expect(path).To(Equal("/repos/myproject-worktrees/feature-login"))
	})
}

func TestManager_BranchName(t *testing.T) {
	t.Run("returns task branch name", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/repos/myproject")
		branch := m.BranchName("TASK-001")

		g.Expect(branch).To(Equal("task/TASK-001"))
	})
}

func TestManager_RepoDir(t *testing.T) {
	t.Run("returns configured repo dir", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/repos/myproject")

		g.Expect(m.RepoDir()).To(Equal("/repos/myproject"))
	})
}

func TestManager_ParentDir(t *testing.T) {
	t.Run("returns worktrees parent directory", func(t *testing.T) {
		g := NewWithT(t)

		m := worktree.NewManager("/Users/joe/repos/personal/projctl")
		parent := m.ParentDir()

		g.Expect(parent).To(Equal("/Users/joe/repos/personal/projctl-worktrees"))
	})
}

func TestNewManager(t *testing.T) {
	t.Run("resolves relative paths to absolute", func(t *testing.T) {
		g := NewWithT(t)

		// This test verifies the manager stores an absolute path
		m := worktree.NewManager(".")
		repoDir := m.RepoDir()

		g.Expect(filepath.IsAbs(repoDir)).To(BeTrue())
	})
}
