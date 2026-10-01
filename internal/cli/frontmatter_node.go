package cli

// Shared YAML-node frontmatter edits: a rewrite that sets or deletes a few
// keys on the parsed mapping, so every key it does not touch — including
// keys no typed note model defines — survives with its value. Used by
// pull-down, adopt/refresh (applySkillNoteBody) and resituate.

import (
	"errors"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"
)

// unexported variables.
var (
	// errFrontmatterAnchoredKey refuses a node edit whose replaced or
	// deleted key carries a YAML anchor: dropping the value would leave any
	// alias to it dangling (or, if moved, silently change what it means).
	errFrontmatterAnchoredKey = errors.New("frontmatter key the edit replaces carries a YAML anchor")
	errFrontmatterNotMapping  = errors.New("frontmatter is not a YAML mapping")
	// errFrontmatterUndecodable refuses to write a node edit whose rendered
	// frontmatter does not decode (a guard behind errFrontmatterAnchoredKey).
	errFrontmatterUndecodable = errors.New("rewritten frontmatter does not decode")
)

// cloneNode deep-copies a YAML node, so building a copy never mutates the
// parsed source.
func cloneNode(node *yaml.Node) *yaml.Node {
	clone := *node
	clone.Content = make([]*yaml.Node, 0, len(node.Content))

	for _, child := range node.Content {
		clone.Content = append(clone.Content, cloneNode(child))
	}

	return &clone
}

// deleteMappingKeys removes each key (and its value) from a YAML mapping.
func deleteMappingKeys(mapping *yaml.Node, keys ...string) {
	kept := make([]*yaml.Node, 0, len(mapping.Content))

	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if slices.Contains(keys, mapping.Content[index].Value) {
			continue
		}

		kept = append(kept, mapping.Content[index], mapping.Content[index+1])
	}

	mapping.Content = kept
}

// encodeNode is value as a YAML node, rendered as yaml.Marshal would.
func encodeNode(value any) *yaml.Node {
	var node yaml.Node

	_ = node.Encode(value) // strings, bools and plain structs always encode

	return &node
}

// hasAnchor reports whether node or any node beneath it carries an anchor.
// Alias nodes are not followed (an alias is a reference, not an anchor).
func hasAnchor(node *yaml.Node) bool {
	if node.Anchor != "" {
		return true
	}

	return slices.ContainsFunc(node.Content, hasAnchor)
}

// parseFrontmatterMapping parses a frontmatter block into its top-level
// YAML mapping node. An empty block yields an empty mapping; a block that is
// not a mapping is an error (errFrontmatterNotMapping).
func parseFrontmatterMapping(frontmatter []byte) (*yaml.Node, error) {
	var document yaml.Node

	parseErr := yaml.Unmarshal(frontmatter, &document)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", parseErr)
	}

	if document.Kind == 0 {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}, nil
	}

	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errFrontmatterNotMapping
	}

	return document.Content[0], nil
}

// refuseAnchoredKeys returns errFrontmatterAnchoredKey naming the first of
// keys present in mapping whose key node, value node, or any node beneath
// the value carries an anchor — the keys a node edit is about to replace or
// delete. Any anchor refuses, aliased elsewhere or not: an unaliased anchor
// is just as hand-authored, and refusing keeps the rule simple (ruling V3).
func refuseAnchoredKeys(mapping *yaml.Node, keys ...string) error {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key, value := mapping.Content[index], mapping.Content[index+1]
		if !slices.Contains(keys, key.Value) {
			continue
		}

		if key.Anchor != "" || hasAnchor(value) {
			return fmt.Errorf("%w: %s", errFrontmatterAnchoredKey, key.Value)
		}
	}

	return nil
}

// setMappingValue sets key's value in a YAML mapping, in place when the
// key exists, otherwise appended.
func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = value

			return
		}
	}

	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

// verifyFrontmatterDecodes decodes content's frontmatter again, as a guard
// before a node edit writes it: errFrontmatterUndecodable when it does not
// decode as a YAML mapping.
func verifyFrontmatterDecodes(content string) error {
	frontmatter, found := splitFrontmatter([]byte(content))
	if !found {
		return fmt.Errorf("%w: no frontmatter", errFrontmatterUndecodable)
	}

	mapping, parseErr := parseFrontmatterMapping(frontmatter)
	if parseErr != nil {
		return fmt.Errorf("%w: %w", errFrontmatterUndecodable, parseErr)
	}

	var decoded map[string]any

	decodeErr := mapping.Decode(&decoded)
	if decodeErr != nil {
		return fmt.Errorf("%w: %w", errFrontmatterUndecodable, decodeErr)
	}

	return nil
}
