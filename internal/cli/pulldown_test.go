package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestActivate_BareLuhmannIDStaysLocal (spec "Bare Luhmann IDs stay
// local", M8): a bare ID — with or without .md, with or without --parent —
// is never sent to the parent, and the ref is reported as failed.
func TestActivate_BareLuhmannIDStaysLocal(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"bare id":          {"activate", "--note", "1100"},
		"bare id with .md": {"activate", "--note", "1100.md"},
		"with --parent":    {"activate", "--parent", "--note", "1100"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)

			_, stderr := env.run(args...)
			g.Expect(env.parent.requests()).To(BeEmpty())
			g.Expect(stderr).To(ContainSubstring("1100"))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
		})
	}
}

// TestActivate_ConcurrentActivatesWriteOneCopy (review H5, M8): two
// activates of the same parent note racing on one vault write one copy —
// the skip rule is re-checked under the write lock.
func TestActivate_ConcurrentActivatesWriteOneCopy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.shared", "c", ""))

	var wait sync.WaitGroup

	for range 2 {
		wait.Go(func() {
			_, _ = executeCapturingBoth(t,
				[]string{"engram", "activate", "--note", parentNote + ".md", "--vault", env.vault}, env.customize)
		})
	}

	wait.Wait()
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
}

// TestActivate_CoveredLinkSkipsAndBumpsLiveNote (spec "A covered parent
// note is not pulled again"): a live local note whose covered link names
// the parent note with its current hash is bumped instead of a new copy
// being written.
func TestActivate_CoveredLinkSkipsAndBumpsLiveNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.covered", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(
		parentVaultID, "    - note: 9.2026-09-01.other\n      via: offered\n      hash: xh1:other\n"+
			"    - note: "+parentNote+"\n      via: covered\n      hash: "+hash+"\n")))
	env.plantSidecar("3.2026-09-20.local.md", "2026-01-01")

	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(stderr).To(BeEmpty())
	g.Expect(env.exitCodes()).To(BeEmpty())
	g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
	g.Expect(env.lastUsed("3.2026-09-20.local.md")).To(Equal("2026-09-28"))
	g.Expect(env.parent.activated()).To(Equal([]string{parentNote + ".md"}))
}

// TestActivate_DeclineIsKeyedByParentVault (ruling S18, S16): a decline
// recorded under another parent vault does not suppress the same
// basename and hash under the current parent.
func TestActivate_DeclineIsKeyedByParentVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.elsewhere", "c", ""))
	env.stampVault("")
	env.writeState("declined.json", fmt.Sprintf(`{"version":1,"entries":[{"vault":%q,"basename":%q,"hash":%q}]}`,
		seqID(77), parentNote, env.parent.hashOf(t, parentNote)))

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
}

// TestActivate_DeclinedPullIsRemembered (spec "A declined pull is
// remembered", "A decline survives a parent rename", H5, r3-8): a bare
// --discard of a pulled copy records a decline; an unchanged re-pull —
// also after the parent renamed the note — writes nothing.
func TestActivate_DeclinedPullIsRemembered(t *testing.T) {
	t.Parallel()

	for name, rename := range map[string]bool{"same name": false, "renamed on the parent": true} {
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
			g.Expect(env.declined()).To(ConsistOf(declinedEntry{Vault: parentVaultID, Basename: parentNote, Hash: hash}))

			if rename {
				env.parent.rename(parentNote, "12.2026-09-25.declined-renamed")
			}

			env.run("activate", "--note", parentNote+".md")
			g.Expect(env.noteFiles()).To(BeEmpty(), "a declined, unchanged parent note is not pulled again")
			g.Expect(env.exitCodes()).To(BeEmpty())
		})
	}
}

// TestActivate_HashMismatchWritesNothing: an envelope whose exchange hash
// does not match its content is refused; nothing is written.
func TestActivate_HashMismatchWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.mismatch", "c", ""))
	env.parent.showMode = "wrong-hash"

	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(stderr).To(ContainSubstring("exchange hash differs"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty())
}

// TestActivate_LocalContentEditOffersToOrigin (decision 3, B1): after an
// accepted pull, a real local edit is an amend-offer naming the parent
// note; clearing the pending marker alone queues nothing.
func TestActivate_LocalContentEditOffersToOrigin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.edited", "c", ""))
	env.run("activate", "--note", parentNote+".md")

	copies := env.linkedCopies(parentNote)
	g.Expect(copies).To(HaveLen(1))

	if len(copies) != 1 {
		return
	}

	target := strings.TrimSuffix(copies[0], ".md")

	env.run("amend", "--target", target, "--clear-pending")
	g.Expect(env.outboxEntries()).To(BeEmpty(), "accepting a pulled note queues nothing")
	g.Expect(env.parent.offers()).To(BeEmpty())

	env.run("amend", "--target", target, "--object", "locally changed")

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) == 1 {
		g.Expect(offers[0].Offer.For).To(Equal(parentNote))
		g.Expect(offers[0].Object).To(Equal("locally changed"))
	}
}

// TestActivate_LocalHitNeverContactsParent (spec "A local hit never
// contacts the parent", M8): a note whose .md exists is a hit — its
// sidecar is bumped, and a note without a sidecar still counts.
func TestActivate_LocalHitNeverContactsParent(t *testing.T) {
	t.Parallel()

	for name, withSidecar := range map[string]bool{"with sidecar": true, "without sidecar": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.plant("3.2026-09-20.local.md", offerTestNote{}.render(t))

			if withSidecar {
				env.plantSidecar("3.2026-09-20.local.md", "2026-01-01")
			}

			_, stderr := env.run("activate", "--note", "3.2026-09-20.local.md")
			g.Expect(stderr).To(BeEmpty())
			g.Expect(env.exitCodes()).To(BeEmpty())
			g.Expect(env.parent.requests()).To(BeEmpty())

			if withSidecar {
				g.Expect(env.lastUsed("3.2026-09-20.local.md")).To(Equal("2026-09-28"))
			}
		})
	}
}

