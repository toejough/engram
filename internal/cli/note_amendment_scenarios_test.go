package cli_test

// Scenario-ID tests of the Note Amendment technical component (docs/specs/technical/note-amendment.md):
// 792b14c12 Activation against the parent, 792b14c13 A discard of a pulled note records its decline,
// 792b14c14 Settling a pending offer. Activation scenarios drive RunActivate over recorded func-field
// collaborators; discard scenarios drive RunAmend over an in-memory vault with the production decline
// recording composed over an in-memory exchange-state filesystem; the --clear-pending scenarios run the
// command, whose flag translation lives in targets.go (real temp vault, the recorded deviation #860).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

//go:generate impgen na1StepFunc --dependency --name MockNa1Step

// Test792b14c12a1_RunActivate_ALocalHitIsBumpedAndTheParentIsNotContacted verifies 792b14c12a1
// (docs/specs/technical/note-amendment.md): a ref whose note file exists, with or without a sidecar, is
// a local hit: its use is recorded and no pull is requested.
func Test792b14c12a1_RunActivate_ALocalHitIsBumpedAndTheParentIsNotContacted(t *testing.T) {
	t.Parallel()

	for name, withSidecar := range map[string]bool{"with a sidecar": true, "without a sidecar": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note file that exists, with or without a sidecar, and a parent configured
			rig := na1NewActivateRig()
			rig.plant(na1LocalName, na1LocalNote)

			if withSidecar {
				rig.plant(na1LocalSidecar, string(validFoldSidecar()))
			}

			// When it is activated
			err := cli.ExportRunActivate(t.Context(),
				cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1LocalRef}}, rig.deps(true))

			// Then its use is recorded and no pull is requested
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(rig.pulls).To(BeEmpty())
			g.Expect(rig.rechecks).To(BeEmpty())
			g.Expect(rig.warnings).To(BeEmpty())

			if !withSidecar {
				g.Expect(rig.writes).To(BeEmpty())

				return
			}

			g.Expect(rig.writes).To(Equal([]string{filepath.Join(na1Vault, na1LocalSidecar)}))

			sidecar, parseErr := embed.UnmarshalSidecar([]byte(rig.file(na1LocalSidecar)))
			g.Expect(parseErr).NotTo(HaveOccurred())

			if parseErr != nil {
				return
			}

			g.Expect(sidecar.LastUsed).To(Equal(na1Today))
		})
	}
}

// Test792b14c12a2_RunActivate_ABasenameThatMissesIsPulledDown verifies 792b14c12a2: with a parent configured,
// a basename (or <basename>.md) with no local note reaches the pull-down, and no vault lock is held then.
func Test792b14c12a2_RunActivate_ABasenameThatMissesIsPulledDown(t *testing.T) {
	t.Parallel()

	for name, ref := range map[string]string{"a basename": na1ParentBase, "a basename with .md": na1ParentBase + ".md"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a parent is configured and a basename that has no local note
			rig := na1NewActivateRig()

			// When it is activated
			err := cli.ExportRunActivate(t.Context(),
				cli.ActivateArgs{Vault: na1Vault, Notes: []string{ref}}, rig.deps(true))

			// Then the pull-down receives the ref with no vault lock held
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(rig.pulls).To(Equal([]string{ref}))
			g.Expect(rig.pullLockHeld).To(Equal([]bool{false}))
			g.Expect(rig.lockCount).To(Equal(1), "the local pass did take the lock, then released it")
		})
	}
}

