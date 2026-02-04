# Architecture: ID Format Simplification

### ARCH-001: ID Pattern Update

No architectural changes required. This is a regex pattern update across 4 files.

#### Affected Components

1. **internal/id/id.go** - ID generation and scanning
2. **cmd/projctl/checker.go** - Precondition validation
3. **cmd/projctl/checker_test.go** - Test assertions
4. **internal/trace/promote.go** - Trace promotion patterns

#### Change Type

Pattern substitution only - no structural changes to APIs or data flow.

**Traces to:** DES-001