// TestActivate_MalformedEnvelopeWritesNothing (final review F6): an
// envelope whose vault_id is not 32 lowercase hex, or whose basename is
// not a Luhmann basename free of '/', '\\' and '|', is a malformed reply:
// nothing is written and the bad vault ID is never cached.
func TestActivate_MalformedEnvelopeWritesNothing(t *testing.T) {
	t.Parallel()

	for name, setup := range map[string]func(*recordingParent){
		"vault_id not hex":   func(p *recordingParent) { p.vaultID = "../../not-a-vault" },
		"basename with pipe": func(p *recordingParent) { p.showMode = "bad-basename" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.malformed", "c", ""))
			setup(env.parent)

			_, stderr := env.run("activate", "--note", parentNote+".md")
			g.Expect(stderr).To(ContainSubstring("malformed"))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.noteFiles()).To(BeEmpty())
			g.Expect(env.parentCache().VaultID).To(BeEmpty())
		})
	}
}

// TestActivate_MissWithoutParentFails: with no parent configured a miss
// is a failed ref and nothing is fetched.
func TestActivate_MissWithoutParentFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parentURL = ""

	_, stderr := env.run("activate", "--note", "7.2026-09-01.gone.md")
	g.Expect(stderr).To(ContainSubstring("7.2026-09-01.gone.md"))
	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.parent.requests()).To(BeEmpty())
}

// TestActivate_NonExchangeTypeRefused: a fetched note that is not a fact,
// feedback or runbook is refused and nothing is written.
func TestActivate_NonExchangeTypeRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(fakeParentNote{
		basename: "7.2026-09-01.moc",
		content:  "---\ntype: moc\nluhmann: \"7\"\ncreated: \"2026-09-01\"\nuser: bob\nvault: team\n---\n\n# Map\n",
	})

	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(stderr).To(ContainSubstring(parentNote))
	g.Expect(stderr).To(ContainSubstring("moc"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty())
}

// TestActivate_ParentActivateFailureIsNotFatal (Q2): the best-effort
// parent bump failing neither fails the command nor queues anything.
func TestActivate_ParentActivateFailureIsNotFatal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.bumpfail", "c", ""))
	env.parent.activateDown = true

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty())
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(env.parentCache().Failures).To(BeZero(), "a failed best-effort bump never backs the parent off")
}

// TestActivate_ParentFlagPathRefIsNotSent (ruling S18): with --parent a
// path ref is refused as "not sent", never as "not found" (it was never
// looked up locally).
func TestActivate_ParentFlagPathRefIsNotSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)

	_, stderr := env.run("activate", "--parent", "--note", "/elsewhere/7.2026-09-01.x.md")
	g.Expect(stderr).To(ContainSubstring("not sent to the parent"))
	g.Expect(stderr).NotTo(ContainSubstring("not found"))
	g.Expect(env.parent.requests()).To(BeEmpty())
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
}

// TestActivate_ParentFlagPullsEvenOnLocalHit (D8 step 2): with --parent a
// ref resolves against the parent even when a local file of that name
// exists.
func TestActivate_ParentFlagPullsEvenOnLocalHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.both", "c", ""))
	env.plant(parentNote+".md", offerTestNote{}.render(t))

	env.run("activate", "--parent", "--note", parentNote+".md")
	g.Expect(env.parent.shown()).To(Equal([]string{parentNote + ".md"}))
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
}

// TestActivate_ParentFlagWithoutParentErrors (spec "--parent without a
// parent configured"): an error, and nothing is read or written.
func TestActivate_ParentFlagWithoutParentErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parentURL = ""
	env.plant("3.2026-09-20.local.md", offerTestNote{}.render(t))
	env.plantSidecar("3.2026-09-20.local.md", "2026-01-01")

	_, stderr := env.run("activate", "--parent", "--note", "3.2026-09-20.local.md")
	g.Expect(stderr).To(ContainSubstring("ENGRAM_PARENT"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.lastUsed("3.2026-09-20.local.md")).To(Equal("2026-01-01"))
	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())
}

// TestActivate_ParentTooOld (spec "A pre-change parent is detected", H7):
// a rendered answer, or JSON without vault_id, is "parent too old", and
// nothing is written.
func TestActivate_ParentTooOld(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"rendered", "no-vault-id"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.old", "c", ""))
			env.parent.showMode = mode

			_, stderr := env.run("activate", "--note", parentNote+".md")
			g.Expect(stderr).To(ContainSubstring("too old"))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.noteFiles()).To(BeEmpty())
		})
	}
}

// TestActivate_ParentUnreachablePausesPulls (design D6, M9): an outage on
// the first fetch records one failure, prints the one backoff warning
// with the queued count, and later refs in the same command make no
// request; every ref fails.
func TestActivate_ParentUnreachablePausesPulls(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)

	_, stderr := env.run("activate", "--note", "7.2026-09-01.first.md", "--note", "8.2026-09-01.second.md")
	g.Expect(env.parent.requests()).To(HaveLen(1))
	g.Expect(strings.Count(stderr, "engram: parent unreachable (retry after ")).To(Equal(1))
	g.Expect(stderr).To(ContainSubstring("0 offer(s) queued"))
	g.Expect(stderr).To(ContainSubstring("7.2026-09-01.first.md"))
	g.Expect(stderr).To(ContainSubstring("8.2026-09-01.second.md"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.parentCache().Failures).To(Equal(1))
}

