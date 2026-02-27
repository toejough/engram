package worktree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/worktree"
)

func TestManager_BranchName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager(t.TempDir())

	name := m.BranchName("TASK-42")
	g.Expect(name).To(Equal("task/TASK-42"))
}

func TestManager_CleanupAll_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	// CleanupAll calls List which fails on non-git repo
	err := m.CleanupAll()
	g.Expect(err).To(HaveOccurred())
}

func TestManager_CleanupProject_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	err := m.CleanupProject("my-feature")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_Cleanup_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	// Cleanup on non-git dir succeeds (gracefully ignores git errors)
	err := m.Cleanup("TASK-1")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestManager_CreateProject_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	_, err := m.CreateProject("my-feature", "main")
	g.Expect(err).To(HaveOccurred())
}

func TestManager_Create_NotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	_, err := m.Create("TASK-1", "main")
	g.Expect(err).To(HaveOccurred())
}

func TestManager_DetectBaseBranch_NotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	_, err := m.DetectBaseBranch()
	g.Expect(err).To(HaveOccurred())
}

func TestManager_List_NotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	_, err := m.List()
	g.Expect(err).To(HaveOccurred())
}

func TestManager_MergeProject_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	err := m.MergeProject("my-feature", "main")
	g.Expect(err).To(HaveOccurred())
}

func TestManager_Merge_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	err := m.Merge("TASK-1", "main")
	g.Expect(err).To(HaveOccurred())
}

func TestManager_ParentDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager("/repo/myproject")

	parent := m.ParentDir()
	g.Expect(parent).To(ContainSubstring("myproject-worktrees"))
}

func TestManager_Path_NoDots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager("/repo/myproject")

	path := m.Path("TASK-42")
	g.Expect(path).To(HaveSuffix("TASK-42"))
}

func TestManager_Path_SanitizesDots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager("/repo/myproject")

	path := m.Path("TASK-1.2.3")
	g.Expect(path).To(ContainSubstring("TASK-1-2-3"))
	g.Expect(path).To(ContainSubstring("myproject-worktrees"))
}

func TestManager_ProjectBranchName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager(t.TempDir())

	name := m.ProjectBranchName("my-feature")
	g.Expect(name).To(Equal("project/my-feature"))
}

func TestManager_ProjectPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	m := worktree.NewManager("/repo/myproject")

	path := m.ProjectPath("my-feature")
	g.Expect(path).To(ContainSubstring("project-my-feature"))
	g.Expect(path).To(ContainSubstring("myproject-worktrees"))
}

func TestManager_RepoDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	m := worktree.NewManager(dir)

	repoDir := m.RepoDir()
	// Should be equal or be the resolved canonical path
	g.Expect(repoDir).ToNot(BeEmpty())
}

func TestMergeConflictError_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := &worktree.MergeConflictError{
		TaskID:  "TASK-1",
		Message: "conflict in main.go",
	}
	g.Expect(err.Error()).To(ContainSubstring("TASK-1"))
	g.Expect(err.Error()).To(ContainSubstring("merge conflict"))
}

func TestNewManager_CreatesManager(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	m := worktree.NewManager(dir)
	g.Expect(m).ToNot(BeNil())
}

func TestRunCleanupAll_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunCleanupAll(worktree.CleanupAllArgs{RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunCleanup_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Cleanup succeeds trivially on non-git dirs (git errors are ignored)
	err := worktree.RunCleanup(worktree.CleanupArgs{TaskID: "TASK-1", RepoDir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunCreateProject_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunCreateProject(worktree.CreateProjectArgs{ProjectName: "my-feature", BaseBranch: "main", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunCreate_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunCreate(worktree.CreateArgs{TaskID: "TASK-1", BaseBranch: "main", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunList_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunList(worktree.ListArgs{RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunMergeProject_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunMergeProject(worktree.MergeProjectArgs{ProjectName: "my-feature", Onto: "main", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunMerge_NonGitRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := worktree.RunMerge(worktree.MergeArgs{TaskID: "TASK-1", Onto: "main", RepoDir: dir})
	g.Expect(err).To(HaveOccurred())
}
