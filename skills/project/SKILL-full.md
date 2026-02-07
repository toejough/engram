# Project Orchestrator - Full Documentation

This document provides detailed orchestrator behavior, state persistence, resumption flow, and error handling for the two-role project orchestrator architecture.

## Two-Role Architecture

The project orchestrator uses a two-role split:

1. **Team Lead (Opus)** - Spawns teammates, coordinates high-level flow, delegates all work
2. **Orchestrator Teammate (Haiku)** - Runs the mechanical step loop, manages state persistence

This split achieves 30x cost savings by moving mechanical orchestration work from opus to haiku.

## Orchestrator Behavior

The orchestrator teammate is a mechanical step loop agent that:

1. Reads next action from `projctl step next`
2. Executes the action (spawn producer/QA, commit, transition)
3. Reports completion via `projctl step complete`
4. Loops until `all-complete` action received

### Spawn Request Protocol

When `projctl step next` returns `spawn-producer` or `spawn-qa`:

1. Orchestrator composes SendMessage with spawn request
2. Message includes `task_params` JSON, `expected_model`, `action`, `phase`
3. Team lead receives request and spawns teammate via Task tool
4. Team lead validates model handshake
5. Team lead sends spawn confirmation back to orchestrator
6. Orchestrator proceeds with teammate's work

### Error Handling and Retry-Backoff

The orchestrator implements retry logic with exponential backoff:

- Retry delays: 1s, 2s, 4s (after attempts 1, 2, 3)
- Wraps `projctl step next` and `projctl step complete` calls
- After 3 failed attempts, escalates to team lead
- Team lead presents error to user via AskUserQuestion

## State Persistence

The orchestrator owns state persistence via `projctl state` commands:

- `projctl state init` - Initialize new project state
- `projctl state set` - Update workflow type, phase, iteration
- `projctl state get` - Read current state for resumption

State file location: `.claude/projects/<project-name>/state.toml`

State includes:
- Current phase and sub-phase
- Workflow type (new/task/adopt/align)
- Active issue
- Pair loop iteration count
- QA verdict and feedback

The team lead never touches state files - this is the orchestrator's exclusive responsibility.

## Resumption Flow

When the orchestrator terminates unexpectedly:

1. Team lead detects termination (no response, error signal)
2. Team lead checks for state file existence
3. If state exists, team lead respawns orchestrator with same params
4. Respawned orchestrator reads state via `projctl state get --format json`
5. Orchestrator skips init if state.phase is not empty
6. Orchestrator resumes from last saved phase without repeating work

This ensures long-running projects can survive context limits and interruptions.

## Shutdown Protocol

When `projctl step next` returns `all-complete`:

1. Orchestrator sends "all-complete" message to team lead with project summary
2. Team lead runs end-of-command sequence
3. Team lead sends shutdown_request to all active teammates (including orchestrator)
4. Team lead waits for shutdown confirmations
5. Team lead calls TeamDelete after all confirmations
6. Team lead reports completion to user

## Model Handshake Validation

After spawning any teammate:

1. Team lead reads teammate's first message
2. Performs case-insensitive substring match of `expected_model`
3. On match: sends spawn confirmation to orchestrator, proceeds with work
4. On mismatch: calls `projctl step complete --status failed --reported-model "<model>"`, sends failure message to orchestrator

This prevents wrong-model executions from wasting tokens and producing incorrect artifacts.