// TestActivate_PartialFailureIsVisible (spec "Partial failure is
// visible", #746): one good ref and one the parent lacks — stderr names
// the failed ref and the command exits non-zero.
func TestActivate_PartialFailureIsVisible(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.plant("3.2026-09-20.local.md", offerTestNote{}.render(t))

	_, stderr := env.run("activate", "--note", "3.2026-09-20.local.md", "--note", "8.2026-09-01.nowhere.md")
	g.Expect(stderr).To(ContainSubstring("8.2026-09-01.nowhere.md"))
	g.Expect(stderr).NotTo(ContainSubstring("3.2026-09-20.local"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
}

// TestActivate_PauseChecks (design D2, rulings S2, S4): a failed location
// check, the backoff window, and the self-parent guard each stop the
// pull-down with its one warning; nothing is written.
func TestActivate_PauseChecks(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		setup   func(env *wiringEnv, parentNote string)
		warning string
		fetches int
	}{
		"location check fails": {
			setup: func(env *wiringEnv, _ string) {
				env.stampVault("/somewhere/else")
			},
			warning: "location check",
		},
		"backoff window": {
			setup: func(env *wiringEnv, _ string) {
				env.stampVault("")
				env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"backoff_until":%q,"failures":2}`,
					env.parentURL, env.now().Add(time.Minute).Format(time.RFC3339)))
			},
			warning: "parent unreachable (retry after",
		},
		"self-parent": {
			setup: func(env *wiringEnv, _ string) {
				env.stampVault("")
				env.parent.vaultID = env.localID()
			},
			warning: "vault-id --regenerate",
			fetches: 1,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.paused", "c", ""))
			testCase.setup(env, parentNote)

			before := len(env.parent.shown())

			_, stderr := env.run("activate", "--note", parentNote+".md")
			g.Expect(stderr).To(ContainSubstring(testCase.warning))
			g.Expect(env.exitCodes()).NotTo(BeEmpty())
			g.Expect(env.parent.shown()).To(HaveLen(before + testCase.fetches))
			g.Expect(env.noteFiles()).To(BeEmpty())
		})
	}
}

// TestActivate_PullDownDrainsTheOutbox (design D6, W7 carry): a
// successful pull-down is a parent contact, after which queued offers
// drain.
func TestActivate_PullDownDrainsTheOutbox(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.drain", "c", ""))

	env.parent.setDown(true)
	env.learnFact("queued")
	env.parent.setDown(false)
	env.advance(time.Minute)

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.parent.offers()).To(HaveLen(1))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// TestActivate_PullDownFetchesWithNoLockHeld (D8 steps 3 and 5): the raw
// fetch and the parent /activate happen with no vault lock held; the
// write happens under it.
func TestActivate_PullDownFetchesWithNoLockHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.nolock", "c", ""))

	probe := &lockProbe{}
	env.wrap = probe.wrap

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
	g.Expect(probe.fetchesUnderLock()).To(BeEmpty())
	g.Expect(probe.fetched()).To(ContainElements(
		ContainSubstring("/show?"), ContainSubstring("/activate")))
}

// TestActivate_PullSequencesNeverDuplicate is 6.2's rapid property (H5,
// M8): any sequence of activate, clear-pending, discard and fold (the real
// `amend --discard --into`, design D10; ruling S17) on an unchanged parent
// note never leaves more than one local note linked to it.
func TestActivate_PullSequencesNeverDuplicate(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		env := newWiringEnvIn(rt, t.TempDir())
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.sequence", "c", ""))
		env.plant(foldTargetFile, []byte(localNoteWithLinks(parentVaultID,
			"    - note: 9.2026-09-01.other\n      via: offered\n      hash: xh1:other\n")))

		steps := rapid.SliceOfN(rapid.SampledFrom([]string{"activate", "clear-pending", "discard", "fold"}), 1, 8).
			Draw(rt, "steps")

		for _, step := range steps {
			applyPullStep(rt, env, parentNote, step)

			linked := env.linkedCopies(parentNote)
			if len(linked) > 1 {
				rt.Fatalf("after %v: %d local notes linked to %s: %v", steps, len(linked), parentNote, linked)
			}
		}
	})
}

// TestActivate_PullsParentNoteAsPendingCopy (spec "A parent-only ref
// pulls down", "Pulled note is pending and linked", "Remote identity and
// registration fields are stripped", "Chunk provenance does not travel",
// "Parent recency is bumped"; D8 step 4).
func TestActivate_PullsParentNoteAsPendingCopy(t *testing.T) {
	t.Parallel()

	for name, parentNote := range map[string]fakeParentNote{
		"fact":    pulledFact("7.2026-09-01.pulled", "c", parentOnlyFields),
		"runbook": pulledRunbook("8.2026-09-02.steps"),
		// A basename whose slug a local filename cannot carry.
		"odd slug": pulledFact("9.2026-09-03.Odd_Slug", "c", ""),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.parent.addNote(parentNote)
			hash := env.parent.hashOf(t, parentNote.basename)

			_, stderr := env.run("activate", "--note", parentNote.basename+".md")
			g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

			files := env.noteFiles()
			g.Expect(files).To(HaveLen(1))

			if len(files) != 1 {
				return
			}

			g.Expect(files[0]).To(MatchRegexp(`^1\.2026-09-28\.[a-z-]+\.md$`), "a fresh top-level local ID")

			raw := []byte(readFileString(t, filepath.Join(env.vault, files[0])))
			copied := decodePulledCopy(t, raw)

			g.Expect(copied.Pending).To(BeTrue())
			g.Expect(copied.XID).To(MatchRegexp(`^[0-9a-f]{32}$`))
			g.Expect(copied.XID).NotTo(Equal(xidC), "the parent's own xid is stripped")
			g.Expect(copied.Luhmann).To(Equal("1"))
			g.Expect(copied.Parent.Vault).To(Equal(parentVaultID))
			g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote.basename, Via: "pulled", Hash: hash}}))
			g.Expect(copied.Parent.Author).To(Equal(pulledAuthor{Repo: "github.com/parent/repo", User: "bob", Vault: "team"}))
			g.Expect(copied.Vault).To(Equal("personal"), "top-level identity is stamped locally")
			g.Expect(copied.User).NotTo(Equal("bob"))
			g.Expect(copied.Aliases).To(BeEmpty())
			g.Expect(copied.Offer).To(BeEmpty())
			g.Expect(copied.Sources).To(BeEmpty())
			g.Expect(copied.Supersedes).To(BeEmpty())
			g.Expect(copied.VocabVersion).To(BeEmpty())
			g.Expect(copied.Tags).NotTo(ContainElement("vocab/parent-term"))
			g.Expect(copied.SkillHash + copied.SkillKey + copied.SkillSource).To(BeEmpty())

			g.Expect(mustHash(t, raw)).To(Equal(hash), "the body and every offered field are verbatim")
			g.Expect(string(embed.BodyText(raw))).To(Equal(string(embed.BodyText([]byte(parentNote.content)))))
			g.Expect(filepath.Join(env.vault, strings.TrimSuffix(files[0], ".md")+".vec.json")).
				To(BeAnExistingFile(), "the copy is embedded on write")

			g.Expect(env.parent.shown()).To(Equal([]string{parentNote.basename + ".md"}))
			g.Expect(env.parent.activated()).To(Equal([]string{parentNote.basename + ".md"}))
			g.Expect(env.parent.offers()).To(BeEmpty(), "a pulled note is never offered back")
			g.Expect(env.outboxEntries()).To(BeEmpty())
			g.Expect(env.parentCache().VaultID).To(Equal(parentVaultID), "the envelope's vault ID is cached")

			stdout, _ := env.run("query", "--phrase", "pulling", "--chunks-dir", t.TempDir())
			g.Expect(stdout).NotTo(ContainSubstring(files[0]), "a pending copy stays out of local results")
			g.Expect(stdout).To(ContainSubstring("pending_offers: true"))
		})
	}
}

// TestActivate_ReActivationSkipsUnlessChanged (spec "Re-activating an
// unchanged parent note", "A changed parent note arrives as a new offer";
// D3 unknown-is-not-changed).
func TestActivate_ReActivationSkipsUnlessChanged(t *testing.T) {
	t.Parallel()

	t.Run("pending copy, unchanged", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.twice", "c", ""))

		env.run("activate", "--note", parentNote+".md")
		env.run("activate", "--note", parentNote+".md")
		g.Expect(env.linkedCopies(parentNote)).To(HaveLen(1))
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.parent.activated()).To(HaveLen(2), "a skip still signals use to the parent")
	})

	t.Run("accepted copy, unchanged, bumps it", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.accepted", "c", ""))

		env.run("activate", "--note", parentNote+".md")

		copies := env.linkedCopies(parentNote)
		g.Expect(copies).To(HaveLen(1))

		if len(copies) != 1 {
			return
		}

		env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--clear-pending")
		env.advance(48 * time.Hour)
		env.run("activate", "--note", parentNote+".md")
		g.Expect(env.linkedCopies(parentNote)).To(Equal(copies))
		g.Expect(env.lastUsed(copies[0])).To(Equal("2026-09-30"))
	})

	t.Run("accepted copy, parent changed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.changing", "c", ""))

		env.run("activate", "--note", parentNote+".md")

		copies := env.linkedCopies(parentNote)
		g.Expect(copies).To(HaveLen(1))

		if len(copies) != 1 {
			return
		}

		env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--clear-pending")
		env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
		env.run("activate", "--note", parentNote+".md")

		after := env.linkedCopies(parentNote)
		g.Expect(after).To(HaveLen(2))

		fresh := slices.DeleteFunc(after, func(name string) bool { return name == copies[0] })
		g.Expect(fresh).To(HaveLen(1))

		if len(fresh) == 1 {
			copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, fresh[0]))))
			g.Expect(copied.Pending).To(BeTrue())
			g.Expect(copied.Parent.Links).To(Equal([]pulledLink{
				{Note: parentNote, Via: "pulled", Hash: env.parent.hashOf(t, parentNote)},
			}))
		}
	})

	t.Run("link under an unknown hash version is not changed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.versioned", "c", ""))
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(parentVaultID,
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh0:legacy\n")))

		env.run("activate", "--note", parentNote+".md")
		g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
	})
}

// TestActivate_RecheckUnderTheWriteLock (review H5): a copy that lands
// between the fetch and the write lock is seen by the re-check — no second
// copy.
func TestActivate_RecheckUnderTheWriteLock(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.raced", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	var planted sync.Once

	env.wrap = func(deps *cli.Deps) {
		fetch := deps.Fetch
		deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
			resp, err := fetch(ctx, method, target, body)
			if strings.Contains(target, "/show?") {
				planted.Do(func() {
					env.plant("5.2026-09-28.winner.md", []byte(localNoteWithLinks(parentVaultID,
						"    - note: "+parentNote+"\n      via: pulled\n      hash: "+hash+"\n")))
				})
			}

			return resp, err
		}
	}

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.noteFiles()).To(Equal([]string{"5.2026-09-28.winner.md"}))
	g.Expect(env.exitCodes()).To(BeEmpty())
}

// TestActivate_UnrecordableBackoffIsReported (ruling S18): when the
// failed contact cannot be recorded, the warning names that error instead
// of a zero retry time.
func TestActivate_UnrecordableBackoffIsReported(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.stampVault("")
	env.parent.setDown(true)
	env.wrap = func(deps *cli.Deps) { deps.FS = parentCacheWriteFailsFS{EdgeFS: deps.FS} }

	_, stderr := env.run("activate", "--note", "7.2026-09-01.gone.md")
	g.Expect(stderr).To(ContainSubstring("the backoff could not be recorded"))
	g.Expect(stderr).NotTo(ContainSubstring("retry after 0001"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
}

// TestActivate_UnusableExchangeStatePauses: exchange state that cannot be
// read or stamped pauses the pull-down with a warning; nothing is fetched
// or written.
func TestActivate_UnusableExchangeStatePauses(t *testing.T) {
	t.Parallel()

	cases := map[string]func(env *wiringEnv){
		"unreadable parent cache": func(env *wiringEnv) {
			env.stampVault("")
			env.writeState("parent.json", "{not json")
		},
		"vault ID cannot be minted": func(env *wiringEnv) {
			env.randFails = true
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.unusable", "c", ""))
			setup(env)

			_, stderr := env.run("activate", "--note", parentNote+".md")
			g.Expect(stderr).To(ContainSubstring("warning: activate: "))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.parent.requests()).To(BeEmpty())
			g.Expect(env.noteFiles()).To(BeEmpty())
		})
	}
}

// TestAmend_DiscardDeclineRecording (design D8, H5): a decline that
// cannot be recorded fails the discard and keeps the note (so it cannot
// come back unrecorded); a repeated decline is recorded once; discarding
// a note that is not pulled records nothing.
func TestAmend_DiscardDeclineRecording(t *testing.T) {
	t.Parallel()

	pulledLinks := "    - note: 7.2026-09-01.p\n      via: pulled\n      hash: xh1:p\n"

	t.Run("unrecordable decline keeps the note", func(t *testing.T) {
		t.Parallel()

		for name, corrupt := range map[string]func(env *wiringEnv){
			"corrupt declined.json": func(env *wiringEnv) { env.writeState("declined.json", "{not json") },
			"corrupt vault ID":      func(env *wiringEnv) { env.plant(".engram-vault-id", []byte("garbage\n")) },
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				env := newWiringEnv(t)
				env.stampVault("")
				corrupt(env)
				env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(parentVaultID, pulledLinks)))

				_, stderr := env.run("amend", "--target", "3.2026-09-20.local", "--discard")
				g.Expect(stderr).To(ContainSubstring("amend: discard:"))
				g.Expect(env.exitCodes()).To(Equal([]int{1}))
				g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
			})
		}
	})

	t.Run("a repeated decline is recorded once", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(parentVaultID, pulledLinks)))
		env.plant("4.2026-09-20.again.md", []byte(localNoteWithLinks(parentVaultID, pulledLinks)))

		env.run("amend", "--target", "3.2026-09-20.local", "--discard")
		env.run("amend", "--target", "4.2026-09-20.again", "--discard")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.declined()).To(Equal([]declinedEntry{{Vault: parentVaultID, Basename: "7.2026-09-01.p", Hash: "xh1:p"}}))
	})

	t.Run("an offered note is not declined", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(parentVaultID,
			"    - note: 7.2026-09-01.p\n      via: offered\n      hash: xh1:p\n")))

		env.run("amend", "--target", "3.2026-09-20.local", "--discard")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(BeEmpty())
		g.Expect(filepath.Join(env.vault, ".engram", "declined.json")).NotTo(BeAnExistingFile())
	})
}

// TestAmend_DiscardOfPulledNoteStampsNothingUnconfigured (ruling S18, spec
// vault-local-first): with no parent configured and no vault ID, a bare
// discard of a pulled note deletes it and records no decline — no vault ID
// and no .engram/ are created.
func TestAmend_DiscardOfPulledNoteStampsNothingUnconfigured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parentURL = ""
	env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(parentVaultID,
		"    - note: 7.2026-09-01.p\n      via: pulled\n      hash: xh1:p\n")))

	_, stderr := env.run("amend", "--target", "3.2026-09-20.local", "--discard")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)
	g.Expect(env.noteFiles()).To(BeEmpty())
	g.Expect(filepath.Join(env.vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())
}

// TestServeActivate_StatusByOutcome (ruling S18): a served activate
// answers 5xx only for a server-side failure — none found is 404 and a
// partial result 200, each with the per-ref result.
func TestServeActivate_StatusByOutcome(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		notes     []string
		corrupt   bool
		status    int
		activated []string
		failed    []string
	}{
		"none found": {notes: []string{"9.2026-01-01.gone.md"}, status: 404, failed: []string{"9.2026-01-01.gone.md"}},
		"partial": {
			notes: []string{"1.2026-01-01.a-note.md", "9.2026-01-01.gone.md"}, status: 200,
			activated: []string{"1.2026-01-01.a-note.md"}, failed: []string{"9.2026-01-01.gone.md"},
		},
		"unreadable sidecar": {notes: []string{"1.2026-01-01.a-note.md"}, corrupt: true, status: 500},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			notePath := writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

			sidecar := embed.MarshalSidecar(embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion})
			if testCase.corrupt {
				sidecar = []byte("{not json")
			}

			g.Expect(os.WriteFile(embed.SidecarPath(notePath), sidecar, 0o600)).To(Succeed())

			routes := cli.ServeRoutes(newTestDeps(&strings.Builder{}, &strings.Builder{}), vault, "personal", t.TempDir())

			body, marshalErr := json.Marshal(map[string][]string{"notes": testCase.notes})
			g.Expect(marshalErr).NotTo(HaveOccurred())

			resp := routeFor(t, routes, "/activate").Serve(t.Context(), cli.ServeRequest{Body: body})
			g.Expect(resp.Status).To(Equal(testCase.status))

			if testCase.failed == nil {
				return
			}

			var result struct {
				Activated []string `json:"activated"`
				Failed    []struct {
					Ref string `json:"ref"`
				} `json:"failed"`
			}

			g.Expect(json.Unmarshal(resp.Body, &result)).To(Succeed())
			g.Expect(result.Activated).To(Equal(testCase.activated))

			failedRefs := make([]string, 0, len(result.Failed))
			for _, failure := range result.Failed {
				failedRefs = append(failedRefs, failure.Ref)
			}

			g.Expect(failedRefs).To(Equal(testCase.failed))
		})
	}
}

// unexported constants.
const (
	foldTargetFile = "2.2026-09-20.fold-target.md"
	// parentOnlyFields are the parent-side fields a pull-down strips.
	parentOnlyFields = "skill_key: claude:x\nsources:\n    - chunk.jsonl#1\nvocab_version: v3\n" +
		"tags:\n    - vocab/parent-term\nsupersedes:\n    - note: 6.2026-08-01.old\n      type: updates\n" +
		"      claim: old claim\nxid: " + xidC + "\nparent:\n  vault: " + xidD + "\n  links:\n" +
		"    - note: 1.2026-01-01.grand\n      via: offered\n      hash: xh1:grand\n" +
		"aliases:\n    - 4.2026-08-01.older-name\noffer:\n  origin: " + xidD + ":" + xidB + "\n"
)

// unexported variables.
var (
	errParentCacheWrite = errors.New("parent cache write refused")
)

// declinedEntry is one .engram/declined.json entry.
type declinedEntry struct {
	Vault    string `json:"vault"`
	Basename string `json:"basename"`
	Hash     string `json:"hash"`
}

// fakeParentNote is one note the fake parent serves.
type fakeParentNote struct {
	basename string
	content  string
}

// lockProbe records fetches made while a vault lock is held.
type lockProbe struct {
	mu      sync.Mutex
	held    int
	under   []string
	fetches []string
}

func (p *lockProbe) fetched() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.fetches)
}

func (p *lockProbe) fetchesUnderLock() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.under)
}

func (p *lockProbe) wrap(deps *cli.Deps) {
	deps.Lock = probedLocker{inner: deps.Lock, probe: p}
	fetch := deps.Fetch
	deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
		p.mu.Lock()
		p.fetches = append(p.fetches, target)

		if p.held > 0 {
			p.under = append(p.under, target)
		}
		p.mu.Unlock()

		return fetch(ctx, method, target, body)
	}
}

// parentCacheWriteFailsFS fails every write of .engram/parent.json.
type parentCacheWriteFailsFS struct {
	cli.EdgeFS
}

func (f parentCacheWriteFailsFS) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if filepath.Base(path) == "parent.json" {
		return errParentCacheWrite
	}

	return f.EdgeFS.WriteFileAtomic(path, data, perm)
}

// probedLocker counts held locks for lockProbe.
type probedLocker struct {
	inner cli.FileLocker
	probe *lockProbe
}

func (l probedLocker) Lock(path string) (func() error, error) {
	unlock, err := l.inner.Lock(path)
	if err != nil {
		return nil, err
	}

	l.probe.mu.Lock()
	l.probe.held++
	l.probe.mu.Unlock()

	return func() error {
		l.probe.mu.Lock()
		l.probe.held--
		l.probe.mu.Unlock()

		return unlock()
	}, nil
}

// pulledAuthor is a pulled copy's parent.author.
type pulledAuthor struct {
	Repo  string `yaml:"repo"`
	User  string `yaml:"user"`
	Vault string `yaml:"vault"`
}

// pulledCopy is the frontmatter of a pulled local copy.
type pulledCopy struct {
	Luhmann      string   `yaml:"luhmann"`
	User         string   `yaml:"user"`
	Vault        string   `yaml:"vault"`
	Pending      bool     `yaml:"pending"`
	XID          string   `yaml:"xid"`
	Aliases      []string `yaml:"aliases"`
	Sources      []string `yaml:"sources"`
	Tags         []string `yaml:"tags"`
	VocabVersion string   `yaml:"vocab_version"`
	Supersedes   []any    `yaml:"supersedes"`
	SkillHash    string   `yaml:"skill_hash"`
	SkillKey     string   `yaml:"skill_key"`
	SkillSource  string   `yaml:"skill_source"`
	Offer        map[string]any
	Parent       struct {
		Vault  string       `yaml:"vault"`
		Links  []pulledLink `yaml:"links"`
		Author pulledAuthor `yaml:"author"`
	} `yaml:"parent"`
}

// pulledLink is one parent link.
type pulledLink struct {
	Note string `yaml:"note"`
	Via  string `yaml:"via"`
	Hash string `yaml:"hash"`
}

func (p *recordingParent) activateResponse() (cli.FetchResponse, error) {
	if p.activateDown {
		return cli.FetchResponse{Status: 500, Body: []byte(`{"error":"boom"}`)}, nil
	}

	return cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok"}`)}, nil
}

// activated lists the notes the parent was asked to activate.
func (p *recordingParent) activated() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	notes := make([]string, 0, len(p.log))

	for _, request := range p.log {
		if !strings.HasSuffix(request.url, "/activate") {
			continue
		}

		var body struct {
			Notes []string `json:"notes"`
		}

		_ = json.Unmarshal(request.body, &body)
		notes = append(notes, body.Notes...)
	}

	return notes
}

