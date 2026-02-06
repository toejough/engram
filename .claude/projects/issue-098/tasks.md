# Tasks: Model Validation for Teammate Spawning (ISSUE-98)

## Dependency Graph

```
TASK-1 (TaskParams struct + ExpectedModel field)
    |
    +---> TASK-3 (Prompt assembly in Next())
    |         |
    |         +---> TASK-5 (Orchestrator SKILL.md validation instructions)
    |
    +---> TASK-2 (SpawnAttempts + FailedModels on PairState)
              |
              +---> TASK-4 (Failed spawn path in Complete() + retry/escalation in Next())
                        |
                        +---> TASK-6 (CLI --reported-model flag)
                                  |
                                  +---> TASK-5 (Orchestrator SKILL.md validation instructions)
```

---

### TASK-1: Add TaskParams struct and ExpectedModel to NextResult

**Priority:** P0
**Estimated scope:** Small (one file, ~30 lines)

Add the `TaskParams` struct and two new fields to `NextResult`: `TaskParams *TaskParams` and `ExpectedModel string`. Populate `TaskParams` and `ExpectedModel` in the three spawn-emitting branches of `Next()` (initial producer, initial QA, improvement-request re-spawn).

`TaskParams.Prompt` is left as an empty string in this task -- prompt assembly is TASK-3.

**Affected files:**
- `internal/step/next.go`

**Tests:**
- `TaskParams` is non-nil for spawn actions and nil for non-spawn actions
- `ExpectedModel` matches `PhaseInfo.ProducerModel` for `spawn-producer` and `PhaseInfo.QAModel` for `spawn-qa`
- `TaskParams.SubagentType`, `Name`, `Model` are populated correctly
- Existing tests continue passing (backward compat of existing fields)

**Depends on:** (none)

**Traces to:** ARCH-001, ARCH-003, DES-001, DES-003, REQ-001, REQ-003

---

### TASK-2: Add SpawnAttempts and FailedModels to PairState

**Priority:** P0
**Estimated scope:** Small (one file, ~5 lines)

Add `SpawnAttempts int` (`toml:"spawn_attempts"`) and `FailedModels []string` (`toml:"failed_models,omitempty"`) to `PairState` in `internal/state/state.go`.

**Affected files:**
- `internal/state/state.go`

**Tests:**
- Round-trip: write PairState with SpawnAttempts/FailedModels, read back, values match
- Backward compat: load existing state.toml without these fields, SpawnAttempts == 0, FailedModels == nil
- TOML serialization: fields appear with correct keys

**Depends on:** (none)

**Traces to:** ARCH-004, DES-005, REQ-005, REQ-010

---

### TASK-3: Prompt assembly with handshake instruction

**Priority:** P0
**Estimated scope:** Medium (one file, ~40 lines)

Add `HandshakeInstruction` constant and `buildPrompt()` helper in `internal/step/next.go`. Wire `buildPrompt()` into the three spawn branches of `Next()` to populate `TaskParams.Prompt`. The prompt concatenates: handshake instruction, skill invocation, context (issue, QA feedback, prior artifacts), and task-specific instructions.

**Affected files:**
- `internal/step/next.go`

**Tests:**
- `TaskParams.Prompt` starts with `HandshakeInstruction`
- Prompt contains skill invocation instruction (e.g., "Then invoke /skill-name.")
- Prompt includes issue reference when present
- Prompt includes QA feedback when present (improvement-request case)
- Prompt does NOT contain handshake instruction for non-spawn actions (TaskParams is nil)

**Depends on:** TASK-1

**Traces to:** ARCH-002, ARCH-009, DES-001, DES-002, REQ-001, REQ-002

---

### TASK-4: Failed spawn path in Complete() and retry/escalation in Next()

**Priority:** P1
**Estimated scope:** Medium (one file, ~50 lines)

Modify `Complete()` to branch on `result.Status` for `spawn-producer` and `spawn-qa`:
- **"done" (or empty):** existing behavior + reset `SpawnAttempts = 0`, `FailedModels = nil`
- **"failed":** increment `SpawnAttempts`, append `result.ReportedModel` to `FailedModels`, do NOT advance sub-phase

