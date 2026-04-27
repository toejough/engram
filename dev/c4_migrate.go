//go:build targ

// THROWAWAY: This file is deleted in Task 11 after migration is complete (#585).
package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4Migrate).Name("c4-migrate").
		Description(migrateDesc))
}

// C4MigrateArgs configures the c4-migrate target.
type C4MigrateArgs struct {
	Dir string `targ:"flag,name=dir,desc=Directory to migrate (default architecture/c4)"`
}

// unexported constants.
const (
	defaultMigrateDir = "architecture/c4"
	migrateDesc       = "THROWAWAY: Migrate flat E<n> IDs to hierarchical S/N/M/P paths in-place." +
		" Deleted after issue #585."
	specFileMode = 0o644
)

func c4Migrate(_ context.Context, args C4MigrateArgs) error {
	dir := args.Dir
	if dir == "" {
		dir = defaultMigrateDir
	}

	// Step 1: Migrate L1 — assign S-IDs; rewrite edges to use IDs.
	l1Path := filepath.Join(dir, "c1-engram-system.json")
	l1Map, err := migrateL1(l1Path)
	if err != nil {
		return fmt.Errorf("migrate L1 %s: %w", l1Path, err)
	}

	// Step 2: Migrate L2 — focus path = in_scope element's L1 ID;
	// new containers get N-IDs; carried-over elements get their L1 path.
	l2Path := filepath.Join(dir, "c2-engram-plugin.json")
	l2Map, err := migrateL2(l2Path, l1Map)
	if err != nil {
		return fmt.Errorf("migrate L2 %s: %w", l2Path, err)
	}

	// Step 3: Migrate each L3 — focus path = L2 path; new components get M-IDs.
	l3Files, err := filepath.Glob(filepath.Join(dir, "c3-*.json"))
	if err != nil {
		return fmt.Errorf("glob L3 files: %w", err)
	}
	sort.Strings(l3Files)
	l3Maps := map[string]map[string]string{}
	for _, l3File := range l3Files {
		l3Map, l3Err := migrateL3(l3File, l2Map)
		if l3Err != nil {
			return fmt.Errorf("migrate L3 %s: %w", l3File, l3Err)
		}
		l3Maps[filepath.Base(l3File)] = l3Map
	}

	// Step 4: Migrate each L4 — look up parent L3 map; assign P-IDs for properties.
	l4Files, err := filepath.Glob(filepath.Join(dir, "c4-*.json"))
	if err != nil {
		return fmt.Errorf("glob L4 files: %w", err)
	}
	sort.Strings(l4Files)
	for _, l4File := range l4Files {
		if l4Err := migrateL4(l4File, l3Maps); l4Err != nil {
			return fmt.Errorf("migrate L4 %s: %w", l4File, l4Err)
		}
	}
	return nil
}

// migrateL1 assigns S<n> to L1 elements in JSON order, rewrites edges from
// names to IDs, drops the is_system field, and returns a name→path map.
func migrateL1(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, unmarshalErr)
	}

	elements, _ := spec["elements"].([]any)
	nameToID := make(map[string]string, len(elements))
	for index, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		newID := fmt.Sprintf("S%d", index+1)
		element["id"] = newID
		name, _ := element["name"].(string)
		nameToID[name] = newID
		delete(element, "is_system")
	}

	rels, _ := spec["relationships"].([]any)
	for _, relAny := range rels {
		rel, _ := relAny.(map[string]any)
		if endpointErr := rewriteEndpointByName(rel, "from", nameToID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L1 relationship from: %w", endpointErr)
		}
		if endpointErr := rewriteEndpointByName(rel, "to", nameToID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L1 relationship to: %w", endpointErr)
		}
	}

	if writeErr := writeSpecJSON(path, spec); writeErr != nil {
		return nil, writeErr
	}
	return nameToID, nil
}

