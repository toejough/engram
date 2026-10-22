package cli_test

// Scenario tests for the parent-sync product capability, part 2 (861c2zy): a vault with no parent (792a21m), the
// local write that is offered (792a21n), what counts as a change (792a21o), the offer built for the parent
// (792a21p) and the offer queued and never lost offline (792a21q). Spec: docs/specs/product/parent-sync.md.
//
// The CLI journeys run over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860) and a
// fake parent behind Deps.Fetch, with no network and no real home. The env's git identity is a scripted commander
// (ps2Identity), so no journey reads the machine's git configuration. The one interaction scenario (792a21p1a)
// drives Deps.Fetch through MockPvc1Fetch.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/cli"
)

// Test792a21m1a_Targets_Query_NoParentMakesNoRequestAndMatchesTheSingleSourcePipeline verifies 792a21m1a (No parent
// configured): with ENGRAM_PARENT unset, `engram query --limit 2` over three local notes makes no request to any
// parent, prints the same payload as the single-source query pipeline run on its own (two items, the cap of
// --limit, none tagged from_parent), and writes no exchange state.
func Test792a21m1a_Targets_Query_NoParentMakesNoRequestAndMatchesTheSingleSourcePipeline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		limit     = 2
		noteCount = 3
	)

	// Given ENGRAM_PARENT is not set, over a vault of three notes
	env := ps2Env(t)
	env.parentURL = ""

	vectors := [noteCount][]float32{{1, 0, 0, 0}, {0.8, 0.6, 0, 0}, {0, 1, 0, 0}}
	for index, vector := range vectors {
		plantRealVaultNote(t, env.vault, strconv.Itoa(index+1)+".fact.md",
			"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody "+strconv.Itoa(index+1)+"\n", vector)
	}

	chunks := t.TempDir()

	// When the user runs engram query
	stdout, stderr := env.run("query", "--phrase", "x", "--limit", strconv.Itoa(limit), "--chunks-dir", chunks)

	// Then no network request is made
	g.Expect(stderr).To(BeEmpty())
	g.Expect(env.parent.requests()).To(BeEmpty())

	// and the output matches the single-source pipeline, including the --limit cap
	var single strings.Builder

	singleErr := cli.RunQuery(t.Context(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: env.vault, ChunksDir: chunks, Limit: limit},
		cli.ExportNewQueryDeps(env.deps()), &single)
	g.Expect(singleErr).NotTo(HaveOccurred())
	g.Expect(stdout).To(Equal(single.String()))

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(answer.Items).To(HaveLen(limit), "the cap of --limit")

	for _, item := range answer.Items {
		g.Expect(item.FromParent).To(BeNil(), "no item is tagged by the merge")
	}

	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())
	g.Expect(filepath.Join(env.vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
}

// Test792a21n1c_Targets_Amend_ARunbookFieldAmendIsOffered verifies 792a21n1c (A runbook-field amend is offered):
// `engram amend --target R --red-flag=...` on a linked runbook changes R's exchange hash, queues an entry (the
// parent is unreachable, so it stays queued), and the drain then sends an amend-offer for R's parent note that
// carries the new red flag.
func Test792a21n1c_Targets_Amend_ARunbookFieldAmendIsOffered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		runbook = "1.2026-09-27.note"
		flag    = "check the rollback first"
	)

	// Given a linked runbook R
	env := ps2Env(t)
	env.plant(runbook+".md", offerTestNote{
		noteType: "runbook", parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:old",
	}.render(t))
	env.parent.setDown(true)

	hashBefore := env.exchangeHashOf(runbook + ".md")

	// When the user runs engram amend --target R --red-flag="..."
	env.run("amend", "--target", runbook, "--red-flag", flag)

	// Then an amend-offer is queued, because R's exchange hash changed
	g.Expect(env.exchangeHashOf(runbook + ".md")).NotTo(Equal(hashBefore))
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	env.parent.setDown(false)
	env.advance(time.Minute)
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].Offer.For).To(Equal("5.2026-09-27.p"), "an amend-offer, naming R's parent note")
	g.Expect(offers[0].RedFlags).To(Equal([]string{flag}))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21n1d_Targets_Amend_AContentAmendOfAnUnlinkedNoteIsALearnOffer verifies 792a21n1d (A content amend of an
// unlinked note is a learn-offer): `engram amend --target L --object=...` on a note with no parent link sends the
// parent one learn-offer (no offer.for) carrying L's current content.
func Test792a21n1d_Targets_Amend_AContentAmendOfAnUnlinkedNoteIsALearnOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const note = "1.2026-09-27.note"

	// Given local note L with no parent link
	env := ps2Env(t)
	env.plant(note+".md", offerTestNote{}.render(t))

	// When the user runs engram amend --target L --object="..."
	env.run("amend", "--target", note, "--object", "a changed object")

	// Then the parent receives a learn-offer carrying L's current content
	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].Offer.For).To(BeEmpty(), "a learn-offer names no parent note")
	g.Expect(offers[0].Type).To(Equal("fact"))
	g.Expect(offers[0].Slug).To(Equal("note"))
	g.Expect(offers[0].Situation).To(Equal("when testing"))
	g.Expect(offers[0].Subject).To(Equal("a"))
	g.Expect(offers[0].Predicate).To(Equal("b"))
	g.Expect(offers[0].Object).To(Equal("a changed object"))
}

