package cli_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792a11l1a_Update_TheFlagDeploysGuidance verifies Scenario 792a11l1a: with the source holding the guidance
// document recall.md and --with-guidance given, ~/.claude/engram/guidance/recall.md is written with the source's
// content.
func Test792a11l1a_Update_TheFlagDeploysGuidance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the recall guidance")

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu3ClaudeGuidance + "/recall.md")).To(Equal("file"))
	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/recall.md")).To(Equal("the recall guidance"))
}

// Test792a11l1b_Update_AnExistingImportKeepsGuidanceCurrent verifies Scenario 792a11l1b: with ~/.claude/CLAUDE.md
// importing the guidance file, spelled with ~ or with the home directory spelled out, and no flag, a later update
// replaces the deployed guidance with the source's new content and reports it refreshed.
func Test792a11l1b_Update_AnExistingImportKeepsGuidanceCurrent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		importLine string
	}{
		{name: "the tilde spelling", importLine: "@~/.claude/engram/recall.md"},
		{name: "the home directory spelled out", importLine: "@" + iu2ClaudeRoot + "/recall.md"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu3NewMachine()
			machine.guidance("recall.md", "version one")
			machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", testCase.importLine+"\n")

			first := machine.update(t, cli.UpdateArgs{WithGuidance: true})
			g.Expect(first.err).NotTo(HaveOccurred())
			g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/recall.md")).To(Equal("version one"))

			machine.guidance("recall.md", "version two")

			second := machine.update(t, cli.UpdateArgs{})
			g.Expect(second.err).NotTo(HaveOccurred())

			if first.err != nil || second.err != nil {
				return
			}

			g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/recall.md")).To(Equal("version two"))
			g.Expect(iu2Lines(second.stdout)).To(ContainElement(
				"guidance refreshed: ~/.claude/engram/recall.md (canonical: ~/.claude/engram/guidance/recall.md)",
			))
		})
	}
}

// Test792a11l1c_Update_WithoutOptInGuidanceIsUntouched verifies Scenario 792a11l1c: guidance deployed earlier, no
// flag and a config file that no longer imports it, with the source changed in one document and another gone, leave
// every file and link of the engram root as it was.
func Test792a11l1c_Update_WithoutOptInGuidanceIsUntouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "recall one")
	machine.guidance("old.md", "old one")
	machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", "@~/.claude/engram/recall.md\n")

	first := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/old.md")).To(Equal("old one"))

	machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", "no imports any more\n")
	machine.guidance("recall.md", "recall two")
	machine.fileSystem.removeAll(iu2Source + "/guidance/old.md")

	before := machine.fileSystem.snapshot(iu2ClaudeRoot)

	second := machine.update(t, cli.UpdateArgs{})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance+"/recall.md")).To(Equal("recall one"), "not refreshed")
	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance+"/old.md")).To(Equal("old one"), "not deleted")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/old.md")).To(Equal("link"), "its link stays")
	g.Expect(machine.fileSystem.snapshot(iu2ClaudeRoot)).To(Equal(before), "nothing in the engram root changed")
	g.Expect(second.stdout).NotTo(ContainSubstring("guidance refreshed"))
	g.Expect(second.stdout).NotTo(ContainSubstring("guidance deployed"))
}

// Test792a11l1d_Update_AnImportInsideACodeFenceDoesNotCount verifies Scenario 792a11l1d: when the only line
// importing guidance sits inside a fenced code block of ~/.claude/CLAUDE.md, an update without the flag deploys no
// guidance and links none.
func Test792a11l1d_Update_AnImportInsideACodeFenceDoesNotCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the recall guidance")
	machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md",
		"# Notes\n\nTo import guidance, write:\n\n```\n@~/.claude/engram/recall.md\n```\n")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu3ClaudeGuidance)).To(BeEmpty(), "no guidance subtree")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/recall.md")).To(BeEmpty(), "no import path")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("guidance refreshed"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("guidance deployed"))
	g.Expect(outcome.stdout).To(ContainSubstring("run 'engram update --with-guidance' to deploy it"),
		"the report still offers the opt-in")
}

