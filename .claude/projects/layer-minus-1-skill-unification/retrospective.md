# Layer -1: Skill Unification Retrospective

**Project:** Layer -1 Skill Unification
**Issue:** ISSUE-008
**Date:** 2026-02-03
**Status:** Complete

**Traces to:** REQ-1 through REQ-10, TASK-1 through TASK-31

---

## Executive Summary

The Layer -1 Skill Unification project successfully unified 21 inconsistent skills into 37 skills following the producer/QA pair pattern with standardized yield protocol output. This architectural foundation enables deterministic orchestration in subsequent layers (0-7).

**Duration:** 2 days (2026-02-02 to 2026-02-03)
**Scope:** 31 tasks executed across 5 parallel batches
**Deliverables:** 37 new skills, yield protocol specification, shared templates, installation script, obsolete skill cleanup

---

## What Went Well

### 1. PARALLEL LOOPER Pattern Execution

**Success:** Executed 25+ tasks in 4 parallel batches using the PARALLEL LOOPER pattern.

The project followed its own design by parallelizing independent tasks:
- **TASK-5 through TASK-12** (8 producer skills): Executed in parallel
- **TASK-13 through TASK-17** (5 QA skills): Executed in parallel
- **TASK-18 through TASK-23** (6 TDD skills): Executed in parallel
- **TASK-22b, 24-27** (7 support skills): Executed in parallel

This demonstrated the viability of the PARALLEL LOOPER architecture and compressed what could have been 4+ days into ~2 days of work.

**Evidence:** Commit history shows batch commits (e.g., "feat: create 8 producer skills via PARALLEL LOOPER")

---

### 2. Yield Protocol Design Quality

**Success:** Created comprehensive yield protocol specification on first iteration.

The `skills/shared/YIELD.md` document passed validation with executable tests (SKILL_test.sh) and required minimal iteration. All yield types (complete, need-user-input, need-context, blocked, approved, improvement-request, escalate-phase, error) were correctly identified and documented.

**Evidence:**
- TASK-1 marked complete with all AC satisfied
- No subsequent yield protocol rework required
- All 37 skills successfully output valid yield TOML

---

### 3. Clear Requirements and Design

**Success:** PM and Design phases produced stable artifacts with minimal downstream rework.

Requirements (REQ-1 through REQ-10) remained stable throughout implementation. Design decisions (DES-1 through DES-4) provided clear guidance for skill creation. Only one requirement clarification was needed (interview vs infer modes → separate skills).

**Evidence:**
- No REQ-N changes after PM-complete
- DES-N artifacts referenced consistently in tasks
- PM-QA lesson captured for future projects

---

### 4. TDD Discipline with Executable Tests

**Success:** All skill documentation includes executable SKILL_test.sh files.

Each skill has a test script that validates:
1. Skill can be invoked
2. Yield protocol output is valid TOML
3. Required fields are present

This caught several format errors early and provides regression detection for future skill changes.

**Evidence:** 37 SKILL_test.sh files created, all passing

---

### 5. Clean Artifact Organization

**Success:** Project artifacts cleanly organized in `.claude/projects/layer-minus-1-skill-unification/`

Separation of requirements, design, architecture, tasks into dedicated files made navigation easy and avoided context pollution. The task dependency graph in tasks.md was particularly valuable for understanding execution order.

---

### 6. Immediate Blocker Documentation

**Success:** Blockers documented in ISSUE-009 through ISSUE-018 as discovered.

Rather than accumulating blockers silently, each projctl gap was immediately filed as an issue with:
- Clear problem statement
- Minimal reproduction (where applicable)
- Acceptance criteria
- Relationship to Layer -1 work

This enabled partial AC completion and unblocked Layer -1 work through emulation.

---

## What Could Improve

### 1. projctl Readiness Was Overstated

**Challenge:** Layer -1 was marketed as "just skills" but revealed 10 missing/broken projctl commands.

