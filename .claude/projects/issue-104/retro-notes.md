# Retro Notes - ISSUE-104

Captured during project execution for use by retro-producer at end of project.

## Observations

### O-1: TaskList tool not used reliably by team lead

**When:** During breakdown-complete → tdd-red transition
**What happened:** Team lead (orchestrator) did not create TaskList entries from tasks.md until Joe explicitly asked "what happened to using the claude task tool to track the live task status?" Despite system reminders appearing multiple times, they were ignored.
**Impact:** No live task tracking visible to user during PM, design, architect, and breakdown phases. User had no dashboard view of progress.
**Root cause:** The SKILL.md step loop doesn't explicitly include "create/update TaskList entries" as a step. It's implied by the Looper Pattern section but not enforced in the main control loop.
**Recommendation:** Add explicit TaskList creation step to the control loop, either at project init (from tasks.md) or at implementation phase entry. Make it a required action, not optional.
