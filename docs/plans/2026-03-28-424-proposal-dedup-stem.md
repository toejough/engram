# Fix: Proposal Dedup Matches on Directive Stem (#424)

## Problem

`FilterDuplicateProposals` and `DeduplicateProposed` use full `Directive+Dimension` as the dedup key. Directives embed run-specific stats (e.g. `"retire pol-021: follow rate 37% (was 38%), mean effectiveness 21.2 (was 21.9)"`), so each run produces a unique string. This causes 16 unique retire proposals × ~28 runs = 448 duplicates.

## Design

Add a `DirectiveStem` function to `policy` package that extracts the semantic action from a directive by truncating at the first colon. Both dedup functions use this stem as the key.

```
"retire pol-021: follow rate 37% ..." → "retire pol-021"
"de-prioritize keyword \"spec\": 88% ..."  → "de-prioritize keyword \"spec\""
"Extract only actionable insights"  → "Extract only actionable insights" (no colon = unchanged)
```

## Steps

### S1: Add `DirectiveStem` to `policy` package (TDD)

**File:** `internal/policy/policy.go`, `internal/policy/policy_test.go`

- Add `func DirectiveStem(directive string) string` — returns substring before first `:`, trimmed
- If no colon, returns full string
- Tests: with stats suffix, without colon, empty string, colon at start

### S2: Update `DeduplicateProposed` to use stem key (TDD)

**File:** `internal/policy/policy.go`, `internal/policy/policy_test.go`

- Change key from `pol.Directive` to `DirectiveStem(pol.Directive)`
- Add test case: two proposed policies with same stem but different stats → second is removed

### S3: Update `FilterDuplicateProposals` to use stem key (TDD)

**File:** `internal/cli/cli.go`, `internal/cli/adapt_test.go`

- Change key from `pol.Directive`/`prop.Directive` to `policy.DirectiveStem(...)`
- Add test case: existing has `"retire pol-001: follow rate 40%"`, new proposal has `"retire pol-001: follow rate 38%"` → filtered out

### S4: Run dedup to clean existing duplicates

- Run `engram adapt --dedup` to clean the 448 existing duplicates
- Verify `engram adapt list | grep proposed | wc -l` drops to ~23

### S5: Run `targ check-full`

- Verify all lints, tests, coverage pass
