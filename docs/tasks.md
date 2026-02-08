# ISSUE-53: Universal QA Skill Tasks

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
- [ ] Implements LOAD phase: read context, extract contract from producer SKILL.md, read producer output, read artifacts
- [ ] Implements VALIDATE phase: execute checks against artifacts, record pass/fail
- [ ] Implements RETURN phase: sends message with `approved`, `improvement-request`, `escalate-phase`, `escalate-user`, or `error`
- [ ] Contract extraction uses markdown parsing per ARCH-021 algorithm
- [ ] Falls back to prose extraction when no contract section found per ARCH-024
- [ ] Tracks iteration count per ARCH-028 (max 3 iterations)
- [ ] Sends `escalate-user` when max iterations reached
- [ ] Error handling: sends `improvement-request` for malformed output per DES-006
- [ ] Error handling: sends `improvement-request` for missing artifacts per DES-007
- [ ] Error handling: sends `error` for unreadable producer SKILL.md per DES-009
- [ ] Output format shows full checklist per DES-003

**Files:** `skills/qa/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-019, ARCH-021, ARCH-023, ARCH-024, ARCH-028, ARCH-029, REQ-005, DES-003, DES-005, DES-006, DES-007, DES-008, DES-009, DES-010, DES-011, DES-012

---

### TASK-3: Perform gap analysis for all QA skills

**Description:** Compare each existing QA skill's checklist against its corresponding producer's SKILL.md to identify validation logic that might be lost. This must be completed before any QA skill deletion.

**Status:** Complete

**Acceptance Criteria:**
- [x] Gap analysis performed for pm-qa vs pm-interview-producer and pm-infer-producer
- [x] Gap analysis performed for design-qa vs design-interview-producer and design-infer-producer
- [x] Gap analysis performed for arch-qa vs arch-interview-producer and arch-infer-producer
- [x] Gap analysis performed for breakdown-qa vs breakdown-producer
- [x] Gap analysis performed for tdd-qa vs tdd-producer
- [x] Gap analysis performed for tdd-red-qa vs tdd-red-producer and tdd-red-infer-producer
- [x] Gap analysis performed for tdd-green-qa vs tdd-green-producer
- [x] Gap analysis performed for tdd-refactor-qa vs tdd-refactor-producer
- [x] Gap analysis performed for doc-qa vs doc-producer
- [x] Gap analysis performed for context-qa vs context-explorer
- [x] Gap analysis performed for alignment-qa vs alignment-producer
- [x] Gap analysis performed for retro-qa vs retro-producer
- [x] Gap analysis performed for summary-qa vs summary-producer
- [x] Each gap report documents: covered checks, gaps (QA checks not in producer), decision required
- [x] All gaps have explicit decision: add to producer contract OR drop with justification
- [x] Gap analysis results documented in `docs/gap-analysis.md`

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
- [ ] SKILL-full.md phase details updated to reference universal QA
- [ ] Resume map updated for universal QA yield handling
- [ ] No references to old phase-specific QA skills remain

**Files:**
- `skills/project/SKILL.md`
- `skills/project/SKILL-full.md`

**Dependencies:** TASK-2

**Traces to:** ARCH-022, REQ-010, DES-004, DES-013

---

### TASK-6: Delete old QA skills

**Description:** Remove all 13 phase-specific QA skill directories after verifying universal QA is functional and all gap analysis decisions are implemented.

**Status:** Complete

**Acceptance Criteria:**
- [x] pm-qa directory deleted
- [x] design-qa directory deleted
- [x] arch-qa directory deleted
- [x] breakdown-qa directory deleted
- [x] tdd-qa directory deleted
- [x] tdd-red-qa directory deleted
- [x] tdd-green-qa directory deleted
- [x] tdd-refactor-qa directory deleted
- [x] doc-qa directory deleted
- [x] context-qa directory deleted
- [x] alignment-qa directory deleted
- [x] retro-qa directory deleted
- [x] summary-qa directory deleted
- [x] QA-TEMPLATE.md updated to reference universal QA or marked deprecated
- [x] No broken references to deleted skills in documentation
- [x] Universal QA skill (TASK-2) is verified functional
- [x] All producer contracts (TASK-4) are complete
- [x] Gap analysis (TASK-3) decisions are all implemented

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

## ISSUE-56: Inferred Specification Warning Tasks

Implementation tasks for adding inference detection and user confirmation to producer skills.

---

### Dependency Graph (ISSUE-56)

```
TASK-11 (YIELD.md extension)
    |
    +---> TASK-12 (PRODUCER-TEMPLATE.md update)
    |         |
    |         +---> TASK-14 (update 6 producer SKILL.md files)
    |
    +---> TASK-13 (orchestrator inferred handling)
