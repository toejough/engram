package cli_test

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a12h1a_RegisterSkills_AnAgentRelaysTheUsersAnswer verifies 792a12h1a (skill-runbook-registration.md):
// with offers for claude:curate, claude:route and claude:write-memory, a non-interactive run with --accept
// claude:curate and --decline claude:route carries out curate's offer (its note is registered), records route's
// current hash as declined, and only reports the third offer: nothing is prompted, answered from stdin or written
// for it.
func Test792a12h1a_RegisterSkills_AnAgentRelaysTheUsersAnswer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	routeText := srr1SkillText("route")
	sourceFS := skillsHomeFixture(map[string][]byte{
		"curate": srr1SkillText("curate"), "route": routeText, "write-memory": srr1SkillText("write-memory"),
	})
	vault := newSkillAcceptFixtureVault()

	stdout, err := srr1Run(vault, sourceFS,
		cli.SkillRegistrationArgs{Accept: []string{"claude:curate"}, Decline: []string{"claude:route"}},
		func(deps *cli.SkillRegistrationDeps) {
			deps.Stdin = strings.NewReader(strings.Repeat("y\n", srr2PromptAnswers))
		})
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(HaveLen(1), "only the accepted offer wrote a note")

	if len(names) != 1 {
		return
	}

	g.Expect(names[0]).To(HaveSuffix(".skill-claude-curate.md"), "claude:curate's offer was carried out")

	declined, declinedErr := cli.ReadSkillRegistrations(srr1Vault, vault.ReadFile)
	g.Expect(declinedErr).NotTo(HaveOccurred())
	g.Expect(declined).To(Equal(map[string]string{"claude:route": srr1Hash(routeText)}),
		"claude:route's current hash is recorded as declined, and nothing is recorded for the third offer")

	g.Expect(stdout).To(ContainSubstring("engram: 1 skill runbook offer awaiting an answer: @claude-user 1"),
		"the third offer is only reported")
	g.Expect(stdout).NotTo(ContainSubstring("Register skill"), "nothing was prompted")
}

