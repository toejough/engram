# Issue #605 — L4↔L3 Cross-Level Carryover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce the L4↔L3 cross-level invariant — every L4 node exists on the L3 parent with matching kind, and every L3 node connected to the focus's L3 representation appears on L4. Block builds, emit audit findings, and bring all 19 existing L4 specs into compliance.

**Architecture:** New `validateL4Carryover(l4, l3) error` and `loadL3Parent(l4, dir)` helpers in `dev/c4_l4.go`. `validateL4Spec` gains an `l3 *L3Spec` parameter; `c4L4Build` loads the parent before validation. `auditFile` extends its L4 branch to load both JSONs and emit one `l4_carryover` finding per leaf error from `errors.Join`.

**Tech Stack:** Go 1.x, `targ` build tool, `errors.Join`/`Unwrap() []error`.

**Spec:** `docs/superpowers/specs/2026-04-28-issue-605-c4-l4-l3-cross-level-carryover-design.md`

**Branch state:** worktree `/Users/joe/repos/personal/engram/.worktrees/issue-605` on branch `issue-605-c4-l4-l3-carryover`, branched from main at `f73b20b1` (spec only — no implementation yet).

---

## Pre-flight

- [ ] **Step P-1: Verify clean baseline**

```bash
cd /Users/joe/repos/personal/engram/.worktrees/issue-605
git status                          # clean
git log --oneline main..HEAD        # empty
targ check-full
targ test
```

Expected: all green. STOP if not.

---

## Task 1: `loadL3Parent` helper

**Files:** `dev/c4_l4.go`, `dev/c4_l4_test.go`.

- [ ] **Step 1.1: Add failing test in `dev/c4_l4_test.go`**

```go
func TestLoadL3Parent_ReadsSiblingJSON(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    l3JSON := []byte(`{"schema_version":"1","level":3,"name":"engram-cli-binary","parent":"c2-engram-plugin.md","focus":{"id":"S2-N3","name":"engram CLI binary","responsibility":"x"},"elements":[],"relationships":[]}`)
    if err := os.WriteFile(filepath.Join(dir, "c3-engram-cli-binary.json"), l3JSON, 0o600); err != nil {
        t.Fatalf("write: %v", err)
    }
    l4 := &L4Spec{Parent: "c3-engram-cli-binary.md"}
    l3, err := loadL3Parent(l4, dir)
    if err != nil {
        t.Fatalf("load: %v", err)
    }
    if l3 == nil || l3.Name != "engram-cli-binary" {
        t.Fatalf("bad load: %+v", l3)
    }
}

func TestLoadL3Parent_MissingFileWrapsError(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    l4 := &L4Spec{Parent: "c3-nope.md"}
    _, err := loadL3Parent(l4, dir)
    if err == nil || !strings.Contains(err.Error(), "c3-nope.json") {
        t.Fatalf("expected wrapped error mentioning c3-nope.json, got: %v", err)
    }
}
```

- [ ] **Step 1.2: Confirm tests fail**

```bash
targ test 2>&1 | grep -E 'TestLoadL3Parent|FAIL'
```

Expected: undefined `loadL3Parent`.

- [ ] **Step 1.3: Implement `loadL3Parent`**

Add near the other L4-spec helpers in `dev/c4_l4.go`:

```go
// loadL3Parent reads the L3 spec sibling of an L4 spec from dirPath. The
// filename is derived from l4.Parent by replacing the .md suffix with .json.
func loadL3Parent(l4 *L4Spec, dirPath string) (*L3Spec, error) {
    parentJSON := strings.TrimSuffix(l4.Parent, ".md") + ".json"
    fullPath := filepath.Join(dirPath, parentJSON)
    raw, err := os.ReadFile(fullPath)
    if err != nil {
        return nil, fmt.Errorf("loading L3 parent %q: %w", parentJSON, err)
    }
    decoder := json.NewDecoder(bytes.NewReader(raw))
    decoder.DisallowUnknownFields()
    var spec L3Spec
    if err := decoder.Decode(&spec); err != nil {
        return nil, fmt.Errorf("decoding L3 parent %q: %w", parentJSON, err)
    }
    return &spec, nil
}
```

- [ ] **Step 1.4: Confirm tests pass + check-full green**

```bash
targ test 2>&1 | grep -E 'TestLoadL3Parent|FAIL|PASS' | head
targ check-full
```

- [ ] **Step 1.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): loadL3Parent helper for cross-level checks

