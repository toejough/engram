package cli_test

// Scenario tests for the vault-serving product capability, part 2 (861c2zzg): a served learn finds the note an offer
// belongs to (792a20g), an invalid offer is refused with nothing written (792a20h), and placement and tags are the
// server's (792a20i). Spec: docs/specs/product/vault-serving.md.
//
// The scenarios run over the registered /learn route handler with the deps the neighbouring serve tests use: a vault
// in the test's own t.TempDir() (the recorded deviation, #860: the vault lock is a real flock, and no in-memory
// EdgeFS models the exclusive create and the directory listing a learn needs), a deterministic stub embedder, a fixed
// clock and a scripted git, so the server's own detection differs from every value a caller declares. A Given that
// needs an accepted offer, a renamed note or a rewritten pending offer is produced by the commands that make them
// (a served learn, engram amend, the rename pass), never planted by hand. No network and no real home.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a20g1a_ServeLearn_AnAmendBeforeCurationUpdatesThePendingOffer verifies 792a20g1a (An amend before curation
// updates the pending offer): a child's offered note creates pending N1 and the child's amend, with its own key and
// new content, arrives before the host curated N1; N1 carries the amended content with a re-embedded sidecar and no
// second pending note exists.
func Test792a20g1a_ServeLearn_AnAmendBeforeCurationUpdatesThePendingOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	_, created := vs2Offer(t, deps, vault, vs2Fact("offered", "first object", "first"))
	notePath := filepath.Join(vault, created.Basename+".md")
	sidecarBefore := readFileString(t, embed.SidecarPath(notePath))

	resp, amended := vs2Offer(t, deps, vault, vs2Fact("offered", "amended object", "second"))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	written := readFileString(t, notePath)
	g.Expect(amended.Basename).To(Equal(created.Basename), "the amend lands on N1")
	g.Expect(amended.Pending).To(BeTrue())
	g.Expect(written).To(ContainSubstring("object: amended object"), "N1 carries the amended content")
	g.Expect(written).NotTo(ContainSubstring("first object"))
	g.Expect(written).To(MatchRegexp(`(?m)^pending: true$`), "N1 is still pending")
	g.Expect(vs2OfferBlock(t, notePath)["key"]).To(Equal(vs2Key("second")))
	g.Expect(readFileString(t, embed.SidecarPath(notePath))).NotTo(Equal(sidecarBefore),
		"the sidecar is re-embedded for the amended content")
	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "no second pending note exists")
	g.Expect(vs2PendingNotes(t, vault)).To(HaveLen(1))
}

// Test792a20g1b_ServeLearn_ADuplicateKeyWritesNothing verifies 792a20g1b (A duplicate key writes nothing): a second
// served learn with the same offer.origin and offer.key as an existing pending note answers that note's receipt and
// no vault file changes.
func Test792a20g1b_ServeLearn_ADuplicateKeyWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)
	args := vs2Fact("offered", "the object", "only")

	_, first := vs2Offer(t, deps, vault, args)
	before := snapshotVault(t, vault)

	resp, second := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(second).To(Equal(first), "the response is the existing note's receipt")
	g.Expect(second.Pending).To(BeTrue())
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "no vault file changes")
}

// Test792a20g1c_ServeLearn_ARetryAfterAcceptanceWritesNothing verifies 792a20g1c (A retry after acceptance writes
// nothing): an offer that was accepted, so its note is live and still carries offer.origin, retried with the same
// key answers the live note's receipt with pending false, and no pending note is created. The accepting amend runs
// on the fixed clock.
func Test792a20g1c_ServeLearn_ARetryAfterAcceptanceWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)
	args := vs2Fact("accepted", "the object", "key-one")

	_, created := vs2Offer(t, deps, vault, args)
	vs2Accept(t, deps, vault, created)

	livePath := filepath.Join(vault, created.Basename+".md")
	g.Expect(readFileString(t, livePath)).NotTo(ContainSubstring("pending: true"), "the note is live")
	g.Expect(vs2OfferBlock(t, livePath)["origin"]).To(Equal(childOrigin), "and still carries offer.origin")

	before := snapshotVault(t, vault)

	resp, retried := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(retried.Basename).To(Equal(created.Basename), "the response is the live note's receipt")
	g.Expect(retried.Pending).To(BeFalse())
	g.Expect(retried.StoredHash).To(Equal(fileExchangeHash(t, livePath)))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")
	g.Expect(vs2PendingNotes(t, vault)).To(BeEmpty(), "no pending note is created")
}

