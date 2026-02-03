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

## ISSUE-002: TDD for documentation tasks

**Priority:** Medium-term
**Status:** Open
**Created:** 2026-02-01

### Summary

Documentation tasks should follow TDD discipline just like code tasks. Documentation is a feature - the same rigor should apply.

### Problem

Currently, documentation tasks (like updating SKILL.md files) are treated as "just edits" without the same TDD discipline applied to code:
- No failing test first
- No validation that the documentation achieves its purpose
- No structured review of whether the changes fit the overall document structure

This leads to:
- Documentation drift from actual behavior
- Inconsistent structure across documents
- Missing or redundant sections
- No regression detection when docs change

### Proposed Approach

Apply TDD to documentation:

1. **Red phase (what should docs say?):**
   - Write a test/validation that checks for expected content
   - Could be: grep patterns, section presence, word count limits
   - Could be: consistency checks against other docs
   - Test fails because content doesn't exist yet

2. **Green phase (do they say it?):**
   - Write the documentation to make tests pass
   - Minimal content to satisfy the requirements

3. **Refactor phase (does it fit?):**
   - Review structure in context of full document
   - Check for redundancy with other sections
   - Verify it doesn't break other "doc tests"

### Implementation Considerations

- `projctl docs validate` command to check documentation structure
- Schema for expected sections in skill files
- Linting rules for documentation consistency
- Integration with trace validate for doc-to-code traceability

### Questions to Resolve

- What tooling exists for "testing" markdown?
- How to define expected structure without over-constraining?
- How to handle doc tests that are inherently subjective (readability)?
- Should doc tests be code tests or a separate validation pass?

### Relationship to Other Work

- Affects all documentation-related tasks
- Should be incorporated into skill definitions (what makes a "valid" skill doc?)
- May need updates to task-audit to include doc validation

### Acceptance Criteria

- [ ] Document what "TDD for docs" means in practice
- [ ] Update relevant skills to include doc validation
- [ ] Create `projctl docs validate` or similar tooling
- [ ] Apply to existing skill SKILL.md files as proof of concept

**Traces to:** Process improvement

---

## ISSUE-003: End-to-end integration test for /project workflows

**Priority:** High
**Status:** Open
**Created:** 2026-02-01

### Summary

No automated test verifies that `/project new` (or adopt/align) can run through a complete workflow. Manual testing is the only validation that the orchestrator works end-to-end.

### Problem

Individual components are unit tested:
- State transitions work
- Preconditions work
- Context read/write works
- Skills exist

But nothing tests that these components work together in a real workflow:
- Does the orchestrator actually call state transitions in the right order?
- Does skill dispatch produce usable results?
- Does the control loop continue until completion?
- Does error recovery work when skills fail?

### Proposed Solution

Create an integration test that runs `/project new` through at least the PM interview phase:

```bash
# Setup
projctl state init --name test --dir $TMPDIR
projctl state transition --to pm-interview

# Simulate skill dispatch (mock or real)
# ... skill runs, produces result ...

# Verify orchestrator continues
projctl state next  # Should return continue with next phase
projctl state transition --to pm-complete

# Verify completion
projctl state get | grep "phase = \"pm-complete\""
```

Could also create a "dry-run" mode that validates the control loop without invoking actual skills.

### Acceptance Criteria

- [ ] Integration test script exists
- [ ] Test runs in CI (or can be run manually)
- [ ] Covers at least one complete phase transition
- [ ] Documents expected behavior for each step

**Traces to:** Phase 1 (CLI Completeness)

---

## ISSUE-004: State machine does not track completed tasks

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-01

### Summary

`projctl state next` always returns the current task from state.toml, even after that task is complete. The state machine has no concept of "this task is done, move to next task."

### Problem

After completing TASK-065:
```
$ projctl state next
{
  "action": "continue",
  "next_phase": "task-start",
  "next_task": "TASK-065"  // Same task we just finished
}
```

The orchestrator must manually track which tasks are complete and choose the next task. This is error-prone and defeats the purpose of deterministic orchestration.

### Observed Impact

- Orchestrator keeps suggesting the same task
- Manual intervention required to advance to next task
- No visibility into overall task progress

### Proposed Solutions

**Option A: Track completed tasks in state.toml**
```toml
[progress]
completed_tasks = ["TASK-063", "TASK-064", "TASK-065"]
```

Then `state next` skips completed tasks when suggesting next work.

**Option B: Read tasks.md for completion status**

Parse tasks.md for tasks with all AC marked `[x]` and skip those.

**Option C: Explicit task queue**
```toml
[queue]
pending = ["TASK-066", "TASK-067"]
in_progress = "TASK-065"
completed = ["TASK-063", "TASK-064"]
```

