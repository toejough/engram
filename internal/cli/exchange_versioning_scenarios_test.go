package cli_test

// Scenario tests for the exchange-versioning technical component (861c2u): comparing two hashes (792b33c5), the
// lazy exchange ID (792b33c7) and recording an offer receipt (792b33c9). Spec:
// docs/specs/technical/exchange-versioning.md.
//
// The pure scenarios call the exported seams of the package directly. The one interaction scenario drains an
// in-memory vault (outboxEnv) with MockOo1Send for the drain's send step and the real receipt writer.

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/cli"
)

// Test792b33c5a3_CompareExchangeHashes_EqualMatchesAndChangedChanges verifies 792b33c5a3 (Equal matches and
// changed changes): for two equal current hashes the equality check is true and the change check is false; for two
// different current hashes the equality check is false and the change check is true.
func Test792b33c5a3_CompareExchangeHashes_EqualMatchesAndChangedChanges(t *testing.T) {
	t.Parallel()

	first := cli.ExportExchangeHashPrefix + strings.Repeat("ab", 32)
	second := cli.ExportExchangeHashPrefix + strings.Repeat("cd", 32)

	cases := map[string]struct {
		left, right string
		wantMatch   bool
		wantChanged bool
	}{
		"two equal current hashes":     {first, first, true, false},
		"two different current hashes": {first, second, false, true},
		"the different pair reversed":  {second, first, false, true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.ExportExchangeHashesMatch(tc.left, tc.right)).To(Equal(tc.wantMatch), "the equality check")
			g.Expect(cli.ExportExchangeHashChanged(tc.left, tc.right)).To(Equal(tc.wantChanged), "the change check")
		})
	}
}

// Test792b33c7a2_SetXIDField_AnExistingXIDIsKept verifies 792b33c7a2 (An existing xid is kept): stamping another
// xid onto a note that already has one returns the note unchanged, whether the xid sits among other keys or ahead
// of the exchange blocks.
func Test792b33c7a2_SetXIDField_AnExistingXIDIsKept(t *testing.T) {
	t.Parallel()

	notes := map[string]string{
		"xid ahead of aliases": "---\ntype: fact\nxid: " + xidB + "\naliases:\n  - 0.old\n---\n\nbody\n",
		"xid after the body keys": "---\ntype: fact\nsituation: s\nxid: " + xidB + "\nparent:\n    vault: " +
			parentVaultID + "\n---\n\nbody\n",
		"xid only": "---\nxid: " + xidB + "\n---\n",
	}

	for name, note := range notes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			stamped, err := cli.ExportSetXIDField(note, xidC)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(stamped).To(Equal(note), "the note is unchanged")
			g.Expect(stamped).NotTo(ContainSubstring(xidC), "the other xid is not written")
		})
	}
}

// Test792b33c7a3_SetXIDField_ANoteWithNoFrontmatterCannotBeStamped verifies 792b33c7a3 (A note with no frontmatter
// cannot be stamped): stamping a note with no frontmatter fails and returns no content.
func Test792b33c7a3_SetXIDField_ANoteWithNoFrontmatterCannotBeStamped(t *testing.T) {
	t.Parallel()

	for name, note := range ev1NotesWithNoFrontmatter() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			stamped, err := cli.ExportSetXIDField(note, xidB)

			g.Expect(err).To(MatchError(ContainSubstring("no frontmatter")))
			g.Expect(stamped).To(BeEmpty(), "no content is returned to write")
		})
	}
}

// Test792b33c9a6_ApplyReceiptToContent_ANoteWithNoFrontmatterCannotTakeAReceipt verifies 792b33c9a6 (A note with
// no frontmatter cannot take a receipt): recording a receipt on a note with no frontmatter fails and returns no
// content, so nothing is written.
func Test792b33c9a6_ApplyReceiptToContent_ANoteWithNoFrontmatterCannotTakeAReceipt(t *testing.T) {
	t.Parallel()

	for name, note := range ev1NotesWithNoFrontmatter() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			updated, err := cli.ExportApplyReceiptToContent([]byte(note), testReceipt("1.2026-09-27.x", ""))

			g.Expect(err).To(MatchError(ContainSubstring("no frontmatter")))
			g.Expect(updated).To(BeEmpty(), "no content is returned to write")
		})
	}
}

// Test792b33c9a7_DrainOutbox_ACRLFNoteTakesTheReceiptAsLF verifies 792b33c9a7 (A CRLF note takes the receipt as
// LF): a CRLF note queued for an offer is sent (the send step is called once, for its xid, and is handed the CRLF
// bytes), and the receipt that follows is recorded by a write that converts the note to LF; the entry is removed.
func Test792b33c9a7_DrainOutbox_ACRLFNoteTakesTheReceiptAsLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	crlf := toCRLF(plainParentNote("body three"))
	env.fsys.put(attentionNotePath(env), []byte(crlf))
	env.enqueue(xidA)

	var receipts []cli.OfferReceiptForTest

	apply := func(note cli.OfferNoteForTest, receipt cli.OfferReceiptForTest) error {
		receipts = append(receipts, receipt)

		updated, applyErr := cli.ExportApplyReceiptToContent(note.Raw, receipt)
		if applyErr != nil {
			return applyErr
		}

		env.fsys.put(note.Path, []byte(updated))

		return nil
	}

	send, sendCalls := MockOo1Send(t)
	done := oo1StartDrain(env, send, apply)

	// the expectation runs apart, so that a call that does not match it cannot hold the test
	sent := make(chan cli.OfferNoteForTest, 1)

	go func() {
		call := sendCalls.ArgsShould(match.BeAny, HaveField("XID", xidA))
		sent <- call.GetArgs().Note

		call.Return(accepted("9.2026-09-27.parent-copy"))
	}()

	outcome := oo1AwaitDrain(t, done)

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.result.Sent).To(Equal(1), "the offer was sent once")

	select {
	case note := <-sent:
		g.Expect(string(note.Raw)).To(Equal(crlf), "the send step was handed the note as it lay, in CRLF")
	case <-time.After(mockTimeout):
		t.Fatal("the offer was not sent")
	}

	g.Expect(receipts).To(HaveLen(1), "one receipt is recorded")

	written := string(env.noteRaw(attentionNoteName))
	g.Expect(written).To(ContainSubstring("note: 9.2026-09-27.parent-copy"), "the receipt is recorded")
	g.Expect(written).NotTo(ContainSubstring("\r"), "the note is written with no CRLF")
	g.Expect(written).To(ContainSubstring("body three"), "the body is kept")
	g.Expect(env.outbox().Entries).To(BeEmpty(), "the outbox entry is removed")
}

// ev1NotesWithNoFrontmatter is the notes that have no frontmatter: plain text, an empty note, and a leading
// delimiter that never closes (792b33c3a5 An unterminated frontmatter is the whole text as body).
func ev1NotesWithNoFrontmatter() map[string]string {
	return map[string]string{
		"plain text":               "body only\n",
		"empty note":               "",
		"unterminated frontmatter": "---\ntype: fact\nsituation: s\n",
	}
}
