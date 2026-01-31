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
