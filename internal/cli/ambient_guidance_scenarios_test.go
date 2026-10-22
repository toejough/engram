package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a19u1a_Update_AConfigWithThreeImportsGainsTheShim verifies Scenario 792a19u1a: with ~/.claude/CLAUDE.md
// importing recall.md, delegate.md and learn.md and not shim.md, the update reports shim.md deployed with the line to
// add, and the three existing imports are untouched: the config file is byte for byte as it was and the three
// documents are reported refreshed.
func Test792a19u1a_Update_AConfigWithThreeImportsGainsTheShim(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine(ag1AllDocuments...)
	config := ag1ImportLines("recall.md", "delegate.md", "learn.md")
	machine.fileSystem.seedFile(ag1ClaudeConfig, config)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(ag1ClaudeDeployed("shim.md")))
	g.Expect(ag1GuidanceLines(lines)).To(HaveLen(len(ag1AllDocuments)), "one line per deployed document")

	for _, name := range []string{"recall.md", "delegate.md", "learn.md"} {
		g.Expect(lines).To(ContainElement(ag1ClaudeRefreshed(name)))
	}

	g.Expect(machine.fileSystem.text(ag1ClaudeConfig)).To(Equal(config), "the user's config is not edited")
}

// Test792a19u1b_Update_TheShimDoesNotReplaceTheOtherDocuments verifies Scenario 792a19u1b: with the recall, learn
// and write-memory skills in the source as skills, and a config that imports the shim as well as recall.md,
// delegate.md and learn.md, the update reports all four documents refreshed (none replaced, none dropped), leaves the
// config as it was, and deploys the three skills as skills.
func Test792a19u1b_Update_TheShimDoesNotReplaceTheOtherDocuments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine(ag1AllDocuments...)
	machine.skill("learn", "SKILL.md", "the learn skill")
	machine.skill("write-memory", "SKILL.md", "the write-memory skill")

	config := ag1ImportLines(ag1AllDocuments...)
	machine.fileSystem.seedFile(ag1ClaudeConfig, config)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	for _, name := range ag1AllDocuments {
		g.Expect(lines).To(ContainElement(ag1ClaudeRefreshed(name)))
	}

	g.Expect(ag1GuidanceLines(lines)).To(HaveLen(len(ag1AllDocuments)), "no document is added or dropped")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("guidance deployed"), "every document stays imported")
	g.Expect(machine.fileSystem.text(ag1ClaudeConfig)).To(Equal(config), "all four imports remain")

	for _, skill := range []string{"recall", "learn", "write-memory"} {
		g.Expect(machine.fileSystem.kindOf(iu2ClaudeSkills+"/"+skill)).To(Equal("link"), "the %s skill ships", skill)
	}
}

// Test792a19v1a_Update_AClaudeCodeDocumentNotYetImportedIsDeployedWithItsLineToAdd verifies Scenario 792a19v1a:
// with --with-guidance deploying recall.md for Claude Code and ~/.claude/CLAUDE.md not importing it, the report
// carries the deployed line with the line to add, the config file to edit and the one-time approval.
func Test792a19v1a_Update_AClaudeCodeDocumentNotYetImportedIsDeployedWithItsLineToAdd(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md")

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(HavePrefix(
		"guidance deployed to ~/.claude/engram/recall.md — add '@~/.claude/engram/recall.md' to ~/.claude/CLAUDE.md " +
			"to activate it (Claude Code will ask you to approve the import once)",
	)))
	g.Expect(lines).To(ContainElement(ag1ClaudeDeployed("recall.md")))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("guidance refreshed"))
}

// Test792a19v1b_Update_AnImportedDocumentIsRefreshed verifies Scenario 792a19v1b: with ~/.claude/CLAUDE.md
// importing recall.md, the report says "guidance refreshed: ~/.claude/engram/recall.md" and gives no line to add.
func Test792a19v1b_Update_AnImportedDocumentIsRefreshed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md")
	machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(HavePrefix("guidance refreshed: ~/.claude/engram/recall.md")))
	g.Expect(ag1GuidanceLines(lines)).To(HaveLen(1))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("add '@"), "no line to add")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("guidance deployed"))
}

// Test792a19v1c_Update_APiDocumentIsDeployedWithPisPaths verifies Scenario 792a19v1c: with --with-guidance deploying
// recall.md for Pi and ~/.pi/agent/AGENTS.md not importing it, the report carries Pi's deployed line with Pi's guidance
// directory, Pi's config file and Pi's one-time approval, and no Claude Code path.
func Test792a19v1c_Update_APiDocumentIsDeployedWithPisPaths(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md")
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	piLines := iu3LinesWithPrefix(lines, "guidance deployed to ~/.pi/")

	g.Expect(piLines).To(Equal([]string{ag1PiDeployed("recall.md")}))

	for _, line := range piLines {
		g.Expect(line).NotTo(ContainSubstring(".claude"))
	}
}

