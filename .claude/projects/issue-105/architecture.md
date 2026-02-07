# Architecture - ISSUE-105: Remove Composite Skill Redundancy

**Status:** Draft
**Created:** 2026-02-06
**Issue:** ISSUE-105

**Traces to:** ISSUE-105, REQ-105-002, REQ-105-003, DES-105-005, DES-105-006, DES-105-008

---

## Executive Summary

This architecture eliminates composite skills (tdd-producer, parallel-looper) by implementing state machine-driven orchestration where `projctl step next` dictates all workflow transitions. The orchestrator becomes stateless - it simply executes commands returned by the state machine. All iteration tracking, retry logic, and phase sequencing live in `state.toml` and the state machine implementation.

**Key Decision:** Stateless orchestrator with iteration tracking in `projctl step next` (Option 1 from architecture interview).

---

## Architecture Overview

### ARCH-105-001: System Layering

**Description:** Three-layer architecture separating user interface, orchestration control, and state persistence.

**Layers:**

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: User Interface                                      │
│ - /project skill (CLI entry point)                          │
│ - User commands: /project new|task|adopt|align               │
│ - Progress reporting and error display                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Orchestration Engine                                │
│ - State machine (projctl step next)                         │
│ - Phase registry (phase definitions)                         │
│ - Transition logic (legal state transitions)                 │
│ - Iteration enforcement (max retry limits)                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Persistent State                                    │
│ - state.toml (current phase, iteration count, QA verdict)   │
│ - Artifact tracking (producer outputs, QA feedback)          │
│ - Team configuration (spawned teammates)                     │
└─────────────────────────────────────────────────────────────┘
```

**Responsibilities:**

- **Layer 1 (UI):** Displays progress, handles user input, spawns teammates per Layer 2 instructions
- **Layer 2 (Engine):** Owns ALL workflow logic, determines next action based on state
- **Layer 3 (State):** Persists workflow state, survives process restarts

**Eliminated Layer:**
- ~~Composite skill layer~~ (tdd-producer, parallel-looper) - redundant orchestration removed

**Traces to:** REQ-105-002, REQ-105-003, DES-105-005, ISSUE-105

---

### ARCH-105-002: Stateless Orchestrator Pattern

**Description:** The `/project` orchestrator skill maintains NO internal state. All workflow state lives in `state.toml` and is accessed via `projctl step next` / `projctl step complete`.

**Architecture Decision:** Stateless orchestrator (Option 1 from interview)

**Orchestrator Behavior:**

```
loop until workflow complete:
  1. action = projctl step next          # State machine reads state.toml
  2. if action.action == "spawn-producer":
       - Spawn producer teammate with action.task_params
       - Wait for completion
       - projctl step complete --action spawn-producer --status done
  3. if action.action == "spawn-qa":
       - Spawn QA teammate with action.task_params
       - Wait for verdict
       - projctl step complete --action spawn-qa --qa-verdict <verdict> --qa-feedback <feedback>
  4. if action.action == "escalate-user":
       - Display error, await user intervention
       - User provides guidance
       - projctl step complete --action escalate-user --user-decision <decision>
  5. if action.action == "all-complete":
       - Report completion, exit loop
```

**Key Properties:**

1. **No local variables:** Orchestrator stores nothing between `step next` calls
2. **Deterministic:** Same state.toml always produces same `step next` output
3. **Resumable:** Orchestrator can crash and resume from state.toml without loss
4. **Testable:** State machine behavior can be tested independently

**Why Stateless:**

- Matches existing `projctl step next` pattern
- Simplifies orchestrator implementation (just follow instructions)
- Centralizes workflow logic in state machine (single source of truth)
- Enables replay and debugging from state snapshots

**Traces to:** REQ-105-003, DES-105-005, DES-105-006, ISSUE-105

---

### ARCH-105-003: State Machine Contract

**Description:** `projctl step next` is the single source of truth for workflow transitions. It reads `state.toml`, applies transition logic, and returns the next action for the orchestrator to execute.

**Input:** None (reads from `.claude/projects/<issue>/state.toml`)

**Output:** JSON action object

```go
type Action struct {
    Action        string                 // "spawn-producer" | "spawn-qa" | "commit" | "transition" | "escalate-user" | "all-complete"
    Skill         string                 // Skill name to spawn (for spawn-* actions)
    SkillPath     string                 // Path to SKILL.md
    Model         string                 // Model to use for teammate
    Phase         string                 // Current phase
    Iteration     int                    // Current iteration count (0-based)
    Context       ActionContext          // Context for spawned skill
    TaskParams    TaskParams             // Parameters for Task tool call
    ExpectedModel string                 // Model verification
    Details       string                 // Escalation details (for escalate-user)
}

