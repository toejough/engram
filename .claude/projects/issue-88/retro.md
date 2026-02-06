# Retrospective: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Date:** 2026-02-06

## 1. Project Summary

**Scope:** Remove all references to the deprecated yield infrastructure (TOML-based skill communication) from documentation and code, replacing with current team messaging (SendMessage) equivalents.

**Deliverables:**
- 4 requirements (REQ-001 through REQ-004)
- 7 design decisions (DES-001 through DES-007)
- 5 architecture decisions (ARCH-001 through ARCH-005)
- 15 implementation tasks (TASK-1 through TASK-15)
- ~40 files modified across docs, skills, configs, and Go code
- 4 obsolete doc files deleted
- 2 root-level yield.toml files deleted
- 7 context config files cleaned
- Verification test script and Go test added

**Phases completed:** PM, Design, Architecture, Breakdown, Implementation (TDD red/green/refactor), Documentation, Alignment

**Team roles:** team-lead (opus), pm-interview-producer (sonnet), design-producer (sonnet), arch-producer (sonnet), breakdown-producer (sonnet), tdd-red-producer (sonnet), tdd-green-producer (sonnet), tdd-refactor-producer (sonnet), doc-producer (sonnet), alignment-producer (sonnet), QA agents (haiku), various worker agents (sonnet)

## 2. What Went Well

**Design and architecture phases passed QA cleanly.** Both the design-producer and arch-producer artifacts were approved on the first QA pass, indicating clear scope definition from the PM phase.

**Correct identification of live yield code.** During implementation, worker-code correctly identified that `internal/memory/` files contain active runtime code for `projctl memory extract --yield` and left them untouched. This prevented breaking live functionality.

**Comprehensive verification tooling.** The TDD red phase produced both a shell script (`scripts/test-yield-cleanup.sh`) with 7 checks and a Go test (`internal/memory/yield_cleanup_test.go`), giving strong verification coverage. All 7 checks pass after cleanup.

**Parallelization during implementation.** Multiple workers ran concurrently on independent file sets (docs, tests, configs, project docs), completing the bulk of ~40 file modifications efficiently.

**Established patterns from Phase 1/2 migration reused.** The yield-to-messaging replacement mappings (DES-007) were clear and consistent, reducing ambiguity during implementation.

## 3. What Could Improve

### C-001: Skills with `context: fork` fail in teammate mode (Priority: High)

Skills that declare `context: fork` in their frontmatter attempt to spawn sub-agents when invoked by teammates. This caused the pm-interview-producer to go idle twice at the start of the project, wasting ~10 minutes. The retro-producer also failed to produce output due to the same issue. Filed as ISSUE-105.

**Impact:** Multiple teammates went idle without producing output, requiring manual intervention and re-spawning with explicit instructions.

### C-002: Haiku QA agents frequently hallucinate findings (Priority: High)

QA agents running on haiku fabricated findings in at least 2 instances:
- qa-pm-2: Referenced nonexistent REQ-005, quoted phrases not in the file
- qa-alignment-2: Referenced ARCH-006 and ARCH-010 (project only has ARCH-001 through ARCH-005)

Both required team-lead override, adding friction to the process.

**Impact:** Each hallucinated QA verdict required manual file verification and override, adding ~5-10 minutes per occurrence.

### C-003: Team lead went off-process during implementation (Priority: Medium)

Instead of following the step loop's tdd-green-producer action, the team lead spawned ad-hoc workers and manually debugged file contents, doing manual work instead of orchestrating teammates. User corrected: "you shouldn't be doing anything other than orchestrating."

**Impact:** Manual work mixed with orchestration created confusion about what was committed and what wasn't.

**Lesson:** The team lead role is to orchestrate, not to implement. When implementation is needed, spawn appropriate worker-* teammates with clear task assignments.

### C-004: worker-docs annotated instead of removing yield content (Priority: Medium)

The docs worker added "Historical Note" paragraphs to yield sections instead of removing them per REQ-001. The verification tests should have caught this but the worker didn't run them before reporting completion.

**Impact:** Required additional cleanup pass to fix ~4 files with incorrect annotations.

**Root cause:** Tests existed but were not enforced in the worker contract. The TDD cycle should require workers to run and report test results before marking tasks complete.

