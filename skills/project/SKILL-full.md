---
name: project
description: State-machine-driven project orchestrator (team lead)
user-invocable: true
---

# Project Orchestrator - Full Reference

See SKILL.md for the overview. This document covers phase details and edge cases.

---

## Control Loop

Every phase follows this pattern:

```
[D] = Deterministic (CLI)    [A] = Agentic (LLM)

[D] 1. projctl state get --dir .
[D] 2. projctl territory map --cached
[D] 3. projctl memory query "relevant to current task"
[A] 4. Spawn producer teammate with context (project info, artifacts, prior results)
[A] 5. Receive producer result via message
[A] 6. Spawn QA teammate with context (producer SKILL.md, artifacts, iteration)
[A] 7. Receive QA verdict via message
[A] 8. Handle verdict: advance, iterate, or escalate
[D] 9. /commit (after each phase completion)
[D] 10. projctl state transition --dir . --to <next-phase>
```

### Verdict Handling

| Verdict | Action |
|---------|--------|
| `approved` | Commit, advance to next phase |
| `improvement-request` | Spawn new producer with QA feedback (max 3x) |
| `escalate-phase` | Return to prior phase |
| `escalate-user` | Present to user |

### Anti-patterns

- NEVER say "No response requested" when work remains
- NEVER ask "Should I continue?" if `state next` returns `continue`
- NEVER wait for confirmation between TDD phases
- NEVER relay user questions through messages — teammates ask directly

---

## Team Lifecycle

### Startup

```
Teammate(operation: "spawnTeam", team_name: "<project-name>")
```

Creates a team. All subsequent Task calls with `team_name` join this team.

### Spawning Teammates

Each phase spawns a producer teammate and a QA teammate:

```
Task(subagent_type: "general-purpose",
     team_name: "<project>",
     name: "pm-producer",
     prompt: "Invoke /pm-interview-producer for this project.

Project: <name>
Issue: ISSUE-NNN
Phase: pm
Docs dir: docs/
Requirements path: docs/requirements.md

Prior context:
<territory map summary>
<memory query results>
<issue description>

When complete, send me a message with:
- Artifact path
- IDs created (REQ-N list)
- Files modified
- Key decisions made")
```

### Receiving Results

Messages from teammates are delivered automatically. Parse the message to determine:
- Was it a completion (artifact produced)?
- Was it a blocker (needs escalation)?

### Shutdown

After all phases complete:
```
SendMessage(type: "shutdown_request", recipient: "<teammate-name>")
Teammate(operation: "cleanup")
```

---

## Looper Pattern

Controls iteration within a phase or across tasks:

```
1. Create/Recreate Queue (items by dependencies, impact, simplicity)
2. Identify next batch:
   - Find all items with no blocking dependencies
   - Single item → PAIR LOOP
   - N independent items → spawn one teammate per item (parallel)
3. Execute batch
4. Re-evaluate queue (dependencies may have resolved)
5. Repeat until queue empty or entirely blocked
```

**Git Worktrees for Parallel Tasks:**

When running parallel tasks, each agent works in an isolated git worktree:

```bash
# On task start (per parallel agent)
projctl worktree create --task TASK-NNN
# Agent works in .worktrees/task-NNN/ directory

# On agent completion - MERGE IMMEDIATELY
projctl worktree merge --task TASK-NNN
# Rebases onto main, merges, cleans up
```

**Merge-on-Complete Pattern (REQUIRED):**

When a parallel agent completes, merge its branch immediately — do NOT wait for all agents:

| When agent completes... | Do this |
|------------------------|---------|
| Task succeeded | `projctl worktree merge --task TASK-NNN` immediately |
| Task failed | Cleanup worktree, log failure, continue with others |
| Merge conflict | Pause, prompt user to resolve, then continue |
| Cleanup failure | Log error, continue, report at end |

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

## Project Layout

Project artifacts live at the **project root directory**, not in a `docs/` subdirectory.

