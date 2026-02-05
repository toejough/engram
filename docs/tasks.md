# ISSUE-053: Universal QA Skill Tasks

Implementation tasks for replacing 13 phase-specific QA skills with one universal QA skill.

---

## Dependency Graph

```
TASK-1 (CONTRACT.md)
    |
    +---> TASK-2 (qa/SKILL.md)
    |         |
    |         +---> TASK-5 (orchestrator dispatch)
    |
    +---> TASK-3 (gap analysis)
              |
              +---> TASK-4 (producer contracts)
                        |
                        +---> TASK-6 (delete old QA skills)
```

---

### TASK-1: Create contract standard documentation

**Description:** Create `skills/shared/CONTRACT.md` defining the YAML format that producers use in their Contract sections. This is the foundation that both producers (who write contracts) and QA (who validates against contracts) depend on.

**Status:** Ready

**Acceptance Criteria:**
- [ ] File exists at `skills/shared/CONTRACT.md`
- [ ] Documents the contract YAML schema per DES-001
- [ ] Includes `outputs` section specification (path, id_format)
- [ ] Includes `traces_to` section specification
- [ ] Includes `checks` section specification (id, description, severity)
- [ ] Documents severity levels: `error` (fails QA) vs `warning` (passes with note)
- [ ] Includes complete examples for each producer type
- [ ] Documents version field and evolution policy
- [ ] References DES-002 contract section placement rules

**Files:** `skills/shared/CONTRACT.md`

**Dependencies:** None

**Traces to:** ARCH-020, REQ-006, DES-001, DES-002

---

### TASK-2: Create universal QA skill

**Description:** Create `skills/qa/SKILL.md` implementing the universal QA skill that validates any producer against its SKILL.md contract. This skill replaces all 13 phase-specific QA skills.

**Status:** Ready

**Acceptance Criteria:**
- [ ] File exists at `skills/qa/SKILL.md`
- [ ] Frontmatter has `name: qa`, `model: haiku`, `role: qa`
- [ ] Implements LOAD phase: read context, extract contract from producer SKILL.md, read yield, read artifacts
- [ ] Implements VALIDATE phase: execute checks against artifacts, record pass/fail
- [ ] Implements RETURN phase: yield `approved`, `improvement-request`, `escalate-phase`, `escalate-user`, or `error`
- [ ] Contract extraction uses markdown parsing per ARCH-021 algorithm
- [ ] Falls back to prose extraction when no contract section found per ARCH-024
- [ ] Tracks iteration count per ARCH-028 (max 3 iterations)
- [ ] Yields `escalate-user` when max iterations reached
- [ ] Error handling: yields `improvement-request` for malformed yield per DES-006
- [ ] Error handling: yields `improvement-request` for missing artifacts per DES-007
- [ ] Error handling: yields `error` for unreadable producer SKILL.md per DES-009
- [ ] Output format shows full checklist per DES-003

**Files:** `skills/qa/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-019, ARCH-021, ARCH-023, ARCH-024, ARCH-027, ARCH-028, ARCH-029, REQ-005, DES-003, DES-005, DES-006, DES-007, DES-008, DES-009, DES-010, DES-011, DES-012

---

### TASK-3: Perform gap analysis for all QA skills

**Description:** Compare each existing QA skill's checklist against its corresponding producer's SKILL.md to identify validation logic that might be lost. This must be completed before any QA skill deletion.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Gap analysis performed for pm-qa vs pm-interview-producer and pm-infer-producer
- [ ] Gap analysis performed for design-qa vs design-interview-producer and design-infer-producer
- [ ] Gap analysis performed for arch-qa vs arch-interview-producer and arch-infer-producer
- [ ] Gap analysis performed for breakdown-qa vs breakdown-producer
- [ ] Gap analysis performed for tdd-qa vs tdd-producer
- [ ] Gap analysis performed for tdd-red-qa vs tdd-red-producer and tdd-red-infer-producer
- [ ] Gap analysis performed for tdd-green-qa vs tdd-green-producer
- [ ] Gap analysis performed for tdd-refactor-qa vs tdd-refactor-producer
- [ ] Gap analysis performed for doc-qa vs doc-producer
- [ ] Gap analysis performed for context-qa vs context-explorer
- [ ] Gap analysis performed for alignment-qa vs alignment-producer
- [ ] Gap analysis performed for retro-qa vs retro-producer
- [ ] Gap analysis performed for summary-qa vs summary-producer
- [ ] Each gap report documents: covered checks, gaps (QA checks not in producer), decision required
- [ ] All gaps have explicit decision: add to producer contract OR drop with justification
- [ ] Gap analysis results documented in `docs/gap-analysis.md`

**Files:** `docs/gap-analysis.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-025, REQ-008

