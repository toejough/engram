package cli_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// This file exercises the internally-composed EdgeFS/FileLocker/debug-sink
// implementations over REAL os/syscall primitives — the relocated
// cmd/engram adapter integration suite (#700 rework). realPrimitives()
// mirrors cmd/engram/main.go's Primitives literal (doctrine flag DRIFT:
// cli_test.go's end-to-end binary tests guard the production literal).

func TestRealDebugSink_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "debug.log")

	first := debugSinkAt(path)
	g.Expect(first).NotTo(gomega.BeNil())

	if first == nil {
		return
	}

	_, err := first.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Re-open the same path: append mode must preserve the first line —
	// the tail -F contract debuglog documents.
	second := debugSinkAt(path)
	g.Expect(second).NotTo(gomega.BeNil())

	if second == nil {
		return
	}

	_, err = second.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	contents, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(contents)).To(gomega.Equal("line one\nline two\n"))
}

func TestRealDebugSink_UnopenablePathYieldsNilSink(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Parent is a regular file, so opening a child path fails -> nil sink
	// (the CLI must run without debug logging rather than fail).
	dir := t.TempDir()
	blocked := filepath.Join(dir, "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), realFSFilePerm)).To(gomega.Succeed())

	g.Expect(debugSinkAt(filepath.Join(blocked, "debug.log"))).To(gomega.BeNil())
}

func TestRealEdgeFS_MkdirAllStatReadDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	fsys := realFSForTest()

	g.Expect(fsys.MkdirAll(nested, realFSDirPerm)).To(gomega.Succeed())

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

func TestRealEdgeFS_MkdirTempAndWalkDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	tmpDir, err := fsys.MkdirTemp(dir, "pat-*")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(tmpDir)).To(gomega.HavePrefix("pat-"))
	g.Expect(fsys.WriteFile(filepath.Join(tmpDir, "leaf.txt"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())

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

func TestRealEdgeFS_ReadFileMissingSatisfiesErrNotExist(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := realFSForTest()

	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "missing.txt"))
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
}

func TestRealEdgeFS_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("hello"), realFSFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("hello"))
}

func TestRealEdgeFS_RenameRemoveRemoveAll(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	g.Expect(fsys.WriteFile(oldPath, []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.Rename(oldPath, newPath)).To(gomega.Succeed())
	g.Expect(newPath).To(gomega.BeAnExistingFile())
	g.Expect(oldPath).NotTo(gomega.BeAnExistingFile())

	g.Expect(fsys.Remove(newPath)).To(gomega.Succeed())
	g.Expect(newPath).NotTo(gomega.BeAnExistingFile())

	sub := filepath.Join(dir, "sub")
	g.Expect(fsys.MkdirAll(sub, realFSDirPerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFile(filepath.Join(sub, "f"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.RemoveAll(sub)).To(gomega.Succeed())
	g.Expect(sub).NotTo(gomega.BeADirectory())
}

// TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent proves the restored
// Chmod step (P-4) makes WriteFileAtomic's final perm exact regardless of
// the process umask — parity with the pre-#700 dance.
//
//nolint:paralleltest // serial by design: syscall.Umask is process-global; parallel file-creating tests would flake
func TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent(t *testing.T) {
	g := gomega.NewWithT(t)

	old := syscall.Umask(umaskParityRestrictiveMask)
	defer syscall.Umask(old)

	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileAtomic(target, []byte("v1"), umaskParityPerm)).To(gomega.Succeed())

	info, err := os.Stat(target)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Mode().Perm()).To(gomega.Equal(umaskParityPerm))
}

func TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("v1"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFileAtomic(path, []byte("v2"), realFSFilePerm)).To(gomega.Succeed())

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

// TestRealEdgeFS_WriteFileExclRefusesExistingFile is survivor S-1's named
// behavior-mirror test: the real WriteFileExcl primitive backs both the
// atomic-write dance's unique-temp creation (P-4) and EdgeFS.WriteFileExcl
// (X-1) — this proves the O_EXCL contract over the real primitive.
func TestRealEdgeFS_WriteFileExclRefusesExistingFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileExcl(path, []byte("first"), realFSFilePerm)).To(gomega.Succeed())

	err := fsys.WriteFileExcl(path, []byte("second"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist), "O_EXCL contract: existing path must satisfy fs.ErrExist")

	data, readErr := fsys.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("first"), "the losing writer must not clobber the existing note")
}

// TestRealFlockLocker_SecondLockWaitsForUnlock is the relocated ADR-0013
// lock regression guard: a second locker on the same path must block until
// the first unlocks — never proceed concurrently, never fail.
func TestRealFlockLocker_SecondLockWaitsForUnlock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := realDepsForTest().Lock

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

// unexported constants.
const (
	realFSDirPerm  fs.FileMode = 0o750
	realFSFilePerm fs.FileMode = 0o600
	// umaskParityPerm is the target perm for the umask-independence parity
	// test (P-4 restored chmod step).
	umaskParityPerm fs.FileMode = 0o644
	// umaskParityRestrictiveMask is a deliberately restrictive umask (would
	// mask 0o644 down to 0o600 without the explicit chmod step).
	umaskParityRestrictiveMask = 0o077
)

// debugSinkAt composes a real debug sink for path via NewDeps (Getenv fake
// points ENGRAM_DEBUG_LOG at path; the open primitive is real).
func debugSinkAt(path string) io.Writer {
	prims := realPrimitives()
	prims.Getenv = func(key string) string {
		if key == "ENGRAM_DEBUG_LOG" {
			return path
		}

		return ""
	}

	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog
}

// realDepsForTest composes production Deps over real OS primitives.
func realDepsForTest() cli.Deps {
	return cli.NewDeps(realPrimitives(), io.Discard, io.Discard, func(int) {})
}

// realFSForTest composes the production EdgeFS over real OS primitives.
func realFSForTest() cli.EdgeFS {
	return realDepsForTest().FS
}

// realPrimitives mirrors cmd/engram/main.go's production Primitives literal
// (minus the signal starter — tests must not subscribe process signals).
func realPrimitives() cli.Primitives {
	return cli.Primitives{
		ReadFile:    os.ReadFile,
		WriteFile:   os.WriteFile,
		MkdirAll:    os.MkdirAll,
		MkdirTemp:   os.MkdirTemp,
		Stat:        os.Stat,
		ReadDir:     os.ReadDir,
		Remove:      os.Remove,
		RemoveAll:   os.RemoveAll,
		Rename:      os.Rename,
		WalkDir:     filepath.WalkDir,
		Chmod:       os.Chmod,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err
			}

			_, err = file.Write(data)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			return err
		},
		OpenLockFile: func(path string, perm fs.FileMode) (uintptr, error) {
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(perm))

			return uintptr(fd), err
		},
		FlockExclusive: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_EX)
		},
		FlockUnlock: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_UN)
		},
		CloseFD: func(fd uintptr) error {
			return syscall.Close(int(fd))
		},
		OpenDebugFile: func(path string, perm fs.FileMode) (cli.WriteSyncer, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
	}
}
