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
	errFrontmatterNotMapping = errors.New("frontmatter is not a YAML mapping")
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
