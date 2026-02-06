---
name: design-infer-producer
description: Infer design decisions from existing UI/UX analysis
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: design
variant: infer
---

# Design Infer Producer

Analyze existing UI/UX to infer design decisions. Produces design.md with DES-N IDs.

**Template:** [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)
**Yield Protocol:** [YIELD.md](../shared/YIELD.md)

## User Experience First

Design phase focuses on **user experience** and **interaction patterns**. Implementation details belong in Architecture phase.

**When inferring design decisions:**
- Extract user workflows and task flows
- Document screen layouts and navigation patterns
- Identify interaction patterns and affordances
- Note visual hierarchy and information architecture
- Document feedback mechanisms

**Avoid documenting:**
- File formats or data structures (belongs in Architecture)
- Validation logic or error handling mechanisms (belongs in Architecture)
- Internal APIs or system interfaces (belongs in Architecture)
- Algorithms or processing pipelines (belongs in Architecture)

---

## Purpose

Deduce design decisions from existing user interfaces without user interview. Used for:
- Adoption of existing projects
- Documenting implicit design patterns
- Creating design artifacts from implemented UI

---

## Workflow: GATHER → SYNTHESIZE → PRODUCE

### 1. GATHER

Collect information about existing UI/UX:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Check for `[query_results]` (resuming after need-context, legacy mode)
3. If missing interface information, yield `need-context`:

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:30:00Z

[[payload.queries]]
type = "file"
path = "cmd/app/main.go"  # CLI entry point

[[payload.queries]]
type = "semantic"
question = "What are the user-facing interfaces in this project?"

[[payload.queries]]
type = "territory"
scope = "ui"  # UI-related files

[context]
phase = "design"
subphase = "GATHER"
awaiting = "context-results"
```

**Sources to analyze:**
| Source | Extract |
|--------|---------|
| CLI --help output | Command structure, help text layout |
| Error messages | Error presentation patterns |
| Output formatting | Tables, JSON, colors, progress indicators |
| Interactive prompts | Question format, confirmations |
| Web UI screens | Layout patterns, component usage |
| API responses | Response structure, error format |

### 2. SYNTHESIZE

Process gathered UI/UX information:

1. Identify implicit design patterns
2. Categorize decisions (layout, interaction, feedback, etc.)
3. Map decisions to requirements (REQ-N)
4. Check for conflicts with existing design.md
5. If blocked, yield `blocked` with details

### 2b. CLASSIFY (Inference Detection)

Classify each planned design decision as explicit or inferred per [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md) inference guidelines.

1. For each design decision from SYNTHESIZE, determine if it was directly present in analyzed code/UI or inferred by the producer
2. If any inferred design decisions exist, yield `need-user-input` with `payload.inferred = true` (see [YIELD.md](../shared/YIELD.md))
3. Wait for user accept/reject decisions
4. Drop rejected items, proceed to PRODUCE with only explicit + accepted items

### 3. PRODUCE

Create design.md artifact:

1. Write DES-N entries with proper format:

```markdown
### DES-001: Decision Title

Description of the inferred design decision.

**Rationale:** Why this pattern was likely chosen.

**Traces to:** REQ-001, REQ-002
```

2. Include `**Traces to:**` links to requirements
3. Write to configured path from context
4. Send results to team lead via `SendMessage`:
   - Artifact path
   - DES IDs created
   - Files modified
   - Key decisions made
5. In legacy mode, yield `complete`:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:00:00Z

[payload]
artifact = "docs/design.md"
ids_created = ["DES-1", "DES-2", "DES-3"]
files_modified = ["docs/design.md"]

[[payload.decisions]]
context = "Output format inference"
choice = "Table-based CLI output"
reason = "Observed consistent table formatting in existing --help output"
alternatives = ["JSON output", "Plain text"]

[context]
phase = "design"
subphase = "complete"
```

---

## Input Context

In team mode, context is provided via the spawn prompt. In legacy mode, read from `<project>/.claude/context/design-infer-producer-context.toml`:

```toml
[invocation]
mode = "infer"
timestamp = 2026-02-02T10:00:00Z

[inputs]
requirements_path = "docs/requirements.md"
design_path = "docs/design.md"
territory_path = "context/territory.toml"

[config]
preserve_existing_ids = true
output_path = "docs/design.md"

[output]
yield_path = "context/design-infer-producer-yield.toml"
```

---

## Yield Types

| Type | When to Use |
|------|-------------|
| `complete` | Design artifact created successfully |
| `need-context` | Need UI/UX files, screenshots, or semantic exploration |
| `need-user-input` (inferred) | Present inferred design decisions for user accept/reject |
| `need-decision` | Multiple valid design interpretations |
| `blocked` | Cannot proceed (missing visual assets, unclear patterns) |
| `error` | Something failed (retryable) |

---

## DES Format

```markdown
### DES-001: CLI Help Layout

Help text uses two-column format with command on left, description on right.
Subcommands are grouped by category with headers.

**Rationale:** Standard CLI convention, improves scanability.

**Traces to:** REQ-003
```

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

### Legacy Mode (yield protocol)

| Yield Type | When Used |
|------------|-----------|
| `need-context` | Need UI/UX files, screenshots, or semantic exploration |
| `need-user-input` (inferred) | Present inferred design decisions for user accept/reject |
| `need-decision` | Multiple valid design interpretations |
| `blocked` | Cannot proceed (missing visual assets, unclear patterns) |
| `complete` | Design artifact created successfully |
| `error` | Something failed (retryable) |

See [YIELD.md](../shared/YIELD.md) for yield format examples.

---

## Rules

1. **Preserve existing IDs** - Never reassign DES-N numbers
2. **Trace to requirements** - Every DES must link to REQ
3. **Focus on user-facing** - Infer from what users see, not implementation
4. **Document rationale** - Explain why the pattern was likely chosen
5. **Identify patterns** - Group related decisions

---

## Modes

| Mode | Action |
|------|--------|
| infer | Create design.md from UI/UX analysis |
| update | Add new decisions, preserve existing |
| normalize | Convert legacy format to DES-N headers |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/design.md"
      id_format: "DES-N"

  traces_to:
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every design decision has DES-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every DES-N traces to at least one REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "All user-facing REQ-N have corresponding DES-N (coverage)"
      severity: error

    - id: "CHECK-004"
      description: "No conflicting design decisions (consistency)"
      severity: error

    - id: "CHECK-005"
      description: "All screens and flows addressed (completeness)"
      severity: error

    - id: "CHECK-006"
      description: "Design describes user-facing behavior, not implementation"
      severity: warning

    - id: "CHECK-007"
      description: "No orphan REQ-N references"
      severity: warning

    - id: "CHECK-008"
      description: "Inferred designs include rationale"
      severity: warning
```
