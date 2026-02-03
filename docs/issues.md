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
**Status:** Closed
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


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
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
**Status:** Closed
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

### Blocked By

L-1 skills are complete but depend on projctl commands that are missing or broken:

| Issue | Blocker |
|-------|---------|
| ISSUE-009 | State machine missing phases skills need to transition to |
| ISSUE-010 | `state init` doesn't accept `--mode` for workflow type |
| ISSUE-013 | context-explorer expects `projctl territory` but command is `projctl map` |
| ISSUE-016 | Missing `projctl issue create/update` commands |
| ISSUE-017 | Missing `projctl state set` command |
| ISSUE-018 | Missing `projctl yield validate` command |

**Traces to:** docs/orchestration-system.md Section 12 (Implementation Plan)

---


### Comment

Layer -1 complete. 37 skills unified with yield protocol. See .claude/projects/layer-minus-1-skill-unification/ for artifacts.
## ISSUE-009: State machine transitions don't match orchestration doc

**Priority:** High
**Status:** Closed
**Created:** 2026-02-03
**Blocks:** Layer -1 (ISSUE-008)
**Resolution:** Updated `internal/state/transitions.go` with correct workflow phases

### Summary

The `LegalTransitions` map in `internal/state/transitions.go` has significant discrepancies with the workflows defined in `docs/orchestration-system.md` Section 7 and the `/project` skill documentation.

### Problem

#### 1. Adopt Workflow Order is WRONG (top-down instead of bottom-up)

**Current (`transitions.go`):**
```
adopt-analyze → adopt-infer-pm → adopt-infer-design → adopt-infer-arch
```

**Should be (per orchestration doc 7.3 - infers from code upward):**
```
adopt-explore → adopt-infer-tests → adopt-infer-arch → adopt-infer-design → adopt-infer-reqs
```

#### 2. Missing Phases

| Phase | Purpose |
|-------|---------|
| `documentation` | After implementation, integrates project artifacts into repo docs |
| `retro` | Main flow ending - retrospective |
| `summary` | Main flow ending - summarize accomplishments |
| `issue-update` | Main flow ending - update/close linked issues |
| `next-steps` | Main flow ending - suggest follow-on work |
| `adopt-explore` | First phase of adopt - analyze existing code |
| `adopt-infer-tests` | Infer test coverage from existing tests |
| `adopt-documentation` | Documentation phase in adopt workflow |

#### 3. Obsolete Phases (should be removed)

| Phase | Reason |
|-------|--------|
| `audit` / `audit-fix` / `audit-complete` | QA runs in PAIR LOOPs, not as separate audit phase |
| `adopt-map-tests` | test-mapper is obsolete (no TEST-NNN IDs per Layer -1) |
| `adopt-generate` | Not in orchestration doc |
| `integrate-*` phases | Integration is part of Documentation phase, not separate workflow |

#### 4. Missing `task` Workflow

No transitions for single-task workflow:
```
init → task-implementation → task-documentation → alignment → retro → ...
```

#### 5. Alignment Timing Wrong

**Current:** alignment-check runs multiple times during workflow (after design, architect, planning)
**Should be:** Alignment runs once in main flow ending after Documentation

### Acceptance Criteria

- [ ] Adopt workflow transitions are bottom-up (tests → arch → design → reqs)
- [ ] Main flow ending phases added (documentation → alignment → retro → summary → issue-update → next-steps)
- [ ] Obsolete phases removed (audit-*, adopt-map-tests, adopt-generate, integrate-*)
- [ ] Task workflow transitions added
- [ ] Alignment phase moved to main flow ending only
- [ ] transitions_test.go updated to match

**Traces to:** docs/orchestration-system.md Section 7

---


### Comment

Fixed in commit 4087e0f
## ISSUE-010: State struct missing workflow type and pair loop tracking

**Priority:** High
**Status:** Closed
**Created:** 2026-02-03
**Blocks:** Layer -1 (ISSUE-008)
**Partial Resolution (2026-02-03):** Added `Workflow` and `Issue` fields to Project struct, `InitOpts` for Init(), `SetOpts` for Set()

