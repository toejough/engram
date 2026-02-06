# Project Summary: ISSUE-89 -- Deterministic Orchestration via `projctl step next`

## Executive Overview

ISSUE-89 delivered deterministic orchestration for the projctl agent system. The core architectural pivot: the LLM becomes an executor, and `projctl` becomes the planner. Instead of relying on SKILL.md prose (which the LLM may skip or misinterpret), `projctl step next` returns exactly one structured JSON action -- spawn-producer, spawn-qa, commit, or transition -- with all context needed to execute it.

**Scope:** Single task workflow, single session (2026-02-05).

**Deliverables:**
- `internal/step/` package: phase registry (22 phases, 4 workflows), `Next()` and `Complete()` functions
- `cmd/projctl/step.go`: CLI commands `projctl step next` and `projctl step complete`
- 49 tests (including 2 property-based via rapid)

**Outcome:** All acceptance criteria met. QA approved after 2 iterations (3 genuine findings in iteration 1, all resolved in iteration 2).

**Traces to:** REQ-001, ARCH-012, ARCH-013

---

## Key Decisions

### 1. Reuse Existing PairState for Sub-Phase Tracking (ARCH-012)

**Context:** Needed a mechanism to track where each phase stands in the producer -> QA -> commit -> transition sub-phase cycle.

**Options considered:**
1. New sub-phase tracking abstraction (`subphase.go`)
2. Reuse existing `PairState` from `internal/state/`

**Choice:** Reuse `PairState`. The existing struct already had `ProducerComplete`, `QAVerdict`, `ImprovementRequest`, and `Iteration` fields -- exactly what `Next()` needs to determine the next action.

**Outcome:** Zero modifications to existing packages. The `subphase.go` file was initially created as an abstraction but identified as dead code by QA and removed. PairState provided everything needed.

**Traces to:** REQ-001, ARCH-012

---

### 2. Static Phase Registry in Go Code (ARCH-012)

**Context:** The registry maps each phase to its producer skill, QA skill, model, artifact, and completion phase. This data could live in Go code or in a configuration file.

**Options considered:**
1. Static Go map literal (compile-time validated)
2. TOML/YAML config file (runtime parsed)

**Choice:** Static Go map literal in `registry.go`. Adding a new skill or changing a model requires recompiling, but the registry changes infrequently and compile-time safety eliminates an entire class of runtime config-parsing errors.

**Open question:** ISSUE-95 tracks whether to revisit this decision as the registry grows.

**Traces to:** REQ-001, ARCH-012

---

### 3. QA Enforcement on Commit (ARCH-013)

**Context:** A critical requirement is that QA cannot be skipped. The previous SKILL.md-based process relied on the LLM following instructions, which it sometimes didn't.

**Choice:** The `Complete()` function for a "commit" action checks `pair.QAVerdict != "approved"` and returns an error if QA hasn't approved. This is a deterministic gate -- no amount of LLM creativity can bypass it.

**Outcome:** Hard enforcement of the QA gate. The commit action is blocked until QA passes.

**Traces to:** REQ-001, ARCH-012, ARCH-013

---

### 4. Mandatory Ending Phases (ARCH-013)

**Context:** Workflows should not reach "complete" without passing through all ending phases (retro, summary, documentation).

**Choice:** The state machine's `LegalTargets()` function controls valid transitions. The phase graph requires passing through retro, summary, and documentation phases before reaching the terminal state. `Next()` cannot suggest skipping these.

**Traces to:** REQ-001, ARCH-013

---

## Outcomes and Deliverables

### Features Delivered

| Feature | Status | Evidence |
|---------|--------|----------|
| `projctl step next` command | Delivered | `cmd/projctl/step.go:15-29`, returns structured JSON |
| `projctl step complete` command | Delivered | `cmd/projctl/step.go:40-55`, records results and advances state |
| Phase registry (22 phases, 4 workflows) | Delivered | `internal/step/registry.go`, 297 lines |
| Sub-phase state machine (producer -> QA -> commit -> transition) | Delivered | `internal/step/next.go:77-138` |
| QA enforcement on commit | Delivered | `internal/step/next.go:179-182` |
| Improvement-request loop with QA feedback | Delivered | `internal/step/next.go:105-116` |

### Quality Metrics

