# ISSUE-053: Universal QA Skill Design

Design decisions for replacing 13 phase-specific QA skills with one universal QA skill.

---

### DES-001: Contract YAML Format

Simple flat YAML structure optimized for LLM parsing (Sonnet producing, Haiku validating).

```yaml
contract:
  outputs:
    - path: "docs/design.md"
      id_format: "DES-N"

  traces_to:
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every entry has a DES-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every DES-N traces to at least one REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "No orphan ID references"
      severity: warning
```

**Design rationale:**
- Flat structure (no nested categories) for straightforward parsing
- `severity: error` means QA fails; `severity: warning` means QA passes with note
- `outputs` specifies what files the producer creates and their ID format
- `traces_to` specifies what upstream artifacts must be referenced
- `checks` is an ordered list of validation criteria

**Traces to:** REQ-006

---

### DES-002: Contract Section Placement

Contracts live in producer SKILL.md files as a fenced YAML code block under a `## Contract` heading.

```markdown
## Contract

```yaml
contract:
  outputs:
    - path: "docs/requirements.md"
      id_format: "REQ-N"
  traces_to:
    - "issue description"
  checks:
    - id: "CHECK-001"
      description: "Every requirement has REQ-N ID"
      severity: error
```
```

**Design rationale:**
- Single source of truth in producer's own documentation
- QA skill receives SKILL.md path, extracts contract section
- No separate contract files to maintain

**Traces to:** REQ-006, REQ-007

---

### DES-003: QA Output Format

Full checklist display for every QA run, regardless of pass/fail status.

**Pass example:**
```
QA Results: PASSED

[x] CHECK-001: Every entry has a DES-N identifier
[x] CHECK-002: Every DES-N traces to at least one REQ-N
[x] CHECK-003: No orphan ID references
```

**Fail example:**
```
QA Results: FAILED

[x] CHECK-001: Every entry has a DES-N identifier
[ ] CHECK-002: Every DES-N traces to at least one REQ-N
    - DES-003 has no traces
    - DES-007 has no traces
[x] CHECK-003: No orphan ID references
```

