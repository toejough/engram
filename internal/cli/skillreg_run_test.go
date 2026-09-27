package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"maps"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunSkillRegistration_AdoptKeyRegisteredElsewhere_WritesNothing covers
// a multi-entry --adopt whose later entry (sorted order) adopts a key the
// vault already registers to a different note: the conflict is caught in
// validation, before the earlier entry renames its note, so the vault is
// byte-unchanged (the reproduced partial write: claude:route keyed on 1036,
// then --adopt claude:curate=1049 --adopt claude:route=1045).
func TestRunSkillRegistration_AdoptKeyRegisteredElsewhere_WritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1036.2026-09-22.skill-claude-route.md", keyedSkillNote("route-hash", "claude:route"))
	vault.put("1045.2026-09-23.route-draft.md", promotedRunbookFixture("1045", "2026-09-23", "never guess"))
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	before := maps.Clone(vault.files)

	deps := skillRegistrationDepsFor(vault, skillsHomeFixture(map[string][]byte{
		"curate": []byte("# Curate\n"), "route": []byte("# Route\n"),
	}))

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome,
		Adopt: map[string]string{"claude:curate": "1049", "claude:route": "1045"},
	}, deps, &stdout)

	g.Expect(err).To(MatchError(cli.ErrAdoptConflictForTest))
	g.Expect(vault.files).To(Equal(before), "a refused adopt run must write nothing")
}

// TestRunSkillRegistration_AdoptKeyWhileBatchRekeysItsNote_WritesNothing
// covers an entry adopting key K onto note N while the vault's K-note is
// another note that a second entry in the same batch would re-key away:
// refused before any write, never resolved by ordering.
func TestRunSkillRegistration_AdoptKeyWhileBatchRekeysItsNote_WritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1036.2026-09-22.skill-claude-route.md", keyedSkillNote("route-hash", "claude:route"))
	vault.put("1045.2026-09-23.route-draft.md", promotedRunbookFixture("1045", "2026-09-23", "never guess"))

	before := maps.Clone(vault.files)

	deps := skillRegistrationDepsFor(vault, skillsHomeFixture(map[string][]byte{
		"route": []byte("# Route\n"), "write-memory": []byte("# Write memory\n"),
	}))

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome,
		Adopt: map[string]string{"claude:route": "1045", "claude:write-memory": "1036"},
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(vault.files).To(Equal(before), "a refused adopt run must write nothing")
}

// TestRunSkillRegistration_AdoptLinkedNotes_OrderIndependent covers several
// adopts in one run where one adopted note links to another (the real vault:
// route's red_flags link write-memory's note): the link is rewritten to the
// linked note's new basename, both notes' sidecars are fresh, and the vault
// ends the same whichever order the adopts run in.
func TestRunSkillRegistration_AdoptLinkedNotes_OrderIndependent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	routeContent := []byte("# Route\n\n1. Pick a tier.\n")
	writeMemoryContent := []byte("# Write memory\n\n1. Compose the command.\n")
	sources := map[string]cli.SkillNoteSource{
		"claude:route":        installedEngramSkill("route", routeContent),
		"claude:write-memory": installedEngramSkill("write-memory", writeMemoryContent),
	}
	refs := map[string]string{"claude:route": "1036", "claude:write-memory": "1053"}

	// Each order adopts through AdoptSkillNote directly; the run itself
	// (sorted keys) is a third arm.
	orders := [][]string{
		{"claude:route", "claude:write-memory"},
		{"claude:write-memory", "claude:route"},
	}
	finals := make([]map[string]string, 0, len(orders)+1)

	for _, order := range orders {
		vault := linkedAdoptFixtureVault(t.Context(), g)
		deps := skillRegistrationDepsFor(vault, skillsHomeFixture(nil)).Adopt

		for _, key := range order {
			var stdout bytes.Buffer

			g.Expect(cli.AdoptSkillNote(t.Context(), "/vault", sources[key], refs[key], deps, &stdout)).To(Succeed())
		}

		finals = append(finals, vault.files)
	}

	runVault := linkedAdoptFixtureVault(t.Context(), g)
	runDeps := skillRegistrationDepsFor(runVault, skillsHomeFixture(map[string][]byte{
		"route": routeContent, "write-memory": writeMemoryContent,
	}))

	var stdout bytes.Buffer

	g.Expect(cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome, Adopt: refs,
	}, runDeps, &stdout)).To(Succeed())

	finals = append(finals, runVault.files)

	const (
		routeBasename       = "1036.2026-09-22.skill-claude-route"
		writeMemoryBasename = "1053.2026-09-22.skill-claude-write-memory"
	)

	for _, final := range finals[1:] {
		g.Expect(final).To(Equal(finals[0]), "the adopt order must not change the result")
	}

	routeNote := finals[0]["/vault/"+routeBasename+".md"]
	g.Expect(routeNote).To(ContainSubstring("[[" + writeMemoryBasename + "]]"))
	g.Expect(routeNote).NotTo(ContainSubstring("[[1053.2026-09-22.write-memory-worker]]"))

	for _, basename := range []string{routeBasename, writeMemoryBasename} {
		g.Expect(embed.ComputeState(runVault, "/vault/"+basename+".md", skillAcceptFakeEmbedder{}.ModelID())).
			To(Equal(embed.StateOK), basename+"'s sidecar must be fresh")
	}

	for _, old := range []string{"1036.2026-09-22.route-dispatch", "1053.2026-09-22.write-memory-worker"} {
		g.Expect(finals[0]).NotTo(HaveKey("/vault/" + old + ".md"))
		g.Expect(finals[0]).NotTo(HaveKey("/vault/" + old + ".vec.json"))
	}
}

