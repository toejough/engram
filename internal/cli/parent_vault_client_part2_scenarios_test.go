package cli_test

// Scenario tests for the parent-vault-client technical component, part 2 (861c2b): what a pull-down skips
// (792b37c14), telling the parent the note was used (792b37c15) and re-checking a linked note's parent copy
// (792b37c16). Spec: docs/specs/technical/parent-vault-client.md.
//
// The CLI journeys run over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860) and
// a fake parent behind Deps.Fetch, with no network and no real home. The race of 792b37c14a6 is forced, not left
// to the scheduler: both pulls must have fetched before either takes the vault lock.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// Test792b37c14a1_Activate_ALinkedLiveNoteIsBumpedNotCopied verifies 792b37c14a1 (A linked live note is bumped,
// not copied): a live local note with a covered or primary link to the parent note at its current hash is not
// copied when the note is pulled again; its last-used date becomes today, and a live note without that link is
// left alone.
func Test792b37c14a1_Activate_ALinkedLiveNoteIsBumpedNotCopied(t *testing.T) {
	t.Parallel()

	for _, via := range []string{"covered", "pulled", "offered"} {
		t.Run("a "+via+" link", func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.bumped", "c", ""))
			hash := env.parent.hashOf(t, parentNote)

			env.plant(pvc2LinkedFile, []byte(localNoteWithLinks(parentVaultID,
				"    - note: "+parentNote+"\n      via: "+via+"\n      hash: "+hash+"\n")))
			env.plantSidecar(pvc2LinkedFile, pvc2StaleDate)
			env.plant(pvc2BystanderFile, []byte(localNoteWithLinks(parentVaultID,
				"    - note: 9.2026-09-01.other\n      via: offered\n      hash: xh1:other\n")))
			env.plantSidecar(pvc2BystanderFile, pvc2StaleDate)

			stdout, stderr := env.run("activate", "--note", parentNote+".md")

			g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
			g.Expect(stdout).To(BeEmpty(), "no copy path is printed")
			g.Expect(env.noteFiles()).To(Equal([]string{pvc2LinkedFile, pvc2BystanderFile}), "nothing is written")
			g.Expect(env.lastUsed(pvc2LinkedFile)).To(Equal(pvc2Today))
			g.Expect(env.lastUsed(pvc2BystanderFile)).To(Equal(pvc2StaleDate), "a note without the link is not bumped")
		})
	}
}

// Test792b37c14a2_Activate_ADeclineSuppressesAnUnchangedNote verifies 792b37c14a2 (A decline suppresses an
// unchanged note): after the owner discards a pulled copy, pulling the unchanged note again writes nothing,
// whether the decline was recorded under the note's current name or under an old name now in its aliases.
func Test792b37c14a2_Activate_ADeclineSuppressesAnUnchangedNote(t *testing.T) {
	t.Parallel()

	const renamedNote = "12.2026-09-25.declined-renamed"

	cases := map[string]struct {
		renamed bool
		byNew   bool
	}{
		"under its own name":                     {},
		"under an old name, pulled by that name": {renamed: true},
		"under an old name, pulled by the new":   {renamed: true, byNew: true},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.declined", "c", ""))
			hash := env.parent.hashOf(t, parentNote)

			env.run("activate", "--note", parentNote+".md")

			copies := env.linkedCopies(parentNote)
			g.Expect(copies).To(HaveLen(1))

			if len(copies) != 1 {
				return
			}

			_, discardErr := env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--discard")
			g.Expect(env.exitCodes()).To(BeEmpty(), discardErr)

			recorded := []declinedEntry{{Vault: parentVaultID, Basename: parentNote, Hash: hash}}
			g.Expect(env.declined()).To(Equal(recorded))

			ref := parentNote
			if testCase.renamed {
				env.parent.rename(parentNote, renamedNote)
			}

			if testCase.byNew {
				ref = renamedNote
			}

			shownBefore := len(env.parent.shown())

			stdout, stderr := env.run("activate", "--note", ref+".md")

			g.Expect(env.parent.shown()).To(HaveLen(shownBefore+1), "the parent was asked, and the decline then applied")
			g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
			g.Expect(stdout).To(BeEmpty())
			g.Expect(env.noteFiles()).To(BeEmpty(), "nothing is written")
			g.Expect(env.declined()).To(Equal(recorded))
		})
	}
}