// Test792a12h1b_RegisterSkills_ADryRun verifies 792a12h1b: with a register, a refresh and a removal offer in
// the user skills scope and a register offer in the user commands scope, --dry-run prints each offer it would
// make under its scope header, in scope then key order, writes nothing, and never asks whether stdin is a
// terminal.
func Test792a12h1b_RegisterSkills_ADryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const goneNote = "1053.2026-09-21.skill-claude-write-memory"

	vault := newSkillAcceptFixtureVault()
	vault.put("1036.2026-09-18.skill-claude-route.md",
		sourcedSkillNote(offerNote{key: "claude:route", hash: srr1Hash([]byte("an older route"))}))
	vault.put(goneNote+".md", sourcedSkillNote(offerNote{key: "claude:write-memory", hash: "wm-hash"}))

	sourceFS := skillsHomeFixture(map[string][]byte{
		"curate": srr1SkillText("curate"), "route": srr1SkillText("route"),
	}).file(skillRegHome+"/.claude/commands/audit.md", "# Audit\n")
	before := maps.Clone(vault.files)

	var writes []string

	stdout, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{DryRun: true},
		func(deps *cli.SkillRegistrationDeps) {
			deps.IsTerminal = func() bool {
				t.Fatal("a dry run must not consult whether stdin is a terminal")

				return true
			}

			recordWrites(deps, vault, &writes)
		})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(Equal("@claude-cmd (1)\n" +
		"  would offer: register claude:cmd:audit (" + skillRegHome + "/.claude/commands/audit.md)\n" +
		"@claude-user (3)\n" +
		"  would offer: register claude:curate (" + skillRegSourcePath("curate") + ")\n" +
		"  would offer: refresh claude:route (" + skillRegSourcePath("route") + ")\n" +
		"  would offer: remove claude:write-memory (note " + goneNote + ")\n"))
	g.Expect(writes).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12h1c_RegisterSkills_AnAnswerMatchingNothingIsNoted verifies 792a12h1c: an --accept (and likewise a
// --decline) naming the key nosuch, which no offer has, prints the no-pending-offer line naming the flag and
// carries on: a second answer in the same run for a real offer is still carried out, and the offer left
// unanswered is still reported.
func Test792a12h1c_RegisterSkills_AnAnswerMatchingNothingIsNoted(t *testing.T) {
	t.Parallel()

	skills := map[string][]byte{"curate": srr1SkillText("curate"), "route": srr1SkillText("route")}

	t.Run("accept", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillAcceptFixtureVault()

		stdout, err := srr1Run(vault, skillsHomeFixture(skills),
			cli.SkillRegistrationArgs{Accept: []string{"nosuch", "claude:curate"}})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(ContainSubstring("engram: no pending offer for skill \"nosuch\" — ignoring --accept\n"))
		g.Expect(stdout).To(ContainSubstring("engram: 1 skill runbook offer awaiting an answer: @claude-user 1"))

		names, _ := vault.ListMD(srr1Vault)
		g.Expect(names).To(HaveLen(1), "the real offer was still carried out")

		if len(names) != 1 {
			return
		}

		g.Expect(names[0]).To(HaveSuffix(".skill-claude-curate.md"))
	})

	t.Run("decline", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillAcceptFixtureVault()

		stdout, err := srr1Run(vault, skillsHomeFixture(skills),
			cli.SkillRegistrationArgs{Decline: []string{"nosuch", "claude:curate"}})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(ContainSubstring("engram: no pending offer for skill \"nosuch\" — ignoring --decline\n"))
		g.Expect(stdout).To(ContainSubstring("engram: 1 skill runbook offer awaiting an answer: @claude-user 1"))

		declined, declinedErr := cli.ReadSkillRegistrations(srr1Vault, vault.ReadFile)
		g.Expect(declinedErr).NotTo(HaveOccurred())
		g.Expect(declined).To(Equal(map[string]string{"claude:curate": srr1Hash(skills["curate"])}),
			"the real offer was still declined, and nothing is recorded for nosuch")
	})
}

// Test792a12h1d_RegisterSkills_ASkillsDirRunIsAPreview verifies 792a12h1d over the real command wiring on a
// scratch working directory and vault: register-skills --skills-dir agent-instructions/skills (relative to the
// working directory) lists offers keyed claude:<name> (a register for a skill with no note, a refresh for one
// whose note is stale), lists no removal for a note whose skill is gone, and leaves the vault byte-identical, so
// the stale note's preamble is never rewritten even with a terminal answering yes. It runs on the real file
// system (the recorded default-suite deviation, #860) so a changed or added file would show.
func Test792a12h1d_RegisterSkills_ASkillsDirRunIsAPreview(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cwd := srr2ResolvedTempDir(t, g)
	home := t.TempDir()
	vault := t.TempDir()
	skillsDir := filepath.Join(cwd, "agent-instructions", "skills")

	srr1WriteFile(g, filepath.Join(skillsDir, "curate", "SKILL.md"), string(srr1SkillText("curate")))
	srr1WriteFile(g, filepath.Join(skillsDir, "route", "SKILL.md"), string(srr1SkillText("route")))
	srr1WriteFile(g, filepath.Join(vault, "1036.2026-09-18.skill-claude-route.md"),
		sourcedSkillNote(offerNote{key: "claude:route", hash: srr1Hash([]byte("an older route"))}))
	srr1WriteFile(g, filepath.Join(vault, "1053.2026-09-21.skill-claude-write-memory.md"),
		sourcedSkillNote(offerNote{
			key: "claude:write-memory", hash: "wm-hash",
			source: filepath.Join(skillsDir, "write-memory", "SKILL.md"),
		}))

	before := srr1Snapshot(g, vault)

	stdout, stderr := srr2RunCLI(t, cwd, home, strings.NewReader(strings.Repeat("y\n", srr2PromptAnswers)),
		"register-skills", "--vault", vault, "--skills-dir", "agent-instructions/skills")

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("@claude-user (2)\n" +
		"  would offer: register claude:curate (" + filepath.Join(skillsDir, "curate", "SKILL.md") + ")\n" +
		"  would offer: refresh claude:route (" + filepath.Join(skillsDir, "route", "SKILL.md") + ")\n"))
	g.Expect(srr1Snapshot(g, vault)).To(Equal(before), "the vault is untouched: no preamble rewritten, nothing removed")
}

