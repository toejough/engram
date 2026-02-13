---
name: design-infer-producer
description: |
  Core: Analyzes existing UI/UX to infer and document design decisions with DES-N IDs for codebase adoption scenarios.
  Triggers: infer design, reverse-engineer UX, analyze existing interface, document existing design patterns.
  Domains: design-inference, ux-analysis, reverse-engineering, interaction-patterns, traceability.
  Anti-patterns: NOT for new design work, NOT for requirements inference (that's pm-infer-producer), NOT for architecture analysis (that's arch-infer-producer).
  Related: design-interview-producer (interview variant), pm-infer-producer, arch-infer-producer (parallel infer skills).
context: inherit
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

## Workflow Context

- **Phase**: `align_infer_design_produce` (states.align_infer_design_produce)
- **Upstream**: Align plan approval (`align_plan_approve`), parallel infer fork (`align_infer_fork`)
- **Downstream**: `align_infer_join` → `align_crosscut_qa` → decide → retry or commit
- **Model**: opus (default_model in workflows.toml)

This skill infers design decisions from existing UI/UX in the align workflow for codebase adoption.

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
2. Check for `[query_results]` (resuming after context request, legacy mode)
3. Execute `projctl memory query "prior design decisions for <project-domain>"` to load past UI/UX choices
4. Execute `projctl memory query "design patterns for <feature-area>"` to find established patterns
   If memory is unavailable, proceed gracefully without blocking
5. If missing interface information, send context request to team-lead with needed queries:
   - File queries for entry points
   - Semantic queries for user-facing interfaces
   - Territory queries for UI-related files

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
5. If blocked, send blocker message to team-lead

### 2b. CLASSIFY (Inference Detection)

Classify each planned design decision as explicit or inferred per [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md) inference guidelines.

1. For each design decision from SYNTHESIZE, determine if it was directly present in analyzed code/UI or inferred by the producer
2. If any inferred design decisions exist, use `AskUserQuestion` to present them for accept/reject
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

---

## Input Context

Context is provided via the spawn prompt.

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
```

---

## Communication

When to use different communication methods:

| Scenario | Tool |
|----------|------|
| Design artifact created successfully | Send completion message to team-lead |
| Need UI/UX files, screenshots, or semantic exploration | Send context request to team-lead |
| Present inferred design decisions for user accept/reject | Use `AskUserQuestion` with multiSelect |
| Multiple valid design interpretations | Use `AskUserQuestion` with options |
| Cannot proceed (missing visual assets, unclear patterns) | Send blocker message to team-lead |
| Something failed (retryable) | Send error message to team-lead |

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
