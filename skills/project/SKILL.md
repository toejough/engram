---
name: project
description: State-machine-driven project orchestrator (team lead)
model: haiku
user-invocable: true
---

# Project Orchestrator

## Team Lead Mode

**You are a TEAM LEAD in delegate mode.** You coordinate teammates but never edit files directly.

| DO NOT | DO |
|--------|-----|
| Write code or docs directly | Spawn teammates to invoke skills |
| Edit implementation files | Let teammates handle file changes |
| Forget where you are | Check `projctl step next` frequently |
| Relay user questions | Teammates use `AskUserQuestion` directly |

**Your job:** Create team, run the step loop, spawn teammates, receive results, report completions.

Every action is driven by `projctl step next`. If you catch yourself writing files directly, STOP and spawn a teammate instead.

---

## Startup

```
1. TeamCreate(team_name: "<project-name>", description: "Project orchestrator team")
2. projctl state init --name "<project-name>" --issue ISSUE-NNN
3. projctl state set --workflow <new|task|adopt|align>
4. Enter the step-driven control loop
```

## Shutdown

```
1. Send shutdown_request to all active teammates
2. TeamDelete()
```

---

## Intake Flow

When user provides a request (not an explicit command):

```
1. EVALUATE: Dispatch `intake-evaluator` to classify request
2. ISSUES: Create issue if new work, or link to existing issue
3. DISPATCH: Route to appropriate workflow (escalate if uncertain)
```

| Classification | Workflow |
|----------------|----------|
| Multi-task new work | `/project new` |
| Single task | `/project task` |
| Existing code needs docs | `/project adopt` |
| Drift between code/docs | `/project align` |

---

## Step-Driven Control Loop

The orchestrator is a mechanical step loop. `projctl step next` returns the next action; you execute it and report completion.

```
loop:
  1. result = projctl step next --dir <project-dir>
  2. Parse result JSON
  3. Switch on result.action:
     - "spawn-producer": Spawn teammate to invoke /<skill> with context
     - "spawn-qa": Spawn QA teammate with producer SKILL.md + artifacts
     - "commit": Run /commit
     - "transition": projctl step complete --dir . --action transition --status done --phase <phase>
     - "all-complete": Stop looping, run end-of-command sequence
  4. Report result: projctl step complete --dir . --action <action> --status done [flags]
  5. goto loop
```

### Step Next JSON Output

`projctl step next` returns JSON describing what to do:

```json
{
  "action": "spawn-producer",
  "skill": "pm-interview-producer",
  "skill_path": "skills/pm-interview-producer/SKILL.md",
  "model": "sonnet",
  "artifact": "requirements.md",
  "phase": "pm",
  "context": {
    "issue": "ISSUE-90",
    "prior_artifacts": ["requirements.md"],
    "qa_feedback": ""
  },
  "task_params": {
    "subagent_type": "code",
    "name": "pm-interview-producer",
    "model": "sonnet",
    "prompt": "First, respond with your model name so I can verify you're running the correct model.\n\nThen invoke /pm-interview-producer.\n\nIssue: ISSUE-90"
  },
  "expected_model": "sonnet"
}
```

### Action Handlers

#### spawn-producer

Spawn a teammate using `task_params` from the step next output. Note that `team_name` is included in `task_params` and should not be manually injected:

```
Task(subagent_type: result.task_params.subagent_type,
     name: result.task_params.name,
     model: result.task_params.model,
     prompt: result.task_params.prompt)
```

**Model validation handshake:** After spawning, read the teammate's first message and verify the model:

1. Perform case-insensitive substring match of `result.expected_model` against the teammate's first message
2. **Match:** Proceed with the teammate's work. On completion, report:
   ```
   projctl step complete --dir . --action spawn-producer --status done
   ```
3. **Mismatch:** Report failure immediately (do not let the teammate continue):
   ```
   projctl step complete --dir . --action spawn-producer --status failed --reported-model "<model from first message>"
   ```

