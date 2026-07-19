package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewLearnDeps_InitVault_IdempotentAndPreservesEdits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	g.Expect(deps.InitVault(vault)).To(Succeed())

	readme := filepath.Join(vault, "README.md")
	g.Expect(os.WriteFile(readme, []byte("user edit"), 0o644)).To(Succeed())

	g.Expect(deps.InitVault(vault)).To(Succeed())

	got, readErr := os.ReadFile(readme)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("user edit"), "re-init must not clobber existing files")
}

func TestNewLearnDeps_InitVault_MkdirFailureSurfaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Parent is a regular file, so the vault MkdirAll fails with ENOTDIR.
	blocked := filepath.Join(t.TempDir(), "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	g.Expect(deps.InitVault(filepath.Join(blocked, "vault"))).
		To(MatchError(ContainSubstring("mkdir")))
}

func TestNewLearnDeps_ListBasenames_SkipsSubdirsAndNonLuhmann(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-09.foo.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "README.md"), nil, 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	got, err := deps.ListBasenames(vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(ConsistOf("1.2026-05-09.foo"))
}

func TestNewLearnDeps_ListIDs_MissingVaultIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	got, err := deps.ListIDs(filepath.Join(t.TempDir(), "absent"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(BeEmpty())
}

func TestNewLearnDeps_Lock_AcquiresVaultLuhmannLockFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	release, err := deps.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	release()

	_, statErr := os.Stat(filepath.Join(vault, ".luhmann.lock"))
	g.Expect(statErr).NotTo(HaveOccurred(), "lock must live at vault/.luhmann.lock")
}

func TestNewLearnDeps_LogWarning_WritesToDepsStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	d := realFSDepsForTest()
	d.Stderr = &stderr

	deps := cli.ExportNewLearnDeps(d)
	deps.LogWarning("hello %s", "world")

	g.Expect(stderr.String()).To(Equal("warning: hello world\n"))
}

func TestNewLearnDeps_StatDir_FileIsNotADirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	g.Expect(deps.StatDir(path)).To(MatchError(ContainSubstring("not a directory")))
}

func TestNewLearnDeps_StatDir_MissingReturnsErrNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	err := deps.StatDir(filepath.Join(t.TempDir(), "absent"))
	g.Expect(errors.Is(err, fs.ErrNotExist)).To(BeTrue())
}

func TestNewLearnDeps_WriteNew_PreservesErrExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "existing.md")
	g.Expect(os.WriteFile(path, []byte("already"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	err := deps.WriteNew(path, []byte("nope"))
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(), "O_EXCL backstop must survive composition")

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("already"))
}

func TestNewLearnDeps_WriteSidecar_WritesAtomicallyAndWrapsErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.vec.json")

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	g.Expect(deps.WriteSidecar(path, []byte(`{"v":1}`))).To(Succeed())

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal(`{"v":1}`))

	// A sidecar write into a missing directory surfaces a wrapped error.
	err := deps.WriteSidecar(filepath.Join(dir, "missing", "note.vec.json"), []byte("x"))
	g.Expect(err).To(MatchError(ContainSubstring("write sidecar")))
}

func TestNewQaDeps_ListMD_WrapsNonMissingErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// dir is a regular file: ReadDir fails with ENOTDIR (not fs.ErrNotExist),
	// so the missing-dir→empty contract must NOT swallow it.
	filePath := filepath.Join(t.TempDir(), "isfile")
	g.Expect(os.WriteFile(filePath, []byte("x"), 0o600)).To(Succeed())

	deps := cli.ExportNewQaDeps(realFSDepsForTest())
	_, err := deps.ListMD(filePath)
	g.Expect(err).To(MatchError(ContainSubstring("reading dir")))
}

func TestNewQaDeps_WiresRemoveAndReadThroughFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	g.Expect(os.WriteFile(path, []byte("data"), 0o600)).To(Succeed())

	deps := cli.ExportNewQaDeps(realFSDepsForTest())

	got, readErr := deps.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("data"))

	g.Expect(deps.RemoveFile(path)).To(Succeed())
	_, statErr := os.Stat(path)
	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
}
