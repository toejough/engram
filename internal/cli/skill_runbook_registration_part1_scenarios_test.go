package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// Test792a12a1a_RegisterSkills_ADeployedSkillIsByteIdentical verifies 792a12a1a (skill-runbook-registration.md):
// registering a deployed skill whose frontmatter has only name and description leaves its SKILL.md byte-identical
// and adds no file to the skill directory (or anywhere else under the home). It runs over a scratch home on the
// real file system (the recorded default-suite deviation, #860) so an added or rewritten file would show.
func Test792a12a1a_RegisterSkills_ADeployedSkillIsByteIdentical(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const skillText = "---\nname: route\ndescription: dispatch work\n---\n\n1. Dispatch the work.\n"

	home := t.TempDir()
	vault := t.TempDir()
	deployed := filepath.Join(home, ".claude", "engram", "skills", "route")

	srr1WriteFile(g, filepath.Join(deployed, "SKILL.md"), skillText)
	srr1LinkDeployedSkill(g, home, "route")

	before := srr1Snapshot(g, home)

	stderr := srr1RunCLI(t, home, "register-skills", "--accept", "claude:route", "--vault", vault)
	g.Expect(stderr).NotTo(ContainSubstring("register-skills:"))

	content, readErr := os.ReadFile(filepath.Join(deployed, "SKILL.md"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(content)).To(Equal(skillText))

	entries, dirErr := os.ReadDir(deployed)
	g.Expect(dirErr).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1), "no file is added to the skill directory")
	g.Expect(srr1Snapshot(g, home)).To(Equal(before), "nothing under the home was added or changed")
	g.Expect(srr1VaultNotes(g, vault, ".skill-claude-route.md")).To(HaveLen(1), "the skill was registered")
}

// Test792a12a1b_RegisterSkills_CommandAndPromptFilesAreUntouched verifies 792a12a1b: a command .md file with
// description and argument-hint frontmatter and a Pi prompt template, both registered, are byte-identical
// afterward with no file added beside them, and their keys come from the file names, not from a frontmatter
// field. It runs over a scratch home on the real file system (the recorded default-suite deviation, #860).
func Test792a12a1b_RegisterSkills_CommandAndPromptFilesAreUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		commandText = "---\ndescription: check the work\nargument-hint: <scope>\n---\n\nCheck $ARGUMENTS.\n"
		promptText  = "---\ndescription: look over a file\nargument-hint: <file>\n---\n\nLook over $1.\n"
	)

	home := t.TempDir()
	vault := t.TempDir()

	srr1WriteFile(g, filepath.Join(home, ".claude", "commands", "audit.md"), commandText)
	srr1WriteFile(g, filepath.Join(home, ".pi", "agent", "prompts", "review.md"), promptText)

	before := srr1Snapshot(g, home)

	stderr := srr1RunCLI(t, home, "register-skills",
		"--accept", "claude:cmd:audit", "--accept", "pi-prompt:review", "--vault", vault)
	g.Expect(stderr).NotTo(ContainSubstring("register-skills:"))

	g.Expect(srr1ReadFile(g, filepath.Join(home, ".claude", "commands", "audit.md"))).To(Equal(commandText))
	g.Expect(srr1ReadFile(g, filepath.Join(home, ".pi", "agent", "prompts", "review.md"))).To(Equal(promptText))
	g.Expect(srr1Snapshot(g, home)).To(Equal(before), "no file was added beside them or changed")

	commandNotes := srr1VaultNotes(g, vault, ".skill-claude-cmd-audit.md")
	promptNotes := srr1VaultNotes(g, vault, ".skill-pi-prompt-review.md")
	g.Expect(commandNotes).To(HaveLen(1))
	g.Expect(promptNotes).To(HaveLen(1))

	if len(commandNotes) != 1 || len(promptNotes) != 1 {
		return
	}

	g.Expect(srr1ReadFile(g, filepath.Join(vault, commandNotes[0]))).To(ContainSubstring("skill_key: claude:cmd:audit\n"))
	g.Expect(srr1ReadFile(g, filepath.Join(vault, promptNotes[0]))).To(ContainSubstring("skill_key: pi-prompt:review\n"))
}

// Test792a12b1a_RegisterSkills_NoteLocatedBySkillKey verifies 792a12b1a: a runbook note with skill_hash and
// skill_key superpowers:brainstorming is the note for that key, so an accepted refresh rewrites it in place
// instead of registering a second note.
func Test792a12b1a_RegisterSkills_NoteLocatedBySkillKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1100.2026-09-26.skill-superpowers-brainstorming.md"

	newText := "---\nname: brainstorming\ndescription: d\n---\n\nThink first.\n"
	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, keyedSkillNoteFixture(pluginSkillSource("6.4.1", "older text")))
	sourceFS := srr1PluginHome(g, "6.4.1", map[string]string{"brainstorming": newText})

	_, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Accept: []string{"superpowers:brainstorming"}})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(Equal([]string{noteName}), "the existing note was refreshed, no second note was registered")

	refreshed, _ := vault.get(noteName)
	g.Expect(parseSkillAcceptFrontmatter(g, refreshed).SkillHash).To(Equal(srr1Hash([]byte(newText))))
	g.Expect(skillNoteBodyOf(refreshed)).To(ContainSubstring("Think first."))
}

