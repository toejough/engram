package cli_test

// A refused receipt needs attention (#789 defect 3, design D3; spec
// vault-parent-offers "A receipt SHALL record the parent counterpart on the
// local note" and "Outbox state SHALL be reported"): the entry stops being
// re-sent, is reported, warns once, and resumes on its own once the note
// can take the receipt.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestDrainForCommand_KeptReceiptFromAnotherParentDiscarded: a kept receipt
// whose vault ID is not the configured parent's (the parent was
// reconfigured, or reports another vault) is never recorded: it is
// discarded with a warning and the entry is queued for the current parent.
func TestDrainForCommand_KeptReceiptFromAnotherParentDiscarded(t *testing.T) {
	t.Parallel()

	for name, cache := range map[string]struct{ url, id string }{
		"parent URL reconfigured":      {"http://old-parent:9", seqID(91)},
		"parent reports another vault": {parentURL, seqID(92)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env, path := heldReceiptWiringEnv(t, cache.url, cache.id, seqID(91))

			env.learnFact("another")
			g.Expect(readFileString(t, path)).NotTo(ContainSubstring("parent-copy"), "the stale receipt is not recorded")
			g.Expect(env.lastStderr).To(ContainSubstring("discarding the kept receipt"))

			states := map[string]any{}
			for _, raw := range env.outboxEntries() {
				entry, isMap := raw.(map[string]any)
				g.Expect(isMap).To(BeTrue())

				xid, isString := entry["xid"].(string)
				g.Expect(isString).To(BeTrue())

				states[xid] = entry["state"]
				g.Expect(entry).NotTo(HaveKey("receipt"))
			}

			g.Expect(states).To(HaveKeyWithValue(xidA, "queued"))
		})
	}
}

// TestDrainForCommand_KeptReceiptRecordedInsideBackoff: a fixed note's
// kept receipt is recorded by the next command's drain step even while the
// parent is backed off, with no request to the parent (it needs none).
func TestDrainForCommand_KeptReceiptRecordedInsideBackoff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env, path := heldReceiptWiringEnv(t, parentURL, seqID(91), seqID(91))

	env.learnFact("another")
	g.Expect(env.parent.requests()).To(BeEmpty(), "inside the backoff window nothing contacts the parent")
	g.Expect(readFileString(t, path)).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))
	g.Expect(env.outboxEntries()).To(HaveLen(1), "only the new note's entry is left")
}

// TestDrainOutbox_AttentionEntryWithoutReceiptIsSent: an attention entry
// with no kept receipt (an older binary saved the outbox and dropped it) is
// sent like a queued one.
func TestDrainOutbox_AttentionEntryWithoutReceiptIsSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{
		{"xid": xidA, "queued": testNow, "attempts": 1, "state": "attention", "last_error": "old"},
	}})

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA}))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_AttentionResumesAndRequeuesAChangedNote: when the anchor
// is removed and the note's content also changed since it was sent, the
// kept receipt is recorded without a send and the entry is queued again,
// so the next drain offers the new content.
func TestDrainOutbox_AttentionResumesAndRequeuesAChangedNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.fsys.put(attentionNotePath(env), []byte(anchoredParentNote("body one")))
	env.enqueue(xidA)

	err := env.drainApplying(&fakeParent{
		script: map[string]cli.OfferSendResultForTest{xidA: accepted("9.2026-09-27.parent-copy")},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(env.outbox().Entries[0].State).To(Equal("attention"))

	env.fsys.put(attentionNotePath(env), []byte(plainParentNote("body two")))

	// Recording the kept receipt is local; the entry is then queued again
	// because the note changed since it was sent.
	g.Expect(cli.ExportRecordKeptReceipts(env.store, env.vault, parentURL, parentFor(env).apply)).To(Succeed())
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA}))
	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(box.Entries[0].Receipt).To(BeNil())

	next := &fakeParent{}
	nextErr := env.drainApplying(next)
	g.Expect(nextErr).NotTo(HaveOccurred())
	g.Expect(sentXIDs(next)).To(Equal([]string{xidA}), "the changed note is offered again")
}

