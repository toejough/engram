package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestBuildOfferPayload_AmendOfferTargetsPrimaryLink: a note whose primary
// link is under the configured parent's vault ID is an amend-offer whose
// offer.for is that parent note (spec: "A content amend of a linked note is
// an amend-offer").
func TestBuildOfferPayload_AmendOfferTargetsPrimaryLink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := payloadNote(t, offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:x"})

	payload := decodePayload(t, note, parentVaultID)
	g.Expect(payload.Offer.For).To(Equal("5.2026-09-27.p"))
}

// TestBuildOfferPayload_CoveredOrForeignLinkIsLearnOffer: only a primary
// link under the current parent's vault ID makes an amend-offer.
func TestBuildOfferPayload_CoveredOrForeignLinkIsLearnOffer(t *testing.T) {
	t.Parallel()

	cases := map[string]offerTestNote{
		"unlinked":        {},
		"covered only":    {parentVault: parentVaultID, linkVia: "covered", linkHash: "xh1:x"},
		"another parent":  {parentVault: seqID(77), linkVia: "offered", linkHash: "xh1:x"},
		"pulled, foreign": {parentVault: seqID(78), linkVia: "pulled", linkHash: "xh1:x"},
		// S16: with the parent's ID unknown, no link stands in for it.
		"parent ID unknown": {parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:x", unknownParent: true},
	}

	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			known := parentVaultID
			if spec.unknownParent {
				known = ""
			}

			payload := decodePayload(t, payloadNote(t, spec), known)
			g.Expect(payload.Offer.For).To(BeEmpty())
		})
	}
}

// TestBuildOfferPayload_DeclaresNotesOwnIdentity (M5): the note's own
// user/repo are declared; detection fills in only when they are empty.
func TestBuildOfferPayload_DeclaresNotesOwnIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	own := decodePayload(t, payloadNote(t, offerTestNote{}), parentVaultID)
	g.Expect(own.User).To(Equal("alice"))

	raw := strings.Replace(string(offerTestNote{}.render(t)), "user: alice\n", "user: \"\"\n", 1)
	note := cli.OfferNoteForTest{
		XID: xidA, Basename: "1.2026-09-27.note", Raw: []byte(raw), Hash: mustHash(t, []byte(raw)),
	}

	body, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "detected-repo", "bob", nil)
	g.Expect(err).NotTo(HaveOccurred())

	var fallback cli.LearnArgs
	g.Expect(json.Unmarshal(body, &fallback)).To(Succeed())
	g.Expect(fallback.User).To(Equal("bob"))
	g.Expect(fallback.Repo).To(Equal("detected-repo"))
}

// TestBuildOfferPayload_EmptyPathSeedsFromOrigin (S16): an accepted served
// offer that recorded no offer.path starts its path from its origin vault.
func TestBuildOfferPayload_EmptyPathSeedsFromOrigin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	child := seqID(40)
	payload := decodePayload(t, payloadNote(t, offerTestNote{origin: child + ":" + xidB}), parentVaultID)
	g.Expect(payload.Offer.Path).To(Equal([]string{child, seqID(60)}))
}

// TestBuildOfferPayload_OriginKeyAndPath: offer.origin is <local id>:<xid>,
// offer.key is sha256(origin + exchange hash), and offer.path is the local
// ID for a direct offer.
func TestBuildOfferPayload_OriginKeyAndPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := payloadNote(t, offerTestNote{})
	payload := decodePayload(t, note, parentVaultID)

	origin := seqID(60) + ":" + xidA
	sum := sha256.Sum256([]byte(origin + note.Hash))

	g.Expect(payload.Offer.Origin).To(Equal(origin))
	g.Expect(payload.Offer.Key).To(Equal(hex.EncodeToString(sum[:])))
	g.Expect(payload.Offer.Path).To(Equal([]string{seqID(60)}))
}

// TestBuildOfferPayload_PlacementIsNotSent: the request carries no target,
// position, chunkSources or vault key at all (spec "Placement is not
// sent"), even for a note with chunk provenance.
func TestBuildOfferPayload_PlacementIsNotSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := strings.Replace(string(offerTestNote{}.render(t)), "vault: personal\n",
		"vault: personal\nsources:\n  - s#a\n", 1)
	note := cli.OfferNoteForTest{
		XID: xidA, Basename: "12a.2026-09-27.note", Raw: []byte(raw), Hash: mustHash(t, []byte(raw)),
	}

	body, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "", "bob", nil)
	g.Expect(err).NotTo(HaveOccurred())

	var keys map[string]any
	g.Expect(json.Unmarshal(body, &keys)).To(Succeed())

	for _, forbidden := range []string{"target", "position", "chunkSources", "vault", "pending"} {
		g.Expect(keys).NotTo(HaveKey(forbidden))
	}

	g.Expect(keys).To(HaveKeyWithValue("slug", "note"))
}