// Test792a12b1b_RegisterSkills_NoteNotLocatedBySlug verifies 792a12b1b: the vault's
// 1049.2026-09-21.skill-curate.md is a runbook with a skill_hash and no skill_key, so it is not the note of
// claude:curate or of any other key: registering both scanned skills is offered and the note is not rewritten.
func Test792a12b1b_RegisterSkills_NoteNotLocatedBySlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curate := srr1SkillText("curate")
	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.skill-curate.md", srr1Note("skill_hash: \""+srr1Hash(curate)+"\"\n"))
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": curate, "route": srr1SkillText("route")})
	before := maps.Clone(vault.files)

	stdout, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{DryRun: true})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout).To(Equal("@claude-user (2)\n" +
		"  would offer: register claude:curate (" + skillRegSourcePath("curate") + ")\n" +
		"  would offer: register claude:route (" + skillRegSourcePath("route") + ")\n"))
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12b1c_RegisterSkills_DuplicateNotesAreAnError verifies 792a12b1c: two runbook notes with skill_hash
// that both carry skill_key claude:curate make registration report an error naming both and change nothing.
func Test792a12b1c_RegisterSkills_DuplicateNotesAreAnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		firstNote  = "1049.2026-09-21.skill-claude-curate"
		secondNote = "1050.2026-09-22.skill-claude-curate"
	)

	vault := newSkillAcceptFixtureVault()
	vault.put(firstNote+".md", sourcedSkillNote(offerNote{key: "claude:curate", hash: "first-hash"}))
	vault.put(secondNote+".md", sourcedSkillNote(offerNote{key: "claude:curate", hash: "second-hash"}))
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")})
	before := maps.Clone(vault.files)

	stdout, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}})

	g.Expect(err).To(MatchError(ContainSubstring("duplicate skill note")))
	g.Expect(err).To(MatchError(ContainSubstring(firstNote)))
	g.Expect(err).To(MatchError(ContainSubstring(secondNote)))
	g.Expect(stdout).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before), "no change for claude:curate")
}

// Test792a12b1d_RegisterSkills_UnkeyedLegacyNotesAreIgnoredUntilAdopted verifies 792a12b1d: six legacy notes
// (1036, 1045, 1049, 1053, 1067, 1068) with skill-<name> slugs, no skill_key and a skill_hash equal to the
// scanned skill's hash are not rewritten, draw no refresh or removal, and registering claude:<name> is offered
// for each of the six skills.
func Test792a12b1d_RegisterSkills_UnkeyedLegacyNotesAreIgnoredUntilAdopted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	legacy := []struct{ id, name string }{
		{"1036", "route"}, {"1045", "please"}, {"1049", "curate"},
		{"1053", "write-memory"}, {"1067", "learn"}, {"1068", "recall"},
	}
	vault := newSkillAcceptFixtureVault()
	skills := map[string][]byte{}

	for _, skill := range legacy {
		text := srr1SkillText(skill.name)
		skills[skill.name] = text
		vault.put(skill.id+".2026-09-22.skill-"+skill.name+".md", srr1Note("skill_hash: \""+srr1Hash(text)+"\"\n"))
	}

	before := maps.Clone(vault.files)

	stdout, err := srr1Run(vault, skillsHomeFixture(skills), cli.SkillRegistrationArgs{DryRun: true})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout).To(Equal("@claude-user (6)\n" + srr1RegisterOffers(
		"curate", "learn", "please", "recall", "route", "write-memory")))
	g.Expect(vault.files).To(Equal(before), "none is rewritten")
}

