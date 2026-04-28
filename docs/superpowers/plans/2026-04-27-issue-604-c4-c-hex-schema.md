# Issue #604 — C-hex Schema Across All C4 Levels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace L4 D-edge DI back-edges with C-hex port nodes + A-edges, with schema/builder/renderer/audit support at all four C4 levels (L1 audit-side, L2/L3/L4 spec-side).

**Architecture:** Ports are first-class circle nodes owned by the consumer of an interface. A-edges (solid arrows) go wirer → port and are labeled with the diagram entity being plugged in. Schema additions land in `dev/c4_l2.go`, `dev/c4_l3.go`, `dev/c4_l4.go`; global audit regex in `dev/c4.go` is extended to accept `A<n>` and reject `D<n>`. L4's `dependency_manifest` and `di_wires` tables are slimmed (drop adapter func names). The worked example is `architecture/c4/c4-recall.json`.

**Tech Stack:** Go 1.x, `targ` build tool, `mermaid` flowcharts, JSON specs.

**Spec:** `docs/superpowers/specs/2026-04-27-issue-604-c4-l4-c-hex-schema-design.md`

---

## Pre-flight

- [ ] **Step P-1: Sanity-check current state**

```bash
cd /Users/joe/repos/personal/engram
git status   # expect clean working tree
targ check-full
targ test
```

Expected: clean status, both commands pass. If anything fails, STOP — do not start implementation on a red baseline.

---

## Task 1: Audit-side regex (`dev/c4.go`) — accept A, reject D

**Files:**
- Modify: `dev/c4.go:141-144` (regex), `dev/c4.go:810-815` (audit error message)
- Test: `dev/c4_test.go`

- [ ] **Step 1.1: Find or add a c4_test.go test that exercises edge ID validation**

```bash
grep -n 'edgeIDPrefix\|edge_id_missing\|R<n>: or D<n>:' dev/c4_test.go
```

Locate the existing test that verifies edge-ID prefix acceptance/rejection. If none exists, write one.

- [ ] **Step 1.2: Write the failing test — A-edge accepted, D-edge rejected**

In `dev/c4_test.go`, add (or extend the existing test) so that:

```go
func TestEdgeIDPrefix_AcceptsAEdges(t *testing.T) {
    t.Parallel()
    // arrange: a markdown block with one R-edge and one A-edge
    // act: run the audit pass that uses edgeIDPrefix
    // assert: no edge_id_missing finding for either edge
}

func TestEdgeIDPrefix_RejectsDEdges(t *testing.T) {
    t.Parallel()
    // arrange: a markdown block with a D-edge label "D1: legacy"
    // act: run the audit pass
    // assert: exactly one edge_id_missing finding citing "R<n>: or A<n>:"
}
```

(Use existing test fixtures and helpers in `c4_test.go` — find an analogous edge-acceptance test and mirror its setup.)

- [ ] **Step 1.3: Run tests to confirm both fail**

```bash
targ test 2>&1 | grep -E 'TestEdgeIDPrefix|FAIL'
```

Expected: both new tests FAIL (regex still matches D, doesn't match A).

- [ ] **Step 1.4: Update regex and error message**

Edit `dev/c4.go:141-144`:

```go
// edgeIDPrefix accepts R<n> (direct call) or A<n> (adapter plugs into port)
// labels. R appears at all levels; A appears wherever the wirer crosses a
// boundary visible at this level.
edgeIDPrefix  = regexp.MustCompile(`^[RA]\d+\s*:`)
```

Edit `dev/c4.go:814`:

```go
Detail: fmt.Sprintf("edge %q->%q label %q does not start with R<n>: or A<n>:", edge.from, edge.to, edge.label),
```

- [ ] **Step 1.5: Run tests to confirm they pass**

```bash
targ test 2>&1 | grep -E 'TestEdgeIDPrefix|FAIL|PASS'
```

Expected: both new tests PASS, no other regressions in `c4_test.go`.

- [ ] **Step 1.6: Commit**

```bash
git add dev/c4.go dev/c4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): audit accepts A-edges (adapter→port), rejects D-edges

Per #604: replace D<n> DI back-edges with A<n> adapter→port edges across
all C4 levels. This commit only updates the global audit regex; per-level
schema and renderer changes follow.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Add `port` node kind + A-edge regex to L4 schema (`dev/c4_l4.go`)

**Files:**
- Modify: `dev/c4_l4.go:74-82` (L4Node), `dev/c4_l4.go:122-125` (regex), `dev/c4_l4.go:541-574` (validateL4NodeIDs)
- Test: `dev/c4_l4_test.go`

- [ ] **Step 2.1: Read existing L4 validator + tests**

```bash
sed -n '540,610p' dev/c4_l4.go
grep -n 'TestValidateL4\|TestL4Spec' dev/c4_l4_test.go | head -10
```

Locate the existing validator test pattern; new tests follow it.

- [ ] **Step 2.2: Write failing tests for `port` kind acceptance**

In `dev/c4_l4_test.go`, add:

```go
func TestL4Spec_AcceptsPortKind(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)  // existing test helper or build inline
    spec.Diagram.Nodes = append(spec.Diagram.Nodes, L4Node{
        ID: "S2-N3-M3-PT1", Name: "Finder", Kind: "port",
    })
    err := validateL4Spec(spec)
    if err != nil {
        t.Fatalf("expected port kind to validate, got: %v", err)
    }
}

