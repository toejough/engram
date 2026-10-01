package cli

import (
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// exchangeBlockCount is the number of exchange blocks editExchangeBlocks
	// can edit (aliases:, parent:).
	exchangeBlockCount = 2
	parentLinksKey     = "links"
)

// exchangeBlocksEdit is a frontmatter edit of a note's exchange blocks: the
// aliases: list (nil leaves it alone) and the parent: links (applied only
// when after differs from before, the parent block as decoded).
type exchangeBlocksEdit struct {
	aliases       []string
	parentBefore  parentLinks
	parentAfter   parentLinks
	setAliases    bool
	setParentLink bool
}

// editExchangeBlocks applies edit to raw's frontmatter as YAML-node edits
// (ruling V7): the aliases: list is replaced whole, and parent: is edited
// in place — its vault and author set, and each link's node reused when the
// edit keeps a link to the same note — so unknown keys under parent: and
// inside its links survive. Every other key is left as parsed. A key it
// edits that carries an anchor refuses the edit (errFrontmatterAnchoredKey),
// and the result is decoded again before it is returned
// (errFrontmatterUndecodable). A note the edit leaves unchanged is returned
// byte-for-byte. New blocks go where the frontmatter writer puts them
// (parent:, aliases:, offer: order).
func editExchangeBlocks(raw []byte, edit exchangeBlocksEdit) (string, error) {
	setParent := edit.setParentLink && !sameParentLinks(edit.parentBefore, edit.parentAfter)
	if !edit.setAliases && !setParent {
		return string(raw), nil
	}

	frontmatter, rest, found := embed.SplitFrontmatter(raw)
	if !found {
		return "", errNoteNoFrontmatter
	}

	mapping, parseErr := parseFrontmatterMapping(frontmatter)
	if parseErr != nil {
		return "", parseErr
	}

	anchorErr := refuseAnchoredKeys(mapping, editedExchangeKeys(edit.setAliases, setParent)...)
	if anchorErr != nil {
		return "", anchorErr
	}

	order := []string{parentKey, aliasesKey, offerFieldKey}

	if edit.setAliases {
		setMappingValueOrdered(mapping, aliasesKey, encodeNode(edit.aliases), order)
	}

	if setParent {
		editParentNode(mapping, edit.parentBefore, edit.parentAfter, order)
	}

	return renderVerifiedFrontmatter(mapping, rest)
}

// editParentNode edits mapping's parent: node from before to after: vault
// and author as typed edits (applyTypedEdit), the links sequence rebuilt
// from after with each link reusing the node of before's link to the same
// note (so its unknown keys survive). A missing or non-mapping parent: is
// written fresh.
func editParentNode(mapping *yaml.Node, before, after parentLinks, order []string) {
	keyIndex := mappingKeyIndex(mapping, parentKey)
	if keyIndex < 0 || mapping.Content[keyIndex+1].Kind != yaml.MappingNode {
		setMappingValueOrdered(mapping, parentKey, encodeNode(after), order)

		return
	}

	parent := mapping.Content[keyIndex+1]
	beforeScalars, afterScalars := before, after
	beforeScalars.Links, afterScalars.Links = nil, nil

	// Anchors were refused above, so the typed edit cannot fail.
	_ = applyTypedEdit(parent, encodeNode(beforeScalars), encodeNode(afterScalars))

	if len(after.Links) == 0 {
		deleteMappingKeys(parent, parentLinksKey)

		return
	}

	held := heldLinkNodes(parent, before)
	links := make([]*yaml.Node, 0, len(after.Links))

	for _, link := range after.Links {
		index := slices.IndexFunc(before.Links, func(previous parentLink) bool { return previous.Note == link.Note })
		if index < 0 || index >= len(held) {
			links = append(links, encodeNode(link))

			continue
		}

		node := held[index]
		_ = applyTypedEdit(node, encodeNode(before.Links[index]), encodeNode(link))
		links = append(links, node)
	}

	linksIndex := mappingKeyIndex(parent, parentLinksKey)
	if linksIndex >= 0 && parent.Content[linksIndex+1].Kind == yaml.SequenceNode {
		parent.Content[linksIndex+1].Content = links

		return
	}

	sequence := encodeNode(after.Links)
	sequence.Content = links
	setMappingValueOrdered(parent, parentLinksKey, sequence, []string{"vault", parentLinksKey, "author"})
}

// editedExchangeKeys is the exchange block keys an edit changes.
func editedExchangeKeys(setAliases, setParent bool) []string {
	edited := make([]string, 0, exchangeBlockCount)
	if setAliases {
		edited = append(edited, aliasesKey)
	}

	if setParent {
		edited = append(edited, parentKey)
	}

	return edited
}

// heldLinkNodes is parent's links sequence entries, aligned with
// before.Links (nil when parent: holds no links sequence or its entries are
// not mappings).
func heldLinkNodes(parent *yaml.Node, before parentLinks) []*yaml.Node {
	index := mappingKeyIndex(parent, parentLinksKey)
	if index < 0 || parent.Content[index+1].Kind != yaml.SequenceNode {
		return nil
	}

	entries := parent.Content[index+1].Content
	if len(entries) != len(before.Links) {
		return nil
	}

	for _, entry := range entries {
		if entry.Kind != yaml.MappingNode {
			return nil
		}
	}

	return entries
}

// renderVerifiedFrontmatter renders mapping as the note's frontmatter ahead
// of rest (the text after the closing delimiter, verbatim) and decodes the
// result again (errFrontmatterUndecodable) before returning it.
func renderVerifiedFrontmatter(mapping *yaml.Node, rest []byte) (string, error) {
	rendered, marshalErr := yaml.Marshal(mapping)
	if marshalErr != nil {
		return "", fmt.Errorf("rendering frontmatter: %w", marshalErr)
	}

	content := fmStart + string(rendered) + fmStart + string(rest)

	verifyErr := verifyFrontmatterDecodes(content)
	if verifyErr != nil {
		return "", verifyErr
	}

	return content, nil
}