func (p *recordingParent) addNote(note fakeParentNote) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.notes = append(p.notes, note)

	return note.basename
}

func (p *recordingParent) hashOf(t *testing.T, basename string) string {
	t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, note := range p.notes {
		if note.basename == basename {
			return mustHash(t, []byte(note.content))
		}
	}

	t.Fatalf("no parent note %s", basename)

	return ""
}

// rename renames a parent note, recording the old basename in its aliases
// (design D4, H2); its exchange hash is unchanged.
func (p *recordingParent) rename(from, renamed string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for index, note := range p.notes {
		if note.basename == from {
			p.notes[index] = fakeParentNote{
				basename: renamed,
				content:  strings.Replace(note.content, "\n---\n", "\naliases:\n    - "+from+"\n---\n", 1),
			}
		}
	}
}

// replace swaps in a new version of a parent note with the same basename.
func (p *recordingParent) replace(note fakeParentNote) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for index, existing := range p.notes {
		if existing.basename == note.basename {
			p.notes[index] = note
		}
	}
}

// showResponse answers GET /show (the caller holds mu): the raw envelope,
// resolving the ref by basename then aliases, or — per showMode — an old
// parent's rendered text or an envelope without vault_id.
func (p *recordingParent) showResponse(target string) cli.FetchResponse {
	parsed, _ := url.Parse(target)
	ref := strings.TrimSuffix(parsed.Query().Get("note"), ".md")

	for _, note := range p.notes {
		aliases := regexp.MustCompile(`(?m)^    - (\S+)$`).FindAllStringSubmatch(note.content, -1)
		matches := note.basename == ref ||
			slices.ContainsFunc(aliases, func(alias []string) bool { return alias[1] == ref })

		if !matches {
			continue
		}

		switch p.showMode {
		case "rendered":
			return cli.FetchResponse{Status: 200, Body: []byte(note.content)}
		case "no-vault-id":
			return cli.FetchResponse{Status: 200, Body: []byte(`{"basename":"` + note.basename + `"}`)}
		}

		basename := note.basename
		if p.showMode == "bad-basename" {
			basename += "|claim"
		}

		hash, _ := cli.ExportExchangeHash([]byte(note.content))
		if p.showMode == "wrong-hash" {
			hash = "xh1:" + strings.Repeat("0", 64)
		}

		body, marshalErr := json.Marshal(map[string]string{
			"vault_id": p.reportedVaultID(), "basename": basename,
			"content": note.content, "exchange_hash": hash,
		})
		if marshalErr != nil {
			return cli.FetchResponse{Status: 500}
		}

		return cli.FetchResponse{Status: 200, Body: body}
	}

	return cli.FetchResponse{Status: 404, Body: []byte(`{"error":"show: note not found"}`)}
}

