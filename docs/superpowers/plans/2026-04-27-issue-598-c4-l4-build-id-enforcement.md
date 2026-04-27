# Issue #598 — c4-l4-build ID Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `c4-l4-build` reject node/edge IDs outside `E<n>` / `R<n>` / `D<n>` namespaces (cross-checked against the L3 registry), and delete the now-redundant L4 mermaid audit checks since the JSON spec is the sole authoring surface.

**Architecture:** Validate `L4Spec.Diagram.Nodes[i].ID` and `L4Spec.Diagram.Edges[i].ID` directly from the JSON struct inside `c4L4Build` (no mermaid parsing). Reuse existing regexes (`mermaidIDPrefix`, `dEdgeIDPrefix`) and the existing registry-derivation pipeline (`scanRegistryDir` → `deriveRegistry`). Aggregate violations and report all in one error. Then delete `collectL4MermaidFindings`, `checkL4NodesInRegistry`, their tests, and the L4 branch in `auditFile` that calls them.

**Tech Stack:** Go, `targ` build system, existing dev/c4*.go infrastructure.

---

## File Structure

- `dev/c4_l4.go` — add `validateL4DiagramIDs(ctx, spec, inputPath) error`, call from `c4L4Build` after `loadAndValidateL4Spec`.
- `dev/c4_l4_test.go` (new) — TDD tests for each violation class, plus a happy-path regression.
- `dev/c4.go` — delete `collectL4MermaidFindings` (~80 lines, around 813–855), `checkL4NodesInRegistry` (~40 lines, 660–699), and the L4 branch in `auditFile` (lines 401–411) that called them.
- `dev/c4_test.go` — delete the audit tests asserting `node_id_missing` / `edge_id_missing` / `node_id_unknown` against rendered L4 `.md` fixtures.
- `dev/testdata/c4/` — delete any L4 audit fixtures that exclusively exercised the deleted checks (verify each before deleting).
- `skills/c4/references/mermaid-conventions.md` — minimal note pointing at the build gate (TDD via `superpowers:writing-skills`).
- `skills/c4/tests/` — pressure test added/updated by `writing-skills` flow.

---

### Task 1: Red — failing build-gate tests

**Files:**
- Test: `dev/c4_l4_test.go` (new)

- [ ] **Step 1: Inspect existing JSON spec shape**

Run: `ls architecture/c4/c4-recall.json && head -60 architecture/c4/c4-recall.json`
Expected: confirms `diagram.nodes[*].id`, `diagram.edges[*].id`, etc. — copy a minimal valid spec for fixtures.

- [ ] **Step 2: Write failing tests for each violation class**

Create `dev/c4_l4_test.go`. Use the existing test conventions in `dev/c4_test.go` (build tag `targ`, `_test.go` package, `t.Parallel()`, table-driven where it helps). The tests should call `c4L4Build` directly with a `C4L4BuildArgs{Input: <fixture path>}` against testdata fixtures placed in `dev/testdata/c4/l4/` — or, if simpler, marshal a `L4Spec` programmatically and pass through a temp file. Each test asserts the returned error mentions the offending ID and the rule.

