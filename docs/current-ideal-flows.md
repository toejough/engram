# Current Ideal Process Flows

This document captures the intended workflows for `/project new`, `/project adopt`, and `/project align`, including all skills and projctl commands at each step.

**Source:** Synthesized from `review-2025-01.md` (git history), full skill docs, and projctl command structure.

---

## Key Learnings Applied

From `review-2025-01.md` and Vercel research:

| Learning | Implication |
|----------|-------------|
| Agents only invoke skills 44% of the time | Don't rely on agent calling projctl commands |
| Passive context beats skills (100% vs 53%) | Critical rules in CLAUDE.md, not just skills |
| Sub-agents run to completion | No mid-execution user interaction possible |
| Memory/territory must be injected | Orchestrator does it, not agent judgment |
| Corrections detection must be deterministic | Orchestrator parses user messages (step 0) |
| Relentless continuation | Don't stop unless legitimate blocker |

---

## Control Loop (All Workflows)

Every workflow uses this deterministic control loop:

```
┌─────────────────────────────────────────────────────────────────┐
│                     CONTROL LOOP                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  0. [A] Detect corrections in user message                       │
│         → if found: projctl corrections log                      │
│                                                                  │
│  1. [D] projctl state get --dir .                                │
│         → current phase, task, subphase                          │
│                                                                  │
│  2. [D] projctl state transition --dir . --to <next>             │
│         → validates preconditions, updates state                 │
│                                                                  │
│  3. [D] projctl map --cached                                     │
│         → territory map for context injection                    │
│                                                                  │
│  4. [A] Dispatch skill (Skill tool or inline)                    │
│         ┌─────────────────────────────────────────┐              │
│         │ AGENTIC BLACK BOX                       │              │
│         │ Input: context + territory + memories   │              │
│         │ Output: result.toml with decisions      │              │
│         └─────────────────────────────────────────┘              │
│                                                                  │
│  5. [D] projctl context read --result                            │
│         → parse skill output                                     │
│                                                                  │
│  6. [D] projctl corrections count --session                      │
│         → if >= 2: dispatch /meta-audit                          │
│                                                                  │
│  7. [D] projctl state next --dir .                               │
│         → returns {action: continue|stop, reason: ...}           │
│                                                                  │
│  8. [A] If continue: GOTO 1                                      │
│         If stop: present reason to user                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

[D] = Deterministic (projctl CLI)
[A] = Agentic (LLM reasoning)
```

---

## Flow 1: `/project new` (Greenfield)

New project from scratch with full interview cycle.