**Design rationale:**
- Always show full checklist so user sees what was validated
- Failed checks include specific details (which IDs, what's wrong)
- Warnings show as passed with note: `[x] CHECK-003: ... (warning: found 1 unused ID)`

**Traces to:** REQ-005

---

### DES-004: QA Context Input

Orchestrator provides QA skill with three inputs via context file:

```toml
[inputs]
producer_skill_path = "skills/design-interview-producer/SKILL.md"
producer_yield_path = ".projctl/yields/design-producer-yield.toml"
artifact_paths = ["docs/design.md"]
```

**Design rationale:**
- `producer_skill_path`: QA extracts contract from this file
- `producer_yield_path`: QA reads what producer claims it did
- `artifact_paths`: QA validates these files against contract

**Traces to:** REQ-005, REQ-010

---

### DES-005: QA Yield Types

QA skill yields one of four types based on validation results:

| Condition | Yield Type |
|-----------|------------|
| All checks pass | `approved` |
| Check failures that producer can fix | `improvement-request` |
| Problem in upstream phase artifact | `escalate-phase` |
| Cannot resolve without user | `escalate-user` |

**approved payload:**
```toml
[payload]
reviewed_artifact = "docs/design.md"
checklist = [
    { id = "CHECK-001", description = "Every entry has DES-N ID", passed = true },
    { id = "CHECK-002", description = "Traces to REQ-N", passed = true }
]
```

**improvement-request payload:**
```toml
[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "CHECK-002: DES-003 has no traces",
    "CHECK-002: DES-007 has no traces"
]
```

**Traces to:** REQ-005

---

### DES-006: Error Handling - Malformed Yield

When producer yield is invalid (bad TOML, missing required fields):

- QA yields `improvement-request`
- Issues list contains parse error details
- Producer receives feedback and can fix

```toml
[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "Yield parse error: missing required field 'artifact' in [payload]",
    "Line 5: invalid TOML syntax"
]
```

**Traces to:** REQ-005

---

### DES-007: Error Handling - Missing Artifacts

When artifact files don't exist:

- QA yields `improvement-request`
- Issues list contains missing file paths
- Producer can create the missing files

```toml
[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "Missing artifact: docs/design.md (file not found)"
]
```

**Traces to:** REQ-005

---

### DES-008: Error Handling - Missing Contract

When producer SKILL.md has no Contract section:

- QA falls back to reading entire SKILL.md
- QA extracts implicit requirements from prose (best effort)
- QA logs warning that producer should add contract section
- Validation continues with extracted requirements

**Fallback behavior:**
1. Search SKILL.md for structured patterns (checklists, tables, "must" statements)
2. Convert found patterns to implicit checks
3. Validate artifact against implicit checks
4. Include warning in output: "Warning: No contract section found, using prose extraction"

**Traces to:** REQ-005, REQ-011

---

### DES-009: Error Handling - Unreadable SKILL.md

When producer SKILL.md cannot be read (file not found, permissions):

- QA yields `error` type (not improvement-request)
- Cannot validate without contract
- Orchestrator must resolve before QA can proceed

```toml
[yield]
type = "error"

[payload]
error = "Cannot read producer SKILL.md"
details = "File not found: skills/design-interview-producer/SKILL.md"
recoverable = false
```

**Traces to:** REQ-005

---

### DES-010: Escalation to Upstream Phase

When QA discovers problem in upstream artifact (not current producer's fault):

- QA yields `escalate-phase`
- Includes proposed changes for upstream phase
- Orchestrator routes back to correct phase

**Example: Design QA finds missing requirement**
```toml
[yield]
type = "escalate-phase"

[payload.escalation]
from_phase = "design"
to_phase = "pm"
reason = "gap"

[payload.issue]
summary = "Design references capability not in requirements"
context = "DES-005 describes error recovery but no REQ addresses error handling"

[[payload.proposed_changes.requirements]]
action = "add"
id = "REQ-012"
title = "Error Recovery"
content = "System must provide clear error messages when validation fails"
```

**Traces to:** REQ-005

---

### DES-011: Escalation to User

When QA cannot resolve conflict or ambiguity:

- QA yields `escalate-user`
- Presents question with options
- Orchestrator prompts user, resumes with answer

**Example: Conflicting requirements**
```toml
[yield]
type = "escalate-user"

[payload]
reason = "Conflicting traces"
context = "DES-003 traces to both REQ-002 and REQ-005, which contradict each other"
question = "Which requirement takes priority?"
options = ["REQ-002 (offline-first)", "REQ-005 (real-time sync)", "Both with user toggle"]
```

**Traces to:** REQ-005

---

### DES-012: Iteration Limits

QA tracks producer-QA iterations to prevent infinite loops:

- Maximum 3 iterations per producer-QA pair
- After max iterations with issues remaining: yield `escalate-user`
- Iteration count tracked in yield context

```toml
[context]
iteration = 3
max_iterations = 3
```

**Traces to:** REQ-005

---

### DES-013: Single QA Skill Invocation

User invokes QA with producer name:

```
/qa design-interview-producer
```

Orchestrator resolves this to:
1. Find producer SKILL.md at `skills/design-interview-producer/SKILL.md`
2. Find producer's most recent yield
3. Find artifact paths from yield
4. Pass all three to universal QA skill

**Traces to:** REQ-005, REQ-010

---

## Summary

| Decision | Choice |
|----------|--------|
| Contract format | Flat YAML, no versions |
| Contract location | `## Contract` section in producer SKILL.md |
| QA output | Full checklist always |
| Malformed yield | `improvement-request` with parse errors |
| Missing artifacts | `improvement-request` with file list |
| Missing contract | Prose fallback with warning |
| Unreadable SKILL.md | `error` (cannot proceed) |
| Upstream issues | `escalate-phase` with proposed changes |
| Unresolvable | `escalate-user` with options |
| Max iterations | 3, then escalate to user |