// Test792b37c14a3_Activate_ADeclineUnderAnotherParentDoesNotApply verifies 792b37c14a3 (A decline under another
// parent does not apply): a decline of the same note name and hash recorded under another parent vault does not
// suppress the pull, which writes the pending copy.
func Test792b37c14a3_Activate_ADeclineUnderAnotherParentDoesNotApply(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.elsewhere", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	otherParent := seqID(pvc2OtherParentSeed)
	env.stampVault("")
	env.writeState("declined.json", fmt.Sprintf(`{"version":1,"entries":[{"vault":%q,"basename":%q,"hash":%q}]}`,
		otherParent, parentNote, hash))

	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	copies := env.linkedCopies(parentNote)
	g.Expect(copies).To(HaveLen(1), "the pull is written")

	if len(copies) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, copies[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: hash}}))
	g.Expect(env.declined()).To(Equal([]declinedEntry{{Vault: otherParent, Basename: parentNote, Hash: hash}}),
		"the other parent's decline is left as it was")
}

// Test792b37c14a4_Activate_AChangedNoteArrivesAgain verifies 792b37c14a4 (A changed note arrives again): a note
// pulled, accepted and then changed on the parent is pulled again as a second, pending copy holding the new
// hash, beside the accepted copy that keeps the old one.
func Test792b37c14a4_Activate_AChangedNoteArrivesAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.changing", "c", ""))
	oldHash := env.parent.hashOf(t, parentNote)
	accepted := acceptedPulledCopy(t, env, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))

	newHash := env.parent.hashOf(t, parentNote)
	g.Expect(newHash).NotTo(Equal(oldHash))

	stdout, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	second := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == accepted })
	g.Expect(second).To(HaveLen(1), "a second copy is written")

	if len(second) != 1 {
		return
	}

	g.Expect(stdout).To(ContainSubstring(second[0]))

	pending := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, second[0]))))
	g.Expect(pending.Pending).To(BeTrue())
	g.Expect(pending.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: newHash}}))

	kept := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, accepted))))
	g.Expect(kept.Pending).To(BeFalse(), "the accepted copy is not touched")
	g.Expect(kept.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: oldHash}}))
}

// Test792b37c14a5_Activate_ACopyThatLandsMidPullIsSeen verifies 792b37c14a5 (A copy that lands mid-pull is seen):
// a copy written between the fetch and the lock is found by the re-check under the lock, so the pull writes
// nothing and prints no path.
func Test792b37c14a5_Activate_ACopyThatLandsMidPullIsSeen(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.raced", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	landed := localNoteWithLinks(parentVaultID, "    - note: "+parentNote+"\n      via: pulled\n      hash: "+hash+"\n")

	var landing sync.Once

	env.wrap = func(deps *cli.Deps) {
		fetch := deps.Fetch
		deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
			resp, err := fetch(ctx, method, target, body)
			if strings.Contains(target, "/show?") {
				landing.Do(func() {
					env.plant(pvc2LandedFile, []byte(landed))
					env.plantSidecar(pvc2LandedFile, pvc2StaleDate)
				})
			}

			return resp, err
		}
	}

	stdout, stderr := env.run("activate", "--note", parentNote+".md")

	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
	g.Expect(stdout).To(BeEmpty(), "no copy path is printed")
	g.Expect(env.noteFiles()).To(Equal([]string{pvc2LandedFile}), "the re-check finds the copy and writes nothing")
	g.Expect(readFileString(t, filepath.Join(env.vault, pvc2LandedFile))).To(Equal(landed))
	g.Expect(env.lastUsed(pvc2LandedFile)).To(Equal(pvc2Today), "the live copy that landed is bumped")
}

