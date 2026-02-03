# Layer -1: Skill Unification Tasks

**Project:** Layer -1 Skill Unification
**Issue:** ISSUE-008
**Created:** 2026-02-02
**Status:** Draft

**Traces to:** ARCH-1, ARCH-2, ARCH-3, ARCH-4, ARCH-5, ARCH-6, ARCH-7

---

## Task Dependency Graph

```
TASK-1 (shared/YIELD.md)
    ↓
TASK-2 (shared/CONTEXT.md updates)
    ↓
TASK-3 (PRODUCER-TEMPLATE.md)
TASK-4 (QA-TEMPLATE.md)
    ↓
TASK-5 through TASK-12 (Producer skills) ──────┐
TASK-13 through TASK-24 (QA skills)            │
    ↓                                          │
TASK-25a (parallel-looper) ────────────────────┤
TASK-25b (consistency-checker) ────────────────┤
TASK-25c (context-qa) ─────────────────────────┤
TASK-25 (context-explorer)                     │
TASK-26 (intake-evaluator)                     │
TASK-27 (next-steps)                           │
    ↓                                          │
TASK-28 (commit skill updates)                 │
    ↓                                          ↓
TASK-29 (/project skill updates) ◄─────────────┘
    ↓
TASK-30 (install-skills.sh script)
    ↓
TASK-31 (validation & cleanup)
```

---

## Tasks

### TASK-1: Create shared/YIELD.md ✓

**Description:** Create the yield protocol documentation replacing RESULT.md.

**Status:** Complete

**Acceptance Criteria:**
- [x] Document all yield types (complete, need-user-input, need-context, blocked, approved, improvement-request, escalate-phase, error)
- [x] Include TOML format examples for each type
- [x] Document payload fields per type
- [x] Document context serialization for resumption
- [x] Reference orchestration-system.md Section 3

**Files:** `skills/shared/YIELD.md`

**Dependencies:** None

**Traces to:** ARCH-2

---

### TASK-2: Update shared/CONTEXT.md ✓

**Description:** Add yield_path and query results fields to context format.

**Status:** Complete

**Acceptance Criteria:**
- [x] Add `[output]` section with `yield_path` field
- [x] Document query result injection for need-context resumption
- [x] Maintain backward compatibility notes

**Files:** `skills/shared/CONTEXT.md`, `skills/shared/CONTEXT_test.sh`

**Dependencies:** TASK-1

**Traces to:** ARCH-2, DES-2

---

### TASK-3: Create PRODUCER-TEMPLATE.md ✓

**Description:** Create template for producer skills.

**Status:** Complete

**Acceptance Criteria:**
- [x] Include frontmatter with role, phase, variant fields
- [x] Document GATHER → SYNTHESIZE → PRODUCE pattern
- [x] Include yield format section
- [x] Reference YIELD.md for protocol details

**Files:** `skills/shared/PRODUCER-TEMPLATE.md`, `skills/shared/PRODUCER-TEMPLATE_test.sh`

**Dependencies:** TASK-1

**Traces to:** ARCH-2, DES-3, DES-4

---

### TASK-4: Create QA-TEMPLATE.md ✓

**Description:** Create template for QA skills.

**Status:** Complete

**Acceptance Criteria:**
- [x] Include frontmatter with role field
- [x] Document REVIEW → RETURN pattern
- [x] Include escalation responsibilities (error/gap/conflict)
- [x] Document proposed_changes format for escalate-phase
- [x] Reference YIELD.md for protocol details

**Files:** `skills/shared/QA-TEMPLATE.md`, `skills/shared/QA-TEMPLATE_test.sh`

**Dependencies:** TASK-1

**Traces to:** ARCH-2, DES-3, DES-4

---

### TASK-5: Create pm-interview-producer ✓

**Description:** Create PM interview producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Implements GATHER → SYNTHESIZE → PRODUCE for requirements
- [x] Outputs yield protocol TOML
- [x] Can yield need-user-input for interview questions
- [x] Can yield need-context for existing docs
- [x] Produces requirements.md with REQ-N IDs

**Files:** `skills/pm-interview-producer/SKILL.md`, `skills/pm-interview-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-6: Create pm-infer-producer ✓

**Description:** Create PM infer producer skill for code analysis.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Analyzes existing code to infer requirements
- [x] Outputs yield protocol TOML
- [x] Can yield need-context for code exploration
- [x] Produces requirements.md with REQ-N IDs

**Files:** `skills/pm-infer-producer/SKILL.md`, `skills/pm-infer-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-7: Create design-interview-producer ✓

**Description:** Create Design interview producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Implements UX design interview flow
- [x] Outputs yield protocol TOML
- [x] Produces design.md with DES-N IDs

