# Design - ISSUE-105: Remove Composite Skill Redundancy

**Status:** Draft
**Created:** 2026-02-06
**Issue:** ISSUE-105

**Traces to:** ISSUE-105

---

## Design Principles

Based on user design interview (2026-02-06):

### DES-105-001: Invisible Architecture Changes

**Description:** All composite skill elimination changes are internal architecture only. Users experience no change in workflow commands or usage patterns.

**User Experience:** When running `/project new ISSUE-XXX`, the user sees the same workflow as before. The command syntax, workflow phases, and completion behavior remain unchanged.

**Rationale:** User selected option A ("Completely unchanged") for user workflow experience. The refactoring eliminates redundant agent nesting without altering the user-facing interface.

**Traces to:** REQ-105-001, REQ-105-002, ISSUE-105

---

### DES-105-002: Error Message Continuity

**Description:** Error messages and recovery paths maintain existing format and behavior.

**User Experience:** When a state transition fails (e.g., QA rejects after max iterations), users see the same error message format they currently receive. No new error types or recovery commands are introduced.

**Rationale:** User selected option A ("Same as current") for error messages. Consistency reduces cognitive load and preserves existing runbooks.

**Traces to:** REQ-105-001, ISSUE-105

---

### DES-105-003: Step-by-Step Progress Visibility

**Description:** Progress tracking exposes every state transition as it happens, providing full visibility into the workflow execution.

**User Experience:** For long-running workflows with multiple state transitions (like full TDD cycle), users see each step as it executes:
```
Spawning tdd-red-producer...
Spawning qa for tdd-red...
QA approved tdd-red
Transitioning to commit-red...
Spawning commit-producer for red phase...
Spawning qa for commit-red...
QA approved commit-red
Transitioning to tdd-green...
```

**Implementation:** The `/project` orchestrator outputs status messages after each `projctl step next` call and teammate spawn. Each state transition triggers a user-visible progress update.

**Rationale:** User selected option C ("Step-by-step updates") for progress tracking. This maximizes transparency and helps users understand where time is being spent in complex workflows.

**Traces to:** REQ-105-002, REQ-105-003, ISSUE-105

---

### DES-105-004: Full QA Feedback Display

**Description:** When QA iterations occur (improvement requested), users see complete QA feedback including what was flagged and why.

**User Experience:** When a producer/QA loop iterates:
```
QA iteration 1: Improvement requested
Issues flagged by QA:
  - Missing trace to REQ-042 in DES-007
  - ID format incorrect: "DESIGN-003" should be "DES-003"
  - Section "User Flows" has no content

Spawning design-interview-producer again with QA feedback...
```

**Implementation:** The orchestrator displays the full `qa-feedback` field from `projctl step complete --qa-verdict improvement-request --qa-feedback "<details>"` before spawning the next producer iteration.

**Rationale:** User selected option C ("Full feedback") for QA iteration visibility. This helps users understand what's being corrected and builds confidence in the QA process.

**Traces to:** REQ-105-002, REQ-105-003, ISSUE-105

---

## Architectural Patterns

### DES-105-005: State Machine Orchestration Pattern

**Description:** The `/project` orchestrator drives all workflow coordination via `projctl step next` state machine queries, eliminating composite skills that spawn sub-agents internally.

**Current Architecture (Redundant):**
```
User -> /project orchestrator
  -> spawns tdd-producer teammate
    -> tdd-producer spawns tdd-red-producer (REDUNDANT!)
      -> tdd-red-producer spawns qa (REDUNDANT!)
    -> tdd-producer spawns tdd-green-producer (REDUNDANT!)
      -> tdd-green-producer spawns qa (REDUNDANT!)
```

**New Architecture (State-Driven):**
```
User -> /project orchestrator
  loop:
    -> projctl step next -> {action: "spawn-producer", skill: "tdd-red-producer"}
    -> spawns tdd-red-producer teammate
    -> projctl step next -> {action: "spawn-qa", skill: "qa"}
    -> spawns qa teammate
    -> projctl step next -> {action: "spawn-producer", skill: "tdd-green-producer"}
    -> spawns tdd-green-producer teammate
    -> projctl step next -> {action: "spawn-qa", skill: "qa"}
    -> spawns qa teammate
    -> ... (continue for refactor)
```

