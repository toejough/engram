package cli_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestBuildOfferPayload_UnbuildableNotes: a note whose basename has no
// slug, or whose frontmatter is missing or corrupt, is not built (the
// drain skips it without contacting the parent).
func TestBuildOfferPayload_UnbuildableNotes(t *testing.T) {
	t.Parallel()

	good := offerTestNote{}.render(t)
	cases := map[string]cli.OfferNoteForTest{
		"no slug":        {XID: xidA, Basename: "weird", Raw: good},
		"no frontmatter": {XID: xidA, Basename: "1.2026-09-27.x", Raw: []byte("just text\n")},
		"corrupt yaml":   {XID: xidA, Basename: "1.2026-09-27.x", Raw: []byte("---\ntype: [\n---\n\nbody\n")},
	}

	for name, note := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "", "bob", nil)
			g.Expect(err).To(HaveOccurred())
		})
	}
}

// TestClassifyOffer_NonExchangeNotesAreNotOffered: a note without
// frontmatter, of another type, or with corrupt frontmatter is not offered.
func TestClassifyOffer_NonExchangeNotesAreNotOffered(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"no frontmatter": "plain text\n",
		"qa note":        "---\ntype: question\nsituation: s\n---\n\nQ?\n",
		"corrupt yaml":   "---\ntype: [\n---\n\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.ExportClassifyOffer(cli.ExportOfferCmdLearn, cli.AmendArgs{}, []byte(raw), "")).To(BeFalse())
		})
	}
}

// TestNewPayloadBuilder_LocalReadFailures: an unreadable vault ID or an
// unreadable note in the supersedes scan fails the build.
func TestNewPayloadBuilder_LocalReadFailures(t *testing.T) {
	t.Parallel()

	note := payloadNote(t, offerTestNote{})

	t.Run("corrupt vault id", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte("bad\n"), 0o600)).To(Succeed())

		_, err := cli.ExportNewPayloadBuilder(newTestDeps(nil, nil), vault, parentURL)(note)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("unreadable note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte(seqID(60)+"\n"), 0o600)).To(Succeed())
		unreadable := filepath.Join(vault, "2.2026-09-27.locked.md")
		g.Expect(os.WriteFile(unreadable, []byte("---\ntype: fact\n---\n"), 0o000)).To(Succeed())

		_, err := cli.ExportNewPayloadBuilder(newTestDeps(nil, nil), vault, parentURL)(note)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("builds once ready", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte(seqID(60)+"\n"), 0o600)).To(Succeed())

		build := cli.ExportNewPayloadBuilder(newTestDeps(nil, nil), vault, parentURL)

		for range 2 {
			body, err := build(note)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(string(body)).To(ContainSubstring(seqID(60) + ":" + xidA))
		}
	})
}

// TestOfferHooks_FailuresNeverFailTheWrite: a staging or queueing failure
// is warned; the write proceeds unstaged and unqueued.
func TestOfferHooks_FailuresNeverFailTheWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var warnings []string

	warn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }
	hooks := cli.ExportNewOfferHooks(
		func(string, cli.OfferWriteForTest) ([]byte, string, error) { return nil, "", errors.New("stage boom") },
		func(string, string) error { return errors.New("enqueue boom") },
		warn,
	)

	content, xid := cli.ExportOfferHooksStageWrite(hooks, "/vault", []byte("raw"))
	g.Expect(string(content)).To(Equal("raw"))
	g.Expect(xid).To(BeEmpty())

	g.Expect(cli.ExportOfferHooksQueue(hooks, "/vault", xidA)).To(BeFalse())
	g.Expect(cli.ExportOfferHooksQueue(hooks, "/vault", "")).To(BeFalse())
	g.Expect(strings.Join(warnings, "\n")).To(And(ContainSubstring("stage boom"), ContainSubstring("enqueue boom")))

	var disabled cli.OfferHooksForTest

	disabledContent, disabledXID := cli.ExportOfferHooksStageWrite(disabled, "/vault", []byte("raw"))
	g.Expect(string(disabledContent)).To(Equal("raw"))
	g.Expect(disabledXID).To(BeEmpty())
	g.Expect(cli.ExportOfferHooksQueue(disabled, "/vault", xidA)).To(BeFalse())
}

// TestSetXIDField: the stamp goes before parent/aliases/offer, an existing
// xid is kept, and a note without frontmatter is an error.
func TestSetXIDField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stamped, err := cli.ExportSetXIDField("---\ntype: fact\naliases:\n  - 0.old\n---\n\nbody\n", xidB)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal("---\ntype: fact\nxid: " + xidB + "\naliases:\n  - 0.old\n---\n\nbody\n"))

	kept, keptErr := cli.ExportSetXIDField(stamped, xidC)
	g.Expect(keptErr).NotTo(HaveOccurred())
	g.Expect(kept).To(Equal(stamped))

	_, noFrontmatterErr := cli.ExportSetXIDField("body only\n", xidB)
	g.Expect(noFrontmatterErr).To(HaveOccurred())

	_, receiptErr := cli.ExportApplyReceiptToContent([]byte("body only\n"), testReceipt("1.x", ""))
	g.Expect(receiptErr).To(HaveOccurred())
}
