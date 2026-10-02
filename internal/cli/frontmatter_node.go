package cli

// Shared YAML-node frontmatter edits: a rewrite that sets or deletes a few
// keys on the parsed mapping, so every key it does not touch — including
// keys no typed note model defines — survives with its value. Used by
// pull-down, adopt/refresh (applySkillNoteBody), resituate, amend and
// identity backfill.

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	// mappingStride is the step between a YAML mapping node's key nodes:
	// its Content alternates key, value.
	mappingStride = 2
	// yamlStringTag is the YAML tag of a plain string scalar.
	yamlStringTag = "!!str"
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

// applyTypedEdit edits mapping — a note's parsed frontmatter — so it carries
// a typed doc's edit, as node edits: before and after are the typed doc's
// encodings (encodeNode) as decoded from mapping and after the edit. A key
// after carries is set in place when its encoding differs from before's, or
// inserted at after's key position when mapping lacks it (as the typed
// writer always emitted it); a key after omits is deleted. Each normalized
// key is also re-emitted whenever mapping's own text for it differs from
// after's encoding (the typed writer's canonical form, e.g. a quoted
// created: date). Every other key — an unedited modeled key, or one the
// typed doc does not define — is left exactly as parsed, anchors and style
// included. A key the edit would set or delete that carries an anchor
// refuses the whole edit (errFrontmatterAnchoredKey) before anything
// changes.
func applyTypedEdit(mapping, before, after *yaml.Node, normalized ...string) error {
	order, changed, deleted := planTypedEdit(mapping, before, after, normalized)

	anchorErr := refuseAnchoredKeys(mapping, append(slices.Collect(maps.Keys(changed)), deleted...)...)
	if anchorErr != nil {
		return anchorErr
	}

	deleteMappingKeys(mapping, deleted...)

	for _, key := range order {
		if value, ok := changed[key]; ok {
			setMappingValueOrdered(mapping, key, value, order)
		}
	}

	return nil
}

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

// mappingKeyIndex is the Content index of key's key node in mapping, or -1.
func mappingKeyIndex(mapping *yaml.Node, key string) int {
	for index := 0; index+1 < len(mapping.Content); index += mappingStride {
		if mapping.Content[index].Value == key {
			return index
		}
	}

	return -1
}

// nodeEditFrontmatter writes a typed doc's edit onto a note's parsed
// frontmatter mapping as node edits (applyTypedEdit, with created:
// re-emitted in the typed writer's quoted form), prepends the result to
// body, and decodes it again before returning it. before is the typed
// doc's encoding as decoded from mapping; after is the edited doc. Every
// key the edit does not change is left exactly as parsed. An anchor on a
// key the edit sets or deletes refuses it (errFrontmatterAnchoredKey); a
// result that does not decode is refused (errFrontmatterUndecodable).
func nodeEditFrontmatter(mapping, before *yaml.Node, after any, body string) (string, error) {
	editErr := applyTypedEdit(mapping, before, encodeNode(after), "created")
	if editErr != nil {
		return "", editErr
	}

	rendered := marshalFrontmatter(mapping) + body

	verifyErr := verifyFrontmatterDecodes(rendered)
	if verifyErr != nil {
		return "", verifyErr
	}

	return rendered, nil
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

// planTypedEdit is applyTypedEdit's plan: after's key order, the keys to
// set (with their new value nodes), and the keys to delete.
func planTypedEdit(
	mapping, before, after *yaml.Node, normalized []string,
) (order []string, changed map[string]*yaml.Node, deleted []string) {
	beforeValues := make(map[string]string, len(before.Content)/mappingStride)
	for index := 0; index+1 < len(before.Content); index += mappingStride {
		beforeValues[before.Content[index].Value] = renderNode(before.Content[index+1])
	}

	order = make([]string, 0, len(after.Content)/mappingStride)
	changed = make(map[string]*yaml.Node, len(after.Content)/mappingStride)

	for index := 0; index+1 < len(after.Content); index += mappingStride {
		key, value := after.Content[index].Value, after.Content[index+1]
		order = append(order, key)

		if typedKeyChanged(mapping, key, value, beforeValues, normalized) {
			changed[key] = value
		}
	}

	deleted = make([]string, 0, len(beforeValues))

	for key := range beforeValues {
		if !slices.Contains(order, key) {
			deleted = append(deleted, key)
		}
	}

	return order, changed, deleted
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

// renderNode is a value node's YAML text, for comparing two encodings.
func renderNode(node *yaml.Node) string {
	rendered, _ := yaml.Marshal(node) // an encoded value node always marshals

	return string(rendered)
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

	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: yamlStringTag, Value: key}, value)
}

// setMappingValueOrdered sets key's value in mapping: in place when the key
// exists, otherwise inserted after the nearest key that precedes it in
// order (before the nearest following one when none precedes it, else at
// the end), so a new key lands where the typed frontmatter writer puts it.
func setMappingValueOrdered(mapping *yaml.Node, key string, value *yaml.Node, order []string) {
	if index := mappingKeyIndex(mapping, key); index >= 0 {
		mapping.Content[index+1] = value

		return
	}

	position := slices.Index(order, key)
	insertAt := -1

	for previous := position - 1; previous >= 0 && insertAt < 0; previous-- {
		if index := mappingKeyIndex(mapping, order[previous]); index >= 0 {
			insertAt = index + mappingStride
		}
	}

	if insertAt < 0 {
		insertAt = len(mapping.Content)

		for next := position + 1; next < len(order); next++ {
			if index := mappingKeyIndex(mapping, order[next]); index >= 0 {
				insertAt = index

				break
			}
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: yamlStringTag, Value: key}
	mapping.Content = slices.Insert(mapping.Content, insertAt, keyNode, value)
}

// typedKeyChanged reports whether key, with value as after's encoding, must
// be written: mapping lacks it, before lacked it or encoded it differently,
// or it is a normalized key whose text in mapping differs from value's.
func typedKeyChanged(
	mapping *yaml.Node, key string, value *yaml.Node, beforeValues map[string]string, normalized []string,
) bool {
	keyIndex := mappingKeyIndex(mapping, key)
	previous, known := beforeValues[key]
	rendered := renderNode(value)

	if keyIndex < 0 || !known || previous != rendered {
		return true
	}

	return slices.Contains(normalized, key) && renderNode(mapping.Content[keyIndex+1]) != rendered
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
