# Irrelevant Count Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Track `irrelevant_count` on memory TOML files, use it to gate surfacing (suppress memories with high irrelevance ratio), and display it in `engram show` output.

**Architecture:** Add `irrelevant_count` field to memory TOML. Update `engram feedback --irrelevant` to increment it. Add a relevance ratio gate to surfacing (`irrelevant / total_feedback > 0.7` with ≥5 feedback). Update `EffectivenessStat` to carry the irrelevant count. Update `engram show` to display relevance ratio.

**Tech Stack:** Go, gomega, targ

---

## File Structure

| File | Change | Responsibility |
|------|--------|---------------|
| `internal/cli/feedback.go` | Modify | Increment `irrelevant_count` on `--irrelevant` |
| `internal/cli/feedback_test.go` | Modify | Test irrelevant_count increments |
| `internal/cli/show.go` | Modify | Display relevance ratio in output |
| `internal/cli/show_test.go` | Modify | Test relevance display |
| `internal/memory/memory.go` | Modify | Add `IrrelevantCount` to `Stored` struct |
| `internal/effectiveness/aggregate.go` | Modify | Include irrelevant count in `Stat` |
| `internal/surface/surface.go` | Modify | Add relevance ratio gate |
| `internal/surface/surface_test.go` | Modify | Test relevance gating |

---

### Task 1: Add IrrelevantCount to memory.Stored and feedback

**Files:**
- Modify: `internal/memory/memory.go` (add `IrrelevantCount int` to Stored)
- Modify: `internal/cli/feedback.go` (increment `irrelevant_count` on `--irrelevant`)
- Modify: `internal/cli/feedback_test.go` (verify increment)

- [ ] **Step 1: Write failing test — irrelevant increments irrelevant_count**

In `internal/cli/feedback_test.go`, find `TestFeedback_Irrelevant_NoCounterChange` and rename it. Change the assertion: instead of "no counters changed", verify `irrelevant_count` incremented from 0 to 1. The existing test asserts no change — this is the behavior we're changing.

```go
func TestFeedback_Irrelevant_IncrementsIrrelevantCount(t *testing.T) {
    // ... same setup as existing test ...
    // Assert: irrelevant_count = 1 in the updated file
    g.Expect(string(updated)).To(ContainSubstring("irrelevant_count = 1"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `--irrelevant` currently doesn't update any counter.

- [ ] **Step 3: Implement**

In `internal/memory/memory.go`, add to `Stored` struct:
```go
IrrelevantCount   int
```

In `internal/cli/feedback.go`, update `applyFeedbackCounters`:
```go
func applyFeedbackCounters(record map[string]any, relevant, used, notused bool) string {
    if !relevant {
        current, _ := record["irrelevant_count"].(int64)
        record["irrelevant_count"] = current + 1
        return "irrelevant"
    }
    // ... rest unchanged
}
```

In `internal/cli/show.go`, add `IrrelevantCount` to `showTOMLRecord`:
```go
IrrelevantCount   int      `toml:"irrelevant_count"`
```

And map it in `loadMemoryTOML`:
```go
IrrelevantCount:   record.IrrelevantCount,
```

- [ ] **Step 4: Run tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory/memory.go internal/cli/feedback.go internal/cli/feedback_test.go internal/cli/show.go
git commit -m "feat(feedback): track irrelevant_count on memory TOML (#343)"
```

---

### Task 2: Add relevance ratio to EffectivenessStat and surfacing gate

**Files:**
- Modify: `internal/effectiveness/aggregate.go` (add IrrelevantCount to Stat)
- Modify: `internal/surface/surface.go` (add relevance gate)
- Modify: `internal/surface/surface_test.go` (test gating)

- [ ] **Step 1: Write failing test — high-irrelevance memory is gated out**

In `internal/surface/surface_test.go`, add a test:

```go
// Memory with >=5 total feedback and >70% irrelevance is gated out of surfacing.
func TestHighIrrelevanceMemory_GatedOut(t *testing.T) {
    // Create two memories matching the query.
    // Memory A: 8 irrelevant, 2 followed → 80% irrelevance → gated
    // Memory B: 1 irrelevant, 5 followed → 17% irrelevance → surfaced
    // Assert only Memory B appears in output.
}
```

