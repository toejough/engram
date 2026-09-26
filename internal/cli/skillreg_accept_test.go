package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestAdoptSkillNote_ClearsPreExistingPendingMarker documents and tests the
// decision recorded on AdoptSkillNote: an existing pending: true marker is
// unconditionally cleared, not merely left alone, so the postcondition "SHALL
// NOT be marked pending" holds regardless of the target's prior state.
func TestAdoptSkillNote_ClearsPreExistingPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.curate-review-pending-offers"
	vault.put(basename+".md", curatePromotedNoteFixturePending())

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\n1. Judge offers.\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, _ := vault.get("1049.2026-09-21.skill-curate.md")
	doc := parseSkillAcceptFrontmatter(g, updated)
	g.Expect(doc.Pending).To(BeFalse())
}

// TestAdoptSkillNote_ErrorsOnConflictingExistingSkillNote covers "errors for
// ... conflicting existing skill note": a different note is already
// registered as curate's skill note.
func TestAdoptSkillNote_ErrorsOnConflictingExistingSkillNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("2001.2026-01-01.curate-v2.md", curatePromotedNoteFixtureAt("2001", "2026-01-01"))
	vault.put("9999.2026-01-01.skill-curate.md", curateSkillNoteFixture("already-registered-hash"))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "2001", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("already registered")))

	// Nothing was renamed or overwritten by the failed attempt.
	_, stillThere := vault.get("2001.2026-01-01.curate-v2.md")
	g.Expect(stillThere).To(BeTrue())
}

// TestAdoptSkillNote_ErrorsOnMissingTarget covers "errors for ... missing
// target".
func TestAdoptSkillNote_ErrorsOnMissingTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "9999", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("not found")))
}

// TestAdoptSkillNote_ErrorsOnNonRunbookTarget covers "errors for non-runbook
// target".
func TestAdoptSkillNote_ErrorsOnNonRunbookTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("2001.2026-01-01.a-fact.md",
		"---\ntype: fact\nsituation: s\nsubject: sub\npredicate: pred\nobject: obj\n"+
			"luhmann: \"2001\"\ncreated: 2026-01-01\nsource: s\nuser: u\nvault: v\n---\n\nInformation learned: s.\n")

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "2001", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("not a runbook")))
}

// TestAdoptSkillNote_ErrorsOnUnparseableBasename covers skillNoteBasename's
// error branch reached through AdoptSkillNote: a runbook note whose basename
// has no leading Luhmann id/date cannot be adopted.
func TestAdoptSkillNote_ErrorsOnUnparseableBasename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("README.md",
		"---\ntype: runbook\nsituation: s\ndone_when: d\nsource: s\nuser: u\nvault: v\n---\n\nbody\n")

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "README", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("no Luhmann id/date")))
}

// TestAdoptSkillNote_IdempotentOnAlreadyAdoptedNote covers "Idempotent:
// adopting a note already named skill-<name> just refreshes body/hash" — no
// rename occurs (the basename is unchanged), but the body and hash still
// update.
func TestAdoptSkillNote_IdempotentOnAlreadyAdoptedNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", curateSkillNoteFixture(cli.SkillContentHash([]byte("previous adopt"))))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\n1. Newer text.\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, ok := vault.get(basename + ".md")
	g.Expect(ok).To(BeTrue(), "the note must still live at its already-adopted basename")

	doc := parseSkillAcceptFrontmatter(g, updated)
	g.Expect(doc.SkillHash).To(Equal(cli.SkillContentHash(skill.Content)))
	g.Expect(doc.Pending).To(BeFalse())
	g.Expect(updated).To(ContainSubstring("1. Newer text."))
}