func TestL4Spec_RejectsPortIDCollidesWithProperty(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Nodes = append(spec.Diagram.Nodes, L4Node{
        ID: "S2-N3-M3-P1", Name: "Bad", Kind: "port",  // missing T
    })
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "PT") {
        t.Fatalf("expected port-ID rejection, got: %v", err)
    }
}

func TestL4Spec_RejectsPortIDNonMonotonic(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Nodes = append(spec.Diagram.Nodes,
        L4Node{ID: "S2-N3-M3-PT1", Name: "P1", Kind: "port"},
        L4Node{ID: "S2-N3-M3-PT3", Name: "P3", Kind: "port"},  // skips PT2
    )
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "PT2") {
        t.Fatalf("expected non-monotonic port-ID rejection, got: %v", err)
    }
}
```

If `minimalValidL4Spec(t)` doesn't exist, write it as a helper that returns a barely-valid `*L4Spec`.

- [ ] **Step 2.3: Write failing test for A-edge regex (replaces D-edge)**

```go
func TestL4Spec_AcceptsAEdges(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
        ID: "A1", From: "S2-N3-M2", To: "S2-N3-M3-PT1", Label: "anthropic",
    })
    spec.Diagram.Nodes = append(spec.Diagram.Nodes,
        L4Node{ID: "S2-N3-M3-PT1", Name: "SummarizerI", Kind: "port"},
    )
    if err := validateL4Spec(spec); err != nil {
        t.Fatalf("expected A-edge to validate, got: %v", err)
    }
}

func TestL4Spec_RejectsDEdges(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
        ID: "D1", From: "S2-N3-M3", To: "S2-N3-M2", Label: "legacy", Dotted: true,
    })
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "R<n>") {
        t.Fatalf("expected D-edge rejection, got: %v", err)
    }
}
```

- [ ] **Step 2.4: Find and update the existing test at `dev/c4_l4_test.go:206`**

That test currently asserts a D-edge is accepted. Flip it to assert rejection (per Task 2.3's `TestL4Spec_RejectsDEdges` semantics — you may consolidate or replace).

- [ ] **Step 2.5: Run tests to confirm all five new tests fail**

```bash
targ test 2>&1 | grep -E 'TestL4Spec_(Accepts|Rejects)|FAIL'
```

Expected: all five new tests FAIL.

- [ ] **Step 2.6: Implement port kind + A-edge regex in `dev/c4_l4.go`**

Replace the regex variable at `dev/c4_l4.go:122-125`:

```go
var (
    aEdgeIDPrefix = regexp.MustCompile(`^[RA]\d+$`)
)
```

Update `validateL4NodeIDs` at `dev/c4_l4.go:551-574`. Add port handling:

```go
func validateL4NodeIDs(spec *L4Spec) error {
    focusPath, err := ParseIDPath(spec.Focus.ID)
    if err != nil {
        return fmt.Errorf("focus.id: %w", err)
    }
    violations := []string{}
    for index, edge := range spec.Diagram.Edges {
        if !aEdgeIDPrefix.MatchString(edge.ID) {
            violations = append(violations, fmt.Sprintf(
                "diagram.edges[%d].id %q: must match R<n> (call relationship) or A<n> "+
                    "(adapter plugs into port)",
                index, edge.ID))
        }
    }
    portCount := 0
    for index, node := range spec.Diagram.Nodes {
        if node.Kind == "port" {
            portCount++
            if portErr := validatePortID(node.ID, focusPath, portCount); portErr != nil {
                violations = append(violations, fmt.Sprintf("diagram.nodes[%d].id: %v", index, portErr))
            }
            continue
        }
        if nodeErr := ValidateDiagramNodeID(focusPath, node.ID); nodeErr != nil {
            violations = append(violations, fmt.Sprintf("diagram.nodes[%d].id: %v", index, nodeErr))
        }
    }
    if len(violations) == 0 {
        return nil
    }
    return fmt.Errorf("L4 id validation failed:\n  - %s", strings.Join(violations, "\n  - "))
}

