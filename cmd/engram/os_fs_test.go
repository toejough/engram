package main

import (
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

// TestFlockLocker_SecondLockWaitsForUnlock is the cmd-side ADR-0013 lock
// regression guard: a second locker on the same path must block until the
// first unlocks — never proceed concurrently, never fail.
func TestFlockLocker_SecondLockWaitsForUnlock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := flockLocker{}

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	acquired := make(chan struct{})

	go func() {
		secondUnlock, secondErr := locker.Lock(lockPath)
		if secondErr == nil {
			_ = secondUnlock()
		}

		close(acquired)
	}()

	const holdWindow = 100 * time.Millisecond

	select {
	case <-acquired:
		t.Fatal("second locker acquired while first still held the lock")
	case <-time.After(holdWindow):
		// good — second locker is blocked while the lock is held
	}

	g.Expect(unlock()).To(gomega.Succeed())

	const releaseTimeout = 2 * time.Second

	select {
	case <-acquired:
		// good — released lock was acquired
	case <-time.After(releaseTimeout):
		t.Fatal("second locker did not acquire after unlock")
	}
}

func TestOsFS_MkdirAllStatReadDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	fsys := osFS{}

	g.Expect(fsys.MkdirAll(nested, testDirPerm)).To(gomega.Succeed())

	info, err := fsys.Stat(nested)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.IsDir()).To(gomega.BeTrue())

	entries, readErr := fsys.ReadDir(filepath.Join(dir, "a"))
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Name()).To(gomega.Equal("b"))
}

func TestOsFS_MkdirTempAndWalkDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := osFS{}

	tmpDir, err := fsys.MkdirTemp(dir, "pat-*")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(tmpDir)).To(gomega.HavePrefix("pat-"))
	g.Expect(fsys.WriteFile(filepath.Join(tmpDir, "leaf.txt"), []byte("x"), testFilePerm)).To(gomega.Succeed())

	visited := make([]string, 0, 3)
	walkErr := fsys.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		visited = append(visited, path)

		return nil
	})
	g.Expect(walkErr).NotTo(gomega.HaveOccurred())
	g.Expect(visited).To(gomega.ContainElement(filepath.Join(tmpDir, "leaf.txt")))
}

func TestOsFS_ReadFileMissingSatisfiesErrNotExist(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := osFS{}

	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "missing.txt"))
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
}

func TestOsFS_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	fsys := osFS{}

	g.Expect(fsys.WriteFile(path, []byte("hello"), testFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("hello"))
}

func TestOsFS_RenameRemoveRemoveAll(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := osFS{}

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	g.Expect(fsys.WriteFile(oldPath, []byte("x"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.Rename(oldPath, newPath)).To(gomega.Succeed())
	g.Expect(newPath).To(gomega.BeAnExistingFile())
	g.Expect(oldPath).NotTo(gomega.BeAnExistingFile())

	g.Expect(fsys.Remove(newPath)).To(gomega.Succeed())
	g.Expect(newPath).NotTo(gomega.BeAnExistingFile())

	sub := filepath.Join(dir, "sub")
	g.Expect(fsys.MkdirAll(sub, testDirPerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFile(filepath.Join(sub, "f"), []byte("x"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.RemoveAll(sub)).To(gomega.Succeed())
	g.Expect(sub).NotTo(gomega.BeADirectory())
}

func TestOsFS_WriteFileAtomicReplacesContentAndCleansTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	fsys := osFS{}

	g.Expect(fsys.WriteFile(path, []byte("v1"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFileAtomic(path, []byte("v2"), testFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("v2"))

	entries, readErr := fsys.ReadDir(dir)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1), "temp files must be renamed or removed")
}

// unexported constants.
const (
	testDirPerm  fs.FileMode = 0o750
	testFilePerm fs.FileMode = 0o600
)
