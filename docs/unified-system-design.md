# Unified System Design

A simplified, co-routine-based orchestration system combining user vision with tool review learnings.

---

## Design Principles

| Principle | Rationale |
|-----------|-----------|
| Thin orchestrator | Orchestrator passes messages, doesn't make decisions |
| Co-routine yields | Agents serialize state to disk, resume with answers |
| Role-based agents | 9 roles mimicking human engineering teams |
| Relentless continuation | Continue until legitimately blocked |
| Passive critical rules | CLAUDE.md always visible, roles for workflow |
| Local semantic memory | ONNX + SQLite-vec, no API calls |
| External state | TOML files survive crashes |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              USER                                        │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     ORCHESTRATOR (projctl pm)                            │
│                     Go program, deterministic                            │
│                                                                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐         │
│  │   State    │  │  Territory │  │   Memory   │  │   Yield    │         │
│  │ state.toml │  │ Mapper     │  │   System   │  │  Handler   │         │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘         │
│                                                                          │
│  Control Loop:                                                           │
│    1. Read state                                                         │
│    2. Check for pending yield                                            │
│    3. If yield needs user input → prompt user                            │
│    4. If yield needs other agent → spawn agent, get result               │
│    5. Spawn/resume agent with context + answer                           │
│    6. Parse agent output for yield                                       │
│    7. If complete → advance phase, loop                                  │
│    8. If yield → store yield, handle it                                  │
│    9. Continue until all_complete or escalation                          │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │   Agent     │ │   Agent     │ │   Agent     │
            │  (claude)   │ │  (claude)   │ │  (claude)   │
            │             │ │             │ │             │
            │ Receives:   │ │ Receives:   │ │ Receives:   │
            │ - Role      │ │ - Role      │ │ - Role      │
            │ - Context   │ │ - Context   │ │ - Context   │
            │ - Territory │ │ - Answer    │ │ - Memory    │
            │             │ │             │ │             │
            │ Returns:    │ │ Returns:    │ │ Returns:    │
            │ - Yield OR  │ │ - Yield OR  │ │ - Yield OR  │
            │ - Complete  │ │ - Complete  │ │ - Complete  │
            └─────────────┘ └─────────────┘ └─────────────┘
```

---

## Roles (9 Total)

Consolidated from 19 current skills:

| Role | Responsibility | Current Skills Merged |
|------|----------------|----------------------|
| **Product Manager** | Problem-space discovery, requirements | pm-interview, pm-infer, pm-audit |
| **Designer** | UX solution-space, visual design | design-interview, design-infer, design-audit |
| **Architect** | Code solution-space, technology decisions | architect-interview, architect-infer, architect-audit |
| **Planner** | Task breakdown, dependency ordering | task-breakdown |
| **Implementer** | TDD: write tests, implement, refactor | tdd-red, tdd-green, tdd-refactor |
| **QA** | Verify acceptance criteria, audit discipline | task-audit, alignment-check |
| **Retro** | Analyze corrections, propose improvements | meta-audit |
| **Project Manager** | Orchestration (this is the Go program) | project skill → becomes projctl pm |
| **Tech Writer** | Documentation maintenance | (new) |

### Role Modes

Each role can operate in different modes:

| Mode | Description | User Interaction |
|------|-------------|------------------|
| **interview** | Discover via questions | Yes - yields for answers |
| **infer** | Deduce from existing artifacts | No |
| **audit** | Verify against spec | No |
| **execute** | Do the work (implement, refactor) | No |

Example: Product Manager can run as:
- `product-manager:interview` - Ask user about problem
- `product-manager:infer` - Deduce requirements from code
- `product-manager:audit` - Verify implementation matches requirements

---

## Yield Protocol

### Yield Message Format

```toml
# Written by agent to: context/<role>-yield.toml

[yield]
type = "need-user-input"    # See types below
timestamp = 2026-02-02T10:30:00Z

[yield.question]
text = "What problem are you trying to solve?"
context = "Starting problem discovery phase"

[yield.state]
phase = "PROBLEM"           # Where in the workflow
progress = "1/5"            # How far along
context_file = "context/product-manager-state.toml"
```

### Yield Types

| Type | Meaning | Orchestrator Action |
|------|---------|---------------------|
| `need-user-input` | Question for user | Prompt user, resume with answer |
| `need-consult` | Need another role's input | Spawn that role, resume with result |
| `need-decision` | Ambiguous, need user choice | Present options, resume with choice |
| `complete` | Role finished successfully | Advance phase, continue |
| `blocked` | Cannot proceed | Present blocker, await resolution |
| `error` | Something went wrong | Retry or escalate |

### Consult Yield (nested agent call)

```toml
[yield]
type = "need-consult"

