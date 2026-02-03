# Architecture: state-machine-improvements

## ARCH-001: State package changes

**File:** `internal/state/state.go`

Add `RepoDir` field to Project struct and update Init/Get functions to handle it.

**Traces to:** DES-001, DES-005

---

## ARCH-002: Repo detection utility

**File:** `internal/state/repo.go` (new)

Isolate git root detection logic:
- `FindRepoRoot(startDir string) (string, error)`
- Handles non-git directories gracefully
- Returns absolute path

**Traces to:** DES-003

---

## ARCH-003: State init command

**File:** `cmd/projctl/state.go`

Update `stateInit` to:
- Accept `--repo-dir` flag in args struct
- Call `FindRepoRoot` if not provided
- Pass to `state.Init`

**Traces to:** DES-002

---

## ARCH-004: Transition precondition wiring

**File:** `internal/state/transitions.go`

Update transition validation to:
- Load state to get both `projectDir` and `repoDir`
- Pass `repoDir` to code-related preconditions
- Pass `projectDir` to artifact-related preconditions

**Traces to:** DES-004

---

## ARCH-005: Checker implementation

**File:** `cmd/projctl/checker.go`

Update `TestsExist` to use the directory it receives (will now be repo dir).
No interface change needed - just correct usage by callers.

**Traces to:** DES-004

---

## ARCH-006: Integration test

**File:** `internal/state/state_integration_test.go` (new)

Integration test using real git repo in temp dir:
- Validates full TDD cycle transitions
- Confirms preconditions work with repo dir

**Traces to:** DES-006
