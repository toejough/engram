# Issue #604 — Two-Diagram L4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** L4 specs render as two diagrams — a strict C4 call view (R-edges with property IDs) and a small wiring view (wirer→focus edges labeled with SNM IDs of wrapped entities). Drop D-edges entirely. No port nodes, no A-edges.

**Architecture:** Schema change in `dev/c4_l4.go`: `L4Edge.Properties` adds inline property linkage to R-edges; `L4DepRow`/`L4WireRow` slim to `{Field, Type, Wired by, Wrapped entity, Properties}` with `wrapped_entity_id` carrying the wiring-diagram label. Renderer emits two `.mmd` files per L4 spec; markdown embeds two SVG references. Audit regex stays `^R\d+:` (D rejected, A never added).

**Tech Stack:** Go 1.x, `targ` build tool, mermaid flowcharts.

**Spec:** `docs/superpowers/specs/2026-04-27-issue-604-c4-l4-c-hex-schema-design.md`

**Branch state:** worktree `/Users/joe/repos/personal/engram-issue-604` on branch `issue-604-c4-c-hex-schema`, reset to `main` (no implementation commits yet).

---

## Pre-flight

- [ ] **Step P-1: Verify clean baseline**

```bash
cd /Users/joe/repos/personal/engram-issue-604
git status                  # clean tree, on issue-604-c4-c-hex-schema
git log --oneline main..HEAD # empty (no commits ahead of main)
targ check-full
targ test
```

Expected: all green. STOP if not.

---

## Task 1: Audit-side regex — reject D, keep only R

**Files:** `dev/c4.go` (regex + audit error string), `dev/c4_test.go` (new tests).

- [ ] **Step 1.1: Add failing tests in `dev/c4_test.go`**

Mirror the existing `collectMermaidFindings` test pattern. Add:

```go
func TestEdgeIDPrefix_AcceptsROnly(t *testing.T) {
    t.Parallel()
    block := blockWithEdgeLabel("R1: ok")
    findings := collectMermaidFindings(block, /* …path/level args… */)
    for _, f := range findings {
        if f.ID == "edge_id_missing" {
            t.Fatalf("R-edge unexpectedly flagged: %+v", f)
        }
    }
}

func TestEdgeIDPrefix_RejectsDEdges(t *testing.T) {
    t.Parallel()
    block := blockWithEdgeLabel("D1: legacy")
    findings := collectMermaidFindings(block, /* … */)
    var hits int
    for _, f := range findings {
        if f.ID == "edge_id_missing" && strings.Contains(f.Detail, "R<n>:") {
            hits++
        }
    }
    if hits != 1 {
        t.Fatalf("expected 1 D-edge rejection citing R<n>:, got %d", hits)
    }
}
```

If a `blockWithEdgeLabel` helper does not yet exist, write the smallest possible synthetic `mermaidBlock` inline.

Both tests must use `t.Parallel()`.

- [ ] **Step 1.2: Confirm tests fail**

```bash
targ test 2>&1 | grep -E 'TestEdgeIDPrefix|FAIL'
```

- [ ] **Step 1.3: Update `dev/c4.go`**

Replace `dev/c4.go:141-144`:

```go
// edgeIDPrefix accepts R<n>: (runtime call). D<n>: (legacy DI back-edge)
// is rejected — the wiring relationship now lives in a separate wiring
// diagram, not as a back-edge in the call diagram.
edgeIDPrefix  = regexp.MustCompile(`^R\d+\s*:`)
```

Update audit error message at `dev/c4.go:810-815`:

```go
Detail: fmt.Sprintf("edge %q->%q label %q does not start with R<n>:", edge.from, edge.to, edge.label),
```

- [ ] **Step 1.4: Confirm tests pass**

```bash
targ test 2>&1 | grep -E 'TestEdgeIDPrefix|FAIL|PASS'
targ check-full
```

- [ ] **Step 1.5: Commit**

