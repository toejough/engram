# Orchestration System

A unified project orchestration system with explicit agent flows, yield protocol, and full traceability.

---

## 1. Core Patterns

### 1.1 Phases

Every workflow progresses through phases. The "new project" workflow uses all phases:

| Phase          | Agent                             | Purpose                                 |
| -------------- | --------------------------------- | --------------------------------------- |
| PM             | PM Producer + PM QA               | Discover problems, produce requirements |
| DESIGN         | Design Producer + Design QA       | Define user experience                  |
| ARCHITECTURE   | Architect Producer + Architect QA | Define technical approach               |
| TASK BREAKDOWN | Breakdown Producer + Breakdown QA | Decompose into executable tasks         |
| IMPLEMENTATION | TDD Loop (per task)               | Build with test-first discipline        |
| DOCUMENTATION  | Tech Writer + Tech Writer QA      | Update repo-level docs                  |
| RETROSPECTIVE  | Retro Agent                       | Capture learnings, file issues          |
| SUMMARY        | Summary Agent                     | Summarize accomplishments               |
| NEXT STEPS     | Next Steps Agent                  | Suggest follow-on work                  |

### 1.2 Looper Agent

Controls iteration within a phase or across tasks. Decides between sequential and parallel execution:

```
LOOPER AGENT:
1. Create/Recreate Queue (items to do based on dependencies, impact, simplicity)
2. Identify next batch:
   - Find all items with no blocking dependencies
   - If single item → execute via PAIR LOOP
   - If N independent items → delegate to PARALLEL LOOPER
3. Execute batch
4. On completion, re-evaluate queue (dependencies may have resolved)
5. Return to step 1
6. Stop & return when queue is empty or entirely blocked
```

**Execution paths:**

```
LOOPER
├── PAIR LOOP (single item)
│   └── producer + qa
└── PARALLEL LOOPER (N independent items)
    ├── N × PAIR LOOP (in parallel)
    └── consistency-checker (validates batch)
```

**Queue ordering uses:**

- **Dependencies**: Automated from explicit TASK-N references in `Dependencies:` field
- **Structural Impact**: LLM analysis (haiku) - tasks that enable others run first
- **Simplicity**: LLM analysis (haiku) - simpler tasks run earlier when impact is equal

**Parallelization principle:** Maximize batch size. If 5 items have no dependencies on each other, run all 5 in parallel rather than picking one.

### 1.3 Pair Loop

Every producer has a paired QA agent:

```
PAIR LOOP:
1. Run PRODUCER agent
2. Run QA agent
3. Evaluate outcome:
   - If APPROVED → return outcome to LOOPER AGENT
   - If NEEDS IMPROVEMENT → return to PRODUCER (max 3x, then escalate)
   - If NEEDS ESCALATION → return to prior PHASE or user
```

Pair state is tracked in the project state file (see Section 4.1), enabling parallel execution of multiple phases or tasks.

### 1.4 Parallel Looper

Invoked by LOOPER when N independent items can be processed in parallel. Runs N PAIR LOOPs concurrently, then validates consistency.

```
PARALLEL LOOPER:
1. Receive items from LOOPER (all verified independent)
2. SPAWN: Launch PAIR LOOP for each item (in parallel)
3. WAIT: Collect all yields
4. DISPATCH: Run CONSISTENCY CHECKER on aggregated results
5. RETURN: Aggregated result to LOOPER (or remediation needs)

CONSISTENCY CHECKER (QA for parallel batches):
1. REVIEW: Compare outputs across all parallel results
2. CHECK: Apply domain-specific consistency rules
3. RETURN:
   - If consistent → approved (aggregate and return)
   - If minor issues → approved with remediations applied
   - If major issues → improvement-request (batch fails, items need rework)
```

**Key principle:** Each item runs through a full PAIR LOOP. The consistency-checker acts as batch-level QA, ensuring parallel results are coherent.

**Applies to:**

- Context exploration queries (need-context with multiple queries)
- Independent task execution
- Parallel skill creation
- Batch file analysis
- Any N independent items

**Consistency checks are domain-specific:**

- For context queries: results don't contradict, all queries answered
- For task execution: no conflicting file changes
- For skill creation: naming, format, cross-references align
- For file analysis: findings categorized consistently

**Implementation by layer:**

| Layer | Parallel Looper            | Consistency Checker             |
| ----- | -------------------------- | ------------------------------- |
| -1    | `parallel-looper` skill    | `consistency-checker` skill     |
| 0+    | `projctl parallel` command | `projctl consistency` or inline |

**Parallel Looper Yield:**

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
items_processed = 8
items_succeeded = 7
items_failed = 1
consistency_status = "passed"  # passed | failed | remediated

[[payload.results]]
item = "TASK-5"
status = "complete"
artifact = "skills/pm-interview-producer/SKILL.md"

[[payload.results]]
item = "TASK-6"
status = "complete"
artifact = "skills/pm-infer-producer/SKILL.md"

[[payload.consistency_issues]]
type = "naming"
items = ["TASK-7", "TASK-8"]
issue = "Inconsistent frontmatter phase values"
resolution = "Standardized to 'design'"
```

### 1.5 Producer Agent Pattern

Every producing agent follows this pattern:

```
PRODUCER AGENT:
1. EVALUATE
   - What problem is being solved?
   - What is the current state?
   - What is the desired state?
   - What guidance exists for getting from here to there?

2. GATHER
   - Query territory map: projctl territory show
   - Query memory: projctl memory query "<relevant terms>"
   - Read relevant artifacts and code
   - Research external sources if needed

3. SYNTHESIZE
   - Summarize findings
   - Draft artifact content

4. CONFIRM (workflow-dependent)
   - Interview agents (new project workflow): Present to user, get approval
   - Infer agents (adopt/align workflows): Validate against code/artifacts, collect escalations
   - Incorporate feedback or flag escalations
   - Repeat as needed

5. PRODUCE
   - Get next ID: projctl id next --type REQ|DES|ARCH|TASK
   - Write artifact with traceable IDs
   - Add **Traces to:** field linking to upstream IDs

6. COMMIT
   - Invoke /commit skill
   - Checkpoint state
```

### 1.6 QA Agent Pattern

```
QA AGENT:
1. REVIEW
   - Completeness: Does artifact address all inputs?
   - Clarity: Are items unambiguous and specific?
   - Traceability: Do all IDs have **Traces to:** fields?
   - Guidelines: Does artifact follow our guidelines?
   - Alignment: Any gaps between input and output?