// TestRunSkillRegistration_AdoptOneNoteToTwoKeys_ErrorsAndWritesNothing
// covers adopt never silently re-keying a note within one run: two --adopt
// entries naming the same note are refused before any adopt writes.
func TestRunSkillRegistration_AdoptOneNoteToTwoKeys_ErrorsAndWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	before := maps.Clone(vault.files)

	deps := skillRegistrationDepsFor(vault, skillsHomeFixture(map[string][]byte{
		"curate": []byte("# Curate\n"), "route": []byte("# Route\n"),
	}))

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome,
		Adopt: map[string]string{"claude:curate": "1049", "claude:route": "1049.2026-09-21.curate-review-pending-offers"},
	}, deps, &stdout)

	g.Expect(err).To(MatchError(cli.ErrAdoptTargetKeyedForTest))
	g.Expect(vault.files).To(Equal(before), "a refused adopt run must write nothing")
}

// TestRunSkillRegistration_AdoptRunsBeforeOffers_NoDuplicateOffer covers
// "adopt runs before offers": adopting curate's existing promoted note first
// means the offer comparison (run afterward) sees curate already carrying a
// matching skill_hash, so no register offer fires for it too.
func TestRunSkillRegistration_AdoptRunsBeforeOffers_NoDuplicateOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	skillContent := []byte("# Curate\n\n1. Judge offers.\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})

	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Adopt:     map[string]string{"claude:curate": "1049"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Adopted: renamed to the skill- slug, hash stamped.
	adopted, ok := vault.get("1049.2026-09-21.skill-claude-curate.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(adopted).To(ContainSubstring("skill_hash: " + cli.SkillContentHash(skillContent)))

	// No further offer (prompt, decline, or summary line) for curate.
	g.Expect(stdout.String()).NotTo(ContainSubstring("Register skill"))
	g.Expect(stdout.String()).NotTo(ContainSubstring("awaiting an answer"))
}

// TestRunSkillRegistration_AdoptUnparseableBasenameLater_WritesNothing covers
// a multi-entry --adopt whose later entry (sorted order) targets a runbook
// note whose basename has no Luhmann id/date: refused in validation, before
// the earlier entry writes.
func TestRunSkillRegistration_AdoptUnparseableBasenameLater_WritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())
	vault.put("README.md",
		"---\ntype: runbook\nsituation: s\ndone_when: d\nsource: s\nuser: u\nvault: v\n---\n\nbody\n")

	before := maps.Clone(vault.files)

	deps := skillRegistrationDepsFor(vault, skillsHomeFixture(map[string][]byte{
		"curate": []byte("# Curate\n"), "route": []byte("# Route\n"),
	}))

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome,
		Adopt: map[string]string{"claude:curate": "1049", "claude:route": "README"},
	}, deps, &stdout)

	g.Expect(err).To(MatchError(ContainSubstring("no Luhmann id/date")))
	g.Expect(vault.files).To(Equal(before), "a refused adopt run must write nothing")
}

