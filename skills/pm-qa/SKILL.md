---
name: pm-qa
description: Reviews requirements for completeness, clarity, and testability
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: pm
---

# PM QA Skill

Reviews requirements produced by pm-interview-producer or pm-infer-producer for quality and completeness.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's requirements artifact:

1. Read producer's artifact from context (`docs/requirements.md`)
2. Load acceptance criteria for requirements
3. Check each criterion:
   - All requirements have REQ-N IDs?
   - Traces to upstream issue/feature request?
   - Acceptance criteria are testable and measurable?
   - No conflicts between requirements?
   - Content complete and accurate?
4. Compile findings

#### Requirements Checklist

- [ ] Each requirement has a unique REQ-N identifier
- [ ] Requirements trace to source (issue, user request, discovery)
- [ ] Acceptance criteria are specific and measurable
- [ ] No ambiguous language ("should", "may", "might")
- [ ] No conflicting requirements
- [ ] Edge cases identified where applicable
- [ ] Dependencies between requirements documented
- [ ] Priority/scope clearly indicated

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All criteria pass. Requirements are ready for design phase.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/requirements.md"
checklist = [
    { item = "All requirements have REQ-N IDs", passed = true },
    { item = "Acceptance criteria are testable", passed = true },
    { item = "Traces to issue", passed = true },
    { item = "No conflicting requirements", passed = true }
]

[context]
phase = "pm"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that the producer can fix.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "pm-qa"
to_agent = "pm-interview-producer"
iteration = 2
issues = [
    "REQ-3 acceptance criteria 'should be fast' is not measurable",
    "REQ-5 missing edge case for empty input",
    "REQ-7 conflicts with REQ-2 on authentication method"
]

[context]
phase = "pm"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Producer made a mistake that violates constraints
- **gap**: Missing content that should exist based on upstream context
- **conflict**: Contradictory statements across artifacts

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "pm"
to_phase = "pm"  # Can escalate within same phase
reason = "gap"  # error | gap | conflict

[payload.issue]
summary = "User interview revealed unaddressed security concern"
context = "User mentioned PII handling but no requirements address it"

[[payload.proposed_changes.requirements]]
action = "add"
id = "REQ-12"
title = "PII Data Protection"
content = "System must encrypt all personally identifiable information at rest and in transit"

[context]
phase = "pm"
role = "qa"
escalating = true
```

#### `escalate-user`

Cannot resolve issue without user input.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Ambiguous scope decision"
context = "REQ-4 and REQ-8 suggest conflicting priorities"
question = "Should we prioritize offline-first or real-time sync?"
options = ["Offline first", "Real-time first", "User configurable"]

[context]
phase = "pm"
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

Requirements must be:

1. **Complete**: All user needs captured
2. **Clear**: Unambiguous language, no "should/may/might"
3. **Testable**: Acceptance criteria are measurable
4. **Consistent**: No conflicts between requirements
5. **Traceable**: Links to source and downstream artifacts