```go
//go:build targ

package dev

import (
	"context"
	"strings"
	"testing"
)

func TestC4L4Build_RejectsLetterSuffixedEdgeID(t *testing.T) {
	t.Parallel()
	path := writeFixtureWithEdgeID(t, "R2a")
	err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path})
	if err == nil {
		t.Fatalf("want error for edge id R2a, got nil")
	}
	if !strings.Contains(err.Error(), "R2a") {
		t.Errorf("error should name offending id; got %v", err)
	}
}

func TestC4L4Build_RejectsWrongNamespaceEdgeID(t *testing.T) {
	t.Parallel()
	path := writeFixtureWithEdgeID(t, "X1")
	err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path})
	if err == nil || !strings.Contains(err.Error(), "X1") {
		t.Fatalf("want error naming X1; got %v", err)
	}
}

func TestC4L4Build_RejectsNonEPrefixedNodeID(t *testing.T) {
	t.Parallel()
	path := writeFixtureWithNodeID(t, "EXT1")
	err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path})
	if err == nil || !strings.Contains(err.Error(), "EXT1") {
		t.Fatalf("want error naming EXT1; got %v", err)
	}
}

func TestC4L4Build_RejectsNodeIDNotInRegistry(t *testing.T) {
	t.Parallel()
	// Place the L4 spec in a directory whose siblings define a registry that
	// does NOT include E99. writeFixtureInRegistryDir handles staging
	// c1/c2/c3 spec siblings derived from architecture/c4/.
	path := writeFixtureInRegistryDir(t, "E99")
	err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path})
	if err == nil || !strings.Contains(err.Error(), "E99") {
		t.Fatalf("want error naming unknown E99; got %v", err)
	}
}

func TestC4L4Build_AggregatesAllViolations(t *testing.T) {
	t.Parallel()
	path := writeFixtureWithMultipleViolations(t) // EXT1 node + R2a edge
	err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path})
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "EXT1") || !strings.Contains(err.Error(), "R2a") {
		t.Errorf("want aggregated error naming both EXT1 and R2a; got %v", err)
	}
}

func TestC4L4Build_AcceptsValidSpec(t *testing.T) {
	t.Parallel()
	path := writeValidFixture(t)
	if err := c4L4Build(context.Background(), C4L4BuildArgs{Input: path, NoConfirm: true}); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
}
```

Implement the `writeFixture*` helpers in the same file. Each helper writes a complete valid L4Spec JSON to a `t.TempDir()` and mutates one (or several) fields per the test's needs. For registry-dependent tests, also stage minimal `c1-foo.json` / `c2-foo.json` / `c3-foo.json` siblings in the same temp dir (look at `dev/c4_registry_test.go` for the pattern — reuse helpers there if available).

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test -- -run TestC4L4Build_ ./dev/`
Expected: tests FAIL — current `c4L4Build` does not validate diagram IDs, so `EXT1` / `R2a` / `X1` / `E99` all reach disk and the build returns nil.

---

### Task 2: Green — implement validateL4DiagramIDs

**Files:**
- Modify: `dev/c4_l4.go` (add function, wire into `c4L4Build`)

- [ ] **Step 1: Add validateL4DiagramIDs**

Add to `dev/c4_l4.go` (place near `validateL4Spec`):

```go
// validateL4DiagramIDs enforces L4 namespace discipline against the JSON spec:
//   - every node id matches E<n> and resolves to the L1-L3 registry derived
//     from sibling c{1,2,3}-*.json files in inputDir
//   - every edge id matches R<n> or D<n> (no letter suffix, no other prefix)
// All violations are collected and returned as one combined error so authors
// see the full list in one pass.
func validateL4DiagramIDs(ctx context.Context, spec *L4Spec, inputDir string) error {
	violations := []string{}

	// Edges: bare R<n> or D<n>.
	for index, edge := range spec.Diagram.Edges {
		if !dEdgeIDPrefix.MatchString(edge.ID) {
			violations = append(violations, fmt.Sprintf(
				"diagram.edges[%d].id %q: must match R<n> (call relationship) or D<n> (DI back-edge); "+
					"no letter suffixes — allocate a new sequential R<n> for related calls",
				index, edge.ID))
		}
	}

	// Nodes: shape check.
	badShape := map[string]bool{}
	for index, node := range spec.Diagram.Nodes {
		if !mermaidIDPrefix.MatchString(node.ID) {
			violations = append(violations, fmt.Sprintf(
				"diagram.nodes[%d].id %q: must match E<n>; if this represents an external system, "+
					"add it to the L3 registry first or describe it in the Dependency Manifest's "+
					`"Concrete adapter" column instead of inventing an L4-only id`,
				index, node.ID))
			badShape[node.ID] = true
		}
	}

	// Nodes: registry resolution. Only check shape-passing ids; skip the focus
	// id (already validated against E<n> by validateL4Spec and known-present).
	files, records, err := scanRegistryDir(ctx, inputDir)
	if err != nil {
		return fmt.Errorf("scan registry dir for L4 id check: %w", err)
	}
	if len(files) > 0 {
		view := deriveRegistry(inputDir, files, records)
		known := map[string]bool{}
		for _, element := range view.Elements {
			known[element.ID] = true
		}
		for index, node := range spec.Diagram.Nodes {
			if badShape[node.ID] || node.ID == spec.Focus.ID {
				continue
			}
			if !known[node.ID] {
				violations = append(violations, fmt.Sprintf(
					"diagram.nodes[%d].id %q: not in L1-L3 registry for parent %q — "+
						"add the element at L3 first, or describe it in the Dependency "+
						`Manifest's "Concrete adapter" column instead of inventing an L4-only id`,
					index, node.ID, spec.Parent))
			}
		}
	}

	if len(violations) == 0 {
		return nil
	}
	return fmt.Errorf("L4 id validation failed:\n  - %s", strings.Join(violations, "\n  - "))
}
```

- [ ] **Step 2: Wire into c4L4Build**

In `dev/c4_l4.go` `c4L4Build`, immediately after `loadAndValidateL4Spec` succeeds, add:

```go
	if err := validateL4DiagramIDs(ctx, spec, filepath.Dir(args.Input)); err != nil {
		return err
	}
