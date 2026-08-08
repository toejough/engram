package update_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

func TestRun_DryRun_NeverInstallsOrSpawns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := &fakeCmd{}
	spawner := &fakeSpawner{}

	updater := &update.Updater{
		FS:    reexecFixture(),
		Cmd:   cmd,
		Env:   &fakeEnv{home: reexecHome, cwd: "/repo"},
		Spawn: spawner,
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Revision resolution (git rev-parse) still runs under --dry-run for
	// report visibility (update-local-install-safety task 2.1); go install
	// and the spawn never do.
	g.Expect(cmd.calls).To(Equal([][]string{{"git", "rev-parse", "--short", "HEAD"}}))
	g.Expect(spawner.calls).To(BeEmpty())
	g.Expect(report.ReexecExitCode).To(BeNil())
}

// TestRun_HandoffFails_PropagatesErrorWithoutSpawningOrFallback covers
// design-review fix B: a Handoff.WriteHandoff failure is a stdout-write
// failure, not a re-exec failure (re-exec was never attempted) — it must
// propagate as a hard error out of Run, not get smoothed into
// ReexecFallbackErr and an in-process fallback (the same broken stdout
// would just resurface on that fallback's own report write).
func TestRun_HandoffFails_PropagatesErrorWithoutSpawningOrFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errHandoff := errors.New("write: broken pipe")
	spawner := &fakeSpawner{}

	updater := &update.Updater{
		FS:  reexecFixture(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: reexecHome, cwd: "/repo"},
		Handoff: &fakeHandoff{fn: func(update.Report) error {
			return errHandoff
		}},
		Spawn: spawner,
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errHandoff))
	g.Expect(err.Error()).To(ContainSubstring("writing re-exec handoff report:"))

	g.Expect(spawner.calls).To(BeEmpty(), "a handoff-report failure must never reach the spawn call")
	g.Expect(report.ReexecExitCode).To(BeNil())
	g.Expect(report.ReexecFallbackErr).To(BeEmpty(), "a handoff failure is not a re-exec fallback")
	g.Expect(report.Harnesses).To(BeEmpty(), "the run aborts — no in-process fallback runs either")
}

func TestRun_InstallSucceeds_ReportNotMarkedReexecChild(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var gotChild bool

	updater := &update.Updater{
		FS:  reexecFixture(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: reexecHome, cwd: "/repo"},
		Handoff: &fakeHandoff{fn: func(r update.Report) error {
			gotChild = r.ReexecChild

			return nil
		}},
		Spawn: &fakeSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotChild).To(BeFalse(), "the parent's own report is never the child report")
}

func TestRun_InstallSucceeds_SpawnsInstalledPathWithSentinelEnvAndOriginalArgs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := &fakeCmd{}
	spawner := &fakeSpawner{exitCode: 7}

	updater := &update.Updater{
		FS:    reexecFixture(),
		Cmd:   cmd,
		Env:   &fakeEnv{home: reexecHome, cwd: "/repo"},
		Spawn: spawner,
	}

	report, err := updater.Run(context.Background(), update.Options{ReexecArgs: []string{"--with-guidance"}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cmd.calls).To(HaveLen(2), "revision resolution, then install, exactly once before the re-exec")
	g.Expect(cmd.calls[1]).To(Equal([]string{"go", "install", "./cmd/engram/"}))
	g.Expect(spawner.calls).To(HaveLen(1))

	call := spawner.calls[0]
	g.Expect(call.name).To(Equal(report.BinaryPath), "re-exec target is resolveBinaryPath, never os.Args[0]")
	g.Expect(call.args).To(Equal([]string{"update", "--with-guidance"}))
	g.Expect(call.env).To(ContainElement("ENGRAM_UPDATE_REEXEC=1"))

	g.Expect(report.ReexecExitCode).NotTo(BeNil())

	if report.ReexecExitCode == nil {
		return
	}

	g.Expect(*report.ReexecExitCode).To(Equal(7), "a non-zero child exit code propagates verbatim")
	g.Expect(report.Harnesses).To(BeEmpty(), "parent performs no planning/apply after handing off")
}

