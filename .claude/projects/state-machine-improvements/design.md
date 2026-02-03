# Design: state-machine-improvements

## DES-001: State struct extension

Add `RepoDir` field to `ProjectState.Project`:

```go
type Project struct {
    Name      string    `toml:"name"`
    Created   time.Time `toml:"created"`
    Phase     string    `toml:"phase"`
    Workflow  string    `toml:"workflow"`
    Issue     string    `toml:"issue,omitempty"`
    RepoDir   string    `toml:"repo_dir,omitempty"`  // NEW
}
```

**Traces to:** REQ-001

---

## DES-002: Init command changes

Update `projctl state init` to:
1. Accept optional `--repo-dir` flag
2. If not provided, auto-detect git root via `git rev-parse --show-toplevel`
3. Store relative path from project dir to repo dir (or absolute if outside project tree)

```
projctl state init --name foo                    # auto-detect repo
projctl state init --name foo --repo-dir /path   # explicit
```

**Traces to:** REQ-002

---

## DES-003: Repo root detection

Add helper function to find git root:

```go
func FindRepoRoot(startDir string) (string, error) {
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    cmd.Dir = startDir
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("not in a git repository")
    }
    return strings.TrimSpace(string(output)), nil
}
```

**Traces to:** REQ-002

---

## DES-004: PreconditionChecker interface update

Update the interface to accept both directories:

```go
type PreconditionChecker interface {
    // Artifact checks (use projectDir)
    RequirementsExist(projectDir string) bool
    DesignExists(projectDir string) bool
    AcceptanceCriteriaComplete(projectDir, taskID string) bool

    // Code checks (use repoDir)
    TestsExist(repoDir string) bool
    TestsFail(repoDir string) bool
    TestsPass(repoDir string) bool
}
```

Transition logic passes appropriate directory to each check.

**Traces to:** REQ-003

---

## DES-005: Backward compatibility handling

When loading state.toml:
- If `repo_dir` is empty/missing, call `FindRepoRoot(projectDir)`
- Cache result for session (don't repeatedly shell out)

**Traces to:** REQ-005

---

## DES-006: Integration test design

Create integration test that:
1. Sets up temp repo with git init
2. Creates test file in repo
3. Initializes project state
4. Runs through task-start → tdd-red → commit-red → tdd-green transitions
5. Verifies `TestsExist` precondition passes without `--force`
6. Verifies task completion is tracked in state

**Traces to:** REQ-004
