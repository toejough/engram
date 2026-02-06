# Architecture: Model Validation for Teammate Spawning

### ARCH-001: TaskParams Struct Added to NextResult

`NextResult` gains an optional `TaskParams *TaskParams` field. `TaskParams` is a new struct in `internal/step` containing `SubagentType`, `Name`, `Model`, and `Prompt` fields with JSON tags matching the Task tool call parameter names exactly.

When `step next` emits a spawn action (`spawn-producer`, `spawn-qa`), it populates `TaskParams` with deterministic values from the phase registry. The `Prompt` field is assembled by concatenating the handshake instruction (ARCH-002), a separator, and the skill/context content. The existing `Skill`, `SkillPath`, `Model` fields remain populated for backward compatibility but `TaskParams` is authoritative for spawn execution.

Non-spawn actions (`commit`, `transition`, `all-complete`) leave `TaskParams` as nil.

**Affected files:**
- `internal/step/next.go` -- new `TaskParams` struct, `TaskParams` field on `NextResult`, prompt assembly logic in `Next()`

**Traces to:** DES-001, REQ-001

---

### ARCH-002: Handshake Instruction Prefix

A package-level constant `HandshakeInstruction` in `internal/step` holds the literal string:

```
First, respond with your model name so I can verify you're running the correct model.
```

This constant is prepended to every generated `TaskParams.Prompt`. It is not configurable -- the exact wording is part of the protocol contract between `step next` output and the orchestrator's validation logic (which lives in SKILL.md, not in Go code).

**Affected files:**
- `internal/step/next.go` -- new constant, used in prompt assembly

**Traces to:** DES-002, REQ-002

---

### ARCH-003: ExpectedModel Field on NextResult

`NextResult` gains an `ExpectedModel string` field (`json:"expected_model,omitempty"`). For spawn actions, this is populated from `PhaseInfo.ProducerModel` or `PhaseInfo.QAModel` depending on the action. For non-spawn actions, it is empty.

The orchestrator (SKILL.md) performs case-insensitive substring matching of `ExpectedModel` against the teammate's first message. This matching logic is NOT in Go code -- it is instruction-level in the orchestrator skill. Go only provides the value.

**Affected files:**
- `internal/step/next.go` -- new field on `NextResult`, populated in `Next()`

**Traces to:** DES-003, REQ-003

---

### ARCH-004: SpawnAttempts and FailedModels on PairState

`PairState` in `internal/state/state.go` gains two fields:

- `SpawnAttempts int` with tag `toml:"spawn_attempts"`
- `FailedModels []string` with tag `toml:"failed_models,omitempty"`

Go zero-value semantics (`int` defaults to 0, `[]string` defaults to nil) provide backward compatibility with existing `state.toml` files that lack these fields. No migration is needed.

**Affected files:**
- `internal/state/state.go` -- two new fields on `PairState`

**Traces to:** DES-005, REQ-005, REQ-010

---

### ARCH-005: CompleteResult Gains ReportedModel Field

`CompleteResult` gains a `ReportedModel string` field (`json:"reported_model,omitempty"`). When the orchestrator calls `step complete` with `status: "failed"` for a spawn action, it passes the model name the teammate actually reported. This value is appended to `PairState.FailedModels`.

The CLI surfaces this as a `--reported-model` flag on the `step complete` subcommand.

**Affected files:**
- `internal/step/next.go` -- new field on `CompleteResult`
- `cmd/projctl/` (or equivalent CLI wiring) -- new `--reported-model` flag

**Traces to:** DES-007, DES-009, REQ-004

---

### ARCH-006: Failed Spawn Path in Complete()

`Complete()` currently handles `spawn-producer` and `spawn-qa` as success-only paths (sets `ProducerComplete = true` or records QA verdict). The new logic branches on `result.Status`:

- **Status "done"** (or empty for backward compat): existing behavior, plus resets `SpawnAttempts = 0` and `FailedModels = nil` (ARCH-007).
- **Status "failed"**: increments `SpawnAttempts`, appends `result.ReportedModel` to `FailedModels`, does NOT advance sub-phase (producer stays incomplete, QA verdict stays empty). Persists updated `PairState`.

This branching is added to both the `spawn-producer` and `spawn-qa` cases in the `Complete()` switch statement.

**Affected files:**
- `internal/step/next.go` -- modified `Complete()` function, both spawn-producer and spawn-qa cases

**Traces to:** DES-006, DES-009, REQ-006, REQ-008

---