```bash
git add dev/c4.go dev/c4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): audit accepts R-edges only, rejects D-edges

Per #604: DI is represented in a separate wiring diagram, not as a
back-edge in the call diagram. The global audit regex no longer accepts
D<n>: prefixes.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Drop D-edge handling, drop Dotted, restrict L4 edge regex to R

**Files:** `dev/c4_l4.go` (regex + L4Edge struct + validateL4NodeIDs), `dev/c4_l4_test.go`.

- [ ] **Step 2.1: Find existing D references**

```bash
grep -n 'dEdgeIDPrefix\|D-edge\|D<n>\|Rdi\|Dotted' dev/c4_l4.go dev/c4_l4_test.go
```

Catalog every site for the implementer to address.

- [ ] **Step 2.2: Add failing test for R-only at L4**

In `dev/c4_l4_test.go`, add:

```go
func TestL4Spec_RejectsDEdges(t *testing.T) {
    t.Parallel()
    spec := validL4Spec()
    spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
        ID: "D1", From: "S2-N3-M3", To: "S2-N3-M2", Label: "legacy",
    })
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "R<n>") {
        t.Fatalf("expected D-edge rejection mentioning R<n>, got: %v", err)
    }
}
```

If the existing `validL4Spec()` fixture happens to set a D-edge or `Dotted: true`, also remove that from the fixture (the fixture should be a happy-path only).

- [ ] **Step 2.3: Confirm test fails**

- [ ] **Step 2.4: Update `dev/c4_l4.go`**

- Replace the regex variable with:

  ```go
  var rEdgeIDPrefix = regexp.MustCompile(`^R\d+$`)
  ```

- Update `validateL4NodeIDs` to use `rEdgeIDPrefix` and emit the error message `"diagram.edges[%d].id %q: must match R<n> (call relationship); D<n> (legacy DI back-edge) is no longer accepted"`.

- Remove the `Dotted bool` field from `L4Edge` and any references (likely in `emitL4MermaidEdge` — simplify it to always emit `-->`).

- [ ] **Step 2.5: Confirm tests pass + targ check-full green**

- [ ] **Step 2.6: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
refactor(c4): L4 accepts R-edges only; drop D + Dotted

L4Edge no longer carries a Dotted field. The per-level validator regex
is now ^R\d+$. D-edges produce a clear rejection.

AI-Used: [claude]
EOF
)"
```

---

## Task 3: Add `Properties []string` to `L4Edge`; render bracketed suffix

**Files:** `dev/c4_l4.go` (L4Edge struct + emitL4MermaidEdge), `dev/c4_l4_test.go`.

- [ ] **Step 3.1: Add failing renderer test**

```go
func TestEmitL4MermaidEdge_AppendsPropertySuffix(t *testing.T) {
    t.Parallel()
    edge := L4Edge{
        ID: "R8", From: "S2-N3-M3", To: "S2-N3-M4",
        Label: "strips transcript", Properties: []string{"P3", "P4", "P9"},
    }
    var buf bytes.Buffer
    emitL4MermaidEdge(&buf, edge)
    out := buf.String()
    if !strings.Contains(out, "R8: strips transcript [P3, P4, P9]") {
        t.Fatalf("expected bracketed property suffix, got: %s", out)
    }
}

func TestEmitL4MermaidEdge_OmitsSuffixWhenNoProperties(t *testing.T) {
    t.Parallel()
    edge := L4Edge{
        ID: "R3", From: "S2-N3-M2", To: "S2-N3-M3",
        Label: "constructs + invokes",
    }
    var buf bytes.Buffer
    emitL4MermaidEdge(&buf, edge)
    out := buf.String()
    if strings.Contains(out, "[") {
        t.Fatalf("expected no brackets, got: %s", out)
    }
}
```

- [ ] **Step 3.2: Confirm tests fail**

- [ ] **Step 3.3: Implement**

In `L4Edge`:

```go
type L4Edge struct {
    ID         string   `json:"id"`
    From       string   `json:"from"`
    To         string   `json:"to"`
    Label      string   `json:"label"`
    Properties []string `json:"properties,omitempty"`
}
```

In `emitL4MermaidEdge`:

```go
func emitL4MermaidEdge(buf *bytes.Buffer, edge L4Edge) {
    from := strings.ToLower(edge.From)
    to := strings.ToLower(edge.To)
    label := fmt.Sprintf("%s: %s", edge.ID, edge.Label)
    if len(edge.Properties) > 0 {
        label = fmt.Sprintf("%s [%s]", label, strings.Join(edge.Properties, ", "))
    }
    fmt.Fprintf(buf, "    %s -->|%q| %s\n", from, label, to)
}
```

- [ ] **Step 3.4: Confirm tests pass**

- [ ] **Step 3.5: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): L4 R-edges carry inline property IDs in brackets

L4Edge.Properties is rendered as " [P<a>, P<b>, …]" appended to the
mermaid label. Restores property linkage to outward relationships.

AI-Used: [claude]
EOF
)"
```

---

## Task 4: Slim `L4DepRow` and `L4WireRow`; add wrapped-entity validation

**Files:** `dev/c4_l4.go` (struct + emitters + validator), `dev/c4_l4_test.go`.

- [ ] **Step 4.1: Add failing tests**

```go
func TestL4DepRow_HasSlimSchema(t *testing.T) {
    t.Parallel()
    row := L4DepRow{
        Field: "summarizer", Type: "SummarizerI",
        WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "c3-engram-cli-binary.md",
        WrappedEntityID: "S2-N3-M7",
        Properties: []string{"P5", "P6"},
    }
    raw, err := json.Marshal(row)
    if err != nil { t.Fatalf("marshal: %v", err) }
    s := string(raw)
    for _, want := range []string{"field", "wired_by_id", "wrapped_entity_id"} {
        if !strings.Contains(s, want) {
            t.Errorf("missing %q: %s", want, s)
        }
    }
    for _, gone := range []string{"concrete_adapter", "wired_adapter", "concrete_value", "consumer_field"} {
        if strings.Contains(s, gone) {
            t.Errorf("legacy field %q still present: %s", gone, s)
        }
    }
}

func TestL4Spec_RejectsManifestWrappedEntityNotInDiagram(t *testing.T) {
    t.Parallel()
    spec := validL4Spec()
    spec.DependencyManifest = []L4DepRow{
        {
            Field: "ghost", Type: "Ghost",
            WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "c3-engram-cli-binary.md",
            WrappedEntityID: "S99-NOT-IN-DIAGRAM",
            Properties: nil,
        },
    }
    err := validateL4Spec(spec)
    if err == nil || !strings.Contains(err.Error(), "S99-NOT-IN-DIAGRAM") {
        t.Fatalf("expected wrapped-entity validation failure, got: %v", err)
    }
}
```

- [ ] **Step 4.2: Confirm tests fail**

- [ ] **Step 4.3: Replace structs**

```go
type L4DepRow struct {
    Field           string   `json:"field"`
    Type            string   `json:"type"`
    WiredByID       string   `json:"wired_by_id"`
    WiredByName     string   `json:"wired_by_name"`
    WiredByL3       string   `json:"wired_by_l3"`
    WiredByL4       string   `json:"wired_by_l4,omitempty"`
    WrappedEntityID string   `json:"wrapped_entity_id"`
    Properties      []string `json:"properties"`
}

