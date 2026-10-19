package cli_test

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func Test793d4b1f_WriteUpdateReport_ListsAModFromTheModsDirectory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:       "/home/joe",
		Source:     update.SourceInfo{Mode: update.SourceLocal, Root: "/home/joe/src/engram"},
		GoInstall:  "go install ./cmd/engram/",
		BinaryPath: "/home/joe/go/bin/engram",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
				SkillDirs: []update.SkillDirCount{
					{Name: "learn", Files: 3},
					{Name: "engram-memory", Files: 3, Mod: true},
				},
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)"))
	g.Expect(out).To(ContainSubstring(
		"agent-instructions/mods/engram-memory/ → ~/.claude/skills/engram-memory/  (3 files)"))
	g.Expect(out).NotTo(ContainSubstring("agent-instructions/skills/engram-memory/"))
}

func Test855d4w1a_WriteUpdateReport_TellsThatOpenSessionsMustBeRestartedToLoadTheMod(t *testing.T) {
	t.Parallel()

	modReport := func(dryRun bool) update.Report {
		return update.Report{
			Home:   "/home/joe",
			DryRun: dryRun,
			Source: update.SourceInfo{Mode: update.SourceLocal, Root: "/home/joe/src/engram"},
			Harnesses: []update.HarnessReport{{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
				SkillDirs:  []update.SkillDirCount{{Name: "engram-memory", Files: 3, Mod: true}},
			}},
		}
	}

	const notice = "restart open sessions to load the mod engram-memory"

	t.Run("a deployed mod", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		var buffer bytes.Buffer

		g.Expect(cli.ExportWriteUpdateReport(&buffer, modReport(false))).To(Succeed())
		g.Expect(buffer.String()).To(ContainSubstring(notice))
		g.Expect(bytes.Count(buffer.Bytes(), []byte("restart open sessions"))).To(Equal(1), "one line")
	})

	t.Run("a dry run deploys nothing and says nothing", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		var buffer bytes.Buffer

		g.Expect(cli.ExportWriteUpdateReport(&buffer, modReport(true))).To(Succeed())
		g.Expect(buffer.String()).NotTo(ContainSubstring("restart open sessions"))
	})

	t.Run("a harness with only skills says nothing", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		report := modReport(false)
		report.Harnesses[0].SkillDirs = []update.SkillDirCount{{Name: "learn", Files: 3}}

		var buffer bytes.Buffer

		g.Expect(cli.ExportWriteUpdateReport(&buffer, report)).To(Succeed())
		g.Expect(buffer.String()).NotTo(ContainSubstring("restart open sessions"))
	})
}
