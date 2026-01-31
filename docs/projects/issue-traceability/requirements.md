# Requirements: Issue Traceability

## Problem Statement

The traceability chain currently starts at REQ, but issues should be the optional head of the chain: `ISSUE → REQ → DES → ARCH → TASK → TEST`. This allows tracking work from its origin (an issue) through requirements to implementation.

## Success Criteria

SC-01: `projctl trace add --from ISSUE-001 --to REQ-001` works correctly
SC-02: `projctl trace validate` recognizes ISSUE- prefix without errors
SC-03: ISSUE links only to REQ (not directly to DES/ARCH/TASK)
SC-04: REQs without upstream ISSUE are valid (no warning, truly optional)
SC-05: Existing traceability without ISSUE continues to work unchanged

## Requirements

### REQ-001: Trace Add Accepts ISSUE Prefix
**Priority:** P0
**Links:** ISSUE-002

The `projctl trace add` command MUST accept ISSUE-NNN as the `--from` argument.

**Acceptance Criteria:**
- [ ] `projctl trace add --from ISSUE-001 --to REQ-001` succeeds
- [ ] ISSUE can only link to REQ (not DES, ARCH, TASK directly)
- [ ] Error message if ISSUE tries to link to non-REQ target

### REQ-002: Trace Validate Recognizes ISSUE Prefix
**Priority:** P0
**Links:** ISSUE-002

The `projctl trace validate` command MUST recognize ISSUE-NNN IDs in traceability.toml.

**Acceptance Criteria:**
- [ ] ISSUE-NNN IDs don't cause "unknown prefix" errors
- [ ] ISSUE IDs are included in coverage analysis
- [ ] Orphan ISSUE detection works (ISSUE in matrix but not in issues.md)

### REQ-003: ISSUE is Optional Head
**Priority:** P0
**Links:** ISSUE-002

The traceability chain MUST work with or without ISSUE as the head.

**Acceptance Criteria:**
- [ ] REQ without upstream ISSUE is valid (no warning)
- [ ] REQ with upstream ISSUE is valid
- [ ] Existing traceability.toml files without ISSUE work unchanged

### REQ-004: ISSUE Links Only to REQ
**Priority:** P1
**Links:** ISSUE-002

ISSUE MUST only be allowed to link to REQ, not to DES/ARCH/TASK/TEST directly.

**Acceptance Criteria:**
- [ ] `projctl trace add --from ISSUE-001 --to DES-001` returns error
- [ ] `projctl trace add --from ISSUE-001 --to ARCH-001` returns error
- [ ] Error message is clear: "ISSUE can only link to REQ"

### REQ-005: Trace Impact Includes ISSUE
**Priority:** P1
**Links:** ISSUE-002

The `projctl trace impact` command MUST include ISSUE in forward/backward analysis.

**Acceptance Criteria:**
- [ ] `projctl trace impact --id REQ-001 --reverse` shows upstream ISSUE if linked
- [ ] `projctl trace impact --id ISSUE-001` shows downstream REQs

## Out of Scope

- Automatic ISSUE creation from git commits or PR descriptions
- ISSUE status tracking (open/closed) in traceability
- Integration with external issue trackers (GitHub Issues, Jira)

## Traceability

| ID | Links To |
|----|----------|
| REQ-001 | ISSUE-002 |
| REQ-002 | ISSUE-002 |
| REQ-003 | ISSUE-002 |
| REQ-004 | ISSUE-002 |
| REQ-005 | ISSUE-002 |