// Test792a12h1e_RegisterSkills_ASkillsDirRunRefusesAnswers verifies 792a12h1e: --skills-dir with --accept
// claude:route is refused with the read-only-preview error, both from the registration run (before it reads the
// skills directory, the vault or the working directory) and through the command, which writes nothing to the
// vault.
func Test792a12h1e_RegisterSkills_ASkillsDirRunRefusesAnswers(t *testing.T) {
	t.Parallel()

	const refusal = "register-skills: --skills-dir is a read-only preview; " +
		"it cannot be combined with --accept, --decline or --adopt"

	t.Run("registration run", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillAcceptFixtureVault()
		deps := skillRegistrationDepsFor(vault, untouchableSkillFS{g: g})
		deps.Getwd = failingGetwd(g)
		deps.ListMD = func(string) ([]string, error) {
			g.Expect("the vault").To(BeEmpty(), "read before the refusal")

			return nil, nil
		}

		err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
			Vault: srr1Vault, VaultName: srr1VaultName, Home: skillRegHome,
			PreviewDirs: []string{"/checkout/agent-instructions/skills"}, Accept: []string{"claude:route"},
		}, deps, &bytes.Buffer{})

		g.Expect(err).To(MatchError(refusal))
	})

	t.Run("command", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cwd := srr2ResolvedTempDir(t, g)
		vault := t.TempDir()

		stdout, stderr := srr2RunCLI(t, cwd, t.TempDir(), nil, "register-skills", "--vault", vault,
			"--skills-dir", "agent-instructions/skills", "--accept", "claude:route")

		g.Expect(stderr).To(ContainSubstring(refusal))
		g.Expect(stdout).To(BeEmpty())

		entries, readErr := os.ReadDir(vault)
		g.Expect(readErr).NotTo(HaveOccurred())
		g.Expect(entries).To(BeEmpty())
	})
}

// Test792a12h1f_RegisterSkills_OneAnswerInBothFlagsIsRefused verifies 792a12h1f: the same key, pattern or
// selector named in both --accept and --decline is refused before anything is acted on: the sources are not
// scanned, the vault is not listed or written and nothing is recorded.
func Test792a12h1f_RegisterSkills_OneAnswerInBothFlagsIsRefused(t *testing.T) {
	t.Parallel()

	for name, token := range map[string]string{
		"key": "claude:route", "pattern": "claude:*", "selector": "@claude-user",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()

			var writes []string

			_, err := srr1Run(vault, untouchableSkillFS{g: g},
				cli.SkillRegistrationArgs{Accept: []string{token}, Decline: []string{token}},
				func(deps *cli.SkillRegistrationDeps) {
					deps.Getwd = failingGetwd(g)
					deps.ListMD = func(string) ([]string, error) {
						g.Expect("the vault").To(BeEmpty(), "listed before the refusal")

						return nil, nil
					}

					recordWrites(deps, vault, &writes)
				})

			g.Expect(err).To(MatchError(cli.ErrAnswerInBothFlagsForTest))
			g.Expect(err).To(MatchError(ContainSubstring(token)))
			g.Expect(writes).To(BeEmpty())
			g.Expect(vault.files).To(BeEmpty())
		})
	}
}