// TestRunSkillRegistration_AdoptUnshippedSkill_Errors covers --adopt naming a
// skill that isn't in the loaded shipped set.
func TestRunSkillRegistration_AdoptUnshippedSkill_Errors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	sourceFS := skillsHomeFixture(nil)
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Adopt:     map[string]string{"claude:curate": "1049"},
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("key names no scanned skill")))

	// Untouched: the note under its old basename is still there.
	_, stillThere := vault.get("1049.2026-09-21.curate-review-pending-offers.md")
	g.Expect(stillThere).To(BeTrue())
}

// TestRunSkillRegistration_BothNamedError covers the "skill named in both
// --accept and --decline" refusal, before anything is acted on.
func TestRunSkillRegistration_BothNamedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Accept:    []string{"claude:curate"},
		Decline:   []string{"claude:curate"},
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("both --accept and --decline")))

	// Nothing was written — the vault holds no note or decline file.
	_, noteWritten := vault.get("1.2026-09-25.skill-claude-curate.md")
	g.Expect(noteWritten).To(BeFalse())
	_, declinedWritten := vault.get("skill-registrations.json")
	g.Expect(declinedWritten).To(BeFalse())
}

// TestRunSkillRegistration_CompareSkillOffersError_Propagates covers
// computeSkillOffers' CompareSkillOffers failure branch: two runbook notes
// both carrying a skill_hash and ending in ".skill-claude-curate.md" are a
// duplicate the comparison refuses to resolve.
func TestRunSkillRegistration_CompareSkillOffersError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("9998.2026-01-01.skill-claude-curate.md", curateSkillNoteFixture("hash-a"))
	vault.put("9999.2026-01-02.skill-claude-curate.md", curateSkillNoteFixture("hash-b"))

	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
}

// TestRunSkillRegistration_DryRun_PreviewsOffersWritesNothing covers
// "dry-run writes nothing": every computed offer is previewed as "would
// offer: <kind> <key> (<source>)" under its scope header, nothing is written, and nothing is prompted (an
// interactive IsTerminal is wired but must never be consulted).
func TestRunSkillRegistration_DryRun_PreviewsOffersWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()

	curateContent := []byte("# Curate\n\nNew text.\n")
	routeContent := []byte("# Route\n")

	sourceFS := skillsHomeFixture(map[string][]byte{"curate": curateContent, "route": routeContent})

	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool {
		t.Fatal("dry-run must never consult IsTerminal")

		return false
	}

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		DryRun:    true,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("@claude-user (2)\n" +
		"  would offer: register claude:curate (" + skillRegSourcePath("curate") + ")\n" +
		"  would offer: register claude:route (" + skillRegSourcePath("route") + ")\n"))

	g.Expect(vault.files).To(BeEmpty())
}

// TestRunSkillRegistration_ExplicitAccept_NonInteractive_ActsWithoutPrompting
// covers "explicit accept/decline non-interactively": --accept answers an
// offer directly, with no prompt printed.
func TestRunSkillRegistration_ExplicitAccept_NonInteractive_ActsWithoutPrompting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n\nStep 1.\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Accept:    []string{"claude:curate"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).NotTo(ContainSubstring("Register skill"))

	_, noteWritten := vault.get("1.2026-09-25.skill-claude-curate.md")
	g.Expect(noteWritten).To(BeTrue())
}

// TestRunSkillRegistration_ExplicitAnswerWithNoOffer_PrintsNoteAndContinues
// covers "unknown-name note": a --accept naming a skill with no current
// offer is reported (not an error), and the run still reports curate's real
// offer as awaiting an answer.
func TestRunSkillRegistration_ExplicitAnswerWithNoOffer_PrintsNoteAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Accept:    []string{"bogus"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(`no pending offer for skill "bogus"`))
	g.Expect(stdout.String()).To(ContainSubstring("--accept"))
	g.Expect(stdout.String()).To(ContainSubstring("awaiting an answer: @claude-user 1"))
}

// TestRunSkillRegistration_ExplicitDecline_NonInteractive_RecordsWithoutPrompting
// covers the decline half of "explicit accept/decline non-interactively".
func TestRunSkillRegistration_ExplicitDecline_NonInteractive_RecordsWithoutPrompting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		Decline:   []string{"claude:curate"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).NotTo(ContainSubstring("Register skill"))

	declined, ok := vault.get("skill-registrations.json")
	g.Expect(ok).To(BeTrue())
	g.Expect(declined).To(ContainSubstring(cli.SkillContentHash(skillContent)))
}

// TestRunSkillRegistration_ListMDError_Propagates covers computeSkillOffers'
// vault-listing failure branch.
func TestRunSkillRegistration_ListMDError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.ListMD = func(string) ([]string, error) { return nil, errSkillsDirForTest }

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillsDirForTest))
}