// migrateL2 assigns N<n> to new containers, uses L1 paths for carried-over
// elements, normalizes edges, and returns a map keyed by both name and old ID.
func migrateL2(path string, l1Map map[string]string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, unmarshalErr)
	}

	elements, _ := spec["elements"].([]any)

	// First pass: find the in_scope element and its L1 focus path.
	focusPath := ""
	for _, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		isInScope, _ := element["in_scope"].(bool)
		if isInScope {
			name, _ := element["name"].(string)
			if l1Path, ok := l1Map[name]; ok {
				focusPath = l1Path
			}
		}
	}
	if focusPath == "" {
		return nil, fmt.Errorf("L2 %s: no in_scope element resolvable via L1", path)
	}

	// Second pass: assign IDs.
	resolveByOldID := make(map[string]string, len(elements))
	resolveByName := make(map[string]string, len(elements))
	containerCounter := 0
	for _, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		oldID, _ := element["id"].(string)
		name, _ := element["name"].(string)
		fromParent, _ := element["from_parent"].(bool)
		isInScope, _ := element["in_scope"].(bool)

		var newID string
		switch {
		case isInScope:
			newID = focusPath
		case fromParent:
			l1Path, ok := l1Map[name]
			if !ok {
				return nil, fmt.Errorf(
					"L2 %s: element %q (id=%s) has from_parent but name not found in L1",
					path, name, oldID,
				)
			}
			newID = l1Path
		default:
			containerCounter++
			newID = fmt.Sprintf("%s-N%d", focusPath, containerCounter)
		}

		element["id"] = newID
		resolveByOldID[oldID] = newID
		resolveByName[name] = newID
		delete(element, "from_parent")
	}

	rels, _ := spec["relationships"].([]any)
	for _, relAny := range rels {
		rel, _ := relAny.(map[string]any)
		if endpointErr := rewriteEndpoint(rel, "from", resolveByName, resolveByOldID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L2 relationship from: %w", endpointErr)
		}
		if endpointErr := rewriteEndpoint(rel, "to", resolveByName, resolveByOldID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L2 relationship to: %w", endpointErr)
		}
	}

	if writeErr := writeSpecJSON(path, spec); writeErr != nil {
		return nil, writeErr
	}

	// Merge both maps for L3 lookup (L3 may reference by old E-ID or by name).
	merged := make(map[string]string, len(resolveByOldID)+len(resolveByName))
	for key, value := range resolveByOldID {
		merged[key] = value
	}
	for key, value := range resolveByName {
		merged[key] = value
	}
	return merged, nil
}

// migrateL3 sets focus.id to its L2 path, assigns M-IDs to new components,
// uses L2 paths for carried-over elements, and returns a merged (name|oldID)→path map.
func migrateL3(path string, l2Map map[string]string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, unmarshalErr)
	}

	focus, _ := spec["focus"].(map[string]any)
	focusOldID, _ := focus["id"].(string)
	focusPath, ok := l2Map[focusOldID]
	if !ok {
		return nil, fmt.Errorf("L3 %s: focus id %q not found in L2 map", path, focusOldID)
	}
	focus["id"] = focusPath

	elements, _ := spec["elements"].([]any)
	resolveByOldID := make(map[string]string, len(elements))
	resolveByName := make(map[string]string, len(elements))
	componentCounter := 0
	for _, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		oldID, _ := element["id"].(string)
		name, _ := element["name"].(string)
		fromParent, _ := element["from_parent"].(bool)

		var newID string
		if fromParent {
			l2Path, found := l2Map[oldID]
			if !found {
				l2Path, found = l2Map[name]
			}
			if !found {
				return nil, fmt.Errorf(
					"L3 %s: element %q (id=%s) has from_parent but not found in L2 map",
					path, name, oldID,
				)
			}
			newID = l2Path
		} else {
			componentCounter++
			newID = fmt.Sprintf("%s-M%d", focusPath, componentCounter)
		}

		element["id"] = newID
		resolveByOldID[oldID] = newID
		resolveByName[name] = newID
		delete(element, "from_parent")
	}

	rels, _ := spec["relationships"].([]any)
	for _, relAny := range rels {
		rel, _ := relAny.(map[string]any)
		if endpointErr := rewriteEndpoint(rel, "from", resolveByName, resolveByOldID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L3 relationship from: %w", endpointErr)
		}
		if endpointErr := rewriteEndpoint(rel, "to", resolveByName, resolveByOldID); endpointErr != nil {
			return nil, fmt.Errorf("rewrite L3 relationship to: %w", endpointErr)
		}
	}

	if writeErr := writeSpecJSON(path, spec); writeErr != nil {
		return nil, writeErr
	}

	merged := make(map[string]string, len(resolveByOldID)+len(resolveByName))
	for key, value := range resolveByOldID {
		merged[key] = value
	}
	for key, value := range resolveByName {
		merged[key] = value
	}
	return merged, nil
}