2. RETURN one of:
   - APPROVED: Artifact meets all criteria
   - NEEDS IMPROVEMENT: Specific issues for producer to fix
   - ESCALATE UP: Prior phase artifact needs correction
   - ESCALATE USER: Clarification needed from user
```

---

## 2. Phase Definitions

### 2.1 PM Phase

| Aspect             | Details                                                |
| ------------------ | ------------------------------------------------------ |
| Entry Criteria     | Issue identified or user request received              |
| Producer           | PM Interview Agent                                     |
| QA                 | PM QA Agent                                            |
| Artifacts Produced | `.claude/projects/<name>/requirements.md`              |
| IDs Created        | REQ-N                                                  |
| Traces To          | ISSUE-N (if applicable)                                |
| Exit Criteria      | All requirements have acceptance criteria, QA approved |

**Domain:** Problem space - what problem, for whom, why it matters. Applies core evaluation questions to understand the problem and produce user stories with acceptance criteria.

### 2.2 Design Phase

| Aspect             | Details                                                   |
| ------------------ | --------------------------------------------------------- |
| Entry Criteria     | Requirements approved                                     |
| Producer           | Design Interview Agent                                    |
| QA                 | Design QA Agent                                           |
| Artifacts Produced | `.claude/projects/<name>/design.md`, `.pen` files (if UI) |
| IDs Created        | DES-N                                                     |
| Traces To          | REQ-N                                                     |
| Exit Criteria      | All user experience impacts addressed, QA approved        |

**Domain:** User experience space - workflows, interactions, visual design. Applies core evaluation questions to understand UX impacts and produce design specs, wireframes, and .pen files.

### 2.3 Architecture Phase

| Aspect             | Details                                         |
| ------------------ | ----------------------------------------------- |
| Entry Criteria     | Design approved                                 |
| Producer           | Architect Interview Agent                       |
| QA                 | Architect QA Agent                              |
| Artifacts Produced | `.claude/projects/<name>/architecture.md`       |
| IDs Created        | ARCH-N                                          |
| Traces To          | DES-N                                           |
| Exit Criteria      | All technical decisions documented, QA approved |

**Domain:** Technical space - technology choices, layers, data models, APIs, infrastructure. Applies core evaluation questions to understand technical impacts and produce architecture decisions with rationale.

### 2.4 Task Breakdown Phase

| Aspect             | Details                                           |
| ------------------ | ------------------------------------------------- |
| Entry Criteria     | Architecture approved                             |
| Producer           | Task Breakdown Agent                              |
| QA                 | Task Breakdown QA Agent                           |
| Artifacts Produced | `.claude/projects/<name>/tasks.md`                |
| IDs Created        | TASK-N                                            |
| Traces To          | ARCH-N                                            |
| Exit Criteria      | DAG with no cycles, all ACs testable, QA approved |

**Task Requirements:**

- TASK-N ID (sequential)
- Clear title + description
- Acceptance criteria (checkboxes)
- Files to create/modify
- Dependencies (explicit TASK-N IDs or `None`)
- **Traceability:** field with upstream IDs

Execution order determined by Looper Agent based on dependencies, structural impact, and simplicity.

### 2.5 Implementation Phase

| Aspect             | Details                                           |
| ------------------ | ------------------------------------------------- |
| Entry Criteria     | Tasks defined, DAG valid                          |
| Producer           | TDD Producer (composite - runs nested pair loops) |
| QA                 | TDD QA (overall compliance after nested loops)    |
| Artifacts Produced | Test files, implementation code                   |
| IDs Created        | None (test names trace to TASK-N)                 |
| Traces To          | TASK-N (via test name/comments)                   |
| Exit Criteria      | All tests pass, linter clean, TDD QA approved     |

**TDD as Composite PAIR LOOP:**

TDD follows the universal PAIR LOOP pattern, but the producer is composite - it runs nested pair loops internally:

```
TDD PAIR LOOP (per task):
├── Producer: tdd-producer (composite)
│   └── Internally runs (always serial):
│       1. RED PAIR LOOP
│       │   ├── red-producer: Write failing tests → /commit
│       │   └── red-qa: Verify tests cover ACs, fail for right reasons
│       │
│       2. GREEN PAIR LOOP
│       │   ├── green-producer: Minimal implementation → /commit
│       │   └── green-qa: Verify all tests pass, no regressions
│       │
│       3. REFACTOR PAIR LOOP
│           ├── refactor-producer: Improve quality → /commit
│           └── refactor-qa: Verify tests still pass, code improved
│
└── QA: tdd-qa
    └── Verify overall AC compliance and TDD discipline

On completion: Mark task complete, LOOPER re-evaluates queue
```

**Why nested pair loops are always serial:** RED must complete before GREEN (can't implement without tests), GREEN must complete before REFACTOR (can't refactor broken code). The LOOPER/PARALLEL LOOPER pattern applies at the task level, not within TDD.

**Parallelization at task level:** Multiple independent tasks can run through TDD PAIR LOOPs in parallel via the PARALLEL LOOPER.

### 2.6 Documentation Phase

| Aspect             | Details                                            |
| ------------------ | -------------------------------------------------- |
| Entry Criteria     | All tasks complete                                 |
| Producer           | Tech Writer Agent                                  |
| QA                 | Tech Writer QA Agent                               |
| Artifacts Produced | README updates, API docs, user guides              |
| IDs Created        | None                                               |
| Traces To          | REQ-N, DES-N, ARCH-N                               |
| Exit Criteria      | All user-facing documentation updated, QA approved |

**Integration Responsibility:**
This phase integrates project artifacts into repo-level docs:

- `.claude/projects/<name>/requirements.md` → `docs/requirements.md`
- `.claude/projects/<name>/design.md` → `docs/design.md`
- `.claude/projects/<name>/architecture.md` → `docs/architecture.md`

Tasks remain in `.claude/projects/<name>/tasks.md` as project-specific history.

Integration merges new IDs into existing docs, resolving any conflicts or duplicates.

### 2.7 Alignment Phase

| Aspect             | Details                                           |
| ------------------ | ------------------------------------------------- |
| Entry Criteria     | Workflow complete (runs in main flow)             |
| Producer           | Alignment Check Agent                             |
| QA                 | None (auto-fix or escalate)                       |
| Artifacts Produced | Updated **Traces to:** fields, gap fixes          |
| IDs Created        | None                                              |
| Exit Criteria      | `projctl trace validate` passes or gaps escalated |

**Alignment Verifies:**

- Traceability chain: ISSUE → REQUIREMENT → DESIGN → ARCHITECTURE → TASK → test → implementation
- IDs point UP the stack (implementation → test → task → architecture → design → requirement → issue)
- No orphan IDs (referenced but not defined)
- No unlinked IDs (defined but not connected)

### 2.8 Retrospective Phase

| Aspect             | Details                                                    |
| ------------------ | ---------------------------------------------------------- |
| Entry Criteria     | Alignment complete (runs in main flow after all workflows) |
| Producer           | Retro Agent                                                |
| QA                 | None (user confirms)                                       |
| Artifacts Produced | `.claude/projects/<name>/retro.md`, follow-up issues       |
| Exit Criteria      | User confirms learnings, issues filed                      |

**Retro Evaluates:**

- What went well?
- What could be improved?
- Blockers and challenges?
- Action items for future?
- Patterns to adopt or avoid?

### 2.9 Summary Phase

| Aspect             | Details                              |
| ------------------ | ------------------------------------ |
| Entry Criteria     | Retro complete                       |
| Producer           | Summary Agent                        |
| QA                 | None (user confirms)                 |
| Artifacts Produced | `.claude/projects/<name>/summary.md` |
| Exit Criteria      | User confirms summary                |

### 2.10 Next Steps Phase

| Aspect             | Details                |
| ------------------ | ---------------------- |
| Entry Criteria     | Summary complete       |
| Producer           | Next Steps Agent       |
| QA                 | None                   |
| Artifacts Produced | Suggested next actions |
| Exit Criteria      | Suggestions presented  |

---

## 3. Yield Protocol

Agents serialize state and yield for user input, other agents, or decisions.

### 3.1 Yield Message Format

```toml
# Written to: .claude/agents/<role>-yield.toml