// Test792b14c12a3_RunActivate_AnIDOrAPathIsNotSent verifies 792b14c12a3: a bare Luhmann ID with no local
// note, and with --parent a path ref, fail as not sent to the parent, and nothing is requested of the parent.
// Pull, Recheck and Finish are imptest mocks with no expectation: a call to any of them leaves the run waiting.
func Test792b14c12a3_RunActivate_AnIDOrAPathIsNotSent(t *testing.T) {
	t.Parallel()

	const (
		notSent  = "not sent to the parent"
		notFound = "note not found"
	)

	cases := map[string]struct {
		ref          string
		parent       bool
		wantNotFound bool
	}{
		"a bare ID":               {ref: "1100", wantNotFound: true},
		"a bare ID with .md":      {ref: "1100.md", wantNotFound: true},
		"a bare ID with --parent": {ref: "1100", parent: true},
		"a path with --parent":    {ref: "/elsewhere/" + na1ParentBase + ".md", parent: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a ref that is a bare ID with no local note, or with --parent a path
			rig := na1NewActivateRig()

			// When it is activated
			err := na1RunMockedActivate(t, rig,
				cli.ActivateArgs{Vault: na1Vault, Notes: []string{tc.ref}, Parent: tc.parent}, nil)

			// Then it fails as not sent to the parent and nothing is requested
			g.Expect(err).To(MatchError("activate: note ref(s) failed: 1 of 1"))
			g.Expect(rig.warnings).To(HaveLen(1))

			if len(rig.warnings) != 1 {
				return
			}

			g.Expect(rig.warnings[0]).To(ContainSubstring(tc.ref))
			g.Expect(rig.warnings[0]).To(ContainSubstring(notSent))

			if tc.wantNotFound {
				g.Expect(rig.warnings[0]).To(ContainSubstring(notFound))
			} else {
				g.Expect(rig.warnings[0]).NotTo(ContainSubstring(notFound))
			}
		})
	}
}

// Test792b14c12a4_RunActivate_ParentResolvesEveryRefAgainstTheParent verifies 792b14c12a4: with --parent a ref
// whose local file exists still goes to the pull-down, and the local file is neither looked up nor bumped.
func Test792b14c12a4_RunActivate_ParentResolvesEveryRefAgainstTheParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a local file of the ref's name exists, with a sidecar
	rig := na1NewActivateRig()
	rig.plant(na1ParentBase+".md", na1LocalNote)
	rig.plant(na1ParentBase+".vec.json", string(validFoldSidecar()))
	sidecarBefore := rig.file(na1ParentBase + ".vec.json")

	// When it is activated with --parent
	err := cli.ExportRunActivate(t.Context(),
		cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1ParentBase}, Parent: true}, rig.deps(true))

	// Then the pull-down receives it and the local file is not bumped
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(rig.pulls).To(Equal([]string{na1ParentBase}))
	g.Expect(rig.noteLookups).To(BeEmpty(), "with --parent no ref is looked up locally")
	g.Expect(rig.writes).To(BeEmpty())
	g.Expect(rig.file(na1ParentBase + ".vec.json")).To(Equal(sidecarBefore))
}

// Test792b14c12a5_RunActivate_ParentWithNoParentFailsBeforeAnyLookup verifies 792b14c12a5: with --parent and no
// pull-down configured, the command fails with the not-configured error and nothing is read or written.
func Test792b14c12a5_RunActivate_ParentWithNoParentFailsBeforeAnyLookup(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given no pull-down is configured, and a local note with a sidecar
	rig := na1NewActivateRig()
	rig.plant(na1LocalName, na1LocalNote)
	rig.plant(na1LocalSidecar, string(validFoldSidecar()))

	// When it is activated with --parent
	err := cli.ExportRunActivate(t.Context(),
		cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1LocalRef}, Parent: true}, rig.deps(false))

	// Then it fails with the not-configured error and nothing is read or written
	g.Expect(err).To(MatchError("--parent requires ENGRAM_PARENT to be configured"))
	g.Expect(rig.events).To(BeEmpty(), "not even the vault lock was taken")
	g.Expect(rig.noteLookups).To(BeEmpty())
	g.Expect(rig.reads).To(BeEmpty())
	g.Expect(rig.writes).To(BeEmpty())
	g.Expect(rig.warnings).To(BeEmpty())
}

