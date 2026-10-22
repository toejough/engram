package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792a12s1a_RegisterSkills_AVersion1FileFailsLoudly verifies 792a12s1a: a skill-registrations.json that reads
// {"schema_version":1,"declined":{"route":"<h>"}} makes register-skills report an error and write nothing, whether
// the run is plain, answers an offer with --accept or prompts in a terminal.
func Test792a12s1a_RegisterSkills_AVersion1FileFailsLoudly(t *testing.T) {
	t.Parallel()

	declines := `{"schema_version":1,"declined":{"route":"` + srr1Hash(srr1SkillText("route")) + `"}}`

	srr6ExpectLoudFailure(t, declines)
}

// Test792a12s1b_RegisterSkills_AnUnknownSchemaVersionFailsLoudly verifies 792a12s1b: a skill-registrations.json at
// schema_version 3 (and at any other version above 2) makes register-skills report an error and write nothing.
func Test792a12s1b_RegisterSkills_AnUnknownSchemaVersionFailsLoudly(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"3", "4", "100"} {
		t.Run("schema_version "+version, func(t *testing.T) {
			t.Parallel()

			srr6ExpectLoudFailure(t, `{"schema_version":`+version+`,"declined":{"claude:route":"h"}}`)
		})
	}
}

// Test792a12s1c_RegisterSkills_AMissingFileIsNoDeclines verifies 792a12s1c: with no skill-registrations.json,
// register-skills offers every scanned skill that has no note (here curate and recall, while route already has its
// note), a dry run and a plain run both leave the file absent, and the plain run reports the two offers waiting.
func Test792a12s1c_RegisterSkills_AMissingFileIsNoDeclines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skills := map[string][]byte{
		"curate": srr1SkillText("curate"), "recall": srr1SkillText("recall"), "route": srr1SkillText("route"),
	}
	vault := srr5Vault(srr5Note{
		key: "claude:route", hash: srr1Hash(skills["route"]), source: "~/.claude/engram/skills/route/SKILL.md",
	})
	_, declinesBefore := vault.get(srr5DeclinesFile)
	g.Expect(declinesBefore).To(BeFalse(), "the Given: no skill-registrations.json")

	notesBefore := srr5Notes(vault)

	preview, previewErr := srr1Run(vault, skillsHomeFixture(skills), cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(previewErr).NotTo(HaveOccurred())
	g.Expect(srr3Offers(preview)).To(Equal([]string{
		srr3Register("claude:curate", skillRegSourcePath("curate")),
		srr3Register("claude:recall", skillRegSourcePath("recall")),
	}))

	plain, plainErr := srr1Run(vault, skillsHomeFixture(skills), cli.SkillRegistrationArgs{})
	g.Expect(plainErr).NotTo(HaveOccurred())
	g.Expect(plain).To(ContainSubstring("engram: 2 skill runbook offers awaiting an answer: @claude-user 2"))

	_, declinesAfter := vault.get(srr5DeclinesFile)
	g.Expect(declinesAfter).To(BeFalse(), "no decline file is written for offers nobody answered")
	g.Expect(srr5Notes(vault)).To(Equal(notesBefore), "no note is written either")
}

// Test792a12t1a_Update_AnInteractiveUpdateOffersRegistration verifies 792a12t1a: engram update run in a terminal
// on a machine with Claude Code and Pi, where the shipped skill recall has no note, deploys recall to both harnesses
// and then prompts to register it (end of input answers nothing, so nothing is written).
func Test792a12t1a_Update_AnInteractiveUpdateOffersRegistration(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	world := srr6NewUpdateWorld()
	world.machine.fileSystem.seedDir(iu2PiDir)

	outcome := world.update(t, cli.UpdateArgs{}, srr5Terminal(""))
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())

	deployed := world.machine.fileSystem
	g.Expect(deployed.kindOf(iu2ClaudeSkills + "/recall")).To(Equal("link"))
	g.Expect(deployed.text(iu2ClaudeSkills + "/recall/SKILL.md")).To(Equal(string(srr1SkillText("recall"))))
	g.Expect(deployed.kindOf(iu2PiSkills + "/recall")).To(Equal("link"))
	g.Expect(deployed.text(iu2PiSkills + "/recall/SKILL.md")).To(Equal(string(srr1SkillText("recall"))))

	g.Expect(outcome.stdout).To(ContainSubstring(
		"Register skill `claude:recall` (" + iu2ClaudeRoot + "/skills/recall/SKILL.md) as a vault runbook? [y/N] "))
	g.Expect(world.vault.files).To(BeEmpty(), "an unanswered prompt writes no note and records no decline")
}

