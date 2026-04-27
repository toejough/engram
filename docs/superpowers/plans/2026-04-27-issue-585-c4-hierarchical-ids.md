# C4 Hierarchical Per-Owner IDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace flat global `E<n>` IDs in `architecture/c4/` with hierarchical per-owner IDs (`S<n>` / `N<n>` / `M<n>` / `P<n>` chained by hyphens), drop the `from_parent` field, and delete the global-namespace registry machinery.

**Architecture:** Each diagram owns the IDs it allocates. L1 owns `S<n>`, L2 owns `N<n>`, each L3 owns `M<n>`, each L4 owns `P<n>`. Cross-doc references use the full hierarchical path verbatim (e.g. `S2-N3-M5`). Edges (`R<n>`, `D<n>`) stay flat per-diagram. JSON is canonical; markdown is rendered.

**Tech Stack:** Go 1.x, `targ` build system, JSON specs in `architecture/c4/`, mermaid diagrams, no new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-27-issue-585-c4-hierarchical-ids-design.md`

---

## File structure

| Path | Action | Responsibility |
|---|---|---|
| `dev/c4_idpath.go` | Create | Parse/validate hierarchical path strings (`S2-N3-M5`); used by all builders and the audit. |
| `dev/c4_idpath_test.go` | Create | Table-driven tests for the path helper. |
| `dev/c4_migrate.go` | Create (throwaway) | One-shot `targ c4-migrate` target; rewrites all JSON specs from flat `E<n>` to hierarchical IDs. **Deleted in final task.** |
| `dev/c4.go` | Modify | L1 builder: assign `S<n>` IDs to elements, normalize edges to ID-based. |
| `dev/c4_l2.go` | Modify | L2 builder: assign `N<n>` for new containers, accept hierarchical paths for carried-over, drop `FromParent`. |
| `dev/c4_l3.go` | Modify | L3 builder: assign `M<n>` for new components, accept hierarchical paths for carried-over, drop `FromParent`. |
| `dev/c4_l4.go` | Modify | L4 builder: assign `P<n>` for new properties, accept hierarchical paths for siblings/ancestors, remove parent-registry load step. |
| `dev/c4_audit_ext.go` | Modify | Drop namespace-collision findings; accept hierarchical IDs in mermaid checks. |
| `dev/c4_registry.go` | Delete | Global-namespace registry; replaced by syntactic prefix validation in builders. |
| `dev/c4_registry_test.go` | Delete | Test for deleted file. |
| `architecture/c4/c1-*.json` (1) | Modify | Migrated by `c4-migrate`. |
| `architecture/c4/c2-*.json` (1) | Modify | Migrated. |
| `architecture/c4/c3-*.json` (3) | Modify | Migrated. |
| `architecture/c4/c4-*.json` (19) | Modify | Migrated. |
| `architecture/c4/*.md` (24) | Regenerated | Re-rendered by builders. |
| `skills/c4/SKILL.md` | Modify | Drop registry coordination guidance, document hierarchical path scheme. |
| `skills/c4/references/mermaid-conventions.md` | Modify | Update node-ID examples to hierarchical form. |
| `skills/c4/references/property-ledger-format.md` | Modify | Update P-ID examples. |
| `skills/c4/references/templates/*.md` | Modify | Update placeholder IDs. |

---

## Phase 1: ID-path helper

### Task 1: Add `IDPath` parser and validator

**Files:**
- Create: `dev/c4_idpath.go`
- Test: `dev/c4_idpath_test.go`

The IDPath helper is a tiny pure-Go module that all builders depend on. It parses a hyphen-separated hierarchical ID into segments, validates the letter sequence (`S`, then optional `N`, then optional `M`, then optional `P`), and exposes whether a path is an ancestor of another.

- [ ] **Step 1.1: Write failing tests**

```go
// dev/c4_idpath_test.go
//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestT1_IDPath_Parse(t *testing.T) {
	tests := []struct {
		input    string
		segments []string
		level    int // 1=S, 2=SN, 3=SNM, 4=SNMP
		ok       bool
	}{
		{"S1", []string{"S1"}, 1, true},
		{"S2-N3", []string{"S2", "N3"}, 2, true},
		{"S2-N3-M5", []string{"S2", "N3", "M5"}, 3, true},
		{"S2-N3-M5-P12", []string{"S2", "N3", "M5", "P12"}, 4, true},
		{"E27", nil, 0, false},                  // legacy flat
		{"S2-M5", nil, 0, false},                // skipped N
		{"N1", nil, 0, false},                   // missing S
		{"S2-N3-M5-P12-X1", nil, 0, false},      // too deep
		{"s2-n3", nil, 0, false},                // wrong case
		{"S2-N0", nil, 0, false},                // zero-numbered
		{"S2-N", nil, 0, false},                 // letter without number
		{"S2-N3a", nil, 0, false},               // letter suffix
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			path, err := ParseIDPath(test.input)
			if !test.ok {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}
			g.Expect(path.Segments).To(Equal(test.segments))
			g.Expect(path.Level).To(Equal(test.level))
		})
	}
}

func TestT2_IDPath_IsAncestorOf(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent, _ := ParseIDPath("S2-N3")
	child, _ := ParseIDPath("S2-N3-M5")
	sibling, _ := ParseIDPath("S2-N4")
	g.Expect(parent.IsAncestorOf(child)).To(BeTrue())
	g.Expect(child.IsAncestorOf(parent)).To(BeFalse())
	g.Expect(sibling.IsAncestorOf(child)).To(BeFalse())
	g.Expect(parent.IsAncestorOf(parent)).To(BeFalse())
}

func TestT3_IDPath_String(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	path, err := ParseIDPath("S2-N3-M5")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(path.String()).To(Equal("S2-N3-M5"))
}

func TestT4_IDPath_Append(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent, _ := ParseIDPath("S2-N3")
	child, err := parent.Append("M", 5)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(child.String()).To(Equal("S2-N3-M5"))

	// Wrong letter for level should fail
	_, err = parent.Append("P", 1) // L3 expects M, not P
	g.Expect(err).To(HaveOccurred())
}
```

- [ ] **Step 1.2: Run tests; expect fail**

Run: `targ test --pkg ./dev --run TestT1_IDPath`
Expected: undefined: ParseIDPath, IDPath.

- [ ] **Step 1.3: Implement `c4_idpath.go`**

```go
//go:build targ

package dev

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IDPath is a hierarchical C4 element identifier (e.g. "S2-N3-M5").
type IDPath struct {
	Segments []string // raw segments, e.g. ["S2","N3","M5"]
	Level    int      // 1..4: depth = number of segments
}

var (
	idPathLevelLetters = []string{"S", "N", "M", "P"}
	idPathSegmentRE    = regexp.MustCompile(`^([SNMP])([1-9][0-9]*)$`)
)

// ParseIDPath parses a hyphen-separated hierarchical path. The first segment
// must start with S; each subsequent segment uses the next letter in the
// fixed sequence S, N, M, P. Numbers must be positive integers without
// letter suffixes.
func ParseIDPath(input string) (IDPath, error) {
	if input == "" {
		return IDPath{}, fmt.Errorf("empty id path")
	}
	segments := strings.Split(input, "-")
	if len(segments) > len(idPathLevelLetters) {
		return IDPath{}, fmt.Errorf("id path %q has %d segments; max %d", input, len(segments), len(idPathLevelLetters))
	}
	for index, segment := range segments {
		match := idPathSegmentRE.FindStringSubmatch(segment)
		if match == nil {
			return IDPath{}, fmt.Errorf("segment %q in id path %q is not <Letter><N>", segment, input)
		}
		expected := idPathLevelLetters[index]
		if match[1] != expected {
			return IDPath{}, fmt.Errorf("segment %q in id path %q expected letter %q at depth %d", segment, input, expected, index+1)
		}
	}
	return IDPath{Segments: segments, Level: len(segments)}, nil
}

// String renders the canonical form.
func (path IDPath) String() string {
	return strings.Join(path.Segments, "-")
}

// IsAncestorOf reports whether this path is a strict prefix of `other`.
func (path IDPath) IsAncestorOf(other IDPath) bool {
	if len(path.Segments) >= len(other.Segments) {
		return false
	}
	for index, segment := range path.Segments {
		if other.Segments[index] != segment {
			return false
		}
	}
	return true
}

// Append adds a new segment at the next level. The letter must match the
// fixed sequence (S, N, M, P) for the resulting depth.
func (path IDPath) Append(letter string, number int) (IDPath, error) {
	if number <= 0 {
		return IDPath{}, fmt.Errorf("append: number must be positive, got %d", number)
	}
	nextLevel := path.Level + 1
	if nextLevel > len(idPathLevelLetters) {
		return IDPath{}, fmt.Errorf("append: cannot extend beyond level %d", len(idPathLevelLetters))
	}
	expected := idPathLevelLetters[nextLevel-1]
	if letter != expected {
		return IDPath{}, fmt.Errorf("append: level %d expects letter %q, got %q", nextLevel, expected, letter)
	}
	segment := letter + strconv.Itoa(number)
	newSegments := make([]string, 0, nextLevel)
	newSegments = append(newSegments, path.Segments...)
	newSegments = append(newSegments, segment)
	return IDPath{Segments: newSegments, Level: nextLevel}, nil
}
```

- [ ] **Step 1.4: Run tests; expect pass**

Run: `targ test --pkg ./dev --run TestT1_IDPath`
Expected: PASS for T1, T2, T3, T4.

- [ ] **Step 1.5: Lint clean**

Run: `targ check-full --pkg ./dev`
Expected: clean.

- [ ] **Step 1.6: Commit**

```bash
git add dev/c4_idpath.go dev/c4_idpath_test.go
git commit -m "$(cat <<'EOF'
feat(c4): add hierarchical IDPath parser/validator (#585)

AI-Used: [claude]
EOF
)"
```

---

## Phase 2: Migration script

### Task 2: Build the one-shot `c4-migrate` target

**Files:**
- Create: `dev/c4_migrate.go`

A throwaway targ target that walks `architecture/c4/c*.json` in dependency order (L1 → L2 → each L3 → each L4), assigns hierarchical IDs deterministically, builds a flat-`E<n>` → hierarchical-path map by following parent links, then rewrites every spec replacing:
- `focus.id` to the doc's full path
- each element's `id` (or assigning one for L1)
- each edge's `from`/`to` to ID-based references (resolving names against the elements list when needed)
- removing `from_parent`, `in_scope`, `is_system` boolean fields once IDs encode meaning (keep `in_scope` only if needed by L2 renderer; will revisit if Phase 3 needs it)

Determinism rule: numbering starts at 1 in JSON element order, ancestors first.

- [ ] **Step 2.1: Implement `dev/c4_migrate.go`**

```go
//go:build targ

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
		Description("THROWAWAY: Migrate flat E<n> IDs to hierarchical S/N/M/P paths in-place. Deleted after issue #585."))
}

// C4MigrateArgs configures the c4-migrate target.
type C4MigrateArgs struct {
	Dir string `targ:"flag,name=dir,desc=Directory to migrate (default architecture/c4)"`
}

// migrateMap holds the flat-E to hierarchical-path translation built up
// during a top-down walk.
type migrateMap struct {
	// keyed by the JSON file's basename; each file has its own E-to-path map
	// because L1's flat E-IDs are scoped only within L1's file.
	byFile map[string]map[string]string
}

func c4Migrate(ctx context.Context, args C4MigrateArgs) error {
	dir := args.Dir
	if dir == "" {
		dir = "architecture/c4"
	}
	// 1) Migrate L1: assign S-IDs to elements; rewrite edges to use IDs.
	l1Path := filepath.Join(dir, "c1-engram-system.json")
	l1Map, err := migrateL1(l1Path)
	if err != nil {
		return fmt.Errorf("migrate L1 %s: %w", l1Path, err)
	}
	// 2) Migrate L2: focus path = the L1 ID it refines (the in_scope element);
	//    new containers get N-IDs; carried-over elements get the L1 path.
	l2Path := filepath.Join(dir, "c2-engram-plugin.json")
	l2Map, err := migrateL2(l2Path, l1Map)
	if err != nil {
		return fmt.Errorf("migrate L2 %s: %w", l2Path, err)
	}
	// 3) Migrate each L3: focus path = the L2 N-ID it refines; new components
	//    get M-IDs prefixed by focus path; carried-over keep their existing
	//    hierarchical path (looked up in l2Map).
	l3Files, err := filepath.Glob(filepath.Join(dir, "c3-*.json"))
	if err != nil {
		return fmt.Errorf("glob L3: %w", err)
	}
	sort.Strings(l3Files)
	l3Maps := map[string]map[string]string{}
	for _, file := range l3Files {
		l3Map, migrateErr := migrateL3(file, l2Map)
		if migrateErr != nil {
			return fmt.Errorf("migrate L3 %s: %w", file, migrateErr)
		}
		l3Maps[filepath.Base(file)] = l3Map
	}
	// 4) Migrate each L4: parent file is named in spec; look up focus and
	//    sibling/ancestor IDs in that file's L3 map; assign P-IDs for new.
	l4Files, err := filepath.Glob(filepath.Join(dir, "c4-*.json"))
	if err != nil {
		return fmt.Errorf("glob L4: %w", err)
	}
	sort.Strings(l4Files)
	for _, file := range l4Files {
		if migrateErr := migrateL4(file, l3Maps); migrateErr != nil {
			return fmt.Errorf("migrate L4 %s: %w", file, migrateErr)
		}
	}
	return nil
}

// migrateL1 assigns S<n> to L1 elements in JSON order, rewrites edges from
// names to IDs, and returns the name→path map for use by L2.
func migrateL1(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	elements, _ := spec["elements"].([]any)
	nameToID := map[string]string{}
	for index, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		id := fmt.Sprintf("S%d", index+1)
		element["id"] = id
		name, _ := element["name"].(string)
		nameToID[name] = id
		// drop fields no longer needed
		delete(element, "is_system")
	}
	rels, _ := spec["relationships"].([]any)
	for _, relAny := range rels {
		rel, _ := relAny.(map[string]any)
		if from, ok := rel["from"].(string); ok {
			if id, ok := nameToID[from]; ok {
				rel["from"] = id
			}
		}
		if to, ok := rel["to"].(string); ok {
			if id, ok := nameToID[to]; ok {
				rel["to"] = id
			}
		}
	}
	return nameToID, writeJSON(path, spec)
}

// migrateL2 assigns N<n> to in-scope elements (or new containers), uses the
// L1 path for carried-over elements, normalizes edges, and returns a map
// keyed by both Name and the previous flat E-ID for L3 lookup.
func migrateL2(path string, l1Map map[string]string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	// Find the in_scope element; its new ID is its L1 path.
	elements, _ := spec["elements"].([]any)
	resolveByOldID := map[string]string{}
	resolveByName := map[string]string{}
	// Determine the focus's L1 path (the in_scope element).
	var focusPath string
	for _, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		isInScope, _ := element["in_scope"].(bool)
		if isInScope {
			name, _ := element["name"].(string)
			if pathLookup, ok := l1Map[name]; ok {
				focusPath = pathLookup
			}
		}
	}
	if focusPath == "" {
		return nil, fmt.Errorf("L2 has no in_scope element resolvable to L1")
	}
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
			// Carried over from L1; use the L1 path by name lookup.
			pathLookup, ok := l1Map[name]
			if !ok {
				return nil, fmt.Errorf("L2 element %q (id=%s) marked from_parent but not in L1", name, oldID)
			}
			newID = pathLookup
		default:
			// New container.
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
		if err := rewriteEndpoint(rel, "from", resolveByName, resolveByOldID); err != nil {
			return nil, err
		}
		if err := rewriteEndpoint(rel, "to", resolveByName, resolveByOldID); err != nil {
			return nil, err
		}
	}
	if err := writeJSON(path, spec); err != nil {
		return nil, err
	}
	// Merge maps for L3 lookup.
	merged := map[string]string{}
	for key, value := range resolveByOldID {
		merged[key] = value
	}
	for key, value := range resolveByName {
		merged[key] = value
	}
	return merged, nil
}

// migrateL3 mirrors migrateL2 with the focus inheriting L2 path,
// new components getting M-IDs.
func migrateL3(path string, l2Map map[string]string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	focus, _ := spec["focus"].(map[string]any)
	focusOldID, _ := focus["id"].(string)
	focusPath, ok := l2Map[focusOldID]
	if !ok {
		return nil, fmt.Errorf("L3 focus %s not found in L2 map", focusOldID)
	}
	focus["id"] = focusPath

	elements, _ := spec["elements"].([]any)
	resolveByOldID := map[string]string{}
	resolveByName := map[string]string{}
	componentCounter := 0
	for _, elementAny := range elements {
		element, _ := elementAny.(map[string]any)
		oldID, _ := element["id"].(string)
		name, _ := element["name"].(string)
		fromParent, _ := element["from_parent"].(bool)
		var newID string
		if fromParent {
			pathLookup, found := l2Map[oldID]
			if !found {
				pathLookup, found = l2Map[name]
			}
			if !found {
				return nil, fmt.Errorf("L3 element %q (id=%s) marked from_parent not in L2", name, oldID)
			}
			newID = pathLookup
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
		if err := rewriteEndpoint(rel, "from", resolveByName, resolveByOldID); err != nil {
			return nil, err
		}
		if err := rewriteEndpoint(rel, "to", resolveByName, resolveByOldID); err != nil {
			return nil, err
		}
	}
	if err := writeJSON(path, spec); err != nil {
		return nil, err
	}
	merged := map[string]string{}
	for key, value := range resolveByOldID {
		merged[key] = value
	}
	for key, value := range resolveByName {
		merged[key] = value
	}
	return merged, nil
}

// migrateL4 looks up parent file from spec.parent and consults that
// L3 map for focus and ancestor element IDs; new properties are P<n>.
func migrateL4(path string, l3Maps map[string]map[string]string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var spec map[string]any
	if unmarshalErr := json.Unmarshal(raw, &spec); unmarshalErr != nil {
		return unmarshalErr
	}
	parent, _ := spec["parent"].(string) // e.g. "c3-engram-cli-binary.md"
	parentJSON := strings.TrimSuffix(parent, ".md") + ".json"
	parentMap, ok := l3Maps[parentJSON]
	if !ok {
		return fmt.Errorf("L4 parent %s not migrated", parentJSON)
	}
	focus, _ := spec["focus"].(map[string]any)
	focusOldID, _ := focus["id"].(string)
	focusPath, ok := parentMap[focusOldID]
	if !ok {
		return fmt.Errorf("L4 focus %s not in parent map", focusOldID)
	}
	focus["id"] = focusPath
	delete(focus, "l3_container") // redundant once focus.id is hierarchical

	// Diagram nodes: every node references either a sibling (in parent map)
	// or the focus itself.
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
			return fmt.Errorf("L4 node %s in %s not in parent map", oldID, path)
		}
		edges, _ := diagram["edges"].([]any)
		for _, edgeAny := range edges {
			edge, _ := edgeAny.(map[string]any)
			for _, end := range []string{"from", "to"} {
				oldEndpoint, _ := edge[end].(string)
				if oldEndpoint == focusOldID {
					edge[end] = focusPath
					continue
				}
				if newEndpoint, found := parentMap[oldEndpoint]; found {
					edge[end] = newEndpoint
				}
			}
		}
	}
	// Dependency manifest references parent IDs by `wired_by_id`; rewrite.
	manifest, _ := spec["dependency_manifest"].([]any)
	for _, rowAny := range manifest {
		row, _ := rowAny.(map[string]any)
		if oldID, ok := row["wired_by_id"].(string); ok {
			if newID, found := parentMap[oldID]; found {
				row["wired_by_id"] = newID
			}
		}
	}
	// Properties get sequential P<n> in JSON order.
	properties, _ := spec["properties"].([]any)
	for index, propertyAny := range properties {
		property, _ := propertyAny.(map[string]any)
		property["id"] = fmt.Sprintf("%s-P%d", focusPath, index+1)
	}
	return writeJSON(path, spec)
}

func rewriteEndpoint(rel map[string]any, key string, byName, byOldID map[string]string) error {
	value, ok := rel[key].(string)
	if !ok {
		return fmt.Errorf("relationship %s missing", key)
	}
	if newID, found := byOldID[value]; found {
		rel[key] = newID
		return nil
	}
	if newID, found := byName[value]; found {
		rel[key] = newID
		return nil
	}
	// Already an ID we couldn't resolve — leave it; builder validation
	// will catch genuine breakage.
	return nil
}

func writeJSON(path string, spec map[string]any) error {
	encoded, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}
```

- [ ] **Step 2.2: Build to verify it compiles**

Run: `targ build`
Expected: clean build.

- [ ] **Step 2.3: Commit**

```bash
git add dev/c4_migrate.go
git commit -m "$(cat <<'EOF'
feat(c4): add throwaway c4-migrate target for #585

AI-Used: [claude]
EOF
)"
```

### Task 3: Run migration on all JSON specs

**Files:**
- Modify: all `architecture/c4/c*.json` (in place via migration tool)

- [ ] **Step 3.1: Backup current state for diffing**

```bash
cp -r architecture/c4 /tmp/c4-pre-585
```

- [ ] **Step 3.2: Run migration**

Run: `targ c4-migrate --dir architecture/c4`
Expected: no error.

- [ ] **Step 3.3: Spot-check JSON**

```bash
jq '.elements[] | {id, name}' architecture/c4/c1-engram-system.json | head
jq '.focus' architecture/c4/c3-engram-cli-binary.json
jq '.elements[] | {id, name}' architecture/c4/c3-engram-cli-binary.json | head
jq '.focus' architecture/c4/c4-recall.json
jq '.diagram.nodes[] | {id, name}' architecture/c4/c4-recall.json
```

Expected:
- L1 elements have ids `S1`–`S6`.
- L3 focus.id is `S2-N3` (or whatever the engram CLI binary container path is).
- L3 elements: carried-over use S-IDs, new components use `S2-N3-M1` … `S2-N3-M9`.
- L4 focus.id like `S2-N3-M5`; sibling node IDs all use the parent path prefix.
- No `from_parent` field anywhere.

- [ ] **Step 3.4: Commit migrated JSON**

```bash
git add architecture/c4
git commit -m "$(cat <<'EOF'
refactor(c4): migrate JSON specs to hierarchical S/N/M/P IDs (#585)

Mechanical rewrite via targ c4-migrate; markdown will be regenerated
in subsequent commits as builders are updated to match.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3: Update builders

After this phase, each builder reads the new JSON shape, validates hierarchical IDs syntactically, and emits markdown using the new IDs. The strategy per builder:

1. Update `LxElement` / `LxSpec` Go structs (drop `FromParent`).
2. Replace any auto-assignment of E-IDs with validation that explicit hierarchical IDs are present and well-formed via `ParseIDPath`.
3. Replace markdown-anchor generation to use the full hierarchical ID.
4. Update tests' fixtures and expectations.
5. Re-run `targ c4-l<n>-build` against real specs and commit regenerated `.md`.

### Task 4: Update L1 builder

**Files:**
- Modify: `dev/c4.go` (L1 build path)
- Modify: `dev/c4_test.go` (any L1 fixtures)
- Regen: `architecture/c4/c1-engram-system.md`

- [ ] **Step 4.1: Update tests first**

Update L1 tests to assert:
- elements have explicit `S<n>` IDs (no auto-assign needed since migration already added them);
- markdown anchors are `s<n>-<slug>` not `e<n>-<slug>`;
- relationships render with IDs not names in any embedded tables;
- `is_system` field is no longer expected on inputs.

(Run `grep -n "func TestT.*L1\|c4L1Build" dev/c4_test.go` to find the existing test; mirror its shape.)

- [ ] **Step 4.2: Run failing tests**

Run: `targ test --pkg ./dev --run L1`
Expected: failures showing old E-IDs / e-anchors.

- [ ] **Step 4.3: Update L1 builder code**

In `dev/c4.go`:
- In any element-rendering path: read `element.ID` (now an `S<n>`), use it verbatim, generate anchors like `s1-<slug>` lowercased.
- Remove any `is_system` handling.
- Validate every element has a non-empty ID via `ParseIDPath`; on parse error, error out.

- [ ] **Step 4.4: Run tests until green**

Run: `targ test --pkg ./dev --run L1`
Expected: PASS.

- [ ] **Step 4.5: Regenerate `c1-engram-system.md`**

Run: `targ c4-l1-build --input architecture/c4/c1-engram-system.json --noconfirm`
Expected: file rewritten.

- [ ] **Step 4.6: Audit the rendered file**

Run: `targ c4-audit --file architecture/c4/c1-engram-system.md`
Expected: zero findings (after Phase 4's audit changes — *until then* expect namespace findings; if new errors appear that aren't namespace-related, fix here).

- [ ] **Step 4.7: Commit**

```bash
git add dev/c4.go dev/c4_test.go architecture/c4/c1-engram-system.md
git commit -m "$(cat <<'EOF'
feat(c4): L1 builder emits S<n> IDs and hierarchical anchors (#585)

AI-Used: [claude]
EOF
)"
```

### Task 5: Update L2 builder

**Files:**
- Modify: `dev/c4_l2.go` (drop `FromParent`, drop ID auto-assignment, validate hierarchical IDs)
- Modify: `dev/c4_l2_test.go` (rewrite fixtures using S/N IDs)
- Regen: `architecture/c4/c2-engram-plugin.md`

- [ ] **Step 5.1: Update L2 struct**

In `dev/c4_l2.go`, change:

```go
type L2Element struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	SystemOfRecord string  `json:"system_of_record"`
	InScope        bool    `json:"in_scope,omitempty"`
}
```

(Removed `FromParent`. ID is now required.)

- [ ] **Step 5.2: Replace `assignL2ElementIDs` with `validateL2ElementIDs`**

Replace the function. New behavior: assert every element has a non-empty ID, parseable via `ParseIDPath`. The element with `InScope=true` must have an `S<n>` ID (level 1, ancestor). Other elements either have `S<n>` (carried over) or `<focusPath>-N<m>` (new container).

```go
func validateL2ElementIDs(elements []L2Element) error {
	var focusPath IDPath
	for _, element := range elements {
		path, err := ParseIDPath(element.ID)
		if err != nil {
			return fmt.Errorf("element %q: %w", element.Name, err)
		}
		if element.InScope {
			if path.Level != 1 {
				return fmt.Errorf("in_scope element %q must have an L1 (S<n>) id, got %s", element.Name, element.ID)
			}
			focusPath = path
		}
	}
	if focusPath.Level == 0 {
		return fmt.Errorf("L2 spec has no in_scope element")
	}
	for _, element := range elements {
		if element.InScope {
			continue
		}
		path, _ := ParseIDPath(element.ID)
		switch path.Level {
		case 1:
			// carried over from L1 (e.g. external/person)
		case 2:
			if !focusPath.IsAncestorOf(path) {
				return fmt.Errorf("element %q id %s is not under focus %s", element.Name, element.ID, focusPath.String())
			}
		default:
			return fmt.Errorf("element %q has unsupported L2 id depth %d (%s)", element.Name, path.Level, element.ID)
		}
	}
	return nil
}
```

- [ ] **Step 5.3: Update markdown rendering and anchor logic**

Anywhere L2 emitted `E<n>` or `e<n>-<slug>`, use `element.ID` and `strings.ToLower(strings.ReplaceAll(element.ID, "-", "-")) + "-" + slug`. Drop `from_parent`-conditional formatting.

- [ ] **Step 5.4: Update `dev/c4_l2_test.go`**

Rewrite fixtures and assertions to use the new shape. Tests `TestT19_L2AssignIDs_CarryOverFromParent` and `TestT* L2*FromParent*` no longer apply — replace with:

- `TestT19_L2ValidateIDs_RequiresInScope`: spec without InScope returns error.
- `TestT20_L2ValidateIDs_AcceptsCarriedAncestor`: InScope=`S2`, peers=`S1`,`S3`,`N1`,`N2` validates.
- `TestT21_L2ValidateIDs_RejectsBadDepth`: element id `S2-N3-M5` rejected (level 3 in L2 spec).
- Existing fixtures under `dev/testdata/c4/` need parallel updates; the file `invalid_l2_in_scope_not_from_parent.json` becomes `invalid_l2_no_in_scope.json` etc.

- [ ] **Step 5.5: Run tests; fix until green**

Run: `targ test --pkg ./dev --run L2`
Expected: PASS.

- [ ] **Step 5.6: Regenerate `c2-engram-plugin.md`**

Run: `targ c4-l2-build --input architecture/c4/c2-engram-plugin.json --noconfirm`
Expected: file rewritten with hierarchical IDs.

- [ ] **Step 5.7: Commit**

```bash
git add dev/c4_l2.go dev/c4_l2_test.go dev/testdata/c4 architecture/c4/c2-engram-plugin.md
git commit -m "$(cat <<'EOF'
feat(c4): L2 builder emits N<n> IDs, drops from_parent (#585)

AI-Used: [claude]
EOF
)"
```

### Task 6: Update L3 builder

**Files:**
- Modify: `dev/c4_l3.go` (mirror L2 change pattern)
- Modify: `dev/c4_l3_test.go`
- Regen: `architecture/c4/c3-*.md` (3 files)

- [ ] **Step 6.1: Update `L3Element` struct**

Drop `FromParent`. ID is required.

- [ ] **Step 6.2: Replace registry-based validation with `validateL3ElementIDs`**

Same shape as `validateL2ElementIDs` but at one level deeper:
- Focus.ID is parsed; must be level 2 (`S<n>-N<n>`).
- Each element's path is either an ancestor of focus (depth 1 or 2) or focus + new `M<n>` (depth 3).
- Error if otherwise.

Delete the `from_parent` name-mismatch check (was: `T48_L3BuildRegistryRejection_FromParentNameMismatch`).

- [ ] **Step 6.3: Update L3 markdown rendering and anchors**

Use `element.ID` verbatim; lowercase for anchors.

- [ ] **Step 6.4: Update `dev/c4_l3_test.go`**

Rewrite `TestT48_*FromParent*` and any registry-related test to assert the new validation. Update `dev/testdata/c4/*l3*.json` fixtures.

- [ ] **Step 6.5: Run tests**

Run: `targ test --pkg ./dev --run L3`
Expected: PASS.

- [ ] **Step 6.6: Regenerate L3 markdown**

```bash
for spec in architecture/c4/c3-*.json; do
  targ c4-l3-build --input "$spec" --noconfirm
done
```

- [ ] **Step 6.7: Commit**

```bash
git add dev/c4_l3.go dev/c4_l3_test.go dev/testdata/c4 architecture/c4/c3-*.md
git commit -m "$(cat <<'EOF'
feat(c4): L3 builder emits M<n> IDs, drops from_parent and registry call (#585)

AI-Used: [claude]
EOF
)"
```

### Task 7: Update L4 builder

**Files:**
- Modify: `dev/c4_l4.go` (drop registry-load step, accept hierarchical IDs)
- Modify: `dev/c4_l4_test.go`
- Regen: `architecture/c4/c4-*.md` (19 files)

- [ ] **Step 7.1: Update `L4Spec` shape**

- `L4Focus` simplifies: drop `L3Container` field (the focus path encodes it).
- L4Node IDs are now hierarchical paths.
- Edge IDs remain `R<n>`/`D<n>` flat.

- [ ] **Step 7.2: Drop registry-load logic**

In `dev/c4_l4.go`, remove the call to `scanRegistryDir` (or whatever loads the parent L3 spec). Replace L4 node-validation with:

```go
func validateL4Nodes(focus IDPath, nodes []L4Node) error {
	for _, node := range nodes {
		path, err := ParseIDPath(node.ID)
		if err != nil {
			return fmt.Errorf("node %q: %w", node.ID, err)
		}
		// node must be focus, an ancestor of focus, a sibling at focus's level,
		// or a child of focus's parent (i.e. depth ≤ focus.Level and either
		// matches focus exactly or shares prefix with focus's parent).
		if path.String() == focus.String() {
			continue
		}
		if path.IsAncestorOf(focus) {
			continue
		}
		// sibling: same depth as focus, shares all-but-last segment with focus
		if path.Level == focus.Level && shareParent(path, focus) {
			continue
		}
		return fmt.Errorf("node %s in L4 diagram is neither focus, ancestor, nor sibling", node.ID)
	}
	return nil
}

func shareParent(a, b IDPath) bool {
	if a.Level != b.Level || a.Level == 0 {
		return false
	}
	for index := 0; index < a.Level-1; index++ {
		if a.Segments[index] != b.Segments[index] {
			return false
		}
	}
	return true
}
```

Edge validation remains: `from`/`to` must reference a node ID present in `diagram.nodes`, plus the `^R\d+$` / `^D\d+$` syntactic rule per existing code.

Properties get sequential P<n> validation: each property `id` must be `<focus>-P<n>` with sequential numbering starting at 1.

- [ ] **Step 7.3: Update L4 markdown rendering and anchors**

Use `node.ID` verbatim. Anchors lowercased: `#s2-n3-m5-recall` for the focus heading; properties anchored as `#s2-n3-m5-p1-<slug>`.

- [ ] **Step 7.4: Update `dev/c4_l4_test.go`**

Rewrite fixtures with hierarchical IDs. Replace registry-dependent tests with the new sibling/ancestor checks.

- [ ] **Step 7.5: Run tests**

Run: `targ test --pkg ./dev --run L4`
Expected: PASS.

- [ ] **Step 7.6: Regenerate all L4 markdown**

```bash
for spec in architecture/c4/c4-*.json; do
  targ c4-l4-build --input "$spec" --noconfirm
done
```

- [ ] **Step 7.7: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go dev/testdata/c4 architecture/c4/c4-*.md
git commit -m "$(cat <<'EOF'
feat(c4): L4 builder emits P<n> IDs, drops parent registry load (#585)

AI-Used: [claude]
EOF
)"
```

---

## Phase 4: Update audit + render

### Task 8: Update audit to accept hierarchical IDs and drop namespace findings

**Files:**
- Modify: `dev/c4_audit_ext.go`
- Modify: `dev/c4_audit_ext_test.go`

- [ ] **Step 8.1: Identify all uses of `E<n>` regex and namespace-collision logic**

Run: `grep -n 'E\\\\d\|E[0-9]\|namespace\|registry' dev/c4_audit_ext.go`

- [ ] **Step 8.2: Replace ID-shape regex**

Anywhere the audit rejects mermaid node IDs not matching `^E\d+$`, switch to `ParseIDPath`. Edge IDs continue to enforce `^R\d+$` / `^D\d+$`.

- [ ] **Step 8.3: Drop `name_id_split` / `id_collision` cross-file checks**

These were the c4-registry's responsibility. Remove the audit findings that mentioned them.

- [ ] **Step 8.4: Update audit tests**

Tests that asserted the audit caught flat-E mismatches should be replaced by tests asserting it accepts hierarchical IDs and rejects malformed paths.

- [ ] **Step 8.5: Run tests**

Run: `targ test --pkg ./dev --run Audit`
Expected: PASS.

- [ ] **Step 8.6: Audit every rendered file**

```bash
for md in architecture/c4/c*.md; do
  targ c4-audit --file "$md" || echo "FAILED: $md"
done
```

Expected: all clean.

- [ ] **Step 8.7: Commit**

```bash
git add dev/c4_audit_ext.go dev/c4_audit_ext_test.go
git commit -m "$(cat <<'EOF'
feat(c4): audit accepts hierarchical ids, drops namespace findings (#585)

AI-Used: [claude]
EOF
)"
```

### Task 9: Re-render mermaid SVGs

- [ ] **Step 9.1: Run render**

Run: `targ c4-render`
Expected: all `.mmd` files re-rendered to `.svg` (mermaid CLI invocation).

- [ ] **Step 9.2: Spot-check one SVG opens cleanly**

```bash
ls architecture/c4/svg/*.svg | head -3
```

- [ ] **Step 9.3: Commit**

```bash
git add architecture/c4/svg
git commit -m "$(cat <<'EOF'
chore(c4): re-render SVGs after hierarchical ID migration (#585)

AI-Used: [claude]
EOF
)"
```

---

## Phase 5: Delete legacy machinery

### Task 10: Delete `c4-registry` and global-namespace plumbing

**Files:**
- Delete: `dev/c4_registry.go`
- Delete: `dev/c4_registry_test.go`

- [ ] **Step 10.1: Confirm nothing still references it**

Run:
```bash
grep -rn 'c4Registry\|c4-registry\|RegistryView\|scanRegistryDir' dev/ skills/
```

If anything outside `c4_registry.go` itself references it, fix the caller first.

- [ ] **Step 10.2: Delete the files**

```bash
rm dev/c4_registry.go dev/c4_registry_test.go
```

- [ ] **Step 10.3: Verify build + tests**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 10.4: Commit**

```bash
git add dev/
git commit -m "$(cat <<'EOF'
refactor(c4): delete c4-registry; superseded by hierarchical IDs (#585)

AI-Used: [claude]
EOF
)"
```

### Task 11: Delete the migration tool

**Files:**
- Delete: `dev/c4_migrate.go`

- [ ] **Step 11.1: Delete**

```bash
rm dev/c4_migrate.go
```

- [ ] **Step 11.2: Verify build clean**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 11.3: Commit**

```bash
git add dev/
git commit -m "$(cat <<'EOF'
chore(c4): delete throwaway c4-migrate target (#585)

AI-Used: [claude]
EOF
)"
```

---

## Phase 6: Skill + reference docs

### Task 12: Update `skills/c4/SKILL.md`

**Files:**
- Modify: `skills/c4/SKILL.md`

- [ ] **Step 12.1: Identify sections to rewrite**

Run:
```bash
grep -n 'c4-registry\|E<n>\|E-id\|E[0-9]\|registry' skills/c4/SKILL.md
```

- [ ] **Step 12.2: Replace the global-E-namespace coordination section**

Drop the "Run targ c4-registry to learn which E-IDs are taken" guidance. Replace with: "Each diagram owns its own ID space. L1 owns `S<n>`, L2 owns `N<n>`, L3 owns `M<n>`, L4 owns `P<n>`. When referencing an element owned by a higher level, copy that element's full hierarchical path verbatim from the parent doc."

- [ ] **Step 12.3: Update example IDs throughout**

Any `E27`/`E22` examples become hierarchical paths. Anchor examples become `#s2-n3-m9-tokenresolver`.

- [ ] **Step 12.4: Drop the `c4-registry conflict-free` step in propagation discipline**

Replace with: "Run `targ c4-audit` on every affected file."

- [ ] **Step 12.5: Run skill tests if any**

Run: `targ test --pkg ./skills/c4/tests/...` (if any tests exist there)
Expected: PASS or no-op.

- [ ] **Step 12.6: Commit**

```bash
git add skills/c4/SKILL.md
git commit -m "$(cat <<'EOF'
docs(c4-skill): document hierarchical IDs, drop registry guidance (#585)

AI-Used: [claude]
EOF
)"
```

### Task 13: Update reference docs

**Files:**
- Modify: `skills/c4/references/mermaid-conventions.md`
- Modify: `skills/c4/references/property-ledger-format.md`
- Modify: `skills/c4/references/templates/*.md`

- [ ] **Step 13.1: Update mermaid-conventions.md**

Replace E/R/D node-ID examples to use hierarchical S/N/M/P node IDs (edges keep R/D flat). Update the build-time-validation paragraph that mentions "E<n>" to say "the doc's hierarchical ID scheme."

- [ ] **Step 13.2: Update property-ledger-format.md**

Anywhere a property ID was `P1` / `P2` / `P3`, show the hierarchical form `S2-N3-M5-P1` etc.

- [ ] **Step 13.3: Update template files**

`skills/c4/references/templates/c1-template.md`, `c3-template.md` etc.: replace `<E1>` placeholders with `<S1>` / `<S2-N1>` / `<S2-N3-M1>` / `<S2-N3-M5-P1>` as appropriate to level.

- [ ] **Step 13.4: Commit**

```bash
git add skills/c4/references
git commit -m "$(cat <<'EOF'
docs(c4-skill): update references and templates for hierarchical IDs (#585)

AI-Used: [claude]
EOF
)"
```

---

## Final verification

### Task 14: End-to-end clean check

- [ ] **Step 14.1: Full check**

Run: `targ check-full`
Expected: zero findings, all tests pass.

- [ ] **Step 14.2: Audit every C4 markdown**

```bash
for md in architecture/c4/c*.md; do
  targ c4-audit --file "$md" || { echo "FAILED: $md"; exit 1; }
done
echo "All audits clean"
```

- [ ] **Step 14.3: Diff report against pre-migration state**

```bash
diff -r /tmp/c4-pre-585 architecture/c4 | head -200 > /tmp/c4-585-diff.txt
echo "Spot-check /tmp/c4-585-diff.txt for unexpected content changes (only IDs and anchors should differ)"
```

- [ ] **Step 14.4: Cleanup**

```bash
rm -rf /tmp/c4-pre-585 /tmp/c4-585-diff.txt
```

- [ ] **Step 14.5: Verify acceptance criteria from spec**

| AC | Check |
|---|---|
| Hierarchical IDs in all JSON | `! grep -E '"E[0-9]+"' architecture/c4/*.json` returns nothing |
| `targ c4-audit` zero findings | done in 14.2 |
| `targ c4-l*-build` all pass | re-run as smoke |
| `targ check-full` clean | done in 14.1 |
| `c4_registry.go` deleted | `! test -f dev/c4_registry.go` |
| `from_parent` removed | `! grep -rn 'from_parent\|FromParent' dev/ architecture/c4/` |
| SKILL.md updated | manual review |
| Migration tool deleted | `! test -f dev/c4_migrate.go` |
| Anchors resolve | spot-check by clicking 3-5 cross-file links |

- [ ] **Step 14.6: Final summary commit if any cleanup happened, otherwise no-op**

(Only if Step 14.3+ surfaced minor doc fixes.)

---

## Self-review notes

- **Spec coverage:** All 9 acceptance criteria map to verification in Task 14. Naming scheme and codebase simplifications are addressed in Phase 3 (builders) and Phase 5 (deletions). Migration approach matches spec.
- **Type consistency:** `IDPath` introduced in Task 1, used by name in all subsequent validators. `validateL2ElementIDs` / `validateL3ElementIDs` / `validateL4Nodes` use a parallel pattern; signatures distinct per level.
- **Risk areas:** the migration script's name-resolution heuristic at L2 (`from_parent` lookup by name into L1 map). If L1 has duplicate names, this fails. Pre-mitigation: `targ c4-audit` already requires unique names within a doc; cross-doc duplicates are caught only after migration. The migration's pre-check that `l1Map[name]` is non-empty surfaces this.
- **What's not covered:** rename-tracking layer, SVG verbosity trimming — both explicitly out of scope per spec.
