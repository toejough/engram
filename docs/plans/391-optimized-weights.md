# Implementation Plan: Apply Optimized Surfacing Weights (#391)

Apply the coordinate-ascent-optimized weights from the playground experiment to production
scoring in `internal/frecency/frecency.go` and `internal/surface/surface.go`.

## Changes

### 1. `internal/frecency/frecency.go` — constant updates

| Constant | Old | New |
|---|---|---|
| `defaultAlpha` | `1.0` | `0` |
| `defaultWEff` | `1.5` | `0.3` |
| `defaultWRec` | `0.5` | `0` |
| `defaultWFreq` | `1.0` | `1.0` (unchanged) |

Add new constants:
```go
defaultWTier     = 0.3
defaultTierABoost = 1.2
defaultTierBBoost = 0.2
// Tier C boost is 0 (no constant needed)
```

Add `wTier`, `tierABoost`, `tierBBoost` fields to `Scorer` struct and wire them in `New`.

Add `Tier` field to `frecency.Input` (string: `"A"`, `"B"`, `"C"`, or `""`).

Update `Quality` to include tier signal:
```go
func (s *Scorer) Quality(input Input) float64 {
    return s.wEff*s.effectiveness(input) +
        s.wRec*s.recency(input) +
        s.wFreq*s.frequency(input) +
        s.wTier*s.tierBoost(input)
}
```

Add `tierBoost` method:
```go
func (s *Scorer) tierBoost(input Input) float64 {
    switch input.Tier {
    case "A":
        return s.tierABoost
    case "B":
        return s.tierBBoost
    default:
        return 0
    }
}
```

### 2. `internal/surface/surface.go` — constant updates

| Constant | Old | New |
|---|---|---|
| `minRelevanceScore` | `0.05` | `0` (remove BM25 floor) |
| `coldStartBudget` | `1` | `2` |
| `irrelevancePenaltyHalfLife` | `5` | `5` (unchanged) |
| `promptLimit` | `2` | `2` (unchanged) |

Note: `unprovenBM25FloorPrompt = 0.20` is for unproven memories only and is not in the
optimized param table — leave it unchanged unless the issue is updated.

Update `toFrecencyInput` to pass `Confidence` from `memory.Stored` as `Tier`:
```go
func toFrecencyInput(mem *memory.Stored) frecency.Input {
    return frecency.Input{
        ...
        Tier: mem.Confidence,
    }
}
```

`memory.Stored` does not currently have a `Confidence` field. Add it to `internal/memory/memory.go`:
```go
Confidence string // "A", "B", or "C"
```

Verify the TOML reader already populates `Confidence` (check `internal/store` or equivalent).

Also update the gen penalty strength: find where `GenFactor` cross-project penalties are
applied. The issue calls for gen penalty strength `1.5` (was `1.0`). Locate how penalty
factors are combined and introduce a configurable strength multiplier, or adjust the
`penaltyByGeneralizability` table to reflect a `1.5×` scaling of the penalty gap from 1.0.

**Concrete genfactor change:** The current table encodes penalty as a multiplier. "Strength 1.5"
means the distance from 1.0 is scaled by 1.5. New values:

| Gen | Old | New (strength=1.5) |
|---|---|---|
| 5 (universal) | 1.0 | 1.0 |
| 4 (similar) | 0.8 | 0.7 |
| 3 (moderate) | 0.5 | 0.25 |
| 2 (narrow) | 0.2 | 0.0 (floor at 0) |
| 1 (this-project) | 0.05 | 0.0 |
| 0 (unset) | 0.5 | 0.25 |

Add a `genPenaltyStrength` constant (`1.5`) and derive the table values from it, or just
update the literal values. The simpler approach (given the table is already hardcoded) is to
update the literals and add a comment referencing the strength.

## TDD Approach

### Red phase — update existing tests to expect new values, add new tests

**`internal/frecency/frecency_test.go`:**

