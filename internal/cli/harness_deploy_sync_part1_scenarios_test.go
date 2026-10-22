package cli_test

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792b29c17a9_WriteUpdateReport_SkillRowsMapSourceToSkillsDirectory verifies 792b29c17a9: for each harness, a
// skill of 3 files and a skill of 1 file are reported as one row each, from the source directory to that harness's
// skills directory, with the file count (docs/specs/technical/harness-deploy-sync.md, 792b29c17a).
func Test792b29c17a9_WriteUpdateReport_SkillRowsMapSourceToSkillsDirectory(t *testing.T) {
	t.Parallel()

	skills := []update.SkillDirCount{{Name: "learn", Files: 3}, {Name: "recall", Files: 1}}

	table := []struct {
		name    string
		harness update.HarnessReport
		want    []string
	}{
		{
			name:    "Claude Code",
			harness: claudeHarnessReport(),
			want: []string{
				"    agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)",
				"    agent-instructions/skills/recall/ → ~/.claude/skills/recall/  (1 file)",
			},
		},
		{
			name:    "Pi",
			harness: piHarnessReport(),
			want: []string{
				"    agent-instructions/skills/learn/ → ~/.pi/agent/skills/learn/  (3 files)",
				"    agent-instructions/skills/recall/ → ~/.pi/agent/skills/recall/  (1 file)",
			},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			harness := tc.harness
			harness.SkillDirs = skills

			lines := hds1ReportLines(t, update.Report{
				Home:      "/home/joe",
				Harnesses: []update.HarnessReport{harness},
			})

			rows := make([]string, 0, len(tc.want))

			for _, line := range lines {
				if strings.HasPrefix(line, "    agent-instructions/") {
					rows = append(rows, line)
				}
			}

			g.Expect(rows).To(Equal(tc.want))
		})
	}
}

// Test792b29c18a10_WriteUpdateReport_DryRunPrefixesTheGuidanceLines verifies 792b29c18a10: in a dry run, the line
// reporting a deployed, not imported file begins "[dry-run] guidance deployed to"
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a10_WriteUpdateReport_DryRunPrefixesTheGuidanceLines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		DryRun:    true,
		Home:      "/home/joe",
		Harnesses: []update.HarnessReport{claudeHarnessReport("recall.md")},
	})

	guidanceLines := hds1LinesHolding(lines, "guidance deployed to")

	g.Expect(guidanceLines).To(HaveLen(1))

	if len(guidanceLines) != 1 {
		return
	}

	g.Expect(guidanceLines[0]).To(HavePrefix("[dry-run] guidance deployed to " + hds1ClaudeDir + "/recall.md"))
}

// Test792b29c18a2_WriteUpdateReport_ImportedClaudeCodeFileIsRefreshed verifies 792b29c18a2: a Claude Code recall.md
// that is deployed and imported is reported refreshed, with no line to add its import
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a2_WriteUpdateReport_ImportedClaudeCodeFileIsRefreshed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:             "/home/joe",
		GuidanceImported: true,
		GuidanceImports:  map[update.Harness]map[string]bool{update.HarnessClaude: {"recall.md": true}},
		Harnesses:        []update.HarnessReport{claudeHarnessReport("recall.md")},
	})

	g.Expect(lines).To(ContainElement("guidance refreshed: " + hds1ClaudeDir + "/recall.md"))
	g.Expect(hds1LinesHolding(lines, "add '@"+hds1ClaudeDir+"/recall.md'")).To(BeEmpty())
}

// Test792b29c18a3_WriteUpdateReport_PiFileNotImportedIsDeployedWithPiPaths verifies 792b29c18a3: a Pi recall.md that
// is deployed and not imported is reported with Pi's guidance directory, its AGENTS.md and the Pi approval wording
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a3_WriteUpdateReport_PiFileNotImportedIsDeployedWithPiPaths(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:      "/home/joe",
		Harnesses: []update.HarnessReport{piHarnessReport("recall.md")},
	})

	g.Expect(lines).To(ContainElement(
		"guidance deployed to " + hds1PiDir + "/recall.md — add '@" + hds1PiDir + "/recall.md' to " +
			hds1PiConfig + " to activate it (Pi will ask you to approve the import once)",
	))
}

// Test792b29c18a4_WriteUpdateReport_ImportedPiFileIsRefreshed verifies 792b29c18a4: a Pi recall.md that is deployed
// and imported is reported refreshed with Pi's path, and no line to add anything
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a4_WriteUpdateReport_ImportedPiFileIsRefreshed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:             "/home/joe",
		GuidanceImported: true,
		GuidanceImports:  map[update.Harness]map[string]bool{update.HarnessPi: {"recall.md": true}},
		Harnesses:        []update.HarnessReport{piHarnessReport("recall.md")},
	})

	g.Expect(lines).To(ContainElement("guidance refreshed: " + hds1PiDir + "/recall.md"))
	g.Expect(hds1LinesHolding(lines, "add '@")).To(BeEmpty())
}