// Test792a12i1a_RegisterSkills_AdoptingAPreviouslyPromotedNote verifies 792a12i1a: --adopt claude:curate=1049
// over note 1049.2026-09-21.skill-curate.md (no skill_key, still marked pending from its promotion) renames it
// to 1049.2026-09-21.skill-claude-curate.md, records the old basename in aliases, rewrites a link to the old
// basename, moves its sidecar, makes its body the curate skill after a preamble naming its skill_source, leaves
// its runbook fields unchanged, and gives it skill_hash, skill_key and skill_source and no pending marker. The
// adopt runs before the offers, so no register offer follows for curate.
func Test792a12i1a_RegisterSkills_AdoptingAPreviouslyPromotedNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", curatePromotedNoteFixturePending())
	vault.put(srr2OldBasename+".vec.json", `{"model_id":"old"}`)
	vault.put(srr2Referrer+".md", referencingNoteFixture(srr2OldBasename))

	stdout, err := srr2Adopt(vault)
	g.Expect(err).NotTo(HaveOccurred())

	names, _ := vault.ListMD(srr1Vault)
	g.Expect(names).To(ConsistOf(srr2NewBasename+".md", srr2Referrer+".md"), "renamed; nothing else added")

	adopted, _ := vault.get(srr2NewBasename + ".md")
	fields := frontmatterOf(adopted)
	before := frontmatterOf(curatePromotedNoteFixturePending())

	for _, key := range []string{
		"type", "situation", "done_when", "red_flags", "triggers", "luhmann", "created", "source", "user", "vault",
	} {
		g.Expect(fields).To(HaveKeyWithValue(key, before[key]), "runbook field %s is unchanged", key)
	}

	g.Expect(fields).To(HaveKeyWithValue("aliases", ContainElement(srr2OldBasename)))
	g.Expect(fields).To(HaveKeyWithValue("skill_hash", srr1Hash(srr1SkillText("curate"))))
	g.Expect(fields).To(HaveKeyWithValue("skill_key", "claude:curate"))
	g.Expect(fields).To(HaveKeyWithValue("skill_source", srr2CurateSkillSource))
	g.Expect(fields).NotTo(HaveKey("pending"))
	g.Expect(skillNoteBodyOf(adopted)).To(Equal(
		preambleNaming(srr2CurateSkillSource) + "\n" + string(srr1SkillText("curate"))))

	referrer, _ := vault.get(srr2Referrer + ".md")
	g.Expect(referrer).To(ContainSubstring("[[" + srr2NewBasename + "]]"))
	g.Expect(referrer).NotTo(ContainSubstring("[[" + srr2OldBasename + "]]"))

	_, oldSidecar := vault.get(srr2OldBasename + ".vec.json")
	_, newSidecar := vault.get(srr2NewBasename + ".vec.json")
	g.Expect(oldSidecar).To(BeFalse(), "the old sidecar was moved")
	g.Expect(newSidecar).To(BeTrue())
	g.Expect(stdout).To(Equal(srr1Vault+"/"+srr2NewBasename+".md\n"), "no offer is left for curate")
}

// Test792a12i1b_RegisterSkills_AdoptKeepsAnUnmodeledKey verifies 792a12i1b: a runbook note whose frontmatter
// carries keys the runbook note model does not define (luhmann_old: "12" and a nested provenance map) still
// carries them, with the same values, after --adopt.
func Test792a12i1b_RegisterSkills_AdoptKeepsAnUnmodeledKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", strings.Replace(curatePromotedNoteFixture(), "vault: personal\n",
		"vault: personal\nluhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps: [a, b]\n", 1))

	_, err := srr2Adopt(vault)
	g.Expect(err).NotTo(HaveOccurred())

	adopted, found := vault.get(srr2NewBasename + ".md")
	g.Expect(found).To(BeTrue(), "the note was adopted")

	fields := frontmatterOf(adopted)
	g.Expect(fields).To(HaveKeyWithValue("luhmann_old", "12"))
	g.Expect(fields).To(HaveKeyWithValue("provenance",
		map[string]any{"origin": "import", "steps": []any{"a", "b"}}))
}

