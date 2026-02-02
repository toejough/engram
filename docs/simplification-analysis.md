# Simplification Analysis

Analysis of the current `/project` orchestration system to identify simplification opportunities.

---

## Current System Overview

### Core Intent

**Goal:** Autonomous project delivery from idea to implementation with:
- Structured interviews → artifacts → tasks → TDD → audit
- Traceability from requirements through to code
- Correction detection and learning
- Minimal human intervention

---

## Components

### 1. Orchestrator Skill (`/project`)

**Purpose:** Manage project lifecycle through state machine phases.

**Current Implementation:**
- 68-line compressed SKILL.md
- 986-line full documentation
- LLM-driven control loop
- Dispatches sub-skills via Skill tool

**Control Loop (10 steps):**
| Step | Type | Action |
|------|------|--------|
| 0 | [A] | Detect corrections |
| 1 | [D] | `projctl state get` |
| 2 | [D] | `projctl state transition` |
| 3 | [D] | `projctl map --cached` |
| 4 | [A] | Dispatch skill |
| 5 | [D] | `projctl context read --result` |
| 6 | [D] | `projctl corrections count` |
| 7 | [D] | `projctl state next` |
| 8 | [A] | If corrections >= 2: /meta-audit |
| 9 | [A] | If continue: loop |

**Key Issue:** Steps marked [D] are deterministic but still LLM-executed, polluting context.

---

### 2. Interview Skills

| Skill | Output | ID Format |
|-------|--------|-----------|
| `/pm-interview` | requirements.md | REQ-NNN |
| `/design-interview` | design.md + .pen | DES-NNN |
| `/architect-interview` | architecture.md | ARCH-NNN |

**Phases:** Each follows interview structure:
- PM: PROBLEM → CURRENT STATE → FUTURE STATE → SUCCESS CRITERIA → EDGE CASES
- Design: UNDERSTAND → PREFERENCES → DESIGN SYSTEM → BUILD
- Architect: UNDERSTAND → RESEARCH → INTERVIEW → SPECIFY

**Key Issue:** When dispatched to sub-agents, questions asked but user can't respond.

---

### 3. TDD Skills

| Skill | Purpose | Rules |
|-------|---------|-------|
| `/tdd-red` | Write failing tests | Tests ONLY, MUST fail, cover ALL criteria |
| `/tdd-green` | Minimal implementation | MINIMAL code, ALL tests pass, NO refactoring |
| `/tdd-refactor` | Improve code quality | Tests STAY GREEN, fix linter issues |

**Commit Cycle:** red → commit → green → commit → refactor → commit

**Key Issue:** Works well as sub-agents since no user interaction needed.

---

### 4. Audit Skills

| Skill | Purpose |
|-------|---------|
| `/task-audit` | Verify AC met, TDD discipline followed |
| `/pm-audit` | Validate against requirements |
| `/design-audit` | Validate visual output |
| `/architect-audit` | Validate against architecture |
| `/meta-audit` | Analyze correction patterns |

**Key Issue:** Audits loop until pass, which is good for enforcement.

---

### 5. Support Skills

| Skill | Purpose |
|-------|---------|
| `/task-breakdown` | Decompose ARCH into TASK-NNN |
| `/alignment-check` | Verify traceability coverage |
| `/negotiate` | Resolve cross-skill conflicts |
| `/commit` | Create well-formatted commits |

---

### 6. CLI Commands (`projctl`)

| Command | Purpose |
|---------|---------|
| `state get/transition/next` | State machine control |
| `context write/read` | Skill handoffs |
| `log write` | Structured logging |
| `trace validate/repair` | Traceability checking |
| `map --cached` | Territory mapping |
| `corrections log/count` | Correction tracking |
| `integrate features` | Merge project docs |
| `escalation list/resolve` | Escalation management |

---

### 7. Tracking Files

| File | Purpose |
|------|---------|
| `state.toml` | Current phase, task, history |
| `context/*.toml` | Skill input/output handoffs |
| `result.toml` | Skill completion status |
| `docs/requirements.md` | REQ-NNN definitions |
| `docs/design.md` | DES-NNN definitions |
| `docs/architecture.md` | ARCH-NNN definitions |
| `docs/tasks.md` | TASK-NNN definitions |