// Test792b37c14a6_Activate_RacingPullsWriteOneCopy verifies 792b37c14a6 (Racing pulls write one copy): two
// pulls of one note on one vault, held until both have fetched so the second to take the lock meets the first
// one's copy, complete with one copy in the vault and one path printed.
func Test792b37c14a6_Activate_RacingPullsWriteOneCopy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.shared", "c", ""))
	gate := newPvc2Gate(pvc2RacingPulls)

	// the vault is stamped already: the race is over the copy, not over the first contact
	env.stampVault("")

	env.wrap = func(deps *cli.Deps) {
		fetch := deps.Fetch
		deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
			resp, err := fetch(ctx, method, target, body)
			if strings.Contains(target, "/show?") {
				gate.arrive()
			}

			return resp, err
		}
	}

	outputs := make([]string, pvc2RacingPulls)

	var wait sync.WaitGroup

	for index := range pvc2RacingPulls {
		wait.Go(func() {
			stdout, _ := executeCapturingBoth(t,
				[]string{"engram", "activate", "--note", parentNote + ".md", "--vault", env.vault}, env.customize)
			outputs[index] = stdout
		})
	}

	wait.Wait()

	g.Expect(gate.opened()).To(BeTrue(), "both pulls fetched before either one wrote")
	g.Expect(env.exitCodes()).To(BeEmpty())

	copies := env.linkedCopies(parentNote)
	g.Expect(copies).To(HaveLen(1), "one copy exists")
	g.Expect(env.noteFiles()).To(Equal(copies))

	printed := slices.DeleteFunc(outputs, func(stdout string) bool { return stdout == "" })
	g.Expect(printed).To(HaveLen(1), "only the pull that wrote prints a path")
}

// Test792b37c14a7_Activate_NoSequenceDuplicates verifies 792b37c14a7 (No sequence duplicates) as a rule over
// sequences: whatever sequence of pull, clear-pending, discard and fold is applied to an unchanged parent note,
// at most one local note is linked to it after every step.
func Test792b37c14a7_Activate_NoSequenceDuplicates(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		env := newWiringEnvIn(rt, t.TempDir())
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.sequence", "c", ""))
		env.plant(foldTargetFile, []byte(localNoteWithLinks(parentVaultID,
			"    - note: 9.2026-09-01.other\n      via: offered\n      hash: xh1:other\n")))

		steps := rapid.SliceOfN(rapid.SampledFrom([]string{"activate", "clear-pending", "discard", "fold"}),
			1, pvc2MaxSequence).Draw(rt, "steps")

		for index, step := range steps {
			applyPullStep(rt, env, parentNote, step)

			linked := env.linkedCopies(parentNote)
			if len(linked) > 1 {
				rt.Fatalf("after step %d of %v: %d local notes linked to %s: %v",
					index+1, steps, len(linked), parentNote, linked)
			}
		}
	})
}

// Test792b37c15a1_Activate_TheParentsRecencyIsBumped verifies 792b37c15a1 (The parent's recency is bumped): a
// pulled note and a reachable parent end with exactly one POST /activate for the note, sent while no vault lock
// is held.
func Test792b37c15a1_Activate_TheParentsRecencyIsBumped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.bumped", "c", ""))

	probe := &lockProbe{}
	env.wrap = probe.wrap

	_, stderr := env.run("activate", "--note", parentNote+".md")

	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1), "the note was pulled")

	signals := pvc2ActivateRequests(env)
	g.Expect(signals).To(HaveLen(1), "one POST /activate")

	if len(signals) != 1 {
		return
	}

	g.Expect(signals[0].method).To(Equal("POST"))
	g.Expect(env.parent.activated()).To(Equal([]string{parentNote + ".md"}), "for the pulled note")
	g.Expect(probe.fetchesUnderLock()).To(BeEmpty(), "no request is made while the vault lock is held")

	fetched := probe.fetched()
	showAt := slices.IndexFunc(fetched, func(target string) bool { return strings.Contains(target, "/show?") })
	activateAt := slices.IndexFunc(fetched, func(target string) bool { return strings.HasSuffix(target, "/activate") })
	g.Expect(showAt).To(BeNumerically(">=", 0))
	g.Expect(activateAt).To(BeNumerically(">", showAt), "the signal follows the fetch")
}