### Acceptance Criteria

- [ ] `state next` suggests a different task after `task-complete`
- [ ] Completed tasks are tracked persistently
- [ ] `state get` shows task completion progress

**Traces to:** Phase 12 (Relentless Continuation)

---

## ISSUE-005: Trace validation blocks transitions due to historical debt

**Priority:** Low
**Status:** Closed (2026-02-01)
**Created:** 2026-02-01

### Summary

`projctl trace validate` reports 64+ unlinked TASKs, blocking clean state transitions that require trace validation to pass.

### Problem

Many tasks were created without proper `**Traces to:**` fields, or the linked artifacts don't exist. This causes:

```
$ projctl state transition --to task-complete
Error: precondition failed: trace validation must pass
```

Workaround is `--force`, but this bypasses a useful check.

### Options

1. **Bulk fix:** Add `**Traces to:**` fields to all existing tasks
2. **Amnesty:** Mark historical tasks as exempt from validation
3. **Scope limit:** Only validate trace for current task, not entire repo
4. **Deferred:** Accept `--force` as standard practice until debt cleared

### Acceptance Criteria

- [x] Decide on approach
- [x] Either fix existing trace gaps or adjust validation scope
- [x] Clean transitions without `--force` for new work

### Resolution

Fixed by creating `docs/requirements.md` (REQ-001) and `docs/architecture.md` (ARCH-001 through ARCH-017). Updated all tasks to trace to ARCH-XXX instead of "Phase X". Added `// traces:` comments to test files. Trace validation now passes.

**Traces to:** Housekeeping

---

## ISSUE-006: Precondition checker hardcodes `docs/` subdirectory for artifact files

**Priority:** High
**Status:** Open
**Created:** 2026-02-01

### Summary

`DefaultChecker` in `cmd/projctl/checker.go` hardcodes paths like `docs/requirements.md` and `docs/design.md`, but projects may have these files at the project root instead.

### Problem

The `RequirementsExist`, `DesignExists`, and similar functions hardcode paths:

```go
func (c *DefaultChecker) RequirementsExist(dir string) bool {
    _, err := os.Stat(filepath.Join(dir, "docs", "requirements.md"))
    return err == nil
}
```

This fails when projects have `requirements.md` at the root:
```
projects/spacer-p0/requirements.md  ← actual location
projects/spacer-p0/docs/requirements.md  ← expected by checker
```

### Minimal Reproduction

```bash
# Setup
mkdir -p /tmp/projctl-bug-repro
cd /tmp/projctl-bug-repro
projctl state init --dir . --name "test-project"
projctl state transition --dir . --to pm-interview

# Create requirements at root (not docs/)
echo "# Requirements" > requirements.md
echo "## REQ-001: Test" >> requirements.md

# Attempt transition - FAILS
projctl state transition --dir . --to pm-complete
# Error: precondition failed: requirements.md must exist

# Fix by moving to docs/
mkdir -p docs
mv requirements.md docs/
projctl state transition --dir . --to pm-complete
# Transitioned to "pm-complete" (task: , subphase: )
```

### Proposed Solutions

**Option A: Make paths configurable via state.toml**
```toml
[paths]
requirements = "requirements.md"  # or "docs/requirements.md"
design = "design.md"
```

**Option B: Check multiple locations**
```go
func (c *DefaultChecker) RequirementsExist(dir string) bool {
    paths := []string{
        filepath.Join(dir, "requirements.md"),
        filepath.Join(dir, "docs", "requirements.md"),
    }
    for _, p := range paths {
        if _, err := os.Stat(p); err == nil {
            return true
        }
    }
    return false
}
```

**Option C: Document expected structure**

Add to docs that projects MUST have a `docs/` subdirectory. Update existing projects to match.

### Acceptance Criteria

- [ ] `projctl state transition --to pm-complete` works when `requirements.md` is at project root
- [ ] `projctl state transition --to design-complete` works when `design.md` is at project root
- [ ] Either paths are configurable, or multiple locations are checked, or structure is documented

**Traces to:** CLI robustness

---

## ISSUE-007: Visual verification required for CLI/TUI/GUI changes in TDD

**Priority:** High
**Status:** Open
**Created:** 2026-02-01

### Summary

When making changes that affect CLI output, TUI interfaces, or GUI components, TDD should include visual verification by default - validating not just structure but also behavior.

### Problem

Current TDD practice focuses on:
- Unit tests for logic
- Integration tests for data flow
- Structural assertions (element exists, text matches)