// Test792a12b1e_RegisterSkills_AnUnrecognizedSkillKeyIsNeverMatchedOrRemoved verifies 792a12b1e: a runbook note
// with skill_hash and skill_key route (no colon) is matched to no entry, and with every source root read no
// removal of it is offered.
func Test792a12b1e_RegisterSkills_AnUnrecognizedSkillKeyIsNeverMatchedOrRemoved(t *testing.T) {
	t.Parallel()

	route := srr1SkillText("route")
	note := srr1Note("skill_hash: \"" + srr1Hash(route) + "\"\nskill_key: route\n")

	cases := map[string]struct {
		skills map[string][]byte
		want   string
	}{
		"a skill named route is scanned": {
			skills: map[string][]byte{"route": route},
			want:   "@claude-user (1)\n" + srr1RegisterOffers("route"),
		},
		"no skill is scanned": {skills: map[string][]byte{}, want: ""},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put("1036.2026-09-22.skill-route.md", note)
			before := maps.Clone(vault.files)

			stdout, err := srr1Run(vault, skillsHomeFixture(testCase.skills), cli.SkillRegistrationArgs{DryRun: true})

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stdout).To(Equal(testCase.want), "no match (so a register offer) and no removal offer")
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a12b1f_RegisterSkills_ASkillPrefixedSlugWithoutSkillHashIsNotASkillNote verifies 792a12b1f: a runbook
// note whose slug is skill-edits-validated-by-baseline-pressure-tests and that has no skill_hash is neither
// matched to a skill nor offered for removal. The note carries a skill_key too, so a match would show.
func Test792a12b1f_RegisterSkills_ASkillPrefixedSlugWithoutSkillHashIsNotASkillNote(t *testing.T) {
	t.Parallel()

	const noteName = "1070.2026-09-23.skill-edits-validated-by-baseline-pressure-tests.md"

	note := srr1Note("skill_key: claude:curate\n")

	cases := map[string]struct {
		skills map[string][]byte
		want   string
	}{
		"claude:curate is scanned": {
			skills: map[string][]byte{"curate": srr1SkillText("curate")},
			want:   "@claude-user (1)\n" + srr1RegisterOffers("curate"),
		},
		"no skill is scanned": {skills: map[string][]byte{}, want: ""},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put(noteName, note)
			before := maps.Clone(vault.files)

			stdout, err := srr1Run(vault, skillsHomeFixture(testCase.skills), cli.SkillRegistrationArgs{DryRun: true})

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stdout).To(Equal(testCase.want), "neither matched to a skill nor offered for removal")
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a12b1g_RegisterSkills_AKeyedNoteWithAnotherSlugIsNotFound verifies 792a12b1g: a runbook note with
// skill_hash and skill_key claude:curate whose slug does not begin skill- is not the note of claude:curate, so
// registering claude:curate is offered.
func Test792a12b1g_RegisterSkills_AKeyedNoteWithAnotherSlugIsNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curate := srr1SkillText("curate")
	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md",
		srr1Note("skill_hash: \""+srr1Hash(curate)+"\"\nskill_key: claude:curate\n"))
	before := maps.Clone(vault.files)

	stdout, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": curate}),
		cli.SkillRegistrationArgs{DryRun: true})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout).To(Equal("@claude-user (1)\n" + srr1RegisterOffers("curate")))
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12c1a_RegisterSkills_ANewSkillIsOfferedOnce verifies 792a12c1a: a scanned skill with no note that the
// user declined, unchanged since, draws no offer when register-skills runs again, and the second run writes
// nothing.
func Test792a12c1a_RegisterSkills_ANewSkillIsOfferedOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")})

	_, declineErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Decline: []string{"claude:curate"}})
	g.Expect(declineErr).NotTo(HaveOccurred())

	afterDecline := maps.Clone(vault.files)

	again, againErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{})
	g.Expect(againErr).NotTo(HaveOccurred())
	g.Expect(again).To(BeEmpty(), "no offer is made for the declined skill")

	preview, previewErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(previewErr).NotTo(HaveOccurred())
	g.Expect(preview).To(BeEmpty())
	g.Expect(vault.files).To(Equal(afterDecline))
}

