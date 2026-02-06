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
- Read context from spawn prompt and referenced files
- If information is missing, read files directly (Glob, Grep, Read tools)
- If user input is needed, use `AskUserQuestion`
- If decision is needed, present options via `AskUserQuestion`

```markdown
## GATHER Phase

1. Read context from spawn prompt (project info, artifact paths, prior outputs)
2. Read referenced files directly
3. If missing information:
   - Read additional files using Glob, Grep, Read tools
   - OR use `AskUserQuestion` for interview questions
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
4. If blocked, send message to lead describing the blocker
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
3. If any inferred items exist: use `AskUserQuestion` to present them for accept/reject
4. Wait for user accept/reject decisions
5. Drop rejected items, keep accepted items
6. Proceed to PRODUCE with only explicit + accepted items

### 3. PRODUCE

Create the artifact:
- Write output file with proper IDs (REQ-N, DES-N, ARCH-N, TASK-N)
- Include traceability links
- Send completion message to team lead

```markdown
## PRODUCE Phase

1. Generate artifact with proper ID format
2. Include `**Traces to:**` links to upstream artifacts
3. Write to configured path from context
4. Send completion message to lead with artifact path and IDs created
```

## Receiving Context

Context arrives in your spawn prompt from the team lead. It includes:
- Project name and issue ID
- Current phase
- Artifact paths (where to write output)
- Prior phase outputs (upstream artifacts to read)
- Territory map summary and memory query results

Read referenced files directly.

## User Interaction

Use `AskUserQuestion` directly for:
- Interview questions
- CLASSIFY phase (presenting inferred items for accept/reject)
- Conflict resolution (need-decision scenarios)

## Reporting Results

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
| `interview` | Interactive Q&A, uses `AskUserQuestion` frequently |
| `infer` | Analyzes existing code/docs, reads files for exploration |

Both variants produce the same artifact format - they differ in how they gather information.