// Test792a12t1b_Update_ANonInteractiveUpdateOnlyReports verifies 792a12t1b: engram update with stdin not a
// terminal deploys the skills, writes no vault note and no decline record, and reports the waiting offers by scope
// selector and count with the commands that answer them.
func Test792a12t1b_Update_ANonInteractiveUpdateOnlyReports(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	world := srr6NewUpdateWorld()
	world.machine.fileSystem.seedDir(iu2PiDir)

	outcome := world.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())

	deployed := world.machine.fileSystem
	g.Expect(deployed.kindOf(iu2ClaudeSkills + "/recall")).To(Equal("link"))
	g.Expect(deployed.kindOf(iu2PiSkills + "/recall")).To(Equal("link"))

	g.Expect(world.vault.files).To(BeEmpty(), "no vault note and no skill-registrations.json")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("Register skill"), "nothing is prompted")
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(
		"engram: 1 skill runbook offer awaiting an answer: @claude-user 1 — run " +
			"`engram register-skills` in a terminal, or `engram register-skills --accept <key|prefix*|@scope>` / " +
			"`--decline <key|prefix*|@scope>`"))
}

// Test792a12t1c_Update_ADryRunPreviewsTheOffersToo verifies 792a12t1c: with the user skill c4 outstanding as an
// offer, engram update --dry-run lists "would offer: register claude:c4" in the same report as the sync plan for
// the shipped recall skill, and changes nothing on the machine or in the vault.
func Test792a12t1c_Update_ADryRunPreviewsTheOffersToo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	world := srr6NewUpdateWorld()
	world.machine.fileSystem.seedFile(iu2ClaudeSkills+"/c4/SKILL.md", string(srr1SkillText("c4")))

	before := world.machine.fileSystem.snapshot("")

	outcome := world.update(t, cli.UpdateArgs{DryRun: true}, srr5Terminal(""))
	g.Expect(outcome.err).NotTo(HaveOccurred())

	lines := iu2Lines(outcome.stdout)
	offer := "  would offer: register claude:c4 (" + iu2ClaudeSkills + "/c4/SKILL.md)"
	plan := "    agent-instructions/skills/recall/ → ~/.claude/skills/recall/  (1 file)"

	g.Expect(lines).To(ContainElement(offer))
	g.Expect(lines).To(ContainElement(plan), "the sync operations are in the same report")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("Register skill"), "a dry run prompts nothing")
	g.Expect(world.machine.fileSystem.snapshot("")).To(Equal(before), "nothing is written on the machine")
	g.Expect(world.vault.files).To(BeEmpty(), "nothing is written to the vault")
}

// Test792a12t1e_Update_ARegistrationFailureDoesNotRollBackTheDeploy verifies 792a12t1e: with a
// skill-registrations.json that registration refuses (schema version 1), engram update still deploys the skills,
// returns no error and exits 0, and its report holds a "skill registration error:" line.
func Test792a12t1e_Update_ARegistrationFailureDoesNotRollBackTheDeploy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	world := srr6NewUpdateWorld()
	world.vault.put(srr5DeclinesFile, `{"schema_version":1,"declined":{"route":"h"}}`)

	outcome := world.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred(), "the update does not fail")
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(outcome.exits).To(Equal([]int{0}))

	errorLines := make([]string, 0, 1)

	for _, line := range iu2Lines(outcome.stdout) {
		if strings.HasPrefix(line, "skill registration error: ") {
			errorLines = append(errorLines, line)
		}
	}

	g.Expect(errorLines).To(HaveLen(1))

	if len(errorLines) != 1 {
		return
	}

	g.Expect(errorLines[0]).To(ContainSubstring("register-skills: "))
	g.Expect(errorLines[0]).To(ContainSubstring("skill-registrations.json"))

	deployed := world.machine.fileSystem
	g.Expect(deployed.kindOf(iu2ClaudeSkills+"/recall")).To(Equal("link"), "the deployed skill remains")
	g.Expect(deployed.text(iu2ClaudeSkills + "/recall/SKILL.md")).To(Equal(string(srr1SkillText("recall"))))
	g.Expect(world.vault.files).To(Equal(map[string]string{
		srr1Vault + "/" + srr5DeclinesFile: `{"schema_version":1,"declined":{"route":"h"}}`,
	}), "the registration wrote nothing")
}