---

### TASK-4: Add Contract sections to all producer skills

**Description:** Update all producer SKILL.md files to include a `## Contract` section with YAML-formatted validation criteria per CONTRACT.md standard. This enables QA to validate producers programmatically.

**Status:** Ready

**Acceptance Criteria:**
- [ ] pm-interview-producer/SKILL.md has Contract section
- [ ] pm-infer-producer/SKILL.md has Contract section
- [ ] design-interview-producer/SKILL.md has Contract section
- [ ] design-infer-producer/SKILL.md has Contract section
- [ ] arch-interview-producer/SKILL.md has Contract section
- [ ] arch-infer-producer/SKILL.md has Contract section
- [ ] breakdown-producer/SKILL.md has Contract section
- [ ] tdd-red-producer/SKILL.md has Contract section
- [ ] tdd-red-infer-producer/SKILL.md has Contract section
- [ ] tdd-green-producer/SKILL.md has Contract section
- [ ] tdd-refactor-producer/SKILL.md has Contract section
- [ ] doc-producer/SKILL.md has Contract section
- [ ] alignment-producer/SKILL.md has Contract section
- [ ] retro-producer/SKILL.md has Contract section
- [ ] summary-producer/SKILL.md has Contract section
- [ ] All contracts follow CONTRACT.md format
- [ ] All gaps from TASK-3 that were decided to keep are included in contracts
- [ ] Existing prose requirements converted to structured checks

**Files:**
- `skills/pm-interview-producer/SKILL.md`
- `skills/pm-infer-producer/SKILL.md`
- `skills/design-interview-producer/SKILL.md`
- `skills/design-infer-producer/SKILL.md`
- `skills/arch-interview-producer/SKILL.md`
- `skills/arch-infer-producer/SKILL.md`
- `skills/breakdown-producer/SKILL.md`
- `skills/tdd-red-producer/SKILL.md`
- `skills/tdd-red-infer-producer/SKILL.md`
- `skills/tdd-green-producer/SKILL.md`
- `skills/tdd-refactor-producer/SKILL.md`
- `skills/doc-producer/SKILL.md`
- `skills/alignment-producer/SKILL.md`
- `skills/retro-producer/SKILL.md`
- `skills/summary-producer/SKILL.md`

**Dependencies:** TASK-1, TASK-3

**Traces to:** ARCH-030, REQ-007, DES-002

---

### TASK-5: Update orchestrator dispatch table

**Description:** Update `skills/project/SKILL.md` and `skills/project/SKILL-full.md` to dispatch the universal `qa` skill for all phases instead of phase-specific QA skills.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Dispatch table in SKILL.md uses single `qa` for all phases
- [ ] Orchestrator writes QA context with `producer_skill_path`, `producer_yield_path`, `artifact_paths`
- [ ] Context file format matches ARCH-026 schema
- [ ] SKILL-full.md phase details updated to reference universal QA
- [ ] Resume map updated for universal QA yield handling
- [ ] No references to old phase-specific QA skills remain

**Files:**
- `skills/project/SKILL.md`
- `skills/project/SKILL-full.md`

**Dependencies:** TASK-2

**Traces to:** ARCH-022, ARCH-026, REQ-010, DES-004, DES-013

---

### TASK-6: Delete old QA skills

**Description:** Remove all 13 phase-specific QA skill directories after verifying universal QA is functional and all gap analysis decisions are implemented.

**Status:** Ready

