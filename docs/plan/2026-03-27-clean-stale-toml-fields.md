# Plan: Clean Stale Data Fields (#396)

## Context

After #392, `[[links]]` and `LinkRecord` are already removed. The remaining work is removing `enforcement_level` and the entire escalation system — the enforcement reader was removed in hook consolidation, making all escalation code dead.

## Pre-conditions
- `[[links]]` — already cleaned by #392 (confirmed: no TOML files contain `[[links]]`)
- `LinkRecord` — already removed by #392

## Tasks

### Task 1: Remove `EnforcementLevel` and `TransitionRecord` from memory record
- `internal/memory/record.go`: Remove `EnforcementLevel string` field (~line 55), `TransitionRecord` struct (~lines 62-68)
- `internal/memory/record_test.go`: Remove `EnforcementLevel:` and `Transitions:` from test fixtures (~lines 70-73)

### Task 2: Delete escalation engine entirely
- `internal/maintain/escalation.go`: Delete entire file — `EscalationLevel` type, constants (`LevelAdvisory`, `LevelEmphasizedAdvisory`, `LevelReminder`), `EscalationEngine`, `EnforcementApplier` interface, `EscalationMemory` struct, `EscalationHistoryEntry`, escalation ladder, `nextEscalationLevel`, `prevEscalationLevel`, `predictImpact`
- `internal/maintain/escalation_test.go`: Delete entire file
- `internal/maintain/p6e_escalation_test.go`: Delete entire file

### Task 3: Remove escalation wiring from signal package
- `internal/signal/apply.go`: Remove `enforcementApplier` field (~line 30), `WithEnforcementApplier()` option (~lines 302-306), `escalationLadder` var (~line 347), and any escalation invocation logic
- `internal/signal/apply_test.go`: Remove `TestApply_EscalateCallsEnforcementApplier` (~lines 213-224), `stubEnforcementApplier` (~lines 617-624)
- `internal/signal/consolidate_transfer.go`: Remove enforcement max logic (~lines 16, 24-25, 48), `enforcementRank()` function (~line 57+)
- `internal/signal/consolidate_transfer_test.go`: Remove `TestTransferFields_MaxEnforcementLevel` (~lines 98-111)

### Task 4: Remove CLI enforcement applier
- `internal/cli/signal.go`: Remove `funcEnforcementApplier` type and `SetEnforcementLevel` method (~lines 57-59), the write path at `record.EnforcementLevel = level` (~line 314)
- `internal/cli/signal_test.go`: Remove `TestFuncEnforcementApplier_SetEnforcementLevel` (~line 51), fixture strings with `enforcement_level` (~line 292)
- `internal/cli/cli_test.go`: Remove `enforcement_level` from inline TOML fixture (~line 2125)

### Task 5: Strip `enforcement_level` from runtime memory TOML files
- Find all `.toml` memory files (in user data dirs, not in repo) that contain `enforcement_level` — these are runtime files, so this is a note for the user, not a code change
- OR: add a migration that strips the field on read (if the TOML parser ignores unknown fields, this may be unnecessary)

### Task 6: Verify
- `targ test` passes
- `targ check-full` passes (or only pre-existing failures)
- No references to `EnforcementLevel`, `enforcement_level`, `EscalationLevel`, `EscalationEngine`, `EnforcementApplier`, `TransitionRecord` remain in Go code (docs/specs prose is fine)

## Acceptance Criteria (from issue, updated)
- `[[links]]` — already done (#392)
- `enforcement_level` field removed from `MemoryRecord` struct
- Escalation engine and all related types/interfaces/tests deleted
- All tests pass

## Risk
Removing the escalation engine is larger than the issue anticipated. The entire `maintain/escalation.go` and its test files become dead code once `EnforcementLevel` is removed from the record. Confirm with Joe if he wants escalation removed entirely or just the field.
