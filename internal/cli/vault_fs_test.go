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
