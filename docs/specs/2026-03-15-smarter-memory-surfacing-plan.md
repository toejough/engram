# Smarter Memory Surfacing Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce noise from unproven memories by applying stricter BM25 relevance thresholds and a cold-start budget across all surfacing modes.

**Architecture:** Two new filtering stages inserted into the existing surfacing pipeline: (1) per-memory BM25 floor based on proven/unproven status, (2) a cold-start budget that caps unproven memories to 1 per invocation. Both apply after existing BM25/effectiveness ranking but before suppression passes and token budgets.

**Tech Stack:** Go, gomega test framework, `targ` build system

---

## Task 1: Add `isUnproven` helper and new constants

**Files:**
- Modify: `internal/surface/surface.go` (constants block + new function)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing tests for `isUnproven`**

Three cases: no effectiveness data, 0 surfacings, 1+ surfacings. `isUnproven` is unexported, so test via exported behavior — but since it's a pure helper used by other functions, test it indirectly through the integration tests in later tasks. Instead, add the constants and helper now, tested in Task 2+3.

- [ ] **Step 2: Add constants and `isUnproven` to `surface.go`**

In the unexported constants block, add:
```go
coldStartBudget              = 1
unprovenBM25FloorPrompt      = 0.20
unprovenBM25FloorTool        = 0.30
unprovenDefaultEffectiveness = 30.0
```

Add helper function:
```go
func isUnproven(path string, effectiveness map[string]EffectivenessStat) bool {
	if effectiveness == nil {
		return true
	}
	stat, ok := effectiveness[path]
	return !ok || stat.SurfacedCount == 0
}
```

- [ ] **Step 3: Update `effectivenessScoreFor` to use lower default for unproven**

Change the function so that memories with 0 surfacings (or no data) get `unprovenDefaultEffectiveness` (30%) instead of `sessionStartDefaultEffectiveness` (50%). Memories with 1-4 surfacings (insufficient but not unproven) keep the 50% default.

```go
func effectivenessScoreFor(path string, effectiveness map[string]EffectivenessStat) float64 {
	if effectiveness == nil {
		return unprovenDefaultEffectiveness
	}
	stat, ok := effectiveness[path]
	if !ok || stat.SurfacedCount == 0 {
		return unprovenDefaultEffectiveness
	}
	if stat.SurfacedCount < insufficientDataThreshold {
		return sessionStartDefaultEffectiveness
	}
	return stat.EffectivenessScore
}
```

- [ ] **Step 4: Run `targ check-full` to verify compilation**

Run: `targ check-full`
Expected: passes (no behavioral tests broken yet — existing tests use SurfacedCount >= 2 for "insufficient data" cases)

- [ ] **Step 5: Commit**

```
feat(surface): add unproven memory constants and helpers (#307)
```

---

## Task 2: Higher BM25 floor for unproven memories in prompt and tool modes

**Files:**
- Modify: `internal/surface/surface.go` (`matchPromptMemories`, `matchToolMemories`)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test — unproven prompt memory below higher floor is filtered**

Add test `TestUnprovenPromptMemoryFilteredByHigherBM25Floor`. Create memories where an unproven memory has a BM25 score between 0.05 and 0.20 (passes old floor, fails new floor). Needs enough filler docs for IDF contrast. Pass effectiveness data showing the target memory has 0 surfacings.

Key: `matchPromptMemories` needs to accept effectiveness data now — the signature change will cause compilation failure = red.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error — `matchPromptMemories` doesn't accept effectiveness parameter yet.

- [ ] **Step 3: Update `matchPromptMemories` signature and implementation**

Add `effectiveness map[string]EffectivenessStat` parameter. In the BM25 filter loop, use `unprovenBM25FloorPrompt` when `isUnproven(sd.ID, effectiveness)`:

```go
func matchPromptMemories(
	message string,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []promptMatch {
	// ... existing BM25 scoring ...
	for _, sd := range scored {
		floor := minRelevanceScore
		if isUnproven(sd.ID, effectiveness) {
			floor = unprovenBM25FloorPrompt
		}
		if sd.Score < floor {
			continue
		}
		// ... rest unchanged ...
	}
	return matches
}
```

Update caller in `runPrompt` to pass `effectiveness`.