```

---

### TASK-11: Extend YIELD.md with inferred specification format

**Description:** Add documentation for the `inferred` flag on `need-user-input` yields to `skills/shared/YIELD.md`. Define the `payload.inferred`, `payload.items` array, and `source` enum fields.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Producer communication documentation documents `inferred` message pattern
- [ ] Documents `items` array with `specification`, `reasoning`, `source` fields
- [ ] Documents `source` enum: `best-practice`, `edge-case`, `implicit-need`, `professional-judgment`
- [ ] Includes complete example of an inferred specification message
- [ ] Documents that `inferred` field is optional (backward compatible)
- [ ] Documents the inferred decisions response format

**Files:** `skills/shared/YIELD.md`

**Dependencies:** None

**Traces to:** ARCH-031, REQ-012, DES-014

---

### TASK-12: Add inference classification guidelines to PRODUCER-TEMPLATE.md

**Description:** Add a CLASSIFY step and inference detection guidelines to `skills/shared/PRODUCER-TEMPLATE.md`. Define how producers distinguish explicit from inferred specifications.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `skills/shared/PRODUCER-TEMPLATE.md` documents the CLASSIFY step between SYNTHESIZE and PRODUCE
- [ ] Defines "explicit": directly traceable to user input, issue description, or gathered context
- [ ] Defines "inferred": added based on best practices, edge cases, implicit needs, or professional judgment
- [ ] Provides examples of explicit vs inferred specifications
- [ ] Documents the conservative default: when in doubt, classify as inferred
- [ ] Documents the yield-before-produce pattern for inferred items

**Files:** `skills/shared/PRODUCER-TEMPLATE.md`

**Dependencies:** TASK-11

**Traces to:** ARCH-033, REQ-015, DES-016

---

### TASK-13: Update orchestrator to handle inferred specification yields

**Description:** Update `skills/project/SKILL.md` and `skills/project/SKILL-full.md` to detect `inferred = true` messages from producers and present inferred items as a numbered accept/reject list.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Orchestrator PAIR LOOP pattern handles `inferred = true` messages from producers
- [ ] Inferred items presented as numbered list with reasoning
- [ ] User can accept all, reject all, or selectively accept/reject
- [ ] Accepted/rejected decisions sent back to producer
- [ ] Producer continues with decisions after user responds
- [ ] SKILL-full.md updated for inferred message handling

**Files:**
- `skills/project/SKILL.md`
- `skills/project/SKILL-full.md`

**Dependencies:** TASK-11

**Traces to:** ARCH-032, REQ-014, DES-015

---

### TASK-14: Update 6 producer SKILL.md files to reference inference guidelines

**Description:** Update all 6 affected producer skills (pm-interview, pm-infer, design-interview, design-infer, arch-interview, arch-infer) to reference the shared inference classification guidelines and include the CLASSIFY step in their workflows.

**Status:** Ready

**Acceptance Criteria:**
- [ ] pm-interview-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] pm-infer-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] design-interview-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] design-infer-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] arch-interview-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] arch-infer-producer/SKILL.md references PRODUCER-TEMPLATE.md inference guidelines
- [ ] Each skill's workflow section includes the CLASSIFY step between SYNTHESIZE and PRODUCE
- [ ] Each skill's communication pattern includes the inferred message type

**Files:**
- `skills/pm-interview-producer/SKILL.md`
- `skills/pm-infer-producer/SKILL.md`
- `skills/design-interview-producer/SKILL.md`
- `skills/design-infer-producer/SKILL.md`
- `skills/arch-interview-producer/SKILL.md`
- `skills/arch-infer-producer/SKILL.md`

**Dependencies:** TASK-12

**Traces to:** ARCH-033, REQ-013

---

### ISSUE-56 Task Summary

| Task | Title | Dependencies | Key Traces |
|------|-------|--------------|------------|
| TASK-11 | Extend YIELD.md with inferred specification format | None | ARCH-031, REQ-012 |
| TASK-12 | Add inference classification guidelines to PRODUCER-TEMPLATE.md | TASK-11 | ARCH-033, REQ-015 |
| TASK-13 | Update orchestrator to handle inferred specification yields | TASK-11 | ARCH-032, REQ-014 |
| TASK-14 | Update 6 producer SKILL.md files to reference inference guidelines | TASK-12 | ARCH-033, REQ-013 |

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

---

## ISSUE-92: Per-Phase QA in TDD Loop Tasks

Implementation tasks for restructuring the TDD loop with per-phase QA and commit validation.

---

### Simplicity Rationale

The per-phase QA approach applies the existing PAIR LOOP pattern (producer → QA) consistently across all TDD sub-phases. While it adds more state machine phases (10 new phases total), each phase has a single, focused concern. This is simpler than deferring all validation to a complex end-of-cycle QA that must validate three phases worth of work at once.

**Alternatives considered:**
- Keep QA at end: Issues compound, late detection, one QA must validate 3 phases
- QA between phases only (no commit QA): Misses commit-specific issues like staging errors or secrets
- Pre-commit hooks only: Not enforced in orchestrator, inconsistent behavior

**Why current approach is appropriate:**
- Consistent with existing phase/QA pattern used in pm, design, arch phases
- Smaller QA scope per phase means simpler validation logic
- State machine enforcement prevents shortcuts programmatically
- Reuses existing universal QA skill (no new skill needed)
- Immediate feedback loop catches issues early when context is fresh

---

### Dependency Graph (ISSUE-92)

```
TASK-15 (foundation: state machine phases)
    |
    +---> TASK-16 (transitions.go)
    |
    +---> TASK-17 (tests for transitions)
    |
TASK-18 (step registry updates)
    |
    +---> TASK-19 (step next logic verification)
    |
TASK-20 (commit-producer skill)
    |
    +---> TASK-21 (commit-QA contract)
            |
            +---> TASK-22 (integration test)
