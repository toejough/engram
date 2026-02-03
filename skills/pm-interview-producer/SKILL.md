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

---

## Workflow

### 1. GATHER Phase

Collect requirements through structured interview:

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

### 2. SYNTHESIZE Phase

Process gathered interview responses:

1. Extract core requirements from user answers
2. Identify implicit requirements from context
3. Resolve conflicts between stated needs
4. Structure into user stories with acceptance criteria
5. Assign priorities (P0/P1/P2)
6. If blocked by contradictions, yield `blocked` with details

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

## Boundaries

| In Scope | Out of Scope |
|----------|--------------|
| Problem discovery | UI/UX design |
| User needs | Technology choices |
| Success criteria | Implementation details |
| Edge cases | Architecture decisions |

Out-of-scope topics are noted for downstream phases (Design, Architecture) and conversation redirects to problem discovery.