// Test792b14c12a6_RunActivate_ALinkedHitIsRecheckedAndAFailureIsNotFatal verifies 792b14c12a6: a local hit
// linked to a parent note under its parent vault is re-checked after the local pass, with no lock held, and
// a re-check that fails does not fail the activate.
func Test792b14c12a6_RunActivate_ALinkedHitIsRecheckedAndAFailureIsNotFatal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a local hit linked to a parent note under its parent vault, and a re-check that fails
	rig := na1NewActivateRig()
	rig.plant(na1LocalName, na1LinkedNote)
	rig.plant(na1LocalSidecar, string(validFoldSidecar()))
	rig.recheckErr = errNa1Recheck

	// When it is activated
	err := cli.ExportRunActivate(t.Context(),
		cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1LocalRef}}, rig.deps(true))

	// Then the re-check is attempted after the local pass and the activate succeeds
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}
	g.Expect(rig.rechecks).To(Equal([]cli.ExportLinkedParentNote{
		cli.ExportNewLinkedParentNote(foldParentVault, na1ParentBase),
	}))
	g.Expect(rig.events).To(Equal([]string{"lock", "bump", "unlock", "recheck", "finish"}))
	g.Expect(rig.warnings).To(BeEmpty())
}

// Test792b14c12a7_RunActivate_TheFinishStepRunsOnce verifies 792b14c12a7: when any pull or re-check was
// attempted the finish step runs exactly once, after all of them; when none was attempted it does not run.
// Pull, Recheck and Finish are imptest mocks: the script names each call in order, and a call beyond it (a
// second finish) leaves the run waiting.
func Test792b14c12a7_RunActivate_TheFinishStepRunsOnce(t *testing.T) {
	t.Parallel()

	const (
		secondBase  = "8.2026-09-02.second"
		linkedBaseA = "4.2026-09-03.linked-a"
		linkedBaseB = "4.2026-09-04.linked-b"
	)

	t.Run("several pulls and re-checks", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given two pulls and a local hit that links two parent notes, so two re-checks
		rig := na1NewActivateRig()
		rig.plant(na1LocalName, foldNote(foldExchange{
			vault: foldParentVault,
			links: []foldLink{{linkedBaseA, "pulled", na1ParentHash}, {linkedBaseB, "pulled", na1ParentHash}},
		}, false))
		rig.plant(na1LocalSidecar, string(validFoldSidecar()))

		// When they have all run
		err := na1RunMockedActivate(t, rig,
			cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1ParentBase, na1LocalRef, secondBase}},
			func(steps *Na1StepMockMethod) {
				steps.ArgsEqual(na1StepPull, na1ParentBase, cli.ExportLinkedParentNote{}).Return(nil)
				steps.ArgsEqual(na1StepPull, secondBase, cli.ExportLinkedParentNote{}).Return(nil)
				steps.ArgsEqual(na1StepRecheck, "", cli.ExportNewLinkedParentNote(foldParentVault, linkedBaseA)).Return(nil)
				steps.ArgsEqual(na1StepRecheck, "", cli.ExportNewLinkedParentNote(foldParentVault, linkedBaseB)).Return(nil)
				steps.ArgsEqual(na1StepFinish, "", cli.ExportLinkedParentNote{}).Return(nil)
			})

		// Then the finish step ran exactly once, after every pull and re-check (the run ended after one finish)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}
	})

	t.Run("a pull that fails was still attempted", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a pull that fails
		rig := na1NewActivateRig()

		// When the ref is activated
		err := na1RunMockedActivate(t, rig,
			cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1ParentBase}},
			func(steps *Na1StepMockMethod) {
				steps.ArgsEqual(na1StepPull, na1ParentBase, cli.ExportLinkedParentNote{}).Return(errNa1Pull)
				steps.ArgsEqual(na1StepFinish, "", cli.ExportLinkedParentNote{}).Return(nil)
			})

		// Then the finish step still ran once
		g.Expect(err).To(MatchError("activate: note ref(s) failed: 1 of 1"))
	})

	t.Run("none attempted", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a local hit with no parent link, and a ref that is refused before any pull
		rig := na1NewActivateRig()
		rig.plant(na1LocalName, na1LocalNote)
		rig.plant(na1LocalSidecar, string(validFoldSidecar()))

		// When they are activated
		err := na1RunMockedActivate(t, rig,
			cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1LocalRef, "1100"}}, nil)

		// Then no pull, re-check or finish ran (the run ended with no call to a mock)
		g.Expect(err).To(MatchError("activate: note ref(s) failed: 1 of 2"))
	})
}