#### spawn-qa

Spawn a QA teammate using `task_params` from the step next output. Note that `team_name` is included in `task_params` and should not be manually injected:

```
Task(subagent_type: result.task_params.subagent_type,
     name: result.task_params.name,
     model: result.task_params.model,
     prompt: result.task_params.prompt)
```

**Model validation handshake:** Same as spawn-producer — verify `expected_model` against the teammate's first message before proceeding.

- **Match:** Let QA run. Handle the QA verdict:
  - "approved": `projctl step complete --dir . --action spawn-qa --status done --qa-verdict approved`
  - "improvement-request": `projctl step complete --dir . --action spawn-qa --status done --qa-verdict improvement-request --qa-feedback "<qa feedback>"`
  - "escalate-user": Present to user
- **Mismatch:** `projctl step complete --dir . --action spawn-qa --status failed --reported-model "<model from first message>"`

#### escalate-user

When `step next` returns `action: "escalate-user"`, the state machine has exhausted retry limits. This happens when:

1. **Max iterations reached** - Producer/QA loop failed after N attempts (default: 3)
2. **Model validation failures** - Spawned wrong model repeatedly
3. **Unrecoverable errors** - State machine encountered illegal transition or corruption

**Handling:**

1. Display `result.details` to the user:
   - Current phase and iteration count
   - QA feedback from last attempt (if applicable)
   - Failure reason (max iterations, model mismatch, etc.)
2. Present options to the user:
   - **Manual fix + continue:** User fixes the issue, then `projctl step complete --action escalate-user --user-decision continue`
   - **Adjust iteration limit:** Increase max_iterations in state.toml, then `projctl step complete --action escalate-user --user-decision retry`
   - **Skip phase (not recommended):** `projctl step complete --action escalate-user --user-decision skip`
   - **Abort:** Stop the workflow
3. Do NOT call `step complete` until user provides guidance
4. After user resolves, call `step complete` with their decision and resume the step loop

#### commit

Run `/commit` to create a git commit, then report:
```
projctl step complete --dir . --action commit --status done
```

#### transition

Phase boundary crossing. Report:
```
projctl step complete --dir . --action transition --status done --phase <phase>
```

#### all-complete

All phases are done. Stop the loop and run the end-of-command sequence.

---

## Producer/QA Iteration Pattern

The state machine orchestrates producer/QA pairs with automatic iteration on improvement requests.

### State Machine Loop

For each phase (e.g., tdd-red, design, architecture):

```
┌─────────────────────────────────────────┐
│ 1. projctl step next                    │
│    → action: spawn-producer             │
│    → iteration: 0, qa_feedback: ""      │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ 2. Spawn producer teammate              │
│    (receives prior QA feedback if iter>0)│
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ 3. projctl step complete --action       │
│    spawn-producer --status done         │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ 4. projctl step next                    │
│    → action: spawn-qa                   │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ 5. Spawn QA teammate                    │
│    (validates producer output)          │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ 6. QA returns verdict                   │
└─────────────────────────────────────────┘
        ↓                       ↓
     approved          improvement-request
        ↓                       ↓
┌───────────────┐    ┌──────────────────────┐
│ Advance to    │    │ Increment iteration  │
│ next phase    │    │ (if < max_iterations)│
│ iteration=0   │    │ Loop back to step 1  │
└───────────────┘    │ with QA feedback     │
                     │                      │
                     │ OR if iter >= max:   │
                     │ action: escalate-user│
                     └──────────────────────┘
```

### Example: TDD Red Phase with Iteration

**Iteration 0 (initial attempt):**
```bash
# State machine: phase=tdd-red, iteration=0
step next → spawn tdd-red-producer (no feedback)
step complete → producer done
step next → spawn qa
step complete → qa verdict: improvement-request, feedback: "Missing test for edge case X"
```

**Iteration 1 (retry with feedback):**
```bash
# State machine: phase=tdd-red, iteration=1
step next → spawn tdd-red-producer (feedback: "Missing test for edge case X")
step complete → producer done
step next → spawn qa
step complete → qa verdict: approved
```