// Test792a21n1i_BackfillIdentity_IdentityBackfillStaysLocal verifies 792a21n1i (Identity backfill stays local): the
// identity backfill that `engram update --backfill-identity` runs stamps repo/user/vault onto a note that lacks
// them (it did real work) with a parent configured, and queues no offer, sends no request and creates no
// exchange state.
func Test792a21n1i_BackfillIdentity_IdentityBackfillStaysLocal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const note = "1.2026-09-27.note.md"

	// Given notes missing identity fields
	env := ps2Env(t)
	withoutIdentity := strings.Replace(string(offerTestNote{}.render(t)), "user: alice\nvault: personal\n", "", 1)
	env.plant(note, []byte(withoutIdentity))

	// When the user runs engram update --backfill-identity (its backfill step, over the env's production deps)
	stamped, err := cli.ExportBackfillIdentity(t.Context(), env.vault, cli.ExportNewIdentityDeps(env.deps()), false)

	// Then no offer is queued for any of them
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1), "the backfill did stamp the note")
	g.Expect(readFileString(t, filepath.Join(env.vault, note))).To(ContainSubstring("user: " + ps2User))
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(env.parent.requests()).To(BeEmpty())
	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())
}

// Test792a21n1j_Targets_Learn_RegistrationNotesAndQANotesAreNeverOffered verifies 792a21n1j (Registration notes and
// qa notes are never offered): with a parent configured but unreachable (so any offer queued would stay in the
// outbox), writing a note that carries skill_hash and writing an `engram learn qa` note each queue nothing and
// make no request.
func Test792a21n1j_Targets_Learn_RegistrationNotesAndQANotesAreNeverOffered(t *testing.T) {
	t.Parallel()

	writes := map[string]func(env *wiringEnv){
		"a registration note": func(env *wiringEnv) {
			env.runLearn(cli.LearnArgs{SkillHash: "abc", SkillKey: "k", SkillSource: "~/s"})
		},
		"a qa note": func(env *wiringEnv) {
			env.run("learn", "qa", "--slug", "qa", "--question", "why?", "--answer", "because", "--source", "t")
			env.expectOutput(".md")
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note written by registration, or by engram learn qa, with a parent that is down
			env := ps2Env(t)
			env.parent.setDown(true)

			// When it is written
			write(env)

			// Then no offer is queued for it
			g.Expect(env.lastStderr).NotTo(ContainSubstring("error"))
			g.Expect(env.outboxEntries()).To(BeEmpty())
			g.Expect(env.parent.requests()).To(BeEmpty())
		})
	}
}

// Test792a21n1k_Targets_Learn_APendingNoteIsNeverOffered verifies 792a21n1k (A pending note is never offered): with
// a parent configured but unreachable, a note written with the pending marker and a pending note amended with a
// content flag each queue nothing and make no request.
func Test792a21n1k_Targets_Learn_APendingNoteIsNeverOffered(t *testing.T) {
	t.Parallel()

	writes := map[string]func(env *wiringEnv){
		"written": func(env *wiringEnv) {
			env.runLearn(cli.LearnArgs{Pending: true})
		},
		"amended": func(env *wiringEnv) {
			env.plant("1.2026-09-27.note.md", offerTestNote{pending: true}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--object", "changed")
			env.expectOutput("1.2026-09-27.note.md")
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note that carries the pending marker, with a parent that is down
			env := ps2Env(t)
			env.parent.setDown(true)

			// When it is written or amended
			write(env)

			// Then no offer is queued for it
			g.Expect(env.lastStderr).NotTo(ContainSubstring("error"))
			g.Expect(env.outboxEntries()).To(BeEmpty())
			g.Expect(env.parent.requests()).To(BeEmpty())
		})
	}
}

