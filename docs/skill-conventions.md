# Skill Conventions

**Status:** Active
**Created:** 2026-02-06
**Issue:** ISSUE-105

**Traces to:** ISSUE-105, REQ-105-006, DES-105-013

---

## Overview

This document defines architectural conventions and rules for skill design in the projctl workflow system. These conventions ensure skills remain focused, composable, and maintainable.

---

## Orchestration Prohibition Rule

### Rule Statement

**Skills MUST NOT spawn sub-agents via Task tool for orchestration purposes.**

### Rationale

**Orchestration is the orchestrator's job:**
- The project orchestrator (`skills/project/SKILL.md`) is the ONLY skill authorized to spawn teammates
- The orchestrator spawns teammates based on instructions from the state machine (`projctl step next`)
- All workflow logic (phase sequencing, iteration, retry) lives in the state machine, not in skills

**Why this rule exists:**
1. **Prevents redundant nesting** - Composite skills that wrap other skills add unnecessary spawn overhead
2. **Centralizes workflow logic** - State machine is the single source of truth for phase transitions
3. **Reduces token cost** - Eliminates context loading for intermediate wrapper agents
4. **Makes iteration explicit** - Retry logic visible in state.toml, not hidden in skill internals
5. **Improves testability** - State machine behavior can be tested independently of skill execution

### Historical Context

**Before ISSUE-105:** Composite skills orchestrated sub-phases internally.

Example: `tdd-producer` (composite):
```
tdd-producer (composite wrapper)
    ↓
    +→ spawn tdd-red-producer
    +→ spawn qa
    +→ spawn tdd-green-producer
    +→ spawn qa
    +→ spawn tdd-refactor-producer
    +→ spawn qa
```

**Problems with this approach:**
- Nested spawning: 7 agents total (1 orchestrator + 1 composite + 5 sub-agents)
- Hidden iteration: tdd-producer decided retry logic internally
- Token overhead: ~11,000 tokens loaded for composite wrapper
- Unclear state: Was iteration tracked in composite or orchestrator?

**After ISSUE-105:** State machine sequences phases directly.

```
project orchestrator
    ↓ (follows projctl step next)
    +→ spawn tdd-red-producer
    +→ spawn qa
    +→ spawn tdd-green-producer
    +→ spawn qa
    +→ spawn tdd-refactor-producer
    +→ spawn qa
```

**Benefits:**
- Flat spawning: 6 agents total (1 orchestrator + 5 workers)
- Explicit iteration: state.toml tracks iteration count and QA feedback
- No token overhead: No wrapper agent context
- Clear state: All workflow state in state.toml

---

## Allowed vs. Prohibited Task Tool Usage

### Prohibited: Orchestration Pattern

**DO NOT** use Task tool to orchestrate producer/QA workflows or sequence sub-phases.

**Examples of prohibited patterns:**

```markdown
# PROHIBITED: Composite skill spawning sub-phases
FOR EACH phase IN [red, green, refactor]:
    1. Spawn producer for phase via Task tool
    2. Wait for completion
    3. Spawn QA via Task tool
    4. Handle QA verdict
```

```markdown
# PROHIBITED: Skill spawning PAIR LOOP
1. Spawn producer via Task tool
2. Spawn QA via Task tool
3. If QA improvement-request, repeat
```

```markdown
# PROHIBITED: Skill spawning parallel orchestration
FOR EACH item IN items:
    Spawn Task(producer, context=item)
Wait for all, aggregate results
```

**Why prohibited:**
- Duplicates state machine responsibility
- Hides workflow state from orchestrator
- Creates nested agent hierarchies
- Makes iteration/retry logic implicit

### Allowed: Utility Pattern

**Allowed use cases for Task tool:**

#### 1. The Project Orchestrator

The `project` orchestrator is the ONLY skill that spawns teammates, and it does so by following state machine instructions:

```markdown
# ALLOWED: Orchestrator following state machine
1. action = projctl step next
2. IF action.action == "spawn-producer":
     Task(subagent_type: action.task_params.subagent_type,
          name: action.task_params.name,
          model: action.task_params.model,
          prompt: action.task_params.prompt)
3. projctl step complete --action spawn-producer --status done
```

