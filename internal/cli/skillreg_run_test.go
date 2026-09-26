package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent},
		[]string{"curate"},
	)

	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Adopt:     map[string]string{"curate": "1049"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Adopted: renamed to the skill- slug, hash stamped.
	adopted, ok := vault.get("1049.2026-09-21.skill-curate.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(adopted).To(ContainSubstring("skill_hash: " + cli.SkillContentHash(skillContent)))

	// No further offer (prompt, decline, or summary line) for curate.
	g.Expect(stdout.String()).NotTo(ContainSubstring("Register skill"))
	g.Expect(stdout.String()).NotTo(ContainSubstring("awaiting an answer"))
}

// TestRunSkillRegistration_AdoptUnshippedSkill_Errors covers --adopt naming a
// skill that isn't in the loaded shipped set.
func TestRunSkillRegistration_AdoptUnshippedSkill_Errors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	listDir, readSkillFile := skillsDirFixture(map[string][]byte{}, nil)
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Adopt:     map[string]string{"curate": "1049"},
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("not currently shipped")))

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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Accept:    []string{"curate"},
		Decline:   []string{"curate"},
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("both --accept and --decline")))

	// Nothing was written — the vault holds no note or decline file.
	_, noteWritten := vault.get("1.2026-09-25.skill-curate.md")
	g.Expect(noteWritten).To(BeFalse())
	_, declinedWritten := vault.get("skill-registrations.json")
	g.Expect(declinedWritten).To(BeFalse())
}

// TestRunSkillRegistration_CompareSkillOffersError_Propagates covers
// computeSkillOffers' CompareSkillOffers failure branch: two runbook notes
// both carrying a skill_hash and ending in ".skill-curate.md" are a
// duplicate the comparison refuses to resolve.
func TestRunSkillRegistration_CompareSkillOffersError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("9998.2026-01-01.skill-curate.md", curateSkillNoteFixture("hash-a"))
	vault.put("9999.2026-01-02.skill-curate.md", curateSkillNoteFixture("hash-b"))

	skillContent := []byte("# Curate\n")
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", SkillsDir: skillsFixtureDir, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
}

// TestRunSkillRegistration_DryRun_PreviewsOffersWritesNothing covers
// "dry-run writes nothing": every computed offer is previewed as "would
// offer: <kind> <skill>", nothing is written, and nothing is prompted (an
// interactive IsTerminal is wired but must never be consulted).
func TestRunSkillRegistration_DryRun_PreviewsOffersWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()

	curateContent := []byte("# Curate\n\nNew text.\n")
	routeContent := []byte("# Route\n")

	listDir, readSkillFile := skillsDirFixture(map[string][]byte{
		skillsFixtureDir + "/curate/SKILL.md": curateContent,
		skillsFixtureDir + "/route/SKILL.md":  routeContent,
	}, []string{"curate", "route"})

	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool {
		t.Fatal("dry-run must never consult IsTerminal")

		return false
	}

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		DryRun:    true,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("would offer: register curate\nwould offer: register route\n"))

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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Accept:    []string{"curate"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).NotTo(ContainSubstring("Register skill"))

	_, noteWritten := vault.get("1.2026-09-25.skill-curate.md")
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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Accept:    []string{"bogus"},
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(`no pending offer for skill "bogus"`))
	g.Expect(stdout.String()).To(ContainSubstring("--accept"))
	g.Expect(stdout.String()).To(ContainSubstring("awaiting an answer: curate"))
}

// TestRunSkillRegistration_ExplicitDecline_NonInteractive_RecordsWithoutPrompting
// covers the decline half of "explicit accept/decline non-interactively".
func TestRunSkillRegistration_ExplicitDecline_NonInteractive_RecordsWithoutPrompting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		Decline:   []string{"curate"},
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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.ListMD = func(string) ([]string, error) { return nil, errSkillsDirForTest }

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", SkillsDir: skillsFixtureDir, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillsDirForTest))
}

// TestRunSkillRegistration_ListSkillsDirError_Propagates covers
// loadShippedSkills' listing-failure branch, reached through
// RunSkillRegistration.
func TestRunSkillRegistration_ListSkillsDirError_Propagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	deps := skillRegistrationDepsFor(vault,
		func(string) ([]fs.DirEntry, error) { return nil, errSkillsDirForTest },
		func(string) ([]byte, error) { return nil, fs.ErrNotExist },
	)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", SkillsDir: skillsFixtureDir,
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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return false }

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal(
		"engram: skill runbook offers awaiting an answer: curate — run `engram register-skills` in a " +
			"terminal, or `engram register-skills --accept <name>` / `--decline <name>`\n",
	))
	g.Expect(vault.files).To(BeEmpty())
}

// TestRunSkillRegistration_PromptEOF_Declines covers "EOF -> decline": an
// exhausted stdin during an interactive prompt declines the offer.
func TestRunSkillRegistration_PromptEOF_Declines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skillContent := []byte("# Curate\n")
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	declined, ok := vault.get("skill-registrations.json")
	g.Expect(ok).To(BeTrue())
	g.Expect(declined).To(ContainSubstring(cli.SkillContentHash(skillContent)))
}

