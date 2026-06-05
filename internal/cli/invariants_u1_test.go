package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// TestInvariant_U1_MissingGoFailsLoudly locks the half of U1's missing-go
// failure surface that DOES hold: a missing `go` binary must make update
// FAIL (never silently succeed). It asserts the true, un-weakened sub-claim
// — that the failure propagates — and does not assert the spec's sentinel
// identity, which is unmet.
//
// REPORTED GAP (real bug, not forced): U1 requires the sentinel to be
// update.ErrGoNotFound. That sentinel is declared (update.go:35) but NEVER
// returned by production — resolveSource wraps the raw command failure as
// "go install (local): %w". So errors.Is(err, ErrGoNotFound) is false today.
// This test deliberately does NOT pin that wrong behavior as correct
// (asserting Is==false would be weakening); it asserts only the failure
// surface that holds. The sentinel-typing clause is flagged in the report.
func TestInvariant_U1_MissingGoFailsLoudly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS := newU1FS()
	memFS.seedLocalRepo()

	// Commander fails as if `go` were not on PATH.
	updater := &update.Updater{
		FS:  memFS,
		Cmd: &u1FailCmd{err: errors.New(`exec: "go": executable file not found in $PATH`)},
		Env: u1LocalEnv(),
	}

	_, runErr := updater.Run(context.Background(), update.Options{})
	g.Expect(runErr).To(HaveOccurred(), "U1: a missing go binary must not silently succeed")
	g.Expect(runErr).To(MatchError(ContainSubstring("go install")))

	// And the loud failure survives the CLI tail (finishUpdate wraps it).
	var out bytes.Buffer

	g.Expect(cli.ExportFinishUpdate(&out, update.Report{}, runErr)).To(HaveOccurred())
}

// TestInvariant_U1_NoHarnessSentinel locks U1's no-harness failure surface:
// with no supported harness dir present, update fails with ErrNoHarness, and
// the sentinel survives the CLI's finishUpdate wrapping (errors.Is /
// MatchError).
func TestInvariant_U1_NoHarnessSentinel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS := newU1FS()
	memFS.seedLocalRepo()
	// Remove both harness probe dirs so detectHarnesses finds none.
	delete(memFS.dirs, filepath.Join(u1Home, ".claude"))
	delete(memFS.dirs, filepath.Join(u1Home, ".config", "opencode"))

	updater := &update.Updater{FS: memFS, Cmd: &u1OKCmd{}, Env: u1LocalEnv()}

	_, runErr := updater.Run(context.Background(), update.Options{})
	g.Expect(runErr).To(MatchError(update.ErrNoHarness))

	// The CLI tail wraps the run error but must preserve the sentinel chain.
	var out bytes.Buffer

	finishErr := cli.ExportFinishUpdate(&out, update.Report{}, runErr)
	g.Expect(finishErr).To(MatchError(update.ErrNoHarness))
}

// TestInvariant_U1_SkillsSrcMissingSentinel locks U1's missing-skills failure
// surface: a harness is present but the skills source dir is absent → update
// fails with ErrSkillsSrcMissing, preserved through finishUpdate.
func TestInvariant_U1_SkillsSrcMissingSentinel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS := newU1FS()
	memFS.seedLocalRepo()
	// Drop the entire skills source subtree (files AND dir entries) so
	// planSkillCopies' walk hits the missing-root case and returns the sentinel.
	memFS.removeSubtree(filepath.Join(u1RepoRoot, "skills"))

	updater := &update.Updater{FS: memFS, Cmd: &u1OKCmd{}, Env: u1LocalEnv()}

	_, runErr := updater.Run(context.Background(), update.Options{})
	g.Expect(runErr).To(MatchError(update.ErrSkillsSrcMissing))

	var out bytes.Buffer

	finishErr := cli.ExportFinishUpdate(&out, update.Report{}, runErr)
	g.Expect(finishErr).To(MatchError(update.ErrSkillsSrcMissing))
}

