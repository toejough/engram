package cli_test

// The fold (`amend --discard --into`) and the offer receipt edit aliases:
// and parent: as YAML nodes (ruling V7, re-review N1): an anchor on an
// edited key refuses the write with both files untouched, and unknown keys
// under parent: and its links survive.

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestApplyReceiptToContent_KeepsUnknownParentKeys: unknown keys under
// parent: and inside a kept link survive a receipt.
func TestApplyReceiptToContent_KeepsUnknownParentKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := withUnknownParentKeys(foldNote(foldExchange{
		xid: foldXIDE, vault: parentVaultID, links: []foldLink{{"9.2026-09-01.other", "covered", "xh1:other"}},
	}, true))

	updated, err := cli.ExportApplyReceiptToContent([]byte(note), testReceipt("1100.2026-09-27.x", ""))
	g.Expect(err).NotTo(HaveOccurred())

	expectUnknownParentKeysKept(g, updated)
	g.Expect(parsedParent(t, updated).Links).To(HaveLen(2))
}

// TestApplyReceiptToContent_RefusesAnchoredParent: a receipt on a note whose
// parent: carries an anchor another key aliases is refused unwritten.
func TestApplyReceiptToContent_RefusesAnchoredParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := anchorBlock(foldNote(foldExchange{
		xid: foldXIDE, vault: parentVaultID, links: []foldLink{{"1.2026-09-01.prev", "offered", "xh1:prev"}},
	}, true), "parent")

	updated, err := cli.ExportApplyReceiptToContent([]byte(note), testReceipt("1100.2026-09-27.x", ""))
	g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
	g.Expect(updated).To(BeEmpty())
}

// TestFoldAndReceipt_NodeEditProperty: over drawn exchange notes, with
// optional unknown keys under parent: and its first link and an optional
// aliased anchor on parent:, the fold and the receipt either refuse the
// anchored edit with every file untouched, or write frontmatter that decodes
// and keeps every unknown key and the anchored value.
func TestFoldAndReceipt_NodeEditProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		existing := drawFoldExchange(rt, "existing")
		before := foldNote(existing, false)
		unknown := len(existing.links) > 0 && rapid.Bool().Draw(rt, "unknown")
		anchored := len(existing.links) > 0 && rapid.Bool().Draw(rt, "anchored")

		if unknown {
			before = withUnknownParentKeys(before)
		}

		if anchored {
			before = anchorBlock(before, "parent")
		}

		if rapid.Bool().Draw(rt, "receipt") {
			receipt := testReceipt("1100.2026-09-27.x", "")
			receipt.VaultID = rapid.SampledFrom([]string{parentVaultID, foldParentVault}).Draw(rt, "receiptVault")
			updated, err := cli.ExportApplyReceiptToContent([]byte(before), receipt)
			assertNodeEditOutcome(rt, before, updated, err, unknown, anchored, nil)

			return
		}

		offer := drawFoldExchange(rt, "offer")
		offer.xid = foldXIDO
		vault := newFoldVault()
		vault.put(foldExistingName, before)
		vault.put(foldOfferName, foldNote(offer, true))
		snapshot := maps.Clone(vault.files)

		err := vault.amend(context.Background(), cli.AmendArgs{
			Target: foldOfferBase, Discard: true, Into: foldExistingBase,
		})
		assertNodeEditOutcome(rt, before, string(vault.files[foldExistingName]), err, unknown, anchored,
			func() bool { return reflect.DeepEqual(snapshot, vault.files) })
	})
}

// TestRunAmend_DiscardIntoKeepsUnknownParentKeys: unknown keys under
// into's parent: and inside its links survive the fold.
func TestRunAmend_DiscardIntoKeepsUnknownParentKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldExistingName, withUnknownParentKeys(foldNote(foldExchange{
		xid: foldXIDE, vault: foldParentVault, links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
	}, false)))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	expectUnknownParentKeysKept(g, string(vault.files[foldExistingName]))
	g.Expect(decodeFoldExchange(t, vault.files[foldExistingName]).Parent.Links).To(HaveLen(2))
}