// Test792a21n1l_Targets_Amend_AnUnchangedRoundTripIsSuppressed verifies 792a21n1l (An unchanged round trip is
// suppressed): a note whose exchange hash equals its primary link hash, amended with the value it already has, queues
// no offer, while an amend that changes it does (the control). A first offer that failed to reach the parent keeps
// the backoff window open, so the drain does not run and the outbox shows exactly what was queued.
func Test792a21n1l_Targets_Amend_AnUnchangedRoundTripIsSuppressed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		linked = "7.2026-09-27.linked"
		fresh  = "8.2026-09-27.fresh"
	)

	// Given a note whose exchange hash equals its primary link hash, under a parent whose ID is known
	env := ps2Env(t)
	env.learnFact("seed")
	env.plant(linked+".md", offerTestNote{
		parentVault: parentVaultID, linkVia: "offered", linkMatches: true,
	}.render(t))
	env.plant(fresh+".md", []byte(strings.Replace(string(offerTestNote{}.render(t)), xidA, xidB, 1)))

	env.parent.setDown(true)
	env.run("amend", "--target", fresh, "--object", "changed")
	g.Expect(env.outboxEntries()).To(HaveLen(1), "the changed note is queued, and its failed send opens the backoff")

	// When an amend leaves it so
	env.run("amend", "--target", linked, "--object", "c")

	// Then no offer is queued
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	// (control) an amend that changes the content is queued
	env.run("amend", "--target", linked, "--object", "d")
	g.Expect(env.outboxEntries()).To(HaveLen(2))
}

// Test792a21o1a_ExchangeHash_OnlyOfferedFieldsMoveTheHash verifies 792a21o1a (Only offered fields move the hash):
// changing only a feedback note's impact, or only a runbook note's done_when, red_flags or triggers, changes the
// note's exchange hash.
func Test792a21o1a_ExchangeHash_OnlyOfferedFieldsMoveTheHash(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ original, from, to string }{
		"a feedback note's impact": {
			original: ps2Feedback, from: "impact: the release broke\n", to: "impact: the release was late\n",
		},
		"a runbook note's done_when": {
			original: ps2Runbook, from: "done_when: all steps ran\n", to: "done_when: the checks passed\n",
		},
		"a runbook note's red_flags": {
			original: ps2Runbook, from: "red_flags:\n    - skipping a step\n",
			to: "red_flags:\n    - skipping a step\n    - rushing\n",
		},
		"a runbook note's triggers": {
			original: ps2Runbook, from: "triggers:\n    - ship it\n", to: "triggers:\n    - release now\n",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a feedback or runbook note, and the same note with only that field changed
			changed := strings.Replace(testCase.original, testCase.from, testCase.to, 1)
			g.Expect(changed).NotTo(Equal(testCase.original), "the fixture changes the field")

			// When the two are hashed
			before, beforeErr := cli.ExportExchangeHash([]byte(testCase.original))
			after, afterErr := cli.ExportExchangeHash([]byte(changed))

			// Then the note's exchange hash changes
			g.Expect(beforeErr).NotTo(HaveOccurred())
			g.Expect(afterErr).NotTo(HaveOccurred())
			g.Expect(after).NotTo(Equal(before))
		})
	}
}

// Test792a21o1b_ExchangeHash_NonContentFieldsDoNotMoveTheHash verifies 792a21o1b (Non-content fields do not move the
// hash): changing only a note's repo, user, pending marker, tags or parent link leaves its exchange hash unchanged.
func Test792a21o1b_ExchangeHash_NonContentFieldsDoNotMoveTheHash(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ from, to string }{
		"repo":    {from: "repo: github.com/a/b\n", to: "repo: github.com/c/d\n"},
		"user":    {from: "user: alice\n", to: "user: bob\n"},
		"pending": {from: "vault: personal\n", to: "vault: personal\npending: true\n"},
		"tags":    {from: "tags:\n    - vocab/one\n", to: "tags:\n    - vocab/one\n    - vocab/two\n"},
		"parent link": {
			from: "vault: personal\n",
			to: "vault: personal\nparent:\n    vault: " + seqID(90) +
				"\n    links:\n        - note: 5.2026-09-27.p\n          via: offered\n          hash: xh1:x\n",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note, and the same note with only that field changed
			changed := strings.Replace(ps2Fact, testCase.from, testCase.to, 1)
			g.Expect(changed).NotTo(Equal(ps2Fact), "the fixture changes the field")

			// When the two are hashed
			before, beforeErr := cli.ExportExchangeHash([]byte(ps2Fact))
			after, afterErr := cli.ExportExchangeHash([]byte(changed))

			// Then its exchange hash is unchanged
			g.Expect(beforeErr).NotTo(HaveOccurred())
			g.Expect(afterErr).NotTo(HaveOccurred())
			g.Expect(after).To(Equal(before))
		})
	}
}