```

---

### TASK-15: Add TDD sub-phase and commit phases to state machine

**Description:** Add 10 new phases to the state machine phase enumeration to support per-phase QA in the TDD loop.

**Status:** Complete

**Acceptance Criteria:**
- [ ] Phase constants added: `tdd-red-qa`, `tdd-green-qa`, `tdd-refactor-qa`
- [ ] Phase constants added: `commit-red`, `commit-red-qa`, `commit-green`, `commit-green-qa`, `commit-refactor`, `commit-refactor-qa`
- [ ] All phase constants follow existing naming convention
- [ ] Phase validation logic accepts new phases
- [ ] `go test ./internal/state` passes

**Files:** `internal/state/state.go`

**Dependencies:** None

**Traces to:** ARCH-034, ARCH-035, ARCH-041

---

### TASK-16: Update state transition rules

**Description:** Modify the legal transitions map to enforce QA phases between producer and commit, and between commit and next TDD phase.

**Status:** Complete

**Acceptance Criteria:**
- [ ] `tdd-red` legal targets: `["tdd-red-qa"]`
- [ ] `tdd-red-qa` legal targets: `["commit-red"]`
- [ ] `commit-red` legal targets: `["commit-red-qa"]`
- [ ] `commit-red-qa` legal targets: `["tdd-green"]`
- [ ] `tdd-green` legal targets: `["tdd-green-qa"]`
- [ ] `tdd-green-qa` legal targets: `["commit-green"]`
- [ ] `commit-green` legal targets: `["commit-green-qa"]`
- [ ] `commit-green-qa` legal targets: `["tdd-refactor"]`
- [ ] `tdd-refactor` legal targets: `["tdd-refactor-qa"]`
- [ ] `tdd-refactor-qa` legal targets: `["commit-refactor"]`
- [ ] `commit-refactor` legal targets: `["commit-refactor-qa"]`
- [ ] `commit-refactor-qa` legal targets: `["task-audit"]`
- [ ] No illegal shortcuts exist (e.g., tdd-red → commit-red blocked)

**Files:** `internal/state/transitions.go`

**Dependencies:** TASK-15

**Traces to:** ARCH-036, ARCH-041

---

### TASK-17: Add transition enforcement tests

**Description:** Expand the existing test file `internal/state/tdd_qa_phases_test.go` to validate all legal and illegal transitions for the new TDD QA phases.

**Status:** Complete

**Acceptance Criteria:**
- [ ] Tests verify all legal transitions succeed (12 new transitions)
- [ ] Tests verify illegal transitions fail with "illegal transition" error
- [ ] Test verifies full chain from tdd-red to task-audit works
- [ ] Test verifies shortcut attempts (e.g., tdd-red → commit-red) fail
- [ ] `go test ./internal/state` passes
- [ ] All tests use table-driven test pattern for clarity

**Files:** `internal/state/tdd_qa_phases_test.go`

**Dependencies:** TASK-16

**Traces to:** ARCH-036, ARCH-041

---

### TASK-18: Add TDD sub-phase QA entries to step registry

**Description:** Add registry entries for tdd-red-qa, tdd-green-qa, and tdd-refactor-qa phases in the step registry, mapping each to the universal QA skill.

**Status:** Complete

**Acceptance Criteria:**
- [ ] Registry entry for `tdd-red-qa`: QA skill is `"qa"`, QAPath is `"skills/qa/SKILL.md"`
- [ ] Registry entry for `tdd-green-qa`: QA skill is `"qa"`, QAPath is `"skills/qa/SKILL.md"`
- [ ] Registry entry for `tdd-refactor-qa`: QA skill is `"qa"`, QAPath is `"skills/qa/SKILL.md"`
- [ ] All entries use model `"haiku"` (per ARCH-029)
- [ ] Registry lookup returns correct PhaseInfo for each new phase
- [ ] `go test ./internal/step` passes

**Files:** `internal/step/registry.go`

**Dependencies:** TASK-15

**Traces to:** ARCH-034, ARCH-037

---

### TASK-19: Verify step next returns QA actions for TDD sub-phases

**Description:** Ensure `projctl step next` correctly returns QA actions for the new TDD sub-phase QA phases.

**Status:** Complete

**Acceptance Criteria:**
- [ ] When phase is `tdd-red-qa`, `step next` returns action `"spawn-qa"` with skill `"qa"`
- [ ] When phase is `tdd-green-qa`, `step next` returns action `"spawn-qa"` with skill `"qa"`
- [ ] When phase is `tdd-refactor-qa`, `step next` returns action `"spawn-qa"` with skill `"qa"`
- [ ] Response includes producer_skill_path (e.g., `"skills/tdd-red-producer/SKILL.md"`)
- [ ] `go test ./internal/step` passes
- [ ] Manual verification: `projctl state set tdd-red-qa && projctl step next` returns expected QA action

**Files:** `internal/step/next.go`, `internal/step/next_test.go`

**Dependencies:** TASK-18

**Traces to:** ARCH-037

---

### TASK-20: Create commit-producer skill specification

**Description:** Define the commit-producer skill behavior in a SKILL.md file. This skill handles staging files and creating commits for each TDD phase.

**Status:** Complete

**Acceptance Criteria:**
- [ ] SKILL.md documents phase-specific staging rules (red=tests, green=tests+impl, refactor=impl)
- [ ] SKILL.md documents secret detection patterns
- [ ] SKILL.md documents conventional commit message format
- [ ] SKILL.md includes Contract section with outputs and checks
- [ ] Skill uses model `"haiku"` (lightweight task)
- [ ] Skill workflow documented: read phase → stage files → validate no secrets → generate message → commit → report

**Files:** `skills/commit-producer/SKILL.md`

**Dependencies:** None

**Traces to:** ARCH-039

---

### TASK-21: Define commit-QA validation contract

**Description:** Document the validation contract for commit-QA phases. This defines what the universal QA skill checks when validating commits.

**Status:** Complete

**Acceptance Criteria:**
- [ ] Contract includes CHECK-COMMIT-001: Files staged match phase scope (error severity)
- [ ] Contract includes CHECK-COMMIT-002: No secrets in staged files (error severity)
- [ ] Contract includes CHECK-COMMIT-003: Commit message follows conventional format (error severity)
- [ ] Contract includes CHECK-COMMIT-004: Commit message describes change accurately (warning severity)
- [ ] Contract includes CHECK-COMMIT-005: No blanket lint suppressions added (error severity)
- [ ] Contract includes CHECK-COMMIT-006: Commit created successfully (error severity)
- [ ] Phase-specific file scope validation documented (red vs green vs refactor)
- [ ] QA response patterns documented for each failure type

**Files:** `skills/qa/SKILL.md` (update with commit validation guidance) or separate doc reference

**Dependencies:** TASK-20

**Traces to:** ARCH-038, ARCH-040

---

### TASK-22: Add integration test for full TDD cycle with per-phase QA

**Description:** Create an end-to-end integration test that validates the full TDD loop with per-phase QA executes correctly from tdd-red through task-audit.

**Status:** Pending

**Acceptance Criteria:**
- [ ] Test starts at `tdd-red` phase
- [ ] Test executes each transition in sequence: tdd-red → tdd-red-qa → commit-red → commit-red-qa → tdd-green → tdd-green-qa → commit-green → commit-green-qa → tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa → task-audit
- [ ] Test verifies state updates correctly at each transition
- [ ] Test verifies `step next` returns correct actions at each phase
- [ ] `go test ./internal/state -run Integration` passes
- [ ] Test output clearly shows the full phase progression

**Files:** `internal/state/tdd_integration_test.go`

**Dependencies:** TASK-17, TASK-19, TASK-21

**Traces to:** ARCH-041

---

### ISSUE-92 Task Summary

| Task | Title | Dependencies | Key Traces |
|------|-------|--------------|------------|
| TASK-15 | Add TDD sub-phase and commit phases to state machine | None | ARCH-034, ARCH-035, ARCH-041 |
| TASK-16 | Update state transition rules | TASK-15 | ARCH-036, ARCH-041 |
| TASK-17 | Add transition enforcement tests | TASK-16 | ARCH-036, ARCH-041 |
| TASK-18 | Add TDD sub-phase QA entries to step registry | TASK-15 | ARCH-034, ARCH-037 |
| TASK-19 | Verify step next returns QA actions for TDD sub-phases | TASK-18 | ARCH-037 |
| TASK-20 | Create commit-producer skill specification | None | ARCH-039 |
| TASK-21 | Define commit-QA validation contract | TASK-20 | ARCH-040 |
| TASK-22 | Add integration test for full TDD cycle with per-phase QA | TASK-17, TASK-19, TASK-21 | ARCH-041 |

---

### Summary Metrics (ISSUE-92)

| Metric | Count |
|--------|-------|
| Total tasks | 8 |
| New files | 2 (commit-producer/SKILL.md, tdd_integration_test.go) |
| Modified files | 6 (state.go, transitions.go, tdd_qa_phases_test.go, registry.go, next.go, next_test.go) |
| New phases | 10 |
| New transitions | 12 |
| Architecture items covered | 8 (ARCH-034 through ARCH-041) |

**Critical path:** TASK-15 → TASK-16 → TASK-17 (state machine foundation must be solid before step registry and skill work)

**Parallel opportunities:** TASK-18 and TASK-20 can proceed in parallel after TASK-15 completes

**Testing strategy:** Progressive - unit tests for transitions (TASK-17), step logic tests (TASK-19), integration test for full cycle (TASK-22)

---

## ISSUE-152: Integrate Semantic Memory into Orchestration Workflow

Implementation tasks for integrating ONNX-based semantic similarity search into the orchestration workflow and skills.

---

### Simplicity Rationale

The universal yield capture (TASK-37) is the key simplification: instead of adding `projctl memory decide` to 18 producer SKILL.md files (18 integration points), we add ONE integration point in the orchestrator that covers all producers automatically. This is N→1 reduction.

**Alternatives considered:**
- Per-producer memory writes: Would require 18×2 changes (read+write in each SKILL.md), fragile, easy to forget in new producers
- No memory integration: Learnings lost, agents repeat mistakes
- Manual capture: Users forget, incomplete

**Why current approach is appropriate:**
- Single orchestrator integration covers ALL producers universally
- New producers get memory capture for free (no SKILL.md changes needed)
- Clear separation: producers read (proactive), orchestrator writes (automatic)
- Graceful degradation: memory failures are non-blocking

---

### Dependency Graph (ISSUE-152)

```
TASK-23 (BLOCKING: fix tokenizer + e5 switch)
    |
    ├─────> TASK-24 (Session-end in orchestrator)
    ├─────> TASK-25 (PM interview memory reads)
    ├─────> TASK-26 (Design interview memory reads)
    ├─────> TASK-27 (Arch interview memory reads)
    ├─────> TASK-28 (QA memory reads + writes)
    ├─────> TASK-29 (Context explorer auto-memory)
    ├─────> TASK-30 (Retro memory reads + writes)
    ├─────> TASK-31 (Orchestrator startup reads)
    ├─────> TASK-32 (Breakdown producer memory reads)
    ├─────> TASK-33 (TDD green producer memory reads)
    ├─────> TASK-34 (TDD refactor producer memory reads)
    ├─────> TASK-35 (Alignment producer memory reads)
    ├─────> TASK-36 (TDD red producers memory reads)
    ├─────> TASK-37 (Universal yield capture - orchestrator)
    ├─────> TASK-38 (Infer producers memory reads)
    ├─────> TASK-39 (Doc + summary producers memory reads)
    ├─────> TASK-40 (Next-steps memory reads)
    ├─────> TASK-41 (Promotion pipeline - Go code)
    ├─────> TASK-42 (External knowledge capture - Go code + SKILL.md)
    └─────> TASK-43 (Memory hygiene - Go code + SKILL.md)
