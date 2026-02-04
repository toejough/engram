# Architecture: Parallel Execution Improvements

This project is documentation-only. No code architecture decisions required.

---

## ARCH-001: File Modification Order

Documentation updates should be made in this order to maintain consistency:

1. **`docs/orchestration-system.md`** - Add "Parallel Execution" section with architectural overview
2. **`.claude/skills/project/SKILL-full.md`** - Add merge-on-complete operational details
3. **`.claude/skills/project/SKILL.md`** - Add brief reference/pointer

### Rationale

Start with the architectural reference (orchestration-system.md), then add operational details to the skill, then update the quick reference. This ensures the authoritative source exists before references to it.

**Traces to:** DES-001, DES-002, DES-003

---

## ARCH-002: No Code Changes

This project requires zero code changes to projctl. The worktree infrastructure already exists (`projctl worktree create/merge/cleanup`). This project only documents how to use it effectively.

**Traces to:** REQ-001, REQ-002

---

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Implementation type | Documentation only | Worktree commands exist; need usage guidance |
| Primary location | orchestration-system.md | Authoritative architectural reference |
| Operational guidance | SKILL-full.md | Where orchestrators look for detailed procedures |
