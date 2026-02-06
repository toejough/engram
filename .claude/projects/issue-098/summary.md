# Project Summary: ISSUE-98 -- Model Validation for Teammate Spawning

## Executive Overview

ISSUE-98 added model validation to the teammate spawning pipeline in projctl. The problem: orchestrator LLMs sometimes spawned teammates on the wrong model (e.g., opus instead of haiku), wasting compute and producing artifacts with inappropriate quality/cost tradeoffs. This was directly observed during ISSUE-89 development.

**Scope:** Full lifecycle (PM, Design, Architecture, Breakdown, Implementation, Documentation, Alignment, Retro) over ~4.5 hours on 2026-02-06.

**Deliverables:**
- `TaskParams` struct with literal Task tool call parameters in `step next` output
- Model handshake protocol (teammate reports model name as first message)
- Retry/escalation logic (3 attempts, then escalate to user)
- Orchestrator SKILL.md validation instructions
- CLI `--reported-model` flag on `step complete`

**Outcome:** All 6 tasks completed. 10 requirements satisfied. QA caught 4+ real issues across multiple iterations. 12 commits.

**Traces to:** REQ-001 through REQ-010, ISSUE-98

---

## Key Decisions

### 1. Literal Task Tool Parameters Instead of LLM Interpretation (ARCH-001, ARCH-009)

**Context:** The orchestrator previously interpreted fields like `Skill`, `SkillPath`, `Model` to construct Task tool calls. This interpretation was the root cause of wrong-model spawns.

**Choice:** `step next` now emits a `TaskParams` struct with exact `subagent_type`, `name`, `model`, and `prompt` fields. The orchestrator copies these directly -- no interpretation.

**Outcome:** Eliminates the "LLM interprets pseudocode" failure mode entirely. Prompt construction moved from LLM to deterministic Go code.

**Traces to:** REQ-001, DES-001, ARCH-001, ARCH-009

---

### 2. Validation Logic in SKILL.md, Not Go Code (ADR-1)

**Context:** Model validation requires comparing the teammate's reported model against the expected model. This could live in Go code or in orchestrator instructions.

**Options considered:**
1. Go code performs validation and returns pass/fail
2. SKILL.md instructions tell the orchestrator to compare strings

**Choice:** SKILL.md instructions. Go provides the `ExpectedModel` value; the orchestrator executes the case-insensitive substring match. This keeps Go focused on state management and avoids coupling it to the LLM interaction protocol.

**Traces to:** REQ-004, DES-004, ARCH-010

---

### 3. Backward Compatibility via Go Zero Values (ADR-2)

**Context:** New `PairState` fields (`SpawnAttempts`, `FailedModels`) could break existing `state.toml` files.

**Choice:** Go's zero-value semantics (`int` = 0, `[]string` = nil) mean existing files without these fields load correctly with no migration. This was a zero-cost compatibility strategy.

**Traces to:** REQ-010, DES-005, ARCH-004

---

### 4. Status Branching on Existing Actions (ADR-3)

**Context:** Failed spawns needed a signaling mechanism. Options: new action types (`spawn-producer-failed`) or status field on existing actions.

**Choice:** Reuse `spawn-producer`/`spawn-qa` action names with `status: "failed"` distinction. Minimizes CLI surface changes and action vocabulary growth.

**Traces to:** REQ-006, DES-009, ARCH-006

---

### 5. Case-Insensitive Substring Matching for Model Validation

**Context:** Model name strings vary across providers and versions (e.g., "sonnet" vs "Claude Sonnet 4.5" vs "claude-sonnet-4-5-20250929"). The registry stores short names.

**Choice:** Substring match rather than exact match. If `ExpectedModel` is "sonnet", any response containing "sonnet" (case-insensitive) passes. Robust to version changes without maintaining a mapping table.

**Traces to:** REQ-004, DES-003, ARCH-003

---

## Outcomes and Deliverables

### Features Delivered