**Key Change:** The orchestrator (`/project`) maintains control at the top level and delegates to `projctl step next` for each workflow decision. Composite skills like `tdd-producer` are eliminated because they duplicated the orchestrator's job.

**Benefits:**
- **Single source of truth:** State machine in `projctl step next` is the only orchestration logic
- **Reduced nesting:** One fewer agent layer per composite skill invocation
- **Simplified debugging:** Single conversation trace instead of nested agent contexts
- **Lower latency:** Eliminate redundant agent spawn overhead
- **Lower API costs:** One fewer context load per composite skill

**Traces to:** REQ-105-002, REQ-105-003, ARCH-105-012, ARCH-105-013, ARCH-105-018, ISSUE-105

---

### DES-105-006: Producer-QA Iteration via State Machine

**Description:** QA iteration loops (improvement-request cycles) are driven by state machine transitions, not by composite skill logic.

**Iteration Flow:**
```
phase=tdd-red, iteration=0
  -> spawn tdd-red-producer
  -> producer completes
  -> spawn qa
  -> qa verdict: improvement-request

phase=tdd-red, iteration=1
  -> spawn tdd-red-producer (with qa-feedback)
  -> producer completes
  -> spawn qa
  -> qa verdict: improvement-request

phase=tdd-red, iteration=2
  -> spawn tdd-red-producer (with qa-feedback)
  -> producer completes
  -> spawn qa
  -> qa verdict: approved

phase=tdd-red-qa (approved)
  -> transition to next phase
```

**State Machine Behavior:**
- On `qa-verdict=improvement-request`, `projctl step next` returns action to re-spawn the same producer with `qa_feedback` in context
- Iteration counter increments with each producer spawn
- Max iteration limit (default: 3) enforced by state machine
- On max iterations, state machine returns `action: "escalate-user"` instead of re-spawning producer

**Comparison to Current:**
- **Before:** `tdd-producer` composite skill contained nested loop logic to spawn `tdd-red-producer`, check QA verdict, re-spawn if needed
- **After:** State machine in `internal/state/` tracks iteration count and returns appropriate next action based on QA verdict

**Traces to:** REQ-105-002, REQ-105-003, ARCH-105-009, ISSUE-105

---

### DES-105-007: Composite Skill Identification Criteria

**Description:** Skills are classified as composite orchestrators if they spawn sub-agents via Task tool for sequential workflow coordination.

**Classification Algorithm:**

1. **Read SKILL.md content**
2. **Check for Task tool usage** (search for `Task(` patterns)
3. **Analyze Task tool context:**
   - If spawning skills for sequential workflow phases → **Composite orchestrator**
   - If spawning for parallel independent work (like `parallel-looper`) → **Composite orchestrator**
   - If using Task for isolated utility (e.g., background exploration) → **Leaf skill**

4. **Look for orchestration language:**
   - Phrases: "nested loops", "pair loops", "orchestrates", "coordinates", "sequential phases"
   - Workflow diagrams showing skill → skill chains
   - Producer/QA iteration logic

**Example - Composite Orchestrator:**
```markdown
# TDD Producer (Composite)

Orchestrates the complete TDD cycle by running nested pair loops for RED, GREEN, and REFACTOR phases.

## Nested Pair Loops
This skill runs three sequential pair loops, each with producer/QA iteration:
...
```

**Example - Leaf Skill:**
```markdown
# TDD Red Producer

Writes failing tests for acceptance criteria.

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE
...
```

**Classification Results (from audit):**
- **Composite orchestrators:** `tdd-producer`, `parallel-looper` (deprecated)
- **Leaf skills:** All other producers, qa, commit, context-explorer, etc.

**Traces to:** REQ-105-001, ISSUE-105

---

## State Machine Design

### DES-105-008: TDD Phase State Transitions

**Description:** State machine transitions for the TDD workflow replace `tdd-producer` composite skill orchestration logic.

