package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	. "github.com/onsi/gomega"
)

func TestListJSONLIndexes_FiltersAndTreatsMissingDirAsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "s.jsonl"},
			fakeDirEntry{name: "manifest.json"},
			fakeDirEntry{name: "nested", dir: true},
		}, nil
	}})

	paths, err := lister("/chunks")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(Equal([]string{"/chunks/s.jsonl"}))

	missing := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	paths, err = missing("/gone")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(BeEmpty())
}

func TestLogWarningTo_FormatsWithWarningPrefixAndNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	logWarningTo(&buf)("amend: %s failed after %d tries", "embed", 2)
	g.Expect(buf.String()).To(Equal("warning: amend: embed failed after 2 tries\n"))
}

func TestNewVaultFS_ListMD_FiltersToMDFilesSkippingDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "1.2026-01-01.note.md"},
			fakeDirEntry{name: "sidecar.vec.json"},
			fakeDirEntry{name: "subdir", dir: true},
		}, nil
	}})

	names, err := vfs.ListMD("/vault")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(Equal([]string{"1.2026-01-01.note.md"}))
}

func TestNewVaultFS_ListMD_MissingDirIsEmptyNotError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	names, err := vfs.ListMD("/missing")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(BeEmpty())
}

func TestNewVaultFS_ReadFile_WrapsWithDistinctVerb(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readFile: func(string) ([]byte, error) {
		return nil, errInjectedCompose
	}})

	_, err := vfs.ReadFile("/vault/x.md")
	g.Expect(err).To(MatchError(errInjectedCompose))
	g.Expect(err).To(MatchError(ContainSubstring("vault read")))
}

func TestVaultLockFromLocker_LocksVaultLuhmannLockFileAndReleases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var lockedPath string

	unlocked := false
	locker := fakeLocker{lock: func(path string) (func() error, error) {
		lockedPath = path

		return func() error { unlocked = true; return nil }, nil
	}}

	lock := vaultLockFromLocker(locker)

	release, err := lock("/vault")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockedPath).To(Equal("/vault/.luhmann.lock"))

	release()
	g.Expect(unlocked).To(BeTrue())
}

// unexported variables.
var (
	errInjectedCompose = errors.New("injected compose failure")
)

// fakeDirEntry is a minimal fs.DirEntry for ListMD/lister tests.
type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func (f fakeDirEntry) IsDir() bool { return f.dir }

func (f fakeDirEntry) Name() string { return f.name }

func (f fakeDirEntry) Type() fs.FileMode { return 0 }

// fakeEdgeFS overrides only the EdgeFS methods a test exercises; calling an
// un-overridden method panics via the embedded nil interface (test bug, loud).
type fakeEdgeFS struct {
	EdgeFS

	readDir  func(string) ([]fs.DirEntry, error)
	readFile func(string) ([]byte, error)
}

func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

// fakeLocker records the locked path.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (f fakeLocker) Lock(path string) (func() error, error) { return f.lock(path) }
