---
name: intake-evaluator
description: Classifies user requests into workflow types (new, adopt, align, single-task)
context: fork
model: haiku
user-invocable: true
role: evaluator
---

# Intake Evaluator

Standalone skill that evaluates incoming user requests and classifies them into the appropriate workflow type. Outputs a yield protocol TOML with the classification. Escalates to user if classification is uncertain.

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

---

## Classification Types

| Type | Description | Signals |
|------|-------------|---------|
| `new` | New feature or project from scratch | "build X", "create Y", "new feature", no existing code mentioned |
| `adopt` | Take ownership of existing code | "adopt", "take over", "existing code", references to files that exist |
| `align` | Check alignment with existing artifacts | "align", "verify", "check if", "drift", references to requirements/design |
| `single-task` | One-off task, not a full project workflow | "fix this bug", "update this file", "refactor X", scoped to single change |

---

## Classification Criteria

Evaluate the request against these indicators:

### New Project Signals
- Request mentions building something from scratch
- No references to existing implementation
- Describes a problem that needs a solution designed
- Uses words like "create", "build", "implement", "new"

### Adopt Project Signals
- References existing code or codebase
- Mentions "take over", "adopt", "inherit"
- Wants to add project management to existing work
- Code exists but lacks requirements/design artifacts

### Alignment Check Signals
- Mentions verifying or checking something
- References drift between code and documentation
- Asks if implementation matches requirements
- Uses words like "verify", "check", "align", "validate"

### Single-Task Signals
- Scoped to a specific file or function
- Bug fix without broader context
- Small refactoring request
- "Just do X" without project framing
- Doesn't need full PM/design/arch workflow

---

## Confidence Handling

When classification is ambiguous or uncertain:

1. **High confidence (>80%)**: Yield `complete` with classification
2. **Medium confidence (50-80%)**: Yield `complete` but include `confidence` field
3. **Low confidence (<50%)**: Yield `need-decision` with options for user

Escalate to user when:
- Request contains signals for multiple categories
- Key context is missing to make determination
- Request is vague or could be interpreted multiple ways

---

## Workflow

1. Read the user's request from context
2. Analyze against classification criteria
3. Determine confidence level
4. If confident: yield `complete` with classification
5. If uncertain: yield `need-decision` with options

---

## Yield Types Used

| Yield Type | When Used |
|------------|-----------|
| `complete` | Classification determined with sufficient confidence |
| `need-decision` | Uncertain classification, need user to choose |

### complete Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
classification = "new"
confidence = 0.95
reasoning = "Request mentions building a new feature from scratch with no existing code references"

[[payload.signals]]
signal = "uses 'build' terminology"
weight = "strong"

[[payload.signals]]
signal = "no existing code referenced"
weight = "supporting"

[context]
phase = "intake"
subphase = "complete"
```

### complete Example (adopt)

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
classification = "adopt"
confidence = 0.88
reasoning = "User references existing codebase and wants to add structured project management"

[[payload.signals]]
signal = "mentions 'existing code'"
weight = "strong"

[[payload.signals]]
signal = "wants to add requirements to what exists"
weight = "strong"

[context]
phase = "intake"
subphase = "complete"
```

### complete Example (single-task)

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
classification = "single-task"
confidence = 0.92
reasoning = "Request is a scoped bug fix in a specific file, no project workflow needed"

[[payload.signals]]
signal = "references specific file"
weight = "strong"

[[payload.signals]]
signal = "uses 'fix' terminology"
weight = "strong"

[[payload.signals]]
signal = "no mention of requirements or design"
weight = "supporting"

[context]
phase = "intake"
subphase = "complete"
```

### need-decision Example (uncertain)

```toml
[yield]
type = "need-decision"
timestamp = 2026-02-02T10:30:00Z

[payload]
question = "How should I classify this request?"
context = "The request has signals for both 'new' and 'adopt' workflows"
options = [
    { label = "new", description = "Treat as new project - design from scratch" },
    { label = "adopt", description = "Treat as adoption - map existing code first" },
    { label = "single-task", description = "Treat as one-off task - no full workflow" }
]
recommendation = "adopt"
recommendation_reason = "References to existing files suggest adoption is more appropriate"

[[payload.ambiguous_signals]]
signal = "mentions 'new feature'"
leans_toward = "new"

[[payload.ambiguous_signals]]
signal = "references existing codebase"
leans_toward = "adopt"

[context]
phase = "intake"
subphase = "DECISION"
awaiting = "user-choice"
```

---

## Boundaries

| In Scope | Out of Scope |
|----------|--------------|
| Classify request type | Execute any workflow |
| Analyze request signals | Gather requirements |
| Escalate uncertainty | Make design decisions |
| Output classification | Modify any files |

This skill only classifies - it does not start any workflow. The orchestrator uses the classification to determine which workflow to invoke.