**Current TDD State Flow (with composite):**
```
phase=tdd
  -> spawn tdd-producer (composite)
    [internally spawns tdd-red-producer, qa, tdd-green-producer, qa, tdd-refactor-producer, qa]
  -> tdd-producer completes
  -> transition to commit-tdd
```

**New TDD State Flow (state-driven):**
```
phase=tdd-red
  -> spawn tdd-red-producer
  -> spawn qa (tdd-red-qa)
  -> transition to commit-red

phase=commit-red
  -> spawn commit-producer
  -> spawn qa (commit-red-qa)
  -> transition to tdd-green

phase=tdd-green
  -> spawn tdd-green-producer
  -> spawn qa (tdd-green-qa)
  -> transition to commit-green

phase=commit-green
  -> spawn commit-producer
  -> spawn qa (commit-green-qa)
  -> transition to tdd-refactor

phase=tdd-refactor
  -> spawn tdd-refactor-producer
  -> spawn qa (tdd-refactor-qa)
  -> transition to commit-refactor

phase=commit-refactor
  -> spawn commit-producer
  -> spawn qa (commit-refactor-qa)
  -> transition to task-audit
```

**State Machine Implementation:**

New phases defined in `internal/step/registry.go`:
- `tdd-red`, `tdd-red-qa`, `commit-red`, `commit-red-qa`
- `tdd-green`, `tdd-green-qa`, `commit-green`, `commit-green-qa`
- `tdd-refactor`, `tdd-refactor-qa`, `commit-refactor`, `commit-refactor-qa`

Each phase entry specifies:
- `Producer`: Skill to spawn for producer action
- `QA`: Skill to spawn for QA action (`qa`)
- `Model`: Model to use for spawned teammate
- `ProducerPath`, `QAPath`: SKILL.md file paths

**Transition Enforcement:**

Legal transitions in `internal/state/transitions.go`:
```go
"tdd-red": []string{"tdd-red-qa"},
"tdd-red-qa": []string{"commit-red"},
"commit-red": []string{"commit-red-qa"},
"commit-red-qa": []string{"tdd-green"},
"tdd-green": []string{"tdd-green-qa"},
"tdd-green-qa": []string{"commit-green"},
"commit-green": []string{"commit-green-qa"},
"commit-green-qa": []string{"tdd-refactor"},
"tdd-refactor": []string{"tdd-refactor-qa"},
"tdd-refactor-qa": []string{"commit-refactor"},
"commit-refactor": []string{"commit-refactor-qa"},
"commit-refactor-qa": []string{"task-audit"},
```

**Illegal Transitions (prevented by state machine):**
- `tdd-red` → `commit-red` (must go through `tdd-red-qa`)
- `tdd-red` → `tdd-green` (must complete commit-red cycle)
- Any phase skipping (state machine enforces sequential progression)

**Traces to:** REQ-105-002, ARCH-105-003, ARCH-105-005, ARCH-105-006, ARCH-105-007, ARCH-105-008, ISSUE-105

---

### DES-105-009: Step Next Action JSON Schema

**Description:** `projctl step next` returns structured JSON describing the next action for the orchestrator to execute. Schema remains unchanged from current implementation; state machine internal logic changes to return correct actions for new TDD sub-phases.

**Action Types:**

1. **spawn-producer** - Spawn a producer teammate
2. **spawn-qa** - Spawn QA teammate with producer context
3. **commit** - Create git commit
4. **transition** - Phase boundary crossing
5. **escalate-user** - Max retries reached, user intervention needed
6. **all-complete** - All phases done

**JSON Schema:**

```json
{
  "action": "spawn-producer | spawn-qa | commit | transition | escalate-user | all-complete",
  "skill": "<skill-name>",
  "skill_path": "<path-to-SKILL.md>",
  "model": "sonnet | haiku | opus",
  "phase": "<current-phase>",
  "iteration": "<iteration-count>",
  "context": {
    "issue": "<ISSUE-NNN>",
    "prior_artifacts": ["<paths>"],
    "qa_feedback": "<feedback-from-prior-qa>"
  },
  "task_params": {
    "subagent_type": "<agent-type>",
    "name": "<teammate-name>",
    "model": "<model>",
    "prompt": "<spawn-prompt>",
    "team_name": "<team-name>"
  },
  "expected_model": "<model>",
  "details": "<escalation-details>"
}
```

