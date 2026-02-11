//go:build integration

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

// setupTestRepo creates a temporary git repo for testing.
// Returns the repo directory and the name of the default branch.
func setupTestRepo(t *testing.T) (string, string) {
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

	// Get the default branch name
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	output, err := cmd.Output()
	g.Expect(err).ToNot(HaveOccurred())
	branchName := strings.TrimSpace(string(output))

	return dir, branchName
}

func TestManager_DetectBaseBranch(t *testing.T) {
	t.Parallel()
	t.Run("detects current branch as base", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, _ := setupTestRepo(t)
		m := worktree.NewManager(repoDir)

		branch, err := m.DetectBaseBranch()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(branch).ToNot(BeEmpty())
	})

	t.Run("returns the default branch name", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, expectedBranch := setupTestRepo(t)
		m := worktree.NewManager(repoDir)

		branch, err := m.DetectBaseBranch()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(branch).To(Equal(expectedBranch))
	})
}

func TestManager_Create(t *testing.T) {
	t.Parallel()
	t.Run("creates worktree and branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)
		path, err := m.Create("TASK-001", branchName)

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
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Parent shouldn't exist yet
		_, err := os.Stat(m.ParentDir())
		g.Expect(os.IsNotExist(err)).To(BeTrue())

		// Create should make it
		path, err := m.Create("TASK-002", branchName)
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
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create first time
		_, err := m.Create("TASK-003", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Create second time should fail
		_, err = m.Create("TASK-003", branchName)
		g.Expect(err).To(HaveOccurred())

		// Cleanup
		_ = m.Cleanup("TASK-003")
	})
}

func TestManager_Merge(t *testing.T) {
	t.Parallel()
	t.Run("rebases and fast-forward merges task branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, targetBranch := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		wtPath, err := m.Create("TASK-001", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Make a commit in the worktree
		testFile := filepath.Join(wtPath, "task-001.txt")
		g.Expect(os.WriteFile(testFile, []byte("task work"), 0o644)).To(Succeed())
		cmd := exec.Command("git", "add", ".")
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
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("TASK-001 work"))

		// Verify task file exists in main repo
		_, err = os.Stat(filepath.Join(repoDir, "task-001.txt"))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("handles merge with diverged history", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, targetBranch := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		wtPath, err := m.Create("TASK-001", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Make commit in worktree
		g.Expect(os.WriteFile(filepath.Join(wtPath, "task.txt"), []byte("task"), 0o644)).To(Succeed())
		cmd := exec.Command("git", "add", ".")
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
		repoDir, targetBranch := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		wtPath, err := m.Create("TASK-001", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Modify same file in worktree
		g.Expect(os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("task version"), 0o644)).To(Succeed())
		cmd := exec.Command("git", "add", ".")
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

func TestManager_List(t *testing.T) {
	t.Parallel()
	t.Run("returns empty slice when no task worktrees exist", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, _ := setupTestRepo(t)

		m := worktree.NewManager(repoDir)
		worktrees, err := m.List()

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(BeEmpty())
	})

	t.Run("returns worktree after create", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create a worktree
		wtPath, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// List should return it
		worktrees, err := m.List()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(HaveLen(1))

		wt := worktrees[0]
		g.Expect(wt.TaskID).To(Equal("TASK-001"))
		g.Expect(wt.Path).To(Equal(wtPath))
		g.Expect(wt.Branch).To(Equal("task/TASK-001"))

		// Cleanup
		_ = m.Cleanup("TASK-001")
	})

	t.Run("returns multiple worktrees", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create multiple worktrees
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// List should return both
		worktrees, err := m.List()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(HaveLen(2))

		// Extract task IDs
		taskIDs := make([]string, len(worktrees))
		for i, wt := range worktrees {
			taskIDs[i] = wt.TaskID
		}
		g.Expect(taskIDs).To(ConsistOf("TASK-001", "TASK-002"))

		// Cleanup
		_ = m.Cleanup("TASK-001")
		_ = m.Cleanup("TASK-002")
	})

	t.Run("filters out non-task branches", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create a task worktree
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Create a non-task worktree manually (feature branch)
		featurePath := filepath.Join(m.ParentDir(), "feature-x")
		cmd := exec.Command("git", "worktree", "add", "-b", "feature/x", featurePath)
		cmd.Dir = repoDir
		g.Expect(cmd.Run()).To(Succeed())

		// List should only return task worktree
		worktrees, err := m.List()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(HaveLen(1))
		g.Expect(worktrees[0].TaskID).To(Equal("TASK-001"))

		// Cleanup
		_ = m.Cleanup("TASK-001")
		cmd = exec.Command("git", "worktree", "remove", "--force", featurePath)
		cmd.Dir = repoDir
		_ = cmd.Run()
	})
}