// Test792a12c1b_RegisterSkills_AChangedSkillIsOfferedAgainAfterADecline verifies 792a12c1b: the user declined a
// skill's refresh at hash A and its text now has hash B, so the refresh is offered once for hash B, and
// declining it records hash B.
func Test792a12c1b_RegisterSkills_AChangedSkillIsOfferedAgainAfterADecline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	textA := []byte("# Curate\n\nText A.\n")
	textB := []byte("# Curate\n\nText B.\n")
	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.skill-claude-curate.md", curateSkillNoteFixture(srr1Hash([]byte("older"))))

	sourceA := skillsHomeFixture(map[string][]byte{"curate": textA})
	_, declineErr := srr1Run(vault, sourceA, cli.SkillRegistrationArgs{Decline: []string{"claude:curate"}})
	g.Expect(declineErr).NotTo(HaveOccurred())

	declinedA, _ := vault.get("skill-registrations.json")
	g.Expect(declinedA).To(ContainSubstring(srr1Hash(textA)))

	unchanged, unchangedErr := srr1Run(vault, sourceA, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(unchangedErr).NotTo(HaveOccurred())
	g.Expect(unchanged).To(BeEmpty(), "hash A stays declined")

	sourceB := skillsHomeFixture(map[string][]byte{"curate": textB})

	preview, previewErr := srr1Run(vault, sourceB, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(previewErr).NotTo(HaveOccurred())
	g.Expect(preview).To(Equal("@claude-user (1)\n  would offer: refresh claude:curate (" +
		skillRegSourcePath("curate") + ")\n"))

	prompted, promptErr := srr1Run(vault, sourceB, cli.SkillRegistrationArgs{}, func(deps *cli.SkillRegistrationDeps) {
		deps.IsTerminal = func() bool { return true }
		deps.Stdin = strings.NewReader("n\n")
	})
	g.Expect(promptErr).NotTo(HaveOccurred())
	g.Expect(strings.Count(prompted, "changed since its note was last synced")).To(Equal(1), "offered once")

	declinedB, _ := vault.get("skill-registrations.json")
	g.Expect(declinedB).To(ContainSubstring(srr1Hash(textB)))
	g.Expect(declinedB).NotTo(ContainSubstring(srr1Hash(textA)))
}

// Test792a12c1c_RegisterSkills_AMatchingHashMakesNoOffer verifies 792a12c1c: a skill whose note has a
// skill_hash equal to the skill's current hash draws no offer (an interactive answer is never asked for) and
// nothing is written.
func Test792a12c1c_RegisterSkills_AMatchingHashMakesNoOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curate := srr1SkillText("curate")
	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.skill-claude-curate.md", curateSkillNoteFixture(srr1Hash(curate)))
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": curate})
	before := maps.Clone(vault.files)

	var writes []string

	stdout, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{}, func(deps *cli.SkillRegistrationDeps) {
		deps.IsTerminal = func() bool { return true }
		deps.Stdin = strings.NewReader("yes\n")
		recordWrites(deps, vault, &writes)
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout).To(BeEmpty(), "no prompt and no summary: no offer was made")
	g.Expect(writes).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12c1d_RegisterSkills_ADeclinedRemovalIsNotOfferedAgain verifies 792a12c1d: a removal the user
// declined at the note's current skill_hash is not offered again, and the note stays.
func Test792a12c1d_RegisterSkills_ADeclinedRemovalIsNotOfferedAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1053.2026-09-21.skill-claude-write-memory.md"

	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, sourcedSkillNote(offerNote{key: "claude:write-memory", hash: "wm-hash"}))
	sourceFS := skillsHomeFixture(nil)

	offered, offeredErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(offeredErr).NotTo(HaveOccurred())
	g.Expect(offered).To(ContainSubstring("would offer: remove claude:write-memory"), "the removal is offered first")

	_, declineErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Decline: []string{"claude:write-memory"}})
	g.Expect(declineErr).NotTo(HaveOccurred())

	declined, _ := vault.get("skill-registrations.json")
	g.Expect(declined).To(ContainSubstring("wm-hash"))

	again, againErr := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(againErr).NotTo(HaveOccurred())
	g.Expect(again).To(BeEmpty(), "no removal is offered for the note")

	_, noteKept := vault.get(noteName)
	g.Expect(noteKept).To(BeTrue())
}

// Test792a12d1a_RegisterSkills_AcceptedRegistration verifies 792a12d1a: accepting registration of claude:curate,
// whose resolved source is ~/.claude/engram/skills/curate/SKILL.md, creates a runbook note ending in
// .skill-claude-curate.md with the skill's text after a preamble naming that path, skill_hash, pending: true, no
// runbook fields and a .vec.json sidecar.
func Test792a12d1a_RegisterSkills_AcceptedRegistration(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curate := srr1SkillText("curate")
	vault := newSkillAcceptFixtureVault()

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": curate}),
		cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	g.Expect(names[0]).To(HaveSuffix(".skill-claude-curate.md"))

	note, _ := vault.get(names[0])
	frontmatter := parseSkillAcceptFrontmatter(g, note)

	const source = "~/.claude/engram/skills/curate/SKILL.md"

	g.Expect(frontmatter.Type).To(Equal("runbook"))
	g.Expect(frontmatter.SkillHash).To(Equal(srr1Hash(curate)))
	g.Expect(frontmatter.SkillKey).To(Equal("claude:curate"))
	g.Expect(frontmatter.SkillSource).To(Equal(source))
	g.Expect(frontmatter.Pending).To(BeTrue())
	g.Expect(skillNoteBodyOf(note)).To(Equal(preambleNaming(source) + "\n" + string(curate)))

	for _, field := range []string{"situation:", "triggers:", "done_when:", "red_flags:"} {
		g.Expect(note).NotTo(ContainSubstring(field), "no runbook field is written at registration")
	}

	_, sidecarWritten := vault.get(strings.TrimSuffix(names[0], ".md") + ".vec.json")
	g.Expect(sidecarWritten).To(BeTrue(), "a .vec.json sidecar is written")
}

// Test792a12d1b_RegisterSkills_APluginSkillsPreambleNamesItsRealFile verifies 792a12d1b: accepting registration
// of superpowers:brainstorming, whose resolved source lies under the plugin cache, yields a note with that
// skill_key, that skill_source and a preamble naming the path.
func Test792a12d1b_RegisterSkills_APluginSkillsPreambleNamesItsRealFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const source = "~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md"

	text := "---\nname: brainstorming\ndescription: d\n---\n\nThink first.\n"
	vault := newSkillAcceptFixtureVault()
	sourceFS := srr1PluginHome(g, "6.4.1", map[string]string{"brainstorming": text})

	_, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Accept: []string{"superpowers:brainstorming"}})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	note, _ := vault.get(names[0])
	frontmatter := parseSkillAcceptFrontmatter(g, note)

	g.Expect(names[0]).To(HaveSuffix(".skill-superpowers-brainstorming.md"))
	g.Expect(frontmatter.SkillKey).To(Equal("superpowers:brainstorming"))
	g.Expect(frontmatter.SkillSource).To(Equal(source))
	g.Expect(skillNoteBodyOf(note)).To(Equal(preambleNaming(source) + "\n" + text))
}

