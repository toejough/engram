# Irrelevant Count Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Track `irrelevant_count` on memory TOML files, apply a continuous BM25 penalty proportional to irrelevance (no binary gate), propose `refine-keywords` in maintain for high-irrelevance memories, and display relevance ratio in `engram show`.

**Architecture:** Three integration points: (1) `engram feedback --irrelevant` increments `irrelevant_count` in TOML, (2) BM25 scoring multiplies base score by `K/(K+irrelevantCount)` where K=5 (half-life), (3) `maintain` proposes `refine-keywords` when irrelevance ratio >60% with ≥5 feedback. Reset `irrelevant_count` to 0 when keywords change.

**Tech Stack:** Go, gomega, targ

---

## Design Summary

**Relevance is NOT a quadrant dimension.** The 2x2 matrix (surfaced × effectiveness) is unchanged. Relevance operates as a continuous BM25 modifier that bends memory trajectories within the existing matrix by suppressing wrong-context surfacing.

**BM25 penalty:** `penalizedScore = baseScore × K / (K + irrelevantCount)` with K=5.

| Irrelevant marks | Factor | Effect |
|-----------------|--------|--------|
| 0 | 1.00 | No penalty |
| 2 | 0.71 | Mild |
| 5 | 0.50 | Halved |
| 10 | 0.33 | Mostly suppressed for generic queries |
| 20 | 0.20 | Only very specific queries surface it |

No death spiral — specific queries can always overcome the penalty.

**Reset:** `irrelevant_count` resets to 0 when keywords are modified (via apply-proposal or manual edit).

---

## File Structure

| File | Change | Responsibility |
|------|--------|---------------|
| `internal/memory/memory.go` | Modify | Add `IrrelevantCount` to `Stored` |
| `internal/cli/feedback.go` | Modify | Increment `irrelevant_count` on `--irrelevant` |
| `internal/cli/feedback_test.go` | Modify | Test irrelevant counter |
| `internal/cli/show.go` | Modify | Display relevance ratio |
| `internal/cli/show_test.go` | Modify | Test relevance display |
| `internal/surface/surface.go` | Modify | Apply BM25 penalty in matchPromptMemories and matchToolMemories |
| `internal/surface/surface_test.go` | Modify | Test penalty reduces score |
| `internal/effectiveness/aggregate.go` | Modify | Include `IrrelevantCount` in `Stat` |
| `internal/maintain/maintain.go` | Modify | Propose `refine-keywords` for high-irrelevance |
| `internal/maintain/maintain_test.go` | Modify | Test refine-keywords proposal |

---

### Task 1: Track irrelevant_count in feedback

**Files:**
- Modify: `internal/memory/memory.go`
- Modify: `internal/cli/feedback.go`
- Modify: `internal/cli/feedback_test.go`
- Modify: `internal/cli/show.go` (add field to showTOMLRecord)

- [ ] **Step 1: Write failing test — irrelevant increments irrelevant_count**

Update the existing `TestFeedback_Irrelevant_NoCounterChange` test (rename it):

```go
func TestFeedback_Irrelevant_IncrementsIrrelevantCount(t *testing.T) {
    // Same setup: memory with irrelevant_count = 0
    // Call feedback --irrelevant
    // Assert file now contains irrelevant_count = 1
}
```

- [ ] **Step 2: Run test — verify it fails**

Expected: FAIL — `--irrelevant` currently doesn't update any counter.

- [ ] **Step 3: Implement**

In `internal/memory/memory.go`, add to `Stored`:
```go
IrrelevantCount   int
```

In `internal/cli/feedback.go`, update `applyFeedbackCounters`:
```go
if !relevant {
    current, _ := record["irrelevant_count"].(int64)
    record["irrelevant_count"] = current + 1
    return "irrelevant"
}
```

In `internal/cli/show.go`, add to `showTOMLRecord`:
```go
IrrelevantCount   int      `toml:"irrelevant_count"`
```

And map it in `loadMemoryTOML`:
```go
IrrelevantCount:   record.IrrelevantCount,
```