// Test792a11l1e_Update_AGuidanceDocumentRemovedFromTheSourceIsDeletedOnceManaged verifies Scenario 792a11l1e: with
// guidance managed and a deployed document gone from the source, the next update deletes it from the engram root's
// guidance subtree and removes its link, and keeps the other document.
func Test792a11l1e_Update_AGuidanceDocumentRemovedFromTheSourceIsDeletedOnceManaged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "recall")
	machine.guidance("old.md", "old")

	first := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot + "/old.md")).To(Equal("link"))

	machine.fileSystem.removeAll(iu2Source + "/guidance/old.md")

	second := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu3ClaudeGuidance+"/old.md")).To(BeEmpty(), "the document is deleted")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/old.md")).To(BeEmpty(), "its link is gone")
	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/recall.md")).To(Equal("recall"))
	g.Expect(iu2Section(second.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    engram root: deleted guidance/old.md",
	))
}

// Test792a11l1f_Update_PiGuidanceIsLinkedIntoPisGuidanceDirectory verifies Scenario 792a11l1f: with --with-guidance
// and ~/.pi present, ~/.pi/agent/guidance/recall.md is a link to ~/.pi/agent/engram/guidance/recall.md, which holds
// the source's content.
func Test792a11l1f_Update_PiGuidanceIsLinkedIntoPisGuidanceDirectory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the recall guidance")
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu3PiGuidance + "/recall.md")).To(Equal("link"))
	g.Expect(machine.fileSystem.linkTarget(iu3PiGuidance + "/recall.md")).To(Equal(iu3PiRoot + "/guidance/recall.md"))
	g.Expect(machine.fileSystem.text(iu3PiGuidance + "/recall.md")).To(Equal("the recall guidance"))
}

// Test792a11l1g_Update_OneHarnessesImportManagesGuidanceForEveryHarness verifies Scenario 792a11l1g: with only
// ~/.claude/CLAUDE.md importing a guidance file, no flag and ~/.pi present, guidance is deployed and linked for Pi
// too, and its report line asks to wire Pi's own import.
func Test792a11l1g_Update_OneHarnessesImportManagesGuidanceForEveryHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the recall guidance")
	machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", "@~/.claude/engram/recall.md\n")
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu3PiRoot + "/guidance/recall.md")).To(Equal("file"))
	g.Expect(machine.fileSystem.kindOf(iu3PiGuidance + "/recall.md")).To(Equal("link"))
	g.Expect(machine.fileSystem.text(iu3PiGuidance + "/recall.md")).To(Equal("the recall guidance"))
	g.Expect(iu2Lines(outcome.stdout)).To(
		ContainElement(HavePrefix("guidance deployed to ~/.pi/agent/guidance/recall.md")),
	)
}

// Test792a11m1a_Update_AnExistingImportStillResolves verifies Scenario 792a11m1a: with a real flat recall.md from
// an earlier version in ~/.claude/engram and CLAUDE.md importing ~/.claude/engram/recall.md, an update moves the
// content into the root's guidance subtree and leaves the flat path as a link to it, so the import reads the
// deployed content.
func Test792a11m1a_Update_AnExistingImportStillResolves(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the deployed guidance")
	machine.fileSystem.seedFile(iu2ClaudeRoot+"/recall.md", "the guidance of an earlier version")

	importLine := "@~/.claude/engram/recall.md"
	machine.fileSystem.seedFile(iu2ClaudeDir+"/CLAUDE.md", importLine+"\n")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.text(iu3ClaudeGuidance + "/recall.md")).To(Equal("the deployed guidance"))
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot + "/recall.md")).To(Equal("link"))
	g.Expect(machine.fileSystem.linkTarget(iu2ClaudeRoot + "/recall.md")).To(Equal(iu3ClaudeGuidance + "/recall.md"))

	imported := iu2Home + "/" + strings.TrimPrefix(importLine, "@~/")
	g.Expect(machine.fileSystem.text(imported)).To(Equal("the deployed guidance"), "the import resolves to the content")
}