// Test792a20g1d_ServeLearn_ALateRetryNeverRevertsANewerAmend verifies 792a20g1d (A late retry never reverts a newer
// amend): an accepted live note with offer.origin O and key K1, a newer pending amend from O with key K2, and a late
// retry of O with K1 answer the live note's receipt and leave the pending amend unchanged.
func Test792a20g1d_ServeLearn_ALateRetryNeverRevertsANewerAmend(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)
	older := vs2Fact("offered", "older object", "k1")

	_, accepted := vs2Offer(t, deps, vault, older)
	vs2Accept(t, deps, vault, accepted)

	_, newer := vs2Offer(t, deps, vault, vs2Fact("offered", "newer object", "k2"))
	g.Expect(newer.Basename).NotTo(Equal(accepted.Basename), "the newer amend is its own pending note")
	g.Expect(newer.Pending).To(BeTrue())

	amendPath := filepath.Join(vault, newer.Basename+".md")
	g.Expect(vs2OfferBlock(t, amendPath)["origin"]).To(Equal(childOrigin), "the pending amend is from the same origin")

	before := snapshotVault(t, vault)

	resp, late := vs2Offer(t, deps, vault, older)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(late.Basename).To(Equal(accepted.Basename), "the response is the live note's receipt")
	g.Expect(late.Pending).To(BeFalse())
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "the pending amend is unchanged")
	g.Expect(readFileString(t, amendPath)).To(ContainSubstring("object: newer object"))
}

// Test792a20g1e_ServeLearn_ALateRetryOfASupersededKeyNeverRevertsTheNote verifies 792a20g1e (A late retry of a
// superseded key never reverts the note): a child's note with key K1 creates pending N1, its amend rewrites N1 in
// place with key K2, and K1 retried late answers N1's current receipt, N1 still carries the K2 content and no vault
// file changes.
func Test792a20g1e_ServeLearn_ALateRetryOfASupersededKeyNeverRevertsTheNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)
	older := vs2Fact("offered", "older object", "k1")

	_, created := vs2Offer(t, deps, vault, older)
	_, amended := vs2Offer(t, deps, vault, vs2Fact("offered", "newer object", "k2"))
	g.Expect(amended.Basename).To(Equal(created.Basename), "N1 was rewritten in place")

	notePath := filepath.Join(vault, created.Basename+".md")
	g.Expect(readFileString(t, notePath)).To(ContainSubstring("object: newer object"))

	before := snapshotVault(t, vault)

	resp, late := vs2Offer(t, deps, vault, older)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(late).To(Equal(amended), "the response is N1's current receipt")
	g.Expect(readFileString(t, notePath)).To(ContainSubstring("object: newer object"), "N1 still carries the K2 content")
	g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("older object"))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "no vault file changes")
}

// Test792a20g1f_ServeLearn_ANewKeyFromAnAcceptedOriginMakesANewPendingNote verifies 792a20g1f (A new key from an
// accepted origin makes a new pending note): a live note that carries offer.origin O and a served learn from O with
// a new key, which names no offer.for of its own, write a new pending note whose offer.for names the live note, and
// the receipt carries for.
func Test792a20g1f_ServeLearn_ANewKeyFromAnAcceptedOriginMakesANewPendingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	_, accepted := vs2Offer(t, deps, vault, vs2Fact("accepted", "the object", "k1"))
	vs2Accept(t, deps, vault, accepted)

	livePath := filepath.Join(vault, accepted.Basename+".md")
	liveBefore := readFileString(t, livePath)

	resp, amend := vs2Offer(t, deps, vault, vs2Fact("accepted", "changed object", "k2"))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(amend.Basename).NotTo(Equal(accepted.Basename), "a new note is written")
	g.Expect(amend.Pending).To(BeTrue())
	g.Expect(amend.For).To(Equal(accepted.Basename), "the receipt carries for")

	amendPath := filepath.Join(vault, amend.Basename+".md")
	g.Expect(readFileString(t, amendPath)).To(MatchRegexp(`(?m)^pending: true$`))
	g.Expect(vs2OfferBlock(t, amendPath)["for"]).To(Equal(accepted.Basename), "offer.for names the live note")
	g.Expect(readFileString(t, livePath)).To(Equal(liveBefore), "the live note is unchanged")
	g.Expect(noteFiles(t, vault)).To(HaveLen(2))
}