// Test792b14c12a8_RunActivate_FailedRefsAreCounted verifies 792b14c12a8: of two refs one fails; a warning
// names the failed ref and the command fails saying 1 of 2 failed.
func Test792b14c12a8_RunActivate_FailedRefsAreCounted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given two refs of which one has no note and no pull-down is configured
	const missingRef = "9.2026-09-01.gone"

	rig := na1NewActivateRig()
	rig.plant(na1LocalName, na1LocalNote)
	rig.plant(na1LocalSidecar, string(validFoldSidecar()))

	// When they are activated
	err := cli.ExportRunActivate(t.Context(),
		cli.ActivateArgs{Vault: na1Vault, Notes: []string{na1LocalRef, missingRef}}, rig.deps(false))

	// Then a warning names the failed ref and the command fails saying 1 of 2 failed
	g.Expect(err).To(MatchError("activate: note ref(s) failed: 1 of 2"))
	g.Expect(rig.warnings).To(HaveLen(1))

	if len(rig.warnings) != 1 {
		return
	}

	g.Expect(rig.warnings[0]).To(ContainSubstring(missingRef))
	g.Expect(rig.warnings[0]).NotTo(ContainSubstring(na1LocalRef))
}

// Test792b14c13a1_RunAmend_ADeclineThatCannotBeRecordedKeepsTheNote verifies 792b14c13a1: a bare discard of a
// pulled note whose decline cannot be recorded fails, and the note and its sidecar remain.
func Test792b14c13a1_RunAmend_ADeclineThatCannotBeRecordedKeepsTheNote(t *testing.T) {
	t.Parallel()

	declinedPath := filepath.Join(na1Vault, ".engram", "declined.json")

	cases := map[string]func(state *memVaultFS){
		"the record cannot be written": func(state *memVaultFS) { state.failAtomicWrite(declinedPath, errNa1DiskFull) },
		"the record cannot be read":    func(state *memVaultFS) { state.failReadOf(declinedPath, errNa1DiskFull) },
	}

	for name, breakRecord := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a pulled note with a sidecar and a declined-pull record that cannot be written
			rig := na1NewDiscardRig(true)
			rig.putNote(na1PulledNote())
			breakRecord(rig.state)

			// When the note is discarded
			err := rig.discard(t.Context())

			// Then the discard fails and the note and its sidecar remain
			g.Expect(err).To(MatchError(errNa1DiskFull))
			g.Expect(rig.vault.files).To(HaveKey(foldOfferName))
			g.Expect(rig.vault.files).To(HaveKey(na1OfferSidecar))
			g.Expect(rig.stdout.String()).To(BeEmpty())
		})
	}
}

// Test792b14c13a2_RunAmend_ARepeatedDeclineIsRecordedOnce verifies 792b14c13a2: a pulled note whose decline is
// already recorded, then another pulled note with the same basename and hash discarded: the record holds the
// entry once.
func Test792b14c13a2_RunAmend_ARepeatedDeclineIsRecordedOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a pulled note whose decline is already recorded
	rig := na1NewDiscardRig(true)
	rig.putNote(na1PulledNote())
	g.Expect(rig.discard(t.Context())).To(Succeed())

	declinedPath := filepath.Join(na1Vault, ".engram", "declined.json")
	writesAfterFirst := na1CountPaths(rig.state.atomicWrites(), declinedPath)
	g.Expect(writesAfterFirst).To(Equal(1))

	// When another pulled note with the same basename and hash is discarded
	rig.putNote(na1PulledNote())
	g.Expect(rig.discard(t.Context())).To(Succeed())

	// Then the record holds the entry once
	var record struct {
		Entries []struct {
			Vault    string `json:"vault"`
			Basename string `json:"basename"`
			Hash     string `json:"hash"`
		} `json:"entries"`
	}

	g.Expect(json.Unmarshal(rig.state.read(declinedPath), &record)).To(Succeed())
	g.Expect(record.Entries).To(HaveLen(1))

	if len(record.Entries) != 1 {
		return
	}

	g.Expect(record.Entries[0].Vault).To(Equal(foldParentVault))
	g.Expect(record.Entries[0].Basename).To(Equal(na1ParentBase))
	g.Expect(record.Entries[0].Hash).To(Equal(na1ParentHash))
	g.Expect(rig.vault.files).NotTo(HaveKey(foldOfferName), "the second note was deleted too")
}