// Test792b29c18a5_WriteUpdateReport_EachHarnessIsJudgedOnItsOwnImports verifies 792b29c18a5: with recall.md deployed
// to both harnesses and imported only by Claude Code, Claude Code's file is refreshed and Pi's gets the line to add
// to its own AGENTS.md (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a5_WriteUpdateReport_EachHarnessIsJudgedOnItsOwnImports(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:             "/home/joe",
		GuidanceImported: true,
		GuidanceImports:  map[update.Harness]map[string]bool{update.HarnessClaude: {"recall.md": true}},
		Harnesses:        []update.HarnessReport{claudeHarnessReport("recall.md"), piHarnessReport("recall.md")},
	})

	g.Expect(lines).To(ContainElement("guidance refreshed: " + hds1ClaudeDir + "/recall.md"))
	g.Expect(hds1LinesHolding(lines, "add '@"+hds1PiDir+"/recall.md' to "+hds1PiConfig)).To(HaveLen(1))
	g.Expect(hds1LinesHolding(lines, "add '@"+hds1ClaudeDir+"/recall.md'")).To(BeEmpty())
}

// Test792b29c18a6_WriteUpdateReport_PlainUpdateWithNothingDeployedOrImportedGivesTheNudge verifies 792b29c18a6: a
// Claude Code report with no guidance deployed, no --with-guidance and no import holds the one nudge line
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a6_WriteUpdateReport_PlainUpdateWithNothingDeployedOrImportedGivesTheNudge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:      "/home/joe",
		Harnesses: []update.HarnessReport{claudeHarnessReport()},
	})

	g.Expect(hds1LinesHolding(lines, "engram ships recall- and delegation-firing guidance")).To(Equal([]string{hds1Nudge}))
}

// Test792b29c18a7_WriteUpdateReport_ExistingImportSilencesTheNudge verifies 792b29c18a7: a Claude Code report with no
// guidance deployed, no --with-guidance and a guidance import holds neither the nudge nor an activation line
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a7_WriteUpdateReport_ExistingImportSilencesTheNudge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:             "/home/joe",
		GuidanceImported: true,
		Harnesses:        []update.HarnessReport{claudeHarnessReport()},
	})

	g.Expect(hds1LinesHolding(lines, "engram ships")).To(BeEmpty())
	g.Expect(hds1LinesHolding(lines, "activate it")).To(BeEmpty())
}

// Test792b29c18a8_WriteUpdateReport_EachFileIsJudgedOnItsOwnImports verifies 792b29c18a8: with recall.md and
// delegate.md deployed to Claude Code and only recall.md imported, recall.md is reported refreshed and delegate.md
// is reported deployed with its own line to add (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a8_WriteUpdateReport_EachFileIsJudgedOnItsOwnImports(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:             "/home/joe",
		GuidanceImported: true,
		GuidanceImports:  map[update.Harness]map[string]bool{update.HarnessClaude: {"recall.md": true}},
		Harnesses:        []update.HarnessReport{claudeHarnessReport("recall.md", "delegate.md")},
	})

	g.Expect(lines).To(ContainElement("guidance refreshed: " + hds1ClaudeDir + "/recall.md"))
	g.Expect(lines).To(ContainElement(
		"guidance deployed to " + hds1ClaudeDir + "/delegate.md — add '@" + hds1ClaudeDir + "/delegate.md' to " +
			hds1ClaudeConfig + " to activate it (Claude Code will ask you to approve the import once)",
	))
	g.Expect(hds1LinesHolding(lines, "add '@"+hds1ClaudeDir+"/recall.md'")).To(BeEmpty())
}

// Test792b29c18a9_WriteUpdateReport_FlagSilencesTheNudge verifies 792b29c18a9: a report from a run with
// --with-guidance in which no harness deployed guidance does not hold the nudge
// (docs/specs/technical/harness-deploy-sync.md, 792b29c18a).
func Test792b29c18a9_WriteUpdateReport_FlagSilencesTheNudge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lines := hds1ReportLines(t, update.Report{
		Home:         "/home/joe",
		WithGuidance: true,
		Harnesses:    []update.HarnessReport{claudeHarnessReport(), piHarnessReport()},
	})

	g.Expect(hds1LinesHolding(lines, "engram ships")).To(BeEmpty())
}

// unexported constants.
const (
	hds1ClaudeConfig = "~/.claude/CLAUDE.md"
	hds1ClaudeDir    = "~/.claude/engram"
	hds1Nudge        = "engram ships recall- and delegation-firing guidance; " +
		"run 'engram update --with-guidance' to deploy it"
	hds1PiConfig = "~/.pi/agent/AGENTS.md"
	hds1PiDir    = "~/.pi/agent/guidance"
)

// hds1LinesHolding returns the lines that contain needle.
func hds1LinesHolding(lines []string, needle string) []string {
	holding := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.Contains(line, needle) {
			holding = append(holding, line)
		}
	}

	return holding
}

// hds1ReportLines writes the update report and returns its lines.
func hds1ReportLines(t *testing.T, report update.Report) []string {
	t.Helper()

	var buffer bytes.Buffer

	err := cli.ExportWriteUpdateReport(&buffer, report)
	if err != nil {
		t.Fatalf("writing the update report: %v", err)
	}

	return strings.Split(strings.TrimRight(buffer.String(), "\n"), "\n")
}