// Test792a20g1g_ServeLearn_NoCrossOriginOverwrite verifies 792a20g1g (No cross-origin overwrite): an offer from
// origin B whose offer.for names a pending note that came from origin A leaves A's pending note unchanged and writes
// a new pending note whose offer.for names A's note.
func Test792a20g1g_ServeLearn_NoCrossOriginOverwrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	fromA := vs2Fact("from-a", "object of a", "key-a")
	fromA.Offer.Origin = otherOrigin
	fromA.Offer.Path = []string{vs2VaultPart(otherOrigin)}

	_, pendingA := vs2Offer(t, deps, vault, fromA)
	pathA := filepath.Join(vault, pendingA.Basename+".md")
	bytesA := readFileString(t, pathA)
	sidecarA := readFileString(t, embed.SidecarPath(pathA))

	fromB := vs2Fact("from-b", "object of b", "key-b")
	fromB.Offer.For = pendingA.Basename

	resp, pendingB := vs2Offer(t, deps, vault, fromB)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(pendingB.Basename).NotTo(Equal(pendingA.Basename), "B's offer is a new note")
	g.Expect(pendingB.Pending).To(BeTrue())
	g.Expect(readFileString(t, pathA)).To(Equal(bytesA), "A's pending note is unchanged")
	g.Expect(readFileString(t, embed.SidecarPath(pathA))).To(Equal(sidecarA))

	pathB := filepath.Join(vault, pendingB.Basename+".md")
	g.Expect(vs2OfferBlock(t, pathB)["for"]).To(Equal(pendingA.Basename), "offer.for names A's note")
	g.Expect(vs2OfferBlock(t, pathB)["origin"]).To(Equal(childOrigin))
	g.Expect(noteFiles(t, vault)).To(HaveLen(2))
}

// Test792a20g1h_ServeLearn_AnAmendOfferNamesALiveTarget verifies 792a20g1h (An amend-offer names a live target): a
// served learn whose offer.for names live note E, and no note shares its origin, writes a new pending note carrying
// offer.for E, leaves E unchanged and live, and the receipt carries for E. A live note that is not the target sits
// beside E, so naming any live note would not satisfy it.
func Test792a20g1h_ServeLearn_AnAmendOfferNamesALiveTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs2LearnLive(t, deps, vault, "bystander")
	vs2LearnLive(t, deps, vault, "target")

	target := vs2NoteBasename(t, vault, "target")
	targetPath := filepath.Join(vault, target+".md")
	targetBefore := readFileString(t, targetPath)
	sidecarBefore := readFileString(t, embed.SidecarPath(targetPath))

	args := vs2Fact("amend-offer", "amended object", "k1")
	args.Offer.For = target

	resp, receipt := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(receipt.For).To(Equal(target), "the receipt carries for E")
	g.Expect(receipt.Pending).To(BeTrue())
	g.Expect(receipt.Basename).NotTo(Equal(target))

	offerPath := filepath.Join(vault, receipt.Basename+".md")
	g.Expect(vs2OfferBlock(t, offerPath)["for"]).To(Equal(target), "the new pending note carries offer.for E")
	g.Expect(readFileString(t, targetPath)).To(Equal(targetBefore), "E is unchanged")
	g.Expect(readFileString(t, embed.SidecarPath(targetPath))).To(Equal(sidecarBefore))
	g.Expect(targetBefore).NotTo(ContainSubstring("pending: true"), "and live")
	g.Expect(vs2PendingNotes(t, vault)).To(ConsistOf(offerPath))
}

// Test792a20g1i_ServeLearn_ARenamedTargetResolvesThroughItsAliases verifies 792a20g1i (A renamed target resolves
// through its aliases): live note E is renamed by the rename pass, which records its old basename in aliases, and an
// offer whose offer.for names that old basename resolves to E under its new basename.
func Test792a20g1i_ServeLearn_ARenamedTargetResolvesThroughItsAliases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs2LearnLive(t, deps, vault, "renamed")

	oldName := vs2NoteBasename(t, vault, "renamed")
	newName := vs2Rename(t, vault, oldName)
	g.Expect(newName).NotTo(Equal(oldName))
	g.Expect(readFileString(t, filepath.Join(vault, newName+".md"))).To(ContainSubstring(oldName),
		"the renamed note lists its old basename in aliases")

	args := vs2Fact("amend-offer", "amended object", "k1")
	args.Offer.For = oldName

	resp, receipt := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(receipt.For).To(Equal(newName), "the offer resolves to the renamed note")
	g.Expect(vs2OfferBlock(t, filepath.Join(vault, receipt.Basename+".md"))["for"]).To(Equal(newName))
}