[yield]
type = "need-user-input"    # See types below
timestamp = 2026-02-02T10:30:00Z

[yield.question]
text = "What problem are you trying to solve?"
context = "Starting problem discovery phase"
options = []                # For need-decision type

[yield.state]
phase = "pm"
subphase = "PROBLEM"
state_file = ".claude/agents/pm-state.toml"
```

### 3.2 Yield Types

| Type                  | Meaning                     | Orchestrator Action                         |
| --------------------- | --------------------------- | ------------------------------------------- |
| `need-user-input`     | Question for user           | Prompt user, resume with answer             |
| `need-agent`          | Need another agent's work   | Spawn agent, resume with result             |
| `need-decision`       | Ambiguous, need user choice | Present options, resume with choice         |
| `need-context`        | Need context from sources   | Run queries (parallel), resume with results |
| `improvement-request` | QA returning to producer    | Resume producer with feedback               |
| `escalate-phase`      | Prior phase needs update    | Return to prior phase agent                 |
| `escalate-user`       | Cannot resolve              | Present to user                             |
| `complete`            | Phase finished              | Advance to next phase                       |
| `blocked`             | Cannot proceed              | Present blocker, await resolution           |
| `error`               | Something went wrong        | Retry (max 3x) or escalate                  |

### 3.3 Complete Yield

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:30:00Z

[yield.result]
artifact = "docs/requirements.md"
ids_created = ["REQ-001", "REQ-002", "REQ-003"]
files_modified = ["docs/requirements.md"]

[[yield.decisions]]
context = "Scope definition"
choice = "Focus on CLI only"
reason = "User's immediate need"
alternatives = ["Include GUI", "API first"]

[[yield.learnings]]
content = "User prioritizes simplicity over flexibility"
```

### 3.4 Improvement Request Yield

```toml
[yield]
type = "improvement-request"

[yield.feedback]
from_agent = "pm-qa"
to_agent = "pm"
iteration = 2
issues = [
    "REQ-003 acceptance criteria are not measurable",
    "REQ-005 missing edge case for empty input"
]
```

### 3.5 Need Context Yield

```toml
[yield]
type = "need-context"

[[yield.queries]]
type = "file"
path = "docs/requirements.md"

[[yield.queries]]
type = "memory"
query = "caching patterns"

[[yield.queries]]
type = "territory"
scope = "tests"

[[yield.queries]]
type = "web"
url = "https://example.com/docs"
prompt = "Extract the API format"

[[yield.queries]]
type = "semantic"
question = "How does authentication work in this codebase?"
```

**Query types:**

| Type      | Parameters      | What it fetches                         |
| --------- | --------------- | --------------------------------------- |
| file      | `path`          | File contents                           |
| memory    | `query`         | ONNX semantic memory results            |
| territory | `scope`         | Codebase structure map                  |
| web       | `url`, `prompt` | URL content, interpreted by prompt      |
| semantic  | `question`      | Answer about codebase (LLM exploration) |

**Execution as PAIR LOOP:**

Context gathering follows the universal PAIR LOOP pattern:

```
CONTEXT PAIR LOOP:
├── Producer: context-explorer
│   └── Gathers results for all queries
│   └── May use PARALLEL LOOPER if multiple queries
└── QA: context-qa
    └── Validates results are useful, relevant, consistent
    └── Checks all queries were answered
    └── Flags contradictions or stale data
```

**Implementation by layer:**

- Layer -1 (B1): All queries handled by `context-explorer` + `context-qa` agents
- Layer 2+ (B2): Deterministic queries (file, memory, territory) via projctl, semantic via agent, then `context-qa`

**Parallelization:** When multiple queries have no dependencies, `context-explorer` delegates to PARALLEL LOOPER. The consistency-checker validates the batch, then `context-qa` validates the aggregate is useful for the requesting producer.

### 3.6 Escalate Phase Yield

```toml
[yield]
type = "escalate-phase"

[yield.escalation]
from_phase = "design"
to_phase = "pm"
reason = "gap"              # error | gap | conflict

[yield.issue]
summary = "Parallelism not addressed in requirements"
context = "Design phase discovered need for context exploration"

[yield.proposed_changes]
# QA drafts the upstream changes, not just flags the issue

[[yield.proposed_changes.requirements]]
action = "add"
id = "REQ-10"
title = "Context Exploration via Yield"
content = "Producer skills can yield need-context..."

[[yield.proposed_changes.source_docs]]
file = "docs/orchestration-system.md"
section = "3.2 Yield Types"
change = "Add need-context yield type"
```

**Escalation reasons:**

