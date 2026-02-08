---
name: summary-producer
description: Produce project summary with key decisions and outcomes
context: inherit
model: sonnet
user-invocable: false
role: producer
phase: summary
---

# Summary Producer

Produce a comprehensive project summary capturing key decisions, outcomes, and lessons learned from the project lifecycle.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context from spawn prompt: artifact paths |
| Analysis | Requirements, design, architecture, tasks, implementation |
| Output | Project summary with decisions and outcomes |

## Workflow Context

- **Phase**: `summary_produce` (states.summary_produce) - DEPRECATED in favor of evaluation-producer
- **Upstream**: Retro commit (`retro_commit`) in old sequential flow
- **Downstream**: `summary_qa` → `summary_decide` → retry or `summary_commit` → issue update
- **Model**: sonnet (default_model in workflows.toml)

**NOTE**: This skill is deprecated. Use evaluation-producer instead for combined retrospective and summary.

---

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Load REQ-N from requirements.md
3. Load DES-N from design.md
4. Load ARCH-N from architecture.md
5. Load TASK-N from tasks.md
6. Review implementation commits and changes
7. Query memory for past learnings:
   - `projctl memory query "project summary patterns"`
   - `projctl memory query "known failures in summary validation"`
   - Memory queries are non-blocking - if unavailable, continue without them
8. If missing information, yield `need-context` with queries

### SYNTHESIZE

1. Extract key decisions from all phases:
   - Requirements scoping decisions
   - Design trade-offs
   - Architectural choices
   - Implementation approaches
2. Identify project outcomes:
   - Features delivered
   - Quality metrics achieved
   - Performance results
3. Compile lessons learned:
   - What worked well
   - What could improve
   - Patterns to reuse

### PRODUCE

1. Generate project summary document:
   - Executive overview
   - Key decisions with rationale
   - Outcomes and deliverables
   - Lessons learned
2. Include traceability to REQ-N, DES-N, ARCH-N, TASK-N
3. Send a message to team-lead with:
   - Artifact path
   - Files modified
   - Key decisions documented
   - Outcomes summary

## Yield Protocol

### Yield Types

| Type | When |
|------|------|
| `complete` | Summary generated successfully |
| `need-context` | Need files, artifacts, or history |
| `blocked` | Cannot proceed (missing artifacts) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "docs/project-summary.md"
files_modified = ["docs/project-summary.md"]

[[payload.decisions]]
context = "Architecture selection"
choice = "Modular plugin architecture"
reason = "Extensibility for future integrations"
phase = "arch"
traces_to = ["ARCH-3", "REQ-2"]

[[payload.outcomes]]
category = "features"
description = "All 12 requirements implemented"
evidence = "TASK-1 through TASK-12 completed"

[[payload.outcomes]]
category = "quality"
description = "95% test coverage achieved"
evidence = "Coverage report in artifacts/"

[[payload.learnings]]
content = "Early prototyping reduced rework significantly"
category = "process"

[context]
phase = "summary"
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
question = "What were the major decisions made during implementation?"

[context]
phase = "summary"
subphase = "GATHER"
awaiting = "context-results"
```

## Traceability

Summary traces to all upstream artifacts:

- **REQ-N**: Requirements that drove the project
- **DES-N**: Design decisions made
- **ARCH-N**: Architectural choices
- **TASK-N**: Implementation tasks completed

Example traceability in summary:

```markdown
## Key Decisions

### Plugin Architecture (ARCH-3)
**Traces to:** REQ-2, DES-5

Chose modular plugin system to support...
```

## Summary Content Structure

### Executive Overview
- Project goals and scope
- High-level outcome summary
- Timeline and milestones

### Key Decisions
For each significant decision:
- Context and constraints
- Options considered
- Choice made and rationale
- Outcome/impact

### Outcomes
- Features delivered (with REQ-N references)
- Quality metrics
- Performance results
- Known limitations

### Lessons Learned
- Process improvements
- Technical insights
- Patterns to reuse

## Result Format

`result.toml`: `[status]`, files modified, `[[decisions]]`, `[[outcomes]]`, `[[learnings]]`

## Full Documentation

`projctl skills docs --skillname summary-producer` or see SKILL-full.md

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
    - path: "docs/project-summary.md"
      id_format: "N/A"

  traces_to:
    - "docs/requirements.md"
    - "docs/design.md"
    - "docs/architecture.md"
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Executive overview present"
      severity: error

    - id: "CHECK-002"
      description: "All major decisions documented"
      severity: error

    - id: "CHECK-003"
      description: "Outcomes and deliverables documented"
      severity: error

    - id: "CHECK-004"
      description: "Lessons learned section included"
      severity: error

    - id: "CHECK-005"
      description: "Traces to REQ-N, DES-N, ARCH-N, TASK-N"
      severity: error

    - id: "CHECK-006"
      description: "Decision descriptions match actual choices made (accuracy)"
      severity: error

    - id: "CHECK-007"
      description: "Outcomes reflect actual implementation results (accuracy)"
      severity: error

    - id: "CHECK-008"
      description: "Metrics are verifiable (coverage %, performance numbers)"
      severity: error

    - id: "CHECK-009"
      description: "No contradictions with upstream artifacts"
      severity: error

    - id: "CHECK-010"
      description: "Key trade-offs explained"
      severity: warning

    - id: "CHECK-011"
      description: "Known limitations documented"
      severity: warning

    - id: "CHECK-012"
      description: "Timeline and milestones accurate"
      severity: warning
```
