# Tasks: Issue Traceability

## Overview

7 tasks to add ISSUE- prefix support to projctl trace commands. This is actual code implementation with TDD.

## Tasks

### TASK-001: Update V1 ID Pattern in trace.go
**Priority:** P0
**Links:** REQ-001, REQ-002, ARCH-004
**Depends on:** None

Update the `idPattern` regex in `internal/trace/trace.go` to accept ISSUE- prefix.

**Acceptance Criteria:**
- [ ] `idPattern` regex updated to include ISSUE
- [ ] `ValidID("ISSUE-1")` returns true
- [ ] Existing REQ/DES/ARCH/TASK IDs still validate

**Files:**
- `internal/trace/trace.go` (line ~18)
- `internal/trace/trace_test.go` (add tests)

---

### TASK-002: Add ISSUE→REQ Validation in Add()
**Priority:** P0
**Links:** REQ-001, REQ-004, DES-001, ARCH-004
**Depends on:** TASK-001

Add validation in `trace.Add()` that ISSUE can only link to REQ.

**Acceptance Criteria:**
- [ ] `trace.Add("ISSUE-1", "REQ-001")` succeeds
- [ ] `trace.Add("ISSUE-1", "DES-001")` returns error
- [ ] Error message: "ISSUE can only link to REQ (got DES-001)"

**Files:**
- `internal/trace/trace.go` (Add function)
- `internal/trace/trace_test.go`

---

### TASK-003: Update V2 Item Types
**Priority:** P0
**Links:** REQ-001, REQ-002, ARCH-005
**Depends on:** None

Add ISSUE to item.go constants, validNodeTypes map, and itemIDPattern.

**Acceptance Criteria:**
- [ ] `NodeTypeISSUE = "ISSUE"` constant added
- [ ] ISSUE in `validNodeTypes` map
- [ ] `itemIDPattern` regex includes ISSUE
- [ ] `TraceItem{ID: "ISSUE-1"}.Validate()` passes

**Files:**
- `internal/trace/item.go`
- `internal/trace/item_test.go`

---

### TASK-004: Update Artifact Scanning Pattern
**Priority:** P0
**Links:** REQ-002, DES-004, ARCH-006
**Depends on:** None

Update `scanArtifacts()` in validate.go to recognize ISSUE- prefix in artifact files.

**Acceptance Criteria:**
- [ ] Pattern in scanArtifacts includes ISSUE
- [ ] ISSUE-NNN IDs found in docs/issues.md are detected
- [ ] Add issues.md to scanned files list

**Files:**
- `internal/trace/validate.go` (line ~281)
- `internal/trace/validate_test.go`

---

### TASK-005: Update Coverage Rules for ISSUE
**Priority:** P1
**Links:** REQ-003, ARCH-007
**Depends on:** TASK-001, TASK-003

Ensure ISSUE has no mandatory downstream coverage (optional head of chain).

**Acceptance Criteria:**
- [ ] ISSUE with downstream REQ passes coverage validation
- [ ] ISSUE with no downstream passes coverage validation
- [ ] REQ without upstream ISSUE passes (no warning)

**Files:**
- `internal/trace/validate.go` or `builder.go`
- `internal/trace/validate_test.go`

---

### TASK-006: Update Trace Impact for ISSUE
**Priority:** P1
**Links:** REQ-005, DES-003
**Depends on:** TASK-001, TASK-003

Ensure `trace impact` includes ISSUE in forward/reverse analysis.

**Acceptance Criteria:**
- [ ] `trace impact --id ISSUE-1` shows downstream REQs
- [ ] `trace impact --id REQ-001 --reverse` shows upstream ISSUE

**Files:**
- `internal/trace/trace.go` (Impact function, if exists)
- `cmd/projctl/trace.go` (traceImpact)

---

### TASK-007: Update CLI Help Text
**Priority:** P2
**Links:** DES-001, DES-002, DES-003
**Depends on:** TASK-001, TASK-002, TASK-006

Update help text and examples to include ISSUE prefix.

**Acceptance Criteria:**
- [ ] `projctl trace add --help` shows ISSUE in examples
- [ ] `projctl trace validate --help` mentions ISSUE
- [ ] `projctl trace impact --help` mentions ISSUE

**Files:**
- `cmd/projctl/trace.go`

---

## Dependency Graph

```
TASK-001 (V1 regex) ────┬──→ TASK-002 (Add validation)
                        │
TASK-003 (V2 types) ────┼──→ TASK-005 (Coverage rules)
                        │
TASK-004 (Scan pattern) ┼──→ TASK-006 (Impact)
                        │
                        └──→ TASK-007 (Help text)
```

TASK-001, TASK-003, TASK-004 have no dependencies and can run in parallel.

## Traceability

| TASK | REQ | DES | ARCH |
|------|-----|-----|------|
| TASK-001 | REQ-001, REQ-002 | - | ARCH-004 |
| TASK-002 | REQ-001, REQ-004 | DES-001 | ARCH-004 |
| TASK-003 | REQ-001, REQ-002 | - | ARCH-005 |
| TASK-004 | REQ-002 | DES-004 | ARCH-006 |
| TASK-005 | REQ-003 | - | ARCH-007 |
| TASK-006 | REQ-005 | DES-003 | - |
| TASK-007 | - | DES-001, DES-002, DES-003 | - |
