---
name: retro-producer
description: Produce project retrospective with process improvement recommendations
context: fork
model: sonnet
user-invocable: true
role: producer
phase: retro
---

# Retrospective Producer

Produce a project retrospective analyzing what went well, what could improve, and actionable recommendations for process improvement.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML with project artifacts and session data |
| Analysis | Requirements, design, implementation, decisions, blockers |
| Output | Retrospective with successes, challenges, and action items |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read context from `[inputs]` section
2. Load project artifacts (requirements, design, architecture, tasks)
3. Review decision log and blockers encountered
4. Analyze iteration history and QA feedback loops
5. If missing information, yield `need-context` with queries

### SYNTHESIZE

1. Identify successes: what worked well during the project
   - Smooth phase transitions
   - Clean first-pass approvals
   - Effective tooling/patterns
2. Identify challenges: what could improve
   - Pain points and blockers
   - Rework cycles and iterations
   - Missing context or unclear requirements
3. Extract patterns from QA escalations
4. Formulate actionable improvement recommendations

### PRODUCE

1. Generate retrospective document with:
   - **Successes**: What went well
   - **Challenges**: What could improve
   - **Recommendations**: Action items for future projects
2. Include metrics where available (iteration counts, blockers)
3. Yield `complete` with artifact path

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `complete` | Retrospective generated successfully |
| `need-context` | Need session data, artifacts, or logs |
| `blocked` | Cannot proceed (missing project data) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T16:00:00Z

[payload]
artifact = "docs/retrospective.md"
files_modified = ["docs/retrospective.md"]

[[payload.successes]]
area = "Requirements Phase"
description = "Clear problem statement led to minimal iteration"

[[payload.challenges]]
area = "Architecture Phase"
description = "Missing context on existing auth system caused rework"

[[payload.recommendations]]
priority = "high"
action = "Include system inventory in project kickoff"
rationale = "Would have avoided ARCH-3 rework"

[context]
phase = "retro"
subphase = "complete"
```

## Retrospective Structure

The produced retrospective should cover:

### 1. Project Summary

- Duration and scope
- Key deliverables produced
- Team/agent roles involved

### 2. What Went Well (Successes)

- Phases that passed QA on first iteration
- Effective patterns or tooling
- Clear requirements that prevented ambiguity
- Good decisions and their outcomes

### 3. What Could Improve (Challenges)

- Phases requiring multiple iterations
- Blockers encountered and resolution time
- Missing context that caused delays
- Unclear requirements or scope creep

### 4. Process Improvement Recommendations

Each recommendation should be:
- **Actionable**: Specific change to implement
- **Measurable**: How to verify improvement
- **Prioritized**: High/Medium/Low impact

Example recommendations:
- "Add system inventory step before architecture phase"
- "Include edge case checklist in requirements template"
- "Establish context caching for frequently-accessed artifacts"

## Traceability

Retrospective traces to:
- **TASK-N**: Implementation tasks and their outcomes
- **Decisions**: Choices made and their rationale
- **Blockers**: Issues encountered and resolutions

## Result Format

`result.toml`: `[status]`, artifact path, `[[successes]]`, `[[challenges]]`, `[[recommendations]]`

## Full Documentation

`projctl skills docs --skillname retro-producer` or see SKILL-full.md