```

**TASK-23 is BLOCKING** - all memory queries depend on accurate embeddings.

**TASKs 24-40 are SKILL.md-only edits** - all parallel after TASK-23 completes.

**TASKs 41-43 are Go code tasks** - parallel with each other and with SKILL.md edits after TASK-23 completes. TASK-42 and TASK-43 coordinate on schema changes (both modify `CREATE TABLE` statement).

---

### TASK-23: [BLOCKING] Fix tokenizer and complete e5-small-v2 switch

**Description:** Replace hash-based tokenization with proper BERT WordPiece tokenization and complete the switch from all-MiniLM-L6-v2 to e5-small-v2 model. This is the foundation for accurate semantic similarity search - all other memory tasks depend on this.

**Status:** Ready

**Acceptance Criteria:**
- [ ] New file `internal/memory/tokenizer.go` implements WordPiece tokenizer (~80 lines)
- [ ] Tokenizer loads vocab from `intfloat/e5-small-v2/resolve/main/vocab.txt` (232KB)
- [ ] Tokenizer returns token IDs wrapped with `[CLS]` ... `[SEP]` special tokens
- [ ] New file `internal/memory/tokenizer_test.go` with unit tests (known tokens, subword splitting, property-based tests via rapid)
- [ ] `embeddings.go` updated: replace hash tokenization block (lines 388-407) with WordPiece call
- [ ] `embeddings.go` updated: `e5SmallModelURL` constant changed to `intfloat/e5-small-v2/resolve/main/model.onnx`
- [ ] `embeddings.go` updated: `e5SmallVocabURL` constant added
- [ ] `embeddings.go` updated: `generateEmbeddingONNX` adds `isQuery bool` parameter
- [ ] `embeddings.go` updated: text prefixed with `"query: "` when `isQuery=true`, `"passage: "` when `isQuery=false`
- [ ] `memory.go` updated: `Query()` passes `isQuery=true`, `createEmbeddings()` passes `isQuery=false`
- [ ] `hashString` function removed from `embeddings.go`
- [ ] Vocab download and caching implemented (same pattern as model download)
- [ ] `ClearSessionCache` updated to reset vocab cache for test isolation
- [ ] Model ID marker file created: stores `model_id.txt` alongside embeddings.db, deletes DB on model change
- [ ] `TestIntegration_SemanticSimilarityExampleErrorAndException` passes
- [ ] `TestIntegration_SemanticSimilarityRanksRelatedHigher` passes
- [ ] `go test ./internal/memory/...` passes
- [ ] `mage check` passes

**Files:**
- `internal/memory/tokenizer.go` (new)
- `internal/memory/tokenizer_test.go` (new)
- `internal/memory/embeddings.go` (modify)
- `internal/memory/memory.go` (modify)
- `internal/memory/extract.go` (modify)

**Dependencies:** None

**Traces to:** ARCH-052, ARCH-053, REQ-006

---

### TASK-24: Session-end capture in orchestrator

**Description:** Add `projctl memory session-end` to orchestrator's end-of-command sequence, running FIRST before integrate/trace commands.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `skills/project/SKILL.md` end-of-command block includes `projctl memory session-end -p "<issue-id>"`
- [ ] Session-end runs BEFORE `projctl integrate features`
- [ ] Session-end receives issue ID from orchestrator context
- [ ] `grep -c "memory session-end" ~/.claude/skills/project/SKILL.md` returns 1

**Files:** `skills/project/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-054, REQ-007