// Test792a19v1d_Update_TheHarnessesAreWiredIndependently verifies Scenario 792a19v1d: with Claude Code importing
// recall.md and Pi not, the report says refreshed for Claude Code and gives Pi's line to add, and gives no line to add
// for Claude Code.
func Test792a19v1d_Update_TheHarnessesAreWiredIndependently(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md")
	machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(ag1ClaudeRefreshed("recall.md")))
	g.Expect(lines).To(ContainElement(ag1PiDeployed("recall.md")))
	g.Expect(ag1GuidanceLines(lines)).To(HaveLen(2))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("add '@~/.claude/"), "Claude Code has nothing to add")
}

// Test792a19v1e_Update_EachDocumentIsJudgedOnItsOwn verifies Scenario 792a19v1e: with Claude Code importing
// recall.md and not delegate.md, recall.md is reported refreshed and delegate.md is reported deployed with its line to
// add.
func Test792a19v1e_Update_EachDocumentIsJudgedOnItsOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md", "delegate.md")
	machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(ag1ClaudeRefreshed("recall.md")))
	g.Expect(lines).To(ContainElement(ag1ClaudeDeployed("delegate.md")))
	g.Expect(ag1GuidanceLines(lines)).To(HaveLen(2))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("add '@~/.claude/engram/recall.md'"))
}

// Test792a19v1f_Update_AUserWhoHasNotOptedInIsNudged verifies Scenario 792a19v1f: with no --with-guidance, no
// guidance imported and no guidance deployed, the report prints the nudge once, naming engram update --with-guidance.
func Test792a19v1f_Update_AUserWhoHasNotOptedInIsNudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine(ag1AllDocuments...)
	machine.fileSystem.seedFile(ag1ClaudeConfig, "# Notes\n\nNo imports here.\n")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)

	g.Expect(lines).To(ContainElement(ag1Nudge))
	g.Expect(iu3LinesWithPrefix(lines, "engram ships")).To(HaveLen(1))
	g.Expect(ag1GuidanceLines(lines)).To(BeEmpty(), "no guidance was deployed")
}

// Test792a19v1g_Update_AUserWhoAlreadyImportsIsNotNudged verifies Scenario 792a19v1g: with no --with-guidance flag
// and a config that imports at least one guidance document, the report prints no nudge, and each document is reported
// refreshed or deployed on its own; the config's import also keeps the nudge away when the source ships no document.
func Test792a19v1g_Update_AUserWhoAlreadyImportsIsNotNudged(t *testing.T) {
	t.Parallel()

	t.Run("each document is judged on its own", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		machine := ag1NewMachine("recall.md", "delegate.md")
		machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))

		outcome := machine.update(t, cli.UpdateArgs{})
		g.Expect(outcome.err).NotTo(HaveOccurred())

		if outcome.err != nil {
			return
		}

		lines := iu2Lines(outcome.stdout)

		g.Expect(outcome.stdout).NotTo(ContainSubstring("engram ships"))
		g.Expect(outcome.stdout).NotTo(ContainSubstring("run 'engram update --with-guidance'"))
		g.Expect(lines).To(ContainElement(ag1ClaudeRefreshed("recall.md")))
		g.Expect(lines).To(ContainElement(ag1ClaudeDeployed("delegate.md")))
	})

	t.Run("the source ships no document", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		machine := ag1NewMachine()
		machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))

		outcome := machine.update(t, cli.UpdateArgs{})
		g.Expect(outcome.err).NotTo(HaveOccurred())

		if outcome.err != nil {
			return
		}

		g.Expect(ag1GuidanceLines(iu2Lines(outcome.stdout))).To(BeEmpty())
		g.Expect(outcome.stdout).NotTo(ContainSubstring("engram ships"))
		g.Expect(outcome.stdout).NotTo(ContainSubstring("run 'engram update --with-guidance'"))
	})
}

// Test792a19v1h_Update_TheFlagSuppressesTheNudge verifies Scenario 792a19v1h: with --with-guidance, nothing imported
// and no guidance document in the source, so none deployed for any harness, the report does not print the nudge.
func Test792a19v1h_Update_TheFlagSuppressesTheNudge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine()
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(ag1GuidanceLines(iu2Lines(outcome.stdout))).To(BeEmpty(), "no guidance deployed for any harness")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("engram ships"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("run 'engram update --with-guidance'"))
}