// TestRunSkillRegistration_NonInteractive_NoAnswers_WritesNothingPrintsSummary
// covers "non-interactive writes nothing and prints the summary line": with
// stdin not a terminal and no explicit answers, nothing is written and the
// exact one-line summary names the waiting skill.
func TestRunSkillRegistration_NonInteractive_NoAnswers_WritesNothingPrintsSummary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return false }

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal(
		"engram: 1 skill runbook offer awaiting an answer: @claude-user 1 — run `engram register-skills` in a " +
			"terminal, or `engram register-skills --accept <key|prefix*|@scope>` / `--decline <key|prefix*|@scope>`\n",
	))
	g.Expect(vault.files).To(BeEmpty())
}

// TestRunSkillRegistration_PromptEOF_RecordsNothing covers ruling R29: an
// exhausted stdin at an interactive per-offer prompt is no answer, so no
// note and no decline is written.
func TestRunSkillRegistration_PromptEOF_RecordsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vault.files).To(BeEmpty())
}

// TestRunSkillRegistration_PromptRefresh_Yes_ReplacesBody covers the refresh
// prompt's exact wording and a "y" answer accepting it.
func TestRunSkillRegistration_PromptRefresh_Yes_ReplacesBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	oldHash := cli.SkillContentHash([]byte("old curate body"))
	vault.put("1049.2026-09-21.skill-claude-curate.md", curateSkillNoteFixture(oldHash))

	newContent := []byte("# Curate\n\nRevised.\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": newContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("y\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(
		"Skill `claude:curate` (" + skillRegSourcePath("curate") + ") changed since its note was last synced. " +
			"Update the note? [y/N] "))

	updated, ok := vault.get("1049.2026-09-21.skill-claude-curate.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(updated).To(ContainSubstring("Revised."))
	g.Expect(updated).To(ContainSubstring("skill_hash: " + cli.SkillContentHash(newContent)))
}

// TestRunSkillRegistration_PromptRegister_No_RecordsDecline covers a "n"
// answer to the register prompt declining and recording the hash.
func TestRunSkillRegistration_PromptRegister_No_RecordsDecline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("n\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	_, noteWritten := vault.get("1.2026-09-25.skill-claude-curate.md")
	g.Expect(noteWritten).To(BeFalse())

	declined, ok := vault.get("skill-registrations.json")
	g.Expect(ok).To(BeTrue())
	g.Expect(declined).To(ContainSubstring(cli.SkillContentHash(skillContent)))
}

// TestRunSkillRegistration_PromptRegister_Yes_CreatesNote covers the
// register prompt's exact wording and a "yes" (full word, case-insensitive)
// answer accepting it.
func TestRunSkillRegistration_PromptRegister_Yes_CreatesNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n\nStep 1. Judge offers.\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("YES\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(
		"Register skill `claude:curate` (" + skillRegSourcePath("curate") + ") as a vault runbook? [y/N] "))

	written, ok := vault.get("1.2026-09-25.skill-claude-curate.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(ContainSubstring("pending: true"))
}

// TestRunSkillRegistration_PromptRemove_Yes_DeletesNote covers the remove
// prompt's exact wording and a "y" answer accepting it.
func TestRunSkillRegistration_PromptRemove_Yes_DeletesNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1053.2026-09-21.skill-write-memory"
	vault.put(basename+".md", sourcedSkillNote(offerNote{key: "claude:write-memory", hash: "wm-hash"}))
	vault.put(basename+".vec.json", `{"model_id":"m"}`)

	// No shipped skills at all — write-memory's note is now orphaned.
	sourceFS := skillsHomeFixture(nil)
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("y\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(
		"Skill `claude:write-memory` is no longer found in its source. Remove its runbook note? [y/N] "))

	_, noteStillThere := vault.get(basename + ".md")
	g.Expect(noteStillThere).To(BeFalse())
	_, sidecarStillThere := vault.get(basename + ".vec.json")
	g.Expect(sidecarStillThere).To(BeFalse())
}

// TestRunSkillRegistration_ReadSkillRegistrationsError_Propagates covers
// computeSkillOffers' decline-state-read failure branch: skill-
// registrations.json exists but fails to read for a reason other than
// "missing".
func TestRunSkillRegistration_ReadSkillRegistrationsError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": skillContent})
	deps := skillRegistrationDepsFor(vault, sourceFS)
	deps.Accept.Read = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "skill-registrations.json") {
			return nil, errSkillsDirForTest
		}

		return vault.ReadFile(path)
	}

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillsDirForTest))
}