**Files:** `skills/design-interview-producer/SKILL.md`, `skills/design-interview-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-8: Create design-infer-producer ✓

**Description:** Create Design infer producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Analyzes existing UI/UX to infer design decisions
- [x] Outputs yield protocol TOML
- [x] Produces design.md with DES-N IDs

**Files:** `skills/design-infer-producer/SKILL.md`, `skills/design-infer-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-9: Create arch-interview-producer ✓

**Description:** Create Architecture interview producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Implements architecture decision interview flow
- [x] Outputs yield protocol TOML
- [x] Produces architecture.md with ARCH-N IDs

**Files:** `skills/arch-interview-producer/SKILL.md`, `skills/arch-interview-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-10: Create arch-infer-producer ✓

**Description:** Create Architecture infer producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Analyzes existing code to infer architecture decisions
- [x] Outputs yield protocol TOML
- [x] Produces architecture.md with ARCH-N IDs

**Files:** `skills/arch-infer-producer/SKILL.md`, `skills/arch-infer-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-11: Create breakdown-producer ✓

**Description:** Create Task Breakdown producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Decomposes architecture into tasks
- [x] Outputs yield protocol TOML
- [x] Produces tasks.md with TASK-N IDs
- [x] Includes dependency graph

**Files:** `skills/breakdown-producer/SKILL.md`, `skills/breakdown-producer/SKILL_test.sh`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-12: Create doc-producer ✓

**Description:** Create Documentation producer skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows PRODUCER-TEMPLATE structure
- [x] Produces/updates README, API docs, user guides
- [x] Outputs yield protocol TOML
- [x] Traces to REQ-N, DES-N, ARCH-N

**Files:** `skills/doc-producer/SKILL.md`, `skills/doc-producer/SKILL-full.md`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-2

---

### TASK-13: Create pm-qa ✓

**Description:** Create PM QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows QA-TEMPLATE structure
- [x] Implements REVIEW → RETURN for requirements
- [x] Can yield approved, improvement-request, escalate-phase
- [x] Includes escalation with reason and proposed_changes

**Files:** `skills/pm-qa/SKILL.md`, `skills/pm-qa/SKILL_test.sh`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-2

---

### TASK-14: Create design-qa ✓

**Description:** Create Design QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows QA-TEMPLATE structure
- [x] Implements REVIEW → RETURN for design
- [x] Can yield approved, improvement-request, escalate-phase

**Files:** `skills/design-qa/SKILL.md`, `skills/design-qa/SKILL_test.sh`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-2

---

### TASK-15: Create arch-qa ✓

**Description:** Create Architecture QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows QA-TEMPLATE structure
- [x] Implements REVIEW → RETURN for architecture
- [x] Can yield approved, improvement-request, escalate-phase

**Files:** `skills/arch-qa/SKILL.md`, `skills/arch-qa/SKILL_test.sh`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-2

---

### TASK-16: Create breakdown-qa ✓

**Description:** Create Breakdown QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows QA-TEMPLATE structure
- [x] Validates task decomposition completeness
- [x] Checks dependency graph for cycles
- [x] Can yield approved, improvement-request, escalate-phase

**Files:** `skills/breakdown-qa/SKILL.md`, `skills/breakdown-qa/SKILL_test.sh`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-2

---

### TASK-17: Create doc-qa ✓

**Description:** Create Documentation QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [x] Follows QA-TEMPLATE structure
- [x] Validates documentation completeness and accuracy
- [x] Can yield approved, improvement-request, escalate-phase

**Files:** `skills/doc-qa/SKILL.md`, `skills/doc-qa/SKILL_test.sh`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-2

---

### TASK-18: Create tdd-red-producer

**Description:** Create TDD Red producer skill.

**Acceptance Criteria:**
- [ ] Follows PRODUCER-TEMPLATE structure
- [ ] Writes failing tests for task
- [ ] Outputs yield protocol TOML
- [ ] Tests must fail (verifies correct red state)

**Files:** `skills/tdd-red-producer/SKILL.md`, `skills/tdd-red-producer/SKILL-full.md`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-3

---

### TASK-19: Create tdd-red-infer-producer

**Description:** Create TDD Red infer producer for existing code.

**Acceptance Criteria:**
- [ ] Follows PRODUCER-TEMPLATE structure
- [ ] Analyzes existing implementation to infer needed tests
- [ ] Outputs yield protocol TOML
- [ ] Can yield need-context for code exploration

**Files:** `skills/tdd-red-infer-producer/SKILL.md`, `skills/tdd-red-infer-producer/SKILL-full.md`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-3

---

### TASK-20: Create tdd-green-producer

**Description:** Create TDD Green producer skill.