// TestAdoptSkillNote_PropagatesEmbedError covers AdoptSkillNote's sidecar
// rebuild failure branch.
func TestAdoptSkillNote_PropagatesEmbedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFailingEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestAdoptSkillNote_PropagatesFindSkillNoteError covers checkAdoptConflict's
// FindSkillNote failure branch: two existing notes both carry a skill_hash
// matching curate's slug, so FindSkillNote reports a duplicate.
func TestAdoptSkillNote_PropagatesFindSkillNoteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())
	vault.put("9998.2026-01-01.skill-curate.md", curateSkillNoteFixture("hash-a"))
	vault.put("9999.2026-01-02.skill-curate.md", curateSkillNoteFixture("hash-b"))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
}

// TestAdoptSkillNote_PropagatesListMDError covers checkAdoptConflict's ListMD
// failure branch.
func TestAdoptSkillNote_PropagatesListMDError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	renameDeps := skillAcceptRenameDeps(vault)
	renameDeps.ListMD = func(string) ([]string, error) { return nil, errSkillAcceptForTest }

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   renameDeps,
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestAdoptSkillNote_PropagatesLockError covers AdoptSkillNote's
// lock-acquisition failure branch.
func TestAdoptSkillNote_PropagatesLockError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     func(string) (func(), error) { return nil, errSkillAcceptForTest },
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestAdoptSkillNote_PropagatesScanError covers resolveAdoptTarget's scan
// failure branch.
func TestAdoptSkillNote_PropagatesScanError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(string) ([]vaultgraph.Note, error) { return nil, errSkillAcceptForTest },
		Rename:   skillAcceptRenameDeps(newSkillAcceptFixtureVault()),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestAdoptSkillNote_PropagatesWriteError covers AdoptSkillNote's final
// write failure branch.
func TestAdoptSkillNote_PropagatesWriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	renameDeps := skillAcceptRenameDeps(vault)
	renameDeps.WriteFile = func(string, []byte) error { return errSkillAcceptForTest }

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   renameDeps,
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestAdoptSkillNote_RenamesRewritesLinksPreservesFieldsClearsPending covers
// the spec's "Adopting a previously promoted note" scenario: note 1049 is
// renamed to the skill-curate slug, inbound links are rewritten, its fields
// are unchanged, and it carries skill_hash with no pending marker.
func TestAdoptSkillNote_RenamesRewritesLinksPreservesFieldsClearsPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	oldBasename := "1049.2026-09-21.curate-review-pending-offers"
	vault.put(oldBasename+".md", curatePromotedNoteFixture())
	vault.put(oldBasename+".vec.json", `{"model_id":"old"}`)
	vault.put("2000.2026-09-22.some-other-note.md", referencingNoteFixture(oldBasename))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\n1. Judge offers.\n")}

	deps := cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}

	var stdout bytes.Buffer

	err := cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	newBasename := "1049.2026-09-21.skill-curate"

	newContent, ok := vault.get(newBasename + ".md")
	g.Expect(ok).To(BeTrue())

	_, oldStillThere := vault.get(oldBasename + ".md")
	g.Expect(oldStillThere).To(BeFalse())

	doc := parseSkillAcceptFrontmatter(g, newContent)

	g.Expect(doc.Type).To(Equal("runbook"))
	g.Expect(doc.Situation).To(Equal("reviewing pending offers in a vault and judging each against existing notes"))
	g.Expect(doc.DoneWhen).To(Equal("every pending offer has been discarded, folded, or accepted"))
	g.Expect(doc.RedFlags).To(Equal([]string{"do not compose a new learn", "do not report outcome to caller"}))
	g.Expect(doc.Triggers).To(Equal([]string{"curate", "pending offers"}))
	g.Expect(doc.Luhmann).To(Equal("1049"), "the Luhmann id must survive the rename unchanged")
	g.Expect(doc.Created).To(Equal("2026-09-21"))
	g.Expect(doc.SkillHash).To(Equal(cli.SkillContentHash(skill.Content)))
	g.Expect(doc.Pending).To(BeFalse())
	g.Expect(newContent).To(ContainSubstring("Mirrors skill `agent-instructions/skills/curate/SKILL.md`"))
	g.Expect(newContent).To(ContainSubstring("1. Judge offers."))

	referencing, _ := vault.get("2000.2026-09-22.some-other-note.md")
	g.Expect(referencing).To(ContainSubstring("[[" + newBasename + "]]"))
	g.Expect(referencing).NotTo(ContainSubstring("[[" + oldBasename + "]]"))

	_, oldSidecarStillThere := vault.get(oldBasename + ".vec.json")
	g.Expect(oldSidecarStillThere).To(BeFalse())

	_, newSidecarExists := vault.get(newBasename + ".vec.json")
	g.Expect(newSidecarExists).To(BeTrue())
}

