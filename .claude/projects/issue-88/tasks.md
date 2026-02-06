# Tasks: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Project:** Clean up remaining yield references in docs
**Date:** 2026-02-06

## Overview

This task breakdown decomposes the yield reference cleanup work into focused implementation tasks. Tasks are organized by file group and can run in parallel when they touch different files.

---

## Task List

### TASK-1: Discover All Yield References

**Description:** Run comprehensive grep searches to identify all files containing yield references and generate a prioritized work list.

**Files:**
- (no files modified - grep output only)

**Acceptance Criteria:**
- [ ] Case-insensitive grep for "yield" completed across all markdown files
- [ ] Secondary pattern greps completed for `yield_path`, `producer_yield_path`, `yield.type`, `[yield]`
- [ ] File lists generated for: SKILL.md files, docs/ files, .claude/projects/ files, source code files
- [ ] Results filtered to exclude ISSUE-88 project files and historical retrospectives
- [ ] Prioritized work list created showing file counts per category

**Traces to:** ARCH-001, DES-001, REQ-001, REQ-002, REQ-003, ISSUE-88

---

### TASK-2: Clean Up Skill Documentation (Batch 1)

**Description:** Update first batch of SKILL.md files to replace yield protocol sections with messaging-based communication protocol. Process skills in alphabetical order, first half of alphabet (A-M).

**Files:**
- `skills/*/SKILL.md` (files A-M based on TASK-1 results)

**Acceptance Criteria:**
- [ ] All "Yield Protocol" sections replaced with "Communication Protocol" sections
- [ ] All yield TOML examples replaced with `SendMessage` tool examples
- [ ] All `yield.type` references replaced with appropriate messaging patterns per DES-002
- [ ] All workflow steps mentioning "write yield file" replaced with "send message to team-lead"
- [ ] No broken internal links to removed yield sections
- [ ] Changes follow DES-007 replacement mappings

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-002, DES-007, REQ-002, REQ-004, ISSUE-88

---

### TASK-3: Clean Up Skill Documentation (Batch 2)

**Description:** Update second batch of SKILL.md files to replace yield protocol sections with messaging-based communication protocol. Process skills in alphabetical order, second half of alphabet (N-Z).

**Files:**
- `skills/*/SKILL.md` (files N-Z based on TASK-1 results)

**Acceptance Criteria:**
- [ ] All "Yield Protocol" sections replaced with "Communication Protocol" sections
- [ ] All yield TOML examples replaced with `SendMessage` tool examples
- [ ] All `yield.type` references replaced with appropriate messaging patterns per DES-002
- [ ] All workflow steps mentioning "write yield file" replaced with "send message to team-lead"
- [ ] No broken internal links to removed yield sections
- [ ] Changes follow DES-007 replacement mappings

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-002, DES-007, REQ-002, REQ-004, ISSUE-88

---

### TASK-4: Clean Up Skill Test Scripts

**Description:** Update SKILL_test.sh files to remove yield-related test assertions and update to test messaging-based communication.

**Files:**
- `skills/*/SKILL_test.sh` (based on TASK-1 results)

**Acceptance Criteria:**
- [ ] All yield file existence checks removed
- [ ] All yield TOML parsing tests removed or updated to test message content
- [ ] All `yield.type` assertions replaced with message content assertions
- [ ] All test setup removing yield paths updated or removed
- [ ] Tests still pass after updates (run `mage check` or equivalent)

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-002, REQ-002, ISSUE-88

---

### TASK-5: Clean Up Architecture Documentation

**Description:** Update core architecture documentation files to replace yield-based communication model with team messaging model.

**Files:**
- `docs/architecture.md`
- `docs/design.md`
- `docs/orchestration-system.md`
- Any other architecture docs from TASK-1 results

**Acceptance Criteria:**
- [ ] All "yield schema" sections replaced with "message payload schema" sections
- [ ] All flow diagrams updated to show `SendMessage` instead of file I/O
- [ ] All state machine descriptions updated to remove yield file reading steps
- [ ] All "orchestrator reads yield" references replaced with "team lead receives message"
- [ ] All protocol specifications updated per DES-003
- [ ] No broken internal links to removed yield sections

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-003, REQ-003, REQ-004, ISSUE-88

---

### TASK-6: Clean Up Project Documentation (Active Projects)

**Description:** Update documentation in .claude/projects/ for any active or in-progress projects that reference yield infrastructure.

**Files:**
- `.claude/projects/*/requirements.md` (active projects with yield references from TASK-1)
- `.claude/projects/*/design.md` (active projects with yield references from TASK-1)
- `.claude/projects/*/architecture.md` (active projects with yield references from TASK-1)
- Exclude `.claude/projects/issue-88/` (this project)