// shown lists the refs the parent was asked to show.
func (p *recordingParent) shown() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	refs := make([]string, 0, len(p.log))

	for _, request := range p.log {
		if !strings.Contains(request.url, "/show?") {
			continue
		}

		parsed, _ := url.Parse(request.url)
		refs = append(refs, parsed.Query().Get("note"))
	}

	return refs
}

func (e *wiringEnv) declined() []declinedEntry {
	e.t.Helper()

	raw, err := os.ReadFile(filepath.Join(e.vault, ".engram", "declined.json"))
	if err != nil {
		return nil
	}

	var file struct {
		Entries []declinedEntry `json:"entries"`
	}

	if json.Unmarshal(raw, &file) != nil {
		return nil
	}

	return file.Entries
}

func (e *wiringEnv) exitCodes() []int {
	e.exitsMu.Lock()
	defer e.exitsMu.Unlock()

	codes := slices.Clone(e.exits)
	e.exits = nil

	return codes
}

func (e *wiringEnv) lastUsed(name string) string {
	e.t.Helper()

	raw, err := os.ReadFile(filepath.Join(e.vault, strings.TrimSuffix(name, ".md")+".vec.json"))
	if err != nil {
		return ""
	}

	sidecar, parseErr := embed.UnmarshalSidecar(raw)
	if parseErr != nil {
		return ""
	}

	return sidecar.LastUsed
}