func TestManager_CreateProject(t *testing.T) {
	t.Parallel()
	t.Run("creates project worktree and branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)
		path, err := m.CreateProject("my-feature", branchName)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(path).To(Equal(m.ProjectPath("my-feature")))

		// Verify worktree directory exists
		info, err := os.Stat(path)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.IsDir()).To(BeTrue())

		// Verify branch was created
		cmd := exec.Command("git", "branch", "--list", "project/my-feature")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("project/my-feature"))

		// Cleanup
		_ = m.CleanupProject("my-feature")
	})

	t.Run("returns error if project worktree already exists", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		_, err := m.CreateProject("my-feature", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		_, err = m.CreateProject("my-feature", branchName)
		g.Expect(err).To(HaveOccurred())

		_ = m.CleanupProject("my-feature")
	})
}

func TestManager_MergeProject(t *testing.T) {
	t.Parallel()
	t.Run("rebases and fast-forward merges project branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, targetBranch := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		wtPath, err := m.CreateProject("my-feature", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Make a commit in the project worktree
		testFile := filepath.Join(wtPath, "feature.txt")
		g.Expect(os.WriteFile(testFile, []byte("feature work"), 0o644)).To(Succeed())
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "feature work")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())

		// Merge back
		err = m.MergeProject("my-feature", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify commit is on target branch
		cmd = exec.Command("git", "log", "--oneline", "-1")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("feature work"))

		// Verify file exists in main repo
		_, err = os.Stat(filepath.Join(repoDir, "feature.txt"))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error on conflict", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, targetBranch := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		wtPath, err := m.CreateProject("my-feature", targetBranch)
		g.Expect(err).ToNot(HaveOccurred())

		// Modify same file in project worktree
		g.Expect(os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("project version"), 0o644)).To(Succeed())
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = wtPath
		g.Expect(cmd.Run()).To(Succeed())
		cmd = exec.Command("git", "commit", "-m", "project modifies readme")
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
		err = m.MergeProject("my-feature", targetBranch)
		g.Expect(err).To(HaveOccurred())

		var conflictErr *worktree.MergeConflictError
		g.Expect(errors.As(err, &conflictErr)).To(BeTrue())
	})
}

func TestManager_CleanupProject(t *testing.T) {
	t.Parallel()
	t.Run("removes project worktree and branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		wtPath, err := m.CreateProject("my-feature", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(wtPath).To(BeADirectory())

		err = m.CleanupProject("my-feature")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(wtPath).ToNot(BeADirectory())

		// Branch should be gone
		cmd := exec.Command("git", "branch", "--list", "project/my-feature")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(strings.TrimSpace(string(output))).To(BeEmpty())
	})
}

func TestManager_ListIncludesProjects(t *testing.T) {
	t.Parallel()
	t.Run("lists both task and project worktrees", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.CreateProject("my-feature", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		worktrees, err := m.List()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(HaveLen(2))

		// Extract types
		types := make(map[string]bool)
		for _, wt := range worktrees {
			types[wt.Type] = true
		}
		g.Expect(types).To(HaveKey("task"))
		g.Expect(types).To(HaveKey("project"))

		_ = m.Cleanup("TASK-001")
		_ = m.CleanupProject("my-feature")
	})

	t.Run("project worktree has correct type field", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		_, err := m.CreateProject("my-project", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		worktrees, err := m.List()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(worktrees).To(HaveLen(1))
		g.Expect(worktrees[0].Type).To(Equal("project"))
		g.Expect(worktrees[0].TaskID).To(Equal("my-project"))
		g.Expect(worktrees[0].Branch).To(Equal("project/my-project"))

		_ = m.CleanupProject("my-project")
	})
}