// TestBuildOfferPayload_PropagatedOfferAppendsPath (M14, D12): an accepted
// served offer goes onward with its offer.path plus this vault's ID, and
// its own origin.
func TestBuildOfferPayload_PropagatedOfferAppendsPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	child := seqID(40)
	note := payloadNote(t, offerTestNote{origin: child + ":" + xidB, path: []string{child}})

	payload := decodePayload(t, note, parentVaultID)
	g.Expect(payload.Offer.Path).To(Equal([]string{child, seqID(60)}))
	g.Expect(payload.Offer.Origin).To(Equal(seqID(60) + ":" + xidA))
}

// TestBuildOfferPayload_RoundTripsThroughServedLearnProperty: rendering
// the payload the way a served learn does reproduces the note's exchange
// hash — every offered field survives the wire (a learn with no
// supersedes), and offer.key moves exactly when the hash does.
func TestBuildOfferPayload_RoundTripsThroughServedLearnProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		args := drawLearnArgs(rt)

		content, err := cli.ExportAssembleLearnContent(args, "7", testNow, "repo", "alice", "personal")
		if err != nil {
			rt.Fatal(err)
		}

		hash := mustHash(rt, []byte(content))
		note := cli.OfferNoteForTest{
			XID: xidA, Basename: "7.2026-09-27." + args.Slug, Raw: []byte(content), Hash: hash,
		}

		body, buildErr := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "", "", nil)
		if buildErr != nil {
			rt.Fatal(buildErr)
		}

		var wire cli.LearnArgs
		if json.Unmarshal(body, &wire) != nil {
			rt.Fatalf("payload does not decode: %s", body)
		}

		served, renderErr := cli.ExportAssembleLearnContent(wire, "9", testNow.Add(time.Hour), wire.Repo, wire.User, "")
		if renderErr != nil {
			rt.Fatalf("served render: %v (payload %s)", renderErr, body)
		}

		if got := mustHash(rt, []byte(served)); got != hash {
			rt.Fatalf("served hash %s != note hash %s\nnote:\n%s\nserved:\n%s", got, hash, content, served)
		}

		changed := note
		changed.Hash = "xh1:" + strings.Repeat("f", 64)

		changedBody, _ := cli.ExportBuildOfferPayload(changed, seqID(60), parentVaultID, "", "", nil)

		var changedWire cli.LearnArgs
		_ = json.Unmarshal(changedBody, &changedWire)

		if changedWire.Offer.Key == wire.Offer.Key {
			rt.Fatalf("offer.key did not move with the hash")
		}
	})
}

// TestBuildOfferPayload_RunbookFields: a runbook's steps, done_when, red
// flags and triggers are sent; the body carries no Supersedes lines.
func TestBuildOfferPayload_RunbookFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := "---\ntype: runbook\ntier: L2\nsituation: when x\ndone_when: done\nred_flags:\n  - rf\n" +
		"triggers:\n  - trig\nluhmann: \"4\"\ncreated: \"2026-09-27\"\nsource: s\nuser: alice\nvault: personal\n" +
		"supersedes:\n  - note: 2.2026-09-27.old\n    type: updates\n    claim: c\nxid: " + xidA + "\n---\n\n" +
		"1. step one\n2. step two\n\nSupersedes: [[2.2026-09-27.old]] — updates: c\n"
	note := cli.OfferNoteForTest{
		XID: xidA, Basename: "4.2026-09-27.rb", Raw: []byte(raw), Hash: mustHash(t, []byte(raw)),
	}

	body, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "", "", nil)
	g.Expect(err).NotTo(HaveOccurred())

	var wire cli.LearnArgs
	g.Expect(json.Unmarshal(body, &wire)).To(Succeed())
	g.Expect(wire.Type).To(Equal("runbook"))
	g.Expect(wire.DoneWhen).To(Equal("done"))
	g.Expect(wire.RedFlags).To(Equal([]string{"rf"}))
	g.Expect(wire.Triggers).To(Equal([]string{"trig"}))
	g.Expect(wire.Body).To(Equal("1. step one\n2. step two\n"))
	g.Expect(wire.Supersedes).To(BeEmpty(), "unlinked supersedes target is dropped")
}