// Test792a19v1i_Update_ADryRunPrefixesTheGuidanceLines verifies Scenario 792a19v1i: a dry run with --with-guidance
// for Claude Code and Pi, with one document imported and one not, begins every guidance line with "[dry-run] ", and
// changes nothing.
func Test792a19v1i_Update_ADryRunPrefixesTheGuidanceLines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := ag1NewMachine("recall.md", "delegate.md")
	machine.fileSystem.seedFile(ag1ClaudeConfig, ag1ImportLines("recall.md"))
	machine.fileSystem.seedDir(iu2PiDir)

	before := machine.fileSystem.snapshot("")

	outcome := machine.update(t, cli.UpdateArgs{DryRun: true, WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	guidanceLines := ag1GuidanceLines(lines)

	g.Expect(guidanceLines).To(HaveLen(len(ag1AllHarnessLines)), "refreshed and deployed lines of both harnesses")

	for _, want := range ag1AllHarnessLines {
		g.Expect(lines).To(ContainElement(HavePrefix("[dry-run] " + want)))
	}

	for _, line := range lines {
		if strings.Contains(line, "guidance refreshed") || strings.Contains(line, "guidance deployed") {
			g.Expect(line).To(HavePrefix("[dry-run] "), "guidance line %q", line)
		}
	}

	g.Expect(machine.fileSystem.snapshot("")).To(Equal(before), "a dry run changes nothing")
}

// unexported constants.
const (
	ag1ClaudeConfig = iu2ClaudeDir + "/CLAUDE.md"
	ag1Nudge        = "engram ships recall- and delegation-firing guidance; run 'engram update --with-guidance' " +
		"to deploy it"
)

// unexported variables.
var (
	// ag1AllDocuments is the set of guidance documents engram expects a config to import (792a19u).
	ag1AllDocuments = []string{"recall.md", "delegate.md", "learn.md", "shim.md"}
	// ag1AllHarnessLines are the guidance lines (without the dry-run prefix) of scenario 792a19v1i.
	ag1AllHarnessLines = []string{
		"guidance refreshed: ~/.claude/engram/recall.md",
		"guidance deployed to ~/.claude/engram/delegate.md",
		"guidance deployed to ~/.pi/agent/guidance/recall.md",
		"guidance deployed to ~/.pi/agent/guidance/delegate.md",
	}
)

// ag1ClaudeDeployed is the report line of a Claude Code document that is deployed and not imported.
func ag1ClaudeDeployed(name string) string {
	return "guidance deployed to ~/.claude/engram/" + name + " — add '@~/.claude/engram/" + name +
		"' to ~/.claude/CLAUDE.md to activate it (Claude Code will ask you to approve the import once)" +
		" (canonical: ~/.claude/engram/guidance/" + name + ")"
}

// ag1ClaudeRefreshed is the report line of a Claude Code document that is imported.
func ag1ClaudeRefreshed(name string) string {
	return "guidance refreshed: ~/.claude/engram/" + name + " (canonical: ~/.claude/engram/guidance/" + name + ")"
}

// ag1GuidanceLines returns the lines of a report that state a guidance document, deployed or refreshed.
func ag1GuidanceLines(lines []string) []string {
	matching := []string{}

	for _, line := range lines {
		trimmed := strings.TrimPrefix(line, "[dry-run] ")
		if strings.HasPrefix(trimmed, "guidance deployed to ") || strings.HasPrefix(trimmed, "guidance refreshed: ") {
			matching = append(matching, line)
		}
	}

	return matching
}

// ag1ImportLines is the text of a config file that imports each document from Claude Code's engram root.
func ag1ImportLines(names ...string) string {
	var text strings.Builder

	for _, name := range names {
		text.WriteString("@~/.claude/engram/" + name + "\n")
	}

	return text.String()
}

// ag1NewMachine is a machine with Claude Code, the recall skill and the given guidance documents in its source.
func ag1NewMachine(documents ...string) *iu2Machine {
	machine := iu3NewMachine()

	for _, name := range documents {
		machine.guidance(name, "the "+name+" guidance")
	}

	return machine
}

// ag1PiDeployed is the report line of a Pi document that is deployed and not imported.
func ag1PiDeployed(name string) string {
	return "guidance deployed to ~/.pi/agent/guidance/" + name + " — add '@~/.pi/agent/guidance/" + name +
		"' to ~/.pi/agent/AGENTS.md to activate it (Pi will ask you to approve the import once)"
}
