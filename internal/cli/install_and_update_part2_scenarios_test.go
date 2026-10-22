package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792a11f1a_Update_ADryRunOnANeverSyncedHarnessLeavesNoTrace verifies Scenario 792a11f1a: with no engram root
// under ~/.claude, a dry run lists the skill mapping unprefixed between the "[dry-run] engram update" title and the
// "[dry-run] installed:" line, and afterwards no engram root, marker, link or file exists.
func Test792a11f1a_Update_ADryRunOnANeverSyncedHarnessLeavesNoTrace(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")

	before := machine.fileSystem.snapshot("")

	outcome := machine.update(t, cli.UpdateArgs{DryRun: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	title := slices.Index(lines, "[dry-run] engram update")
	mapping := slices.Index(lines, "    agent-instructions/skills/recall/ → ~/.claude/skills/recall/  (1 file)")
	installed := slices.Index(lines, "[dry-run] installed: Claude Code")

	g.Expect(title).To(Equal(0), "the title carries the prefix")
	g.Expect(mapping).To(BeNumerically(">", title), "the mapping is listed under the title, unprefixed")
	g.Expect(installed).To(BeNumerically(">", mapping), "the mapping is listed above the installed line")

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot)).To(BeEmpty(), "no engram root")
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/"+iu2Marker)).To(BeEmpty(), "no marker")
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills+"/recall")).To(BeEmpty(), "no link")
	g.Expect(fileSystem.snapshot("")).To(Equal(before), "no file, directory or link changed anywhere")
}

// Test792a11f1b_Update_ActionLinesCarryThePrefixNoticesDoNot verifies Scenario 792a11f1b: a dry run that would
// delete a root file, adopt a copied skill, remove a dangling link and refresh guidance, and that finds an unmanaged
// entry, begins each action line with "[dry-run] " and prints the unmanaged line without it.
func Test792a11f1b_Update_ActionLinesCarryThePrefixNoticesDoNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.guidance("recall.md", "guidance")

	fileSystem := machine.fileSystem
	// Claude Code: an owned root with a stale file, a copied skill, a dangling engram link and a guidance import.
	fileSystem.seedFile(iu2ClaudeRoot+"/"+iu2Marker, "")
	fileSystem.seedFile(iu2ClaudeRoot+"/skills/old/SKILL.md", "stale")
	fileSystem.seedFile(iu2ClaudeSkills+"/recall/SKILL.md", "an earlier copy")
	fileSystem.seedLink(iu2ClaudeSkills+"/gone", iu2ClaudeRoot+"/skills/gone")
	fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", "@~/.claude/engram/recall.md\n")
	// Pi: a first update that finds a skill the user installed.
	fileSystem.seedFile(iu2PiSkills+"/mine/SKILL.md", "mine")

	before := fileSystem.snapshot("")

	outcome := machine.update(t, cli.UpdateArgs{DryRun: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	actionLines := []string{
		"    [dry-run] engram root: deleted skills/old/SKILL.md",
		"    [dry-run] adopted: ~/.claude/skills/recall",
		"    [dry-run] cleanup: removed dangling link ~/.claude/skills/gone",
		"[dry-run] guidance refreshed: ~/.claude/engram/recall.md (canonical: ~/.claude/engram/guidance/recall.md)",
	}

	for _, want := range actionLines {
		g.Expect(lines).To(ContainElement(want))
	}

	g.Expect(lines).To(ContainElement("    unmanaged (left alone): ~/.pi/agent/skills/mine"), "the notice is unprefixed")

	actionMarkers := []string{"engram root: deleted", "adopted:", "cleanup: removed", "guidance ", "installed:"}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if iu2ContainsAny(trimmed, actionMarkers) {
			g.Expect(trimmed).To(HavePrefix("[dry-run] "), "action line %q", line)
		}
	}

	g.Expect(outcome.stdout).NotTo(ContainSubstring("[dry-run] unmanaged"))
	g.Expect(fileSystem.snapshot("")).To(Equal(before), "a dry run changes nothing")
}

// Test792a11f1c_Update_ADryRunInstallsNothing verifies Scenario 792a11f1c: with a local clone newer than the
// installed binary, a dry run leaves the installed binary unchanged and its binary line shows the planned install
// command without "ok".
func Test792a11f1c_Update_ADryRunInstallsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2Binary, "the installed binary")
	machine.cmd.answers = append(machine.cmd.answers, iu1Answer{
		prefix: "go version", stdout: iu1BinaryInfo + "\tbuild\tvcs.revision=" + iu1InstalledRevision + "\n",
	})

	outcome := machine.update(t, cli.UpdateArgs{DryRun: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty(), "nothing is installed")
	g.Expect(machine.spawn.spawned()).To(BeEmpty(), "the installed binary is not started")
	g.Expect(machine.fileSystem.text(iu2Binary)).To(Equal("the installed binary"), "the installed binary is unchanged")
	g.Expect(iu1BinaryLine(outcome.stdout)).To(Equal("  binary: go install ./cmd/engram/"))
	g.Expect(iu1BinaryLine(outcome.stdout)).NotTo(ContainSubstring("ok"))
}

