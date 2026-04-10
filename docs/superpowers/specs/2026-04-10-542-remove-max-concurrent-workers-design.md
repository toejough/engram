# Design: Remove MaxConcurrentWorkers Cap (Issue #542)

**Date:** 2026-04-10
**Issue:** #542 — binary enforces undocumented 3-worker cap that contradicts engram-tmux-lead skill's 9-pane design

---

## Problem

`internal/agent/agent.go` exports `MaxConcurrentWorkers = 3`. `internal/cli/cli_agent.go` enforces it at spawn time via `preSpawnGuards`, returning `errWorkerQueueFull` when a fourth agent is spawned. This blocks legitimate parallel workflows where the engram-tmux-lead skill orchestrates up to 8 workers (9 panes total: 1 coordinator + 8 agents).

The cap was introduced in Phase 5 planning without a corresponding spec, issue, or design doc. It has no documented rationale and contradicts the skill's own pane budget.

---

## Decision

Remove the cap entirely. The tmux layout is the correct enforcement boundary — the skill already limits pane allocation. Having the binary enforce a separate, lower cap creates two sources of truth that can drift.

---

## Changes

### `internal/agent/agent.go`

- Remove the `const MaxConcurrentWorkers = 3` declaration and its containing `const` block.

### `internal/cli/cli_agent.go`

- Remove `errWorkerQueueFull` sentinel error (line 46).
- Remove the cap check from `preSpawnGuards` (lines 586–588): the `if agentpkg.ActiveWorkerCount(state) >= agentpkg.MaxConcurrentWorkers` block.
- Update the `preSpawnGuards` doc comment to remove the mention of "worker queue limit" (lines 562–564).

### `internal/cli/cli_test.go`

- Delete `TestRunAgentSpawn_MaxWorkers_ReturnsError` entirely — it validates the removed behavior.

---

## What Is Not Changed

- `ActiveWorkerCount` function — still useful for observability/reporting; remains in place.
- Duplicate-name guard in `preSpawnGuards` — unrelated, stays.
- All other tests — unaffected.

---

## Acceptance Criteria

- `MaxConcurrentWorkers` constant is gone from `internal/agent/agent.go`.
- `errWorkerQueueFull` is gone from `internal/cli/cli_agent.go`.
- The cap check in `preSpawnGuards` is gone.
- `TestRunAgentSpawn_MaxWorkers_ReturnsError` is deleted.
- `targ check-full` passes with no errors or lint warnings.
- Spawning a 4th (or 9th) agent succeeds without a "worker queue full" error.
