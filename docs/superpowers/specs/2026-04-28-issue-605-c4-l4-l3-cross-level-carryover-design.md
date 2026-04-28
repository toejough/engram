# Issue #605 — c4 L4↔L3 cross-level carryover validation

## Background

#604 landed the two-diagram L4 schema with `validateL4Manifest` enforcing that every `WrappedEntityID` in the dependency manifest matches some node on the L4 diagram. That check is intra-spec only.

#605 adds the cross-level invariant: an L4 spec is a refinement of one L3 component (`focus.id`), and its node set must agree with the L3 parent's view of that component's relationships. Two directions:

- **L4→L3:** every node on the L4 diagram exists on the L3 parent with the same `id` and `kind`.
- **L3→L4:** every L3 node connected (one hop, either edge direction) to the focus's L3 node appears on the L4 diagram.

Equal severity. Both block at build time and emit standard findings at audit time.

## Scope

- L4 builder (`dev/c4_l4.go`): extend validation to load the L3 parent and run cross-level checks.
- Audit (`dev/c4.go` or `dev/c4_audit_*.go`): same validator called over JSON, emitting findings.
- Data fixup: bring all 19 L4 specs into compliance, one commit per spec.
- Out of scope: skill rewrite (#606), L1/L2 cross-level rules, finding-format changes.

## Vocabulary

- **L3 parent JSON.** `architecture/c4/<basename>.json` where `<basename>` is `l4.Parent` with `.md`→`.json`.
- **Focus node.** The L3 element whose `id == l4.Focus.ID`. Required to exist on L3.
- **Connected (one hop).** Given L3 relationships `[{from, to, ...}]`, a node `n` is connected to `focusID` iff there exists a relationship where `from == focusID && to == n.id`, or `from == n.id && to == focusID`. Direction is ignored. Self-loops do not produce neighbors.
- **L4 node set.** `l4.Diagram.Nodes`, including the focus's own node.

## Validator

### Signature

```go
// validateL4Carryover enforces the L4↔L3 cross-level invariant. Both
// directions are checked; all violations are returned as a joined error.
func validateL4Carryover(l4 *L4Spec, l3 *L3Spec) error
```

### Algorithm

1. **Build L3 element index:** `l3ByID := map[string]L3Element` keyed by `Element.ID`.
2. **Verify focus exists on L3.** If `l3ByID[l4.Focus.ID]` is missing, fail immediately with `focus_id %q: not present on L3 parent %q`.
3. **L4→L3 check.** For each `node` in `l4.Diagram.Nodes`:
   - If `l3ByID[node.ID]` is missing → error: `diagram.nodes[%d] %q: not present on L3 parent %q`.
   - Else if `l3ByID[node.ID].Kind != node.Kind` → error: `diagram.nodes[%d] %q: kind %q does not match L3 parent kind %q`.
4. **L3→L4 check.** Compute the connected set on L3:
   ```go
   connected := map[string]bool{}
   for _, rel := range l3.Relationships {
       switch {
       case rel.From == l4.Focus.ID && rel.To != l4.Focus.ID:
           connected[rel.To] = true
       case rel.To == l4.Focus.ID && rel.From != l4.Focus.ID:
           connected[rel.From] = true
       }
   }
   ```
   Build `l4Nodes := map[string]bool` from `l4.Diagram.Nodes`. For each `id` in `connected`, if `!l4Nodes[id]` → error: `L3 parent %q has node %q connected to focus %q, but %q is missing from L4 diagram.nodes`.
5. **Return joined error.** Use `errors.Join` so call sites can split per violation when emitting findings.

### Loader helper

```go
// loadL3Parent reads the L3 spec sibling of an L4 spec. dirPath is the
// directory both files live in (architecture/c4/ in production).
func loadL3Parent(l4 *L4Spec, dirPath string) (*L3Spec, error)
```

- Derive `parentJSON := strings.TrimSuffix(l4.Parent, ".md") + ".json"`.
- Open `filepath.Join(dirPath, parentJSON)`; decode into `L3Spec` with `DisallowUnknownFields`.
- On missing file or decode error, return wrapped error (`fmt.Errorf("loading L3 parent %q: %w", parentJSON, err)`).

## Build-time wiring

`validateL4Spec` extended to take the L3 parent. New signature:

```go
func validateL4Spec(spec *L4Spec, l3 *L3Spec) error
```

`c4L4Build` (or its delegate) loads the L3 parent before calling `validateL4Spec` and aborts the build if loading fails. The cross-level check runs after the existing intra-spec validations so callers see schema errors first when both apply.

## Audit-time wiring

In `dev/c4.go` (or a new `dev/c4_audit_l4_carryover.go`):

- For each L4 markdown file in scope, locate its sibling L4 JSON and parent L3 JSON in the same directory.
- Decode both. Call `validateL4Carryover(l4, l3)`.
- Walk the joined error (`errors.Unwrap` / type assertion to `interface{ Unwrap() []error }`) and emit one finding per leaf error using the existing finding struct:
  - `ID: "l4_carryover"`
  - `Severity:` whatever `l4_external_missing` and other L4 findings already use (no new severity tier).
  - `Detail:` the leaf error string verbatim.

Audit makes no attempt to read rendered markdown for L3 — JSON is the source of truth (per Q4 decision).

## Test plan

`dev/c4_l4_test.go`:

- `TestValidateL4Carryover_HappyPath` — two minimal in-test specs: L3 has focus `F` plus neighbor `N`; L4 has both. Pass.
- `TestValidateL4Carryover_FocusMissingFromL3` — L3 has no `F`; expect focus-id error.
- `TestValidateL4Carryover_L4HasExtraNode` — L4 introduces `X` not on L3. Expect L4→L3 error citing `X`.
- `TestValidateL4Carryover_KindMismatch` — same `id` on both sides, different `kind`. Expect kind error.
- `TestValidateL4Carryover_MissingNeighbor_Outbound` — L3 has `F → M`, L4 omits `M`. Expect L3→L4 error citing `M`.
- `TestValidateL4Carryover_MissingNeighbor_Inbound` — L3 has `C → F`, L4 omits `C`. Expect L3→L4 error citing `C`.
- `TestValidateL4Carryover_SelfLoopIgnored` — L3 has `F → F`. Connected set empty; no errors from that edge.
- `TestValidateL4Carryover_AggregatesViolations` — multiple violations of both kinds; verify all surface in the joined error.

`dev/c4_test.go`:

- `TestAudit_L4CarryoverEmitsOneFindingPerViolation` — fixture L4 + L3 with two known violations; audit produces exactly two `l4_carryover` findings.

`dev/c4_l4_test.go` already-existing fixtures (`validL4Spec()`) updated to take an L3 parent helper so the existing tests continue to compile.

## Data fixup

After the validator lands and tests pass, run:

```bash
targ c4-l4-build --noconfirm
```

across all 19 L4 specs (one at a time, since the build target processes a single input). For each violation:

- **L4→L3 extra:** if the relationship at the container level is real, add the node + relationship to L3 (`c3-*.json`) and rebuild the L3 markdown. If the relationship is not real at L3, remove the node from L4.
- **L4→L3 kind mismatch:** trace which side is correct from the source code; correct the wrong one.
- **L3→L4 missing neighbor:** add the node to L4 plus an `R<n>` edge mirroring the L3 relationship's direction and description.

Each L4 spec's fix-up is one commit. Do the L3-side edits in their own commit when needed, ahead of the L4 fix.

## Out of scope

- Audit-finding ID taxonomy beyond `l4_carryover` (single ID covers both directions; the `Detail` string distinguishes).
- Changes to L1/L2 rules.
- Skill text (#606 will reference the new rule once the validator is in place).

## Acceptance

- `targ check-full` and `targ test` pass.
- `validateL4Carryover` enforced at both build and audit time.
- All 19 L4 specs pass the new check; `targ c4-l4-build` and `targ c4-audit` clean tree-wide.
- No new finding-severity tier; reuses existing severity.
- L3 parent JSON loaded once per L4 build; failure is fatal.