// Test792b37c15a2_Activate_AFailedSignalIsHarmless verifies 792b37c15a2 (A failed signal is harmless): when the
// parent's activate request fails, by a server error or by no answer at all, the command succeeds with the copy
// written, nothing is queued and the parent is not backed off.
func Test792b37c15a2_Activate_AFailedSignalIsHarmless(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		setup    func(env *wiringEnv, attempts *atomic.Int32)
		attempts int32
	}{
		"the parent answers a server error": {setup: func(env *wiringEnv, _ *atomic.Int32) {
			env.parent.activateDown = true
		}},
		"the request gets no answer": {attempts: 1, setup: func(env *wiringEnv, attempts *atomic.Int32) {
			env.wrap = func(deps *cli.Deps) {
				fetch := deps.Fetch
				deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
					if strings.HasSuffix(target, "/activate") {
						attempts.Add(1)

						return cli.FetchResponse{}, errPvc2NoAnswer
					}

					return fetch(ctx, method, target, body)
				}
			}
		}},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.bumpfail", "c", ""))

			var attempts atomic.Int32

			testCase.setup(env, &attempts)

			stdout, stderr := env.run("activate", "--note", parentNote+".md")

			g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
			g.Expect(stderr).NotTo(ContainSubstring("unreachable"), "no backoff warning")
			g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
			g.Expect(stdout).To(ContainSubstring(".md"))
			g.Expect(env.outboxEntries()).To(BeEmpty(), "nothing is queued")

			cache := pvc2ParentCache(t, env)
			g.Expect(cache).To(HaveKeyWithValue("failures", BeNumerically("==", 0)), "no failure is recorded")
			g.Expect(cache).NotTo(HaveKey("backoff_until"), "the parent is not backed off")

			g.Expect(attempts.Load()).To(Equal(testCase.attempts), "an unanswered request was made and failed")
		})
	}
}

// Test792b37c15a3_Activate_AFailedLocalWriteSendsNoSignal verifies 792b37c15a3 (A failed local write sends no
// signal): of two notes pulled in one command, the one whose local write fails makes the command fail and gets no POST
// /activate, while the other, written, does.
func Test792b37c15a3_Activate_AFailedLocalWriteSendsNoSignal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	doomed := env.parent.addNote(pulledFact("7.2026-09-01.doomed", "c", ""))
	spared := env.parent.addNote(pulledFact("8.2026-09-01.spared", "c", ""))

	env.wrap = func(deps *cli.Deps) { deps.FS = pvc2NoteWriteFailsFS{EdgeFS: deps.FS, marker: ".doomed."} }

	_, stderr := env.run("activate", "--note", doomed+".md", "--note", spared+".md")

	g.Expect(env.exitCodes()).To(Equal([]int{1}), "the command fails")
	g.Expect(stderr).To(ContainSubstring(doomed + ".md"))
	g.Expect(stderr).To(ContainSubstring(errPvc2NoteWrite.Error()))
	g.Expect(env.linkedCopies(doomed)).To(BeEmpty(), "the failed write left no copy")
	g.Expect(env.linkedCopies(spared)).To(HaveLen(1), "the other note was written")
	g.Expect(env.parent.activated()).To(Equal([]string{spared + ".md"}), "no POST /activate for the failed note")
}

// Test792b37c16a1_Activate_AChangedParentNoteArrivesThroughTheLocalCopy verifies 792b37c16a1 (A changed parent
// note arrives through the local copy): after a pulled copy is accepted and the parent's note changes,
// activating the local copy writes a pending copy with the new hash once, and activating it again writes nothing
// more.
func Test792b37c16a1_Activate_AChangedParentNoteArrivesThroughTheLocalCopy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.round-trip", "c", ""))
	local := acceptedPulledCopy(t, env, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))

	newHash := env.parent.hashOf(t, parentNote)

	pulled, stderr := env.run("activate", "--note", local)
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	update := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == local })
	g.Expect(update).To(HaveLen(1), "a copy is written")

	if len(update) != 1 {
		return
	}

	g.Expect(pulled).To(ContainSubstring(update[0]))

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, update[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: newHash}}))

	again, againErr := env.run("activate", "--note", local)
	g.Expect(env.exitCodes()).To(BeEmpty(), againErr)
	g.Expect(again).To(BeEmpty(), "no path is printed the second time")
	g.Expect(env.noteFiles()).To(ConsistOf(local, update[0]), "the second activation writes nothing more")
}