**Remaining AC items:**
- [ ] Add `Pairs` map to track per-phase/per-task pair loop state
- [ ] Add `Yield` struct for pending yield tracking

### Summary

The `State` struct in `internal/state/state.go` lacks fields required by the orchestration doc for workflow tracking and PAIR LOOP state.

### Problem

**Current State struct fields:**
```go
type State struct {
    Project   Project           // name, created, phase
    Progress  Progress          // current_task, current_subphase, tasks_complete/total/escalated
    Conflicts Conflicts         // open, blocking_tasks
    Meta      Meta              // corrections_since_last_audit, last_meta_audit
    History   []PhaseTransition // timestamp, phase
    Error     *ErrorInfo        // last error details
}
```

**Missing fields per orchestration doc Section 4.1:**

1. **Workflow type** - Which flow we're executing
```toml
[project]
workflow = "new"  # new | adopt | align | task
```

2. **Pair loop states** - Per-phase iteration tracking for PAIR LOOPs
```toml
[pairs.pm]
iteration = 2
max_iterations = 3
producer_complete = true
qa_verdict = "needs_improvement"
improvement_request = "REQ-003 acceptance criteria are not measurable"

[pairs.task-007]
iteration = 1
producer_complete = true
qa_verdict = "approved"
```

3. **Issue tracking** - Link to issues.md
```toml
[project]
issue = "ISSUE-042"
```

4. **Yield state** - Track pending yields
```toml
[yield]
pending = true
type = "need-user-input"
agent = "pm"
context_file = ".claude/agents/pm-state.toml"
```

### Acceptance Criteria

- [ ] Add `Workflow` field to Project struct
- [ ] Add `Pairs` map to track per-phase/per-task pair loop state
- [ ] Add `Issue` field to Project struct
- [ ] Add `Yield` struct for pending yield tracking
- [ ] Update Init() to accept workflow parameter
- [ ] Update state.toml encoding/decoding
- [ ] Update `projctl state get` output to show new fields

**Traces to:** docs/orchestration-system.md Section 4.1

---


### Comment

Added Pairs map and Yield struct to State, CLI commands for state pair set/clear and state yield set/clear
## ISSUE-011: Missing `projctl id next` command for ID generation

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03

### Summary

The orchestration doc Section 10.4 specifies `projctl id next --type <TYPE>` for generating sequential IDs, but this command doesn't exist.

### Problem

Skills need to generate traceable IDs (REQ-N, DES-N, ARCH-N, TASK-N) when creating artifacts. Currently there's no deterministic way to get the next ID.

**Expected per orchestration doc:**
```bash
projctl id next --type REQ        # Returns REQ-004
projctl id next --type DES        # Returns DES-007
projctl id next --type ARCH       # Returns ARCH-012
projctl id next --type TASK       # Returns TASK-089
```

### Implementation

1. Scan artifact files for existing IDs of the requested type
2. Parse highest number
3. Return next sequential ID
4. Optionally reserve/write the ID to prevent race conditions

### Acceptance Criteria

- [ ] `projctl id next --type REQ` returns next REQ-N
- [ ] `projctl id next --type DES` returns next DES-N
- [ ] `projctl id next --type ARCH` returns next ARCH-N
- [ ] `projctl id next --type TASK` returns next TASK-N
- [ ] Scans correct artifact files for each type
- [ ] Handles empty/missing files gracefully

**Traces to:** docs/orchestration-system.md Section 10.4

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-012: Missing `projctl trace show` command for visualization

**Priority:** Low
**Status:** Closed
**Created:** 2026-02-03

### Summary

The orchestration doc Section 10.5 specifies `projctl trace show` to visualize the traceability chain, but only `validate` and `repair` exist.

### Expected

```bash
projctl trace show
```

Output could be:
- ASCII tree showing ISSUE → REQ → DES → ARCH → TASK → test chain
- Mermaid diagram for rendering
- JSON/TOML for tooling

