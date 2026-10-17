package update_test

import (
	"context"
	"maps"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

func Test793d4b1a_PlanModCopies_PlansEveryFileOfAModForTheHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/src/agent-instructions/mods"] = true

	for _, file := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "hooks/register.js"} {
		fileSystem.files["/src/agent-instructions/mods/engram-memory/"+file] = []byte("x")
	}

	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, ProbeRel: ".claude", SkillsTargetRel: ".claude/skills"},
	}

	ops, err := update.ExportPlanModCopies("/src/agent-instructions/mods", "/home/joe", harnesses, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	copyOf := func(file string) update.CopyOp {
		return update.CopyOp{
			Harness:  update.HarnessClaude,
			Src:      "/src/agent-instructions/mods/engram-memory/" + file,
			Dst:      "/home/joe/.claude/skills/engram-memory/" + file,
			SkillDir: "engram-memory",
			Mod:      true,
		}
	}

	g.Expect(ops).To(ConsistOf(
		copyOf(".claude-plugin/plugin.json"), copyOf("hooks/hooks.json"), copyOf("hooks/register.js")))
}

func Test793d4b1b_PlanModCopies_ASourceWithNoModsPlansNone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, ProbeRel: ".claude", SkillsTargetRel: ".claude/skills"},
	}

	ops, err := update.ExportPlanModCopies("/src/agent-instructions/mods", "/home/joe", harnesses, newMemFS())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ops).To(BeEmpty())
}

func Test793d4b1c_Updater_Run_DeploysAndLinksTheMod(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater, fileSystem := updaterOverCheckoutWithMod()

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.written).To(HaveKey("/home/joe/.claude/engram/skills/engram-memory/hooks/register.js"))
	g.Expect(fileSystem.symlinks["/home/joe/.claude/skills/engram-memory"]).
		To(Equal("/home/joe/.claude/engram/skills/engram-memory"))
}

func Test793d4b1d_PlanModCopies_PiIsPlannedNoMod(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/src/agent-instructions/mods"] = true

	for _, file := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "hooks/register.js"} {
		fileSystem.files["/src/agent-instructions/mods/engram-memory/"+file] = []byte("x")
	}

	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, ProbeRel: ".claude", SkillsTargetRel: ".claude/skills"},
		{Name: update.HarnessPi, ProbeRel: ".pi", SkillsTargetRel: ".pi/agent/skills"},
	}

	ops, err := update.ExportPlanModCopies("/src/agent-instructions/mods", "/home/joe", harnesses, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ops).To(HaveLen(3))

	for _, op := range ops {
		g.Expect(op.Harness).To(Equal(update.HarnessClaude), op.Dst)
	}
}

func Test793d4b1e_Updater_Run_ReportsTheModAsAMod(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater, _ := updaterOverCheckoutWithMod()

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	g.Expect(report.Harnesses[0].SkillDirs).To(ConsistOf(
		update.SkillDirCount{Name: "learn", Files: 1},
		update.SkillDirCount{Name: "engram-memory", Files: 1, Mod: true},
	))
}

func Test793d4b1g_Updater_Run_ADryRunReportsTheModAndWritesNoneOfIt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater, fileSystem := updaterOverCheckoutWithMod()

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	g.Expect(report.Harnesses[0].SkillDirs).To(
		ContainElement(update.SkillDirCount{Name: "engram-memory", Files: 1, Mod: true}))

	// Not one byte was written anywhere: no file of the mod, no root, no link.
	g.Expect(fileSystem.written).To(BeEmpty())
	g.Expect(fileSystem.symlinks).To(BeEmpty())
	g.Expect(fileSystem.dirs["/home/joe/.claude/engram"]).To(BeFalse())
	g.Expect(fileSystem.dirs["/home/joe/.claude/skills"]).To(BeFalse())
}

func Test793d4b1h_Updater_Run_ASecondUpdateLeavesTheModAsTheFirstLeftIt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	updater, fileSystem := updaterOverCheckoutWithMod()

	_, firstErr := updater.Run(context.Background(), update.Options{})
	g.Expect(firstErr).NotTo(HaveOccurred())

	filesAfterFirst := maps.Clone(fileSystem.written)
	linksAfterFirst := maps.Clone(fileSystem.symlinks)

	g.Expect(filesAfterFirst).To(HaveKey("/home/joe/.claude/engram/skills/engram-memory/hooks/register.js"))
	g.Expect(linksAfterFirst).To(HaveKey("/home/joe/.claude/skills/engram-memory"))

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.written).To(Equal(filesAfterFirst))
	g.Expect(fileSystem.symlinks).To(Equal(linksAfterFirst))
	g.Expect(report.Harnesses[0].SkillDirs).To(ConsistOf(
		update.SkillDirCount{Name: "learn", Files: 1},
		update.SkillDirCount{Name: "engram-memory", Files: 1, Mod: true},
	))
}

// updaterOverCheckoutWithMod returns an updater over an in-memory machine with Claude Code and a checkout that holds
// the skill learn and the mod engram-memory, and the file system it works on.
func updaterOverCheckoutWithMod() (*update.Updater, *memFS) {
	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/repo"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")
	fileSystem.files["/repo/agent-instructions/mods/engram-memory/hooks/register.js"] = []byte("the mod")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: "/home/joe", cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	return updater, fileSystem
}
