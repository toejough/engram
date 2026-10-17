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