### ARCH-007: SpawnAttempts Reset on Successful Spawn

When `Complete()` processes a spawn action with `status: "done"` (or empty), it sets `SpawnAttempts = 0` and `FailedModels = nil` on the `PairState` before persisting. This ensures each sub-phase gets a fresh retry budget.

This is part of the same code path modified in ARCH-006, not a separate function.

**Affected files:**
- `internal/step/next.go` -- within `Complete()` spawn-producer and spawn-qa success paths

**Traces to:** DES-008, REQ-009

---

### ARCH-008: Retry and Escalation Logic in Next()

`Next()` gains awareness of `SpawnAttempts` when deciding spawn actions. The existing spawn conditions remain, but before emitting a spawn action, `Next()` checks the `PairState`:

- If `SpawnAttempts >= 3`: emit action `"escalate-user"` with a `Details` string formatted as `"spawn failed 3 times for <phase> <sub-phase>: expected model '<expected>', got models: ['<m1>', '<m2>', '<m3>']"`. The `NextResult` uses a new `Details string` field (`json:"details,omitempty"`).
- If `SpawnAttempts > 0 && SpawnAttempts < 3`: emit the same spawn action as normal (retry is indistinguishable from first attempt).
- If `SpawnAttempts == 0`: normal spawn (first attempt).

This keeps all retry/escalation logic in `Next()`/`Complete()`. Teammates are stateless and unaware of retries.

**Affected files:**
- `internal/step/next.go` -- modified `Next()` function, new `Details` field on `NextResult`

**Traces to:** DES-006, DES-007, REQ-006, REQ-007, REQ-008

---

### ARCH-009: Prompt Assembly in Next()

The `TaskParams.Prompt` is assembled deterministically in `Next()` by concatenating:

1. `HandshakeInstruction` (ARCH-002)
2. A blank line separator
3. Skill invocation instruction: `"Then invoke /<skill-name>."`
4. Context block: issue reference, QA feedback (if any), prior artifacts
5. Task-specific instructions from the `NextResult` fields (artifact name, phase)

This moves prompt construction from the orchestrator LLM to deterministic Go code. The orchestrator copies `TaskParams` fields directly into the Task tool call.

**Affected files:**
- `internal/step/next.go` -- new `buildPrompt()` helper function called from `Next()`

**Traces to:** DES-001, DES-002, REQ-001, REQ-002

---

### ARCH-010: Orchestrator Model Validation Flow

The orchestrator (SKILL.md) performs model validation after each teammate spawn. The flow is:

1. Orchestrator sends the teammate a prompt prefixed with the handshake instruction (ARCH-002).
2. Teammate's first message contains its model name (e.g., "I'm running claude-opus-4-6").
3. Orchestrator performs **case-insensitive substring matching** of `ExpectedModel` (ARCH-003) against the teammate's first response. For example, if `ExpectedModel` is `"opus"`, then `"I'm running claude-opus-4-6"` matches.
4. **Match succeeds**: Orchestrator calls `step complete` with `status: "done"` after the teammate finishes its work. Normal flow continues.
5. **Match fails**: Orchestrator calls `step complete` with `status: "failed"` and `--reported-model` set to the model string the teammate actually reported. This triggers the retry/escalation path (ARCH-006, ARCH-008).

This validation logic lives entirely in SKILL.md instructions, not in Go code (see ADR-1). Go provides the data (`ExpectedModel`, `HandshakeInstruction`); the orchestrator executes the comparison and branches on the result.

**Affected files:**
- SKILL.md (orchestrator skill) -- validation instructions, done/failed branching

**Traces to:** DES-004, REQ-004

---

## Key Architectural Decisions

### ADR-1: Validation in SKILL.md, not Go code

The case-insensitive substring matching of the teammate's model handshake is orchestrator-level logic specified in SKILL.md instructions, not in Go code. Go provides `ExpectedModel` as data; the orchestrator executes the comparison. This keeps the Go code focused on state management and avoids coupling it to the LLM interaction protocol.

### ADR-2: Backward compatibility via zero values

No migration is needed for existing `state.toml` files. Go's zero-value semantics (`int` = 0, `[]string` = nil, `string` = "") mean new fields are silently absent in old files and correctly default to "no attempts, no failures."

### ADR-3: Status branching, not new actions

Failed spawns reuse the existing `spawn-producer`/`spawn-qa` action names with a `status: "failed"` distinction, rather than introducing new action types like `spawn-producer-failed`. This minimizes changes to the action vocabulary and CLI surface.