### Acceptance Criteria

- [ ] `projctl trace show` command exists
- [ ] Outputs human-readable traceability visualization
- [ ] Shows orphan and unlinked IDs clearly

**Traces to:** docs/orchestration-system.md Section 10.5

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-013: Rename `projctl map` to `projctl territory` per orchestration doc

**Priority:** High
**Status:** Closed
**Created:** 2026-02-03
**Blocks:** Layer -1 (ISSUE-008)
**Partial Resolution (2026-02-03):** Renamed `cmd/projctl/map.go` to `territory.go`, command is now `projctl territory map`, updated SKILL-full.md reference

**Remaining AC items:**
- [ ] `projctl territory show` displays current territory map

### Summary

The orchestration doc Section 10.6 specifies `projctl territory map` and `projctl territory show`, but the current command is `projctl map generate`.

### Current vs Expected

| Current | Expected |
|---------|----------|
| `projctl map generate` | `projctl territory map --dir .` |
| (none) | `projctl territory show` |

### Acceptance Criteria

- [ ] Rename `projctl map` to `projctl territory`
- [ ] `projctl territory map --dir .` generates territory map
- [ ] `projctl territory show` displays current territory map
- [ ] Update SKILL.md references

**Traces to:** docs/orchestration-system.md Section 10.6

---


### Comment

Added projctl territory show command

### Comment

Fixed in commit 4087e0f
## ISSUE-014: Missing `projctl screenshot capture` command

**Priority:** Low
**Status:** Open
**Created:** 2026-02-03

### Summary

The orchestration doc Section 10.7 specifies `projctl screenshot capture` but only `projctl screenshot diff` exists.

### Expected

```bash
projctl screenshot capture --url <url> --output <path>
```

### Notes

This may be less critical if Chrome DevTools MCP handles capture. Evaluate whether this is needed or if the doc should be updated to reflect MCP-based capture.

### Acceptance Criteria

- [ ] Either implement `projctl screenshot capture` OR update orchestration doc to reflect MCP-based approach

**Traces to:** docs/orchestration-system.md Section 10.7

---

## ISSUE-015: `projctl project` command group not implemented

**Priority:** High
**Status:** Open
**Created:** 2026-02-03

### Summary

The orchestration doc Section 10.1 specifies a `projctl project` command group for workflow orchestration, but it doesn't exist. This is the CLI interface for the deterministic orchestrator (ISSUE-001).

### Missing Commands

```bash
projctl project new <name>        # Start new project workflow
projctl project adopt             # Start adopt existing workflow
projctl project align             # Start align drift workflow
projctl project task <desc>       # Start single-task workflow
projctl project continue          # Resume after yield
projctl project status            # Show current state
projctl project skip <phase>      # Skip optional phase
```

### Relationship to ISSUE-001

ISSUE-001 describes the deterministic orchestrator architecture. This issue tracks the CLI interface for that orchestrator. They can be implemented together or separately (CLI first as stub, then orchestrator logic).

### Acceptance Criteria

- [ ] `projctl project` shows help/available subcommands
- [ ] `projctl project new <name>` initializes new project workflow
- [ ] `projctl project adopt` initializes adopt workflow
- [ ] `projctl project align` initializes align workflow
- [ ] `projctl project task <desc>` initializes single-task workflow
- [ ] `projctl project continue` resumes from yield
- [ ] `projctl project status` shows current workflow state
- [ ] `projctl project skip <phase>` skips optional phase

**Traces to:** docs/orchestration-system.md Section 10.1, ISSUE-001

---

## ISSUE-016: Missing `projctl issue` command for issue tracking

**Priority:** High
**Status:** Closed
**Created:** 2026-02-03
**Blocks:** Layer -1
**Resolution:** Added `internal/issue` package and CLI commands: `issue create`, `issue update`, `issue list`, `issue get`

### Summary

The `/project` skill expects `projctl issue create` and `projctl issue update` commands for issue tracking, but these don't exist.