```
┌─────────────────────────────────────────────────────────────────┐
│                      /project new <name>                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: init                                                      │
├─────────────────────────────────────────────────────────────────┤
│ projctl state init --name <name> --dir <project-dir>             │
│ projctl log write --level phase --subject init                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: pm-interview                                              │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to pm-interview                       │
│ projctl context write --skill pm-interview                       │
│                                                                  │
│ SKILL: /pm-interview                                             │
│   Phases: PROBLEM → CURRENT STATE → FUTURE STATE →               │
│           SUCCESS CRITERIA → EDGE CASES                          │
│   Output: docs/requirements.md with REQ-NNN IDs                  │
│   Needs: USER INTERACTION (questions/answers)                    │
│                                                                  │
│ projctl context read --result                                    │
│ projctl state transition --to pm-complete                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: design-interview                                          │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to design-interview                   │
│ projctl context write --skill design-interview                   │
│                                                                  │
│ SKILL: /design-interview                                         │
│   Phases: UNDERSTAND spec → PREFERENCES → DESIGN SYSTEM → BUILD  │
│   Output: docs/design.md with DES-NNN IDs                        │
│           .pen files (if UI work)                                │
│   Needs: USER INTERACTION (preferences)                          │
│                                                                  │
│ projctl context read --result                                    │
│ projctl state transition --to design-complete                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: alignment-check (post-design)                             │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to alignment-check                    │
│ projctl trace validate --dir .                                   │
│                                                                  │
│ If validation fails:                                             │
│   SKILL: /alignment-check                                        │
│   → interpret gaps, propose fixes                                │
│   → apply fixes to **Traces to:** fields                         │
│   → re-run projctl trace validate                                │
│   Max 2 iterations, then escalate                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: architect-interview                                       │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to architect-interview                │
│ projctl context write --skill architect-interview                │
│                                                                  │
│ SKILL: /architect-interview                                      │
│   Phases: UNDERSTAND reqs → RESEARCH → INTERVIEW → SPECIFY       │
│   Output: docs/architecture.md with ARCH-NNN IDs                 │
│   Needs: USER INTERACTION (technology preferences)               │
│                                                                  │
│ projctl context read --result                                    │
│ projctl state transition --to architect-complete                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: alignment-check (post-architect)                          │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to alignment-check                    │
│ projctl trace validate --dir .                                   │
│ (same as post-design alignment)                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: task-breakdown                                            │
├─────────────────────────────────────────────────────────────────┤
│ projctl state transition --to task-breakdown                     │
│ projctl context write --skill task-breakdown                     │
│                                                                  │
│ SKILL: /task-breakdown                                           │
│   Input: requirements.md, design.md, architecture.md             │
│   Output: docs/tasks.md with TASK-NNN IDs                        │
│           Each task has: AC, files, dependencies, traces         │
│   NO user interaction needed                                     │
│                                                                  │
│ projctl context read --result                                    │
│ projctl state transition --to tasks-complete                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: implementation (loop over tasks)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│ FOR EACH unblocked TASK-NNN:                                     │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ SUBPHASE: tdd-red                                        │   │
│   ├─────────────────────────────────────────────────────────┤   │
│   │ projctl state transition --to tdd-red --task TASK-NNN    │   │
│   │ projctl context write --skill tdd-red --task TASK-NNN    │   │
│   │                                                          │   │
│   │ SKILL: /tdd-red                                          │   │
│   │   Write failing tests covering ALL acceptance criteria   │   │
│   │   Tests MUST fail                                        │   │
│   │   NO implementation code                                 │   │
│   │                                                          │   │
│   │ projctl context read --result                            │   │
│   │ projctl state transition --to commit-red                 │   │
│   │                                                          │   │
│   │ SKILL: /commit                                           │   │
│   │   Commit: "test(TASK-NNN): add failing tests for..."     │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ SUBPHASE: tdd-green                                      │   │
│   ├─────────────────────────────────────────────────────────┤   │
│   │ projctl state transition --to tdd-green                  │   │
│   │ projctl context write --skill tdd-green --task TASK-NNN  │   │
│   │                                                          │   │
│   │ SKILL: /tdd-green                                        │   │
│   │   Write MINIMAL code to make tests pass                  │   │
│   │   ALL tests must pass                                    │   │
│   │   NO refactoring yet                                     │   │
│   │                                                          │   │
│   │ projctl context read --result                            │   │
│   │ projctl state transition --to commit-green               │   │
│   │                                                          │   │
│   │ SKILL: /commit                                           │   │
│   │   Commit: "feat(TASK-NNN): implement..."                 │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ SUBPHASE: tdd-refactor                                   │   │
│   ├─────────────────────────────────────────────────────────┤   │
│   │ projctl state transition --to tdd-refactor               │   │
│   │ projctl context write --skill tdd-refactor               │   │
│   │                                                          │   │
│   │ SKILL: /tdd-refactor                                     │   │
│   │   Run linter, fix issues                                 │   │
│   │   Improve code quality                                   │   │
│   │   Tests MUST stay green                                  │   │
│   │                                                          │   │
│   │ projctl context read --result                            │   │
│   │ projctl state transition --to commit-refactor            │   │
│   │                                                          │   │
│   │ SKILL: /commit (if changes)                              │   │
│   │   Commit: "refactor(TASK-NNN): improve..."               │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ SUBPHASE: task-audit                                     │   │
│   ├─────────────────────────────────────────────────────────┤   │
│   │ projctl state transition --to task-audit                 │   │
│   │ projctl context write --skill task-audit                 │   │
│   │                                                          │   │
│   │ SKILL: /task-audit                                       │   │
│   │   Verify ALL acceptance criteria met                     │   │
│   │   Check TDD discipline (no test weakening)               │   │
│   │   Check linter compliance                                │   │
│   │                                                          │   │
│   │ If audit fails:                                          │   │
│   │   → Loop back to appropriate TDD phase                   │   │
│   │   → Fix issues, re-audit                                 │   │
│   │                                                          │   │
│   │ If audit passes:                                         │   │
│   │   projctl state transition --to task-complete            │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│   projctl state next → if more unblocked tasks, continue        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: completion                                                │
├─────────────────────────────────────────────────────────────────┤
│ projctl integrate features --dir .                               │
│ projctl trace repair --dir .                                     │
│ projctl trace validate --dir .                                   │
│                                                                  │
│ If validation fails: present options to user                     │
│ If validation passes: present summary, done                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Flow 2: `/project adopt` (Existing Codebase)

Adopt existing codebase by inferring artifacts.

```
┌─────────────────────────────────────────────────────────────────┐
│                     /project adopt                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: init                                                      │
├─────────────────────────────────────────────────────────────────┤
│ projctl state init --name <name> --dir .                         │
│ projctl map --dir . (initial territory mapping)                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: analysis                                                  │
├─────────────────────────────────────────────────────────────────┤
│ Analyze existing docs and code structure                         │
│ Identify what artifacts exist vs need inference                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: pm-infer                                                  │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /pm-infer (can run in parallel with others)               │
│   Analyze: README, tests, CLI help, API docs                     │
│   Output: docs/requirements.md with REQ-NNN IDs                  │
│   NO user interaction (inference from code)                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: design-infer                                              │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /design-infer (can run in parallel)                       │
│   Analyze: UI code, CLI output, error messages                   │
│   Output: docs/design.md with DES-NNN IDs                        │
│   NO user interaction                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: architect-infer                                           │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /architect-infer (can run in parallel)                    │
│   Analyze: package structure, dependencies, patterns             │
│   Output: docs/architecture.md with ARCH-NNN IDs                 │
│   NO user interaction                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: test-mapper                                               │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /test-mapper                                              │
│   Scan existing tests                                            │
│   Map tests to TASK-NNN / REQ-NNN IDs                            │
│   Output: TEST-NNN annotations in tasks.md                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: escalation-resolution                                     │
├─────────────────────────────────────────────────────────────────┤
│ Batch present all escalations (ESC-NNN) from infer phases        │
│ User resolves ambiguities                                        │
│ Update artifacts with user decisions                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: traceability-generation                                   │
├─────────────────────────────────────────────────────────────────┤
│ projctl trace repair --dir .                                     │
│ projctl trace validate --dir .                                   │
│ Fill gaps in **Traces to:** fields                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: adopt-complete                                            │
├─────────────────────────────────────────────────────────────────┤
│ projctl integrate features --dir .                               │
│ projctl trace validate --dir .                                   │
│ Present summary of adopted artifacts                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Flow 3: `/project align` (Sync Existing Docs)

