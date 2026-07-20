package update_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/update"
)

// Property tests --------------------------------------------------------

func TestDetectHarnesses_Property_ResultIsSubset(t *testing.T) {
	t.Parallel()

	// Generate any combination of harness probes; verify result is a subset
	// in stable canonical order.
	for _, combo := range [][]string{
		{},
		{".claude"},
		{".config/opencode"},
		{".claude", ".config/opencode"},
		{".config/opencode", ".claude"}, // order of insertion shouldn't matter
	} {
		fileSystem := newMemFS()
		for _, probe := range combo {
			fileSystem.dirs["/home/joe/"+probe] = true
		}

		got, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
		if err != nil {
			t.Fatalf("detect: %v", err)
		}

		// Order check.
		for i := 1; i < len(got); i++ {
			if got[i-1].Name == update.HarnessOpencode && got[i].Name == update.HarnessClaude {
				t.Fatalf("unexpected order: %v", got)
			}
		}
	}
}

func TestFirstModuleLine_LeadingBlankAndComment(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("\n// comment\nmodule github.com/toejough/engram\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestFirstModuleLine_NotModule(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("go 1.25.0\nmodule github.com/toejough/engram\n")

	_, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

func TestFirstModuleLine_TrailingComment(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram // generated\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestUpdater_Run_DryRun_NoWritesNoCommands(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.DryRun).To(BeTrue())
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(1))
	// No real writes occurred.
	g.Expect(fileSystem.written).To(BeEmpty())
	// No commands invoked.
	g.Expect(cmd.calls).To(BeEmpty())
}

func TestUpdater_Run_GoInstallFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true

	cmdErr := errors.New("go install boom")
	cmd := &fakeCmd{err: cmdErr}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, cmdErr)).To(BeTrue())
}

func TestUpdater_Run_Local_BothHarnesses_CommandsOnlyOpencode(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/recall"] = true
	fileSystem.files["/repo/skills/recall/SKILL.md"] = []byte("r")
	fileSystem.dirs["/repo/commands"] = true
	fileSystem.files["/repo/commands/recall.md"] = []byte("c")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(2))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses[1].Err).NotTo(HaveOccurred())
	// Only opencode receives the command file.
	g.Expect(report.Harnesses[0].CommandFiles).To(BeEmpty())
	g.Expect(report.Harnesses[1].CommandFiles).To(ContainElement("recall.md"))
}

func TestUpdater_Run_Local_CommandRemoveAllFailureIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mem := newMemFS()
	mem.dirs["/home/joe/.config/opencode"] = true
	mem.dirs["/repo"] = true
	mem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	mem.dirs["/repo/skills"] = true
	mem.dirs["/repo/skills/recall"] = true
	mem.files["/repo/skills/recall/SKILL.md"] = []byte("r")
	mem.dirs["/repo/commands"] = true
	mem.files["/repo/commands/recall.md"] = []byte("c")

	// Fail RemoveAll for command path only (skill RemoveAll succeeds).
	fileSystem := &failRemoveAllFS{memFS: mem, failOn: "commands"}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("clear"))
}

func TestUpdater_Run_Local_GoInstallRunsFromModuleRoot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo/internal/update"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	// go install must run from the module root, not from the process cwd.
	g.Expect(cmd.calls).To(HaveLen(1))
	g.Expect(cmd.dirs[0]).To(Equal("/repo"))
}

func TestUpdater_Run_Local_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	// One harness present.
	fileSystem.dirs["/home/joe/.claude"] = true
	// Local repo with go.mod, skills, no slash commands.
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("learn skill")
	fileSystem.files["/repo/skills/learn/tests/baseline.md"] = []byte("baseline")
	fileSystem.dirs["/repo/skills/learn/tests"] = true

	cmd := &fakeCmd{}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo/internal/update"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Source.Mode).To(Equal(update.SourceLocal))
	g.Expect(report.Source.Root).To(Equal("/repo"))
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Name).To(Equal(update.HarnessClaude))
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(2))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	// Confirm files written under home, not under repo.
	_, ok := fileSystem.written["/home/joe/.claude/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.claude/skills/learn/tests/baseline.md"]
	g.Expect(ok).To(BeTrue())

	// `go install ./cmd/engram/` should have been invoked.
	g.Expect(cmd.calls).To(HaveLen(1))
	g.Expect(cmd.calls[0]).To(Equal([]string{"go", "install", "./cmd/engram/"}))
}