Add `ReportedModel string` field to `CompleteResult`.

Modify `Next()` spawn logic: before emitting a spawn action, check `SpawnAttempts`:
- `>= 3`: emit `escalate-user` action with `Details` string
- `> 0 && < 3`: emit normal spawn (retry)
- `== 0`: normal spawn (first attempt)

Add `Details string` field to `NextResult`.

**Affected files:**
- `internal/step/next.go`

**Tests:**
- `Complete()` with status "failed" increments SpawnAttempts and appends FailedModels
- `Complete()` with status "failed" does NOT set ProducerComplete or QAVerdict
- `Complete()` with status "done" resets SpawnAttempts to 0 and FailedModels to nil
- `Next()` with SpawnAttempts == 0 emits normal spawn
- `Next()` with SpawnAttempts == 1 or 2 emits same spawn (retry)
- `Next()` with SpawnAttempts >= 3 emits "escalate-user" with Details containing expected model and failed models
- Backward compat: Complete() with empty status still works (done path)

**Depends on:** TASK-1, TASK-2

**Traces to:** ARCH-005, ARCH-006, ARCH-007, ARCH-008, DES-006, DES-007, DES-008, DES-009, REQ-004, REQ-006, REQ-007, REQ-008, REQ-009

---

### TASK-5: Orchestrator SKILL.md model validation instructions

**Priority:** P0
**Estimated scope:** Small (one file, ~20 lines of instruction text)

Update the orchestrator SKILL.md to include model validation flow instructions:
1. After spawning a teammate, read the first message
2. Case-insensitive substring match of `expected_model` against first message
3. On match: proceed, then `step complete --action <action> --status done`
4. On mismatch: `step complete --action <action> --status failed --reported-model "<model>"`
5. On `escalate-user` action from `step next`: present escalation to user

Also update instructions to use `TaskParams` fields directly for Task tool calls instead of interpreting `Skill`/`SkillPath`/`Model` fields.

**Affected files:**
- Orchestrator SKILL.md (path TBD based on project structure)

**Tests:**
- SKILL.md contains handshake validation instructions
- SKILL.md references `expected_model` field
- SKILL.md contains `--reported-model` flag usage
- SKILL.md contains `escalate-user` handling
- SKILL.md references `TaskParams` fields for spawn execution

**Depends on:** TASK-3, TASK-6

**Traces to:** ARCH-010, DES-004, REQ-004

---

### TASK-6: CLI --reported-model flag on step complete

**Priority:** P1
**Estimated scope:** Small (one file, ~5 lines)

Add `ReportedModel string` flag (`--reported-model`) to `stepCompleteArgs` in `cmd/projctl/step.go` and pass it through to `CompleteResult`.

**Affected files:**
- `cmd/projctl/step.go`

**Tests:**
- Flag parses correctly and populates CompleteResult.ReportedModel
- Flag is optional (empty when not provided)

**Depends on:** TASK-4

**Traces to:** ARCH-005, DES-009, REQ-004

---

## Simplicity Rationale

**Why 6 tasks, not fewer:**
- TASK-1 and TASK-2 are independent struct changes that can be developed in parallel
- TASK-3 depends on TASK-1 (needs TaskParams struct to exist)
- TASK-4 depends on both TASK-1 and TASK-2 (needs both structs)
- TASK-5 depends on TASK-3 and TASK-6 (needs all Go changes done before updating instructions)
- TASK-6 depends on TASK-4 (needs ReportedModel on CompleteResult)

**Why not fewer:**
- Combining TASK-1 + TASK-3 would mix struct definition with prompt logic -- different concerns
- Combining TASK-4 + TASK-6 would mix core logic with CLI wiring -- different layers

**Why not more:**
- TASK-4 combines retry, escalation, and reset logic because they're all branches in the same switch statement
- Splitting retry from escalation would create artificial task boundaries within `Next()` and `Complete()`

**Parallel opportunities:**
- TASK-1 and TASK-2 can run concurrently (no shared changes)
