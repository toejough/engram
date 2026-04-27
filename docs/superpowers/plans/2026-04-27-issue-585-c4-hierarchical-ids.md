# C4 Hierarchical Per-Owner IDs Implementation Plan (revised)

> **Revised 2026-04-27** after realizing the original 14-task structure preserved the per-level builder bespoke-ness that the new ID scheme renders unnecessary. Each spec now carries its own coordinates (`level` + `focus.id`), so all four builders collapse to one shared validator + thin per-level renderers.

**Goal:** Replace flat `E<n>` IDs in `architecture/c4/` with hierarchical `S/N/M/P` paths and *delete* the per-level bespoke ID logic.

**Spec:** `docs/superpowers/specs/2026-04-27-issue-585-c4-hierarchical-ids-design.md`

---

## Done so far

- ✅ **Task 1** (`37eb3cdb`): `dev/c4_idpath.go` — `IDPath` parser/validator with tests.
- ✅ **Task 2** (`ab720ec1`): `dev/c4_migrate.go` — throwaway one-shot migration target.
- ✅ **Task 3** (`653b5e77`): all `architecture/c4/*.json` migrated to hierarchical IDs.
- ✅ **Task 4** (`5ccba3ab`): L1 builder reads `S<n>` IDs and emits `s<n>-<slug>` anchors.

## Remaining

### Task 5R — Shared ID helpers

**File:** extend `dev/c4_idpath.go` (and tests).

Add the primitives every builder will share:

```go
// LocalLetter returns the letter that this spec's level allocates.
//   1 → "S", 2 → "N", 3 → "M", 4 → "P".
func LocalLetter(level int) (string, error)

// Anchor returns the canonical markdown anchor for an element.
//   Anchor("S2-N3-M5", "Recall") => "s2-n3-m5-recall".
func Anchor(id, name string) string

// ValidateElementID returns nil iff `id` is acceptable for an element on a
// spec at `level` whose focus path is `focus`. The valid shapes are:
//   - identical to focus (the focus element itself)
//   - any strict ancestor of focus (carried-over inherited element)
//   - focus + LocalLetter(level) + <positive int> (a new local element)
// L1 has Level==1 and an empty focus IDPath; level-1 element IDs are S<n>.
func ValidateElementID(level int, focus IDPath, id string) error

// NextLocalID returns the next sequential local ID at this level.
//   NextLocalID(3, "S2-N3", []string{"S2-N3-M1","S2-N3-M3"}) => "S2-N3-M4".
// (Used by tooling that allocates new IDs; not currently used by builders
// since IDs are read from JSON, but unifies the rule.)
func NextLocalID(level int, focus IDPath, existing []string) (string, error)
```

Tests cover each rule with a small table.

### Task 6R — Collapse builders + delete registry

Refactor `dev/c4_l2.go`, `dev/c4_l3.go`, `dev/c4_l4.go` to:

- Drop `FromParent` from every struct (already gone from JSON).
- Replace per-level `assign{L2,L3}ElementIDs` and `validateL{2,3,4}ElementIDs` with a single inline call to `ValidateElementID(spec.Level, focus, element.ID)`.
- L3: delete `checkFocusAgainstRegistry`, `checkL3ElementAgainstRegistry`, `discoverL3Siblings`'s registry coupling, and any other `c4_registry`-dependent code paths.
- L4: delete the `scanRegistryDir` / parent-load step. Validation becomes syntactic via `ValidateElementID` plus a same-parent-as-focus check for siblings (focus depth + 1 not allowed; same depth as focus with shared parent allowed).
- All four builders use `Anchor(id, name)` for markdown anchors and direct `id` strings for mermaid node IDs.

Delete:
- `dev/c4_registry.go`
- `dev/c4_registry_test.go`

Update tests: collapse `*FromParent*` and `*Registry*` tests into the shared validator's table tests; remove fixture files that exist only to test the old shape.

Regenerate all markdown:
```bash
for spec in architecture/c4/c2-*.json; do targ c4-l2-build --input "$spec" --noconfirm; done
for spec in architecture/c4/c3-*.json; do targ c4-l3-build --input "$spec" --noconfirm; done
for spec in architecture/c4/c4-*.json; do targ c4-l4-build --input "$spec" --noconfirm; done
```

Net code delta: expect **−800 to −1200 LOC** (registry deleted, per-level validators collapsed, FromParent paths removed).

### Task 7R — Update audit

`dev/c4_audit_ext.go`: drop cross-file namespace-collision findings (their contract was the deleted registry); switch any flat-`E<n>` mermaid checks to use `ParseIDPath`.

### Task 9 — Re-render SVGs

`targ c4-render`. Commit regenerated `.svg` files.

### Task 11 — Delete migration tool

`rm dev/c4_migrate.go`.

### Task 12 — Update SKILL.md

Drop registry coordination guidance; document hierarchical scheme in one short paragraph.

### Task 13 — Update reference docs

`mermaid-conventions.md`, `property-ledger-format.md`, `templates/*.md`: replace E-ID examples with hierarchical examples.

### Task 14 — End-to-end verification

`targ check-full`; audit every C4 markdown; verify spec acceptance criteria.

---

## Why this is shorter than the original plan

The original plan kept four builders' bespoke logic and migrated each in turn. Once IDs are hierarchical and self-validating, the cross-file registry, the per-level ID assignment, and the `from_parent` boolean all become unnecessary. One validator + one anchor function + one renderer-per-level is all that's left. Most of the work in tasks 5–8 is *deletion*, not transformation.