**Acceptance Criteria:**
- [ ] Follows PRODUCER-TEMPLATE structure
- [ ] Writes minimal implementation to pass tests
- [ ] Outputs yield protocol TOML
- [ ] All targeted tests must pass

**Files:** `skills/tdd-green-producer/SKILL.md`, `skills/tdd-green-producer/SKILL-full.md`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-3

---

### TASK-21: Create tdd-refactor-producer

**Description:** Create TDD Refactor producer skill.

**Acceptance Criteria:**
- [ ] Follows PRODUCER-TEMPLATE structure
- [ ] Improves code quality while keeping tests green
- [ ] Outputs yield protocol TOML
- [ ] All tests must still pass after refactor

**Files:** `skills/tdd-refactor-producer/SKILL.md`, `skills/tdd-refactor-producer/SKILL-full.md`

**Dependencies:** TASK-3

**Traces to:** ARCH-1, REQ-3

---

### TASK-22: Create tdd-red-qa, tdd-green-qa, tdd-refactor-qa

**Description:** Create QA skills for each TDD phase.

**Acceptance Criteria:**
- [ ] Each follows QA-TEMPLATE structure
- [ ] tdd-red-qa: Verifies tests cover ACs, fail for right reasons
- [ ] tdd-green-qa: Verifies all tests pass, no regressions
- [ ] tdd-refactor-qa: Verifies tests still pass, code improved
- [ ] All can yield approved, improvement-request, escalate-phase

**Files:** `skills/tdd-red-qa/`, `skills/tdd-green-qa/`, `skills/tdd-refactor-qa/`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-3

---

### TASK-22b: Create tdd-producer

**Description:** Create composite TDD producer that runs nested RED/GREEN/REFACTOR pair loops.

**Acceptance Criteria:**
- [ ] Follows PRODUCER-TEMPLATE structure (composite variant)
- [ ] Runs RED PAIR LOOP (red-producer + red-qa) internally
- [ ] Runs GREEN PAIR LOOP (green-producer + green-qa) internally
- [ ] Runs REFACTOR PAIR LOOP (refactor-producer + refactor-qa) internally
- [ ] Handles iteration/improvement within each nested pair loop
- [ ] Outputs yield protocol TOML after all nested loops complete

**Files:** `skills/tdd-producer/SKILL.md`, `skills/tdd-producer/SKILL-full.md`

**Dependencies:** TASK-18, TASK-19, TASK-20, TASK-21, TASK-22

**Traces to:** ARCH-1, REQ-3

---

### TASK-23: Create tdd-qa

**Description:** Create overall TDD QA skill.

**Acceptance Criteria:**
- [ ] Follows QA-TEMPLATE structure
- [ ] Validates overall AC compliance after RED/GREEN/REFACTOR
- [ ] Checks TDD discipline was followed
- [ ] Can yield approved, improvement-request, escalate-phase

**Files:** `skills/tdd-qa/SKILL.md`, `skills/tdd-qa/SKILL-full.md`

**Dependencies:** TASK-4

**Traces to:** ARCH-1, REQ-3

---

### TASK-24: Create support producer/QA pairs

**Description:** Create alignment, retro, summary producer and QA skills.

**Acceptance Criteria:**
- [ ] alignment-producer: Validates traceability chain
- [ ] alignment-qa: Reviews traceability validation
- [ ] retro-producer: Project retrospective + process improvement
- [ ] retro-qa: Reviews retro completeness
- [ ] summary-producer: Project summary
- [ ] summary-qa: Reviews summary accuracy
- [ ] All follow appropriate templates
- [ ] All output yield protocol TOML

**Files:** `skills/alignment-producer/`, `skills/alignment-qa/`, `skills/retro-producer/`, `skills/retro-qa/`, `skills/summary-producer/`, `skills/summary-qa/`

**Dependencies:** TASK-3, TASK-4

**Traces to:** ARCH-1, REQ-4

---

### TASK-25a: Create parallel-looper

**Description:** Create parallel looper skill for running N PAIR LOOPs in parallel.

**Acceptance Criteria:**
- [ ] Receives list of independent items from LOOPER
- [ ] Spawns PAIR LOOP for each item via Task tool (in parallel)
- [ ] Aggregates results from all parallel PAIR LOOPs
- [ ] Dispatches to consistency-checker for batch QA
- [ ] Handles partial failures (some items fail, others succeed)
- [ ] Outputs yield protocol TOML