type L4WireRow struct {
    Field           string `json:"field"`
    Type            string `json:"type"`
    ConsumerID      string `json:"consumer_id"`
    ConsumerName    string `json:"consumer_name"`
    ConsumerL3      string `json:"consumer_l3"`
    ConsumerL4      string `json:"consumer_l4,omitempty"`
    WrappedEntityID string `json:"wrapped_entity_id"`
}
```

- [ ] **Step 4.4: Add wrapped-entity validation**

Add a `validateL4Manifest` helper called from `validateL4Spec`:

```go
func validateL4Manifest(spec *L4Spec) error {
    known := map[string]bool{}
    for _, n := range spec.Diagram.Nodes {
        known[n.ID] = true
    }
    for i, row := range spec.DependencyManifest {
        if row.WrappedEntityID == "" {
            return fmt.Errorf("dependency_manifest[%d]: wrapped_entity_id must be non-empty", i)
        }
        if !known[row.WrappedEntityID] {
            return fmt.Errorf("dependency_manifest[%d]: wrapped_entity_id %q does not match any diagram node",
                i, row.WrappedEntityID)
        }
    }
    return nil
}
```

- [ ] **Step 4.5: Update emitters**

`emitL4DependencyManifest`:

```go
buf.WriteString("## Dependency Manifest\n\n")
buf.WriteString("Each row is one DI seam the focus consumes. The wrapped entity is the diagram\n")
buf.WriteString("node (component or external) the seam ultimately drives behavior against; it\n")
buf.WriteString("must also appear on the call diagram. The wiring diagram dedupes manifest\n")
buf.WriteString("rows by wrapped entity.\n\n")
buf.WriteString("| Field | Type | Wired by | Wrapped entity | Properties |\n")
buf.WriteString("|---|---|---|---|---|\n")
for _, row := range spec.DependencyManifest {
    emitL4DepRow(buf, row)
}
buf.WriteString("\n")
```

`emitL4DepRow`:

```go
func emitL4DepRow(buf *bytes.Buffer, row L4DepRow) {
    wiredBy := fmt.Sprintf("[%s · %s](%s#%s)",
        row.WiredByID, row.WiredByName, row.WiredByL3,
        Anchor(row.WiredByID, row.WiredByName))
    if row.WiredByL4 != "" {
        wiredBy += fmt.Sprintf(" ([%s](%s))", row.WiredByL4, row.WiredByL4)
    }
    fmt.Fprintf(buf, "| `%s` | `%s` | %s | `%s` | %s |\n",
        row.Field, row.Type, wiredBy, row.WrappedEntityID,
        formatPropertyList(row.Properties))
}
```

`emitL4DIWires` and `emitL4WireRow`: parallel updates (drop adapter func columns, replace with `Wrapped entity` column showing `WrappedEntityID`). Header becomes `| Field | Type | Consumer | Wrapped entity |`.

- [ ] **Step 4.6: Confirm tests pass + targ check-full green**

- [ ] **Step 4.7: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
refactor(c4): slim L4 manifest/wires; validate wrapped_entity_id

L4DepRow becomes {field, type, wired_by_*, wrapped_entity_id,
properties}. L4WireRow drops adapter func names and gains
wrapped_entity_id. wrapped_entity_id is validated against
diagram.nodes — no manifest row may name an entity not on the
diagram (strict alignment).

AI-Used: [claude]
EOF
)"
```

---

## Task 5: Emit wiring mermaid block + second SVG embed

**Files:** `dev/c4_l4.go` (new `emitL4WiringMermaid`, updates to `c4L4Build` and `emitL4ContextSection` or equivalent), `dev/c4_l4_test.go`.

- [ ] **Step 5.1: Add failing renderer test**

```go
func TestEmitL4WiringMermaid_DedupesByWrappedEntity(t *testing.T) {
    t.Parallel()
    spec := validL4Spec()
    spec.DependencyManifest = []L4DepRow{
        {Field: "f1", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S3"},
        {Field: "f2", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S3"},
        {Field: "f3", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S2-N3-M7"},
    }
    spec.Diagram.Nodes = append(spec.Diagram.Nodes,
        L4Node{ID: "S3", Name: "Claude Code", Kind: "external"},
        L4Node{ID: "S2-N3-M7", Name: "anthropic", Kind: "component"},
    )
    var buf bytes.Buffer
    emitL4WiringMermaid(&buf, spec)
    out := buf.String()
    // Expect exactly two cli→focus edges, labeled "S3" and "S2-N3-M7".
    s3Count := strings.Count(out, `|"S3"|`)
    antCount := strings.Count(out, `|"S2-N3-M7"|`)
    if s3Count != 1 || antCount != 1 {
        t.Fatalf("expected one S3 edge and one S2-N3-M7 edge, got s3=%d ant=%d in:\n%s", s3Count, antCount, out)
    }
}
```

- [ ] **Step 5.2: Confirm test fails (function does not yet exist)**

- [ ] **Step 5.3: Implement `emitL4WiringMermaid`**

