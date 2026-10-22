package update_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// Test792b28c1a2_ResolveBinaryPath_GOPATHIsNext verifies Scenario 792b28c1a2: with GOBIN unset and GOPATH set,
// the install location is <GOPATH>/bin/engram.
func Test792b28c1a2_ResolveBinaryPath_GOPATHIsNext(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportResolveBinaryPath("/home/joe", envVars{"GOPATH": "/custom/gopath"})
	g.Expect(got).To(Equal("/custom/gopath/bin/engram"))
}

// Test792b28c1a3_ResolveBinaryPath_HomeDirectoryIsTheFallback verifies Scenario 792b28c1a3: with neither GOBIN nor
// GOPATH set, the install location is <home>/go/bin/engram.
func Test792b28c1a3_ResolveBinaryPath_HomeDirectoryIsTheFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportResolveBinaryPath("/home/joe", envVars{})
	g.Expect(got).To(Equal("/home/joe/go/bin/engram"))
}

// Test792b28c1a5_ResolveBinaryPath_FirstGOPATHEntryIsUsed verifies Scenario 792b28c1a5: with GOBIN unset and a
// two-entry GOPATH, the install location is under the first entry only.
func Test792b28c1a5_ResolveBinaryPath_FirstGOPATHEntryIsUsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportResolveBinaryPath("/home/joe", envVars{"GOPATH": "/gp1:/gp2"})
	g.Expect(got).To(Equal("/gp1/bin/engram"))
}

// Test792b28c4a3_Updater_Run_Local_UnresolvableRevisionIsUnknown verifies Scenario 792b28c4a3: when git fails or
// prints nothing for rev-parse, the source's revision is "unknown" and the install still goes ahead.
func Test792b28c4a3_Updater_Run_Local_UnresolvableRevisionIsUnknown(t *testing.T) {
	t.Parallel()

	const revParseKey = "git rev-parse --short HEAD"

	table := map[string]*fakeCmd{
		"git fails": {errs: map[string]error{revParseKey: errors.New("not a git repository")}},
		"git prints nothing": {
			responses: map[string][]byte{revParseKey: []byte("\n")},
		},
	}

	for name, cmd := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			updater := &update.Updater{
				FS:    sip2LocalFixture(),
				Cmd:   cmd,
				Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
				Spawn: noopSpawner{},
			}

			report, err := updater.Run(context.Background(), update.Options{})
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(report.Source.Version).To(Equal("unknown"))
			g.Expect(report.BinaryVersion).To(Equal("unknown"))
			g.Expect(cmd.calls).To(ContainElement([]string{"go", "install", "./cmd/engram/"}))
		})
	}
}

// Test792b28c5a2_Updater_Run_Remote_EarlierCloneIsRemovedFirst verifies Scenario 792b28c5a2: a scratch clone left
// by an earlier run is removed before the new clone starts.
func Test792b28c5a2_Updater_Run_Remote_EarlierCloneIsRemovedFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs[sip2CloneDir] = true
	fileSystem.files[sip2CloneDir+"/left-by-an-earlier-run"] = []byte("stale")

	var (
		removedAtClone []string
		staleAtClone   bool
	)

	cmd := &fakeCmd{}
	cmd.hook = func(call []string) {
		if call[0] == "git" && call[1] == "clone" {
			removedAtClone = slices.Clone(fileSystem.removed)
			_, staleAtClone = fileSystem.files[sip2CloneDir+"/left-by-an-earlier-run"]

			seedCloneFixture(fileSystem, sip2CloneDir, sip2RealModel())
		}
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(removedAtClone).To(ContainElement(sip2CloneDir), "the earlier clone is removed before git clone runs")
	g.Expect(staleAtClone).To(BeFalse(), "nothing of the earlier clone is left when the new clone starts")
	g.Expect(fileSystem.files).NotTo(HaveKey(sip2CloneDir + "/left-by-an-earlier-run"))
}

// Test792b28c5a4_Updater_Run_Remote_AnotherCloneFailureSurfaces verifies Scenario 792b28c5a4: a git clone that
// fails with a network error fails the run with an error that wraps it and names git clone.
func Test792b28c5a4_Updater_Run_Remote_AnotherCloneFailureSurfaces(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	networkErr := errors.New("network down")
	cmd := &fakeCmd{errs: map[string]error{sip2CloneKey: networkErr}}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(networkErr))
	g.Expect(err).To(MatchError(ContainSubstring("git clone")))
	g.Expect(err).NotTo(MatchError(update.ErrGitNotFound))
	g.Expect(sip2GoInstallRan(cmd)).To(BeFalse(), "a failed clone builds nothing")
}

// Test792b28c6a2_Updater_Run_Remote_FileUnderOneMebibyteIsRefused verifies Scenario 792b28c6a2: a cloned
// model.onnx of a few bytes that is not a pointer fails with ErrModelLFSStub and nothing is installed.
func Test792b28c6a2_Updater_Run_Remote_FileUnderOneMebibyteIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := sip2CloningCmd(fileSystem, []byte("a few bytes"))

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(update.ErrModelLFSStub))
	g.Expect(sip2GoInstallRan(cmd)).To(BeFalse(), "a too-small model is never built into the binary")
}

// Test792b28c6a3_Updater_Run_Remote_UnreadableModelIsAnError verifies Scenario 792b28c6a3: a clone without
// model.onnx fails with an error containing "reading cloned model" and nothing is installed.
func Test792b28c6a3_Updater_Run_Remote_UnreadableModelIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(ContainSubstring("reading cloned model")))
	g.Expect(err).NotTo(MatchError(update.ErrModelLFSStub))
	g.Expect(sip2GoInstallRan(cmd)).To(BeFalse())
}

