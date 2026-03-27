# Implementation Plan: Remove Recency Signal from Surfacing Quality Score (#394)

Remove recency computation from `internal/frecency/frecency.go`. After #391, `defaultWRec` is already zeroed,
making the `recency()` method and related constants dead code.

## Context

Issue #391 sets `defaultWRec = 0`, which disables the recency signal entirely. This plan removes the disabled
constants, fields, and the `recency()` method.

## Changes

### 1. `internal/frecency/frecency.go` — remove fields, constants, and method

**Delete constants (lines 107–115):**
```go
const (
	defaultAlpha         = 1.0
	defaultEffectiveness = 0.5
	defaultHalfLifeDays  = 7.0      // DELETE THIS
	defaultWEff          = 1.5
	defaultWFreq         = 1.0
	defaultWRec          = 0.5      // (already 0 after #391, but delete this line too)
	hoursPerDay          = 24.0
)
```

Delete:
- `defaultHalfLifeDays`
- The entire `defaultWRec` line (or leave as part of the `defaultWRec = 0` state already set by #391; for clarity, delete the constant and the field initialization)

**Delete field from `Scorer` struct (line 27):**
```go
type Scorer struct {
	now          time.Time
	maxSurfaced  int
	halfLifeDays float64   // DELETE THIS
	wEff         float64
	wRec         float64   // (kept for #391, but zeroed; OK to keep or delete, recommend delete)
	wFreq        float64
	alpha        float64
}
```

Delete `halfLifeDays` field. After #391, `wRec = 0`, so technically it can stay (the `recency()` method will never be called);
however, since we're removing the method, delete `wRec` too for clarity (or update comments explaining it's unused).
**Recommendation:** Delete both `halfLifeDays` and `wRec` to make the dead code removal complete.

**Update `New()` constructor (lines 35–51):**

Remove initialization of `halfLifeDays` and `wRec`:
```go
func New(now time.Time, maxSurfaced int, opts ...Option) *Scorer {
	s := &Scorer{
		now:         now,
		maxSurfaced: maxSurfaced,
		// halfLifeDays: defaultHalfLifeDays,  // DELETE THIS
		wEff:  defaultWEff,
		// wRec:  defaultWRec,                  // DELETE THIS
		wFreq: defaultWFreq,
		alpha: defaultAlpha,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
```

**Update `Quality()` method (lines 64–68):**

Change from:
```go
func (s *Scorer) Quality(input Input) float64 {
	return s.wEff*s.effectiveness(input) +
		s.wRec*s.recency(input) +
		s.wFreq*s.frequency(input)
}
```

To:
```go
func (s *Scorer) Quality(input Input) float64 {
	return s.wEff*s.effectiveness(input) +
		s.wFreq*s.frequency(input)
}
```

(If `wTier` field was added in #391, also include `+ s.wTier*s.tierBoost(input)`.)

**Delete `recency()` method (lines 88–104):**

Remove entirely:
```go
func (s *Scorer) recency(input Input) float64 {
	ref := input.LastSurfacedAt
	if ref.IsZero() {
		ref = input.UpdatedAt
	}

	if ref.IsZero() {
		return 0
	}

	daysSince := s.now.Sub(ref).Hours() / hoursPerDay
	if daysSince < 0 {
		daysSince = 0
	}

	return 1.0 / (1.0 + daysSince/s.halfLifeDays)
}
```

### 2. `internal/surface/surface.go` — no changes

The `toFrecencyInput` function still populates `LastSurfacedAt` and `UpdatedAt` fields on the `frecency.Input`.
Leave these fields and their population unchanged — they may be useful for diagnostics or future features.
The `memory.Stored.LastSurfacedAt` field is not deleted (per the issue: "keep for diagnostics").

### 3. `internal/frecency/frecency_test.go` — update tests

**Delete or rewrite tests that verify recency behavior:**

- `TestRecency_FallbackToUpdatedAt` (lines 222–248) — verifies that fallback from `LastSurfacedAt` to `UpdatedAt` works.
  **Delete this test.** Recency is no longer computed.

- `TestRecency_HalfLife` (lines 250–291) — verifies recency score decays by half-life.
  **Delete this test.** Recency is no longer computed.

**Update tests that include recency in their expected formula:**

- `TestQuality_AllSignals` (lines 164–195) — currently expects formula:
  ```
  expected := 1.5*expectedEff + 0.5*expectedRecency + 1.0*expectedFreq
  ```
  **Update to:**
  ```
  expected := 1.5*expectedEff + 1.0*expectedFreq
  ```
  (Or, if #391 already updated this test, use its new weights: `expected := 0.3*expectedEff + 1.0*expectedFreq + 0.3*tierBoost`.)

- `TestQuality_NoEvaluations` (lines 197–220) — currently expects formula:
  ```
  expected := 1.5*expectedEff + 0.5*expectedRecency + 1.0*expectedFreq
  ```
  **Update to:**
  ```
  expected := 1.5*expectedEff + 1.0*expectedFreq
  ```
  (Or use #391 weights if already applied.)

**Update frequency extraction tests:**

- `TestFrequency_Normalized` (lines 115–162) — extraction formula (line 155):
  ```
  freq = quality - 0.75 // subtract eff component
  ```
  **Update to:**
  ```
  freq = quality - 0.75 // subtract eff component (wEff * defaultEffectiveness = 1.5 * 0.5)
  ```
  (No change if formula remains the same; only update if #391 changed weights.)

## Files to Change

1. `/Users/joe/repos/personal/engram/internal/frecency/frecency.go` — delete `defaultHalfLifeDays`, delete/zero `wRec`, delete `halfLifeDays` field from `Scorer`, update `Quality()`, delete `recency()` method
2. `/Users/joe/repos/personal/engram/internal/frecency/frecency_test.go` — delete `TestRecency_*` tests, update `Quality` formula in other tests

## Verification

After green phase: `targ check-full` must pass with zero errors.
Run `targ test` and confirm all tests pass.

Manually verify: `Quality` formula no longer depends on `LastSurfacedAt` or `UpdatedAt` timing. A memory
surfaced 1 day ago and another surfaced 30 days ago, with identical effectiveness and frequency, should score identically.