### C-005: State machine TDD loop doesn't know when to stop (Priority: Medium)

After all 15 tasks were completed and tests passed, `projctl step next` kept cycling back to tdd-red. There's no mechanism to signal "all tasks done" to the state machine. Required `projctl state transition --force` to advance to documentation phase.

**Impact:** Manual force-transition through multiple states (implementation → documentation → alignment), breaking the automated flow.

**Context:** The state machine appears to loop through TDD phases (red/green/refactor) indefinitely unless explicitly told to exit. Need either task completion detection or explicit exit signal.

### C-006: Discovery task (TASK-1) felt unnecessary (Priority: Low)

By the breakdown phase, the work scope was already well-understood from the PM, design, and architecture phases. Having a separate "discover all yield references" task added ceremony without value — the grep patterns were already documented in ARCH-001.

**Impact:** Minor — added task tracking overhead but didn't delay actual work.

**Team lead observation:** "Discovery task felt unnecessary — by breakdown phase we should already know the work."

## 4. Recommendations

### R-001: Fix skill `context: fork` in teammate mode (Priority: High)

**Action:** Update skill runner to detect when already running as a teammate and skip the fork, executing the skill in the current agent context.
**Rationale:** Would have prevented pm-interview-producer and retro-producer failures (C-001).
**Measure:** Skills invoked by teammates produce output without going idle.
**Issue:** ISSUE-105

### R-002: Increase QA model or add hallucination guards (Priority: High)

**Action:** Either run QA on sonnet instead of haiku, or add structural validation that QA findings reference IDs that actually exist in the artifact.
**Rationale:** Would have prevented 2 QA overrides needed due to hallucinated findings (C-002).
**Measure:** Zero QA findings referencing nonexistent IDs.
**Issue:** ISSUE-113

### R-003: Add task completion signal to state machine (Priority: Medium)

**Action:** Add a mechanism for the implementation phase to signal "all tasks complete" to the state machine, either via `projctl step complete --all-tasks-done` or by the state machine checking tasks.md for all-checked acceptance criteria.
**Rationale:** Would have prevented manual force-transition (C-005).
**Measure:** State machine auto-advances from implementation when all tasks pass.
**Issue:** ISSUE-114

### R-004: Require verification test run before worker completion (Priority: Medium)

**Action:** Add to tdd-green-producer contract: workers must run the verification script/tests and include pass/fail in their completion message.
**Rationale:** Would have caught worker-docs annotation problem (C-004) before team lead had to debug it.
**Measure:** All worker completion messages include test results.
**Issue:** ISSUE-115

### R-005: Eliminate discovery tasks for grep-based cleanup (Priority: Low)

**Action:** For cleanup issues where search patterns are defined in architecture, skip the discovery task and have implementation tasks run their own grep.
**Rationale:** Reduces unnecessary task overhead (C-006).
**Measure:** Cleanup projects have fewer tasks without losing coverage.

## 5. Open Questions

### Q-001: What should happen to live yield code in internal/memory/?

The `projctl memory extract --yield` command still uses yield infrastructure at runtime. This was correctly excluded from ISSUE-88 (doc cleanup only), but the live code may need its own migration or deprecation plan.

**Issue:** ISSUE-116

### Q-002: Should QA agents validate their own findings against artifact IDs?

When QA references "REQ-005" or "ARCH-006", the system could validate these IDs exist in the target artifact before accepting the finding. This would be a structural check, not a semantic one.

**Context:** Related to R-002 but specifically about whether validation should be in the QA skill itself or in the orchestrator that receives QA results.

## 6. Metrics

| Metric | Value |
|--------|-------|
| Total commits | 10 |
| Files modified | ~40 |
| Files deleted | 6 (4 docs + 2 yield.toml) |
| QA iterations (PM) | 2 (1 hallucinated) |
| QA iterations (Design) | 1 |
| QA iterations (Architecture) | 1 |
| QA iterations (Breakdown) | 1 |
| QA iterations (Implementation) | 1 |
| QA iterations (Documentation) | 1 |
| QA iterations (Alignment) | 2 (1 hallucinated) |
| QA overrides due to hallucination | 2 |
| Teammate failures (idle/stuck) | 3 (2x pm-interview-producer, 1x retro-producer) |
| Verification checks | 7/7 passing |
