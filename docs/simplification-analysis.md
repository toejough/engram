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

## Vision (from User)

### Core Desires

1. **Simple, generic process** - works across API, docs, design, GUI, TUI, CLI, code, deployment, testing
2. **Scalable orchestration** - top-level agent orchestrates calls, not decisions; sub-agents can yield needs for other sub-agents
3. **Learning** - capture and apply learnings within and across projects
4. **Role-based agents** mimicking modern human engineering teams

### Role-Based Agent Model

| Role | Responsibility |
|------|----------------|
| **Product Manager** | Problem-space discovery |
| **Designer** | User-experience solution-space discovery |
| **Architect** | Code solution-space discovery |
| **Task Breakdown** | Decompose arch into implementable units |
| **Implementer** | TDD: test → implement |
| **QA** | Audit and verify |
| **Retro** | Reflect on success/struggles per-part and process-wide |
| **Project Manager** | Thread it all together, keep on rails |
| **Tech Writer** | Keep documentation up to date |

### Traceability Chain

```
(issue) → requirement → (design) → arch → (task) → test → implementation
```

Parentheses = optional depending on context:
- Not every work item starts from an issue
- Not everything needs design (pure backend, CLI)
- Tasks are derived, not always explicit

Audits and corrections fill gaps when traceability breaks.

### Co-routine Model

**Key insight:** Top-level orchestrator should orchestrate *calls*, not make *decisions*.

**Flow:**
```
PM spawns product-manager
  product-manager works...
  product-manager yields: {need: "architect-consult", question: "REST or GraphQL?"}
PM spawns architect with question
  architect responds: "GraphQL because..."
PM passes response back to product-manager
  product-manager continues with answer
  product-manager completes: {result: requirements.md}
PM receives result, continues to next phase
```

**Benefits:**
- Sub-agents can request other sub-agents without nesting
- Orchestrator remains thin (just message passing)
- Each agent focuses on its role
- User interaction happens at orchestrator level (not buried in sub-agents)

---

## Current vs Vision Comparison

| Aspect | Current | Vision |
|--------|---------|--------|
| Orchestrator | LLM runs control loop + dispatches | Thin message-passer only |
| Sub-agent communication | Fire-and-forget | Yield/resume with needs |
| User interaction | Broken (sub-agents can't ask) | Surfaced to orchestrator level |
| Roles | Skills (20+) | Agents (9 roles) |
| Traceability | REQ→DES→ARCH→TASK | issue→req→(des)→arch→(task)→test→impl |
| Learning | corrections log, meta-audit | Active cross-project memory |

---

## Simplification Path

### Step 1: Consolidate Skills into Roles

Map 20+ skills to 9 roles:

| Role | Current Skills |
|------|----------------|
| Product Manager | pm-interview, pm-infer, pm-audit |
| Designer | design-interview, design-infer, design-audit |
| Architect | architect-interview, architect-infer, architect-audit |
| Task Breakdown | task-breakdown |
| Implementer | tdd-red, tdd-green, tdd-refactor |
| QA | task-audit, alignment-check |
| Retro | meta-audit |
| Project Manager | project (orchestrator) |
| Tech Writer | (new - or integrate docs into other roles) |

### Step 2: Define Yield Protocol

Sub-agents communicate via structured yields:

```toml
# Sub-agent needs something
[yield]
type = "need-consult"  # or "need-user-input", "need-sub-agent", "complete"
target = "architect"   # which role to consult
question = "Should we use REST or GraphQL for the API?"
context = "Building user service, need to decide API style"

# Sub-agent completes
[yield]
type = "complete"
result = "requirements.md created with REQ-001 through REQ-005"
files_modified = ["docs/requirements.md"]
```

### Step 3: Simplify State Machine

Current: ~15 phases with complex transitions
Proposed: Role-based phases

```
discovery (product-manager + designer + architect)
  → breakdown (task-breakdown)
  → implement (implementer per task, with QA)
  → retro (per task and overall)
  → done
```

### Step 4: Learning Integration

Every agent interaction captures:
- Decisions made (choice + reason + alternatives)
- Corrections received
- What worked / what struggled

`projctl memory` stores cross-project, `projctl retro` synthesizes.

---

## Open Questions

1. **Yield implementation:** How does a sub-agent "yield" in Claude Code? Options:
   - Structured output format parsed by orchestrator
   - Exit with specific status + state file
   - Stream markers in JSONL output

2. **Role vs skill granularity:** Should "product-manager" be one agent that does interview/infer/audit, or separate?

3. **Optional phases:** How does orchestrator know to skip design for CLI work? User declares? Auto-detect?

4. **Tech writer timing:** After each phase? Only at end? On-demand?

5. **Cross-agent memory:** How do agents access learnings from other agents/projects?

---

## Next Steps

1. Define yield protocol concretely
2. Prototype thin orchestrator with one role (e.g., product-manager)
3. Test yield/resume flow with user interaction
4. Expand to full role set if prototype works

---
