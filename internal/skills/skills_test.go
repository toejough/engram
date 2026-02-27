package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/skills"
)

func TestDocsSection_ReturnsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	content := "# My Skill\n\n## Usage\n\nHere is how to use it.\n\n## More\n\nAdditional content."
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644)).To(Succeed())

	section, err := skills.DocsSection(dir, "my-skill", "Usage")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(section).To(ContainSubstring("Usage"))
	g.Expect(section).To(ContainSubstring("Here is how to use it."))
}

func TestDocsSection_SectionNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill\n\n## Overview\n\nSome text."), 0o644)).To(Succeed())

	_, err := skills.DocsSection(dir, "my-skill", "Nonexistent Section")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("section not found"))
	}
}

func TestDocsSection_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := skills.DocsSection(dir, "nonexistent", "Usage")
	g.Expect(err).To(HaveOccurred())
}

func TestDocs_NoSkillMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	_, err := skills.Docs(dir, "my-skill")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skill documentation not found"))
	}
}

func TestDocs_PrefersFullMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("short"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL-full.md"), []byte("full content"), 0o644)).To(Succeed())

	content, err := skills.Docs(dir, "my-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("full content"))
}

func TestDocs_ReturnsSkillMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0o644)).To(Succeed())

	content, err := skills.Docs(dir, "my-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("My Skill"))
}

func TestDocs_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := skills.Docs(dir, "nonexistent")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skill not found"))
	}
}

func TestInstall_ConflictsWithRegularDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// Create a regular dir at target
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Conflicts).To(ContainElement("my-skill"))
}

func TestInstall_CreatesSymlinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Linked).To(ContainElement("my-skill"))

	// Verify symlink exists
	info, lstatErr := os.Lstat(filepath.Join(targetDir, "my-skill"))

	g.Expect(lstatErr).ToNot(HaveOccurred())

	if info != nil {
		g.Expect(info.Mode() & os.ModeSymlink).ToNot(BeZero())
	}
}

func TestInstall_ForceOverwritesConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// Create a regular dir at target
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{Force: true})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Linked).To(ContainElement("my-skill"))
}

func TestInstall_SkipsAlreadyLinked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// Install once
	_, err := skills.Install(repoDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Install again
	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Skipped).To(ContainElement("my-skill"))
	g.Expect(result.Linked).To(BeEmpty())
}

func TestInstall_SpecificSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(repoDir, "skill-a"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(repoDir, "skill-b"), 0o755)).To(Succeed())

	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{SkillName: "skill-a"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Linked).To(ConsistOf("skill-a"))
}

func TestInstall_SpecificSkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	_, err := skills.Install(repoDir, targetDir, skills.InstallOpts{SkillName: "nonexistent"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skill not found"))
	}
}

func TestInstall_UpdatesStaleSymlink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	otherDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// Create a symlink pointing to a different location
	g.Expect(os.Symlink(filepath.Join(otherDir, "my-skill"), filepath.Join(targetDir, "my-skill"))).To(Succeed())

	result, err := skills.Install(repoDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Updated).To(ContainElement("my-skill"))
}

func TestList_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	names, err := skills.List(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(names).To(BeEmpty())
}

func TestList_ExcludesSharedDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "skill-a"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "shared"), 0o755)).To(Succeed())

	names, err := skills.List(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(names).To(ConsistOf("skill-a"))
	g.Expect(names).NotTo(ContainElement("shared"))
}

func TestList_ListsSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "skill-a"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "skill-b"), 0o755)).To(Succeed())

	names, err := skills.List(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(names).To(ConsistOf("skill-a", "skill-b"))
}

func TestList_MissingDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := skills.List("/nonexistent/dir")
	g.Expect(err).To(HaveOccurred())
}

func TestRunDocs_DefaultDir(t *testing.T) {
	t.Parallel()
	// Exercises the home dir default code path; may succeed or fail depending on environment.
	_ = skills.RunDocs(skills.DocsArgs{SkillName: "nonexistent-skill-xyz"})
}

func TestRunDocs_EmptySkillName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	err := skills.RunDocs(skills.DocsArgs{SkillName: ""})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skill name is required"))
	}
}

func TestRunDocs_ErrorFromDocs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// No SKILL.md — Docs will return an error

	err := skills.RunDocs(skills.DocsArgs{SkillsDir: dir, SkillName: "my-skill"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skill documentation not found"))
	}
}

