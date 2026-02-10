//go:build sqlite_fts5

package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestMigrateMemoryGenSkills verifies that migrateMemoryGenSkills moves
// skills from memory-gen/{slug}/ to mem-{slug}/ and removes memory-gen/.
func TestMigrateMemoryGenSkills(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create old-style memory-gen/foo/SKILL.md
	fooDir := filepath.Join(skillsDir, "memory-gen", "foo")
	g.Expect(os.MkdirAll(fooDir, 0755)).To(Succeed())
	fooContent := "# Foo Skill\n\nFoo content here."
	g.Expect(os.WriteFile(filepath.Join(fooDir, "SKILL.md"), []byte(fooContent), 0644)).To(Succeed())

	// Create old-style memory-gen/bar/SKILL.md
	barDir := filepath.Join(skillsDir, "memory-gen", "bar")
	g.Expect(os.MkdirAll(barDir, 0755)).To(Succeed())
	barContent := "# Bar Skill\n\nBar content here."
	g.Expect(os.WriteFile(filepath.Join(barDir, "SKILL.md"), []byte(barContent), 0644)).To(Succeed())

	// Call migration
	err := memory.MigrateMemoryGenSkillsForTest(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Assert mem-foo/SKILL.md exists with same content
	memFooPath := filepath.Join(skillsDir, "mem-foo", "SKILL.md")
	content, err := os.ReadFile(memFooPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(fooContent))

	// Assert mem-bar/SKILL.md exists with same content
	memBarPath := filepath.Join(skillsDir, "mem-bar", "SKILL.md")
	content, err = os.ReadFile(memBarPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(barContent))

	// Assert memory-gen/ directory is gone
	memoryGenDir := filepath.Join(skillsDir, "memory-gen")
	_, err = os.Stat(memoryGenDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "memory-gen directory should be removed")
}

// TestMigrateMemoryGenSkillsIdempotent verifies that calling migration twice
// is safe and doesn't cause errors.
func TestMigrateMemoryGenSkillsIdempotent(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")

	// Create old-style memory-gen/foo/SKILL.md
	fooDir := filepath.Join(skillsDir, "memory-gen", "foo")
	g.Expect(os.MkdirAll(fooDir, 0755)).To(Succeed())
	fooContent := "# Foo Skill\n\nFoo content here."
	g.Expect(os.WriteFile(filepath.Join(fooDir, "SKILL.md"), []byte(fooContent), 0644)).To(Succeed())

	// First migration call
	err := memory.MigrateMemoryGenSkillsForTest(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Second migration call (should be idempotent)
	err = memory.MigrateMemoryGenSkillsForTest(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify mem-foo/SKILL.md still exists with correct content
	memFooPath := filepath.Join(skillsDir, "mem-foo", "SKILL.md")
	content, err := os.ReadFile(memFooPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(fooContent))

	// Verify memory-gen/ directory is still gone
	memoryGenDir := filepath.Join(skillsDir, "memory-gen")
	_, err = os.Stat(memoryGenDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "memory-gen directory should remain removed")
}

// TestMigrateMemoryGenSkillsNoOp verifies that migration is a no-op when
// there is no memory-gen/ directory to migrate.
func TestMigrateMemoryGenSkillsNoOp(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, ".claude", "skills")
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Call migration with no memory-gen/ directory present
	err := memory.MigrateMemoryGenSkillsForTest(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
}
