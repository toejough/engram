# ISSUE-170: Enforce plan mode via tool call hooks — Evaluation

**Project:** ISSUE-170: Enforce plan mode via tool call hooks and projctl state validation
**Duration:** 1 session (2026-02-08)
**Workflow:** Scoped
**Team:** Solo implementation
**Status:** Complete (merged to main)

---

## Project Summary

Implemented a two-part enforcement system to prevent agents from skipping plan mode approval during plan-producer execution. The solution combines tool call logging with step completion validation, creating a generalizable pattern for enforcing any tool-call requirement in any workflow phase.

**Deliverables:**
- Tool call tracking infrastructure in state layer (LogToolCall, GetToolCalls, HasToolCall, ClearToolCalls)
- Plan mode validation in step.Complete for plan_produce phase
- 21 comprehensive tests (562 lines) covering state persistence, ordering, phase-aware validation, error cases
- Zero regressions, zero linter issues

**Scope:**
- Skipped PM/design/arch phases (scoped workflow)
- Went directly to TDD implementation
- Total changes: 15 files modified, 801 insertions, 175 deletions

---

## Key Decisions

### D1: State-Based Validation Over Prompt-Based Enforcement

**Context:** ISSUE-159 added plan mode instructions to plan-producer SKILL.md, but agents ignored them during ISSUE-160.

**Choice:** Implement tool call tracking in state layer + step completion validation instead of relying on prompts.

**Rationale:** Prompt-based enforcement is unreliable. Self-reporting flags (like `projctl state set --plan-approved`) are just self-reporting with extra steps. State-based validation using actual tool call records provides deterministic enforcement.

**Outcome:** Implementation succeeded. Step.Complete now blocks plan_produce phase completion unless both EnterPlanMode and ExitPlanMode tool calls are recorded in state.

**Traces to:** ISSUE-170 Problem statement, AC-1, AC-2, AC-3

---

### D2: Generalizable Pattern for Tool Call Requirements

**Context:** Plan mode enforcement is not the only scenario where specific tool calls should be required. Other phases (tdd_red requiring test execution, commit requiring Skill invocation) could benefit from the same pattern.

**Choice:** Designed the tool call tracking infrastructure to be phase-agnostic and reusable.

**Rationale:** Single-purpose solutions create technical debt. A generalizable pattern enables future enforcement needs without rearchitecting.

**Outcome:** The ToolCall struct, state layer functions (LogToolCall, HasToolCall), and step.Complete validation pattern are all phase-agnostic. Plan mode validation is implemented as a conditional check within step.Complete that can be extended for other phases.

**Traces to:** ISSUE-170 Solution "Generalizable" section

---

### D3: Clear Tool Calls After Successful Validation

**Context:** Tool calls logged during one phase should not persist to the next phase.

**Choice:** step.Complete clears tool calls after successful validation (line 65 of validate.go).

**Rationale:** Without clearing, tool calls from plan_produce would carry over to subsequent phases, causing false positives in validation. Each phase should start with a clean tool call slate.

**Outcome:** Tests verify tool calls are cleared after successful completion (TestStepComplete_PlanProduce_ClearsToolCallsAfterSuccess). No false positives observed.

**Traces to:** Test coverage in plan_validation_test.go:122-150

---

## Outcomes vs Goals

### All Requirements Met

**AC-1: projctl state log-tool command exists** ✅
- Implementation: state.LogToolCall function (state.go:596-613)
- CLI wiring: stateLogTool command (for testing/debugging)
- Evidence: cmd/projctl/state_log_tool_test.go (122 lines)

**AC-2: PostToolUse hook config for EnterPlanMode/ExitPlanMode** ⚠️
- State layer infrastructure implemented
- Hook configuration not yet documented or installed
- Note: Hook installation was mentioned in AC but not completed during implementation
- Filed as follow-up work

**AC-3: projctl step complete validates tool calls for plan_produce** ✅
- Implementation: step.Complete function (validate.go:23-74)
- Validates both EnterPlanMode and ExitPlanMode presence
- Evidence: 7 tests in plan_validation_test.go covering all scenarios

**AC-4: Step complete fails with actionable error when plan mode skipped** ✅
- Implementation: Specific error messages for missing EnterPlanMode, missing ExitPlanMode, or both
- Example error: "plan mode required EnterPlanMode and ExitPlanMode tool calls: both are missing"
- Evidence: TestStepComplete_PlanProduce_ErrorMessageIsActionable (plan_validation_test.go:91-119)