- [ ] **Step 4: Write failing test — unproven tool memory below higher floor is filtered**

Add test `TestUnprovenToolMemoryFilteredByHigherBM25Floor`. Same pattern but for tool mode with `unprovenBM25FloorTool = 0.30`.

- [ ] **Step 5: Update `matchToolMemories` signature and implementation**

Add `effectiveness map[string]EffectivenessStat` parameter. Same pattern as prompt mode but with `unprovenBM25FloorTool`. Update caller in `runTool`.

- [ ] **Step 6: Write test — proven memory at same BM25 score passes**

Add test `TestProvenMemoryPassesAtLowerBM25Score` confirming that a memory with SurfacedCount >= 1 and BM25 score of 0.10 still surfaces (passes the 0.05 floor, not the higher unproven floor).

- [ ] **Step 7: Run `targ check-full`**

Run: `targ check-full`
Expected: all pass

- [ ] **Step 8: Commit**

```
feat(surface): apply higher BM25 floor for unproven memories (#307)
```

---

## Task 3: Cold-start budget

**Files:**
- Modify: `internal/surface/surface.go` (`runPrompt`, `runTool`, `runSessionStart`)
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test — cold-start budget limits unproven in tool mode**

Add test `TestColdStartBudgetLimitsUnprovenToolMemories`. Create 3 anti-pattern memories all with 0 surfacings, all highly relevant (high BM25). Verify only 1 surfaces (cold-start budget = 1).

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — currently 2 would surface (toolLimit).

- [ ] **Step 3: Implement `applyColdStartBudgetTool` and wire into `runTool`**

```go
func applyColdStartBudgetTool(
	candidates []toolMatch,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
	result := make([]toolMatch, 0, len(candidates))
	unprovenCount := 0
	for _, c := range candidates {
		if isUnproven(c.mem.FilePath, effectiveness) {
			unprovenCount++
			if unprovenCount > coldStartBudget {
				continue
			}
		}
		result = append(result, c)
	}
	return result
}
```

Call in `runTool` after `sortToolMatchesByActivation`, before the `toolLimit` cap.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write failing test — cold-start budget limits unproven in prompt mode**

Add test `TestColdStartBudgetLimitsUnprovenPromptMemories`. Same pattern for prompt mode.

- [ ] **Step 6: Implement `applyColdStartBudgetPrompt` and wire into `runPrompt`**

Same pattern as tool variant but for `[]promptMatch`. Call after `sortPromptMatchesByActivation`, before `promptLimit` cap.

- [ ] **Step 7: Write failing test — cold-start budget limits unproven in session-start mode**

Add test `TestColdStartBudgetLimitsUnprovenSessionStart`. Create 3 unproven memories, verify only 1 surfaces.

- [ ] **Step 8: Implement `applyColdStartBudgetStored` and wire into `runSessionStart`**

Same pattern for `[]*memory.Stored`. Call after `sortByEffectivenessScore` (and spreading activation if present), before `sessionStartLimit` cap.

- [ ] **Step 9: Write test — proven memories unaffected by cold-start budget**

Add test `TestColdStartBudgetDoesNotLimitProvenMemories`. Create 5 proven memories (SurfacedCount > 0), verify all surface up to mode limit.

- [ ] **Step 10: Run `targ check-full`**

Run: `targ check-full`
Expected: all pass

- [ ] **Step 11: Commit**

```
feat(surface): add cold-start budget for unproven memories (#307)
```

---

## Task 4: Verify existing tests still pass and fix any regressions

- [ ] **Step 1: Run full test suite**

Run: `targ check-full`
Expected: all pass. Some existing tests may need effectiveness data adjusted if they relied on unproven memories surfacing freely.

- [ ] **Step 2: Fix any broken tests**

Likely candidates: tests in `p4e_test.go` that create memories with `SurfacedCount: 2` (insufficient data) — these are still "proven" (SurfacedCount > 0) so should be unaffected. Tests with no effectiveness data at all may now see fewer memories surfacing due to lower default score and cold-start budget.

- [ ] **Step 3: Final `targ check-full`**

Run: `targ check-full`
Expected: clean

- [ ] **Step 4: Commit any test fixes**

```
test(surface): adjust tests for stricter unproven memory filtering (#307)
```
