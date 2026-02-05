---
name: pm-interview-producer
description: Gathers requirements via user interview, produces requirements.md with REQ-N IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: pm
variant: interview
---

# PM Interview Producer

Producer skill that gathers requirements through structured user interview and produces requirements.md with traceable REQ-N IDs.

**Pattern:** GATHER -> SYNTHESIZE -> PRODUCE (see [PRODUCER-TEMPLATE](../shared/PRODUCER-TEMPLATE.md))

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Problem Discovery First

PM phase focuses on **problem discovery** and **user needs**. Implementation details, UI/UX design, and technology choices belong in downstream phases.

**Do not** ask about or include:
- UI/UX design patterns or visual elements
- Technology choices (languages, frameworks, databases)
- Implementation strategies or algorithms
- Architecture decisions or system design
- File formats or data structures

**Do** focus on:
- Problem identification and pain points
- User personas and affected parties
- Current state assessment
- Desired outcomes and benefits
- Success criteria and measurement
- Constraints and limitations
- Edge cases and exceptional scenarios

---

## Workflow

### 1. GATHER Phase

Collect requirements through structured interview. Focus on what problems need solving and who they affect, not how to solve them.

1. Read context from `[inputs]` section for project info
2. Check `[query_results]` for previous responses (if resuming)
3. Yield `need-context` for existing docs (README, prior requirements, etc.)
4. Yield `need-user-input` with interview questions through phases:
   - **PROBLEM**: What's broken? Who's affected? Impact?
   - **CURRENT STATE**: How does it work today? Pain points?
   - **FUTURE STATE**: What should happen instead?
   - **SUCCESS CRITERIA**: How will we know it's working?
   - **EDGE CASES**: What could go wrong?
5. Accumulate responses until sufficient for synthesis

**Avoid asking about** implementation details like UI design, technology choices, or system architecture. These belong in Design and Architecture phases.

### 2. SYNTHESIZE Phase

Process gathered interview responses:

1. Extract core requirements from user answers
2. Identify implicit requirements from context
3. Resolve conflicts between stated needs
4. Structure into user stories with acceptance criteria
5. Assign priorities (P0/P1/P2)
6. If blocked by contradictions, yield `blocked` with details

### 2b. CLASSIFY Phase (Inference Detection)

Classify each planned requirement as explicit or inferred per [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md) inference guidelines.

1. For each requirement from SYNTHESIZE, determine if it was directly requested by the user or inferred
2. If any inferred requirements exist, yield `need-user-input` with `payload.inferred = true` (see [YIELD.md](../shared/YIELD.md))
3. Wait for user accept/reject decisions
4. Drop rejected items, proceed to PRODUCE with only explicit + accepted items

### 3. PRODUCE Phase

Generate requirements.md artifact:

1. Write requirements with REQ-N format:
   ```markdown
   ### REQ-1: Feature Name

   As a [persona], I want [capability], so that [benefit].

   **Acceptance Criteria:**
   - [ ] Criterion 1
   - [ ] Criterion 2

   **Priority:** P1

   **Traces to:** ISSUE-XXX
   ```
2. Include `**Traces to:**` links to upstream artifacts
3. Yield `complete` with artifact path and REQ IDs created

---

## Yield Types Used

| Yield Type | When Used |
|------------|-----------|
| `need-context` | Gather existing docs before interview |
| `need-user-input` | Each interview question |
| `need-user-input` (inferred) | Present inferred requirements for user accept/reject |
| `need-decision` | When user provides conflicting requirements |
| `blocked` | Cannot proceed without resolution |
| `complete` | requirements.md artifact produced |

### need-user-input Example

```toml
[yield]
type = "need-user-input"
timestamp = 2026-02-02T10:30:00Z

[payload]
question = "What problem are you trying to solve?"
context = "PROBLEM phase - identify core pain point"

[context]
phase = "pm"
subphase = "GATHER"
interview_phase = "PROBLEM"
awaiting = "user-response"
```

### need-context Example

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:25:00Z

[[payload.queries]]
type = "file"
path = "README.md"

[[payload.queries]]
type = "file"
path = "docs/requirements.md"

[context]
phase = "pm"
subphase = "GATHER"
awaiting = "context-results"
```

### complete Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:30:00Z

[payload]
artifact = "docs/requirements.md"
ids_created = ["REQ-1", "REQ-2", "REQ-3"]
files_modified = ["docs/requirements.md"]

[[payload.decisions]]
context = "Scope definition"
choice = "CLI only, no GUI"
reason = "User's immediate need"
alternatives = ["Include GUI", "API first"]

[context]
phase = "pm"
subphase = "complete"
```

---

## Output Format

**Artifact:** `docs/requirements.md` (or path from context config)

**ID Format:** REQ-N (REQ-1, REQ-2, etc.)

Each requirement includes:
- User story format
- Acceptance criteria (checkboxes)
- Priority (P0/P1/P2)
- Traceability to upstream issue

---

## Interview Phases

| Phase | Goal | Key Questions |
|-------|------|---------------|
| PROBLEM | Identify the pain | What's broken? Who's affected? Impact? |
| CURRENT STATE | Map the present | How does it work today? Pain points? |
| FUTURE STATE | Define success | What should happen instead? |
| SUCCESS CRITERIA | Make measurable | How will we know it's working? |
| EDGE CASES | Handle exceptions | What could go wrong? |

---

## Rules

| Rule | Action |
|------|--------|
| Problem discovery first | PM focuses on problem space and user needs, not solution implementation |
| Implementation details → Architecture | Do not ask about technology choices, algorithms, or system design |
| UI/UX details → Design | Do not ask about visual design, layouts, or interaction patterns |
| Missing context | Yield `need-context` to request existing docs |
| Conflicting needs | Yield `need-decision` with clarifying question |
| Every REQ-N | Must have acceptance criteria and trace to issue |
| Measurable criteria | Acceptance criteria must be testable and unambiguous |

## Boundaries

| In Scope | Out of Scope |
|----------|--------------|
| Problem discovery | UI/UX design patterns |
| User needs | Technology choices |
| Success criteria | Implementation details |
| Edge cases | Architecture decisions |
| Pain points | Visual design elements |
| Constraints | File formats or data structures |

Out-of-scope topics are noted for downstream phases (Design, Architecture) and conversation redirects to problem discovery.

---

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
      description: "Every requirement has REQ-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every REQ-N has acceptance criteria"
      severity: error

    - id: "CHECK-003"
      description: "Every REQ-N traces to the issue"
      severity: error

    - id: "CHECK-004"
      description: "Acceptance criteria are measurable and testable"
      severity: error

    - id: "CHECK-005"
      description: "No ambiguous language (should, may, might)"
      severity: error

    - id: "CHECK-006"
      description: "No conflicting requirements"
      severity: error

    - id: "CHECK-007"
      description: "Edge cases identified where applicable"
      severity: warning

    - id: "CHECK-008"
      description: "Dependencies between requirements documented"
      severity: warning
```