// Test792a11g1a_Update_OnlyTheHarnessesPresentAreUpdated verifies Scenario 792a11g1a: with ~/.claude present and
// ~/.pi absent the report has a Claude Code section and no Pi section, and ~/.pi is not created.
func Test792a11g1a_Update_OnlyTheHarnessesPresentAreUpdated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(iu2Section(outcome.stdout, iu2ClaudeHeader)).NotTo(BeEmpty(), "the report has a Claude Code section")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("  Pi ("), "the report has no Pi section")
	g.Expect(outcome.stdout).To(ContainSubstring("installed: Claude Code\n"))
	g.Expect(machine.fileSystem.kindOf(iu2PiDir)).To(BeEmpty(), "~/.pi is not created")
	g.Expect(machine.fileSystem.snapshot(iu2PiDir)).To(BeEmpty(), "nothing is written under ~/.pi")
}

// Test792a11g1b_Update_BothHarnessesAreUpdatedInAStableOrder verifies Scenario 792a11g1b: with ~/.claude and ~/.pi
// both present the report lists Claude Code then Pi, and the installed line names both.
func Test792a11g1b_Update_BothHarnessesAreUpdatedInAStableOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	claude := slices.Index(lines, iu2ClaudeHeader)
	pi := slices.Index(lines, iu2PiHeader)

	g.Expect(claude).To(BeNumerically(">=", 0), "the report has a Claude Code section")
	g.Expect(pi).To(BeNumerically(">", claude), "Pi is listed after Claude Code")
	g.Expect(lines).To(ContainElement("installed: Claude Code, Pi"))
	g.Expect(machine.fileSystem.kindOf(iu2PiSkills+"/recall")).To(Equal("link"), "Pi is updated too")
}

// Test792a11g1c_Update_NoHarnessIsAnErrorBeforeAnyInstall verifies Scenario 792a11g1c: with neither ~/.claude nor
// ~/.pi the update fails saying no supported harness was found at ~/.claude/ or ~/.pi/, and nothing was installed.
func Test792a11g1c_Update_NoHarnessIsAnErrorBeforeAnyInstall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.removeAll(iu2ClaudeDir)

	before := machine.fileSystem.snapshot("")

	outcome := machine.update(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(update.ErrNoHarness))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("no supported harness found at ~/.claude/ or ~/.pi/")))
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty(), "nothing was installed")
	g.Expect(machine.spawn.spawned()).To(BeEmpty(), "nothing was handed off")
	g.Expect(outcome.stdout).To(BeEmpty())
	g.Expect(machine.fileSystem.snapshot("")).To(Equal(before), "nothing was written")
}

// Test792a11g1d_Update_OneFailingHarnessDoesNotStopTheOther verifies Scenario 792a11g1d: with writing under ~/.claude
// failing and ~/.pi healthy, Pi is updated, the Claude Code section shows "error: <reason>", the installed line names
// only Pi, and the exit status is non-zero.
func Test792a11g1d_Update_OneFailingHarnessDoesNotStopTheOther(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedDir(iu2PiDir)
	machine.fileSystem.denyWritesUnder(iu2ClaudeDir)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred(), "the parent hands the exit status of the update on")

	if outcome.err != nil {
		return
	}

	claude := iu2Section(outcome.stdout, iu2ClaudeHeader)
	g.Expect(claude).To(HaveLen(1), "the Claude Code section holds only its error")

	if len(claude) != 1 {
		return
	}

	g.Expect(claude[0]).To(HavePrefix("    error: "))
	g.Expect(claude[0]).To(ContainSubstring("permission denied"))

	g.Expect(iu2Section(outcome.stdout, iu2PiHeader)).To(ContainElement(
		"    agent-instructions/skills/recall/ → ~/.pi/agent/skills/recall/  (1 file)",
	))
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement("installed: Pi"), "the installed line names only Pi")
	g.Expect(machine.fileSystem.kindOf(iu2PiSkills+"/recall")).To(Equal("link"), "Pi is updated")
	g.Expect(outcome.exits).To(Equal([]int{1}), "the exit status is non-zero")
	g.Expect(outcome.childErrs).To(HaveLen(1))

	if len(outcome.childErrs) != 1 {
		return
	}

	g.Expect(outcome.childErrs[0]).To(MatchError(ContainSubstring("one or more detected harnesses failed")))
}

// Test792a11g1e_Update_AMissingSkillsSourceIsAnError verifies Scenario 792a11g1e: with no
// agent-instructions/skills directory in the source checkout the update fails with an error that the skills source
// directory is missing.
func Test792a11g1e_Update_AMissingSkillsSourceIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()

	before := machine.fileSystem.snapshot("")

	outcome := machine.update(t, cli.UpdateArgs{})

	g.Expect(outcome.err).NotTo(HaveOccurred(), "the parent hands the exit status of the update on")
	g.Expect(outcome.childErrs).To(HaveLen(1))

	if len(outcome.childErrs) != 1 {
		return
	}

	g.Expect(outcome.childErrs[0]).To(MatchError(update.ErrSkillsSrcMissing))
	g.Expect(outcome.childErrs[0]).To(MatchError(ContainSubstring("skills source dir missing")))
	g.Expect(outcome.exits).To(Equal([]int{1}), "the exit status is non-zero")
	g.Expect(machine.fileSystem.snapshot("")).To(Equal(before), "nothing is deployed")
}

