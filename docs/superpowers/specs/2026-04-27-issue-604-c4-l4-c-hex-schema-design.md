# Issue #604 — c4 L4 builder: C-hex schema (ports + A-edges, externals required, D-edges dropped)

## Background

The current L4 schema represents dependency injection with `D<n>` back-edges from focus → wirer (dotted, wrong direction per C4's "arrow = initiates" rule). Brainstorm under #603 chose the C-hex (ports & adapters) shape: ports are first-class nodes owned by the focus, and edges from the wirer indicate which diagram entity is plugged into each port.

This spec covers the schema, builder, renderer, and audit changes that make C-hex possible. Bulk regeneration of all 19 L4 specs and full external-completeness enforcement (#605) and the skill rewrite (#606) are tracked separately.

## Schema additions to `L4Spec`

### New node kind: `port`

- Owned by the focus component. One port per DI seam the focus consumes.
- **Shape:** circle (`((Name))` in mermaid). Add `classDef port` for styling.
- **ID:** `<focus-path>-PT<n>`, e.g. `S2-N3-M3-PT1`. Does not collide with `-P<n>` property IDs (different prefix length).
- **Label:** the interface/field name as the focus declares it (`Finder`, `Reader`, `SummarizerI`).
- **Validation:** ID must extend the focus node's hierarchical ID by exactly one `-PT<n>` segment; `<n>` must be 1-indexed and monotonic within the focus.

### New edge kind: `A<n>` (Adapter plugs into Port)

- **Direction:** wirer → port. Solid arrow.
- **Label:** the diagram entity being plugged in. Names a node already present on this L4 diagram (a component, person, external, or — when the focus drives an outside system through the port — the appropriate external node).
  - Examples: `A1: anthropic`, `A2: memory`, `A3: S6 · operating system`.
- **Edge ID regex:** `^[RA]\d+\s*:`. D is removed in the same change (see "Dropped: `D<n>` edges").
- **Validation:** A-edge label must reference a node ID or label that exists on the diagram. If the referenced entity is not present as a node, validation fails (this is the schema-level external-completeness signal; full audit lives in #605).

### Dropped: `D<n>` edges

- No fallback, no migration window. The schema rejects any edge whose ID matches `^D\d+`.
- All 19 existing L4 specs will be regenerated under the new schema. Scope of bulk regen for the other 18 components is tracked outside this ticket; #604 includes regenerating one representative focus (`recall`) to verify end-to-end.

### `dependency_manifest` table — kept, slimmed

- Purpose: human-readable catalog of what the diagram already shows. Not consumed by the binary or audit logic.
- Row schema: `{port_id, port_type, wirer_id, wrapped_entity_id}`.
- Drop fields: `concrete_adapter` (adapter func names like `NewSummarizer` are not represented anywhere — not on edges, not in the table).
- Markdown rendering: a small table under the diagram. Columns map 1:1 to the row schema.

## Builder, renderer, audit changes

### `dev/c4_l4.go`

- Add `port` to allowed node kinds.
- Add `validatePortID(id, focusID, index)` enforcing the `<focus-path>-PT<n>` shape, level, and monotonicity.
- Add A-edge handling alongside R: edge ID regex, label parsing, label-references-known-node check.
- Remove all D-edge handling: validator branches, JSON struct fields specific to D, any `dotted: true` defaulting tied to D.
- Update `DependencyManifest` row struct: drop `ConcreteAdapter`, drop `WiredBy*` if redundant with `WirerID`. New shape: `{PortID, PortType, WirerID, WrappedEntityID}`.

### `dev/c4_render.go`

- Render `port` nodes as `((Label))` mermaid syntax.
- Add `classDef port fill:<color>,stroke:<color>,color:<color>` (pick palette consistent with existing `classDef external` / `classDef focus`).
- Render A-edges as solid arrows with the label format `A<n>: <wrapped-entity-label>`.
- Remove D-edge rendering branches.
- Render the slimmed `dependency_manifest` table after the mermaid block.

### `dev/c4_audit_ext.go`

- Extend the unified edge ID regex to recognize `A`.
- Remove D-edge handling.
- Add finding kind: `l4_a_edge_unknown_target` when an A-edge label references a node ID/label not present on the diagram. (This is the local schema check; cross-level external-completeness checking against the L3 parent stays in #605.)
- Reject any D-edge encountered in input as `l4_d_edge_legacy`.

## Test additions

### Schema (`dev/c4_l4_test.go`)

- Port ID accepted when shape `<focus-path>-PT<n>`, level matches focus + 1, monotonic.
- Port ID rejected when shape collides with `-P<n>`, when level wrong, when non-monotonic, when focus mismatch.
- A-edge accepted with label that matches an existing node ID/label.
- A-edge rejected with label referencing unknown entity.
- D-edge rejected always (legacy guard).
- Slim manifest row accepted; manifest row with `concrete_adapter` rejected (forward guard).

### Renderer

- Golden mermaid for a representative port-bearing focus (`recall`): asserts circle nodes, `classDef port` line, A-edge labels, R-edges still solid, no D-edges, slim manifest table.

### Audit (`dev/c4_audit_ext_test.go`)

- D-edge in input → `l4_d_edge_legacy` finding.
- A-edge with unknown target → `l4_a_edge_unknown_target` finding.
- Clean spec → no findings.

## Sample regeneration

As part of this ticket, regenerate `architecture/c4/c4-recall.json` (and the rendered markdown) under the new schema. This serves as the worked example referenced by #606 and verifies the builder/renderer end-to-end. The other 18 L4 specs are regenerated outside this ticket's scope.

## Out of scope

- **#605:** Full external-completeness audit pass against the L3 parent's external set. #604 only adds the local "A-edge label must reference a diagram node" check.
- **#606:** Rewrite of `skills/c4/SKILL.md` to teach the C-hex convention.
- **#586:** From-scratch validation regen.
- Bulk regeneration of the 18 non-recall L4 specs.

## Acceptance

- `targ check-full` passes.
- `targ test` passes including new schema, renderer, and audit tests.
- `architecture/c4/c4-recall.json` regenerated under the new schema; rendered markdown shows port nodes, A-edges, no D-edges, slimmed manifest.
- No D-edge handling remains in any of `c4_l4.go`, `c4_render.go`, `c4_audit_ext.go`.