// Test792b28c6a4_Updater_Run_Remote_RealModelPasses verifies Scenario 792b28c6a4: a cloned model.onnx of more
// than one mebibyte that is not a pointer lets the build go ahead, from the clone.
func Test792b28c6a4_Updater_Run_Remote_RealModelPasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := sip2CloningCmd(fileSystem, sip2RealModel())

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	installIndex := slices.IndexFunc(cmd.calls, func(call []string) bool {
		return slices.Equal(call, []string{"go", "install", "./cmd/engram/"})
	})
	g.Expect(installIndex).To(BeNumerically(">=", 0), "go install runs")

	if installIndex < 0 {
		return
	}

	g.Expect(cmd.dirs[installIndex]).To(Equal(sip2CloneDir))
}

// Test792b28c7a6_Updater_Run_Local_DescendantInstalls verifies Scenario 792b28c7a6: when the ancestry check
// succeeds, the update installs locally, after the check.
func Test792b28c7a6_Updater_Run_Local_DescendantInstalls(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// No error is registered for the merge-base call: it exits zero, so the installed revision is an ancestor.
	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD":           []byte(downgradeResolvedRev + "\n"),
			"go version -m " + downgradeBinaryPath: []byte("build\tvcs.revision=" + downgradeInstalledRev + "\n"),
		},
	}

	updater := &update.Updater{
		FS:    downgradeFixture(),
		Cmd:   cmd,
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	ancestryIndex := slices.IndexFunc(cmd.calls, func(call []string) bool {
		return slices.Equal(call, []string{
			"git", "merge-base", "--is-ancestor", downgradeInstalledRev, downgradeResolvedRev,
		})
	})
	installIndex := slices.IndexFunc(cmd.calls, func(call []string) bool {
		return slices.Equal(call, []string{"go", "install", "./cmd/engram/"})
	})

	g.Expect(ancestryIndex).To(BeNumerically(">=", 0), "the ancestry check ran")
	g.Expect(installIndex).To(BeNumerically(">", ancestryIndex), "go install ran after the ancestry check succeeded")
}

// Test792b28c8a4_Updater_Run_Remote_DryRunStillClones verifies Scenario 792b28c8a4: with no local source,
// --dry-run clones the repository and checks its model, and go install does not run.
func Test792b28c8a4_Updater_Run_Remote_DryRunStillClones(t *testing.T) {
	t.Parallel()

	t.Run("a real model passes and nothing is installed", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := newMemFS()
		fileSystem.dirs["/home/joe/.claude"] = true

		cmd := sip2CloningCmd(fileSystem, sip2RealModel())

		updater := &update.Updater{
			FS:    fileSystem,
			Cmd:   cmd,
			Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
			Spawn: noopSpawner{},
		}

		report, err := updater.Run(context.Background(), update.Options{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(cmd.calls).To(ContainElement(
			[]string{"git", "clone", "--depth", "1", "https://github.com/toejough/engram.git", sip2CloneDir}))
		g.Expect(sip2GoInstallRan(cmd)).To(BeFalse())
		g.Expect(report.Source.Mode).To(Equal(update.SourceRemote))
		g.Expect(report.Source.Root).To(Equal(sip2CloneDir))
	})

	t.Run("a pointer model is refused, so the model is checked", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := newMemFS()
		fileSystem.dirs["/home/joe/.claude"] = true

		cmd := sip2CloningCmd(fileSystem, []byte("version https://git-lfs.github.com/spec/v1\n"))

		updater := &update.Updater{
			FS:    fileSystem,
			Cmd:   cmd,
			Env:   &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
			Spawn: noopSpawner{},
		}

		_, err := updater.Run(context.Background(), update.Options{DryRun: true})
		g.Expect(err).To(MatchError(update.ErrModelLFSStub))
		g.Expect(sip2GoInstallRan(cmd)).To(BeFalse())
	})
}

// unexported constants.
const (
	sip2CloneDir = "/tmp/engram-update-clone"
	sip2CloneKey = "git clone --depth 1 https://github.com/toejough/engram.git " + sip2CloneDir
	// sip2MinModelBytes is the size, one mebibyte, below which a cloned model is refused.
	sip2MinModelBytes = 1 << 20
)

// sip2CloningCmd returns a fakeCmd whose git clone seeds a clone holding model.
func sip2CloningCmd(fileSystem *memFS, model []byte) *fakeCmd {
	cmd := &fakeCmd{}
	cmd.hook = func(call []string) {
		if call[0] == "git" && call[1] == "clone" {
			seedCloneFixture(fileSystem, sip2CloneDir, model)
		}
	}

	return cmd
}

// sip2GoInstallRan reports whether the commander was asked to run go install.
func sip2GoInstallRan(cmd *fakeCmd) bool {
	return slices.ContainsFunc(cmd.calls, func(call []string) bool {
		return len(call) > 1 && call[0] == "go" && call[1] == "install"
	})
}

// sip2LocalFixture builds a local-mode repo with one detected harness and no prior binary.
func sip2LocalFixture() *memFS {
	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("x")

	return fileSystem
}

// sip2RealModel returns a model one byte over a mebibyte that does not begin with an LFS pointer line.
func sip2RealModel() []byte {
	return bytes.Repeat([]byte{0x08}, sip2MinModelBytes+1)
}