// TestDrainOutbox_AttentionRetryFailureIsReported: when retrying a kept
// receipt fails for a reason other than the note refusing it (a write
// error), the drain reports it, records it, and the entry keeps waiting in
// the attention state without a send.
func TestDrainOutbox_AttentionRetryFailureIsReported(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.fsys.put(attentionNotePath(env), []byte(plainParentNote("body one")))
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{{
		"xid": xidA, "queued": testNow, "attempts": 1, "state": "attention", "last_error": "anchored",
		"receipt":   accepted("9.2026-09-27.parent-copy").Receipt,
		"sent_hash": mustHash(t, []byte(plainParentNote("body one"))),
	}}})

	env.fsys.put(filepath.Join(env.vault, ".engram", "parent.json"),
		[]byte(`{"url":"`+parentURL+`","vault_id":"`+parentVaultID+`","failures":0}`))

	parent := &fakeParent{applyErr: errors.New("disk full")}
	err := cli.ExportRecordKeptReceipts(env.store, env.vault, parentURL, parent.apply)
	g.Expect(err).To(MatchError(ContainSubstring("disk full")))
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(parent.applied).To(Equal([]string{"1.2026-09-27.a"}))

	box := env.outbox()
	g.Expect(box.Entries[0].State).To(Equal("attention"))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("disk full"))
}

// TestDrainOutbox_NoUserEntryRequeuedOnceBuilt: an entry waiting in
// attention for a user identity goes back to queued as soon as its payload
// builds, so a network failure on that send keeps it queued (counted, and
// not listed as needing attention).
func TestDrainOutbox_NoUserEntryRequeuedOnceBuilt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{{
		"xid": xidA, "queued": testNow, "attempts": 0, "state": "attention",
		"last_error": "cannot offer: no user identity detected; set git user.email",
	}}})

	down := cli.ExportClassifyOfferResponse(cli.FetchResponse{}, errors.New("down"))
	_, err := env.drain(&fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: down}})
	g.Expect(err).NotTo(HaveOccurred())

	box := env.outbox()
	g.Expect(box.Entries).To(HaveLen(1))
	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(cli.ExportQueuedOfferCount(box)).To(Equal(1))
}

// TestDrainOutbox_RefusedReceiptNeedsAttention: the parent accepts, but the
// note's anchored parent: refuses the receipt. The note is untouched, the
// entry moves to attention with the receipt, the sent hash and the reason,
// and one warning names the note. Later drains send nothing and stay
// silent; once the anchor is removed, the next drain records the kept
// receipt without a send and the entry is done.
func TestDrainOutbox_RefusedReceiptNeedsAttention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	anchored := []byte(anchoredParentNote("body one"))
	env.fsys.put(attentionNotePath(env), anchored)
	env.enqueue(xidA)

	first := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: accepted("9.2026-09-27.parent-copy")}}
	err := env.drainApplying(first)
	g.Expect(err).NotTo(HaveOccurred(), "the refusal is warned, not returned as a drain error")
	g.Expect(sentXIDs(first)).To(Equal([]string{xidA}))
	g.Expect(env.noteRaw(attentionNoteName)).To(Equal(anchored), "a refused receipt leaves the note untouched")

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA}))
	g.Expect(box.Entries[0].State).To(Equal("attention"))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("YAML anchor: parent"))
	g.Expect(box.Entries[0].SentHash).To(Equal(mustHash(t, anchored)))
	g.Expect(box.Entries[0].Receipt).NotTo(BeNil())

	warning := env.stderr.String()
	g.Expect(strings.Count(warning, "\n")).To(Equal(1), "exactly one warning line")
	g.Expect(warning).To(ContainSubstring(strings.TrimSuffix(attentionNoteName, ".md")))
	g.Expect(warning).To(ContainSubstring("YAML anchor: parent"))

	for range 2 {
		again := &fakeParent{}
		againErr := env.drainApplying(again)
		g.Expect(againErr).NotTo(HaveOccurred())
		g.Expect(again.sent).To(BeEmpty(), "an attention entry is never re-sent")
		g.Expect(env.outbox().Entries[0].State).To(Equal("attention"))
	}

	g.Expect(env.stderr.String()).To(Equal(warning), "no further warning")

	env.fsys.put(attentionNotePath(env), []byte(plainParentNote("body one")))

	resumed := &fakeParent{}
	resumeErr := env.drainApplying(resumed)
	g.Expect(resumeErr).NotTo(HaveOccurred())
	g.Expect(resumed.sent).To(BeEmpty())
	g.Expect(env.outbox().Entries).To(BeEmpty(), "the receipt is recorded and the entry is done")
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("via: offered"))
}