```
.claude/projects/<project-name>/
├── state.toml           # Project state (phase, task, progress)
├── requirements.md      # REQ-N items
├── design.md            # DES-N items
├── architecture.md      # ARCH-N items
├── tasks.md             # TASK-N breakdown with dependencies
├── escalations.md       # ESC-N unresolved issues
├── retro.md             # Project retrospective
└── summary.md           # Project summary
```

**Key points:**
- `projctl state init --name foo` creates `.claude/projects/foo/` and `state.toml`
- All artifact lookups use the project dir directly (not `project-dir/docs/`)
- The **documentation phase** copies completed artifacts to repo-level `docs/`:
  - `requirements.md` → `docs/requirements.md`
  - `design.md` → `docs/design.md`
  - `architecture.md` → `docs/architecture.md`
- `tasks.md` stays in the project dir as historical record

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
| new | `projctl state init --name NAME` |
| adopt | `projctl state init --name NAME --mode adopt` |
| align | `projctl state init --name NAME-align --mode align` |
| task | `projctl state init --name NAME --mode task` |

Note: `--dir` defaults to `.claude/projects/<NAME>/` and creates it if needed.

### PM Phase

**Interview mode** (new): Spawn teammate → invoke `pm-interview-producer`
**Infer mode** (adopt/align): Spawn teammate → invoke `pm-infer-producer`

Output: requirements.md with REQ-N IDs

QA: Spawn teammate → invoke `qa` with context:
```
Producer SKILL.md: skills/pm-interview-producer/SKILL.md
Artifacts: .claude/projects/<name>/requirements.md
Iteration: 1/3
```

### Design Phase

**Interview mode** (new): Spawn teammate → invoke `design-interview-producer`
**Infer mode** (adopt/align): Spawn teammate → invoke `design-infer-producer`

Output: design.md with DES-N IDs, optional .pen files

QA: Spawn teammate → invoke `qa` with context:
```
Producer SKILL.md: skills/design-interview-producer/SKILL.md
Artifacts: .claude/projects/<name>/design.md
Iteration: 1/3
```

### Architecture Phase

**Interview mode** (new): Spawn teammate → invoke `arch-interview-producer`
**Infer mode** (adopt/align): Spawn teammate → invoke `arch-infer-producer`

Output: architecture.md with ARCH-N IDs

QA: Spawn teammate → invoke `qa` with context:
```
Producer SKILL.md: skills/arch-interview-producer/SKILL.md
Artifacts: .claude/projects/<name>/architecture.md
Iteration: 1/3
```

### Alignment Phase (main flow ending)

Runs once after workflow completes, before Retro/Summary/Next Steps.

```bash
projctl trace validate --dir .
```

If fails:
1. Spawn alignment-producer teammate to interpret gaps
2. Apply fixes to `**Traces to:**` fields
3. Re-validate (max 2 iterations)
4. Escalate unresolved gaps to user

### Task Breakdown Phase

Spawn teammate → invoke `breakdown-producer`

Output: tasks.md with TASK-N IDs and dependency graph

QA: Spawn teammate → invoke `qa` with context:
```
Producer SKILL.md: skills/breakdown-producer/SKILL.md
Artifacts: .claude/projects/<name>/tasks.md
Iteration: 1/3
```

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
| task-audit | `qa` | task-complete or retry |

**After audit:**
- Pass → `task-complete`
- Fail (< 3 attempts) → `task-retry` → back to tdd-red
- Fail (>= 3 attempts) → `task-escalated`

### Documentation-Focused Tasks

Documentation tasks get full TDD treatment when ANY of these indicators are present:

| Indicator | Example |
|-----------|---------|
| Issue mentions docs | "Update SKILL.md with new yield types" |
| Task AC target .md files | "- [ ] README.md includes installation" |
| Task explicitly about docs | "Document the API endpoints" |

**Do NOT skip TDD for doc tasks.** Apply the same red-green-refactor cycle:
- RED: Write tests for what the doc should contain (grep, semantic matching, structure)
- GREEN: Write the doc to make tests pass
- REFACTOR: Improve clarity, structure, readability

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