[yield.consult]
target_role = "architect"
mode = "interview"          # or "infer", "audit"
question = "Should we use REST or GraphQL for the API?"
context = "Designing user service, need API style decision"
```

### Complete Yield

```toml
[yield]
type = "complete"

[yield.result]
artifact = "docs/requirements.md"
ids_created = ["REQ-001", "REQ-002", "REQ-003"]
files_modified = ["docs/requirements.md"]

[yield.decisions]
[[yield.decisions.items]]
context = "API style"
choice = "REST"
reason = "Team familiarity, simpler for CRUD operations"
alternatives = ["GraphQL", "gRPC"]

[yield.learnings]
[[yield.learnings.items]]
content = "User prioritizes simplicity over flexibility"
```

---

## Context Serialization

Agents serialize their continuation state to enable resume:

### Context State File

```toml
# context/<role>-state.toml

[state]
role = "product-manager"
mode = "interview"
started = 2026-02-02T10:00:00Z
last_updated = 2026-02-02T10:30:00Z

[progress]
phase = "CURRENT_STATE"     # Which interview phase
questions_asked = 2
questions_remaining = 3

[gathered]
# Information collected so far
problem = "Build process takes 10+ minutes"
affected = "All developers on the team"
impact = "Slows iteration cycles, frustrates team"

[pending]
# Next questions to ask
next_question = "How does the build work today?"
remaining = ["Pain points?", "What should happen instead?"]

[decisions]
# Decisions made during this session
[[decisions.items]]
context = "Scope definition"
choice = "Focus on CI build, not local"
reason = "CI is the bottleneck per user"

[artifacts]
# Files being built
draft_requirements = """
### REQ-001: Fast CI Build

As a developer, I want CI builds to complete in under 2 minutes,
so that I get fast feedback on my changes.

**Acceptance Criteria:**
- [ ] CI build completes in < 2 minutes for typical PR
- [ ] Build cache is effective across runs
"""
```

### Resume Flow

```
1. Orchestrator reads context/<role>-state.toml
2. Orchestrator has user's answer to pending question
3. Orchestrator spawns new agent with prompt:

   "You are Product Manager in interview mode.

    Resume from this state:
    <contents of context/product-manager-state.toml>

    User's answer to your question "How does the build work today?":
    <user's answer>

    Continue the interview. When you need another answer, yield.
    When interview is complete, yield with type=complete."

4. Agent continues, updates state file, yields
5. Loop until complete
```

---

## Memory System

### Structure

```
~/.projctl/memory/
├── index.md              # Human-readable learnings (grep-able)
├── embeddings.db         # SQLite-vec for semantic search
├── sessions/
│   └── <project>-<date>.md   # Session summaries
└── decisions/
    └── <project>.jsonl   # Decision log
```

### Index Format

```markdown
# Memory Index

## Learnings

### 2026-02-02: projctl
- Event sourcing was overkill for this scale
- Property tests caught edge case with empty input
- Recursive descent simpler than table-driven for our grammar

### 2026-02-01: other-project
- React Query better than manual fetch for this use case
```

### Embeddings

- **Engine:** ONNX runtime with e5-small model (local, no API)
- **Storage:** SQLite-vec (single file, no server)
- **Indexed:** Session summaries + explicit learnings
- **Query:** "What do we know about caching?" → returns relevant memories

### Memory Commands

```bash
# Orchestrator extracts from result (not agent calling)
projctl memory extract --result context/product-manager-result.toml

# User says "remember this"
projctl memory learn --message "GraphQL adds complexity we don't need"

# Semantic query before spawning agent
projctl memory query "build performance patterns"

# Structural search
projctl memory grep "caching"

# Session end summary
projctl memory session-end --project myproject
```

### Memory Injection

Orchestrator queries memory before spawning agent:

```toml
# Injected into agent context
[memory]
relevant = [
    "Previous project: Gradle build cache reduced CI from 8min to 2min",
    "Learned: Incremental compilation key for Go builds",
]
query = "build performance"
```

---

## Territory Mapping

Before main work, cheap agent maps the codebase:

```bash
projctl territory map --dir . --output context/territory.toml
```

### Territory Format

```toml
# context/territory.toml (~500 tokens)

[structure]
root = "/Users/joe/repos/personal/projctl"
languages = ["go"]
build_tool = "mage"
test_framework = "go test + gomega + rapid"

[entry_points]
cli = "cmd/projctl/main.go"
public_api = "projctl.go"

[packages]
count = 12
internal = ["config", "context", "state", "trace", "memory"]

[tests]
pattern = "*_test.go"
count = 45

[docs]
readme = "README.md"
artifacts = ["docs/requirements.md", "docs/architecture.md"]

