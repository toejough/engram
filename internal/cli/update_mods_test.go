package cli_test

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func Test793e8e3a_WriteUpdateReport_ListsTheModBesideTheSkillsItIsDeployedWith(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home: "/home/joe",
		Harnesses: []update.HarnessReport{{
			Name:       update.HarnessClaude,
			ProbeRoot:  ".claude",
			SkillsRoot: "/home/joe/.claude/skills",
			SkillDirs: []update.SkillDirCount{
				{Name: "learn", Files: 3},
				{Name: "engram-memory", Files: 5, Mod: true},
			},
		}},
	}

	var buffer bytes.Buffer

	g.Expect(cli.ExportWriteUpdateReport(&buffer, report)).To(Succeed())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)"))
	g.Expect(out).To(ContainSubstring(
		"agent-instructions/mods/engram-memory/ → ~/.claude/skills/engram-memory/  (5 files)"))
	g.Expect(out).NotTo(ContainSubstring("agent-instructions/skills/engram-memory/"))
	g.Expect(out).NotTo(ContainSubstring("mods:"), "no notice without a ModNotice")
}

func Test793e8e3b_WriteUpdateReport_ReportsWhyTheModMayNotRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notice := "Claude Code 2.1.100 is older than 2.1.287, the first release that loads mods"
	report := update.Report{
		Home: "/home/joe",
		Harnesses: []update.HarnessReport{{
			Name:       update.HarnessClaude,
			ProbeRoot:  ".claude",
			SkillsRoot: "/home/joe/.claude/skills",
			SkillDirs:  []update.SkillDirCount{{Name: "engram-memory", Files: 5, Mod: true}},
			ModNotice:  notice,
		}},
	}

	var buffer bytes.Buffer

	g.Expect(cli.ExportWriteUpdateReport(&buffer, report)).To(Succeed())
	g.Expect(buffer.String()).To(ContainSubstring("    mods: " + notice + "\n"))
}