// TestRunAmend_DiscardIntoRefusesAnchoredEditedKey: a fold whose into note
// carries an anchor on parent: (or aliases:) that another key aliases is
// refused, and neither into nor the offer is touched.
func TestRunAmend_DiscardIntoRefusesAnchoredEditedKey(t *testing.T) {
	t.Parallel()

	for name, anchor := range map[string]func(string) string{
		"parent":  func(note string) string { return anchorBlock(note, "parent") },
		"aliases": func(note string) string { return anchorBlock(note, "aliases") },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newFoldVault()
			existing := anchor(foldNote(foldExchange{
				xid: foldXIDE, vault: foldParentVault, aliases: []string{"2.2026-01-01.old-name"},
				links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
			}, false))
			offer := foldNote(foldExchange{
				xid: foldXIDO, vault: foldParentVault, aliases: []string{"5.2026-01-01.prev-name"},
				links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
			}, true)
			vault.put(foldExistingName, existing)
			vault.put(foldOfferName, offer)

			err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
			g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
			g.Expect(err).To(MatchError(ContainSubstring(name)))
			g.Expect(string(vault.files[foldExistingName])).To(Equal(existing), "into must be untouched")
			g.Expect(string(vault.files[foldOfferName])).To(Equal(offer), "the offer must not be deleted")
		})
	}
}

// anchorBlock anchors key's block in note (`key: &x`) and adds a key
// aliasing it.
func anchorBlock(note, key string) string {
	note = strings.Replace(note, "\n"+key+":\n", "\n"+key+": &x\n", 1)

	return strings.Replace(note, "\n---\n\n", "\n"+key+"_copy: *x\n---\n\n", 1)
}

// assertNodeEditOutcome checks one fold or receipt: a refusal is the
// anchored-key error and leaves everything untouched (untouched reports
// that for a fold; a receipt returns ""); success decodes, keeps the
// anchored copy equal to parent:, and keeps every unknown key that still
// has its link.
func assertNodeEditOutcome(
	rt *rapid.T, before, after string, err error, unknown, anchored bool, untouched func() bool,
) {
	if err != nil {
		assertRefusedUntouched(rt, after, err, anchored, untouched)

		return
	}

	fields := frontmatterOf(after)
	if len(fields) == 0 {
		rt.Fatalf("output does not decode:\n%s", after)
	}

	if anchored && !reflect.DeepEqual(fields["parent"], fields["parent_copy"]) && after != before {
		// The anchor survived only if parent: was not edited; an edit refuses.
		rt.Fatalf("anchored parent: was edited:\n%s", after)
	}

	if unknown {
		assertUnknownParentKeysKept(rt, before, after, fields)
	}
}

// assertRefusedUntouched checks a refusal is the anchored-key error on an
// anchored note and wrote nothing.
func assertRefusedUntouched(rt *rapid.T, after string, err error, anchored bool, untouched func() bool) {
	if !anchored || !errors.Is(err, cli.ErrFrontmatterAnchoredKeyForTest) {
		rt.Fatalf("unexpected error (anchored %v): %v", anchored, err)
	}

	if untouched != nil && !untouched() || untouched == nil && after != "" {
		rt.Fatalf("a refused edit wrote something")
	}
}

// assertUnknownParentKeysKept checks the parent-level unknown key survived,
// and the link-level one did wherever its link (by note) is still held.
func assertUnknownParentKeysKept(rt *rapid.T, before, after string, fields map[string]any) {
	parent, _ := fields["parent"].(map[string]any)
	if parent["extra_parent_field"] != "keep" {
		rt.Fatalf("parent-level unknown key dropped:\n%s", after)
	}

	beforeParent, _ := frontmatterOf(before)["parent"].(map[string]any)
	beforeLinks, _ := beforeParent["links"].([]any)

	for _, raw := range beforeLinks {
		link, _ := raw.(map[string]any)
		if link["extra_link_field"] == nil {
			continue
		}

		afterLinks, _ := parent["links"].([]any)
		for _, rawAfter := range afterLinks {
			kept, _ := rawAfter.(map[string]any)
			if kept["note"] == link["note"] && kept["extra_link_field"] != "keep" {
				rt.Fatalf("link-level unknown key dropped for %v:\n%s", link["note"], after)
			}
		}
	}
}

// expectUnknownParentKeysKept asserts the unknown keys withUnknownParentKeys
// adds are still in content, at the same place.
func expectUnknownParentKeysKept(g Gomega, content string) {
	fields := frontmatterOf(content)
	g.Expect(fields).NotTo(BeEmpty(), "the note must decode:\n%s", content)

	parent, _ := fields["parent"].(map[string]any)
	g.Expect(parent).To(HaveKeyWithValue("extra_parent_field", "keep"))

	links, _ := parent["links"].([]any)
	g.Expect(links).To(ContainElement(HaveKeyWithValue("extra_link_field", "keep")))
}

// withUnknownParentKeys adds an unknown key to note's parent: and to its
// first link.
func withUnknownParentKeys(note string) string {
	note = strings.Replace(note, "\nparent:\n", "\nparent:\n    extra_parent_field: keep\n", 1)

	return strings.Replace(note, "\n          hash: ", "\n          extra_link_field: keep\n          hash: ", 1)
}