// Test792a12d1c_RegisterSkills_ACommandIsRegisteredFromItsFile verifies 792a12d1c: accepting registration of
// the project command project:github.com/toejough/engram:cmd:opsx:apply writes a note whose body is that
// command file after a preamble naming it, in a basename ending in
// .skill-project-github-com-toejough-engram-cmd-opsx-apply.md.
func Test792a12d1c_RegisterSkills_ACommandIsRegisteredFromItsFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		key         = "project:github.com/toejough/engram:cmd:opsx:apply"
		commandText = "---\ndescription: apply a change\n---\n\nApply the change.\n"
	)

	commandPath := projectTop + "/.claude/commands/opsx/apply.md"
	vault := newSkillAcceptFixtureVault()
	sourceFS := skillsHomeFixture(nil).file(commandPath, commandText)

	_, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Accept: []string{key}},
		func(deps *cli.SkillRegistrationDeps) {
			deps.Sources.Commander = scriptedGit{
				"rev-parse --show-toplevel": {out: projectTop + "\n"},
				"remote get-url origin":     {out: "git@github.com:toejough/engram.git\n"},
			}
			deps.Getwd = func() (string, error) { return projectTop, nil }
		})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	note, _ := vault.get(names[0])

	g.Expect(names[0]).To(HaveSuffix(".skill-project-github-com-toejough-engram-cmd-opsx-apply.md"))
	g.Expect(parseSkillAcceptFrontmatter(g, note).SkillKey).To(Equal(key))
	g.Expect(skillNoteBodyOf(note)).To(Equal(preambleNaming(commandPath) + "\n" + commandText))
}

// Test792a12e1a_RegisterSkills_AcceptedRefresh verifies 792a12e1a: accepting the refresh of a note whose skill
// changed gives it the new body and hash, leaves its authored fields and basename unchanged, and marks it
// pending: true.
func Test792a12e1a_RegisterSkills_AcceptedRefresh(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1049.2026-09-21.skill-claude-curate.md"

	newText := []byte("# Curate\n\nStep 1. New procedure text.\n")
	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, curateSkillNoteFixture(srr1Hash([]byte("old curate body"))))

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": newText}),
		cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(Equal([]string{noteName}), "the basename is unchanged")

	refreshed, _ := vault.get(noteName)
	frontmatter := parseSkillAcceptFrontmatter(g, refreshed)

	g.Expect(frontmatter.SkillHash).To(Equal(srr1Hash(newText)))
	g.Expect(frontmatter.Pending).To(BeTrue())
	g.Expect(skillNoteBodyOf(refreshed)).To(ContainSubstring("Step 1. New procedure text."))
	g.Expect(refreshed).NotTo(ContainSubstring("old curate body"))
	srr1ExpectAuthoredFields(g, frontmatter)
}

// Test792a12e1b_RegisterSkills_RefreshFollowsAPluginVersionBump verifies 792a12e1b: a plugin skill whose bytes
// changed under a new installPath version, refresh accepted, gets a skill_source and a preamble that name the
// new version's path.
func Test792a12e1b_RegisterSkills_RefreshFollowsAPluginVersionBump(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1100.2026-09-26.skill-superpowers-brainstorming.md"

	oldSource := pluginSkillSource("6.4.0", "older text").SkillSource
	newSource := pluginSkillSource("6.4.1", "").SkillSource
	newText := "---\nname: brainstorming\ndescription: d\n---\n\nThink first, then act.\n"

	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, keyedSkillNoteFixture(pluginSkillSource("6.4.0", "older text")))
	sourceFS := srr1PluginHome(g, "6.4.1", map[string]string{"brainstorming": newText})

	_, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Accept: []string{"superpowers:brainstorming"}})
	g.Expect(err).NotTo(HaveOccurred())

	refreshed, _ := vault.get(noteName)

	g.Expect(parseSkillAcceptFrontmatter(g, refreshed).SkillSource).To(Equal(newSource))
	g.Expect(skillNoteBodyOf(refreshed)).To(Equal(preambleNaming(newSource) + "\n" + newText))
	g.Expect(refreshed).NotTo(ContainSubstring(oldSource))
}

// Test792a12e1c_RegisterSkills_RefreshKeepsAnUnmodeledKey verifies 792a12e1c: a refresh accepted for a note
// whose frontmatter carries a key the runbook note model does not define, luhmann_old: "12", still carries that
// key with the same value.
func Test792a12e1c_RegisterSkills_RefreshKeepsAnUnmodeledKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1049.2026-09-21.skill-claude-curate.md"

	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, strings.Replace(
		curateSkillNoteFixture(srr1Hash([]byte("older"))), "skill_hash:", "luhmann_old: \"12\"\nskill_hash:", 1))

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
		cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}})
	g.Expect(err).NotTo(HaveOccurred())

	refreshed, _ := vault.get(noteName)
	frontmatter, split := cli.ExportSplitFrontmatter([]byte(refreshed))
	g.Expect(split).To(BeTrue())

	var keys map[string]any

	g.Expect(yaml.Unmarshal(frontmatter, &keys)).To(Succeed())
	g.Expect(keys).To(HaveKeyWithValue("luhmann_old", "12"))
	g.Expect(parseSkillAcceptFrontmatter(g, refreshed).SkillHash).To(Equal(srr1Hash(srr1SkillText("curate"))))
}