// TestRefreshSkill_PendingClearedBeforehandIsSetAgain covers "refresh of a
// note with pending cleared sets it again": a note whose pending marker was
// already cleared by curation still ends up pending: true after a refresh.
func TestRefreshSkill_PendingClearedBeforehandIsSetAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", curateSkillNoteFixture(cli.SkillContentHash([]byte("old"))))

	newSkill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\nRevised again.\n")}

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", newSkill, basename, skillAcceptDeps(vault), &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, _ := vault.get(basename + ".md")
	doc := parseSkillAcceptFrontmatter(g, updated)

	g.Expect(doc.Pending).To(BeTrue())
}

// TestRefreshSkill_PropagatesEmbedError covers RefreshSkill's sidecar rebuild
// failure branch.
func TestRefreshSkill_PropagatesEmbedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", curateSkillNoteFixture(cli.SkillContentHash([]byte("old"))))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := skillAcceptDeps(vault)
	deps.Embedder = skillAcceptFailingEmbedder{}

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", skill, basename, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestRefreshSkill_PropagatesLockError covers RefreshSkill's lock-acquisition
// failure branch.
func TestRefreshSkill_PropagatesLockError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}
	deps := skillAcceptDeps(newSkillAcceptFixtureVault())
	deps.Lock = func(string) (func(), error) { return nil, errSkillAcceptForTest }

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", skill, "1049.2026-09-21.skill-curate", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestRefreshSkill_PropagatesMalformedFrontmatterError covers
// applySkillNoteBody's no-frontmatter error branch reached through
// RefreshSkill.
func TestRefreshSkill_PropagatesMalformedFrontmatterError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", "no frontmatter here\n")

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", skill, basename, skillAcceptDeps(vault), &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestRefreshSkill_PropagatesReadError covers RefreshSkill's note-read
// failure branch.
func TestRefreshSkill_PropagatesReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}
	deps := skillAcceptDeps(newSkillAcceptFixtureVault())

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", skill, "1049.2026-09-21.skill-curate", deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestRefreshSkill_PropagatesWriteError covers RefreshSkill's write failure
// branch.
func TestRefreshSkill_PropagatesWriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", curateSkillNoteFixture(cli.SkillContentHash([]byte("old"))))

	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n")}

	deps := skillAcceptDeps(vault)
	deps.Write = func(string, []byte) error { return errSkillAcceptForTest }

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", skill, basename, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSkillAcceptForTest))
}

// TestRefreshSkill_RedFlagsSurviveRefresh is the learn-runbook-capture
// scenario "Authored fields survive a refresh": red_flags authored on the
// note (simulating a prior `engram amend`) are still present after an
// accepted refresh.
func TestRefreshSkill_RedFlagsSurviveRefresh(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	vault.put(basename+".md", curateSkillNoteFixture(cli.SkillContentHash([]byte("old"))))

	newSkill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\nRevised procedure.\n")}

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", newSkill, basename, skillAcceptDeps(vault), &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, _ := vault.get(basename + ".md")
	doc := parseSkillAcceptFrontmatter(g, updated)

	g.Expect(doc.RedFlags).To(Equal([]string{"do not compose a new learn", "do not report outcome to caller"}))
}

