# Requirements: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Project:** Clean up remaining yield references in docs
**Date:** 2026-02-06

## Overview

The old "yield" infrastructure (projctl yield validate, yield-based skill outputs) was removed in earlier issues, but some documentation files still reference yield concepts. This project addresses finding and cleaning up those remaining references.

## Requirements

### REQ-001: Complete Yield Reference Removal

**Description:** All references to the deprecated yield infrastructure must be completely removed from the codebase and documentation.

**Acceptance Criteria:**
- No remaining references to `projctl yield` commands in documentation
- No remaining references to yield-based skill outputs in documentation
- No orphaned yield-related content or broken links
- Changes are verified through grep/search to confirm completeness

**Priority:** High

**Traces to:** ISSUE-88

---

### REQ-002: Search Skill Documentation

**Description:** All SKILL.md files must be searched for yield-related references and updated.

**Acceptance Criteria:**
- All SKILL.md files in the repository are searched for yield references
- Each yield reference found is evaluated for appropriate replacement
- Updated skill documentation reflects current non-yield workflows
- No broken links to yield-related sections remain

**Priority:** High

**Traces to:** REQ-001, ISSUE-88

---

### REQ-003: Search Project Documentation

**Description:** Project documentation and guides (docs/ directories and related files) must be searched for yield-related references and updated.

**Acceptance Criteria:**
- All markdown files in docs/ directories are searched
- User-facing guides and documentation are updated
- Project planning documents are cleaned up
- Issue tracking documents referencing yield are updated

**Priority:** High

**Traces to:** REQ-001, ISSUE-88

---

### REQ-004: Replace with Current Workflows

**Description:** Yield references should not simply be deleted, but replaced with equivalent new commands/workflows where applicable.

**Acceptance Criteria:**
- Each yield command reference is replaced with its current equivalent
- Workflow documentation is updated to show current approach
- Users can follow updated documentation without confusion
- Migration path from old to new is clear (if historical context needed)

**Priority:** High

**Traces to:** REQ-001, ISSUE-88

---

### REQ-005: Verify Code Cleanliness

**Description:** Verify that no code still references yield infrastructure, ensuring both documentation and code are consistent.

**Acceptance Criteria:**
- Source code is searched for yield-related imports/functions
- Any remaining code references are identified and removed
- Build/test passes with yield references removed
- No runtime errors from missing yield infrastructure

**Priority:** High

**Traces to:** REQ-001, ISSUE-88

---

## Out of Scope

- Restoring or maintaining yield functionality
- Creating new yield-based features
- Updating tests that specifically test yield infrastructure (these should be removed)

## Success Metrics

- Zero grep matches for yield-related patterns in documentation
- Zero grep matches for yield-related patterns in code (excluding this requirements doc and historical issues)
- All documentation links functional
- All skill documentation reflects current workflows
- Build/test suite passes