// Test792a11h1a_Update_ASkillDeploysAsADirectoryLink verifies Scenario 792a11h1a: with skill recall in the source,
// ~/.claude/skills/recall is a symbolic link to the skills/recall directory of ~/.claude/engram, and its SKILL.md is
// readable through the link.
func Test792a11h1a_Update_ASkillDeploysAsADirectoryLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "the recall skill")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills + "/recall")).To(Equal("link"))
	g.Expect(fileSystem.linkTarget(iu2ClaudeSkills + "/recall")).To(Equal(iu2ClaudeRoot + "/skills/recall"))
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/skills/recall")).To(Equal("dir"), "the content lives in the root")
	g.Expect(fileSystem.text(iu2ClaudeSkills + "/recall/SKILL.md")).To(Equal("the recall skill"))
}

// Test792a11h1b_Update_AddedAndChangedSourcesDeploy verifies Scenario 792a11h1b: after a new skill and an edited file
// in an existing skill since the last update, the new skill appears in the engram root with a link from the harness,
// and the edited file in the root matches the source.
func Test792a11h1b_Update_AddedAndChangedSourcesDeploy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall, first version")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())

	machine.skill("recall", "SKILL.md", "recall, edited")
	machine.skill("learn", "SKILL.md", "learn, new")

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.text(iu2ClaudeRoot + "/skills/learn/SKILL.md")).To(Equal("learn, new"))
	g.Expect(fileSystem.linkTarget(iu2ClaudeSkills + "/learn")).To(Equal(iu2ClaudeRoot + "/skills/learn"))
	g.Expect(fileSystem.text(iu2ClaudeRoot + "/skills/recall/SKILL.md")).To(Equal("recall, edited"))
}

// Test792a11h1c_Update_ALinkThatPointsElsewhereIsRepointed verifies Scenario 792a11h1c: with
// ~/.claude/skills/recall a symbolic link to a different directory, it becomes a link to the engram root's
// skills/recall.
func Test792a11h1c_Update_ALinkThatPointsElsewhereIsRepointed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile("/elsewhere/recall/SKILL.md", "not engram's")
	machine.fileSystem.seedLink(iu2ClaudeSkills+"/recall", "/elsewhere/recall")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills + "/recall")).To(Equal("link"))
	g.Expect(fileSystem.linkTarget(iu2ClaudeSkills + "/recall")).To(Equal(iu2ClaudeRoot + "/skills/recall"))
	g.Expect(fileSystem.text("/elsewhere/recall/SKILL.md")).To(Equal("not engram's"), "the old target is untouched")
}

// Test792a11h1d_Update_TheReportListsEachSkillAndItsFileCount verifies Scenario 792a11h1d: with a source skill learn
// of 3 files and recall of 1 the report lists each as
// "agent-instructions/skills/<skill>/ → ~/.claude/skills/<skill>/  (<N> file[s])".
func Test792a11h1d_Update_TheReportListsEachSkillAndItsFileCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("learn", "SKILL.md", "learn")
	machine.skill("learn", "references/one.md", "one")
	machine.skill("learn", "references/two.md", "two")
	machine.skill("recall", "SKILL.md", "recall")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	claude := iu2Section(outcome.stdout, iu2ClaudeHeader)
	g.Expect(claude).To(ContainElement("    agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)"))
	g.Expect(claude).To(ContainElement("    agent-instructions/skills/recall/ → ~/.claude/skills/recall/  (1 file)"))
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement("installed: Claude Code"))
}

// Test792a11h1e_Update_ReRunningChangesNothing verifies Scenario 792a11h1e: after an update has completed against
// the same source, running it again leaves every root file and link as it was after the first run.
func Test792a11h1e_Update_ReRunningChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("learn", "SKILL.md", "learn")
	machine.skill("learn", "references/one.md", "one")
	machine.skill("recall", "SKILL.md", "recall")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())

	afterFirst := machine.fileSystem.snapshot(iu2ClaudeDir)

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	g.Expect(afterFirst).NotTo(BeEmpty())
	g.Expect(machine.fileSystem.snapshot(iu2ClaudeDir)).To(Equal(afterFirst))
	g.Expect(second.stdout).NotTo(ContainSubstring("deleted"))
	g.Expect(second.stdout).NotTo(ContainSubstring("adopted"))
	g.Expect(second.stdout).NotTo(ContainSubstring("cleanup"))
}

