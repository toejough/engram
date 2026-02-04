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

## Skill Dispatch

| Phase         | Producer                                              | QA             |
| ------------- | ----------------------------------------------------- | -------------- |
| PM            | `pm-interview-producer` / `pm-infer-producer`         | `pm-qa`        |
| Design        | `design-interview-producer` / `design-infer-producer` | `design-qa`    |
| Architecture  | `arch-interview-producer` / `arch-infer-producer`     | `arch-qa`      |
| Breakdown     | `breakdown-producer`                                  | `breakdown-qa` |
| TDD           | `tdd-producer` (composite)                            | `tdd-qa`       |
| Documentation | `doc-producer`                                        | `doc-qa`       |
| Alignment     | `alignment-producer`                                  | `alignment-qa` |
| Retro         | `retro-producer`                                      | `retro-qa`     |
| Summary       | `summary-producer`                                    | `summary-qa`   |

## PAIR LOOP Pattern

```
1. Write context with output.yield_path
2. Dispatch PRODUCER skill
3. Read yield
4. If yield.type = "complete": dispatch QA skill
5. Handle QA yield:
   - "approved" → advance
   - "improvement-request" → resume producer (max 3x)
   - "escalate-phase" → return to prior phase
   - "escalate-user" → present to user
```

## Yield Types

| Type                  | Action                              |
| --------------------- | ----------------------------------- |
| `complete`            | Advance to QA or next phase         |
| `approved`            | Phase complete, advance             |
| `need-user-input`     | Prompt user, resume with answer     |
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