| Reason     | Meaning                               | Example                                          |
| ---------- | ------------------------------------- | ------------------------------------------------ |
| `error`    | Prior phase output is incorrect       | "REQ-3 contradicts REQ-1"                        |
| `gap`      | Discovery reveals missing content     | "Parallelism not addressed"                      |
| `conflict` | Can't proceed without upstream change | "Design requires capability not in requirements" |

**QA responsibility:** When escalating, QA should draft the upstream changes (proposed_changes), not just flag the issue. This enables the prior phase producer to review and apply, rather than re-discover.

**Propagation:** Escalation may cascade - a gap in requirements may also require updating the source issue or orchestration spec.

---

## 4. State Serialization

### 4.1 Project State

```toml
# .claude/state.toml

[project]
name = "my-feature"
created = 2026-02-02T10:00:00Z
workflow = "new"            # new | adopt | align | single-task | intake

[phase]
current = "pm"
subphase = "PROBLEM"

[progress]
tasks_total = 0
tasks_complete = 0

[yield]
pending = true
type = "need-user-input"
agent = "pm"
context_file = ".claude/agents/pm-state.toml"

[[history]]
timestamp = 2026-02-02T10:00:00Z
phase = "init"
action = "started"

[errors]
last_error = ""
retry_count = 0
max_retries = 3

# Pair loop states - supports parallel execution
[pairs.pm]
iteration = 2
max_iterations = 3
producer_complete = true
qa_verdict = "needs_improvement"
improvement_request = "REQ-003 acceptance criteria are not measurable"

[pairs.design]
iteration = 0
max_iterations = 3
producer_complete = false
qa_verdict = ""

# Task-level pair states (for parallel task execution)
[pairs.task-007]
iteration = 1
max_iterations = 3
producer_complete = true
qa_verdict = "approved"
```

### 4.2 Agent State

```toml
# .claude/agents/<role>-state.toml

[state]
role = "pm"
mode = "interview"
started = 2026-02-02T10:00:00Z
last_updated = 2026-02-02T10:30:00Z

[progress]
phase = "CURRENT_STATE"
questions_asked = 2
questions_remaining = 3

[gathered]
problem = "Build process takes 10+ minutes"
affected = "All developers on the team"
impact = "Slows iteration cycles"

[pending]
next_question = "How does the build work today?"
remaining = ["Pain points?", "What should happen instead?"]

[[decisions]]
context = "Scope definition"
choice = "Focus on CI build, not local"
reason = "CI is the bottleneck per user"

[artifacts]
draft = """
### REQ-001: Fast CI Build

As a developer, I want CI builds to complete in under 2 minutes...
"""
```

---

## 5. Traceability Chain

### 5.1 Downward (New Work)

During project work, artifacts live in `.claude/projects/<name>/`. After DOCUMENTATION phase, they're integrated into `docs/`.

```
ISSUE-N (optional)
    │
    ▼
REQ-N (requirements.md)
    │    **Traces to:** ISSUE-N
    ▼
DES-N (design.md)
    │    **Traces to:** REQ-N
    ▼
ARCH-N (architecture.md)
    │    **Traces to:** DES-N
    ▼
TASK-N (tasks.md)
    │    **Traceability:** ARCH-N
    ▼
Test (by name)
    │    // traces: TASK-N
    ▼
Implementation
         // implements: <test name>
```

**Key principle:** IDs point UP the stack, not down. Tests use their function/subtest names as identifiers.

### 5.2 Upward (Understanding Existing Work)

```
IMPLEMENTATION → TEST → (TASK) → ARCHITECTURE → DESIGN → REQUIREMENT → (ISSUE)
```

TASK and ISSUE are optional when inferring - only create if mapping is clear.

When exploring existing work:

- TASK and ISSUE are optional (don't create if mapping is unclear)
- Tests, architecture, design, requirements should always be inferred
- If inference unclear at one level, escalate up to get context
- Ultimate escalation: ask user why something exists

### 5.3 Artifact Format

```markdown
### REQ-001: Feature Name

As a [persona], I want [capability], so that [benefit].

**Acceptance Criteria:**

- [ ] Criterion 1
- [ ] Criterion 2

**Priority:** P1

**Traces to:** ISSUE-42
```

```markdown
### ARCH-003: Build Cache Strategy

Use content-addressable cache with...

**Traces to:** REQ-001, DES-002
```

```markdown
### TASK-007: Implement cache lookup

Implement the cache lookup function...

**Acceptance Criteria:**

- [ ] Function returns cached value if present
- [ ] Function returns nil if not cached
- [ ] Cache hit/miss is logged

**Files:** internal/cache/lookup.go, internal/cache/lookup_test.go
**Dependencies:** TASK-005, TASK-006
**Traceability:** ARCH-003, REQ-001
```

---

## 6. Support Systems

### 6.1 Territory Mapping

Before major work, map the codebase:

```bash
projctl territory map --dir .
projctl territory show
```

**Territory Format:**

```toml
[structure]
root = "/path/to/project"
languages = ["go"]
build_tool = "mage"
test_framework = "go test + gomega + rapid"

[entry_points]
cli = "cmd/projctl/main.go"
public_api = "projctl.go"

[packages]
count = 12
internal = ["config", "context", "state", "trace", "memory"]

[patterns]
dependency_injection = true
table_driven_tests = true
property_tests = true
```

**When to Map:**

- `/project new` - Before PM interview
- `/project adopt` - Before inference
- Task start - Before TDD
- Cache invalidates on significant file changes

### 6.2 Memory System

Local semantic memory with no API calls:

```
~/.projctl/memory/
├── index.md              # Human-readable learnings (grep-able)
├── embeddings.db         # SQLite-vec for semantic search
├── sessions/
│   └── <project>-<date>.md
└── decisions/
    └── <project>.jsonl
```

**Embedding Engine (ONNX):**

Local semantic search using ONNX runtime - no API calls required:

| Component | Choice     | Rationale                                      |
| --------- | ---------- | ---------------------------------------------- |
| Runtime   | ONNX       | Cross-platform, no Python dependency           |
| Model     | e5-small   | Good quality/size tradeoff, ~130MB             |
| Storage   | SQLite-vec | Single file, no server, vector search built-in |

**When embeddings are generated:**

- `projctl memory learn` - embeds the message, stores in SQLite-vec
- `projctl memory extract` - embeds extracted insights from agent results
- `projctl memory session-end` - embeds session summary

**When embeddings are queried:**

- `projctl memory query` - embeds the query, returns top-k similar memories
- Orchestrator control loop - queries memory before spawning each agent

**Memory Commands:**

```bash
# Semantic query before spawning agent (uses ONNX to embed query)
projctl memory query "build performance patterns"

# Structural search (no ONNX, just grep)
projctl memory grep "caching"

# Learn from result (uses ONNX to embed)
projctl memory extract --result .claude/context/pm-result.toml

# User says "remember this" (uses ONNX to embed)
projctl memory learn --message "GraphQL adds complexity we don't need"

# Session end summary (uses ONNX to embed summary)
projctl memory session-end --project myproject
```

**Memory Injection:**
Orchestrator queries memory and injects into agent context:

```toml
[memory]
relevant = [
    "Previous project: Gradle cache reduced CI from 8min to 2min",
    "Learned: Incremental compilation key for Go builds",
]
query = "build performance"
```

### 6.3 Visual Verification

For UI work, use image diffing:

```bash
projctl screenshot diff --expected expected.png --actual actual.png
```

Returns SSIM score for regression detection. Visual verification is required for UI tasks.

### 6.4 Model Routing

```toml
# ~/.projctl/config.toml

[routing]
# By agent
pm = "opus"
design = "opus"
architect = "opus"
breakdown = "sonnet"
tdd-red = "sonnet"
tdd-green = "sonnet"
tdd-refactor = "sonnet"
task-audit = "sonnet"
retro = "opus"
tech-writer = "haiku"

# Special cases
territory-mapping = "haiku"
task-ordering = "haiku"
```

### 6.5 Git Worktrees for Parallel Execution

When running multiple tasks in parallel, each task operates in an isolated git worktree to prevent file conflicts during execution.

**Commands:**

| Command | Purpose |
|---------|---------|
| `projctl worktree create --taskid TASK-001` | Create isolated worktree for task |
| `projctl worktree merge --taskid TASK-001 [--onto main]` | Merge task branch to target |
| `projctl worktree cleanup --taskid TASK-001` | Remove single task worktree |
| `projctl worktree cleanup-all` | Remove all task worktrees |
| `projctl worktree list` | List active worktrees |

**Worktree Workflow:**

```
1. CREATE: projctl worktree create --taskid TASK-001
   - Creates branch: task-001
   - Creates worktree: .worktrees/task-001/
   - Agent works in isolated directory

2. WORK: Agent executes TDD cycle in worktree
   - All file changes isolated to worktree
   - Normal commits on task branch

3. MERGE (on completion): projctl worktree merge --taskid TASK-001
   - Rebase branch onto target (main)
   - Fast-forward merge if clean
   - Remove worktree directory
   - Delete branch after merge

4. CLEANUP: projctl worktree cleanup-all
   - Remove any orphaned worktrees
   - Report cleanup failures (don't block)
```

**Merge-on-Complete Pattern:**

When a parallel agent completes, merge immediately - don't wait for all agents to finish:

| Pattern           | Behavior                                   | Result                               |
| ----------------- | ------------------------------------------ | ------------------------------------ |
| Batch merge (old) | Wait for all agents, merge all at end      | More conflicts, duplicate code       |
| Merge-on-complete | Merge each branch when its agent completes | Less conflicts, later agents benefit |

Benefits of merge-on-complete:

- Later-completing agents rebase onto already-merged work
- Reduces conflict window
- Simplifies conflict resolution
- No batch of N-way merges at the end

**Error Handling:**

| Situation                      | Behavior                                                                |
| ------------------------------ | ----------------------------------------------------------------------- |
| Rebase conflict                | Pause orchestration, prompt user to resolve                             |
| Agent failure mid-execution    | Don't merge branch, cleanup worktree, log failure, continue with others |
| Cleanup failure (locked files) | Log error, continue, report at end                                      |
| Simultaneous completions       | Serialize by completion timestamp (earliest first)                      |

**Decision Factors for Parallel Execution:**

| Factor             | Parallel-friendly                      | Sequential-preferred                |
| ------------------ | -------------------------------------- | ----------------------------------- |
| Task independence  | No shared state or coordination needed | Tasks depend on each other's output |
| File overlap       | Low/no shared files                    | High overlap, tight coupling        |
| Coordination needs | None during execution                  | Need to coordinate decisions        |
| Task granularity   | Atomic, well-bounded work              | Large, sprawling changes            |

**Examples:**

Good parallel candidates:

- Independent feature additions to different modules
- Test coverage for separate components
- Documentation updates for different subsystems

Poor parallel candidates:

- Refactoring same function with different goals
- Sequential pipeline stages (design → implement → test same feature)
- Tasks that need to share intermediate decisions

**Known Limitations:**

- Agents cannot coordinate during execution (no shared state)
- File overlap causes merge conflicts (handled via rebase/manual resolution)
- No pre-flight file overlap detection (accepted tradeoff - rebase handles it)
- Simultaneous completions require serialization

---

## 7. Workflows

### 7.1 Intake (Main Flow)

All work starts here. Dispatch is **automatic with escalation** - the system makes its best guess and escalates to user if uncertain. Any user corrections are captured in the retrospective.

```
1. EVALUATE REQUEST (automatic)
   - Is this a new issue to file?
   - Is this work on an existing issue/project?
   - Is this a simple task or multi-task project?
   - If uncertain: escalate to user for clarification

2. CREATE/LINK ISSUES
   - If new work: create issue first
   - If existing: link to issue

3. DISPATCH TO WORKFLOW (automatic)
   - Multi-task new work → New Project (7.2)
   - Single task → Single Task (7.5)
   - Existing codebase needs docs → Adopt Existing (7.3)
   - Drift detected → Align Drift (7.4)
   - If user corrects dispatch: capture in retro

4. ALIGNMENT
   - Run alignment check on all artifacts
   - Verify traceability chain is complete
   - Fix gaps or escalate

5. RETROSPECTIVE
   - What went well? What could improve?
   - Capture learnings, file follow-up issues

6. SUMMARY
   - Summarize accomplishments
   - User confirms

7. ON COMPLETION
   - Update/close issues
   - Return to user

8. NEXT STEPS
   - Suggest follow-on work based on open issues
```

### 7.2 New Project

Full flow for greenfield work. Uses **interview agents** that interact with user:

| Phase          | Agent                              | User Interaction              |
| -------------- | ---------------------------------- | ----------------------------- |
| PM             | PM Interview + PM QA               | Yes - questions about problem |
| DESIGN         | Design Interview + Design QA       | Yes - preferences, approvals  |
| ARCHITECTURE   | Architect Interview + Architect QA | Yes - technology choices      |
| TASK BREAKDOWN | Breakdown + Breakdown QA           | No                            |
| IMPLEMENTATION | TDD Agent + TDD QA                 | No                            |
| DOCUMENTATION  | Tech Writer + Tech Writer QA       | No                            |

Returns to main flow for ALIGNMENT → RETROSPECTIVE → SUMMARY → ON COMPLETION → NEXT STEPS.

### 7.3 Adopt Existing

Infer artifacts from existing code. Uses **infer agents** that analyze code and collect escalations:

| Phase                 | Agent                          | User Interaction    |
| --------------------- | ------------------------------ | ------------------- |
| EXPLORE               | Implementation Explorer        | No                  |
| INFER TESTS           | Test Mapper                    | No                  |
| INFER ARCH            | Architect Infer + Architect QA | Escalations only    |
| INFER DESIGN          | Design Infer + Design QA       | Escalations only    |
| INFER REQS            | PM Infer + PM QA               | Escalations only    |
| ESCALATION RESOLUTION | -                              | Yes - batch resolve |
| DOCUMENTATION         | Tech Writer Infer              | No                  |

Escalate progressively up the stack when inference is unclear.

Returns to main flow for ALIGNMENT → RETROSPECTIVE → SUMMARY → ON COMPLETION → NEXT STEPS.

### 7.4 Align Drift

Same as Adopt Existing - detect and fix drift between code and docs.

### 7.5 Single Task

Lightweight flow for simple work:

| Phase          | Agent                        | User Interaction |
| -------------- | ---------------------------- | ---------------- |
| IMPLEMENTATION | TDD Agent + TDD QA           | No               |
| DOCUMENTATION  | Tech Writer (if user-facing) | No               |

Returns to main flow for ALIGNMENT → RETROSPECTIVE → SUMMARY → ON COMPLETION → NEXT STEPS.

---

## 8. Guidelines Reference

### 8.1 Requirements Guidelines

| Guideline           | Details                                                  |
| ------------------- | -------------------------------------------------------- |
| User story format   | "As a [persona], I want [capability], so that [benefit]" |
| Acceptance criteria | Checkboxes, specific and measurable                      |
| One sentence test   | Can articulate problem in one sentence before proceeding |
| Scope boundaries    | Note items for Design/Architecture, redirect to problem  |
| No implementation   | Never discuss how - that's Architecture's job            |

### 8.2 Design Guidelines

| Guideline           | Details                                        |
| ------------------- | ---------------------------------------------- |
| Design system first | Establish tokens and components BEFORE screens |
| Pencil MCP          | All visual designs in .pen files               |
| Every DES traces    | Must link to REQ                               |
| Dependency order    | Build screens in order of dependencies         |

### 8.3 Architecture Guidelines

| Guideline              | Details                                    |
| ---------------------- | ------------------------------------------ |
| Progressive disclosure | High-level overview → detailed sections    |
| Pure business logic    | Dependency injection for testability       |
| Clean separation       | Domain, storage, UI, infrastructure layers |
| Document alternatives  | Always note what was considered            |
| Every ARCH traces      | Must link to REQ/DES                       |

### 8.4 Task Guidelines

| Guideline             | Details                                      |
| --------------------- | -------------------------------------------- |
| Explicit dependencies | TASK-N IDs only - never "All previous"       |
| DAG structure         | No cycles allowed                            |
| Size appropriately    | One function = one task (for pure functions) |
| Testable ACs          | Every criterion must be verifiable           |

### 8.5 TDD Guidelines

**Red Phase:**
| Rule | Details |
|------|---------|
| Tests ONLY | No implementation code |
| Must FAIL | Verify tests fail for right reason |
| Cover ALL criteria | Map each AC to at least one test |
| Behavior focus | Test action → event → handler → state → UI chain |
| Property tests | Use rapid (Go) or fast-check (TS) |
| Blackbox only | `package foo_test` in Go |

**Green Phase:**
| Rule | Details |
|------|---------|
| MINIMAL code | Just enough to pass |
| NO refactoring | That comes next |
| ALL tests pass | Including existing tests |
| Follow arch patterns | Use established conventions |

**Refactor Phase:**
| Rule | Details |
|------|---------|
| Tests STAY GREEN | Revert if they break |
| NO behavior changes | Only structure improvements |
| Fix linter issues | High priority: complexity, security, duplication |
| NO blanket overrides | Never add exclusions without asking |

### 8.6 Audit Red Flags

| Violation             | Examples                                        |
| --------------------- | ----------------------------------------------- |
| Test Weakening        | Removed tests, weakened assertions, added .skip |
| Linter Gaming         | New nolint, config changes, threshold changes   |
| Missing Coverage      | AC without corresponding test                   |
| Structural-only tests | Testing DOM exists but not behavior             |

### 8.7 Code Quality Guidelines

| Guideline                 | Details                                    |
| ------------------------- | ------------------------------------------ |
| Entry points thin         | Only re-exports and DI wiring              |
| Side effects at edges     | Never in internal logic                    |
| No flaky tests            | Use DI to avoid IO-based flakiness         |
| Failing tests = impl bugs | Investigate implementation first, not test |
| No "pre-existing" excuses | Fix ALL failures when discovered           |
| No TODO for incomplete    | Implement or ask - never silently defer    |

---

## 9. File Layout

```
project/
├── .claude/
│   ├── state.toml                # Orchestrator state (includes pair states)
│   ├── territory.toml            # Codebase map
│   ├── agents/
│   │   ├── pm-state.toml         # Agent state (resume)
│   │   ├── pm-yield.toml         # Pending yield
│   │   ├── pm-result.toml        # Completed result
│   │   ├── task-007-state.toml   # Per-task state (parallel execution)
│   │   └── ...
│   └── projects/
│       └── <project-name>/
│           ├── requirements.md   # WIP: REQ-N for this project
│           ├── design.md         # WIP: DES-N for this project
│           ├── architecture.md   # WIP: ARCH-N for this project
│           ├── tasks.md          # TASK-N (stays here permanently)
│           ├── retro.md          # Project retrospective
│           └── summary.md        # Project summary
├── docs/
│   ├── requirements.md           # Integrated REQ-N (all projects)
│   ├── design.md                 # Integrated DES-N (all projects)
│   └── architecture.md           # Integrated ARCH-N (all projects)
└── README.md

~/.projctl/
├── memory/
│   ├── index.md
│   ├── embeddings.db
│   ├── sessions/
│   └── decisions/
├── agents/
│   ├── pm.md                     # Agent prompt
│   ├── pm-qa.md
│   ├── design.md
│   └── ...
└── config.toml
```

**Artifact Flow:**

1. During project: artifacts written to `.claude/projects/<name>/`
2. DOCUMENTATION phase: integrates project artifacts into `docs/`
3. After integration: project folder retained for history/retro

---

## 10. CLI Commands

### 10.1 Orchestration

```bash
# Start workflows
projctl project new <name>
projctl project adopt
projctl project align
projctl project task <description>

# Control
projctl project continue          # Resume after yield
projctl project status            # Show current state
projctl project skip <phase>      # Skip optional phase
```

### 10.2 State Management

```bash
projctl state get                 # Current phase/task/pair states
projctl state transition --to <phase>
projctl state next                # Determine continue/stop
```

### 10.3 Context

```bash
projctl context write --skill <name>
projctl context read --result
```

### 10.4 IDs

```bash
projctl id next --type REQ        # Get REQ-N
projctl id next --type DES        # Get DES-N
projctl id next --type ARCH       # Get ARCH-N
projctl id next --type TASK       # Get TASK-N
```

### 10.5 Traceability

```bash
projctl trace validate            # Check for orphans/gaps
projctl trace repair              # Auto-fix where possible
projctl trace show                # Visualize chain
```

### 10.6 Territory & Memory

```bash
projctl territory map --dir .
projctl territory show
projctl memory query "<terms>"
projctl memory learn --message "<insight>"
projctl memory grep "<pattern>"
```

### 10.7 Visual Verification

```bash
projctl screenshot diff --expected <path> --actual <path>
projctl screenshot capture --url <url> --output <path>
```

---

## 11. Orchestrator Control Loop

```go
// Pseudocode for projctl project

func main() {
    state := loadState(".claude/state.toml")

    for {
        // 1. Check for pending yield
        if state.Yield.Pending {
            answer := handleYield(state.Yield)
            state.Yield.Pending = false
        }

        // 2. Determine next action
        action := determineAction(state)

        switch action {
        case "run-pair-loop":
            // 3. Prepare context
            territory := loadTerritory()
            memory := queryMemory(state.Phase.Current)
            context := buildContext(state, territory, memory)

            // 4. Run pair loop
            for iteration := 0; iteration < 3; iteration++ {
                // Producer
                producerYield := spawnAgent(
                    state.Phase.Current,
                    "producer",
                    context,
                )

                if producerYield.Type != "complete" {
                    state.Yield = producerYield
                    break
                }

                // QA
                qaYield := spawnAgent(
                    state.Phase.Current + "-qa",
                    "qa",
                    producerYield.Result,
                )

                if qaYield.Type == "complete" && qaYield.Verdict == "approved" {
                    extractMemory(qaYield)
                    advancePhase(state)
                    break
                }

                if qaYield.Type == "improvement-request" {
                    context.Feedback = qaYield.Feedback
                    continue
                }

                if qaYield.Type == "escalate-phase" {
                    rewindToPhase(state, qaYield.TargetPhase)
                    break
                }

                if qaYield.Type == "escalate-user" {
                    state.Yield = qaYield
                    break
                }
            }

        case "prompt-user":
            answer = promptUser(state.Yield.Question)

        case "all-complete":
            runValidation()
            printSummary()
            return

        case "blocked":
            presentBlocker(state)
            return
        }

        // 5. Save state (crash recovery)
        saveState(state)
    }
}
```

---

## 12. Implementation Plan

Two-phase migration: first unify skills to the new agent patterns, then migrate orchestration from `/project` skill to projctl. Each layer is testable independently before building the next.

### Layer -1: Skill Unification

Update all skills to unified pattern before projctl takes over orchestration. Test with current `/project` skill.

**Phase Agent Skills** (producer + QA pairs):

| Phase         | Interview Producer          | Infer Producer          | QA             |
| ------------- | --------------------------- | ----------------------- | -------------- |
| PM            | `pm-interview-producer`     | `pm-infer-producer`     | `pm-qa`        |
| Design        | `design-interview-producer` | `design-infer-producer` | `design-qa`    |
| Architecture  | `arch-interview-producer`   | `arch-infer-producer`   | `arch-qa`      |
| Breakdown     | `breakdown-producer`        | -                       | `breakdown-qa` |
| Documentation | `doc-producer`              | -                       | `doc-qa`       |

Interview producers: gather requirements via user Q&A (New Project workflow)
Infer producers: analyze existing code to infer artifacts (Adopt Existing workflow)

**TDD Agent Skills** (nested producer + QA pairs):

| Phase    | Producer                | Infer Producer           | QA                |
| -------- | ----------------------- | ------------------------ | ----------------- |
| RED      | `tdd-red-producer`      | `tdd-red-infer-producer` | `tdd-red-qa`      |
| GREEN    | `tdd-green-producer`    | -                        | `tdd-green-qa`    |
| REFACTOR | `tdd-refactor-producer` | -                        | `tdd-refactor-qa` |
| Overall  | -                       | -                        | `tdd-qa`          |

**Support Agent Skills**:

- `alignment-producer` / `alignment-qa` - traceability validation
- `retro-producer` / `retro-qa` - retrospective (includes process improvement)
- `summary-producer` / `summary-qa` - project summary
- `intake-evaluator` - request type classification
- `next-steps` - suggest follow-up work
- `context-explorer` / `context-qa` - gathers context, validates usefulness/consistency
- `parallel-looper` - runs N PAIR LOOPs in parallel, dispatches consistency check
- `consistency-checker` - batch QA for parallel results, validates consistency
- `commit` - commit changes (unchanged from current)

**Context Exploration (B1 approach):**
Producer skills can yield `need-context` with a list of queries. The `context-explorer` agent handles all query types via LLM. This is simple but uses LLM even for deterministic operations. Layer 0+ optimizes with B2 hybrid approach.

**All skills must**:

- Accept context via standard input format (from orchestrator)
- Output yield protocol TOML (to orchestrator)
- Follow producer or QA role guidelines

**Skills to delete** (functionality merged into new skills):

- `pm-audit`, `design-audit`, `architect-audit`, `task-audit` → merged into QA skills
- `negotiate` → merged into QA escalate-phase capability
- `meta-audit` → merged into retro-producer
- `test-mapper` → obsolete (no TEST-NNN IDs)

**Proves:** Unified agent patterns work with existing `/project` orchestrator before projctl migration.

### Layer 0: Foundation

Build core projctl infrastructure without agent spawning:

```
projctl state get|transition|next
projctl context write|read
projctl id next --type REQ|DES|ARCH|TASK
projctl trace validate|repair
projctl territory map|show
projctl memory query|learn|grep|extract|session-end
```

**Context write must include:**

- `output.yield_path` with unique session/task ID for parallel execution support
- Skills write to provided path, enabling multiple simultaneous invocations

**Dependencies:**

- ONNX runtime (for embedding generation)
- e5-small model (~130MB, downloaded on first use)
- SQLite-vec (for vector storage/search)

**Proves:** State management, context serialization, ID generation, semantic memory work.

**Skill Updates:** None - skills already unified in Layer -1.

### Layer 1: Leaf Commands

Wrap simplest skills that don't spawn sub-agents:

```
projctl commit
  └── spawns: claude -p "run /commit skill"
  └── returns: success/failure
  └── updates: state
```

**Proves:** projctl can spawn Claude CLI, capture output, update state.

**Skill Updates:** Verify `/commit` skill works when spawned by projctl. May need adjustments for non-interactive invocation.

### Layer 2: Single Pair Loop

Wrap a single producer + QA cycle:

```
projctl pair --phase pm
  └── spawns: PM Producer agent
  └── parses: yield (need-user-input | complete | improvement-request)
  └── handles: user prompts (CLI), QA dispatch, iteration (max 3x)
  └── commits: via projctl commit
  └── returns: complete | escalate
```

**Proves:** Yield protocol works, pair loop logic is correct.

**Context Exploration (B2 hybrid approach):**
Upgrade from B1 (all LLM) to B2 (hybrid):

| Query Type | B1 (Layer -1) | B2 (Layer 2+)                        |
| ---------- | ------------- | ------------------------------------ |
| file       | LLM reads     | projctl reads (deterministic)        |
| memory     | LLM queries   | projctl queries ONNX (deterministic) |
| territory  | LLM maps      | projctl maps (deterministic)         |
| web        | LLM fetches   | projctl fetches, LLM summarizes      |
| semantic   | LLM explores  | LLM explores (unchanged)             |

Orchestrator classifies each query in `need-context` yield:

- Deterministic queries → projctl tools (parallel, no LLM)
- Semantic queries → context-explorer agent

This reduces LLM usage while maintaining capability.

**Skill Updates:** Validate phase skills (created in Layer -1) work when spawned by `projctl pair`.

### Layer 3: Nested Pair Loop (TDD)

Wrap TDD loop with nested pairs:

```
projctl tdd --task TASK-1
  └── runs: RED pair loop → projctl commit
  └── runs: GREEN pair loop → projctl commit
  └── runs: REFACTOR pair loop → projctl commit
  └── runs: TDD QA
  └── returns: complete | blocked
```

**Proves:** Nested loops work, task-level state management.

**Skill Updates:** Validate TDD skills (created in Layer -1) work when spawned by `projctl tdd`.

### Layer 4: Phase Orchestration

Wrap full phases with looper logic:

```
projctl phase pm|design|arch|breakdown|implementation|documentation
  └── runs: projctl pair --phase <phase> (looping until complete)
  └── handles: escalations to user or prior phase
  └── advances: state to next phase
```

For implementation phase:

```
projctl phase implementation
  └── builds: task queue from dependencies, impact, simplicity
  └── loops: projctl tdd --task <id> for each unblocked task
  └── re-evaluates: queue after each task
```

**Parallel execution (future consideration):**

- Independent tasks (no shared dependencies) could run in parallel
- Orchestrator provides unique yield paths per invocation (prepared in Layer 0)
- Decision: Start sequential, add parallelism when proven stable

**Proves:** Phase-level orchestration, task queue management.

**Skill Updates:** Skills stabilize at this layer - projctl controls all orchestration logic. Skills now purely define agent behavior (prompts, guidelines) without orchestration concerns. Update `/project` skill to delegate phase execution to `projctl phase`.

### Layer 5: Workflow Orchestration

Wrap complete workflows:

```
projctl workflow new|adopt|align|task
  └── runs: sequence of projctl phase commands
  └── manages: workflow-specific logic
```

Example for `new`:

```
projctl workflow new
  └── projctl phase pm
  └── projctl phase design
  └── projctl phase architecture
  └── projctl phase breakdown
  └── projctl phase implementation
  └── projctl phase documentation
```

**Skill Updates:** Update `/project` skill to delegate workflow selection to `projctl workflow`. Add alignment, retro, summary agents if not already present.

### Layer 6: Main Flow

Full intake + workflow + common ending:

```
projctl project [description]
  └── evaluates: request type (automatic, escalate if uncertain)
  └── creates: issue if needed
  └── dispatches: projctl workflow new|adopt|align|task
  └── runs: projctl phase alignment
  └── runs: projctl phase retro
  └── runs: projctl phase summary
  └── handles: issue updates
  └── runs: projctl phase next-steps
```

**Skill Updates:** `/project` skill becomes a one-liner: invoke `projctl project` and display results. Add intake evaluation agent, next-steps agent.

### Layer 7: TUI

Wrap CLI in bubbletea for better UX:

- Pretty prompts for yields
- Progress visualization
- State dashboard
- Same underlying projctl commands

**Skill Updates:** No new skills needed - TUI is a presentation layer over existing projctl commands.

---

## 13. Migration Strategy

### Incremental Adoption

At each layer, the current `/project` skill can call `projctl <command>` instead of doing work inline. Skills become thinner over time.

| Layer Complete | `/project` Skill Does     | projctl Does        | Skills                  |
| -------------- | ------------------------- | ------------------- | ----------------------- |
| -1             | Everything (old patterns) | Nothing             | Unified to new patterns |
| 0              | Everything (new patterns) | State, IDs, tracing | Ready                   |
| 1              | Orchestration + phases    | + commit            | Ready                   |
| 2              | Orchestration + phases    | + pair loops        | Ready                   |
| 3              | Orchestration + phases    | + TDD loops         | Ready                   |
| 4              | Orchestration             | + all phases        | Ready                   |
| 5              | Dispatch only             | + workflows         | Ready                   |
| 6              | Nothing                   | Everything          | Ready                   |

### Final State

```
When user invokes /project:
  Run: projctl project
  Display: results
```

The `/project` skill becomes a one-liner that invokes projctl.

### Testing at Each Layer

Before moving to next layer:

1. Unit tests for projctl commands
2. Integration test: run full flow with mock agents
3. End-to-end test: run on real project with Claude CLI