// Test792a12u1a_Amend_PreservesTheIdentityFields verifies 792a12u1a: engram amend on a note carrying skill_hash,
// skill_key and skill_source, both with --clear-pending and with a content change, writes the note with all three
// byte-unchanged.
func Test792a12u1a_Amend_PreservesTheIdentityFields(t *testing.T) {
	t.Parallel()

	notPending := false
	cases := map[string]struct {
		args    cli.AmendArgs
		changed func(g Gomega, written string)
	}{
		"clear-pending": {
			args: cli.AmendArgs{Vault: "/vault", Target: "1aa", Pending: &notPending},
			changed: func(g Gomega, written string) {
				g.Expect(written).NotTo(ContainSubstring("pending: true"))
			},
		},
		"a content amend": {
			args: cli.AmendArgs{
				Vault: "/vault", Target: "1aa",
				Situation: "brainstorming a design before building", DoneWhen: "the design is approved",
			},
			changed: func(g Gomega, written string) {
				g.Expect(written).To(ContainSubstring("situation: brainstorming a design before building"))
			},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := skillSurvivalNote()

			var (
				written []byte
				writes  int
			)

			err := cli.ExportRunAmend(
				t.Context(), testCase.args, runbookAmendDeps([]byte(before), &written, &writes), &bytes.Buffer{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(writes).To(Equal(1))

			testCase.changed(g, string(written))
			expectSkillIdentityUnchanged(g, before, string(written))
		})
	}
}

// Test792a12u1b_Resituate_PreservesTheIdentityFields verifies 792a12u1b: engram resituate on a skill note refuses
// (a runbook cannot be resituated, 792a4l1b), so the note in the vault is unchanged and all three identity fields
// are byte-identical.
func Test792a12u1b_Resituate_PreservesTheIdentityFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	stored := before
	writes := 0

	deps := cli.ResituateDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: skillSurvivalNoteName, LuhmannID: "1a"}}, nil
		},
		Read:     func(string) ([]byte, error) { return []byte(stored), nil },
		Write:    func(_ string, data []byte) error { writes++; stored = string(data); return nil },
		Embedder: skillAcceptFakeEmbedder{},
		Now:      func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.ResituateArgs{Vault: "/vault", Note: "1a", Situation: "a new situation"}

	err := cli.RunResituate(t.Context(), args, deps, &bytes.Buffer{})
	g.Expect(err).To(MatchError(ContainSubstring("unknown note type")))
	g.Expect(writes).To(Equal(0), "a skill note is never written")
	g.Expect(stored).To(Equal(before))
	expectSkillIdentityUnchanged(g, before, stored)
}

// Test792a12u1c_ReparentLuhmann_PreservesTheIdentityFields verifies 792a12u1c: engram update --reparent-luhmann
// answering that the skill note 12 is a continuation of note 7 renames it to 7a and leaves skill_hash, skill_key
// and skill_source byte-unchanged in the renamed note.
func Test792a12u1c_ReparentLuhmann_PreservesTheIdentityFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(skillSurvivalNote(), `luhmann: "1a"`, `luhmann: "12"`, 1)
	files, renames := srr6Reparent(t, map[string]string{srr6SecondNote: before})

	g.Expect(renames).To(HaveKeyWithValue("/vault/"+srr6SecondNote, "/vault/"+srr6RenamedNote))

	after := string(files["/vault/"+srr6RenamedNote])
	g.Expect(after).To(ContainSubstring(`luhmann: "7a"`), "the note was renamed and its id rewritten")
	expectSkillIdentityUnchanged(g, before, after)
}

// Test792a12u1d_ReparentLuhmann_AWikilinkRewritePreservesTheIdentityFields verifies 792a12u1d: when a re-parenting
// rename rewrites a wikilink inside the frontmatter of the skill note 20 (its red_flags name note 12), the skill
// note's skill_hash, skill_key and skill_source are byte-unchanged.
func Test792a12u1d_ReparentLuhmann_AWikilinkRewritePreservesTheIdentityFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(
		strings.ReplaceAll(skillSurvivalNote(), skillSurvivalLinkTarget, strings.TrimSuffix(srr6SecondNote, ".md")),
		`luhmann: "1a"`, `luhmann: "20"`, 1)
	files, _ := srr6Reparent(t, map[string]string{srr6UnrelatedNote: before})

	after := string(files["/vault/"+srr6UnrelatedNote])
	g.Expect(after).To(ContainSubstring("red_flags:\n    - skipped the checklist in [["+
		strings.TrimSuffix(srr6RenamedNote, ".md")+"]]"), "the frontmatter wikilink was rewritten")
	expectSkillIdentityUnchanged(g, before, after)
}