Note: The effectiveness data structure (`EffectivenessStat`) needs to carry `IrrelevantCount` for the gate to work. The gate checks: `irrelevant / (irrelevant + followed + contradicted + ignored) > 0.7`.

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL — no relevance gating exists yet.

- [ ] **Step 3: Implement**

In `internal/effectiveness/aggregate.go`, add to `Stat`:
```go
type Stat struct {
    FollowedCount      int
    ContradictedCount  int
    IgnoredCount       int
    IrrelevantCount    int     // NEW
    EffectivenessScore float64
}
```

Update `FromMemories()` to populate it from memory TOML:
```go
stat.IrrelevantCount = mem.IrrelevantCount
```

In `internal/surface/surface.go`, add to `EffectivenessStat`:
```go
type EffectivenessStat struct {
    SurfacedCount      int
    EffectivenessScore float64
    IrrelevantCount    int     // NEW
}
```

Add a gating function:
```go
const (
    minRelevanceFeedback = 5
    maxIrrelevanceRatio  = 0.7
)

func isIrrelevanceGated(stat EffectivenessStat) bool {
    totalFeedback := stat.IrrelevantCount +
        int(stat.EffectivenessScore) // ... need total evaluations
    // Actually: need followed+contradicted+ignored+irrelevant
    // This requires carrying all counts, not just the score.
}
```

**Important design decision:** The current `EffectivenessStat` only carries `SurfacedCount` and `EffectivenessScore` (a percentage). To compute the relevance ratio, we need the raw `IrrelevantCount` AND the raw evaluation counts (`followed + contradicted + ignored`). Either:
- (A) Add `IrrelevantCount` and `TotalEvaluations` to `EffectivenessStat`
- (B) Add `IrrelevantCount` and compute total from `SurfacedCount` (but surfaced ≠ evaluated)

Option (A) is cleaner. Add `TotalEvaluations int` and `IrrelevantCount int` to `EffectivenessStat`.

Then the gate function:
```go
func isIrrelevanceGated(stat EffectivenessStat) bool {
    totalFeedback := stat.TotalEvaluations + stat.IrrelevantCount
    if totalFeedback < minRelevanceFeedback {
        return false
    }
    ratio := float64(stat.IrrelevantCount) / float64(totalFeedback)
    return ratio > maxIrrelevanceRatio
}
```

Wire it into `filterToolMatchesByEffectivenessGate` and the session-start gating loop (around line 1185). Add the check alongside the existing effectiveness gate:
```go
if isIrrelevanceGated(stat) {
    continue // gated out — too often irrelevant
}
```

- [ ] **Step 4: Run tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/effectiveness/aggregate.go internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): gate out memories with high irrelevance ratio (#343)"
```

---

### Task 3: Display relevance ratio in engram show

**Files:**
- Modify: `internal/cli/show.go` (add relevance display)
- Modify: `internal/cli/show_test.go` (test display)

- [ ] **Step 1: Write failing test**

```go
func TestShow_DisplaysRelevanceRatio(t *testing.T) {
    // Memory with followed_count=5, irrelevant_count=3
    // Assert output contains "Relevance: 62%" (5 out of 8 total feedback was relevant)
}
```

- [ ] **Step 2: Implement**

In `renderMemory`, after the effectiveness line, add:
```go
if mem.IrrelevantCount > 0 {
    totalFeedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
    relevantCount := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
    relevancePct := relevantCount * 100 / totalFeedback
    _, _ = fmt.Fprintf(writer,
        "Relevance: %d%% (%d relevant, %d irrelevant out of %d feedback)\n",
        relevancePct, relevantCount, mem.IrrelevantCount, totalFeedback)
}
```

- [ ] **Step 3: Run tests + check-full**

Run: `targ check-full`
Expected: all checks pass

- [ ] **Step 4: Commit**

```bash
git add internal/cli/show.go internal/cli/show_test.go
git commit -m "feat(show): display relevance ratio when irrelevant feedback exists (#343)"
```
