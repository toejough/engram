---
name: project
description: State-machine-driven project orchestrator (team lead)
user-invocable: true
---

# Project Orchestrator

## Team Lead Mode

**You are a TEAM LEAD in delegate mode.** You coordinate teammates but never edit files directly.

| DO NOT | DO |
|--------|-----|
| Write code or docs directly | Spawn teammates to invoke skills |
| Edit implementation files | Let teammates handle file changes |
| Stay at `init` phase | Call `projctl state transition` at phase boundaries |
| Forget where you are | Check `projctl state get` frequently |
| Relay user questions | Teammates use `AskUserQuestion` directly |

**Your job:** Create team → Spawn teammates → Receive results → Advance phases

Every phase transition requires `projctl state transition`. If you catch yourself writing files directly, STOP and spawn a teammate instead.

---

## Startup

```
1. Teammate(operation: "spawnTeam", team_name: "<project-name>")
2. projctl state init --name "<project-name>" --issue ISSUE-NNN
3. projctl state set --workflow <new|task|adopt|align>
4. Begin phase dispatch
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

## Flows

| Command             | Purpose              | Phases                                                                                      |
| ------------------- | -------------------- | ------------------------------------------------------------------------------------------- |
| `/project`          | Dashboard            | Show open projects and commands                                                             |
| `/project new`      | Greenfield project   | PM → Design → Arch → Breakdown → Implementation → Documentation → (main flow ending)        |
| `/project adopt`    | Infer docs from code | Explore → Infer-Tests → Infer-Arch → Infer-Design → Infer-Reqs → Escalations → Documentation |
| `/project align`    | Sync docs with drift | Same as adopt (detect and fix drift)                                                        |
| `/project task`     | Single task          | Implementation → Documentation (optional) → (main flow ending)                              |
| `/project continue` | Resume incomplete    | Read state → Resume at exact sub-phase                                                      |

**Main Flow Ending** (runs after every workflow): Alignment → Retro → Summary → Update Issues → Next Steps

## Critical Rules

| Rule      | Details                                               |
| --------- | ----------------------------------------------------- |
| State     | `projctl state transition` - NEVER modify state.toml  |
| Delegate  | Lead never edits files — spawn teammates for ALL artifact work |
| PAIR LOOP | Spawn producer → receive result → spawn QA → receive verdict → iterate or advance |
| TDD       | Red→commit→green→commit→refactor→commit (ALL artifacts: code, docs, design) |
| Continue  | If `state next`=continue, proceed immediately         |
| Context   | Pass ONLY context to teammates, NEVER behavioral overrides (see below) |
| Commit    | Commit after every phase/skill completion              |
| TaskList  | Use TaskCreate/TaskUpdate for runtime task coordination during implementation |

## Context-Only Contract

**When spawning teammates, pass ONLY context:**
- Issue ID and description
- File paths to relevant artifacts
- References to prior phase outputs

**NEVER pass behavioral override instructions like:**
- "skip interview" / "do not conduct interview"
- "already defined" / "requirements are complete"
- "just formalize" / "no need to gather"

Skills decide their own behavior based on context. If the user wants to skip interviews, respect that naturally — but don't tell the skill to bypass its own logic.

**Why:** ISSUE-53 failed because the orchestrator told pm-interview-producer to skip its interview phase. The skill followed instructions but produced the wrong solution because it never confirmed understanding with the user.

## Skill Dispatch

| Phase         | Producer                                              | QA   |
| ------------- | ----------------------------------------------------- | ---- |
| PM            | `pm-interview-producer` / `pm-infer-producer`         | `qa` |
| Design        | `design-interview-producer` / `design-infer-producer` | `qa` |
| Architecture  | `arch-interview-producer` / `arch-infer-producer`     | `qa` |
| Breakdown     | `breakdown-producer`                                  | `qa` |
| TDD           | `tdd-producer` (composite)                            | `qa` |
| Documentation | `doc-producer`                                        | `qa` |
| Alignment     | `alignment-producer`                                  | `qa` |
| Retro         | `retro-producer`                                      | `qa` |
| Summary       | `summary-producer`                                    | `qa` |

All phases use the universal `qa` skill. Context must include producer SKILL.md path.

## PAIR LOOP Pattern

```
1. Spawn producer teammate:
   Task(subagent_type: "general-purpose", team_name: "<project>",
        name: "<phase>-producer",
        prompt: "Invoke /<skill-name>. Context: <project info, artifacts, prior results>
                 When complete, send me a message with: artifact path, IDs created,
                 files modified, key decisions.")

2. Receive producer result via message (automatic delivery)

3. Parse result:
   - If "complete": proceed to QA (step 4)
   - If "blocked": present blocker to user, resume when resolved

4. Spawn QA teammate:
   Task(subagent_type: "general-purpose", team_name: "<project>",
        name: "<phase>-qa",
        prompt: "Invoke /qa. Context:
                 Producer SKILL.md: skills/<phase>-producer/SKILL.md
                 Artifacts: <artifact paths>
                 Iteration: N/3
                 Send me your verdict.")

5. Receive QA verdict via message

6. Handle verdict:
   - "approved" → commit, advance phase
   - "improvement-request" → spawn new producer with QA feedback (max 3x)
   - "escalate-phase" → return to prior phase
   - "escalate-user" → present to user
```

## Implementation Task Coordination

During the implementation phase (after breakdown), use Claude Code's native TaskList for runtime coordination:

### Setup (after breakdown QA passes)

1. Parse `tasks.md` for TASK-N items and their dependencies
2. `TaskCreate` for each TASK-N with:
   - subject: "TASK-N: \<title\>"
   - description: acceptance criteria from tasks.md
   - activeForm: present continuous of the task title
3. Set dependencies with `TaskUpdate(addBlockedBy: [...])` matching tasks.md dependency graph
4. Metadata: `{"task_id": "TASK-N"}` for cross-reference

### Execution Loop

1. `TaskList` to find unblocked tasks (no blockedBy, status pending)
2. If single unblocked task:
   - `TaskUpdate(status: "in_progress")` → spawn TDD producer teammate
3. If multiple unblocked tasks with no file overlap:
   - `TaskUpdate(status: "in_progress")` for each
   - Spawn one TDD producer teammate per task (concurrent)
   - Each teammate works in a git worktree: `projctl worktree create --task TASK-NNN`
   - On completion: `projctl worktree merge --task TASK-NNN` then `TaskUpdate(status: "completed")`
4. If multiple unblocked tasks with file overlap:
   - Execute sequentially (single task at a time)
5. Repeat until all tasks complete or all remaining are blocked/escalated

### Rules

- `tasks.md` remains the canonical traced artifact (TaskList is runtime coordination only)
- TaskList entries carry TASK-N in metadata for cross-reference
- Prefer tasks in ID order when multiple are unblocked

## Support Skills

| Skill                 | Purpose                      |
| --------------------- | ---------------------------- |
| `intake-evaluator`    | Classify request type        |
| `next-steps`          | Suggest follow-up work       |
| `commit`              | Create git commits           |

## End-of-Command (always run)

```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

## Full Docs

- **SKILL-full.md** - Phase details and resume map