// TestRunSkillRegistration_RealRunsPrintScanWarnings covers the scanner and
// key-builder warnings on every path, not only --dry-run: a `:` skill name
// is skipped with a warning on an interactive run and on a non-interactive
// one alike (skill-runbook-registration: "An entry whose name contains `:`
// SHALL be skipped with a warning").
func TestRunSkillRegistration_RealRunsPrintScanWarnings(t *testing.T) {
	t.Parallel()

	for name, interactive := range map[string]bool{"interactive": true, "non-interactive": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			sourceFS := skillsHomeFixture(nil).file(skillRegHome+"/.claude/skills/bad:name/SKILL.md", "# Bad\n")
			deps := skillRegistrationDepsFor(vault, sourceFS)
			deps.IsTerminal = func() bool { return interactive }
			deps.Stdin = strings.NewReader("")

			var stdout bytes.Buffer

			err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
				Vault: "/vault", VaultName: "personal", Home: skillRegHome,
			}, deps, &stdout)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stdout.String()).To(ContainSubstring(`engram: skipping skill "bad:name"`))
			g.Expect(vault.files).To(BeEmpty())
		})
	}
}

// TestRunSkillRegistration_ResolveError_Propagates covers the default
// set's resolution failure (a harness probe failing for a reason other than
// not-exist), reached through RunSkillRegistration: nothing is offered.
func TestRunSkillRegistration_ResolveError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	deps := skillRegistrationDepsFor(vault, skillsHomeFixture(nil).failLstat(skillRegHome+"/.claude"))

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome,
	}, deps, &stdout)

	g.Expect(err).To(MatchError(ContainSubstring("resolving skill sources")))
	g.Expect(err).To(MatchError(fs.ErrPermission))
	g.Expect(stdout.String()).To(BeEmpty())
}

// TestRunSkillRegistration_ResolvesFromTheWorkingDirectory covers design D1:
// the default set is resolved from the working directory Getwd supplies — a
// project command in the cwd's repository is offered under its project key —
// and a run with no working directory fails instead of silently dropping
// the project sources.
func TestRunSkillRegistration_ResolvesFromTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	getwdFailure := errors.New("getwd fixture: no cwd")

	cases := map[string]struct {
		getwd   func() (string, error)
		want    string
		wantErr error
	}{
		"project cwd": {
			getwd: func() (string, error) { return projectTop, nil },
			want: "@claude-user (1)\n  would offer: register claude:curate (" + skillRegSourcePath("curate") + ")\n" +
				"@project:github.com/toejough/engram (1)\n" +
				"  would offer: register project:github.com/toejough/engram:cmd:ship (" +
				projectTop + "/.claude/commands/ship.md)\n",
		},
		"getwd fails": {getwd: func() (string, error) { return "", getwdFailure }, wantErr: getwdFailure},
		"no getwd":    {wantErr: cli.ErrSkillSourcesNeedWorkingDirForTest},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			sourceFS := skillsHomeFixture(map[string][]byte{"curate": []byte("# Curate\n")}).
				file(projectTop+"/.claude/commands/ship.md", "Ship it.\n")
			deps := skillRegistrationDepsFor(vault, sourceFS)
			deps.Sources.Commander = scriptedGit{
				"rev-parse --show-toplevel": {out: projectTop + "\n"},
				"remote get-url origin":     {out: "git@github.com:toejough/engram.git\n"},
			}
			deps.Getwd = testCase.getwd

			var stdout bytes.Buffer

			err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
				Vault: "/vault", VaultName: "personal", Home: skillRegHome, DryRun: true,
			}, deps, &stdout)

			if testCase.wantErr != nil {
				g.Expect(err).To(MatchError(testCase.wantErr))
				g.Expect(err).To(MatchError(cli.ErrSkillSourcesNeedWorkingDirForTest))
				g.Expect(stdout.String()).To(BeEmpty())

				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stdout.String()).To(Equal(testCase.want))
		})
	}
}

// TestRunSkillRegistration_SkillsDirEntryWithoutSkillMD_Ignored covers "a
// skills dir entry without SKILL.md is ignored": the entry produces no
// offer and no error.
func TestRunSkillRegistration_SkillsDirEntryWithoutSkillMD_Ignored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	curateContent := []byte("# Curate\n")
	sourceFS := skillsHomeFixture(map[string][]byte{"curate": curateContent}).
		dir(skillRegHome + "/.claude/skills/no-skill-md")
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		Home:      skillRegHome,
		DryRun:    true,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal(
		"@claude-user (1)\n  would offer: register claude:curate (" + skillRegSourcePath("curate") + ")\n"))
}