But for user-facing interfaces, this misses critical issues:
- CLI output that is technically correct but unreadable
- TUI layouts that render incorrectly despite correct data
- GUI components that exist in DOM but don't display properly
- Interactive elements that render but don't respond to input
- Styling/formatting issues invisible to structural tests

The CLAUDE.md lesson captures part of this:
> "UI testing verifies visual correctness, not just DOM existence"
> "Test behavior, not just presence"

But there's no systematic enforcement during TDD.

### Proposed Solution

Integrate visual verification into TDD phases for UI-affecting changes:

**Red phase:**
1. Write structural test (element/output exists)
2. Write behavioral test (interaction produces expected result)
3. **Add visual verification step** (screenshot comparison or manual check)

**Green phase:**
1. Make structural test pass
2. Make behavioral test pass
3. **Verify visual output matches expectation**

**Refactor phase:**
1. Clean up code
2. **Re-verify visual output unchanged**

### Implementation Considerations

For CLI:
- Capture stdout/stderr and compare against expected output
- Use ANSI-aware diffing for colored output
- `projctl screenshot` could be extended for terminal screenshots

For TUI:
- Headless terminal rendering with screenshot comparison
- SSIM-based regression detection (already exists in projctl)

For GUI:
- Chrome DevTools MCP for browser-based UI
- Screenshot comparison with `projctl screenshot diff`
- Behavioral verification via click/interaction tests

### Integration with Skills

Update TDD skills to prompt for visual verification:
- `/tdd-red`: "For UI changes, include visual acceptance criteria"
- `/tdd-green`: "Verify visual output matches design"
- `/task-audit`: "Include visual verification evidence for UI tasks"

### Acceptance Criteria

- [ ] Document visual verification requirements in TDD skill docs
- [ ] Add `ui` flag or marker to tasks requiring visual verification
- [ ] `/tdd-green` prompts for visual check when `ui` flag present
- [ ] `/task-audit` fails if UI task lacks visual evidence
- [ ] CLAUDE.md lesson updated to make this standard practice

**Traces to:** REQ-001

---

## ISSUE-008: Layer -1 - Unify skills to new orchestration patterns

**Priority:** High
**Status:** Open
**Created:** 2026-02-02

### Summary

Update all skills to the unified orchestration patterns (producer/QA pairs, yield protocol) before migrating orchestration from `/project` skill to projctl. This is Layer -1 of the implementation plan in `docs/orchestration-system.md`.

### Problem

The current skills are inconsistent:
- Some have interview/infer/audit variants, others don't
- No standard yield protocol output format
- No standard context input format
- Orchestration logic mixed with agent behavior
- Skills don't follow the producer/QA pair pattern

This blocks projctl from taking over orchestration (Layers 0-7) because it needs predictable skill interfaces.

### Desired State

All skills follow the unified pattern:
1. **Producer/QA pairs** for each phase (pm, design, arch, breakdown, doc, tdd-red, tdd-green, tdd-refactor, alignment, retro, summary)
2. **Yield protocol TOML output** (type, payload, optional context)
3. **Standard context input** from orchestrator
4. **No orchestration logic** - skills just do their work and yield

### Scope

**Phase Agent Skills** (producer + QA pairs):
- `pm-producer` / `pm-qa`
- `design-producer` / `design-qa`
- `arch-producer` / `arch-qa`
- `breakdown-producer` / `breakdown-qa`
- `doc-producer` / `doc-qa`

**TDD Agent Skills** (nested producer + QA pairs):
- `tdd-red-producer` / `tdd-red-qa`
- `tdd-green-producer` / `tdd-green-qa`
- `tdd-refactor-producer` / `tdd-refactor-qa`
- `tdd-qa` (overall TDD quality gate)

**Support Agent Skills**:
- `alignment-producer` / `alignment-qa`
- `retro-producer` / `retro-qa`
- `summary-producer` / `summary-qa`
- `intake-evaluator`
- `next-steps`
- `commit` (already exists, verify compatibility)

### Acceptance Criteria

- [ ] All producer skills output yield protocol TOML
- [ ] All producer skills accept context from orchestrator
- [ ] All QA skills output yield protocol TOML (approved | improvement-request | escalate)
- [ ] Existing `/project` skill can orchestrate the new skills
- [ ] Skills contain no orchestration logic (state transitions, next phase selection)
- [ ] Each skill has clear guidelines in its SKILL.md

### Relationship to Other Work

- Prerequisite for ISSUE-001 (deterministic orchestrator)
- Implements Layer -1 from `docs/orchestration-system.md`
- Must complete before Layers 0-7 can begin

**Traces to:** docs/orchestration-system.md Section 12 (Implementation Plan)

---