// Test792a21o1c_Activate_AHashVersionChangeIsUnknownNotChanged verifies 792a21o1c (A hash version change is
// unknown, not changed): activating the parent note again, when a local link records an older-version hash for it,
// writes nothing (the note on disk is byte-identical and no copy appears); and a later exchange of that link (an
// amend that is offered and acknowledged) records the receipt's current-version hash in place of the old one.
func Test792a21o1c_Activate_AHashVersionChangeIsUnknownNotChanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		local      = "3.2026-09-20.local"
		legacyHash = "xh0:legacy"
	)

	currentHash := cli.ExportExchangeHashPrefix + strings.Repeat("ab", ps2HashBytes)

	// Given a link that records a hash with an older version prefix
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.versioned", "c", ""))
	env.plant(local+".md", []byte(localNoteWithLinks(parentVaultID,
		"    - note: "+parentNote+"\n      via: pulled\n      hash: "+legacyHash+"\n")))

	before := readFileString(t, filepath.Join(env.vault, local+".md"))

	// When the parent note is activated again
	env.run("activate", "--note", parentNote+".md")

	// Then the pull-down treats it as not changed and writes nothing
	g.Expect(env.lastStderr).To(BeEmpty())
	g.Expect(env.noteFiles()).To(Equal([]string{local + ".md"}))
	g.Expect(readFileString(t, filepath.Join(env.vault, local+".md"))).To(Equal(before))

	// and a later exchange of that link records the current-version hash
	env.parent.setStoredHash(currentHash)
	env.run("amend", "--target", local, "--object", "changed upstream")

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].Offer.For).To(Equal(parentNote), "the link still names the parent note")

	after := readFileString(t, filepath.Join(env.vault, local+".md"))
	g.Expect(after).To(ContainSubstring("hash: " + currentHash))
	g.Expect(after).NotTo(ContainSubstring(legacyHash))
}

// Test792a21p1a_Targets_Learn_PlacementIsNotSent verifies 792a21p1a (Placement is not sent): a note learned with
// --target 12 --position continuation is placed under 12 locally, and the offer request that reaches the parent (the
// body of the one POST /learn the mock sees) carries no target and no position.
func Test792a21p1a_Targets_Learn_PlacementIsNotSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note learned with --target 12 --position continuation, with Deps.Fetch an imptest mock
	env := ps2Env(t)
	env.plant("12.2026-09-27.target.md", offerTestNote{}.render(t))

	fetch, fetchCalls := MockPvc1Fetch(t)
	identity := env.wrap
	env.wrap = func(deps *cli.Deps) {
		identity(deps)

		deps.Fetch = fetch
	}

	offerBody := make(chan []byte, 1)
	script := ps2Script(env, fetchCalls, offerBody)

	// When it is offered
	stdout, _ := pvc1RunWithin(t, env, script, "learn", "fact", "--slug", "placed", "--situation", "s",
		"--subject", "a", "--predicate", "b", "--object", "c", "--source", "t",
		"--target", "12", "--position", "continuation")

	// Then the offer request carries no target and no position
	g.Expect(filepath.Base(strings.TrimSpace(stdout))).To(HavePrefix("12a."), "the note was placed under 12")

	select {
	case body := <-offerBody:
		var wire map[string]any

		g.Expect(json.Unmarshal(body, &wire)).To(Succeed())
		g.Expect(wire).To(HaveKeyWithValue("slug", "placed"))
		g.Expect(wire).NotTo(HaveKey("target"))
		g.Expect(wire).NotTo(HaveKey("position"))
	case <-time.After(mockTimeout):
		t.Fatal("no offer reached the parent")
	}
}

// Test792a21p1b_Targets_Amend_SupersedesIsTranslatedOrDropped verifies 792a21p1b (Supersedes is translated or
// dropped): an offered note that supersedes local note A (linked to parent note PA) and local note B (unlinked) is
// offered with a supersedes entry naming PA, and neither A nor B is named anywhere in the request.
func Test792a21p1b_Targets_Amend_SupersedesIsTranslatedOrDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		noteA = "2.2026-09-27.a"
		noteB = "3.2026-09-27.b"
		noteN = "1.2026-09-27.note"
	)

	// Given an offered note that supersedes local note A, linked to PA, and local note B, which is unlinked
	env := ps2Env(t)
	env.plant(noteA+".md", []byte(strings.Replace(string(offerTestNote{
		parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:x",
	}.render(t)), xidA, xidC, 1)))
	env.plant(noteB+".md", []byte(strings.Replace(string(offerTestNote{}.render(t)), xidA, xidD, 1)))
	env.plant(noteN+".md", []byte(strings.Replace(string(offerTestNote{}.render(t)), "xid: ",
		"supersedes:\n  - note: "+noteA+"\n    type: updates\n    claim: claim a\n"+
			"  - note: "+noteB+"\n    type: refutes\n    claim: claim b\nxid: ", 1)))

	// When it is offered
	env.run("amend", "--target", noteN, "--object", "changed")

	// Then the offer's supersedes names PA and does not name A or B
	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].Supersedes).To(Equal([]string{"5.2026-09-27.p|updates|claim a"}))

	body := ps2OfferBody(t, env)
	g.Expect(body).NotTo(ContainSubstring(noteA))
	g.Expect(body).NotTo(ContainSubstring(noteB))
}