---

### TASK-25: PM interview producer memory reads

**Description:** Add memory queries to pm-interview-producer GATHER phase for past requirements and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase includes `projctl memory query "prior requirements for <project-domain>"`
- [ ] GATHER phase includes `projctl memory query "decisions about <feature-area>"`
- [ ] GATHER phase includes `projctl memory query "known failures in requirements validation"`
- [ ] Memory queries run BEFORE interview questions
- [ ] Graceful degradation: "If memory is unavailable, continue without it"
- [ ] `grep -c "projctl memory query" pm-interview-producer/SKILL.md` returns >= 3

**Files:** `skills/pm-interview-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-26: Design interview producer memory reads

**Description:** Add memory queries to design-interview-producer GATHER phase for past design decisions and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase includes `projctl memory query "design patterns for <project-domain>"`
- [ ] GATHER phase includes `projctl memory query "user experience decisions for <project-domain>"`
- [ ] GATHER phase includes `projctl memory query "known failures in design validation"`
- [ ] Memory queries run BEFORE interview questions
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" design-interview-producer/SKILL.md` returns >= 3

**Files:** `skills/design-interview-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-27: Architecture interview producer memory reads

**Description:** Add memory query to arch-interview-producer GATHER phase for known validation failures (domain queries already exist).

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase includes `projctl memory query "known failures in architecture validation"`
- [ ] Query added alongside existing domain queries
- [ ] Graceful degradation documented
- [ ] `grep -c "known failures" arch-interview-producer/SKILL.md` returns >= 1

**Files:** `skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-28: QA memory reads and writes

**Description:** Add memory queries to QA LOAD phase (verification backstop) and memory writes to RETURN phase (failure persistence).

**Status:** Ready

**Acceptance Criteria:**
- [ ] LOAD phase (after line 60) includes `projctl memory query "known failures in <artifact-type> validation"`
- [ ] LOAD phase documents cross-reference with artifact: did producer address known patterns?
- [ ] RETURN phase documents: when reporting `improvement-request` or `escalate-phase`, persist via `projctl memory learn`
- [ ] Learn message format: `"QA failure in <artifact-type>: <check-id> - <failure description>"`
- [ ] Only persist on failure verdicts (not `approved`)
- [ ] `grep -c "projctl memory query" qa/SKILL.md` returns >= 1
- [ ] `grep -c "projctl memory learn" qa/SKILL.md` returns >= 1

**Files:** `skills/qa/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-056, REQ-010

---

### TASK-29: Context explorer auto-memory enrichment

**Description:** Add auto-memory enrichment policy to context-explorer for automatic memory queries when missing.

**Status:** Ready

**Acceptance Criteria:**
- [ ] "Auto-memory enrichment" section added after "Execute Queries" section (after line 68)
- [ ] Policy: when query list lacks explicit memory query, auto-add one from first semantic/file query's topic
- [ ] Skip if: request has memory queries, or topic text < 3 words
- [ ] Memory failures are non-blocking
- [ ] `grep -c "Auto-memory" context-explorer/SKILL.md` returns 1

**Files:** `skills/context-explorer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-063, REQ-008

---

### TASK-30: Retro producer memory reads and writes

