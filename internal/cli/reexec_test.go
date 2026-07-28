package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func TestRunUpdate_ReexecFallback_ProceedsAsToday(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errSpawn := errors.New("exec: no such file")
	spawner := &fakeReexecSpawner{err: errSpawn}

	var exitCalls int

	deps := cli.ExportNewUpdateDepsWithSpawn(
		singleClaudeHarnessFS(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"}, spawner,
		func(int) { exitCalls++ },
	)

	stdout := &bytes.Buffer{}
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(exitCalls).To(Equal(0), "deps.Exit must not be called on the in-process fallback path")

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("re-exec failed, completed with pre-update logic:"))
	g.Expect(out).To(ContainSubstring(errSpawn.Error()))
	g.Expect(out).To(ContainSubstring("installed:"), "fallback completes the full in-process run and reports it")
}

func TestRunUpdate_ReexecHandoff_ExitsBeforeVaultChecks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	spawner := &fakeReexecSpawner{exitCode: 9}

	var gotExitCode int

	var exitCalls int

	deps := cli.ExportNewUpdateDepsWithSpawn(
		singleClaudeHarnessFS(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"}, spawner,
		func(code int) { gotExitCode, exitCalls = code, exitCalls+1 },
	)

	stdout := &bytes.Buffer{}
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(spawner.calls).To(Equal(1))
	g.Expect(exitCalls).To(Equal(1), "deps.Exit must be called exactly once on a re-exec handoff")
	g.Expect(gotExitCode).To(Equal(9), "the child's exit code propagates verbatim")

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("source: local clone at "))
	g.Expect(out).NotTo(ContainSubstring("installed:"), "no harness/vault output — the child owns that report")

	// "engram update" (the header line) must appear exactly once in the
	// combined output: the parent's install-result header, never repeated.
	g.Expect(strings.Count(out, "engram update\n")).To(Equal(1))
}

func TestRunUpdate_ReexecHandoff_WritesHeaderBeforeSpawning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stdout := &bytes.Buffer{}

	var stdoutLenAtSpawn int

	spawner := &fakeReexecSpawner{exitCode: 0, hook: func() { stdoutLenAtSpawn = stdout.Len() }}

	deps := cli.ExportNewUpdateDepsWithSpawn(
		singleClaudeHarnessFS(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"}, spawner,
		func(int) {},
	)

	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdoutLenAtSpawn).To(BeNumerically(">", 0),
		"the parent's header must already be on stdout by the time the child is spawned — "+
			"a re-execed child inherits this same stdout and would otherwise interleave or precede it")
	g.Expect(stdout.String()[:stdoutLenAtSpawn]).To(ContainSubstring("source: local clone at "))
}

// TestRunUpdate_SentinelRun_NeverClaimsAnInstall drives runUpdate directly
// with the loop-guard sentinel already set in Env — the shape a re-execed
// child actually runs in (Updater.Run reads it via Env.Getenv, same seam a
// real child process's environment would carry). The child must never print
// the header it never earned: no install ran, so no "binary: go install ...
// ok" claim, and no duplicated "engram update" line (the parent already
// printed its own header before spawning this "child").
func TestRunUpdate_SentinelRun_NeverClaimsAnInstall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.ExportNewUpdateDepsWithSpawn(
		singleClaudeHarnessFS(), stubCommander{}, sentinelEnv{home: reexecHome, cwd: "/repo"},
		&fakeReexecSpawner{}, func(int) {},
	)

	stdout := &bytes.Buffer{}
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())

	out := stdout.String()
	g.Expect(out).NotTo(ContainSubstring("engram update\n"), "the sentinel run's report is not the header report")
	g.Expect(out).NotTo(ContainSubstring("go install"), "sentinel run performed no install and must not claim one")
	g.Expect(out).To(ContainSubstring("installed:"), "sync/checks still complete and report normally")
}

// unexported constants.
const (
	reexecHome = "/home/joe"
)

// fakeReexecSpawner is a hand-written test double for update.Spawner
// (mirrors internal/cli/spawner_test.go's inline fakes): it records the
// last call and returns a canned (exitCode, err) pair.
type fakeReexecSpawner struct {
	exitCode int
	err      error
	calls    int
	// hook runs before the call resolves — lets a test observe ordering
	// against stdout writes (defect: header must land before this runs).
	hook func()
}