func TestUpdater_Run_Local_Idempotent_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		fileCount := rapid.IntRange(1, 4).Draw(rt, "fileCount")
		preExisting := rapid.Bool().Draw(rt, "preExisting")

		fs1 := newMemFS()
		buildLocalRepoForRapid(fs1, fileCount)

		if preExisting {
			fs1.dirs["/home/joe/.claude/skills/learn"] = true
			fs1.files["/home/joe/.claude/skills/learn/leftover.md"] = []byte("stale")
		}

		updater := &update.Updater{
			FS:  fs1,
			Cmd: &fakeCmd{},
			Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
		}

		_, runErr := updater.Run(rapidCtx(), update.Options{})
		if runErr != nil {
			rt.Fatalf("first run: %v", runErr)
		}

		afterFirst := snapshotDestFiles(fs1)

		_, runErr = updater.Run(rapidCtx(), update.Options{})
		if runErr != nil {
			rt.Fatalf("second run: %v", runErr)
		}

		afterSecond := snapshotDestFiles(fs1)
		if !equalStringByteMap(afterFirst, afterSecond) {
			rt.Fatalf("re-run not idempotent: %v vs %v", afterFirst, afterSecond)
		}
	})
}

func TestUpdater_Run_Local_OpencodeOnly_CopiesCommands(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.config/opencode"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/recall"] = true
	fileSystem.files["/repo/skills/recall/SKILL.md"] = []byte("recall")
	fileSystem.dirs["/repo/commands"] = true
	fileSystem.files["/repo/commands/recall.md"] = []byte("recall cmd")
	fileSystem.files["/repo/commands/learn.md"] = []byte("learn cmd")
	// Non-md should be ignored.
	fileSystem.files["/repo/commands/README.txt"] = []byte("readme")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	if len(report.Harnesses) == 0 {
		return
	}

	g.Expect(report.Harnesses[0].Name).To(Equal(update.HarnessOpencode))
	g.Expect(skillFileCount(report.Harnesses[0])).To(Equal(1))
	g.Expect(report.Harnesses[0].CommandFiles).To(HaveLen(2))

	_, ok := fileSystem.written["/home/joe/.config/opencode/commands/recall.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.config/opencode/commands/README.txt"]
	g.Expect(ok).To(BeFalse())
}

func TestUpdater_Run_Local_OverwritesExistingCommandFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.config/opencode"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/recall"] = true
	fileSystem.files["/repo/skills/recall/SKILL.md"] = []byte("recall")
	fileSystem.dirs["/repo/commands"] = true
	fileSystem.files["/repo/commands/recall.md"] = []byte("new cmd")

	// Pre-existing command file at the destination (simulates a stale link).
	fileSystem.files["/home/joe/.config/opencode/commands/recall.md"] = []byte("old cmd")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.removed).To(ContainElement("/home/joe/.config/opencode/commands/recall.md"))
	g.Expect(fileSystem.files["/home/joe/.config/opencode/commands/recall.md"]).
		To(Equal([]byte("new cmd")))
}

func TestUpdater_Run_Local_OverwritesExistingSkillDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true
	fileSystem.files["/repo/skills/learn/SKILL.md"] = []byte("new")

	// Pre-existing stale content at the destination.
	fileSystem.dirs["/home/joe/.claude/skills/learn"] = true
	fileSystem.files["/home/joe/.claude/skills/learn/stale.md"] = []byte("old")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	// Destination dir was removed before being recreated, so the stale
	// file is gone and the new file is in place.
	g.Expect(fileSystem.removed).To(ContainElement("/home/joe/.claude/skills/learn"))

	_, staleStillThere := fileSystem.files["/home/joe/.claude/skills/learn/stale.md"]
	g.Expect(staleStillThere).To(BeFalse())

	_, newPresent := fileSystem.files["/home/joe/.claude/skills/learn/SKILL.md"]
	g.Expect(newPresent).To(BeTrue())
}

func TestUpdater_Run_Local_RemoveAllFailureIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mem := newMemFS()
	mem.dirs["/home/joe/.claude"] = true
	mem.dirs["/repo"] = true
	mem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	mem.dirs["/repo/skills"] = true
	mem.dirs["/repo/skills/learn"] = true
	mem.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	fileSystem := &failRemoveAllFS{memFS: mem, failOn: "learn"}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("remove boom"))
}

func TestUpdater_Run_NoCwdReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe"}, // empty cwd
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("getwd"))
}

func TestUpdater_Run_NoHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater := &update.Updater{
		FS:  newMemFS(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/anywhere"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrNoHarness)).To(BeTrue())
}

func TestUpdater_Run_NoHomeReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater := &update.Updater{
		FS:  newMemFS(),
		Cmd: &fakeCmd{},
		Env: &fakeEnv{}, // empty home
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("resolving home"))
}