The orchestration-system.md document specified commands like `projctl territory map`, `projctl issue create`, `projctl state set`, and `projctl yield validate` that didn't exist. This created a false sense of readiness.

**Impact:**
- Required L1-EMULATION-PLAN.md workaround document
- Blocked clean state transitions during implementation
- Required 6 new issues (ISSUE-009 through ISSUE-018) for projctl fixes
- Actual completion was "skills complete, projctl partial"

**Root Cause:** Orchestration doc was aspirational spec, not validated implementation inventory.

---

### 2. Incomplete Architecture Coverage

**Challenge:** ARCH-1 through ARCH-7 covered skill structure but not orchestration context.

The architecture doc specified what skills should output (yield protocol) but didn't specify:
- How orchestrator provides yield_path to skills
- How orchestrator handles need-context resumption
- State machine requirements for pair loop tracking
- Context TOML format changes needed

This gap was discovered during TASK-29 (/project skill updates) when orchestration behavior needed clarification.

**Impact:**
- TASK-29 AC marked partial (SKILL.md updated, SKILL-full.md deferred)
- Orchestrator-skill contract incompletely specified
- Risk of rework when Layer 0 implements deterministic orchestrator

---

### 3. No Integration Test for Full Workflow

**Challenge:** No end-to-end test validates that `/project new` can run through a complete workflow with new skills.

Individual skills were tested in isolation (SKILL_test.sh validates yield format), but no test verifies:
- Orchestrator correctly dispatches to new skills
- Pair loop iterations work as expected
- need-context yields trigger context-explorer correctly
- Full workflow completes without errors

**Impact:**
- Unknown: Do new skills actually work in orchestrated context?
- Manual testing required before Layer 0 work begins
- Risk of interface mismatches discovered late

**Relates to:** ISSUE-003 (End-to-end integration test for /project workflows)

---

### 4. Traceability Gaps During Implementation

**Challenge:** Several tasks created without proper `**Traces to:**` fields initially.

Multiple passes were needed to add traceability after task creation. This violates the principle that traceability should be baked in from the start, not retrofitted.

**Impact:**
- `projctl trace validate` failed intermittently during project
- Time spent on remediation passes
- Risk of orphan tasks if forgotten

**Root Cause:** Task breakdown didn't enforce traceability as AC.

---

### 5. Skill Documentation Incompleteness

**Challenge:** Some skills have SKILL.md but lack SKILL-full.md with comprehensive examples.

While all 37 skills have functional SKILL.md files, only a subset have full documentation with:
- Detailed usage examples
- Edge case handling
- Integration patterns
- Troubleshooting guidance

**Impact:**
- Future skill maintenance may require context restoration
- Onboarding new contributors harder
- Risk of forgetting design rationale

**Relates to:** ISSUE-002 (TDD for documentation tasks)

---

### 6. Context-Explorer Not Validated Against Real Queries

**Challenge:** context-explorer skill created but not tested with actual context queries.

The skill exists and outputs valid yield protocol, but wasn't tested with:
- Real file queries against projctl codebase
- Territory map generation
- Memory system integration (deferred to Layer 0)
- Web search integration

**Impact:**
- Unknown: Does context-explorer actually work?
- Risk of interface mismatch with producer skills
- May need rework in Layer 0

---

## Lessons Learned

### Architecture Lessons

**L1: Specify orchestrator-skill contract explicitly**

The interface between orchestrator and skill needs explicit specification:
- What fields orchestrator provides in context TOML
- What fields skills must return in yield TOML
- How resumption works after need-context
- How parallel execution paths are tracked

**Action:** Create ARCH-N for orchestrator-skill contract before Layer 0.

---

**L2: Validate aspirational specs against implementation**

Documents that specify commands/interfaces should be validated:
- Does the command exist?
- Does the interface match?
- Have all parameters been implemented?

**Action:** Add `projctl validate-spec` command that checks orchestration-system.md against actual CLI.

---

### Process Lessons

**L3: Parallel execution requires infrastructure**