// Test792a11i1a_Update_ARemovedSkillDisappears verifies Scenario 792a11i1a: when a deployed skill is deleted from the
// source, the next update deletes its subtree from the engram root, the link in the harness skills directory is
// gone, and the report lists the deletion.
func Test792a11i1a_Update_ARemovedSkillDisappears(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("learn", "SKILL.md", "learn")
	machine.skill("recall", "SKILL.md", "recall")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())

	machine.fileSystem.removeAll(iu2Source + "/skills/learn")

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/skills/learn")).To(BeEmpty(), "the subtree is deleted from the root")
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills+"/learn")).To(BeEmpty(), "the link is gone in the same run")
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills+"/recall")).To(Equal("link"), "the kept skill stays linked")
	g.Expect(iu2Section(second.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    engram root: deleted skills/learn/SKILL.md",
	))
}

// Test792a11i1b_Update_ARemovedFileInsideAKeptSkillDisappears verifies Scenario 792a11i1b: when a file is deleted
// from a skill that is still in the source, the next update deletes the file and any directory it leaves empty from
// the engram root.
func Test792a11i1b_Update_ARemovedFileInsideAKeptSkillDisappears(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("learn", "SKILL.md", "learn")
	machine.skill("learn", "references/one.md", "one")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())

	machine.fileSystem.removeAll(iu2Source + "/skills/learn/references")

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/skills/learn/references/one.md")).To(BeEmpty(), "the file is deleted")
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/skills/learn/references")).To(BeEmpty(), "the empty directory is deleted")
	g.Expect(fileSystem.text(iu2ClaudeRoot+"/skills/learn/SKILL.md")).To(Equal("learn"), "the kept file stays")
	g.Expect(iu2Section(second.stdout, iu2ClaudeHeader)).To(ContainElements(
		"    engram root: deleted skills/learn/references",
		"    engram root: deleted skills/learn/references/one.md",
	))
}

// Test792a11j1a_Update_AnUnmarkedRootIsNotPurged verifies Scenario 792a11j1a: with the engram root present without
// the ownership marker and holding a file that is not in the deploy set, the file is left in place and listed as
// unattributable, the marker is written, and nothing is deleted in that directory.
func Test792a11j1a_Update_AnUnmarkedRootIsNotPurged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeRoot+"/notes.txt", "the user's notes")
	machine.fileSystem.seedFile(iu2ClaudeRoot+"/skills/old/SKILL.md", "not in the deploy set")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	claude := iu2Section(outcome.stdout, iu2ClaudeHeader)

	g.Expect(fileSystem.text(iu2ClaudeRoot+"/notes.txt")).To(Equal("the user's notes"), "the file is left in place")
	g.Expect(fileSystem.text(iu2ClaudeRoot + "/skills/old/SKILL.md")).To(Equal("not in the deploy set"))
	g.Expect(claude).To(ContainElement("    unattributable: notes.txt"))
	g.Expect(claude).To(ContainElement(
		"    engram root ~/.claude/engram exists without the .engram-owned marker — sync-deletion refused there",
	))
	g.Expect(fileSystem.kindOf(iu2ClaudeRoot+"/"+iu2Marker)).To(Equal("file"), "the marker is written")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("engram root: deleted"), "no deletion happens")
	g.Expect(fileSystem.text(iu2ClaudeRoot+"/skills/recall/SKILL.md")).To(Equal("recall"), "the deploy set is written")
}

// Test792a11j1b_Update_TheMarkerEnablesDeletionOnTheNextRun verifies Scenario 792a11j1b: after an update wrote the
// marker into a root that held an extra file, the next update deletes the file in the managed subtrees that is not in
// the deploy set.
func Test792a11j1b_Update_TheMarkerEnablesDeletionOnTheNextRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeRoot+"/skills/old/SKILL.md", "not in the deploy set")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())

	if first.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/skills/old/SKILL.md")).To(Equal("file"), "kept by the first run")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot + "/" + iu2Marker)).To(Equal("file"))

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if second.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/skills/old/SKILL.md")).To(BeEmpty(), "deleted by the second run")
	g.Expect(iu2Section(second.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    engram root: deleted skills/old/SKILL.md",
	))
	g.Expect(machine.fileSystem.text(iu2ClaudeRoot + "/skills/recall/SKILL.md")).To(Equal("recall"))
}

// Test792a11j1c_Update_ADanglingEngramLinkIsRemovedAndAForeignLinkKept verifies Scenario 792a11j1c: of a link into
// the engram root whose target was removed and a second link to a path outside the root, the first is deleted and
// the second is untouched.
func Test792a11j1c_Update_ADanglingEngramLinkIsRemovedAndAForeignLinkKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	// The foreign link dangles too, so that only its target outside the root keeps it.
	machine.fileSystem.seedLink(iu2ClaudeSkills+"/removed", iu2ClaudeRoot+"/skills/removed")
	machine.fileSystem.seedLink(iu2ClaudeSkills+"/foreign", "/elsewhere/missing")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills+"/removed")).To(BeEmpty(), "the dangling engram link is deleted")
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills+"/foreign")).To(Equal("link"), "the foreign link stays")
	g.Expect(fileSystem.linkTarget(iu2ClaudeSkills + "/foreign")).To(Equal("/elsewhere/missing"))
	g.Expect(iu2Section(outcome.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    cleanup: removed dangling link ~/.claude/skills/removed",
	))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("foreign"))
}