// Test792b37c16a2_Activate_AnUnchangedOrDeclinedNoteWritesNothing verifies 792b37c16a2 (An unchanged or declined
// note writes nothing): activating a linked note whose parent hash equals the link's, and one whose changed
// parent version was declined, writes nothing in either case, though the parent is asked once each time.
func Test792b37c16a2_Activate_AnUnchangedOrDeclinedNoteWritesNothing(t *testing.T) {
	t.Parallel()

	t.Run("a parent hash equal to the link's", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.steady", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)
		shownBefore := len(env.parent.shown())

		stdout, stderr := env.run("activate", "--note", local)

		g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
		g.Expect(stdout).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{local}))
		g.Expect(env.parent.shown()[shownBefore:]).To(Equal([]string{parentNote + ".md"}), "the parent was asked once")
	})

	t.Run("a changed version that was declined", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.declined-update", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)

		env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
		env.run("activate", "--note", local)

		update := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == local })
		g.Expect(update).To(HaveLen(1))

		if len(update) != 1 {
			return
		}

		_, discardErr := env.run("amend", "--target", strings.TrimSuffix(update[0], ".md"), "--discard")
		g.Expect(env.exitCodes()).To(BeEmpty(), discardErr)

		declined := env.declined()
		g.Expect(declined).To(HaveLen(1))

		shownBefore := len(env.parent.shown())

		stdout, stderr := env.run("activate", "--note", local)

		g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
		g.Expect(stdout).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{local}), "the declined version is not pulled again")
		g.Expect(env.declined()).To(Equal(declined))
		g.Expect(env.parent.shown()[shownBefore:]).To(Equal([]string{parentNote + ".md"}), "the parent was asked once")
	})
}