```

- [ ] **Step 3: Run new tests**

Run: `targ test -- -run TestC4L4Build_ ./dev/`
Expected: all tests PASS.

- [ ] **Step 4: Run full dev tests for regressions**

Run: `targ test`
Expected: all green. If any audit tests now fail because the build pre-empts the audit-on-fabricated-ID path, note them — they will be deleted in Task 4.

- [ ] **Step 5: Sanity-check against real ledgers**

Run: `for f in architecture/c4/c4-*.json; do targ c4-l4-build --input "$f" --check >/dev/null || echo "FAIL: $f"; done`
Expected: no `FAIL` output — all 19 ledgers still build cleanly (they were swept clean in #597/#600/#601).

- [ ] **Step 6: Commit**

```bash
git add dev/c4_l4.go dev/c4_l4_test.go
git commit -m "$(cat <<'EOF'
feat(c4): enforce L4 node/edge id namespaces at build time (#598)

c4-l4-build now validates diagram ids from the JSON spec and rejects
fabricated ids before any output reaches disk: nodes must match E<n> and
resolve to the L1-L3 registry; edges must match bare R<n> or D<n> (no
letter suffixes, no other prefixes). Violations are aggregated into one
error so authors see the whole list in one pass.

AI-Used: [claude]
EOF
)"
```

---

### Task 3: Delete redundant L4 audit checks

**Files:**
- Modify: `dev/c4.go` (delete `collectL4MermaidFindings`, `checkL4NodesInRegistry`, the L4 branch in `auditFile`)
- Modify: `dev/c4_test.go` (delete tests of the deleted functions)
- Modify: `dev/testdata/c4/` (delete fixtures that only exercised the deleted checks)

- [ ] **Step 1: Identify the L4-only audit branch**

Re-read `dev/c4.go:401-411` (`auditFile` L4 branch). The block to delete:

```go
	if matter.level == 4 {
		block, mermaidFindings := loadMermaidBlock(raw, path)
		findings = append(findings, mermaidFindings...)
		if block != nil {
			findings = append(findings, collectL4MermaidFindings(block)...)
			findings = append(findings, checkL4NodesInRegistry(ctx, block, path)...)
		}
		return findings, nil
	}