**Field Usage by Action:**

| Field | spawn-producer | spawn-qa | commit | transition | escalate-user | all-complete |
|-------|----------------|----------|--------|------------|---------------|--------------|
| action | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| skill | ✓ | ✓ | - | - | - | - |
| skill_path | ✓ | ✓ | - | - | - | - |
| model | ✓ | ✓ | - | - | - | - |
| phase | ✓ | ✓ | ✓ | ✓ | ✓ | - |
| iteration | ✓ | ✓ | - | - | - | - |
| context | ✓ | ✓ | - | - | - | - |
| task_params | ✓ | ✓ | - | - | - | - |
| expected_model | ✓ | ✓ | - | - | - | - |
| details | - | - | - | - | ✓ | - |

**Example - spawn-producer:**
```json
{
  "action": "spawn-producer",
  "skill": "tdd-red-producer",
  "skill_path": "skills/tdd-red-producer/SKILL.md",
  "model": "sonnet",
  "phase": "tdd-red",
  "iteration": 0,
  "context": {
    "issue": "ISSUE-105",
    "prior_artifacts": ["requirements.md", "design.md", "architecture.md"],
    "qa_feedback": ""
  },
  "task_params": {
    "subagent_type": "general-purpose",
    "name": "tdd-red-producer",
    "model": "sonnet",
    "prompt": "First, respond with your model name...\n\nThen invoke /tdd-red-producer.\n\nIssue: ISSUE-105",
    "team_name": "issue-105"
  },
  "expected_model": "sonnet"
}
```

**Example - spawn-qa (with iteration):**
```json
{
  "action": "spawn-qa",
  "skill": "qa",
  "skill_path": "skills/qa/SKILL.md",
  "model": "haiku",
  "phase": "tdd-red-qa",
  "iteration": 1,
  "context": {
    "issue": "ISSUE-105",
    "prior_artifacts": ["tests/new_feature_test.go"],
    "qa_feedback": ""
  },
  "task_params": {
    "subagent_type": "general-purpose",
    "name": "qa",
    "model": "haiku",
    "prompt": "Invoke the /qa skill...\n\nProducer SKILL.md: skills/tdd-red-producer/SKILL.md\nIteration: 1",
    "team_name": "issue-105"
  },
  "expected_model": "haiku"
}
```

**Traces to:** REQ-105-002, REQ-105-003, ARCH-105-007, ISSUE-105

---

### DES-105-010: Iteration Limit Enforcement

**Description:** The state machine enforces maximum iteration limits for producer-QA loops to prevent infinite retry cycles.

**Default Limits:**
- Producer-QA iteration limit: **3 attempts**
- After 3rd QA `improvement-request`, state machine returns `action: "escalate-user"` instead of re-spawning producer

**Iteration Tracking:**

State maintained in `internal/state/state.go`:
```go
type PairState struct {
    Phase          string
    Iteration      int    // 0-based (0, 1, 2 = 3 total attempts)
    MaxIterations  int    // default: 3
    QAVerdict      string // "approved", "improvement-request", "escalate-user"
    QAFeedback     string
}
```

**State Machine Logic in `internal/step/next.go`:**

```go
func (e *Engine) Next() (*Action, error) {
    state := e.loadState()

    if state.PairState.Iteration >= state.PairState.MaxIterations {
        return &Action{
            Action: "escalate-user",
            Details: fmt.Sprintf("Max iterations (%d) reached for phase %s",
                state.PairState.MaxIterations, state.PairState.Phase),
        }, nil
    }

    if state.PairState.QAVerdict == "improvement-request" {
        state.PairState.Iteration++
        return &Action{
            Action: "spawn-producer",
            Skill: phaseRegistry[state.PairState.Phase].Producer,
            Iteration: state.PairState.Iteration,
            Context: ActionContext{
                QAFeedback: state.PairState.QAFeedback,
            },
        }, nil
    }

    // ... normal phase progression
}
```