// Test792a21p1c_Targets_Query_TheNotesAuthorIsDeclaredNotTheDrainers verifies 792a21p1c (The note's author is
// declared, not the drainer's): a note written by alice (its user: is alice's) and queued while the parent was
// down is drained by a command run as bob; the offer declares alice, not bob.
func Test792a21p1c_Targets_Query_TheNotesAuthorIsDeclaredNotTheDrainers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		alice = "alice@example.com"
		bob   = "bob@example.com"
	)

	// Given a note whose user: is alice, queued by alice's command
	env := ps2Env(t)
	env.wrap = ps2Identity(alice)
	env.parent.setDown(true)

	notePath := env.learnFact("authored")
	g.Expect(readFileString(t, notePath)).To(ContainSubstring("user: " + alice))

	// drained by a command run as bob
	env.wrap = ps2Identity(bob)
	env.parent.setDown(false)
	env.advance(time.Minute)

	// When it is offered
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then the offer declares user alice
	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].User).To(Equal(alice))
	g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring(bob))
}

// Test792a21p1d_Targets_Learn_TheOfferNamesItsOriginKeyAndPath verifies 792a21p1d (The offer names its origin, key
// and path): a direct offer's offer.path is this vault's ID, and an offer propagated from an accepted served offer
// has the accepted note's offer.path with this vault's ID appended; each carries offer.origin (this vault's ID and
// the note's xid) and offer.key (sha256 of the origin and the note's exchange hash).
func Test792a21p1d_Targets_Learn_TheOfferNamesItsOriginKeyAndPath(t *testing.T) {
	t.Parallel()

	t.Run("a direct offer", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a direct offer
		env := ps2Env(t)

		// When it is built
		notePath := env.learnFact("direct")

		// Then its path is this vault's ID, with an origin and a key
		offers := env.parent.offers()
		g.Expect(offers).To(HaveLen(1))

		if len(offers) != 1 {
			return
		}

		xid := regexp.MustCompile(`(?m)^xid: ([0-9a-f]{32})$`).FindStringSubmatch(readFileString(t, notePath))
		g.Expect(xid).To(HaveLen(2))

		if len(xid) != 2 {
			return
		}

		origin := env.localID() + ":" + xid[1]
		g.Expect(offers[0].Offer.Path).To(Equal([]string{env.localID()}))
		g.Expect(offers[0].Offer.Origin).To(Equal(origin))
		g.Expect(offers[0].Offer.Key).To(Equal(ps2OfferKey(origin, env.exchangeHashOf(filepath.Base(notePath)))))
	})

	t.Run("an offer propagated from an accepted served offer", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const served = "1.2026-09-27.note.md"

		// Given an accepted served offer, which carries the offer.path it travelled
		child := seqID(40)
		env := ps2Env(t)
		env.plant(served, offerTestNote{pending: true, origin: child + ":" + xidB, path: []string{child}}.render(t))

		// When it is built
		env.run("amend", "--target", strings.TrimSuffix(served, ".md"), "--clear-pending",
			"--expect-hash", env.exchangeHashOf(served))

		// Then its path is the accepted note's path with this vault's ID appended, with an origin and a key
		offers := env.parent.offers()
		g.Expect(offers).To(HaveLen(1))

		if len(offers) != 1 {
			return
		}

		origin := env.localID() + ":" + xidA
		g.Expect(offers[0].Offer.Path).To(Equal([]string{child, env.localID()}))
		g.Expect(offers[0].Offer.Origin).To(Equal(origin))
		g.Expect(offers[0].Offer.Key).To(Equal(ps2OfferKey(origin, env.exchangeHashOf(served))))
	})
}