// Test792b14c13a3_RunAmend_ANoteThatIsNotPulledRecordsNothing verifies 792b14c13a3: a note whose primary link
// is offered, or that has no link, is deleted by a bare discard and nothing is recorded.
func Test792b14c13a3_RunAmend_ANoteThatIsNotPulledRecordsNothing(t *testing.T) {
	t.Parallel()

	cases := map[string]foldExchange{
		"a primary link that is offered": {
			vault: foldParentVault, links: []foldLink{{na1ParentBase, "offered", na1ParentHash}},
		},
		"no link": {},
	}

	for name, exchange := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note whose primary link is offered, or that has no link (a parent is configured)
			rig := na1NewDiscardRig(true)
			rig.putNote(foldNote(exchange, true))

			// When it is discarded
			err := rig.discard(t.Context())

			// Then it is deleted and nothing is recorded
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}
			g.Expect(rig.vault.files).NotTo(HaveKey(foldOfferName))
			g.Expect(rig.vault.files).NotTo(HaveKey(na1OfferSidecar))
			g.Expect(rig.state.files).To(BeEmpty(), "no declined record, no vault ID")
			g.Expect(rig.state.dirs).NotTo(HaveKey(na1StateDir), "no .engram directory")
		})
	}
}

// Test792b14c13a4_RunAmend_AnUnconfiguredVaultRecordsNothing verifies 792b14c13a4: in a vault with no parent
// configured and no vault ID, discarding a pulled note deletes it, records no decline, and creates neither a
// vault ID nor an .engram directory.
func Test792b14c13a4_RunAmend_AnUnconfiguredVaultRecordsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with no parent configured and no vault ID, holding a pulled note
	rig := na1NewDiscardRig(false)
	rig.putNote(na1PulledNote())

	// When the pulled note is discarded
	err := rig.discard(t.Context())

	// Then it is deleted, no decline is recorded, and no vault ID or .engram directory is created
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}
	g.Expect(rig.vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(rig.vault.files).NotTo(HaveKey(na1OfferSidecar))
	g.Expect(rig.state.files).To(BeEmpty(), "no declined record, no .engram-vault-id")
	g.Expect(rig.state.dirs).NotTo(HaveKey(na1StateDir), "no .engram directory")
	g.Expect(rig.stdout.String()).To(ContainSubstring(foldOfferName))
}

// Test792b14c14a1_Targets_Amend_ClearingTheMarkerMakesTheNoteLive verifies 792b14c14a1: `engram amend --target
// <note> --clear-pending` on a pending fact note leaves a note that no longer carries pending: true, with
// every other line unchanged.
func Test792b14c14a1_Targets_Amend_ClearingTheMarkerMakesTheNoteLive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a pending fact note
	vault := t.TempDir()
	before := foldNote(foldExchange{}, true)
	na1PlantNote(t, vault, na1PendingName, before)
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(before))).To(BeTrue(), "the fixture is a pending offer")

	// When engram amend --target <note> --clear-pending runs
	stdout, stderr := na1RunAmendCommand(t, vault, "--target", strings.TrimSuffix(na1PendingName, ".md"),
		"--clear-pending")

	// Then the note no longer carries pending: true, and nothing else about it changed
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(ContainSubstring(na1PendingName))

	after := na1ReadNote(t, vault, na1PendingName)
	g.Expect(after).NotTo(ContainSubstring("pending:"))
	g.Expect(after).To(Equal(strings.Replace(before, "pending: true\n", "", 1)))
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(after))).To(BeFalse())
}

