# Tasks: ID Format Simplification

### TASK-001: Update ID patterns to use simple numbers

Update all regex patterns from exactly 3 digits (`\d{3}`) to any number of digits (`\d+`), and change ID generation from zero-padded (`%03d`) to simple numbers (`%d`).

#### Files to modify

1. `internal/id/id.go` - Change scanning pattern and format string
2. `cmd/projctl/checker.go` - Update validation patterns
3. `cmd/projctl/checker_test.go` - Update test patterns
4. `internal/trace/promote.go` - Update trace patterns

#### Acceptance Criteria

- [ ] `internal/id/id.go` generates REQ-1, not REQ-001
- [ ] `internal/id/id.go` scans for `\d+` pattern
- [ ] `cmd/projctl/checker.go` validates `\d+` pattern
- [ ] `cmd/projctl/checker_test.go` updated for new format
- [ ] `internal/trace/promote.go` uses `\d+` pattern
- [ ] Existing 3-digit IDs still work (backward compatible)
- [ ] All existing tests pass

**Traces to:** ARCH-001
