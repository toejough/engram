package worktree_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/worktree"
)

func TestManager_CleanupAll_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)
	err := m.CleanupAll()
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_CleanupAll_WithWorktreesCoverage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = m.Create("TASK-cleanup-all", branch)
	g.Expect(err).ToNot(HaveOccurred())

	// CleanupAll covers the loop body
	err = m.CleanupAll()
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_CreateProject_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	path, err := m.CreateProject("cov-project", branch)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	_ = m.CleanupProject("cov-project")
}

func TestManager_Create_AlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = m.Create("TASK-dup", branch)
	g.Expect(err).ToNot(HaveOccurred())

	// Second create on same taskID should fail — worktree path already exists
	_, err = m.Create("TASK-dup", branch)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("already exists"))

	_ = m.Cleanup("TASK-dup")
}

func TestManager_Create_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	path, err := m.Create("TASK-cov", branch)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	// Cleanup to avoid leaking state (ignore errors — test already passed)
	_ = m.Cleanup("TASK-cov")
}

func TestManager_DetectBaseBranch_DetachedHEAD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	// Detach HEAD so rev-parse returns "HEAD"
	out, err := exec.Command("git", "-C", repoDir, "checkout", "--detach").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	m := worktree.NewManager(repoDir)
	_, err = m.DetectBaseBranch()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("detached HEAD"))
}

// --- Git repo success paths ---

func TestManager_DetectBaseBranch_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)
	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(branch).ToNot(BeEmpty())
}

// TestManager_DetectBaseBranch_ViaRemote exercises the symbolic-ref success path by
// cloning a local repo — the clone has refs/remotes/origin/HEAD already set.
func TestManager_DetectBaseBranch_ViaRemote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	remoteDir := makeGitRepo(t)

	root := t.TempDir()
	cloneDir := filepath.Join(root, "clone")

	out, err := exec.Command("git", "clone", "file://"+remoteDir, cloneDir).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	m := worktree.NewManager(cloneDir)
	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(branch).ToNot(BeEmpty())
}

func TestManager_List_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)
	wts, err := m.List()
	g.Expect(err).ToNot(HaveOccurred())
	// No task/ worktrees yet — result should be an empty (non-nil) slice or nil
	g.Expect(wts).To(BeEmpty())
}

func TestManager_MergeProject_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = m.CreateProject("cov-merge-proj", branch)
	g.Expect(err).ToNot(HaveOccurred())

	err = m.MergeProject("cov-merge-proj", branch)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_MergeProject_RebaseConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)
	projectName := "conflictproj"

	// Add a shared file
	g.Expect(os.WriteFile(filepath.Join(dir, "conflict2.txt"), []byte("original\n"), 0o644)).To(Succeed())

	out, err := exec.Command("git", "-C", dir, "add", "conflict2.txt").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	out, err = exec.Command("git", "-C", dir, "commit", "-m", "add conflict2.txt").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	// Create project branch with conflicting change
	out, err = exec.Command("git", "-C", dir, "checkout", "-b", "project/"+projectName).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	g.Expect(os.WriteFile(filepath.Join(dir, "conflict2.txt"), []byte("project-change\n"), 0o644)).To(Succeed())

	out, err = exec.Command("git", "-C", dir, "commit", "-a", "-m", "project change").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	// Return to base and commit a different conflicting change
	out, err = exec.Command("git", "-C", dir, "checkout", base).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	g.Expect(os.WriteFile(filepath.Join(dir, "conflict2.txt"), []byte("base-change\n"), 0o644)).To(Succeed())

	out, err = exec.Command("git", "-C", dir, "commit", "-a", "-m", "base change").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	err = mgr.MergeProject(projectName, base)
	g.Expect(err).To(HaveOccurred())

	var conflictErr *worktree.MergeConflictError
	g.Expect(errors.As(err, &conflictErr)).To(BeTrue())
}