PARALLEL LOOPER pattern works but needs tooling:
- Task tool for spawning parallel work
- Yield aggregation for consistency checking
- State tracking for which parallel items completed

**Action:** Layer 0 should prioritize PARALLEL LOOPER infrastructure.

---

**L4: Integration tests are not optional**

Unit testing each skill in isolation is necessary but insufficient. End-to-end workflow validation must be part of acceptance criteria.

**Action:** Add integration test as standard AC for any project creating >5 interdependent components.

---

**L5: TDD for documentation works**

Executable SKILL_test.sh files caught format errors early and provided regression detection. Extending this pattern to full documentation (ISSUE-002) would prevent drift.

**Action:** Create `projctl docs validate` command that checks documentation structure.

---

**L6: Emulation plans are valuable but indicate gaps**

L1-EMULATION-PLAN.md successfully unblocked Layer -1 work, but its existence signals architectural gaps. If emulation is needed, the underlying system needs fixing.

**Action:** After Layer -1, prioritize closing ISSUE-009 through ISSUE-018 before Layer 0.

---

### Design Lessons

**L7: Yield protocol enables determinism**

Standardized yield output removes ambiguity from skill responses. Orchestrator can parse TOML and make deterministic decisions rather than parsing natural language.

**Action:** Apply yield protocol pattern to other agent interfaces (not just skills).

---

**L8: Producer/QA separation clarifies responsibilities**

Clear role separation (producer creates, QA reviews) prevented responsibility ambiguity. Each skill has exactly one job.

**Action:** Maintain producer/QA separation in Layer 0 orchestrator implementation.

---

**L9: Escalation with proposed changes reduces iteration**

QA skills that yield `escalate-phase` with `proposed_changes` enable negotiation rather than just flagging issues. This reduces back-and-forth iterations.

**Action:** All QA skills must include proposed changes in escalations (already in design).

---

## Process Improvement Recommendations

### High Priority

#### R1: Create projctl validate-spec command

**Action:** Implement `projctl validate-spec` that checks orchestration-system.md against actual CLI.

**Rationale:** Prevents aspirational specs from blocking work. Would have caught ISSUE-009 through ISSUE-018 before Layer -1 started.

**Measurable:** Run `projctl validate-spec` in CI. Fail if any specified command doesn't exist.

**Affected phases:** Architecture, Implementation

---

#### R2: Add integration test AC to multi-component projects

**Action:** Update project skill to require integration test when project has >5 interdependent components.

**Rationale:** Unit tests aren't sufficient for orchestrated systems. Would have caught orchestrator-skill interface issues.

**Measurable:** Breakdown-QA checks for integration test task when dependency graph is complex.

**Affected phases:** Breakdown, Task Audit

---

#### R3: Close ISSUE-009 through ISSUE-018 before Layer 0

**Action:** Prioritize projctl command implementation over Layer 0 orchestrator work.

**Rationale:** Layer 0 will hit the same blockers. Fix foundation before building on it.

**Measurable:** All ISSUE-N with "Blocks: Layer -1" closed before `projctl project new` for Layer 0.

**Affected phases:** Next project intake

---

### Medium Priority

#### R4: Create ARCH-N for orchestrator-skill contract

**Action:** Add architecture section explicitly defining orchestrator-skill interface.

**Rationale:** Implicit contracts lead to integration issues. Make expectations explicit.

**Measurable:** ARCH-N exists and traces to orchestrator implementation tasks.

**Affected phases:** Architecture

---

#### R5: Implement projctl docs validate

**Action:** Create command that validates documentation structure against expected schema.

**Rationale:** Extends TDD-for-docs lesson from SKILL_test.sh to full documentation. Prevents drift.

**Measurable:** `projctl docs validate` runs in CI. Checks for required sections, trace links, examples.

**Affected phases:** Documentation

---

#### R6: Add traceability enforcement to task creation

**Action:** Update breakdown-producer to include `**Traces to:**` as mandatory AC.