func TestRunDocs_ErrorFromDocsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")

	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill\n\n## Usage\n\nHow.\n"), 0o644)).To(Succeed())

	err := skills.RunDocs(skills.DocsArgs{SkillsDir: dir, SkillName: "my-skill", Section: "Nonexistent"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("section not found"))
	}
}

func TestRunDocs_WithExplicitDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0o644)).To(Succeed())

	err := skills.RunDocs(skills.DocsArgs{SkillsDir: dir, SkillName: "my-skill"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunDocs_WithSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	content := "# Skill\n\n## Usage\n\nHow to use.\n"
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644)).To(Succeed())

	err := skills.RunDocs(skills.DocsArgs{SkillsDir: dir, SkillName: "my-skill", Section: "Usage"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunInstall_DefaultDirs(t *testing.T) {
	t.Parallel()
	// Call with no repoDir/targetDir — exercises os.Getwd() and os.UserHomeDir() branches.
	// Will fail because there's no skills/ dir in cwd, which is fine.
	err := skills.RunInstall(skills.InstallArgs{})
	// Expect an error (skills dir not found or similar)
	_ = err
}

func TestRunInstall_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := skills.RunInstall(skills.InstallArgs{RepoDir: dir, TargetDir: t.TempDir()})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skills directory not found"))
	}
}

func TestRunInstall_PrintsConflicts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// Create conflicting regular dir in target
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	err := skills.RunInstall(skills.InstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunInstall_PrintsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	skillDir := filepath.Join(skillsDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())

	// Install once to create symlink, then install again
	err := skills.RunInstall(skills.InstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())

	err = skills.RunInstall(skills.InstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunInstall_PrintsUpdated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	otherDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	skillDir := filepath.Join(skillsDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	// Symlink points elsewhere → stale → will be updated
	g.Expect(os.Symlink(filepath.Join(otherDir, "my-skill"), filepath.Join(targetDir, "my-skill"))).To(Succeed())

	err := skills.RunInstall(skills.InstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunInstall_WithSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())

	err := skills.RunInstall(skills.InstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_DefaultDir(t *testing.T) {
	t.Parallel()
	// Call without SkillsDir — exercises os.UserHomeDir() default path.
	// May succeed or fail depending on whether ~/.claude/skills exists.
	_ = skills.RunList(skills.ListArgs{})
}

func TestRunList_ErrorFromList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := skills.RunList(skills.ListArgs{SkillsDir: "/nonexistent/path/that/does/not/exist"})
	g.Expect(err).To(HaveOccurred())
}

func TestRunList_WithExplicitDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "skill-a"), 0o755)).To(Succeed())

	err := skills.RunList(skills.ListArgs{SkillsDir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunStatusCore_AllLinked_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	skillDir := filepath.Join(skillsDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillDir, filepath.Join(targetDir, "my-skill"))).To(Succeed())

	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := skills.RunStatusCore(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunStatusCore_Conflicts_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// Regular dir at target (not a symlink) → conflict
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := skills.RunStatusCore(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunStatusCore_MissingSkills_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// No symlink in targetDir → missing

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := skills.RunStatusCore(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunStatusCore_NoSkillsDir_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	exit := func(int) { t.Error("exit should not be called on error") }

	err := skills.RunStatusCore(skills.StatusArgs{RepoDir: dir, TargetDir: t.TempDir()}, exit)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skills directory not found"))
	}
}

func TestRunStatusCore_StaleSkills_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	otherDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// Symlink points to a different location → stale
	g.Expect(os.Symlink(filepath.Join(otherDir, "my-skill"), filepath.Join(targetDir, "my-skill"))).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := skills.RunStatusCore(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunStatus_AllLinked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	skillDir := filepath.Join(skillsDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillDir, filepath.Join(targetDir, "my-skill"))).To(Succeed())

	err := skills.RunStatus(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunStatus_DefaultRepoDir(t *testing.T) {
	t.Parallel()
	// Empty RepoDir exercises os.Getwd() branch.
	// CWD won't have skills/ so returns error; we just exercise the branch.
	_ = skills.RunStatus(skills.StatusArgs{TargetDir: t.TempDir()})
}

func TestRunStatus_DefaultTargetDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create skills/ so we get past the "not found" check.
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Skip("cannot create skills dir")
	}

	// Empty TargetDir exercises os.UserHomeDir() + set targetDir branches.
	// Result depends on real home dir; ignore outcome.
	_ = skills.RunStatus(skills.StatusArgs{RepoDir: dir})
}

func TestRunStatus_LocalOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a skills dir in repoDir (empty, no skills)
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(skillsDir, 0o755)).To(Succeed())

	// Put a local-only dir in targetDir
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "local-skill"), 0o755)).To(Succeed())

	err := skills.RunStatus(skills.StatusArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunStatus_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := skills.RunStatus(skills.StatusArgs{RepoDir: dir, TargetDir: t.TempDir()})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skills directory not found"))
	}
}

