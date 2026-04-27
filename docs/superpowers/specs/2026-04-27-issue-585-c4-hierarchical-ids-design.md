# Hierarchical Per-Owner C4 IDs

**Issue:** [#585](https://github.com/toejough/engram/issues/585)
**Date:** 2026-04-27

## Problem

C4 element IDs (`E1`, `E2`, …) are flat across all diagrams in `architecture/c4/`. The
`c4-registry` build target exists *because* the namespace is shared — adding or
removing a component anywhere shifts the namespace, and contributors coordinate
through it.

Cross-file references are also opaque about provenance: a link like
`[E27 · tokenresolver](c3-engram-cli-binary.md#e27-tokenresolver)` requires the
reader to know which c3 file owns `E27`.

## Goal

Each diagram owns the IDs it allocates. Provenance is encoded in the ID itself.
`c4-registry`'s coordination role disappears.

## Design

### Naming scheme

| Doc level | Owner letter | Meaning |
|---|---|---|
| L1 (`c1-*.json`) | `S<n>` | **S**ystem (and L1 externals/people) |
| L2 (`c2-*.json`) | `N<n>` | co**N**tainer |
| L3 (`c3-*.json`) | `M<n>` | co**M**ponent |
| L4 (`c4-*.json`) | `P<n>` | **P**roperty |

Per-diagram edges keep flat numbering:

| Edge kind | Pattern | Scope |
|---|---|---|
| Runtime call | `R<n>` | Per-diagram, sequential |
| DI back-edge | `D<n>` | Per-diagram, sequential |

### Path form

Hyphen-separated, no dots: `S2-N3-M5-P1`. One form everywhere — JSON keys, prose,
markdown anchors, mermaid node IDs. No render-time translation.

Markdown anchors are lowercased: `#s2-n3-m5-recall`.

### Cross-doc references

When a doc references an element it owns, it uses its own ID with the level's
letter prefix (e.g. an L3 doc allocates `M1`, `M2`, … for its components).

When a doc references an element owned by an ancestor, it uses that element's
**full hierarchical path** verbatim. Example: an L4 spec for `recall` (which is
component 5 of L3 doc whose focus is container 3 of system 2) references its
sibling component `cli` as `S2-N3-M1`, and references the external `Anthropic API`
as `S5`.

Lower-level docs **consult their parent doc** to discover the path of any
ancestor-owned element they reference. They never re-allocate or re-letter.

### Schema changes

#### `focus.id`

Becomes the full hierarchical path of the element this doc refines. L1 has no
`focus` block (it is the system context itself) and is exempt.

| Doc | Today | After |
|---|---|---|
| `c1-engram-system.json` | (no focus) | (no focus) |
| `c2-engram-plugin.json` | `"id": "E2"` (or absent) | `"id": "S2"` |
| `c3-engram-cli-binary.json` | `"id": "E9"` | `"id": "S2-N3"` |
| `c4-recall.json` | `"id": "E22"` | `"id": "S2-N3-M5"` |

#### Element IDs

Each element has `"id": "<full-path>"`. New elements use the doc's prefix +
local letter+number; carried-over elements use their ancestor path verbatim.

#### `from_parent` field

**Removed.** Provenance is derived at render/audit time by comparing an element's
ID-path depth to the doc's level. An element whose ID has fewer path segments
than the doc is necessarily inherited.

#### Edges

Unchanged in form (`R<n>`, `D<n>`). `from`/`to` reference element IDs by full
path (e.g. `"from": "S2-N3-M5", "to": "S2-N3-M9"`).

Today, L1/L2/L3 edges use element **names** (`"from": "Developer"`) while L4
uses IDs. Migration normalizes all levels to ID-based references. This is part
of "simplify" — one convention, no per-level rules.

### Validation

The audit/build pipeline replaces global-namespace registry checks with one
syntactic rule per spec:

> Every element ID is either (a) a strict prefix-form path of an ancestor — for
> a doc at level *k*, this means depth < *k*, with the right letters in order —
> or (b) the doc's own focus path + a new local `<letter><n>`.

The doc's own letter is determined by its level (L1→S, L2→N, L3→M, L4→P).
Local numbers must be sequential starting at 1, no gaps, within the set of
elements the doc owns.

Edge IDs continue to be validated as `^R\d+$` / `^D\d+$`, sequential per
diagram, no gaps. The `from`/`to` of every edge must be either the focus path or
an element ID that appears in the spec.

The L4 build target's "load parent L3 registry" step is removed. Cross-doc
consistency (does the parent actually define this ID?) becomes a separate audit
pass that walks the doc tree and verifies references resolve, but it is not
required for a single doc's syntactic validation.

### Mermaid

Node IDs use full hierarchical form (`S2-N3-M5`). Hyphens are valid mermaid
identifier characters; no translation. Edge IDs remain flat (`R1`, `D1`).

If the rendered SVG ends up too verbose with full paths in node labels, we
trim later — out of scope for this design.

## Codebase simplifications

| File | Today | After |
|---|---|---|
| `dev/c4_registry.go` (489 LOC) | Global E-namespace uniqueness across all files; coordination point | Deleted, or shrunk to a thin path-uniqueness sanity check (~30 LOC) at render time |
| `dev/c4_l4.go` (613 LOC) | Loads parent L3 registry, validates E-IDs against it | Drops parent-load; syntactic prefix check |
| `dev/c4_audit_ext.go` | Cross-file E-collision findings | Namespace-collision findings removed |
| Builders (`c4_l1.go` etc.) | Allocate from shared global E-counter | Each owns its own per-letter, per-doc counter |
| `skills/c4/SKILL.md` | Documents global-E coordination, registry target | Removes registry coordination guidance, documents path scheme |

Estimated net deletion: 400-600 lines of Go and 60-100 lines of skill markdown.

## Migration

A one-time mechanical migration (no LLM regen):

1. **Implement new schema + validators** alongside existing ones (no break).
2. **Migration tool** at `dev/c4_migrate.go` (throwaway, deleted in same PR):
   1. Walk JSON files in dependency order: L1 → L2 → each L3 → each L4.
   2. Assign hierarchical IDs deterministically (by JSON element order).
   3. Build a flat-`E<n>` → hierarchical-path map.
   4. Rewrite all JSON files using the map (focus.id, element ids, edge from/to).
3. **Re-render** all markdown via the existing `c4-render` target.
4. **Switch builders/audits** to the new validators.
5. **Delete** old registry code, `from_parent` plumbing, and the migration tool.
6. **Update** `skills/c4/SKILL.md` and `skills/c4/references/*.md`.

LLM regen-from-scratch (issue #586) is **not** the migration path here — it
would burn tokens re-deriving identical prose. Issue #586 is a separate
end-to-end validation; this issue is a mechanical rename.

## Out of scope

- Stable-ID / rename-tracking layer. Renaming or removing an element still
  shifts later siblings' numbers. Per "don't optimize too early."
- SVG verbosity trimming. Full paths in node labels for now.
- Issue #586's end-to-end regen validation. Separate work.

## Acceptance criteria

1. All `architecture/c4/*.json` files use hierarchical IDs; no `E<n>` remains.
2. `targ c4-audit` reports zero findings.
3. `targ c4-l1-build`, `c4-l2-build`, `c4-l3-build`, `c4-l4-build` all pass.
4. `targ check-full` clean.
5. `dev/c4_registry.go` is deleted or ≤50 LOC.
6. `from_parent` field is gone from all JSON specs and from Go code.
7. `skills/c4/SKILL.md` no longer references the global E-namespace or
   `c4-registry` coordination.
8. Migration tool deleted.
9. Markdown anchors in cross-file links resolve (no broken anchors).

## Related

- #586 — end-to-end regen validation (separate)
- #598 — L4 build-time ID enforcement (already merged; this issue rewrites
  the IDs that #598 validates)