Reads the L3 spec sibling of an L4 spec by translating l4.Parent
(*.md) to *.json in the same directory. Wraps errors with the
parent filename for diagnostics. Used by the upcoming carryover
validator.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: `validateL4Carryover` — focus existence + L4→L3 direction

**Files:** `dev/c4_l4.go`, `dev/c4_l4_test.go`.

- [ ] **Step 2.1: Add failing tests**

```go
func TestValidateL4Carryover_FocusMissingFromL3(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{Focus: L4Focus{ID: "S2-N3-M3", Name: "recall"}, Parent: "c3-x.md"}
    l3 := &L3Spec{Elements: []L3Element{}}
    err := validateL4Carryover(l4, l3)
    if err == nil || !strings.Contains(err.Error(), "S2-N3-M3") {
        t.Fatalf("expected focus-id error, got: %v", err)
    }
}

func TestValidateL4Carryover_L4ExtraNode(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}, {ID: "X", Name: "ghost", Kind: "component"}}},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}},
    }
    err := validateL4Carryover(l4, l3)
    if err == nil || !strings.Contains(err.Error(), `"X"`) {
        t.Fatalf("expected extra-node error citing X, got: %v", err)
    }
}

func TestValidateL4Carryover_KindMismatch(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}, {ID: "S5", Name: "anthropic", Kind: "component"}}},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "S5", Name: "anthropic", Kind: "external"}},
    }
    err := validateL4Carryover(l4, l3)
    if err == nil || !strings.Contains(err.Error(), "kind") || !strings.Contains(err.Error(), "S5") {
        t.Fatalf("expected kind mismatch citing S5, got: %v", err)
    }
}
```

NOTE: the focus on L4 has `kind: focus` while on L3 the same ID has `kind: component` (the focus is a component refined into L4). The validator must treat L4's `focus` kind as matching L3's `component` kind for the focus node only. All other node kinds compare strictly.

- [ ] **Step 2.2: Confirm tests fail**

- [ ] **Step 2.3: Implement skeleton + L4→L3 direction**

Add after `validateL4Manifest`:

```go
// validateL4Carryover enforces the L4↔L3 cross-level invariant. Both
// directions are checked; all violations are aggregated via errors.Join.
//
// The L4 focus is rendered with kind "focus" but corresponds to a
// "component" on the L3 parent — that one ID receives a relaxed kind
// comparison.
func validateL4Carryover(l4 *L4Spec, l3 *L3Spec) error {
    l3ByID := map[string]L3Element{}
    for _, el := range l3.Elements {
        l3ByID[el.ID] = el
    }
    if _, ok := l3ByID[l4.Focus.ID]; !ok {
        return fmt.Errorf("focus.id %q: not present on L3 parent %q", l4.Focus.ID, l4.Parent)
    }

    var errs []error
    for i, node := range l4.Diagram.Nodes {
        l3el, ok := l3ByID[node.ID]
        if !ok {
            errs = append(errs, fmt.Errorf("diagram.nodes[%d] %q: not present on L3 parent %q",
                i, node.ID, l4.Parent))
            continue
        }
        if !kindsMatch(node.ID, node.Kind, l3el.Kind, l4.Focus.ID) {
            errs = append(errs, fmt.Errorf("diagram.nodes[%d] %q: kind %q does not match L3 parent kind %q",
                i, node.ID, node.Kind, l3el.Kind))
        }
    }

    // L3→L4 direction added in Task 3.
    return errors.Join(errs...)
}

// kindsMatch reports whether an L4 node kind is compatible with the L3
// element kind. The L4 focus has kind "focus" but the L3 element it
// refines has kind "component"; for that one ID the comparison relaxes.
func kindsMatch(nodeID, l4Kind, l3Kind, focusID string) bool {
    if nodeID == focusID && l4Kind == "focus" && l3Kind == "component" {
        return true
    }
    return l4Kind == l3Kind
}
```

Imports needed in `dev/c4_l4.go`: `errors` (if not already present).

- [ ] **Step 2.4: Confirm tests pass + check-full green**

- [ ] **Step 2.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): validateL4Carryover with focus + L4->L3 checks