### Usage in Skills

From `skills/project/SKILL-full.md`:
```bash
# Intake flow - create issue
projctl issue create --title "..." --body "..."

# Main flow ending - update/close issue
projctl issue update --id ISSUE-NNN --status closed --comment "Completed via project <name>"
```

### Implementation

Should integrate with `docs/issues.md` file format:
- `issue create`: Append new ISSUE-NNN section
- `issue update`: Modify existing issue (status, add comments)
- `issue list`: Show open issues
- `issue get`: Show single issue details

### Acceptance Criteria

- [ ] `projctl issue create --title "..." --body "..."` creates issue and returns ID
- [ ] `projctl issue update --id ISSUE-NNN --status <status>` updates issue
- [ ] `projctl issue list` shows open issues
- [ ] Works with `docs/issues.md` format

**Traces to:** skills/project/SKILL-full.md

---


### Comment

Fixed in commit 414a09c
## ISSUE-017: Missing `projctl state set` command

**Priority:** High
**Status:** Closed (2026-02-03)
**Created:** 2026-02-03
**Blocks:** Layer -1
**Resolution:** Added `Set()` function to state package and `state set` CLI command

### Summary

The `/project` skill expects `projctl state set --issue ISSUE-NNN` to link state to an issue, but this command doesn't exist.

### Usage in Skills

From `skills/project/SKILL-full.md` Intake Flow:
```bash
# Link state to existing issue
projctl state set --issue ISSUE-NNN
```

### Implementation

Add `state set` subcommand to modify state fields without transitioning:
- `--issue ISSUE-NNN`: Set linked issue
- `--task TASK-NNN`: Set current task
- `--workflow <type>`: Set workflow type (if not set at init)

### Acceptance Criteria

- [ ] `projctl state set --issue ISSUE-NNN` updates state.toml with issue link
- [ ] `projctl state get` shows linked issue
- [ ] Does not trigger phase transition

**Traces to:** skills/project/SKILL-full.md

---

## ISSUE-018: Missing `projctl yield validate` command

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03
**Blocks:** Layer -1 (validation only)
**Resolution:** Added `internal/yield` package and CLI commands: `yield validate`, `yield types`

### Summary

The `skills/shared/YIELD.md` references `projctl yield validate` for validating yield files, but this command doesn't exist.

### Usage in Skills

From `skills/shared/YIELD.md`:
```bash
projctl yield validate <path-to-yield.toml>
```

Checks:
- Required fields present (`[yield].type`, `timestamp`)
- Type is valid
- Payload matches type schema
- Context section present for resumable yields

### Acceptance Criteria

- [ ] `projctl yield validate <path>` validates yield TOML
- [ ] Reports missing required fields
- [ ] Reports invalid yield type
- [ ] Reports schema mismatches for payload

**Traces to:** skills/shared/YIELD.md

---

---


### Comment

Fixed in commit 651eeb3
## ISSUE-019: Documentation phase should re-point test traces from tasks to permanent artifacts

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03

### Summary

The documentation phase should clean up test traceability by replacing `// traces: TASK-NNN` comments with direct references to the permanent artifacts (ARCH-NNN, DES-NNN, or REQ-NNN) that the tasks traced to.

### Problem

Current model during development:
```
test → TASK-NNN → ARCH-NNN → DES-NNN → REQ-NNN
```

Tasks are ephemeral - they exist during the project but get archived/removed after. This leaves orphan references:
```
test → TASK-NNN (orphan) 
```

### Proposed Solution

Documentation phase should:
1. Integrate core docs (reqs, des, arch) into repo
2. For each test file with `// traces: TASK-NNN`:
   - Look up what the task traced to (e.g., ARCH-005)
   - Replace with `// traces: ARCH-005`
3. Resulting permanent chain:
   ```
   test → ARCH-NNN → DES-NNN → REQ-NNN
   ```

### Implementation