**Acceptance Criteria:**
- [ ] All yield references replaced with messaging equivalents per DES-002/DES-003
- [ ] Historical context preserved where appropriate per DES-005
- [ ] No broken traceability links
- [ ] No broken internal documentation links

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-003, DES-005, REQ-003, REQ-004, ISSUE-88

---

### TASK-7: Clean Up Project Documentation (Completed Projects)

**Description:** Review and update documentation in .claude/projects/ for completed projects, adding deprecation notices where yield references are historical.

**Files:**
- `.claude/projects/*/retro.md` (completed projects with yield references from TASK-1)
- `.claude/projects/*/tasks.md` (completed projects with yield references from TASK-1)
- `.claude/projects/*/README.md` (if exists and has yield references)

**Acceptance Criteria:**
- [ ] Historical yield references marked with deprecation notices per DES-004
- [ ] Purely historical content preserved (retrospectives documenting yield migration)
- [ ] Non-historical yield workflow references updated to messaging equivalents
- [ ] README files updated with note about historical yield directories

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-004, DES-005, REQ-003, ISSUE-88

---

### TASK-8: Clean Up Root Documentation

**Description:** Update root-level markdown files (README.md, CONTRIBUTING.md, etc.) that may reference yield infrastructure.

**Files:**
- `README.md` (if has yield references)
- `CONTRIBUTING.md` (if has yield references)
- Any other root-level .md files from TASK-1 results

**Acceptance Criteria:**
- [ ] All user-facing documentation updated to show current workflows
- [ ] No references to `projctl yield` commands
- [ ] Examples updated to show messaging-based patterns
- [ ] Getting started guides reference current skill structure

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-002, REQ-003, REQ-004, ISSUE-88

---

### TASK-9: Clean Up docs/ Directory Files

**Description:** Update all remaining markdown files in docs/ directory that reference yield infrastructure.

**Files:**
- `docs/**/*.md` (all remaining files with yield references from TASK-1, excluding those in TASK-5)

**Acceptance Criteria:**
- [ ] All yield workflow descriptions updated to messaging workflows
- [ ] All code examples updated to show `SendMessage` usage
- [ ] All references to yield files and directories removed or marked deprecated
- [ ] User guides reflect current system state

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-002, DES-003, REQ-003, REQ-004, ISSUE-88

---

### TASK-10: Clean Up Source Code Comments

**Description:** Review and update source code comments in Go files that reference yield infrastructure, removing outdated explanations.

**Files:**
- `internal/**/*.go` (files with yield references from TASK-1)
- `cmd/**/*.go` (files with yield references from TASK-1)

**Acceptance Criteria:**
- [ ] All comments explaining yield protocol removed or updated
- [ ] Historical decision comments kept if marked as "deprecated" per DES-005
- [ ] No comments referencing yield file paths or yield.type values
- [ ] Code builds successfully after comment updates

**Depends on:** TASK-1

**Traces to:** ARCH-002, DES-005, REQ-005, ISSUE-88

---

### TASK-11: Delete Active Yield Files

**Description:** Locate and delete root-level yield TOML files that are active or stale (not historical artifacts).

**Files:**
- `yield.toml` (if exists)
- `.claude/yield.toml` (if exists)

**Acceptance Criteria:**
- [ ] Root-level yield.toml deleted if present
- [ ] .claude/yield.toml deleted if present
- [ ] Verification confirms files no longer exist
- [ ] Historical yield directories in .claude/projects/ISSUE-*/ preserved
- [ ] Git commit documents which files were deleted

**Depends on:** TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9 (wait until docs updated before deleting files)

**Traces to:** ARCH-004, DES-004, REQ-001, ISSUE-88

---

### TASK-12: Update Context Configuration Files

**Description:** Update any .toml configuration files that contain yield_path or producer_yield_path fields, removing those fields.

**Files:**
- Any `.toml` files in skills/, .claude/, or project root with `yield_path` fields (from TASK-1 grep)

**Acceptance Criteria:**
- [ ] All `output.yield_path` lines removed from context configurations
- [ ] All `producer_yield_path` lines removed from QA context configurations
- [ ] Empty `[output]` sections removed if no other fields remain
- [ ] TOML syntax remains valid after edits (verify by parsing)
- [ ] No broken references to removed configuration fields

**Depends on:** TASK-1

**Traces to:** ARCH-004, DES-004, REQ-001, ISSUE-88

---

### TASK-13: Verify Complete Yield Removal (Grep Validation)

**Description:** Run comprehensive grep validation to verify all yield references removed except those in explicit allowlist.

**Files:**
- (no files modified - verification only)

