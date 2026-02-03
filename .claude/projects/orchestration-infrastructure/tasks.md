# Orchestration Infrastructure Tasks

Task breakdown with dependency DAG.

---

## TASK-001: Add CompletedTasks tracking to state package

Extend Progress struct with CompletedTasks slice. Add MarkTaskComplete() and IsTaskComplete() methods. Update Next() to exclude completed tasks.

**Acceptance Criteria:**

- [ ] Progress struct has `CompletedTasks []string` field
- [ ] `MarkTaskComplete(taskID)` appends to slice and persists
- [ ] `IsTaskComplete(taskID)` returns true if task in slice
- [ ] `Next()` filters completed tasks from suggestions
- [ ] Unit tests cover all new methods

**Traces to:** ARCH-001

---

## TASK-002: Add `projctl state complete` CLI command

CLI interface for marking tasks complete.

**Acceptance Criteria:**

- [ ] `projctl state complete --task TASK-NNN` marks task complete
- [ ] `projctl state get` shows completed tasks in output
- [ ] Error if task doesn't exist in tasks.md

**Depends on:** TASK-001

**Traces to:** ARCH-001

---

## TASK-003: Create internal/id package for ID generation

New package that scans artifact files and returns next sequential ID.

**Acceptance Criteria:**

- [ ] `id.Next(dir, "REQ")` returns next REQ-NNN
- [ ] `id.Next(dir, "DES")` returns next DES-NNN
- [ ] `id.Next(dir, "ARCH")` returns next ARCH-NNN
- [ ] `id.Next(dir, "TASK")` returns next TASK-NNN
- [ ] `id.Next(dir, "ISSUE")` returns next ISSUE-NNN
- [ ] Handles missing files (returns TYPE-001)
- [ ] Scans both root and docs/ subdirectory
- [ ] Unit tests with property-based coverage

**Traces to:** ARCH-002

---

## TASK-004: Add `projctl id next` CLI command

CLI interface for ID generation.

**Acceptance Criteria:**

- [ ] `projctl id next --type REQ` outputs next REQ ID
- [ ] `projctl id next --type TASK` outputs next TASK ID
- [ ] Output is plain text suitable for shell capture
- [ ] `--dir` flag specifies project directory

**Depends on:** TASK-003

**Traces to:** ARCH-002

---

## TASK-005: Add trace show functionality to internal/trace

Extend trace package with Show() function for visualization.

**Acceptance Criteria:**

- [ ] `trace.Show(dir, "ascii")` returns ASCII tree
- [ ] `trace.Show(dir, "json")` returns JSON graph
- [ ] Orphan IDs marked with `[ORPHAN]`
- [ ] Unlinked IDs marked with `[UNLINKED]`
- [ ] Uses existing Validate() graph-building logic
- [ ] Unit tests for both formats

**Traces to:** ARCH-003

---

## TASK-006: Add `projctl trace show` CLI command

CLI interface for trace visualization.

**Acceptance Criteria:**

- [ ] `projctl trace show` outputs ASCII tree (default)
- [ ] `projctl trace show --format json` outputs JSON
- [ ] `--dir` flag specifies project directory

**Depends on:** TASK-005

**Traces to:** ARCH-003

---

## TASK-007: Add trace promote functionality to internal/trace

New Promote() function that replaces TASK traces with permanent artifact IDs.

**Acceptance Criteria:**

- [ ] `trace.Promote(dir)` finds test files with TASK traces
- [ ] Looks up TASK's Traces-to field in tasks.md
- [ ] Replaces `// traces: TASK-NNN` with permanent ID
- [ ] Returns list of promotions made
- [ ] Handles Go, TypeScript, JavaScript test files
- [ ] Unit tests with example test files

**Depends on:** TASK-005

**Traces to:** ARCH-004

---

## TASK-008: Add `projctl trace promote` CLI command

CLI interface for trace promotion.

**Acceptance Criteria:**

- [ ] `projctl trace promote` promotes all TASK traces
- [ ] `--dry-run` flag shows what would be changed
- [ ] Reports number of files modified

**Depends on:** TASK-007

**Traces to:** ARCH-004

---

## TASK-009: Update tdd-qa skill for AC completeness

Update skill to enforce complete acceptance criteria.

**Acceptance Criteria:**

- [ ] tdd-qa parses AC from task definition in tasks.md
- [ ] Yields improvement-request if any AC is `[ ]`
- [ ] Yields escalate-user if producer claims deferral
- [ ] Skill documentation updated with new behavior

**Traces to:** ARCH-005

---

## TASK-010: Update retro-producer skill for issue creation

Update skill to create issues from recommendations.

**Acceptance Criteria:**

- [ ] Parses recommendations from retrospective.md
- [ ] Creates issue for each High/Medium priority item
- [ ] Creates issue for each open question
- [ ] Includes created issue IDs in yield payload
- [ ] Skill documentation updated

**Traces to:** ARCH-006

---

## TASK-011: Update breakdown-qa skill for traceability

Update skill to enforce mandatory Traces-to fields.

**Acceptance Criteria:**

- [ ] Validates every task has Traces-to field
- [ ] Invokes `projctl trace validate` as part of review
- [ ] Rejects breakdown if orphan tasks exist
- [ ] Skill documentation updated

**Traces to:** ARCH-007

---

## Dependency Graph

```
TASK-001 ─────────────────┐
                          ├─→ TASK-002
                          │
TASK-003 ─────────────────┼─→ TASK-004
                          │
TASK-005 ─────────────────┼─→ TASK-006
    │                     │
    └─→ TASK-007 ─────────┼─→ TASK-008
                          │
TASK-009 ─────────────────┤
TASK-010 ─────────────────┤
TASK-011 ─────────────────┘
```

**Parallel Groups:**
1. TASK-001, TASK-003, TASK-005 (no dependencies)
2. TASK-002, TASK-004, TASK-006, TASK-007, TASK-009, TASK-010, TASK-011 (after group 1)
3. TASK-008 (after TASK-007)

---