| Metric | Value |
|--------|-------|
| Total tests | 49 (including 2 property-based) |
| QA iterations | 2 (3 findings in iteration 1, 0 in iteration 2) |
| Lines added | 1,407 |
| Files created | 6 (4 source + 2 test) |
| Files modified | 1 (`cmd/projctl/main.go` -- CLI registration) |
| Existing code modified | 0 internal packages changed |

### Workflow Coverage

| Workflow | Phases Registered |
|----------|------------------|
| new | 11 (pm, design, architect, breakdown, tdd-red/green/refactor, documentation, retro, summary, alignment) |
| adopt | 5 (adopt-infer-tests, adopt-infer-arch, adopt-infer-design, adopt-infer-reqs, adopt-documentation) |
| align | 5 (align-infer-tests, align-infer-arch, align-infer-design, align-infer-reqs, align-documentation) |
| task | 1 (task-documentation) |

### Known Limitations

1. **TDD sub-phase mismatch** -- `projctl step next` tracks individual red/green/refactor sub-phases, but the composite `tdd-producer` skill runs the full cycle internally. This creates friction requiring manual state walking. ISSUE-92 addresses this.
2. **No failure recovery paths** -- `step complete` accepts `status: failed` but doesn't encode retry/escalation behavior. ISSUE-96 tracks this decision.
3. **Static registry** -- Adding phases requires recompiling. ISSUE-95 tracks whether to make the registry dynamic.

**Traces to:** REQ-001, ARCH-012, ARCH-013

---

## Superseded Issues

| Issue | Relationship | Reason |
|-------|-------------|--------|
| ISSUE-1 | Superseded | External orchestrator loop replaced by `projctl step next` |
| ISSUE-86 | Superseded | Model from frontmatter now embedded in phase registry |
| ISSUE-84 | Partially superseded | Hooks approach replaced by step commands for process enforcement |

---

## Follow-Up Issues Created

| Issue | Title | Priority |
|-------|-------|----------|
| ISSUE-90 | Simplify orchestrator SKILL.md | High |
| ISSUE-91 | Rename task-audit to tdd-qa | Medium |
| ISSUE-92 | Per-phase QA in TDD loop | High |
| ISSUE-93 | Guard against duplicate role assignments | High |
| ISSUE-94 | Enforce teammate naming convention | Medium |
| ISSUE-95 | Decision: phase registry static vs runtime | Low |
| ISSUE-96 | Decision: step complete failure handling | Low |

**Traces to:** ISSUE-89

---

## Lessons Learned

### L1: Reuse Before Abstraction

The `PairState` reuse worked because the existing struct already captured the needed sub-phase state. The premature `subphase.go` abstraction was dead on arrival. When existing infrastructure fits, use it rather than creating new abstractions.

### L2: QA Catches Real Issues

QA iteration 1 found 3 genuine issues: dead code, a missing test, and incomplete registry entries (missing `align-documentation` and `task-documentation` phases). The missing registry entries would have caused runtime failures for those workflows. Two QA iterations was the right cost for this quality.

### L3: Team Coordination Needs Deterministic Guards

Duplicate QA teammates and inconsistent naming (C1, C2 in retro) are process failures, not implementation failures. These should be prevented by tooling (ISSUE-93, ISSUE-94), not by relying on the LLM to follow instructions -- the same principle that motivated `projctl step next` in the first place.

### L4: Static Code Registries Provide Compile-Time Safety

The phase registry as a Go map literal means typos in phase names, skill paths, or model names are caught at compile time. Runtime config would defer these to runtime, where they're harder to diagnose. The tradeoff (recompile to change) is acceptable for data that changes infrequently.

---

## Traceability

**Traces to:**
- REQ-001 (Dependable Agent Orchestrator)
- ARCH-012 (Deterministic Workflow Enforcement)
- ARCH-013 (Relentless Continuation)
- ISSUE-89 (parent issue)

**Implementation artifacts:**
- `internal/step/step.go` -- Package doc
- `internal/step/next.go` -- Next() and Complete() functions
- `internal/step/registry.go` -- Phase registry (22 phases)
- `internal/step/next_test.go` -- Next/Complete tests
- `internal/step/registry_test.go` -- Registry tests (including property-based)
- `cmd/projctl/step.go` -- CLI commands
- `cmd/projctl/main.go` -- CLI registration (modified)

**Related documents:**
- `docs/retro-issue89.md` -- Project retrospective
- Commits: 732f998, c7fa14f, 700e3b9
