# Producer Skill Template

Template for creating producer skills that follow the GATHER → SYNTHESIZE → PRODUCE pattern.

## Frontmatter

```yaml
---
name: <phase>-<variant>-producer
description: <Brief description of what this producer creates>
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: <pm | design | arch | breakdown | doc | tdd-red | tdd-green | tdd-refactor | alignment | retro | summary>
variant: <interview | infer>  # Optional, omit if single variant
---
```

## Workflow Pattern

Producer skills follow the GATHER → SYNTHESIZE → PRODUCE pattern:

### 1. GATHER

Collect information needed for the task:
- Read context file for inputs and previous phase data
- If information is missing, yield `need-context` with queries
- If user input is needed, yield `need-user-input` with question
- If decision is needed, yield `need-decision` with options

```markdown
## GATHER Phase

1. Read context from `[inputs]` section
2. Check for `[query_results]` (resuming after need-context)
3. If missing information:
   - Yield `need-context` with specific queries
   - OR yield `need-user-input` for interview questions
4. Proceed to SYNTHESIZE when sufficient information gathered
```

### 2. SYNTHESIZE

Process gathered information:
- Analyze patterns and relationships
- Identify conflicts or gaps
- Structure findings for output

```markdown
## SYNTHESIZE Phase

1. Analyze gathered information
2. Identify key decisions and their rationale
3. Check for conflicts with existing artifacts
4. If blocked, yield `blocked` with details
5. Prepare structured output
```

### 3. PRODUCE

Create the artifact:
- Write output file with proper IDs (REQ-N, DES-N, ARCH-N, TASK-N)
- Include traceability links
- Yield `complete` with artifact details

```markdown
## PRODUCE Phase

1. Generate artifact with proper ID format
2. Include `**Traces to:**` links to upstream artifacts
3. Write to configured path from context
4. Yield `complete` with artifact path and IDs created
```

## Yield Format

See [YIELD.md](./YIELD.md) for full protocol specification.

Producer skills can yield:

| Type | When to Use |
|------|-------------|
| `complete` | Work finished, artifact created |
| `need-user-input` | Need answer from user (interview mode) |
| `need-context` | Need files, memory, or exploration |
| `need-decision` | Multiple valid approaches, user must choose |
| `need-agent` | Need another agent's work |
| `blocked` | Cannot proceed without external action |
| `error` | Something failed (retryable) |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

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

### Need-Context Yield Example

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "file"
path = "docs/architecture.md"

[[payload.queries]]
type = "semantic"
question = "How is error handling implemented?"

[context]
phase = "pm"
subphase = "GATHER"
awaiting = "context-results"
```

## Context Reading

On invocation, read context from path provided by orchestrator:

```markdown
1. Read context file at `<project>/.claude/context/<skill>-context.toml`
2. Check `[query_results]` - if present, resuming after need-context
3. Extract mode from `invocation.mode`:
   - `interview`: Interactive Q&A flow
   - `infer`: Analysis-based generation
4. Read artifact paths from `[config]`
5. Write yield to `[output].yield_path`
```

## ID Format

Each phase produces artifacts with specific ID prefixes:

| Phase | ID Format | Example |
|-------|-----------|---------|
| pm | REQ-N | REQ-1, REQ-2 |
| design | DES-N | DES-1, DES-2 |
| arch | ARCH-N | ARCH-1, ARCH-2 |
| breakdown | TASK-N | TASK-1, TASK-2 |

## Traceability

All IDs must include `**Traces to:**` linking to upstream artifacts:

```markdown
### REQ-1: User Authentication

Users can log in with email/password.

**Traces to:** ISSUE-001
```

## Interview vs Infer Variants

| Variant | Behavior |
|---------|----------|
| `interview` | Interactive Q&A, yields `need-user-input` frequently |
| `infer` | Analyzes existing code/docs, yields `need-context` for exploration |

Both variants produce the same artifact format - they differ in how they gather information.