// Test792a21p1e_Targets_Amend_AnAmendOfferNamesItsTarget verifies 792a21p1e (An amend-offer names its target): an
// amend of a note linked to parent note P under the configured parent carries offer.for P, and an amend of a note
// whose only link is covered, or is under another parent, is offered as a learn-offer (no offer.for).
func Test792a21p1e_Targets_Amend_AnAmendOfferNamesItsTarget(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		note    offerTestNote
		wantFor string
	}{
		"linked under the configured parent": {
			note:    offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:x"},
			wantFor: "5.2026-09-27.p",
		},
		"only a covered link": {
			note: offerTestNote{parentVault: parentVaultID, linkVia: "covered", linkHash: "xh1:x"},
		},
		"linked under another parent": {
			note: offerTestNote{parentVault: seqID(77), linkVia: "offered", linkHash: "xh1:x"},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note with that link
			env := ps2Env(t)
			env.plant("1.2026-09-27.note.md", testCase.note.render(t))

			// When its amend is offered
			env.run("amend", "--target", "1.2026-09-27.note", "--object", "changed")

			// Then the offer carries offer.for P, or none (a learn-offer)
			offers := env.parent.offers()
			g.Expect(offers).To(HaveLen(1))

			if len(offers) != 1 {
				return
			}

			g.Expect(offers[0].Offer.For).To(Equal(testCase.wantFor))
		})
	}
}

// Test792a21p1f_Targets_Amend_TagsAreNotOffered verifies 792a21p1f (Tags are not offered): an amended note that
// carries vocabulary tags is offered with no tags key in the request and no tag text anywhere in it.
func Test792a21p1f_Targets_Amend_TagsAreNotOffered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const note = "1.2026-09-27.note"

	// Given a note that carries vocabulary tags
	env := ps2Env(t)
	env.plant(note+".md", []byte(strings.Replace(string(offerTestNote{}.render(t)), "vault: personal\n",
		"vault: personal\ntags:\n  - vocab/alpha\n  - vocab/beta\n", 1)))

	// When it is offered
	env.run("amend", "--target", note, "--object", "changed")

	// Then the offer carries no tags
	g.Expect(readFileString(t, filepath.Join(env.vault, note+".md"))).To(ContainSubstring("vocab/alpha"),
		"the note still carries its tags")

	var wire map[string]any

	g.Expect(env.parent.offers()).To(HaveLen(1))
	g.Expect(json.Unmarshal([]byte(ps2OfferBody(t, env)), &wire)).To(Succeed())
	g.Expect(wire).NotTo(HaveKey("tags"))
	g.Expect(ps2OfferBody(t, env)).NotTo(ContainSubstring("vocab/"))
}