**Orchestrator Handling:**

When `step next` returns `action: "escalate-user"`:
1. Orchestrator displays escalation message to user:
   ```
   QA iteration limit (3) reached for tdd-red phase.
   Remaining issues after 3 attempts:
   - <qa-feedback>

   Options:
   1. Manually fix and continue
   2. Adjust iteration limit and retry
   3. Skip this phase (not recommended)
   ```

2. Orchestrator waits for user decision
3. Does NOT call `projctl step complete` until user provides guidance

**Traces to:** REQ-105-003, ARCH-105-009, DES-105-006, ISSUE-105

---

## Implementation Changes

### DES-105-011: Files to Delete

**Description:** Composite skill directories and their contents are deleted after state machine transitions are verified working.

**Deletion List:**

1. **skills/tdd-producer/**
   - `SKILL.md` (composite orchestrator for TDD cycle)
   - All supporting files in directory

2. **skills/parallel-looper/**
   - Already marked deprecated (ISSUE-83)
   - `SKILL.md` (composite orchestrator for parallel execution)
   - All supporting files in directory

**Deletion Verification Steps:**

Before deletion:
1. Verify `projctl step next` returns correct actions for all TDD sub-phases
2. Run integration tests for full TDD workflow (red → green → refactor → audit)
3. Confirm no references to deleted skills exist:
   ```bash
   grep -r "tdd-producer" skills/ docs/
   grep -r "parallel-looper" skills/ docs/
   ```

4. Test `/project new ISSUE-XXX` end-to-end with real task

After deletion:
1. Remove symlinks from `~/.claude/skills/`:
   ```bash
   rm ~/.claude/skills/tdd-producer
   rm ~/.claude/skills/parallel-looper
   ```

2. Update skill catalog/index if one exists

**Commit Message:**
```
refactor(issue-105): remove composite skill redundancy

Delete tdd-producer and parallel-looper composite skills.
Orchestration now happens via projctl step next state machine.

Deleted:
- skills/tdd-producer/
- skills/parallel-looper/

Traces to: REQ-105-004, DES-105-011, ISSUE-105
```

**Traces to:** REQ-105-004, ISSUE-105

---

### DES-105-012: Orchestrator Documentation Updates

**Description:** Update `/project` orchestrator skill documentation to reflect removal of composite skills and clarify state-machine-driven orchestration pattern.

**File:** `skills/project/SKILL.md`

**Changes:**

1. **Remove composite skill references:**
   - Delete any mentions of `tdd-producer` as a spawnable skill
   - Remove examples showing nested skill spawning
   - Eliminate language about "composite producers"

2. **Clarify orchestration responsibility:**
   - Add explicit statement: "Skills MUST NOT spawn sub-agents via Task tool - orchestration is the orchestrator's job"
   - Document that ALL orchestration happens via `projctl step next` state transitions
   - Skills are stateless executors, not workflow coordinators

3. **Add state-driven producer/QA loop examples:**
   ```markdown
   ### Producer/QA Iteration Pattern

   The orchestrator drives all producer/QA loops via state machine:

   ```
   loop until phase complete:
     1. action = projctl step next
     2. if action == "spawn-producer":
        - Spawn producer with context (issue, prior artifacts, qa feedback)
        - Wait for completion
        - projctl step complete --action spawn-producer --status done
     3. if action == "spawn-qa":
        - Spawn qa with producer SKILL.md + artifacts
        - Wait for verdict
        - projctl step complete --action spawn-qa --qa-verdict <verdict>
     4. if action == "escalate-user":
        - Present issue to user, await guidance
   ```

   The state machine tracks iteration counts and enforces limits.
   ```

4. **Update error recovery section:**
   - Document escalate-user handling
   - Clarify max iteration behavior
   - Add troubleshooting steps for state machine issues

**File:** `skills/project/SKILL-full.md`

**Changes:**

1. **Update phase detail tables:**
   - Replace `tdd` phase entry (which spawned `tdd-producer`) with new sub-phase entries:
     - `tdd-red`, `tdd-red-qa`, `commit-red`, `commit-red-qa`
     - `tdd-green`, `tdd-green-qa`, `commit-green`, `commit-green-qa`
     - `tdd-refactor`, `tdd-refactor-qa`, `commit-refactor`, `commit-refactor-qa`

2. **Update resume map:**
   - Add resume instructions for new TDD sub-phases
   - Document iteration state recovery

**Traces to:** REQ-105-005, ISSUE-105

---

### DES-105-013: Skill Convention Documentation

**Description:** Document architectural rule prohibiting internal Task tool usage for orchestration in skills.

**File:** `docs/skill-conventions.md` (create if doesn't exist)

**Section to Add:**

```markdown
## Orchestration Prohibition

**Rule:** Skills MUST NOT spawn sub-agents via Task tool for workflow orchestration.

**Rationale:**
- Orchestration is the `/project` orchestrator's responsibility
- Composite skills that spawn sub-agents create redundant nesting (ISSUE-105)
- State machine in `projctl step next` is the single source of truth for workflow transitions

**Allowed Task tool usage:**
- Background utility work (e.g., exploration, analysis)
- Isolated tasks that don't coordinate other skills

**Prohibited Task tool usage:**
- Sequential skill spawning (e.g., spawn producer, then QA, then next phase)
- Producer/QA iteration loops
- Multi-phase workflow coordination

**Example - PROHIBITED:**
```markdown
# My Composite Skill

Orchestrates phases A, B, C by spawning sub-skills:

1. Spawn skill-a via Task tool
2. Check result
3. Spawn skill-b via Task tool
4. Check result
5. Spawn skill-c via Task tool
```

**Example - ALLOWED:**
```markdown
# My Leaf Skill

Performs work directly:

1. Read context from spawn prompt
2. Generate output
3. Send completion message
```

**Enforcement:**
- Manual audit during skill reviews
- Grep check: `grep -r "Task(" skills/*/SKILL.md`
- Future: Linter rule to flag Task tool usage in SKILL.md files

**Traces to:** ISSUE-105, REQ-105-004, REQ-105-005
```