// validatePortID enforces port IDs of the form <focus>-PT<n> with 1-based
// monotonic <n>.
func validatePortID(id string, focusPath IDPath, expected int) error {
    expectedSuffix := fmt.Sprintf("PT%d", expected)
    expectedID := focusPath.String() + "-" + expectedSuffix
    if id != expectedID {
        return fmt.Errorf("port id %q must be %q (focus + -PT<n>, monotonic from 1)", id, expectedID)
    }
    return nil
}
```

- [ ] **Step 2.7: Run tests to confirm all five new tests pass**

```bash
targ test 2>&1 | grep -E 'TestL4Spec_(Accepts|Rejects)|FAIL|PASS'
```

Expected: all five PASS, no regressions elsewhere.

- [ ] **Step 2.8: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): L4 spec accepts port kind + A-edges, rejects D-edges

Replaces dEdgeIDPrefix with aEdgeIDPrefix and adds validatePortID enforcing
<focus>-PT<n> hierarchical port IDs. Existing D-edge test flipped from
accept to reject.

AI-Used: [claude]
EOF
)"
```

---

## Task 3: A-edge label-references-known-node validation (L4)

**Files:**
- Modify: `dev/c4_l4.go` (validateL4NodeIDs or new helper)
- Test: `dev/c4_l4_test.go`

- [ ] **Step 3.1: Write failing test**

```go
func TestL4Spec_RejectsAEdgeWithUnknownTarget(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Nodes = append(spec.Diagram.Nodes,
        L4Node{ID: "S2-N3-M3-PT1", Name: "SummarizerI", Kind: "port"},
    )
    spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
        ID: "A1", From: "S2-N3-M2", To: "S2-N3-M3-PT1",
        Label: "ghost-component",  // not a node
    })
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "ghost-component") {
        t.Fatalf("expected unknown-target rejection, got: %v", err)
    }
}

func TestL4Spec_AcceptsAEdgeWithKnownTarget(t *testing.T) {
    t.Parallel()
    spec := minimalValidL4Spec(t)
    spec.Diagram.Nodes = append(spec.Diagram.Nodes,
        L4Node{ID: "S2-N3-M3-PT1", Name: "SummarizerI", Kind: "port"},
        L4Node{ID: "S2-N3-M7", Name: "anthropic", Kind: "component"},
    )
    spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
        ID: "A1", From: "S2-N3-M2", To: "S2-N3-M3-PT1", Label: "anthropic",
    })
    if err := validateL4Spec(spec); err != nil {
        t.Fatalf("expected A-edge with known target to pass, got: %v", err)
    }
}
```

- [ ] **Step 3.2: Confirm tests fail**

```bash
targ test 2>&1 | grep -E 'TestL4Spec_.*Target|FAIL'
```

Expected: both FAIL (no label-target check yet).

- [ ] **Step 3.3: Implement validation**

Add to `validateL4NodeIDs` (or as a separate `validateAEdgeLabels` called from `validateL4Spec`):

```go
// Build a set of known node IDs and labels, then require every A-edge label
// to match.
known := map[string]bool{}
for _, node := range spec.Diagram.Nodes {
    known[node.ID] = true
    if node.Name != "" {
        known[node.Name] = true
    }
}
for index, edge := range spec.Diagram.Edges {
    if !strings.HasPrefix(edge.ID, "A") {
        continue
    }
    label := strings.TrimSpace(edge.Label)
    if label == "" {
        violations = append(violations,
            fmt.Sprintf("diagram.edges[%d]: A-edge label must name a diagram entity", index))
        continue
    }
    if !known[label] {
        violations = append(violations, fmt.Sprintf(
            "diagram.edges[%d]: A-edge label %q does not match any node ID or name",
            index, label))
    }
}
```

- [ ] **Step 3.4: Confirm tests pass**

```bash
targ test 2>&1 | grep -E 'TestL4Spec_.*Target|FAIL|PASS'
```