// Test792a11j1d_Update_UserOwnedRealFilesAreNeverDeleted verifies Scenario 792a11j1d: a real skill directory the user
// installed by other means in the harness skills directory is untouched by the sync, the cleanup and the migration,
// on the first update and on the next.
func Test792a11j1d_Update_UserOwnedRealFilesAreNeverDeleted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeSkills+"/mine/SKILL.md", "installed by the user")
	machine.fileSystem.seedFile(iu2ClaudeSkills+"/mine/notes/todo.md", "also the user's")

	mine := machine.fileSystem.snapshot(iu2ClaudeSkills + "/mine")

	for _, run := range []string{"first", "second"} {
		outcome := machine.update(t, cli.UpdateArgs{})
		g.Expect(outcome.err).NotTo(HaveOccurred(), run)

		if outcome.err != nil {
			return
		}

		claude := iu2Section(outcome.stdout, iu2ClaudeHeader)
		g.Expect(claude).NotTo(ContainElement(HavePrefix("    error: ")), run)
		g.Expect(machine.fileSystem.kindOf(iu2ClaudeSkills+"/mine")).To(Equal("dir"), run)
		g.Expect(machine.fileSystem.snapshot(iu2ClaudeSkills+"/mine")).To(Equal(mine), run)
	}
}

// Test792a11j1e_Update_ARootThatIsAFileIsRefused verifies Scenario 792a11j1e: with the engram root path a regular
// file, that harness reports an error that the engram root exists and is not a directory, and the file is untouched.
func Test792a11j1e_Update_ARootThatIsAFileIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeRoot, "not a directory")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred(), "the parent hands the exit status of the update on")

	if outcome.err != nil {
		return
	}

	claude := iu2Section(outcome.stdout, iu2ClaudeHeader)
	g.Expect(claude).To(HaveLen(1))

	if len(claude) != 1 {
		return
	}

	g.Expect(claude[0]).To(HavePrefix("    error: "))
	g.Expect(claude[0]).To(ContainSubstring("engram root exists and is not a directory"))
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot)).To(Equal("file"))
	g.Expect(machine.fileSystem.text(iu2ClaudeRoot)).To(Equal("not a directory"), "the file is untouched")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeSkills+"/recall")).To(BeEmpty(), "nothing is linked")
	g.Expect(outcome.exits).To(Equal([]int{1}), "the exit status is non-zero")
}

// Test792a11k1a_Update_ACopiedSkillBecomesALink verifies Scenario 792a11k1a: on the first update of a harness whose
// skills directory holds recall as a real directory from an earlier copy-mode deploy, recall's content lives in the
// engram root, the harness path is a link to it, the skill is still discoverable, and the report lists it as adopted.
func Test792a11k1a_Update_ACopiedSkillBecomesALink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall from the source")
	machine.fileSystem.seedFile(iu2ClaudeSkills+"/recall/SKILL.md", "an earlier copy")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	fileSystem := machine.fileSystem
	g.Expect(fileSystem.text(iu2ClaudeRoot + "/skills/recall/SKILL.md")).To(Equal("recall from the source"))
	g.Expect(fileSystem.kindOf(iu2ClaudeSkills + "/recall")).To(Equal("link"))
	g.Expect(fileSystem.linkTarget(iu2ClaudeSkills + "/recall")).To(Equal(iu2ClaudeRoot + "/skills/recall"))
	g.Expect(fileSystem.text(iu2ClaudeSkills+"/recall/SKILL.md")).To(Equal("recall from the source"), "discoverable")
	g.Expect(iu2Section(outcome.stdout, iu2ClaudeHeader)).To(ContainElement("    adopted: ~/.claude/skills/recall"))
}

// Test792a11k1b_Update_AnUnmanagedEntryIsReportedNotDeleted verifies Scenario 792a11k1b: when the first update finds
// a real directory in the skills directory that is not in the deploy set, it is left in place and the report lists
// it as unmanaged, left alone.
func Test792a11k1b_Update_AnUnmanagedEntryIsReportedNotDeleted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeSkills+"/custom/SKILL.md", "the user's skill")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.text(iu2ClaudeSkills+"/custom/SKILL.md")).To(Equal("the user's skill"), "left in place")
	g.Expect(iu2Section(outcome.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    unmanaged (left alone): ~/.claude/skills/custom",
	))
}

// Test792a11k1c_Update_TheUnmanagedListIsNotRepeated verifies Scenario 792a11k1c: on a later update of the same
// harness with the same unmanaged entry, the report does not list it.
func Test792a11k1c_Update_TheUnmanagedListIsNotRepeated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")
	machine.fileSystem.seedFile(iu2ClaudeSkills+"/custom/SKILL.md", "the user's skill")

	first := machine.update(t, cli.UpdateArgs{})
	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(first.stdout).To(ContainSubstring("unmanaged (left alone): ~/.claude/skills/custom"), "listed once")

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu2ClaudeSkills+"/custom")).To(Equal("dir"), "still there")
	g.Expect(second.stdout).NotTo(ContainSubstring("unmanaged"))
	g.Expect(second.stdout).NotTo(ContainSubstring("custom"))
}

