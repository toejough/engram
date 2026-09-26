package cli

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

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
		FS:  skillRegUpdateStubFS{},
		Env: skillRegUpdateStubEnv{},
		SkillReg: skillRegUpdateDepsFixture(
			nil, nil, func(string) ([]fs.DirEntry, error) { return nil, errSkillRegUpdateFixture },
		),
	}

	report := update.Report{Home: "/home/x", Source: update.SourceInfo{Root: "/repo"}}

	var stdout bytesBufferForTest

	err := runPostUpdateChecks(context.Background(), UpdateArgs{}, deps, &report, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.SkillRegistrationErr).To(ContainSubstring("injected"))
}

// TestRunUpdateSkillRegistration_DryRunListsOffers proves the hook builds
// SkillsDir as <source.Root>/agent-instructions/skills and forwards update's
// own --dry-run flag.
func TestRunUpdateSkillRegistration_DryRunListsOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := filepath.Join("/repo", "agent-instructions", "skills")
	skillContent := []byte("# Curate\n")

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(
		map[string][]byte{filepath.Join(skillsDir, "curate", "SKILL.md"): skillContent},
		[]string{"curate"}, nil,
	)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), true, "/vault", "/repo", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("would offer: register curate\n"))
}

// TestRunUpdateSkillRegistration_NoOpWhenSkillRegUnconfigured proves the
// zero-value updateDeps.SkillReg (older test fixtures built before this hook
// existed) is a safe no-op.
func TestRunUpdateSkillRegistration_NoOpWhenSkillRegUnconfigured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/repo", updateDeps{}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

// TestRunUpdateSkillRegistration_NoOpWhenSourceRootEmpty proves an
// unresolved source (empty Root) skips registration entirely — including
// never consulting deps.SkillReg.
func TestRunUpdateSkillRegistration_NoOpWhenSourceRootEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(
		nil, nil, func(string) ([]fs.DirEntry, error) {
			t.Fatal("must not list a skills dir when source.Root is empty")

			return nil, nil
		},
	)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

// TestRunUpdateSkillRegistration_NonInteractive_ReportsSummary covers the
// update-deploy-sync non-interactive scenario: nothing is written, and the
// output names the waiting skill.
func TestRunUpdateSkillRegistration_NonInteractive_ReportsSummary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := filepath.Join("/repo", "agent-instructions", "skills")
	skillContent := []byte("# Curate\n")

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(
		map[string][]byte{filepath.Join(skillsDir, "curate", "SKILL.md"): skillContent},
		[]string{"curate"}, nil,
	)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/repo", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("awaiting an answer: curate"))
}

// TestRunUpdateSkillRegistration_PropagatesError proves a registration
// failure is returned to the caller (runPostUpdateChecks then records it on
// the report rather than failing).
func TestRunUpdateSkillRegistration_PropagatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := updateDeps{Env: skillRegUpdateStubEnv{}, SkillReg: skillRegUpdateDepsFixture(
		nil, nil, func(string) ([]fs.DirEntry, error) { return nil, errSkillRegUpdateFixture },
	)}

	var stdout bytesBufferForTest

	err := runUpdateSkillRegistration(context.Background(), false, "/vault", "/repo", deps, &stdout)

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

// skillRegUpdateDepsFixture builds a minimal SkillRegistrationDeps: skills
// content/dir names back ListSkillsDir/ReadSkillFile (skipped when
// listOverride is non-nil, which replaces ListSkillsDir outright — for
// tests asserting it's never called, or that it fails); ListMD reports an
// empty vault (no existing notes, no declines).
func skillRegUpdateDepsFixture(
	skillsContent map[string][]byte,
	skillDirNames []string,
	listOverride func(string) ([]fs.DirEntry, error),
) SkillRegistrationDeps {
	listSkillsDir := listOverride
	if listSkillsDir == nil {
		listSkillsDir = func(string) ([]fs.DirEntry, error) {
			entries := make([]fs.DirEntry, 0, len(skillDirNames))
			for _, name := range skillDirNames {
				entries = append(entries, fakeDirEntry{name: name, dir: true})
			}

			return entries, nil
		}
	}

	return SkillRegistrationDeps{
		ListSkillsDir: listSkillsDir,
		ReadSkillFile: func(path string) ([]byte, error) {
			content, ok := skillsContent[path]
			if !ok {
				return nil, fs.ErrNotExist
			}

			return content, nil
		},
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