// Test792a20g1j_ServeLearn_AnUnresolvableTargetIsDropped verifies 792a20g1j (An unresolvable target is dropped): an
// offer.for that names no live note, no alias and no pending note is dropped, and the new pending note carries no
// offer.for. Near misses sit in the vault (a live note whose basename differs from the name only in its last
// characters, a pending note of another name) so a loose match would resolve.
func Test792a20g1j_ServeLearn_AnUnresolvableTargetIsDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs2LearnLive(t, deps, vault, "near-miss")

	nearMiss := vs2NoteBasename(t, vault, "near-miss")

	bystander := vs2Fact("bystander", "bystander object", "kb")
	bystander.Offer.Origin = otherOrigin
	bystander.Offer.Path = []string{vs2VaultPart(otherOrigin)}
	vs2Offer(t, deps, vault, bystander)

	args := vs2Fact("amend-offer", "amended object", "k1")
	args.Offer.For = nearMiss[:len(nearMiss)-1] + "-gone"

	resp, receipt := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(receipt.Pending).To(BeTrue())
	g.Expect(receipt.For).To(BeEmpty())
	g.Expect(string(resp.Body)).NotTo(ContainSubstring(`"for"`), "the receipt carries no for")
	g.Expect(vs2OfferBlock(t, filepath.Join(vault, receipt.Basename+".md"))).NotTo(HaveKey("for"),
		"the new pending note carries no offer.for")
}

// Test792a20g1k_ServeLearn_PriorKeysAreBounded verifies 792a20g1k (Prior keys are bounded): a pending note rewritten
// in place twelve times, once per new key, holds in offer.prior_keys exactly the eight most recent superseded keys,
// oldest first, with the older ones dropped.
func Test792a20g1k_ServeLearn_PriorKeysAreBounded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		rewrites   = 12
		keptPrior  = 8
		firstKept  = rewrites - 1 - keptPrior
		lastKeyTag = rewrites - 1
	)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	var receipt receiptBody

	for index := range rewrites {
		resp, current := vs2Offer(t, deps, vault, vs2Fact("rewritten", fmt.Sprintf("object %02d", index),
			fmt.Sprintf("key-%02d", index)))
		g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

		receipt = current
	}

	notePath := filepath.Join(vault, receipt.Basename+".md")
	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "every rewrite landed on the one pending note")

	offer := vs2OfferBlock(t, notePath)
	g.Expect(offer["key"]).To(Equal(vs2Key(fmt.Sprintf("key-%02d", lastKeyTag))), "the current key is the last one")

	want := make([]any, 0, keptPrior)
	for index := firstKept; index < lastKeyTag; index++ {
		want = append(want, vs2Key(fmt.Sprintf("key-%02d", index)))
	}

	g.Expect(offer["prior_keys"]).To(Equal(want), "the eight most recent superseded keys, oldest dropped")
}

// Test792a20g1l_ServeLearn_AnUnreadableNoteFailsTheRequest verifies 792a20g1l (An unreadable note fails the request):
// a note of the served vault that cannot be read for a reason other than being removed makes a served learn answer an
// error and write nothing, so the child retries. The contrasting case is a note removed since the listing, which is
// skipped and does not fail the request.
func Test792a20g1l_ServeLearn_AnUnreadableNoteFailsTheRequest(t *testing.T) {
	t.Parallel()

	t.Run("a note that cannot be read fails the request", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := vs1ServerDeps()
		vault := newServeVault(t)

		_, pending := vs2Offer(t, deps, vault, vs2Fact("offered", "first object", "key-one"))
		before := snapshotVault(t, vault)

		denying := deps
		denying.FS = vs2ReadFailFS{
			EdgeFS: deps.FS, failing: filepath.Join(vault, pending.Basename+".md"),
			err: fmt.Errorf("reading the note: %w", fs.ErrPermission),
		}

		resp, _ := vs2Offer(t, denying, vault, vs2Fact("offered", "second object", "key-two"))
		g.Expect(resp.Status).To(Equal(vs2StatusServerError), string(resp.Body))
		g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written, so no second pending note")
		g.Expect(vs2PendingNotes(t, vault)).To(HaveLen(1))
	})

	t.Run("a note removed since the listing is skipped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := vs1ServerDeps()
		vault := newServeVault(t)

		_, pending := vs2Offer(t, deps, vault, vs2Fact("offered", "first object", "key-one"))

		vanishing := deps
		vanishing.FS = vs2ReadFailFS{
			EdgeFS: deps.FS, failing: filepath.Join(vault, pending.Basename+".md"),
			err: &fs.PathError{Op: "open", Path: pending.Basename, Err: fs.ErrNotExist},
		}

		resp, _ := vs2Offer(t, vanishing, vault, vs2Fact("offered", "second object", "key-two"))
		g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
		g.Expect(vs2PendingNotes(t, vault)).To(HaveLen(2), "the vanished note is not an error")
	})
}