**Acceptance Criteria:**
- [ ] Case-insensitive grep for "yield" in *.md files returns only allowlisted matches
- [ ] Allowlisted matches confirmed: ISSUE-88 requirements.md, design.md, architecture.md, historical retrospectives
- [ ] Case-insensitive grep for "yield" in *.go files returns zero matches (or only test fixtures)
- [ ] Grep for `yield_path`, `producer_yield_path`, `yield.type` returns zero matches outside ISSUE-88
- [ ] Context review (-C 3) of any unexpected matches shows legitimate usage
- [ ] File list generated showing all remaining yield references with justification

**Depends on:** TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9, TASK-10, TASK-11, TASK-12

**Traces to:** ARCH-003, DES-006, REQ-001, ISSUE-88

---

### TASK-14: Verify Build and Tests Pass

**Description:** Run full project build and test suite to ensure no runtime dependencies on yield infrastructure remain.

**Files:**
- (no files modified - verification only)

**Acceptance Criteria:**
- [ ] `mage check` completes successfully (exit code 0)
- [ ] All unit tests pass
- [ ] All integration tests pass (if applicable)
- [ ] All builds succeed for CLI and other binaries
- [ ] No runtime errors about missing yield files or functions
- [ ] No import errors for removed yield packages
- [ ] Test output reviewed for any yield-related warnings

**Depends on:** TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9, TASK-10, TASK-11, TASK-12

**Traces to:** ARCH-005, DES-006, REQ-005, ISSUE-88

---

### TASK-15: Verify Documentation Links

**Description:** Spot-check documentation for broken links to removed yield sections and verify all internal references are valid.

**Files:**
- (no files modified - verification only)

**Acceptance Criteria:**
- [ ] Sample of SKILL.md files checked for broken links
- [ ] Architecture documentation checked for broken links
- [ ] Links to messaging documentation sections verified as valid
- [ ] No references to removed yield protocol sections found
- [ ] Cross-references between updated documentation files work correctly

**Depends on:** TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9

**Traces to:** ARCH-003, DES-006, REQ-001, ISSUE-88

---

## Dependency Graph

```
TASK-1 (Discover)
├─→ TASK-2 (Skills A-M)
├─→ TASK-3 (Skills N-Z)
├─→ TASK-4 (Skill tests)
├─→ TASK-5 (Architecture docs)
├─→ TASK-6 (Active project docs)
├─→ TASK-7 (Completed project docs)
├─→ TASK-8 (Root docs)
├─→ TASK-9 (docs/ directory)
├─→ TASK-10 (Source code comments)
└─→ TASK-12 (Context configs)

TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9, TASK-10, TASK-12
└─→ TASK-11 (Delete yield files)

TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9
└─→ TASK-15 (Verify links)

TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9, TASK-10, TASK-11, TASK-12
├─→ TASK-13 (Verify grep)
└─→ TASK-14 (Verify build/tests)
```

## Execution Notes

**Parallelization Opportunities:**
- TASK-2 and TASK-3 can run in parallel (different skill files)
- TASK-4, TASK-5, TASK-6, TASK-7, TASK-8, TASK-9, TASK-10, TASK-12 can all run in parallel after TASK-1 (different file sets)
- TASK-13 and TASK-14 can run in parallel (both verification, no file conflicts)

**Sequential Dependencies:**
- TASK-1 must complete first (generates work lists for all other tasks)
- TASK-11 must wait for doc tasks to complete (avoid confusion while docs reference yields)
- TASK-15 must wait for doc editing tasks (verifies links in updated docs)
- TASK-13 and TASK-14 must wait for all editing and deletion tasks

**Commit Strategy:**
- Commit after each task or after each batch of parallel tasks
- Use descriptive commit messages referencing TASK-N IDs
- Final commit should reference TASK-13, TASK-14, TASK-15 verification results

**Estimated Effort:**
- TASK-1: ~10-15 minutes (grep execution and result aggregation)
- TASK-2, TASK-3: ~30-45 minutes each (assuming ~20-25 skill files each)
- TASK-4: ~15-20 minutes (fewer test files, simpler updates)
- TASK-5: ~20-30 minutes (complex architecture docs)
- TASK-6, TASK-7: ~20-30 minutes each (variable project file count)
- TASK-8: ~10 minutes (few root-level files)
- TASK-9: ~15-20 minutes (docs/ directory)
- TASK-10: ~10-15 minutes (code comments)
- TASK-11: ~5 minutes (simple deletion)
- TASK-12: ~10 minutes (config file updates)
- TASK-13: ~5-10 minutes (grep verification)
- TASK-14: ~5-10 minutes (build/test run, longer if failures)
- TASK-15: ~10-15 minutes (spot-checking links)

**Total estimated effort:** ~3-4 hours with parallelization, ~5-6 hours sequential

**Traces to:** ISSUE-88