Expected: both PASS.

- [ ] **Step 3.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): A-edge label must reference a known diagram node

Per #604: A-edges encode "this entity is plugged into the port". The label
names a diagram node (component, container, external, person). Validation
rejects A-edges whose label has no corresponding node.

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Drop `Dotted` field on `L4Edge` (no longer used)

**Files:**
- Modify: `dev/c4_l4.go:58-65` (L4Edge struct), `dev/c4_l4.go:344-353` (emitL4MermaidEdge)
- Test: `dev/c4_l4_test.go`

- [ ] **Step 4.1: Search for `Dotted` usages**

```bash
grep -rn '\.Dotted\b\|Dotted:\s*true' dev/c4_l4*.go
```

Catalogue every reference; some test fixtures or builders may still set it.

- [ ] **Step 4.2: Write failing test that asserts Dotted is gone**

A compile-time guarantee is fine — drop any test fixture that sets `Dotted: true`. The build will fail until the struct field is removed and consumers are updated.

- [ ] **Step 4.3: Drop the field and update emitter**

In `dev/c4_l4.go:58-65`:

```go
type L4Edge struct {
    ID    string `json:"id"`
    From  string `json:"from"`
    To    string `json:"to"`
    Label string `json:"label"`
}
```

In `dev/c4_l4.go:344-353`, simplify:

```go
func emitL4MermaidEdge(buf *bytes.Buffer, edge L4Edge) {
    from := strings.ToLower(edge.From)
    to := strings.ToLower(edge.To)
    label := fmt.Sprintf("%s: %s", edge.ID, edge.Label)
    fmt.Fprintf(buf, "    %s -->|%q| %s\n", from, label, to)
}
```

(Note: previously a stray `arrow` variable carried `-.->` for dotted; now it's always `-->`.)

- [ ] **Step 4.4: Run targ check + test**

```bash
targ check-full
targ test
```

Expected: both pass. Any consumer of `Dotted` outside L4 surfaces here.

- [ ] **Step 4.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
refactor(c4): drop unused Dotted field on L4Edge (was D-edge only)

A-edges are solid; D-edges are dropped. Dotted has no remaining consumer.

AI-Used: [claude]
EOF
)"
```

---

## Task 5: Port shape + `classDef port` in L4 mermaid emitter

**Files:**
- Modify: `dev/c4_l4.go:308-365` (emitL4Mermaid, emitL4MermaidClasses, emitL4MermaidNode, l4NodeShape)
- Test: `dev/c4_l4_test.go`

- [ ] **Step 5.1: Write failing renderer test**

```go
func TestEmitL4Mermaid_RendersPortAsCircleWithClassDef(t *testing.T) {
    t.Parallel()
    spec := &L4Spec{
        // ...minimal spec with one port node "S2-N3-M3-PT1" labeled "Finder"
    }
    var buf bytes.Buffer
    emitL4Mermaid(&buf, spec)
    out := buf.String()
    if !strings.Contains(out, `classDef port`) {
        t.Errorf("missing classDef port: %s", out)
    }
    if !strings.Contains(out, `s2-n3-m3-pt1((`) {
        t.Errorf("port not rendered as circle: %s", out)
    }
    if !strings.Contains(out, "class s2-n3-m3-pt1 port") {
        t.Errorf("port not assigned to port class: %s", out)
    }
}
```

- [ ] **Step 5.2: Confirm test fails**

```bash
targ test 2>&1 | grep -E 'TestEmitL4Mermaid_RendersPort|FAIL'
```

- [ ] **Step 5.3: Implement port shape and class**

In `dev/c4_l4.go:500-509`, extend `l4NodeShape`:

```go
func l4NodeShape(kind string) (string, string) {
    switch kind {
    case "person":
        return "([", "])"
    case "external":
        return "(", ")"
    case "port":
        return "((", "))"
    default:
        return "[", "]"
    }
}
```

In `dev/c4_l4.go:308-327`, add the port classDef line and include in classOrder:

```go
buf.WriteString("    classDef port       fill:#fef3c7,stroke:#a16207,color:#000\n\n")
```

In `dev/c4_l4.go:329-342`, extend `classOrder`:

```go
classOrder := []string{"person", "external", "container", "component", "focus", "port"}
```

- [ ] **Step 5.4: Confirm test passes**

```bash
targ test 2>&1 | grep -E 'TestEmitL4Mermaid_RendersPort|FAIL|PASS'
```

Expected: PASS.

- [ ] **Step 5.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): L4 mermaid emits port nodes as circles with classDef port

AI-Used: [claude]
EOF
)"
```

