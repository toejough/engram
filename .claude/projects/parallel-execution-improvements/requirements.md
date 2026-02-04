# Requirements: Parallel Execution Improvements

This document captures requirements for improving parallel task execution in the projctl orchestration system.

---

## REQ-001: Merge-on-Complete Pattern for Parallel Execution

As a project orchestrator, I want parallel agent branches to be merged immediately when each agent completes, so that later-completing agents benefit from already-merged work and conflict complexity is reduced.

### Problem Statement

During parallel worktree execution, all agent branches were merged at the end after all agents completed. This caused:
- Increased conflict complexity (later branches couldn't rebase onto already-merged work)
- Duplicate method implementations (agents independently implemented similar functionality)
- More manual conflict resolution needed
- Later agents working without visibility into earlier completions

### Acceptance Criteria

- [ ] The `/project` skill detects when an individual parallel agent completes its task
- [ ] On completion, the agent's branch is immediately:
  - Rebased onto the target branch (main)
  - Fast-forward merged if clean (clean = no merge conflicts, no uncommitted changes, no divergent history that prevents fast-forward)
  - Conflict resolution handling:
    - Automated: Attempt rebase with conflict markers generated
    - Manual intervention required: If rebase fails, orchestration pauses and prompts user to resolve conflicts manually
    - Expected outcome: Branch is either successfully merged or user is given clear instructions on conflict resolution steps
  - Deleted after successful merge
- [ ] Later-completing agents benefit from earlier merges (their rebases include prior work)
- [ ] Worktree cleanup happens immediately after merge:
  - If cleanup fails (locked files, permissions errors): Log error, continue with remaining agents, report cleanup failures at end
  - Cleanup failure does NOT block subsequent merges
- [ ] The merge-on-complete pattern is documented in the "Parallel Execution" section of `docs/orchestration-system.md` (create new section if doesn't exist)
- [ ] Edge case handling:
  - Agent failure mid-execution: Branch is NOT merged, worktree is cleaned up, failure logged, orchestration continues with remaining agents
  - Merge conflict resolution failure: Orchestration stops, user intervention required, clear error message with resolution steps provided
  - Simultaneous agent completions: Merges are serialized by completion timestamp (earliest completes first)

### Priority

P1

### Traces to

ISSUE-039

---

## REQ-002: Parallel Execution Best Practices Documentation

As a developer using the orchestration system, I want comprehensive documentation on parallel execution patterns, so that I understand when to parallelize, how the worktree workflow works, and what limitations exist.

### Problem Statement

The parallel-worktree-strategy project proved the worktree-based parallel execution works, but lessons learned were not captured in documentation:
- When to use parallel execution vs sequential
- How the worktree workflow operates
- Merge-on-complete pattern and rationale
- Known limitations and workarounds
- File overlap considerations (accepted as normal rebase/conflict workflow)

### Acceptance Criteria

- [ ] Documentation exists covering parallel execution patterns in:
  - `docs/orchestration-system.md` (architectural overview in new "Parallel Execution" section)
  - `.claude/skills/project/SKILL.md` (skill-level operational guidance for when/how to use parallel execution)
- [ ] Documentation includes decision factors for parallel vs sequential execution:
  - Task independence (can run without coordination)
  - File overlap risk (low/no shared files preferred)
  - Coordination needs (tasks requiring shared state should be sequential)
  - Task granularity (atomic, well-bounded work)
- [ ] Worktree workflow documented with diagram/steps:
  - Create worktree from target branch
  - Agent works in isolated directory
  - On completion: merge immediately (not batched)
  - Cleanup worktree after merge
- [ ] Merge-on-complete pattern explanation and benefits documented:
  - Reduces conflict window
  - Later agents benefit from earlier completions
  - Simplifies conflict resolution
- [ ] Known limitations explicitly documented (from ISSUE-041):
  - Agents cannot coordinate during execution (no shared state)
  - File overlap causes merge conflicts (handled via rebase/manual resolution)
  - No pre-flight file overlap detection (accepted tradeoff)
  - Simultaneous completions require serialization
- [ ] Minimum 3 examples of good parallel task selection:
  - Independent feature additions to different modules
  - Test coverage for separate components
  - Documentation updates for different subsystems
  - (Each with 2-3 sentence scenario describing why parallel works)
- [ ] Minimum 2 examples of poor parallel task selection:
  - Refactoring same function with different goals
  - Sequential pipeline stages (design → implement → test same feature)
  - (Each with 2-3 sentence scenario describing why conflicts arise)

### Priority

P2

### Traces to

ISSUE-041

---

## Out of Scope

**ISSUE-040: File Overlap Detection** - User rejected this requirement. File overlap between parallel tasks is handled through normal rebasing and conflict resolution processes. Pre-flight analysis is not needed because we cannot always predict what else needs to change during implementation.