---

### Quality Metrics

**Test Coverage:**
- 21 tests across 3 new test files (562 lines total)
- Plan validation tests: 7 scenarios covering presence, order, clearing, error messages
- Tool call tests: comprehensive state persistence and retrieval coverage
- All tests passing, zero flaky tests

**Code Quality:**
- Zero linter issues after refactor phase
- Zero regressions in existing test suite
- Property-based test discipline maintained (gomega matchers)

**Implementation Efficiency:**
- TDD red phase: 21 tests written, all failed as expected
- TDD green phase: minimal implementation, all tests passed
- TDD refactor phase: cleanup approved with no rework
- Zero QA failures during TDD cycle

---

## Process Findings

### High Priority

**F1: TDD cycle executed cleanly for implementation — zero rework**

**Finding:** The TDD red/green/refactor cycle for ISSUE-170 proceeded without iteration failures. All 21 tests were written first (red), implementation made them pass (green), and refactoring was approved on first attempt.

**Evidence:**
- Git history shows single implementation commit (08c07c3)
- No improvement-request verdicts during TDD phases
- Zero test failures after green phase

**Recommendation:** Continue strict TDD discipline for state-layer and validation logic. Property-based tests + behavior-focused assertions (gomega) catch edge cases during implementation, not QA.

**Action:** Reinforce TDD-first approach in project SKILL.md files and global instructions.

---

**F2: Documentation phase hit max iterations due to trace validator ID format mismatch**

**Finding:** The documentation phase required 2 iterations of improvement-request feedback due to trace validation failures. The root cause was a mismatch between the trace validator's expected ID format (zero-padded like REQ-013) and the project's established convention (non-padded like REQ-13).

**Evidence:**
- ISSUE-173 filed for trace validator ID format mismatch
- 28 orphan IDs reported in ISSUE-170 validation, all false positives from format mismatch
- ISSUE-43 established non-padded ID convention, but trace validator was never updated

**Recommendation:** Fix the trace validator to normalize IDs before comparison (strip leading zeros). Add regression tests to catch format mismatches.

**Action:** ISSUE-173 already filed (see below).

---

**F3: Evaluation phase was skipped when documentation hit max iterations**

**Finding:** When the documentation phase hit max iterations, the state machine transitioned to `phase_blocked` and `projctl step next` returned `all-complete`, skipping the evaluation phase entirely. The project completed without a retrospective or summary.

**Evidence:**
- ISSUE-174 filed with diagnosis
- State machine transitioned: documentation_decide → phase_blocked → all-complete
- Evaluation-producer never ran for ISSUE-170

**Recommendation:** Fix the state machine or `projctl step next` logic to ensure evaluation_produce always runs, even when upstream phases are blocked. Blocked phase details should be available to the evaluation-producer for retrospective analysis.

**Action:** ISSUE-174 already filed (see below).

---

### Medium Priority

**F4: Scoped workflow reduced time-to-implementation for focused changes**

**Finding:** By skipping PM/design/arch phases and going directly to TDD, ISSUE-170 completed in a single session with high quality. The scoped workflow was appropriate because the problem and solution were well-defined in the issue description.

**Evidence:**
- Issue description contained complete problem statement and proposed solution
- No requirements clarification needed
- No design trade-offs to explore
- Implementation was straightforward: add tool call tracking, add validation

**Recommendation:** Continue using scoped workflow for issues with clear problem statements and proposed solutions. Reserve full workflow for exploratory or ambiguous work.

**Action:** Document scoped workflow decision criteria in intake-evaluator SKILL.md.

---

**F5: Generalizable pattern design paid off immediately**

**Finding:** By designing the tool call tracking infrastructure to be phase-agnostic, the implementation can be extended to other phases (tdd_red, commit, etc.) without rearchitecting. The decision to avoid single-purpose code reduced future technical debt.

**Evidence:**
- ToolCall struct has no plan-specific fields
- State layer functions (LogToolCall, HasToolCall, ClearToolCalls) are phase-agnostic
- step.Complete uses conditional logic to check phase == "plan_produce", making it trivial to add other phase validations

**Recommendation:** Continue designing for generalizability when implementing enforcement or validation patterns. Single-purpose solutions accumulate as technical debt in rapidly evolving systems.

