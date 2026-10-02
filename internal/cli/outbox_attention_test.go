package cli_test

// A refused receipt needs attention (#789 defect 3, design D3; spec
// vault-parent-offers "A receipt SHALL record the parent counterpart on the
// local note" and "Outbox state SHALL be reported"): the entry stops being
// re-sent, is reported, warns once, and resumes on its own once the note
// can take the receipt.

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

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

	_, err := env.drainApplying(&fakeParent{
		script: map[string]cli.OfferSendResultForTest{xidA: accepted("9.2026-09-27.parent-copy")},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(env.outbox().Entries[0].State).To(Equal("attention"))

	env.fsys.put(attentionNotePath(env), []byte(plainParentNote("body two")))

	parent := &fakeParent{}
	_, resumeErr := env.drainApplying(parent)
	g.Expect(resumeErr).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(BeEmpty(), "the kept receipt is recorded without contacting the parent")
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA}))
	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(box.Entries[0].Receipt).To(BeNil())

	next := &fakeParent{}
	_, nextErr := env.drainApplying(next)
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

	parent := &fakeParent{applyErr: errors.New("disk full")}
	_, err := env.drain(parent)
	g.Expect(err).To(MatchError(ContainSubstring("disk full")))
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(parent.applied).To(Equal([]string{"1.2026-09-27.a"}))

	box := env.outbox()
	g.Expect(box.Entries[0].State).To(Equal("attention"))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("disk full"))
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
	_, err := env.drainApplying(first)
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
		_, againErr := env.drainApplying(again)
		g.Expect(againErr).NotTo(HaveOccurred())
		g.Expect(again.sent).To(BeEmpty(), "an attention entry is never re-sent")
		g.Expect(env.outbox().Entries[0].State).To(Equal("attention"))
	}

	g.Expect(env.stderr.String()).To(Equal(warning), "no further warning")

	env.fsys.put(attentionNotePath(env), []byte(plainParentNote("body one")))

	resumed := &fakeParent{}
	_, resumeErr := env.drainApplying(resumed)
	g.Expect(resumeErr).NotTo(HaveOccurred())
	g.Expect(resumed.sent).To(BeEmpty())
	g.Expect(env.outbox().Entries).To(BeEmpty(), "the receipt is recorded and the entry is done")
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))
	g.Expect(string(env.noteRaw(attentionNoteName))).To(ContainSubstring("via: offered"))
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
func (e *outboxEnv) drainApplying(parent *fakeParent) (cli.DrainResultForTest, error) {
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

	return cli.ExportDrainOutbox(context.Background(), e.store, e.vault, parentURL, parent.send, apply)
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

// plainParentNote is anchoredParentNote with the anchor and its alias
// removed.
func plainParentNote(body string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nxid: " + xidA + "\n" +
		"parent:\n    vault: " + strings.Repeat("cd", 16) + "\n---\n\n" + body + "\n"
}
