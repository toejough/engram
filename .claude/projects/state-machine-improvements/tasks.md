# Tasks: state-machine-improvements

## TASK-001: Add RepoDir field to state

Add `RepoDir` field to `Project` struct in state package.

**Files:** `internal/state/state.go`

**Acceptance Criteria:**
- [x] `Project` struct has `RepoDir string` field with toml tag
- [x] Tests verify RepoDir is persisted to state.toml
- [x] Tests verify RepoDir is loaded from state.toml

**Traces to:** ARCH-001

---

## TASK-002: Create repo root detection utility

Create function to find git repository root.

**Files:** `internal/state/repo.go` (new)

**Acceptance Criteria:**
- [x] `FindRepoRoot` returns git root for directories inside a repo
- [x] `FindRepoRoot` returns error for directories outside a repo
- [x] Function is tested with real git repo in temp dir

**Traces to:** ARCH-002

---

## TASK-003: Update state init to accept repo-dir

Add `--repo-dir` flag to init command with auto-detection fallback.

**Files:** `cmd/projctl/state.go`

**Acceptance Criteria:**
- [x] `--repo-dir` flag is accepted by init command
- [x] If not provided, auto-detects git root
- [x] RepoDir is stored in state.toml

**Dependencies:** TASK-001, TASK-002

**Traces to:** ARCH-003

---

## TASK-004: Wire repo dir to preconditions

Update transition logic to pass repo dir to code-related preconditions.

**Files:** `internal/state/transitions.go`, `cmd/projctl/checker.go`

**Acceptance Criteria:**
- [x] `TestsExist` receives repo dir, not project dir
- [x] Artifact checks still receive project dir
- [x] Existing tests pass

**Dependencies:** TASK-001

**Traces to:** ARCH-004, ARCH-005

---

## TASK-005: Integration test for TDD cycle

Create integration test validating full TDD transitions with proper repo detection.

**Files:** `internal/state/state_integration_test.go` (new)

**Acceptance Criteria:**
- [x] Test creates temp git repo with test file
- [x] Test runs through task-start → tdd-red → tdd-green transitions
- [x] Transitions succeed without `--force`
- [x] Task completion is tracked in state

**Dependencies:** TASK-003, TASK-004

**Traces to:** ARCH-006

---

## Dependency Graph

```
TASK-001 ──┬──► TASK-003 ──┬──► TASK-005
           │               │
TASK-002 ──┘               │
           │               │
           └──► TASK-004 ──┘
```

TASK-001 and TASK-002 can run in parallel. TASK-003 and TASK-004 depend on TASK-001. TASK-005 depends on both TASK-003 and TASK-004.