**Acceptance Criteria:**
- [ ] pm-qa directory deleted
- [ ] design-qa directory deleted
- [ ] arch-qa directory deleted
- [ ] breakdown-qa directory deleted
- [ ] tdd-qa directory deleted
- [ ] tdd-red-qa directory deleted
- [ ] tdd-green-qa directory deleted
- [ ] tdd-refactor-qa directory deleted
- [ ] doc-qa directory deleted
- [ ] context-qa directory deleted
- [ ] alignment-qa directory deleted
- [ ] retro-qa directory deleted
- [ ] summary-qa directory deleted
- [ ] QA-TEMPLATE.md updated to reference universal QA or marked deprecated
- [ ] No broken references to deleted skills in documentation
- [ ] Universal QA skill (TASK-2) is verified functional
- [ ] All producer contracts (TASK-4) are complete
- [ ] Gap analysis (TASK-3) decisions are all implemented

**Files:**
- `skills/pm-qa/` (delete)
- `skills/design-qa/` (delete)
- `skills/arch-qa/` (delete)
- `skills/breakdown-qa/` (delete)
- `skills/tdd-qa/` (delete)
- `skills/tdd-red-qa/` (delete)
- `skills/tdd-green-qa/` (delete)
- `skills/tdd-refactor-qa/` (delete)
- `skills/doc-qa/` (delete)
- `skills/context-qa/` (delete)
- `skills/alignment-qa/` (delete)
- `skills/retro-qa/` (delete)
- `skills/summary-qa/` (delete)
- `skills/shared/QA-TEMPLATE.md` (update)

**Dependencies:** TASK-2, TASK-3, TASK-4, TASK-5

**Traces to:** ARCH-030, REQ-009

---

## Summary

| Task | Title | Dependencies | Key Traces |
|------|-------|--------------|------------|
| TASK-1 | Create contract standard documentation | None | ARCH-020, REQ-006 |
| TASK-2 | Create universal QA skill | TASK-1 | ARCH-019, REQ-005 |
| TASK-3 | Perform gap analysis for all QA skills | TASK-1 | ARCH-025, REQ-008 |
| TASK-4 | Add Contract sections to all producer skills | TASK-1, TASK-3 | ARCH-030, REQ-007 |
| TASK-5 | Update orchestrator dispatch table | TASK-2 | ARCH-022, REQ-010 |
| TASK-6 | Delete old QA skills | TASK-2, TASK-3, TASK-4, TASK-5 | ARCH-030, REQ-009 |

---

## Roadmap Tasks

These tasks track projctl architecture roadmap items that are implemented or in progress but predate formal task tracking.

### TASK-7: CLI and cost infrastructure

**Description:** CLI completeness (context, escalation, conflict, integrate commands) and token usage tracking.

**Status:** In Progress

**Dependencies:** None

**Traces to:** ARCH-004, ARCH-016

---

### TASK-8: Intelligence and learning systems

**Description:** Learning loop (correction tracking, pattern detection), cross-project memory, and background territory mapping.

**Status:** In Progress

**Dependencies:** None

**Traces to:** ARCH-005, ARCH-007, ARCH-014

---

### TASK-9: Execution infrastructure

**Description:** Parallel skill dispatch, graceful degradation, and CLAUDE.md migration for passive context.

**Status:** In Progress

**Dependencies:** None

**Traces to:** ARCH-006, ARCH-010

---

### TASK-10: Developer tooling

**Description:** LSP integration for deterministic refactoring, visual acceptance criteria with screenshot diff, and skill compression.

**Status:** Planned

**Dependencies:** None

**Traces to:** ARCH-009, ARCH-015

---

## Test Stub Tasks

These TASK IDs are referenced in trace tool test files as examples.
They exist to satisfy trace validation.

### TASK-001: Test trace example

Test file example for trace promotion testing.

**Traces to:** ARCH-001

---

### TASK-007: Test cache lookup

Test file example referenced in trace tests.

**Traces to:** ARCH-001

---

### TASK-008: Test dry-run

Test file example for trace promotion dry-run testing.

**Traces to:** ARCH-001

---

### TASK-048: Test trace scanning

Test file example for trace scanning tests.

**Traces to:** ARCH-001

---

### TASK-049: Test trace validation

Test file example for trace validation tests.

**Traces to:** ARCH-001

---

### TASK-050: Test orphan detection

Test file example for orphan ID detection tests.

**Traces to:** ARCH-001

---

### TASK-051: Test unlinked detection

Test file example for unlinked ID detection tests.

**Traces to:** ARCH-001

---

### TASK-052: Test phase-aware validation

Test file example for phase-aware validation tests.

**Traces to:** ARCH-001