type ActionContext struct {
    Issue          string   // ISSUE-NNN
    PriorArtifacts []string // Paths to existing artifacts
    QAFeedback     string   // Feedback from prior QA iteration (if any)
}

type TaskParams struct {
    SubagentType string // "general-purpose"
    Name         string // Teammate name
    Model        string // Model selection
    Prompt       string // Full spawn prompt with context
    TeamName     string // Team identifier
}
```

**State Transitions:**

The state machine enforces legal transitions and prevents illegal jumps:

```
tdd-red → tdd-red-qa → commit-red → commit-red-qa → tdd-green → ...
  ✓ Legal: tdd-red → tdd-red-qa
  ✗ Illegal: tdd-red → commit-red (must go through QA)
  ✗ Illegal: tdd-red → tdd-green (must complete commit-red cycle)
```

**Iteration Logic:**

```go
// Pseudo-code for iteration handling
func (sm *StateMachine) Next() (*Action, error) {
    state := sm.loadState()

    // Check max iterations
    if state.Iteration >= state.MaxIterations {
        return &Action{
            Action: "escalate-user",
            Details: fmt.Sprintf("Max iterations (%d) reached for %s",
                state.MaxIterations, state.Phase),
        }, nil
    }

    // Check QA verdict
    if state.QAVerdict == "improvement-request" {
        state.Iteration++ // Increment for retry
        sm.saveState(state)
        return &Action{
            Action: "spawn-producer",
            Skill: phaseRegistry[state.Phase].Producer,
            Iteration: state.Iteration,
            Context: ActionContext{
                QAFeedback: state.QAFeedback,
            },
        }, nil
    }

    if state.QAVerdict == "approved" {
        // Advance to next phase
        nextPhase := transitions[state.Phase][0]
        state.Phase = nextPhase
        state.Iteration = 0 // Reset for new phase
        sm.saveState(state)
        return sm.Next() // Recurse to get action for new phase
    }

    // Normal phase action
    return sm.phaseAction(state.Phase), nil
}
```

**Traces to:** REQ-105-003, DES-105-006, DES-105-009, DES-105-010, ISSUE-105

---

## Component Architecture

### ARCH-105-004: State Storage Schema

**Description:** `state.toml` format for workflow state persistence.

**File Location:** `.claude/projects/<issue>/state.toml`

**Schema:**

```toml
[workflow]
type = "task"                    # new | task | adopt | align
issue = "ISSUE-105"
status = "in-progress"          # in-progress | complete | escalated

[phase]
current = "tdd-red"              # Current phase identifier
iteration = 0                    # Current iteration count (0-based)
max_iterations = 3               # Max producer/QA retries before escalation

[qa]
verdict = ""                     # "approved" | "improvement-request" | ""
feedback = ""                    # QA feedback text for producer retry

[artifacts]
prior = [                        # Artifacts from completed phases
    "requirements.md",
    "design.md",
    "architecture.md"
]
current = []                     # Artifacts produced in current phase

[team]
name = "issue-105"
lead = "team-lead"
members = ["tdd-red-producer", "qa"]
```

**State Updates:**

Via `projctl step complete`:

```bash
# Producer completion
projctl step complete --action spawn-producer --status done

# QA approval
projctl step complete --action spawn-qa --qa-verdict approved

# QA improvement request
projctl step complete --action spawn-qa --qa-verdict improvement-request \
  --qa-feedback "Missing trace to REQ-042"

# User escalation resolution
projctl step complete --action escalate-user --user-decision continue
```

**Traces to:** REQ-105-003, DES-105-006, ARCH-105-003, ISSUE-105

---

### ARCH-105-005: Phase Registry

**Description:** Phase definitions mapping phase identifiers to producer/QA skills, models, and file paths.

**Implementation:** `internal/step/registry.go`

**Registry Structure:**

```go
type PhaseDefinition struct {
    Producer     string // Producer skill name
    ProducerPath string // Path to producer SKILL.md
    QA           string // QA skill name (usually "qa")
    QAPath       string // Path to QA SKILL.md
    Model        string // Model for producer ("sonnet" | "haiku" | "opus")
    QAModel      string // Model for QA (usually "haiku")
}

