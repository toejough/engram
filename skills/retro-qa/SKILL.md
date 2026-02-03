---
name: retro-qa
description: Reviews retrospective for completeness and actionable recommendations
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: retro
---

# Retrospective QA Skill

Reviews retrospectives produced by retro-producer for completeness, thoroughness, and actionable recommendations.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's retrospective artifact:

1. Read producer's artifact from context (`docs/retrospective.md`)
2. Load acceptance criteria for retrospectives
3. Check each criterion:
   - Successes section is present and comprehensive?
   - Challenges section identifies real pain points?
   - Recommendations are actionable and prioritized?
   - Traces to project artifacts where applicable?
4. Compile findings

#### Retrospective Checklist

- [ ] Project summary is accurate
- [ ] What went well section has specific examples
- [ ] What could improve section identifies real challenges
- [ ] Recommendations are actionable (not vague)
- [ ] Recommendations are prioritized (high/medium/low)
- [ ] Recommendations include rationale
- [ ] Metrics/data support observations where available
- [ ] No critical successes or challenges omitted

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All criteria pass. Retrospective is complete and thorough.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T16:30:00Z

[payload]
reviewed_artifact = "docs/retrospective.md"
checklist = [
    { item = "Successes section is comprehensive", passed = true },
    { item = "Challenges identify real pain points", passed = true },
    { item = "Recommendations are actionable", passed = true },
    { item = "Recommendations are prioritized", passed = true }
]

[context]
phase = "retro"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that the producer can fix.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T16:35:00Z

[payload]
from_agent = "retro-qa"
to_agent = "retro-producer"
iteration = 2
issues = [
    "Recommendation 2 is too vague - 'improve communication' lacks specifics",
    "Missing challenge: multiple ARCH escalations not addressed",
    "Success about 'good testing' lacks specific example"
]

[context]
phase = "retro"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires revisiting project data. Used when:

- **error**: Retrospective contains factual inaccuracies
- **gap**: Missing data needed for complete retrospective
- **conflict**: Observations contradict project records

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T16:40:00Z

[payload.escalation]
from_phase = "retro"
to_phase = "retro"
reason = "gap"

[payload.issue]
summary = "Missing iteration data for design phase"
context = "Cannot verify QA cycles without access to design-qa yields"

[[payload.proposed_changes.requirements]]
action = "add"
id = "RETRO-DATA-1"
title = "Include QA iteration logs in retro context"
content = "Retrospective context should include all QA yield files"

[context]
phase = "retro"
role = "qa"
escalating = true
```

#### `escalate-user`

Cannot resolve issue without user input.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T16:45:00Z

[payload]
reason = "Subjective assessment needed"
context = "Recommendation priority unclear without stakeholder input"
question = "Should 'improve onboarding docs' be high or medium priority?"
options = ["High - blocks new contributors", "Medium - nice to have", "Low - defer"]

[context]
phase = "retro"
role = "qa"
escalating = true
```

## Iteration Limits

QA tracks iterations to prevent infinite loops:

```toml
[context]
iteration = 2
max_iterations = 3
```

After max iterations:
1. Yield `escalate-user` if issues remain unresolved
2. Or yield `approved` with caveats noted in payload

## Quality Criteria

Retrospectives must be:

1. **Complete**: Covers successes, challenges, and recommendations
2. **Specific**: Examples and data, not vague statements
3. **Actionable**: Recommendations can be implemented
4. **Prioritized**: Clear importance ranking
5. **Traceable**: Links to project artifacts where relevant

## Validation Details

### Successes Validation

- Each success should cite specific evidence
- Balance between different project phases
- Avoid generic praise without substance

### Challenges Validation

- Real pain points, not hypotheticals
- Connected to observable outcomes (rework, delays)
- Constructive framing, not blame

### Recommendations Validation

- Action items, not observations
- Include who/what/when where possible
- Rationale explains why this matters
- Priority based on impact and effort