// Test792b37c16a3_Activate_ALinkUnderAnotherParentIsNeverWritten verifies 792b37c16a3 (A link under another
// parent is never written): a link whose vault differs from the configured parent's cached ID is not contacted,
// and a link whose vault differs from the answer's gets its answer discarded; neither writes a note.
func Test792b37c16a3_Activate_ALinkUnderAnotherParentIsNeverWritten(t *testing.T) {
	t.Parallel()

	otherParent := seqID(pvc2OtherParentSeed)

	t.Run("a vault that differs from the cached ID", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.elsewhere", "c", ""))
		env.stampVault("")
		env.writeState("parent.json", `{"url":"`+parentURL+`","vault_id":"`+parentVaultID+`","failures":0}`)
		env.plant(pvc2LinkedFile, []byte(localNoteWithLinks(otherParent,
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

		_, stderr := env.run("activate", "--note", pvc2LinkedFile)

		g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
		g.Expect(env.parent.requests()).To(BeEmpty(), "the parent is not contacted")
		g.Expect(env.noteFiles()).To(Equal([]string{pvc2LinkedFile}))
	})

	t.Run("a vault that differs from the answer's", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.other-vault", "c", ""))
		env.plant(pvc2LinkedFile, []byte(localNoteWithLinks(otherParent,
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

		stdout, stderr := env.run("activate", "--note", pvc2LinkedFile)

		g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
		g.Expect(stdout).To(BeEmpty())
		g.Expect(env.parent.shown()).To(Equal([]string{parentNote + ".md"}), "the parent answered once")
		g.Expect(env.noteFiles()).To(Equal([]string{pvc2LinkedFile}), "the answer is discarded")
	})
}

// Test792b37c16a4_Activate_AnUnreachableParentDoesNotFailTheActivate verifies 792b37c16a4 (An unreachable parent
// does not fail the activate): activating a linked note while the parent is unreachable succeeds, bumps the
// local note, and records one failure with the first backoff window from the clock.
func Test792b37c16a4_Activate_AnUnreachableParentDoesNotFailTheActivate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.offline", "c", ""))
	env.plant(pvc2LinkedFile, []byte(localNoteWithLinks(parentVaultID,
		"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))
	env.plantSidecar(pvc2LinkedFile, pvc2StaleDate)
	env.parent.setDown(true)

	_, stderr := env.run("activate", "--note", pvc2LinkedFile)

	g.Expect(env.exitCodes()).To(BeEmpty(), "the activate succeeds")
	g.Expect(env.lastUsed(pvc2LinkedFile)).To(Equal(pvc2Today), "the local note was activated")
	g.Expect(env.noteFiles()).To(Equal([]string{pvc2LinkedFile}))
	g.Expect(stderr).To(ContainSubstring("parent unreachable (retry after "))

	cache := pvc2ParentCache(t, env)
	g.Expect(cache).To(HaveKeyWithValue("failures", BeNumerically("==", 1)), "the failure is recorded")
	g.Expect(cache).To(HaveKeyWithValue("backoff_until", env.now().Add(pvc2FirstBackoff).Format(time.RFC3339)))
}

// unexported constants.
const (
	pvc2BarrierTimeout  = 10 * time.Second
	pvc2BystanderFile   = "4.2026-09-20.bystander.md"
	pvc2FirstBackoff    = 30 * time.Second
	pvc2LandedFile      = "5.2026-09-28.landed.md"
	pvc2LinkedFile      = "3.2026-09-20.local.md"
	pvc2MaxSequence     = 8
	pvc2OtherParentSeed = 77
	pvc2RacingPulls     = 2
	pvc2StaleDate       = "2026-01-01"
	pvc2Today           = "2026-09-28"
)

// unexported variables.
var (
	errPvc2NoAnswer  = errors.New("dial tcp: connection refused")
	errPvc2NoteWrite = errors.New("note write refused")
)

// pvc2Gate holds each arriving party until all of them have arrived, or the timeout passes.
type pvc2Gate struct {
	mu      sync.Mutex
	parties int
	arrived int
	open    chan struct{}
}

// arrive blocks until every party has arrived.
func (gate *pvc2Gate) arrive() {
	gate.mu.Lock()

	gate.arrived++
	if gate.arrived == gate.parties {
		close(gate.open)
	}

	gate.mu.Unlock()

	select {
	case <-gate.open:
	case <-time.After(pvc2BarrierTimeout):
	}
}

// opened reports whether every party arrived.
func (gate *pvc2Gate) opened() bool {
	select {
	case <-gate.open:
		return true
	default:
		return false
	}
}

// pvc2NoteWriteFailsFS fails the exclusive create of every note file whose path holds marker.
type pvc2NoteWriteFailsFS struct {
	cli.EdgeFS

	marker string
}

func (f pvc2NoteWriteFailsFS) WriteFileExcl(path string, data []byte, perm os.FileMode) error {
	if strings.Contains(path, f.marker) {
		return errPvc2NoteWrite
	}

	return f.EdgeFS.WriteFileExcl(path, data, perm)
}

// newPvc2Gate is a gate for parties arrivals.
func newPvc2Gate(parties int) *pvc2Gate {
	return &pvc2Gate{parties: parties, open: make(chan struct{})}
}

// pvc2ActivateRequests lists the requests the parent saw for POST /activate.
func pvc2ActivateRequests(env *wiringEnv) []recordedRequest {
	return slices.DeleteFunc(env.parent.requests(), func(request recordedRequest) bool {
		return !strings.HasSuffix(request.url, "/activate")
	})
}

// pvc2ParentCache is .engram/parent.json as a generic map, so a key the file omits stays absent.
func pvc2ParentCache(t *testing.T, env *wiringEnv) map[string]any {
	t.Helper()

	cache := map[string]any{}

	raw, err := os.ReadFile(filepath.Join(env.vault, ".engram", "parent.json"))
	if err != nil {
		return cache
	}

	err = json.Unmarshal(raw, &cache)
	if err != nil {
		t.Fatal(err)
	}

	return cache
}
