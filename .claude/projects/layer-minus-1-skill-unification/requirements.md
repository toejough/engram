# Layer -1: Skill Unification Requirements

**Project:** Layer -1 Skill Unification
**Issue:** ISSUE-8
**Created:** 2026-02-02
**Status:** PM Complete

---

## Problem Statement

The current skill system uses inconsistent patterns (interview/infer/audit) that don't align with the new orchestration system's producer/QA pair pattern. Skills output RESULT.md format instead of the yield protocol. This blocks projctl from taking over orchestration.

## Current State

- 21 skills exist with interview/infer/audit naming
- Skills use TOML context input (CONTEXT.md format) - this is good
- Skills use RESULT.md output format with `[result]` and `status` field
- Missing: alignment QA, retro, summary, intake-evaluator, next-steps skills
- `/project` skill contains orchestration logic mixed with dispatch

## Desired State

- All skills follow producer/QA pair pattern
- All skills output yield protocol TOML with `[yield]` and `type` field
- All skills accept context from orchestrator via CONTEXT.md format
- No orchestration logic in skills - they do work and yield
- `/project` skill can orchestrate unified skills before projctl migration

---

## Requirements

### REQ-1: Yield Protocol Output

As an orchestrator, I want all producer and QA skills to output yield protocol TOML, so that I can deterministically parse their responses and decide next actions.

**Acceptance Criteria:**
- [ ] All producer skills output `[yield]` section with `type` field
- [ ] Valid yield types: `complete`, `need-user-input`, `blocked`
- [ ] All QA skills output `[yield]` section with `type` field
- [ ] Valid QA yield types: `approved`, `improvement-request`, `escalate-phase`
- [ ] Yield includes `[context]` section for state serialization
- [ ] Yield includes optional `[payload]` for type-specific data
- [ ] Format follows Section 4 (Yield Protocol) of `docs/orchestration-system.md`

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-2: Producer/QA Pair Pattern

As an orchestrator, I want skills organized as producer/QA pairs, so that I can run the pair loop pattern (producer → QA → iterate or advance).

**Acceptance Criteria:**
- [ ] PM phase has `pm-interview-producer`, `pm-infer-producer`, and `pm-qa` skills
- [ ] Design phase has `design-interview-producer`, `design-infer-producer`, and `design-qa` skills
- [ ] Architecture phase has `arch-interview-producer`, `arch-infer-producer`, and `arch-qa` skills
- [ ] Breakdown phase has `breakdown-producer` and `breakdown-qa` skills
- [ ] Documentation phase has `doc-producer` and `doc-qa` skills
- [ ] Each producer skill follows GATHER → SYNTHESIZE → PRODUCE pattern
- [ ] Each QA skill follows REVIEW → RETURN pattern
- [ ] QA skills include negotiation capability for `escalate-phase` yields
- [ ] QA skills draft proposed upstream changes when escalating (not just flag issues)
- [ ] QA escalations include reason: error | gap | conflict

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-3: TDD Producer/QA Pairs

As an orchestrator, I want TDD skills organized as nested producer/QA pairs, so that I can run the nested TDD loop (RED pair → GREEN pair → REFACTOR pair → TDD QA).

**Acceptance Criteria:**
- [ ] RED phase has `tdd-red-producer` (new project) and `tdd-red-infer-producer` (adopt) and `tdd-red-qa` skills
- [ ] GREEN phase has `tdd-green-producer` and `tdd-green-qa` skills
- [ ] REFACTOR phase has `tdd-refactor-producer` and `tdd-refactor-qa` skills
- [ ] Overall TDD validation has `tdd-qa` skill
- [ ] Each TDD producer outputs yield protocol
- [ ] Each TDD QA outputs yield protocol
- [ ] `tdd-red-infer-producer` analyzes existing implementation to infer needed tests

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-4: Support Agent Skills

As an orchestrator, I want support skills for alignment, retro, summary, intake, next-steps, and context exploration, so that I can run the complete workflow.

**Acceptance Criteria:**
- [ ] `alignment-producer` and `alignment-qa` skills exist
- [ ] `retro-producer` and `retro-qa` skills exist
- [ ] `retro-producer` covers both project retrospective AND process improvement (meta-audit)
- [ ] `summary-producer` and `summary-qa` skills exist
- [ ] `intake-evaluator` skill exists (classifies request type)
- [ ] `next-steps` skill exists (suggests follow-up work)
- [ ] `context-explorer` skill exists (handles all context query types)
- [ ] All support skills output yield protocol

**Priority:** P1

**Traces to:** ISSUE-8

---

### REQ-10: Context Exploration via Yield

As a producer skill, I want to yield `need-context` with a list of queries, so that I can request parallel context gathering without implementing exploration myself.

**Acceptance Criteria:**
- [ ] Producer skills can yield type `need-context`
- [ ] Yield payload contains list of queries with type and parameters
- [ ] Supported query types: file, memory, territory, web, semantic
- [ ] `context-explorer` agent handles all query types (B1 approach)
- [ ] Orchestrator runs explorer, aggregates results, resumes producer with context
- [ ] Producer receives aggregated results in resumed context

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-5: Context Input Compatibility

As a skill, I want to receive context via the existing CONTEXT.md format, so that context preparation doesn't need to change.