// migrateL4 looks up the parent L3 map via spec.parent, rewrites focus.id,
// diagram node/edge IDs, dependency_manifest wired_by_id, and assigns P<n> to properties.
func migrateL4(path string, l3Maps map[string]map[string]string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return fmt.Errorf("unmarshal %s: %w", path, unmarshalErr)
	}

	parentMD, _ := spec["parent"].(string) // e.g. "c3-engram-cli-binary.md"
	parentJSON := strings.TrimSuffix(parentMD, ".md") + ".json"
	parentMap, ok := l3Maps[parentJSON]
	if !ok {
		return fmt.Errorf("L4 %s: parent %q not in migrated L3 map", path, parentJSON)
	}

	focus, _ := spec["focus"].(map[string]any)
	focusOldID, _ := focus["id"].(string)
	focusPath, ok := parentMap[focusOldID]
	if !ok {
		return fmt.Errorf("L4 %s: focus id %q not found in parent map", path, focusOldID)
	}
	focus["id"] = focusPath
	delete(focus, "l3_container") // redundant once focus.id is hierarchical

	// Rewrite diagram node IDs and edge endpoints.
	diagram, _ := spec["diagram"].(map[string]any)
	if diagram != nil {
		nodes, _ := diagram["nodes"].([]any)
		for _, nodeAny := range nodes {
			node, _ := nodeAny.(map[string]any)
			oldID, _ := node["id"].(string)
			if oldID == focusOldID {
				node["id"] = focusPath
				continue
			}
			if newID, found := parentMap[oldID]; found {
				node["id"] = newID
				continue
			}
			return fmt.Errorf("L4 %s: diagram node %q not found in parent map", path, oldID)
		}

		edges, _ := diagram["edges"].([]any)
		for _, edgeAny := range edges {
			edge, _ := edgeAny.(map[string]any)
			for _, endpoint := range []string{"from", "to"} {
				oldVal, _ := edge[endpoint].(string)
				if oldVal == focusOldID {
					edge[endpoint] = focusPath
					continue
				}
				if newVal, found := parentMap[oldVal]; found {
					edge[endpoint] = newVal
				}
			}
		}
	}

	// Rewrite dependency_manifest wired_by_id references and inline P<n>
	// property shorthand references.
	manifest, _ := spec["dependency_manifest"].([]any)
	for _, rowAny := range manifest {
		row, _ := rowAny.(map[string]any)
		if oldID, hasField := row["wired_by_id"].(string); hasField {
			if newID, found := parentMap[oldID]; found {
				row["wired_by_id"] = newID
			}
		}
		propRefs, _ := row["properties"].([]any)
		for index, propRefAny := range propRefs {
			propRef, isString := propRefAny.(string)
			if !isString {
				continue
			}
			if strings.HasPrefix(propRef, "P") {
				propRefs[index] = focusPath + "-" + propRef
			}
		}
	}

	// Rewrite di_wires consumer_id references (provider-side DI manifest).
	diWires, _ := spec["di_wires"].([]any)
	for _, wireAny := range diWires {
		wire, _ := wireAny.(map[string]any)
		if oldID, hasField := wire["consumer_id"].(string); hasField {
			if newID, found := parentMap[oldID]; found {
				wire["consumer_id"] = newID
			}
		}
	}

	// Assign sequential P<n> IDs to properties in JSON order.
	properties, _ := spec["properties"].([]any)
	for index, propertyAny := range properties {
		property, _ := propertyAny.(map[string]any)
		property["id"] = fmt.Sprintf("%s-P%d", focusPath, index+1)
	}

	return writeSpecJSON(path, spec)
}

// rewriteEndpoint rewrites a single endpoint key in rel, trying byOldID first
// then byName. If neither resolves, the value is left unchanged (builder
// validation will catch genuine breakage).
func rewriteEndpoint(rel map[string]any, key string, byName, byOldID map[string]string) error {
	value, ok := rel[key].(string)
	if !ok {
		return fmt.Errorf("relationship missing %q field", key)
	}
	if newID, found := byOldID[value]; found {
		rel[key] = newID
		return nil
	}
	if newID, found := byName[value]; found {
		rel[key] = newID
	}
	return nil
}

// rewriteEndpointByName rewrites a single endpoint key in rel using a name→ID map.
// If the value is not found in the map it is left unchanged (already an ID or unknown).
func rewriteEndpointByName(rel map[string]any, key string, byName map[string]string) error {
	value, ok := rel[key].(string)
	if !ok {
		return fmt.Errorf("relationship missing %q field", key)
	}
	if newID, found := byName[value]; found {
		rel[key] = newID
	}
	return nil
}

// writeSpecJSON marshals spec to indented JSON and writes it with a trailing newline.
func writeSpecJSON(path string, spec map[string]any) error {
	encoded, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	encoded = append(encoded, '\n')
	if writeErr := os.WriteFile(path, encoded, specFileMode); writeErr != nil {
		return fmt.Errorf("write %s: %w", path, writeErr)
	}
	return nil
}