// Test792a20h1a_ServeLearn_AKeyWithoutAnOriginIsRejected verifies 792a20h1a (A key without an origin is rejected): a
// served learn whose offer carries offer.key or offer.path and no offer.origin answers 400 and writes nothing.
func Test792a20h1a_ServeLearn_AKeyWithoutAnOriginIsRejected(t *testing.T) {
	t.Parallel()

	cases := map[string]cli.LearnOffer{
		"offer.key alone":               {Key: vs2Key("k1")},
		"offer.path alone":              {Path: []string{childVaultID}},
		"offer.key and offer.path":      {Key: vs2Key("k1"), Path: []string{childVaultID}},
		"offer.for with key, no origin": {Key: vs2Key("k1"), For: "7.2026-01-07.existing"},
	}

	for name, offer := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			refusal := vs2Refuse(t, newServeVault(t), offer)
			g.Expect(refusal.status).To(Equal(vs1StatusBadRequest), refusal.body)
			g.Expect(refusal.message).NotTo(BeEmpty(), "the refusal says why")
			g.Expect(refusal.changed).To(BeFalse(), "nothing is written")
		})
	}
}

// Test792a20h1b_ServeLearn_AMalformedOrOversizedPathIsRejected verifies 792a20h1b (A malformed or oversized path is
// rejected): a served learn whose offer.path has 17 entries, or an entry that is not 32 lowercase hex characters,
// answers 400 and writes nothing; sixteen well-formed entries are accepted, so the bound is exactly seventeen.
func Test792a20h1b_ServeLearn_AMalformedOrOversizedPathIsRejected(t *testing.T) {
	t.Parallel()

	const (
		oversized = 17
		allowed   = 16
	)

	paths := map[string][]string{
		"17 entries":                  vs2PathOf(oversized),
		"an uppercase entry":          {strings.ToUpper(childVaultID)},
		"a 31-character entry":        {childVaultID[:31]},
		"a 33-character entry":        {childVaultID + "0"},
		"a non-hex entry":             {strings.Repeat("g", 32)},
		"an empty entry":              {""},
		"a bad entry after good ones": append(vs2PathOf(2), "not-a-vault-id"),
	}

	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			refusal := vs2Refuse(t, newServeVault(t),
				cli.LearnOffer{Origin: childOrigin, Key: vs2Key("k1"), Path: path})
			g.Expect(refusal.status).To(Equal(vs1StatusBadRequest), refusal.body)
			g.Expect(refusal.message).NotTo(BeEmpty(), "the refusal says why")
			g.Expect(refusal.changed).To(BeFalse(), "nothing is written")
		})
	}

	t.Run("16 well-formed entries are accepted", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		offer := cli.LearnOffer{Origin: childOrigin, Key: vs2Key("k1"), Path: vs2PathOf(allowed)}

		resp, receipt := vs2Offer(t, vs1ServerDeps(), vault, offeredFact("accepted", "object", offer))
		g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
		g.Expect(receipt.Pending).To(BeTrue())
	})
}

