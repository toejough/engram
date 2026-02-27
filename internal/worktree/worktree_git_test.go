package worktree_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/worktree"
)

func TestManager_CleanupAll_ValidGitRepoNoWorktrees(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, mgr := setupGitRepo(t)

	err := mgr.CleanupAll()
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_CleanupAll_WithWorktrees(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	_, err := mgr.Create("TASK-cleanupall1", base)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = mgr.Create("TASK-cleanupall2", base)
	g.Expect(err).ToNot(HaveOccurred())

	err = mgr.CleanupAll()
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_CreateProject_WhenDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, mgr := setupGitRepo(t)

	g.Expect(os.MkdirAll(mgr.ProjectPath("proj-existsdir"), 0o755)).To(Succeed())

	_, err := mgr.CreateProject("proj-existsdir", "main")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("project worktree already exists"))
}

func TestManager_Create_WhenDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, mgr := setupGitRepo(t)

	g.Expect(os.MkdirAll(mgr.Path("TASK-existsdir"), 0o755)).To(Succeed())

	_, err := mgr.Create("TASK-existsdir", "main")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("worktree already exists"))
}

func TestManager_DetectBaseBranch_DetachedHead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)

	cmd := exec.Command("git", "-C", dir, "checkout", "--detach", "HEAD")
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "setup detach: %s", out)

	_, err = mgr.DetectBaseBranch()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("detached HEAD"))
}

// --- Manager tests using real git repos ---

func TestManager_DetectBaseBranch_GitRepoSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	expected := gitBaseBranch(t, dir)

	branch, err := mgr.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(branch).To(Equal(expected))
	g.Expect(branch).ToNot(BeEmpty())
}

func TestManager_List_ValidGitRepoNoManagedWorktrees(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, mgr := setupGitRepo(t)

	wts, err := mgr.List()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(wts).To(BeEmpty())
}

func TestManager_List_WithTaskAndProjectWorktrees(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	_, err := mgr.Create("TASK-listboth", base)
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.Cleanup("TASK-listboth") })

	_, err = mgr.CreateProject("proj-listboth", base)
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.CleanupProject("proj-listboth") })

	wts, err := mgr.List()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(wts).To(HaveLen(2))

	types := map[string]bool{}
	for _, wt := range wts {
		types[wt.Type] = true
	}

	g.Expect(types).To(HaveKey("task"))
	g.Expect(types).To(HaveKey("project"))
}

func TestManager_MergeProject_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	cmd := exec.Command("git", "-C", dir, "branch", "project/proj-mergeok", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	err = mgr.MergeProject("proj-mergeok", base)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_Merge_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	cmd := exec.Command("git", "-C", dir, "branch", "task/TASK-mergeok", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	err = mgr.Merge("TASK-mergeok", base)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCleanupAll_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	t.Chdir(dir)

	err := worktree.RunCleanupAll(worktree.CleanupAllArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

// --- CLI function tests using real git repos ---

func TestRunCleanup_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	t.Chdir(dir)

	err := worktree.RunCleanup(worktree.CleanupArgs{TaskID: "TASK-chdircleanup"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCreateProject_AutoDetectBaseBranch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)

	err := worktree.RunCreateProject(worktree.CreateProjectArgs{
		ProjectName: "proj-autobase",
		RepoDir:     dir,
		// BaseBranch intentionally empty → auto-detect
	})
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.CleanupProject("proj-autobase") })
}

func TestRunCreateProject_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)
	t.Chdir(dir)

	err := worktree.RunCreateProject(worktree.CreateProjectArgs{
		ProjectName: "proj-chdircreate",
		BaseBranch:  base,
	})
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.CleanupProject("proj-chdircreate") })
}

func TestRunCreate_AutoDetectBaseBranch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)

	err := worktree.RunCreate(worktree.CreateArgs{
		TaskID:  "TASK-autobase",
		RepoDir: dir,
		// BaseBranch intentionally empty → auto-detect
	})
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.Cleanup("TASK-autobase") })
}

func TestRunCreate_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)
	t.Chdir(dir)

	err := worktree.RunCreate(worktree.CreateArgs{
		TaskID:     "TASK-chdircreate",
		BaseBranch: base,
	})
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.Cleanup("TASK-chdircreate") })
}

func TestRunList_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	t.Chdir(dir)

	err := worktree.RunList(worktree.ListArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_EmptyGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)

	err := worktree.RunList(worktree.ListArgs{RepoDir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_WithWorktrees(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	_, err := mgr.Create("TASK-listrun", base)
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = mgr.Cleanup("TASK-listrun") })

	err = worktree.RunList(worktree.ListArgs{RepoDir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMergeProject_AutoDetectOnto(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base, err := mgr.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("git", "-C", dir, "branch", "project/proj-mergeontoauto", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	err = worktree.RunMergeProject(worktree.MergeProjectArgs{
		ProjectName: "proj-mergeontoauto",
		RepoDir:     dir,
		// Onto intentionally empty → auto-detect
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMergeProject_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	cmd := exec.Command("git", "-C", dir, "branch", "project/proj-mergechdirrun", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	t.Chdir(dir)

	err = worktree.RunMergeProject(worktree.MergeProjectArgs{
		ProjectName: "proj-mergechdirrun",
		Onto:        base,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_AutoDetectOnto(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base, err := mgr.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("git", "-C", dir, "branch", "task/TASK-mergeontoauto", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	err = worktree.RunMerge(worktree.MergeArgs{
		TaskID:  "TASK-mergeontoauto",
		RepoDir: dir,
		// Onto intentionally empty → auto-detect
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_DefaultRepoDir(t *testing.T) {
	// No t.Parallel: uses t.Chdir
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	cmd := exec.Command("git", "-C", dir, "branch", "task/TASK-mergechdirrun", base)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "create branch: %s", out)

	t.Chdir(dir)

	err = worktree.RunMerge(worktree.MergeArgs{
		TaskID: "TASK-mergechdirrun",
		Onto:   base,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

// gitBaseBranch returns the current branch name for the given dir.
func gitBaseBranch(t *testing.T, dir string) string {
	t.Helper()

	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("branch --show-current failed: %v", err)
	}

	return strings.TrimSpace(string(out))
}

// setupGitRepo creates a temporary git repo with an initial commit.
// Returns (repoDir, manager).
func setupGitRepo(t *testing.T) (string, *worktree.Manager) {
	t.Helper()
	g := NewWithT(t)

	base := t.TempDir()
	dir := filepath.Join(base, "repo")

	g.Expect(os.MkdirAll(dir, 0o755)).To(Succeed())

	run := func(name string, args ...string) {
		t.Helper()

		cmd := exec.Command(name, args...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v %v failed: %s", name, args, out)
		}
	}

	run("git", "init", dir)
	run("git", "-C", dir, "config", "user.email", "test@test.com")
	run("git", "-C", dir, "config", "user.name", "Test User")
	g.Expect(os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0o644)).To(Succeed())
	run("git", "-C", dir, "add", ".")
	run("git", "-C", dir, "commit", "-m", "initial commit")

	return dir, worktree.NewManager(dir)
}