func (f *fakeReexecSpawner) Run(string, []string, []string) (int, error) {
	if f.hook != nil {
		f.hook()
	}

	f.calls++

	return f.exitCode, f.err
}

// fakeUpdateFS is a minimal in-memory update.Filesystem — deliberately NOT
// the real disk (liveUpdateFS), since the fallback-continues path below
// completes a full non-dry-run apply, and running that against a real home
// directory would actually write to ~/.claude.
type fakeUpdateFS struct {
	dirs     map[string]bool
	files    map[string][]byte
	symlinks map[string]string
}

func (f *fakeUpdateFS) Lstat(path string) (update.FileInfo, error) { return f.Stat(path) }

func (f *fakeUpdateFS) MkdirAll(path string, _ fs.FileMode) error {
	f.dirs[path] = true

	return nil
}

func (f *fakeUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
	prefix := path + "/"
	seen := map[string]bool{}

	entries := []update.DirEntry{}

	for name := range f.files {
		addFakeDirEntry(&entries, seen, prefix, name, false)
	}

	for name := range f.dirs {
		addFakeDirEntry(&entries, seen, prefix, name, true)
	}

	return entries, nil
}

func (f *fakeUpdateFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return data, nil
}

func (f *fakeUpdateFS) ReadLink(path string) (string, error) {
	target, ok := f.symlinks[path]
	if !ok {
		return "", fs.ErrNotExist
	}

	return target, nil
}

func (f *fakeUpdateFS) RemoveAll(path string) error {
	delete(f.dirs, path)

	for name := range f.files {
		if name == path || strings.HasPrefix(name, path+"/") {
			delete(f.files, name)
		}
	}

	return nil
}

func (f *fakeUpdateFS) Stat(path string) (update.FileInfo, error) {
	if f.dirs[path] {
		return fakeFileInfo{dir: true}, nil
	}

	if _, ok := f.files[path]; ok {
		return fakeFileInfo{}, nil
	}

	return nil, fs.ErrNotExist
}

func (f *fakeUpdateFS) Symlink(target, link string) error {
	f.symlinks[link] = target

	return nil
}

func (f *fakeUpdateFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	f.files[path] = data

	return nil
}

// reexecEnv is a minimal update.Env fixture for a local-mode run against
// fakeUpdateFS: fixed home/cwd, "" for every Getenv key (never set the
// re-exec sentinel — these tests exercise the PARENT, which never sees it).
type reexecEnv struct{ home, cwd string }

func (e reexecEnv) Getenv(string) string { return "" }

func (e reexecEnv) Getwd() (string, error) { return e.cwd, nil }

func (e reexecEnv) UserHomeDir() (string, error) { return e.home, nil }

// sentinelEnv is reexecEnv plus ENGRAM_UPDATE_REEXEC=1 on every Getenv call
// for that one key — the environment shape a re-execed child actually runs
// in.
type sentinelEnv struct{ home, cwd string }

func (e sentinelEnv) Getenv(key string) string {
	if key == "ENGRAM_UPDATE_REEXEC" {
		return "1"
	}

	return ""
}

func (e sentinelEnv) Getwd() (string, error) { return e.cwd, nil }

func (e sentinelEnv) UserHomeDir() (string, error) { return e.home, nil }

func addFakeDirEntry(entries *[]update.DirEntry, seen map[string]bool, prefix, path string, dirEntry bool) {
	if !strings.HasPrefix(path, prefix) {
		return
	}

	rest := strings.TrimPrefix(path, prefix)
	name, _, hasChild := strings.Cut(rest, "/")

	if seen[name] {
		return
	}

	seen[name] = true
	*entries = append(*entries, fakeDirEntry{name: name, dir: dirEntry || hasChild})
}

func newFakeUpdateFS() *fakeUpdateFS {
	return &fakeUpdateFS{dirs: map[string]bool{}, files: map[string][]byte{}, symlinks: map[string]string{}}
}

// singleClaudeHarnessFS builds the minimal local-mode fixture (go.mod at
// cwd, one Claude harness, no guidance) reexecFixture builds for
// internal/update's own tests, sized for driving runUpdate end-to-end.
func singleClaudeHarnessFS() *fakeUpdateFS {
	fileSystem := newFakeUpdateFS()
	fileSystem.dirs[reexecHome+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true

	return fileSystem
}