var phaseRegistry = map[string]PhaseDefinition{
    // TDD Red
    "tdd-red": {
        Producer:     "tdd-red-producer",
        ProducerPath: "skills/tdd-red-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "sonnet",
        QAModel:      "haiku",
    },

    // Commit Red
    "commit-red": {
        Producer:     "commit-producer",
        ProducerPath: "skills/commit-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "haiku",
        QAModel:      "haiku",
    },

    // TDD Green
    "tdd-green": {
        Producer:     "tdd-green-producer",
        ProducerPath: "skills/tdd-green-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "sonnet",
        QAModel:      "haiku",
    },

    // Commit Green
    "commit-green": {
        Producer:     "commit-producer",
        ProducerPath: "skills/commit-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "haiku",
        QAModel:      "haiku",
    },

    // TDD Refactor
    "tdd-refactor": {
        Producer:     "tdd-refactor-producer",
        ProducerPath: "skills/tdd-refactor-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "sonnet",
        QAModel:      "haiku",
    },

    // Commit Refactor
    "commit-refactor": {
        Producer:     "commit-producer",
        ProducerPath: "skills/commit-producer/SKILL.md",
        QA:           "qa",
        QAPath:       "skills/qa/SKILL.md",
        Model:        "haiku",
        QAModel:      "haiku",
    },

    // Other phases...
    "req-interview": { /* ... */ },
    "req-producer": { /* ... */ },
    "design-interview": { /* ... */ },
    "design-producer": { /* ... */ },
    // ... etc
}
```

**Phase Naming Convention:**

- `<artifact>-interview` - Interview phase (e.g., req-interview, design-interview)
- `<artifact>-producer` - Producer phase (e.g., req-producer, design-producer)
- `<artifact>-qa` - QA phase (e.g., req-qa, design-qa)
- `commit-<stage>` - Commit phase (e.g., commit-red, commit-green)
- `tdd-<stage>` - TDD producer phase (e.g., tdd-red, tdd-green, tdd-refactor)

**Traces to:** REQ-105-002, DES-105-008, ISSUE-105

---

### ARCH-105-006: Transition Table

**Description:** Legal state transitions enforced by the state machine.

**Implementation:** `internal/state/transitions.go`

**Transition Table:**

```go
var transitions = map[string][]string{
    // Requirements workflow
    "req-interview":     {"req-producer"},
    "req-producer":      {"req-qa"},
    "req-qa":            {"design-interview"},

    // Design workflow
    "design-interview":  {"design-producer"},
    "design-producer":   {"design-qa"},
    "design-qa":         {"arch-interview"},

    // Architecture workflow
    "arch-interview":    {"arch-producer"},
    "arch-producer":     {"arch-qa"},
    "arch-qa":           {"tdd-red"},

    // TDD Red workflow
    "tdd-red":           {"tdd-red-qa"},
    "tdd-red-qa":        {"commit-red"},
    "commit-red":        {"commit-red-qa"},
    "commit-red-qa":     {"tdd-green"},

    // TDD Green workflow
    "tdd-green":         {"tdd-green-qa"},
    "tdd-green-qa":      {"commit-green"},
    "commit-green":      {"commit-green-qa"},
    "commit-green-qa":   {"tdd-refactor"},

    // TDD Refactor workflow
    "tdd-refactor":      {"tdd-refactor-qa"},
    "tdd-refactor-qa":   {"commit-refactor"},
    "commit-refactor":   {"commit-refactor-qa"},
    "commit-refactor-qa": {"task-audit"},

    // Audit workflow
    "task-audit":        {"task-audit-qa"},
    "task-audit-qa":     {"retro"},

    // Retrospective
    "retro":             {"complete"},

    // Terminal state
    "complete":          {},
}
```

**Validation:**

```go
func (sm *StateMachine) isLegalTransition(from, to string) bool {
    allowed, ok := transitions[from]
    if !ok {
        return false // Unknown source phase
    }

    for _, legal := range allowed {
        if legal == to {
            return true
        }
    }

    return false
}
```

**Illegal Transition Examples:**

- `tdd-red` → `commit-red` (must go through `tdd-red-qa`)
- `tdd-red` → `tdd-green` (must complete commit-red cycle)
- `req-qa` → `arch-interview` (must go through design phases)

**Traces to:** REQ-105-002, REQ-105-003, DES-105-008, ARCH-105-003, ISSUE-105

---

### ARCH-105-007: Iteration Enforcement

**Description:** Max iteration limits prevent infinite producer/QA retry loops.

**Default Limits:**

- `max_iterations = 3` (allows 4 total producer runs: 0, 1, 2, 3)
- Configurable per-phase in future (not in ISSUE-105 scope)

**Enforcement Logic:**

```go
func (sm *StateMachine) Next() (*Action, error) {
    state := sm.loadState()

    // Check iteration limit
    if state.Iteration >= state.MaxIterations {
        return &Action{
            Action: "escalate-user",
            Phase: state.Phase,
            Iteration: state.Iteration,
            Details: fmt.Sprintf(
                "Max iterations (%d) reached for phase %s.\n" +
                "Remaining issues after %d attempts:\n%s",
                state.MaxIterations,
                state.Phase,
                state.Iteration + 1,
                state.QAFeedback,
            ),
        }, nil
    }

    // Normal iteration handling
    if state.QAVerdict == "improvement-request" {
        state.Iteration++
        sm.saveState(state)

        return &Action{
            Action: "spawn-producer",
            Skill: phaseRegistry[state.Phase].Producer,
            Iteration: state.Iteration,
            Context: ActionContext{
                Issue: state.Issue,
                PriorArtifacts: state.PriorArtifacts,
                QAFeedback: state.QAFeedback,
            },
        }, nil
    }

    // ... other logic
}
```

**Escalation Handling:**

When `action: "escalate-user"` is returned:

1. Orchestrator displays error message to user
2. Shows QA feedback and iteration count
3. Presents options:
   - Manually fix and continue
   - Adjust iteration limit and retry
   - Skip phase (not recommended)
4. Waits for user decision
5. Calls `projctl step complete --action escalate-user --user-decision <choice>`

**Iteration Reset:**

Iteration counter resets to 0 on:
- Phase transition (approved QA verdict)
- User manual fix + continue

**Traces to:** REQ-105-003, DES-105-006, DES-105-010, ARCH-105-003, ISSUE-105

---

## Workflow Architecture

### ARCH-105-008: TDD Workflow State Machine

**Description:** Complete TDD workflow with RED, GREEN, REFACTOR sub-phases replacing composite `tdd-producer` skill.

**Current Architecture (Composite - REMOVED):**

```
phase=tdd
  → spawn tdd-producer (composite skill)
    [internally: spawn tdd-red-producer, qa, tdd-green-producer, qa, tdd-refactor-producer, qa]
  → tdd-producer completes
  → phase=commit-tdd