```go
// emitL4WiringMermaid emits the L4 wiring diagram: wirer→focus edges,
// one per (wirer, wrapped_entity) pair, label = WrappedEntityID. Multiple
// manifest rows that share both fields collapse into a single edge.
func emitL4WiringMermaid(buf *bytes.Buffer, spec *L4Spec) {
    buf.WriteString("%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%\n")
    buf.WriteString("flowchart LR\n")
    buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
    buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
    buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n")
    buf.WriteString("    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000\n")
    buf.WriteString("    classDef focus       fill:#facc15,stroke:#a16207,color:#000\n\n")

    // Build a node lookup so wirer and wrapped-entity nodes can be rendered with
    // their canonical shape and class. The focus is always rendered.
    nodeByID := map[string]L4Node{}
    for _, n := range spec.Diagram.Nodes {
        nodeByID[n.ID] = n
    }

    // Collect distinct nodes referenced by the wiring diagram.
    referenced := map[string]bool{spec.Focus.ID: true}
    type edgeKey struct{ wirer, wrapped string }
    seen := map[edgeKey]bool{}
    edges := []edgeKey{}
    for _, row := range spec.DependencyManifest {
        referenced[row.WiredByID] = true
        referenced[row.WrappedEntityID] = true
        key := edgeKey{row.WiredByID, row.WrappedEntityID}
        if seen[key] {
            continue
        }
        seen[key] = true
        edges = append(edges, key)
    }

    // Emit nodes in deterministic order (the order they appear on the call diagram).
    for _, n := range spec.Diagram.Nodes {
        if !referenced[n.ID] {
            continue
        }
        emitL4MermaidNode(buf, n)
    }
    buf.WriteString("\n")

    // Emit deduped wiring edges.
    for _, e := range edges {
        from := strings.ToLower(e.wirer)
        to := strings.ToLower(spec.Focus.ID)
        fmt.Fprintf(buf, "    %s -->|%q| %s\n", from, e.wrapped, to)
    }
    buf.WriteString("\n")

    emitL4MermaidClassesForNodes(buf, spec, referenced)
}

// emitL4MermaidClassesForNodes writes class assignments restricted to the
// referenced node set, mirroring emitL4MermaidClasses' structure.
func emitL4MermaidClassesForNodes(buf *bytes.Buffer, spec *L4Spec, referenced map[string]bool) {
    groups := map[string][]string{}
    for _, n := range spec.Diagram.Nodes {
        if !referenced[n.ID] {
            continue
        }
        groups[n.Kind] = append(groups[n.Kind], strings.ToLower(n.ID))
    }
    for _, class := range []string{"person", "external", "container", "component", "focus"} {
        ids := groups[class]
        if len(ids) == 0 {
            continue
        }
        fmt.Fprintf(buf, "    class %s %s\n", strings.Join(ids, ","), class)
    }
}
```

- [ ] **Step 5.4: Update `c4L4Build` to write the wiring mmd**

```go
mmdPath := filepath.Join(filepath.Dir(args.Input), "svg",
    strings.TrimSuffix(filepath.Base(args.Input), ".json")+".mmd")
wiringMmdPath := filepath.Join(filepath.Dir(args.Input), "svg",
    strings.TrimSuffix(filepath.Base(args.Input), ".json")+"-wiring.mmd")

var mmdBuf bytes.Buffer
emitL4Mermaid(&mmdBuf, spec)
var wiringBuf bytes.Buffer
emitL4WiringMermaid(&wiringBuf, spec)

if err := writeOrCheckMarkdown(mdPath, mdBuf.Bytes(), args.Check, args.NoConfirm); err != nil {
    return err
}
if err := writeOrCheckMarkdown(mmdPath, mmdBuf.Bytes(), args.Check, args.NoConfirm); err != nil {
    return err
}
return writeOrCheckMarkdown(wiringMmdPath, wiringBuf.Bytes(), args.Check, args.NoConfirm)
```

- [ ] **Step 5.5: Update markdown emit to embed both SVGs**

In `emitL4ContextSection` (or a new `emitL4WiringSection`), after the existing call-diagram embed, emit a wiring-diagram embed:

```go
fmt.Fprintf(buf, "## Wiring\n\n")
fmt.Fprintf(buf,
    "Each edge is one or more DI seams the wirer plugs into %s, deduped by the\n"+
        "wrapped entity (label = SNM ID). The Dependency Manifest below shows the\n"+
        "per-seam breakdown.\n\n",
    spec.Focus.Name)
fmt.Fprintf(buf, "![C4 %s wiring diagram](svg/c4-%s-wiring.svg)\n\n", spec.Name, spec.Name)
fmt.Fprintf(buf,
    "> Diagram source: [svg/c4-%s-wiring.mmd](svg/c4-%s-wiring.mmd). Re-render with\n"+
        "> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-%s-wiring.mmd "+
        "-o architecture/c4/svg/c4-%s-wiring.svg`.\n\n",
    spec.Name, spec.Name, spec.Name, spec.Name)
```

Place the call to this section between the existing context section and the dependency-manifest section in `emitL4Markdown`.

- [ ] **Step 5.6: Confirm tests pass + targ check-full green**

- [ ] **Step 5.7: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): emit L4 wiring diagram as a second mermaid block

c4-l4-build now writes <name>.mmd (call) and <name>-wiring.mmd
(wiring). The L4 markdown embeds both SVGs sequentially. Wiring
edges go wirer→focus, deduped by wrapped entity, label = SNM ID.

AI-Used: [claude]
EOF
)"
```

---

## Task 6: Doc-comment cleanup

**Files:** `dev/c4*.go` (comments only).

- [ ] **Step 6.1: Find stale references**

```bash
grep -rn 'D<n>\|Rdi\|DI back-edge\|D-edge\|R/D edges\|dotted back\|port\b\|A<n>' dev/c4*.go | grep -v _test
```

(Test names that assert D-edge rejection are intentional and stay.)

- [ ] **Step 6.2: Update each comment**

Replace D-edge references with R-only; rewrite the "separate bidirectional R/D edges" rationale in `emitL4ContextSection` to reflect the new (no-D) world. Remove any comment that mentions ports or A-edges.

- [ ] **Step 6.3: Run targ check-full + targ test (no behavior change expected)**

- [ ] **Step 6.4: Commit**

```bash
git add dev/c4*.go
git commit -m "$(cat <<'EOF'
docs(c4): drop stale D-edge / port / A-edge references in comments

AI-Used: [claude]
EOF
)"
```

---

## Task 7: Regenerate `c4-recall.json` under new schema (worked example)

**Files:** `architecture/c4/c4-recall.json`, `architecture/c4/c4-recall.md`, `architecture/c4/svg/c4-recall.mmd`, `architecture/c4/svg/c4-recall-wiring.mmd`, both SVGs.

The spec section `## Sample regeneration (worked example)` lists the exact node + R-edge + manifest content.

- [ ] **Step 7.1: Read spec and verify wiring against code**

Read `docs/superpowers/specs/2026-04-27-issue-604-c4-l4-c-hex-schema-design.md` `## Sample regeneration` section. Cross-check the wrap-target assignments against `internal/cli/cli.go` and `internal/cli/externalsources_adapters.go`. Confirm:

- `Finder` → `recall.NewSessionFinder(&osDirLister{})` → S3 (Claude Code session files).
- `Reader` → `recall.NewTranscriptReader(&osFileReader{})` → S3.
- `SummarizerI` → `recall.NewSummarizer(&haikuCallerAdapter{...})` → S2-N3-M7 (anthropic).
- `MemoryLister` → `memory.NewLister(...)` → S2-N3-M5.
- `externalFiles` → `externalsources.Discover(...)` result → S2-N3-M6.
- `fileCache` → `externalsources.NewFileCache(os.ReadFile)` → S2-N3-M6.
- `statusWriter` → `os.Stderr` → S3.

If any wiring disagrees with the spec's example, STOP and report — the spec needs an update before proceeding.

- [ ] **Step 7.2: Edit `architecture/c4/c4-recall.json`**

Replace the spec content with:

- 7 component nodes (cli, recall focus, context, memory, externalsources, anthropic) + 1 external node (S3 · Claude Code).
- 6 R-edges with property arrays per the spec.
- 7 manifest rows with `wrapped_entity_id` per the spec.
- `di_wires`: empty array (recall consumes only).
- Drop any old `concrete_adapter` strings, drop `dotted` flags, drop port nodes / A-edges if present.

- [ ] **Step 7.3: Run the L4 builder**

```bash
targ c4-l4-build --input=architecture/c4/c4-recall.json --noconfirm
```

Iterate until the validator accepts the JSON.

- [ ] **Step 7.4: Render both SVGs**

```bash
npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall.mmd -o architecture/c4/svg/c4-recall.svg
npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall-wiring.mmd -o architecture/c4/svg/c4-recall-wiring.svg
```

- [ ] **Step 7.5: Eyeball the rendered markdown**

Open `architecture/c4/c4-recall.md`. Confirm:

- Two SVG embeds (call + wiring) in that order.
- Call diagram: 7 nodes, 6 R-edges with `[P…]` suffixes, S3 visible as external, no D-edge, no port circles.
- Wiring diagram: cli + recall + 4 wrapped-entity nodes, 4 wiring edges, deduplicated.
- Manifest table: 5 columns (Field | Type | Wired by | Wrapped entity | Properties), 7 rows.

- [ ] **Step 7.6: Run audit**

```bash
targ c4-audit --file=architecture/c4/c4-recall.md
```

Expected clean. If pre-existing findings unrelated to #604 surface, do not fix them in this commit.

- [ ] **Step 7.7: Run full check + tests**

```bash
targ check-full
targ test
```

- [ ] **Step 7.8: Commit**

```bash
git add architecture/c4/c4-recall.json architecture/c4/c4-recall.md architecture/c4/svg/c4-recall.mmd architecture/c4/svg/c4-recall-wiring.mmd architecture/c4/svg/c4-recall.svg architecture/c4/svg/c4-recall-wiring.svg
git commit -m "$(cat <<'EOF'
feat(c4): regenerate c4-recall under two-diagram schema (#604 worked example)

Call diagram: 7 nodes incl. S3 carryover, 6 R-edges with property
linkage. Wiring diagram: 4 deduped edges from cli to recall.
Manifest: 7 seam rows with wrapped_entity_id alignment.

AI-Used: [claude]
EOF
)"
```

---

## Task 8: Final verification

- [ ] **Step 8.1: Acceptance grep**

```bash
cd /Users/joe/repos/personal/engram-issue-604

# No D-edge / port / A-edge handling left in source (test names allowed).
grep -rn 'D<n>\|Rdi\|DI back-edge\|D-edge\|port\b\|A<n>' dev/c4*.go | grep -v _test || echo OK

# No adapter func name fields anywhere.
grep -rn 'concrete_adapter\|wired_adapter\|concrete_value' dev/c4*.go architecture/c4/c4-recall.json || echo OK

targ check-full
targ test
```

- [ ] **Step 8.2: Architecture diff matches scope**

```bash
git diff main..HEAD --name-only architecture/c4/
```

Expected only: `c4-recall.json`, `c4-recall.md`, `svg/c4-recall.mmd`, `svg/c4-recall-wiring.mmd`, `svg/c4-recall.svg`, `svg/c4-recall-wiring.svg`.

- [ ] **Step 8.3: Branch summary**

```bash
git log --oneline main..HEAD
git status
```

- [ ] **Step 8.4: Report**

Hand control back to the human for review. Do not push or merge.

---

## Self-review notes

- **Spec coverage:** Each schema/builder/renderer/audit item from the spec maps to a task (1: audit regex; 2: drop D + Dotted + R-only validator; 3: Properties on R-edges; 4: slim manifest + wrapped-entity validation; 5: wiring diagram emit; 6: doc cleanup; 7: worked example; 8: verification).
- **Type consistency:** `L4DepRow.WrappedEntityID` and the validator and the wiring emitter all use the same field name. `L4Edge.Properties` is the same name everywhere.
- **No placeholders.** Each step has exact code or exact commands.
- **TDD:** every behavior task starts with a failing test.
