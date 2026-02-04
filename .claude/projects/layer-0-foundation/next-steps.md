# Layer 0: Foundation - Next Steps

**Project:** layer-0-foundation
**Status:** Complete

**Traces to:** ISSUE-045

---

## Immediate Follow-up

### 1. Layer 1 Implementation (ISSUE-TBD)

With Layer 0 complete, proceed to Layer 1: Leaf Commands as defined in docs/orchestration-system.md Section 13.4.

**Scope:**
- Commands that spawn single Claude CLI invocation
- `projctl skill invoke` command
- Context preparation and result collection
- No orchestration loops (single skill invocation)

### 2. Retro Recommendations (from R1-R4)

**R1: Task Completion Detection** (High)
- Enhance `projctl state next` to detect all tasks complete
- Create ISSUE for this enhancement

**R2: ONNX Session Caching** (Medium)
- Cache sessions in memory integration tests
- Could reduce test time from ~290s to ~60s

**R3: Skill Execution Validation** (Medium)
- Skills should validate output before yielding success
- Prevents silent failures in orchestration

**R4: Worktree Workflow Documentation** (Low)
- Document the parallel execution pattern used successfully here

---

## Suggested Issues to Create

| Priority | Issue | Description |
|----------|-------|-------------|
| High | Layer 1: Leaf Commands | Implement single-skill invocation layer |
| Medium | State machine task detection | Auto-detect task completion |
| Medium | ONNX test caching | Reduce memory test duration |
| Low | Worktree workflow docs | Document parallel execution pattern |

---

## Documentation to Reference

- `docs/orchestration-system.md` Section 13.4 - Layer 1 specification
- `docs/commands/memory.md` - Memory system architecture
- `docs/layer-0-implementation.md` - Layer 0 summary
- `.claude/projects/layer-0-foundation/` - Full project artifacts

---

## Learnings to Apply

1. **Git worktrees** work well for parallel task execution - use for Layer 1
2. **DI pattern** enables fast unit tests - continue using
3. **Fail-fast validation** catches errors early - apply to skill invocation
4. **Integration tests** with `testing.Short()` allow fast dev cycle