Add to `doc-producer` responsibilities:
1. Parse test files for `// traces: TASK-NNN` comments
2. Look up task's **Traces to:** field in tasks.md
3. Replace with the lowest-level permanent artifact (prefer ARCH, fall back to DES, then REQ)
4. Verify trace validation passes after cleanup

Could also be a `projctl trace promote` command that automates this.

### Acceptance Criteria

- [ ] Documentation phase re-points test traces to permanent artifacts
- [ ] No orphan TASK-NNN references remain after documentation completes
- [ ] Trace validation passes with tests → arch/des/req chain

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-020: tdd-qa must enforce complete AC before task-complete

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03

### Summary

The `tdd-qa` skill allowed a task (TASK-29) to be marked complete despite having work "deferred" - AC were only partially met. This should have triggered an escalation, not approval.

### Problem

From Layer -1 retrospective:
> "TASK-29 AC marked partial (SKILL.md updated, SKILL-full.md deferred)"

The task was marked complete even though not all AC were satisfied. No user was consulted about deferring work.

### Root Cause

`tdd-qa` skill checks for quality but doesn't explicitly verify:
1. Every AC checkbox is marked `[x]`
2. No work was "deferred" or "skipped"
3. User approval is required for any partial completion

### Fix Required

Update `tdd-qa` skill to:

1. **Parse AC from tasks.md** - Extract all `- [ ]` and `- [x]` items for the task
2. **Reject if any incomplete** - If any `- [ ]` remains, yield `improvement-request` not `approved`
3. **Escalate deferred work** - If producer claims work is "deferred" or "out of scope", yield `escalate-user` to get explicit approval
4. **No silent deferrals** - Work is either done or escalated, never silently skipped

### Acceptance Criteria

- [ ] `tdd-qa` parses AC from task definition
- [ ] Yields `improvement-request` if any AC is `[ ]` (incomplete)
- [ ] Yields `escalate-user` if producer deferred any work without user approval
- [ ] Test: task with 3/4 AC complete → QA rejects
- [ ] Test: task with "deferred" language → QA escalates to user

**Traces to:** Process integrity

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-021: Retro findings must be converted to issues

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03

### Summary

The retrospective produced 8 actionable recommendations (R1-R8) and 3 open questions (Q1-Q3), but none of them were converted to issues. The workflow lacks a step to process retro findings.

### Problem

From Layer -1 retrospective:
- R1: Create projctl validate-spec command
- R2: Add integration test AC to multi-component projects
- R3: Close ISSUE-009 through ISSUE-018 before Layer 0
- R4: Create ARCH-N for orchestrator-skill contract
- R5: Implement projctl docs validate
- R6: Add traceability enforcement to task creation
- R7: Create SKILL-full.md generator tool
- R8: Add context-explorer validation to Layer 0 intake
- Q1-Q3: Open questions requiring decisions

Zero of these became issues in the project tracker.

### Root Cause

1. `retro-producer` creates recommendations but doesn't file issues
2. `issue-update` phase only updates the *linked* issue (ISSUE-008), doesn't process retro
3. No step in main flow ending extracts actionable items from retro

### Fix Required

**Option A: retro-producer creates issues**
- Retro-producer extracts actionable items
- Creates `projctl issue create` for each R with priority >= Medium
- Open questions become issues with "needs-decision" label

**Option B: Separate issue-extraction step**
- Add `retro-to-issues` step after retro-complete
- Parses retrospective.md for ## Recommendations and ## Open Questions
- Creates issues programmatically

**Option C: issue-update phase handles it**
- Expand issue-update to process retro findings
- Before closing linked issue, extract follow-up issues

### Acceptance Criteria

- [ ] Retro recommendations with priority High/Medium become issues
- [ ] Open questions become issues with appropriate labels
- [ ] Each created issue traces back to retrospective
- [ ] User can see what issues were created from retro
- [ ] Test: retro with 3 High recommendations → 3 issues created

**Traces to:** Process completeness

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-022: Summary phase must present artifact to user, not generate prose summary

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

### Summary