// unexported constants.
const (
	iu2Binary       = iu1HomeBinary
	iu2ClaudeDir    = iu2Home + "/.claude"
	iu2ClaudeHeader = "  Claude Code (~/.claude/):"
	iu2ClaudeRoot   = iu2ClaudeDir + "/engram"
	iu2ClaudeSkills = iu2ClaudeDir + "/skills"
	iu2Home         = "/home/u"
	iu2Marker       = ".engram-owned"
	// iu2MaxLinkDepth bounds the chain of links the in-memory filesystem follows, as an operating system does.
	iu2MaxLinkDepth = 16
	iu2PiDir        = iu2Home + "/.pi"
	iu2PiHeader     = "  Pi (~/.pi/):"
	iu2PiSkills     = iu2PiDir + "/agent/skills"
	iu2Source       = "/repo/agent-instructions"
)

// iu2Kind is what a node of the in-memory filesystem is.
type iu2Kind int

// iu2Kind values.
const (
	iu2KindDir iu2Kind = iota + 1
	iu2KindFile
	iu2KindLink
)

// unexported variables.
var (
	_ update.DirEntry   = iu2DirEntry{}
	_ update.FileInfo   = iu2Info{}
	_ update.Filesystem = (*iu2FS)(nil)
)

// iu2DirEntry is one entry a directory lists, typed by the entry itself: a link is not a directory, as in os.ReadDir.
type iu2DirEntry struct {
	name string
	dir  bool
}

func (e iu2DirEntry) IsDir() bool { return e.dir }

func (e iu2DirEntry) Name() string { return e.name }

// iu2FS is an in-memory update.Filesystem that models the parts of a disk the update depends on: directories, files
// and symbolic links, links followed by every call but Lstat, ReadLink, RemoveAll and Symlink, and parents that must
// exist before a file is written. It can deny every write under one path, as a permission error does.
type iu2FS struct {
	nodes  map[string]*iu2Node
	denied string
}

func (f *iu2FS) Lstat(path string) (update.FileInfo, error) {
	resolved, err := f.resolve(path, false)
	if err != nil {
		return nil, err
	}

	return f.info(path, resolved)
}

func (f *iu2FS) MkdirAll(path string, _ fs.FileMode) error {
	current := "/"

	for part := range strings.SplitSeq(strings.Trim(filepath.Clean(path), "/"), "/") {
		if part == "" {
			continue
		}

		resolved, err := f.resolve(filepath.Join(current, part), true)
		if err != nil {
			return err
		}

		node := f.nodes[resolved]

		switch {
		case node == nil:
			if f.isDenied(resolved) {
				return iu2PermissionDenied(resolved)
			}

			f.nodes[resolved] = &iu2Node{kind: iu2KindDir}
		case node.kind != iu2KindDir:
			return fmt.Errorf("%s: not a directory", resolved)
		}

		current = resolved
	}

	return nil
}

func (f *iu2FS) ReadDir(path string) ([]update.DirEntry, error) {
	resolved, err := f.resolve(path, true)
	if err != nil {
		return nil, err
	}

	node := f.nodes[resolved]
	if node == nil {
		return nil, iu2NotExist(path)
	}

	if node.kind != iu2KindDir {
		return nil, fmt.Errorf("%s: not a directory", path)
	}

	entries := []update.DirEntry{}

	for key, child := range f.nodes {
		if key != resolved && filepath.Dir(key) == resolved {
			entries = append(entries, iu2DirEntry{name: filepath.Base(key), dir: child.kind == iu2KindDir})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return entries, nil
}

func (f *iu2FS) ReadFile(path string) ([]byte, error) {
	resolved, err := f.resolve(path, true)
	if err != nil {
		return nil, err
	}

	node := f.nodes[resolved]
	if node == nil {
		return nil, iu2NotExist(path)
	}

	if node.kind != iu2KindFile {
		return nil, fmt.Errorf("%s: is a directory", path)
	}

	return slices.Clone(node.data), nil
}

func (f *iu2FS) ReadLink(path string) (string, error) {
	resolved, err := f.resolve(path, false)
	if err != nil {
		return "", err
	}

	node := f.nodes[resolved]
	if node == nil {
		return "", iu2NotExist(path)
	}

	if node.kind != iu2KindLink {
		return "", fmt.Errorf("%s: invalid argument", path)
	}

	return node.target, nil
}

func (f *iu2FS) RemoveAll(path string) error {
	resolved, err := f.resolve(path, false)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}

	if f.nodes[resolved] == nil {
		return nil
	}

	if f.isDenied(resolved) {
		return iu2PermissionDenied(resolved)
	}

	f.drop(resolved)

	return nil
}

func (f *iu2FS) Stat(path string) (update.FileInfo, error) {
	resolved, err := f.resolve(path, true)
	if err != nil {
		return nil, err
	}

	return f.info(path, resolved)
}

func (f *iu2FS) Symlink(target, link string) error {
	resolved, err := f.resolve(link, false)
	if err != nil {
		return err
	}

	if f.nodes[resolved] != nil {
		return fmt.Errorf("%s: file exists", link)
	}

	parent := f.nodes[filepath.Dir(resolved)]
	if parent == nil || parent.kind != iu2KindDir {
		return iu2NotExist(link)
	}

	if f.isDenied(resolved) {
		return iu2PermissionDenied(resolved)
	}

	f.nodes[resolved] = &iu2Node{kind: iu2KindLink, target: target}

	return nil
}

func (f *iu2FS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	resolved, err := f.resolve(path, true)
	if err != nil {
		return err
	}

	parent := f.nodes[filepath.Dir(resolved)]
	if parent == nil || parent.kind != iu2KindDir {
		return iu2NotExist(path)
	}

	if existing := f.nodes[resolved]; existing != nil && existing.kind != iu2KindFile {
		return fmt.Errorf("%s: is a directory", path)
	}

	if f.isDenied(resolved) {
		return iu2PermissionDenied(resolved)
	}

	f.nodes[resolved] = &iu2Node{kind: iu2KindFile, data: slices.Clone(data)}

	return nil
}

// denyWritesUnder makes every later write under path fail with a permission error. Seeding is not a write.
func (f *iu2FS) denyWritesUnder(path string) { f.denied = path }

// drop removes the node at path and, for a directory, everything beneath it.
func (f *iu2FS) drop(path string) {
	delete(f.nodes, path)

	for key := range f.nodes {
		if strings.HasPrefix(key, path+"/") {
			delete(f.nodes, key)
		}
	}
}

// followLink is the path a link found in directory dir leads to, a relative target being relative to dir.
func (f *iu2FS) followLink(dir string, link *iu2Node, depth int) (string, error) {
	target := link.target
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}

	return f.resolveAt(target, true, depth+1)
}

