# Execution Planning Principles

Distilled from planning sessions. Reference these when writing implementation plans.

## Context & I/O

1. **Context flows from the very top.** Never `context.Background()` in handlers. Context is created at the entry point and threaded through so Ctrl+C works everywhere.

2. **DI for all I/O -- no exceptions in `internal/`.** Pure handler functions accept interfaces. The only place real implementations (`http.DefaultClient`, `os.Stdout`) appear is in thin wiring at the edge. Separate `doXxx` (pure, testable) from `runXxx` (thin wiring).

## Testing

3. **Use imptest for interactive mocks.** Don't hand-write fakes. Use `//go:generate impgen` for type-safe mocks with interactive control.

4. **Property-based tests with rapid everywhere.** Canonical invariant format: "X always Y", "all X satisfy Y". Not just example-based tests.

5. **Full TDD cycle as explicit steps.** Red (failing test) -> Green (minimal implementation) -> Refactor (DRY, SOLID, simplification, deduplication). The refactor step is not optional -- it's a named step in every task.

## Completion

6. **No untracked deferrals.** If an implementation step defers critical wiring as "future work," that work must be captured as a tracked issue or plan amendment before the step is marked complete. A commit that says "future integration will do X" without a corresponding task is a silent feature deletion.

7. **Stage completion requires a smoke test.** Before marking any stage complete, run the actual binary, exercise the new code paths through the real entry point, and inspect the output. Document what was verified. Passing unit tests is necessary but not sufficient — unit tests verify components, smoke tests verify wiring.

8. **Untestable gaps need manual checklists.** When an automated test identifies something it can't cover (e.g., "can't test without real binary"), create a manual verification step in the plan. "Can't test X automatically" means "test X manually and document the result," not "skip testing X."