// Test792a12u1e_BackfillIdentity_LeavesASkillNoteAndItsIdentityFieldsAlone verifies 792a12u1e as modified in
// 861a4zw2: engram update --backfill-identity over a vault holding a skill note without repo, user and vault and a
// fact note without them stamps the fact note only; the skill note is not rewritten, so its three skill fields are
// unchanged.
func Test792a12u1e_BackfillIdentity_LeavesASkillNoteAndItsIdentityFieldsAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const factName = "2.2026-01-01.a-fact.md"

	before := strings.Replace(skillSurvivalNote(), "user: agent@example.com\nvault: personal\n", "", 1)
	vault := newSkillSurvivalFS(map[string]string{
		skillSurvivalNoteName: before,
		factName: "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nluhmann: \"2\"\n" +
			"created: 2026-01-01\nsource: agent\n---\n\nbody\n",
	})

	deps := cli.IdentityDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     vault.listMD,
		ReadFile:   vault.readFile,
		WriteFile:  vault.writeFile,
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}

	stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1), "the fact note was stamped")
	g.Expect(vault.writes).To(ConsistOf("/vault/"+factName), "and nothing else was written")
	g.Expect(vault.content(factName)).To(ContainSubstring("user: agent@example.com"))
	g.Expect(vault.content(skillSurvivalNoteName)).To(Equal(before))
	expectSkillIdentityUnchanged(g, before, vault.content(skillSurvivalNoteName))
}

// Test792a12u1f_Vocabulary_RewritesPreserveTheIdentityFields verifies 792a12u1f: vocabulary assignment, the tag
// rewrite that clears a removed term, the legacy vocab: cleanup and definition tagging each leave a skill note's
// skill_hash, skill_key and skill_source byte-unchanged (the first three rewrite its tags; definition tagging never
// writes it).
func Test792a12u1f_Vocabulary_RewritesPreserveTheIdentityFields(t *testing.T) {
	t.Parallel()

	t.Run("vocabulary assignment", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		before := skillSurvivalNote()
		after := cli.ExportWriteVocabAssignment(before, []string{"new-term", "other-term"})

		g.Expect(after).To(ContainSubstring("vocab/new-term"), "the assignment rewrote the tags")
		expectSkillIdentityUnchanged(g, before, after)
	})

	t.Run("a removed term cleared from the tags", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		before := skillSurvivalNote()
		vault := newSkillSurvivalFS(map[string]string{skillSurvivalNoteName: before})
		deps := cli.VocabDeps{
			ListMD: vault.listMD, ReadFile: vault.readFile, WriteFile: vault.writeFile,
			LogWarning: func(string, ...any) {},
		}

		g.Expect(cli.ExportClearRemovedTermsFromMembers(deps, "/vault", []string{"old-term"})).To(Succeed())

		after := vault.content(skillSurvivalNoteName)
		g.Expect(after).NotTo(ContainSubstring("vocab/old-term"), "the rewrite removed the tag")
		expectSkillIdentityUnchanged(g, before, after)
	})

	t.Run("the legacy vocab line removed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		before := strings.Replace(skillSurvivalNote(), "tier: L2\n", "tier: L2\nvocab: old-term\n", 1)
		after, changed := cli.ExportStripLegacyVocabChannel(before)

		g.Expect(changed).To(BeTrue())
		g.Expect(after).NotTo(ContainSubstring("vocab: old-term"))
		expectSkillIdentityUnchanged(g, before, after)
	})

	t.Run("definition tagging", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		before := skillSurvivalNote()
		vault := newSkillSurvivalFS(map[string]string{skillSurvivalNoteName: before})
		deps := cli.VocabDeps{
			Lock:   func(string) (func(), error) { return func() {}, nil },
			ListMD: vault.listMD, ReadFile: vault.readFile, WriteFile: vault.writeFile,
			LogWarning: func(string, ...any) {},
		}

		err := cli.ExportRunVocabTagDefinitions(t.Context(), "/vault", deps, &bytes.Buffer{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(vault.writes).To(BeEmpty(), "definition tagging writes definition notes only")
		expectSkillIdentityUnchanged(g, before, vault.content(skillSurvivalNoteName))
	})
}