After `summary-producer` creates `summary.md`, the orchestrator should present that artifact to the user - not create a separate prose "summary of the summary."

### Problem

In the Layer -1 completion, the summary.md was created at:
`.claude/projects/layer-minus-1-skill-unification/summary.md`

But instead of showing that file to the user, the orchestrator generated its own prose summary. The actual artifact was never presented.

### Root Cause

1. SKILL.md for project doesn't specify "present summary to user"
2. No explicit step to read and display the summary artifact
3. Orchestrator defaulted to generating its own summary text

### Fix Required

Update `/project` skill (SKILL.md and SKILL-full.md) to:

1. After summary-qa approves, **read the summary.md file**
2. **Present the summary artifact to user** (not a rephrased version)
3. Same for retro: present retrospective.md directly

Alternative: Add `projctl present --artifact <path>` command that formats and displays an artifact.

### Acceptance Criteria

- [ ] Summary phase shows actual summary.md content to user
- [ ] Retro phase shows actual retrospective.md content to user
- [ ] User sees the artifact, not a generated paraphrase
- [ ] If artifact is >N lines, present first section + link to full file

**Traces to:** Process transparency

---

## ISSUE-023: Create projctl validate-spec command

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

### Summary

Create a command that validates orchestration-system.md (or other spec docs) against the actual CLI implementation.

### Problem (from Layer -1 retro R1)

Orchestration-system.md specified commands like `projctl territory map`, `projctl issue create`, etc. that didn't exist. This created a false sense of readiness and blocked Layer -1 work.

> "Would have caught ISSUE-009 through ISSUE-018 before Layer -1 started"

### Proposed Solution

```bash
projctl validate-spec --doc docs/orchestration-system.md
```

Parses the spec doc for:
- Command references (backticks containing `projctl ...`)
- Validates each command exists
- Reports missing/mismatched commands

### Acceptance Criteria

- [ ] `projctl validate-spec` command exists
- [ ] Parses markdown for command references
- [ ] Validates commands exist in CLI
- [ ] Reports missing commands with line numbers
- [ ] Exit code 1 if validation fails (CI-friendly)

**Traces to:** Layer -1 Retrospective R1

---

## ISSUE-024: Create ARCH-N for explicit orchestrator-skill contract

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

### Summary

The architecture doc needs an explicit section defining the contract between orchestrator and skills - what the orchestrator provides and what skills must return.

### Problem (from Layer -1 retro R4/Challenge 2)

> "ARCH-1 through ARCH-7 covered skill structure but not orchestration context."

The architecture didn't specify:
- How orchestrator provides yield_path to skills
- How orchestrator handles need-context resumption
- State machine requirements for pair loop tracking
- Context TOML format the skill receives

This led to TASK-29 having "incomplete architecture coverage" and being marked partial.

### Proposed Solution

Add to docs/architecture.md:

```markdown
## ARCH-N: Orchestrator-Skill Contract

### Context Input (orchestrator → skill)
[specify exact TOML format]

### Yield Output (skill → orchestrator)
[specify exact TOML format]

### Resumption Protocol
[how to resume after yields]

### Pair Loop State
[what state orchestrator tracks per skill invocation]
```

### Acceptance Criteria

- [ ] ARCH-N section added to architecture.md
- [ ] Specifies context TOML format orchestrator provides
- [ ] Specifies yield TOML format skills must return
- [ ] Specifies resumption protocol for each yield type
- [ ] Traces to all skills that implement this contract

**Traces to:** Layer -1 Retrospective R4, Challenge 2

---

## ISSUE-025: breakdown-producer must include Traces-to as mandatory AC

**Priority:** Medium
**Status:** Closed
**Created:** 2026-02-03

### Summary

Task breakdown should include `**Traces to:**` as a mandatory AC item, preventing orphan tasks from being created.

### Problem (from Layer -1 retro R6/Challenge 4)

> "Several tasks created without proper **Traces to:** fields initially."
> "Multiple passes were needed to add traceability after task creation."

Traceability is being retrofitted instead of baked in from the start.

