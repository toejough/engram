package worktree_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// setupTestRepo creates a temporary git repo for testing.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	g := NewWithT(t)

	dir := t.TempDir()
	// Resolve symlinks for macOS
	dir, _ = filepath.EvalSymlinks(dir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	g.Expect(cmd.Run()).To(Succeed())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	g.Expect(cmd.Run()).To(Succeed())
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = dir
	g.Expect(cmd.Run()).To(Succeed())

	// Create initial commit (required for worktrees)
	testFile := filepath.Join(dir, "README.md")
	g.Expect(os.WriteFile(testFile, []byte("# Test"), 0o644)).To(Succeed())
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	g.Expect(cmd.Run()).To(Succeed())
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	g.Expect(cmd.Run()).To(Succeed())

	return dir
}

func TestManager_Create(t *testing.T) {
	t.Run("creates worktree and branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)
		path, err := m.Create("TASK-001")

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(path).To(Equal(m.Path("TASK-001")))

		// Verify worktree directory exists
		info, err := os.Stat(path)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.IsDir()).To(BeTrue())

		// Verify branch was created
		cmd := exec.Command("git", "branch", "--list", "task/TASK-001")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("task/TASK-001"))

		// Cleanup
		_ = m.Cleanup("TASK-001")
	})

	t.Run("creates parent directory if needed", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Parent shouldn't exist yet
		_, err := os.Stat(m.ParentDir())
		g.Expect(os.IsNotExist(err)).To(BeTrue())

		// Create should make it
		path, err := m.Create("TASK-002")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(path).ToNot(BeEmpty())

		// Parent should exist now
		info, err := os.Stat(m.ParentDir())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.IsDir()).To(BeTrue())

		// Cleanup
		_ = m.Cleanup("TASK-002")
	})

	t.Run("returns error if worktree already exists", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create first time
		_, err := m.Create("TASK-003")
		g.Expect(err).ToNot(HaveOccurred())

		// Create second time should fail
		_, err = m.Create("TASK-003")
		g.Expect(err).To(HaveOccurred())

		// Cleanup
		_ = m.Cleanup("TASK-003")
	})
}

func TestManager_Merge(t *testing.T) {
	t.Run("rebases and fast-forward merges task branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Get current branch name
		cmd := exec.Command("git", "branch", "--show-current")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		targetBranch := strings.TrimSpace(string(output))

		// Create worktree
		wtPath, err := m.Create("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Make a commit in the worktree
		testFile := filepath.Join(wtPath, "task-001.txt")
		g.Expect(os.WriteFile(testFile, []byte("task work"), 0o644)).To(Succeed())
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "TASK-001 work")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())

		// Merge back
		err = m.Merge("TASK-001", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify commit is now on target branch
		cmd = exec.Command("git", "log", "--oneline", "-1")
		cmd.Dir = repoDir
		output, err = cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("TASK-001 work"))

		// Verify task file exists in main repo
		_, err = os.Stat(filepath.Join(repoDir, "task-001.txt"))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("handles merge with diverged history", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		cmd := exec.Command("git", "branch", "--show-current")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		targetBranch := strings.TrimSpace(string(output))

		// Create worktree
		wtPath, err := m.Create("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Make commit in worktree
		g.Expect(os.WriteFile(filepath.Join(wtPath, "task.txt"), []byte("task"), 0o644)).To(Succeed())
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "task commit")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())

		// Make commit in main repo (diverge)
		g.Expect(os.WriteFile(filepath.Join(repoDir, "main.txt"), []byte("main"), 0o644)).To(Succeed())
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repoDir
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "main commit")
		cmd.Dir = repoDir
		g.Expect(cmd.Run()).To(Succeed())

		// Merge should rebase and succeed
		err = m.Merge("TASK-001", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Both files should exist
		_, err = os.Stat(filepath.Join(repoDir, "task.txt"))
		g.Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(filepath.Join(repoDir, "main.txt"))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error on conflict", func(t *testing.T) {
		g := NewWithT(t)
		repoDir := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		cmd := exec.Command("git", "branch", "--show-current")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		targetBranch := strings.TrimSpace(string(output))

		// Create worktree
		wtPath, err := m.Create("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Modify same file in worktree
		g.Expect(os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("task version"), 0o644)).To(Succeed())
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "task modifies readme")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())

		// Modify same file in main repo
		g.Expect(os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("main version"), 0o644)).To(Succeed())
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repoDir
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "main modifies readme")
		cmd.Dir = repoDir
		g.Expect(cmd.Run()).To(Succeed())

		// Merge should fail with conflict
		err = m.Merge("TASK-001", targetBranch)
		g.Expect(err).To(HaveOccurred())

		// Should be a MergeConflictError
		var conflictErr *worktree.MergeConflictError
		g.Expect(errors.As(err, &conflictErr)).To(BeTrue())
		g.Expect(conflictErr.TaskID).To(Equal("TASK-001"))
	})
}