Verifies the L4 focus exists on the L3 parent and every L4 node
matches an L3 element by id and kind (focus's "focus" kind relaxes
to L3's "component" for that one id). L3->L4 direction added next.

AI-Used: [claude]
EOF
)"
```

---

## Task 3: `validateL4Carryover` — L3→L4 connected-set direction

**Files:** `dev/c4_l4.go`, `dev/c4_l4_test.go`.

- [ ] **Step 3.1: Add failing tests**

```go
func TestValidateL4Carryover_MissingNeighborOutbound(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "M", Name: "memory", Kind: "component"}},
        Relationships: []L1Relationship{{From: "F", To: "M", Description: "writes"}},
    }
    err := validateL4Carryover(l4, l3)
    if err == nil || !strings.Contains(err.Error(), `"M"`) {
        t.Fatalf("expected missing-neighbor M, got: %v", err)
    }
}

func TestValidateL4Carryover_MissingNeighborInbound(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "C", Name: "cli", Kind: "component"}},
        Relationships: []L1Relationship{{From: "C", To: "F", Description: "calls"}},
    }
    err := validateL4Carryover(l4, l3)
    if err == nil || !strings.Contains(err.Error(), `"C"`) {
        t.Fatalf("expected missing-neighbor C, got: %v", err)
    }
}

func TestValidateL4Carryover_SelfLoopIgnored(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}},
        Relationships: []L1Relationship{{From: "F", To: "F", Description: "self"}},
    }
    if err := validateL4Carryover(l4, l3); err != nil {
        t.Fatalf("self-loop should not produce a neighbor: %v", err)
    }
}

func TestValidateL4Carryover_HappyPath(t *testing.T) {
    t.Parallel()
    l4 := &L4Spec{
        Focus:   L4Focus{ID: "F", Name: "focus"},
        Parent:  "c3-x.md",
        Diagram: L4Diagram{Nodes: []L4Node{
            {ID: "F", Name: "focus", Kind: "focus"},
            {ID: "M", Name: "memory", Kind: "component"},
            {ID: "C", Name: "cli", Kind: "component"},
        }},
    }
    l3 := &L3Spec{
        Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
        Elements: []L3Element{
            {ID: "F", Name: "focus", Kind: "component"},
            {ID: "M", Name: "memory", Kind: "component"},
            {ID: "C", Name: "cli", Kind: "component"},
        },
        Relationships: []L1Relationship{
            {From: "F", To: "M", Description: "writes"},
            {From: "C", To: "F", Description: "calls"},
        },
    }
    if err := validateL4Carryover(l4, l3); err != nil {
        t.Fatalf("happy path should pass: %v", err)
    }
}
```

- [ ] **Step 3.2: Confirm tests fail**

- [ ] **Step 3.3: Implement L3→L4 direction**

Replace the placeholder comment in `validateL4Carryover` with:

```go
    l4Nodes := map[string]bool{}
    for _, node := range l4.Diagram.Nodes {
        l4Nodes[node.ID] = true
    }
    connected := map[string]bool{}
    for _, rel := range l3.Relationships {
        switch {
        case rel.From == l4.Focus.ID && rel.To != l4.Focus.ID:
            connected[rel.To] = true
        case rel.To == l4.Focus.ID && rel.From != l4.Focus.ID:
            connected[rel.From] = true
        }
    }
    for id := range connected {
        if !l4Nodes[id] {
            errs = append(errs, fmt.Errorf("L3 parent %q has node %q connected to focus %q, but %q is missing from L4 diagram.nodes",
                l4.Parent, id, l4.Focus.ID, id))
        }
    }

    return errors.Join(errs...)
```

Sort iteration order to keep error output deterministic — replace the `for id := range connected` loop with:

```go
    connectedIDs := make([]string, 0, len(connected))
    for id := range connected {
        connectedIDs = append(connectedIDs, id)
    }
    sort.Strings(connectedIDs)
    for _, id := range connectedIDs {
        if !l4Nodes[id] {
            errs = append(errs, fmt.Errorf("L3 parent %q has node %q connected to focus %q, but %q is missing from L4 diagram.nodes",
                l4.Parent, id, l4.Focus.ID, id))
        }
    }
