# Execution Planning Principles

Distilled from planning sessions. Reference these when writing implementation plans.

## Context & I/O

1. **Context flows from the very top.** Never `context.Background()` in handlers. Context is created at the entry point and threaded through so Ctrl+C works everywhere.

2. **DI for all I/O -- no exceptions in `internal/`.** Pure handler functions accept interfaces. The only place real implementations (`http.DefaultClient`, `os.Stdout`) appear is in thin wiring at the edge. Separate `doXxx` (pure, testable) from `runXxx` (thin wiring).

## Testing

3. **Use imptest for interactive mocks.** Don't hand-write fakes. Use `//go:generate impgen` for type-safe mocks with interactive control.

4. **Property-based tests with rapid everywhere.** Canonical invariant format: "X always Y", "all X satisfy Y". Not just example-based tests.

5. **Full TDD cycle as explicit steps.** Red (failing test) -> Green (minimal implementation) -> Refactor (DRY, SOLID, simplification, deduplication). The refactor step is not optional -- it's a named step in every task.
