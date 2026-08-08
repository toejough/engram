package update_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

func TestUpdater_Run_Local_NoPriorBinary_ProceedsNoCheck(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := downgradeFixture()
	// No prior binary at the install path this time.
	delete(fileSystem.files, downgradeBinaryPath)

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD": []byte(downgradeResolvedRev + "\n"),
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	var sawInstall, sawVersionCheck bool

	for _, call := range cmd.calls {
		if len(call) == 3 && call[0] == "go" && call[1] == "install" {
			sawInstall = true
		}

		if len(call) > 1 && call[0] == "go" && call[1] == "version" {
			sawVersionCheck = true
		}
	}

	g.Expect(sawInstall).To(BeTrue())
	g.Expect(sawVersionCheck).To(BeFalse(), "no prior binary means go version -m must never run")
}

func TestUpdater_Run_Local_ProvableDowngrade_AllowDowngradeBypasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := downgradeFixture()
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD":           []byte(downgradeResolvedRev + "\n"),
			"go version -m " + downgradeBinaryPath: []byte("build\tvcs.revision=" + downgradeInstalledRev + "\n"),
		},
		errs: map[string]error{
			"git merge-base --is-ancestor " + downgradeInstalledRev + " " + downgradeResolvedRev: errors.New("exit status 1"),
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{AllowDowngrade: true})
	g.Expect(err).NotTo(HaveOccurred())

	var sawInstall bool

	for _, call := range cmd.calls {
		if len(call) == 3 && call[0] == "go" && call[1] == "install" {
			sawInstall = true
		}
	}

	g.Expect(sawInstall).To(BeTrue(), "--allow-downgrade must let the install proceed")

	// The ancestry check must never even run once the bypass is known.
	mergeBaseCall := []string{"git", "merge-base", "--is-ancestor", downgradeInstalledRev, downgradeResolvedRev}
	for _, call := range cmd.calls {
		g.Expect(call).NotTo(Equal(mergeBaseCall))
	}
}

func TestUpdater_Run_Local_ProvableDowngrade_Refused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := downgradeFixture()
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD":           []byte(downgradeResolvedRev + "\n"),
			"go version -m " + downgradeBinaryPath: []byte("build\tvcs.revision=" + downgradeInstalledRev + "\n"),
		},
		errs: map[string]error{
			"git merge-base --is-ancestor " + downgradeInstalledRev + " " + downgradeResolvedRev: errors.New("exit status 1"),
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrLocalDowngrade)).To(BeTrue())
	g.Expect(err.Error()).To(ContainSubstring(downgradeInstalledRev))
	g.Expect(err.Error()).To(ContainSubstring(downgradeResolvedRev))
	g.Expect(err.Error()).To(ContainSubstring("--allow-downgrade"))

	for _, call := range cmd.calls {
		g.Expect(call).NotTo(Equal([]string{"go", "install", "./cmd/engram/"}),
			"a provable downgrade must never run go install")
	}
}

func TestUpdater_Run_Local_UnknownInstalledRevision_FailsOpen(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := downgradeFixture()
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD":           []byte(downgradeResolvedRev + "\n"),
			"go version -m " + downgradeBinaryPath: []byte("build\tvcs.revision=" + downgradeInstalledRev + "\n"),
		},
		errs: map[string]error{
			"git merge-base --is-ancestor " + downgradeInstalledRev + " " + downgradeResolvedRev: errors.New("exit status 128"),
		},
		stderrs: map[string][]byte{
			"git merge-base --is-ancestor " + downgradeInstalledRev + " " + downgradeResolvedRev: []byte(
				"fatal: Not a valid commit name " + downgradeInstalledRev + "\n",
			),
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Source.Version).To(Equal(downgradeResolvedRev), "revision still shown for visibility")

	var sawInstall bool

	for _, call := range cmd.calls {
		if len(call) == 3 && call[0] == "go" && call[1] == "install" {
			sawInstall = true
		}
	}

	g.Expect(sawInstall).To(BeTrue(), "an unresolvable ancestry check must fail open")
}

func TestUpdater_Run_Local_UnparseableInstalledRevision_FailsOpen(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := downgradeFixture()
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD": []byte(downgradeResolvedRev + "\n"),
			// Simulates a binary built before Go's automatic VCS-embedding
			// (or one built with -buildvcs=false): no vcs.revision line.
			"go version -m " + downgradeBinaryPath: []byte("/home/joe/go/bin/engram: go1.23.0\n"),
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	var sawInstall, sawMergeBase bool

	for _, call := range cmd.calls {
		if len(call) == 3 && call[0] == "go" && call[1] == "install" {
			sawInstall = true
		}

		if len(call) > 1 && call[0] == "git" && call[1] == "merge-base" {
			sawMergeBase = true
		}
	}

	g.Expect(sawInstall).To(BeTrue(), "an unparseable installed revision must fail open")
	g.Expect(sawMergeBase).To(BeFalse(), "no installed revision means nothing to compare — ancestry check is skipped")
}

// unexported constants.
const (
	downgradeBinaryPath   = "/home/joe/go/bin/engram"
	downgradeInstalledRev = "deadbeefcafefeed0000000000000000000000"
	downgradeResolvedRev  = "abc1234"
)

// downgradeFixture builds a local-mode repo with one detected harness and a
// pre-existing binary at the resolved install path.
func downgradeFixture() *memFS {
	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("x")
	fileSystem.dirs["/home/joe/go/bin"] = true
	fileSystem.files[downgradeBinaryPath] = []byte("old binary")

	return fileSystem
}