// Test792a12u1g_Learn_OrdinaryCaptureCarriesNoSkillFields verifies 792a12u1g: a runbook, a fact and a feedback
// note written by engram learn (as its commands build their arguments) carry no skill_hash, skill_key or
// skill_source, and engram learn runbook refuses --skill-hash, --skill-key and --skill-source as unknown flags,
// leaving its vault untouched.
func Test792a12u1g_Learn_OrdinaryCaptureCarriesNoSkillFields(t *testing.T) {
	t.Parallel()

	t.Run("written notes", func(t *testing.T) {
		t.Parallel()

		captures := map[string]cli.LearnArgs{
			"runbook": {
				Type: "runbook", Slug: "release-flow", Vault: "/vault", Source: "test", Position: "top",
				Situation: "releasing a module", DoneWhen: "tag pushed", Body: "1. tag\n",
			},
			"fact": {
				Type: "fact", Slug: "a-fact", Vault: "/vault", Source: "test", Position: "top",
				Situation: "s", Subject: "a", Predicate: "b", Object: "c",
			},
			"feedback": {
				Type: "feedback", Slug: "a-lesson", Vault: "/vault", Source: "test", Position: "top",
				Situation: "s", Behavior: "b", Impact: "i", Action: "act",
			},
		}

		for name, args := range captures {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				var written []byte

				deps := cli.LearnDeps{
					DetectRepo: func(context.Context) string { return "" },
					DetectUser: func(context.Context) string { return "" },
					Now:        func() time.Time { return time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) },
					Getenv:     func(string) string { return "" },
					ListIDs:    func(string) ([]string, error) { return nil, nil },
					Lock:       func(string) (func(), error) { return func() {}, nil },
					WriteNew:   func(_ string, data []byte) error { written = data; return nil },
				}

				err := cli.RunLearn(t.Context(), args, deps, &bytes.Buffer{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(written).NotTo(BeEmpty())

				for _, field := range skillIdentityKeys {
					g.Expect(string(written)).NotTo(ContainSubstring(field))
				}
			})
		}
	})

	for _, flag := range []string{"--skill-hash", "--skill-key", "--skill-source"} {
		t.Run("the "+flag+" flag is refused", func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := filepath.Join(t.TempDir(), "vault")

			var stdout, stderr bytes.Buffer

			result, err := targ.Execute([]string{
				"engram", "learn", "runbook", "--vault", vault, "--source", "test", "--slug", "x",
				"--situation", "s", "--done-when", "d", flag, "x",
			}, cli.Targets(newTestDeps(&stdout, &stderr))...)

			g.Expect(err).To(HaveOccurred())
			g.Expect(result.Output).To(ContainSubstring("flag provided but not defined: " + flag))
			g.Expect(vault).NotTo(BeADirectory(), "nothing was written")
		})
	}
}

// Test792a12v1a_Query_ASkillNoteSurfacesLikeACapturedOne verifies 792a12v1a: a non-pending skill note with the
// triggers curate and pending offers, queried with --text "please curate the offers", is the first item with
// provenance trigger and kind runbook, as an otherwise identical captured runbook with the same triggers is.
func Test792a12v1a_Query_ASkillNoteSurfacesLikeACapturedOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		skillName    = "1049.2026-09-21.skill-claude-curate.md"
		capturedName = "1049.2026-09-21.curate-pending-offers.md"
	)

	skillNote := curateSkillNoteFixture("abc123")
	capturedNote := srr6WithoutSkillFields(skillNote)

	results := map[string]queryParsedItem{}

	for name, notes := range map[string]map[string]string{
		"skill":    {skillName: skillNote},
		"captured": {capturedName: capturedNote},
	} {
		memFS := newInMemoryFS()
		plantNoteWithSidecar(t, memFS, srr1Vault, triggerFeedbackName, triggerFeedbackNote)

		for noteName, content := range notes {
			plantNoteWithSidecar(t, memFS, srr1Vault, noteName, content)
		}

		parsed, _ := runTriggerQuery(t, memFS, srr1Vault, cli.QueryArgs{
			Phrases: triggerPhrases, Text: "please curate the offers", Limit: 20,
		})
		g.Expect(parsed.Items).NotTo(BeEmpty(), name)

		if len(parsed.Items) == 0 {
			return
		}

		results[name] = parsed.Items[0]
	}

	g.Expect(results["skill"].Path).To(Equal(skillName), "the skill note is first")
	g.Expect(results["skill"].Kind).To(Equal("runbook"))
	g.Expect(results["skill"].Provenances).To(ContainElement("trigger"))
	g.Expect(results["captured"].Path).To(Equal(capturedName), "control: the captured runbook is first")
	g.Expect(results["skill"].Kind).To(Equal(results["captured"].Kind))
	g.Expect(results["skill"].Provenances).To(Equal(results["captured"].Provenances))
}

