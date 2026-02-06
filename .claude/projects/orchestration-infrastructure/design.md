# Orchestration Infrastructure Design

Design decisions for CLI commands and skill enforcement.

---

## CLI Commands

### DES-001: Completed task tracking format

Tasks are tracked in state.toml under `[progress]`:

```toml
[progress]
completed_tasks = ["TASK-001", "TASK-002", "TASK-003"]
```

`state next` queries this list and excludes completed tasks when suggesting next work.

**Traces to:** REQ-001

---

### DES-002: ID generation command interface

```bash
projctl id next --type <TYPE> [--dir <path>]
```

Returns the next sequential ID by scanning artifact files:
- REQ: requirements.md
- DES: design.md
- ARCH: architecture.md
- TASK: tasks.md

Output is plain text (e.g., `REQ-008`) suitable for shell capture.

**Traces to:** REQ-002

---

### DES-003: Trace visualization interface

```bash
projctl trace show [--format ascii|json] [--dir <path>]
```

ASCII output (default):
```
ISSUE-26 "Orchestration Infrastructure"
├── REQ-001 "Completed task tracking"
│   ├── DES-001 [ARCH-001]
│   │   └── TASK-001 → test_completed_tracking.go [OK]
│   └── [UNLINKED] DES-002
└── REQ-002 "ID generation"
    └── [ORPHAN] ARCH-999 (referenced but undefined)
```

JSON output provides machine-readable graph for tooling integration.

**Traces to:** REQ-003

---

## Skill Enforcement

### DES-004: Test trace promotion mechanism

`doc-producer` skill gains a "trace promotion" step:

1. Scan test files for `// traces: TASK-NNN` comments
2. Look up TASK-NNN's `**Traces to:**` field in tasks.md
3. Replace with the permanent artifact ID (prefer ARCH, fall back to DES, REQ)
4. Verify with `projctl trace validate`

Alternative: `projctl trace promote` command that doc-producer invokes.

**Traces to:** REQ-004

---

### DES-005: AC completeness check in tdd-qa

`tdd-qa` skill adds validation step before yielding approval:

1. Parse current task from tasks.md
2. Extract all AC items (lines matching `- [ ]` or `- [x]`)
3. If any `- [ ]` found → yield `improvement-request`
4. If producer output contains "defer", "skip", "out of scope" → yield `escalate-user`

**Traces to:** REQ-005

---

### DES-006: Retro-to-issues conversion

`retro-producer` skill gains issue creation responsibility:

1. After generating retrospective.md, parse for `## Recommendations` section
2. For each recommendation with Priority High or Medium:
   - `projctl issue create --title "Retro: <recommendation>" --body "..."`
3. For each open question:
   - `projctl issue create --title "Decision needed: <question>" --body "..." --labels needs-decision`
4. Include created issue IDs in yield payload

**Traces to:** REQ-006

---

### DES-007: Mandatory traceability enforcement

`breakdown-producer` template requires `**Traces to:**` field:

```markdown
### TASK-NNN: <title>

<description>

**Acceptance Criteria:**
- [ ] ...

**Traces to:** <ARCH-NNN or DES-NNN or REQ-NNN>
```

`breakdown-qa` validates:
1. Every task has `**Traces to:**` field
2. Referenced IDs exist in artifacts
3. No orphan tasks

**Traces to:** REQ-007

---