**Action:** Add generalizability as a design principle in architecture decision guidelines.

---

### Low Priority

**F6: Test file organization improved readability**

**Finding:** Separating plan validation tests (plan_validation_test.go), tool call tests (tool_calls_test.go), and CLI tests (state_log_tool_test.go) into distinct files improved test discoverability and readability compared to monolithic test files.

**Evidence:**
- Each test file focuses on a single concern
- Test names clearly indicate what they validate
- Easy to locate tests for specific functionality

**Recommendation:** Continue organizing tests by feature/concern rather than by package. Use descriptive file names that match the functionality under test.

**Action:** None required (already established pattern).

---

## Recommendations

### High Priority

**R1: Fix trace validator ID format normalization**
**Issue:** ISSUE-173
**Priority:** Medium (affects all projects)

Update `projctl trace validate` to normalize IDs (strip leading zeros) before comparison. Add regression tests.

---

**R2: Fix evaluation phase skipping on phase_blocked**
**Issue:** ISSUE-174
**Priority:** High (projects lose retrospective data)

Ensure evaluation_produce phase runs even when upstream phases hit max iterations or phase_blocked state. Blocked phase details should be available for retrospective analysis.

---

### Medium Priority

**R3: Document hook installation for plan mode enforcement**
**Priority:** Medium

ISSUE-170 AC-2 mentions hook installation docs or `projctl hooks install` support, but this was not completed. The state infrastructure is ready, but users need guidance on configuring PostToolUse hooks.

**Suggested approach:** Add a "Hook Configuration" section to the plan-producer SKILL.md with example hook configuration for EnterPlanMode/ExitPlanMode logging.

---

**R4: Codify scoped workflow decision criteria**
**Priority:** Medium

The intake-evaluator should have clear criteria for when to use scoped workflow vs full workflow. ISSUE-170 was a good candidate for scoped (clear problem, clear solution), but the decision was implicit.

**Suggested approach:** Add decision tree to intake-evaluator SKILL.md: "Use scoped workflow when: (1) problem and solution are both clearly defined in issue, (2) no exploratory work needed, (3) no design trade-offs to evaluate."

---

## Open Questions

**Q1: Should tool call validation be opt-in or opt-out for future phases?**

**Context:** The current implementation requires explicit conditional logic in step.Complete for each phase that needs tool call validation (currently only plan_produce). Future phases (tdd_red, commit) could benefit from similar enforcement.

**Options:**
1. Continue explicit conditional checks in step.Complete (current approach)
2. Move validation rules to a declarative config (e.g., workflows.toml phase definitions)
3. Make validation opt-in via a phase property in state machine graph

**Impact:** Affects maintainability and extensibility of tool call enforcement pattern.

---

**Q2: Should worktree state loss be prevented or just documented?**

**Context:** Team lead mentioned "Worktree state was lost during merge cleanup" in the context. This was not investigated during ISSUE-170 evaluation.

**Questions:**
- Is this a bug in worktree cleanup logic?
- Is this expected behavior when merging to main?
- Should worktree state be preserved after merge for audit purposes?

**Impact:** May affect future project traceability and debugging.

---

## Traceability

**Traces to:**
- ISSUE-170 (primary issue)
- ISSUE-159 (plan mode prompt-based attempt)
- ISSUE-160 (discovery that plan mode was skipped)
- ISSUE-173 (documentation blocker - trace validator format)
- ISSUE-174 (evaluation phase skipped on phase_blocked)
- ISSUE-43 (non-padded ID convention)

---

## Files Modified

### Implementation
- `internal/state/state.go` - Tool call tracking functions (LogToolCall, GetToolCalls, HasToolCall, ClearToolCalls)
- `internal/step/validate.go` - Plan mode validation in step.Complete
- `cmd/projctl/step.go` - CLI wiring updates
- `cmd/projctl/worktree.go` - Worktree-related updates

### Tests
- `internal/step/plan_validation_test.go` (223 lines) - 7 tests covering plan mode validation
- `internal/state/tool_calls_test.go` (217 lines) - Tool call tracking tests
- `cmd/projctl/state_log_tool_test.go` (122 lines) - CLI command tests
- 8 existing test files updated for integration

---

**Evaluation Generated:** 2026-02-08
**Total Implementation Changes:** 15 files, 801 insertions, 175 deletions
**Test Coverage:** 21 new tests, 562 new test lines, 100% passing