// Test792a12i1c_RegisterSkills_AdoptRefusesAnAnchoredKey verifies 792a12i1c: a runbook note whose skill_hash,
// skill_key, skill_source or pending key carries a YAML anchor, or whose value does, is refused by --adopt naming
// the key, and the vault is byte-identical afterward: the note is neither renamed nor rewritten.
func Test792a12i1c_RegisterSkills_AdoptRefusesAnAnchoredKey(t *testing.T) {
	t.Parallel()

	base := strings.Replace(curatePromotedNoteFixture(), "vault: personal\n", "vault: personal\n"+
		"skill_hash: \"old-hash\"\nskill_key: \"claude:curate\"\nskill_source: \""+srr2CurateSkillSource+"\"\n"+
		"pending: true\n", 1)

	for _, key := range []string{"skill_hash", "skill_key", "skill_source", "pending"} {
		for anchored, mark := range map[string]string{"key": "&anchor " + key + ": ", "value": key + ": &anchor "} {
			t.Run(key+" with an anchored "+anchored, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				vault := newSkillAcceptFixtureVault()
				vault.put(srr2OldBasename+".md", strings.Replace(base, key+": ", mark, 1))
				vault.put(srr2OldBasename+".vec.json", `{"model_id":"old"}`)
				vault.put(srr2Referrer+".md", referencingNoteFixture(srr2OldBasename))
				before := maps.Clone(vault.files)

				_, err := srr2Adopt(vault)

				g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
				g.Expect(err).To(MatchError(ContainSubstring(key)))
				g.Expect(vault.files).To(Equal(before), "neither renamed nor rewritten")
			})
		}
	}
}

// Test792a12i1d_RegisterSkills_AdoptConvertsACRLFNote verifies 792a12i1d: --adopt on a runbook note whose lines
// end in \r\n converts it to LF, renames it, and rewrites the links to it.
func Test792a12i1d_RegisterSkills_AdoptConvertsACRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", crlf(curatePromotedNoteFixture()))
	vault.put(srr2Referrer+".md", referencingNoteFixture(srr2OldBasename))

	_, err := srr2Adopt(vault)
	g.Expect(err).NotTo(HaveOccurred())

	_, oldThere := vault.get(srr2OldBasename + ".md")
	g.Expect(oldThere).To(BeFalse(), "the note was renamed")

	adopted, found := vault.get(srr2NewBasename + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(adopted).NotTo(ContainSubstring("\r"), "the note was converted to LF")
	g.Expect(frontmatterOf(adopted)).To(HaveKeyWithValue("skill_key", "claude:curate"))

	referrer, _ := vault.get(srr2Referrer + ".md")
	g.Expect(referrer).To(ContainSubstring("[[" + srr2NewBasename + "]]"))
	g.Expect(referrer).NotTo(ContainSubstring("[[" + srr2OldBasename + "]]"))
}

// Test792a12i1e_RegisterSkills_AdoptRefusesANoteThatIsNotARunbook verifies 792a12i1e: --adopt on a fact note is
// refused, and the vault is byte-identical afterward.
func Test792a12i1e_RegisterSkills_AdoptRefusesANoteThatIsNotARunbook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.skill-curate.md",
		"---\ntype: fact\nsituation: s\nsubject: sub\npredicate: pred\nobject: obj\n"+
			"luhmann: \"1049\"\ncreated: 2026-09-21\nsource: s\nuser: u\nvault: v\n---\n\nInformation learned: s.\n")
	before := maps.Clone(vault.files)

	_, err := srr2Adopt(vault)

	g.Expect(err).To(MatchError(ContainSubstring("not a runbook")))
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12i1f_RegisterSkills_AdoptRefusesANoteKeyedToAnotherSkill verifies 792a12i1f: --adopt claude:curate on
// the note whose skill_key is claude:route is refused, and the note is not re-keyed: the vault is byte-identical
// afterward.
func Test792a12i1f_RegisterSkills_AdoptRefusesANoteKeyedToAnotherSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	routeNote := strings.Replace(curateSkillNoteFixture(srr1Hash(srr1SkillText("route"))),
		`skill_key: "claude:curate"`, `skill_key: "claude:route"`, 1)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.skill-claude-route.md", routeNote)
	before := maps.Clone(vault.files)

	_, err := srr1Run(vault,
		skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate"), "route": srr1SkillText("route")}),
		cli.SkillRegistrationArgs{Adopt: map[string]string{"claude:curate": "1049"}})

	g.Expect(err).To(MatchError(cli.ErrAdoptTargetKeyedForTest))
	g.Expect(err).To(MatchError(ContainSubstring(`"claude:route"`)))
	g.Expect(vault.files).To(Equal(before), "the note is not re-keyed")
}