// Test792a12e1d_RegisterSkills_RefreshConvertsACRLFNote verifies 792a12e1d: a refresh accepted for a skill note
// whose lines end in \r\n succeeds, the note is converted to LF inside its single write and its authored fields
// are unchanged.
func Test792a12e1d_RegisterSkills_RefreshConvertsACRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteName = "1049.2026-09-21.skill-claude-curate.md"

	vault := newSkillAcceptFixtureVault()
	vault.put(noteName, strings.ReplaceAll(curateSkillNoteFixture(srr1Hash([]byte("older"))), "\n", "\r\n"))

	var writes []string

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
		cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}}, func(deps *cli.SkillRegistrationDeps) {
			recordWrites(deps, vault, &writes)
		})
	g.Expect(err).NotTo(HaveOccurred())

	refreshed, _ := vault.get(noteName)

	g.Expect(refreshed).NotTo(ContainSubstring("\r"))
	g.Expect(srr1Count(writes, srr1Vault+"/"+noteName)).To(Equal(1), "the note is written once")
	srr1ExpectAuthoredFields(g, parseSkillAcceptFrontmatter(g, refreshed))
}

// Test792a12e1e_RegisterSkills_RefreshRefusesAnAnchoredKey verifies 792a12e1e: an accepted refresh that would
// replace or delete a skill_hash, skill_key, skill_source or pending key that carries a YAML anchor, or whose
// value does, is refused naming the key, and the note is unchanged.
func Test792a12e1e_RegisterSkills_RefreshRefusesAnAnchoredKey(t *testing.T) {
	t.Parallel()

	const noteName = "1049.2026-09-21.skill-claude-curate.md"

	base := strings.Replace(curateSkillNoteFixture(srr1Hash([]byte("older"))),
		"skill_hash:", "skill_source: \"~/.claude/engram/skills/curate/SKILL.md\"\nskill_hash:", 1)

	for _, key := range []string{"skill_hash", "skill_key", "skill_source", "pending"} {
		for anchored, mark := range map[string]string{"key": "&anchor " + key + ": ", "value": key + ": &anchor "} {
			t.Run(key+" with an anchored "+anchored, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				vault := newSkillAcceptFixtureVault()
				vault.put(noteName, strings.Replace(base, key+": ", mark, 1))
				before := maps.Clone(vault.files)

				_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
					cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}})

				g.Expect(err).To(MatchError(ContainSubstring(key)))
				g.Expect(vault.files).To(Equal(before), "the note is unchanged")
			})
		}
	}
}

// Test792a12e1f_RegisterSkills_AuthoredFieldsSurviveARefresh verifies 792a12e1f over the real command wiring on
// a scratch home and vault: registering a skill, authoring red_flags (with the fields the note needs) on the
// note with engram amend, changing the skill and accepting the refresh leaves the amended red_flags present.
// The earlier Test792b31c7a9_RefreshSkill_RedFlagsSurviveRefresh keeps the technical 792b31c7a9.
func Test792a12e1f_RegisterSkills_AuthoredFieldsSurviveARefresh(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	home := t.TempDir()
	vault := t.TempDir()
	skillPath := filepath.Join(home, ".claude", "engram", "skills", "curate", "SKILL.md")
	redFlags := []string{"never compose a new learn", "never skip the pending review"}

	srr1WriteFile(g, skillPath, "---\nname: curate\ndescription: first\n---\n\n1. Judge offers.\n")
	srr1LinkDeployedSkill(g, home, "curate")

	registerErr := srr1RunCLI(t, home, "register-skills", "--accept", "claude:curate", "--vault", vault)
	g.Expect(registerErr).NotTo(ContainSubstring("register-skills:"))

	notes := srr1VaultNotes(g, vault, ".skill-claude-curate.md")
	g.Expect(notes).To(HaveLen(1))

	if len(notes) != 1 {
		return
	}

	amendErr := srr1RunCLI(t, home, "amend", "--target", strings.TrimSuffix(notes[0], ".md"),
		"--situation", "curating pending offers", "--done-when", "every offer is settled",
		"--red-flag", redFlags[0], "--red-flag", redFlags[1], "--clear-pending",
		"--vault", vault, "--chunks-dir", t.TempDir())
	g.Expect(amendErr).To(BeEmpty())

	changed := "---\nname: curate\ndescription: second\n---\n\n1. Judge offers, then report.\n"
	srr1WriteFile(g, skillPath, changed)

	refreshErr := srr1RunCLI(t, home, "register-skills", "--accept", "claude:curate", "--vault", vault)
	g.Expect(refreshErr).NotTo(ContainSubstring("register-skills:"))

	g.Expect(srr1VaultNotes(g, vault, ".skill-claude-curate.md")).To(Equal(notes), "the basename is unchanged")

	frontmatter := parseSkillAcceptFrontmatter(g, srr1ReadFile(g, filepath.Join(vault, notes[0])))

	g.Expect(frontmatter.RedFlags).To(Equal(redFlags), "the amended red_flags are still present")
	g.Expect(frontmatter.Situation).To(Equal("curating pending offers"))
	g.Expect(frontmatter.SkillHash).To(Equal(srr1Hash([]byte(changed))), "and the refresh ran")
}