**Files:** `skills/parallel-looper/SKILL.md`, `skills/parallel-looper/SKILL-full.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-1

---

### TASK-25b: Create consistency-checker

**Description:** Create consistency checker skill for validating parallel results.

**Acceptance Criteria:**
- [ ] Reviews outputs across all parallel results
- [ ] Applies domain-specific consistency rules (passed as input)
- [ ] Yields approved if consistent
- [ ] Yields improvement-request (batch) if inconsistent
- [ ] Documents specific inconsistencies and resolutions
- [ ] Outputs yield protocol TOML

**Files:** `skills/consistency-checker/SKILL.md`, `skills/consistency-checker/SKILL-full.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-1

---

### TASK-25c: Create context-qa

**Description:** Create context QA skill to validate gathered context.

**Acceptance Criteria:**
- [ ] Follows QA-TEMPLATE structure
- [ ] Validates all queries were answered
- [ ] Checks results are relevant to the request
- [ ] Flags contradictions between sources
- [ ] Identifies stale or outdated information
- [ ] Can yield approved, improvement-request
- [ ] Outputs yield protocol TOML

**Files:** `skills/context-qa/SKILL.md`, `skills/context-qa/SKILL-full.md`

**Dependencies:** TASK-4

**Traces to:** ARCH-1

---

### TASK-25: Create context-explorer

**Description:** Create context explorer skill for need-context queries.

**Acceptance Criteria:**
- [ ] Handles all query types: file, memory, territory, web, semantic
- [ ] Can parallelize queries internally (via Task tool)
- [ ] Returns aggregated context
- [ ] Outputs yield protocol TOML (complete with results)

**Files:** `skills/context-explorer/SKILL.md`, `skills/context-explorer/SKILL-full.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-7, REQ-10

---

### TASK-26: Create intake-evaluator

**Description:** Create intake evaluator standalone skill.

**Acceptance Criteria:**
- [ ] Classifies request type (new, adopt, align, single-task)
- [ ] Outputs yield protocol TOML with classification
- [ ] Can escalate to user if classification uncertain

**Files:** `skills/intake-evaluator/SKILL.md`, `skills/intake-evaluator/SKILL-full.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-1, REQ-4

---

### TASK-27: Create next-steps

**Description:** Create next-steps standalone skill.

**Acceptance Criteria:**
- [ ] Suggests follow-up work based on completed project
- [ ] References open issues
- [ ] Outputs yield protocol TOML

**Files:** `skills/next-steps/SKILL.md`, `skills/next-steps/SKILL-full.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-1, REQ-4

---

### TASK-28: Update commit skill

**Description:** Update commit skill for yield protocol compatibility.

**Acceptance Criteria:**
- [ ] Outputs yield protocol TOML (complete or error)
- [ ] Works when spawned by orchestrator
- [ ] Maintains current functionality

**Files:** `skills/commit/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-1

---

### TASK-29: Update /project skill

**Description:** Update project orchestrator for new skill dispatch.

**Acceptance Criteria:**
- [ ] Dispatches to new skill names (pm-interview-producer, pm-qa, etc.)
- [ ] Parses yield protocol TOML from skills
- [ ] Implements pair loop logic (producer → QA → iterate/advance)
- [ ] Handles need-context yields (dispatch to context-explorer)
- [ ] Provides unique yield_path in context
- [ ] Handles all yield types appropriately

**Files:** `skills/project/SKILL.md`, `skills/project/SKILL-full.md`

**Dependencies:** TASK-5 through TASK-28

**Traces to:** ARCH-6, REQ-8

---

### TASK-30: Create install-skills.sh

**Description:** Create installation script for skill symlinks.

**Acceptance Criteria:**
- [ ] Removes old skill symlinks
- [ ] Creates new skill symlinks
- [ ] Handles shared directory
- [ ] Atomic switchover (all or nothing)
- [ ] Includes rollback instructions

**Files:** `scripts/install-skills.sh`

**Dependencies:** TASK-29

**Traces to:** ARCH-5

---

### TASK-31: Validation and cleanup

**Description:** Validate all skills work and clean up old skills.

**Acceptance Criteria:**
- [ ] All new skills can be invoked
- [ ] Yield protocol output validates
- [ ] /project can orchestrate full workflow
- [ ] Delete old skill directories
- [ ] Delete shared/RESULT.md
- [ ] Update any remaining references

**Files:** Multiple deletions per ARCH-1

**Dependencies:** TASK-30

**Traces to:** REQ-7, REQ-9

---

## Execution Order

Based on dependencies and structural impact:

1. **Foundation (TASK-1 through TASK-4)** - Shared content first
2. **Producers (TASK-5 through TASK-12, TASK-18 through TASK-21)** - Can parallelize
3. **QA skills (TASK-13 through TASK-17, TASK-22 through TASK-24)** - Can parallelize
4. **Support skills (TASK-25 through TASK-27)** - Can parallelize
5. **Updates (TASK-28, TASK-29)** - Sequential, depends on all skills
6. **Installation (TASK-30, TASK-31)** - Final validation