// Test792a21q1c_Targets_Query_ARenamedNoteKeepsItsEntry verifies 792a21q1c (A renamed note keeps its entry): a note
// queued while the parent was down and then renamed by the Luhmann rename primitive is found by its xid when the
// outbox drains; the drain sends its offer and records the link on the renamed note, and no entry is left.
func Test792a21q1c_Targets_Query_ARenamedNoteKeepsItsEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a queued note renamed by Luhmann reparenting before the outbox drains
	env := ps2Env(t)
	env.parent.setDown(true)

	oldName := filepath.Base(env.learnFact("renamed"))
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	newName := "3a" + strings.TrimPrefix(oldName, strings.SplitN(oldName, ".", 2)[0])
	renamed, renameErr := cli.RenameAndRewriteReferences(ps2RenameDeps(), env.vault,
		map[string]string{strings.TrimSuffix(oldName, ".md"): strings.TrimSuffix(newName, ".md")})
	g.Expect(renameErr).NotTo(HaveOccurred())
	g.Expect(renamed).NotTo(BeNil())
	g.Expect(env.noteFiles()).To(Equal([]string{newName}))

	env.parent.setDown(false)
	env.advance(time.Minute)

	// When the outbox drains
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then the drain sends the renamed note's offer and records the link on the renamed note
	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	g.Expect(offers[0].Slug).To(Equal("renamed"))
	g.Expect(env.noteFiles()).To(Equal([]string{newName}), "the old name is not recreated")

	content := readFileString(t, filepath.Join(env.vault, newName))
	g.Expect(content).To(ContainSubstring("vault: " + parentVaultID))
	g.Expect(content).To(ContainSubstring("note: 1100.2026-09-28.renamed"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21q1d_Targets_Query_ADiscardedNotesEntryIsDropped verifies 792a21q1d (A discarded note's entry is
// dropped): a note queued while the parent was down and then discarded locally sends nothing when the outbox
// drains, and its entry is removed.
func Test792a21q1d_Targets_Query_ADiscardedNotesEntryIsDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a queued note discarded locally before the outbox drains
	env := ps2Env(t)
	env.parent.setDown(true)

	notePath := env.learnFact("discarded")
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	env.run("amend", "--target", noteBasename(notePath), "--discard")
	g.Expect(env.noteFiles()).To(BeEmpty())
	g.Expect(env.outboxEntries()).To(HaveLen(1), "the discard leaves the entry for the drain to drop")

	env.parent.setDown(false)
	env.advance(time.Minute)

	// When the outbox drains
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then it sends nothing for it and removes its entry
	g.Expect(env.parent.offers()).To(BeEmpty())
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21q1f_Targets_Query_APendingNotesEntryIsDropped verifies 792a21q1f (A pending note's entry is dropped): a
// note queued while the parent was down that has since become pending sends nothing when the outbox drains, and
// its entry is removed.
func Test792a21q1f_Targets_Query_APendingNotesEntryIsDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a queued note that has since become pending
	env := ps2Env(t)
	env.parent.setDown(true)

	notePath := env.learnFact("pending-later")
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	queued := readFileString(t, notePath)
	pending := strings.Replace(queued, "\nxid: ", "\npending: true\nxid: ", 1)
	g.Expect(pending).NotTo(Equal(queued))
	env.mustWrite(notePath, pending)

	env.parent.setDown(false)
	env.advance(time.Minute)

	// When the outbox drains
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then it sends nothing for it and removes its entry
	g.Expect(env.parent.offers()).To(BeEmpty())
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21q1g_Targets_Learn_TheFirstOfferCreatesTheExchangeStateInTheRightOrder verifies 792a21q1g (The first
// offer creates the exchange state in the right order): the first offer queued in a vault with no vault ID and no
// .engram directory writes the vault ID before it creates .engram, and the location record then in .engram/home.json
// names that ID and the vault's path.
func Test792a21q1g_Targets_Learn_TheFirstOfferCreatesTheExchangeStateInTheRightOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with no vault ID and no .engram directory
	env := ps2Env(t)
	env.parent.setDown(true)

	var events ps2Events

	identity := env.wrap
	env.wrap = func(deps *cli.Deps) {
		identity(deps)

		deps.FS = ps2OrderFS{EdgeFS: deps.FS, events: &events}
	}

	g.Expect(filepath.Join(env.vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())

	// When the first offer is queued
	env.learnFact("first")
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	// Then the vault ID is stamped before .engram exists
	order := events.list()
	idAt := ps2IndexOf(order, ps2EventVaultID)
	dirAt := ps2IndexOf(order, ps2EventStateDir)

	g.Expect(idAt).To(BeNumerically(">=", 0), "the vault ID was written")
	g.Expect(dirAt).To(BeNumerically(">=", 0), ".engram was created")
	g.Expect(idAt).To(BeNumerically("<", dirAt), "the vault ID came first: %v", order)

	// and its location record names that ID and this vault
	var record map[string]string

	g.Expect(json.Unmarshal([]byte(readFileString(t, filepath.Join(env.vault, ".engram", "home.json"))),
		&record)).To(Succeed())
	g.Expect(record).To(HaveKeyWithValue("vault_id", env.localID()))
	g.Expect(env.localID()).To(MatchRegexp(`^[0-9a-f]{32}$`))

	canonical, evalErr := filepath.EvalSymlinks(env.vault)
	g.Expect(evalErr).NotTo(HaveOccurred())
	g.Expect(record).To(HaveKeyWithValue("path", canonical))
}

// unexported constants.
const (
	ps2EventStateDir = "mkdir .engram"
	ps2EventVaultID  = "write .engram-vault-id"
	// ps2Fact is a fact note with every non-content field the exchange hash ignores.
	ps2Fact = "---\ntype: fact\ntier: L2\nsituation: when releasing\nsubject: a widget\npredicate: needs\n" +
		"object: a checklist\nluhmann: \"1\"\ncreated: \"2026-09-27\"\nsource: test\nrepo: github.com/a/b\n" +
		"user: alice\nvault: personal\ntags:\n    - vocab/one\nxid: " + xidA + "\n---\n\n" +
		"Information learned: when releasing, a widget needs a checklist.\n"
	// ps2Feedback is a feedback note.
	ps2Feedback = "---\ntype: feedback\ntier: L2\nsituation: releasing a widget\nbehavior: skipped the checklist\n" +
		"impact: the release broke\naction: run the checklist\nluhmann: \"1\"\ncreated: \"2026-09-27\"\n" +
		"source: test\nuser: alice\nvault: personal\n---\n\nLesson learned: when releasing, run the checklist.\n"
	// ps2HashBytes is the digest length, in bytes, of an exchange hash.
	ps2HashBytes = 32
	// ps2Runbook is a runbook note.
	ps2Runbook = "---\ntype: runbook\ntier: L2\nsituation: when releasing\ndone_when: all steps ran\n" +
		"red_flags:\n    - skipping a step\ntriggers:\n    - ship it\nluhmann: \"1\"\ncreated: \"2026-09-27\"\n" +
		"source: test\nuser: alice\nvault: personal\n---\n\n1. Run the checks.\n2. Tag the release.\n"
	// ps2User is the git identity of the env's own commands.
	ps2User = "alice@example.com"
)

// ps2Events is an ordered, goroutine-safe event log.
type ps2Events struct {
	mu      sync.Mutex
	entries []string
}

func (e *ps2Events) add(event string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.entries = append(e.entries, event)
}

func (e *ps2Events) list() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]string(nil), e.entries...)
}

// ps2OrderFS records the filesystem steps that matter to the order of the exchange state: the write of the vault ID
// file and the creation of the .engram directory.
type ps2OrderFS struct {
	cli.EdgeFS

	events *ps2Events
}

func (f ps2OrderFS) MkdirAll(path string, perm fs.FileMode) error {
	if filepath.Base(path) == ".engram" {
		f.events.add(ps2EventStateDir)
	}

	return f.EdgeFS.MkdirAll(path, perm)
}

func (f ps2OrderFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if filepath.Base(path) == ".engram-vault-id" {
		f.events.add(ps2EventVaultID)
	}

	return f.EdgeFS.WriteFileExcl(path, data, perm)
}

// ps2Env is a wiringEnv whose commands run as ps2User, through a scripted git commander.
func ps2Env(t *testing.T) *wiringEnv {
	t.Helper()

	env := newWiringEnv(t)
	env.wrap = ps2Identity(ps2User)

	return env
}

// ps2Identity scripts git so that `git config user.email` answers email and no directory is a repository.
func ps2Identity(email string) func(*cli.Deps) {
	return func(deps *cli.Deps) {
		deps.Commander = fakeGitCommander{
			configOut: email + "\n", remoteErr: errFakeGitFailed, revParseErr: errFakeGitFailed,
		}
		deps.Username = func() (string, error) { return "", errFakeGitFailed }
	}
}

// ps2IndexOf is the position of the first event equal to want, or -1.
func ps2IndexOf(events []string, want string) int {
	for index, event := range events {
		if event == want {
			return index
		}
	}

	return -1
}

// ps2OfferBody is the body of the one offer the env's fake parent received.
func ps2OfferBody(t *testing.T, env *wiringEnv) string {
	t.Helper()

	for _, request := range env.parent.requests() {
		if request.method == "POST" && strings.HasSuffix(request.url, "/learn") && !request.failed {
			return string(request.body)
		}
	}

	t.Fatal("the parent received no offer")

	return ""
}

// ps2OfferKey is offer.key: the hex sha256 of the offer's origin followed by the note's exchange hash.
func ps2OfferKey(origin, hash string) string {
	sum := sha256.Sum256([]byte(origin + hash))

	return hex.EncodeToString(sum[:])
}

// ps2RenameDeps is the Luhmann rename primitive over the real files of a test vault.
func ps2RenameDeps() cli.RenameRewriteDeps {
	return cli.RenameRewriteDeps{
		ListMD: func(vault string) ([]string, error) {
			entries, err := os.ReadDir(vault)
			if err != nil {
				return nil, err
			}

			names := make([]string, 0, len(entries))

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					names = append(names, entry.Name())
				}
			}

			return names, nil
		},
		ReadFile:  os.ReadFile,
		WriteFile: func(path string, data []byte) error { return os.WriteFile(path, data, 0o600) },
		Rename:    os.Rename,
	}
}

// ps2Script answers the probe (GET /query) and then the offer (POST /learn) through the mock Deps.Fetch, by the
// env's fake parent, and hands the offer's body to offerBody. It closes the returned channel when both are answered.
func ps2Script(env *wiringEnv, calls *Pvc1FetchMockMethod, offerBody chan<- []byte) <-chan struct{} {
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		probe := calls.ArgsShould(match.BeAny, Equal("GET"), ContainSubstring("/query"), match.BeAny)
		probeArgs := probe.GetArgs()
		resp, err := env.parent.fetch(probeArgs.Ctx, probeArgs.Method, probeArgs.Url, probeArgs.Body)
		probe.Return(resp, err)

		offer := calls.ArgsShould(match.BeAny, Equal("POST"), HaveSuffix("/learn"), match.BeAny)
		offerArgs := offer.GetArgs()
		offerBody <- append([]byte(nil), offerArgs.Body...)

		resp, err = env.parent.fetch(offerArgs.Ctx, offerArgs.Method, offerArgs.Url, offerArgs.Body)
		offer.Return(resp, err)
	}()

	return finished
}
