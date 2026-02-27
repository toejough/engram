---
name: team-orchestration
description: This skill should be used when about to "create a team", "team", "spawn agents", "dispatch teammates", "parallelize work", "parallelize", "run in parallel", "launch subagents", or whenever work involves creating any subagent, helper, or starting tasks that are parallelizable or long-running. Enforces structured teams with task lists, paired QA agents, model-minimal teammates, and maximum parallelism while keeping the team lead free for user interaction.
---

# Team Orchestration

Coordinate parallel work through structured teams with task lists, cost-efficient model selection, paired QA validation, and a lead that stays available for the user.

**This skill supersedes `dispatching-parallel-agents` and `subagent-driven-development` for all parallel work.** Never use bare `Task` calls when work can be parallelized. Always create a team.

## Core Principles

1. **Always TeamCreate** -- every multi-agent effort gets a real team with a task list
2. **Maximum parallelism** -- spawn all independent teammates simultaneously
3. **No idle teammates** -- working or shut down, never waiting
4. **Minimal viable model** -- Sonnet for execution, Haiku for read-only/QA
5. **Paired QA** -- every executor gets a dedicated Haiku QA agent
6. **Lead stays free** -- teammates talk to each other; lead handles the user

## When This Triggers

Any time the lead is about to:
- Spawn a subagent, helper, or teammate
- Start work that will take more than a few minutes
- Begin parallelizable tasks (2+ independent work items)
- Execute an implementation plan with multiple steps

If in doubt, create a team. The overhead is minimal; the coordination benefit is large.

## Workflow

### 1. Map All Known Work to Tasks

Before spawning anyone, populate the task list with every known work item using `TaskCreate`. Include:
- Clear subject (imperative form)
- Description with acceptance criteria and file scope
- `activeForm` for spinner display

As new work is discovered during execution, immediately add it with `TaskCreate`. The task list is the single source of truth.

### 2. Identify Dependencies and Parallelism

Use `TaskUpdate` with `addBlocks`/`addBlockedBy` to map dependencies. Independent tasks (no shared files, no ordering requirement) launch simultaneously. Blocked tasks wait.

Verify zero file overlap between parallel tasks to prevent merge conflicts. If two tasks touch the same file, serialize them or split the file-touching portion into its own task.

### 3. Select Models and Spawn Teammates

| Role | Model | subagent_type | When |
|------|-------|---------------|------|
| Executor | `sonnet` | `general-purpose` | Writing code, editing files, running commands |
| Read-only researcher | `haiku` | `Explore` | Searching, reading, gathering information |
| QA validator | `haiku` | `general-purpose` | Validating executor output (read-only by instruction) |

Spawn all independent executors simultaneously in a single message with multiple `Task` calls, each with `team_name` set. For every executor, spawn its paired QA agent at the same time -- the QA agent's task should be `addBlockedBy` the executor's task so it activates only after execution completes.

### 4. Paired QA Pattern

Every execution task gets a dedicated Haiku QA agent. The QA agent:

- **Is read-only by instruction** -- inspects artifacts, reads files, checks test output
- **Validates against the task's acceptance criteria** in the task list
- **Reports to the executor** (via `SendMessage`), not to the lead
- **If the executor got something wrong**: tells the executor exactly what to fix
- **If insufficient info to evaluate**: tells the executor what evidence to produce
- **Only messages the lead** when validation passes or after 2 failed fix attempts

QA prompt structure -- see `references/agent-prompts.md` for templates.

### 5. Communication Rules

Teammates communicate peer-to-peer:
- **QA <-> Executor**: QA sends findings to executor; executor sends fixes back
- **Executor -> Lead**: Only when task is fully complete and QA-validated
- **QA -> Lead**: Only when validation passes, or after 2 failed remediation cycles
- **Lead -> Teammate**: To assign new work, unblock, or answer questions teammates escalate

The lead does NOT poll teammates. Idle notifications arrive automatically. The lead uses the time between notifications to interact with the user -- brainstorming, launching new tasks, answering questions.

### 6. Lifecycle: No Idle Teammates

After a teammate completes its task and QA validates:
1. Check `TaskList` for unblocked pending tasks
2. If available: assign via `TaskUpdate` and `SendMessage` with the new task
3. If none available: `SendMessage` with `type: "shutdown_request"`

Never let a teammate sit idle waiting for work. Shut down completed teammates immediately and spawn fresh ones if new work appears later.

### 7. Team Shutdown: All Teammates First

**Never call `TeamDelete` until every teammate has been shut down successfully.** The shutdown sequence:

1. Send `shutdown_request` to each remaining teammate
2. Wait for each `shutdown_response` confirming approval
3. Verify no teammates are still running (check idle notifications)
4. Only after all teammates have confirmed shutdown: call `TeamDelete`

If a teammate rejects shutdown (still working), wait for it to finish before retrying. `TeamDelete` will fail if active members remain -- do not force it.

### 8. Lead Availability Pattern

The lead's primary job is **staying available for the user**. Delegate everything possible:

- **Do**: Create teams, populate task lists, spawn teammates, answer user questions, handle brainstorming, launch new work
- **Don't**: Implement tasks yourself, block on teammate output, micromanage progress, do QA yourself

If the user sends a message while teammates are running, respond immediately. The team runs autonomously in the background.

## Model Selection Details

**Default to Sonnet** for any task that writes files, runs commands, or edits code.

**Use Haiku** for:
- QA validation (reading artifacts, checking output)
- Codebase exploration and research
- Gathering information before planning
- Any task that only reads, never writes

**Use Opus** only when explicitly requested by the user or for genuinely hard architectural decisions.

## Anti-Patterns

| Anti-Pattern | Correct Approach |
|---|---|
| Bare `Task` calls without TeamCreate | Always create a team first |
| Sequential executor dispatch | Spawn all independent executors simultaneously |
| Lead doing implementation work | Delegate to a teammate |
| Lead polling teammates for status | Wait for automatic notifications |
| Idle teammate waiting for next task | Shut down or assign immediately |
| QA reporting directly to lead | QA reports to executor first |
| Sonnet for read-only work | Use Haiku |
| Skipping QA for "simple" tasks | Every executor gets QA |
| `TeamDelete` before all teammates shut down | Shut down every teammate first, then delete team |

## Task List Discipline

The task list reflects reality at all times:
- New work discovered -> `TaskCreate` immediately
- Starting a task -> `TaskUpdate` status to `in_progress`
- Task done -> `TaskUpdate` status to `completed`
- Task obsolete -> `TaskUpdate` status to `deleted`
- Scope changed -> `TaskUpdate` description

Never let the task list go stale. It is the coordination backbone.

## Quick Reference: Team Startup Sequence

1. `TeamCreate` with descriptive name
2. `TaskCreate` for every known work item
3. `TaskUpdate` to set dependencies
4. Spawn executors (Sonnet) + paired QA (Haiku) for all independent tasks in one message
5. Respond to user while team works
6. On teammate completion notifications: assign next task or shut down teammate
7. When all tasks complete: shut down every teammate (wait for confirmation)
8. After all teammates confirmed shutdown: `TeamDelete`, report to user

## Additional Resources

### Reference Files

- **`references/agent-prompts.md`** -- Prompt templates for executors and QA agents with examples