// linkedCopies lists the vault's notes with any link to parentNote under
// the parent's vault ID.
func (e *wiringEnv) linkedCopies(parentNote string) []string {
	e.t.Helper()

	linked := make([]string, 0, 1)

	for _, name := range e.noteFiles() {
		copied := decodePulledCopy(e.t, []byte(readFileString(e.t, filepath.Join(e.vault, name))))
		if copied.Parent.Vault != parentVaultID {
			continue
		}

		if slices.ContainsFunc(copied.Parent.Links, func(link pulledLink) bool { return link.Note == parentNote }) {
			linked = append(linked, name)
		}
	}

	return linked
}

// noteFiles lists the vault's note files (sorted).
func (e *wiringEnv) noteFiles() []string {
	e.t.Helper()

	entries, err := os.ReadDir(e.vault)
	if err != nil {
		e.t.Fatal(err)
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}

	return names
}

func (e *wiringEnv) parentCache() struct {
	VaultID  string `json:"vault_id"` //nolint:tagliatelle // parent.json's keys
	Failures int    `json:"failures"`
} {
	var cache struct {
		VaultID  string `json:"vault_id"` //nolint:tagliatelle // parent.json's keys
		Failures int    `json:"failures"`
	}

	raw, err := os.ReadFile(filepath.Join(e.vault, ".engram", "parent.json"))
	if err == nil {
		_ = json.Unmarshal(raw, &cache)
	}

	return cache
}

