package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestComputeMainProjectDir_NotInWorktreeReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportComputeMainProjectDir(context.Background(), t.TempDir(), "/home/user")).To(BeEmpty())
}

func TestOsDirListMd_FlatNonRecursive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	g.Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subDir, "nested.md"), []byte("c"), 0o600)).To(Succeed())

	got, err := cli.ExportOsDirListMd(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(ConsistOf(filepath.Join(dir, "a.md")))
}

func TestOsDirListMd_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got, err := cli.ExportOsDirListMd(filepath.Join(t.TempDir(), "missing"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(BeEmpty())
}

func TestOsMatchAny_TrueWhenAtLeastOneFileMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x"), 0o600)).To(Succeed())

	matchAny := cli.ExportOsMatchAny(dir)
	g.Expect(matchAny([]string{"*.go"})).To(BeTrue())
	g.Expect(matchAny([]string{"*.rs"})).To(BeFalse())
	g.Expect(matchAny(nil)).To(BeFalse())
}

func TestOsStatExists_FalseForMissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	exists, err := cli.ExportOsStatExists(filepath.Join(t.TempDir(), "missing.md"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(exists).To(BeFalse())
}

func TestOsStatExists_TrueForExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "exists.md")
	g.Expect(os.WriteFile(path, []byte("body"), 0o600)).To(Succeed())

	exists, err := cli.ExportOsStatExists(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(exists).To(BeTrue())
}

func TestOsWalkMd_FindsMarkdownRecursively(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	g.Expect(os.MkdirAll(subDir, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "top.md"), []byte("a"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("b"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subDir, "nested.md"), []byte("c"), 0o600)).To(Succeed())

	found := cli.ExportOsWalkMd(dir)
	g.Expect(found).To(ContainElement(filepath.Join(dir, "top.md")))
	g.Expect(found).To(ContainElement(filepath.Join(subDir, "nested.md")))
	g.Expect(found).NotTo(ContainElement(filepath.Join(dir, "skip.txt")))
}

func TestOsWalkMd_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportOsWalkMd(filepath.Join(t.TempDir(), "no-such"))).To(BeEmpty())
}

func TestOsWalkSkills_FindsOnlySkillMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	skillDir := filepath.Join(dir, "myskill")
	g.Expect(os.MkdirAll(skillDir, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("a"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "other.md"), []byte("b"), 0o600)).To(Succeed())

	found := cli.ExportOsWalkSkills(dir)
	g.Expect(found).To(ConsistOf(filepath.Join(skillDir, "SKILL.md")))
}

func TestReadAutoMemoryDirectorySetting_HonorsLocalThenUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	home := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(home, ".claude"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(home, ".claude", "settings.json"),
		[]byte(`{"autoMemoryDirectory": "/from/user/settings"}`),
		0o600,
	)).To(Succeed())

	settings := cli.ExportReadAutoMemoryDirectorySetting(home)
	dir, ok := settings()
	g.Expect(ok).To(BeTrue())
	g.Expect(dir).To(Equal("/from/user/settings"))
}

func TestReadAutoMemoryDirectorySetting_MissingReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := cli.ExportReadAutoMemoryDirectorySetting(filepath.Join(t.TempDir(), "no-claude"))
	dir, ok := settings()
	g.Expect(ok).To(BeFalse())
	g.Expect(dir).To(BeEmpty())
}
