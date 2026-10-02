package cli_test

// Exchange readers read notes as LF (#789 follow-up, design D4): a note
// with CRLF line endings takes part in exchange as its LF form does — show
// prints its exchange hash, the outbox finds it, a pending CRLF offer is
// still a pending offer, and a CRLF envelope pulls down.

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestActivate_PullsCRLFParentNote: a parent note served with CRLF line
// endings pulls down as an LF pending copy linked at the parent's hash.
func TestActivate_PullsCRLFParentNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := pulledFact("7.2026-09-01.pulled", "c", "")
	note.content = toCRLF(note.content)

	env := newWiringEnv(t)
	env.parent.addNote(note)
	hash := env.parent.hashOf(t, note.basename)

	_, stderr := env.run("activate", "--note", note.basename+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1), stderr)

	if len(files) != 1 {
		return
	}

	raw := []byte(readFileString(t, filepath.Join(env.vault, files[0])))
	g.Expect(string(raw)).NotTo(ContainSubstring("\r\n"))

	copied := decodePulledCopy(t, raw)
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: note.basename, Via: "pulled", Hash: hash}}))
}

// TestDrainOutbox_FindsCRLFNote: a queued note with CRLF line endings is
// sent, and its receipt is recorded (writing it as LF), instead of the
// entry being dropped as if the note were gone.
func TestDrainOutbox_FindsCRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.fsys.put(attentionNotePath(env), []byte(toCRLF(plainParentNote("body one"))))
	env.enqueue(xidA)

	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: accepted("9.2026-09-27.parent-copy")}}
	err := env.drainApplying(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA}))
	g.Expect(env.outbox().Entries).To(BeEmpty())

	written := string(env.noteRaw(attentionNoteName))
	g.Expect(written).To(ContainSubstring("note: 9.2026-09-27.parent-copy"))
	g.Expect(written).NotTo(ContainSubstring("\r\n"), "the receipt write converts the note")
}

// TestExchangeHash_CRLFFrontmatterHashesAsLF: a note whose frontmatter is
// CRLF hashes exactly as its LF form.
func TestExchangeHash_CRLFFrontmatterHashesAsLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lfNote := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)

	lfHash, lfErr := cli.ExportExchangeHash([]byte(lfNote))
	g.Expect(lfErr).NotTo(HaveOccurred())

	crlfHash, crlfErr := cli.ExportExchangeHash([]byte(toCRLF(lfNote)))
	g.Expect(crlfErr).NotTo(HaveOccurred())
	g.Expect(crlfHash).To(Equal(lfHash))
}

// TestNoteHasPendingMarker_CRLF: a pending offer with CRLF line endings is
// still a pending offer, so query excludes it and serve never hands it out
// as live.
func TestNoteHasPendingMarker_CRLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportNoteHasPendingMarker([]byte(toCRLF(foldNote(foldExchange{xid: foldXIDO}, true))))).To(BeTrue())
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(toCRLF(foldNote(foldExchange{xid: foldXIDO}, false))))).To(BeFalse())
}

// TestShow_ExchangeHashHeaderForCRLFNote: show prints the exchange hash
// line for a note with CRLF frontmatter, and that hash passes amend's
// --expect-hash check on the same note.
func TestShow_ExchangeHashHeaderForCRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	crlf := toCRLF(foldNote(foldExchange{xid: foldXIDO, origin: true}, true))

	memFS := newInMemoryFS()
	memFS.files["/vault/"+foldOfferName] = []byte(crlf)

	var shown bytes.Buffer

	g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: "5", VaultPath: "/vault"}, newShowDeps(memFS), &shown)).
		To(Succeed())

	header, _, _ := strings.Cut(shown.String(), "\n")
	g.Expect(header).To(HavePrefix("# exchange_hash: xh1:"))
	g.Expect(strings.TrimPrefix(shown.String(), header+"\n")).To(HavePrefix(crlf), "the body is printed verbatim")

	vault := newCRLFAmendVault(crlf)
	notPending := false
	err := vault.amend(t.Context(), cli.AmendArgs{
		Pending: &notPending, ExpectHash: strings.TrimPrefix(header, "# exchange_hash: "),
	})
	g.Expect(err).NotTo(HaveOccurred())
}