func TestUpdater_Run_PartialFailure_AllFail(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/joe/.claude"] = true
	base.dirs["/repo"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/skills"] = true
	base.dirs["/repo/skills/learn"] = true
	base.files["/repo/skills/learn/SKILL.md"] = []byte("x")

	fileSystem := &failWriteFS{memFS: base, failOn: ".claude/skills"}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
}

func TestUpdater_Run_RemoteCloneFailures(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// git missing entirely → ErrGitNotFound with the clear sentinel.
	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	cmd := &fakeCmd{err: update.ErrCommandNotFound}
	updater := &update.Updater{FS: fileSystem, Cmd: cmd, Env: &fakeEnv{home: "/home/joe", cwd: "/x"}}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(update.ErrGitNotFound))

	// clone fails for another reason → raw error surfaces.
	fileSystem2 := newMemFS()
	fileSystem2.dirs["/home/joe/.claude"] = true

	cloneBoom := errors.New("network down")
	cmd2 := &fakeCmd{err: cloneBoom}
	updater2 := &update.Updater{FS: fileSystem2, Cmd: cmd2, Env: &fakeEnv{home: "/home/joe", cwd: "/x"}}

	_, err = updater2.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(cloneBoom))

	// model file unreadable post-clone (clone hook seeds nothing) → read error.
	fileSystem3 := newMemFS()
	fileSystem3.dirs["/home/joe/.claude"] = true

	cmd3 := &fakeCmd{responses: map[string][]byte{}}
	updater3 := &update.Updater{FS: fileSystem3, Cmd: cmd3, Env: &fakeEnv{home: "/home/joe", cwd: "/x"}}

	_, err = updater3.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading cloned model"))
}

func TestUpdater_Run_RemoteClonesForLFSAndBuildsFromClone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	const cloneDir = "/tmp/engram-update-clone"

	realModel := bytes.Repeat([]byte{0x08, 0x01}, 1<<20) // binary + >1MB: not an LFS pointer

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD": []byte("abc1234\n"),
		},
	}
	cmd.hook = func(call []string) {
		if call[0] == "git" && call[1] == "clone" {
			seedCloneFixture(fileSystem, cloneDir, realModel)
		}
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
	}

	rep, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(rep.Source.Mode).To(Equal(update.SourceRemote))
	g.Expect(rep.Source.Root).To(Equal(cloneDir))
	g.Expect(rep.Source.Version).To(Equal("abc1234"))

	var sawClone, sawInstallFromClone bool

	for i, call := range cmd.calls {
		if call[0] == "git" && call[1] == "clone" {
			sawClone = true

			g.Expect(call).To(ContainElement("https://github.com/toejough/engram.git"),
				"clone runs the LFS smudge filter — the whole point of #645")
		}

		if call[0] == "go" && call[1] == "install" {
			sawInstallFromClone = true

			g.Expect(call).To(ContainElement("./cmd/engram/"))
			g.Expect(cmd.dirs[i]).To(Equal(cloneDir), "build FROM the clone, not the module cache")
		}
	}

	g.Expect(sawClone).To(BeTrue())
	g.Expect(sawInstallFromClone).To(BeTrue())

	for _, call := range cmd.calls {
		g.Expect(strings.Join(call, " ")).NotTo(ContainSubstring("@latest"),
			"the GOPROXY path ships the 133-byte LFS pointer stub — must be gone")
	}
}

func TestUpdater_Run_RemoteModelStubFailsWithGuidance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	const cloneDir = "/tmp/engram-update-clone"

	stub := []byte("version https://git-lfs.github.com/spec/v1\noid sha256:abc\nsize 90405214\n")

	cmd := &fakeCmd{responses: map[string][]byte{}}
	cmd.hook = func(call []string) {
		if call[0] == "git" && call[1] == "clone" {
			seedCloneFixture(fileSystem, cloneDir, stub)
		}
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/elsewhere"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(MatchError(update.ErrModelLFSStub))
	g.Expect(err.Error()).To(ContainSubstring("git-lfs"), "actionable guidance required")

	for _, call := range cmd.calls {
		g.Expect(call[0]+" "+call[1]).NotTo(Equal("go install"),
			"never build a binary that embeds the pointer stub")
	}
}