```

Imports: add `sort` if missing.

- [ ] **Step 3.4: Confirm tests pass + check-full green**

- [ ] **Step 3.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): validateL4Carryover L3->L4 neighbor check

Every L3 node connected (one-hop, either direction) to the focus's
L3 representation must appear on L4. Self-loops do not produce
neighbors. Missing-neighbor errors sorted for deterministic output.

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Wire into build-time `validateL4Spec`

**Files:** `dev/c4_l4.go`, `dev/c4_l4_test.go`.

- [ ] **Step 4.1: Add failing test**

```go
func TestValidateL4Spec_RunsCarryover(t *testing.T) {
    t.Parallel()
    spec := validL4Spec()
    spec.Diagram.Nodes = append(spec.Diagram.Nodes, L4Node{ID: "BOGUS", Name: "ghost", Kind: "component"})
    l3 := minimalL3ParentFor(spec) // helper added below
    err := validateL4Spec(&spec, l3)
    if err == nil || !strings.Contains(err.Error(), "BOGUS") {
        t.Fatalf("expected carryover error citing BOGUS, got: %v", err)
    }
}
```

Add a helper near `validL4Spec`:

```go
// minimalL3ParentFor returns an L3Spec that exactly satisfies the carryover
// check for the given L4 spec — every L4 node mirrored as an L3 element,
// every L3 neighbor of focus drawn from L4's nodes.
func minimalL3ParentFor(l4 L4Spec) *L3Spec {
    l3 := &L3Spec{Focus: L3Focus{ID: "S2-N3", Name: "stub", Responsibility: "stub"}}
    for _, node := range l4.Diagram.Nodes {
        kind := node.Kind
        if node.ID == l4.Focus.ID && kind == "focus" {
            kind = "component"
        }
        l3.Elements = append(l3.Elements, L3Element{ID: node.ID, Name: node.Name, Kind: kind})
        if node.ID != l4.Focus.ID {
            l3.Relationships = append(l3.Relationships, L1Relationship{
                From: l4.Focus.ID, To: node.ID, Description: "stub",
            })
        }
    }
    return l3
}
```

- [ ] **Step 4.2: Confirm test fails**

(Compilation error: `validateL4Spec` takes one argument.)

- [ ] **Step 4.3: Update `validateL4Spec` signature**

Find `func validateL4Spec(spec *L4Spec) error` (around line 733). Change to:

```go
func validateL4Spec(spec *L4Spec, l3 *L3Spec) error {
    // ... existing intra-spec checks unchanged ...

    if l3 != nil {
        if err := validateL4Carryover(spec, l3); err != nil {
            return err
        }
    }
    return nil
}
```

The `l3 != nil` guard preserves the option for callers that don't have an L3 (used only by tests asserting intra-spec behavior).

- [ ] **Step 4.4: Update the build-time call site**

In `c4L4Build` at `dev/c4_l4.go:623`:

```go
l3, err := loadL3Parent(&spec, filepath.Dir(args.Input))
if err != nil {
    return err
}
if err := validateL4Spec(&spec, l3); err != nil {
    return fmt.Errorf("validating %s: %w", args.Input, err)
}
```

(Adjust to match the existing error-wrapping style around the validate call.)

- [ ] **Step 4.5: Update existing tests that call `validateL4Spec(&spec)`**

Find every existing call site in `dev/c4_l4_test.go`:

```bash
grep -n 'validateL4Spec(' dev/c4_l4_test.go
```

For each call, pick the right second argument:
- If the test is asserting an intra-spec error (D-edge rejection, manifest mismatch, etc.) — pass `nil`.
- If the test is asserting carryover behavior — pass the `minimalL3ParentFor(spec)` helper or a hand-built `L3Spec`.

- [ ] **Step 4.6: Confirm all tests pass + check-full green**

- [ ] **Step 4.7: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): validateL4Spec runs carryover when L3 parent provided

Build-time c4-l4-build loads the L3 parent JSON and passes it to
validateL4Spec. Existing tests pass nil where carryover isn't the
focus, or use the minimalL3ParentFor helper to construct a
satisfying parent.

AI-Used: [claude]
EOF
)"
```

---

## Task 5: Audit-time wiring — emit `l4_carryover` findings

**Files:** `dev/c4.go`, `dev/c4_test.go`.

- [ ] **Step 5.1: Add failing test in `dev/c4_test.go`**