| Feature | Traces to |
|---------|-----------|
| `TaskParams` struct on `NextResult` with literal spawn parameters | REQ-001, TASK-1 |
| `ExpectedModel` field sourced from phase registry | REQ-003, TASK-1 |
| `HandshakeInstruction` constant prepended to all spawn prompts | REQ-002, TASK-3 |
| `buildPrompt()` deterministic prompt assembly | REQ-001, REQ-002, TASK-3 |
| `SpawnAttempts` and `FailedModels` on `PairState` | REQ-005, TASK-2 |
| Failed spawn path in `Complete()` (increment attempts, no sub-phase advance) | REQ-006, TASK-4 |
| Spawn reset on success (attempts = 0, failed models cleared) | REQ-009, TASK-4 |
| Escalation after 3 failures (`escalate-user` action with details) | REQ-007, TASK-4 |
| `--reported-model` CLI flag on `step complete` | REQ-004, TASK-6 |
| Orchestrator SKILL.md model validation instructions | REQ-004, TASK-5 |

### Quality Metrics

| Metric | Value |
|--------|-------|
| Tasks completed | 6/6 |
| Tasks escalated | 0 |
| Requirements satisfied | 10/10 |
| QA iterations | Multiple (4+ findings caught and fixed) |
| Implementation commits | 6 (TASK-1 through TASK-6) |
| Total commits | 12 |

### Known Limitations

| Limitation | Severity | Workaround |
|-----------|----------|------------|
| Handshake adds one round-trip per spawn | Low | Latency is minimal; could be skipped for non-model-sensitive phases (see ISSUE-102) |
| Substring match could false-positive on overlapping model names | Low | Current model names (opus, sonnet, haiku) are distinct substrings |
| `FailedModels` accumulates strings from LLM output (unvalidated) | Low | Used only for escalation messaging, not logic |

---

## Lessons Learned

### L1: Deterministic Output Eliminates LLM Interpretation Failures

Moving prompt construction and parameter assembly from the orchestrator LLM to Go code (`buildPrompt()`, `TaskParams`) eliminated the class of bugs where the LLM misinterpreted field values. When the LLM's job is "copy these values", there's nothing to misinterpret.

### L2: Handshake Protocols Catch Model Mismatches Early

The model handshake (teammate reports its model before doing work) catches wrong-model spawns before any compute is wasted on artifact production. The cost is one round-trip message; the savings when a mismatch occurs is an entire skill execution.

### L3: Zero-Value Backward Compatibility Is Free in Go

Adding new fields to persisted structs in Go requires no migration when the zero value is the correct default. `SpawnAttempts = 0` (no attempts yet) and `FailedModels = nil` (no failures) are exactly right for existing projects.

### L4: QA Iteration Catches Real Integration Issues

QA caught: substring vs exact match semantics, missing ARCH entries, missing README documentation, and h3 header format mismatches for trace validation. These are integration-level issues that unit tests alone would not have surfaced.

---

## Timeline and Milestones

| Time | Phase | Key Event |
|------|-------|-----------|
| 07:18 | Init/PM | Requirements produced (10 REQ-N items) |
| 08:56 | Design | Design produced (9 DES-N items) |
| 09:45 | Architecture | Architecture produced (10 ARCH-N items, 3 ADRs) |
| 09:51 | Breakdown | 6 tasks decomposed with dependency graph |
| 09:56 | Implementation | TDD cycle: TASK-1 through TASK-6 |
| 11:12 | Documentation | README and alignment updates |
| 11:31 | Alignment/Retro | Project retrospective completed |

---

## Traceability

**Traces to:**
- REQ-001 through REQ-010 (all requirements)
- DES-001 through DES-009 (all design decisions)
- ARCH-001 through ARCH-010 (all architecture entries)
- TASK-1 through TASK-6 (all implementation tasks)
- ISSUE-98 (parent issue)
- ISSUE-89 (predecessor -- wrong-model spawns observed here)

**Implementation artifacts:**
- `internal/step/next.go` -- TaskParams, ExpectedModel, HandshakeInstruction, buildPrompt(), retry/escalation logic
- `internal/state/state.go` -- SpawnAttempts, FailedModels on PairState
- `cmd/projctl/step.go` -- `--reported-model` CLI flag
- `skills/project/SKILL.md` -- Model validation instructions

**Commits:** 2a968bf, d146001, 7a13b19, 4ec351a, 7a758a9, 7784806, 6ef28b8, 33c184c, 51b2ba8 (and 3 artifact doc commits)

**Related issues:**
- ISSUE-99 -- tdd-producer runs all TDD phases in one agent (filed during retro)
- ISSUE-100 -- Enforce commit-before-completion in worktree teammates (filed during retro)
- ISSUE-101 -- Add worktree commit verification to projctl (filed during retro)
- ISSUE-102 -- Should model validation be enforced for all spawns? (filed during retro)