// TestRefreshSkill_ReplacesBodyKeepsFieldsAndMarksPending covers the
// "Accepted refresh" scenario: the note's new body/hash land, its authored
// fields and basename are unchanged, and it carries pending: true.
func TestRefreshSkill_ReplacesBodyKeepsFieldsAndMarksPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-curate"
	oldHash := cli.SkillContentHash([]byte("old curate body"))
	vault.put(basename+".md", curateSkillNoteFixture(oldHash))

	newSkill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\nStep 1. New procedure text.\n")}

	deps := skillAcceptDeps(vault)

	var stdout bytes.Buffer

	err := cli.RefreshSkill(t.Context(), "/vault", newSkill, basename, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, ok := vault.get(basename + ".md")
	g.Expect(ok).To(BeTrue())

	doc := parseSkillAcceptFrontmatter(g, updated)

	g.Expect(doc.SkillHash).To(Equal(cli.SkillContentHash(newSkill.Content)))
	g.Expect(doc.Pending).To(BeTrue())
	g.Expect(doc.Situation).To(Equal("curating pending offers against the host vault"))
	g.Expect(doc.DoneWhen).To(Equal("every pending offer has been discarded, folded, or accepted"))
	g.Expect(doc.RedFlags).To(Equal([]string{"do not compose a new learn", "do not report outcome to caller"}))
	g.Expect(doc.Triggers).To(Equal([]string{"curate", "pending offers"}))
	g.Expect(doc.Created).To(Equal("2026-09-21"))
	g.Expect(doc.Luhmann).To(Equal("1049"))
	g.Expect(updated).To(ContainSubstring("Step 1. New procedure text."))
	g.Expect(updated).NotTo(ContainSubstring("old curate body"))

	_, sidecarWritten := vault.get(basename + ".vec.json")
	g.Expect(sidecarWritten).To(BeTrue(), "refresh must rebuild the sidecar")
}

// TestRegisterSkill_CreatesPendingNoteWithHashAndPreamble covers the
// "Accepted registration" scenario (skill-runbook-registration): registering
// curate creates a runbook note ending in .skill-curate.md carrying the
// skill's text, skill_hash, pending: true, and a .vec.json sidecar.
func TestRegisterSkill_CreatesPendingNoteWithHashAndPreamble(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skill := cli.ShippedSkill{Name: "curate", Content: []byte("# Curate\n\nStep 1. Judge each pending offer.\n")}

	var (
		writtenPath    string
		writtenContent []byte
	)

	deps := registerSkillLearnDeps(vault, &writtenPath, &writtenContent)

	var stdout bytes.Buffer

	err := cli.RegisterSkill(t.Context(), "/vault", "personal", skill, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("/vault/1.2026-09-25.skill-curate.md"))

	content := string(writtenContent)
	g.Expect(content).To(ContainSubstring("type: runbook"))
	g.Expect(content).To(ContainSubstring("skill_hash: " + cli.SkillContentHash(skill.Content)))
	g.Expect(content).To(ContainSubstring("pending: true"))
	g.Expect(content).To(ContainSubstring("Mirrors skill `agent-instructions/skills/curate/SKILL.md`"))
	g.Expect(content).To(ContainSubstring("Step 1. Judge each pending offer."))

	_, sidecarWritten := vault.get("1.2026-09-25.skill-curate.vec.json")
	g.Expect(sidecarWritten).To(BeTrue(), "register must embed on write")
}

