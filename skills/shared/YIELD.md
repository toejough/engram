# Skill Yield Protocol

Standard format for yield files written by skills for the orchestrator.

**Reference:** See `docs/orchestration-system.md` Section 3 for full protocol specification.

---

## Purpose

Yield files provide the orchestrator with:
1. **Completion status** - what happened
2. **Type-specific payload** - data for next action
3. **Context for resumption** - state to continue later

---

## Yield Types

### Producer Yield Types

| Type | Meaning | Orchestrator Action |
|------|---------|---------------------|
| `complete` | Work finished | Advance to QA or next phase |
| `need-user-input` | Question for user | Prompt user, resume with answer |
| `need-context` | Need information | Run queries, resume with results |
| `need-decision` | Ambiguous choice | Present options, resume with choice |
| `need-agent` | Need another agent | Spawn agent, resume with result |
| `blocked` | Cannot proceed | Present blocker, await resolution |
| `error` | Something failed | Retry (max 3x) or escalate |

### QA Yield Types

| Type | Meaning | Orchestrator Action |
|------|---------|---------------------|
| `approved` | Work passes QA | Advance to next phase |
| `improvement-request` | Needs fixes | Resume producer with feedback |
| `escalate-phase` | Prior phase issue | Return to prior phase with proposed changes |
| `escalate-user` | Cannot resolve | Present to user |

---

## Yield Format

All yields use TOML format written to the path specified in context (`output.yield_path`).

### Base Structure

```toml
[yield]
type = "<yield-type>"
timestamp = 2026-02-02T10:30:00Z

[payload]
# Type-specific fields

[context]
# State for resumption
phase = "pm"
iteration = 1
```

---

## Producer Yield Examples

### complete

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
choice = "Focus on CLI only"
reason = "User's immediate need"
alternatives = ["Include GUI", "API first"]

[[payload.learnings]]
content = "User prioritizes simplicity over flexibility"

[context]
phase = "pm"
subphase = "complete"
```

### need-user-input

```toml
[yield]
type = "need-user-input"
timestamp = 2026-02-02T10:30:00Z

[payload]
question = "What problem are you trying to solve?"
context = "Starting problem discovery phase"

[context]
phase = "pm"
subphase = "PROBLEM"
awaiting = "user-response"
```

### need-user-input (inferred specifications)

When a producer infers specifications not explicitly requested by the user, it yields `need-user-input` with `payload.inferred = true`. The orchestrator presents these for user accept/reject before the producer includes them in the artifact.

**When to use:** During SYNTHESIZE phase, after classifying specifications as explicit or inferred per PRODUCER-TEMPLATE.md guidelines. Yield before PRODUCE phase so rejected items are excluded from the artifact.

```toml
[yield]
type = "need-user-input"
timestamp = 2026-02-05T12:00:00Z

[payload]
inferred = true
question = "The following specifications were inferred and not explicitly requested. Accept or reject each."

[[payload.items]]
specification = "REQ-X: Input validation for empty strings"
reasoning = "Edge case: empty input could cause downstream errors"
source = "edge-case"

[[payload.items]]
specification = "REQ-Y: Rate limiting on API calls"
reasoning = "Implicit need: without rate limiting, external API costs could spike"
source = "implicit-need"

[context]
phase = "pm"
subphase = "SYNTHESIZE"
awaiting = "user-response"
```

**Payload fields (when `inferred = true`):**

| Field | Type | Description |
|-------|------|-------------|
| `inferred` | bool | Must be `true`. Distinguishes from regular questions. |
| `question` | string | Prompt text for the user. |
| `items` | array | List of inferred specifications. |
| `items[].specification` | string | The inferred specification text. |
| `items[].reasoning` | string | Why this was inferred. |
| `items[].source` | string | Category: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment` |

**Orchestrator response format (written to `[query_results]`):**

```toml
[query_results]
[[query_results.inferred_decisions]]
specification = "REQ-X: Input validation for empty strings"
accepted = true

[[query_results.inferred_decisions]]
specification = "REQ-Y: Rate limiting on API calls"
accepted = false
```

**Backward compatibility:** The `inferred` field is optional. Existing `need-user-input` yields without it continue to work as regular questions.

### need-context

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "file"
path = "docs/requirements.md"

[[payload.queries]]
type = "memory"
query = "caching patterns"

[[payload.queries]]
type = "territory"
scope = "tests"

[[payload.queries]]
type = "web"
url = "https://example.com/docs"
prompt = "Extract the API format"

