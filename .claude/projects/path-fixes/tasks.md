# Tasks: path-fixes

## TASK-001: Fix checker.go paths

Remove `"docs"` segment from artifact path lookups in precondition checker.

**Files:** `cmd/projctl/checker.go`

**Changes:**
- `RequirementsExist`: `filepath.Join(dir, "requirements.md")`
- `RequirementsHaveIDs`: `filepath.Join(dir, "requirements.md")`
- `DesignExists`: `filepath.Join(dir, "design.md")`
- `DesignHasIDs`: `filepath.Join(dir, "design.md")`

**Acceptance Criteria:**
- [x] Tests verify checker finds artifacts at project root
- [x] `projctl state transition --to pm-complete` works with root-level requirements.md

**Traces to:** REQ-001, DES-001

---

## TASK-002: Fix task package paths

Remove `"docs"` segment from tasks.md path lookups.

**Files:** `internal/task/deps.go`, `internal/task/validate.go`

**Changes:**
- `deps.go:21`: `filepath.Join(dir, "tasks.md")`
- `validate.go:31`: `filepath.Join(dir, "tasks.md")`
- `validate.go:120`: `filepath.Join(dir, "tasks.md")`

**Acceptance Criteria:**
- [x] Tests verify task functions find tasks.md at project root
- [x] `projctl task` commands work with root-level tasks.md

**Traces to:** REQ-002, DES-001

---

## TASK-003: Fix trace package paths

Remove `"docs"` segment from artifact path lookups in trace validation.

**Files:** `internal/trace/trace.go`

**Changes:**
- Line 405: Change `docsDir` to use `dir` directly or remove `"docs"` segment
- Line 689: Same
- Line 898: Same

**Acceptance Criteria:**
- [x] Tests verify trace validation finds artifacts at project root
- [x] `projctl trace validate --dir .claude/projects/foo/` works

**Traces to:** REQ-003, DES-001

---

## TASK-004: Fix escalation.go paths

Remove `"docs"` segment from escalations.md path lookups.

**Files:** `cmd/projctl/escalation.go`

**Changes:**
- Line 49: `filepath.Join(dir, "escalations.md")`
- Line 106: Same
- Line 181: Same
- Line 223: Same

**Acceptance Criteria:**
- [x] Tests verify escalation commands find escalations.md at project root
- [x] `projctl escalation` commands work with root-level escalations.md

**Traces to:** REQ-004, DES-001

---

## TASK-005: Document project layout

Add canonical project layout documentation to skill docs.

**Files:** `skills/project/SKILL-full.md`

**Changes:**
- Add "Project Layout" section showing expected directory structure

**Acceptance Criteria:**
- [x] SKILL-full.md documents that artifacts live at project root
- [x] Example layout shows all standard artifact files

**Traces to:** REQ-005, DES-003

**Blocked by:** TASK-001, TASK-002, TASK-003, TASK-004 (document after code is fixed)

---

## Dependency Graph

```
TASK-001 ─┐
TASK-002 ─┼─► TASK-005
TASK-003 ─┤
TASK-004 ─┘
```

Tasks 1-4 can be done in parallel. Task 5 depends on all of them.