1. `TestQuality_AllSignals` — update expected formula from
   `1.5*eff + 0.5*recency + 1.0*freq` to
   `0.3*eff + 0*recency + 1.0*freq + 0.3*tierBoost`.
   Use `Tier: "A"` → `tierBoost = 1.2` in one subcase, `Tier: "B"` → `0.2` in another,
   `Tier: ""` → `0` in another.

2. `TestQuality_NoEvaluations` — update expected formula similarly.

3. `TestRecency_HalfLife` — the extraction formula changes:
   `quality = 0.3*0.5 + 0*recency + 0 + 0 = 0.15` when freq=0, tier="".
   Recency weight is now 0, so this test can only verify recency has no effect.
   Rewrite to test that `Quality` is identical for zero-days-ago vs seven-days-ago
   when all other inputs are equal (recency signal is disabled).

4. `TestFrequency_Normalized` — update extraction formula.
   Old: `freq = quality - 0.75`. New: `freq = quality - 0.3*0.5 = quality - 0.15`
   (since wEff=0.3, defaultEffectiveness=0.5, wRec=0).

5. `TestCombinedScore_Basic` — update alpha from 1.0 to 0:
   expected = `(2.0*1.0 + 0*0.5) * (1.0 + quality)`.

6. `TestCombinedScore_SpreadingOnly` — alpha=0 means spreading-only input yields 0.
   Rewrite to verify spreading is effectively disabled with default alpha=0.

7. Add `TestQuality_TierBoost_A` — `Tier: "A"` contributes `0.3*1.2 = 0.36`.
8. Add `TestQuality_TierBoost_B` — `Tier: "B"` contributes `0.3*0.2 = 0.06`.
9. Add `TestQuality_TierBoost_C` — `Tier: "C"` contributes `0`.
10. Add `TestQuality_TierBoost_Empty` — `Tier: ""` contributes `0`.

**`internal/surface/cold_start_budget_test.go`:**

- `TestColdStartBudgetLimitsUnprovenPromptMemories` asserts `Equal(1)`.
  Update to `Equal(2)` after `coldStartBudget` changes to 2.

**`internal/surface/genfactor_test.go`:**

- Update expected values for gen=4 (`0.7`), gen=3 (`0.25`), gen=2 (`0.0`), gen=1 (`0.0`),
  gen=0 (`0.25`) to match strength=1.5 penalty table.

**`internal/surface/helpers_test.go` / `surface_test.go`:**

- Check any test that embeds the old `minRelevanceScore=0.05` floor logic. With floor=0,
  memories that previously scored below 0.05 (but above 0) will now surface. Update
  expectations accordingly (or note that the floor test becomes a no-op).

### Green phase — update production code

Apply all constant and struct changes listed under Changes above.

### Refactor phase

- If `wTier=0` and `tierABoost=0` and `tierBBoost=0` all become constants together,
  consider grouping them as a `TierWeights` struct option — but only if that simplifies
  the `Scorer` struct. Otherwise leave as flat fields.
- Confirm `targ check-full` passes with no lint issues.

## Files to Change

1. `/Users/joe/repos/personal/engram/internal/frecency/frecency.go` — constants, struct, `Quality`, `tierBoost`, `Input`
2. `/Users/joe/repos/personal/engram/internal/frecency/frecency_test.go` — update all affected tests
3. `/Users/joe/repos/personal/engram/internal/surface/surface.go` — `minRelevanceScore`, `coldStartBudget`, `toFrecencyInput`
4. `/Users/joe/repos/personal/engram/internal/surface/genfactor.go` — updated penalty table
5. `/Users/joe/repos/personal/engram/internal/surface/genfactor_test.go` — updated expected values
6. `/Users/joe/repos/personal/engram/internal/surface/cold_start_budget_test.go` — budget=2
7. `/Users/joe/repos/personal/engram/internal/memory/memory.go` — add `Confidence string` to `Stored`
8. Verify the TOML reader that populates `memory.Stored` already maps `confidence` → `Confidence`
   (if not, add it)

## Verification

After green phase: `targ check-full` must pass with zero errors.
Run `targ test` and confirm all tests pass.