func (f *iu2FS) info(path, resolved string) (update.FileInfo, error) {
	node := f.nodes[resolved]
	if node == nil {
		return nil, iu2NotExist(path)
	}

	info := iu2Info{dir: node.kind == iu2KindDir}
	if node.kind == iu2KindLink {
		info.mode = fs.ModeSymlink
	}

	return info, nil
}

func (f *iu2FS) isDenied(path string) bool {
	return f.denied != "" && (path == f.denied || strings.HasPrefix(path, f.denied+"/"))
}

// kindOf is "dir", "file" or "link" for what is at path without following a link at its end, and "" for nothing.
func (f *iu2FS) kindOf(path string) string {
	resolved, err := f.resolve(path, false)
	if err != nil {
		return ""
	}

	node := f.nodes[resolved]
	if node == nil {
		return ""
	}

	switch node.kind {
	case iu2KindDir:
		return "dir"
	case iu2KindFile:
		return "file"
	case iu2KindLink:
		return "link"
	}

	return ""
}

// linkTarget is the destination the link at path was made to hold, or "" when path is no link.
func (f *iu2FS) linkTarget(path string) string {
	target, err := f.ReadLink(path)
	if err != nil {
		return ""
	}

	return target
}

// removeAll deletes path and everything beneath it, as the test arranging a source or a machine does.
func (f *iu2FS) removeAll(path string) { f.drop(filepath.Clean(path)) }

func (f *iu2FS) resolve(path string, followLast bool) (string, error) {
	return f.resolveAt(path, followLast, 0)
}

// resolveAt turns path into the path of the node it names: every link on the way is followed, the last one only when
// followLast. A missing directory on the way is an error; a missing last element is the path where it would be.
func (f *iu2FS) resolveAt(path string, followLast bool, depth int) (string, error) {
	if depth > iu2MaxLinkDepth {
		return "", fmt.Errorf("%s: too many levels of symbolic links", path)
	}

	parts := strings.FieldsFunc(filepath.Clean(path), func(r rune) bool { return r == '/' })
	current := "/"

	for index, part := range parts {
		last := index == len(parts)-1
		next := filepath.Join(current, part)
		node := f.nodes[next]

		if node != nil && node.kind == iu2KindLink && (!last || followLast) {
			resolved, err := f.followLink(current, node, depth)
			if err != nil {
				return "", err
			}

			next = resolved
			node = f.nodes[next]
		}

		if node == nil && !last {
			return "", iu2NotExist(path)
		}

		current = next
	}

	return current, nil
}

// seedDir, seedFile and seedLink arrange the machine without a permission check, creating the directories above.
func (f *iu2FS) seedDir(path string) {
	for current := filepath.Clean(path); current != "/"; current = filepath.Dir(current) {
		if f.nodes[current] == nil {
			f.nodes[current] = &iu2Node{kind: iu2KindDir}
		}
	}
}

func (f *iu2FS) seedFile(path, content string) {
	f.seedDir(filepath.Dir(path))
	f.nodes[filepath.Clean(path)] = &iu2Node{kind: iu2KindFile, data: []byte(content)}
}

func (f *iu2FS) seedLink(link, target string) {
	f.seedDir(filepath.Dir(link))
	f.nodes[filepath.Clean(link)] = &iu2Node{kind: iu2KindLink, target: target}
}