func TestManager_Cleanup(t *testing.T) {
	t.Parallel()
	t.Run("removes worktree directory", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		wtPath, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(wtPath).To(BeADirectory())

		// Cleanup
		err = m.Cleanup("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Worktree directory should be gone
		g.Expect(wtPath).ToNot(BeADirectory())
	})

	t.Run("deletes the task branch", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify branch exists
		cmd := exec.Command("git", "branch", "--list", "task/TASK-001")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(output)).To(ContainSubstring("task/TASK-001"))

		// Cleanup
		err = m.Cleanup("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Branch should be gone
		cmd = exec.Command("git", "branch", "--list", "task/TASK-001")
		cmd.Dir = repoDir
		output, err = cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(strings.TrimSpace(string(output))).To(BeEmpty())
	})

	t.Run("removes parent dir if empty", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktree
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Parent dir should exist
		g.Expect(m.ParentDir()).To(BeADirectory())

		// Cleanup
		err = m.Cleanup("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Parent dir should be gone (was empty)
		g.Expect(m.ParentDir()).ToNot(BeADirectory())
	})

	t.Run("leaves parent dir if other worktrees exist", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create two worktrees
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Cleanup only one
		err = m.Cleanup("TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Parent dir should still exist
		g.Expect(m.ParentDir()).To(BeADirectory())

		// Cleanup the other
		_ = m.Cleanup("TASK-002")
	})

	t.Run("succeeds even if worktree does not exist", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, _ := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Cleanup non-existent worktree should not error
		err := m.Cleanup("TASK-NONEXISTENT")
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func TestManager_CleanupAll(t *testing.T) {
	t.Parallel()
	t.Run("removes all task worktrees", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create multiple worktrees
		path1, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		path2, err := m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		path3, err := m.Create("TASK-003", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// All should exist
		g.Expect(path1).To(BeADirectory())
		g.Expect(path2).To(BeADirectory())
		g.Expect(path3).To(BeADirectory())

		// Cleanup all
		err = m.CleanupAll()
		g.Expect(err).ToNot(HaveOccurred())

		// All worktree directories should be gone
		g.Expect(path1).ToNot(BeADirectory())
		g.Expect(path2).ToNot(BeADirectory())
		g.Expect(path3).ToNot(BeADirectory())
	})

	t.Run("deletes all task branches", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create multiple worktrees
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Cleanup all
		err = m.CleanupAll()
		g.Expect(err).ToNot(HaveOccurred())

		// All branches should be gone
		cmd := exec.Command("git", "branch", "--list", "task/*")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(strings.TrimSpace(string(output))).To(BeEmpty())
	})

	t.Run("removes parent dir when all cleaned", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktrees
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Parent dir should exist
		g.Expect(m.ParentDir()).To(BeADirectory())

		// Cleanup all
		err = m.CleanupAll()
		g.Expect(err).ToNot(HaveOccurred())

		// Parent dir should be gone
		g.Expect(m.ParentDir()).ToNot(BeADirectory())
	})

	t.Run("succeeds when no worktrees exist", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, _ := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// No worktrees created, cleanup should still succeed
		err := m.CleanupAll()
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("leaves no artifacts", func(t *testing.T) {
		g := NewWithT(t)
		repoDir, branchName := setupTestRepo(t)

		m := worktree.NewManager(repoDir)

		// Create worktrees
		_, err := m.Create("TASK-001", branchName)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = m.Create("TASK-002", branchName)
		g.Expect(err).ToNot(HaveOccurred())

		// Cleanup all
		err = m.CleanupAll()
		g.Expect(err).ToNot(HaveOccurred())

		// Verify no task branches remain
		cmd := exec.Command("git", "branch", "--list", "task/*")
		cmd.Dir = repoDir
		output, err := cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(strings.TrimSpace(string(output))).To(BeEmpty())

		// Verify parent directory doesn't exist
		g.Expect(m.ParentDir()).ToNot(BeADirectory())

		// Verify git worktree list only shows main worktree
		cmd = exec.Command("git", "worktree", "list")
		cmd.Dir = repoDir
		output, err = cmd.Output()
		g.Expect(err).ToNot(HaveOccurred())
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		g.Expect(lines).To(HaveLen(1)) // Only the main worktree
	})
}
