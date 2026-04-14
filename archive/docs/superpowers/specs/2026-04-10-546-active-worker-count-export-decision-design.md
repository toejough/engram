# Design: ActiveWorkerCount Export Decision (Issue #546)

**Date:** 2026-04-10
**Issue:** #546 — exported `ActiveWorkerCount` deviates from Phase 5 plan which specified unexported `activeWorkerCount`

---

## Problem

The Phase 5 plan (`docs/superpowers/plans/2026-04-07-phase5-agent-resume.md`, Task 2) specified:

```go
func activeWorkerCount(sf StateFile) int {
```

The implementation exports it:

```go
func ActiveWorkerCount(sf StateFile) int {
```

After issue #542 removed the `MaxConcurrentWorkers` cap, `ActiveWorkerCount` is no longer called from `internal/cli`. It is only referenced in `internal/agent/agent_test.go` (blackbox test package `agent_test`).

---

## Options Considered

### Option A: Make Unexported + Move to Whitebox Tests

Rename to `activeWorkerCount` (unexported). Move tests from `package agent_test` to `package agent`.

**Rejected because:** Violates project convention. CLAUDE.md mandates blackbox tests (`package foo_test`). An unexported function cannot be tested from `package agent_test`. Moving tests to `package agent` (whitebox) creates a convention exception with no compensating benefit.

### Option B: Move to `internal/cli` as Private Helper

Move function body into `internal/cli/cli_agent.go` as an unexported `activeWorkerCount`. Tests would live in `cli_test.go`.

**Rejected because:** Worker-count-from-state-file is domain logic, not CLI logic. Domain logic belongs in `internal/agent`. The `cli` package is for OS wiring and argument parsing, not pure computation. Moving domain logic to `cli` inverts the abstraction hierarchy.

### Option C: Keep Exported — Document Rationale (Selected)

Keep `ActiveWorkerCount` exported in `internal/agent`. Add doc comment explaining the export rationale and intended future use.

**Selected because:**
1. Design spec #542 explicitly preserved the function: "still useful for observability/reporting; remains in place."
2. Export is cost-free — it enables future callers (e.g., `engram agent status`) without touching `internal/agent`.
3. Blackbox tests (`package agent_test`) remain valid and follow project convention.
4. The Phase 5 plan's `activeWorkerCount` naming was aspirational; cross-package call (`agentpkg.ActiveWorkerCount`) requires the export regardless.

---

## Decision

**Keep `ActiveWorkerCount` exported.** Add a godoc comment that documents the export rationale and anticipated observability use case.

---

## Changes

### `internal/agent/agent.go`

Update the `ActiveWorkerCount` doc comment to document the export intent:

```go
// ActiveWorkerCount returns the number of agents in STARTING or ACTIVE state.
// Exported for observability consumers (e.g. status commands, reporters) across
// the internal/ package boundary. Pure function — no I/O.
func ActiveWorkerCount(sf StateFile) int {
```

### `internal/agent/agent_test.go`

Replace the four example-based `TestActiveWorkerCount_*` tests with a single table-driven property test that covers all state transitions. Table-driven tests are more maintainable and document the full property space explicitly:

- ACTIVE agents count
- STARTING agents count  
- SILENT agents do NOT count
- DEAD agents do NOT count
- Unknown/empty state agents do NOT count
- Empty StateFile returns 0
- Mixed states count only ACTIVE + STARTING

---

## Acceptance Criteria

- [ ] `ActiveWorkerCount` remains exported in `internal/agent/agent.go`
- [ ] Doc comment updated to explain the export rationale
- [ ] Example-based tests replaced with a single table-driven property test
- [ ] `targ check-full` passes with no errors or warnings

---

## References

- Issue: #546
- Phase 5 plan: `docs/superpowers/plans/2026-04-07-phase5-agent-resume.md` Task 2
- Design spec #542: `docs/superpowers/specs/2026-04-10-542-remove-max-concurrent-workers-design.md`
- Implementation: `internal/agent/agent.go:49–61`
- Tests: `internal/agent/agent_test.go:12–60`
