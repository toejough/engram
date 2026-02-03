---
name: summary-producer
description: Produce project summary with key decisions and outcomes
context: fork
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
| Input | Context TOML with artifact paths |
| Analysis | Requirements, design, architecture, tasks, implementation |
| Output | Project summary with decisions and outcomes |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read context from `[inputs]` section
2. Load REQ-N from requirements.md
3. Load DES-N from design.md
4. Load ARCH-N from architecture.md
5. Load TASK-N from tasks.md
6. Review implementation commits and changes
7. If missing information, yield `need-context` with queries

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
3. Yield `complete` with artifact path

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

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
