---
name: project
description: State-machine-driven project orchestrator
user-invocable: true
---

# Project Orchestrator

## ⚠️ ORCHESTRATION MODE

**You are an ORCHESTRATOR, not an executor.** For the duration of this project:

| DO NOT | DO |
|--------|-----|
| Write code directly | Dispatch skills (`tdd-producer`, etc.) |
| Edit implementation files | Let skills handle file changes |
| Stay at `init` phase | Call `projctl state transition` at phase boundaries |
| Forget where you are | Check `projctl state get` frequently |

**Your job:** Track state → Dispatch skills → Handle yields → Advance phases

Every phase transition requires `projctl state transition`. If you catch yourself writing implementation code directly, STOP and dispatch the appropriate skill instead.

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
| Log       | `projctl log write` for logging                       |
| Handoffs  | `projctl context write/read` with `output.yield_path` |
| PAIR LOOP | Producer → QA → iterate (max 3x) or advance           |
| TDD       | Red→commit→green→commit→refactor→commit (ALL artifacts: code, docs, design) |
| Continue  | If `state next`=continue, proceed immediately         |
| Dispatch  | ALL artifact work via Skill/Task tool (code, docs, .pen files) |
| Parallel  | Merge-on-complete (not batch at end) - see SKILL-full.md |
| Context   | Pass ONLY context to skills, NEVER behavioral overrides (see below) |

## Context-Only Contract

**When dispatching skills, pass ONLY context:**
- Issue ID and description
- File paths to relevant artifacts
- References to prior phase outputs

**NEVER pass behavioral override instructions like:**
- "skip interview" / "do not conduct interview"
- "already defined" / "requirements are complete"
- "just formalize" / "no need to gather"

Skills decide their own behavior based on context. If the user wants to skip interviews, respect that naturally - but don't tell the skill to bypass its own logic.

**Why:** ISSUE-053 failed because the orchestrator told pm-interview-producer to skip its interview phase. The skill followed instructions but produced the wrong solution because it never confirmed understanding with the user.

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

All phases use the universal `qa` skill. Context must include producer metadata.

## PAIR LOOP Pattern

```
1. Write context with output.yield_path
2. Dispatch PRODUCER skill
3. Read yield
4. If yield.type = "need-user-input" AND payload.inferred = true:
   a. Present inferred items as numbered accept/reject list with reasoning
   b. User responds: "accept all", "reject all", or per-item (e.g., "accept 1, reject 2")
   c. Write decisions to [query_results.inferred_decisions]
   d. Resume producer with decisions
   e. Go to step 3
5. If yield.type = "complete": write QA context and dispatch `qa` skill
6. Handle QA yield:
   - "approved" → advance
   - "improvement-request" → resume producer (max 3x)
   - "escalate-phase" → return to prior phase
   - "escalate-user" → present to user
```

### QA Context Schema

When dispatching the universal `qa` skill, write context with:

```toml
[inputs]
producer_skill_path = "skills/<phase>-producer/SKILL.md"
producer_yield_path = ".projctl/yields/<phase>-producer-yield.toml"
artifact_paths = ["docs/<artifact>.md"]

[context]
iteration = 1
max_iterations = 3
```

## Yield Types

| Type                  | Action                              |
| --------------------- | ----------------------------------- |
| `complete`            | Advance to QA or next phase         |
| `approved`            | Phase complete, advance             |
| `need-user-input`     | Prompt user, resume with answer (see below for inferred) |
| `need-user-input` (inferred) | Present inferred items for accept/reject, resume with decisions |
| `need-context`        | Dispatch `context-explorer`, resume |
| `need-decision`       | Present options, resume with choice |
| `improvement-request` | Resume producer with feedback       |
| `escalate-phase`      | Return to prior phase               |
| `escalate-user`       | Present to user                     |
| `blocked`             | Present blocker, await resolution   |
| `error`               | Retry (max 3x) or escalate          |

## Support Skills

| Skill                 | Purpose                      |
| --------------------- | ---------------------------- |
| `context-explorer`    | Handle need-context queries  |
| `intake-evaluator`    | Classify request type        |
| `parallel-looper`     | Run N PAIR LOOPs in parallel |
| `consistency-checker` | Validate parallel outputs    |
| `next-steps`          | Suggest follow-up work       |

## End-of-Command (always run)

```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

## Full Docs

- **SKILL-full.md** - Phase details and resume map
- **../shared/YIELD.md** - Yield protocol format and examples