// TestRegisterSkill_NeverSetsRunbookFields proves registration's note has no
// situation, done_when, triggers, or red_flags (skill-runbook-registration:
// "Accepting registration SHALL create a pending note without runbook
// fields") — the omitempty added to runbookFrontmatterDoc's Situation/
// DoneWhen (plus the pre-existing omitempty on RedFlags/Triggers) drops them
// from the rendered frontmatter entirely.
func TestRegisterSkill_NeverSetsRunbookFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	skill := cli.ShippedSkill{Name: "route", Content: []byte("# Route\n\nDelegate everything.\n")}

	var (
		writtenPath    string
		writtenContent []byte
	)

	deps := registerSkillLearnDeps(vault, &writtenPath, &writtenContent)

	var stdout bytes.Buffer

	err := cli.RegisterSkill(t.Context(), "/vault", "personal", skill, deps, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	frontmatter, hasFM := cli.ExportSplitFrontmatter(writtenContent)
	g.Expect(hasFM).To(BeTrue())

	var doc skillAcceptFrontmatterProbe

	g.Expect(yaml.Unmarshal(frontmatter, &doc)).To(Succeed())

	g.Expect(doc.Situation).To(BeEmpty())
	g.Expect(doc.DoneWhen).To(BeEmpty())
	g.Expect(doc.RedFlags).To(BeEmpty())
	g.Expect(doc.Triggers).To(BeEmpty())
	g.Expect(string(writtenContent)).NotTo(ContainSubstring("situation:"))
	g.Expect(string(writtenContent)).NotTo(ContainSubstring("done_when:"))
	g.Expect(string(writtenContent)).NotTo(ContainSubstring("triggers:"))
	g.Expect(string(writtenContent)).NotTo(ContainSubstring("red_flags:"))
}

// TestRemoveSkill_DeletesNoteAndSidecar covers the "Skill no longer shipped"
// scenario: accepting removal deletes both the note and its sidecar.
func TestRemoveSkill_DeletesNoteAndSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1053.2026-09-21.skill-write-memory"
	vault.put(basename+".md", curateSkillNoteFixture("wm-hash"))
	vault.put(basename+".vec.json", `{"model_id":"m"}`)

	var stdout bytes.Buffer

	err := cli.RemoveSkill("/vault", basename, skillAcceptDeps(vault), &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	_, noteStillThere := vault.get(basename + ".md")
	g.Expect(noteStillThere).To(BeFalse())

	_, sidecarStillThere := vault.get(basename + ".vec.json")
	g.Expect(sidecarStillThere).To(BeFalse())
}

// TestRemoveSkill_ToleratesMissingSidecar proves a note that predates
// embed-on-write (or already lacks a sidecar) is not an error for removal —
// discardNote's existing tolerance, reused here.
func TestRemoveSkill_ToleratesMissingSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1053.2026-09-21.skill-write-memory"
	vault.put(basename+".md", curateSkillNoteFixture("wm-hash"))

	var stdout bytes.Buffer

	err := cli.RemoveSkill("/vault", basename, skillAcceptDeps(vault), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
}

// unexported variables.
var (
	errSkillAcceptForTest = errors.New("skillreg-accept fixture: injected failure")
)

// skillAcceptFailingEmbedder always fails to embed, modeled on
// auto_embed_test.go's failingEmbedder.
type skillAcceptFailingEmbedder struct{}

func (skillAcceptFailingEmbedder) Dims() int { return 4 }

func (skillAcceptFailingEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errSkillAcceptForTest
}

func (skillAcceptFailingEmbedder) ModelID() string { return "m@4" }

// skillAcceptFakeEmbedder is a deterministic 4-dim stub embedder, modeled on
// auto_embed_test.go's successEmbedder.
type skillAcceptFakeEmbedder struct{}

func (skillAcceptFakeEmbedder) Dims() int { return 4 }