// TestBuildOfferPayload_SupersedesTranslatedOrDropped (spec "Supersedes is
// translated or dropped"): A → PA, B (unlinked) dropped.
func TestBuildOfferPayload_SupersedesTranslatedOrDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := strings.Replace(string(offerTestNote{}.render(t)), "xid: ",
		"supersedes:\n  - note: 2.2026-09-27.a.md\n    type: updates\n    claim: claim a\n"+
			"  - note: 3.2026-09-27.b\n    type: refutes\n    claim: claim b\nxid: ", 1)
	note := cli.OfferNoteForTest{
		XID: xidA, Basename: "1.2026-09-27.note", Raw: []byte(raw), Hash: mustHash(t, []byte(raw)),
	}

	body, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "", "bob",
		map[string]string{"2.2026-09-27.a": "70.2026-09-01.pa"})
	g.Expect(err).NotTo(HaveOccurred())

	var wire cli.LearnArgs
	g.Expect(json.Unmarshal(body, &wire)).To(Succeed())
	g.Expect(wire.Supersedes).To(Equal([]string{"70.2026-09-01.pa|updates|claim a"}))
}

// TestBuildOfferPayload_WithdrawnAtSendTime (S16, D12): with the parent's ID
// known at send time, an offer whose origin vault is the parent, or whose
// content equals its primary link's hash, is withdrawn, not sent.
func TestBuildOfferPayload_WithdrawnAtSendTime(t *testing.T) {
	t.Parallel()

	cases := map[string]offerTestNote{
		"origin is the parent": {origin: parentVaultID + ":" + xidB, path: []string{parentVaultID}},
		"equal link hash":      {parentVault: parentVaultID, linkVia: "offered", linkMatches: true},
	}

	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, err := cli.ExportBuildOfferPayload(payloadNote(t, spec), seqID(60), parentVaultID, "", "bob", nil)
			g.Expect(err).To(MatchError(cli.ErrOfferWithdrawnForTest))
		})
	}
}

func decodePayload(t *testing.T, note cli.OfferNoteForTest, parentID string) cli.LearnArgs {
	t.Helper()

	body, err := cli.ExportBuildOfferPayload(note, seqID(60), parentID, "", "bob", nil)
	if err != nil {
		t.Fatal(err)
	}

	var wire cli.LearnArgs

	unmarshalErr := json.Unmarshal(body, &wire)
	if unmarshalErr != nil {
		t.Fatalf("payload %s: %v", body, unmarshalErr)
	}

	return wire
}

func drawLearnArgs(rt *rapid.T) cli.LearnArgs {
	text := rapid.StringMatching(`[a-z][a-z ]{0,12}[a-z]`)
	args := cli.LearnArgs{
		Type:      rapid.SampledFrom([]string{"fact", "feedback", "runbook"}).Draw(rt, "type"),
		Slug:      rapid.StringMatching(`[a-z0-9][a-z0-9-]{0,10}`).Draw(rt, "slug"),
		Situation: text.Draw(rt, "situation"),
		Source:    text.Draw(rt, "source"),
	}

	switch args.Type {
	case "fact":
		args.Subject, args.Predicate, args.Object = text.Draw(rt, "s"), text.Draw(rt, "p"), text.Draw(rt, "o")
	case "feedback":
		args.Behavior, args.Impact, args.Action = text.Draw(rt, "b"), text.Draw(rt, "i"), text.Draw(rt, "a")
	default:
		args.DoneWhen = text.Draw(rt, "done")
		args.Body = "1. " + text.Draw(rt, "step") + "\n"
		args.RedFlags = rapid.SliceOfN(text, 0, 2).Draw(rt, "red-flags")
		args.Triggers = rapid.SliceOfN(rapid.StringMatching(`[a-z]{3,8}`), 0, 2).Draw(rt, "triggers")
	}

	return args
}

// payloadNote renders spec as a queued note.
func payloadNote(t *testing.T, spec offerTestNote) cli.OfferNoteForTest {
	t.Helper()

	raw := spec.render(t)

	return cli.OfferNoteForTest{
		XID: xidA, Basename: "1.2026-09-27.note", Path: "/vault/1.2026-09-27.note.md", Raw: raw, Hash: mustHash(t, raw),
	}
}