// TestRecordKeptReceipts_DropsEntryOfDeletedNote: a kept receipt whose
// note is gone is dropped with its entry, as any entry of a deleted note.
func TestRecordKeptReceipts_DropsEntryOfDeletedNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{{
		"xid": xidA, "queued": testNow, "attempts": 1, "state": "attention", "last_error": "anchored",
		"receipt": accepted("9.2026-09-27.parent-copy").Receipt, "sent_hash": "xh1:gone",
	}}})

	g.Expect(cli.ExportRecordKeptReceipts(env.store, env.vault, parentURL, (&fakeParent{}).apply)).To(Succeed())
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestRecordKeptReceipts_ScanFailureSurfaces: a vault that cannot be
// listed fails the local pass.
func TestRecordKeptReceipts_ScanFailureSurfaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{{
		"xid": xidA, "queued": testNow, "attempts": 1, "state": "attention", "last_error": "anchored",
		"receipt": accepted("9.2026-09-27.parent-copy").Receipt, "sent_hash": "xh1:x",
	}}})
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.fsys.failReadOf(filepath.Join(env.vault, "1.2026-09-27.a.md"), errors.New("disk gone"))

	err := cli.ExportRecordKeptReceipts(env.store, env.vault, parentURL, (&fakeParent{}).apply)
	g.Expect(err).To(MatchError(ContainSubstring("disk gone")))
}

// TestUpdateExchange_ReportsEntryNeedingAttention: through the production
// wiring, a receipt the note refuses puts its entry in attention; update's
// notice counts and lists it, the parent is not offered it again, and once
// the anchor is removed the next update records the receipt and goes idle.
func TestUpdateExchange_ReportsEntryNeedingAttention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)
	path := env.learnFact("anchored")
	original := readFileString(t, path)
	env.plant(filepath.Base(path), []byte(strings.Replace(original, "\n---\n",
		"\nparent: &p\n    vault: "+strings.Repeat("ab", 16)+"\nparent_copy: *p\n---\n", 1)))
	env.parent.setDown(false)

	var stderr strings.Builder

	deps := newTestDeps(&bytes.Buffer{}, &stderr)
	env.customize(&deps)
	exchange := cli.ExportUpdateExchange(deps)

	notice := exchange(context.Background(), env.vault, false)
	g.Expect(notice).To(ContainSubstring("0 offer(s) queued, 0 rejected, 1 need attention"))
	g.Expect(notice).To(MatchRegexp(`needs attention: \S+\.anchored: .*YAML anchor: parent`))
	g.Expect(stderr.String()).To(ContainSubstring(noteBasename(path)))
	g.Expect(env.parent.offers()).To(HaveLen(1))

	warned := stderr.String()
	g.Expect(exchange(context.Background(), env.vault, false)).To(ContainSubstring("1 need attention"))
	g.Expect(env.parent.offers()).To(HaveLen(1), "the attention entry is not offered again")
	g.Expect(stderr.String()).To(Equal(warned), "the warning is printed once")

	env.plant(filepath.Base(path), []byte(original))
	g.Expect(exchange(context.Background(), env.vault, false)).To(BeEmpty())
	g.Expect(env.parent.offers()).To(HaveLen(1))
	g.Expect(readFileString(t, path)).To(ContainSubstring("via: offered"))
}

