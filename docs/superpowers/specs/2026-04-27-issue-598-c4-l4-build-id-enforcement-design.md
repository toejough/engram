# Issue #598 — c4-l4-build ID enforcement (with deletions)

## Problem

`c4-l4-build` accepts L4 specs containing node/edge IDs outside the documented `E<n>` / `R<n>` / `D<n>` namespaces and renders them to disk. The L4 audit (added in #599) catches the violations post-hoc, but the build itself does not.

Two failure modes recur in authoring:

1. An external system "feels like it should be on the diagram" → the agent invents an L4-only ID (e.g. `EXT1`, `ETARG`, `EJQ`, `EGO`) instead of adding the entity at L3 or routing through the Dependency Manifest's adapter column.
2. Multiple related calls share an R-number → the agent appends letter suffixes (`R2a`, `R2b`, `R9a`, `Rjq`) instead of allocating distinct sequential `R<n>`.

## Goal

Reject these violations at build time, so the loop closes at generation rather than during a separate audit pass. Treat this as a chance to **simplify**, not layer: validate the JSON spec directly (no mermaid parsing), and delete the now-redundant L4 audit checks.

## Approach

### 1. Build gate (dev/c4_l4.go)

Add `validateL4SpecIDs(spec *L4Spec, registry <L3 registry handle>) error` invoked from `c4L4Build` after `loadAndValidateL4Spec`. Validates the JSON struct directly — IDs are typed fields; no mermaid parsing needed.

Checks:
- Every `spec.Diagram.Nodes[i].ID` matches `^E\d+$` AND resolves to the L3 element registry for `spec.Parent`.
- Every `spec.Diagram.Edges[i].ID` matches `^R\d+$` or `^D\d+$` (no letter suffix).

Errors are aggregated (collect all, then return one combined error) so authors see the full violation list in one pass — no whack-a-mole. Each error names the offending ID, the rule it broke, and the fix:

- `edge "R2a": letter suffix not allowed; allocate a new sequential R<n>`
- `edge "Rjq": invalid namespace; use R<n> for call relationships or D<n> for DI back-edges`
- `node "EXT1": does not match E<n>`
- `node "EJQ": E-id not in L3 registry for parent c4-tokenresolver — add to L3 or describe in the Dependency Manifest's "Concrete adapter" column instead of inventing an L4-only ID`

Reuse existing regexes (`mermaidIDPrefix`, `dEdgeIDPrefix`) and the registry-loading helper that audit's `checkL4NodesInRegistry` already uses. If they need to move to a shared location to be callable from both the build target and the audit, do so as part of the refactor step.

### 2. Deletions (dev/c4.go, dev/c4_test.go)

Once (1) lands, audit's L4-specific mermaid validation is redundant — JSON is the only authoring surface, and the build enforces invariants there. Delete:

- `collectL4MermaidFindings` and its call site.
- `checkL4NodesInRegistry` and its call site.
- Audit tests that exercise fabricated-ID detection on rendered L4 `.md` files (e.g. the test asserting `node_id_missing` / `edge_id_missing` for `EXT1` / `X1` against rendered output).

L1-L3 audit checks remain unchanged.

### 3. Skill update (skills/c4/references/mermaid-conventions.md)

Driven by the `superpowers:writing-skills` skill (TDD: baseline behavior test → minimal edit → verify):

- **Red**: add a behavioral test in `skills/c4/tests/` that exercises a fabricated-ID authoring scenario. A model following current skill prose should either invent an `EXT1`-style ID or letter-suffix an R-number — establishing the gap.
- **Green**: minimal edit — one sentence in `mermaid-conventions.md` noting `c4-l4-build` rejects non-conforming IDs and to read the build error for the fix. Delete any prose subsumed by the build error message. No SKILL.md changes unless the pressure test reveals a gap there too. No template inline reminder.
- **Verify**: re-run the pressure test green.

### 4. Test strategy (dev/c4_l4_test.go)

TDD on the build gate:

- **Red**: spec fixtures with each violation class — non-`E<n>` node, valid-shape node not in registry, `R<n>` with letter suffix, `R<n>` from wrong namespace prefix. Each must make `c4-l4-build` exit non-zero with a clear message.
- **Green**: implement `validateL4SpecIDs`. Existing valid-spec tests stay green.
- **Refactor**: shared regex constants live in one place.

### 5. Final validation

- `targ check-full` clean.
- `targ c4-l4-build` against all 19 existing ledgers emits identical output (already clean post-#597/#600/#601).
- `targ c4-audit` still runs across all ledgers cleanly without the deleted L4 checks.

## Out of scope

- R-number cross-check against the L3 relationships table — extra cross-spec coupling for marginal gain over regex+registry; keep simple.
- Template inline reminder — build error is the teacher.
- SKILL.md prose additions beyond what the writing-skills pressure test forces.
- L1-L3 audit changes.

## Affected files

- `dev/c4_l4.go` — add `validateL4SpecIDs`, wire into `c4L4Build`.
- `dev/c4_l4_test.go` (new or existing) — red tests for each violation class.
- `dev/c4.go` — delete `collectL4MermaidFindings`, `checkL4NodesInRegistry`, call sites.
- `dev/c4_test.go` — delete corresponding audit tests.
- `skills/c4/references/mermaid-conventions.md` — minimal note on build-gate enforcement.
- `skills/c4/tests/` — pressure test for the skill edit.

## Related

- #599 (closed) — L4 audit gap that surfaced these violations.
- #597, #600, #601 (closed) — companion sweeps removing the symptoms.