// Test792a12v1b_Show_TheBodyPreambleMarksTheSource verifies 792a12v1b: after a skill, a command and a Pi prompt
// template are registered, engram show on each note prints a body whose first line is "> Mirrors skill `<source>`."
// naming the file the note mirrors, which is the note's home-relative skill_source.
func Test792a12v1b_Show_TheBodyPreambleMarksTheSource(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		key, source string
		fsys        func() *fakeSkillFS
	}{
		"a skill": {
			key: "claude:curate", source: "~/.claude/engram/skills/curate/SKILL.md",
			fsys: func() *fakeSkillFS { return skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}) },
		},
		"a command": {
			key: "claude:cmd:audit", source: "~/.claude/commands/audit.md",
			fsys: func() *fakeSkillFS {
				return skillsHomeFixture(nil).file(skillRegHome+"/.claude/commands/audit.md", "Audit the session.\n")
			},
		},
		"a Pi prompt template": {
			key: "pi-prompt:review", source: "~/.pi/agent/prompts/review.md",
			fsys: func() *fakeSkillFS { return srr3PiHome().file(srr3PiAgent+"/prompts/review.md", "# Review\n") },
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			note := srr6Register(t, g, testCase.fsys(), testCase.key)
			g.Expect(note.frontmatter.SkillSource).To(Equal(testCase.source))
			g.Expect(note.shownBody).To(HavePrefix("> Mirrors skill `"+note.frontmatter.SkillSource+"`.\n"),
				"the body begins with the line naming the file")
			g.Expect(note.shownBody).To(HavePrefix("> Mirrors skill `" + testCase.source + "`.\n"))
		})
	}
}

// Test792a12v1c_Show_ANonEngramSkillsPreambleNamesTheRealFile verifies 792a12v1c: the user skill
// ~/.claude/skills/c4/SKILL.md, registered, shows a preamble naming ~/.claude/skills/c4/SKILL.md and no
// agent-instructions/skills/ path.
func Test792a12v1c_Show_ANonEngramSkillsPreambleNamesTheRealFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(nil).file(skillRegHome+"/.claude/skills/c4/SKILL.md", "---\nname: c4\n---\n\nDiagram.\n")

	note := srr6Register(t, g, fsys, "claude:c4")

	g.Expect(srr6FirstLine(note.shownBody)).To(Equal("> Mirrors skill `~/.claude/skills/c4/SKILL.md`."))
	g.Expect(note.shown).NotTo(ContainSubstring("agent-instructions/skills/"))
}

// Test792a12v1d_Show_AnEngramInstalledSkillsPreambleNamesTheDeployedFile verifies 792a12v1d: claude:route, whose
// resolved source is ~/.claude/engram/skills/route/SKILL.md (linked from ~/.claude/skills/route), registered,
// shows a preamble naming that deployed file and not agent-instructions/skills/route/SKILL.md.
func Test792a12v1d_Show_AnEngramInstalledSkillsPreambleNamesTheDeployedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")})

	note := srr6Register(t, g, fsys, "claude:route")

	g.Expect(srr6FirstLine(note.shownBody)).To(Equal("> Mirrors skill `~/.claude/engram/skills/route/SKILL.md`."))
	g.Expect(note.shown).NotTo(ContainSubstring("agent-instructions/skills/route/SKILL.md"))
}

// Test792a12v1e_Registration_APendingSkillNoteIsAPendingOffer verifies 792a12v1e: a skill note just registered
// carries the line "pending: true", so the scan curation runs over the vault (every note with that line) lists it,
// and engram query over the vault reports pending_offers: true while leaving the note out of its items.
func Test792a12v1e_Registration_APendingSkillNoteIsAPendingOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := srr6Register(t, g, skillsHomeFixture(map[string][]byte{"curate": srr1SkillText("curate")}), "claude:curate")
	g.Expect(note.frontmatter.Pending).To(BeTrue())

	g.Expect(srr6PendingNotes(note.vault)).To(Equal([]string{note.name}), "curation's scan lists the note")

	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, srr1Vault, note.name, note.content)
	plantNoteWithSidecar(t, memFS, srr1Vault, triggerFeedbackName, triggerFeedbackNote)

	parsed, raw := runTriggerQuery(t, memFS, srr1Vault, cli.QueryArgs{
		Phrases: triggerPhrases, Text: "please curate the offers", Limit: 20,
	})

	g.Expect(raw).To(ContainSubstring("pending_offers: true"))

	paths := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		paths = append(paths, item.Path)
	}

	g.Expect(paths).To(ContainElement(triggerFeedbackName), "control: an ordinary note is returned")
	g.Expect(paths).NotTo(ContainElement(note.name), "a pending offer is not returned as a result")
}

// unexported constants.
const (
	srr6RenamedNote   = "7a.2026-01-02.second-note.md"
	srr6SecondNote    = "12.2026-01-02.second-note.md"
	srr6UnrelatedNote = "20.2026-01-03.unrelated-note.md"
)