// Test792a20h1c_ServeLearn_AMalformedOriginIsRejected verifies 792a20h1c (A malformed origin is rejected): a served
// learn whose offer.origin is not two 32-character identifiers joined by a colon answers 400 and writes nothing.
func Test792a20h1c_ServeLearn_AMalformedOriginIsRejected(t *testing.T) {
	t.Parallel()

	origins := map[string]string{
		"no separator":          childVaultID + childXID,
		"a short second half":   childVaultID + ":abc",
		"a short first half":    "abc:" + childXID,
		"three parts":           childOrigin + ":" + childXID,
		"an empty second half":  childVaultID + ":",
		"an empty first half":   ":" + childXID,
		"uppercase hex":         strings.ToUpper(childVaultID) + ":" + childXID,
		"non-hex identifiers":   strings.Repeat("g", 32) + ":" + strings.Repeat("g", 32),
		"a dash for the colon":  childVaultID + "-" + childXID,
		"a 33-character prefix": childVaultID + "0:" + childXID,
	}

	for name, origin := range origins {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			refusal := vs2Refuse(t, newServeVault(t),
				cli.LearnOffer{Origin: origin, Key: vs2Key("k1"), Path: []string{childVaultID}})
			g.Expect(refusal.status).To(Equal(vs1StatusBadRequest), refusal.body)
			g.Expect(refusal.message).NotTo(BeEmpty(), "the refusal says why")
			g.Expect(refusal.changed).To(BeFalse(), "nothing is written")
		})
	}
}

// Test792a20h1d_ServeLearn_ACycleIsRefused verifies 792a20h1d (A cycle is refused): a served learn whose offer.path
// already contains the server's own vault ID answers 409 with reason "loop" and the server's vault_id, and writes
// nothing, wherever in the path the server's ID sits.
func Test792a20h1d_ServeLearn_ACycleIsRefused(t *testing.T) {
	t.Parallel()

	paths := map[string][]string{
		"only the server's ID":          {serverVaultID},
		"the server's ID last":          {childVaultID, serverVaultID},
		"the server's ID first":         {serverVaultID, childVaultID},
		"the server's ID in the middle": {childVaultID, serverVaultID, otherOrigin[:32]},
	}

	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			refusal := vs2Refuse(t, newServeVault(t), cli.LearnOffer{Origin: childOrigin, Key: vs2Key("k1"), Path: path})
			g.Expect(refusal.status).To(Equal(vs2StatusConflict), refusal.body)

			var body map[string]any
			g.Expect(json.Unmarshal([]byte(refusal.body), &body)).To(Succeed())
			g.Expect(body).To(HaveKeyWithValue("reason", "loop"))
			g.Expect(body).To(HaveKeyWithValue("vault_id", serverVaultID))
			g.Expect(refusal.message).NotTo(BeEmpty())
			g.Expect(refusal.changed).To(BeFalse(), "nothing is written")
		})
	}
}

// Test792a20h1e_ServeLearn_InvalidNoteContentIsAClientError verifies 792a20h1e (Invalid note content is a client
// error): a served learn whose note fails validation, such as an unknown type, answers 400 and writes nothing.
func Test792a20h1e_ServeLearn_InvalidNoteContentIsAClientError(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*cli.LearnArgs){
		"an unknown type":                func(args *cli.LearnArgs) { args.Type = "bogus" },
		"an unknown tier":                func(args *cli.LearnArgs) { args.Tier = "L9" },
		"an unparseable supersedes":      func(args *cli.LearnArgs) { args.Supersedes = []string{"no-separators"} },
		"a fact without a situation":     func(args *cli.LearnArgs) { args.Situation = "" },
		"a runbook without a done-when":  func(args *cli.LearnArgs) { args.Type, args.DoneWhen = "runbook", "" },
		"a feedback without a situation": func(args *cli.LearnArgs) { args.Type, args.Situation = "feedback", " " },
		"an invalid slug":                func(args *cli.LearnArgs) { args.Slug = "Not A Slug" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			before := snapshotVault(t, vault)

			args := vs2Fact("invalid", "object", "k1")
			mutate(&args)

			resp, _ := vs2Offer(t, vs1ServerDeps(), vault, args)
			g.Expect(resp.Status).To(Equal(vs1StatusBadRequest), string(resp.Body))
			g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")
		})
	}
}

// Test792a20i1a_ServeLearn_PlacementIsIgnored verifies 792a20i1a (Placement is ignored): a served learn with target
// "12" and position "child" writes a pending note that receives a new top-level Luhmann ID, not a child of 12. The
// position learn itself knows to place a note under a target (continuation) is sent as well: "child" is the
// scenario's word and no local learn accepts it, so only "continuation" would place a note under 12 if the server
// honored the request.
func Test792a20i1a_ServeLearn_PlacementIsIgnored(t *testing.T) {
	t.Parallel()

	for _, position := range []string{"child", "continuation"} {
		t.Run(position, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			writeVaultNote(t, vault, "12.2026-01-12.existing.md", liveFactNote("12", ""))

			args := vs2Fact("placed", "object", "k1")
			args.Target, args.Position = "12", position

			resp, receipt := vs2Offer(t, vs1ServerDeps(), vault, args)
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
			g.Expect(receipt.Luhmann).To(Equal("13"), "the next top-level ID, not a child of 12")
			g.Expect(receipt.Basename).To(HavePrefix("13."))
			g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
				To(MatchRegexp(`(?m)^luhmann: "?13"?$`))
		})
	}
}

