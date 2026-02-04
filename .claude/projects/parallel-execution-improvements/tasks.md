# Tasks: Parallel Execution Improvements

---

## TASK-001: Add Parallel Execution section to orchestration-system.md

Add a new "Parallel Execution" section to `docs/orchestration-system.md` covering:

1. Overview of parallel task execution
2. Worktree workflow (create → work → merge → cleanup)
3. Merge-on-complete pattern and benefits
4. Decision factors for parallel vs sequential execution
5. Known limitations (no agent coordination, file overlap causes conflicts)
6. Examples of good/poor parallel task selection

### Acceptance Criteria

- [ ] New "Parallel Execution" section exists in orchestration-system.md
- [ ] Worktree workflow documented with steps
- [ ] Merge-on-complete pattern explained with rationale
- [ ] Decision factors listed (independence, file overlap risk, coordination needs, granularity)
- [ ] At least 3 good parallel examples, 2 poor parallel examples
- [ ] Known limitations documented

**Traces to:** ARCH-001, DES-002, REQ-002

---

## TASK-002: Add merge-on-complete guidance to SKILL-full.md

Update `.claude/skills/project/SKILL-full.md` with operational guidance for the merge-on-complete pattern:

1. When to use parallel execution (multiple independent tasks)
2. How to detect agent completion
3. Immediate merge workflow (rebase → merge → cleanup)
4. Error handling (conflicts pause, cleanup failures logged)
5. Serialization for simultaneous completions

### Acceptance Criteria

- [ ] SKILL-full.md has parallel execution operational guidance
- [ ] Merge-on-complete workflow steps documented
- [ ] Error handling behavior specified (conflicts, cleanup failures)
- [ ] References orchestration-system.md for architectural details

**Traces to:** ARCH-001, DES-001, REQ-001

---

## TASK-003: Add parallel execution reference to SKILL.md

Update `.claude/skills/project/SKILL.md` with brief parallel execution reference:

1. Add row to relevant table or brief section
2. Point to SKILL-full.md for details
3. Mention key decision: merge-on-complete, not batch-at-end

### Acceptance Criteria

- [ ] SKILL.md mentions parallel execution
- [ ] References SKILL-full.md for operational details
- [ ] Fits existing document structure/style

**Traces to:** ARCH-001, DES-003, REQ-002

---

## Task Dependencies

```
TASK-001 (orchestration-system.md)
    ↓
TASK-002 (SKILL-full.md) - can reference TASK-001
    ↓
TASK-003 (SKILL.md) - can reference both
```

All tasks can be done sequentially or TASK-001 can be parallelized with TASK-002/003 if references are added after.