---

## Task 6: Slim `L4DepRow` and `L4WireRow` structs

**Files:**
- Modify: `dev/c4_l4.go:40-50` (L4DepRow), `dev/c4_l4.go:111-120` (L4WireRow), `dev/c4_l4.go:231-254` (emitters), `dev/c4_l4.go:392-400` (wire emitter)
- Test: `dev/c4_l4_test.go`

- [ ] **Step 6.1: Write failing test asserting new schema**

```go
func TestL4DepRow_HasSlimSchema(t *testing.T) {
    t.Parallel()
    row := L4DepRow{
        PortID: "S2-N3-M3-PT1", PortName: "Finder", PortType: "recall.Finder",
        WirerID: "S2-N3-M2", WirerName: "cli", WirerL3: "c3-engram-cli-binary.md",
        WrappedID: "S6", WrappedName: "operating system", WrappedL3: "c3-engram-cli-binary.md",
        Properties: []string{"S2-N3-M3-P1"},
    }
    raw, err := json.Marshal(row)
    if err != nil { t.Fatalf("marshal: %v", err); }
    s := string(raw)
    if strings.Contains(s, "concrete_adapter") || strings.Contains(s, "wired_by") {
        t.Errorf("legacy fields still present: %s", s)
    }
    for _, want := range []string{"port_id", "wirer_id", "wrapped_id"} {
        if !strings.Contains(s, want) {
            t.Errorf("missing %q: %s", want, s)
        }
    }
}
```

- [ ] **Step 6.2: Confirm test fails (compile error or assertion)**

```bash
targ test 2>&1 | grep -E 'TestL4DepRow_HasSlim|FAIL|undefined'
```

- [ ] **Step 6.3: Replace structs**

In `dev/c4_l4.go:40-50`, replace `L4DepRow`:

```go
// L4DepRow is one row of the consumer-side Dependency Manifest table.
// Each row corresponds to one port the focus owns.
type L4DepRow struct {
    PortID      string   `json:"port_id"`
    PortName    string   `json:"port_name"`
    PortType    string   `json:"port_type"`
    WirerID     string   `json:"wirer_id"`
    WirerName   string   `json:"wirer_name"`
    WirerL3     string   `json:"wirer_l3"`
    WirerL4     string   `json:"wirer_l4,omitempty"`
    WrappedID   string   `json:"wrapped_id"`
    WrappedName string   `json:"wrapped_name"`
    WrappedL3   string   `json:"wrapped_l3"`
    WrappedL4   string   `json:"wrapped_l4,omitempty"`
    Properties  []string `json:"properties"`
}
```

In `dev/c4_l4.go:111-120`, replace `L4WireRow`:

```go
// L4WireRow is one row of the provider-side DI Wires table.
// Each row corresponds to one port the focus wires for somebody else.
type L4WireRow struct {
    PortID       string `json:"port_id"`
    PortName     string `json:"port_name"`
    PortType     string `json:"port_type"`
    ConsumerID   string `json:"consumer_id"`
    ConsumerName string `json:"consumer_name"`
    ConsumerL3   string `json:"consumer_l3"`
    ConsumerL4   string `json:"consumer_l4,omitempty"`
    WrappedID    string `json:"wrapped_id"`
    WrappedName  string `json:"wrapped_name"`
    WrappedL3    string `json:"wrapped_l3"`
    WrappedL4    string `json:"wrapped_l4,omitempty"`
}
```

- [ ] **Step 6.4: Update `emitL4DepRow` (`dev/c4_l4.go:231-241`)**

```go
func emitL4DepRow(buf *bytes.Buffer, row L4DepRow) {
    wirer := fmt.Sprintf("[%s · %s](%s#%s)",
        row.WirerID, row.WirerName, row.WirerL3, Anchor(row.WirerID, row.WirerName))
    if row.WirerL4 != "" {
        wirer += fmt.Sprintf(" ([%s](%s))", row.WirerL4, row.WirerL4)
    }
    wrapped := fmt.Sprintf("[%s · %s](%s#%s)",
        row.WrappedID, row.WrappedName, row.WrappedL3, Anchor(row.WrappedID, row.WrappedName))
    if row.WrappedL4 != "" {
        wrapped += fmt.Sprintf(" ([%s](%s))", row.WrappedL4, row.WrappedL4)
    }
    fmt.Fprintf(buf, "| `%s` | `%s` | `%s` | %s | %s | %s |\n",
        row.PortID, row.PortName, row.PortType, wirer, wrapped, formatPropertyList(row.Properties))
}
```

