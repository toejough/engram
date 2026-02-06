# Orchestration Infrastructure Requirements

Requirements inferred from ISSUE-4, ISSUE-11, ISSUE-12, ISSUE-19, ISSUE-20, ISSUE-21, ISSUE-25.

---

## CLI Commands

### REQ-001: State machine completed task tracking

The state machine must track which tasks have been completed so that `state next` returns a different task after `task-complete`.

**Acceptance Criteria:**

- `state next` suggests a different task after `task-complete` transition
- Completed tasks are tracked persistently in state.toml
- `state get` shows task completion progress

**Traces to:** ISSUE-4

---

### REQ-002: Sequential ID generation command

The CLI must provide `projctl id next` to generate sequential traceable IDs for artifacts.

**Acceptance Criteria:**

- `projctl id next --type REQ` returns next REQ-N
- `projctl id next --type DES` returns next DES-N
- `projctl id next --type ARCH` returns next ARCH-N
- `projctl id next --type TASK` returns next TASK-N
- Scans correct artifact files for each type
- Handles empty/missing files gracefully (returns TYPE-001)

**Traces to:** ISSUE-11

---

### REQ-003: Traceability visualization command

The CLI must provide `projctl trace show` to visualize the traceability chain between artifacts.

**Acceptance Criteria:**

- `projctl trace show` command exists
- Default output is ASCII tree showing ISSUE → REQ → DES → ARCH → TASK → test chain
- `--format json` outputs machine-readable JSON for tooling
- Orphan IDs (referenced but not defined) are marked with `[ORPHAN]`
- Unlinked IDs (defined but not connected) are marked with `[UNLINKED]`

**Traces to:** ISSUE-12

---

## Skill Enforcement

### REQ-004: Test trace promotion to permanent artifacts

The documentation phase must re-point test traceability from ephemeral TASK-NNN references to permanent artifact references (ARCH-NNN, DES-NNN, or REQ-NNN).

**Acceptance Criteria:**

- Documentation phase re-points test traces to permanent artifacts
- No orphan TASK-NNN references remain after documentation completes
- Trace validation passes with tests tracing to arch/des/req chain

**Traces to:** ISSUE-19

---

### REQ-005: Complete AC enforcement in tdd-qa

The tdd-qa skill must enforce that all acceptance criteria are complete before approving task completion. No work may be silently deferred.

**Acceptance Criteria:**

- `tdd-qa` parses AC from task definition
- Yields `improvement-request` if any AC is incomplete (marked `[ ]`)
- Yields `escalate-user` if producer deferred any work without user approval
- Test: task with 3/4 AC complete results in QA rejection
- Test: task with "deferred" language results in QA escalation to user

**Traces to:** ISSUE-20

---

### REQ-006: Retrospective findings issue conversion

Retrospective findings (recommendations and open questions) must be converted to tracked issues, not left as unactionable prose.

**Acceptance Criteria:**

- Retro recommendations with priority High/Medium become issues
- Open questions become issues with appropriate labels
- Each created issue traces back to retrospective
- User can see what issues were created from retro
- Test: retro with 3 High recommendations results in 3 issues created

**Traces to:** ISSUE-21

---

### REQ-007: Mandatory traceability in task breakdown

The breakdown-producer skill must include `**Traces to:**` as a mandatory field in every task definition, preventing orphan tasks.

**Acceptance Criteria:**

- breakdown-producer includes Traces-to in every task definition
- breakdown-qa rejects tasks without Traces-to
- breakdown-complete precondition includes trace validation
- Test: task without Traces-to results in QA rejection

**Traces to:** ISSUE-25

---
