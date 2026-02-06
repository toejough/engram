# ISSUE-53: Universal QA Skill Design

Design decisions for replacing 13 phase-specific QA skills with one universal QA skill using team messaging.

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

Team lead provides QA teammate with context via spawn prompt:

```
Invoke the /qa skill to validate the producer's output.

Producer SKILL.md: skills/design-interview-producer/SKILL.md
Artifact paths: docs/design.md
Iteration: 1

Context:
The design-interview-producer completed and reported:
- Created DES-001 through DES-012
- Modified: docs/design.md
```

**Design rationale:**
- QA reads producer SKILL.md to extract contract
- QA reads artifact files directly using Read tool
- QA validates artifacts against contract checks

**Traces to:** REQ-005, REQ-010

---

### DES-005: QA Message Types

QA teammate sends one of four message patterns to team lead based on validation results:

| Condition | Message Pattern |
|-----------|-----------------|
| All checks pass | `approved` |
| Check failures that producer can fix | `improvement-request: <issues>` |
| Problem in upstream phase artifact | `escalate-phase: <reason>` |
| Cannot resolve without user | `escalate-user: <reason>` |

**approved message:**
```
approved

Reviewed artifact: docs/design.md
Checklist:
[x] CHECK-001: Every entry has DES-N ID
[x] CHECK-002: Traces to REQ-N
```

**improvement-request message:**
```
improvement-request: missing traces

Issues to fix:
- CHECK-002: DES-003 has no traces
- CHECK-002: DES-007 has no traces
```

**Traces to:** REQ-005

---

### DES-006: Error Handling - Invalid Producer Output

When producer's completion message is missing expected information:

- QA sends `improvement-request: incomplete producer output`
- Message includes what information is missing
- Team lead spawns new producer with feedback

**Example message:**
```
improvement-request: incomplete producer output

Missing information:
- No artifact paths provided in completion message
- No IDs reported for created items
```

**Traces to:** REQ-005

---

### DES-007: Error Handling - Missing Artifacts

When artifact files don't exist:

- QA sends `improvement-request: missing artifacts`
- Message includes missing file paths
- Team lead spawns new producer to create the files

**Example message:**
```
improvement-request: missing artifacts

Missing files:
- docs/design.md (file not found)
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

- QA sends `error: cannot read producer SKILL.md`
- Cannot validate without contract
- Team lead must resolve before QA can proceed

**Example message:**
```
error: cannot read producer SKILL.md

Details: File not found: skills/design-interview-producer/SKILL.md
This is not recoverable - team lead must fix the path.
```

**Traces to:** REQ-005

---

### DES-010: Escalation to Upstream Phase

When QA discovers problem in upstream artifact (not current producer's fault):

- QA sends `escalate-phase: <reason>`
- Includes proposed changes for upstream phase
- Team lead routes back to correct phase

**Example: Design QA finds missing requirement**
```
escalate-phase: gap in upstream requirements

From phase: design
To phase: pm
Reason: gap

Issue:
Design references capability not in requirements.
DES-005 describes error recovery but no REQ addresses error handling.

Proposed change:
Add REQ-012: Error Recovery
"System must provide clear error messages when validation fails"
```

**Traces to:** REQ-005

---

### DES-011: Escalation to User

When QA cannot resolve conflict or ambiguity:

- QA sends `escalate-user: <reason>`
- Presents question with options
- Team lead prompts user, sends answer back to QA

**Example: Conflicting requirements**
```
escalate-user: conflicting traces

Reason: Conflicting traces
Context: DES-003 traces to both REQ-002 and REQ-005, which contradict each other.

Question: Which requirement takes priority?
Options:
1. REQ-002 (offline-first)
2. REQ-005 (real-time sync)
3. Both with user toggle
```

**Traces to:** REQ-005

---

### DES-012: Iteration Limits

QA tracks producer-QA iterations to prevent infinite loops:

- Maximum 3 iterations per producer-QA pair
- After max iterations with issues remaining: send `escalate-user: max iterations reached`
- Iteration count tracked in team lead's PairState

**Example message on max iterations:**
```
escalate-user: max iterations reached

