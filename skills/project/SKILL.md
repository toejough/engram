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
1. Teammate(operation: "spawnTeam", team_name: "<project-name>")
2. projctl state init --name "<project-name>" --issue ISSUE-NNN
3. projctl state set --workflow <new|task|adopt|align>
4. Enter the step-driven control loop
```

## Shutdown

```
1. Send shutdown_request to all active teammates
2. Teammate(operation: "cleanup")
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
  }
}
```

### Action Handlers

#### spawn-producer

Spawn a teammate to run the producer skill indicated by `result.skill`:

```
Task(subagent_type: "general-purpose",
     team_name: "<project>",
     name: "<phase>-producer",
     prompt: "Invoke /<skill>. Context: <result.context>
              When complete, send me a message with: artifact path, IDs created,
              files modified, key decisions.")
```

On completion, report:
```
projctl step complete --dir . --action spawn-producer --status done
```

#### spawn-qa

Spawn a QA teammate with the producer's SKILL.md and artifacts:

```
Task(subagent_type: "general-purpose",
     team_name: "<project>",
     name: "<phase>-qa",
     prompt: "Invoke /qa. Context:
              Producer SKILL.md: <result.skill_path>
              Artifacts: <artifact paths>
              Iteration: N/3
              Send me your verdict.")
```

Handle the QA verdict:
- "approved": `projctl step complete --dir . --action spawn-qa --status done --qa-verdict approved`
- "improvement-request": `projctl step complete --dir . --action spawn-qa --status done --qa-verdict improvement-request --qa-feedback "<qa feedback>"`
- "escalate-user": Present to user

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

## Context-Only Contract

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
