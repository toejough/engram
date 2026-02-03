# Architecture: parallel-worktree-strategy

## ARCH-001: Worktree package

New package: `internal/worktree/`

```go
package worktree

type Manager struct {
    RepoDir string
}

func (m *Manager) Path(taskID string) string
func (m *Manager) Create(taskID string) (string, error)
func (m *Manager) List() ([]Worktree, error)
func (m *Manager) Merge(taskID, onto string) error
func (m *Manager) Cleanup(taskID string) error
func (m *Manager) CleanupAll() error

type Worktree struct {
    TaskID  string
    Path    string
    Branch  string
    Status  string
}
```

**Traces to:** DES-001, DES-002, DES-003, DES-004

---

## ARCH-002: Git operations via exec

All git operations via `os/exec`:

```go
func (m *Manager) git(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = m.RepoDir
    output, err := cmd.CombinedOutput()
    return string(output), err
}
```

No git library dependency - matches existing projctl pattern.

**Traces to:** DES-002, DES-003, DES-004

---

## ARCH-003: CLI commands

Add to `cmd/projctl/worktree.go`:

```go
type worktreeCreateArgs struct {
    Task string `targ:"flag,short=t,required,desc=Task ID"`
}

type worktreeMergeArgs struct {
    Task string `targ:"flag,short=t,required,desc=Task ID"`
    Onto string `targ:"flag,short=o,desc=Target branch (default: current)"`
}

// etc.
```

Wire into main command tree.

**Traces to:** DES-005

---

## ARCH-004: State integration

Extend `internal/state/state.go`:

```go
type State struct {
    // existing fields...
    Worktrees map[string]WorktreeState `toml:"worktrees,omitempty"`
}

type WorktreeState struct {
    Path    string    `toml:"path"`
    Branch  string    `toml:"branch"`
    Created time.Time `toml:"created"`
    Status  string    `toml:"status"`
}
```

**Traces to:** DES-007

---

## ARCH-005: Error handling for conflicts

Merge returns structured error:

```go
type MergeConflictError struct {
    TaskID       string
    ConflictFiles []string
    Message      string
}

func (e *MergeConflictError) Error() string
```

Orchestrator can catch and handle appropriately.

**Traces to:** DES-003, REQ-007

---

## ARCH-006: Orchestrator integration point

Orchestrator (future) calls worktree manager:

```go
// In parallel task execution
for _, task := range parallelTasks {
    path, err := wm.Create(task.ID)
    // spawn agent with cwd=path
}

// After all complete
for _, task := range completedTasks {
    if err := wm.Merge(task.ID, currentBranch); err != nil {
        if conflicts, ok := err.(*MergeConflictError); ok {
            // handle conflict
        }
    }
}
wm.CleanupAll()
```

**Traces to:** DES-006
