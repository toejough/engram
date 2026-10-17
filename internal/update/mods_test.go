package update_test

import (
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