// Test792a12i1g_RegisterSkills_AdoptRefusesAnUnscannedKey verifies 792a12i1g: --adopt claude:nosuch=1049, a key
// that no scanned entry has, is refused and nothing is written.
func Test792a12i1g_RegisterSkills_AdoptRefusesAnUnscannedKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", curatePromotedNoteFixture())
	before := maps.Clone(vault.files)

	var writes []string

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
		cli.SkillRegistrationArgs{Adopt: map[string]string{"claude:nosuch": "1049"}},
		func(deps *cli.SkillRegistrationDeps) { recordWrites(deps, vault, &writes) })

	g.Expect(err).To(MatchError(ContainSubstring("names no scanned skill")))
	g.Expect(err).To(MatchError(ContainSubstring("claude:nosuch")))
	g.Expect(writes).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12i1h_RegisterSkills_AdoptRefusesAnAmbiguousReference verifies 792a12i1h: --adopt claude:curate=1049
// where the bare Luhmann ID 1049 is shared by two runbook notes is refused, asking for the full basename, and
// the vault is byte-identical afterward.
func Test792a12i1h_RegisterSkills_AdoptRefusesAnAmbiguousReference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const secondBasename = "1049.2026-09-22.curate-follow-up"

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", curatePromotedNoteFixture())
	vault.put(secondBasename+".md", curatePromotedNoteFixtureAt("1049", "2026-09-22"))
	before := maps.Clone(vault.files)

	_, err := srr2Adopt(vault)

	g.Expect(err).To(MatchError(cli.ErrAdoptAmbiguousRefForTest))
	g.Expect(err).To(MatchError(ContainSubstring(srr2OldBasename)))
	g.Expect(err).To(MatchError(ContainSubstring(secondBasename)))
	g.Expect(err).To(MatchError(ContainSubstring("full basename")))
	g.Expect(vault.files).To(Equal(before))
}

// Test792a12i1i_RegisterSkills_ADryRunAdoptsNothing verifies 792a12i1i: --dry-run with a valid --adopt entry
// leaves note 1049 neither renamed nor rewritten (no write of any kind), and the preview still lists claude:curate
// as an offer to register.
func Test792a12i1i_RegisterSkills_ADryRunAdoptsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put(srr2OldBasename+".md", curatePromotedNoteFixture())
	vault.put(srr2OldBasename+".vec.json", `{"model_id":"old"}`)
	vault.put(srr2Referrer+".md", referencingNoteFixture(srr2OldBasename))
	before := maps.Clone(vault.files)

	var writes []string

	stdout, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
		cli.SkillRegistrationArgs{DryRun: true, Adopt: map[string]string{"claude:curate": "1049"}},
		func(deps *cli.SkillRegistrationDeps) { recordWrites(deps, vault, &writes) })
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(Equal("@claude-user (1)\n" + srr1RegisterOffers("curate")))
	g.Expect(writes).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before), "neither renamed nor rewritten")
}

