# Pressure Test 2: Cross-Layer Update Propagation

Captured: 2026-04-25
Subagent: general-purpose (Agent tool, fresh dispatch from controller)
Tool uses by subagent: 4
Fixture: `/tmp/c4-fixture/architecture/c4/` with c1, c2, and c3 files referencing a container
named `oldname`.

## Prompt

The subagent was asked to follow the c4 skill's `update` workflow for:

> /c4 update test-containers — rename the container 'oldname' to 'newname'.

It was told to NOT actually edit files, but show every diff it would propose.

## Verdict: PASS

The subagent proposed **all three required diffs**:

1. **L2 target** (`c2-test-containers.md`) — updates frontmatter children, mermaid node, catalog row, and refined-by cross-link.
2. **L1 parent** (`c1-test-system.md`) — updates mermaid edge target and catalog row referencing `oldname`. Presented as separate apply/skip/defer choice with one-line reason.
3. **L3 child** (`c3-oldname.md`) — proposes both file rename (`c3-oldname.md` → `c3-newname.md`) AND content updates (frontmatter `name`, body heading, focus callout). Presented as separate apply/skip/defer choice with one-line reason.

**Bonus quality** — the subagent identified a coupling failure mode: if the L3 rename is
deferred but the L2 children-field update is applied, L2's `children: ["c3-newname.md"]`
becomes a broken pointer. It surfaced this coupling explicitly for user awareness, and
noted that deferred items would persist as drift notes per skill rules.

No FAIL conditions observed. Specifically:
- Did NOT silently edit non-target files.
- Did NOT batch all changes into one combined approval.
- Did NOT skip the L1 parent or L3 child propagation.

## Captured Output

(Verbatim subagent response — abbreviated formatting; full diffs preserved.)

---

## 1. Target-layer diff (L2: c2-test-containers.md)

```diff
--- a/architecture/c4/c2-test-containers.md
+++ b/architecture/c4/c2-test-containers.md
-children: ["c3-oldname.md"]
+children: ["c3-newname.md"]
-    oldname[oldname container]
+    newname[newname container]
-| oldname | Container | Does the thing | (fixture) |
+| newname | Container | Does the thing | (fixture) |
-- Refined by: c3-oldname.md
+- Refined by: c3-newname.md
```

**Reason:** User-requested rename of the `oldname` container to `newname` at L2.

---

## 2. Propagation analysis

Renamed element at L2 → must check L1 parent (references `oldname` in mermaid + catalog) and
L3 child (entire file is the refinement of the renamed container).

---

## 3. Per-file propagation proposals

### Proposal A — L1 parent: c1-test-system.md

```diff
-    user --> oldname
+    user --> newname
-| oldname | Container | Does the thing | (fixture) |
+| newname | Container | Does the thing | (fixture) |
```

**Reason:** L1 catalog and mermaid reference the renamed container.
**Presented:** "Propagation 1/2 — apply / skip / defer?"

### Proposal B — L3 child: rename file + update contents

```diff
-name: oldname
+name: newname
-# C3 — oldname (Component)
+# C3 — newname (Component)
-> Container in focus: oldname
+> Container in focus: newname
```

Plus file rename: `c3-oldname.md` → `c3-newname.md`.

**Reason:** L3 file refines the renamed container; filename + frontmatter + body all need updating.
**Presented:** "Propagation 2/2 — apply / skip / defer?"

---

## Coupling note surfaced by the subagent

If user defers Proposal B but applies Proposal A and the target diff, L2's
`children: ["c3-newname.md"]` becomes a broken pointer. Subagent flagged this coupling
explicitly so the user can decide knowingly (and recommended drift-note recording per
skill rules if the defer is chosen anyway).