func TestUpdater_Run_Remote_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true

	// No go.mod under cwd → remote mode → clone-based build (#645).
	const cloneDir = "/tmp/engram-update-clone"

	cmd := &fakeCmd{
		responses: map[string][]byte{
			"git rev-parse --short HEAD": []byte("v0abc12\n"),
		},
	}
	cmd.hook = func(call []string) {
		if call[0] == "git" && call[1] == "clone" {
			seedCloneFixture(fileSystem, cloneDir, bytes.Repeat([]byte{1}, 1<<20))
			fileSystem.files[cloneDir+"/skills/learn/SKILL.md"] = []byte("learn from clone")
			fileSystem.dirs[cloneDir+"/commands"] = true
			fileSystem.files[cloneDir+"/commands/learn.md"] = []byte("learn cmd")
		}
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/tmp"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Source.Mode).To(Equal(update.SourceRemote))
	g.Expect(report.Source.Root).To(Equal(cloneDir))
	g.Expect(report.Source.Version).To(Equal("v0abc12"))
	g.Expect(report.Harnesses).To(HaveLen(2))

	// Both harnesses got the skill.
	_, ok := fileSystem.written["/home/joe/.claude/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.config/opencode/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())

	// OpenCode also got the command file; Claude did not (no commands target).
	_, ok = fileSystem.written["/home/joe/.config/opencode/commands/learn.md"]
	g.Expect(ok).To(BeTrue())
	_, ok = fileSystem.written["/home/joe/.claude/commands/learn.md"]
	g.Expect(ok).To(BeFalse())
}

func TestUpdater_Run_SkillsSrcMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	// No /repo/skills present.

	cmd := &fakeCmd{}
	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: cmd,
		Env: &fakeEnv{home: "/home/joe", cwd: "/repo"},
	}

	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrSkillsSrcMissing)).To(BeTrue())
}

// failRemoveAllFS wraps memFS but returns an error from RemoveAll.
type failRemoveAllFS struct {
	*memFS

	failOn string
}

func (f *failRemoveAllFS) RemoveAll(path string) error {
	if strings.Contains(path, f.failOn) {
		return errors.New("remove boom")
	}

	return f.memFS.RemoveAll(path)
}

// failWriteFS wraps memFS but returns an error from WriteFile.
type failWriteFS struct {
	*memFS

	failOn string
}

func (f *failWriteFS) WriteFile(path string, data []byte, mode fs.FileMode) error {
	if strings.Contains(path, f.failOn) {
		return errors.New("write boom")
	}

	return f.memFS.WriteFile(path, data, mode)
}

type fakeCmd struct {
	responses map[string][]byte
	err       error
	calls     [][]string
	dirs      []string
	// hook runs before each command resolves — lets a test materialize
	// filesystem state as a side effect (e.g. `git clone` producing files).
	hook func(call []string)
}

func (f *fakeCmd) Run(_ context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)
	f.dirs = append(f.dirs, dir)

	if f.hook != nil {
		f.hook(call)
	}

	if f.err != nil {
		return nil, nil, f.err
	}

	var keyBuilder strings.Builder

	keyBuilder.WriteString(name)

	for _, a := range args {
		keyBuilder.WriteString(" ")
		keyBuilder.WriteString(a)
	}

	key := keyBuilder.String()

	if resp, ok := f.responses[key]; ok {
		return resp, nil, nil
	}

	return nil, nil, nil
}

// --- fakes ---------------------------------------------------------------

type fakeEnv struct {
	home string
	cwd  string
}

func (f *fakeEnv) Getenv(_ string) string {
	return ""
}

func (f *fakeEnv) Getwd() (string, error) {
	if f.cwd == "" {
		return "", fs.ErrNotExist
	}

	return f.cwd, nil
}

func (f *fakeEnv) UserHomeDir() (string, error) {
	if f.home == "" {
		return "", fs.ErrNotExist
	}

	return f.home, nil
}

func buildLocalRepoForRapid(fileSystem *memFS, fileCount int) {
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/skills"] = true
	fileSystem.dirs["/repo/skills/learn"] = true

	for i := range fileCount {
		fileSystem.files["/repo/skills/learn/file"+strconv.Itoa(i)+".md"] = []byte("content")
	}
}

func equalStringByteMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

func rapidCtx() context.Context {
	return context.Background()
}

// seedCloneFixture populates the memFS with what a successful LFS-enabled
// clone of the repo would materialize at dir.
func seedCloneFixture(fileSystem *memFS, dir string, model []byte) {
	fileSystem.dirs[dir] = true
	fileSystem.files[dir+"/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs[dir+"/skills"] = true
	fileSystem.dirs[dir+"/skills/learn"] = true
	fileSystem.files[dir+"/skills/learn/SKILL.md"] = []byte("skill body")
	fileSystem.files[dir+"/internal/embed/assets/model/model.onnx"] = model
}

func skillFileCount(h update.HarnessReport) int {
	total := 0

	for _, d := range h.SkillDirs {
		total += d.Files
	}

	return total
}

func snapshotDestFiles(fileSystem *memFS) map[string]string {
	out := map[string]string{}

	for path, data := range fileSystem.files {
		if strings.HasPrefix(path, "/home/") {
			out[path] = string(data)
		}
	}

	return out
}