```go
func TestAuditFile_L4CarryoverEmitsFindings(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    // L3 with focus F and one neighbor M.
    l3 := []byte(`{"schema_version":"1","level":3,"name":"x","parent":"c2-x.md","focus":{"id":"S2-N3","name":"x","responsibility":"r"},"elements":[{"id":"F","name":"focus","kind":"component"},{"id":"M","name":"memory","kind":"component"}],"relationships":[{"from":"F","to":"M","description":"writes","protocol":"go"}]}`)
    if err := os.WriteFile(filepath.Join(dir, "c3-x.json"), l3, 0o600); err != nil { t.Fatalf(err.Error()) }

    // L4 referencing F as focus, no M, plus an extra "GHOST" node — two violations.
    l4JSON := []byte(`{"schema_version":"1","level":4,"name":"focus","parent":"c3-x.md","focus":{"id":"F","name":"focus"},"diagram":{"nodes":[{"id":"F","name":"focus","kind":"focus"},{"id":"GHOST","name":"ghost","kind":"component"}],"edges":[]},"dependency_manifest":[],"di_wires":[]}`)
    if err := os.WriteFile(filepath.Join(dir, "c4-focus.json"), l4JSON, 0o600); err != nil { t.Fatalf(err.Error()) }

    // Minimal L4 markdown: front-matter only; auditFile keys off matter.level == 4.
    md := []byte("---\nlevel: 4\nname: focus\nparent: c3-x.md\nchildren: []\nlast_reviewed_commit: 0000000\n---\n")
    mdPath := filepath.Join(dir, "c4-focus.md")
    if err := os.WriteFile(mdPath, md, 0o600); err != nil { t.Fatalf(err.Error()) }

    findings, err := auditFile(t.Context(), mdPath)
    if err != nil { t.Fatalf("audit: %v", err) }
    var carryover []Finding
    for _, f := range findings {
        if f.ID == "l4_carryover" {
            carryover = append(carryover, f)
        }
    }
    if len(carryover) != 2 {
        t.Fatalf("expected 2 l4_carryover findings (extra GHOST + missing M), got %d: %+v", len(carryover), carryover)
    }
}
```

- [ ] **Step 5.2: Confirm test fails**

- [ ] **Step 5.3: Extend the L4 branch in `auditFile`**

In `dev/c4.go:413-420`, replace the early return with a carryover check:

```go
    if matter.level == 4 {
        findings = append(findings, auditL4Carryover(path, matter)...)
        return findings, nil
    }
```

Add the helper near `auditFile`:

```go
// auditL4Carryover loads the L4 JSON sibling of an audited L4 markdown plus
// the L3 parent JSON, runs validateL4Carryover, and emits one l4_carryover
// finding per leaf error from the joined error.
func auditL4Carryover(mdPath string, matter frontMatter) []Finding {
    dir := filepath.Dir(mdPath)
    base := strings.TrimSuffix(filepath.Base(mdPath), ".md")
    l4Path := filepath.Join(dir, base+".json")
    l4Raw, err := os.ReadFile(l4Path)
    if err != nil {
        return []Finding{{ID: "l4_carryover", Detail: fmt.Sprintf("read %s: %v", l4Path, err)}}
    }
    var l4 L4Spec
    decoder := json.NewDecoder(bytes.NewReader(l4Raw))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&l4); err != nil {
        return []Finding{{ID: "l4_carryover", Detail: fmt.Sprintf("decode %s: %v", l4Path, err)}}
    }
    l3, err := loadL3Parent(&l4, dir)
    if err != nil {
        return []Finding{{ID: "l4_carryover", Detail: err.Error()}}
    }
    err = validateL4Carryover(&l4, l3)
    if err == nil {
        return nil
    }
    return splitJoinedError(err)
}

// splitJoinedError walks an error tree produced by errors.Join and returns one
// l4_carryover Finding per leaf. Falls back to a single finding if the input
// was not produced by Join.
func splitJoinedError(err error) []Finding {
    type joiner interface{ Unwrap() []error }
    if j, ok := err.(joiner); ok {
        var out []Finding
        for _, sub := range j.Unwrap() {
            out = append(out, splitJoinedError(sub)...)
        }
        return out
    }
    return []Finding{{ID: "l4_carryover", Detail: err.Error()}}
}
```

Imports needed in `dev/c4.go`: `bytes`, `encoding/json`, `path/filepath`, `strings` (likely all already present — verify).

- [ ] **Step 5.4: Confirm test passes + check-full green**

- [ ] **Step 5.5: Commit**

```bash
git add dev/c4.go dev/c4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): audit emits l4_carryover findings per violation

c4-audit on an L4 markdown loads the sibling L4 JSON and the L3
parent JSON, runs validateL4Carryover, and emits one
l4_carryover finding per leaf error from the joined error.

AI-Used: [claude]
EOF
)"
```

---

## Task 6: Data fixup — bring 19 L4 specs into compliance