// Test792a12f1a_RegisterSkills_ASkillNoLongerShipped verifies 792a12f1a: for a skill note whose skill is absent
// from the scanned set, accepting its removal deletes the note and its sidecar, and only those.
func Test792a12f1a_RegisterSkills_ASkillNoLongerShipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		goneNote  = "1053.2026-09-21.skill-claude-write-memory"
		keptNote  = "1049.2026-09-21.skill-claude-curate"
		sidecarFx = `{"model_id":"m"}`
	)

	vault := newSkillAcceptFixtureVault()
	vault.put(goneNote+".md", sourcedSkillNote(offerNote{key: "claude:write-memory", hash: "wm-hash"}))
	vault.put(goneNote+".vec.json", sidecarFx)

	curate := srr1SkillText("curate")
	vault.put(keptNote+".md", curateSkillNoteFixture(srr1Hash(curate)))
	vault.put(keptNote+".vec.json", sidecarFx)

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": curate}),
		cli.SkillRegistrationArgs{Accept: []string{"claude:write-memory"}})
	g.Expect(err).NotTo(HaveOccurred())

	_, noteThere := vault.get(goneNote + ".md")
	_, sidecarThere := vault.get(goneNote + ".vec.json")
	_, keptThere := vault.get(keptNote + ".md")
	_, keptSidecarThere := vault.get(keptNote + ".vec.json")

	g.Expect(noteThere).To(BeFalse(), "the note is gone")
	g.Expect(sidecarThere).To(BeFalse(), "its sidecar is gone")
	g.Expect(keptThere).To(BeTrue(), "no note without an accepted offer is deleted")
	g.Expect(keptSidecarThere).To(BeTrue())
}

// Test792a12g1a_UpdateSkillRegistration_AnAgentRunsUpdateThroughAShell verifies 792a12g1a: with stdin not a
// terminal and one scanned skill with no note, the registration step of engram update writes nothing, leaves
// skill-registrations.json unchanged, never reads an answer from stdin, and prints the skill's scope selector with
// its count and the answering command. It runs the update hook over a scratch home and vault on the real file
// system (the recorded default-suite deviation, #860); the deploy half of engram update is not part of it.
func Test792a12g1a_UpdateSkillRegistration_AnAgentRunsUpdateThroughAShell(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const declinesFile = "{\"schema_version\":2,\"declined\":{\"claude:other\":\"hash\"}}"

	home := t.TempDir()
	vault := t.TempDir()

	srr1WriteFile(g, filepath.Join(home, ".claude", "skills", "curate", "SKILL.md"), string(srr1SkillText("curate")))
	srr1WriteFile(g, filepath.Join(vault, "skill-registrations.json"), declinesFile)

	var stdout bytes.Buffer

	deps := newTestDeps(&stdout, &bytes.Buffer{})
	deps.UserHomeDir = func() (string, error) { return home, nil }
	deps.Getwd = func() (string, error) { return home, nil }
	deps.Commander = scriptedGit{}
	deps.Embed = skillAcceptFakeEmbedder{}
	deps.IsTerminal = func() bool { return false }
	deps.Stdin = strings.NewReader("y\n")

	err := cli.ExportRunUpdateSkillRegistrationFromDeps(context.Background(), deps, false, vault, home, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(Equal("engram: 1 skill runbook offer awaiting an answer: @claude-user 1 — run " +
		"`engram register-skills` in a terminal, or `engram register-skills --accept <key|prefix*|@scope>` / " +
		"`--decline <key|prefix*|@scope>`\n"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1), "nothing is written beside skill-registrations.json")
	g.Expect(srr1ReadFile(g, filepath.Join(vault, "skill-registrations.json"))).To(Equal(declinesFile))
}

// unexported constants.
const (
	srr1Vault     = "/vault"
	srr1VaultName = "personal"
)

// srr1Count counts the entries of values equal to want.
func srr1Count(values []string, want string) int {
	count := 0

	for _, value := range values {
		if value == want {
			count++
		}
	}

	return count
}

// srr1ExpectAuthoredFields asserts the runbook fields curateSkillNoteFixture authors on its note are intact.
func srr1ExpectAuthoredFields(g Gomega, frontmatter skillAcceptFrontmatterProbe) {
	g.Expect(frontmatter.Situation).To(Equal("curating pending offers against the host vault"))
	g.Expect(frontmatter.DoneWhen).To(Equal("every pending offer has been discarded, folded, or accepted"))
	g.Expect(frontmatter.RedFlags).To(Equal([]string{"do not compose a new learn", "do not report outcome to caller"}))
	g.Expect(frontmatter.Triggers).To(Equal([]string{"curate", "pending offers"}))
	g.Expect(frontmatter.Created).To(Equal("2026-09-21"))
	g.Expect(frontmatter.Luhmann).To(Equal("1049"))
}

// srr1Hash is the lowercase hex SHA-256 of content, the skill_hash of the spec.
func srr1Hash(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}

// srr1LinkDeployedSkill links ~/.claude/skills/<name> to engram's deployed copy under ~/.claude/engram/skills,
// as `engram update` deploys it.
func srr1LinkDeployedSkill(g Gomega, home, name string) {
	userSkills := filepath.Join(home, ".claude", "skills")
	g.Expect(os.MkdirAll(userSkills, 0o750)).To(Succeed())
	g.Expect(os.Symlink(
		filepath.Join(home, ".claude", "engram", "skills", name), filepath.Join(userSkills, name),
	)).To(Succeed())
}

// srr1Note renders a runbook note whose frontmatter carries extra after its situation and done_when.
func srr1Note(extra string) string {
	return "---\ntype: runbook\nsituation: s\ndone_when: d\n" + extra + "---\n\nbody\n"
}

// srr1PluginHome is skillsHomeFixture(nil) with the plugin superpowers@claude-plugins-official installed and
// enabled at version, holding skills as skills/<name>/SKILL.md.
func srr1PluginHome(g Gomega, version string, skills map[string]string) *fakeSkillFS {
	const pluginKey = "superpowers@claude-plugins-official"

	installPath := skillRegHome + "/.claude/plugins/cache/claude-plugins-official/superpowers/" + version
	fsys := skillsHomeFixture(nil)

	for name, text := range skills {
		fsys.file(installPath+"/skills/"+name+"/SKILL.md", text)
	}

	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: pluginKey, installPath: installPath}}, map[string]bool{pluginKey: true})

	return fsys
}

