# projctl

<!-- Traces: REQ-1 -->

A dependable Claude Code agent orchestrator with maximum autonomy, minimum intervention, and confident traceability from idea to implementation.

## Overview

<!-- Traces: REQ-1, DES-1, ARCH-18 -->

projctl is a CLI tool that orchestrates Claude Code agents through structured workflows for software development projects. It provides:

- **Maximum autonomy** with minimum human intervention through intelligent agent coordination
- **Cheapest operation** using smallest context and most cost-effective models (Haiku → Sonnet → Opus)
- **Deterministic tooling** over LLM judgment for consistency and reliability
- **Learning from corrections** to improve future behavior
- **Behavioral correctness** enforced through tests
- **Confident traceability** from requirements through to implementation
- **Support for all project types**: existing codebases, alignment tasks, and new projects

## Installation

<!-- Traces: REQ-1 -->

### Prerequisites

- Go 1.25.6 or later
- Git
- [targ](https://github.com/toejough/targ) (optional, for build automation)

### Install from Source

```bash
# Clone the repository
git clone https://github.com/toejough/projctl.git
cd projctl

# Option 1: Using targ (recommended)
targ install

# Option 2: Using go install directly
go install ./cmd/projctl
projctl skills install
```

### Verify Installation

```bash
projctl --help
```

## Quick Start

<!-- Traces: REQ-1, ARCH-1, ARCH-12, ARCH-13 -->

### Initialize a Project

```bash
# Initialize project state and configuration
projctl state init
projctl config init

# Check current state
projctl state get
```

### Create an Issue

```bash
# Create a new issue to track work
projctl issue create --title "Add user authentication" \
  --description "Users need to log in with email/password"
```

### Run the Orchestrator

<!-- Traces: REQ-016, ARCH-042, ARCH-048 -->

The `/project` orchestrator uses a **two-role architecture** for cost optimization:

**Team Lead (Opus)**
- Spawns and coordinates teammates
- Validates model handshakes
- Handles user interaction
- Manages team lifecycle (TeamCreate/TeamDelete)
- **Delegates all file operations** to teammates

**Orchestrator Teammate (Haiku)**
- Runs the mechanical `projctl step next` → dispatch → `step complete` loop
- Manages state persistence
- Sends spawn requests to team lead
- Coordinates producer/QA pairs

The orchestrator automatically:

1. Analyzes requirements and gathers context
2. Produces design and architecture artifacts
3. Breaks work into tasks
4. Implements each task using TDD discipline
5. Updates documentation
6. Generates retrospective and summary

Each phase uses a producer-QA pair loop for quality assurance before advancing.

**Why two roles?** Separating team ownership (opus) from mechanical execution (haiku) reduces costs by ~80% while preserving opus context for high-value user interaction.

### Check Progress

```bash
# View current project state
projctl state get

# List all issues
projctl issue list

# View project log
projctl log read

# Check token usage against budget
projctl usage check --dir .
```

## Core Concepts

<!-- Traces: REQ-1, ARCH-18 -->

### Phases and Workflows

<!-- Traces: ARCH-34, ARCH-35, ARCH-36 -->

projctl organizes work into phases, each with a dedicated producer and QA agent:

| Phase | Purpose | Artifacts |
|-------|---------|-----------|
| **PM** | Discover problems, produce requirements | `docs/requirements.md` |
| **Design** | Define user experience | `docs/design.md` |
| **Architecture** | Define technical approach | `docs/architecture.md` |
| **Task Breakdown** | Decompose into executable tasks | `docs/tasks.md` |
| **Implementation** | Build with test-first discipline | Code, tests |
| **Documentation** | Update repo-level docs | README, API docs |
| **Retrospective** | Capture learnings, file issues | `docs/retro.md` |
| **Summary** | Summarize accomplishments | `docs/summary.md` |

#### TDD Loop Structure

<!-- Traces: ARCH-34, ARCH-35, ARCH-36 -->

The implementation phase uses a structured TDD loop with per-phase QA and commits. Each TDD sub-phase (red, green, refactor) has its own producer/QA pair followed by commit validation:

```
tdd-red → tdd-red-qa → commit-red → commit-red-qa →
tdd-green → tdd-green-qa → commit-green → commit-green-qa →
tdd-refactor → tdd-refactor-qa → commit-refactor → commit-refactor-qa →
task-audit
```

**Why per-phase QA?** Immediate feedback catches issues early. If tdd-red writes bad tests, you find out before green/refactor happen. Smaller QA scope per phase keeps validation focused and fast.

**TDD Sub-Phases:**

| Phase | Producer Action | QA Validation | Commit Scope |
|-------|----------------|---------------|--------------|
| **tdd-red** | Write failing tests | Verify tests fail for right reasons | Test files only |
| **tdd-green** | Minimal implementation to pass tests | Verify tests pass, no over-implementation | Tests + implementation |
| **tdd-refactor** | Improve code quality | Verify tests still pass, code improved | Implementation only |

**Commit Validation:**

<!-- Traces: ARCH-39, ARCH-40 -->

Each commit phase uses `commit-producer` skill to create phase-scoped commits, followed by `commit-qa` validation:

- **Staging rules** - Only files modified in current phase (prevents cross-phase contamination)
- **Secret detection** - Catches .env, credentials, API keys before commit
- **Message format** - Conventional commits with AI-Used trailer
- **Lint suppressions** - Flags blanket suppressions (no mass nolint additions)

State machine enforcement prevents shortcuts (e.g., cannot skip from tdd-red directly to commit-red without tdd-red-qa).

### Parallel Task Execution

<!-- Traces: ISSUE-120, REQ-1, REQ-2, REQ-5, DES-1, DES-6, ARCH-2 -->

During the implementation phase, `projctl step next` detects all unblocked tasks and enables parallel execution through git worktrees:

**Immediate Execution Model:**

- After any task completes, orchestrator calls `projctl step next`
- All currently unblocked tasks are returned immediately (no batching)
- Orchestrator spawns parallel work as soon as tasks become unblocked
- Dynamic work discovery: completing task A may unblock tasks B and C for parallel execution

**Response Format:**

```json
{
  "tasks": [
    {"id": "TASK-1", "command": "projctl run TASK-1", "worktree": "/path/worktrees/TASK-1"},
    {"id": "TASK-2", "command": "projctl run TASK-2", "worktree": "/path/worktrees/TASK-2"}
  ]
}
```

- **Empty array**: No tasks unblocked, wait for running tasks to complete
- **Single task with `worktree: null`**: Sequential execution on main branch
- **Multiple tasks with worktree paths**: Parallel execution in isolated worktrees

**Conflict Resolution:**

No pre-execution conflict detection. Git detects conflicts during merge and escalates to user for standard git conflict resolution workflow.

### Team Communication Protocol

<!-- Traces: ARCH-18, ARCH-043 -->

**Orchestrator ↔ Team Lead Communication:**
- Orchestrator sends spawn requests via SendMessage with full task_params JSON
- Team lead spawns teammates via Task tool and validates model handshakes
- Team lead confirms successful spawns back to orchestrator
- On project completion, orchestrator sends shutdown requests to team lead

**Producer/QA ↔ Team Lead Communication:**

Skills communicate with the team lead through the `SendMessage` and `AskUserQuestion` tools. Producer message types:

- `complete` - Work finished successfully (send completion message with artifact paths)
- `blocked` - Cannot proceed without resolution (send blocker message with details)
- User input needed - Use `AskUserQuestion` tool directly
- Context needed - Send message requesting information or use `AskUserQuestion`

QA message types:

- `approved` - Work passes quality checks (send approval message)
- `improvement-request` - Producer needs to fix issues (send message with findings)
- `escalate-phase` - Problem in prior phase artifact (send escalation with proposed changes)
- User input needed - Use `AskUserQuestion` tool directly

### Traceability Chain

<!-- Traces: REQ-1, ARCH-18 -->

Every artifact traces to upstream work using inline `**Traces to:**` fields:

```markdown
### REQ-3: User Authentication

Users must be able to log in with email and password.

**Traces to:** ISSUE-15
```

Validate traceability:

```bash
projctl trace validate --dir .
```

### State Management

<!-- Traces: ARCH-12, ARCH-13 -->

The orchestrator tracks project state to enable resumption and parallel work:

```bash
# View current state
projctl state get

# Manual state transitions (usually not needed)
projctl state transition --from pm --to design
```

State includes:
- Current phase and sub-phase
- Active tasks and their status
- Pair loop iteration counts
- Escalation and conflict records

## CLI Commands

<!-- Traces: REQ-1, ARCH-2 -->

### State Commands

```bash
projctl state init              # Initialize project state file
projctl state get               # Show current state
projctl state set               # Update state fields
projctl state transition        # Transition to new state
projctl state next              # Determine next action
projctl state retry             # Re-attempt failed transition
projctl state complete          # Mark task complete
```

### Configuration

```bash
projctl config init             # Initialize project config
projctl config get              # Get configuration value
projctl config path             # Show config file path
```

### Issue Management

```bash
projctl issue create            # Create new issue
projctl issue update            # Update existing issue
projctl issue list              # List all issues
projctl issue get               # Get issue details
```

### Traceability

```bash
projctl trace validate          # Validate traceability links
projctl trace repair            # Detect and report issues
projctl trace show              # Show traceability graph
projctl trace promote           # Promote TASK traces to artifact IDs
```

### Memory System

<!-- Traces: ARCH-14 -->

```bash
projctl memory learn            # Store a learning
projctl memory decide           # Log a decision with reasoning
projctl memory query            # Find semantically similar memories
projctl memory grep             # Search memory files by pattern
projctl memory extract          # Extract from result files
projctl memory session-end      # Generate session summary
```

### Context and Territory

<!-- Traces: ARCH-7 -->

```bash
projctl context check           # Check context budget usage
projctl territory map           # Generate compressed territory map
projctl territory show          # Show cached territory map
```

### Code Quality

```bash
projctl coverage analyze        # Analyze test coverage
projctl coverage report         # Generate coverage report
projctl refactor rename         # Rename symbol using LSP
projctl refactor extract-function  # Extract function using LSP
```

### Visual Verification

<!-- Traces: ARCH-15 -->

```bash
projctl screenshot diff         # Compare screenshots for differences
```

### Usage Tracking

<!-- Traces: ARCH-4 -->

```bash
projctl usage report            # Generate token usage report
projctl usage check             # Check against budget thresholds
```

### Skills Management

<!-- Traces: ARCH-16 -->

```bash
projctl skills list             # List available skills
projctl skills install          # Install skills (create symlinks)
projctl skills status           # Show installation status
projctl skills uninstall        # Uninstall skills
projctl skills docs             # Show full skill documentation
```

### Other Commands

```bash
projctl log write               # Write log entry
projctl log read                # Read log entries with filtering
projctl conflict create         # Create conflict record
projctl conflict check          # Check for unresolved conflicts
projctl conflict list           # List all conflicts
projctl escalation write        # Write escalation record
projctl escalation review       # Review pending escalations
projctl escalation resolve      # Resolve escalation by ID
projctl worktree create         # Create worktree for task
projctl worktree merge          # Merge task worktree
projctl retro extract           # Extract retro recommendations
projctl step next               # Get next orchestration action (JSON with parallel tasks)
projctl step complete           # Record step result
projctl step complete --reportedmodel <model>  # Record failed spawn with model mismatch
```

### Step Orchestration

<!-- Traces: ARCH-2, ISSUE-120 -->

The `projctl step next` command returns all currently unblocked tasks for immediate parallel execution:

```bash
# Get next action(s) to execute
projctl step next
```

**Response Format:**

```json
{
  "action": "spawn-producer",
  "tasks": [
    {
      "id": "TASK-1",
      "command": "projctl run TASK-1",
      "worktree": "/repo/.git/worktrees/TASK-1"
    },
    {
      "id": "TASK-2",
      "command": "projctl run TASK-2",
      "worktree": "/repo/.git/worktrees/TASK-2"
    }
  ]
}
```

**Parallel Execution:**

- **Empty array** (`[]`) - No tasks unblocked, orchestrator waits
- **Single task** (`worktree: null`) - Sequential execution on main branch
- **Multiple tasks** (`worktree: path`) - Parallel execution in worktrees

The orchestrator calls `projctl step next` after each task completion to discover newly unblocked work, enabling immediate parallel execution without batching delays.

**Worktree Lifecycle:**

For parallel tasks, the orchestrator:

1. Creates worktrees using paths from response
2. Executes task commands in worktree directories
3. Merges worktree branches back to main
4. Cleans up worktrees after merge

Conflicts during merge are detected by git and escalated to the user for resolution.

## Configuration

<!-- Traces: REQ-1 -->

Project configuration lives in `project-config.toml`:

```toml
[project]
name = "my-project"
type = "existing"  # or "new", "alignment"

[docs]
dir = "docs"
requirements_path = "docs/requirements.md"
design_path = "docs/design.md"
architecture_path = "docs/architecture.md"
tasks_path = "docs/tasks.md"

[budget]
warning_tokens = 50000
limit_tokens = 100000

[memory]
index_path = ".projctl/memory/index.sqlite"
embeddings_path = ".projctl/memory/embeddings"
```

Initialize with defaults:

```bash
projctl config init
```

## Skills

<!-- Traces: ARCH-16, ARCH-18 -->

Skills are located in `~/.claude/skills/` and define agent behaviors. Key skills:

- **project** - Main orchestrator skill (invoked via `/project`). Uses two-role architecture: team lead (opus) coordinates, orchestrator teammate (haiku) runs step loop.
- **qa** - Universal QA skill that validates producers against contracts
- **pm-interview-producer** / **pm-infer-producer** - Requirements gathering
- **design-interview-producer** / **design-infer-producer** - Design specification
- **arch-interview-producer** / **arch-infer-producer** - Architecture definition
- **breakdown-producer** - Task decomposition
- **tdd-red-producer** - Write failing tests
- **tdd-green-producer** - Make tests pass
- **tdd-refactor-producer** - Improve code quality
- **commit-producer** - Create phase-scoped git commits (red/green/refactor)
- **doc-producer** - Documentation generation
- **alignment-producer** - Align artifacts with changes
- **retro-producer** - Generate retrospective
- **summary-producer** - Generate summary

Each skill follows the GATHER → SYNTHESIZE → PRODUCE pattern and communicates via `SendMessage`.

## Model Routing

<!-- Traces: ARCH-3 -->

projctl automatically selects the most cost-effective model for each task:

- **Haiku** - Simple validation, QA checks, territory mapping
- **Sonnet** - Most producers, complex analysis
- **Opus** - Complex design decisions, architecture (when explicitly needed)

Skills declare their model in frontmatter. The orchestrator respects these declarations for optimal cost/quality balance.

### Spawn Request Protocol

<!-- Traces: REQ-017, REQ-021, ARCH-043, DES-003, DES-004, DES-005 -->

The orchestrator teammate cannot spawn other agents directly (only team owners can spawn). Instead, it sends spawn requests to the team lead:

1. **Orchestrator** detects `spawn-producer` or `spawn-qa` action from `projctl step next`
2. **Orchestrator** composes SendMessage with spawn request containing full `task_params` JSON:
   ```json
   {
     "subagent_type": "general-purpose",
     "name": "tdd-red-producer",
     "model": "sonnet",
     "prompt": "First, respond with your model name...",
     "team_name": "issue-104"
   }
   ```
3. **Team lead** extracts task_params and calls Task tool to spawn teammate
4. **Team lead** validates model handshake (see below)
5. **Team lead** sends spawn confirmation back to orchestrator

This delegation pattern keeps the team lead (opus) thin while enabling the orchestrator (haiku) to run the full workflow autonomously.

### Model Handshake Enforcement

<!-- Traces: REQ-021, ARCH-043 -->

When spawning a teammate, the team lead validates that the spawned agent is running the correct model:

1. `step next` includes an `expected_model` field and prepends handshake instruction to task prompt
2. Teammate must report its model name as its first message
3. **Team lead** performs case-insensitive substring match against `expected_model`

**On handshake success:**
- Team lead sends "spawn-confirmed" message to orchestrator
- Teammate proceeds with work
- Team lead calls `step complete --action spawn-producer --status done`

**On handshake failure:**
- Team lead sends "spawn-failed: wrong model" message to orchestrator with details
- Team lead calls `step complete --status failed --reportedmodel <model>` immediately
- Failed model appended to `FailedModels` list, `SpawnAttempts` incremented
- Orchestrator retries spawn (up to 3 attempts)
- After 3 failures, `step next` returns `escalate-user` action with full details

On successful spawn, `SpawnAttempts` resets to 0 and `FailedModels` is cleared.

### Task Parameters

For spawn actions, `step next` output includes a `task_params` object with the exact parameters for the Claude Code Task tool call:

```json
{
  "subagent_type": "code",
  "name": "tdd-red-producer",
  "model": "claude-sonnet-4-5-20250929",
  "prompt": "First, respond with your model name so I can verify you're running the correct model.\n\nThen invoke /tdd-red-producer.\n\n..."
}
```

The `expected_model` field at the top level tells the orchestrator what model to validate against the teammate's handshake response.

## Development

### Build

```bash
# Using targ
targ install

# Or using go directly
go build ./cmd/projctl
```

### Test

```bash
go test ./...
```

### Code Quality

```bash
# Analyze coverage
projctl coverage analyze --dir .

# Check traceability
projctl trace validate --dir .
```

## Architecture

<!-- Traces: ARCH-1, ARCH-12, ARCH-13, ARCH-18, ARCH-042 -->

projctl uses a structured result format where all skills return TOML files with:

- **Status** - Success/failure indicator
- **Outputs** - Artifacts produced (file paths, IDs)
- **Decisions** - Key choices made with reasoning
- **Learnings** - Patterns discovered for future use

### Two-Role Orchestration

The orchestrator operates as a control loop split between two roles:

**Team Lead (Opus) - High-Level Coordinator:**
1. Owns team lifecycle (TeamCreate, TeamDelete)
2. Receives spawn requests from orchestrator teammate
3. Spawns teammates via Task tool
4. Validates model handshakes
5. Relays spawn confirmations
6. Handles user interaction and escalations

**Orchestrator Teammate (Haiku) - Mechanical Execution:**
1. Read current state via `projctl step next`
2. Determine next action from JSON output
3. Send spawn requests to team lead (or handle directly for non-spawn actions)
4. Receive spawn confirmations from team lead
5. Update state via `projctl step complete`
6. Resume or advance as needed

State machine preconditions prevent skipping workflow steps, ensuring deterministic behavior. State persistence enables resumption if the orchestrator teammate terminates mid-session.

## Traceability

<!-- Traces: REQ-1 -->

All artifacts use inline traceability. Each section with an ID includes a `**Traces to:**` field:

```markdown
### DES-5: Login Form Layout

The login form has email and password fields with a submit button.

**Traces to:** REQ-3
```

Validate the traceability chain:

```bash
projctl trace validate --dir .
```

This ensures:
- No orphan references (traced-to IDs exist)
- No unlinked IDs (defined IDs are connected to chain)
- Complete chain from ISSUE → REQ → DES → ARCH → TASK → TEST

## Learning System

<!-- Traces: ARCH-5, ARCH-14 -->

projctl learns from corrections to improve future behavior:

1. **Log corrections** - Record when user corrects agent output
2. **Detect patterns** - Analyze corrections for recurring issues
3. **Store learnings** - Persist discoveries to memory system
4. **Apply context** - Inject relevant learnings for future tasks

```bash
# Log a correction
projctl corrections log --context "test naming" \
  --correction "Use descriptive test names, not Test1/Test2"

# Analyze patterns
projctl corrections analyze

# Query relevant learnings
projctl memory query "test naming conventions"
```

## Token Budget

<!-- Traces: ARCH-4 -->

Track and control token usage:

```bash
# Check current usage
projctl usage report --dir .

# Validate against budget
projctl usage check --dir .
```

Set thresholds in `project-config.toml`:

```toml
[budget]
warning_tokens = 50000  # Warn when exceeded
limit_tokens = 100000   # Fail when exceeded
```

## Contributing

projctl is a personal tool for orchestrating Claude Code agents. Issues and learnings are tracked through the tool's own issue management system:

```bash
projctl issue create --title "..." --description "..."
```

## License

<!-- Traces: REQ-1 -->

See LICENSE file for details.

## Documentation

<!-- Traces: REQ-1 -->

- [Requirements](docs/requirements.md) - Detailed requirements with traceability
- [Design](docs/design.md) - Design decisions and user experience
- [Architecture](docs/architecture.md) - Technical architecture and decisions
- [Orchestration System](docs/orchestration-system.md) - Complete orchestration reference
- [Tasks](docs/tasks.md) - Implementation task breakdown

## Support

projctl is a personal tool. For questions or issues, create an issue via:

```bash
projctl issue create --title "..." --description "..."
```