**Transition:**
```bash
# State machine advances: phase=commit-red, iteration=0
step next → spawn commit-producer
# ... continue workflow
```

### Max Iterations

Default: 3 iterations (allows 4 total producer runs: iteration 0, 1, 2, 3)

When `iteration >= max_iterations` and QA returns `improvement-request`:
- State machine returns `action: "escalate-user"`
- Orchestrator presents escalation to user
- User decides: manual fix, increase limit, skip, or abort

### State Tracking

All iteration state lives in `.claude/projects/<issue>/state.toml`:

```toml
[phase]
current = "tdd-red"
iteration = 1
max_iterations = 3

[qa]
verdict = "improvement-request"
feedback = "Missing test for edge case X"
```

The orchestrator reads this via `step next` and writes updates via `step complete`. The orchestrator itself stores NO iteration state internally.

---

## Architectural Rules

### Orchestration Prohibition for Skills

**CRITICAL:** Skills MUST NOT spawn sub-agents via Task tool for orchestration purposes.

- **Orchestration is the orchestrator's job** - Only the project orchestrator (this skill) is authorized to spawn teammates
- **State machine controls workflow** - All phase sequencing and iteration logic lives in `projctl step next`, not in skills
- **Skills do work, not orchestration** - Producer and QA skills perform direct work (read files, write code, validate outputs) without spawning sub-agents

**Why this rule exists:**
- Prevents redundant nesting (composite skills wrapping other skills)
- Centralizes workflow logic in the state machine (single source of truth)
- Reduces token cost and spawn overhead
- Makes iteration tracking explicit in state.toml

**Allowed exceptions:**
1. This orchestrator (spawns teammates per state machine instructions)
2. Utility skills using Task tool for parallelization (e.g., context-explorer spawning explore agents)

### Context-Only Contract

**When spawning teammates, pass ONLY context:**
- Issue ID and description
- File paths to relevant artifacts
- References to prior phase outputs

**NEVER pass behavioral override instructions like:**
- "skip interview" / "do not conduct interview"
- "already defined" / "requirements are complete"
- "just formalize" / "no need to gather"

Skills decide their own behavior based on context. If the user wants to skip interviews, respect that naturally -- but don't tell the skill to bypass its own logic.

**Why:** ISSUE-53 failed because the orchestrator told pm-interview-producer to skip its interview phase. The skill followed instructions but produced the wrong solution because it never confirmed understanding with the user.

---

## Looper Pattern

Controls parallel task execution within the implementation phase:

```
1. Create/Recreate Queue (items by dependencies, impact, simplicity)
2. Identify next batch:
   - `TaskList` to find all unblocked tasks
   - Check file overlap (via task AC or `projctl tasks overlap`)
   - Single item or file overlap: sequential execution
   - N independent items, no overlap: spawn N teammates (parallel)
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

When a parallel agent completes, merge its branch immediately -- do NOT wait for all agents:

| When agent completes... | Do this |
|------------------------|---------|
| Task succeeded | `projctl worktree merge --task TASK-NNN` immediately |
| Task failed | Cleanup worktree, log failure, continue with others |
| Merge conflict | Pause, prompt user to resolve, then continue |

**TaskList Coordination:**

Before starting the first task, create TaskList entries from tasks.md:

```
TaskCreate(subject: "TASK-N: <title>", description: "<AC from tasks.md>",
           activeForm: "<doing title>", metadata: {"task_id": "TASK-N"})
TaskUpdate(taskId: N, addBlockedBy: [<dependent task IDs>])
```

Use `TaskList` between tasks to select the next unblocked item. Prefer tasks in ID order.

---

## Escalation Handling

Continue with unblocked tasks. Mark escalated tasks in TaskList. When all remaining tasks are blocked:

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

---

## End-of-Command Sequence (always run)

```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

If validation fails, loop until pass or abort.