[patterns]
dependency_injection = true
table_driven_tests = true
property_tests = true
```

### When to Map

| Trigger | Why |
|---------|-----|
| `/project new` | Before PM interview, understand codebase |
| `/project adopt` | Before inference, map what exists |
| Task start | Before TDD, map relevant code |

### Cost Savings

| Without mapping | With mapping |
|-----------------|--------------|
| ~5000 tokens exploring | ~500 token map |
| Agent wanders | Agent focused |
| Repeated exploration | Cached map |

---

## State Management

### State File

```toml
# state.toml

[project]
name = "my-feature"
created = 2026-02-02T10:00:00Z
workflow = "new"            # new | adopt | align

[phase]
current = "requirements"    # See phase diagram
role = "product-manager"
mode = "interview"

[progress]
tasks_total = 0             # Filled after breakdown
tasks_complete = 0
current_task = ""

[yield]
pending = true
type = "need-user-input"
role = "product-manager"
context_file = "context/product-manager-state.toml"

[history]
[[history.entries]]
timestamp = 2026-02-02T10:00:00Z
phase = "init"

[[history.entries]]
timestamp = 2026-02-02T10:01:00Z
phase = "territory"

[[history.entries]]
timestamp = 2026-02-02T10:02:00Z
phase = "requirements"

[errors]
last_error = ""
retry_count = 0
max_retries = 3
```

### Phase Diagram

```
                    ┌─────────────────────────────────────┐
                    │              init                    │
                    └─────────────────┬───────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────┐
                    │           territory                  │
                    │     (background, cheap agent)        │
                    └─────────────────┬───────────────────┘
                                      │
                    ┌─────────────────┴───────────────────┐
                    │                                      │
              [new/adopt]                              [align]
                    │                                      │
                    ▼                                      ▼
     ┌──────────────────────────┐           ┌──────────────────────────┐
     │      requirements        │           │       analysis           │
     │   (product-manager)      │           │   (detect drift)         │
     └────────────┬─────────────┘           └────────────┬─────────────┘
                  │                                      │
                  ▼                                      ▼
     ┌──────────────────────────┐           ┌──────────────────────────┐
     │        design            │           │       update             │
     │      (designer)          │           │   (infer modes)          │
     │   [optional for CLI]     │           └────────────┬─────────────┘
     └────────────┬─────────────┘                        │
                  │                                      │
                  ▼                                      │
     ┌──────────────────────────┐                        │
     │      architecture        │                        │
     │      (architect)         │                        │
     └────────────┬─────────────┘                        │
                  │                                      │
                  ▼                                      │
     ┌──────────────────────────┐                        │
     │       breakdown          │                        │
     │       (planner)          │                        │
     └────────────┬─────────────┘                        │
                  │                                      │
                  ├──────────────────────────────────────┘
                  │
                  ▼
     ┌──────────────────────────┐
     │     implementation       │
     │  (implementer + QA)      │
     │                          │
     │  FOR EACH unblocked task:│
     │    tdd-red → commit      │
     │    tdd-green → commit    │
     │    tdd-refactor → commit │
     │    audit                 │
     │                          │
     │  CONTINUE until blocked  │
     │  or all complete         │
     └────────────┬─────────────┘
                  │
                  ▼
     ┌──────────────────────────┐
     │         retro            │
     │   (per-task + overall)   │
     └────────────┬─────────────┘
                  │
                  ▼
     ┌──────────────────────────┐
     │        complete          │
     │   (validate, summarize)  │
     └──────────────────────────┘
```

---

## Traceability Chain

```
ISSUE-NNN (optional)
    │
    ▼
REQ-NNN ←───────────────────────────────────────┐
(requirements.md)                               │
    │                                           │
    ├───────────────┐                           │
    ▼               ▼                           │
DES-NNN         ARCH-NNN                        │
(design.md)     (architecture.md)               │
[optional]          │                           │
    │               │                           │
    └───────┬───────┘                           │
            ▼                                   │
        TASK-NNN ──────────────────────────────►│
        (tasks.md)                              │
            │                                   │
            ▼                                   │
        TEST-NNN                                │
        (in test files)                         │
            │                                   │
            ▼                                   │
        Implementation ─────────────────────────┘
        (code files)        (traces back via comments/coverage)
```

Each artifact has `**Traces to:**` field:

```markdown
### REQ-001: Fast CI Build

As a developer, I want CI builds under 2 minutes...

**Traces to:** ISSUE-042
```

```markdown
### ARCH-003: Build Cache Strategy

Use content-addressable cache with...

**Traces to:** REQ-001
```

---

## Orchestrator Control Loop

```go
// Pseudocode for projctl pm