1. Spawn teammate → invoke `doc-producer`
2. Spawn teammate → invoke `qa` with context:
```
Producer SKILL.md: skills/doc-producer/SKILL.md
Artifacts: docs/requirements.md, docs/design.md, docs/architecture.md
Iteration: 1/3
```
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
| Infer Arch | `arch-infer-producer` + `qa` | architecture.md |
| Infer Design | `design-infer-producer` + `qa` | design.md |
| Infer Reqs | `pm-infer-producer` + `qa` | requirements.md |
| Escalations | User resolution | Resolved ambiguities |
| Documentation | `doc-producer` + `qa` | Repo-level docs |

Escalations collected during inference are batched for user resolution.

### Single Task Workflow (`/project task`)

Lightweight flow for simple work:

| Phase | Skill | Notes |
|-------|-------|-------|
| Implementation | TDD Producer (composite) | Full TDD cycle |
| Documentation | `doc-producer` (if user-facing) | Optional |

Returns to main flow for Alignment → Retro → Summary → Next Steps.

### Retrospective Phase (main flow ending)

1. Spawn teammate → invoke `retro-producer`
2. Spawn teammate → invoke `qa`
3. Output: `.claude/projects/<name>/retro.md`, follow-up issues
4. **Present artifact to user**: Read and display `retro.md` content (not a paraphrase)

### Summary Phase (main flow ending)

1. Spawn teammate → invoke `summary-producer`
2. Spawn teammate → invoke `qa`
3. Output: `.claude/projects/<name>/summary.md`
4. **Present artifact to user**: Read and display `summary.md` content (not a paraphrase)
   - For long summaries (>50 lines), show Executive Overview + link to full file

### Issue Update (main flow ending)

**Automatic issue closure** — no teammate needed.

```bash
# Get linked issue from state
ISSUE=$(projctl state get --dir . | grep 'issue = ' | sed 's/.*issue = "\([^"]*\)".*/\1/')

# If project has a linked issue, close it automatically
if [ -n "$ISSUE" ]; then
  projctl issue update -i "$ISSUE" --status Closed \
    --comment "Completed via project $(projctl state get --dir . --field name)"
fi
```

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
| pm-producer | Resume PM producer |
| pm-qa | Resume QA with PM context |
| pm-complete | Start Design |
| design-producer | Resume Design producer |
| design-qa | Resume QA with Design context |
| design-complete | Start Architecture |
| arch-producer | Resume Architecture producer |
| arch-qa | Resume QA with Architecture context |
| architect-complete | Start Breakdown |
| breakdown-producer | Resume Breakdown producer |
| breakdown-qa | Resume QA with Breakdown context |
| breakdown-complete | Start Implementation |
| task-start, tdd-* | Resume TDD at sub-phase |
| commit-* | Resume commit |
| task-audit | Resume QA with TDD context |
| task-complete | Next task or Implementation complete |
| implementation-complete | Start Documentation |
| doc-producer | Resume Documentation producer |
| doc-qa | Resume QA with Documentation context |
| documentation-complete | Start Alignment (main flow ending) |
| alignment-producer | Resume Alignment producer |
| alignment-qa | Resume QA with Alignment context |
| alignment-complete | Start Retro |
| retro-producer | Resume Retro producer |
| retro-qa | Resume QA with Retro context |
| retro-complete | Start Summary |
| summary-producer | Resume Summary producer |
| summary-qa | Resume QA with Summary context |
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

**Resume**: When resuming, re-create the team (`spawnTeam`) and spawn a new teammate for the current phase. The state machine tracks where you left off; teammates are ephemeral.

---

## Error Handling

| Error | Action |
|-------|--------|
| Teammate fails to respond | Check idle notification, re-spawn with same context |
| Task repeatedly fails | After 3 attempts escalate, continue unblocked |
| State corruption | `projctl state get` reports errors, user inspects state.toml |