**Key distinction:** The orchestrator doesn't decide what to spawn or when. It mechanically executes commands from the state machine.

#### 2. Utility Parallelization

Skills MAY use Task tool to parallelize independent utility operations (NOT producer/QA workflows).

**Example: context-explorer spawning exploration agents**

```markdown
# ALLOWED: Utility skill parallelizing queries
FOR EACH semantic_query IN queries:
    Spawn Task(subagent_type: "explore",
               prompt: query.question)
Collect answers, aggregate results
```

**Why this is allowed:**
- NOT orchestrating workflow phases
- Spawned agents are short-lived explorers, not producers
- No iteration state or QA verdicts
- Returns aggregated data, not workflow outcomes

**Characteristics of allowed utility usage:**
- Spawned agents don't spawn more agents (no nesting)
- No iteration/retry logic (single execution)
- No QA validation of spawned agent outputs
- Used for parallelization, not orchestration
- Returns data, not workflow state

---

## Skill Architecture Patterns

### Leaf Skills (Direct Work)

Most skills are **leaf skills** - they perform direct work without spawning sub-agents.

**Characteristics:**
- Use Read, Write, Edit, Grep, Glob, Bash tools directly
- Produce artifacts (requirements.md, tests, implementation)
- Validate outputs (QA skills)
- No Task tool usage

**Examples:**
- `tdd-red-producer` - Writes tests directly using Read/Write/Edit
- `qa` - Validates artifacts directly using Read and quality checks
- `pm-interview-producer` - Interviews user and writes requirements.md
- `doc-producer` - Generates documentation from code/artifacts

**Pattern:**
```
Producer skill:
1. Read context (prior artifacts, issue description)
2. Generate output (write files, run tests, create commits)
3. Report completion to orchestrator

QA skill:
1. Read producer SKILL.md contract
2. Read producer outputs
3. Validate against contract
4. Return verdict (approved | improvement-request)
```

### The Orchestrator (Team Lead)

The **project orchestrator** is the single skill that coordinates teammates.

**Characteristics:**
- Spawns teammates via Task tool (following state machine)
- Does NOT write files directly
- Does NOT contain workflow logic (delegates to projctl step next)
- Mechanically follows state machine commands

**Pattern:**
```
loop:
  1. action = projctl step next
  2. Execute action:
     - spawn-producer: Spawn producer teammate
     - spawn-qa: Spawn QA teammate
     - escalate-user: Present to user
     - all-complete: Stop
  3. projctl step complete (report outcome)
```

### Utility Skills (Parallelization)

**Utility skills** use Task tool for parallelizing independent operations (NOT workflow orchestration).

**Characteristics:**
- Spawn short-lived agents for specific queries/operations
- No iteration or retry logic
- No QA validation of spawned outputs
- Aggregate results and return

**Example: context-explorer**
```
1. Receive query list from producer
2. Spawn explore agents in parallel (for semantic queries)
3. Execute file/memory/web queries directly
4. Aggregate all results
5. Return to producer
```

**Allowed because:**
- Exploration agents are utilities, not producers
- No workflow state created
- No orchestration of producer/QA pairs
- Returns data, not phase outcomes

---

## Enforcement Mechanisms

### 1. Manual Audit

Audit skills for Task tool usage:

```bash
cd skills
grep -r "Task(" */SKILL.md
```

Classify each match:
- Orchestrator pattern (only project skill - allowed)
- Utility pattern (exploration, parallelization - allowed)
- Orchestration pattern (spawning producers/QA - **prohibited**)

### 2. Grep Check

Verify no prohibited patterns exist:

```bash
# Find all Task tool usage
grep -r "Task(" skills/*/SKILL.md

# Expected results:
# - skills/project/SKILL.md (orchestrator - allowed)
# - skills/context-explorer/SKILL.md (utility - allowed)
# - (nothing else)
```

If additional skills show Task() references, investigate:
- Is it spawning producers/QA? → Prohibited
- Is it spawning utility agents? → Review for approval
- Is it the orchestrator following state machine? → Allowed

### 3. Code Review

When adding new skills or modifying existing ones:

**Checklist:**
- [ ] Does this skill spawn sub-agents via Task tool?
- [ ] If yes, is it the orchestrator following state machine? (allowed)
- [ ] If yes, is it utility parallelization? (needs review)
- [ ] If yes, does it orchestrate producers/QA? (**reject**)

