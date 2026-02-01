# projctl Issues

Tracked issues for future work beyond the current task list.

---

## ISSUE-001: Implement deterministic orchestrator (projctl orchestrate)

**Priority:** Medium-term
**Status:** Open
**Created:** 2026-02-01

### Summary

Replace LLM-driven orchestration with a deterministic outer loop that invokes the LLM only for skill work. This eliminates context pollution as the root cause of orchestrator reliability issues.

### Problem

The current `/project` skill has the LLM manage the entire control loop:
1. Read state
2. Transition state
3. Generate territory map
4. Write context
5. **Do skill work** (only this needs LLM)
6. Read result
7. Check next action
8. Loop

Steps 1-4 and 6-8 are deterministic - they don't require reasoning. But because the LLM handles them, its context fills with code diffs, debug output, file contents, and previous iterations.

This causes control loop instructions to degrade in attention, leading to:
- Forgetting to call state transitions
- Claiming tasks complete when AC aren't met
- Stopping prematurely
- Going off-rails entirely

### Proposed Solution

Implement `projctl orchestrate` as a non-LLM program:

```bash
projctl orchestrate --dir . --task TASK-XXX
```

Internally:
```
while true:
    state = projctl state get --format json
    projctl state transition $next_phase
    projctl map --cached
    projctl context write --task $task --skill $skill

    # ONLY invoke LLM here
    claude --skill $skill --context context/$task-$skill.toml

    projctl context read --result
    next = projctl state next --format json

    if next.action == "stop":
        break
```

The LLM never sees orchestration machinery - it receives context, does work, writes result. Cannot forget control loop because it never knew it.

### Benefits

| Benefit | Details |
|---------|---------|
| Reliability | Deterministic code can't forget instructions |
| Efficiency | LLM only invoked for actual reasoning tasks |
| Debuggability | Shell script/Go code is inspectable |
| Testability | Control loop can be unit tested without LLM |

### Implementation Considerations

- May need `claude` CLI or API access for sub-agent invocation
- Could use Claude Code's Task tool initially, migrate to direct API later
- Session management for multi-task orchestration
- Error handling and recovery flows

### Relationship to Other Work

This is the "medium-term" solution. Short-term mitigations (TASK-060 through TASK-062) make the LLM-based orchestrator thinner, but this issue represents the architectural fix.

### Acceptance Criteria

- [ ] `projctl orchestrate` command implemented
- [ ] Deterministic loop handles all `[D]` steps from control loop
- [ ] LLM invoked only for skill dispatch
- [ ] State preserved entirely in files (state.toml, context/, result.toml)
- [ ] Works with existing skills without modification
- [ ] Error recovery via state.toml (can resume after crash)

**Traces to:** Phase 12 (Relentless Continuation)

---