// unexported constants.
const (
	// skillRegDeployedSkills is the fixture home's ~/.claude/engram/skills,
	// where `engram update` deploys engram's skills.
	skillRegDeployedSkills = skillRegHome + "/.claude/engram/skills"
	// skillRegHome is the fixture home.
	skillRegHome = "/home/reg"
)

// unexported variables.
var (
	errSkillsDirForTest = errors.New("skillreg-run fixture: injected skills-dir failure")
)

// unexported test helpers.

// linkedAdoptFixtureVault holds two promoted, unkeyed runbook notes — route
// (1036) whose red_flags link write-memory's note (1053), mirroring the real
// vault — each with a fresh sidecar.
func linkedAdoptFixtureVault(ctx context.Context, g Gomega) *skillAcceptFixtureVault {
	vault := newSkillAcceptFixtureVault()
	notes := map[string]string{
		"1036.2026-09-22.route-dispatch": promotedRunbookFixture("1036", "2026-09-22",
			"never dispatch without [[1053.2026-09-22.write-memory-worker]]"),
		"1053.2026-09-22.write-memory-worker": promotedRunbookFixture("1053", "2026-09-22", "never write unhanded"),
	}

	for basename, content := range notes {
		vault.put(basename+".md", content)

		sidecar, buildErr := embed.BuildSidecar(ctx, skillAcceptFakeEmbedder{}, []byte(content))
		g.Expect(buildErr).NotTo(HaveOccurred())
		vault.put(basename+".vec.json", string(embed.MarshalSidecar(sidecar)))
	}

	return vault
}

// promotedRunbookFixture renders a promoted, unkeyed runbook note with one
// red flag.
func promotedRunbookFixture(id, date, redFlag string) string {
	return "---\n" +
		"type: runbook\n" +
		"situation: doing the work note " + id + " covers\n" +
		"done_when: the work is done\n" +
		"red_flags:\n" +
		"    - \"" + redFlag + "\"\n" +
		"luhmann: \"" + id + "\"\n" +
		"created: " + date + "\n" +
		"source: promoted\n" +
		"user: joe\n" +
		"vault: personal\n" +
		"---\n\n" +
		"1. Do it.\n"
}

// skillRegSourcePath is the resolved SKILL.md path of an engram skill in
// skillsHomeFixture's home.
func skillRegSourcePath(name string) string {
	return skillRegDeployedSkills + "/" + name + "/SKILL.md"
}

// skillRegistrationDepsFor composes SkillRegistrationDeps over vault (via the
// skillreg_accept_test.go fixture helpers) and sourceFS, the filesystem the
// default source set is resolved from. The working directory lies outside
// any git repository (the scripted git answers nothing), so no project
// source is read. IsTerminal defaults to false (non-interactive); tests
// override it directly on the returned value.
func skillRegistrationDepsFor(vault *skillAcceptFixtureVault, sourceFS cli.SkillSourceFS) cli.SkillRegistrationDeps {
	var writtenPath string

	var writtenContent []byte

	return cli.SkillRegistrationDeps{
		Sources:    cli.SkillSourceDeps{FS: sourceFS, Commander: scriptedGit{}},
		Getwd:      func() (string, error) { return "/outside/any/repo", nil },
		ListMD:     vault.ListMD,
		IsTerminal: func() bool { return false },
		Accept:     skillAcceptDeps(vault),
		Adopt: cli.SkillAdoptDeps{
			Lock:     noLock,
			Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
			Rename:   skillAcceptRenameDeps(vault),
			Embedder: skillAcceptFakeEmbedder{},
		},
		Learn: registerSkillLearnDeps(vault, &writtenPath, &writtenContent),
	}
}

// skillsHomeFixture builds skillRegHome with a Claude Code harness: each of
// skills is deployed under ~/.claude/engram/skills and linked into
// ~/.claude/skills, as `engram update` deploys engram's skills.
func skillsHomeFixture(skills map[string][]byte) *fakeSkillFS {
	fsys := newFakeSkillFS().dir(skillRegHome + "/.claude/skills")

	for name, content := range skills {
		fsys.file(skillRegSourcePath(name), string(content))
		fsys.link(skillRegHome+"/.claude/skills/"+name, skillRegDeployedSkills+"/"+name)
	}

	return fsys
}
