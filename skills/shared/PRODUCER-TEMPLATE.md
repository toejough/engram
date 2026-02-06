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

### 2b. CLASSIFY (Inference Detection)

After SYNTHESIZE, classify each planned specification as explicit or inferred before producing the artifact. This ensures users are never surprised by unwanted scope additions.

**Definitions:**
- **Explicit**: Directly traceable to a specific user statement, issue description field, interview response, or gathered context document passage.
- **Inferred**: Added by the producer based on best practices, edge cases, implicit needs, or professional judgment. The user did not ask for it.

**Classification heuristic:**
- Can you point to a specific sentence the user wrote or said that requested this? -> **explicit**
- Did you add this because it seems like a good idea, covers an edge case, or follows best practices? -> **inferred**
- When in doubt: classify as **inferred** (conservative default)

**Examples:**

| Specification | Classification | Why |
|---------------|---------------|-----|
| "Support --help flag" (user said "build a CLI tool") | Explicit | Standard CLI expectation from user's request |
| "Validate input file exists before processing" | Inferred | Edge case not mentioned by user |
| "Add rate limiting to API calls" | Inferred | Best practice not requested |
| "Return error on empty input" (user said "handle errors gracefully") | Explicit | Directly follows from user's stated need |

**Workflow:**
1. Review each specification planned during SYNTHESIZE
2. Classify as explicit or inferred
3. If any inferred items exist: yield `need-user-input` with `payload.inferred = true` (see [YIELD.md](./YIELD.md))
4. Wait for user accept/reject decisions
5. Drop rejected items, keep accepted items
6. Proceed to PRODUCE with only explicit + accepted items

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

## Team Mode

When invoked as a teammate (via Task tool with team_name), use direct interaction instead of yield files:

### Receiving Context

Context arrives in your spawn prompt from the team lead. It includes:
- Project name and issue ID
- Current phase
- Artifact paths (where to write output)
- Prior phase outputs (upstream artifacts to read)
- Territory map summary and memory query results

Read referenced files directly — no TOML context files needed.

### User Interaction

Use `AskUserQuestion` directly for:
- Interview questions (no yield-resume relay)
- CLASSIFY phase (presenting inferred items for accept/reject)
- Conflict resolution (`need-decision` scenarios)

### Reporting Results

On completion, send a message to the team lead via `SendMessage` with:
- **Artifact path** (e.g., `docs/requirements.md`)
- **IDs created** (e.g., REQ-1, REQ-2, REQ-3)
- **Files modified**
- **Key decisions made**

On failure, send a message describing:
- What went wrong
- Whether it's recoverable (lead should retry) or needs escalation

### Example Completion Message

```
Complete: docs/requirements.md

IDs created: REQ-1, REQ-2, REQ-3
Files modified: docs/requirements.md
Key decisions: CLI only (no GUI), single-user scope

Traces validated: all REQ-N trace to ISSUE-42
```

---

## Legacy Mode (Yield Protocol)

When invoked via the Skill tool with TOML context files, use the yield protocol.

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

## Context Reading (Legacy)

On invocation via Skill tool, read context from path provided by orchestrator:

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

**Traces to:** ISSUE-1
```

## Interview vs Infer Variants

| Variant | Behavior |
|---------|----------|
| `interview` | Interactive Q&A, yields `need-user-input` frequently |
| `infer` | Analyzes existing code/docs, yields `need-context` for exploration |

Both variants produce the same artifact format - they differ in how they gather information.
