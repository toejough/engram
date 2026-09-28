package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestApplyReceiptToContent_FrontmatterOnlyProperty: for any note and any
// receipt, the rewrite (1) leaves the exchange hash, identity, pending,
// xid, aliases and offer untouched, (2) makes the receipt's target the
// one primary link with the stored hash, and (3) is idempotent.
func TestApplyReceiptToContent_FrontmatterOnlyProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := drawOfferTestNote(rt)
		if rapid.Bool().Draw(rt, "linked") {
			note.parentVault = rapid.SampledFrom([]string{parentVaultID, seqID(77)}).Draw(rt, "vault")
			note.linkVia = rapid.SampledFrom([]string{"offered", "pulled", "covered"}).Draw(rt, "via")
			note.linkHash = "xh1:l"
		}

		raw := note.render(rt)
		if rapid.Bool().Draw(rt, "aliases") {
			raw = []byte(strings.Replace(string(raw), "---\n\n", "aliases:\n  - 0.old\n---\n\n", 1))
		}

		basename := rapid.StringMatching(`[1-9][0-9]{0,3}\.2026-09-27\.[a-z]{1,6}`).Draw(rt, "basename")
		target := rapid.SampledFrom([]string{"", "900.2026-09-01.e"}).Draw(rt, "for")
		receipt := testReceipt(basename, target)
		receipt.StoredHash = "xh1:" + rapid.StringMatching(`[0-9a-f]{8}`).Draw(rt, "stored")

		updated, err := cli.ExportApplyReceiptToContent(raw, receipt)
		if err != nil {
			rt.Fatal(err)
		}

		if mustHash(rt, []byte(updated)) != mustHash(rt, raw) {
			rt.Fatalf("exchange hash moved:\n%s\n->\n%s", raw, updated)
		}

		before, after := parsedUntouched(rt, string(raw)), parsedUntouched(rt, updated)
		if !untouchedEqual(before, after) {
			rt.Fatalf("non-parent fields moved: %+v -> %+v", before, after)
		}

		wantNote := basename
		if target != "" {
			wantNote = target
		}

		checkSinglePrimary(rt, updated, wantNote, receipt.StoredHash)

		again, againErr := cli.ExportApplyReceiptToContent([]byte(updated), receipt)
		if againErr != nil || again != updated {
			rt.Fatalf("not idempotent:\n%s\n->\n%s", updated, again)
		}
	})
}

// TestApplyReceiptToContent_LinksTheNote (spec "Receipt links the note"):
// the note records the parent's vault ID and a primary link
// {basename, offered, stored_hash}; nothing outside parent: changes.
func TestApplyReceiptToContent_LinksTheNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := offerTestNote{}.render(t)

	updated, err := cli.ExportApplyReceiptToContent(raw, testReceipt("1100.2026-09-27.x", ""))
	g.Expect(err).NotTo(HaveOccurred())

	parent := parsedParent(t, updated)
	g.Expect(parent.Vault).To(Equal(parentVaultID))
	g.Expect(parent.Links).To(Equal([]receiptLink{{Note: "1100.2026-09-27.x", Via: "offered", Hash: "xh1:stored"}}))
	g.Expect(withoutParentBlock(updated)).To(Equal(string(raw)))
	g.Expect(mustHash(t, []byte(updated))).To(Equal(mustHash(t, raw)))
}

// TestApplyReceiptToContent_OtherParentVaultStartsOver: links recorded
// under another parent's vault ID are meaningless to this one.
func TestApplyReceiptToContent_OtherParentVaultStartsOver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := offerTestNote{parentVault: seqID(77), linkVia: "offered", linkHash: "xh1:old", coveredNote: "3.x"}.render(t)

	updated, err := cli.ExportApplyReceiptToContent(raw, testReceipt("1100.2026-09-27.x", ""))
	g.Expect(err).NotTo(HaveOccurred())

	parent := parsedParent(t, updated)
	g.Expect(parent.Vault).To(Equal(parentVaultID))
	g.Expect(parent.Links).To(HaveLen(1))
}

