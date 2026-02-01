package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/skills"
)

// TEST-055-001 traces: TASK-056
// Test that Install creates symlinks for all skills in repo
func TestInstall_CreatesSymlinksForAllSkills(t *testing.T) {
	g := NewWithT(t)

	// Setup: Create temp directories
	repoDir := t.TempDir()
	targetDir := t.TempDir()

	// Create repo skills
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"), []byte("# Skill B"), 0644)).To(Succeed())

	// When: Install all skills
	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Symlinks created
	g.Expect(result.Linked).To(ConsistOf("skill-a", "skill-b"))

	// Verify symlinks exist and point correctly
	linkA := filepath.Join(targetDir, "skill-a")
	linkB := filepath.Join(targetDir, "skill-b")

	infoA, err := os.Lstat(linkA)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(infoA.Mode()&os.ModeSymlink).ToNot(BeZero(), "skill-a should be symlink")

	targetA, err := os.Readlink(linkA)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(targetA).To(Equal(filepath.Join(skillsDir, "skill-a")))

	infoB, err := os.Lstat(linkB)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(infoB.Mode()&os.ModeSymlink).ToNot(BeZero(), "skill-b should be symlink")
}

// TEST-055-002 traces: TASK-056
// Test that Install creates symlink for specific skill
func TestInstall_CreatesSymlinkForSpecificSkill(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"), []byte("# Skill B"), 0644)).To(Succeed())

	// When: Install specific skill
	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{
		SkillName: "skill-a",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Only skill-a linked
	g.Expect(result.Linked).To(ConsistOf("skill-a"))

	// skill-b should NOT exist
	_, err = os.Lstat(filepath.Join(targetDir, "skill-b"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

// TEST-055-003 traces: TASK-056
// Test that Install warns on existing non-symlink directory
func TestInstall_WarnsOnConflict(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())

	// Create conflicting directory in target
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(targetDir, "skill-a", "SKILL.md"), []byte("# Local"), 0644)).To(Succeed())

	// When: Install (no force)
	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Conflict reported, not linked
	g.Expect(result.Conflicts).To(ConsistOf("skill-a"))
	g.Expect(result.Linked).To(BeEmpty())

	// Original directory still exists (not overwritten)
	content, err := os.ReadFile(filepath.Join(targetDir, "skill-a", "SKILL.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("# Local"))
}

// TEST-055-004 traces: TASK-056
// Test that Install with Force overwrites conflicts
func TestInstall_ForceOverwritesConflict(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Repo Version"), 0644)).To(Succeed())

	// Create conflicting directory
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(targetDir, "skill-a", "SKILL.md"), []byte("# Local"), 0644)).To(Succeed())

	// When: Install with force
	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{
		Force: true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Linked, no conflicts
	g.Expect(result.Linked).To(ConsistOf("skill-a"))
	g.Expect(result.Conflicts).To(BeEmpty())

	// Now points to repo
	info, err := os.Lstat(filepath.Join(targetDir, "skill-a"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.Mode()&os.ModeSymlink).ToNot(BeZero())
}

// TEST-055-005 traces: TASK-056
// Test that Install updates existing symlinks if target changed
func TestInstall_UpdatesExistingSymlink(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())

	// Create existing symlink pointing elsewhere
	oldTarget := t.TempDir()
	g.Expect(os.Symlink(oldTarget, filepath.Join(targetDir, "skill-a"))).To(Succeed())

	// When: Install
	result, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Updated
	g.Expect(result.Updated).To(ConsistOf("skill-a"))

	// Points to new location
	newTarget, err := os.Readlink(filepath.Join(targetDir, "skill-a"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(newTarget).To(Equal(filepath.Join(skillsDir, "skill-a")))
}

// TEST-055-006 traces: TASK-056
// Test that Install returns error for non-existent skill
func TestInstall_ErrorsOnNonexistentSkill(t *testing.T) {
	g := NewWithT(t)

	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// When: Install non-existent skill
	_, err := skills.Install(skillsDir, targetDir, skills.InstallOpts{
		SkillName: "does-not-exist",
	})

	// Then: Error
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("does-not-exist"))
}

// TEST-057-001 traces: TASK-057
// Test that Status returns linked for properly symlinked skills
func TestStatus_ReturnsLinkedForSymlinkedSkills(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())

	// Create proper symlink
	g.Expect(os.Symlink(filepath.Join(skillsDir, "skill-a"), filepath.Join(targetDir, "skill-a"))).To(Succeed())

	// When: Get status
	result, err := skills.Status(skillsDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Shows as linked
	g.Expect(result.Linked).To(ConsistOf("skill-a"))
}

// TEST-057-002 traces: TASK-057
// Test that Status returns missing for repo skills not installed
func TestStatus_ReturnsMissingForUninstalledSkills(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())

	// No symlink created - target dir is empty

	// When: Get status
	result, err := skills.Status(skillsDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Shows as missing
	g.Expect(result.Missing).To(ConsistOf("skill-a"))
}

// TEST-057-003 traces: TASK-057
// Test that Status returns local for skills only in target
func TestStatus_ReturnsLocalForTargetOnlySkills(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Create skill only in target (not in repo)
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "local-skill"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(targetDir, "local-skill", "SKILL.md"), []byte("# Local"), 0644)).To(Succeed())

	// When: Get status
	result, err := skills.Status(skillsDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Shows as local
	g.Expect(result.Local).To(ConsistOf("local-skill"))
}

// TEST-057-004 traces: TASK-057
// Test that Status returns conflict for non-symlink with same name
func TestStatus_ReturnsConflictForNonSymlink(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Repo"), 0644)).To(Succeed())

	// Create non-symlink directory in target with same name
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(targetDir, "skill-a", "SKILL.md"), []byte("# Local"), 0644)).To(Succeed())

	// When: Get status
	result, err := skills.Status(skillsDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Shows as conflict
	g.Expect(result.Conflicts).To(ConsistOf("skill-a"))
}

// TEST-057-005 traces: TASK-057
// Test that Status returns stale for symlink pointing to wrong location
func TestStatus_ReturnsStaleForWrongSymlink(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A"), 0644)).To(Succeed())

	// Create symlink pointing to wrong location
	wrongTarget := t.TempDir()
	g.Expect(os.Symlink(wrongTarget, filepath.Join(targetDir, "skill-a"))).To(Succeed())

	// When: Get status
	result, err := skills.Status(skillsDir, targetDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Shows as stale (needs update)
	g.Expect(result.Stale).To(ConsistOf("skill-a"))
}

// TEST-058-001 traces: TASK-058
// Test that Uninstall removes all symlinks
func TestUninstall_RemovesAllSymlinks(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"), []byte("# B"), 0644)).To(Succeed())

	// Create symlinks
	g.Expect(os.Symlink(filepath.Join(skillsDir, "skill-a"), filepath.Join(targetDir, "skill-a"))).To(Succeed())
	g.Expect(os.Symlink(filepath.Join(skillsDir, "skill-b"), filepath.Join(targetDir, "skill-b"))).To(Succeed())

	// When: Uninstall all
	result, err := skills.Uninstall(skillsDir, targetDir, skills.UninstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Both removed
	g.Expect(result.Removed).To(ConsistOf("skill-a", "skill-b"))

	// Symlinks no longer exist
	_, err = os.Lstat(filepath.Join(targetDir, "skill-a"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
	_, err = os.Lstat(filepath.Join(targetDir, "skill-b"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

// TEST-058-002 traces: TASK-058
// Test that Uninstall removes specific symlink
func TestUninstall_RemovesSpecificSymlink(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"), []byte("# B"), 0644)).To(Succeed())

	// Create symlinks
	g.Expect(os.Symlink(filepath.Join(skillsDir, "skill-a"), filepath.Join(targetDir, "skill-a"))).To(Succeed())
	g.Expect(os.Symlink(filepath.Join(skillsDir, "skill-b"), filepath.Join(targetDir, "skill-b"))).To(Succeed())

	// When: Uninstall specific skill
	result, err := skills.Uninstall(skillsDir, targetDir, skills.UninstallOpts{
		SkillName: "skill-a",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Only skill-a removed
	g.Expect(result.Removed).To(ConsistOf("skill-a"))

	// skill-a gone, skill-b still exists
	_, err = os.Lstat(filepath.Join(targetDir, "skill-a"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
	_, err = os.Lstat(filepath.Join(targetDir, "skill-b"))
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-058-003 traces: TASK-058
// Test that Uninstall preserves non-symlink directories
func TestUninstall_PreservesNonSymlinks(t *testing.T) {
	g := NewWithT(t)

	// Setup
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())

	// Create non-symlink directory (local skill with same name)
	g.Expect(os.MkdirAll(filepath.Join(targetDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(targetDir, "skill-a", "SKILL.md"), []byte("# Local"), 0644)).To(Succeed())

	// When: Uninstall
	result, err := skills.Uninstall(skillsDir, targetDir, skills.UninstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: Skipped (not removed)
	g.Expect(result.Removed).To(BeEmpty())
	g.Expect(result.Skipped).To(ConsistOf("skill-a"))

	// Directory still exists
	content, err := os.ReadFile(filepath.Join(targetDir, "skill-a", "SKILL.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("# Local"))
}

// TEST-058-004 traces: TASK-058
// Test that Uninstall is idempotent
func TestUninstall_Idempotent(t *testing.T) {
	g := NewWithT(t)

	// Setup - no symlinks, just empty dirs
	repoDir := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(repoDir, "skills")
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())

	// When: Uninstall (nothing to uninstall)
	result, err := skills.Uninstall(skillsDir, targetDir, skills.UninstallOpts{})
	g.Expect(err).ToNot(HaveOccurred())

	// Then: No error, nothing removed
	g.Expect(result.Removed).To(BeEmpty())
}

// TEST-600 traces: TASK-038
// Test List returns all skill names
func TestList_ReturnsAllSkills(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-c"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-b", "SKILL.md"), []byte("# B"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-c", "SKILL.md"), []byte("# C"), 0644)).To(Succeed())

	names, err := skills.List(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(names).To(ConsistOf("skill-a", "skill-b", "skill-c"))
}

// TEST-601 traces: TASK-038
// Test List excludes shared directory
func TestList_ExcludesShared(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "shared"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# A"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "shared", "RESULT.md"), []byte("# Shared"), 0644)).To(Succeed())

	names, err := skills.List(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(names).To(ConsistOf("skill-a"))
	g.Expect(names).ToNot(ContainElement("shared"))
}

// TEST-602 traces: TASK-038
// Test Docs returns SKILL-full.md if exists
func TestDocs_PrefersFullMd(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Compressed"), 0644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL-full.md"), []byte("# Full Documentation"), 0644)).To(Succeed())

	content, err := skills.Docs(skillsDir, "skill-a")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("Full Documentation"))
}

// TEST-603 traces: TASK-038
// Test Docs falls back to SKILL.md
func TestDocs_FallsBackToSkillMd(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Standard SKILL.md"), 0644)).To(Succeed())

	content, err := skills.Docs(skillsDir, "skill-a")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("Standard SKILL.md"))
}

// TEST-604 traces: TASK-038
// Test Docs errors for nonexistent skill
func TestDocs_ErrorsForNonexistent(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()

	_, err := skills.Docs(skillsDir, "nonexistent")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TEST-605 traces: TASK-038
// Test DocsSection returns specific section
func TestDocsSection_ReturnsSection(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	content := `# Skill A

## Purpose

This is the purpose section.

## Process

1. Step one
2. Step two

## Rules

- Rule one
- Rule two
`
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte(content), 0644)).To(Succeed())

	section, err := skills.DocsSection(skillsDir, "skill-a", "Process")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(section).To(ContainSubstring("Step one"))
	g.Expect(section).To(ContainSubstring("Step two"))
	g.Expect(section).ToNot(ContainSubstring("Rule one"))
}

// TEST-606 traces: TASK-038
// Test DocsSection errors for nonexistent section
func TestDocsSection_ErrorsForNonexistentSection(t *testing.T) {
	g := NewWithT(t)

	skillsDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "skill-a", "SKILL.md"), []byte("# Skill A\n\n## Purpose\n\nText."), 0644)).To(Succeed())

	_, err := skills.DocsSection(skillsDir, "skill-a", "NonexistentSection")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("section"))
}
