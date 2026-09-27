package cli

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// TestRunPostUpdateChecks_SkillRegistrationError_SetsReportFieldDoesNotFailChecks
// covers update-deploy-sync: "Registration failures SHALL be reported and
// SHALL NOT roll back the deploy" — a registration failure lands on
// report.SkillRegistrationErr, and runPostUpdateChecks itself still returns
// nil.
func TestRunPostUpdateChecks_SkillRegistrationError_SetsReportFieldDoesNotFailChecks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{
		FS:       skillRegUpdateStubFS{},
		Env:      skillRegUpdateStubEnv{},
		SkillReg: skillRegUpdateDepsFixture(nil, failingSkillRegGetwd),
	}

	report := update.Report{Home: "/home/x", Source: update.SourceInfo{Root: "/repo"}}

	var stdout bytesBufferForTest

	err := runPostUpdateChecks(context.Background(), UpdateArgs{}, deps, &report, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.SkillRegistrationErr).To(ContainSubstring("injected"))
}

// TestRunUpdateSkillRegistration_DryRunListsOffers proves the hook resolves
// the default source set from home (a Claude user skill offers, while the
// source checkout's agent-instructions/skills is never read) and forwards
// update's own --dry-run flag.
func TestRunUpdateSkillRegistration_DryRunListsOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(fstest.MapFS{
		"home/x/.claude/skills/curate/SKILL.md":          {Data: []byte("# Curate\n")},
		"repo/agent-instructions/skills/recall/SKILL.md": {Data: []byte("# Recall\n")},
	}, nil)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), true, "/vault", "/home/x", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal(
		"@claude-user (1)\n  would offer: register claude:curate (/home/x/.claude/skills/curate/SKILL.md)\n"))
}

// TestRunUpdateSkillRegistration_NoOpWhenSkillRegUnconfigured proves the
// zero-value updateDeps.SkillReg (older test fixtures built before this hook
// existed) is a safe no-op.
func TestRunUpdateSkillRegistration_NoOpWhenSkillRegUnconfigured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/home", updateDeps{}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

// TestRunUpdateSkillRegistration_NonInteractive_ReportsSummary covers the
// update-deploy-sync non-interactive scenario: nothing is written, and the
// output names the waiting skill.
func TestRunUpdateSkillRegistration_NonInteractive_ReportsSummary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(fstest.MapFS{
		"home/x/.claude/skills/curate/SKILL.md": {Data: []byte("# Curate\n")},
	}, nil)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/home/x", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("awaiting an answer: @claude-user 1"))
}

// TestRunUpdateSkillRegistration_PropagatesError proves a registration
// failure is returned to the caller (runPostUpdateChecks then records it on
// the report rather than failing).
func TestRunUpdateSkillRegistration_PropagatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(nil, failingSkillRegGetwd)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/home/x", deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillRegUpdateFixture))
}

// unexported variables.
var (
	errSkillRegUpdateFixture = errors.New("skillreg-update fixture: injected failure")
)

// unexported test helpers.

// bytesBufferForTest is a minimal io.Writer capturing what's written to it —
// avoids importing bytes.Buffer just for these tests' one use.
type bytesBufferForTest struct {
	data []byte
}

func (b *bytesBufferForTest) String() string { return string(b.data) }

func (b *bytesBufferForTest) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)

	return len(p), nil
}

// mapSkillSourceFS adapts an fstest.MapFS (unrooted paths) to the absolute
// paths SkillSourceFS takes.
type mapSkillSourceFS struct {
	fsys fstest.MapFS
}

func (m mapSkillSourceFS) Lstat(path string) (fs.FileInfo, error) {
	return fs.Lstat(m.fsys, m.rel(path))
}

func (m mapSkillSourceFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return fs.ReadDir(m.fsys, m.rel(path))
}

func (m mapSkillSourceFS) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(m.fsys, m.rel(path))
}

func (m mapSkillSourceFS) Readlink(path string) (string, error) {
	return fs.ReadLink(m.fsys, m.rel(path))
}

func (m mapSkillSourceFS) rel(path string) string {
	rel := strings.TrimPrefix(filepath.Clean(path), "/")
	if rel == "" {
		return "."
	}

	return rel
}

// noGitCommander answers every command as a failed run: the working
// directory is in no git repository.
type noGitCommander struct{}

func (noGitCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, errSkillRegUpdateFixture
}

// skillRegUpdateStubEnv is a minimal update.Env whose Getenv always reports
// unset — every runPostUpdateChecks detector it feeds degrades to its
// self-silencing false/empty default.
type skillRegUpdateStubEnv struct{}

func (skillRegUpdateStubEnv) Getenv(string) string { return "" }

func (skillRegUpdateStubEnv) Getwd() (string, error) { return "/work", nil }

func (skillRegUpdateStubEnv) UserHomeDir() (string, error) { return "/home/x", nil }

// skillRegUpdateStubFS is a minimal update.Filesystem whose ReadDir/ReadFile
// always report "not exist" — every runPostUpdateChecks detector it feeds
// degrades to its self-silencing false default, isolating the test to the
// skill-registration hook alone.
type skillRegUpdateStubFS struct{}

func (skillRegUpdateStubFS) Lstat(string) (update.FileInfo, error) { return nil, fs.ErrNotExist }

func (skillRegUpdateStubFS) MkdirAll(string, fs.FileMode) error { return nil }

func (skillRegUpdateStubFS) ReadDir(string) ([]update.DirEntry, error) { return nil, fs.ErrNotExist }

func (skillRegUpdateStubFS) ReadFile(string) ([]byte, error) { return nil, fs.ErrNotExist }

func (skillRegUpdateStubFS) ReadLink(string) (string, error) { return "", fs.ErrNotExist }

func (skillRegUpdateStubFS) RemoveAll(string) error { return nil }

func (skillRegUpdateStubFS) Stat(string) (update.FileInfo, error) { return nil, fs.ErrNotExist }

func (skillRegUpdateStubFS) Symlink(string, string) error { return nil }

func (skillRegUpdateStubFS) WriteFile(string, []byte, fs.FileMode) error { return nil }

// failingSkillRegGetwd is a Getwd failing with errSkillRegUpdateFixture.
func failingSkillRegGetwd() (string, error) { return "", errSkillRegUpdateFixture }

// skillRegUpdateDepsFixture builds a minimal SkillRegistrationDeps whose
// default source set is resolved from files (an empty home when nil), from a
// working directory outside any git repository (getwd, when non-nil,
// replaces it); ListMD reports an empty vault (no existing notes, no
// declines).
func skillRegUpdateDepsFixture(files fstest.MapFS, getwd func() (string, error)) SkillRegistrationDeps {
	if files == nil {
		files = fstest.MapFS{}
	}

	if getwd == nil {
		getwd = func() (string, error) { return "/outside/any/repo", nil }
	}

	return SkillRegistrationDeps{
		Sources:    SkillSourceDeps{FS: mapSkillSourceFS{fsys: files}, Commander: noGitCommander{}},
		Getwd:      getwd,
		ListMD:     func(string) ([]string, error) { return nil, nil },
		IsTerminal: func() bool { return false },
		Accept: SkillAcceptDeps{
			Lock:   func(string) (func(), error) { return func() {}, nil },
			Read:   func(string) ([]byte, error) { return nil, fs.ErrNotExist },
			Write:  func(string, []byte) error { return nil },
			Remove: func(string) error { return nil },
		},
	}
}
