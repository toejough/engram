# Design: Parallel Execution Improvements

This project updates documentation and skill guidance - no code changes required.

---

## DES-001: Merge-on-Complete Pattern Documentation Structure

The merge-on-complete pattern will be documented in the `/project` skill to guide orchestrators on when and how to merge parallel agent branches.

### Document Changes

**File:** `.claude/skills/project/SKILL-full.md`

**Location:** Add to "Parallel Execution" or "Implementation Phase" section

**Content Structure:**
1. When parallel execution is active (multiple Task tool invocations)
2. Detection: Monitor for agent completion (Task tool returns)
3. Immediate action sequence: rebase → merge → cleanup
4. Error handling guidance (conflicts, failures)
5. Serialization for simultaneous completions

### Rationale

The `/project` skill is the orchestrator that manages parallel execution. Adding this pattern to SKILL-full.md ensures future orchestration follows the merge-on-complete approach.

**Traces to:** REQ-001

---

## DES-002: Parallel Execution Section in Orchestration System

A new "Parallel Execution" section will be added to `docs/orchestration-system.md` providing architectural overview of parallel task execution.

### Document Changes

**File:** `docs/orchestration-system.md`

**Location:** New section (suggest after "Implementation" or as subsection of workflow phases)

**Content Structure:**
1. Overview: What parallel execution is and when it applies
2. Worktree workflow diagram (create → work → merge → cleanup)
3. Merge-on-complete pattern and benefits
4. Decision factors for parallel vs sequential
5. Known limitations
6. Examples (good and poor parallel task selection)

### Rationale

Centralizes architectural knowledge about parallel execution patterns for reference by skills and users.

**Traces to:** REQ-002

---

## DES-003: Skill-Level Operational Guidance

Add operational guidance to `/project` SKILL.md (the short version) for quick reference on parallel execution.

### Document Changes

**File:** `.claude/skills/project/SKILL.md`

**Location:** Add row to existing tables or brief section

**Content:** Brief pointer to parallel execution patterns, decision criteria, and reference to SKILL-full.md for details.

**Traces to:** REQ-002

---

## Out of Scope

- No code changes to projctl CLI
- No changes to worktree commands (already implemented)
- No file overlap detection (ISSUE-40 rejected)