func TestManager_MergeProject_UnregisteredWorktreeDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	out, err := exec.Command("git", "-C", dir, "branch", "project/proj-unwt", base).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	g.Expect(os.MkdirAll(mgr.ProjectPath("proj-unwt"), 0o755)).To(Succeed())

	err = mgr.MergeProject("proj-unwt", base)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_Merge_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	m := worktree.NewManager(repoDir)

	branch, err := m.DetectBaseBranch()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = m.Create("TASK-merge-cov", branch)
	g.Expect(err).ToNot(HaveOccurred())

	// Merge back onto the base branch (already up to date — no-op rebase)
	err = m.Merge("TASK-merge-cov", branch)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_Merge_RebaseConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	repoDir, base, taskBranch := makeConflictingRepo(t)

	m := worktree.NewManager(repoDir)
	err := m.Merge(taskBranch, base)
	g.Expect(err).To(HaveOccurred())

	var conflictErr *worktree.MergeConflictError
	g.Expect(errors.As(err, &conflictErr)).To(BeTrue())
}

// TestManager_Merge_UnregisteredWorktreeDir exercises the path where the worktree
// directory exists on disk but is not registered with git. This forces
// "git worktree remove" to fail, then falls through to os.RemoveAll + prune.
func TestManager_Merge_UnregisteredWorktreeDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, mgr := setupGitRepo(t)
	base := gitBaseBranch(t, dir)

	// Create the task branch directly (not via git worktree add)
	out, err := exec.Command("git", "-C", dir, "branch", "task/TASK-unwt", base).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	// Manually create a directory at the expected worktree path (not registered)
	g.Expect(os.MkdirAll(mgr.Path("TASK-unwt"), 0o755)).To(Succeed())

	// Merge: os.Stat succeeds → git worktree remove fails → os.RemoveAll cleans it → prune → rebase
	err = mgr.Merge("TASK-unwt", base)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCreateProject_NoBaseBranch_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunCreateProject(worktree.CreateProjectArgs{ProjectName: "proj", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("detect base branch"))
	}
}

// --- CLI auto-detect branch tests (non-git repo → DetectBaseBranch fails) ---

func TestRunCreate_NoBaseBranch_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunCreate(worktree.CreateArgs{TaskID: "TASK-99", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("detect base branch"))
	}
}

func TestRunList_GitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := makeGitRepo(t)

	err := worktree.RunList(worktree.ListArgs{RepoDir: repoDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMergeProject_NoOnto_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunMergeProject(worktree.MergeProjectArgs{ProjectName: "proj", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("detect base branch"))
	}
}

func TestRunMerge_NoOnto_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunMerge(worktree.MergeArgs{TaskID: "TASK-99", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("detect base branch"))
	}
}

// makeConflictingRepo creates a git repo with diverging changes on two branches
// so that rebasing the task branch onto base results in a CONFLICT.
func makeConflictingRepo(t *testing.T) (repoDir, base, taskBranch string) {
	t.Helper()
	g := NewWithT(t)

	dir, _ := setupGitRepo(t)
	base = gitBaseBranch(t, dir)
	taskBranch = "TASK-conflictcov"

	// Add a shared file on base
	g.Expect(os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("original\n"), 0o644)).To(Succeed())

	out, err := exec.Command("git", "-C", dir, "add", "conflict.txt").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	out, err = exec.Command("git", "-C", dir, "commit", "-m", "add conflict.txt").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	// Create task branch and commit conflicting change
	out, err = exec.Command("git", "-C", dir, "checkout", "-b", "task/"+taskBranch).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	g.Expect(os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("task-change\n"), 0o644)).To(Succeed())

	out, err = exec.Command("git", "-C", dir, "commit", "-a", "-m", "task change").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	// Return to base and commit a different conflicting change
	out, err = exec.Command("git", "-C", dir, "checkout", base).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	g.Expect(os.WriteFile(filepath.Join(dir, "conflict.txt"), []byte("base-change\n"), 0o644)).To(Succeed())

	out, err = exec.Command("git", "-C", dir, "commit", "-a", "-m", "base change").CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), string(out))

	return dir, base, taskBranch
}

// makeGitRepo creates an isolated git repository with one empty commit.
// The repo lives inside a temp root so that any worktrees created alongside it
// are also inside the temp root and are cleaned up automatically.
func makeGitRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoDir := filepath.Join(root, "repo")

	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", repoDir, err)
	}

	for _, cmd := range [][]string{
		{"git", "-C", repoDir, "init"},
		{"git", "-C", repoDir, "config", "user.email", "t@t.com"},
		{"git", "-C", repoDir, "config", "user.name", "T"},
		{"git", "-C", repoDir, "commit", "--allow-empty", "-m", "init"},
	} {
		out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("cmd %v: %v: %s", cmd, err, out)
		}
	}

	return repoDir
}