**Traces to:** REQ-105-006, ISSUE-105

---

## Validation Strategy

### DES-105-014: Composite Skill Audit Process

**Description:** Systematic audit process to identify all composite skills and verify safe deletion.

**Audit Steps:**

1. **Search for Task tool usage:**
   ```bash
   grep -n "Task(" skills/*/SKILL.md > audit-task-usage.txt
   ```

2. **Analyze each occurrence:**
   - Read context around Task tool call
   - Classify as orchestration vs. utility
   - Note skill name and line numbers

3. **Search for orchestration language:**
   ```bash
   grep -Eni "(nested|composite|orchestrat|coordinat|sequential)" skills/*/SKILL.md > audit-orchestration-terms.txt
   ```

4. **Search for workflow diagrams:**
   - Look for ASCII diagrams showing skill → skill chains
   - Identify producer/QA loop logic

5. **Produce classification report:**

   **Format:**
   ```markdown
   ## Composite Skill Audit Report

   ### Composite Orchestrators (to be deleted)

   #### tdd-producer
   - **File:** skills/tdd-producer/SKILL.md
   - **Evidence:**
     - Line 12: "Orchestrates the complete TDD cycle"
     - Lines 27-49: Nested pair loop diagram (RED/GREEN/REFACTOR)
     - Spawns tdd-red-producer, qa, tdd-green-producer, qa, tdd-refactor-producer, qa
   - **State replacement:** DES-105-008 (TDD sub-phase transitions)

   #### parallel-looper
   - **File:** skills/parallel-looper/SKILL.md
   - **Evidence:**
     - Line 3: Marked [DEPRECATED]
     - Line 69: "FOR EACH item... Invoke: Task(pair-loop)"
     - Spawns multiple PAIR LOOPs in parallel
   - **State replacement:** N/A (deprecated, replaced by native Claude Code teams)

   ### Leaf Skills (preserve)

   #### project
   - **File:** skills/project/SKILL.md
   - **Task usage:** Lines 113-120, 136-143 (spawning teammates per step next)
   - **Classification:** Leaf (top-level orchestrator, not nested composite)

   [... all other skills listed as leaf ...]

   ### Findings Summary

   - **Total skills:** 26
   - **Composite orchestrators:** 2 (tdd-producer, parallel-looper)
   - **Leaf skills:** 24
   - **Skills to delete:** 2
   ```