func (e *wiringEnv) plantSidecar(name, lastUsed string) {
	e.t.Helper()

	sidecar := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@4", Dims: 4,
		SituationVector: []float32{1, 0, 0, 0}, BodyVector: []float32{1, 0, 0, 0},
		ContentHash: "sha256:x", LastUsed: lastUsed,
	}

	e.plant(strings.TrimSuffix(name, ".md")+".vec.json", embed.MarshalSidecar(sidecar))
}

func (e *wiringEnv) recordExit(code int) {
	e.exitsMu.Lock()
	defer e.exitsMu.Unlock()

	e.exits = append(e.exits, code)
}

// stampVault gives the vault an ID, its state directory and a passing
// location record, as a first parent contact would; path overrides the
// recorded path when non-empty.
func (e *wiringEnv) stampVault(path string) {
	e.t.Helper()

	canonical, err := filepath.EvalSymlinks(e.vault)
	if err != nil {
		e.t.Fatal(err)
	}

	if path != "" {
		canonical = path
	}

	id := seqID(50)
	mkErr := os.MkdirAll(filepath.Join(e.vault, ".engram"), 0o700)
	if mkErr != nil {
		e.t.Fatal(mkErr)
	}

	e.plant(".engram-vault-id", []byte(id+"\n"))
	e.writeState(".gitignore", "*\n")
	e.writeState("home.json", `{"vault_id":"`+id+`","path":`+strconvQuote(canonical)+`}`)
}