// Test792b14c14a2_Targets_Amend_ASkillNoteKeepsItsSkillHashWhenCleared verifies 792b14c14a2: clearing the
// marker of a pending runbook note that carries skill_hash removes the marker, leaves skill_hash unchanged and
// leaves the note no longer a pending offer.
func Test792b14c14a2_Targets_Amend_ASkillNoteKeepsItsSkillHashWhenCleared(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a pending runbook note that carries skill_hash
	vault := t.TempDir()
	before := skillSurvivalNote()
	na1PlantNote(t, vault, na1SkillName, before)
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(before))).To(BeTrue(), "the fixture is a pending offer")
	g.Expect(before).To(ContainSubstring("skill_hash:"))

	// When engram amend --target <note> --clear-pending runs
	_, stderr := na1RunAmendCommand(t, vault, "--target", strings.TrimSuffix(na1SkillName, ".md"),
		"--clear-pending")

	// Then the marker is gone, skill_hash is unchanged and the note is no longer a pending offer
	g.Expect(stderr).To(BeEmpty())

	after := na1ReadNote(t, vault, na1SkillName)
	g.Expect(after).NotTo(ContainSubstring("pending:"))
	expectSkillIdentityUnchanged(g, before, after)
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(after))).To(BeFalse())
}

// Test792b14c14a3_RunAmend_ALocalAmendNeverSetsTheMarker verifies 792b14c14a3: amending a live note's content
// (engram amend --target <note> --object="amended", and the same for the feedback and runbook fields) leaves
// a note with no pending line.
func Test792b14c14a3_RunAmend_ALocalAmendNeverSetsTheMarker(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args     cli.AmendArgs
		wantText string
	}{
		"fact":     {args: cli.AmendArgs{Object: "amended"}, wantText: "object: amended"},
		"feedback": {args: cli.AmendArgs{Action: "amended"}, wantText: "action: amended"},
		"runbook":  {args: cli.AmendArgs{DoneWhen: "amended"}, wantText: "done_when: amended"},
	}

	for noteType, tc := range cases {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a live note
			const liveName = "1aa.2026-01-01.live.md"

			vault := newFoldVault()
			vault.put(liveName, amendParityMinimalNote(noteType))

			args := tc.args
			args.Target = strings.TrimSuffix(liveName, ".md")

			// When it is amended
			err := vault.amend(t.Context(), args)

			// Then the amended note carries no pending line
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			amended := string(vault.files[liveName])
			g.Expect(amended).To(ContainSubstring(tc.wantText), "the amend did rewrite the note")
			g.Expect(amended).NotTo(ContainSubstring("pending"))
		})
	}
}

// unexported constants.
const (
	na1ConfiguredURL = "http://parent.example.invalid"
	na1LocalName     = "3.2026-09-20.local.md"
	na1LocalRef      = "3.2026-09-20.local"
	na1LocalSidecar  = "3.2026-09-20.local.vec.json"
	na1OfferSidecar  = "5.2026-09-28.offer.vec.json"
	na1ParentBase    = "7.2026-09-01.parent"
	na1ParentHash    = "xh1:parent"
	na1PendingName   = "2.2026-09-20.pending.md"
	na1SkillName     = "1a.2026-01-01.skill-note.md"
	na1StateDir      = "/vault/.engram"
	na1StepFinish    = "finish"
	na1StepPull      = "pull"
	na1StepRecheck   = "recheck"
	na1Today         = "2026-09-28"
	na1Vault         = "/vault"
)

// unexported variables.
var (
	errNa1DiskFull = errors.New("na1: disk full")
	errNa1Pull     = errors.New("na1: parent unreachable")
	errNa1Recheck  = errors.New("na1: recheck failed")
	na1LinkedNote  = foldNote(foldExchange{
		vault: foldParentVault, links: []foldLink{{na1ParentBase, "pulled", na1ParentHash}},
	}, false)
	na1LocalNote = foldNote(foldExchange{}, false)
)

// na1ActivateRig is one activate run's recorded collaborators: an in-memory vault, an event log of the lock,
// bump, pull, recheck and finish steps, and the arguments each step saw.
type na1ActivateRig struct {
	files        map[string]string
	events       []string
	warnings     []string
	pulls        []string
	pullLockHeld []bool
	rechecks     []cli.ExportLinkedParentNote
	reads        []string
	writes       []string
	noteLookups  []string
	lockCount    int
	lockHeld     bool
	pullErr      error
	recheckErr   error
}

