package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	offerFieldKey = "offer"
	parentKey     = "parent"
	xidKey        = "xid"
)

// unexported variables.
var (
	errNoteNoFrontmatter = errors.New("the note has no frontmatter")
	// errReceiptNoteUndecodable: the note's parent: does not decode, so a
	// receipt cannot be recorded on it.
	errReceiptNoteUndecodable = errors.New("the note's frontmatter does not decode")
)

// applyReceiptToContent records an offer receipt on the note (design D6):
// parent.vault becomes the receipt's vault ID and the primary link becomes
// {note: the resolved target when the receipt names one (H3), else the
// receipt's basename; via: offered; hash: the receipt's stored_hash}.
// Covered links under the same vault are kept; links under another vault
// are dropped. Only the parent: key is edited, as YAML nodes
// (editExchangeBlocks, ruling V7): unknown keys under parent: and inside a
// kept link survive, an anchored parent: refuses the receipt untouched, and
// every other key keeps its decoded value — so the exchange hash, identity
// and every other field are untouched (no re-stamp, and nothing here
// re-embeds).
func applyReceiptToContent(raw []byte, receipt offerReceipt) (string, error) {
	raw = toLF(raw) // a CRLF note takes the receipt as LF, converted in that write (#789 design D4)

	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return "", errNoteNoFrontmatter
	}

	var doc struct {
		Parent parentLinks `yaml:"parent"`
	}

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", fmt.Errorf("offer receipt: %w: parsing parent: %w", errReceiptNoteUndecodable, unmarshalErr)
	}

	updated, editErr := editExchangeBlocks(raw, exchangeBlocksEdit{
		parentBefore: doc.Parent, parentAfter: linkedParent(doc.Parent, receipt), setParentLink: true,
	})
	if editErr != nil {
		return "", fmt.Errorf("offer receipt: %w", editErr)
	}

	return updated, nil
}

// insertFrontmatterBlock inserts the top-level key, rendered as YAML from
// value, into content's frontmatter before the first of beforeKeys present
// (the frontmatter writer's key order), else at the end. Every other line is
// left as is. The caller ensures key is absent.
func insertFrontmatterBlock(content, key string, value any, beforeKeys ...string) (string, error) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return "", errNoteNoFrontmatter
	}

	rendered, marshalErr := yaml.Marshal(value)
	if marshalErr != nil {
		return "", fmt.Errorf("rendering %s: %w", key, marshalErr)
	}

	insertAt := -1

	for _, before := range beforeKeys {
		if index := yamlKeyLineIndex(frontmatter, before); index >= 0 {
			insertAt = index

			break
		}
	}

	block := strings.TrimSuffix(string(rendered), "\n")

	return fmStart + insertYAMLBlock(frontmatter, block, insertAt) + fmEnd + body, nil
}

// linkedParent is the note's parent links after a receipt.
func linkedParent(current parentLinks, receipt offerReceipt) parentLinks {
	if current.Vault != receipt.VaultID {
		current = parentLinks{Vault: receipt.VaultID}
	}

	target := receipt.Basename
	if receipt.For != "" {
		target = receipt.For
	}

	kept := slices.DeleteFunc(slices.Clone(current.Links), func(link parentLink) bool {
		return link.Via == linkViaOffered || link.Via == linkViaPulled || link.Note == target
	})

	current.Links = append([]parentLink{{Note: target, Via: linkViaOffered, Hash: receipt.StoredHash}}, kept...)

	return current
}

// receiptRefused reports whether err is the local note refusing a receipt
// because of its frontmatter — an anchored parent:, or frontmatter that does
// not decode — rather than a failure to write it (#789 design D3).
func receiptRefused(err error) bool {
	return errors.Is(err, errFrontmatterAnchoredKey) || errors.Is(err, errFrontmatterUndecodable) ||
		errors.Is(err, errFrontmatterNotMapping) || errors.Is(err, errNoteNoFrontmatter) ||
		errors.Is(err, errReceiptNoteUndecodable)
}

// setXIDField stamps xid onto a note that has none (the lazy stamp, design
// D4), in the writer's key order (before parent, aliases and offer). A note
// that already carries an xid is returned unchanged.
func setXIDField(content, xid string) (string, error) {
	frontmatter, _, ok := splitFrontmatterAndBody(content)
	if !ok {
		return "", errNoteNoFrontmatter
	}

	if yamlKeyLineIndex(frontmatter, xidKey) >= 0 {
		return content, nil
	}

	return insertFrontmatterBlock(content, xidKey, map[string]string{xidKey: xid}, parentKey, "aliases", "offer")
}
