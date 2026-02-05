---
name: qa
description: Universal QA skill that validates any producer against its SKILL.md contract
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
role: qa
---

# Universal QA Skill

Validates any producer's output against the contract defined in its SKILL.md file.

**Contract Standard:** See [CONTRACT.md](../shared/CONTRACT.md)

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Pattern | LOAD → VALIDATE → RETURN |
| Input | Producer SKILL.md path, producer yield, artifact paths |
| Output | Yield with approval or issues |
| Yields | `approved`, `improvement-request`, `escalate-phase`, `escalate-user`, `error` |

---

## Workflow

### 1. LOAD Phase

Load and parse all inputs needed for validation:

1. **Read context file** containing:
   ```toml
   [inputs]
   producer_skill_path = "skills/design-interview-producer/SKILL.md"
   producer_yield_path = ".projctl/yields/design-producer-yield.toml"
   artifact_paths = ["docs/design.md"]

   [context]
   iteration = 1
   max_iterations = 3
   ```

2. **Extract contract from producer SKILL.md:**
   - Search for `## Contract` section
   - Extract YAML code block immediately following the heading
   - Parse YAML to get `outputs`, `traces_to`, `checks`

3. **Handle missing contract (fallback to prose):**
   - If no `## Contract` section found, scan entire SKILL.md
   - Extract implicit checks from prose patterns:
     - Checklists (`- [ ]` items)
     - "Must" statements
     - Validation tables
   - Log warning: "No contract section found, using prose extraction"

4. **Read producer yield:**
   - Parse TOML yield file
   - Extract artifact path(s) from `[payload]`
   - Handle malformed yield → yield `improvement-request` with parse error

5. **Read artifacts:**
   - Load each file from `artifact_paths`
   - Handle missing artifacts → yield `improvement-request` with file list

6. **Handle unreadable producer SKILL.md:**
   - If cannot read producer SKILL.md → yield `error` (cannot proceed)

---

### 2. VALIDATE Phase

Execute each check from the contract against the artifacts:

1. **For each check in `contract.checks`:**
   - Evaluate check against artifact content
   - Record result: `passed: true/false`
   - If failed, capture specific details (which IDs, what's wrong)

2. **Common check patterns:**

   | Check Type | How to Validate |
   |------------|-----------------|
   | "Every entry has X-N ID" | Scan for `### X-NNN:` pattern |
   | "Traces to upstream" | Look for `**Traces to:**` fields |
   | "No orphan references" | Cross-reference all ID mentions |
   | "Content describes X" | Pattern match for prohibited terms |

3. **Classify failures by severity:**
   - `error` failures → QA fails
   - `warning` failures → QA passes with notes

4. **Compile results:**
   ```
   results = [
     { id: "CHECK-001", description: "...", passed: true },
     { id: "CHECK-002", description: "...", passed: false, details: ["DES-003 missing trace"] }
   ]
   ```

---

### 3. RETURN Phase

Yield result based on validation findings:

#### Decision Tree

```
Has error-severity failures?
├─ YES → Can producer fix them?
│        ├─ YES → yield `improvement-request`
│        └─ NO → Is issue in upstream artifact?
│                ├─ YES → yield `escalate-phase`
│                └─ NO → yield `escalate-user`
└─ NO → yield `approved` (with warnings if any)
```

#### Iteration Tracking

Check iteration count before yielding `improvement-request`:

```
if iteration >= max_iterations (3):
    yield `escalate-user` with:
        reason = "Max iterations reached"
        context = "Issues remain after 3 attempts"
        question = "How should we proceed?"
        options = ["Approve with caveats", "Manual intervention", "Skip this producer"]
```

---

## Output Format

Full checklist display per DES-003:

**Pass example:**
```
QA Results: PASSED

[x] CHECK-001: Every entry has DES-N identifier
[x] CHECK-002: Every DES-N traces to at least one REQ-N
[x] CHECK-003: No orphan ID references (warning: 1 unused ID found)
```

**Fail example:**
```
QA Results: FAILED

[x] CHECK-001: Every entry has DES-N identifier
[ ] CHECK-002: Every DES-N traces to at least one REQ-N
    - DES-003 has no traces
    - DES-007 has no traces
[x] CHECK-003: No orphan ID references
```

---

## Yield Examples

### approved

All checks pass (or only warnings):

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/design.md"
checklist = [
    { id = "CHECK-001", description = "Every entry has DES-N ID", passed = true },
    { id = "CHECK-002", description = "Traces to REQ-N", passed = true },
    { id = "CHECK-003", description = "No orphan references", passed = true, note = "1 unused ID" }
]

[context]
phase = "design"
role = "qa"
iteration = 1
producer = "design-interview-producer"
```

### improvement-request

Producer can fix the issues:

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "CHECK-002: DES-003 has no traces",
    "CHECK-002: DES-007 has no traces"
]

[context]
phase = "design"
role = "qa"
iteration = 2
max_iterations = 3
```

### improvement-request (malformed yield)

Producer yield has parse errors:

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:06:00Z

[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "Yield parse error: missing required field 'artifact' in [payload]",
    "Line 5: invalid TOML syntax"
]

[context]
phase = "design"
role = "qa"
iteration = 1
```

### improvement-request (missing artifacts)

Artifact file not found:

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:07:00Z

[payload]
from_agent = "qa"
to_agent = "design-interview-producer"
issues = [
    "Missing artifact: docs/design.md (file not found)"
]

[context]
phase = "design"
role = "qa"
iteration = 1
```

### escalate-phase

Problem in upstream artifact:

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

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

[context]
phase = "design"
role = "qa"
escalating = true
```

### escalate-user

Cannot resolve without user input (or max iterations reached):

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Max iterations reached"
context = "Issues remain after 3 QA iterations"
question = "How should we proceed with unresolved issues?"
options = ["Approve with caveats noted", "Manual intervention required", "Skip this phase"]

[context]
phase = "design"
role = "qa"
iteration = 3
max_iterations = 3
```

### error

Cannot proceed (unreadable producer SKILL.md):

```toml
[yield]
type = "error"
timestamp = 2026-02-02T12:20:00Z

[payload]
error = "Cannot read producer SKILL.md"
details = "File not found: skills/design-interview-producer/SKILL.md"
recoverable = false

[context]
role = "qa"
```

---

## Error Handling Summary

| Condition | Yield Type | Rationale |
|-----------|------------|-----------|
| Malformed producer yield | `improvement-request` | Producer can fix their yield |
| Missing artifact files | `improvement-request` | Producer can create missing files |
| No contract section | Continue with fallback | Graceful degradation |
| Unreadable producer SKILL.md | `error` | Cannot validate without contract |
| Check failures (error severity) | `improvement-request` | Producer can fix issues |
| Upstream artifact problem | `escalate-phase` | Not producer's fault |
| Unresolvable conflict | `escalate-user` | Need human decision |
| Max iterations reached | `escalate-user` | Prevent infinite loops |

---

## Contract Extraction Algorithm

Per ARCH-021, extract contract from producer SKILL.md:

```
1. Read producer SKILL.md content
2. Search for "## Contract" heading (case-insensitive)
3. If found:
   a. Find next fenced code block (```yaml ... ```)
   b. Parse YAML content
   c. Validate against CONTRACT.md schema
   d. Return contract object
4. If not found (fallback per ARCH-024):
   a. Scan for checklist patterns: "- [ ]" lines
   b. Scan for "must" statements in prose
   c. Convert to implicit checks with severity: warning
   d. Log warning about missing contract
   e. Return implicit contract
```

---

## Related Documents

- **CONTRACT.md**: Contract format specification
- **YIELD.md**: Yield protocol for all skills
- **DES-003**: QA output format (full checklist)
- **DES-004**: QA context input schema
- **DES-005**: QA yield types
- **DES-006**: Malformed yield handling
- **DES-007**: Missing artifact handling
- **DES-009**: Unreadable SKILL.md handling
- **ARCH-021**: Contract extraction algorithm
- **ARCH-024**: Prose fallback behavior
- **ARCH-028**: Iteration tracking (max 3)