// Test792a11m1b_Update_ARealFlatFileFromAnEarlierVersionIsAdopted verifies Scenario 792a11m1b: a real
// ~/.claude/engram/recall.md from an earlier version becomes a link to the canonical copy under the guidance
// subtree when guidance is managed, and the report lists it as adopted.
func Test792a11m1b_Update_ARealFlatFileFromAnEarlierVersionIsAdopted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "the deployed guidance")
	machine.fileSystem.seedFile(iu2ClaudeRoot+"/recall.md", "the guidance of an earlier version")

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot + "/recall.md")).To(Equal("link"))
	g.Expect(machine.fileSystem.linkTarget(iu2ClaudeRoot + "/recall.md")).To(Equal(iu3ClaudeGuidance + "/recall.md"))
	g.Expect(machine.fileSystem.text(iu2ClaudeRoot + "/recall.md")).To(Equal("the deployed guidance"))
	g.Expect(iu2Section(outcome.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    adopted: ~/.claude/engram/recall.md",
	))
}

// Test792a11m1c_Update_ALinkWhoseDocumentLeftTheSourceIsRemoved verifies Scenario 792a11m1c: with guidance managed
// and the compat link of a document that has left the source, the next update deletes that link and keeps the link
// of the document still in the source.
func Test792a11m1c_Update_ALinkWhoseDocumentLeftTheSourceIsRemoved(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "recall")
	machine.guidance("old.md", "old")

	first := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(machine.fileSystem.linkTarget(iu2ClaudeRoot + "/old.md")).To(Equal(iu3ClaudeGuidance + "/old.md"))

	machine.fileSystem.removeAll(iu2Source + "/guidance/old.md")

	second := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(second.err).NotTo(HaveOccurred())

	if first.err != nil || second.err != nil {
		return
	}

	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/old.md")).To(BeEmpty(), "the link is deleted")
	g.Expect(machine.fileSystem.kindOf(iu2ClaudeRoot+"/recall.md")).To(Equal("link"), "the other link stays")
	g.Expect(iu2Section(second.stdout, iu2ClaudeHeader)).To(ContainElement(
		"    cleanup: removed dangling link ~/.claude/engram/old.md",
	))
}

// Test792a11m1d_Update_TheReportStatesTheCanonicalPath verifies Scenario 792a11m1d: with guidance deployed for
// Claude Code and Pi, each Claude Code guidance line of the report ends with the canonical path under
// ~/.claude/engram/guidance and no Pi line carries a canonical suffix.
func Test792a11m1d_Update_TheReportStatesTheCanonicalPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.guidance("recall.md", "recall")
	machine.guidance("delegate.md", "delegate")
	machine.fileSystem.seedDir(iu2PiDir)

	outcome := machine.update(t, cli.UpdateArgs{WithGuidance: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	claudeLines := iu3LinesWithPrefix(lines, "guidance deployed to ~/.claude/engram/")
	piLines := iu3LinesWithPrefix(lines, "guidance deployed to ~/.pi/agent/guidance/")

	g.Expect(claudeLines).To(HaveLen(2))

	for _, name := range []string{"recall.md", "delegate.md"} {
		matching := iu3LinesWithPrefix(claudeLines, "guidance deployed to ~/.claude/engram/"+name)
		g.Expect(matching).To(HaveLen(1))

		for _, line := range matching {
			g.Expect(line).To(HaveSuffix("(canonical: ~/.claude/engram/guidance/" + name + ")"))
		}
	}

	g.Expect(piLines).To(HaveLen(2))

	for _, line := range piLines {
		g.Expect(line).NotTo(ContainSubstring("canonical"))
	}
}

// Test792a11n1a_Update_ReportsThePruneCommandWhenDuplicatesExist verifies Scenario 792a11n1a: a chunk index whose
// content-hash group holds three entries that prune would remove makes the update print the duplicates notice naming
// the prune command and its preview, and leaves the chunk index as it was.
func Test792a11n1a_Update_ReportsThePruneCommandWhenDuplicatesExist(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	iu3SeedManifest(g, machine, map[string]map[string]any{
		"/repo/a.md": cis1Entry("sha256:x"),
		"/repo/b.md": cis1Entry("sha256:x"),
		"/repo/c.md": cis1Entry("sha256:x"),
	})
	iu3SeedIndex(machine, "/repo/a.md", cis1Record("/repo/a.md", "sha256:aaa")+cis1Record("/repo/a.md", "sha256:bbb"))
	iu3SeedIndex(machine, "/repo/b.md", cis1Record("/repo/b.md", "sha256:aaa"))
	iu3SeedIndex(machine, "/repo/c.md", cis1Record("/repo/c.md", "sha256:aaa"))

	before := machine.fileSystem.snapshot(iu3ChunksDir)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(strings.Count(outcome.stdout, iu3DuplicatesNotice+"\n")).To(Equal(1))
	g.Expect(outcome.stdout).NotTo(ContainSubstring(iu3EmptyNotice))
	g.Expect(machine.fileSystem.snapshot(iu3ChunksDir)).To(Equal(before), "nothing is removed")
}

// Test792a11n1b_Update_ARefusalOnlyBacklogIsSilent verifies Scenario 792a11n1b: duplicate groups that prune would
// refuse to remove, because the kept entry's index file is missing or lacks the other's records, make the update
// print no duplicates notice.
func Test792a11n1b_Update_ARefusalOnlyBacklogIsSilent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		indexFiles map[string]string
	}{
		{
			name:       "the kept entry has no index file while the other's exists",
			indexFiles: map[string]string{"/repo/b.md": cis1Record("/repo/b.md", "sha256:aaa")},
		},
		{
			name:       "no member is indexed",
			indexFiles: map[string]string{},
		},
		{
			name: "the other holds a record the kept entry lacks",
			indexFiles: map[string]string{
				"/repo/a.md": cis1Record("/repo/a.md", "sha256:aaa"),
				"/repo/b.md": cis1Record("/repo/b.md", "sha256:ccc"),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu3NewMachine()
			iu3SeedManifest(g, machine, map[string]map[string]any{
				"/repo/a.md": cis1Entry("sha256:x"),
				"/repo/b.md": cis1Entry("sha256:x"),
			})

			for source, content := range testCase.indexFiles {
				iu3SeedIndex(machine, source, content)
			}

			outcome := machine.update(t, cli.UpdateArgs{})
			g.Expect(outcome.err).NotTo(HaveOccurred())

			if outcome.err != nil {
				return
			}

			g.Expect(outcome.stdout).NotTo(ContainSubstring("duplicate chunk-index files found"))
		})
	}
}