6. **Verify state machine coverage:**
   - For each composite skill, confirm state transitions exist to replace its orchestration logic
   - Document in design (DES-105-008)

**Traces to:** REQ-105-001, DES-105-007, ISSUE-105

---

### DES-105-015: State Transition Test Coverage

**Description:** Comprehensive test coverage for new TDD sub-phase state transitions and iteration logic.

**Test Files:**

1. **internal/state/transitions_test.go**
   - Test all legal TDD sub-phase transitions
   - Test illegal transition prevention
   - Test full red → green → refactor → audit chain

2. **internal/step/next_test.go**
   - Test `projctl step next` returns correct action for each TDD sub-phase
   - Test iteration increment on improvement-request
   - Test escalate-user on max iterations
   - Test QA feedback propagation to next producer spawn

3. **internal/step/registry_test.go**
   - Verify all TDD sub-phases have registry entries
   - Verify producer/QA skill paths are correct
   - Verify model selections match design

**Test Scenarios:**

| Scenario | Test Coverage |
|----------|---------------|
| Happy path: all QA approved on first try | Legal transitions execute correctly |
| QA requests improvement once | Iteration increments, producer re-spawns with feedback |
| QA requests improvement 3 times | Escalate-user after 3rd iteration |
| Illegal transition attempt (skip QA) | State machine rejects with error |
| Full TDD cycle (red → green → refactor) | All 12 phases execute in order |

**Acceptance Criteria:**

1. All tests pass before deleting composite skills
2. Test coverage for state machine ≥ 90%
3. Integration test validates end-to-end `/project new` workflow

**Traces to:** REQ-105-003, REQ-105-004, ISSUE-105

---

### DES-105-016: Integration Test Strategy

**Description:** End-to-end integration test validating state-driven TDD workflow without composite skills.

**Test Setup:**

1. Create test repository with:
   - Sample issue (ISSUE-TEST-TDD)
   - Simple acceptance criteria
   - Test file location specified

2. Initialize state machine:
   ```bash
   projctl state init --name test-tdd --issue ISSUE-TEST-TDD
   projctl state set --workflow task
   ```

3. Execute orchestrator (automated):
   ```bash
   /project task ISSUE-TEST-TDD
   ```

**Test Assertions:**

1. **State progression:** Verify state transitions through all TDD sub-phases:
   - tdd-red → tdd-red-qa → commit-red → commit-red-qa
   - commit-red-qa → tdd-green → tdd-green-qa → commit-green → commit-green-qa
   - commit-green-qa → tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa
   - commit-refactor-qa → task-audit

2. **Artifact creation:** Verify expected files exist:
   - Test file (from tdd-red)
   - Implementation file (from tdd-green)
   - Refactored implementation (from tdd-refactor)

3. **Git commits:** Verify 3 commits created (one per TDD sub-phase)

4. **No composite skill usage:** Grep for Task tool calls in orchestrator logs:
   ```bash
   grep "tdd-producer" orchestrator.log
   # Should return: no matches
   ```

5. **QA iteration:** Inject QA improvement-request in one phase, verify:
   - Producer re-spawns with feedback
   - Iteration count increments
   - Max iteration triggers escalate-user

**Test Execution:**

Run as part of CI:
```bash
mage test:integration
```

Include in manual pre-release checklist.

**Traces to:** REQ-105-003, REQ-105-004, ISSUE-105

---

## Traceability

**Traces to:** ISSUE-105

**Satisfies requirements:**
- REQ-105-001: Identify All Composite Skills (DES-105-007, DES-105-014)
- REQ-105-002: Define State Machine Transitions (DES-105-008, DES-105-009)
- REQ-105-003: Update State Machine Implementation (DES-105-006, DES-105-010)
- REQ-105-004: Remove Composite Skill Files (DES-105-011)
- REQ-105-005: Update Orchestrator Skill Documentation (DES-105-012)
- REQ-105-006: Validate No Internal Task Tool Usage (DES-105-013)