[[payload.queries]]
type = "semantic"
question = "How does authentication work in this codebase?"

[context]
phase = "pm"
subphase = "GATHER"
awaiting = "context-results"
```

### need-decision

```toml
[yield]
type = "need-decision"
timestamp = 2026-02-02T10:40:00Z

[payload]
question = "Which authentication method should we use?"
context = "Multiple valid approaches available"
options = [
    { label = "OAuth 2.0", description = "Industry standard, complex setup" },
    { label = "API keys", description = "Simple, less secure" },
    { label = "JWT", description = "Stateless, good for APIs" }
]
recommendation = "JWT"
recommendation_reason = "Best fit for CLI tool with API backend"

[context]
phase = "arch"
subphase = "DECISION"
awaiting = "user-choice"
```

### need-agent

```toml
[yield]
type = "need-agent"
timestamp = 2026-02-02T10:45:00Z

[payload]
agent = "context-explorer"
reason = "Need semantic code analysis"
input = { question = "How is error handling implemented?" }

[context]
phase = "arch"
subphase = "GATHER"
awaiting = "agent-result"
```

### blocked

```toml
[yield]
type = "blocked"
timestamp = 2026-02-02T10:50:00Z

[payload]
blocker = "Missing API credentials"
details = "Cannot test authentication without valid credentials"
suggested_resolution = "User provides API_KEY environment variable"

[context]
phase = "implementation"
task = "TASK-5"
awaiting = "blocker-resolution"
```

### error

```toml
[yield]
type = "error"
timestamp = 2026-02-02T10:55:00Z

[payload]
error = "Failed to parse existing requirements.md"
details = "Line 45: Invalid markdown header format"
recoverable = true
retry_count = 1

[context]
phase = "pm"
subphase = "GATHER"
last_action = "read-requirements"
```

---

## QA Yield Examples

### approved

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/requirements.md"
checklist = [
    { item = "All requirements have IDs", passed = true },
    { item = "Acceptance criteria are testable", passed = true },
    { item = "Traces to issue", passed = true }
]

[context]
phase = "pm"
role = "qa"
iteration = 1
```

### improvement-request

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "pm-qa"
to_agent = "pm-interview-producer"
iteration = 2
issues = [
    "REQ-3 acceptance criteria are not measurable",
    "REQ-5 missing edge case for empty input"
]

[context]
phase = "pm"
role = "qa"
iteration = 2
max_iterations = 3
```

### escalate-phase

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "design"
to_phase = "pm"
reason = "gap"  # error | gap | conflict

[payload.issue]
summary = "Parallelism not addressed in requirements"
context = "Design phase discovered need for context exploration"

[[payload.proposed_changes.requirements]]
action = "add"
id = "REQ-10"
title = "Context Exploration via Yield"
content = "Producer skills can yield need-context..."

[[payload.proposed_changes.source_docs]]
file = "docs/orchestration-system.md"
section = "3.2 Yield Types"
change = "Add need-context yield type"

[context]
phase = "design"
role = "qa"
escalating = true
```

### escalate-user

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Cannot resolve conflict between requirements"
context = "REQ-3 and REQ-7 appear contradictory"
question = "Should offline mode take priority over real-time sync?"
options = ["Offline first", "Real-time first", "User configurable"]

[context]
phase = "design"
role = "qa"
escalating = true
```

---

## Context Serialization

The `[context]` section enables resumption after yields:

| Field | Purpose |
|-------|---------|
| `phase` | Current workflow phase |
| `subphase` | Step within phase (GATHER, SYNTHESIZE, PRODUCE) |
| `iteration` | Pair loop iteration count |
| `task` | Current TASK-N ID (if in implementation) |
| `awaiting` | What we're waiting for |
| `role` | producer or qa |

Skills read context from input, modify during work, write to yield for resumption.

---

## Query Types (for need-context)

| Type | Parameters | What it fetches |
|------|------------|-----------------|
| `file` | `path` | File contents |
| `memory` | `query` | ONNX semantic memory results |
| `territory` | `scope` | Codebase structure map |
| `web` | `url`, `prompt` | URL content, interpreted by prompt |
| `semantic` | `question` | Answer about codebase (LLM exploration) |

---

## Validation

Yield files can be validated with:

```bash
projctl yield validate <path-to-yield.toml>
```

Checks:
- Required fields present (`[yield].type`, `timestamp`)
- Type is valid
- Payload matches type schema
- Context section present for resumable yields