func (e *wiringEnv) writeState(name, content string) {
	e.t.Helper()

	err := os.WriteFile(filepath.Join(e.vault, ".engram", name), []byte(content), 0o600)
	if err != nil {
		e.t.Fatal(err)
	}
}

// applyPullStep runs one step of the rapid sequence.
func applyPullStep(rt *rapid.T, env *wiringEnv, parentNote, step string) {
	if step == "activate" {
		env.run("activate", "--note", parentNote+".md")

		return
	}

	copies := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == foldTargetFile })
	if len(copies) == 0 {
		return
	}

	target := rapid.SampledFrom(copies).Draw(rt, "target")

	switch step {
	case "clear-pending":
		env.run("amend", "--target", strings.TrimSuffix(target, ".md"), "--clear-pending")
	case "discard":
		env.run("amend", "--target", strings.TrimSuffix(target, ".md"), "--discard")
	case "fold":
		_, stderr := env.run("amend", "--target", strings.TrimSuffix(target, ".md"), "--discard",
			"--into", strings.TrimSuffix(foldTargetFile, ".md"), "--expect-hash", env.exchangeHashOf(target))
		if _, stillThere := os.Stat(filepath.Join(env.vault, target)); stillThere == nil {
			rt.Fatalf("fold left %s in place: %s", target, stderr)
		}

		if !slices.Contains(env.linkedCopies(parentNote), foldTargetFile) {
			rt.Fatalf("after folding %s, %s does not link %s", target, foldTargetFile, parentNote)
		}
	}
}

func decodePulledCopy(t failer, raw []byte) pulledCopy {
	t.Helper()

	var copied pulledCopy

	frontmatter, _, found := embed.SplitFrontmatter(raw)
	if !found {
		return copied
	}

	err := yaml.Unmarshal(frontmatter, &copied)
	if err != nil {
		t.Fatal(err)
	}

	return copied
}

// localNoteWithLinks is a live local fact whose parent block holds links
// (rendered YAML list items) under vault.
func localNoteWithLinks(vault, links string) string {
	return "---\ntype: fact\ntier: L2\nsituation: when local\nsubject: l\npredicate: m\nobject: n\n" +
		"luhmann: \"3\"\ncreated: \"2026-09-20\"\nsource: test\nuser: alice\nvault: personal\n" +
		"xid: " + xidA + "\nparent:\n  vault: " + vault + "\n  links:\n" + links +
		"---\n\nInformation learned: when local, l m n.\n"
}

// pulledFact is a parent fact note with parent-side identity; extra
// frontmatter lines are appended.
func pulledFact(basename, object, extra string) fakeParentNote {
	return fakeParentNote{
		basename: basename,
		content: "---\ntype: fact\ntier: L2\nsituation: when pulling\nsubject: a\npredicate: b\nobject: " + object +
			"\nluhmann: \"7\"\ncreated: \"2026-09-01\"\nsource: parent-src\nrepo: github.com/parent/repo\n" +
			"user: bob\nvault: team\n" + extra +
			"---\n\nInformation learned: when pulling, a b " + object + ".\n\nA parent-side paragraph.\n",
	}
}

// pulledRunbook is a registered-skill runbook on the parent.
func pulledRunbook(basename string) fakeParentNote {
	return fakeParentNote{
		basename: basename,
		content: "---\ntype: runbook\ntier: L2\nsituation: when stepping\ndone_when: done\n" +
			"red_flags:\n    - skipping\ntriggers:\n    - step it\nluhmann: \"8\"\ncreated: \"2026-09-02\"\n" +
			"source: parent-src\nrepo: github.com/parent/repo\nuser: bob\nvault: team\n" +
			"skill_hash: abc123\nskill_key: claude:steps\nskill_source: ~/.claude/skills/steps/SKILL.md\n" +
			"tags:\n    - vocab/parent-term\nxid: " + xidC + "\n---\n\n1. Step one.\n2. Step two.\n",
	}
}

func strconvQuote(text string) string {
	return strconv.Quote(text)
}