// Test792a11n1c_Update_EmptyIndexFilesAreReported verifies Scenario 792a11n1c: a zero-byte .jsonl file in the chunk
// index makes the update print the empty-files notice naming the prune command, and the file stays.
func Test792a11n1c_Update_EmptyIndexFilesAreReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	iu3SeedIndex(machine, "/repo/full.md", cis1Record("/repo/full.md", "sha256:aaa"))
	machine.fileSystem.seedFile(iu3ChunksDir+"/empty.jsonl", "")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(iu3EmptyNotice))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("duplicate chunk-index files found"))
	g.Expect(machine.fileSystem.kindOf(iu3ChunksDir+"/empty.jsonl")).To(Equal("file"), "the file is not removed")
}

// Test792a11n1d_Update_ACleanOrMissingIndexIsSilent verifies Scenario 792a11n1d: a chunk index without duplicates or
// empty files, no chunk index, and an index whose manifest is unreadable make the update print neither notice.
func Test792a11n1d_Update_ACleanOrMissingIndexIsSilent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(g *WithT, machine *iu2Machine)
	}{
		{
			name: "a clean index",
			setup: func(g *WithT, machine *iu2Machine) {
				iu3SeedManifest(g, machine, map[string]map[string]any{
					"/repo/a.md": cis1Entry("sha256:x"),
					"/repo/b.md": cis1Entry("sha256:y"),
				})
				iu3SeedIndex(machine, "/repo/a.md", cis1Record("/repo/a.md", "sha256:aaa"))
				iu3SeedIndex(machine, "/repo/b.md", cis1Record("/repo/b.md", "sha256:bbb"))
			},
		},
		{
			name:  "no chunk index",
			setup: func(*WithT, *iu2Machine) {},
		},
		{
			name: "an unreadable manifest",
			setup: func(_ *WithT, machine *iu2Machine) {
				machine.fileSystem.seedFile(iu3ChunksDir+"/manifest.json", "{not valid json")
				iu3SeedIndex(machine, "/repo/a.md", cis1Record("/repo/a.md", "sha256:aaa"))
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu3NewMachine()
			testCase.setup(g, machine)

			outcome := machine.update(t, cli.UpdateArgs{})
			g.Expect(outcome.err).NotTo(HaveOccurred())

			if outcome.err != nil {
				return
			}

			g.Expect(outcome.stdout).NotTo(ContainSubstring("duplicate chunk-index files found"))
			g.Expect(outcome.stdout).NotTo(ContainSubstring("empty chunk-index files found"))
		})
	}
}

