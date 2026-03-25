# Scoring Rebalance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace dead frecency scoring with a two-stage BM25-seeded spreading + quality-weighted ranking system for prompt/tool surfacing modes.

**Architecture:** Stage 1 generates candidates via BM25 + graph neighbor discovery. Stage 2 ranks candidates by `(relevance + spreading) × (1 + quality)` where quality = weighted sum of effectiveness, recency, frequency. All changes are in the scoring/ranking layer — BM25 matching, suppression, and output formatting are unchanged.

**Tech Stack:** Go, gomega test assertions, `targ check-full` for validation

**Spec:** `docs/superpowers/specs/2026-03-25-scoring-rebalance-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/memory/memory.go` | Modify | Add `LastSurfacedAt time.Time` to `Stored` |
| `internal/retrieve/retrieve.go` | Modify | Parse `LastSurfacedAt` from `MemoryRecord` into `Stored` |
| `internal/frecency/frecency.go` | Rewrite | New scoring formula: quality score + combined score |
| `internal/frecency/frecency_test.go` | Rewrite | Tests for new formula |
| `internal/surface/surface.go` | Modify | Wire scoring input, add BM25 score to match structs, add spreading, remove insufficient-data gate |
| `internal/surface/surface_test.go` | Modify | Update affected scoring tests |
| `internal/track/track.go` | Modify | Remove `SurfacingContexts` |
| `internal/track/track_test.go` | Modify | Remove `SurfacingContexts` tests |

---

### Task 1: Add `LastSurfacedAt` to `memory.Stored` and retriever

**Files:**
- Modify: `internal/memory/memory.go:85-100`
- Modify: `internal/retrieve/retrieve.go:66-99`
- Test: `internal/retrieve/retrieve_test.go`

- [ ] **Step 1: Write failing test — Stored includes LastSurfacedAt**

In `internal/retrieve/retrieve_test.go`, add a test that creates a TOML with `last_surfaced_at = "2026-03-01T12:00:00Z"` and verifies the parsed `Stored` has the correct `LastSurfacedAt` time value.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `Stored` has no `LastSurfacedAt` field

- [ ] **Step 3: Add field to Stored and parse in retriever**

In `internal/memory/memory.go`, add to `Stored` struct (after line 98):
```go
LastSurfacedAt time.Time
```

In `internal/retrieve/retrieve.go`, inside `parseMemoryFile` (after line 85 where `updatedAt` is parsed), add:
```go
var lastSurfacedAt time.Time
if record.LastSurfacedAt != "" {
    lastSurfacedAt, _ = time.Parse(time.RFC3339, record.LastSurfacedAt)
    // Silently default to zero on parse failure — memory may never have been surfaced
}
```

Add to the `Stored` literal (after `UpdatedAt: updatedAt`):
```go
LastSurfacedAt: lastSurfacedAt,
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(memory): add LastSurfacedAt to Stored struct (#374)
```

---

### Task 2: Rewrite frecency scoring formula

**Files:**
- Rewrite: `internal/frecency/frecency.go`
- Rewrite: `internal/frecency/frecency_test.go`

- [ ] **Step 1: Write failing tests for new formula**

Replace all tests in `internal/frecency/frecency_test.go` with tests for the new formula. Key tests:

1. **Quality score computation**: Given eff=0.8, recency=0.7, freq=0.5, verify `quality = 1.5*0.8 + 0.5*0.7 + 1.0*0.5 = 2.05`
2. **Combined score**: Given relevance=2.0, spreading=0.5, quality=1.0, verify `score = (2.0 + 1.0*0.5) * (1 + 1.0) = 5.0`
3. **Recency half-life**: Memory surfaced 7 days ago → recency = `1/(1+1)` = 0.5. Memory surfaced 14 days ago → recency = `1/(1+2)` = 0.333.
4. **Frequency normalization**: surfaced=100, maxSurfaced=1000 → `freq = ln(101)/ln(1001) ≈ 0.668`
5. **Default effectiveness**: No eval data → eff=0.5
6. **Zero BM25 with zero spreading**: score = 0 (relevance gatekeeper holds)
7. **Spreading-only candidate**: relevance=0, spreading=0.8 → score = `(0 + 1.0*0.8) * (1 + quality)`

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — old formula doesn't match new expectations

- [ ] **Step 3: Implement new formula**

Rewrite `internal/frecency/frecency.go`:

```go
package frecency

import (
    "math"
    "time"
)

// Input holds the data needed to score one memory.
type Input struct {
    SurfacedCount  int
    LastSurfacedAt time.Time
    UpdatedAt      time.Time // fallback for never-surfaced
    FollowedCount  int
    ContradictedCount int
    IgnoredCount   int
    FilePath       string
}

// Scorer computes quality-weighted scores for memories.
type Scorer struct {
    now            time.Time
    maxSurfaced    int     // corpus-wide max for freq normalization
    halfLifeDays   float64
    wEff           float64
    wRec           float64
    wFreq          float64
    alpha          float64 // spreading weight
}

// New creates a Scorer. maxSurfaced is the corpus-wide max surfaced count.
func New(now time.Time, maxSurfaced int, opts ...Option) *Scorer {
    s := &Scorer{
        now:          now,
        maxSurfaced:  maxSurfaced,
        halfLifeDays: defaultHalfLifeDays,
        wEff:         defaultWEff,
        wRec:         defaultWRec,
        wFreq:        defaultWFreq,
        alpha:        defaultAlpha,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Option configures a Scorer.
type Option func(*Scorer)

// Quality computes the quality multiplier for a memory.
func (s *Scorer) Quality(input Input) float64 {
    return s.wEff*s.effectiveness(input) +
        s.wRec*s.recency(input) +
        s.wFreq*s.frequency(input)
}

// CombinedScore computes (relevance + alpha*spreading) × (1 + quality).
func (s *Scorer) CombinedScore(relevance, spreading float64, input Input) float64 {
    return (relevance + s.alpha*spreading) * (1.0 + s.Quality(input))
}

func (s *Scorer) effectiveness(input Input) float64 {
    total := input.FollowedCount + input.ContradictedCount + input.IgnoredCount
    if total == 0 {
        return defaultEffectiveness
    }
    return float64(input.FollowedCount) / float64(total)
}

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

func (s *Scorer) frequency(input Input) float64 {
    if s.maxSurfaced <= 0 {
        return 0
    }
    return math.Log(1+float64(input.SurfacedCount)) /
        math.Log(1+float64(s.maxSurfaced))
}

const (
    defaultEffectiveness = 0.5
    defaultHalfLifeDays  = 7.0
    defaultWEff          = 1.5
    defaultWRec          = 0.5
    defaultWFreq         = 1.0
    defaultAlpha         = 1.0
    hoursPerDay          = 24.0
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS (no lint, coverage, or nil issues)

- [ ] **Step 6: Commit**

```
feat(frecency): two-stage quality-weighted scoring formula (#374)
```

---

### Task 3: Remove `SurfacingContexts` from track package

**Files:**
- Modify: `internal/track/track.go`
- Modify: `internal/track/track_test.go`

- [ ] **Step 1: Remove `SurfacingContexts` from `SurfacingUpdate` and `ComputeUpdate`**

In `internal/track/track.go`:
- Remove `SurfacingContexts []string` from `SurfacingUpdate` struct (line 17)
- Remove `MaxContextEntries` constant (line 10)
- Remove `currentContexts []string` parameter from `ComputeUpdate` (line 24)
- Remove all context-building logic (lines 30-36)
- Remove `SurfacingContexts` from the return struct (line 40)

Simplified function:
```go
func ComputeUpdate(currentCount int, mode string, now time.Time) SurfacingUpdate {
    return SurfacingUpdate{
        SurfacedCount: currentCount + 1,
        LastSurfaced:  now,
    }
}
```

- [ ] **Step 2: Update tests**

Remove context-related test cases from `internal/track/track_test.go`. Update remaining tests to match new `ComputeUpdate` signature (no `currentContexts` parameter).

- [ ] **Step 3: Fix any callers**

Search for `ComputeUpdate` callers and update signatures. Check `internal/track/recorder.go` and any other files.

- [ ] **Step 4: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 5: Commit**

```
refactor(track): remove SurfacingContexts (#374)

Only 1-3 possible values (session-start/prompt/tool) — not enough
cardinality to be useful as a scoring signal.
```

---

### Task 4: Add BM25 score to match structs and wire scoring input

**Files:**
- Modify: `internal/surface/surface.go`

- [ ] **Step 1: Add `bm25Score` field to match structs**

At `internal/surface/surface.go:1011-1018`, change:
```go
type promptMatch struct {
    mem       *memory.Stored
    bm25Score float64
}

type toolMatch struct {
    mem       *memory.Stored
    bm25Score float64
}
```

- [ ] **Step 2: Populate bm25Score in matchPromptMemories and matchToolMemories**

In `matchPromptMemories` (~line 1336), where matches are appended, set `bm25Score: penalizedScore`.

In `matchToolMemories` (similar), set `bm25Score: penalizedScore`.

- [ ] **Step 3: Replace `toFrecencyInput` with `toScoringInput`**

Replace the function at lines 1467-1476:
```go
func toScoringInput(mem *memory.Stored) frecency.Input {
    return frecency.Input{
        SurfacedCount:     mem.SurfacedCount,
        LastSurfacedAt:    mem.LastSurfacedAt,
        UpdatedAt:         mem.UpdatedAt,
        FollowedCount:     mem.FollowedCount,
        ContradictedCount: mem.ContradictedCount,
        IgnoredCount:      mem.IgnoredCount,
        FilePath:          mem.FilePath,
    }
}
```

- [ ] **Step 4: Update sort functions to use new scoring**

Replace `sortPromptMatchesByActivation` and `sortToolMatchesByActivation` (lines 1451-1465) to sort by `CombinedScore(match.bm25Score, 0, toScoringInput(match.mem))`. Spreading is 0 for now — wired in Task 5.

- [ ] **Step 5: Update Scorer creation in `Surface.Run`**

At line 200, change scorer creation to pass `maxSurfaced`:
```go
maxSurfaced := computeMaxSurfaced(memories)
scorer := frecency.New(time.Now(), maxSurfaced)
```

Add helper:
```go
func computeMaxSurfaced(memories []*memory.Stored) int {
    max := 0
    for _, m := range memories {
        if m.SurfacedCount > max {
            max = m.SurfacedCount
        }
    }
    return max
}
```

Note: `maxSurfaced` must be computed from the full memory list before any filtering. The scorer is created in `Surface.Run` before mode dispatch, but memories are loaded inside each mode's handler. This needs restructuring — either compute maxSurfaced inside each mode handler after loading memories, or pass it as a scorer option.

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(surface): wire scoring input with real tracking data (#374)

SurfacedCount and LastSurfacedAt now flow from memory.Stored into
the frecency scorer. Fixes #376.
```

---

### Task 5: Add BM25-seeded spreading activation to prompt/tool modes

**Files:**
- Modify: `internal/surface/surface.go`
- Test: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test — spreading boosts graph neighbor**

Create a test where:
- Memory A matches BM25 for query "targ build"
- Memory B doesn't match BM25 but is linked to A with weight 0.8
- After scoring with spreading, B appears in results with a spreading score

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — no spreading mechanism exists in prompt/tool path

- [ ] **Step 3: Implement BM25-seeded spreading**

Add a function to compute spreading scores from BM25 matches:
```go
func computeSpreading(
    bm25Matches map[string]float64, // filePath -> bm25 score
    linkReader LinkReader,
) map[string]float64 {
    spreading := make(map[string]float64)
    if linkReader == nil {
        return spreading
    }
    linkCounts := make(map[string]int)

    for matchPath, matchBM25 := range bm25Matches {
        links, err := linkReader.GetEntryLinks(matchPath)
        if err != nil {
            continue
        }
        for _, link := range links {
            spreading[link.Target] += matchBM25 * link.Weight
            linkCounts[link.Target]++
        }
    }

    // Normalize by link count
    for target, count := range linkCounts {
        if count > 0 {
            spreading[target] /= float64(count)
        }
    }

    return spreading
}
```

Wire into `runPrompt` and `runTool`: after BM25 matching, compute spreading, add neighbor candidates to match list, then sort by `CombinedScore(bm25Score, spreadingScore, input)`.

- [ ] **Step 4: Wire LinkReader into prompt/tool modes**

In `cli.go` surfacer construction (~line 1514-1537), add `WithLinkReader` if a link reader is available. Check how session-start wires it (it doesn't currently, per #375, but the `WithLinkReader` option exists).

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(surface): BM25-seeded spreading activation for prompt/tool (#374)

Graph neighbors of BM25-matched memories now get a spreading boost.
Spreading is normalized by link count to prevent link-dense memories
from dominating.
```

---

### Task 6: Remove insufficient-data gate from surfacing

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Write test — memory with <5 evaluations still surfaces**

Create a test where a memory has 2 evaluations (below current threshold of 5) and verify it surfaces in prompt mode with the default eff=0.5.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — insufficient-data gate blocks surfacing

- [ ] **Step 3: Remove the gate**

In `internal/surface/surface.go`:
- In `effectivenessScoreFor` (~line 1201): remove the branch that checks `insufficientDataThreshold` and returns a default. All memories should use their actual effectiveness score, falling back to 50.0 when no evaluations exist.
- In `filterToolMatchesByEffectivenessGate` (~line 1256): remove the `SurfacedCount >= insufficientDataThreshold` condition. Gate only on effectiveness score being below the floor (40%), regardless of evaluation count.
- Keep `insufficientDataThreshold` constant if it's used by the maintenance system (review.go) — check first.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(surface): remove insufficient-data gate from surfacing (#374)

Memories with <5 evaluations now surface normally with eff=0.5
default. The maintenance system keeps its own gate for quadrant
classification.
```

---

### Task 7: Integration validation

- [ ] **Step 1: Run full check suite**

Run: `targ check-full`
Expected: All checks pass clean

- [ ] **Step 2: Manual smoke test**

Run engram surface in prompt mode against real data:
```bash
~/.claude/engram/bin/engram surface --mode prompt --message "targ build test" --data-dir ~/.claude/engram/data
```

Verify output contains ranked memories with quality-influenced ordering (not just BM25).

- [ ] **Step 3: Compare against playground**

Run the same query in `scoring-playground.html` with Approach B settings (alpha=1, wEff=1.5, wRec=0.5, wFreq=1, half-life=7). Verify the top results are similar (exact scores will differ slightly due to JS vs Go BM25 implementation differences, but ranking should be comparable).
