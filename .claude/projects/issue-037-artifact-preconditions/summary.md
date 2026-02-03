# Project: issue-037-artifact-preconditions

**Issue:** ISSUE-037 - State transitions should enforce artifact preconditions
**Workflow:** task
**Status:** complete

## Summary

Added preconditions to enforce that artifact files exist before phase completion:
- `retro-complete` now requires `retro.md` to exist
- `summary-complete` now requires `summary.md` to exist

This prevents skipping phases without producing the required outputs.

## Changes

1. `internal/state/state.go` - Added `RetroExists` and `SummaryExists` to PreconditionChecker interface, added preconditions
2. `internal/state/state_test.go` - Added tests for artifact preconditions
3. `internal/state/transition_test.go` - Updated mock to implement new interface methods
4. `cmd/projctl/checker.go` - Implemented `RetroExists` and `SummaryExists` in DefaultChecker

## Commits

- `97d540e` - test: add failing tests for artifact preconditions
- `ce126e3` - feat: add artifact preconditions for retro-complete and summary-complete