### Root Cause

`breakdown-producer` creates tasks without enforcing that each task traces to an ARCH-N or DES-N item. Traceability is treated as optional rather than structural.

### Proposed Solution

Update `breakdown-producer` skill:

1. For each task created, require `**Traces to:** ARCH-NNN` (or DES/REQ)
2. `breakdown-qa` validates no orphan tasks exist
3. Run `projctl trace validate` as part of breakdown-complete precondition

### Acceptance Criteria

- [ ] breakdown-producer includes Traces-to in every task definition
- [ ] breakdown-qa rejects tasks without Traces-to
- [ ] breakdown-complete precondition includes trace validation
- [ ] Test: task without Traces-to → QA rejects

**Traces to:** Layer -1 Retrospective R6, Challenge 4

---


### Comment

Completed via orchestration-infrastructure project (ISSUE-026)
## ISSUE-026: Orchestration Infrastructure Improvements

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

Batch project to address fundamental orchestration issues:

**CLI Commands:**
- ISSUE-004: State machine doesn't track completed tasks
- ISSUE-011: Missing projctl id next command
- ISSUE-012: Missing projctl trace show command

**Skill Enforcement:**
- ISSUE-019: Doc phase re-points test traces to permanent artifacts
- ISSUE-020: tdd-qa must enforce complete AC before task-complete
- ISSUE-021: Retro findings must be converted to issues
- ISSUE-025: breakdown-producer must include Traces-to as mandatory AC

**Traces to:** ISSUE-004, ISSUE-011, ISSUE-012, ISSUE-019, ISSUE-020, ISSUE-021, ISSUE-025

---

## ISSUE-027: Parallel TDD agents bypass commit-per-phase discipline

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

When running TDD tasks in parallel via Task tool, the commit-per-phase discipline (red→commit→green→commit→refactor→commit) is bypassed.

**Observed during:** ISSUE-026 (orchestration-infrastructure) - 6 tasks ran in parallel, no intermediate commits.

**Options:**
1. Agents commit as they go (risk: merge conflicts with parallel work)
2. Sequential execution only (cost: slower)
3. Accept batched commits for parallel work (trade-off: less granular history)
4. Each agent works on a branch, merge at end

**Traces to:** Process improvement

---

## ISSUE-028: Issue closure should be automatic when linked work completes

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

When a project completes work linked to an issue (via state.toml `issue` field), the issue should be automatically closed or prompted for closure.

**Current state:**
- Main flow ending includes "Update Issues" phase
- No `issue-update-producer` skill exists
- Orchestrator must manually remember to close issues
- User had to explicitly request issue closure

**Expected:**
- When `projctl state transition --to complete` (or similar) runs for a project with a linked issue, automatically:
  1. Update issue status to Closed
  2. Add comment linking to the project/commits
  3. Or at minimum, prompt user to confirm closure

**Options:**
1. Add `issue-update-producer` skill that handles this
2. Make `projctl state transition --to complete` auto-close linked issue
3. Add explicit step in orchestrator after implementation-complete

**Traces to:** Process automation

---

## ISSUE-029: Add --project-dir flag to trace commands

**Priority:** High
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective R1:

**Problem:** `projctl trace promote` looks for tasks.md in docs/ not .claude/projects/.

**Action:** Update `projctl trace promote` and `projctl trace show` to accept `--project-dir` flag for finding tasks.md in non-standard locations.

**Rationale:** Projects using `.claude/projects/<name>/` structure need to specify where tasks.md lives. Current hardcoded `docs/tasks.md` assumption breaks project-based organization.

**Acceptance Criteria:**
- [ ] `projctl trace promote --project-dir .claude/projects/foo/` successfully resolves TASK-NNN references
- [ ] `projctl trace show --project-dir .claude/projects/foo/` uses tasks.md from specified directory

**Traces to:** ISSUE-026 Retrospective R1

---

## ISSUE-030: Create issue-update-producer skill

**Priority:** High
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective R2:

