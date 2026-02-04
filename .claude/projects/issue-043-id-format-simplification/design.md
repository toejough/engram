# Design: ID Format Simplification

### DES-001: Pattern Changes

Change ID patterns from exactly 3 digits to any number of digits.

| Location | Current | New |
|----------|---------|-----|
| `internal/id/id.go` | `\d{3,}` scan, `%03d` format | `\d+` scan, `%d` format |
| `cmd/projctl/checker.go` | `\d{3}` | `\d+` |
| `cmd/projctl/checker_test.go` | `\d{3}` | `\d+` |
| `internal/trace/promote.go` | `\d{3}` | `\d+` |

#### Backward Compatibility

The `\d+` pattern matches both:
- New format: REQ-1, REQ-10, REQ-100
- Legacy format: REQ-001, REQ-010, REQ-100

No migration needed - existing documents remain valid.

**Traces to:** REQ-001
