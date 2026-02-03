---
name: project
description: State-machine-driven project orchestrator
user-invocable: true
---

# Project Orchestrator - Full Reference

See SKILL.md for the overview. This document covers phase details and edge cases.

---

## Control Loop

Every phase follows this pattern:

```
[D] = Deterministic (CLI)    [A] = Agentic (LLM)

[A] 0. Classify message (detect corrections → log lesson)
[D] 1. projctl state get --dir .
[D] 2. projctl territory map --cached
[D] 3. projctl memory query "relevant to current task"
[D] 4. projctl context write --dir . --task TASK --skill SKILL --file context.toml
[A] 5. Dispatch skill via Skill tool
[D] 6. projctl context read --dir . --task TASK --skill SKILL --result
[D] 7. projctl state next --dir .
[A] 8. If action=continue: loop. If action=stop: check reason.
```

### Stop Reasons

| Reason | Action |
|--------|--------|
| `all_complete` | Present summary |
| `escalation_pending` | Present to user |
| `validation_failed` | Run repair loop |
| `retries_exhausted` | Present failure |

### Anti-patterns

- NEVER say "No response requested" when work remains
- NEVER ask "Should I continue?" if `state next` returns `continue`
- NEVER wait for confirmation between TDD phases

---

## Looper Pattern

Controls iteration within a phase or across tasks:

```
1. Create/Recreate Queue (items by dependencies, impact, simplicity)
2. Identify next batch:
   - Find all items with no blocking dependencies
   - Single item → PAIR LOOP
   - N independent items → PARALLEL LOOPER
3. Execute batch
4. Re-evaluate queue (dependencies may have resolved)
5. Repeat until queue empty or entirely blocked
```

**Parallel Looper** (for N independent items):

```
1. Receive items (all verified independent)
2. Launch PAIR LOOP for each item (in parallel)
3. Collect all yields
4. Run `consistency-checker` on aggregated results
5. Return aggregated result or remediation needs
```

Applies to: independent tasks, context queries, batch file analysis.

---

## Intake Flow

When user provides a request without an explicit command, the orchestrator classifies and routes:

### 1. Evaluate Request

Dispatch `intake-evaluator` to analyze:
- Is this a new issue to file?
- Is this work on an existing issue/project?
- Is this a simple task or multi-task project?
- If uncertain: escalate to user for clarification

### 2. Create/Link Issues

```bash
# New work: create issue first
projctl issue create --title "..." --body "..."

# Existing work: link to issue
projctl state set --issue ISSUE-NNN
```

### 3. Dispatch to Workflow

| Signal | Workflow |
|--------|----------|
| Multi-task new work | `new` |
| Single task | `task` |
| Existing codebase needs docs | `adopt` |
| Drift detected between code and docs | `align` |

If user corrects dispatch decision, capture in retrospective as a learning.

---

## Phase Reference

### Dashboard (`/project`)

Show open projects with state.toml files:

```
Project Orchestrator

Open Projects:
  project-name     phase              progress        issues
  my-cli-tool      implementation     7/12 tasks      1 escalated

Commands: new | adopt | align | task | continue
```

### Initialization

| Flow | Command |
|------|---------|
| new | `projctl state init --name NAME --dir DIR` |
| adopt | `projctl state init --name NAME --dir DIR --mode adopt` |
| align | `projctl state init --name NAME-align --dir DIR --mode align` |
| task | `projctl state init --name NAME --dir DIR --mode task` |

### PM Phase

**Interview mode** (new): Dispatch `pm-interview-producer`
**Infer mode** (adopt/align): Dispatch `pm-infer-producer` with mode=create or mode=update

Output: requirements.md with REQ-NNN IDs

### Design Phase

**Interview mode** (new): Dispatch `design-interview-producer`
**Infer mode** (adopt/align): Dispatch `design-infer-producer`

Output: design.md with DES-NNN IDs, optional .pen files

### Architecture Phase

**Interview mode** (new): Dispatch `arch-interview-producer`
**Infer mode** (adopt/align): Dispatch `arch-infer-producer`

Output: architecture.md with ARCH-NNN IDs

### Alignment Phase (main flow ending)

Runs once after workflow completes, before Retro/Summary/Next Steps.

```bash
projctl trace validate --dir .
```

If fails:
1. Dispatch `alignment-producer` to interpret gaps
2. Apply fixes to `**Traces to:**` fields
3. Re-validate (max 2 iterations)
4. Escalate unresolved gaps to user

### Task Breakdown Phase

Dispatch `breakdown-producer`

Output: tasks.md with TASK-NNN IDs and dependency graph

### TDD Implementation Loop

For each task in dependency order:

```bash
projctl state transition --dir . --to task-start --task TASK-NNN
```

**Red → Commit → Green → Commit → Refactor → Commit** (atomic sequence)

| Sub-phase | Skill | Next |
|-----------|-------|------|
| tdd-red | `tdd-red-producer` | commit-red |
| commit-red | `commit` | tdd-green |
| tdd-green | `tdd-green-producer` | commit-green |
| commit-green | `commit` | tdd-refactor |
| tdd-refactor | `tdd-refactor-producer` | commit-refactor |
| commit-refactor | `commit` | task-audit |
| task-audit | `tdd-qa` | task-complete or retry |

**After audit:**
- Pass → `task-complete`
- Fail (< 3 attempts) → `task-retry` → back to tdd-red
- Fail (>= 3 attempts) → `task-escalated`

