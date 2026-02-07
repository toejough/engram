# Retro Notes - ISSUE-104

Captured during project execution for use by retro-producer at end of project.

## Observations

### O-1: TaskList tool not used reliably by team lead

**When:** During breakdown-complete → tdd-red transition
**What happened:** Team lead (orchestrator) did not create TaskList entries from tasks.md until Joe explicitly asked "what happened to using the claude task tool to track the live task status?" Despite system reminders appearing multiple times, they were ignored.
**Impact:** No live task tracking visible to user during PM, design, architect, and breakdown phases. User had no dashboard view of progress.
**Root cause:** The SKILL.md step loop doesn't explicitly include "create/update TaskList entries" as a step. It's implied by the Looper Pattern section but not enforced in the main control loop.
**Recommendation:** Add explicit TaskList creation step to the control loop, either at project init (from tasks.md) or at implementation phase entry. Make it a required action, not optional.

### O-2: Redundant commit-red QA passes waste tokens

**When:** After TDD red commit
**What happened:** State machine has commit-red → commit-red-qa phases that spawn separate commit-producer and QA agents. But the team lead had already committed during the earlier `commit` action from the tdd-red phase. The commit-red phase then spawns a commit-producer (nothing to commit), then QA on the commit, then another commit, then commit-red-qa with yet another QA spawn. That's 4 redundant steps with agent spawns for work already done.
**Impact:** ~4 extra agent spawns (2 haiku producers, 2 haiku QAs) burning tokens on empty commits.
**Root cause:** The state machine has both a `commit` action within the tdd-red phase AND a separate `commit-red` phase with its own producer/QA pair. Double-counting the same commit.
**Recommendation:** Investigate whether commit-red/commit-red-qa phases can be collapsed. The commit QA analysis agent is investigating whether commit QA ever finds real issues.

### O-3: TDD red phase not scoped to specific task

**When:** TDD red phase entry
**What happened:** `projctl step next` returned tdd-red with `current_task = ""`. The tdd-red-producer chose to write tests for TASK-1 only, but the state machine didn't tell it which task to work on.
**Impact:** Unclear contract between state machine and TDD producers about task scope.
**Recommendation:** Under investigation by state-machine-advisor teammate.
