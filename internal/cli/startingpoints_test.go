package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestOsVaultFS_ListMD_FiltersDirsAndNonMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "note.md"), []byte("x"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("y"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsVaultFS()
	names, err := fs.ListMD(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(ConsistOf("note.md"))
}

func TestOsVaultFS_ListMD_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsVaultFS()
	names, err := fs.ListMD("/nonexistent/vault/dir")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(BeEmpty())
}

func TestOsVaultFS_ListMD_NonExistError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// dir is a regular file, not a directory → ReadDir returns ENOTDIR (not IsNotExist).
	path := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsVaultFS()
	_, err := fs.ListMD(path)
	g.Expect(err).To(HaveOccurred())
}

func TestOsVaultFS_ReadFile_MissingPathError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsVaultFS()
	_, err := fs.ReadFile("/nonexistent/path.md")
	g.Expect(err).To(HaveOccurred())
}

func TestOsVaultFS_ReadFile_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "f.md")
	g.Expect(os.WriteFile(path, []byte("hello"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsVaultFS()
	data, err := fs.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello"))
}

func TestStartingPoints_EmptyVaultEmitsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stdout, stderr := executeForTest(t, []string{"engram", "starting-points", "--vault", vault})
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(BeEmpty())
}

func TestStartingPoints_FixtureVaultProducesExpectedOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	// Component A: MOC 7 + member; should emit MOC 7.
	writeNote(t, filepath.Join(vault, "MOCs"), "7.2026-05-09.zk",
		"core MOC linking to [[1.2026-05-09.member]] and [[7a.2026-05-09.dup]]")
	writeNote(t, filepath.Join(vault, "Permanent"), "1.2026-05-09.member",
		"member of zk cluster, links back to [[7.2026-05-09.zk]]")
	writeNote(t, filepath.Join(vault, "Permanent"), "7a.2026-05-09.dup",
		"another zk member, links to [[7.2026-05-09.zk]]")

	// Component B: MOC-less, clear in-degree winner (3.c).
	writeNote(t, filepath.Join(vault, "Permanent"), "2a.2026-05-10.a",
		"[[3.2026-05-10.c]]")
	writeNote(t, filepath.Join(vault, "Permanent"), "2b.2026-05-10.b",
		"[[3.2026-05-10.c]]")
	writeNote(t, filepath.Join(vault, "Permanent"), "3.2026-05-10.c",
		"leaf")

	// Component C: isolated fleeting; should emit itself.
	writeNote(t, filepath.Join(vault, "Fleeting"), "loose-thought",
		"jot")

	stdout, stderr := executeForTest(t, []string{"engram", "starting-points", "--vault", vault})
	g.Expect(stderr).To(BeEmpty())

	// Expected: globally Luhmann-sorted. 3 < 7. "loose-thought" has no Luhmann ID → sorts last.
	g.Expect(stdout).To(Equal(
		"[[3.2026-05-10.c]]\n" +
			"[[7.2026-05-09.zk]]\n" +
			"[[loose-thought]]\n",
	))
}

func TestStartingPoints_MissingSubdirsTolerated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	// Only MOCs/ exists. Permanent/ and Fleeting/ are missing.
	writeNote(t, filepath.Join(vault, "MOCs"), "1.2026-05-10.solo", "no body links")

	stdout, stderr := executeForTest(t, []string{"engram", "starting-points", "--vault", vault})
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("[[1.2026-05-10.solo]]\n"))
}

func TestStartingPoints_RequiresVaultFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeForTest(t, []string{"engram", "starting-points"})
	g.Expect(stderr).To(ContainSubstring("vault path required"))
}

func TestStartingPoints_VaultPathFromEnv(t *testing.T) {
	// No t.Parallel: this test uses t.Setenv which mutates process env.
	g := NewWithT(t)

	vault := t.TempDir()

	writeNote(t, filepath.Join(vault, "Permanent"), "5.2026-05-10.only",
		"isolated single note")

	t.Setenv("ENGRAM_VAULT_PATH", vault)

	stdout, stderr := executeForTest(t, []string{"engram", "starting-points"})
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("[[5.2026-05-10.only]]\n"))
}

// writeNote creates dir/<name>.md with the given body. Returns the basename
// without extension for convenience when building expected wikilink outputs.
func writeNote(t *testing.T, dir, name, body string) {
	t.Helper()

	const (
		dirPerm  = 0o750
		filePerm = 0o600
	)

	err := os.MkdirAll(dir, dirPerm)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	err = os.WriteFile(filepath.Join(dir, name+".md"), []byte(body), filePerm)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
}