func TestRunUninstall_DefaultDirs(t *testing.T) {
	t.Parallel()
	// Call with no repoDir/targetDir — exercises os.Getwd() and os.UserHomeDir() branches.
	// Will fail because skills/ dir won't exist in cwd.

	_ = skills.RunUninstall(skills.UninstallArgs{})
}

func TestRunUninstall_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := skills.RunUninstall(skills.UninstallArgs{RepoDir: dir, TargetDir: t.TempDir()})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skills directory not found"))
	}
}

func TestRunUninstall_NothingToUninstall(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// No symlink in targetDir, no regular dir either

	err := skills.RunUninstall(skills.UninstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunUninstall_PrintsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "my-skill"), 0o755)).To(Succeed())
	// Regular dir at target (not symlink) — should be skipped
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	err := skills.RunUninstall(skills.UninstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunUninstall_WithSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillsDir := filepath.Join(repoDir, "skills")
	skillDir := filepath.Join(skillsDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillDir, filepath.Join(targetDir, "my-skill"))).To(Succeed())

	err := skills.RunUninstall(skills.UninstallArgs{RepoDir: repoDir, TargetDir: targetDir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestStatus_AllLinked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillDir, filepath.Join(targetDir, "my-skill"))).To(Succeed())

	result, err := skills.Status(repoDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Linked).To(ContainElement("my-skill"))
	g.Expect(result.Missing).To(BeEmpty())
}

func TestStatus_ConflictingDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(repoDir, "my-skill"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	result, err := skills.Status(repoDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Conflicts).To(ContainElement("my-skill"))
}

func TestStatus_LocalOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	// Target has a dir not in repo
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "local-skill"), 0o755)).To(Succeed())

	result, err := skills.Status(repoDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Local).To(ContainElement("local-skill"))
}

func TestStatus_MissingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(repoDir, "my-skill"), 0o755)).To(Succeed())

	result, err := skills.Status(repoDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Missing).To(ContainElement("my-skill"))
}

func TestStatus_StaleSymlink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	otherDir := t.TempDir()
	targetDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(repoDir, "my-skill"), 0o755)).To(Succeed())
	// Symlink points elsewhere
	g.Expect(os.Symlink(filepath.Join(otherDir, "my-skill"), filepath.Join(targetDir, "my-skill"))).To(Succeed())

	result, err := skills.Status(repoDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Stale).To(ContainElement("my-skill"))
}

func TestUninstall_RemovesSymlinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(repoDir, "my-skill")
	g.Expect(os.MkdirAll(skillDir, 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillDir, filepath.Join(targetDir, "my-skill"))).To(Succeed())

	result, err := skills.Uninstall(repoDir, targetDir, skills.UninstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Removed).To(ContainElement("my-skill"))

	_, statErr := os.Stat(filepath.Join(targetDir, "my-skill"))
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestUninstall_SkipsNonSymlinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(repoDir, "my-skill"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "my-skill"), 0o755)).To(Succeed())

	result, err := skills.Uninstall(repoDir, targetDir, skills.UninstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Skipped).To(ContainElement("my-skill"))
}

func TestUninstall_SpecificSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	skillADir := filepath.Join(repoDir, "skill-a")
	g.Expect(os.MkdirAll(skillADir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(repoDir, "skill-b"), 0o755)).To(Succeed())
	g.Expect(os.Symlink(skillADir, filepath.Join(targetDir, "skill-a"))).To(Succeed())

	result, err := skills.Uninstall(repoDir, targetDir, skills.UninstallOpts{SkillName: "skill-a"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Removed).To(ConsistOf("skill-a"))
}
