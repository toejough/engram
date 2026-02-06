# Design: Model Validation for Teammate Spawning

## DES-001: Literal Task Tool Parameters in step next Output

`step next` spawn actions (`spawn-producer`, `spawn-qa`) emit a new `TaskParams` field containing the exact Task tool call parameters: `subagent_type`, `name`, `model`, and `prompt`. The orchestrator copies these values directly into the Task tool call with no interpretation.

**Current state:** `NextResult` has `Skill`, `SkillPath`, `Model`, `Artifact`, `Context` fields. The orchestrator must interpret these to construct a Task tool call. This interpretation is the root cause of wrong-model spawns.

**Change:** Add a `TaskParams` struct to `NextResult`:

```go
type TaskParams struct {
    SubagentType string `json:"subagent_type"`
    Name         string `json:"name"`
    Model        string `json:"model"`
    Prompt       string `json:"prompt"`
}
```

`NextResult` gains a `TaskParams *TaskParams` field (nil for non-spawn actions). The existing `Skill`, `SkillPath`, `Model` fields remain for backward compatibility but `TaskParams` is the authoritative source for spawn execution.

The `Prompt` field is assembled by `step.Next()` from the skill path, context (issue, QA feedback, prior artifacts), and the model handshake instruction (DES-002). This moves prompt construction from the LLM to deterministic Go code.

**Traces to:** REQ-001

---

## DES-002: Model Handshake Instruction in Generated Prompt

The `Prompt` field in `TaskParams` begins with:

```
First, respond with your model name so I can verify you're running the correct model.
```

This instruction appears before any skill-specific content. The teammate's first output line is its model name string (e.g., "Claude Sonnet 4.5"). The orchestrator reads this line and compares it against `ExpectedModel` (DES-003).

**Design choice:** The handshake is a plain-text instruction at the top of the prompt, not a separate protocol message. This works because Claude Code teammates output text before tool calls, so the first line of output is guaranteed to be the model name.

**Traces to:** REQ-002

---

## DES-003: ExpectedModel Field in step next Output

`NextResult` gains an `ExpectedModel string` field populated from the phase registry's `ProducerModel` or `QAModel` (depending on the spawn action). This is the ground-truth value the orchestrator compares against the teammate's handshake response.

The registry currently stores short names like `"sonnet"` and `"haiku"`. The `ExpectedModel` value uses these same short names. The orchestrator performs a case-insensitive substring match rather than exact equality — e.g., if `ExpectedModel` is `"sonnet"`, a teammate reporting `"Claude Sonnet 4.5"` or `"claude-sonnet-4-5-20250929"` both match.

**Design choice:** Substring match over exact match. Model name strings vary across providers and versions. The registry stores canonical short names (`sonnet`, `haiku`, `opus`); the handshake response contains the full model identifier. Substring matching is robust to version changes.

**Traces to:** REQ-003

---

## DES-004: Orchestrator Model Validation Flow

After spawning a teammate, the orchestrator:

1. Reads the teammate's first message
2. Checks if `ExpectedModel` (from `step next` output) appears as a case-insensitive substring
3. On match: `step complete --action <action> --status done`
4. On mismatch: `step complete --action <action> --status failed`

This validation logic lives in the orchestrator's SKILL.md instructions, not in Go code. The Go code only provides the `ExpectedModel` value and handles the `done`/`failed` status in `step complete`.

**Traces to:** REQ-004

---

## DES-005: SpawnAttempts Field in PairState

`PairState` gains two fields:
- `SpawnAttempts int` with TOML tag `spawn_attempts` — counts failed attempts
- `FailedModels []string` with TOML tag `failed_models` — accumulates actual model names reported by teammates during failed handshakes

Go's zero-value semantics mean existing `state.toml` files without these fields load with `SpawnAttempts = 0` and `FailedModels = nil`, satisfying backward compatibility (REQ-010) without migration.

Both fields are scoped to the current sub-phase (producer or QA) within a pair loop iteration. They reset on successful spawn (DES-008).

**Traces to:** REQ-005, REQ-010

---

## DES-006: Retry Logic in step complete and step next

**step complete with failed spawn:**
- Reads current `PairState` for the phase
- Increments `SpawnAttempts` by 1
- Does NOT advance sub-phase (producer stays not-complete, QA verdict stays empty)
- Persists updated `PairState`

**step next with SpawnAttempts > 0 and < 3:**
- Detects that the current sub-phase spawn hasn't succeeded yet (same conditions as initial spawn)
- Re-emits the same spawn action with identical `TaskParams`
- The retry is indistinguishable from the first attempt — teammates get the same prompt

This keeps the retry logic entirely in `step next`/`step complete`. No changes to producer or QA skills.

**Traces to:** REQ-006, REQ-008

---

## DES-007: Escalation After 3 Failed Spawns

When `step next` detects `SpawnAttempts >= 3` and the sub-phase spawn hasn't succeeded:

- Emits action `"escalate-user"` instead of a spawn action
- `NextResult` includes a `Details` string containing: expected model, and the actual model names reported by failed teammates across all attempts
- No further retries are attempted

**Tracking failed model names:** `PairState` gains a `FailedModels []string` field (`toml:"failed_models"`) that accumulates the actual model name each teammate reported during failed handshakes. When `step complete` receives a failed spawn action, the orchestrator passes the teammate's reported model name via a new `--reported-model` flag. `step complete` appends this value to `FailedModels` alongside incrementing `SpawnAttempts`. On successful spawn (`status: "done"`), `FailedModels` resets to nil along with `SpawnAttempts` (DES-008).

The escalation `Details` string is formatted as: `"spawn failed 3 times for <phase> <sub-phase>: expected model '<expected>', got models: ['<model1>', '<model2>', '<model3>']"`.

The orchestrator presents this escalation to the user as a blocking issue requiring manual intervention (e.g., model not available, configuration error).

**Traces to:** REQ-007

---

## DES-008: SpawnAttempts Reset on Success

When `step complete` receives `status: "done"` for a `spawn-producer` or `spawn-qa` action:
- Sets `SpawnAttempts = 0` on the `PairState`
- Sets `FailedModels = nil` on the `PairState`
- Advances sub-phase as normal (marks producer complete or records QA verdict)

This ensures each sub-phase gets a fresh retry budget and clean failure history. A producer that took 2 retries does not reduce the QA's retry budget.

**Traces to:** REQ-009

---

## DES-009: step complete Handles Failed Status for Spawn Actions

Currently, `step complete` only handles the `done` path for `spawn-producer` and `spawn-qa`. The `failed` status path needs to be added:

```
case "spawn-producer" with status "failed":
    pair.SpawnAttempts++
    pair.FailedModels = append(pair.FailedModels, reportedModel)
    // Do NOT set pair.ProducerComplete = true
    persist pair

case "spawn-qa" with status "failed":
    pair.SpawnAttempts++
    pair.FailedModels = append(pair.FailedModels, reportedModel)
    // Do NOT set pair.QAVerdict
    persist pair
```

The `reportedModel` value comes from a new `--reported-model` flag on `step complete`. The orchestrator extracts this from the teammate's handshake response (the first message line that failed validation).

The `Status` field on `CompleteResult` already exists but is currently unused by spawn action handlers. This design adds status-dependent branching to those handlers.

**Traces to:** REQ-006, REQ-008