// TestRunSkillRegistration_PromptRefresh_Yes_ReplacesBody covers the refresh
// prompt's exact wording and a "y" answer accepting it.
func TestRunSkillRegistration_PromptRefresh_Yes_ReplacesBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	oldHash := cli.SkillContentHash([]byte("old curate body"))
	vault.put("1049.2026-09-21.skill-curate.md", curateSkillNoteFixture(oldHash))

	newContent := []byte("# Curate\n\nRevised.\n")
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": newContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("y\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(
		"Skill `curate` changed since its note was last synced. Update the note? [y/N] "))

	updated, ok := vault.get("1049.2026-09-21.skill-curate.md")
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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("n\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	_, noteWritten := vault.get("1.2026-09-25.skill-curate.md")
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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("YES\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("Register skill `curate` as a vault runbook? [y/N] "))

	written, ok := vault.get("1.2026-09-25.skill-curate.md")
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
	vault.put(basename+".md", curateSkillNoteFixture("wm-hash"))
	vault.put(basename+".vec.json", `{"model_id":"m"}`)

	// No shipped skills at all — write-memory's note is now orphaned.
	listDir, readSkillFile := skillsDirFixture(map[string][]byte{}, nil)
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader("y\n")

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring(
		"Skill `write-memory` is no longer shipped. Remove its runbook note? [y/N] "))

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
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": skillContent}, []string{"curate"})
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)
	deps.Accept.Read = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "skill-registrations.json") {
			return nil, errSkillsDirForTest
		}

		return vault.ReadFile(path)
	}

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", SkillsDir: skillsFixtureDir, DryRun: true,
	}, deps, &stdout)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillsDirForTest))
}

// TestRunSkillRegistration_SkillsDirEntryWithoutSkillMD_Ignored covers "a
// skills dir entry without SKILL.md is ignored": the entry produces no
// offer and no error.
func TestRunSkillRegistration_SkillsDirEntryWithoutSkillMD_Ignored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	curateContent := []byte("# Curate\n")
	listDir, readSkillFile := skillsDirFixture(
		map[string][]byte{skillsFixtureDir + "/curate/SKILL.md": curateContent},
		[]string{"curate", "no-skill-md"},
	)
	deps := skillRegistrationDepsFor(vault, listDir, readSkillFile)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault:     "/vault",
		VaultName: "personal",
		SkillsDir: skillsFixtureDir,
		DryRun:    true,
	}, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("would offer: register curate\n"))
}

// unexported constants.
const (
	skillsFixtureDir = "/skills"
)

// unexported variables.
var (
	errSkillsDirForTest = errors.New("skillreg-run fixture: injected skills-dir failure")
)

// unexported test helpers.

// skillRegistrationDepsFor composes SkillRegistrationDeps over vault (via the
// skillreg_accept_test.go fixture helpers) and the given skills-dir
// listing/reading funcs. IsTerminal defaults to false (non-interactive);
// tests override it directly on the returned value.
func skillRegistrationDepsFor(
	vault *skillAcceptFixtureVault,
	listSkillsDir func(string) ([]fs.DirEntry, error),
	readSkillFile func(string) ([]byte, error),
) cli.SkillRegistrationDeps {
	var writtenPath string

	var writtenContent []byte

	return cli.SkillRegistrationDeps{
		ListSkillsDir: listSkillsDir,
		ReadSkillFile: readSkillFile,
		ListMD:        vault.ListMD,
		IsTerminal:    func() bool { return false },
		Accept:        skillAcceptDeps(vault),
		Adopt: cli.SkillAdoptDeps{
			Lock:     noLock,
			Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
			Rename:   skillAcceptRenameDeps(vault),
			Embedder: skillAcceptFakeEmbedder{},
		},
		Learn: registerSkillLearnDeps(vault, &writtenPath, &writtenContent),
	}
}

// skillsDirFixture builds ListSkillsDir/ReadSkillFile closures over an
// in-memory skills source dir: dirNames become top-level directory entries,
// and entries maps a full "<skillsDir>/<name>/SKILL.md" path to its bytes —
// a dir name with no matching entry simulates a skill directory without a
// SKILL.md file.
func skillsDirFixture(
	entries map[string][]byte, dirNames []string,
) (func(string) ([]fs.DirEntry, error), func(string) ([]byte, error)) {
	list := make([]fs.DirEntry, 0, len(dirNames))
	for _, name := range dirNames {
		list = append(list, fakeDirEntry{name: name, dir: true})
	}

	listDir := func(string) ([]fs.DirEntry, error) { return list, nil }
	readFile := func(path string) ([]byte, error) {
		content, ok := entries[path]
		if !ok {
			return nil, fmt.Errorf("skillreg-run fixture: %w: %s", fs.ErrNotExist, path)
		}

		return content, nil
	}

	return listDir, readFile
}