**Description:** Add memory queries to retro-producer GATHER phase and memory writes to PRODUCE phase for learnings persistence.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase includes `projctl memory query "retrospective challenges"`
- [ ] GATHER phase includes `projctl memory query "process improvement recommendations"`
- [ ] PRODUCE phase (after line 61) includes `projctl memory learn` for each success
- [ ] PRODUCE phase includes `projctl memory learn` for each challenge
- [ ] PRODUCE phase includes `projctl memory learn` for each High/Medium recommendation
- [ ] Learn message format documented for each type
- [ ] `grep -c "projctl memory query" retro-producer/SKILL.md` returns >= 2
- [ ] `grep -c "projctl memory learn" retro-producer/SKILL.md` returns >= 1

**Files:** `skills/retro-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-057, REQ-011

---

### TASK-31: Orchestrator startup memory reads

**Description:** Add memory queries to orchestrator startup sequence (after workflow set, before step loop) for past learnings.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Startup section (after line 61) includes `projctl memory query "lessons from past projects"`
- [ ] Startup section includes `projctl memory query "common challenges in <workflow-type> projects"`
- [ ] Query results included in orchestrator's working context
- [ ] Queries surface: session summaries, retro learnings, QA failure patterns
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" project/SKILL.md` returns >= 2

**Files:** `skills/project/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-059, REQ-012

---

### TASK-32: Breakdown producer memory reads

**Description:** Add memory queries to breakdown-producer GATHER phase for decomposition patterns and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase (after step 3, line 35) includes `projctl memory query "task decomposition for <project-domain>"`
- [ ] GATHER phase includes `projctl memory query "known failures in task decomposition"`
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" breakdown-producer/SKILL.md` returns >= 2

**Files:** `skills/breakdown-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-33: TDD green producer memory reads

**Description:** Add memory queries to tdd-green-producer GATHER phase for implementation patterns and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase (after step 3, line 28) includes `projctl memory query "implementation patterns for <package-domain>"`
- [ ] GATHER phase includes `projctl memory query "QA corrections for <artifact-type>"`
- [ ] GATHER phase includes `projctl memory query "known failures in implementation validation"`
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" tdd-green-producer/SKILL.md` returns >= 3

**Files:** `skills/tdd-green-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-34: TDD refactor producer memory reads

**Description:** Add memory queries to tdd-refactor-producer GATHER phase for refactoring patterns and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase (after step 3, line 35) includes `projctl memory query "refactoring patterns for <language>"`
- [ ] GATHER phase includes `projctl memory query "code organization preferences"`
- [ ] GATHER phase includes `projctl memory query "known failures in refactor validation"`
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" tdd-refactor-producer/SKILL.md` returns >= 3

**Files:** `skills/tdd-refactor-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-35: Alignment producer memory reads

**Description:** Add memory queries to alignment-producer GATHER phase for alignment errors and domain boundary violations.

**Status:** Ready

**Acceptance Criteria:**
- [ ] GATHER phase (after step 2, line 29) includes `projctl memory query "common alignment errors"`
- [ ] GATHER phase includes `projctl memory query "domain boundary violations"`
- [ ] GATHER phase includes `projctl memory query "known failures in alignment validation"`
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" alignment-producer/SKILL.md` returns >= 3

**Files:** `skills/alignment-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-36: TDD red producers memory reads

**Description:** Add memory queries to tdd-red-producer and tdd-red-infer-producer GATHER phases for test patterns and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `tdd-red-producer/SKILL.md` GATHER phase (after step 4, line 33) includes `projctl memory query "test patterns for <package-domain>"`
- [ ] `tdd-red-producer/SKILL.md` includes `projctl memory query "edge case patterns for <artifact-type>"`
- [ ] `tdd-red-producer/SKILL.md` includes `projctl memory query "known failures in test validation"`
- [ ] `tdd-red-infer-producer/SKILL.md` GATHER phase (after step 2, line 37) includes same 3 queries
- [ ] Both files document graceful degradation
- [ ] `grep -c "projctl memory query" tdd-red-producer/SKILL.md` returns >= 3
- [ ] `grep -c "projctl memory query" tdd-red-infer-producer/SKILL.md` returns >= 3

**Files:**
- `skills/tdd-red-producer/SKILL.md`
- `skills/tdd-red-infer-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-37: Universal yield capture in orchestrator

**Description:** Add `projctl memory extract` to orchestrator's spawn-producer handler to capture decisions and learnings from ALL producers automatically.

**Status:** Ready

**Acceptance Criteria:**
- [ ] spawn-producer handler (after line 158-161) includes `projctl memory extract -f .claude/projects/<issue>/result.toml -p <issue-id>`
- [ ] Extract runs BEFORE `projctl step complete`
- [ ] Extract is best-effort (log warning and continue on failure)
- [ ] Extract applies to ALL producers universally (single integration point)
- [ ] `grep -c "projctl memory extract" project/SKILL.md` returns >= 1

**Files:** `skills/project/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-058, REQ-009

---

### TASK-38: Infer producers memory reads

**Description:** Add memory queries to pm-infer-producer, design-infer-producer, and arch-infer-producer GATHER phases (same patterns as interview producers).

**Status:** Ready

**Acceptance Criteria:**
- [ ] `pm-infer-producer/SKILL.md` GATHER phase (after step 3, line 52) includes requirements queries + known failures
- [ ] `design-infer-producer/SKILL.md` GATHER phase (after step 2, line 54) includes design queries + known failures
- [ ] `arch-infer-producer/SKILL.md` GATHER phase (after step 3, line 49) includes arch queries + known failures
- [ ] All three files document graceful degradation
- [ ] `grep -c "projctl memory query" pm-infer-producer/SKILL.md` returns >= 3
- [ ] `grep -c "projctl memory query" design-infer-producer/SKILL.md` returns >= 3
- [ ] `grep -c "projctl memory query" arch-infer-producer/SKILL.md` returns >= 3

**Files:**
- `skills/pm-infer-producer/SKILL.md`
- `skills/design-infer-producer/SKILL.md`
- `skills/arch-infer-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-39: Doc and summary producers memory reads