// Test792a11n1e_Update_TheNoticesCoexistWithTheOtherUpgradeNotices verifies Scenario 792a11n1e: with both chunk
// index backlogs and old-format vocabulary files in the vault, the report holds the empty-files notice, the
// duplicates notice and the vocabulary notice.
func Test792a11n1e_Update_TheNoticesCoexistWithTheOtherUpgradeNotices(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	iu3SeedManifest(g, machine, map[string]map[string]any{
		"/repo/a.md": cis1Entry("sha256:x"),
		"/repo/b.md": cis1Entry("sha256:x"),
	})
	iu3SeedIndex(machine, "/repo/a.md", cis1Record("/repo/a.md", "sha256:aaa")+cis1Record("/repo/a.md", "sha256:bbb"))
	iu3SeedIndex(machine, "/repo/b.md", cis1Record("/repo/b.md", "sha256:aaa"))
	machine.fileSystem.seedFile(iu3ChunksDir+"/empty.jsonl", "")
	machine.fileSystem.seedFile(iu3Vault+"/vocab.caching.md", "an old-format vocabulary note")

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	lines := iu2Lines(outcome.stdout)
	g.Expect(lines).To(ContainElement(iu3EmptyNotice))
	g.Expect(lines).To(ContainElement(iu3DuplicatesNotice))
	g.Expect(lines).To(ContainElement(iu3VocabNotice))
}

// Test792a11o1a_Update_TheHelpListsEveryOption verifies Scenario 792a11o1a: engram update --help describes the
// update as refreshing the engram binary and harness skills and lists all seven options.
func Test792a11o1a_Update_TheHelpListsEveryOption(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return iu2Home, nil }

	result, err := targ.Execute([]string{"engram", "update", "--help"}, cli.Targets(deps)...)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Output).To(ContainSubstring("Refresh engram binary and harness skills"))

	for _, option := range []string{
		"--dry-run", "--with-guidance", "--allow-downgrade", "--regen-vocab", "--reparent-luhmann", "--answers",
		"--backfill-identity",
	} {
		g.Expect(result.Output).To(ContainSubstring(option), "the help lists "+option)
	}
}

// Test792a11o1b_Update_OptionsReachTheHandedOffRun verifies Scenario 792a11o1b: an update that installs and hands
// off starts the installed binary as engram update with --with-guidance, --regen-vocab and --backfill-identity, and
// with no other option, even when --allow-downgrade is given.
func Test792a11o1b_Update_OptionsReachTheHandedOffRun(t *testing.T) {
	t.Parallel()

	wantArgs := []string{"update", "--with-guidance", "--regen-vocab", "--backfill-identity"}

	cases := []struct {
		name string
		args cli.UpdateArgs
	}{
		{
			name: "the three options",
			args: cli.UpdateArgs{WithGuidance: true, RegenVocab: true, BackfillIdentity: true},
		},
		{
			name: "the three options and --allow-downgrade",
			args: cli.UpdateArgs{WithGuidance: true, RegenVocab: true, BackfillIdentity: true, AllowDowngrade: true},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu1LocalMachine(iu1Repo, iu1Repo)

			outcome := machine.run(t, testCase.args)
			g.Expect(outcome.err).NotTo(HaveOccurred())

			if outcome.err != nil {
				return
			}

			spawned := machine.spawn.spawned()
			g.Expect(spawned).To(HaveLen(1))

			if len(spawned) != 1 {
				return
			}

			g.Expect(spawned[0].name).To(Equal(iu1HomeBinary))
			g.Expect(spawned[0].args).To(Equal(wantArgs))
		})
	}
}