- [ ] **Step 4: Run tests — verify pass**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(feedback): increment irrelevant_count on --irrelevant (#343)"
```

---

### Task 2: BM25 relevance penalty

**Files:**
- Modify: `internal/surface/surface.go` (matchPromptMemories, matchToolMemories)
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test — high-irrelevance memory scores lower**

```go
func TestBM25_IrrelevancePenalty_ReducesScore(t *testing.T) {
    // Two memories with identical keywords and content.
    // Memory A: irrelevant_count = 0
    // Memory B: irrelevant_count = 10
    // Both match the same query.
    // Assert: Memory A's BM25 score > Memory B's BM25 score
    // Assert: Memory B's score is approximately baseScore * 5/15 = 0.33× Memory A's score
}
```

**Key decision:** The penalty needs access to `IrrelevantCount` per memory. Currently `matchPromptMemories` receives `[]*memory.Stored` and `effectiveness map[string]EffectivenessStat`. The `IrrelevantCount` is on the `memory.Stored` struct (added in Task 1), so it's already available via `mem.IrrelevantCount` — no need to thread it through effectiveness.

- [ ] **Step 2: Run test — verify it fails**

Expected: FAIL — no penalty applied.

- [ ] **Step 3: Implement**

Add constant:
```go
const irrelevancePenaltyHalfLife = 5
```

Add penalty function:
```go
func irrelevancePenalty(irrelevantCount int) float64 {
    return float64(irrelevancePenaltyHalfLife) /
        float64(irrelevancePenaltyHalfLife + irrelevantCount)
}
```

In `matchPromptMemories`, after BM25 scoring but before building results, multiply each score:
```go
score := scoredDoc.Score * irrelevancePenalty(mem.IrrelevantCount)
```

Same in `matchToolMemories`.

- [ ] **Step 4: Run tests — verify pass**

- [ ] **Step 5: Run full test suite** — existing BM25 tests should still pass since memories with `IrrelevantCount = 0` get factor 1.0 (no change).

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(surface): apply BM25 irrelevance penalty K/(K+count) (#343)"
```

---

### Task 3: Display relevance ratio in engram show

**Files:**
- Modify: `internal/cli/show.go`
- Modify: `internal/cli/show_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestShow_DisplaysRelevanceRatio(t *testing.T) {
    // Memory with followed_count=5, irrelevant_count=3
    // Total feedback = 8, relevant feedback = 5
    // Assert output contains "Relevance: 62%" or similar
}
```

- [ ] **Step 2: Implement**

In `renderMemory`, after effectiveness line:
```go
if mem.IrrelevantCount > 0 {
    totalFeedback := mem.FollowedCount + mem.ContradictedCount +
        mem.IgnoredCount + mem.IrrelevantCount
    relevantFeedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
    pct := relevantFeedback * 100 / totalFeedback
    _, _ = fmt.Fprintf(writer,
        "Relevance: %d%% (%d relevant, %d irrelevant of %d feedback)\n",
        pct, relevantFeedback, mem.IrrelevantCount, totalFeedback)
}
```

- [ ] **Step 3: Run tests + check-full**

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(show): display relevance ratio when irrelevant feedback exists (#343)"
```

---

### Task 4: Maintain proposes refine-keywords for high-irrelevance

**Files:**
- Modify: `internal/maintain/maintain.go`
- Modify: `internal/maintain/maintain_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestMaintain_HighIrrelevance_ProposesRefineKeywords(t *testing.T) {
    // Memory with: followed_count=3, irrelevant_count=8
    // Total feedback = 11, irrelevance ratio = 73% (>60%)
    // Assert: proposal has action = "refine_keywords"
    // Assert: diagnosis mentions "irrelevance" or "relevance"
}
```

**Important:** The maintain function currently reads effectiveness from `effectiveness.FromMemories()`. This needs to include `IrrelevantCount` so maintain can compute the irrelevance ratio.

- [ ] **Step 2: Run test — verify it fails**

- [ ] **Step 3: Implement**

Add constant:
```go
const refineKeywordsIrrelevanceThreshold = 0.6
const refineKeywordsMinFeedback = 5
```

Add to `internal/effectiveness/aggregate.go` `Stat`:
```go
IrrelevantCount    int
```

Populate it in `FromMemories`:
```go
stat.IrrelevantCount = mem.IrrelevantCount
```

In `internal/maintain/maintain.go`, after quadrant classification but before generating proposals, add a check:

```go
// Check for high-irrelevance memories — propose keyword refinement.
totalFeedback := stat.FollowedCount + stat.ContradictedCount +
    stat.IgnoredCount + stat.IrrelevantCount
if totalFeedback >= refineKeywordsMinFeedback {
    ratio := float64(stat.IrrelevantCount) / float64(totalFeedback)
    if ratio > refineKeywordsIrrelevanceThreshold {
        proposals = append(proposals, Proposal{
            MemoryPath: memPath,
            Quadrant:   classified.Quadrant,
            Action:     "refine_keywords",
            Diagnosis:  fmt.Sprintf("%.0f%% of surfacings are irrelevant — keywords may be too generic", ratio*100),
        })
    }
}
```

- [ ] **Step 4: Run tests + check-full**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(maintain): propose refine-keywords for high-irrelevance memories (#343)"
```

---

### Task 5: Final verification and binary rebuild

- [ ] **Step 1: Run targ check-full**

Expected: all checks pass.

- [ ] **Step 2: Rebuild binary**

```bash
go build -o ~/.claude/engram/bin/engram ./cmd/engram/
```

- [ ] **Step 3: Manual smoke test**

```bash
# Verify feedback increments irrelevant_count
engram feedback --name some-memory --irrelevant --data-dir ~/.claude/engram/data
engram show --name some-memory --data-dir ~/.claude/engram/data
# Should show Relevance: line with irrelevant count
```

- [ ] **Step 4: Push**

```bash
git push origin main
```
