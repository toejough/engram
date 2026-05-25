package cli_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestInitializeVault_CreatesDirsAndStarterFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeFS := newFakeInitFS()
	g.Expect(cli.ExportInitializeVault(fakeFS, "/v")).To(Succeed())

	sort.Strings(fakeFS.mkdirs)
	g.Expect(fakeFS.mkdirs).To(Equal([]string{
		"/v/.obsidian",
		"/v/Permanent",
	}))

	written := make([]string, 0, len(fakeFS.wrote))
	for k := range fakeFS.wrote {
		written = append(written, k)
	}

	sort.Strings(written)
	g.Expect(written).To(Equal([]string{
		"/v/.gitignore",
		"/v/.obsidian/app.json",
		"/v/README.md",
	}))
	g.Expect(string(fakeFS.wrote[filepath.Join("/v", ".obsidian", "app.json")])).To(Equal("{}\n"))
}

func TestInitializeVault_Idempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeFS := newFakeInitFS()
	g.Expect(cli.ExportInitializeVault(fakeFS, "/v")).To(Succeed())

	priorCount := len(fakeFS.wrote)
	g.Expect(cli.ExportInitializeVault(fakeFS, "/v")).To(Succeed())
	g.Expect(fakeFS.wrote).To(HaveLen(priorCount))
}

func TestInitializeVault_PropagatesMkdirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportInitializeVault(&erroringInitFS{mkdirErr: errInjected}, "/v")
	g.Expect(err).To(MatchError(ContainSubstring("mkdir")))
}

func TestInitializeVault_PropagatesWriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportInitializeVault(&erroringInitFS{writeErr: errInjected}, "/v")
	g.Expect(err).To(MatchError(ContainSubstring("write")))
}

func TestResolveVault_EnvFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(name string) string {
		if name == "ENGRAM_VAULT_PATH" {
			return "/from/env"
		}

		return ""
	}
	got := cli.ExportResolveVault("", "/home/u", getenv)
	g.Expect(got).To(Equal("/from/env"))
}

func TestResolveVault_FlagWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportResolveVault("/explicit", "/home/u", func(string) string { return "/env" })
	g.Expect(got).To(Equal("/explicit"))
}

func TestResolveVault_HomeFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportResolveVault("", "/home/u", func(string) string { return "" })
	g.Expect(got).To(Equal("/home/u/.local/share/engram/vault"))
}

func TestResolveVault_XDGDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(name string) string {
		if name == "XDG_DATA_HOME" {
			return "/xdg/data"
		}

		return ""
	}
	got := cli.ExportResolveVault("", "/home/u", getenv)
	g.Expect(got).To(Equal("/xdg/data/engram/vault"))
}

// unexported variables.
var (
	errInjected = errors.New("injected")
)

type erroringInitFS struct {
	mkdirErr error
	writeErr error
}

func (f *erroringInitFS) MkdirAll(string, fs.FileMode) error {
	return f.mkdirErr
}

func (f *erroringInitFS) WriteFileIfMissing(string, []byte, fs.FileMode) error {
	return f.writeErr
}

// fakeInitFS records calls; WriteFileIfMissing is idempotent — second writes
// to the same path are swallowed.
type fakeInitFS struct {
	mkdirs []string
	wrote  map[string][]byte
}

func (f *fakeInitFS) MkdirAll(path string, _ fs.FileMode) error {
	f.mkdirs = append(f.mkdirs, path)

	return nil
}

func (f *fakeInitFS) WriteFileIfMissing(path string, data []byte, _ fs.FileMode) error {
	if _, exists := f.wrote[path]; exists {
		return nil
	}

	f.wrote[path] = data

	return nil
}

func newFakeInitFS() *fakeInitFS {
	return &fakeInitFS{
		wrote: map[string][]byte{},
	}
}