### 4. Future: Automated Linter

**Proposed:** Create `projctl lint skills` command that:
- Parses all SKILL.md files
- Detects Task tool usage
- Flags prohibited orchestration patterns
- Allows orchestrator and approved utility usage

**Implementation deferred** (out of scope for ISSUE-105)

---

## Examples

### ✅ GOOD: Leaf Producer Skill

```markdown
# tdd-red-producer/SKILL.md

## Workflow

1. Read acceptance criteria from task
2. Read existing test files
3. Write failing tests for each AC
4. Run tests to verify they fail correctly
5. Report completion
```

**Why good:** Direct work, no sub-agent spawning.

---

### ✅ GOOD: Orchestrator Following State Machine

```markdown
# project/SKILL.md

## Step Loop

loop:
  1. result = projctl step next
  2. IF result.action == "spawn-producer":
       Task(subagent_type: result.task_params.subagent_type, ...)
  3. projctl step complete --action spawn-producer --status done
```

**Why good:** Orchestrator executes state machine commands, doesn't decide orchestration logic.

---

### ✅ GOOD: Utility Parallelization

```markdown
# context-explorer/SKILL.md

## Workflow

For semantic queries:
  1. Spawn explore agent via Task tool
  2. Agent answers question by reading code

For file queries:
  1. Use Read tool directly

Aggregate all results, return to producer
```

**Why good:** Utility usage for parallelization, not workflow orchestration.

---

### ❌ BAD: Composite Skill Orchestrating Sub-Phases

```markdown
# tdd-producer/SKILL.md (DELETED in ISSUE-105)

## Workflow

1. Spawn tdd-red-producer via Task tool
2. Spawn qa to validate red output
3. If QA approved, spawn tdd-green-producer via Task tool
4. Spawn qa to validate green output
5. If QA approved, spawn tdd-refactor-producer via Task tool
6. Spawn qa to validate refactor output
```

**Why bad:** Orchestrating workflow phases internally. This is the state machine's job.

**Fix:** Delete composite skill. Let state machine sequence tdd-red, tdd-green, tdd-refactor directly.

---

### ❌ BAD: Skill Implementing PAIR LOOP Internally

```markdown
# hypothetical-bad-producer/SKILL.md

## Workflow

iteration = 0
loop:
  1. Do work (write code)
  2. Spawn qa via Task tool
  3. IF qa verdict == "improvement-request":
       iteration++
       goto loop
  4. Report completion
```

**Why bad:** Iteration logic belongs in state machine, not in producer skill.

**Fix:** Producer does work once, returns to orchestrator. State machine handles iteration.

---

## Migration Guide

### For Existing Composite Skills

If you have a composite skill that spawns sub-agents:

**Step 1: Identify sub-phases**

List all sub-agents the composite spawns and their sequence.

Example (tdd-producer):
- tdd-red-producer → qa
- tdd-green-producer → qa
- tdd-refactor-producer → qa

**Step 2: Define state machine phases**

Create phase registry entries for each sub-phase:

```go
// internal/step/registry.go
var phaseRegistry = map[string]PhaseDefinition{
    "tdd-red": {
        Producer: "tdd-red-producer",
        QA: "qa",
        Model: "sonnet",
    },
    "tdd-green": {
        Producer: "tdd-green-producer",
        QA: "qa",
        Model: "sonnet",
    },
    "tdd-refactor": {
        Producer: "tdd-refactor-producer",
        QA: "qa",
        Model: "sonnet",
    },
}
```

**Step 3: Define transitions**

Specify legal phase transitions:

```go
// internal/state/transitions.go
var transitions = map[string][]string{
    "tdd-red": {"tdd-red-qa"},
    "tdd-red-qa": {"commit-red"},
    "commit-red": {"commit-red-qa"},
    "commit-red-qa": {"tdd-green"},
    "tdd-green": {"tdd-green-qa"},
    // ... etc
}
```

**Step 4: Implement backward compatibility**

Auto-migrate legacy state:

```go
// internal/state/state.go
func (sm *StateMachine) loadState() (*State, error) {
    state, _ := sm.loadStateFromFile()

    // Auto-migrate legacy composite phase
    if state.Phase == "tdd" {
        state.Phase = "tdd-red"
        state.Iteration = 0
        sm.saveState(state)
    }

    return state, nil
}
```

**Step 5: Delete composite skill**

```bash
rm -rf skills/tdd-producer/
rm ~/.claude/skills/tdd-producer
```

**Step 6: Update references**

Grep for references to the deleted skill:

```bash
grep -r "tdd-producer" skills/ docs/
```

Update or remove found references.

---

## Decision Log

### D1: Why prohibit internal orchestration?

**Date:** 2026-02-06 (ISSUE-105)

**Decision:** Skills MUST NOT spawn sub-agents for orchestration.

**Rationale:**
- Redundant nesting adds overhead (token cost, spawn latency)
- State machine already knows phase sequence
- Iteration logic should be explicit in state.toml
- Composite skills hide workflow state from orchestrator

**Alternatives considered:**
- Allow composite skills, document them: Rejected (doesn't eliminate redundancy)
- Hybrid (some composite, some direct): Rejected (inconsistent patterns)

**Impact:**
- Deleted tdd-producer, parallel-looper
- All phase sequencing now in state machine
- Clearer separation: orchestrator coordinates, skills work

### D2: Why allow utility parallelization?

**Date:** 2026-02-06 (ISSUE-105)

**Decision:** Utility skills MAY use Task tool for parallelizing operations.

**Rationale:**
- Exploration agents are not producers (no workflow state)
- Parallelization improves performance for independent queries
- No nesting (spawned agents don't spawn more agents)
- Returns data, not phase outcomes

**Example:** context-explorer spawning explore agents for semantic queries.

**Boundary:** If a utility starts orchestrating producers/QA, it's prohibited.

---

## FAQ

### Q1: Can a skill spawn agents for testing purposes?

**A:** No, not in production skills. Testing should use the skill's direct functionality, not spawn sub-agents. If you need integration tests, write them in the skill's test suite, not in the skill logic itself.

### Q2: What if I need to run multiple producers in parallel?

**A:** The orchestrator handles parallel execution. Use the looper pattern in `skills/project/SKILL.md` which spawns N teammates for independent tasks (using git worktrees for isolation). Individual skills should NOT spawn parallel producers.

### Q3: Can a producer skill spawn a QA skill to validate its own output?

**A:** No. The orchestrator handles producer/QA pairing. The state machine sequences:
1. spawn-producer → work done
2. spawn-qa → validation
3. Iterate or advance based on QA verdict

Producers should NOT validate their own output via spawned QA.

### Q4: What about skills that need to "gather context" by spawning exploration agents?

**A:** Use the `context-explorer` utility skill. Producers yield `need-context` with queries, the orchestrator dispatches `context-explorer`, and results are returned to the producer. Producers should NOT spawn explorers directly.

### Q5: I have a complex workflow that needs custom orchestration. Should I create a composite skill?

**A:** No. Complex workflows should be encoded in the state machine:
1. Define phases in `internal/step/registry.go`
2. Define transitions in `internal/state/transitions.go`
3. Let the orchestrator sequence them via `projctl step next`

If the workflow is truly unique and can't fit the state machine model, escalate to discuss architectural changes rather than creating a composite skill.

### Q6: What if the state machine doesn't support my use case?

**A:** Escalate to discuss state machine enhancements. Examples:
- New phase types (beyond producer/qa)
- Conditional branching (beyond linear transitions)
- Parallel phase execution

Do NOT work around limitations by creating composite skills.

---

## References

- **ISSUE-105:** Remove Composite Skill Redundancy
- **skills/project/SKILL.md:** Orchestrator documentation
- **docs/composite-skill-audit.md:** Audit results identifying composite skills
- **internal/step/registry.go:** Phase registry implementation
- **internal/state/transitions.go:** Transition table implementation

---

## Traceability

**Traces to:** ISSUE-105, REQ-105-006, DES-105-013

**Satisfies:**
- REQ-105-006: Document skill conventions prohibiting internal orchestration
- DES-105-013: Create docs/skill-conventions.md with orchestration rule

**Referenced by:**
- TASK-23: Create skill convention documentation
- TASK-24: Validate no remaining Task tool orchestration usage
