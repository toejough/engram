# ActiveWorkerCount Export Decision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Document the rationale for keeping `ActiveWorkerCount` exported and replace four example-based tests with one comprehensive table-driven property test.

**Architecture:** Pure refactor — no behavioral change. Doc comment addition justifies the export; table-driven test replaces four narrow example tests with a single canonical property specification covering all agent states. Implementation is already correct; this plan locks in the contract documentation and cleans up the test layer.

**Tech Stack:** Go, gomega, `targ check-full` for verification.

---

## File Map

| File | Change |
|------|--------|
| `internal/agent/agent_test.go` | Phase 1: add table-driven test. Phase 3: delete four old example tests. |
| `internal/agent/agent.go` | Phase 2: update `ActiveWorkerCount` doc comment to document export rationale. |

---

### Task 1: Write the table-driven property test (Phase 1 — contract specification)

> Writing the new test first locks in the behavioral contract before any refactoring.
> The test passes immediately (the implementation is already correct), confirming the
> new test design is sound before we touch anything else.

**Files:**
- Modify: `internal/agent/agent_test.go` — add `TestActiveWorkerCount` after line 10 (after the imports block), leaving the four old tests in place for now.

- [ ] **Step 1: Add the table-driven test**

In `internal/agent/agent_test.go`, insert this function immediately after the import block (before `TestActiveWorkerCount_CountsActiveAgents`):

```go
func TestActiveWorkerCount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		agents   []agent.AgentRecord
		expected int
	}{
		{
			name:     "empty state file returns zero",
			agents:   nil,
			expected: 0,
		},
		{
			name: "counts ACTIVE agents",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "ACTIVE"},
				{Name: "exec-2", State: "ACTIVE"},
			},
			expected: 2,
		},
		{
			name: "counts STARTING agents",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "STARTING"},
				{Name: "exec-2", State: "STARTING"},
			},
			expected: 2,
		},
		{
			name: "counts ACTIVE and STARTING together",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "ACTIVE"},
				{Name: "exec-2", State: "STARTING"},
			},
			expected: 2,
		},
		{
			name: "ignores SILENT agents",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "SILENT"},
			},
			expected: 0,
		},
		{
			name: "ignores DEAD agents",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "DEAD"},
			},
			expected: 0,
		},
		{
			name: "ignores unknown state agents",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "UNKNOWN"},
			},
			expected: 0,
		},
		{
			name: "mixed states: only ACTIVE and STARTING count",
			agents: []agent.AgentRecord{
				{Name: "exec-1", State: "ACTIVE"},
				{Name: "exec-2", State: "SILENT"},
				{Name: "exec-3", State: "STARTING"},
				{Name: "exec-4", State: "DEAD"},
				{Name: "exec-5", State: "UNKNOWN"},
			},
			expected: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			sf := agent.StateFile{Agents: tc.agents}
			g.Expect(agent.ActiveWorkerCount(sf)).To(Equal(tc.expected))
		})
	}
}
```

- [ ] **Step 2: Run tests to confirm the new test passes**

```bash
targ test 2>&1 | grep -E "FAIL|PASS|ActiveWorkerCount"
```

Expected: all `TestActiveWorkerCount` subtests PASS. The four old example tests also still PASS. No failures.

If any subtest fails, the implementation has a bug — do NOT proceed until passing.

---

### Task 2: Update the doc comment (Phase 2 — document the decision)

**Files:**
- Modify: `internal/agent/agent.go:49–51` — update `ActiveWorkerCount` doc comment.

- [ ] **Step 3: Update the doc comment**

In `internal/agent/agent.go`, replace lines 49–51:

```go
// ActiveWorkerCount returns the number of agents in STARTING or ACTIVE state.
// Pure function — no I/O.
func ActiveWorkerCount(sf StateFile) int {
```

With:

```go
// ActiveWorkerCount returns the number of agents in STARTING or ACTIVE state.
// Exported for observability consumers (e.g. status commands, reporters) across
// the internal/ package boundary. Pure function — no I/O.
func ActiveWorkerCount(sf StateFile) int {
```

- [ ] **Step 4: Run tests to confirm no regressions**

```bash
targ test 2>&1 | grep -E "FAIL|ok"
```

Expected: all packages PASS, no failures.

---

### Task 3: Remove the four old example tests and run full checks (Phase 3 — refactor)

> The table-driven test from Task 1 supersedes these four narrow example tests.
> Each old test covers exactly one case; the table-driven test covers all of them
> plus additional state combinations.

**Files:**
- Modify: `internal/agent/agent_test.go` — delete four functions.

- [ ] **Step 5: Delete the four old example tests**

In `internal/agent/agent_test.go`, delete these four complete functions (currently lines 12–60):

```go
func TestActiveWorkerCount_CountsActiveAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "ACTIVE"},
			{Name: "exec-2", State: "SILENT"},
			{Name: "exec-3", State: "ACTIVE"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(2))
}

func TestActiveWorkerCount_CountsStartingAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "STARTING"},
			{Name: "exec-2", State: "STARTING"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(2))
}

func TestActiveWorkerCount_EmptyStateFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(agent.ActiveWorkerCount(agent.StateFile{})).To(Equal(0))
}

func TestActiveWorkerCount_IgnoresSilentDeadUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stateFile := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "exec-1", State: "SILENT"},
			{Name: "exec-2", State: "DEAD"},
			{Name: "exec-3", State: "UNKNOWN"},
		},
	}

	g.Expect(agent.ActiveWorkerCount(stateFile)).To(Equal(0))
}
```

After deletion, `TestActiveWorkerCount` (the table-driven test from Task 1) is the only `ActiveWorkerCount` test remaining.

- [ ] **Step 6: Run full quality check**

```bash
targ check-full
```

Expected: all lint, vet, and tests pass. No errors or warnings.

- [ ] **Step 7: Verify no stale example test names remain**

```bash
grep -n "TestActiveWorkerCount_" internal/agent/agent_test.go
```

Expected: no output (all four old tests deleted).

- [ ] **Step 8: Commit**

```bash
git add internal/agent/agent.go internal/agent/agent_test.go
git commit -m "$(cat <<'EOF'
refactor(agent): document ActiveWorkerCount export rationale, table-driven tests (#546)

Keeps ActiveWorkerCount exported per design spec #542 ("still useful for
observability/reporting"). Adds godoc explaining the export intent.
Replaces four narrow example tests with one table-driven property test
covering all agent states (ACTIVE, STARTING, SILENT, DEAD, UNKNOWN, mixed).

AI-Used: [claude]
EOF
)"
```