// Test792a11o1c_Update_ReParentingSkipsTheInstallAndTheDeploy verifies Scenario 792a11o1c: on a machine with no
// harness and no Go toolchain, engram update --reparent-luhmann prints the candidates of the vault and runs no
// command, starts no binary and writes nothing.
func Test792a11o1c_Update_ReParentingSkipsTheInstallAndTheDeploy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu2NewMachine()
	machine.fileSystem.removeAll(iu2ClaudeDir)
	machine.fileSystem.removeAll("/repo")
	machine.cmd.answers = append(machine.cmd.answers,
		iu1Answer{prefix: "go ", err: update.ErrCommandNotFound},
		iu1Answer{prefix: "git ", err: update.ErrCommandNotFound},
	)
	machine.env.vars["ENGRAM_VAULT_PATH"] = "/vault"

	files, names := twoRelatedTopLevelNotesFixture()
	reparent, _ := newReparentDeps(files, names)

	before := machine.fileSystem.snapshot("")
	exits := []int{}
	deps := cli.ExportNewUpdateDepsWithSpawnAndReparent(
		machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(code int) { exits = append(exits, code) },
		reparent,
	)

	var stdout bytes.Buffer

	err := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{ReparentLuhmann: true}, deps, &stdout)

	g.Expect(machine.cmd.calls).To(BeEmpty(), "no go install, no git, no other command")
	g.Expect(machine.spawn.spawned()).To(BeEmpty(), "no binary is started")
	g.Expect(exits).To(BeEmpty())
	g.Expect(machine.fileSystem.snapshot("")).To(Equal(before), "nothing is deployed or written")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var payload struct {
		Candidates []struct {
			Note   string `json:"note"`
			Target string `json:"target"`
		} `json:"candidates"`
	}

	decodeErr := json.Unmarshal(stdout.Bytes(), &payload)
	g.Expect(decodeErr).NotTo(HaveOccurred(), "the output is the candidates payload")

	if decodeErr != nil {
		return
	}

	pairs := make([]string, 0, len(payload.Candidates))
	for _, candidate := range payload.Candidates {
		pairs = append(pairs, candidate.Note+">"+candidate.Target)
	}

	slices.Sort(pairs)
	g.Expect(pairs).To(Equal([]string{"12>7", "7>12"}), "candidates are derived")
}

// unexported constants.
const (
	iu3ChunksDir = iu2Home + "/.local/share/engram/chunks"
	// iu3ClaudeGuidance is the guidance subtree of Claude Code's engram root.
	iu3ClaudeGuidance = iu2ClaudeRoot + "/guidance"
	// iu3DuplicatesNotice is the line the update prints when prune --duplicates would remove an index file.
	iu3DuplicatesNotice = "duplicate chunk-index files found — run `engram prune --duplicates` to clear them " +
		"(preview with `engram prune --duplicates --dry-run`)"
	// iu3EmptyNotice is the line the update prints when the chunk index holds a zero-byte index file.
	iu3EmptyNotice = "empty chunk-index files found — run `engram prune --empty` to clear them " +
		"(preview with `engram prune --empty --dry-run`)"
	iu3PiGuidance = iu2PiDir + "/agent/guidance"
	iu3PiRoot     = iu2PiDir + "/agent/engram"
	iu3Vault      = iu2Home + "/.local/share/engram/vault"
	// iu3VocabNotice is the line the update prints when the vault holds old-format vocabulary files.
	iu3VocabNotice = "old-format vocab files found — run `engram update --regen-vocab` to migrate them " +
		"(preview with `engram update --regen-vocab --dry-run`)"
)

// iu3LinesWithPrefix returns the lines that begin with prefix.
func iu3LinesWithPrefix(lines []string, prefix string) []string {
	matching := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			matching = append(matching, line)
		}
	}

	return matching
}

// iu3NewMachine is a machine with Claude Code, a checkout of the engram module and the recall skill in its source.
func iu3NewMachine() *iu2Machine {
	machine := iu2NewMachine()
	machine.skill("recall", "SKILL.md", "recall")

	return machine
}

// iu3SeedIndex puts the per-source index file of source into the chunk index of the machine.
func iu3SeedIndex(machine *iu2Machine, source, content string) {
	machine.fileSystem.seedFile(cli.ExportIndexPathFor(iu3ChunksDir, source), content)
}

// iu3SeedManifest writes the manifest of the chunk index of the machine.
func iu3SeedManifest(g *WithT, machine *iu2Machine, manifest map[string]map[string]any) {
	data, err := json.Marshal(manifest)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	machine.fileSystem.seedFile(iu3ChunksDir+"/manifest.json", string(data))
}