// srr1ReadFile reads path as a string.
func srr1ReadFile(g Gomega, path string) string {
	content, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	return string(content)
}

// srr1RegisterOffers is the dry-run lines offering to register each claude:<name> skill of skillRegHome.
func srr1RegisterOffers(names ...string) string {
	var lines strings.Builder

	for _, name := range names {
		lines.WriteString("  would offer: register claude:" + name + " (" + skillRegSourcePath(name) + ")\n")
	}

	return lines.String()
}

// srr1Run runs RunSkillRegistration over vault and sourceFS with args (its vault, vault name and home filled in),
// after customize has adjusted the composed deps, and returns what it printed with its error.
func srr1Run(
	vault *skillAcceptFixtureVault,
	sourceFS cli.SkillSourceFS,
	args cli.SkillRegistrationArgs,
	customize ...func(*cli.SkillRegistrationDeps),
) (string, error) {
	deps := skillRegistrationDepsFor(vault, sourceFS)

	for _, adjust := range customize {
		adjust(&deps)
	}

	args.Vault, args.VaultName, args.Home = srr1Vault, srr1VaultName, skillRegHome

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(context.Background(), args, deps, &stdout)

	return stdout.String(), err
}

// srr1RunCLI runs `engram <args>` over the real command wiring with the home, working directory, git, embedder
// and terminal pinned to scratch values, and returns stderr.
func srr1RunCLI(t *testing.T, home string, args ...string) string {
	t.Helper()

	return executeForTestWithDeps(t, append([]string{"engram"}, args...), func(deps *cli.Deps) {
		deps.UserHomeDir = func() (string, error) { return home, nil }
		deps.Getwd = func() (string, error) { return home, nil }
		deps.Commander = scriptedGit{}
		deps.Embed = skillAcceptFakeEmbedder{}
	})
}

// srr1SkillText is the SKILL.md of an engram skill: a frontmatter of only name and description.
func srr1SkillText(name string) []byte {
	return []byte("---\nname: " + name + "\ndescription: the " + name + " skill\n---\n\n1. Do " + name + ".\n")
}

// srr1Snapshot maps every path under root (relative) to what it holds: a file's bytes, a symlink's target, or dir.
func srr1Snapshot(g Gomega, root string) map[string]string {
	snapshot := map[string]string{}

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}

			snapshot[relative] = "-> " + target
		case entry.IsDir():
			snapshot[relative] = "dir"
		default:
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}

			snapshot[relative] = "file " + string(content)
		}

		return nil
	})
	g.Expect(walkErr).NotTo(HaveOccurred())

	return snapshot
}

// srr1VaultNotes lists the note files of the on-disk vault whose names end in suffix.
func srr1VaultNotes(g Gomega, vault, suffix string) []string {
	entries, err := os.ReadDir(vault)
	g.Expect(err).NotTo(HaveOccurred())

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), suffix) {
			names = append(names, entry.Name())
		}
	}

	return names
}

// srr1WriteFile writes content to path, creating its directories.
func srr1WriteFile(g Gomega, path, content string) {
	g.Expect(os.MkdirAll(filepath.Dir(path), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
}