// TestInvariant_U1_UpdateIdempotent locks the idempotence half of invariant
// U1: re-running `engram update` with identical source is a copy-equivalent
// no-op — the second run changes nothing on disk. We drive update.Updater
// (the DI'd core behind the CLI's runUpdate) with in-memory fakes over the
// exported Filesystem/Commander/Env interfaces, route each run's outcome
// through the CLI's finishUpdate decision tail, and assert the destination
// file set is byte-identical after the second run.
func TestInvariant_U1_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS := newU1FS()
	memFS.seedLocalRepo()

	updater := &update.Updater{FS: memFS, Cmd: &u1OKCmd{}, Env: u1LocalEnv()}

	report1, err1 := updater.Run(context.Background(), update.Options{})
	g.Expect(err1).NotTo(HaveOccurred())

	if err1 != nil {
		return
	}

	var out1 bytes.Buffer

	g.Expect(cli.ExportFinishUpdate(&out1, report1, err1)).To(Succeed())

	afterFirst := memFS.destSnapshot()

	// Guard against vacuity: the first run must actually have installed the
	// skill files, else "no-op" is trivially true over an empty dest set.
	g.Expect(afterFirst).NotTo(BeEmpty(), "U1: first update installed nothing")
	g.Expect(afterFirst).To(HaveKey(
		filepath.Join(u1Home, ".claude", "skills", "learn", "SKILL.md")),
		"U1: first update did not install the learn skill")

	report2, err2 := updater.Run(context.Background(), update.Options{})
	g.Expect(err2).NotTo(HaveOccurred())

	if err2 != nil {
		return
	}

	var out2 bytes.Buffer

	g.Expect(cli.ExportFinishUpdate(&out2, report2, err2)).To(Succeed())

	afterSecond := memFS.destSnapshot()

	g.Expect(afterSecond).To(Equal(afterFirst),
		"U1: second update run must be a copy-equivalent no-op")
}

// unexported constants.
const (
	u1Home     = "/home/tester"
	u1RepoRoot = "/repo"
)

type u1DirEntry struct {
	name  string
	isDir bool
}

func (e u1DirEntry) IsDir() bool { return e.isDir }

func (e u1DirEntry) Name() string { return e.name }

// u1Env is an Env fake returning a fixed home and cwd.
type u1Env struct {
	home string
	cwd  string
}

func (e *u1Env) Getenv(string) string { return "" }

func (e *u1Env) Getwd() (string, error) { return e.cwd, nil }

func (e *u1Env) UserHomeDir() (string, error) { return e.home, nil }

// u1FS is a map-backed update.Filesystem. files holds regular-file contents;
// dirs records explicitly-existing directories. ReadDir synthesizes children
// from both maps.
type u1FS struct {
	files map[string][]byte
	dirs  map[string]bool
}

func (m *u1FS) MkdirAll(path string, _ fs.FileMode) error {
	m.dirs[filepath.Clean(path)] = true

	return nil
}

func (m *u1FS) ReadDir(path string) ([]update.DirEntry, error) {
	clean := filepath.Clean(path)

	if !m.dirs[clean] && !m.hasChildren(clean) {
		return nil, fs.ErrNotExist
	}

	names := map[string]bool{}
	children := map[string]bool{} // name -> isDir

	collect := func(p string) {
		rel, ok := childName(clean, p)
		if !ok {
			return
		}

		names[rel] = true

		if strings.Contains(strings.TrimPrefix(p, clean+string(filepath.Separator)),
			string(filepath.Separator)) {
			children[rel] = true
		}
	}

	for p := range m.files {
		collect(p)
	}

	for p := range m.dirs {
		collect(p)
	}

	out := make([]update.DirEntry, 0, len(names))

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}

	sort.Strings(ordered)

	for _, name := range ordered {
		out = append(out, u1DirEntry{name: name, isDir: children[name]})
	}

	return out, nil
}

func (m *u1FS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[filepath.Clean(path)]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return data, nil
}

func (m *u1FS) RemoveAll(path string) error {
	m.removeSubtree(path)

	return nil
}