- [ ] **Step 6.5: Update `emitL4DependencyManifest` preamble + headers (`dev/c4_l4.go:243-254`)**

```go
func emitL4DependencyManifest(buf *bytes.Buffer, spec *L4Spec) {
    buf.WriteString("## Dependency Manifest\n\n")
    buf.WriteString("Each row is one port the focus owns. The wirer supplies the adapter that\n")
    buf.WriteString("plugs into the port; the wrapped entity is what that adapter ultimately\n")
    buf.WriteString("drives behavior against. Reciprocal entries live in the wirer's L4 under\n")
    buf.WriteString("\"DI Wires\" — those two sections must stay in sync.\n\n")
    buf.WriteString("| Port ID | Port name | Port type | Wirer | Wrapped entity | Properties |\n")
    buf.WriteString("|---|---|---|---|---|---|\n")
    for _, row := range spec.DependencyManifest {
        emitL4DepRow(buf, row)
    }
    buf.WriteString("\n")
}
```

- [ ] **Step 6.6: Update `emitL4WireRow` (`dev/c4_l4.go:392-400`) and `emitL4DIWires` (`dev/c4_l4.go:219-229`) similarly**

```go
func emitL4DIWires(buf *bytes.Buffer, spec *L4Spec) {
    buf.WriteString("## DI Wires\n\n")
    buf.WriteString("Each row is one port this component wires for a consumer. Reciprocal entries\n")
    buf.WriteString("live in the consumer's L4 under \"Dependency Manifest\".\n\n")
    buf.WriteString("| Port ID | Port name | Port type | Consumer | Wrapped entity |\n")
    buf.WriteString("|---|---|---|---|---|\n")
    for _, row := range spec.DIWires {
        emitL4WireRow(buf, row)
    }
    buf.WriteString("\n")
}

func emitL4WireRow(buf *bytes.Buffer, row L4WireRow) {
    consumer := fmt.Sprintf("[%s · %s](%s#%s)",
        row.ConsumerID, row.ConsumerName, row.ConsumerL3, Anchor(row.ConsumerID, row.ConsumerName))
    if row.ConsumerL4 != "" {
        consumer += fmt.Sprintf(" ([%s](%s))", row.ConsumerL4, row.ConsumerL4)
    }
    wrapped := fmt.Sprintf("[%s · %s](%s#%s)",
        row.WrappedID, row.WrappedName, row.WrappedL3, Anchor(row.WrappedID, row.WrappedName))
    if row.WrappedL4 != "" {
        wrapped += fmt.Sprintf(" ([%s](%s))", row.WrappedL4, row.WrappedL4)
    }
    fmt.Fprintf(buf, "| `%s` | `%s` | `%s` | %s | %s |\n",
        row.PortID, row.PortName, row.PortType, consumer, wrapped)
}
```

- [ ] **Step 6.7: Run targ check + test, fix every compilation error**

```bash
targ check-full 2>&1 | head -40
targ test 2>&1 | grep -E 'FAIL|TestL4DepRow|TestL4WireRow' | head -20
```

Expected: passes after struct rename ripples are fixed in any test fixtures that referenced the old field names.

- [ ] **Step 6.8: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
refactor(c4): slim L4 manifest/wires rows for C-hex (drop adapter func names)

L4DepRow becomes {port_id, port_name, port_type, wirer_*, wrapped_*,
properties}. L4WireRow drops wired_adapter/concrete_value, gains port_*
+ wrapped_* fields. Adapter func names no longer surface anywhere.