**Acceptance Criteria:**
- [ ] All new skills read context via `projctl context read`
- [ ] Context format remains compatible with CONTEXT.md spec
- [ ] Skills extract mode, task, phase from context
- [ ] Skills read artifact paths from context
- [ ] Skills read `output.yield_path` from context and write yield there
- [ ] Skills do NOT hardcode yield output paths (enables parallel execution)

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-6: No Orchestration Logic in Skills

As an orchestrator, I want skills to contain no orchestration logic, so that I can control the workflow deterministically.

**Acceptance Criteria:**
- [ ] Skills do NOT call `projctl state transition`
- [ ] Skills do NOT decide which skill runs next
- [ ] Skills do NOT manage iteration counts
- [ ] Skills only: read context, do work, write yield
- [ ] Orchestration decisions made by yield type, not skill logic

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-7: Delete Obsolete Skills

As a maintainer, I want obsolete skills deleted after migration, so that there's no confusion about which skills to use.

**Acceptance Criteria:**
- [ ] `pm-audit`, `design-audit`, `architect-audit`, `task-audit` deleted
- [ ] `negotiate` deleted (merged into QA escalate-phase)
- [ ] `meta-audit` deleted (merged into retro-producer)
- [ ] `test-mapper` deleted (no TEST-NNN IDs)
- [ ] Symlinks in ~/.claude/skills removed for deleted skills

**Priority:** P1

**Traces to:** ISSUE-8

---

### REQ-8: Update /project Skill for New Dispatch

As an orchestrator, I want the `/project` skill updated to dispatch new skills and parse yield protocol, so that unified skills can be tested before projctl migration.

**Acceptance Criteria:**
- [ ] `/project` dispatches to new producer/QA skill names
- [ ] `/project` parses yield protocol TOML from skill output
- [ ] `/project` implements pair loop logic (producer → QA → iterate/advance)
- [ ] `/project` handles all yield types (complete, need-user-input, blocked, approved, improvement-request, escalate-phase)
- [ ] Existing workflows (new/adopt/align/task) work with new skills

**Priority:** P0

**Traces to:** ISSUE-8

---

### REQ-9: Skill Validation

As a developer, I want a way to validate that skills output correct yield protocol, so that I can catch format errors early.

**Acceptance Criteria:**
- [ ] Each skill can be invoked with test context
- [ ] Skill output can be validated against yield protocol schema
- [ ] Validation errors report which field is malformed
- [ ] CI can run skill validation (or manual test documented)

**Priority:** P2

**Traces to:** ISSUE-8

---

## Skill Mapping

| Old Skill | New Skill(s) | Notes |
|-----------|--------------|-------|
| pm-interview | pm-interview-producer, pm-qa | Separate interview producer |
| pm-infer | pm-infer-producer, pm-qa | Separate infer producer |
| pm-audit | (delete) | Merged into pm-qa |
| design-interview | design-interview-producer, design-qa | Separate interview producer |
| design-infer | design-infer-producer, design-qa | Separate infer producer |
| design-audit | (delete) | Merged into design-qa |
| architect-interview | arch-interview-producer, arch-qa | Separate interview producer |
| architect-infer | arch-infer-producer, arch-qa | Separate infer producer |
| architect-audit | (delete) | Merged into arch-qa |
| task-breakdown | breakdown-producer, breakdown-qa | |
| task-audit | (delete) | Merged into tdd-qa |
| tdd-red | tdd-red-producer, tdd-red-infer-producer, tdd-red-qa | Separate infer for adopt workflow |
| tdd-green | tdd-green-producer, tdd-green-qa | |
| tdd-refactor | tdd-refactor-producer, tdd-refactor-qa | |
| test-mapper | (delete) | Obsolete - no TEST-NNN IDs |
| alignment-check | alignment-producer, alignment-qa | |
| negotiate | (delete) | Merged into QA skills (escalate-phase) |
| meta-audit | (delete) | Merged into retro-producer |
| - (new) | doc-producer, doc-qa | |
| - (new) | retro-producer, retro-qa | Includes process improvement |
| - (new) | summary-producer, summary-qa | |
| - (new) | intake-evaluator | |
| - (new) | next-steps | |
| commit | commit (unchanged) | |

---

## Out of Scope

- projctl command implementation (Layer 0+)
- TUI implementation (Layer 7)
- ONNX/memory system (Layer 0)
- State machine changes (Layer 0)

---

## Resolved Questions

1. **Interview vs infer modes?** → Separate skills (pm-interview-producer, pm-infer-producer)
2. **Old skills after migration?** → Delete them
3. **negotiate skill?** → Incorporated into QA skills. When QA yields `escalate-phase`, it includes evidence and argument for what the prior phase needs to change.
4. **meta-audit skill?** → Incorporated into retro-producer. Retro covers both project retrospective AND process improvement.

---

## PM-QA Lessons (from this project)

**Consistency check with source documents:**

PM-QA must verify that requirements are consistent with source documents (issues, design docs, orchestration specs). In this project, initial requirements diverged from Layer -1 in orchestration-system.md:

- Requirements said `pm-interview-producer` / `pm-infer-producer` (separate skills)
- Layer -1 said `pm-producer` (single skill)
- Section 7.3 (Adopt Existing) referenced "infer agents" separately

This inconsistency was caught during user review, not PM-QA. PM-QA should:

1. Read the source document (issue, spec) that triggered the requirements
2. Verify each requirement traces to something in the source
3. Verify nothing in the source is missing from requirements
4. Flag any divergence for explicit resolution (update source or update requirements)