Iteration 3 of 3 reached with remaining issues:
- CHECK-002: DES-003 still has no traces
- CHECK-005: Missing design rationale

User decision needed: accept as-is, extend iterations, or modify requirements?
```

**Traces to:** REQ-005

---

### DES-013: Single QA Skill Invocation

User invokes QA with producer name:

```
/qa design-interview-producer
```

Team lead resolves this to:
1. Find producer SKILL.md at `skills/design-interview-producer/SKILL.md`
2. Find producer's most recent completion message for artifact paths
3. Spawn QA teammate with producer SKILL.md path and artifact paths

**Traces to:** REQ-005, REQ-010

---

## ISSUE-56: Inferred Specification Warning Design

Design decisions for how producers flag inferred specifications via AskUserQuestion and how the team lead presents them for user approval.

---

### DES-014: Inferred Message Format

Inferred specifications use AskUserQuestion with an `inferred = true` flag in the options. This distinguishes inferred items from regular interview questions.

**AskUserQuestion structure:**

The producer teammate uses AskUserQuestion to present inferred items:

```
AskUserQuestion with multiSelect: true
Question: "The following specifications were inferred. Accept or reject each:"
Options:
  1. REQ-X: Input validation for empty strings
     (Reasoning: Edge case - empty input could cause downstream errors, Source: best-practice)
  2. REQ-Y: Rate limiting on API calls
     (Reasoning: Implicit need - without rate limiting, external API costs could spike, Source: edge-case)
```

**Metadata fields:**
- `inferred = true`: Signals this is inference confirmation
- Each option includes: specification text, reasoning, source category
- `source` values: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment`

**Traces to:** REQ-012

---

### DES-015: Team Lead Relay of Inferred Items

When a producer teammate sends inferred items via AskUserQuestion, the team lead may relay the question to the user or handle it directly based on context.

**User presentation format (via team lead relay):**
```
The producer inferred the following specifications that were not
explicitly requested. Please accept or reject each:

1. REQ-X: Input validation for empty strings
   Reasoning: Edge case - empty input could cause downstream errors
   Source: best-practice

2. REQ-Y: Rate limiting on API calls
   Reasoning: Implicit need - without rate limiting, external API costs could spike
   Source: edge-case

Select which items to accept (e.g., "1, 2" for both, "1" for first only):
```

**User response handling:**
- Selections captured via AskUserQuestion multiSelect
- Teammate receives user decisions and proceeds with only accepted + explicit items

**Traces to:** REQ-014

---

### DES-016: Producer Inference Detection Workflow

During the SYNTHESIZE phase, producers separate gathered information into two categories before producing artifacts:

1. **Explicit**: Directly traceable to user input, issue description, or gathered context
2. **Inferred**: Added by the producer based on professional judgment

**Workflow:**
1. Producer teammate completes GATHER phase (interview or context analysis)
2. During SYNTHESIZE, producer classifies each specification as explicit or inferred
3. If any inferred items exist, producer uses AskUserQuestion with `inferred = true` BEFORE producing the artifact
4. User responds with accepted items (via team lead relay or directly)
5. Producer receives user decisions
6. Producer produces artifact with only explicit + accepted items

**Traces to:** REQ-013, REQ-015

---

## Summary

| Decision | Choice |
|----------|--------|
| Contract format | Flat YAML, no versions |
| Contract location | `## Contract` section in producer SKILL.md |
| QA output | Full checklist always |
| Missing artifacts | `improvement-request: missing artifacts <list>` message |
| Missing contract | Prose fallback with warning |
| Unreadable SKILL.md | `error: cannot read producer SKILL.md` message |
| Upstream issues | `escalate-phase: <reason>` message with proposed changes |
| Unresolvable | `escalate-user: <reason>` message with options |
| Max iterations | 3, then escalate to user |
| Inferred spec format | AskUserQuestion with `inferred = true` metadata |
| Inferred presentation | Numbered list with reasoning, multiSelect for accept/reject |
| Inference detection | SYNTHESIZE phase classifies explicit vs inferred before producing |
