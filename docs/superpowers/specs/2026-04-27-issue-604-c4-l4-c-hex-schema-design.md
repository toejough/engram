# Issue #604 — c4 L4: separate C4 call diagram and wiring diagram (drop D-edges)

## Background

The current L4 schema represents dependency injection with `D<n>` back-edges from focus → wirer (dotted, wrong direction per C4's "arrow = initiates" rule). #603's brainstorm rejected that convention.

The chosen replacement: **two diagrams per L4 spec**, not one combined diagram.

1. **C4 call diagram** — strict C4 with SNMPR-style nodes and `R<n>` runtime-call edges only. Property IDs that an R-edge realizes appear inline in the edge label.
2. **Wiring diagram** — a small companion view showing the wirer plugging dependencies into the focus. Each edge from wirer → focus is labeled with the SNM ID of the entity being wired in. Multiple DI seams that wire the same entity collapse into a single edge.

The previous (wrong) iteration of this spec collapsed both into one diagram with `port` nodes and `A<n>` edges. That direction was abandoned; the correct design is two distinct diagrams.

## Vocabulary

- **R-edge** — a runtime call from one node to another in the C4 sense. ID `R<n>`. Label `R<n>: <description> [P<a>, P<b>, ...]` where the bracketed list is the property IDs (short form) the call realizes. Example: `R8: strips transcript via TranscriptReader [P3, P4, P9, P10]`.
- **DI seam** — one injected interface or function value the focus consumes (Finder, Reader, SummarizerI, etc.).
- **Wrapped entity** — the diagram entity (component or external) the wirer plugs into the focus for a given DI seam. Examples: cli wires `SummarizerI` so it ultimately drives behavior against the `anthropic` component → wrapped entity = `S2-N3-M7`. cli wires `statusWriter` so it ultimately drives behavior against the OS (Claude Code's stderr) → wrapped entity = `S3` (Claude Code).
- **Wiring edge** — an edge in the wiring diagram from wirer → focus, label = SNM ID of the wrapped entity. Multiple DI seams that share a wrapped entity collapse into a single wiring edge.

## Core principles

1. **IO is dependency-injected.** Direct file/network primitives are forbidden inside `internal/`. Every external effect enters through an injected interface. (Project rule from CLAUDE.md, restated here because it drives the strict-alignment rule below.)
2. **External systems must be represented in the architecture.** If recall transitively crosses to S3 via three DI seams, S3 appears as a node on c4-recall's C4 diagram with at least one R-edge from recall to it.
3. **Strict alignment.** Every wiring edge's wrapped-entity label must be a node already on the C4 call diagram. Equivalently: every external the focus crosses to via DI must also be a runtime-call target visible on the L4. The wiring diagram does not introduce nodes the C4 diagram lacks.

## Schema additions to `L4Spec`

### `L4Edge` — property linkage on R-edges

Add `Properties []string` to `L4Edge`. JSON tag `properties`. Each entry is a short-form property ID (e.g. `P3`) — the focus's hierarchical prefix is implied. Optional; absence renders as no `[…]` suffix on the edge label.

Renderer behavior: if `Properties` is non-empty, append ` [P<a>, P<b>, ...]` to the mermaid edge label. Use `formatPropertyList` (already exists for the manifest) to keep collapse-runs behavior consistent.

### `L4DepRow` — manifest row schema

```go
type L4DepRow struct {
    Field           string   `json:"field"`            // DI seam field name in the focus
    Type            string   `json:"type"`             // Go interface or type
    WiredByID       string   `json:"wired_by_id"`      // wirer's hierarchical ID
    WiredByName     string   `json:"wired_by_name"`
    WiredByL3       string   `json:"wired_by_l3"`      // L3 file the wirer lives in
    WiredByL4       string   `json:"wired_by_l4,omitempty"`
    WrappedEntityID string   `json:"wrapped_entity_id"`// SNM ID of the wrapped diagram entity (e.g. S3, S2-N3-M7)
    Properties      []string `json:"properties"`       // property IDs (short form) this seam realizes
}
```

Drops from the original schema: `concrete_adapter` (no adapter func names anywhere in the artifact).

### `L4WireRow` — provider-side reciprocal row

```go
type L4WireRow struct {
    Field           string `json:"field"`              // consumer-side DI seam field
    Type            string `json:"type"`
    ConsumerID      string `json:"consumer_id"`
    ConsumerName    string `json:"consumer_name"`
    ConsumerL3      string `json:"consumer_l3"`
    ConsumerL4      string `json:"consumer_l4,omitempty"`
    WrappedEntityID string `json:"wrapped_entity_id"`
}
```

Drops: `wired_adapter`, `concrete_value`, `consumer_field` (the last is redundant with `Field`).

### Wiring diagram derivation

Not a new JSON section. Derived at render time from the manifest:

- Source nodes = unique `WiredByID` values across rows.
- Destination = the focus.
- Edges = grouped by `(WiredByID, WrappedEntityID)`. Multiple manifest rows that share both a wirer and a wrapped entity produce one wiring edge.
- Edge label = `WrappedEntityID` (the SNM ID, unadorned).
- Wiring-diagram nodes = the union of {wirers, focus, every distinct `WrappedEntityID`}. The wrapped-entity nodes are rendered with the same shape/class as on the C4 call diagram.

### Validation: strict alignment

`validateL4Spec` rejects any `WrappedEntityID` in `DependencyManifest` that does not match an `id` on `Diagram.Nodes`. This is the enforcement of the externals-must-be-represented principle at the L4 level. Cross-level checks (against the L3 parent's external set) are still #605's job.

## Dropped from prior version of this spec

- `port` node kind — gone. No PT IDs anywhere.
- `A<n>` edge kind — gone.
- `Dotted` field on `L4Edge` — gone (no dotted edges remain at L4).
- `D<n>` edges — gone, no fallback. The schema rejects any edge whose ID matches `^D\d+`.
- `concrete_adapter` field on the manifest — gone.

## Builder, renderer, audit changes

### `dev/c4.go` (audit-side)

- Edge ID regex stays `^R\d+\s*:` (D dropped, A never added).
- Audit error message: `does not start with R<n>:`.

### `dev/c4_l4.go`

- `L4Edge`: add `Properties []string` field; remove `Dotted` field.
- Replace `dEdgeIDPrefix` with `rEdgeIDPrefix = ^R\d+$`. `validateL4NodeIDs` rejects non-R edges with a clear error.
- Slim `L4DepRow` and `L4WireRow` per schema above.
- Add validation: every `DependencyManifest[i].WrappedEntityID` must match a `Diagram.Nodes[j].ID`.
- `emitL4MermaidEdge`: append ` [P<a>, P<b>, ...]` when `edge.Properties` non-empty.
- New: `emitL4WiringMermaid` produces a second mermaid file. Outputs `flowchart LR` with the wirer + focus + wrapped-entity nodes (carry shape/class from the call diagram), and one solid edge per `(wirer, wrapped_entity)` group with label = wrapped entity SNM ID.
- `emitL4Markdown` embeds two SVG references in sequence under the call section (call first, wiring second), each with its own caption and re-render hint.
- `c4L4Build` writes both mmd files: `<dir>/svg/<name>.mmd` (call) and `<dir>/svg/<name>-wiring.mmd` (wiring).
- `emitL4DependencyManifest` table preamble + headers updated to match the new row shape: `| Field | Type | Wired by | Wrapped entity | Properties |`.

### `dev/c4_audit_ext.go`

- No structural change. The legacy D-edge defensive guard (Task in earlier wrong-direction plan) is not introduced; audit catches D via the global regex's rejection.

## Test additions

### Schema (`dev/c4_l4_test.go`)

- R-edge with `Properties: ["P3", "P4"]` validates and renders the bracketed suffix.
- D-edge rejected (legacy guard).
- Manifest row whose `WrappedEntityID` doesn't match a diagram node → rejected.
- Manifest row with the old `ConcreteAdapter` field on JSON → rejected (DisallowUnknownFields).

### Renderer

- Golden mermaid for call view of c4-recall: asserts R-edge labels with property suffixes, no port nodes, no A-edges, no D-edges, S3 present as external.
- Golden mermaid for wiring view of c4-recall: asserts wirer + focus + 4 wrapped-entity nodes, exactly 4 wiring edges with deduplicated labels, no R-edges in the wiring block.

### Audit (`dev/c4_test.go`)

- D-edge in input → `edge_id_missing` finding citing `R<n>:` only.
- R-edge with property suffix accepted.

## Sample regeneration (worked example)

Regenerate `architecture/c4/c4-recall.json` and the rendered markdown + two SVGs.

### Concrete content

**Call diagram nodes** (kind/ID):
- `S2-N3-M2 · cli` (component)
- `S2-N3-M3 · recall` (focus)
- `S2-N3-M4 · context` (component)
- `S2-N3-M5 · memory` (component)
- `S2-N3-M6 · externalsources` (component)
- `S2-N3-M7 · anthropic` (component)
- `S3 · Claude Code` (external — carried over from L3)

**Call diagram edges:**
- `R3 cli → recall: constructs Orchestrator + invokes Recall / RecallMemoriesOnly`
- `R8 recall → context: strips transcript via context.StripSeqAndConvertHumanTurns [P3, P4]`
- `R9 recall → memory: lists memories via MemoryLister [P11, P12, P13, P15]`
- `R10 recall → anthropic: ranks + extracts + summarizes via SummarizerI [P5, P6, P7, P8, P11, P12, P13, P14, P15, P16]`
- `R11 recall → externalsources: discovers + reads CLAUDE.md/rules/auto-memory/skills via FileCache + externalFiles [P5, P6, P7, P8, P17]`
- `R12 recall → S3: reads session listings (Finder), transcript bytes (Reader), writes status (statusWriter) [P1, P2, P3, P4, P9, P10, P18]`

**Manifest rows** (one per DI seam):

| Field | Type | Wired by | Wrapped entity | Properties |
|---|---|---|---|---|
| `finder` | `Finder` | cli | S3 | P1, P2, P9 |
| `reader` | `Reader` | cli | S3 | P3, P4, P9, P10 |
| `summarizer` | `SummarizerI` | cli | S2-N3-M7 | P5–P8, P11–P16 |
| `memoryLister` | `MemoryLister` | cli | S2-N3-M5 | P11–P13, P15 |
| `externalFiles` | `[]externalsources.ExternalFile` | cli | S2-N3-M6 | P5–P8 |
| `fileCache` | `*externalsources.FileCache` | cli | S2-N3-M6 | P5–P7, P17 |
| `statusWriter` | `io.Writer` | cli | S3 | P18 |

**Wiring diagram** (derived; 4 edges from 7 manifest rows):
- `cli → recall`, label `S2-N3-M5` (covers `memoryLister`)
- `cli → recall`, label `S2-N3-M7` (covers `summarizer`)
- `cli → recall`, label `S2-N3-M6` (covers `externalFiles`, `fileCache`)
- `cli → recall`, label `S3` (covers `finder`, `reader`, `statusWriter`)

## Out of scope

- L1/L2/L3 schema changes. The two-diagram structure is L4-specific in this iteration. If higher levels gain wiring views in the future, that's a separate ticket.
- Bulk regen of the other 18 L4 specs. Only `c4-recall` is regenerated under #604.
- #605 (cross-level external completeness audit), #606 (skill rewrite), #586 (from-scratch regen).

## Acceptance

- `targ check-full` and `targ test` pass.
- `architecture/c4/c4-recall.json` regenerated under the new schema.
- Rendered markdown shows two SVG embeds (call + wiring) with the content listed above.
- No D-edges, no port nodes, no A-edges, no `concrete_adapter` field anywhere in `dev/c4*.go` or `architecture/c4/c4-recall.*`.
- All 18 non-recall L4 specs still on disk unchanged; they will fail validation under the new schema and be regenerated separately.