---

### 8. ID Systems

| Prefix | Artifact | Contains |
|--------|----------|----------|
| REQ-NNN | requirements.md | User stories, acceptance criteria |
| DES-NNN | design.md | UI/UX specifications |
| ARCH-NNN | architecture.md | Technology decisions |
| TASK-NNN | tasks.md | Implementation tasks with AC |
| ESC-NNN | escalations | Questions needing resolution |

**Traceability:** Each ID has `**Traces to:**` field linking to upstream IDs.

---

## State Machine Phases

### `/project new` Flow
```
init
  → pm-interview → pm-complete
  → design-interview → design-complete → alignment-check
  → architect-interview → architect-complete → alignment-check
  → task-breakdown → tasks-complete
  → implementation
    → task-start
      → tdd-red → commit-red
      → tdd-green → commit-green
      → tdd-refactor → commit-refactor
      → task-audit → task-complete
    → (repeat for each task)
  → audit-loop
  → completion
```

### Key Transitions
- Interview phases require user interaction
- TDD phases are autonomous
- Audit phases loop until pass
- `projctl state next` determines continue/stop

---

## Known Problems

### 1. Sub-Agent Interaction Failure

**Evidence:** Session log agent-a3aba56 shows pm-interview sub-agent asked correct questions but couldn't receive answers.

**Root Cause:** Sub-agents dispatched via Skill tool run to completion without user interaction capability.

**Current Workaround:** Orchestrator passes context as pseudo-answers (wrong).

### 2. Context Pollution

**Problem:** LLM executes deterministic CLI commands, filling context with irrelevant output.

**Impact:** Control loop instructions degrade in attention over time.

### 3. Process Skipping

**Problem:** When LLM doesn't get expected answers from tools, it compensates by skipping steps.

**Evidence:** PM interview skipped entirely, requirements auto-generated from issue text.

### 4. Skill Complexity

**Observation:** SKILL.md files are already compressed (40-70 lines). Full docs are 500-1000 lines.

**Question:** Is the problem instruction length or something else?

---

## Simplification Hypotheses

### H1: Run Interactive Skills Inline

**Idea:** Don't dispatch interview skills to sub-agents. Run them directly in main conversation.

**Pro:** User can actually answer questions.
**Con:** Pollutes main context with interview content.

### H2: Co-routine Style Flow

**Idea:** Skill "yields" when it needs user input, orchestrator captures yield point, resumes with answer.

**Implementation Options:**
- Claude CLI `--resume` for session continuation
- Structured output format indicating "need input"
- State file captures yield point for crash recovery

### H3: Reduce Orchestrator Responsibility

**Idea:** Instead of one orchestrator managing everything, have simpler "do next thing" logic.

**Flow:**
1. Human calls `/pm-interview` directly
2. Skill runs, produces result
3. Human calls `/design-interview` directly
4. ... etc

**Pro:** No orchestrator to skip steps.
**Con:** Loses automation benefit.

### H4: Deterministic Outer Loop

**Idea:** Non-LLM program manages state, invokes LLM only for skill work.

**Pro:** Process guaranteed.
**Con:** Significant implementation effort. (This is ISSUE-001 approach)

### H5: Simpler State Machine

**Idea:** Reduce number of phases/transitions.

**Current:** ~15 distinct phases
**Simplified:** Maybe 5? (interview → breakdown → implement → audit → done)

### H6: Merge Related Skills

**Idea:** Combine pm/design/architect interviews into single "discovery" skill.

**Pro:** Fewer dispatches, less handoff complexity.
**Con:** Loses separation of concerns.

---

## Questions to Explore

1. **What is "co-routine style" specifically?** How would yields/resumes work in practice?

2. **Can we detect question vs completion?** What signal indicates LLM needs user input?

3. **What's the minimal viable orchestration?** If we stripped everything, what's essential?

4. **Where does process actually get skipped?** Is it always at user interaction points?

5. **What would "good enough" look like?** 80% reliable? 95%? 100%?

---

## Next Steps

1. Define what "co-routine based flow" means concretely
2. Identify which skills need user interaction vs which are autonomous
3. Prototype simplest possible orchestration that preserves core value
4. Test whether simplification actually improves reliability

---