```

**New Architecture (State-Driven):**

```
phase=tdd-red, iteration=0
  → spawn tdd-red-producer
  → spawn qa (tdd-red-qa)
  → verdict: approved

phase=commit-red, iteration=0
  → spawn commit-producer
  → spawn qa (commit-red-qa)
  → verdict: approved

phase=tdd-green, iteration=0
  → spawn tdd-green-producer
  → spawn qa (tdd-green-qa)
  → verdict: improvement-request

phase=tdd-green, iteration=1
  → spawn tdd-green-producer (with qa-feedback)
  → spawn qa (tdd-green-qa)
  → verdict: approved

phase=commit-green, iteration=0
  → spawn commit-producer
  → spawn qa (commit-green-qa)
  → verdict: approved

phase=tdd-refactor, iteration=0
  → spawn tdd-refactor-producer
  → spawn qa (tdd-refactor-qa)
  → verdict: approved

phase=commit-refactor, iteration=0
  → spawn commit-producer
  → spawn qa (commit-refactor-qa)
  → verdict: approved

phase=task-audit
  → ...
```

**Phase Sequence:**

1. `tdd-red` → `tdd-red-qa` → `commit-red` → `commit-red-qa`
2. `commit-red-qa` → `tdd-green` → `tdd-green-qa` → `commit-green` → `commit-green-qa`
3. `commit-green-qa` → `tdd-refactor` → `tdd-refactor-qa` → `commit-refactor` → `commit-refactor-qa`
4. `commit-refactor-qa` → `task-audit`

**Benefits:**

- Eliminates 3 agent spawns (tdd-producer, nested producers, nested QAs)
- Makes iteration logic explicit and testable
- Allows independent commit after each TDD stage
- Enables fine-grained progress tracking

**Traces to:** REQ-105-002, REQ-105-003, DES-105-005, DES-105-008, ARCH-105-005, ARCH-105-006, ISSUE-105

---

### ARCH-105-009: Producer/QA Iteration Pattern

**Description:** Standard producer/QA feedback loop driven by state machine.

**Iteration Flow:**

```
┌─────────────────────────────────────────────────────────────┐
│ Phase Start (iteration=0)                                    │
└─────────────────────────────────────────────────────────────┘
                     ↓
         ┌───────────────────────┐
         │ Spawn Producer        │
         │ (with prior QA        │
         │  feedback if iter>0)  │
         └───────────────────────┘
                     ↓
         ┌───────────────────────┐
         │ Producer completes    │
         │ (artifacts created)   │
         └───────────────────────┘
                     ↓
         ┌───────────────────────┐
         │ Spawn QA              │
         │ (validate producer    │
         │  output)              │
         └───────────────────────┘
                     ↓
         ┌───────────────────────┐
         │ QA returns verdict    │
         └───────────────────────┘
                     ↓
        ┌────────────┴────────────┐
        ↓                         ↓
