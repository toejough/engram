# Architecture: Issue Traceability

## Overview

Add ISSUE- prefix support to projctl's traceability system. The existing architecture is well-designed for extension - most changes are regex/enum updates.

## Architecture Principles

ARCH-001: **Minimal invasive change.** Update existing patterns and enums, don't restructure.

ARCH-002: **Backward compatible.** All existing traceability files continue to work.

ARCH-003: **Both validation systems.** Update V1 (TOML-based) and V2 (graph-based) systems.

## Current Architecture

The trace system has two validation pipelines:

1. **V1 (Legacy):** `internal/trace/trace.go` - Simple TOML matrix validation
2. **V2 (Modern):** `internal/trace/validate_v2.go`, `builder.go`, `graph.go`, `item.go` - Graph-based validation

### ID Validation Points (4 locations)

| File | Line | Pattern | Used By |
|------|------|---------|---------|
| `trace.go` | 18 | `^(REQ\|DES\|ARCH\|TASK)-\d{3}$` | `ValidID()`, V1 validation |
| `item.go` | 64 | `^(REQ\|DES\|ARCH\|TASK\|TEST)-\d{3,}$` | `TraceItem.Validate()`, V2 validation |
| `validate.go` | 281 | `(REQ\|DES\|ARCH\|TASK)-\d{3}` | `scanArtifacts()`, file scanning |
| `item.go` | 14-19 | `NodeType` constants | Type enum |

## Files to Modify

### ARCH-004: Update trace.go (V1 System)
**Links:** REQ-001, REQ-002, REQ-004

**File:** `internal/trace/trace.go`

1. **Line 18:** Update `idPattern` regex:
   ```go
   // Before:
   idPattern = regexp.MustCompile(`^(REQ|DES|ARCH|TASK)-\d{3}$`)
   // After:
   idPattern = regexp.MustCompile(`^(ISSUE|REQ|DES|ARCH|TASK)-\d{3}$`)
   ```

2. **Add():** Add ISSUE→REQ validation (around line 40):
   ```go
   func Add(dir, from, to string, fs AddFS) error {
       // ... existing validation ...

       // ISSUE can only link to REQ
       if strings.HasPrefix(from, "ISSUE-") && !strings.HasPrefix(to, "REQ-") {
           return fmt.Errorf("ISSUE can only link to REQ (got %s)", to)
       }
       // ... rest of function ...
   }
   ```

3. **Validate():** Update coverage rules - ISSUE has no mandatory downstream (optional head).

### ARCH-005: Update item.go (V2 System)
**Links:** REQ-001, REQ-002

**File:** `internal/trace/item.go`

1. **Line 14-19:** Add ISSUE to NodeType constants:
   ```go
   const (
       NodeTypeISSUE = "ISSUE"  // NEW
       NodeTypeREQ   = "REQ"
       NodeTypeDES   = "DES"
       NodeTypeARCH  = "ARCH"
       NodeTypeTASK  = "TASK"
       NodeTypeTEST  = "TEST"
   )
   ```

2. **Line 47-53:** Add to `validNodeTypes` map:
   ```go
   var validNodeTypes = map[string]bool{
       NodeTypeISSUE: true,  // NEW
       NodeTypeREQ:   true,
       // ...
   }
   ```

3. **Line 64:** Update `itemIDPattern`:
   ```go
   // Before:
   itemIDPattern = regexp.MustCompile(`^(REQ|DES|ARCH|TASK|TEST)-\d{3,}$`)
   // After:
   itemIDPattern = regexp.MustCompile(`^(ISSUE|REQ|DES|ARCH|TASK|TEST)-\d{3,}$`)
   ```

### ARCH-006: Update validate.go (Artifact Scanning)
**Links:** REQ-002, DES-004

**File:** `internal/trace/validate.go`

1. **Line 281:** Update `scanArtifacts()` pattern:
   ```go
   // Before:
   pattern := regexp.MustCompile(`(REQ|DES|ARCH|TASK)-\d{3}`)
   // After:
   pattern := regexp.MustCompile(`(ISSUE|REQ|DES|ARCH|TASK)-\d{3}`)
   ```

2. **Add artifact path for issues.md** in scan locations (if not already present).

### ARCH-007: Update Coverage Rules
**Links:** REQ-003, REQ-004

ISSUE has special coverage rules:
- **No mandatory downstream:** ISSUE is optional head, so no coverage requirement
- **Can only link to REQ:** Enforced in Add(), not in coverage validation

In `validate.go` or `builder.go`, ensure:
```go
// ISSUE has no mandatory downstream - it's optional
if nodeType == NodeTypeISSUE {
    return true // Always passes coverage
}
```

### ARCH-008: Tests
**Links:** All REQs

**Files to update:**
- `internal/trace/trace_test.go` - Add ISSUE to property tests, add ISSUE→REQ tests
- `internal/trace/item_test.go` - Add ISSUE validation tests
- `internal/trace/validate_test.go` - Add ISSUE in V2 validation tests

**Test cases needed:**
1. `ISSUE-001` is valid ID format
2. `projctl trace add --from ISSUE-001 --to REQ-001` succeeds
3. `projctl trace add --from ISSUE-001 --to DES-001` fails with error
4. `projctl trace validate` passes with ISSUE in traceability.toml
5. Orphan ISSUE detection works
6. REQ without upstream ISSUE passes validation (no warning)

## Implementation Order

### ARCH-009: Task Sequence
**Links:** All REQs

1. **TASK-001:** Update regex in trace.go (V1)
2. **TASK-002:** Add ISSUE→REQ validation in trace.go Add()
3. **TASK-003:** Update item.go constants and regex (V2)
4. **TASK-004:** Update validate.go artifact scanning
5. **TASK-005:** Update coverage rules for ISSUE
6. **TASK-006:** Add tests for all changes
7. **TASK-007:** Update CLI help text

Tasks 1-4 can be done in parallel. Task 5-7 depend on 1-4.

## Traceability

| ID | Links To |
|----|----------|
| ARCH-001 | (principle) |
| ARCH-002 | (principle) |
| ARCH-003 | (principle) |
| ARCH-004 | REQ-001, REQ-002, REQ-004 |
| ARCH-005 | REQ-001, REQ-002 |
| ARCH-006 | REQ-002, DES-004 |
| ARCH-007 | REQ-003, REQ-004 |
| ARCH-008 | REQ-001, REQ-002, REQ-003, REQ-004, REQ-005 |
| ARCH-009 | All REQs |