// deps wires the rig; withParent also wires the pull-down, the re-check and the finish step.
func (r *na1ActivateRig) deps(withParent bool) cli.ActivateDeps {
	deps := cli.ActivateDeps{
		Lock: func(string) (func(), error) {
			r.lockHeld = true
			r.lockCount++
			r.events = append(r.events, "lock")

			return func() {
				r.lockHeld = false

				r.events = append(r.events, "unlock")
			}, nil
		},
		Now: func() time.Time { return time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC) },
		Read: func(path string) ([]byte, error) {
			r.reads = append(r.reads, path)

			content, found := r.files[path]
			if !found {
				return nil, fs.ErrNotExist
			}

			return []byte(content), nil
		},
		Write: func(path string, data []byte) error {
			r.writes = append(r.writes, path)
			r.events = append(r.events, "bump")
			r.files[path] = string(data)

			return nil
		},
		LogWarning: func(format string, args ...any) { r.warnings = append(r.warnings, fmt.Sprintf(format, args...)) },
		NoteExists: func(path string) bool {
			r.noteLookups = append(r.noteLookups, path)
			_, found := r.files[path]

			return found
		},
	}

	if !withParent {
		return deps
	}

	deps.Pull = func(_ context.Context, ref string) error {
		r.pulls = append(r.pulls, ref)
		r.pullLockHeld = append(r.pullLockHeld, r.lockHeld)
		r.events = append(r.events, "pull:"+ref)

		return r.pullErr
	}
	deps.Recheck = func(_ context.Context, link cli.ExportLinkedParentNote) error {
		r.rechecks = append(r.rechecks, link)
		r.events = append(r.events, "recheck")

		return r.recheckErr
	}
	deps.Finish = func(context.Context) {
		r.events = append(r.events, "finish")
	}

	return deps
}

// file is the content of the rig's vault file name.
func (r *na1ActivateRig) file(name string) string { return r.files[filepath.Join(na1Vault, name)] }

// plant stores a file of the rig's vault under name.
func (r *na1ActivateRig) plant(name, content string) {
	r.files[filepath.Join(na1Vault, name)] = content
}

// na1DiscardRig is an in-memory vault driving a bare --discard, whose RecordDecline is the production
// composition (newAmendDeps) over an in-memory exchange-state filesystem.
type na1DiscardRig struct {
	vault  *foldVault
	state  *memVaultFS
	deps   cli.AmendDeps
	stdout bytes.Buffer
}

// discard runs `engram amend --target <note> --discard` on the offer note.
func (r *na1DiscardRig) discard(ctx context.Context) error {
	args := cli.AmendArgs{Vault: na1Vault, Target: foldOfferBase, Discard: true}

	return cli.ExportRunAmend(ctx, args, r.deps, &r.stdout)
}

// putNote plants the offer note and its sidecar.
func (r *na1DiscardRig) putNote(content string) {
	r.vault.put(foldOfferName, content)
	r.vault.put(na1OfferSidecar, validFoldSidecar())
}

// na1StateFS is the EdgeFS of the production amend composition for the discard rig: only the exchange-state
// surface is backed, by the in-memory filesystem.
type na1StateFS struct {
	cli.EdgeFS

	state *memVaultFS
}

func (s na1StateFS) MkdirAll(path string, perm fs.FileMode) error {
	return s.state.MkdirAll(path, perm)
}

func (s na1StateFS) ReadFile(path string) ([]byte, error) { return s.state.ReadFile(path) }

func (s na1StateFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return s.state.WriteFileAtomic(path, data, perm)
}

func (s na1StateFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return s.state.WriteFileExcl(path, data, perm)
}

// na1StepFunc is one parent step of an activate run as a single call: the step name, the ref a pull received and
// the link a recheck received. ActivateDeps.Pull, Recheck and Finish are anonymous func fields, so na1WithSteps
// wires all three to one mock (impgen generates MockNa1Step from this type), which puts the order of the steps
// into the script.
type na1StepFunc func(step, ref string, link cli.ExportLinkedParentNote) error

func na1CountPaths(paths []string, want string) int {
	count := 0

	for _, path := range paths {
		if path == want {
			count++
		}
	}

	return count
}

func na1NewActivateRig() *na1ActivateRig {
	return &na1ActivateRig{files: map[string]string{}}
}