┌──────────────┐          ┌──────────────────┐
│ Approved     │          │ Improvement      │
│              │          │ Request          │
└──────────────┘          └──────────────────┘
        ↓                         ↓
        ↓                 ┌───────────────────┐
        ↓                 │ Iteration++       │
        ↓                 │ if iter < max:    │
        ↓                 │   Re-spawn        │
        ↓                 │   producer with   │
        ↓                 │   feedback        │
        ↓                 │ else:             │
        ↓                 │   Escalate to user│
        ↓                 └───────────────────┘
        ↓                         ↓
        ↓                         ↑ (loop)
        ↓                         │
        ↓←────────────────────────┘
        ↓
┌──────────────────┐
│ Advance to next  │
│ phase (reset     │
│ iteration=0)     │
└──────────────────┘
```

**State Machine Decisions:**

| QA Verdict | Iteration | State Machine Action |
|------------|-----------|----------------------|
| approved | any | Advance to next phase, reset iteration=0 |
| improvement-request | < max | Increment iteration, re-spawn producer with feedback |
| improvement-request | >= max | Return escalate-user action |

**Traces to:** REQ-105-003, DES-105-006, DES-105-010, ARCH-105-003, ARCH-105-007, ISSUE-105

---

## Implementation Architecture

### ARCH-105-010: File Organization

**Description:** File structure for state machine implementation.

**Directory Structure:**

```
internal/
├── state/
│   ├── state.go              # State loading/saving, TOML parsing
│   ├── state_test.go         # State persistence tests
│   ├── transitions.go        # Transition table, validation
│   └── transitions_test.go   # Transition logic tests
│
├── step/
│   ├── registry.go           # Phase registry (phase → skill mappings)
│   ├── registry_test.go      # Registry validation tests
│   ├── next.go               # projctl step next implementation
│   ├── next_test.go          # Next action logic tests
│   ├── complete.go           # projctl step complete implementation
│   └── complete_test.go      # Complete action tests
│
└── cmd/
    └── projctl/
        ├── step_next.go      # CLI wrapper for step next
        └── step_complete.go  # CLI wrapper for step complete
```

**Separation of Concerns:**

- `internal/state/` - State persistence and transition rules (pure logic)
- `internal/step/` - Step command implementation (uses state package)
- `internal/cmd/projctl/` - CLI interface (thin wrapper over step package)

**Traces to:** REQ-105-003, ISSUE-105

---

### ARCH-105-011: Module Dependencies

**Description:** Dependency graph for state machine components.

**Dependency Graph:**

```
cmd/projctl/step_next.go
    ↓
internal/step/next.go
    ↓
    ├→ internal/step/registry.go
    └→ internal/state/state.go
           ↓
       internal/state/transitions.go