**Files:** `architecture/c4/c4-*.json`, `architecture/c4/c4-*.md`, `architecture/c4/svg/*`, possibly `architecture/c4/c3-*.json` and rendered L3 markdown.

- [ ] **Step 6.1: Run audit and collect violations**

```bash
targ c4-audit 2>&1 | grep l4_carryover | sort -u | tee /tmp/l4-carryover.txt
```

If the list is empty, skip to Task 7. If non-empty, group by L4 file. Process one L4 at a time below.

- [ ] **Step 6.2: For each L4 with violations, decide the fix per violation**

Use this decision table:

| Violation | Decision rule | Action |
|-----------|---------------|--------|
| L4→L3 extra node `X`: not on L3 | Is the relationship between focus and X *real* at the L3 container level? | If yes → add `X` to L3 (`c3-*.json`) plus a relationship; if no → remove `X` (and its edges) from L4. |
| L4→L3 kind mismatch: `X` is `component` on L4, `external` on L3 | Trace which is correct from source code. | Edit the wrong side. |
| L3→L4 missing neighbor `N` connected to focus on L3 | The relationship is documented at L3, so it's real. | Add `N` to L4 nodes plus an `R<n>` edge that mirrors the L3 relationship's direction and description. |

- [ ] **Step 6.3: Apply fix-ups one L4 at a time**

For each L4 that has violations (process alphabetically by basename):

```bash
$EDITOR architecture/c4/c4-<name>.json
targ c4-l4-build --input=architecture/c4/c4-<name>.json --noconfirm
npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-<name>.mmd -o architecture/c4/svg/c4-<name>.svg
npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-<name>-wiring.mmd -o architecture/c4/svg/c4-<name>-wiring.svg
targ c4-audit --file=architecture/c4/c4-<name>.md
```

Iterate until that single file is clean. If L3 was edited:

```bash
targ c4-l3-build --input=architecture/c4/c3-<parent>.json --noconfirm
# render that L3 SVG too if the build target requires it
```

- [ ] **Step 6.4: Commit each L4 fix-up as its own commit**

```bash
git add architecture/c4/c4-<name>.json architecture/c4/c4-<name>.md architecture/c4/svg/c4-<name>*.{mmd,svg}
# include c3-<parent>.* if the L3 parent was edited
git commit -m "$(cat <<'EOF'
chore(c4): fix L4↔L3 carryover for c4-<name>

<one-sentence summary of what was added/removed and why>

AI-Used: [claude]
EOF
)"
```

When the L3 parent itself was edited, do that change in a separate commit ahead of the L4 fix-up.

- [ ] **Step 6.5: Confirm tree-wide audit clean**

```bash
targ c4-audit 2>&1 | grep -c l4_carryover  # expect 0
targ check-full
targ test
```

---

## Task 7: Final verification

- [ ] **Step 7.1: Acceptance grep**

```bash
cd /Users/joe/repos/personal/engram/.worktrees/issue-605

# Validator wired at both build and audit.
grep -n 'validateL4Carryover' dev/c4_l4.go dev/c4.go

# All L4 specs pass.
targ c4-audit 2>&1 | grep -E 'l4_carryover|FAIL' || echo OK

targ check-full
targ test
```

- [ ] **Step 7.2: Branch summary**

```bash
git log --oneline main..HEAD
git status
```

- [ ] **Step 7.3: Report**

Hand control back to the human for review. Do not push or merge.

---

## Self-review notes

- **Spec coverage:** Task 1 → loader; Tasks 2-3 → bidirectional validator; Task 4 → build-time wiring; Task 5 → audit-time wiring; Task 6 → data fixup; Task 7 → verification. Spec's "out of scope" items (skill text, L1/L2 rules, finding-format) explicitly excluded.
- **Type consistency:** `L4Spec`, `L4Node`, `L4Focus`, `L3Spec`, `L3Element`, `L3Focus`, `L1Relationship` are the existing names confirmed via grep. `validateL4Carryover`, `loadL3Parent`, `auditL4Carryover`, `splitJoinedError`, `kindsMatch`, `minimalL3ParentFor` are new and used consistently.
- **No placeholders.** Each step has exact code or exact commands.
- **TDD:** every behavior task starts with a failing test.
- **Focus kind quirk** (L4 `focus` ↔ L3 `component`) handled in `kindsMatch`; called out in test fixtures.
- **Determinism:** L3→L4 errors sorted by ID for stable output across runs.