### Escalation Handling

Continue with unblocked tasks. When all blocked:

```
Implementation paused: N tasks escalated.

TASK-005: [description]
  Attempt 1: [failure]
  Attempt 2: [failure]
  Attempt 3: [failure]

Options:
1. Provide guidance
2. Mark won't-fix
3. Pause project
```

### Documentation Phase

Runs after Implementation completes.

1. Dispatch `doc-producer`
2. Dispatch `doc-qa`
3. Integrates project artifacts into repo-level docs:
   - `.claude/projects/<name>/requirements.md` → `docs/requirements.md`
   - `.claude/projects/<name>/design.md` → `docs/design.md`
   - `.claude/projects/<name>/architecture.md` → `docs/architecture.md`

Tasks remain in `.claude/projects/<name>/tasks.md` as project history.

### Adopt/Align Workflow (bottom-up inference)

Infers artifacts from existing code, working upward:

| Phase | Skill | Output |
|-------|-------|--------|
| Explore | Implementation Explorer | Codebase analysis |
| Infer Tests | Test inference | Test mapping |
| Infer Arch | `arch-infer-producer` + `arch-qa` | architecture.md |
| Infer Design | `design-infer-producer` + `design-qa` | design.md |
| Infer Reqs | `pm-infer-producer` + `pm-qa` | requirements.md |
| Escalations | User resolution | Resolved ambiguities |
| Documentation | `doc-producer` + `doc-qa` | Repo-level docs |

Escalations collected during inference are batched for user resolution.

### Single Task Workflow (`/project task`)

Lightweight flow for simple work:

| Phase | Skill | Notes |
|-------|-------|-------|
| Implementation | TDD Producer (composite) | Full TDD cycle |
| Documentation | `doc-producer` (if user-facing) | Optional |

Returns to main flow for Alignment → Retro → Summary → Next Steps.

### Retrospective Phase (main flow ending)

1. Dispatch `retro-producer`
2. Dispatch `retro-qa`
3. Output: `.claude/projects/<name>/retro.md`, follow-up issues

### Summary Phase (main flow ending)

1. Dispatch `summary-producer`
2. Dispatch `summary-qa`
3. Output: `.claude/projects/<name>/summary.md`

### Issue Update (main flow ending)

**Automatic issue closure** - no skill dispatch needed.

```bash
# Get linked issue from state
ISSUE=$(projctl state get --dir . | grep -oP 'issue = "\K[^"]+')

# If project has a linked issue, close it automatically
if [ -n "$ISSUE" ]; then
  projctl issue update -i "$ISSUE" --status Closed \
    --comment "Completed via project $(projctl state get --dir . --field name)"
fi

# Create follow-up issues from retro findings (if any)
# These are created by retro-producer, not this phase
```

**This is deterministic** - just run the commands above. No PAIR LOOP.

### Next Steps Phase (main flow ending)

1. Dispatch `next-steps`
2. Suggest follow-on work based on open issues
3. Run end-of-command sequence

---

## End-of-Command Sequence

**Every command must run:**

```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

**If validation fails:**

```
Validation Issues Found

Unlinked IDs: 3
  REQ-005: No downstream design
  DES-008: No downstream architecture
  ARCH-012: No downstream task

Pending Escalations: 2

Options:
1. Resolve now
2. Defer to issues
3. Abort
```

Loop until pass or abort.

---

## Resume Map (`/project continue`)

| State | Resume Action |
|-------|---------------|
| init | Start PM |
| pm-complete | Start Design |
| design-complete | Start Architecture |
| architect-complete | Start Breakdown |
| breakdown-complete | Start Implementation |
| task-start, tdd-* | Resume TDD at sub-phase |
| commit-* | Resume commit |
| task-complete | Next task or Implementation complete |
| implementation-complete | Start Documentation |
| documentation-complete | Start Alignment (main flow ending) |
| alignment-complete | Start Retro |
| retro-complete | Start Summary |
| summary-complete | Start Issue Update |
| issue-update-complete | Start Next Steps |
| adopt-explore | Resume exploration |
| adopt-infer-tests | Resume test inference |
| adopt-infer-arch | Resume architecture inference |
| adopt-infer-design | Resume design inference |
| adopt-infer-reqs | Resume requirements inference |
| adopt-escalations | Resume escalation resolution |
| adopt-documentation | Resume documentation |
| align-* | Same as adopt (detect and fix drift) |
| task-implementation | Resume single-task TDD |
| task-documentation | Resume single-task documentation |

---

## Context Format

```toml
[input]
phase = "pm"
task = "TASK-5"
# phase-specific data

[output]
yield_path = ".claude/agents/pm-interview-producer-yield.toml"
```

---

## Logging

```bash
projctl log write --dir . --level LEVEL --subject SUBJECT --message "..."
```

| Level | When | Subjects |
|-------|------|----------|
| verbose | Every dispatch/result | thinking, skill-change |
| status | Task completions | skill-result, task-status, lesson |
| phase | Phase transitions | phase-change, phase-result |

---

## Error Handling

| Error | Action |
|-------|--------|
| Skill dispatch fails | Log, notify user, offer retry or skip |
| Task repeatedly fails | After 3 attempts escalate, continue unblocked |
| State corruption | `projctl state get` reports errors, user inspects state.toml |