```

**Key Dependencies:**

- `step/next.go` imports `state`, `registry`
- `state/state.go` imports `transitions`
- No circular dependencies

**External Dependencies:**

- `github.com/pelletier/go-toml/v2` - TOML parsing for state.toml
- `github.com/onsi/gomega` - Test matchers
- `pgregory.net/rapid` - Property-based testing

**Traces to:** REQ-105-003, ISSUE-105

---

## Migration Architecture

### ARCH-105-012: Backward Compatibility Strategy

**Description:** Handling in-flight projects currently in `tdd` phase when `tdd-producer` is deleted.

**Problem:** Projects with `phase=tdd` in state.toml will have no composite skill to spawn after deletion.

**Solution:** Auto-migration in state machine

**Migration Logic:**

```go
func (sm *StateMachine) loadState() (*State, error) {
    state, err := sm.loadStateFromFile()
    if err != nil {
        return nil, err
    }

    // Auto-migrate legacy phases
    if state.Phase == "tdd" {
        log.Info("Auto-migrating legacy phase 'tdd' to 'tdd-red'")
        state.Phase = "tdd-red"
        state.Iteration = 0
        sm.saveState(state) // Persist migration
    }

    return state, nil
}
```

**Migration Table:**

| Legacy Phase | Migrated Phase | Rationale |
|--------------|----------------|-----------|
| `tdd` | `tdd-red` | Start at beginning of TDD cycle |

**User Impact:**

- Transparent (no user action required)
- Logged for debugging
- One-time migration per project
- State persisted after migration

**Future:** If more composite phases are discovered, add migrations here.

**Traces to:** REQ-105-004, DES-105-008, ISSUE-105

---

### ARCH-105-013: Composite Skill Deletion Sequence

**Description:** Safe deletion order for composite skills and supporting code.

**Deletion Phases:**

**Phase 1: State Machine Implementation**
1. Implement new TDD sub-phases in registry
2. Implement transition table with new phases
3. Implement iteration logic in `step next`
4. Add backward compatibility migration
5. Write comprehensive tests

**Phase 2: Verification**
1. Run unit tests (all pass)
2. Run integration test (full TDD workflow)
3. Manual test with real issue
4. Verify migration works for legacy state

**Phase 3: Skill Deletion**
1. Delete `skills/tdd-producer/` directory
2. Delete `skills/parallel-looper/` directory
3. Remove symlinks:
   - `~/.claude/skills/tdd-producer`
   - `~/.claude/skills/parallel-looper`
4. Grep verification:
   ```bash
   grep -r "tdd-producer" skills/ docs/
   grep -r "parallel-looper" skills/ docs/
   ```
5. Remove references found in grep

**Phase 4: Documentation Update**
1. Update `skills/project/SKILL.md`
2. Update `skills/project/SKILL-full.md`
3. Create/update `docs/skill-conventions.md`
4. Update workflow diagrams

**Phase 5: Validation**
1. Run full test suite
2. Integration test
3. Manual end-to-end test
4. Commit with traceability

**Traces to:** REQ-105-004, DES-105-011, ISSUE-105

---

## Quality Architecture

### ARCH-105-014: Test Strategy

**Description:** Comprehensive test coverage for state machine implementation.

**Test Levels:**

**Unit Tests:**

1. `internal/state/state_test.go`
   - TOML parsing (valid/invalid state files)
   - State loading/saving
   - Field validation

2. `internal/state/transitions_test.go`
   - Legal transition acceptance
   - Illegal transition rejection
   - Full phase chain validation

3. `internal/step/registry_test.go`
   - All phases have registry entries
   - Producer/QA skill paths exist
   - Model selections are valid

4. `internal/step/next_test.go`
   - Correct action for each phase
   - Iteration increment on improvement-request
   - Escalate-user on max iterations
   - QA feedback propagation

**Property-Based Tests:**

Using `pgregory.net/rapid`:

1. State machine never gets stuck (always returns valid action)
2. All legal transition sequences eventually reach "complete"
3. Iteration counter never decreases
4. Max iteration limit always enforced

**Integration Tests:**

1. Full TDD workflow (red → green → refactor → audit)
2. QA iteration with feedback
3. Max iteration escalation
4. Backward compatibility migration

**Test Coverage Target:** ≥ 90% for state machine code

**Traces to:** REQ-105-003, DES-105-015, DES-105-016, ISSUE-105

---

### ARCH-105-015: Error Handling Architecture

**Description:** Comprehensive error handling for state machine operations.

**Error Categories:**

**1. State Loading Errors**

```go
// Missing state file
if !fileExists(stateFile) {
    return nil, fmt.Errorf("state file not found: %s (run 'projctl state init')", stateFile)
}

// Malformed TOML
if err := toml.Unmarshal(data, &state); err != nil {
    return nil, fmt.Errorf("invalid state.toml: %w", err)
}
```

**2. Transition Errors**

```go
// Illegal transition attempt
if !sm.isLegalTransition(currentPhase, requestedPhase) {
    return fmt.Errorf("illegal transition: %s → %s (allowed: %v)",
        currentPhase, requestedPhase, transitions[currentPhase])
}
```

**3. Phase Registry Errors**

```go
// Unknown phase
if _, ok := phaseRegistry[phase]; !ok {
    return fmt.Errorf("unknown phase: %s (check registry)", phase)
}

// Missing skill file
if !fileExists(skillPath) {
    return fmt.Errorf("skill file not found: %s", skillPath)
}
```

**4. Iteration Limit Errors**

```go
// Max iterations reached (not an error - returns escalate-user action)
if state.Iteration >= state.MaxIterations {
    return &Action{Action: "escalate-user", Details: "..."}
}
```

**Error Propagation:**

- CLI layer catches errors, displays user-friendly messages
- State machine layer returns structured errors with context
- Logs include full error chain for debugging

**Traces to:** REQ-105-003, DES-105-002, ISSUE-105

---

## Performance Architecture

### ARCH-105-016: Latency Optimization

**Description:** Performance improvements from eliminating composite skill layer.

**Current Latency (with composite tdd-producer):**

```
User command
  → spawn team-lead (200ms)
  → spawn tdd-producer (200ms)
    → spawn tdd-red-producer (200ms)
      → spawn qa (200ms)
    → spawn tdd-green-producer (200ms)
      → spawn qa (200ms)
    → spawn tdd-refactor-producer (200ms)
      → spawn qa (200ms)