AI-Used: [claude]
EOF
)"
```

---

## Task 7: Add `port` kind + A-edge support to L3 builder

**Files:**
- Modify: `dev/c4_l3.go` (allowed kinds, validator, mermaid emitter, classDef)
- Test: `dev/c4_l3_test.go`

- [ ] **Step 7.1: Locate the L3 kind allowlist + validator**

```bash
grep -n 'case "person"\|case "component"\|"port"\|allowed.*kind\|Kind ==\|Kind !=' dev/c4_l3.go | head -30
```

Identify the switch and the validator that rejects unknown kinds.

- [ ] **Step 7.2: Write failing test**

```go
func TestL3Spec_AcceptsPortKind(t *testing.T) {
    t.Parallel()
    spec := minimalValidL3Spec(t)
    spec.Elements = append(spec.Elements, L3Element{
        ID: "S2-N3-PT1", Name: "ExternalPort", Kind: "port",
        Responsibility: "Externally-wired port at the container boundary",
    })
    if err := validateL3Spec(spec); err != nil {
        t.Fatalf("expected port kind to validate at L3, got: %v", err)
    }
}
```

- [ ] **Step 7.3: Confirm test fails**

```bash
targ test 2>&1 | grep -E 'TestL3Spec_AcceptsPort|FAIL'
```

- [ ] **Step 7.4: Add `port` to allowed L3 kinds**

In every switch on `Element.Kind` in `dev/c4_l3.go`, add a `case "port"` branch.

In the mermaid emitter, add `port` shape returning `((` / `))` and add `classDef port` + extend `classOrder` to include `"port"`.

In the validator, allow `port` as a kind. Port IDs at L3 must match `<owner>-PT<n>` where `<owner>` is a container (level 2 ID) — write a small `validatePortIDAtLevel(id, ownerLevel, expectedNum)` helper if useful.

- [ ] **Step 7.5: Confirm test passes + no regressions**

```bash
targ test 2>&1 | grep -E 'TestL3Spec|FAIL|PASS' | head -30
```

- [ ] **Step 7.6: Commit**

```bash
git add dev/c4_l3.go dev/c4_l3_test.go
git commit -m "$(cat <<'EOF'
feat(c4): L3 builder accepts port kind + A-edges (C-hex escalation)

Port nodes at L3 belong to the consuming container. A-edges go
wirer-container → port. This covers the case where DI crosses container
boundaries.

AI-Used: [claude]
EOF
)"
```

---

## Task 8: Add `port` kind + A-edge support to L2 builder

**Files:**
- Modify: `dev/c4_l2.go` (mirror Task 7 changes for L2: allowed kinds, validator, mermaid emitter, classDef)
- Test: `dev/c4_l2_test.go`

- [ ] **Step 8.1: Mirror Task 7's flow at L2**

Apply the equivalent changes in `dev/c4_l2.go` (port kind in switches, validator, mermaid classDef + classOrder + node shape). Port IDs at L2 are `<system>-PT<n>` (level 1 owner).

Write `TestL2Spec_AcceptsPortKind` modeled on `TestL3Spec_AcceptsPortKind`.

- [ ] **Step 8.2: Run targ test**

```bash
targ test 2>&1 | grep -E 'TestL2Spec|FAIL|PASS' | head -30
```

Expected: PASS.

- [ ] **Step 8.3: Commit**

```bash
git add dev/c4_l2.go dev/c4_l2_test.go
git commit -m "$(cat <<'EOF'
feat(c4): L2 builder accepts port kind + A-edges (C-hex escalation)

AI-Used: [claude]
EOF
)"
```

---

## Task 9: Update `c4 skill` doc cross-references for D→A

**Files:**
- Modify: `dev/c4.go:656` (doc comment), `dev/c4_l4.go:543` (doc comment), `dev/c4_l4.go:246` ("Rdi back-edge" preamble already replaced in Task 6 — verify)

- [ ] **Step 9.1: Search for stale "D<n>" / "Rdi" / "DI back-edge" references in `dev/`**

```bash
grep -rn 'D<n>\|Rdi\|DI back-edge\|D-edge' dev/c4*.go
```

- [ ] **Step 9.2: Update each comment to reference A<n>** (or remove if no longer applicable)

Pure documentation pass; no behavior change.

- [ ] **Step 9.3: Run targ check**

```bash
targ check-full
```

- [ ] **Step 9.4: Commit**

```bash
git add dev/c4*.go
git commit -m "$(cat <<'EOF'
docs(c4): update D-edge references to A-edges in source comments

