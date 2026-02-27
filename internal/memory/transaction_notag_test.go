package memory

import (
	"errors"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

// TestCopyDir_MkdirAllError verifies copyDir returns error when MkdirAll fails.
func TestCopyDir_MkdirAllError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	src := t.TempDir()
	ops := &fakeDirOps{
		mkdirAllFn: func(string, os.FileMode) error {
			return errors.New("no space left on device")
		},
	}

	err := copyDir(ops, src, t.TempDir()+"/dst")

	g.Expect(err).To(MatchError(ContainSubstring("no space left on device")))
}

// TestCopyDir_ReadDirError verifies copyDir returns error when ReadDir fails.
func TestCopyDir_ReadDirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	src := t.TempDir()
	ops := &fakeDirOps{
		readDirFn: func(string) ([]os.DirEntry, error) {
			return nil, errors.New("permission denied")
		},
	}

	err := copyDir(ops, src, t.TempDir()+"/dst")

	g.Expect(err).To(MatchError(ContainSubstring("permission denied")))
}

// TestCopyDir_ReadlinkError verifies copyDir returns error when Readlink fails for a symlink entry.
func TestCopyDir_ReadlinkError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a real directory with a symlink so ReadDir returns a symlink entry.
	src := t.TempDir()
	target := t.TempDir() + "/target.txt"

	if err := os.WriteFile(target, []byte("t"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, src+"/link.txt"); err != nil {
		t.Fatal(err)
	}

	ops := &fakeDirOps{
		readlinkFn: func(string) (string, error) {
			return "", errors.New("readlink failed")
		},
	}

	err := copyDir(ops, src, t.TempDir()+"/dst")

	g.Expect(err).To(MatchError(ContainSubstring("readlink failed")))
}

// TestCopyDir_SymlinkError verifies copyDir returns error when Symlink fails.
func TestCopyDir_SymlinkError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a real directory with a symlink.
	src := t.TempDir()
	target := t.TempDir() + "/target.txt"

	if err := os.WriteFile(target, []byte("t"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, src+"/link.txt"); err != nil {
		t.Fatal(err)
	}

	ops := &fakeDirOps{
		symlinkFn: func(string, string) error {
			return errors.New("symlink already exists")
		},
	}

	err := copyDir(ops, src, t.TempDir()+"/dst")

	g.Expect(err).To(MatchError(ContainSubstring("symlink already exists")))
}

// fakeDirOps is a test double for dirOps that lets individual methods be overridden.
type fakeDirOps struct {
	statFn     func(string) (os.FileInfo, error)
	mkdirAllFn func(string, os.FileMode) error
	readDirFn  func(string) ([]os.DirEntry, error)
	readlinkFn func(string) (string, error)
	symlinkFn  func(string, string) error
}

func (f *fakeDirOps) MkdirAll(path string, perm os.FileMode) error {
	if f.mkdirAllFn != nil {
		return f.mkdirAllFn(path, perm)
	}

	return os.MkdirAll(path, perm)
}

func (f *fakeDirOps) ReadDir(name string) ([]os.DirEntry, error) {
	if f.readDirFn != nil {
		return f.readDirFn(name)
	}

	return os.ReadDir(name)
}

func (f *fakeDirOps) Readlink(name string) (string, error) {
	if f.readlinkFn != nil {
		return f.readlinkFn(name)
	}

	return os.Readlink(name)
}

func (f *fakeDirOps) Stat(name string) (os.FileInfo, error) {
	if f.statFn != nil {
		return f.statFn(name)
	}

	return os.Stat(name)
}

func (f *fakeDirOps) Symlink(oldname, newname string) error {
	if f.symlinkFn != nil {
		return f.symlinkFn(oldname, newname)
	}

	return os.Symlink(oldname, newname)
}