Total: 1600ms overhead (8 spawns × 200ms)
```

**New Latency (state-driven):**

```
User command
  → spawn team-lead (200ms)
  → spawn tdd-red-producer (200ms)
  → spawn qa (200ms)
  → spawn tdd-green-producer (200ms)
  → spawn qa (200ms)
  → spawn tdd-refactor-producer (200ms)
  → spawn qa (200ms)

Total: 1400ms overhead (7 spawns × 200ms)
```

**Savings:** 200ms per composite skill eliminated (1 spawn removed)

**Additional Savings:**

- Context loading: ~500ms per eliminated agent
- Token processing: Varies based on context size

**Total Estimated Savings:** ~700ms per composite skill invocation

**Traces to:** DES-105-005, ISSUE-105

---

### ARCH-105-017: Token Cost Optimization

**Description:** API cost reduction from eliminating redundant context loading.

**Current Token Usage (with composite):**

```
Composite skill context:
  - System prompt: ~2000 tokens
  - SKILL.md: ~3000 tokens
  - Issue context: ~1000 tokens
  - Prior artifacts: ~5000 tokens

Total: ~11,000 tokens loaded for composite skill
       Then duplicated for each sub-skill spawn
```

**New Token Usage (state-driven):**

```
Leaf skill context:
  - System prompt: ~2000 tokens
  - SKILL.md: ~3000 tokens
  - Issue context: ~1000 tokens
  - Prior artifacts: ~5000 tokens