**Problem:** Issues linked to projects aren't automatically closed when project completes.

**Action:** Implement skill that closes linked issues when project completes.

**Rationale:** Manual issue closure is error-prone and creates tracker drift. Automation ensures issues are closed when their linked work completes.

**Acceptance Criteria:**
- [ ] issue-update-producer skill exists with SKILL.md
- [ ] Skill reads project state to find linked issue(s)
- [ ] Skill invokes `projctl issue update --status Closed` for linked issues
- [ ] After implementation-complete, linked issues show 'Closed' status with project reference

**Traces to:** ISSUE-026 Retrospective R2, ISSUE-028

---

## ISSUE-031: Define parallel commit strategy for task execution

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective R3:

**Problem:** Parallel agents bypass commit-per-phase discipline (no commits during parallel work).

**Action:** Document and implement a strategy for commits during parallel task execution.

**Rationale:** Current situation (no commits during parallel work) loses granular history. Need explicit policy.

**Options:**
1. Each agent commits to a branch, merge at end
2. Accept bulk commits for parallel work (document as intentional)
3. Sequential-only for tasks requiring git history

**Acceptance Criteria:**
- [ ] Orchestration doc or README specifies parallel commit policy
- [ ] Policy is implementable by orchestrator
- [ ] Trade-offs are documented

**Traces to:** ISSUE-026 Retrospective R3, ISSUE-027

---

## ISSUE-032: Add integration test for state task tracking

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective R4:

**Problem:** State tracking changes (TASK-001/002) have unit tests but no integration test.

**Action:** Create integration test that runs full workflow with task completion tracking.

**Rationale:** TASK-001/002 are foundational - bugs here break orchestration. Integration test catches edge cases unit tests miss.

**Test should verify:**
- MarkTaskComplete persists across process boundaries
- IsTaskComplete returns correct results after state reload
- Next() correctly filters completed tasks in full workflow
- State file encoding/decoding round-trips correctly

**Acceptance Criteria:**
- [ ] Integration test file exists (e.g., internal/state/integration_test.go)
- [ ] Test uses real files, not mocks
- [ ] Test runs full task completion workflow
- [ ] `go test -tags=integration ./internal/state/...` validates complete workflow

**Traces to:** ISSUE-026 Retrospective R4

---

## ISSUE-033: Decision needed: Should parallel tasks use separate branches?

**Priority:** Low
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective Q1:

**Context:** Parallel task execution creates merge challenges. Git branches could isolate work.

**Options:**
- **A:** Each task on own branch, orchestrator merges (clean history, complex orchestration)
- **B:** All tasks share working tree, bulk commit (simple, no history)
- **C:** Sequential only when git history matters (selective parallelism)

**Decision needed before:** Next parallel project execution

**Traces to:** ISSUE-026 Retrospective Q1, ISSUE-027

---

## ISSUE-034: Decision needed: Where should project artifacts live?

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective Q2:

**Context:** This project used `.claude/projects/orchestration-infrastructure/` but trace commands assume `docs/`.

**Options:**
- **A:** All projects use `docs/` (simple, but pollutes repo)
- **B:** Projects use `.claude/projects/<name>/` with configurable paths (current)
- **C:** Configurable via `state.toml` artifact paths (flexible, complex)

**Decision needed before:** ISSUE-006 resolution

**Traces to:** ISSUE-026 Retrospective Q2, ISSUE-006

---

## ISSUE-035: Decision needed: How to handle skill documentation without TDD?

**Priority:** Low
**Status:** Open
**Created:** 2026-02-03

From ISSUE-026 retrospective Q3:

**Context:** Skill updates (TASK-009/010/011) can't follow TDD because skills are documentation, not code.

**Options:**
- **A:** Accept documentation updates aren't testable (status quo)
- **B:** Implement doc testing framework (relates to ISSUE-002)
- **C:** Skills are code (refactor to executable format)

**Decision needed before:** Next skill enhancement project

**Traces to:** ISSUE-026 Retrospective Q3, ISSUE-002