```

The L4 path will remain — front-matter, code-pointer, and property-link checks at lines 396–400 still apply. After deletion, an L4 file simply doesn't enter the L1-L3 mermaid/catalog/relationship branch (the `level == 4` early-return will be replaced by an explicit "skip L1-L3 structural checks for L4" guard).

- [ ] **Step 2: Replace the L4 branch with a skip-guard**

In `dev/c4.go` `auditFile`, replace the L4 block with:

```go
	if matter.level == 4 {
		// L4 ledgers use a different schema (no Element Catalog, no JSON
		// registry cross-check at audit time, SVG-rendered with no click
		// handlers). Diagram-id discipline is enforced by c4-l4-build at
		// generation time (#598), so audit only runs front-matter and
		// code-pointer checks for L4.
		return findings, nil
	}
```

- [ ] **Step 3: Delete `collectL4MermaidFindings` and `checkL4NodesInRegistry`**

In `dev/c4.go`, remove the functions entirely:
- `collectL4MermaidFindings` (around lines 813–855)
- `checkL4NodesInRegistry` (around lines 660–699)

If `loadMermaidBlock` becomes unreferenced after the L4 branch is gone, delete it too (verify with `grep -n loadMermaidBlock dev/`). If still used elsewhere, keep.

- [ ] **Step 4: Delete the corresponding tests in `dev/c4_test.go`**

Find and delete any tests that exclusively exercised the L4 audit path. Use:

```bash
grep -n "collectL4MermaidFindings\|checkL4NodesInRegistry\|node_id_missing\|edge_id_missing\|node_id_unknown" dev/c4_test.go
```

Delete each matching test that is L4-specific (the L1-L3 audit also emits `node_id_missing` / `edge_id_missing`; only delete tests asserting those for L4 fixtures or asserting `node_id_unknown` at all). Per the issue's evidence search results, this includes the test at `dev/c4_test.go:540-545` asserting fabricated EXT1/X1 detection in audit output, and `dev/c4_test.go:314` if its `wantIDs` list belongs to an L4 audit test.

- [ ] **Step 5: Delete unreferenced L4 audit fixtures**

Run:

```bash
ls dev/testdata/c4/ | grep -i l4
```

For each, check whether anything still references it after the test deletions:

```bash
grep -rn "<fixture-name>" dev/
```

Delete fixtures with no remaining references.

- [ ] **Step 6: Run all dev tests**

Run: `targ test`
Expected: all green. No reference to deleted symbols.

- [ ] **Step 7: Run check-full**

Run: `targ check-full`
Expected: clean — no unused-import warnings (regexp, etc., may still be used by L1-L3 checks; verify).

- [ ] **Step 8: Run audit across all ledgers**

Run: `targ c4-audit`
Expected: clean output. The deleted L4 checks should not produce missing-coverage findings — L4 ids are now enforced at build time.

- [ ] **Step 9: Commit**

```bash
git add dev/c4.go dev/c4_test.go dev/testdata/c4/
git commit -m "$(cat <<'EOF'
refactor(c4): delete redundant L4 audit id checks (#598)

c4-l4-build now enforces L4 node/edge id discipline at generation time, so
the post-hoc mermaid audit checks (collectL4MermaidFindings,
checkL4NodesInRegistry) and their tests are dead weight. JSON specs are
the sole authoring surface for L4 ledgers, so there is no scenario where
hand-edited rendered output bypasses the build gate.

AI-Used: [claude]
EOF
)"
```

---

### Task 4: Skill update via writing-skills (TDD)

**Files:**
- Modify: `skills/c4/references/mermaid-conventions.md`
- Modify or create: `skills/c4/tests/` (driven by writing-skills flow)

- [ ] **Step 1: Invoke superpowers:writing-skills**

Per CLAUDE.md: ALWAYS use `superpowers:writing-skills` when editing any SKILL.md or skill reference file. Invoke it now and follow its TDD flow:

1. **Baseline test (RED):** add a behavioral pressure test in `skills/c4/tests/` that simulates an L4 authoring scenario where the agent might invent an `EXT1`-style id or letter-suffix an R-number. Verify the current skill prose does not prevent the failure mode.
2. **Edit (GREEN):** make the *minimal* change to `skills/c4/references/mermaid-conventions.md` — one sentence noting that `c4-l4-build` rejects ids outside `E<n>` / `R<n>` / `D<n>` and instructing authors to read the build error for the fix. Delete any prose now subsumed by the build error message.
3. **Verify (GREEN):** re-run the pressure test and confirm it passes.

Do NOT add a separate "Authoring discipline" section to SKILL.md unless the pressure test reveals the conventions reference is insufficient. Do NOT add inline reminders to templates.

- [ ] **Step 2: Run skill tests**

Run the skill test suite per `skills/c4/tests/` README (or however writing-skills directs).
Expected: all pressure tests pass.

- [ ] **Step 3: Commit**

```bash
git add skills/c4/references/mermaid-conventions.md skills/c4/tests/
git commit -m "$(cat <<'EOF'
docs(c4): point to c4-l4-build as the L4 id-namespace enforcer (#598)

Build-time rejection (validateL4DiagramIDs) now catches fabricated ids
with actionable error messages, so the conventions reference can shrink
to a single pointer at the build gate. Pressure-tested via the
superpowers:writing-skills flow.

AI-Used: [claude]
EOF
)"
```

---

### Task 5: Final validation and issue closure

**Files:**
- None modified.

- [ ] **Step 1: Full check**

Run: `targ check-full`
Expected: clean.

- [ ] **Step 2: Build all 19 ledgers in --check mode**

Run: `for f in architecture/c4/c4-*.json; do targ c4-l4-build --input "$f" --check; done`
Expected: every ledger emits identical bytes — the swept-clean ledgers (post-#597/#600/#601) all pass the new validation.

- [ ] **Step 3: Audit all ledgers**

Run: `targ c4-audit`
Expected: clean output across all L1–L4 ledgers.

- [ ] **Step 4: Push (after review per project workflow)**

Per CLAUDE.md: review before merge, ff-only, rebase on main. Push only after reviewer ACKs.

- [ ] **Step 5: Close #598**

```bash
gh issue close 598 --comment "Closed by <commit-hash-of-task-2> + <commit-hash-of-task-3> + <commit-hash-of-task-4>: c4-l4-build now validates diagram ids at generation time (rejecting EXT1/EJQ/R2a/Rjq/X1 patterns with actionable errors), and the redundant L4 audit checks were deleted since JSON is the sole authoring surface."
```

---

## Self-Review

**Spec coverage:**
- Build gate: Task 2 (validateL4DiagramIDs).
- Aggregated errors: Task 2 step 1 (violations slice + joined error).
- Registry resolution: Task 2 step 1 (scanRegistryDir + deriveRegistry reuse).
- Deletions: Task 3.
- Skill update: Task 4 (via writing-skills).
- Validation: Task 5.
- Out-of-scope items (R-number cross-check, template reminder, L1-L3 audit changes): not included. ✓

**Placeholder scan:** No TBDs, no "handle errors", no "similar to". Each step has exact code or commands.

**Type consistency:** `validateL4DiagramIDs(ctx, spec, inputDir)` signature matches its call site in Task 2 step 2. `c4L4Build` already has `ctx` in scope. `mermaidIDPrefix`, `dEdgeIDPrefix`, `scanRegistryDir`, `deriveRegistry` are existing exported-or-package symbols (verified via grep). The L4 audit branch lines (401–411) and helper line ranges are best-effort approximations — Task 3 step 1 instructs re-reading them at execution time.

**Open uncertainty:** Task 1 fixture helpers require a small amount of design work — the executor should look at `dev/c4_registry_test.go` for any reusable staging helpers before writing from scratch. This is bounded scope, not a placeholder.
