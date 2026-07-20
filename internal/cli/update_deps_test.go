package cli_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewUpdateDeps_CommanderPassesThrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := stubCommander{}
	deps := cli.ExportNewUpdateDeps(cli.Deps{Commander: cmd, FS: updateFakeEdgeFS{}})

	stdout, stderr, err := deps.Cmd.Run(context.Background(), "", "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout).To(BeNil())
	g.Expect(stderr).To(BeNil())
}

func TestNewUpdateDeps_EnvDelegatesToDepsFuncs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.ExportNewUpdateDeps(cli.Deps{
		FS:          updateFakeEdgeFS{},
		Getenv:      func(key string) string { return "env:" + key },
		Getwd:       func() (string, error) { return "/cwd", nil },
		UserHomeDir: func() (string, error) { return "/home/x", nil },
	})

	g.Expect(deps.Env.Getenv("K")).To(Equal("env:K"))

	cwd, cwdErr := deps.Env.Getwd()
	g.Expect(cwdErr).NotTo(HaveOccurred())
	g.Expect(cwd).To(Equal("/cwd"))

	home, homeErr := deps.Env.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())
	g.Expect(home).To(Equal("/home/x"))
}

func TestNewUpdateDeps_FSAdapterPassesWriteSideThrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: updateFakeEdgeFS{}})

	// The fake's write side always fails with errUnsupported; the adapter
	// must pass the error through unwrapped chains intact.
	g.Expect(deps.FS.MkdirAll("skills", 0o755)).To(MatchError(errUnsupported))
	g.Expect(deps.FS.RemoveAll("skills")).To(MatchError(errUnsupported))
	g.Expect(deps.FS.WriteFile("skills/x.md", []byte("x"), 0o644)).To(MatchError(errUnsupported))
}

func TestNewUpdateDeps_FSAdapterPreservesNotExist(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: updateFakeEdgeFS{}})

	_, readErr := deps.FS.ReadFile("/missing")
	g.Expect(errors.Is(readErr, fs.ErrNotExist)).To(BeTrue())

	_, dirErr := deps.FS.ReadDir("/missing")
	g.Expect(errors.Is(dirErr, fs.ErrNotExist)).To(BeTrue())

	_, statErr := deps.FS.Stat("/missing")
	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
}

func TestNewUpdateDeps_FSAdapterReadsThroughEdgeFS(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: updateFakeEdgeFS{
		"skills/learn/SKILL.md": &fstest.MapFile{Data: []byte("learn")},
	}})

	data, readErr := deps.FS.ReadFile("skills/learn/SKILL.md")
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal("learn"))

	entries, dirErr := deps.FS.ReadDir("skills")
	g.Expect(dirErr).NotTo(HaveOccurred())

	if dirErr != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Name()).To(Equal("learn"))
	g.Expect(entries[0].IsDir()).To(BeTrue())

	info, statErr := deps.FS.Stat("skills/learn/SKILL.md")
	g.Expect(statErr).NotTo(HaveOccurred())

	if statErr != nil || info == nil {
		return
	}

	g.Expect(info.IsDir()).To(BeFalse())
}

// unexported variables.
var (
	errUnsupported = errors.New("updateFakeEdgeFS: write path not supported")
)

// updateFakeEdgeFS is a read-only in-memory cli.EdgeFS over fstest.MapFS.
// Write-side methods return errUnsupported: the update dry-run/read paths
// under test never invoke them.
type updateFakeEdgeFS fstest.MapFS

func (m updateFakeEdgeFS) MkdirAll(string, fs.FileMode) error { return errUnsupported }

func (m updateFakeEdgeFS) MkdirTemp(string, string) (string, error) { return "", errUnsupported }

func (m updateFakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return fs.ReadDir(fstest.MapFS(m), path) // fake passes chains through
}

func (m updateFakeEdgeFS) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(fstest.MapFS(m), path) // fake passes chains through
}

func (m updateFakeEdgeFS) Remove(string) error { return errUnsupported }

func (m updateFakeEdgeFS) RemoveAll(string) error { return errUnsupported }

func (m updateFakeEdgeFS) Rename(string, string) error { return errUnsupported }

func (m updateFakeEdgeFS) Stat(path string) (fs.FileInfo, error) {
	return fs.Stat(fstest.MapFS(m), path) // fake passes chains through
}

func (m updateFakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(fstest.MapFS(m), root, fn) // fake passes chains through
}

func (m updateFakeEdgeFS) WriteFile(string, []byte, fs.FileMode) error { return errUnsupported }

func (m updateFakeEdgeFS) WriteFileAtomic(string, []byte, fs.FileMode) error { return errUnsupported }

func (m updateFakeEdgeFS) WriteFileExcl(string, []byte, fs.FileMode) error { return errUnsupported }