// snapshot lists every node at or under prefix as one line each, sorted: its path, kind, and content or target.
func (f *iu2FS) snapshot(prefix string) []string {
	lines := []string{}

	for path, node := range f.nodes {
		if prefix != "" && path != prefix && !strings.HasPrefix(path, prefix+"/") {
			continue
		}

		switch node.kind {
		case iu2KindDir:
			lines = append(lines, path+" dir")
		case iu2KindFile:
			lines = append(lines, fmt.Sprintf("%s file %q", path, node.data))
		case iu2KindLink:
			lines = append(lines, fmt.Sprintf("%s link %q", path, node.target))
		}
	}

	sort.Strings(lines)

	return lines
}

// text reads the file at path through any links, and is "" when it cannot be read.
func (f *iu2FS) text(path string) string {
	data, err := f.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(data)
}

// iu2Info is what the in-memory filesystem answers for Stat and Lstat.
type iu2Info struct {
	dir  bool
	mode fs.FileMode
}

func (i iu2Info) IsDir() bool { return i.dir }

func (i iu2Info) Mode() fs.FileMode { return i.mode }

// iu2Machine is one machine an update runs on, in the shape the operator meets it: a home with Claude Code under it,
// a checkout of the engram module as the working directory, and a go toolchain whose install succeeds. The binary the
// install leaves runs the rest of the update, as it does on a disk.
type iu2Machine struct {
	fileSystem *iu2FS
	cmd        *iu1Commander
	env        iu1Env
	spawn      *iu1Spawner
}

// guidance puts one guidance document into the source.
func (m *iu2Machine) guidance(name, content string) {
	m.fileSystem.seedFile(iu2Source+"/guidance/"+name, content)
}

// skill puts one file of a skill into the source, replacing the file when it is there.
func (m *iu2Machine) skill(name, rel, content string) {
	m.fileSystem.seedFile(iu2Source+"/skills/"+name+"/"+rel, content)
}

// update runs `engram update` once: the installed binary the parent starts runs the update again with the
// environment it was handed and the same flags, and its status is the one the parent exits with.
func (m *iu2Machine) update(t *testing.T, args cli.UpdateArgs) iu2Outcome {
	t.Helper()

	var stdout bytes.Buffer

	exits := []int{}
	childErrs := []error{}

	m.spawn.run = func(_ string, _ []string, env []string) (int, error) {
		vars := map[string]string{}

		for _, entry := range env {
			key, value, _ := strings.Cut(entry, "=")
			vars[key] = value
		}

		childEnv := iu1Env{home: m.env.home, cwd: m.env.cwd, vars: vars}
		childDeps := cli.ExportNewUpdateDepsWithSpawn(m.fileSystem, m.cmd, childEnv, &iu1Spawner{}, func(int) {})

		childErr := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{WithGuidance: args.WithGuidance}, childDeps, &stdout)
		if childErr != nil {
			childErrs = append(childErrs, childErr)

			return 1, nil
		}

		return 0, nil
	}

	deps := cli.ExportNewUpdateDepsWithSpawn(
		m.fileSystem, m.cmd, m.env, m.spawn, func(code int) { exits = append(exits, code) },
	)

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return iu2Outcome{err: err, stdout: stdout.String(), exits: exits, childErrs: childErrs}
}

// iu2Node is one entry of the in-memory filesystem.
type iu2Node struct {
	kind   iu2Kind
	data   []byte
	target string
}

// iu2Outcome is what one run of the update returned, wrote and exited with, and what the binary it started returned.
type iu2Outcome struct {
	err       error
	stdout    string
	exits     []int
	childErrs []error
}

// iu2ContainsAny reports whether text holds any of the markers.
func iu2ContainsAny(text string, markers []string) bool {
	return slices.ContainsFunc(markers, func(marker string) bool { return strings.Contains(text, marker) })
}

// iu2Lines splits a report into its lines.
func iu2Lines(stdout string) []string {
	return strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
}

// iu2NewMachine is a machine with Claude Code under /home/u and a checkout of the engram module at /repo with no
// skills in it.
func iu2NewMachine() *iu2Machine {
	fileSystem := &iu2FS{nodes: map[string]*iu2Node{"/": {kind: iu2KindDir}}}
	fileSystem.seedDir(iu2ClaudeDir)
	fileSystem.seedFile("/repo/go.mod", "module github.com/toejough/engram\n")

	return &iu2Machine{
		fileSystem: fileSystem,
		cmd:        &iu1Commander{},
		env:        iu1Env{home: iu2Home, cwd: "/repo", vars: map[string]string{}},
		spawn:      &iu1Spawner{},
	}
}

func iu2NotExist(path string) error { return fmt.Errorf("%s: %w", path, fs.ErrNotExist) }

func iu2PermissionDenied(path string) error { return fmt.Errorf("%s: permission denied", path) }

// iu2Section returns the lines of the report that belong under one harness header, without the header.
func iu2Section(stdout, header string) []string {
	lines := iu2Lines(stdout)
	start := slices.Index(lines, header)

	if start < 0 {
		return []string{}
	}

	section := []string{}

	for _, line := range lines[start+1:] {
		if !strings.HasPrefix(line, "    ") {
			break
		}

		section = append(section, line)
	}

	return section
}
