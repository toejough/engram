package cli_test

// Fold and receipt parity with the pre-change line-based rewrite (ruling
// V7): moving `amend --discard --into` and the offer receipt onto YAML-node
// edits must write the same bytes, and so the same exchange hash, for notes
// without unknown keys or anchors. The goldens under
// testdata/fold_receipt_parity were written by the pre-change code
// (foldedContent and applyReceiptToContent are unchanged from 49cfc120 up
// to the commit before this one); the fold goldens were also cross-checked
// against a binary built from 49cfc120.

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestFoldAndReceipt_ParityWithPreChangeRewrite: every fold and receipt
// case writes exactly the pre-change bytes and exchange hash; a fold also
// still deletes the offer and never re-stamps identity (ruling S7).
func TestFoldAndReceipt_ParityWithPreChangeRewrite(t *testing.T) {
	t.Parallel()

	for _, tc := range foldParityCases() {
		t.Run("fold-"+tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, vault := runFoldParityCase(t, tc)
			g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
			g.Expect(vault.detected).To(BeZero(), "a fold never runs identity detection (S7)")
			expectParityGolden(g, "fold-"+tc.name, got)
		})
	}

	for _, tc := range receiptParityCases() {
		t.Run("receipt-"+tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, err := cli.ExportApplyReceiptToContent([]byte(tc.note), tc.receipt)
			g.Expect(err).NotTo(HaveOccurred())
			expectParityGolden(g, "receipt-"+tc.name, got)
		})
	}
}

// foldParityCase is one fold of an offer into an existing note.
type foldParityCase struct {
	name            string
	existing, offer foldExchange
}

// receiptParityCase is one receipt applied to a note.
type receiptParityCase struct {
	name    string
	note    string
	receipt cli.OfferReceiptForTest
}

// expectParityGolden asserts got equals the named golden byte-for-byte and
// in exchange hash.
func expectParityGolden(g Gomega, name, got string) {
	want, readErr := os.ReadFile(filepath.Join("testdata", "fold_receipt_parity", name+".md"))
	g.Expect(readErr).NotTo(HaveOccurred())

	gotHash, _ := cli.ExportExchangeHash([]byte(got))
	wantHash, _ := cli.ExportExchangeHash(want)
	g.Expect(gotHash).To(Equal(wantHash), "exchange hash moved")
	g.Expect(got).To(Equal(string(want)), "a note without unknown keys must be written byte-for-byte as before")
}

// foldParityCases covers the fold's branches: into with no parent, with a
// primary, with only covered links, under another vault; aliases new,
// overlapping and absent.
func foldParityCases() []foldParityCase {
	pulled := []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}, {"8.2026-09-01.near", "covered", "xh1:near"}}

	return []foldParityCase{
		{"into-bare", foldExchange{}, foldExchange{xid: foldXIDO, vault: foldParentVault, links: pulled}},
		{"into-primary", foldExchange{
			xid: foldXIDE, vault: foldParentVault, aliases: []string{"2.2026-01-01.old-name"},
			links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
		}, foldExchange{
			xid: foldXIDO, vault: foldParentVault, aliases: []string{"5.2026-01-01.prev-name"}, links: pulled,
		}},
		{"into-covered-promoted", foldExchange{
			xid: foldXIDE, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "covered", "xh1:old"}},
		}, foldExchange{xid: foldXIDO, vault: foldParentVault, links: pulled}},
		{"into-other-vault", foldExchange{
			xid: foldXIDE, vault: foldOtherVault, links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
		}, foldExchange{xid: foldXIDO, vault: foldParentVault, links: pulled}},
		{"aliases-only", foldExchange{xid: foldXIDE, aliases: []string{foldOfferBase}},
			foldExchange{xid: foldXIDO, aliases: []string{"6.2026-01-01.s"}}},
		{"nothing-new", foldExchange{xid: foldXIDE, aliases: []string{foldOfferBase}}, foldExchange{xid: foldXIDO}},
	}
}

// receiptParityCases covers the receipt's branches on notes without and
// with parent links: first link, same vault replacing the primary and
// keeping covered, another vault starting over, and a receipt naming a
// for target.
func receiptParityCases() []receiptParityCase {
	plain := foldNote(foldExchange{xid: foldXIDE}, true)
	withAliases := foldNote(foldExchange{xid: foldXIDE, aliases: []string{"0.old"}, origin: true}, true)
	sameVault := foldNote(foldExchange{xid: foldXIDE, vault: parentVaultID, links: []foldLink{
		{"1.2026-09-01.prev", "offered", "xh1:prev"}, {"3.2026-09-01.cov", "covered", "xh1:cov"},
	}}, true)
	otherVault := foldNote(foldExchange{xid: foldXIDE, vault: foldOtherVault, links: []foldLink{
		{"1.2026-09-01.prev", "pulled", "xh1:prev"},
	}}, false)

	return []receiptParityCase{
		{"first-link", plain, testReceipt("1100.2026-09-27.x", "")},
		{"first-link-before-aliases", withAliases, testReceipt("1100.2026-09-27.x", "")},
		{"same-vault", sameVault, testReceipt("1100.2026-09-27.x", "")},
		{"same-vault-for", sameVault, testReceipt("1100.2026-09-27.x", "3.2026-09-01.cov")},
		{"other-vault", otherVault, testReceipt("1100.2026-09-27.x", "")},
	}
}

// runFoldParityCase folds tc.offer into tc.existing and returns E.
func runFoldParityCase(t *testing.T, tc foldParityCase) (string, *foldVault) {
	t.Helper()

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(tc.existing, false))
	vault.put(foldOfferName, foldNote(tc.offer, true))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	if err != nil {
		t.Fatalf("fold %s: %v", tc.name, err)
	}

	return string(vault.files[foldExistingName]), vault
}