// Test792a12i1j_RegisterSkills_AdoptRefusesASkillThatPiDisables verifies 792a12i1j: a Pi user skill ping that the
// global settings.json switches off with "!ping" is not a scanned skill, so --adopt pi:ping=1 over runbook note 1
// is refused as naming no scanned skill and nothing is written. The control run, with the same skill and note but
// no "!ping" in the settings, adopts the note, so the refusal is owed to the setting.
func Test792a12i1j_RegisterSkills_AdoptRefusesASkillThatPiDisables(t *testing.T) {
	t.Parallel()

	const noteBasename = "1.2026-09-21.ping-runbook"

	for name, settings := range map[string]string{
		"disabled": `{"skills": ["!ping"]}`,
		"enabled":  `{"skills": []}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put(noteBasename+".md", promotedRunbookFixture("1", "2026-09-21", "never ping blindly"))
			before := maps.Clone(vault.files)

			sourceFS := skillsHomeFixture(nil).
				file(skillRegHome+"/.pi/agent/skills/ping/SKILL.md", string(srr1SkillText("ping"))).
				file(skillRegHome+"/.pi/agent/settings.json", settings)

			var writes []string

			_, err := srr1Run(vault, sourceFS, cli.SkillRegistrationArgs{Adopt: map[string]string{"pi:ping": "1"}},
				func(deps *cli.SkillRegistrationDeps) { recordWrites(deps, vault, &writes) })

			if name == "enabled" {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(vault.files).To(HaveKey(srr1Vault+"/1.2026-09-21.skill-pi-ping.md"), "the control adopts")

				return
			}

			g.Expect(err).To(MatchError(ContainSubstring("names no scanned skill")))
			g.Expect(err).To(MatchError(ContainSubstring("pi:ping")))
			g.Expect(writes).To(BeEmpty())
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a12j1a_RegisterSkills_ASymlinkedUserSkillIsFound verifies 792a12j1a over the real command wiring on a
// scratch home on the real file system (the recorded default-suite deviation, #860, since the promise is about a
// real symlink): ~/.claude/skills/route is a symlink to ~/.claude/engram/skills/route containing SKILL.md, and
// register-skills offers claude:route with the resolved ~/.claude/engram/skills/route/SKILL.md as its source.
func Test792a12j1a_RegisterSkills_ASymlinkedUserSkillIsFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	home := srr2ResolvedTempDir(t, g)

	srr1WriteFile(g, filepath.Join(home, ".claude", "engram", "skills", "route", "SKILL.md"),
		string(srr1SkillText("route")))
	srr1LinkDeployedSkill(g, home, "route")

	stdout, stderr := srr2RunCLI(t, home, home, nil, "register-skills", "--dry-run", "--vault", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("@claude-user (1)\n  would offer: register claude:route (" +
		filepath.Join(home, ".claude", "engram", "skills", "route", "SKILL.md") + ")\n"))
}

// Test792a12j1b_RegisterSkills_ADanglingSymlinkIsSkipped verifies 792a12j1b over the real command wiring on a
// scratch home on the real file system (the recorded default-suite deviation, #860, since the promise is about a
// real dangling symlink): ~/.claude/skills/qa is a symlink whose target does not exist, next to a good skill;
// register-skills makes no offer for qa and does not fail.
func Test792a12j1b_RegisterSkills_ADanglingSymlinkIsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	home := srr2ResolvedTempDir(t, g)

	srr1WriteFile(g, filepath.Join(home, ".claude", "engram", "skills", "route", "SKILL.md"),
		string(srr1SkillText("route")))
	srr1LinkDeployedSkill(g, home, "route")
	g.Expect(os.Symlink(filepath.Join(home, "nowhere", "qa"), filepath.Join(home, ".claude", "skills", "qa"))).
		To(Succeed())

	stdout, stderr := srr2RunCLI(t, home, home, nil, "register-skills", "--dry-run", "--vault", t.TempDir())

	g.Expect(stderr).To(BeEmpty(), "registration does not fail")
	g.Expect(stdout).To(Equal("@claude-user (1)\n  would offer: register claude:route (" +
		filepath.Join(home, ".claude", "engram", "skills", "route", "SKILL.md") + ")\n"))
	g.Expect(stdout).NotTo(ContainSubstring("qa"), "no offer is made for it")
}

// Test792a12j1c_RegisterSkills_AUserCommandInASubdirectory verifies 792a12j1c: ~/.claude/commands/opsx/apply.md
// is scanned as a command named opsx:apply, so register-skills offers claude:cmd:opsx:apply from that file.
func Test792a12j1c_RegisterSkills_AUserCommandInASubdirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sourceFS := skillsHomeFixture(nil).file(skillRegHome+"/.claude/commands/opsx/apply.md", "# Apply\n")

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), sourceFS, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(Equal("@claude-cmd (1)\n" +
		"  would offer: register claude:cmd:opsx:apply (" + skillRegHome + "/.claude/commands/opsx/apply.md)\n"))
}

// Test792a12j1d_RegisterSkills_SyncedSkillsComeFromManifestBuckets verifies 792a12j1d: ~/.claude/skills/synced/
// holding one bucket directory with a parseable manifest.json and skill pdf/SKILL.md, plus a stray file
// .bucket-x, makes register-skills offer pdf as a synced skill (anthropic-skills:pdf, from the bucket's
// pdf/SKILL.md) and ignore the stray file.
func Test792a12j1d_RegisterSkills_SyncedSkillsComeFromManifestBuckets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const bucket = skillRegHome + "/.claude/skills/synced/4227b111_e76eb279"

	sourceFS := skillsHomeFixture(nil).
		file(bucket+"/manifest.json", `{"skills":[{"skillId":"pdf"}]}`).
		file(bucket+"/pdf/SKILL.md", "pdf").
		file(skillRegHome+"/.claude/skills/synced/.bucket-x", "")

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), sourceFS, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(Equal("@synced (1)\n  would offer: register anthropic-skills:pdf (" + bucket + "/pdf/SKILL.md)\n"))
}

// unexported constants.
const (
	srr2CurateSkillSource = "~/.claude/engram/skills/curate/SKILL.md"
	srr2NewBasename       = "1049.2026-09-21.skill-claude-curate"
	srr2NoteRef           = "1049"
	srr2OldBasename       = "1049.2026-09-21.skill-curate"
	srr2PromptAnswers     = 3
	srr2Referrer          = "2000.2026-09-22.some-other-note"
)

// srr2Adopt runs the registration over vault with the curate skill scanned and --adopt claude:curate=1049.
func srr2Adopt(vault *skillAcceptFixtureVault) (string, error) {
	return srr1Run(vault, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}),
		cli.SkillRegistrationArgs{Adopt: map[string]string{"claude:curate": srr2NoteRef}})
}

// srr2ResolvedTempDir is t.TempDir() with its symlinks resolved, so paths a scan reports compare equal.
func srr2ResolvedTempDir(t *testing.T, g Gomega) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(t.TempDir())
	g.Expect(err).NotTo(HaveOccurred())

	return resolved
}

// srr2RunCLI runs `engram <args>` over the real command wiring with the home, working directory, git and
// terminal pinned to scratch values (stdin, when set, answers prompts on a terminal), and returns stdout and
// stderr.
func srr2RunCLI(t *testing.T, cwd, home string, stdin *strings.Reader, args ...string) (string, string) {
	t.Helper()

	return executeCapturingBoth(t, append([]string{"engram"}, args...), func(deps *cli.Deps) {
		deps.UserHomeDir = func() (string, error) { return home, nil }
		deps.Getwd = func() (string, error) { return cwd, nil }
		deps.Commander = scriptedGit{}
		deps.Embed = skillAcceptFakeEmbedder{}

		if stdin != nil {
			deps.IsTerminal = func() bool { return true }
			deps.Stdin = stdin
		}
	})
}