func (skillAcceptFakeEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func (skillAcceptFakeEmbedder) ModelID() string { return "m@4" }

// unexported test helpers.

// skillAcceptFixtureVault is an in-memory fake vault backing SkillAcceptDeps,
// SkillAdoptDeps, and LearnDeps closures without touching a real filesystem
// (DI everywhere). It directly implements vaultgraph.VaultFS (ListMD,
// ReadFile) so it can also back AdoptSkillNote's Scan dependency via
// vaultgraph.ScanVault.
type skillAcceptFixtureVault struct {
	root  string
	files map[string]string
}

func (v *skillAcceptFixtureVault) ListMD(vault string) ([]string, error) {
	prefix := vault + "/"

	names := []string{}

	for path := range v.files {
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		rest := strings.TrimPrefix(path, prefix)
		if strings.Contains(rest, "/") {
			continue
		}

		if strings.HasSuffix(rest, ".md") {
			names = append(names, rest)
		}
	}

	sort.Strings(names)

	return names, nil
}

func (v *skillAcceptFixtureVault) ReadFile(path string) ([]byte, error) {
	content, ok := v.files[path]
	if !ok {
		return nil, fmt.Errorf("skillreg-accept fixture: %w: %s", fs.ErrNotExist, path)
	}

	return []byte(content), nil
}

func (v *skillAcceptFixtureVault) Remove(path string) error {
	if _, ok := v.files[path]; !ok {
		return fmt.Errorf("skillreg-accept fixture: %w: %s", fs.ErrNotExist, path)
	}

	delete(v.files, path)

	return nil
}

func (v *skillAcceptFixtureVault) Rename(oldPath, newPath string) error {
	content, ok := v.files[oldPath]
	if !ok {
		return fmt.Errorf("skillreg-accept fixture: %w: %s", fs.ErrNotExist, oldPath)
	}

	delete(v.files, oldPath)
	v.files[newPath] = content

	return nil
}

func (v *skillAcceptFixtureVault) WriteFile(path string, data []byte) error {
	v.files[path] = string(data)

	return nil
}

func (v *skillAcceptFixtureVault) get(name string) (string, bool) {
	content, ok := v.files[v.root+"/"+name]

	return content, ok
}

func (v *skillAcceptFixtureVault) put(name, content string) {
	v.files[v.root+"/"+name] = content
}

// skillAcceptFrontmatterProbe reads back the runbook frontmatter fields these
// tests assert on.
type skillAcceptFrontmatterProbe struct {
	Type      string   `yaml:"type"`
	Situation string   `yaml:"situation"`
	DoneWhen  string   `yaml:"done_when"`
	RedFlags  []string `yaml:"red_flags"`
	Triggers  []string `yaml:"triggers"`
	Luhmann   string   `yaml:"luhmann"`
	Created   string   `yaml:"created"`
	SkillHash string   `yaml:"skill_hash"`
	Pending   bool     `yaml:"pending"`
}

// curatePromotedNoteFixture renders a pre-registration promoted note (no
// skill_hash, not pending) at 1049.2026-09-21 — the shape of the real notes
// design D6 describes adopting.
func curatePromotedNoteFixture() string {
	return curatePromotedNoteFixtureAt("1049", "2026-09-21")
}

func curatePromotedNoteFixtureAt(id, date string) string {
	return "---\n" +
		"type: runbook\n" +
		"situation: reviewing pending offers in a vault and judging each against existing notes\n" +
		"done_when: every pending offer has been discarded, folded, or accepted\n" +
		"red_flags:\n" +
		"    - do not compose a new learn\n" +
		"    - do not report outcome to caller\n" +
		"triggers:\n" +
		"    - curate\n" +
		"    - pending offers\n" +
		"luhmann: \"" + id + "\"\n" +
		"created: " + date + "\n" +
		"source: promoted from curate skill\n" +
		"user: joe\n" +
		"vault: personal\n" +
		"---\n\n" +
		"1. Find pending offers.\n2. Judge each.\n"
}

// curatePromotedNoteFixturePending is curatePromotedNoteFixture with a
// pre-existing pending: true marker, for the "clears a pre-existing pending
// marker" decision test.
func curatePromotedNoteFixturePending() string {
	return "---\n" +
		"type: runbook\n" +
		"situation: reviewing pending offers in a vault and judging each against existing notes\n" +
		"done_when: every pending offer has been discarded, folded, or accepted\n" +
		"red_flags:\n" +
		"    - do not compose a new learn\n" +
		"    - do not report outcome to caller\n" +
		"triggers:\n" +
		"    - curate\n" +
		"    - pending offers\n" +
		"luhmann: \"1049\"\n" +
		"created: 2026-09-21\n" +
		"source: promoted from curate skill\n" +
		"user: joe\n" +
		"vault: personal\n" +
		"pending: true\n" +
		"---\n\n" +
		"1. Find pending offers.\n2. Judge each.\n"
}

// curateSkillNoteFixture renders an already-registered skill-curate note
// (basename 1049.2026-09-21.skill-curate) carrying skill_hash and the
// authored fields adoption/refresh must preserve.
func curateSkillNoteFixture(hash string) string {
	return "---\n" +
		"type: runbook\n" +
		"situation: curating pending offers against the host vault\n" +
		"done_when: every pending offer has been discarded, folded, or accepted\n" +
		"red_flags:\n" +
		"    - do not compose a new learn\n" +
		"    - do not report outcome to caller\n" +
		"triggers:\n" +
		"    - curate\n" +
		"    - pending offers\n" +
		"luhmann: \"1049\"\n" +
		"created: 2026-09-21\n" +
		"source: \"skill registration: agent-instructions/skills/curate/SKILL.md\"\n" +
		"user: joe\n" +
		"vault: personal\n" +
		"skill_hash: \"" + hash + "\"\n" +
		"pending: false\n" +
		"---\n\n" +
		"> Mirrors skill `agent-instructions/skills/curate/SKILL.md` — edit the procedure there; " +
		"the runbook fields on this note are authored here.\n\n" +
		"old curate body\n"
}

func newSkillAcceptFixtureVault() *skillAcceptFixtureVault {
	return &skillAcceptFixtureVault{root: "/vault", files: map[string]string{}}
}

func noLock(string) (func(), error) { return func() {}, nil }

func parseSkillAcceptFrontmatter(g Gomega, content string) skillAcceptFrontmatterProbe {
	frontmatter, ok := cli.ExportSplitFrontmatter([]byte(content))
	g.Expect(ok).To(BeTrue())

	var doc skillAcceptFrontmatterProbe

	g.Expect(yaml.Unmarshal(frontmatter, &doc)).To(Succeed())

	return doc
}

// referencingNoteFixture renders a note whose body wikilinks oldBasename —
// used to prove AdoptSkillNote's rename rewrites inbound references.
func referencingNoteFixture(oldBasename string) string {
	return "---\n" +
		"type: fact\n" +
		"situation: s\n" +
		"subject: sub\n" +
		"predicate: pred\n" +
		"object: obj\n" +
		"luhmann: \"2000\"\n" +
		"created: 2026-09-22\n" +
		"source: s\n" +
		"user: u\n" +
		"vault: v\n" +
		"---\n\n" +
		"Information learned: see [[" + oldBasename + "]] for details.\n"
}

func registerSkillLearnDeps(vault *skillAcceptFixtureVault, writtenPath *string, writtenContent *[]byte) cli.LearnDeps {
	return cli.LearnDeps{
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "" },
		Now:        func() time.Time { return time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:     func(string) string { return "" },
		StatDir:    func(string) error { return nil },
		ListIDs:    func(string) ([]string, error) { return nil, nil },
		Lock:       noLock,
		WriteNew: func(path string, data []byte) error {
			*writtenPath = path
			*writtenContent = data
			vault.files[path] = string(data)

			return nil
		},
		Embedder: skillAcceptFakeEmbedder{},
		WriteSidecar: func(path string, data []byte) error {
			vault.files[path] = string(data)

			return nil
		},
		LogWarning: func(string, ...any) {},
	}
}

func skillAcceptDeps(vault *skillAcceptFixtureVault) cli.SkillAcceptDeps {
	return cli.SkillAcceptDeps{
		Lock:     noLock,
		Read:     vault.ReadFile,
		Write:    vault.WriteFile,
		Remove:   vault.Remove,
		Embedder: skillAcceptFakeEmbedder{},
	}
}

func skillAcceptRenameDeps(vault *skillAcceptFixtureVault) cli.RenameRewriteDeps {
	return cli.RenameRewriteDeps{
		ListMD:    vault.ListMD,
		ReadFile:  vault.ReadFile,
		WriteFile: vault.WriteFile,
		Rename:    vault.Rename,
	}
}
