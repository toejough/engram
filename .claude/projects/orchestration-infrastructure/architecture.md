# Orchestration Infrastructure Architecture

Technical decisions for CLI commands and skill updates.

---

## CLI Commands

### ARCH-001: Completed task tracking in state package

Extend `internal/state/state.go`:

```go
type Progress struct {
    CurrentTask      string   `toml:"current_task"`
    CurrentSubphase  string   `toml:"current_subphase"`
    TasksComplete    int      `toml:"tasks_complete"`
    TasksTotal       int      `toml:"tasks_total"`
    TasksEscalated   int      `toml:"tasks_escalated"`
    CompletedTasks   []string `toml:"completed_tasks"`  // NEW
}
```

Add `MarkTaskComplete(taskID string)` and `IsTaskComplete(taskID string) bool` methods.

Update `Next()` to filter completed tasks from suggestions.

**Traces to:** DES-001

---

### ARCH-002: ID generation package

New package: `internal/id/`

```go
package id

// Next returns the next sequential ID for the given type.
// Scans artifact files to find highest existing ID.
func Next(dir string, idType string) (string, error)

// idType: "REQ", "DES", "ARCH", "TASK", "ISSUE"
```

File mapping:
- REQ → requirements.md, docs/requirements.md
- DES → design.md, docs/design.md
- ARCH → architecture.md, docs/architecture.md
- TASK → tasks.md, docs/tasks.md
- ISSUE → docs/issues.md

Uses regex `### (TYPE)-(\d+):` to find existing IDs.

CLI: `cmd/projctl/id.go` with `id next --type <TYPE>` subcommand.

**Traces to:** DES-002

---

### ARCH-003: Trace visualization in trace package

Extend `internal/trace/`:

```go
// Show generates traceability visualization
func Show(dir string, format string) (string, error)

// format: "ascii" (default) or "json"
```

ASCII renderer:
- Build graph from existing `Validate()` logic
- Walk graph depth-first, rendering tree structure
- Annotate orphans with `[ORPHAN]`, unlinked with `[UNLINKED]`

JSON renderer:
- Output graph as `{"nodes": [...], "edges": [...]}`

CLI: Add `trace show [--format ascii|json]` to existing trace command.

**Traces to:** DES-003

---

## Skill Updates

### ARCH-004: Trace promotion command

New command: `projctl trace promote [--dir <path>]`

Implementation in `internal/trace/`:

```go
// Promote replaces TASK-NNN traces in test files with permanent artifact IDs
func Promote(dir string) ([]Promotion, error)

type Promotion struct {
    File     string
    OldTrace string // TASK-001
    NewTrace string // ARCH-005
}
```

Logic:
1. Glob for `*_test.go`, `*.test.ts`, `*.spec.ts` etc.
2. Find `// traces: TASK-NNN` comments
3. Look up TASK-NNN in tasks.md, get its `**Traces to:**` value
4. Replace comment with permanent ID
5. Return list of promotions for reporting

`doc-producer` skill invokes this command.

**Traces to:** DES-004

---

### ARCH-005: tdd-qa AC validation

Update `skills/tdd-qa/SKILL.md` to include AC completeness check.

No projctl changes needed - skill reads tasks.md directly and validates:
1. Parse task's AC section
2. Count `- [ ]` vs `- [x]`
3. Yield appropriately based on findings

**Traces to:** DES-005

---

### ARCH-006: retro-producer issue creation

Update `skills/retro-producer/SKILL.md` to invoke `projctl issue create` for recommendations.

Skill is responsible for:
1. Parsing its own output (retrospective.md)
2. Identifying High/Medium priority recommendations
3. Calling `projctl issue create` for each
4. Including issue IDs in yield

**Traces to:** DES-006

---

### ARCH-007: breakdown-qa traceability validation

Update `skills/breakdown-qa/SKILL.md` to validate traceability.

QA must check:
1. Every task has `**Traces to:**` field
2. Run `projctl trace validate` as part of review
3. Reject if orphan tasks exist

**Traces to:** DES-007

---

## Dependencies

```
ARCH-001 (state) ← no dependencies
ARCH-002 (id)    ← no dependencies
ARCH-003 (trace) ← ARCH-002 (uses similar file scanning)
ARCH-004 (promote) ← ARCH-003 (builds on trace graph)
ARCH-005 (tdd-qa) ← no code dependencies, skill-only
ARCH-006 (retro) ← no code dependencies, skill-only
ARCH-007 (breakdown) ← no code dependencies, skill-only
```

Suggested implementation order:
1. ARCH-001, ARCH-002 (parallel, no dependencies)
2. ARCH-003 (after ARCH-002)
3. ARCH-004 (after ARCH-003)
4. ARCH-005, ARCH-006, ARCH-007 (parallel, skill-only)

---