// Test792a20i1b_ServeLearn_WireTagsAreDropped verifies 792a20i1b (Wire tags are dropped): a served learn that carries
// tags, first as a new offer and then as a same-origin rewrite, leaves the pending note carrying none of those tags.
func Test792a20i1b_ServeLearn_WireTagsAreDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	first := vs2Fact("tagged", "first object", "k1")
	first.Tags = []string{"wire-family/first-tag", "wire-bare"}

	resp, receipt := vs2Offer(t, deps, vault, first)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	notePath := filepath.Join(vault, receipt.Basename+".md")
	asNew := readFileString(t, notePath)
	g.Expect(asNew).To(ContainSubstring("object: first object"))
	g.Expect(asNew).NotTo(MatchRegexp(`(?m)^tags:`), "the new pending note carries no tags")
	g.Expect(asNew).NotTo(ContainSubstring("first-tag"))
	g.Expect(asNew).NotTo(ContainSubstring("wire-bare"))

	second := vs2Fact("tagged", "second object", "k2")
	second.Tags = []string{"wire-family/second-tag", "wire-other"}

	resp, rewritten := vs2Offer(t, deps, vault, second)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(rewritten.Basename).To(Equal(receipt.Basename), "the second offer is a same-origin rewrite")

	asRewritten := readFileString(t, notePath)
	g.Expect(asRewritten).To(ContainSubstring("object: second object"))
	g.Expect(asRewritten).NotTo(MatchRegexp(`(?m)^tags:`), "the rewritten note carries no tags")
	g.Expect(asRewritten).NotTo(ContainSubstring("second-tag"))
	g.Expect(asRewritten).NotTo(ContainSubstring("wire-other"))
}

// unexported constants.
const (
	vs2StatusConflict    = 409
	vs2StatusServerError = 500
)

// unexported variables.
var (
	vs2PendingMarker = regexp.MustCompile(`(?m)^pending: true$`)
)

// vs2ReadFailFS fails the read of one path with err and passes every other call through.
type vs2ReadFailFS struct {
	cli.EdgeFS

	failing string
	err     error
}

// ReadFile fails for the failing path.
func (f vs2ReadFailFS) ReadFile(path string) ([]byte, error) {
	if path == f.failing {
		return nil, f.err
	}

	data, err := f.EdgeFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

// vs2Refusal is what a served learn answered and whether it changed the vault.
type vs2Refusal struct {
	status  int
	body    string
	message any
	changed bool
}

// vs2Accept accepts the pending offer of receipt the way the curation does: engram amend clears the pending marker
// against the hash the curator judged.
func vs2Accept(t *testing.T, deps cli.Deps, vault string, receipt receiptBody) {
	t.Helper()

	cleared := false
	notePath := filepath.Join(vault, receipt.Basename+".md")

	judged, hashErr := cli.ExportExchangeHash([]byte(readFileString(t, notePath)))
	if hashErr != nil {
		t.Fatalf("hashing %s: %v", notePath, hashErr)
	}

	amendErr := cli.ExportRunAmend(t.Context(),
		cli.AmendArgs{Vault: vault, Target: receipt.Luhmann, Pending: &cleared, ExpectHash: judged},
		cli.ExportNewAmendDeps(deps), io.Discard)
	if amendErr != nil {
		t.Fatalf("accepting %s: %v", receipt.Basename, amendErr)
	}
}

// vs2Fact is a served learn body for a fact offered by the child vault of childOrigin, with the offer record the
// child builds: its origin, a key derived from the content tag and the path of the vaults it travelled.
func vs2Fact(slug, object, keyTag string) cli.LearnArgs {
	offer := cli.LearnOffer{Origin: childOrigin, Key: vs2Key(keyTag), Path: []string{childVaultID}}

	return offeredFact(slug, object, offer)
}

// vs2Key is an idempotency key shaped like the child's: the hex sha256 of the origin and a content tag.
func vs2Key(tag string) string {
	sum := sha256.Sum256([]byte(childOrigin + tag))

	return hex.EncodeToString(sum[:])
}

// vs2LearnLive writes a live note through the local learn path.
func vs2LearnLive(t *testing.T, deps cli.Deps, vault, slug string) {
	t.Helper()

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: slug, Position: "top", Source: "local",
		Situation: "a live note about " + slug, Subject: "engram", Predicate: "keeps", Object: slug + " object",
	})
}

