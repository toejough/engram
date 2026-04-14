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

9. **Hook integration tests must exercise the real protocol.** Hook scripts must have at least one test that pipes a representative JSON payload via stdin and asserts the downstream command receives the correct arguments. Static grep-based tests ("does the file contain this string") don't catch format mismatches between what the hook reads and what the system actually provides.

## External Systems

10. **Verify external contract assumptions before writing implementation steps.** When a plan references an external system's behavior (Claude Code hooks, GitHub API, etc.), include a verification step: "Confirm that [system] provides [data] via [mechanism] by checking [source]." Plans that assert "X provides Y" without citing documentation or code evidence are making untested assumptions that propagate into bugs.

11. **Consult the predecessor when rewriting.** Before rewriting a component, read the old implementation and its tests. Identify protocol details it got right (e.g., reading `.prompt` from stdin JSON) and carry them forward explicitly.

12. **Justify every blocking call in hooks.** For any hook that uses a blocking operation (e.g., `engram intent` which long-polls for a response), the plan must explicitly state why blocking adds value for that hook type. If the agent won't use the response (e.g., Stop hook — agent is exiting), use fire-and-forget instead.