func (m *u1FS) Stat(path string) (update.FileInfo, error) {
	clean := filepath.Clean(path)

	if m.dirs[clean] {
		return u1FileInfo{isDir: true}, nil
	}

	if _, ok := m.files[clean]; ok {
		return u1FileInfo{isDir: false}, nil
	}

	return nil, fs.ErrNotExist
}

func (m *u1FS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	clean := filepath.Clean(path)

	m.files[clean] = bytes.Clone(data)
	m.dirs[filepath.Dir(clean)] = true

	return nil
}

// destSnapshot returns a copy of all files written outside the source repo
// (i.e. the installed harness files under home), keyed by path.
func (m *u1FS) destSnapshot() map[string]string {
	out := map[string]string{}
	srcPrefix := u1RepoRoot + string(filepath.Separator)

	for path, data := range m.files {
		if strings.HasPrefix(path, srcPrefix) {
			continue
		}

		out[path] = string(data)
	}

	return out
}

func (m *u1FS) hasChildren(dir string) bool {
	prefix := dir + string(filepath.Separator)
	for p := range m.files {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}

	for p := range m.dirs {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}

	return false
}

// removeSubtree deletes path and every file/dir entry beneath it, matching
// os.RemoveAll semantics (no error when absent).
func (m *u1FS) removeSubtree(path string) {
	clean := filepath.Clean(path)
	prefix := clean + string(filepath.Separator)

	delete(m.dirs, clean)

	for p := range m.files {
		if p == clean || strings.HasPrefix(p, prefix) {
			delete(m.files, p)
		}
	}

	for p := range m.dirs {
		if strings.HasPrefix(p, prefix) {
			delete(m.dirs, p)
		}
	}
}

// seedLocalRepo populates a minimal local-clone layout: a go.mod naming the
// engram module, two skill files, both harness probe dirs, and the go bin dir.
func (m *u1FS) seedLocalRepo() {
	m.files[filepath.Join(u1RepoRoot, "go.mod")] = []byte("module " + update.ModulePath + "\n")
	m.files[filepath.Join(u1RepoRoot, "skills", "learn", "SKILL.md")] = []byte("learn skill\n")
	m.files[filepath.Join(u1RepoRoot, "skills", "recall", "SKILL.md")] = []byte("recall skill\n")
	m.dirs[filepath.Join(u1RepoRoot, "skills")] = true
	m.dirs[filepath.Join(u1RepoRoot, "skills", "learn")] = true
	m.dirs[filepath.Join(u1RepoRoot, "skills", "recall")] = true
	// commands dir intentionally absent → planCommandCopies skips it.
	m.dirs[filepath.Join(u1Home, ".claude")] = true
	m.dirs[filepath.Join(u1Home, ".config", "opencode")] = true
}

// u1FailCmd is a Commander whose Run always fails with a fixed error.
type u1FailCmd struct{ err error }

func (c *u1FailCmd) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, c.err
}

// u1FileInfo / u1DirEntry satisfy the update package's FileInfo / DirEntry.
type u1FileInfo struct{ isDir bool }

func (i u1FileInfo) IsDir() bool { return i.isDir }

// u1OKCmd is a Commander whose Run always succeeds (used to stand in for a
// working `go install` / `go list`).
type u1OKCmd struct{}

func (*u1OKCmd) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, nil
}

// childName returns the immediate child segment of full under dir, if full is
// within dir.
func childName(dir, full string) (string, bool) {
	prefix := dir + string(filepath.Separator)
	if !strings.HasPrefix(full, prefix) {
		return "", false
	}

	rest := strings.TrimPrefix(full, prefix)
	if rest == "" {
		return "", false
	}

	seg, _, _ := strings.Cut(rest, string(filepath.Separator))

	return seg, true
}

func newU1FS() *u1FS {
	return &u1FS{files: map[string][]byte{}, dirs: map[string]bool{}}
}

func u1LocalEnv() *u1Env { return &u1Env{home: u1Home, cwd: u1RepoRoot} }