// vs2NoteBasename returns the extensionless basename of the one note of vault whose slug is slug.
func vs2NoteBasename(t *testing.T, vault, slug string) string {
	t.Helper()

	return strings.TrimSuffix(filepath.Base(vs1NoteWithSlug(t, vault, slug)), ".md")
}

// vs2Offer posts args to the served learn route over vault and returns the response and, when it is a receipt, the
// decoded receipt.
func vs2Offer(t *testing.T, deps cli.Deps, vault string, args cli.LearnArgs) (cli.ServeResponse, receiptBody) {
	t.Helper()

	resp := serveLearnWith(t, deps, vault, args)

	var receipt receiptBody

	if resp.Status == vs1StatusOK {
		receipt = decodeReceipt(t, resp)
	}

	return resp, receipt
}

// vs2OfferBlock returns the offer: mapping of the note at path.
func vs2OfferBlock(t *testing.T, path string) map[string]any {
	t.Helper()

	frontmatter, _, found := strings.Cut(strings.TrimPrefix(readFileString(t, path), "---\n"), "\n---\n")
	if !found {
		t.Fatalf("%s has no frontmatter", path)
	}

	var fields map[string]any

	unmarshalErr := yaml.Unmarshal([]byte(frontmatter), &fields)
	if unmarshalErr != nil {
		t.Fatalf("parsing the frontmatter of %s: %v", path, unmarshalErr)
	}

	offer, isMap := fields["offer"].(map[string]any)
	if !isMap {
		t.Fatalf("%s has no offer: mapping: %v", path, fields)
	}

	return offer
}

// vs2PathOf is count distinct, well-formed vault IDs, none of them the server's.
func vs2PathOf(count int) []string {
	path := make([]string, 0, count)

	for index := range count {
		path = append(path, fmt.Sprintf("%032x", index+1))
	}

	return path
}

// vs2PendingNotes lists the notes of vault that carry the pending marker.
func vs2PendingNotes(t *testing.T, vault string) []string {
	t.Helper()

	pending := make([]string, 0, len(noteFiles(t, vault)))

	for _, note := range noteFiles(t, vault) {
		if vs2PendingMarker.MatchString(readFileString(t, note)) {
			pending = append(pending, note)
		}
	}

	return pending
}

// vs2Refuse posts an offer for a fresh fact and reports the answer, the error it carries and whether the vault
// changed.
func vs2Refuse(t *testing.T, vault string, offer cli.LearnOffer) vs2Refusal {
	t.Helper()

	before := snapshotVault(t, vault)

	resp, _ := vs2Offer(t, vs1ServerDeps(), vault, offeredFact("refused", "object", offer))

	var fields map[string]any

	unmarshalErr := json.Unmarshal(resp.Body, &fields)
	if unmarshalErr != nil {
		t.Fatalf("decoding the answer %s: %v", resp.Body, unmarshalErr)
	}

	return vs2Refusal{
		status: resp.Status, body: string(resp.Body), message: fields["error"],
		changed: !reflect.DeepEqual(snapshotVault(t, vault), before),
	}
}

// vs2Rename renames the note oldName of vault through the rename pass, which records the old basename in the renamed
// note's aliases, and returns the new basename.
func vs2Rename(t *testing.T, vault, oldName string) string {
	t.Helper()

	const newID = "77"

	_, rest, _ := strings.Cut(oldName, ".")
	newName := newID + "." + rest

	_, renameErr := cli.RenameAndRewriteReferences(ps2RenameDeps(), vault,
		map[string]string{oldName: newName})
	if renameErr != nil {
		t.Fatalf("renaming %s: %v", oldName, renameErr)
	}

	return newName
}

// vs2VaultPart is the vault half of an origin.
func vs2VaultPart(origin string) string {
	vaultPart, _, _ := strings.Cut(origin, ":")

	return vaultPart
}