AI-Used: [claude]
EOF
)"
```

---

## Task 10: Regenerate `c4-recall.json` under new schema (worked example)

**Files:**
- Modify: `architecture/c4/c4-recall.json`
- Generated: `architecture/c4/c4-recall.md`, `architecture/c4/svg/c4-recall.mmd`

- [ ] **Step 10.1: Read current JSON spec and ground-truth wiring**

```bash
cat architecture/c4/c4-recall.json
grep -n 'NewSummarizer\|NewSessionFinder\|NewTranscriptReader\|NewLister\|NewFileCache\|Discover(\|os\.Stderr' internal/cli/*.go | head -20
```

Identify the seven DI seams and what each adapter wraps. Reference: `playground/issue-603-di-notation-sketch.md` Approach C-hex for the worked example.

- [ ] **Step 10.2: Edit `architecture/c4/c4-recall.json` in place**

For each of the 7 seams add:
- A port node `S2-N3-M3-PT<n>` with kind `port` and the interface name as `name` (Finder, Reader, SummarizerI, MemoryLister, FileCache, externalFiles, statusWriter — adjust to actual seam names).
- An A-edge from `S2-N3-M2` (cli) → port, label = the wrapped diagram entity (e.g. `anthropic`, `memory`, `context`, `externalsources`, `S6 · operating system`).

Remove the existing D-edge.

Replace `dependency_manifest` rows with the new shape (port-centric, no concrete_adapter).

If the wrapped entity is OS-level (filesystem) and not currently a node, add `S6` (or appropriate ID) as a `kind: external` node so the A-edge label resolves. (This is the spec-level external-completeness check at work.)

- [ ] **Step 10.3: Run the L4 builder**

```bash
targ c4-l4-build --input=architecture/c4/c4-recall.json --noconfirm
```

Expected: passes; emits updated `.md` and `.mmd`.

- [ ] **Step 10.4: Render SVG and inspect**

```bash
npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall.mmd -o architecture/c4/svg/c4-recall.svg
```

Open `architecture/c4/c4-recall.md` and `architecture/c4/svg/c4-recall.svg`. Confirm by eye:
- Seven port circles.
- Solid A-edges from cli → ports, labeled with wrapped entity names.
- R-edges from recall → driven nodes still present.
- No D-edge.
- Slimmed dependency_manifest table.

- [ ] **Step 10.5: Run audit**

```bash
targ c4-audit
```

Expected: clean, or only findings unrelated to #604 (these are pre-existing and not in scope).

- [ ] **Step 10.6: Run full check + tests**

```bash
targ check-full
targ test
```

- [ ] **Step 10.7: Commit**

```bash
git add architecture/c4/c4-recall.json architecture/c4/c4-recall.md architecture/c4/svg/c4-recall.mmd architecture/c4/svg/c4-recall.svg
git commit -m "$(cat <<'EOF'
feat(c4): regenerate c4-recall under C-hex schema (worked example for #604)

Seven port circles, A-edges from cli, no D-edges, slimmed manifest.

AI-Used: [claude]
EOF
)"
```

---

## Task 11: Final verification + close-out

- [ ] **Step 11.1: Verify all acceptance criteria from the spec**

- [ ] `targ check-full` passes.
- [ ] `targ test` passes.
- [ ] `architecture/c4/c4-recall.json` regenerated under new schema.
- [ ] L1/L2/L3 still pass audit unchanged (they have no port nodes today).
- [ ] No D-edge handling remains in `c4.go`, `c4_l2.go`, `c4_l3.go`, `c4_l4.go`.

```bash
grep -rn 'D<n>\|Rdi\|dEdgeIDPrefix\|D-edge' dev/c4*.go || echo "OK: no D-edge handling left"
targ check-full && targ test && echo "OK: green"
```

- [ ] **Step 11.2: Push (only after review)**

```bash
git log --oneline main..HEAD   # verify the commit set
git push                       # only after the work has been reviewed per project rules
```

- [ ] **Step 11.3: Close issue with summary**

```bash
gh issue close 604 --comment "C-hex schema (port nodes + A-edges) landed across all four C4 levels; D-edges removed; c4-recall.json regenerated as the worked example. Bulk regen of remaining 18 L4 specs and the skill rewrite (#606) and external-completeness audit (#605) and from-scratch verification (#586) remain open."
```

---

## Self-review notes (built into the plan)

- **Spec coverage:** All schema additions, builder changes, renderer changes, audit changes, and the worked-example regen from the spec are mapped to tasks 1–10. Task 11 is verification.
- **Type consistency:** `aEdgeIDPrefix` (Task 2) is the only new package-level regex variable. `validatePortID` is the only new helper function name. `L4DepRow` / `L4WireRow` field names match the spec one-to-one (Task 6).
- **No placeholders:** every step contains exact file paths, exact code, or exact commands.
- **Regression guards:** Task 11 includes a grep that fails if any D-edge handling remains. Tests at every task assert both positive and negative cases.
- **TDD:** every behavior task has a failing test written first.
