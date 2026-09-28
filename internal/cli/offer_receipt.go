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
)

// applyReceiptToContent records an offer receipt on the note (design D6):
// parent.vault becomes the receipt's vault ID and the primary link becomes
// {note: the resolved target when the receipt names one (H3), else the
// receipt's basename; via: offered; hash: the receipt's stored_hash}.
// Covered links under the same vault are kept; links under another vault
// are dropped. Only the parent: key is rewritten — no other byte of the
// note changes, so the exchange hash, identity and every other field are
// untouched (no re-stamp, and nothing here re-embeds).
func applyReceiptToContent(raw []byte, receipt offerReceipt) (string, error) {
	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return "", errNoteNoFrontmatter
	}

	var doc struct {
		Parent parentLinks `yaml:"parent"`
	}

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", fmt.Errorf("offer receipt: parsing parent: %w", unmarshalErr)
	}

	return setFrontmatterBlock(string(raw), parentKey, map[string]parentLinks{
		parentKey: linkedParent(doc.Parent, receipt),
	}, "aliases", "offer")
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

// setFrontmatterBlock replaces the top-level key's block in content's
// frontmatter with value rendered as YAML, or — when the key is absent —
// inserts it before the first of beforeKeys present (the frontmatter
// writer's key order), else at the end. Every other line is left as is.
func setFrontmatterBlock(content, key string, value any, beforeKeys ...string) (string, error) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return "", errNoteNoFrontmatter
	}

	rendered, marshalErr := yaml.Marshal(value)
	if marshalErr != nil {
		return "", fmt.Errorf("rendering %s: %w", key, marshalErr)
	}

	block := strings.TrimSuffix(string(rendered), "\n")
	lines := strings.Split(frontmatter, "\n")

	start := yamlKeyLineIndex(frontmatter, key)
	if start >= 0 {
		end := yamlValueEndLine(lines, start, key)
		kept := make([]string, 0, len(lines)-(end-start)+1)
		kept = append(kept, lines[:start]...)
		kept = append(kept, block)
		kept = append(kept, lines[end:]...)

		return fmStart + strings.Join(kept, "\n") + fmEnd + body, nil
	}

	insertAt := -1

	for _, before := range beforeKeys {
		if index := yamlKeyLineIndex(frontmatter, before); index >= 0 {
			insertAt = index

			break
		}
	}

	return fmStart + insertYAMLBlock(frontmatter, block, insertAt) + fmEnd + body, nil
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

	return setFrontmatterBlock(content, xidKey, map[string]string{xidKey: xid}, parentKey, "aliases", "offer")
}
