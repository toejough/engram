# Tasks

## TASK-1: Add phase-aware trace validation

### Summary
Make trace validation accept an optional phase parameter that relaxes validation rules based on workflow stage.

### Acceptance Criteria
- [ ] `ValidateV2Artifacts` accepts optional `phase` parameter
- [ ] At `architect-complete`: ARCH-NNN IDs allowed to be unlinked (no tasks trace to them yet)
- [ ] At `breakdown-complete`: TASK-NNN IDs allowed to be unlinked (no tests trace to them yet)
- [ ] At `task-complete` and later: Full chain required
- [ ] `projctl trace validate` CLI works without phase (strictest validation)
- [ ] `projctl trace validate --phase <phase>` uses phase-aware rules
- [ ] Preconditions pass current phase to validation
- [ ] No more `--force` needed for normal workflow transitions
- [ ] Unit tests verify phase-aware behavior

**Traces to:** ISSUE-044
