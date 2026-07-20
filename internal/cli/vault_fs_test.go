package cli_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestOsVaultFS_ListMD_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := cli.ExportNewOsVaultFS()
	names, err := vfs.ListMD("/nonexistent/vault/dir")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(BeEmpty())
}

func TestOsVaultFS_ListMD_NonDirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// dir is a regular file, not a directory → ReadDir returns ENOTDIR (not not-exist).
	path := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	vfs := cli.ExportNewOsVaultFS()
	_, err := vfs.ListMD(path)
	g.Expect(err).To(MatchError(ContainSubstring("list md")))
}

// TestOsVaultFS_ReadFile_MissingPathError covers the legacy osVaultFS.ReadFile
// error branch (#700 — the T12 constructor flips moved amend/resituate/vocab
// reads onto the pure vaultFS, leaving embed.go's ScanVault as the adapter's
// only production caller until T15/T7; dies with T7).
func TestOsVaultFS_ReadFile_MissingPathError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := cli.ExportNewOsVaultFS()
	_, err := vfs.ReadFile("/nonexistent/path.md")
	g.Expect(err).To(MatchError(ContainSubstring("reading")))
}

// TestOsVaultFS_RoundTrip_ListMDAndReadFile covers the legacy adapter's happy
// paths (listDirBySuffix dir/suffix filtering + ReadFile success). T15 moved
// the embed commands onto the composed newEmbedDeps route, so the legacy
// adapter's only remaining coverage is direct; dies with T7's purge.
func TestOsVaultFS_RoundTrip_ListMDAndReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "note.md"), []byte("body"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("y"), 0o600)).To(Succeed())

	vfs := cli.ExportNewOsVaultFS()

	names, err := vfs.ListMD(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(ConsistOf("note.md"))

	data, readErr := vfs.ReadFile(filepath.Join(dir, "note.md"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal("body"))
}

func TestVaultFS_ListMD_FiltersDirsAndNonMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "note.md"), []byte("x"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("y"), 0o600)).To(Succeed())

	vfs := cli.ExportNewVaultFS(realFSForTest())
	names, err := vfs.ListMD(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(ConsistOf("note.md"))
}

func TestVaultFS_ListMD_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := cli.ExportNewVaultFS(realFSForTest())
	names, err := vfs.ListMD("/nonexistent/vault/dir")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(BeEmpty())
}

func TestVaultFS_ListMD_NonExistError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// dir is a regular file, not a directory → ReadDir returns ENOTDIR (not not-exist).
	path := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	vfs := cli.ExportNewVaultFS(realFSForTest())
	_, err := vfs.ListMD(path)
	g.Expect(err).To(HaveOccurred())
}

func TestVaultFS_ListMD_WrappedNotExistReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// EdgeFS implementations wrap errors with %w; missing-dir detection must
	// survive wrapping (errors.Is unwraps; os.IsNotExist would not).
	vfs := cli.ExportNewVaultFS(wrappedNotExistEdgeFS{EdgeFS: realFSForTest()})
	names, err := vfs.ListMD("/anything")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(names).To(BeEmpty())
}

func TestVaultFS_ReadFile_MissingPathError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := cli.ExportNewVaultFS(realFSForTest())
	_, err := vfs.ReadFile("/nonexistent/path.md")
	g.Expect(err).To(HaveOccurred())
}

func TestVaultFS_ReadFile_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "f.md")
	g.Expect(os.WriteFile(path, []byte("hello"), 0o600)).To(Succeed())

	vfs := cli.ExportNewVaultFS(realFSForTest())
	data, err := vfs.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello"))
}

// wrappedNotExistEdgeFS overrides ReadDir to return a WRAPPED fs.ErrNotExist,
// proving missing-dir detection unwraps through EdgeFS error wrapping.
type wrappedNotExistEdgeFS struct{ cli.EdgeFS }

func (wrappedNotExistEdgeFS) ReadDir(string) ([]fs.DirEntry, error) {
	return nil, fmt.Errorf("listing: %w", fs.ErrNotExist)
}