// srr6FileInfo is the fs.FileInfo of one entry of the in-memory machine filesystem.
type srr6FileInfo struct {
	name string
	info update.FileInfo
}

func (i srr6FileInfo) IsDir() bool { return i.info.IsDir() }

func (i srr6FileInfo) ModTime() time.Time { return time.Time{} }

func (i srr6FileInfo) Mode() fs.FileMode {
	mode := i.info.Mode()
	if i.info.IsDir() {
		mode |= fs.ModeDir
	}

	return mode
}

func (i srr6FileInfo) Name() string { return i.name }

func (i srr6FileInfo) Size() int64 { return 0 }

func (i srr6FileInfo) Sys() any { return nil }

// srr6RegisteredNote is a skill note registration wrote, with what engram show printed for it.
type srr6RegisteredNote struct {
	vault       *skillAcceptFixtureVault
	name        string
	content     string
	frontmatter skillAcceptFrontmatterProbe
	shown       string
	shownBody   string
}

// srr6SourceFS reads the in-memory machine filesystem as the SkillSourceFS that registration scans, so the skills an
// update deployed are the skills registration sees.
type srr6SourceFS struct{ machine *iu2FS }

func (s srr6SourceFS) Lstat(path string) (fs.FileInfo, error) {
	info, err := s.machine.Lstat(path)
	if err != nil {
		return nil, err
	}

	return srr6FileInfo{name: filepath.Base(path), info: info}, nil
}

func (s srr6SourceFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := s.machine.ReadDir(path)
	if err != nil {
		return nil, err
	}

	listed := make([]fs.DirEntry, 0, len(entries))

	for _, entry := range entries {
		info, infoErr := s.Lstat(filepath.Join(path, entry.Name()))
		if infoErr != nil {
			return nil, infoErr
		}

		listed = append(listed, fs.FileInfoToDirEntry(info))
	}

	return listed, nil
}

func (s srr6SourceFS) ReadFile(path string) ([]byte, error) {
	return s.machine.ReadFile(path)
}

func (s srr6SourceFS) Readlink(path string) (string, error) {
	return s.machine.ReadLink(path)
}

// srr6UpdateWorld is a machine an update runs on (Claude Code under the home, the shipped recall skill in the
// source) with the vault registration writes to.
type srr6UpdateWorld struct {
	machine *iu2Machine
	vault   *skillAcceptFixtureVault
}

// update runs `engram update` once on the machine with skill registration configured over the machine's own
// filesystem. As the installed binary does, the binary the parent starts inherits the parent's environment and runs
// the rest of the update, registration included.
func (w *srr6UpdateWorld) update(
	t *testing.T, args cli.UpdateArgs, customize ...func(*cli.SkillRegistrationDeps),
) iu2Outcome {
	t.Helper()

	machine := w.machine
	registration := skillRegistrationDepsFor(w.vault, srr6SourceFS{machine: machine.fileSystem})

	for _, adjust := range customize {
		adjust(&registration)
	}

	var stdout bytes.Buffer

	exits := []int{}
	childErrs := []error{}

	machine.spawn.run = func(_ string, _ []string, env []string) (int, error) {
		vars := maps.Clone(machine.env.vars)

		for _, entry := range env {
			key, value, _ := strings.Cut(entry, "=")
			vars[key] = value
		}

		childEnv := iu1Env{home: machine.env.home, cwd: machine.env.cwd, vars: vars}
		childDeps := cli.ExportNewUpdateDepsWithSpawnAndSkillReg(
			machine.fileSystem, machine.cmd, childEnv, &iu1Spawner{}, func(int) {}, registration)

		childErr := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{WithGuidance: args.WithGuidance}, childDeps, &stdout)
		if childErr != nil {
			childErrs = append(childErrs, childErr)

			return 1, nil
		}

		return 0, nil
	}

	deps := cli.ExportNewUpdateDepsWithSpawnAndSkillReg(
		machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(code int) { exits = append(exits, code) },
		registration)

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return iu2Outcome{err: err, stdout: stdout.String(), exits: exits, childErrs: childErrs}
}