Total: ~11,000 tokens per leaf skill (no composite layer)
```

**Savings:** ~11,000 tokens per composite skill eliminated

**For tdd-producer:** ~11,000 tokens saved per TDD workflow

**Traces to:** DES-105-005, ISSUE-105

---

## Security & Validation Architecture

### ARCH-105-018: State Validation

**Description:** Validation rules for state.toml integrity.

**Validation Checks:**

```go
func (s *State) Validate() error {
    // Required fields
    if s.Workflow.Type == "" {
        return fmt.Errorf("workflow.type is required")
    }
    if s.Workflow.Issue == "" {
        return fmt.Errorf("workflow.issue is required")
    }
    if s.Phase.Current == "" {
        return fmt.Errorf("phase.current is required")
    }

    // Phase exists in registry
    if _, ok := phaseRegistry[s.Phase.Current]; !ok {
        return fmt.Errorf("unknown phase: %s", s.Phase.Current)
    }

    // Iteration non-negative
    if s.Phase.Iteration < 0 {
        return fmt.Errorf("phase.iteration must be >= 0, got: %d", s.Phase.Iteration)
    }

    // Valid QA verdict
    validVerdicts := []string{"", "approved", "improvement-request"}
    if !contains(validVerdicts, s.QA.Verdict) {
        return fmt.Errorf("invalid qa.verdict: %s (allowed: %v)",
            s.QA.Verdict, validVerdicts)
    }

    return nil
}
```

**Validation Timing:**

- On load: Before returning state to caller
- On save: Before writing to file
- On transition: Before applying state change

**Traces to:** REQ-105-003, ISSUE-105

---

## Traceability

**Traces to:** ISSUE-105

**Satisfies Requirements:**
- REQ-105-002: Define State Machine Transitions
  - ARCH-105-003, ARCH-105-005, ARCH-105-006, ARCH-105-008
- REQ-105-003: Update State Machine Implementation
  - ARCH-105-002, ARCH-105-003, ARCH-105-004, ARCH-105-007, ARCH-105-009, ARCH-105-010, ARCH-105-011, ARCH-105-014, ARCH-105-015, ARCH-105-018
- REQ-105-004: Remove Composite Skill Files
  - ARCH-105-012, ARCH-105-013

**Implements Design:**
- DES-105-005: State Machine Orchestration Pattern
  - ARCH-105-001, ARCH-105-002, ARCH-105-003
- DES-105-006: Producer-QA Iteration via State Machine
  - ARCH-105-007, ARCH-105-009
- DES-105-008: TDD Phase State Transitions
  - ARCH-105-005, ARCH-105-006, ARCH-105-008
- DES-105-009: Step Next Action JSON Schema
  - ARCH-105-003
- DES-105-010: Iteration Limit Enforcement
  - ARCH-105-007

---

## Architecture Decision Records

### ADR-105-001: Stateless Orchestrator with State Machine Control

**Status:** Accepted

**Context:** Need to decide whether orchestrator maintains internal state (iteration counters, retry logic) or delegates all state to `projctl step next`.

**Decision:** Stateless orchestrator (Option 1 from interview)

**Rationale:**
1. Matches existing `projctl step next` pattern
2. Simplifies orchestrator (just follow instructions)
3. Centralizes workflow logic in state machine
4. Enables replay and debugging from state snapshots
5. Makes orchestrator deterministic and testable

**Consequences:**
- All iteration tracking in state.toml
- State machine owns all workflow logic
- Orchestrator becomes thin execution layer
- Easier to test state machine independently

**Alternatives Considered:**
- Option 2: Stateful orchestrator (rejected - duplicates state)
- Option 3: Hybrid (rejected - split responsibility)

**Traces to:** ARCH-105-002, ARCH-105-003, DES-105-006, ISSUE-105

---

## Open Issues

### OI-105-001: Parallel Execution After parallel-looper Deletion

**Question:** How do we handle parallel execution of independent tasks after deleting `parallel-looper`?

**Context:** DES-105-011 lists `parallel-looper` for deletion. ISSUE-83 deprecated it in favor of "native Claude Code team parallelism (ISSUE-79)".

**Options:**
1. Parallel execution already implemented via worktrees + concurrent teammates
2. Parallel execution deferred to ISSUE-79
3. Parallel execution out of scope for current architecture

**Resolution:** Clarify in ISSUE-79 or create new architecture issue

**Impact:** Low (parallel-looper already deprecated)

**Traces to:** REQ-105-004, DES-105-011, ISSUE-105

---

### OI-105-002: Commit-Producer Skill Scope

**Question:** Should commit creation use dedicated `commit-producer` skill, existing `/commit` skill, or direct orchestrator calls?

**Context:** ARCH-105-008 shows `commit-red`, `commit-green`, `commit-refactor` phases. Each needs staging, message generation, and commit creation.

**Options:**
1. Dedicated `commit-producer` skill (reusable, testable, follows producer/QA pattern)
2. Extend `/commit` skill (use existing skill with phase-aware staging)
3. Orchestrator direct (no skill spawn, orchestrator calls git directly)

**Current State:** ARCH-105-005 registry assumes `commit-producer` skill exists

**Resolution Needed:** Before TDD implementation (ISSUE-105 or follow-up)

**Impact:** Medium (affects ARCH-039, ARCH-040 from ISSUE-92)

**Traces to:** ARCH-105-005, ARCH-105-008, DES-105-008, ISSUE-105

---

## Architecture Summary

| Architecture ID | Description | Traces to |
|-----------------|-------------|-----------|
| ARCH-105-001 | System Layering (UI, Engine, State) | REQ-105-002, REQ-105-003 |
| ARCH-105-002 | Stateless Orchestrator Pattern | REQ-105-003, DES-105-005, DES-105-006 |
| ARCH-105-003 | State Machine Contract | REQ-105-003, DES-105-006, DES-105-009, DES-105-010 |
| ARCH-105-004 | State Storage Schema | REQ-105-003, DES-105-006 |
| ARCH-105-005 | Phase Registry | REQ-105-002, DES-105-008 |
| ARCH-105-006 | Transition Table | REQ-105-002, REQ-105-003, DES-105-008 |
| ARCH-105-007 | Iteration Enforcement | REQ-105-003, DES-105-006, DES-105-010 |
| ARCH-105-008 | TDD Workflow State Machine | REQ-105-002, REQ-105-003, DES-105-005, DES-105-008 |
| ARCH-105-009 | Producer/QA Iteration Pattern | REQ-105-003, DES-105-006, DES-105-010 |
| ARCH-105-010 | File Organization | REQ-105-003 |
| ARCH-105-011 | Module Dependencies | REQ-105-003 |
| ARCH-105-012 | Backward Compatibility Strategy | REQ-105-004, DES-105-008 |
| ARCH-105-013 | Composite Skill Deletion Sequence | REQ-105-004, DES-105-011 |
| ARCH-105-014 | Test Strategy | REQ-105-003, DES-105-015, DES-105-016 |
| ARCH-105-015 | Error Handling Architecture | REQ-105-003, DES-105-002 |
| ARCH-105-016 | Latency Optimization | DES-105-005 |
| ARCH-105-017 | Token Cost Optimization | DES-105-005 |
| ARCH-105-018 | State Validation | REQ-105-003 |

---

## Next Steps

1. **TDD Implementation** - Write tests for state machine components per ARCH-105-014
2. **State Machine Coding** - Implement registry, transitions, next/complete commands
3. **Integration Testing** - Validate full workflow per DES-105-016
4. **Skill Deletion** - Execute deletion sequence per ARCH-105-013
5. **Documentation Update** - Update orchestrator and convention docs per DES-105-012, DES-105-013
