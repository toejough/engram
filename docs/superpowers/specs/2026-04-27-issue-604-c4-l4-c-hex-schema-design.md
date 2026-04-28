# Issue #604 — c4 C-hex schema across all levels (ports + A-edges, externals required, D-edges dropped)

## Background

The current schema represents dependency injection at L4 with `D<n>` back-edges from focus → wirer (dotted, wrong direction per C4's "arrow = initiates" rule). Brainstorm under #603 chose the C-hex (ports & adapters) shape: ports are first-class nodes owned by the consumer of an interface, and edges from the wirer indicate which diagram entity is plugged into each port.

Original framing put this at L4 only. Final scope is **all four C4 levels**: a port/adapter relationship belongs on every diagram whose level shows the boundary the wire crosses. If the wirer and consumer live in different containers, the port appears at L3 and L4. If they live in different systems, the port appears at L1, L2, L3, and L4.

This spec covers the schema, builder, renderer, and audit changes needed to support C-hex notation. Bulk regeneration of all L4 specs and the skill rewrite (#606) and full external-completeness enforcement (#605) and from-scratch regen verification (#586) remain separately tracked.

## Vocabulary

- **Port** — a named first-class node owned by the consumer of an interface. Shape: circle. Label: the interface or field name (e.g. `Finder`, `SummarizerI`).
- **Wirer** — the node that supplies the adapter that satisfies a port.
- **A-edge** — a solid arrow from wirer → port. Label = the diagram entity being plugged in.
- **Wrapped entity** — the diagram node referenced by the A-edge label. Whatever the adapter ultimately drives behavior against.

## Escalation rule (the principle)

A port owned by consumer C, wired by W:
- **Always** appears at the level showing C natively (a component-level port shows at L4, a container-level port at L3, etc.).
- Also appears at any **higher** level whose nodes can resolve W and C to *different* nodes at that level. (If both collapse to the same container at L3, the port is invisible at L3 and only the L4 view shows it.)

Cross-level *enforcement* of this rule (an audit check that walks parent specs and rejects under-represented ports) is **out of scope for #604**. #604 lands the per-level schema/builder/renderer support so that authors can add port nodes and A-edges where needed; a follow-up ticket will add the cross-level audit.

## Schema additions

These additions apply to every level that has a JSON spec (L2, L3, L4). L1 is hand-authored markdown; the global audit regex update (below) is what makes L1 accept the new edge kind.

### New element kind: `port`

- **Shape:** circle (`((Label))` in mermaid). Add `classDef port` to every level renderer.
- **ID:** `<owner-id>-PT<n>`, where `<owner-id>` is the hierarchical ID of the node that owns the port and `<n>` is 1-indexed and monotonic per owner. Examples:
  - L4 port on `S2-N3-M3` (component) → `S2-N3-M3-PT1`.
  - L3 port on `S2-N3` (container) → `S2-N3-PT1`.
  - L2 port on `S2` (system) → `S2-PT1`.
- **Label:** the interface or field name as the consumer declares it (`Finder`, `Reader`, `SummarizerI`).
- **Validation:** ID must extend the owning node's hierarchical ID by exactly one `-PT<n>` segment; `<n>` must be 1-indexed and monotonic within the owner.
- **No collision with `-P<n>` properties:** `-P` is a single letter prefix at L4 leaves, `-PT` is two letters; the parsers distinguish them.

### New edge kind: `A<n>` (Adapter plugs into Port)

- **Direction:** wirer → port. Solid arrow.
- **Label:** the diagram entity being plugged in. Names a node already present on this diagram (a component, container, system, person, external, or — when the focus drives an outside system through the port — that external).
  - Examples: `A1: anthropic`, `A2: memory`, `A3: S6 · operating system`.
- **Edge ID regex (global, in `c4.go`):** extends to `^[RA]\d+\s*:`. D drops in the same change (D-edges only appear at L4 in current data; verified before scoping).
- **Per-level validator regex (in L4):** extends to `^[RA]\d+$` (replaces current `^[RD]\d+$`).
- **Validation:** A-edge label must reference a node ID or label that exists on the diagram. If the referenced entity is not present as a node, validation fails.

### Dropped: `D<n>` edges

- No fallback, no migration window. Schema rejects any edge whose ID matches `^D\d+`.
- The 19 existing L4 specs are regenerated under the new schema. #604 includes regenerating one representative focus (`recall`) end-to-end as a worked example; bulk regeneration of the other 18 lands separately.

### `dependency_manifest` table (L4) — kept, slimmed

- Purpose: human-readable catalog of what the diagram already shows. Not consumed by the binary or audit.
- Row schema: `{port_id, port_name, port_type, wirer_id, wirer_name, wirer_l3, wirer_l4, wrapped_id, wrapped_name, wrapped_l3, wrapped_l4, properties}`.
  - `port_id`: the port node's hierarchical ID.
  - `port_name`: the interface/field name shown on the port node.
  - `port_type`: the Go interface or type the port satisfies.
  - `wirer_*`: the wirer's identity + cross-link targets (existing pattern, renamed from `wired_by_*`).
  - `wrapped_*`: the wrapped entity's identity + cross-link targets (new — what the adapter ultimately drives behavior against).
  - `properties`: existing list of property IDs that depend on this port.
- Drop fields: `concrete_adapter`. Adapter func names (`NewSummarizer`, `NewTranscriptReader`) are not represented anywhere — not on edges, not in tables.
- Markdown rendering: small table under the diagram. Columns map 1:1.

### `di_wires` table (L4 reciprocal) — kept, slimmed

- The provider-side mirror of `dependency_manifest`. Each row is one adapter the L4's focus wires for somebody else.
- Row schema: `{port_id, port_name, port_type, consumer_id, consumer_name, consumer_l3, consumer_l4, wrapped_id, wrapped_name, wrapped_l3, wrapped_l4}`.
- Drops: `wired_adapter`, `concrete_value` (both encode adapter-func info we no longer surface).

## Builder, renderer, audit changes

### `dev/c4.go` (audit-side, all levels)

- Update global edge ID regex from `^[RD]\d+\s*:` to `^[RA]\d+\s*:`.
- Update audit error messages from "starts with R<n>: or D<n>:" to "starts with R<n>: or A<n>:".
- No node-kind validation lives in c4.go directly; level-specific files are responsible.

### `dev/c4_l2.go` and `dev/c4_l3.go`

- Add `port` to the allowed `Kind` values for L2Element / L3Element.
- Add port ID validator: `<owner-id>-PT<n>`, level + monotonicity check.
- Add `classDef port` line to mermaid emitter and include `port` in the classOrder slice.
- Add `port` case to the level's node-shape switch (returns `((` / `))`).
- Add A-edge handling: solid arrow rendering of any `A<n>` relationship; A-edge label-references-known-node validation.
- No structural change to the `L1Relationship` type — A-edges reuse it; the new ID prefix is the only difference.

### `dev/c4_l4.go`

- Add `port` to allowed node kinds; add `validatePortID(id, focusID, index)` enforcing `<focus-path>-PT<n>` shape, level, and monotonicity.
- Replace `dEdgeIDPrefix = ^[RD]\d+$` with `aEdgeIDPrefix = ^[RA]\d+$`.
- Update `validateL4NodeIDs` error message to reference R/A, not R/D.
- Remove all D-edge handling: validator branches, JSON struct fields specific to D, and `Dotted` field on `L4Edge` (A-edges are solid; D was the only consumer).
- Update `L4DepRow` struct per "kept, slimmed" schema above. Update `L4WireRow` similarly.
- Add A-edge label-references-known-node validation.
- Update `emitL4DependencyManifest` preamble (remove "Rdi back-edge" reference) and `emitL4DepRow` column layout to match the new row.
- Update `emitL4DIWires` preamble and `emitL4WireRow` similarly.
- Add port classDef and port shape to L4 mermaid emitter.

### `dev/c4_audit_ext.go`

- No structural change for #604. (The full external-completeness audit is #605.)
- One addition: a finding `l4_d_edge_legacy` when an old D-edge is encountered in input (defensive guard; should fire only on un-regenerated specs).

## Test additions

### Schema (`dev/c4_l4_test.go`, `dev/c4_l3_test.go`, `dev/c4_l2_test.go`)

- Port ID accepted when shape `<owner-id>-PT<n>`, level matches owner + 1, monotonic.
- Port ID rejected when shape collides with `-P<n>`, when level wrong, when non-monotonic, when owner mismatch.
- A-edge accepted with label referencing an existing node ID/name.
- A-edge rejected with label referencing unknown entity.
- D-edge rejected at L4 always (legacy guard); existing D-edge test in `c4_l4_test.go` flipped from "accepted" to "rejected".
- Slim `L4DepRow` accepted; row containing `concrete_adapter` rejected (forward guard).

### Renderer

- Golden mermaid for `architecture/c4/c4-recall.json` (the worked example): asserts circle port nodes, `classDef port`, A-edge labels, R-edges still solid, no D-edges, slimmed manifest table.

### Audit (`dev/c4_audit_ext_test.go`, `dev/c4_test.go`)

- D-edge in input → rejected (with reasonable error).
- A-edge label "R3:" or other valid prefix accepted.
- Pre-existing R-only L1/L2/L3 specs still pass audit unchanged (regression guard).

## Sample regeneration (worked example)

As part of #604, regenerate `architecture/c4/c4-recall.json` and rendered markdown under the new schema. Concrete content:

- Port nodes for the seven `recall` DI seams: Finder, Reader, SummarizerI, MemoryLister, externalFiles, FileCache, statusWriter.
- A-edges from `cli` (the wirer) to each port, labeled with the wrapped entity name.
- Existing R-edges from `recall` → driven nodes preserved.
- No D-edges.
- Slim dependency_manifest table.

This regeneration verifies the builder/renderer/audit pipeline end-to-end. The other 18 L4 specs, plus any L1/L2/L3 hand-authored content that wants ports, are regenerated/edited separately.

## Out of scope

- **#605:** Full external-completeness audit pass against the L3 parent's external set.
- **#606:** Rewrite of `skills/c4/SKILL.md` to teach the C-hex convention.
- **#586:** From-scratch validation regen.
- Bulk regeneration of the 18 non-recall L4 specs.
- Cross-level escalation enforcement (the audit check that "this L4 port should also appear at L3 because wirer crosses container boundary"). To be filed as a follow-up issue.

## Acceptance

- `targ check-full` passes.
- `targ test` passes including new schema, renderer, and audit tests.
- `architecture/c4/c4-recall.json` regenerated under the new schema; rendered markdown shows port nodes, A-edges, no D-edges, slimmed manifest.
- `architecture/c4/` unchanged at L1/L2/L3 still passes audit (regression guard — no port nodes added there because no wires cross those boundaries today).
- No D-edge handling remains in `c4.go`, `c4_l2.go`, `c4_l3.go`, `c4_l4.go`.