**Referenced by:** TBD (architecture, test artifacts)

---

## Open Questions

### Q1: Parallel Execution Support

**Question:** With `parallel-looper` removed, how do we handle parallel execution of independent tasks?

**Context:** ISSUE-83 deprecated `parallel-looper` in favor of "native Claude Code team parallelism (ISSUE-79)". The design should clarify whether parallel execution is:
- Already implemented via worktrees + concurrent teammates
- Deferred to a future issue
- Out of scope for ISSUE-105

**Resolution needed before:** Architecture phase

---

### Q2: Backward Compatibility for In-Flight Projects

**Question:** What happens to projects currently in `tdd` phase with `tdd-producer` composite skill when we delete that skill?

**Options:**
1. **Graceful migration:** State machine detects old `phase=tdd`, auto-transitions to `phase=tdd-red`
2. **Hard cutoff:** Projects in `tdd` phase require manual intervention
3. **Version compatibility:** Keep `tdd-producer` until all in-flight projects complete

**Recommendation:** Clarify during architecture design

---

### Q3: Commit-Producer Skill Scope

**Question:** Should commit creation be handled by a dedicated `commit-producer` skill, or by the existing `/commit` skill, or directly by the orchestrator?

**Context:** DES-105-008 shows `commit-red`, `commit-green`, `commit-refactor` phases. Each needs to:
- Stage appropriate files
- Generate commit message
- Create commit

**Options:**
1. **Dedicated commit-producer skill:** Reusable, testable, follows producer/QA pattern
2. **Extend /commit skill:** Use existing skill with phase-aware staging rules
3. **Orchestrator direct:** Orchestrator calls git commands directly (no skill spawn)

**Recommendation:** Evaluate during architecture design (likely impacts ARCH-039, ARCH-040 from ISSUE-92)

---

## Success Metrics

### Functional Success
- All composite skills removed, orchestration happens via state machine
- `projctl step next` drives all producer/QA iteration loops
- Existing workflows (`/project new ISSUE-XXX`) work unchanged from user perspective
- No skills spawn sub-agents via Task tool

### Performance Success
- Reduced latency: Measure agent spawn time before/after (target: 1+ fewer spawn per composite invocation)
- Reduced API costs: Token usage comparison (target: eliminate context load for composite layer)
- Simplified debugging: Single orchestrator conversation trace vs. nested traces

### Quality Success
- All state transitions have unit tests (coverage ≥ 90%)
- Documentation updated to reflect new architecture (no orphaned references)
- Integration test passes for full TDD workflow

**Traces to:** ISSUE-105

---

## Design Summary

| Design ID | Description | Traces to |
|-----------|-------------|-----------|
| DES-105-001 | Invisible architecture changes (user workflow unchanged) | REQ-105-001, REQ-105-002 |
| DES-105-002 | Error message continuity (same format as current) | REQ-105-001 |
| DES-105-003 | Step-by-step progress visibility | REQ-105-002, REQ-105-003 |
| DES-105-004 | Full QA feedback display | REQ-105-002, REQ-105-003 |
| DES-105-005 | State machine orchestration pattern | REQ-105-002, REQ-105-003 |
| DES-105-006 | Producer-QA iteration via state machine | REQ-105-002, REQ-105-003 |
| DES-105-007 | Composite skill identification criteria | REQ-105-001 |
| DES-105-008 | TDD phase state transitions | REQ-105-002 |
| DES-105-009 | Step next action JSON schema | REQ-105-002, REQ-105-003 |
| DES-105-010 | Iteration limit enforcement | REQ-105-003 |
| DES-105-011 | Files to delete | REQ-105-004 |
| DES-105-012 | Orchestrator documentation updates | REQ-105-005 |
| DES-105-013 | Skill convention documentation | REQ-105-006 |
| DES-105-014 | Composite skill audit process | REQ-105-001 |
| DES-105-015 | State transition test coverage | REQ-105-003, REQ-105-004 |
| DES-105-016 | Integration test strategy | REQ-105-003, REQ-105-004 |
