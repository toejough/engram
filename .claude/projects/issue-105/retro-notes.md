# Retro Notes - ISSUE-105

## Observations During Execution

### Premature agent replacement wastes resources
**Context:** Multiple times during ISSUE-105, the orchestrator shut down QA agents after a single idle notification, assuming they were stuck. In reality, the agents were actively running subagent commands (e.g., invoking /qa which spawns its own work).

**Impact:** Wasted API tokens respawning agents that were working fine. Created unnecessary churn and delays.

**Lesson:** An idle notification means the agent's turn ended, NOT that it's done or stuck. Agents often spawn subagents or run async work. Wait for at least 2-3 idle cycles or a reasonable timeout before assuming an agent is stuck. Only respawn after confirming no work is in progress.

### Phase QA vs commit QA serve different purposes — don't propose removing them
**Context:** Initially thought the double QA (phase-qa then commit-qa) was redundant and proposed filing a ticket to remove it. The user corrected this — phase QA validates the artifact/code, commit QA validates commit hygiene (correct files staged, message format, no secrets, phase scope). These are different checks.

**Lesson:** Understand the state machine design before suggesting changes. Don't file issues for "improvements" that are actually working as designed. The 6 QA spawns per task cycle (2 per red/green/refactor) are intentional.

### No parallelization within a single task's TDD cycle
**Context:** The red/green/refactor cycle is inherently sequential — each step depends on the previous. Parallelization only applies at the task level (running independent tasks concurrently). The `projctl step next` state machine currently drives one linear path, making task-level parallelization an orchestrator responsibility (ISSUE-120).
