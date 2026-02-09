---
name: evaluation-producer
description: Produces combined retrospective and project summary
context: inherit
model: sonnet
user-invocable: true
role: producer
phase: evaluation
---

# Evaluation Producer

Produce a combined project evaluation capturing retrospective findings, key decisions, outcomes, and actionable recommendations. This replaces the separate retro-producer and summary-producer steps with a single consolidated evaluation.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | All project artifacts, git history, session data |
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE |
| Output | evaluation.md with summary, decisions, outcomes, process findings |

---

## Workflow Context

- **Phase**: `evaluation_produce` (states.evaluation_produce)
- **Upstream**: Alignment commit (`alignment_commit`)
- **Downstream**: `evaluation_interview` → iterate or `evaluation_commit` → issue update
- **Model**: sonnet (default_model in workflows.toml)

This skill produces the combined retrospective and summary evaluation after alignment validation.

---

## GATHER Phase

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)

2. Load ALL project artifacts:
   - requirements.md (REQ-N IDs)
   - design.md (DES-N IDs)
   - architecture.md (ARCH-N IDs)
   - tasks.md (TASK-N IDs)
   - Implementation files and test results

3. Review project history:
   - Git log for commit history and timeline
   - Decision log and blockers encountered
   - Iteration history and QA feedback loops

4. Query semantic memory for context (best-effort, non-blocking):
   - `projctl memory query "retrospective challenges"`
   - `projctl memory query "process improvement recommendations"`
   - `projctl memory query "project summary patterns"`
   - If memory is unavailable, proceed gracefully without blocking

5. If missing information, send context request to team-lead with needed queries

---

## SYNTHESIZE Phase

1. **Assess outcomes vs goals**: Compare delivered results against original requirements
   - Which REQ-N items were fully met
   - Which had scope changes or partial delivery
   - Unexpected outcomes (positive or negative)

2. **Extract key decisions**: Identify significant choices from all phases
   - Requirements scoping decisions
   - Design trade-offs
   - Architectural choices
   - Implementation approaches

3. **Identify successes**: What worked well
   - Phases that passed QA on first iteration
   - Effective patterns or tooling
   - Good decisions and their outcomes

4. **Identify challenges**: What could improve
   - Phases requiring multiple iterations
   - Blockers encountered and resolution time
   - Missing context that caused delays

5. **Formulate recommendations**: Actionable improvements tiered by priority
   - High: Significant process changes
   - Medium: Moderate improvements
   - Low: Minor optimizations

---

## PRODUCE Phase

1. Generate evaluation.md with:
   - **Project Summary**: Duration, scope, key deliverables, team roles
   - **Key Decisions**: Significant choices with rationale and outcome, traced to artifact IDs
   - **Outcomes vs Goals**: REQ-N coverage assessment, quality metrics, performance results
   - **Process Findings**: Tiered recommendations (High/Medium/Low) with rationale
   - **Recommendations**: Actionable items for future projects
   - **Open Questions**: Unresolved decisions or deferred items

2. Include traceability to REQ-N, DES-N, ARCH-N, TASK-N

3. Create issues for actionable items:
   - High/Medium priority recommendations:
     ```bash
     projctl issue create \
       --title "Evaluation: <recommendation action>" \
       --priority <High|Medium> \
       --body "From evaluation: <rationale>"
     ```
   - Open questions:
     ```bash
     projctl issue create \
       --title "Decision needed: <question summary>" \
       --priority Medium \
       --body "Context: <question context>"
     ```
   - Low priority recommendations do not get issues

4. Persist learnings to memory:
   ```bash
   projctl memory learn "Success: <what went well>"
   projctl memory learn "Challenge: <what could improve>"
   projctl memory learn "Evaluation recommendation: <actionable improvement>"
   ```

5. Send results to team lead via `SendMessage`:
   - Artifact path
   - Issue IDs created
   - Files modified
   - Summary of outcomes, findings, recommendations

**Note:** The `evaluation_interview` state follows this producer, so the evaluation will be reviewed with the user before committing.

---

## Yield Protocol

### Yield Types

| Type | When |
|------|------|
| `complete` | Evaluation generated and issues created |
| `need-context` | Need artifacts, session data, or logs |
| `blocked` | Cannot proceed (missing project data) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-08T10:00:00Z

[payload]
artifact = "docs/evaluation.md"
files_modified = ["docs/evaluation.md", "docs/issues.md"]
issues_created = ["ISSUE-10", "ISSUE-11"]

[payload.summary]
duration = "3 sessions"
deliverables = 5
requirements_met = "12/12"

[[payload.decisions]]
context = "Architecture selection"
choice = "Modular plugin architecture"
reason = "Extensibility for future integrations"
traces_to = ["ARCH-3", "REQ-2"]

[[payload.outcomes]]
category = "features"
description = "All 12 requirements implemented"
evidence = "TASK-1 through TASK-12 completed"

[[payload.findings]]
priority = "high"
area = "Requirements Phase"
finding = "Early prototyping reduced rework"
action = "Include prototype step in planning"
issue = "ISSUE-10"

[[payload.findings]]
priority = "medium"
area = "Architecture Phase"
finding = "Missing system inventory caused rework"
action = "Add system inventory to kickoff"
issue = "ISSUE-11"

[[payload.findings]]
priority = "low"
area = "QA Phase"
finding = "Batch validation could be faster"
action = "Consider parallel QA runs"

[[payload.open_questions]]
question = "Should offline mode support bidirectional sync?"
context = "Deferred during implementation"

[context]
phase = "evaluation"
subphase = "complete"
```

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
    - path: "docs/evaluation.md"
      id_format: "N/A"

  traces_to:
    - "docs/requirements.md"
    - "docs/design.md"
    - "docs/architecture.md"
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Project summary is accurate and complete"
      severity: error

    - id: "CHECK-002"
      description: "Key decisions documented with rationale"
      severity: error

    - id: "CHECK-003"
      description: "Outcomes compared against original requirements"
      severity: error

    - id: "CHECK-004"
      description: "Process findings are tiered (High/Medium/Low)"
      severity: error

    - id: "CHECK-005"
      description: "Recommendations are actionable with rationale"
      severity: error

    - id: "CHECK-006"
      description: "Issues created for High/Medium priority items"
      severity: error

    - id: "CHECK-007"
      description: "Traces to REQ-N, DES-N, ARCH-N, TASK-N"
      severity: error

    - id: "CHECK-008"
      description: "Open questions section included"
      severity: error

    - id: "CHECK-009"
      description: "Decision descriptions match actual choices (accuracy)"
      severity: error

    - id: "CHECK-010"
      description: "No contradictions with upstream artifacts"
      severity: error

    - id: "CHECK-011"
      description: "Metrics are verifiable where available"
      severity: warning

    - id: "CHECK-012"
      description: "All project phases reviewed (completeness)"
      severity: warning
```