// na1NewDiscardRig builds the rig; parentConfigured sets ENGRAM_PARENT for the production composition.
func na1NewDiscardRig(parentConfigured bool) *na1DiscardRig {
	state := newMemVaultFS()
	state.mkdir(na1Vault)

	getenv := func(string) string { return "" }
	if parentConfigured {
		getenv = func(key string) string {
			if key == "ENGRAM_PARENT" {
				return na1ConfiguredURL
			}

			return ""
		}
	}

	production := cli.ExportNewAmendDeps(cli.Deps{
		FS:           na1StateFS{state: state},
		Getenv:       getenv,
		RandRead:     seqRand(1),
		EvalSymlinks: identityPath,
		Getwd:        func() (string, error) { return "/", nil },
	})

	vault := newFoldVault()
	deps := vault.deps()
	deps.RecordDecline = production.RecordDecline

	return &na1DiscardRig{vault: vault, state: state, deps: deps}
}

// na1PlantNote writes a note into a real temp vault (the command-level scenarios).
func na1PlantNote(t *testing.T, vault, name, content string) {
	t.Helper()

	writeFoldFile(t, filepath.Join(vault, name), content)
}

// na1PulledNote is a pending local copy whose primary link is via pulled.
func na1PulledNote() string {
	return foldNote(foldExchange{
		vault: foldParentVault, links: []foldLink{{na1ParentBase, "pulled", na1ParentHash}},
	}, true)
}

func na1ReadNote(t *testing.T, vault, name string) string {
	t.Helper()

	return readFileString(t, filepath.Join(vault, name))
}

// na1RunAmendCommand runs `engram amend <args> --vault <vault>` over the real command wiring with the home,
// the environment and the embedder pinned to scratch values; it returns stdout and stderr.
func na1RunAmendCommand(t *testing.T, vault string, args ...string) (string, string) {
	t.Helper()

	home := t.TempDir()
	argv := append([]string{"engram", "amend"}, args...)
	argv = append(argv, "--vault", vault, "--chunks-dir", t.TempDir())

	return executeCapturingBoth(t, argv, func(deps *cli.Deps) {
		deps.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		deps.Getenv = func(string) string { return "" }
		deps.UserHomeDir = func() (string, error) { return home, nil }
	})
}

// na1RunMockedActivate runs RunActivate over the rig with Pull, Recheck and Finish replaced by imptest mocks, in
// a goroutine, plays script (when given) against the mocks' expectation handles, and waits for both with a time
// limit: a call the script does not expect leaves the run waiting on the mock, and a call the script expects but
// never gets leaves the script waiting.
func na1RunMockedActivate(
	t *testing.T, rig *na1ActivateRig, args cli.ActivateArgs, script func(steps *Na1StepMockMethod),
) error {
	t.Helper()

	step, stepCalls := MockNa1Step(t)
	deps := na1WithSteps(rig.deps(true), step)

	done := make(chan error, 1)

	go func() { done <- cli.ExportRunActivate(t.Context(), args, deps) }()

	if script != nil {
		scripted := make(chan struct{})

		go func() {
			defer close(scripted)

			script(stepCalls)
		}()

		select {
		case <-scripted:
		case <-time.After(mockTimeout):
			t.Fatal("the parent steps did not get every call the script expects")
		}
	}

	select {
	case err := <-done:
		return err
	case <-time.After(mockTimeout):
		t.Fatal("the activate did not finish: it made a call to a parent step that the script does not expect")
	}

	return nil
}

// na1WithSteps wires the parent steps of deps to step.
func na1WithSteps(deps cli.ActivateDeps, step na1StepFunc) cli.ActivateDeps {
	deps.Pull = func(_ context.Context, ref string) error {
		return step(na1StepPull, ref, cli.ExportLinkedParentNote{})
	}
	deps.Recheck = func(_ context.Context, link cli.ExportLinkedParentNote) error {
		return step(na1StepRecheck, "", link)
	}
	deps.Finish = func(context.Context) {
		_ = step(na1StepFinish, "", cli.ExportLinkedParentNote{})
	}

	return deps
}
