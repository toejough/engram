---
name: retro-producer
description: Produce project retrospective with process improvement recommendations
context: inherit
model: sonnet
user-invocable: true
role: producer
phase: retro
---

# Retrospective Producer

Produce a project retrospective analyzing what went well, what could improve, and actionable recommendations for process improvement. Creates issues for high-priority recommendations.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context from spawn prompt: project artifacts and session data |
| Analysis | Requirements, design, implementation, decisions, blockers |
| Output | Retrospective with successes, challenges, action items, and issue creation |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
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
   - **Open Questions**: Unresolved decisions or ambiguities
2. Include metrics where available (iteration counts, blockers)
3. Create issues for actionable items (see Issue Creation below)
4. Send a message to team-lead with:
   - Artifact path
   - Issue IDs created
   - Files modified
   - Summary of successes, challenges, recommendations

### Issue Creation

After generating the retrospective, create issues for follow-up work:

1. **High/Medium priority recommendations**: For each recommendation with priority High or Medium:
   ```bash
   projctl issue create \
     --title "Retro: <recommendation action>" \
     --priority <High|Medium> \
     --body "From retrospective: <rationale>"
   ```

2. **Open questions**: For each unresolved question:
   ```bash
   projctl issue create \
     --title "Decision needed: <question summary>" \
     --priority Medium \
     --body "Context: <question context>"
   ```

3. **Track created IDs**: Collect all created issue IDs for the yield payload

Low priority recommendations do not get issues - they are documented for future reference only.

## Yield Protocol

### Yield Types

| Type | When |
|------|------|
| `complete` | Retrospective generated and issues created |
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
files_modified = ["docs/retrospective.md", "docs/issues.md"]
issues_created = ["ISSUE-7", "ISSUE-8", "ISSUE-9"]

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
issue = "ISSUE-7"

[[payload.recommendations]]
priority = "medium"
action = "Add context caching for frequently-accessed artifacts"
rationale = "Reduce redundant file reads"
issue = "ISSUE-8"

[[payload.recommendations]]
priority = "low"
action = "Consider batch yield validation"
rationale = "Minor efficiency gain"
# No issue created for low priority

[[payload.open_questions]]
question = "Should skills support multiple concurrent yields?"
context = "Edge case discovered during implementation"
issue = "ISSUE-9"

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

### 5. Open Questions

Document unresolved decisions or ambiguities discovered during the project:
- Questions that arose but weren't answered
- Deferred decisions that need future attention
- Scope items that were excluded but may be valuable

Example open questions:
- "Should offline mode support bidirectional sync?"
- "What is the retention policy for cached context?"

## Traceability

Retrospective traces to:
- **TASK-N**: Implementation tasks and their outcomes
- **Decisions**: Choices made and their rationale
- **Blockers**: Issues encountered and resolutions

## Result Format

`result.toml`: `[status]`, artifact path, `[[successes]]`, `[[challenges]]`, `[[recommendations]]`, `[[open_questions]]`, `issues_created`

## Full Documentation

`projctl skills docs --skillname retro-producer` or see SKILL-full.md

## Issue Creation Details

The skill creates issues automatically for:

| Item Type | Condition | Issue Title Format |
|-----------|-----------|-------------------|
| Recommendation | Priority = High | "Retro: <action>" |
| Recommendation | Priority = Medium | "Retro: <action>" |
| Open Question | Always | "Decision needed: <question>" |

Issues are NOT created for:
- Low priority recommendations (documented only)
- Successes (no follow-up needed)
- Challenges (addressed by recommendations)

### Issue Body Format

For recommendations:
```
From retrospective: <rationale>

Area: <area>
Related challenges: <if applicable>
```

For open questions:
```
Unresolved question from retrospective.

Context: <question context>
```

### Error Handling

If `projctl issue create` fails:
- Log the error
- Continue with remaining issues
- Include partial `issues_created` list in yield
- Add failed items to `issues_failed` array in payload

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Run projctl commands | `Bash` tool directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/retrospective.md"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"
    - "docs/architecture.md"
    - "docs/design.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Project summary is accurate"
      severity: error

    - id: "CHECK-002"
      description: "What went well section has specific examples"
      severity: error

    - id: "CHECK-003"
      description: "What could improve section identifies real challenges"
      severity: error

    - id: "CHECK-004"
      description: "Recommendations are actionable"
      severity: error

    - id: "CHECK-005"
      description: "Recommendations are prioritized (High/Medium/Low)"
      severity: error

    - id: "CHECK-006"
      description: "Recommendations include rationale"
      severity: error

    - id: "CHECK-007"
      description: "Issues created for High/Medium priority recommendations"
      severity: error

    - id: "CHECK-008"
      description: "Open questions section included"
      severity: error

    - id: "CHECK-009"
      description: "Metrics and data support observations where available"
      severity: warning

    - id: "CHECK-010"
      description: "All phases reviewed (completeness)"
      severity: warning
```