// srr6ExpectLoudFailure asserts that register-skills over a vault whose skill-registrations.json reads
// declinesBody reports the schema-version error and leaves the vault byte-unchanged, for a plain run, a run with
// --accept and a run answering at a terminal.
func srr6ExpectLoudFailure(t *testing.T, declinesBody string) {
	t.Helper()

	runs := map[string]struct {
		args      cli.SkillRegistrationArgs
		customize []func(*cli.SkillRegistrationDeps)
	}{
		"a plain run":        {},
		"an explicit accept": {args: cli.SkillRegistrationArgs{Accept: []string{"claude:route"}}},
		"a terminal run":     {customize: []func(*cli.SkillRegistrationDeps){srr5Terminal("y\n")}},
	}

	for name, run := range runs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put(srr5DeclinesFile, declinesBody)

			before := maps.Clone(vault.files)
			fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")})

			stdout, err := srr1Run(vault, fsys, run.args, run.customize...)

			g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
			g.Expect(err).To(MatchError(ContainSubstring("delete " + srr1Vault + "/" + srr5DeclinesFile)))
			g.Expect(stdout).To(BeEmpty(), "no offer is shown or answered")
			g.Expect(vault.files).To(Equal(before), "nothing is written")
		})
	}
}

// srr6FirstLine is the first line of text.
func srr6FirstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")

	return line
}

// srr6NewUpdateWorld is a machine with Claude Code under the home, the shipped skill recall in the source of the
// engram module checkout, and an empty vault at the path the update resolves.
func srr6NewUpdateWorld() *srr6UpdateWorld {
	machine := iu2NewMachine()
	machine.env.vars["ENGRAM_VAULT_PATH"] = srr1Vault
	machine.skill("recall", "SKILL.md", string(srr1SkillText("recall")))

	return &srr6UpdateWorld{machine: machine, vault: newSkillAcceptFixtureVault()}
}

// srr6PendingNotes lists the notes of vault holding the line "pending: true", the scan curation runs
// (grep -l '^pending: true$' <vault>/*.md).
func srr6PendingNotes(vault *skillAcceptFixtureVault) []string {
	names, err := vault.ListMD(srr1Vault)
	if err != nil || names == nil {
		return []string{}
	}

	pending := make([]string, 0, len(names))

	for _, name := range names {
		content, _ := vault.get(name)

		if slices.Contains(strings.Split(content, "\n"), "pending: true") {
			pending = append(pending, name)
		}
	}

	return pending
}

// srr6Register accepts the offer for key over fsys, in a fresh vault, and returns the note registration wrote with
// what engram show prints for it.
func srr6Register(t *testing.T, g Gomega, fsys *fakeSkillFS, key string) srr6RegisteredNote {
	t.Helper()

	vault := newSkillAcceptFixtureVault()

	_, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{Accept: []string{key}})
	g.Expect(err).NotTo(HaveOccurred())

	names := srr5Notes(vault)
	g.Expect(names).To(HaveLen(1), "registration wrote one note")

	if err != nil || len(names) != 1 {
		return srr6RegisteredNote{vault: vault}
	}

	content, _ := vault.get(names[0])
	deps := cli.ShowDeps{
		Scan: func(root string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, root) },
		Read: vault.ReadFile,
	}

	var shown bytes.Buffer

	showErr := cli.RunShow(t.Context(), cli.ShowArgs{Ref: names[0], VaultPath: srr1Vault}, deps, &shown)
	g.Expect(showErr).NotTo(HaveOccurred())

	_, body, _ := strings.Cut(strings.TrimPrefix(shown.String(), "---\n"), "\n---\n")

	return srr6RegisteredNote{
		vault: vault, name: names[0], content: content, frontmatter: parseSkillAcceptFrontmatter(g, content),
		shown: shown.String(), shownBody: strings.TrimLeft(body, "\n"),
	}
}

// srr6Reparent plants the notes of planted (name to content) over the three notes of the re-parenting fixture,
// derives the candidates, and applies the answer that note 12 is a continuation of note 7. It returns the vault
// files and the renames after the apply.
func srr6Reparent(t *testing.T, planted map[string]string) (map[string][]byte, map[string]string) {
	t.Helper()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()

	for name, content := range planted {
		files["/vault/"+name] = []byte(content)
	}

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)
	answers := map[string]any{
		"reparenting": []map[string]string{{"note": "12", "position": "continuation", "target": "7"}},
		"fingerprint": fingerprint,
	}

	encoded, marshalErr := json.Marshal(answers)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	files["/answers.json"] = encoded

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "/answers.json", false, deps, &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	return files, renames
}

// srr6WithoutSkillFields is the note text with its three skill identity lines and its preamble removed: the
// captured runbook that has the same fields and triggers otherwise.
func srr6WithoutSkillFields(note string) string {
	lines := strings.Split(note, "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.HasPrefix(line, "skill_hash:") || strings.HasPrefix(line, "skill_key:") ||
			strings.HasPrefix(line, "skill_source:") || strings.HasPrefix(line, "> Mirrors skill") {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
}
