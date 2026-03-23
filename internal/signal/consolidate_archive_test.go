package signal_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/signal"
)

func TestFileArchiver_Archive_CallsRenameWithCorrectPaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var renameOld, renameNew string

	rename := func(oldpath, newpath string) error {
		renameOld = oldpath
		renameNew = newpath

		return nil
	}
	mkdir := func(_ string) error { return nil }

	archiver := signal.NewFileArchiver("/data/archive", rename, mkdir)
	err := archiver.Archive("memories/foo.toml")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(renameOld).To(Equal("memories/foo.toml"))
	g.Expect(renameNew).To(Equal("/data/archive/foo.toml"))
}

func TestFileArchiver_Archive_CreatesDirFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	rename := func(_, _ string) error {
		callOrder = append(callOrder, "rename")

		return nil
	}
	mkdir := func(_ string) error {
		callOrder = append(callOrder, "mkdir")

		return nil
	}

	archiver := signal.NewFileArchiver("/data/archive", rename, mkdir)
	err := archiver.Archive("memories/foo.toml")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(callOrder).To(Equal([]string{"mkdir", "rename"}))
}

func TestFileArchiver_Archive_DirCreateError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errMkdir := errors.New("read-only filesystem")
	renameCalled := false

	rename := func(_, _ string) error {
		renameCalled = true

		return nil
	}
	mkdir := func(_ string) error { return errMkdir }

	archiver := signal.NewFileArchiver("/data/archive", rename, mkdir)
	err := archiver.Archive("memories/foo.toml")
	g.Expect(err).To(MatchError(ContainSubstring("creating archive dir")))
	g.Expect(errors.Is(err, errMkdir)).To(BeTrue())
	g.Expect(renameCalled).To(BeFalse())
}

func TestFileArchiver_Archive_RenameError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errRename := errors.New("permission denied")
	rename := func(_, _ string) error { return errRename }
	mkdir := func(_ string) error { return nil }

	archiver := signal.NewFileArchiver("/data/archive", rename, mkdir)
	err := archiver.Archive("memories/foo.toml")
	g.Expect(err).To(MatchError(ContainSubstring("archiving foo.toml")))
	g.Expect(errors.Is(err, errRename)).To(BeTrue())
}