func TestRun_InstallSucceeds_WritesHandoffReportBeforeSpawning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var order []string

	spawner := &fakeSpawner{hook: func() { order = append(order, "spawn") }}

	updater := &update.Updater{
		FS:  reexecFixture(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: reexecHome, cwd: "/repo"},
		Handoff: &fakeHandoff{fn: func(update.Report) error {
			order = append(order, "report")

			return nil
		}},
		Spawn: spawner,
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(order).To(Equal([]string{"report", "spawn"}),
		"the parent's install-result report must be written before the child's inherited-stdio output begins")
}

func TestRun_SentinelSet_ReportMarkedReexecChildAndNoInstallClaim(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater := &update.Updater{
		FS:  reexecFixture(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: reexecHome, cwd: "/repo", sentinel: "1"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report.ReexecChild).To(BeTrue(), "a sentinel-bearing run must mark its report as a re-exec child's")
}

func TestRun_SentinelSet_SkipsInstallAndSpawnRunsInProcess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := &fakeCmd{}
	spawner := &fakeSpawner{}

	updater := &update.Updater{
		FS:    reexecFixture(),
		Cmd:   cmd,
		Env:   &fakeEnv{home: reexecHome, cwd: "/repo", sentinel: "1"},
		Spawn: spawner,
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(cmd.calls).To(Equal([][]string{{"git", "rev-parse", "--short", "HEAD"}}),
		"sentinel run resolves revision for visibility but must not invoke go install")
	g.Expect(spawner.calls).To(BeEmpty(), "sentinel run must not spawn again")
	g.Expect(report.ReexecExitCode).To(BeNil())
	g.Expect(report.Harnesses).To(HaveLen(1), "sentinel run completes sync/checks in-process")
}

func TestRun_SpawnFails_FallsBackInProcessAndRecordsReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errSpawn := errors.New("exec: no such file")

	cmd := &fakeCmd{}
	spawner := &fakeSpawner{err: errSpawn}

	updater := &update.Updater{
		FS:    reexecFixture(),
		Cmd:   cmd,
		Env:   &fakeEnv{home: reexecHome, cwd: "/repo"},
		Spawn: spawner,
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(spawner.calls).To(HaveLen(1))
	g.Expect(report.ReexecExitCode).To(BeNil())
	g.Expect(report.Harnesses).To(HaveLen(1), "fallback completes the full in-process run")
	g.Expect(report.ReexecFallbackErr).To(ContainSubstring("re-exec failed, completed with pre-update logic:"))
	g.Expect(report.ReexecFallbackErr).To(ContainSubstring(errSpawn.Error()))
}

// unexported constants.
const (
	reexecHome = "/home/joe"
)

// fakeHandoff is a hand-written test double for update.HandoffReporter.
type fakeHandoff struct {
	fn func(update.Report) error
}

func (f *fakeHandoff) WriteHandoff(r update.Report) error {
	return f.fn(r)
}

// fakeSpawner is a hand-written test double for update.Spawner: it records
// every invocation and returns a canned (exitCode, err) pair (matching the
// contract — err non-nil ONLY on spawn failure, see update.Spawner doc).
type fakeSpawner struct {
	exitCode int
	err      error
	calls    []spawnCall
	// hook runs before each call resolves — lets a test observe ordering
	// against other side effects (e.g. Handoff.WriteHandoff), mirrors fakeCmd's hook.
	hook func()
}

func (f *fakeSpawner) Run(name string, args []string, env []string) (int, error) {
	if f.hook != nil {
		f.hook()
	}

	f.calls = append(f.calls, spawnCall{name: name, args: args, env: env})

	return f.exitCode, f.err
}

// noopSpawner is a test helper that simulates a spawn error, used for tests
// that don't expect spawning (everything after the install boundary should
// route through the tested in-process path, not a real re-exec). Returning
// an error ensures the fallback logic runs, preserving the original tested
// behavior.
type noopSpawner struct{}

func (noopSpawner) Run(string, []string, []string) (int, error) {
	return 0, errors.New("re-exec unavailable in tests")
}

type spawnCall struct {
	name string
	args []string
	env  []string
}

// reexecFixture builds the minimal local-mode fixture (single Claude
// harness, no guidance) shared by TestRun_WithoutGuidance_SkipsGuidance and
// the re-exec boundary tests below.
func reexecFixture() *memFS {
	fileSystem := newMemFS()
	fileSystem.dirs[reexecHome+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true

	return fileSystem
}
