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
**Con:** Significant implementation effort. (This is ISSUE-1 approach)

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

## Yield Protocol Design

### Core Mechanism

Sub-agents serialize their continuation to disk before yielding:

1. Agent works until it needs something (user input, another agent's output)
2. Agent writes relevant context to `context/<role>-state.md`
3. Agent exits with structured yield message
4. Orchestrator handles the need
5. Orchestrator spawns **new** agent with: role instructions + context file + answer

### Yield Message Format

```toml
[yield]
type = "need-user-input"  # or "need-consult", "complete", "need-decision"
question = "What problem are you trying to solve?"
context_file = "context/product-manager-state.md"

# For need-consult
target_role = "architect"
consult_question = "Should we use REST or GraphQL?"

# For complete
result_file = "docs/requirements.md"
files_modified = ["docs/requirements.md"]
```

### Context File Structure

```markdown
# Product Manager Context

## Phase
PROBLEM (1 of 5: PROBLEM → CURRENT STATE → FUTURE STATE → SUCCESS CRITERIA → EDGE CASES)

## Gathered So Far
(User answers from previous questions)

## Current Question
What problem are you trying to solve?

## Remaining Questions
- What's broken today?
- Who is affected?
- What's the impact?

## Decisions Made
- (none yet)

## Learnings
- (captured insights)
```

### Resume Flow

When orchestrator resumes an agent:

```
Orchestrator prompt:
  "You are Product Manager. Resume from the context below.

   User's answer to your last question: <answer>

   Context file:
   <contents of context/product-manager-state.md>

   Continue the interview from where you left off."
```

### Orchestrator State

Orchestrator tracks minimal state:

```toml
[orchestrator]
current_role = "product-manager"
phase = "discovery"

[pending_yield]
role = "product-manager"
type = "need-user-input"
question = "What problem are you trying to solve?"
context_file = "context/product-manager-state.md"
```

### Example Flow

```
1. PM spawns product-manager
2. product-manager asks "What problem?"
   → writes context/product-manager-state.md
   → yields {type: "need-user-input", question: "What problem?"}
3. PM surfaces question to user
4. User answers: "The build is too slow"
5. PM spawns new product-manager with context + answer
6. product-manager reads context, continues
   → asks "How slow is it today?"
   → updates context file
   → yields {type: "need-user-input", question: "How slow?"}
7. (repeat until complete)
8. product-manager yields {type: "complete", result_file: "docs/requirements.md"}
9. PM moves to next role
```

### Nested Agent Calls

When product-manager needs architect input:

```
1. product-manager yields {type: "need-consult", target: "architect", question: "REST or GraphQL?"}
2. PM spawns architect with the question
3. architect responds (may itself yield for user input)
4. PM gets architect's answer
5. PM spawns new product-manager with context + architect's answer
6. product-manager continues
```

---

## Open Questions

1. **Context file size:** How do we keep context files focused? Agent discipline? Token limits?

2. **Role vs skill granularity:** Should "product-manager" be one agent that does interview/infer/audit, or separate?

3. **Optional phases:** How does orchestrator know to skip design for CLI work? User declares? Auto-detect?

4. **Tech writer timing:** After each phase? Only at end? On-demand?

5. **Cross-agent memory:** How do agents access learnings from other agents/projects?

6. **Context file format:** Markdown (human-readable) vs TOML (structured) vs both?

---

---

## Learnings from Tool Review (review-2025-01.md)

### Gastown (Steve Yegge)

| Adopted | Not Adopted |
|---------|-------------|
| External state (not context window) | Massive parallelization (20-30 agents) |
| Graceful degradation | Complex role hierarchy (Mayor/Polecats) |
| Real-time monitoring | ~$100/hour token burn |

**Key pattern:** Git-backed persistent state survives context loss and crashes.

### oh-my-opencode / oh-my-claudecode

| Adopted | Not Adopted |
|---------|-------------|
| Auto model routing (Haiku→Sonnet→Opus) | Vendor-specific integrations |
| Background territory mapping | - |
| Token-aware routing (30-50% cost savings) | - |
| LSP for deterministic refactoring | - |

**Key pattern:** "Sisyphus agent" fires off background tasks to cheaper models to keep main context lean.

### claude-flow

| Adopted | Not Adopted |
|---------|-------------|
| Task routing by complexity | ML-driven routing (Q-learning, MoE) |
| Cost metrics and visibility | Swarm topologies |
| - | Byzantine consensus |

**Key insight:** Simple routing (simple→Haiku, medium→Sonnet, complex→Opus) gets most benefit without ML complexity.

### Vercel Research (January 2025)

| Finding | Implication |
|---------|-------------|
| Skills invoked only 44% of time | Don't rely on agent calling things |
| AGENTS.md: 100% pass rate | Critical rules in passive context |
| Skills (default): 53% pass rate | Skills unreliable for agent-initiated work |
| 80% compression still works | Compressed index + on-demand retrieval |

**Winning instruction:** "Prefer retrieval-led reasoning over pre-training-led reasoning"

---

## Key Patterns to Implement

### 1. Relentless Continuation ("Won't Quit")

From oh-my-opencode's Sisyphus philosophy:

```
Current (bad):
  Done. Created 2 commits.
  No response requested.    ← STOPS

Target (good):
  Done. Created 2 commits for TASK-003.
  Checking next... TASK-004 unblocked.
  Starting TASK-004.        ← CONTINUES
```

**Legitimate stop conditions:**
- All tasks complete
- Escalation needs user decision
- Validation failed after 3 retries
- Ambiguous requirement needs clarification

**NOT stop conditions:**
- Task complete but more exist
- Phase complete but next phase ready
- Commit done (continue to next TDD phase)

### 2. Background Territory Mapping

Before main work, dispatch cheap agent to explore:

```toml
# context/territory.toml (~500 tokens vs ~5000 for exploration)
[structure]
languages = ["go"]
build_tool = "mage"

[entry_points]
cli = "cmd/projctl/main.go"

[packages]
count = 12
internal = ["config", "context", "state", ...]

[tests]
pattern = "*_test.go"
count = 27
```

**When to map:**
- `/project new` - before pm-interview
- `/project adopt` - before inference
- `tdd-red` - before writing tests
- `tdd-green` - before implementation

### 3. Cross-Project Memory with ONNX

```
~/.claude/memory/
├── index.md           # Greppable learnings
├── sessions/          # Compressed session summaries
├── decisions/         # Decision logs (JSONL)
└── embeddings.db      # SQLite-vec for semantic search
```

**Three-tier capture:**

| Tier | What | Example |
|------|------|---------|
| Events | State transitions | "TASK-003 complete" |
| Decisions | Choice + reason | "recursive descent because simpler" |
| Summary | Compressed narrative | "Refactored parser via method extraction" |

**Key design:**
- Orchestrator extracts from structured results (not agent CLI calls)
- Local embeddings via ONNX (no API calls)
- SQLite-vec for semantic queries (no server)

### 4. Model Routing

```toml
[routing]
simple = "haiku"      # grep, read, simple edits
medium = "sonnet"     # most development
complex = "opus"      # architecture, meta-audit
threshold_lines = 50  # >50 LOC = medium→complex
```

**Reality check:** Claude Code's model set at session start. Routing works for sub-agents (Task tool), advisory-only for inline work.

### 5. Passive Context > Skills

| Put in CLAUDE.md | Keep in Skills |
|------------------|----------------|
| TDD discipline (no test weakening) | Interview workflows |
| Traceability format | TDD phase logic |
| Commit conventions | Audit checklists |
| Evidence-based findings | Domain-specific process |

Skills are user-triggered only. Don't expect agent to self-invoke.

### 6. Skill Compression

Before (full content, ~2000 tokens):
```markdown
## Rules
1. Write minimal code...
2. Do not add features...
... (500 lines)
```

After (compressed, ~500 tokens):
```markdown
## Quick Reference
- Minimal code only | No extra features | No refactoring yet
- On failure: check assertions → logic → dependencies

## Full Docs
projctl skills docs tdd-green
```

---

## Unified Design Principles

Combining user vision + tool review learnings:

| Principle | Source | Implementation |
|-----------|--------|----------------|
| Thin orchestrator | User vision | Message passing only, no decisions |
| Co-routine yields | User vision | Context serialization to disk |
| Role-based agents | User vision | 9 roles mimicking human teams |
| Relentless continuation | oh-my-opencode | Continue until legitimate blocker |
| Background mapping | oh-my-opencode | Cheap agent pre-explores |
| Passive critical rules | Vercel | CLAUDE.md over skills |
| Skill compression | Vercel | Index + on-demand retrieval |
| Local semantic memory | claude-flow | ONNX + SQLite-vec |
| Simple model routing | claude-flow | Haiku/Sonnet/Opus by complexity |
| External state | Gastown | TOML files survive crashes |
| Graceful degradation | Gastown | Retry/skip/escalate on failure |

---

## Next Steps

1. Draft unified system design combining all learnings
2. Define yield protocol with TOML format
3. Map current 19 skills to 9 roles
4. Design thin orchestrator (Go program, not LLM)
5. Prototype with one role (product-manager)

---