**Description:** Add memory queries to doc-producer and summary-producer GATHER phases for documentation conventions and known validation failures.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `doc-producer/SKILL.md` GATHER phase (after step 5, line 33) includes `projctl memory query "documentation conventions for <project-domain>"`
- [ ] `doc-producer/SKILL.md` includes `projctl memory query "known failures in documentation validation"`
- [ ] `summary-producer/SKILL.md` GATHER phase (after step 6, line 34) includes `projctl memory query "project summary patterns"`
- [ ] `summary-producer/SKILL.md` includes `projctl memory query "known failures in summary validation"`
- [ ] Both files document graceful degradation
- [ ] `grep -c "projctl memory query" doc-producer/SKILL.md` returns >= 2
- [ ] `grep -c "projctl memory query" summary-producer/SKILL.md` returns >= 2

**Files:**
- `skills/doc-producer/SKILL.md`
- `skills/summary-producer/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-40: Next-steps memory reads

**Description:** Add memory queries to next-steps GATHER phase for past recommendations and follow-up patterns.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Gather Context phase (after step 4, line 29) includes `projctl memory query "past project recommendations"`
- [ ] Gather Context phase includes `projctl memory query "follow-up patterns for <project-domain>"`
- [ ] Graceful degradation documented
- [ ] `grep -c "projctl memory query" next-steps/SKILL.md` returns >= 2

**Files:** `skills/next-steps/SKILL.md`

**Dependencies:** TASK-23

**Traces to:** ARCH-055, REQ-008

---

### TASK-41: [visual] Memory promotion pipeline (Go code)

**Description:** Implement retrieval tracking and promotion candidate query for high-value memories that should graduate to CLAUDE.md.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `embeddings.go` schema updated: add `retrieval_count`, `last_retrieved`, `projects_retrieved` columns to `CREATE TABLE`
- [ ] `embeddings.go` `searchSimilar` updated: increment counters after returning results
- [ ] `memory.go` `QueryOpts` updated: add `Project` field, pass through to search
- [ ] `memory.go` new function: `Promote(opts PromoteOpts) (*PromoteResult, error)`
- [ ] `Promote` queries: `retrieval_count >= MinRetrievals (default 3)` AND `unique projects >= MinProjects (default 2)`
- [ ] `cmd/projctl/memory.go` new subcommand: `memory promote` with args for thresholds
- [ ] Unit tests in `memory_test.go`: promotion query, retrieval counting, project deduplication
- [ ] CLI tests in `cmd/projctl/memory_test.go`: promote command
- [ ] `retro-producer/SKILL.md` GATHER phase includes `projctl memory promote` check
- [ ] Retro recommendations include: "Consider promoting to CLAUDE.md: <content>"
- [ ] `grep -c "projctl memory promote" retro-producer/SKILL.md` returns >= 1
- [ ] `go test ./internal/memory/...` passes
- [ ] `go test ./cmd/projctl/...` passes
- [ ] `mage check` passes

**Files:**
- `internal/memory/embeddings.go` (modify)
- `internal/memory/memory.go` (modify)
- `internal/memory/memory_test.go` (modify)
- `cmd/projctl/memory.go` (modify)
- `cmd/projctl/memory_test.go` (modify)
- `skills/retro-producer/SKILL.md` (modify)

**Dependencies:** TASK-23

**Traces to:** ARCH-060, REQ-013

---

### TASK-42: [visual] External knowledge capture (Go code + SKILL.md)

**Description:** Implement source attribution for external memories and add optional WebSearch to high-value GATHER phases.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `embeddings.go` schema updated: add `source_type` (default `"internal"`) and `confidence` (default 1.0) columns
- [ ] `memory.go` `LearnOpts` updated: add `Source` field (`"internal"` or `"external"`)
- [ ] `memory.go` `Learn` updated: external memories get initial confidence 0.7
- [ ] `cmd/projctl/memory.go` `memoryLearnArgs` updated: add `--source` flag
- [ ] Unit tests in `memory_test.go`: source_type storage and retrieval, confidence values
- [ ] CLI tests in `cmd/projctl/memory_test.go`: `--source` flag
- [ ] `arch-interview-producer/SKILL.md` GATHER phase: optional WebSearch + `projctl memory learn --source external`
- [ ] `design-interview-producer/SKILL.md` GATHER phase: optional WebSearch + `projctl memory learn --source external`
- [ ] `tdd-green-producer/SKILL.md` GATHER phase: optional WebSearch + `projctl memory learn --source external`
- [ ] All three SKILL.md files document: skip if domain well-covered or web unavailable
- [ ] `grep -c "WebSearch" arch-interview-producer/SKILL.md` returns >= 1
- [ ] `grep -c "WebSearch" design-interview-producer/SKILL.md` returns >= 1
- [ ] `grep -c "WebSearch" tdd-green-producer/SKILL.md` returns >= 1
- [ ] `grep -c "\-\-source external" arch-interview-producer/SKILL.md` returns >= 1
- [ ] `go test ./internal/memory/...` passes
- [ ] `go test ./cmd/projctl/...` passes
- [ ] `mage check` passes

**Files:**
- `internal/memory/embeddings.go` (modify - coordinates with TASK-43 on schema)
- `internal/memory/memory.go` (modify)
- `internal/memory/memory_test.go` (modify)
- `cmd/projctl/memory.go` (modify)
- `cmd/projctl/memory_test.go` (modify)
- `skills/arch-interview-producer/SKILL.md` (modify)
- `skills/design-interview-producer/SKILL.md` (modify)
- `skills/tdd-green-producer/SKILL.md` (modify)

**Dependencies:** TASK-23

**Traces to:** ARCH-061, REQ-014

---

### TASK-43: [visual] Memory hygiene (Go code + SKILL.md)

**Description:** Implement confidence decay, pruning, and conflict detection for memory quality maintenance.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `embeddings.go` schema updated: `confidence` column included (coordinates with TASK-42)
- [ ] `embeddings.go` `searchSimilar` updated: rank by `(cosine_similarity * confidence)` instead of raw similarity
- [ ] `memory.go` new function: `Decay(opts DecayOpts) (*DecayResult, error)` - multiplies confidence by factor (default 0.9)
- [ ] `memory.go` new function: `Prune(opts PruneOpts) (*PruneResult, error)` - removes entries below threshold (default 0.1)
- [ ] `memory.go` `Learn` updated: check for high-similarity existing entries (>0.85), return conflict info
- [ ] `cmd/projctl/memory.go` new subcommands: `memory decay`, `memory prune`
- [ ] Unit tests in `memory_test.go`: decay reduces confidence, prune removes low-confidence, conflict detection, confidence-weighted search
- [ ] CLI tests in `cmd/projctl/memory_test.go`: decay and prune commands
- [ ] `project/SKILL.md` end-of-command sequence: add `projctl memory decay` and `projctl memory prune` after session-end
- [ ] Decay and prune run BEFORE integrate/trace commands
- [ ] `grep -c "memory decay" project/SKILL.md` returns >= 1
- [ ] `grep -c "memory prune" project/SKILL.md` returns >= 1
- [ ] `go test ./internal/memory/...` passes
- [ ] `go test ./cmd/projctl/...` passes
- [ ] `mage check` passes

**Files:**
- `internal/memory/embeddings.go` (modify - coordinates with TASK-42 on schema)
- `internal/memory/memory.go` (modify)
- `internal/memory/memory_test.go` (modify)
- `cmd/projctl/memory.go` (modify)
- `cmd/projctl/memory_test.go` (modify)
- `skills/project/SKILL.md` (modify)

**Dependencies:** TASK-23

**Traces to:** ARCH-062, REQ-015, REQ-016

---

### ISSUE-152 Task Summary

| Task | Title | Dependencies | Type | Key Traces |
|------|-------|--------------|------|------------|
| TASK-23 | [BLOCKING] Fix tokenizer and complete e5-small-v2 switch | None | Go code | ARCH-052, ARCH-053, REQ-006 |
| TASK-24 | Session-end capture in orchestrator | TASK-23 | SKILL.md | ARCH-054, REQ-007 |
| TASK-25 | PM interview producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-26 | Design interview producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-27 | Architecture interview producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-28 | QA memory reads and writes | TASK-23 | SKILL.md | ARCH-056, REQ-010 |
| TASK-29 | Context explorer auto-memory enrichment | TASK-23 | SKILL.md | ARCH-063, REQ-008 |
| TASK-30 | Retro producer memory reads and writes | TASK-23 | SKILL.md | ARCH-057, REQ-011 |
| TASK-31 | Orchestrator startup memory reads | TASK-23 | SKILL.md | ARCH-059, REQ-012 |
| TASK-32 | Breakdown producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-33 | TDD green producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-34 | TDD refactor producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-35 | Alignment producer memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-36 | TDD red producers memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-37 | Universal yield capture in orchestrator | TASK-23 | SKILL.md | ARCH-058, REQ-009 |
| TASK-38 | Infer producers memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-39 | Doc and summary producers memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-40 | Next-steps memory reads | TASK-23 | SKILL.md | ARCH-055, REQ-008 |
| TASK-41 | Memory promotion pipeline | TASK-23 | Go code + SKILL.md | ARCH-060, REQ-013 |
| TASK-42 | External knowledge capture | TASK-23 | Go code + SKILL.md | ARCH-061, REQ-014 |
| TASK-43 | Memory hygiene | TASK-23 | Go code + SKILL.md | ARCH-062, REQ-015, REQ-016 |

---

### Summary Metrics (ISSUE-152)

| Metric | Count |
|--------|-------|
| Total tasks | 21 |
| BLOCKING tasks | 1 (TASK-23) |
| SKILL.md-only tasks | 17 (TASK-24 through TASK-40) |
| Go code tasks | 3 (TASK-41, TASK-42, TASK-43) |
| New Go files | 2 (tokenizer.go, tokenizer_test.go) |
| Modified Go files | 8 (embeddings.go, memory.go, extract.go, memory_test.go, cmd/memory.go, cmd/memory_test.go) |
| Modified SKILL.md files | 18 (all LLM-driven skills + orchestrator) |
| Architecture items covered | 12 (ARCH-052 through ARCH-063) |
| Requirements covered | 12 (REQ-006 through REQ-017) |

**Critical path:** TASK-23 (foundation) → all other tasks parallel

**Parallel opportunities:**
- TASKs 24-40 (SKILL.md edits) are fully independent
- TASKs 41-43 (Go code) are independent of SKILL.md edits
- TASK-42 and TASK-43 coordinate on schema changes but implement different logic

**Testing strategy:**
- Unit tests for tokenizer (TASK-23)
- Property-based tests for tokenizer (rapid)
- Integration tests for semantic similarity (TASK-23)
- Unit tests for promotion/decay/prune (TASKs 41, 43)
- CLI tests for new commands (TASKs 41, 42, 43)
- Grep-based verification for SKILL.md changes (TASKs 24-40)

**Schema coordination:** TASK-42 and TASK-43 both modify the `CREATE TABLE embeddings` statement:
- TASK-42 adds: `source_type`, `confidence`
- TASK-43 uses: `confidence` (coordinates with TASK-42)
- Both add to existing columns from TASK-41: `retrieval_count`, `last_retrieved`, `projects_retrieved`
- Final schema has all columns from TASKs 41, 42, and 43