func main() {
    state := loadState("state.toml")

    for {
        // 1. Check for pending yield
        if state.Yield.Pending {
            answer := handleYield(state.Yield)
            state.Yield.Pending = false
        }

        // 2. Determine next action
        action := determineAction(state)

        switch action {
        case "spawn-agent":
            // 3. Prepare context
            territory := loadTerritory()
            memory := queryMemory(state.Phase.Current)
            context := buildContext(state, territory, memory, answer)

            // 4. Spawn agent
            yield := spawnAgent(state.Phase.Role, state.Phase.Mode, context)

            // 5. Handle result
            if yield.Type == "complete" {
                extractMemory(yield)
                advancePhase(state)
            } else {
                state.Yield = yield
                state.Yield.Pending = true
            }

        case "prompt-user":
            // Display question, get input
            answer = promptUser(state.Yield.Question)

        case "all-complete":
            runValidation()
            printSummary()
            return

        case "blocked":
            presentBlocker(state)
            return
        }

        // 6. Save state (crash recovery)
        saveState(state)

        // 7. Continue (relentless)
    }
}

func spawnAgent(role, mode string, context Context) Yield {
    prompt := buildPrompt(role, mode, context)

    // Invoke Claude CLI
    output := exec.Command("claude",
        "-p", prompt,
        "--output-format", "stream-json",
        "--verbose",
    ).Run()

    // Parse yield from output
    return parseYield(output)
}
```

---

## Model Routing

```toml
# config.toml

[routing]
# By role
product-manager = "sonnet"
designer = "sonnet"
architect = "opus"          # Complex decisions
planner = "sonnet"
implementer = "sonnet"
qa = "sonnet"
retro = "opus"              # Synthesis
tech-writer = "haiku"       # Straightforward

# By mode
interview = "sonnet"        # Needs nuance
infer = "haiku"             # Pattern matching
audit = "haiku"             # Checklist verification
execute = "sonnet"          # Code generation

# Overrides
territory-mapping = "haiku" # Fast, cheap
```

### Implementation

For sub-agents (Task tool), routing works directly.
For inline spawns, pass as hint in context.

---

## CLI Interface

```bash
# Start new project
projctl pm new "feature-name"

# Resume (after yield or crash)
projctl pm continue

# Status
projctl pm status

# Force specific phase
projctl pm phase requirements

# Skip optional phase
projctl pm skip design

# Memory operations
projctl memory learn --message "..."
projctl memory query "..."

# Territory
projctl territory map --dir .
projctl territory show

# Validation
projctl trace validate
projctl trace repair
```

---

## File Layout

```
project/
├── state.toml                    # Orchestrator state
├── context/
│   ├── territory.toml            # Codebase map
│   ├── product-manager-state.toml    # Role state (for resume)
│   ├── product-manager-yield.toml    # Pending yield
│   ├── architect-state.toml
│   └── ...
├── docs/
│   ├── requirements.md           # REQ-NNN
│   ├── design.md                 # DES-NNN (optional)
│   ├── architecture.md           # ARCH-NNN
│   └── tasks.md                  # TASK-NNN
└── .projctl/
    └── log.jsonl                 # Structured log

~/.projctl/
├── memory/
│   ├── index.md
│   ├── embeddings.db
│   ├── sessions/
│   └── decisions/
├── roles/
│   ├── product-manager.md        # Role prompt
│   ├── designer.md
│   ├── architect.md
│   └── ...
└── config.toml                   # Global config
```

---

## Implementation Plan

### Phase 1: Core Orchestrator

1. State management (state.toml read/write)
2. Yield protocol (parse/handle yields)
3. Agent spawning (claude CLI invocation)
4. Basic control loop

### Phase 2: Roles

1. Port product-manager (interview + infer + audit)
2. Port architect (interview + infer + audit)
3. Port implementer (tdd-red + tdd-green + tdd-refactor)
4. Port QA (task-audit + alignment-check)

### Phase 3: Support Systems

1. Territory mapping
2. Memory system (ONNX + SQLite-vec)
3. Model routing
4. TUI (bubbletea)

### Phase 4: Polish

1. Designer role
2. Planner role
3. Retro role
4. Tech Writer role
5. Graceful degradation
6. Cost tracking

---

## Migration from Current System

| Current | New |
|---------|-----|
| `/project` skill | `projctl pm` command |
| 19 skills | 9 roles with modes |
| Skills dispatch skills | Orchestrator dispatches all |
| Agent calls projctl | Orchestrator calls projctl |
| Context window state | TOML file state |
| Hope for continuation | Relentless by design |

### Deprecation Path

1. Build new system alongside existing
2. Test with one project end-to-end
3. If successful, archive old skills
4. Update CLAUDE.md to reference new system

---