Lightweight alignment check for existing project.

```
┌─────────────────────────────────────────────────────────────────┐
│                     /project align                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: align-analyze                                             │
├─────────────────────────────────────────────────────────────────┤
│ Compare code state to documented state                           │
│ Identify drift in: requirements, design, architecture            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: normalize-formats                                         │
├─────────────────────────────────────────────────────────────────┤
│ Ensure ID formats are consistent (REQ-NNN, DES-NNN, etc.)        │
│ Ensure **Traces to:** fields exist                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: backfill-missing                                          │
├─────────────────────────────────────────────────────────────────┤
│ If artifacts missing, run appropriate infer skill                │
│ (pm-infer, design-infer, architect-infer as needed)              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: update-infer                                              │
├─────────────────────────────────────────────────────────────────┤
│ Run infer skills in UPDATE mode (not replace)                    │
│ Add new IDs for undocumented features                            │
│ Mark deprecated IDs                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: test-map                                                  │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /test-mapper                                              │
│ Update TEST-NNN mappings                                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: fill-gaps                                                 │
├─────────────────────────────────────────────────────────────────┤
│ SKILL: /alignment-check                                          │
│ Fill remaining traceability gaps                                 │
│ projctl trace repair --dir .                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE: align-complete                                            │
├─────────────────────────────────────────────────────────────────┤
│ projctl trace validate --dir .                                   │
│ Present alignment summary                                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Skills by Interaction Requirement

| Skill | User Interaction | Can Run as Sub-agent |
|-------|------------------|---------------------|
| /pm-interview | **YES** - questions | NO - needs yields |
| /design-interview | **YES** - preferences | NO - needs yields |
| /architect-interview | **YES** - tech choices | NO - needs yields |
| /pm-infer | No | YES |
| /design-infer | No | YES |
| /architect-infer | No | YES |
| /task-breakdown | No | YES |
| /tdd-red | No | YES |
| /tdd-green | No | YES |
| /tdd-refactor | No | YES |
| /task-audit | No | YES |
| /pm-audit | No | YES |
| /design-audit | No | YES |
| /architect-audit | No | YES |
| /alignment-check | No | YES |
| /meta-audit | No | YES |
| /test-mapper | No | YES |
| /negotiate | No | YES |
| /commit | No | YES |

**Key insight:** Only 3 of 19 skills require user interaction. These are the interview skills.

---

## projctl Commands Used

| Command | Purpose | When Used |
|---------|---------|-----------|
| `state init` | Create project state | Start of workflow |
| `state get` | Read current phase/task | Every loop iteration |
| `state transition` | Move to next phase | After each step |
| `state next` | Determine continue/stop | After each step |
| `context write` | Prepare skill input | Before skill dispatch |
| `context read --result` | Parse skill output | After skill returns |
| `map --cached` | Territory for context | Before skill dispatch |
| `trace validate` | Check traceability | After artifact phases |
| `trace repair` | Fix traceability gaps | Before validate |
| `integrate features` | Merge project docs | End of workflow |
| `corrections log` | Track corrections | On user correction |
| `corrections count` | Check threshold | After each step |
| `log write` | Structured logging | Throughout |

---

## Traceability Chain

```
ISSUE-NNN (optional starting point)
    │
    ▼
REQ-NNN (requirements.md)
    │
    ├─────────────────┐
    ▼                 ▼
DES-NNN           ARCH-NNN
(design.md)       (architecture.md)
    │                 │
    └────────┬────────┘
             ▼
         TASK-NNN
         (tasks.md)
             │
             ▼
         TEST-NNN
         (in test files)
             │
             ▼
         Implementation
         (code files)
```

Each ID has `**Traces to:**` field linking to upstream IDs.
Validation ensures no orphan or unlinked IDs.

---

## What's Missing / Broken

From our experiments and the review:

| Issue | Root Cause | Impact |
|-------|------------|--------|
| Sub-agents can't ask users questions | Claude Code architecture | Interview skills fail when dispatched |
| Orchestrator skips steps | LLM takes shortcuts | Process not followed |
| Context pollution | LLM runs deterministic steps | Control loop degrades |
| Memory not used | Agent must call CLI | Learnings lost |
| Corrections not logged | Agent must detect | No learning loop |

**All of these trace to:** Relying on agent behavior for critical operations.

---
