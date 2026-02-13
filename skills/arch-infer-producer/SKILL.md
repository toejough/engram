---
name: arch-infer-producer
description: |
  Core: Analyzes existing code structure to infer and document architecture decisions with ARCH-N IDs for codebase adoption.
  Triggers: infer architecture, reverse-engineer architecture, analyze code structure, document existing architecture.
  Domains: architecture-inference, code-analysis, reverse-engineering, system-design, traceability.
  Anti-patterns: NOT for new architecture design, NOT for requirements inference (that's pm-infer-producer), NOT for design patterns (that's design-infer-producer).
  Related: arch-interview-producer (interview variant), pm-infer-producer, design-infer-producer (parallel infer skills).
context: inherit
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: arch
variant: infer
---

# Architecture Infer Producer

Analyze existing code to infer architecture decisions and produce architecture.md.

## Technical Decisions Only

Architecture phase focuses on **technology choices** and **system design**. Problem discovery belongs in PM, user experience belongs in Design.

**When inferring architecture decisions:**
- Extract technology stack (languages, frameworks, libraries)
- Document system structure (modules, layers, boundaries)
- Identify data models and schemas
- Note API contracts and interfaces
- Document integration patterns

**Avoid documenting:**
- What problems the code solves (belongs in PM)
- What features users need (belongs in PM)
- UI/UX patterns or visual design (belongs in Design)
- Interaction flows or user workflows (belongs in Design)

## Workflow Context

- **Phase**: `align_infer_arch_produce` (states.align_infer_arch_produce)
- **Upstream**: Align plan approval (`align_plan_approve`), parallel infer fork (`align_infer_fork`)
- **Downstream**: `align_infer_join` → `align_crosscut_qa` → decide → retry or commit
- **Model**: opus (default_model in workflows.toml)

This skill infers architecture from existing code structure in the align workflow for codebase adoption.

---

## Purpose

This skill examines code structure, dependencies, and patterns to reverse-engineer architecture decisions. Used for:
- Adopting existing codebases into the workflow
- Documenting undocumented architecture
- Creating ARCH-N IDs from implementation evidence

## Workflow

### 1. GATHER

Collect information about code structure:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Check for `[query_results]` (resuming after context request, legacy mode)
3. Execute `projctl memory query "prior architecture decisions for <project-domain>"` to load past technology choices
4. Execute `projctl memory query "technology patterns for <feature-area>"` to find established patterns
   If memory is unavailable, proceed gracefully without blocking
5. If code analysis needed, send context request to team-lead with queries:
   - `territory` queries for directory structure
   - `file` queries for go.mod, package.json, Makefile, etc.
   - `semantic` queries for pattern understanding
6. Proceed to SYNTHESIZE when sufficient information gathered

### 2. SYNTHESIZE

Analyze gathered code structure:

1. Identify package/module organization patterns
2. Extract technology choices and dependencies
3. Infer component boundaries from imports
4. Detect configuration and build patterns
5. Map to upstream requirements (REQ-N) and design (DES-N) where traceable

### 2b. CLASSIFY (Inference Detection)

Classify each planned architecture decision as explicit or inferred per [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md) inference guidelines.

1. For each architecture decision from SYNTHESIZE, determine if it was directly present in analyzed code/docs or inferred by the producer
2. If any inferred architecture decisions exist, use `AskUserQuestion` to present them for accept/reject
3. Wait for user accept/reject decisions
4. Drop rejected items, proceed to PRODUCE with only explicit + accepted items

### 3. PRODUCE

Create architecture.md artifact:

1. Generate ARCH-N IDs for each decision
2. Include `**Traces to:**` links to REQ/DES IDs
3. Write to configured output path
4. Send results to team lead via `SendMessage`:
   - Artifact path
   - ARCH IDs created
   - Files modified
   - Key decisions made

## Input

In team mode, context is provided via the spawn prompt. In legacy mode, read from context TOML:

```toml
[inputs]
project_dir = "/path/to/project"
requirements = "docs/requirements.md"  # Optional, for tracing
design = "docs/design.md"              # Optional, for tracing

[config]
output_path = "docs/architecture.md"
mode = "infer"  # infer | update | augment | normalize
```

## Output (architecture.md)

```markdown
# Architecture

## ARCH-001: Module Organization

Project uses Go module structure with internal/ for private packages.

**Traces to:** REQ-001

## ARCH-002: Build System

Uses Mage for build automation with targets defined in dev/.

**Traces to:** DES-003
```

## Communication

When to use different communication methods:

| Scenario | Tool |
|----------|------|
| Need code structure analysis | Send context request to team-lead |
| Present inferred architecture decisions for user accept/reject | Use `AskUserQuestion` with multiSelect |
| Ambiguous architecture choice | Use `AskUserQuestion` with options |
| architecture.md created | Send completion message to team-lead |
| Cannot analyze (e.g., binary-only) | Send blocker message to team-lead |
| Parse/access failure | Send error message to team-lead |

## ARCH ID Format

```markdown
### ARCH-NNN: Decision Title

Description of the architecture decision with rationale.

**Traces to:** REQ-XXX, DES-YYY
```

## Analysis Sources

| Source | What to Extract |
|--------|-----------------|
| go.mod / package.json | Dependencies, module structure |
| Directory structure | Package organization, boundaries |
| Import graph | Component dependencies |
| Build tooling | Makefile, mage, scripts |
| Config patterns | How configuration is loaded |
| Test structure | Testing strategy and boundaries |

## Modes

| Mode | Behavior |
|------|----------|
| `infer` | Create architecture.md from scratch |
| `update` | Add new decisions, preserve existing ARCH IDs |
| `augment` | Add ARCH-N IDs to existing content |
| `normalize` | Fix format, header levels, traceability |

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/architecture.md"
      id_format: "ARCH-N"

  traces_to:
    - "docs/design.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every architecture decision has ARCH-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every ARCH-N traces to at least one DES-N or REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "All technical implications from requirements addressed (completeness)"
      severity: error

    - id: "CHECK-004"
      description: "All technology decisions from design covered"
      severity: error

    - id: "CHECK-005"
      description: "No conflicts with requirements or design"
      severity: error

    - id: "CHECK-006"
      description: "No orphan references (mentions IDs that don't exist)"
      severity: error

    - id: "CHECK-007"
      description: "Technical decisions include rationale"
      severity: warning

    - id: "CHECK-008"
      description: "Inferred architecture traces to observable code patterns"
      severity: warning
```
