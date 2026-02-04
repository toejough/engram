# Requirements: ID Format Simplification

### ISSUE-043: ID format should be simple incrementing numbers

**Traces to:** (root)

### REQ-001: Simple Incrementing ID Format

ID generation and validation should use simple incrementing numbers (REQ-1, REQ-2, REQ-10) instead of zero-padded format (REQ-001, REQ-002).

#### Rationale

- Skills manually creating IDs (REQ-1) currently fail validation
- IDs >= 1000 would fail validation (TASK-1000 doesn't match `\d{3}`)
- Generation and validation patterns are inconsistent

#### Acceptance Criteria

- [ ] ID generation produces simple numbers without zero-padding
- [ ] ID scanning accepts any number of digits (`\d+`)
- [ ] Validation patterns updated across all affected files
- [ ] Existing 3-digit IDs remain valid (backward compatible)

**Traces to:** ISSUE-043