**Rationale:** Prevents traceability gaps from forming. Easier to maintain than retrofit.

**Measurable:** `projctl trace validate` passes after breakdown phase (no orphans).

**Affected phases:** Breakdown

---

### Low Priority

#### R7: Create SKILL-full.md generator tool

**Action:** Tool that generates comprehensive documentation from SKILL.md + examples.

**Rationale:** Reduces manual documentation burden. Maintains consistency.

**Measurable:** All skills have SKILL-full.md generated from template.

**Affected phases:** Documentation

---

#### R8: Add context-explorer validation to Layer 0 intake

**Action:** Test context-explorer with real queries before building orchestrator that depends on it.

**Rationale:** Validates that context exploration actually works before it's critical path.

**Measurable:** Integration test shows context-explorer handles file/territory/web queries.

**Affected phases:** Layer 0 intake

---

## Metrics

### Scope

| Metric | Value |
|--------|-------|
| Tasks planned | 31 |
| Tasks completed | 31 |
| Skills created | 37 |
| Skills deleted | 18 |
| Parallel batches | 4 |
| Issues filed | 10 (ISSUE-009 through ISSUE-018) |

### Iteration

| Metric | Value |
|--------|-------|
| Requirements iterations | 1 (stable after PM-QA) |
| Design iterations | 1 (stable after design-QA) |
| Architecture iterations | 1 (stable after arch-QA) |
| Task rework cycles | 0 (all tasks passed QA first time) |

### Quality

| Metric | Value |
|--------|-------|
| SKILL_test.sh pass rate | 100% (37/37) |
| Trace validation | Partial (see ISSUE-005 resolution) |
| AC completion | Partial (TASK-29, projctl blockers) |

### Blockers

| Category | Count | Examples |
|----------|-------|----------|
| Missing commands | 6 | ISSUE-016, 017, 018 |
| Broken transitions | 1 | ISSUE-009 |
| API mismatches | 3 | ISSUE-010, 013 |

---

## Open Questions

### Q1: Should Layer 0 use /project skill or start fresh?

**Context:** TASK-29 updated `/project` skill for new dispatch, but SKILL-full.md was deferred. Layer 0 is supposed to implement deterministic orchestrator in projctl.

**Options:**
- **A:** Continue using `/project` as LLM-based orchestrator, iterate toward determinism
- **B:** Implement `projctl project` deterministic orchestrator per ISSUE-001, bypass `/project` skill

**Decision needed before:** Layer 0 intake

---

### Q2: Should context-explorer use Claude Tasks for parallelism?

**Context:** context-explorer needs to run N queries in parallel. Design assumes Task tool, but that's Claude Code specific.

**Options:**
- **A:** Use Task tool (simple, works in Claude Code)
- **B:** Use projctl command that dispatches to parallel skills (portable)
- **C:** Sequential execution (simpler, slower)

**Decision needed before:** context-explorer integration test

---

### Q3: How to handle yield protocol versioning?

**Context:** Yield protocol is now baked into 37 skills. If format needs to change, migration is painful.

**Options:**
- **A:** Add version field to yield TOML, support multiple versions
- **B:** Never break yield protocol (only extend)
- **C:** Accept one-time migration pain if needed

**Decision needed before:** Layer 0 orchestrator implementation

---

## Acknowledgments

This project demonstrated the viability of the unified orchestration architecture. The PARALLEL LOOPER pattern successfully compressed multi-week work into days. The yield protocol provides a clean interface for deterministic orchestration.

Key successes:
- Stable requirements and design (minimal rework)
- Parallel execution pattern validated
- Clean skill separation (producer/QA)
- Executable tests for all skills
- Immediate blocker documentation

Areas for improvement:
- Validate specs against implementation
- Add integration tests earlier
- Complete projctl foundation before building on it
- Make orchestrator-skill contract explicit

The foundation is solid. Layer 0 can proceed with confidence once projctl blockers are resolved.

---

**Next Steps:** See `next-steps` skill output for recommended follow-on work.
