# Tasks

## TASK-1: Add phase-aware trace validation

### Summary
Make trace validation accept an optional phase parameter that relaxes validation rules based on workflow stage.

### Acceptance Criteria
- [x] `ValidateV2Artifacts` accepts optional `phase` parameter
- [x] At `architect-complete`: ARCH-NNN IDs allowed to be unlinked (no tasks trace to them yet)
- [x] At `breakdown-complete`: TASK-NNN IDs allowed to be unlinked (no tests trace to them yet)
- [x] At `task-complete` and later: Full chain required
- [x] `projctl trace validate` CLI works without phase (strictest validation)
- [x] `projctl trace validate --phase <phase>` uses phase-aware rules
- [x] Preconditions pass current phase to validation
- [x] No more `--force` needed for normal workflow transitions
- [x] Unit tests verify phase-aware behavior

**Traces to:** ISSUE-044