// unexported constants.
const (
	attentionNoteName = "1.2026-09-27.a.md"
)

// drainApplying drains with a receipt applier that runs the real
// applyReceiptToContent on the in-memory note and writes the result.
func (e *outboxEnv) drainApplying(parent *fakeParent) error {
	apply := func(note cli.OfferNoteForTest, receipt cli.OfferReceiptForTest) error {
		parent.applied = append(parent.applied, note.Basename)

		updated, applyErr := cli.ExportApplyReceiptToContent(note.Raw, receipt)
		if applyErr != nil {
			return applyErr
		}

		if updated != string(note.Raw) {
			e.fsys.put(note.Path, []byte(updated))
		}

		return nil
	}

	// As drainForCommand does: kept receipts first, locally, then the drain.
	keptErr := cli.ExportRecordKeptReceipts(e.store, e.vault, parentURL, apply)
	if keptErr != nil {
		return keptErr
	}

	_, drainErr := cli.ExportDrainOutbox(context.Background(), e.store, e.vault, parentURL, parent.send, apply)

	return drainErr
}

// anchoredParentNote is a fact note queued as xidA whose parent: carries an
// anchor another key aliases, so a receipt cannot be recorded on it.
func anchoredParentNote(body string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nxid: " + xidA + "\n" +
		"parent: &p\n    vault: " + strings.Repeat("cd", 16) + "\nparent_copy: *p\n---\n\n" + body + "\n"
}

func attentionNotePath(env *outboxEnv) string {
	return filepath.Join(env.vault, attentionNoteName)
}

// heldReceiptWiringEnv is a wiring env whose vault holds a fixed note (xidA)
// with an attention entry keeping a receipt from receiptVault, and a parent
// cache for cacheURL/cacheID inside a backoff window.
func heldReceiptWiringEnv(t *testing.T, cacheURL, cacheID, receiptVault string) (*wiringEnv, string) {
	t.Helper()

	env := newWiringEnv(t)
	env.parent.setDown(true)
	env.stampVault("")

	note := plainParentNote("body one")
	env.plant(attentionNoteName, []byte(note))

	sent, err := cli.ExportExchangeHash([]byte(note))
	if err != nil {
		t.Fatal(err)
	}

	receipt := accepted("9.2026-09-27.parent-copy").Receipt
	receipt.VaultID = receiptVault

	box := map[string]any{"version": 1, "entries": []map[string]any{{
		"xid": xidA, "queued": env.now(), "attempts": 1, "state": "attention", "last_error": "anchored",
		"receipt": receipt, "sent_hash": sent,
	}}}

	encoded, marshalErr := json.Marshal(box)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}

	env.writeState("outbox.json", string(encoded))
	env.writeState("parent.json", `{"url":"`+cacheURL+`","vault_id":"`+cacheID+
		`","backoff_until":"2999-01-01T00:00:00Z","failures":3}`)

	return env, filepath.Join(env.vault, attentionNoteName)
}

// parentFor is a fake parent whose receipt applier runs the real
// applyReceiptToContent on env's in-memory notes (as drainApplying does).
func parentFor(env *outboxEnv) *fakeParent {
	return &fakeParent{applyFn: func(note cli.OfferNoteForTest, receipt cli.OfferReceiptForTest) error {
		updated, applyErr := cli.ExportApplyReceiptToContent(note.Raw, receipt)
		if applyErr != nil {
			return applyErr
		}

		env.fsys.put(note.Path, []byte(updated))

		return nil
	}}
}

// plainParentNote is anchoredParentNote with the anchor and its alias
// removed.
func plainParentNote(body string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nxid: " + xidA + "\n" +
		"parent:\n    vault: " + strings.Repeat("cd", 16) + "\n---\n\n" + body + "\n"
}