// TestApplyReceiptToContent_ReceiptForRelinks (H3, spec "Re-link to the
// resolved target"): a receipt naming a resolved target makes it primary.
func TestApplyReceiptToContent_ReceiptForRelinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:old"}.render(t)

	updated, err := cli.ExportApplyReceiptToContent(raw, testReceipt("1101.2026-09-27.pending", "900.2026-09-01.e"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parsedParent(t, updated).Links).To(Equal([]receiptLink{
		{Note: "900.2026-09-01.e", Via: "offered", Hash: "xh1:stored"},
	}))
}

// TestApplyReceiptToContent_ReplacesPrimaryKeepsCovered: the previous
// primary (offered or pulled) is replaced, covered links are kept.
func TestApplyReceiptToContent_ReplacesPrimaryKeepsCovered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := offerTestNote{
		parentVault: parentVaultID, linkVia: "pulled", linkHash: "xh1:old", coveredNote: "3.2026-09-27.c",
	}.render(t)

	updated, err := cli.ExportApplyReceiptToContent(raw, testReceipt("1100.2026-09-27.x", ""))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parsedParent(t, updated).Links).To(ConsistOf(
		receiptLink{Note: "1100.2026-09-27.x", Via: "offered", Hash: "xh1:stored"},
		receiptLink{Note: "3.2026-09-27.c", Via: "covered", Hash: "xh1:c"},
	))
}

// receiptLink mirrors one parent link in frontmatter.
type receiptLink struct {
	Note string `yaml:"note"`
	Via  string `yaml:"via"`
	Hash string `yaml:"hash"`
}

// receiptParent mirrors the parent: field.
type receiptParent struct {
	Vault string        `yaml:"vault"`
	Links []receiptLink `yaml:"links"`
}

// untouchedFields are the fields a receipt write must never change.
type untouchedFields struct {
	Repo    string         `yaml:"repo"`
	User    string         `yaml:"user"`
	Vault   string         `yaml:"vault"`
	Pending bool           `yaml:"pending"`
	XID     string         `yaml:"xid"`
	Aliases []string       `yaml:"aliases"`
	Offer   map[string]any `yaml:"offer"`
}

// checkSinglePrimary asserts updated has exactly one primary link, naming
// wantNote via offered with hash.
func checkSinglePrimary(rt *rapid.T, updated, wantNote, hash string) {
	primaries := 0

	for _, link := range parsedParent(rt, updated).Links {
		if link.Via != "offered" && link.Via != "pulled" {
			continue
		}

		primaries++

		if link.Note != wantNote || link.Hash != hash || link.Via != "offered" {
			rt.Fatalf("primary %+v, want %s/offered/%s", link, wantNote, hash)
		}
	}

	if primaries != 1 {
		rt.Fatalf("%d primaries in:\n%s", primaries, updated)
	}
}

func parsedParent(tester failer, content string) receiptParent {
	tester.Helper()

	var doc struct {
		Parent receiptParent `yaml:"parent"`
	}

	err := yaml.Unmarshal(receiptFrontmatter(tester, content), &doc)
	if err != nil {
		tester.Fatal(err)
	}

	return doc.Parent
}

func parsedUntouched(tester failer, content string) untouchedFields {
	tester.Helper()

	var doc untouchedFields

	err := yaml.Unmarshal(receiptFrontmatter(tester, content), &doc)
	if err != nil {
		tester.Fatal(err)
	}

	return doc
}

func receiptFrontmatter(tester failer, content string) []byte {
	tester.Helper()

	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) < 3 {
		tester.Fatalf("no frontmatter in %q", content)

		return nil
	}

	return []byte(parts[1])
}

func testReceipt(basename, target string) cli.OfferReceiptForTest {
	return cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "1100", Basename: basename, Pending: true,
		VaultID: parentVaultID, StoredHash: "xh1:stored", For: target,
	}
}

func untouchedEqual(first, second untouchedFields) bool {
	firstYAML, _ := yaml.Marshal(first)
	secondYAML, _ := yaml.Marshal(second)

	return string(firstYAML) == string(secondYAML)
}

// withoutParentBlock drops the parent: key and its indented value.
func withoutParentBlock(content string) string {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	inParent := false

	for _, line := range lines {
		if strings.HasPrefix(line, "parent:") {
			inParent = true

			continue
		}

		if inParent && strings.HasPrefix(line, " ") {
			continue
		}

		inParent = false

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
}
