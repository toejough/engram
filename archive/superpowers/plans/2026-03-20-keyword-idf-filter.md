# Keyword IDF Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After the LLM generates keywords during `learn`, filter out any that appear in >30% of existing memories — they're too common to be discriminating.

**Architecture:** Add a keyword filtering step in the learn pipeline, between LLM classification and memory creation. Load all existing memories, compute document frequency per keyword, strip any with DF > 30%. Pure Go — no LLM call needed for the filter itself.

**Tech Stack:** Go, gomega, targ

---

## File Structure

| File | Change | Responsibility |
|------|--------|---------------|
| `internal/keyword/filter.go` | Create | `FilterByDocFrequency` — takes candidate keywords + existing memories, returns filtered keywords |
| `internal/keyword/filter_test.go` | Create | Tests for the filter function |
| `internal/learn/learn.go` | Modify | Wire filter after classification, before memory creation |
| `internal/learn/learn_test.go` | Modify | Test that filtered keywords are used |

---

### Task 1: Keyword document frequency filter

**Files:**
- Create: `internal/keyword/filter.go`
- Create: `internal/keyword/filter_test.go`

- [ ] **Step 1: Write failing test**

```go
package keyword_test

func TestFilterByDocFrequency_RemovesCommonKeywords(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    // 10 existing memories. "test" appears in 4 (40% > 30% threshold).
    // "gomega" appears in 1 (10% < 30% threshold).
    existing := [][]string{
        {"test", "alpha"},
        {"test", "beta"},
        {"test", "gamma"},
        {"test", "delta"},
        {"gomega", "epsilon"},
        {"zeta"}, {"eta"}, {"theta"}, {"iota"}, {"kappa"},
    }

    candidates := []string{"test", "gomega", "targ-check-full"}

    result := keyword.FilterByDocFrequency(candidates, existing, 0.3)

    g.Expect(result).To(ConsistOf("gomega", "targ-check-full"))
    g.Expect(result).NotTo(ContainElement("test"))
}
```

- [ ] **Step 2: Run test — verify it fails**

Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Implement**

Create `internal/keyword/filter.go`:

```go
package keyword

// FilterByDocFrequency removes keywords that appear in more than
// maxRatio of existing memory keyword sets. Keywords not present
// in any existing memory always pass.
func FilterByDocFrequency(
    candidates []string,
    existingKeywordSets [][]string,
    maxRatio float64,
) []string {
    if len(existingKeywordSets) == 0 {
        return candidates
    }

    // Build document frequency: how many memories contain each keyword.
    docFreq := make(map[string]int, len(candidates))
    for _, kws := range existingKeywordSets {
        seen := make(map[string]bool, len(kws))
        for _, kw := range kws {
            if !seen[kw] {
                docFreq[kw]++
                seen[kw] = true
            }
        }
    }

    corpusSize := float64(len(existingKeywordSets))
    filtered := make([]string, 0, len(candidates))

    for _, kw := range candidates {
        ratio := float64(docFreq[kw]) / corpusSize
        if ratio <= maxRatio {
            filtered = append(filtered, kw)
        }
    }

    return filtered
}
```

- [ ] **Step 4: Add edge case tests**

```go
func TestFilterByDocFrequency_EmptyExisting_KeepsAll(t *testing.T) {
    // No existing memories → all candidates pass
}

func TestFilterByDocFrequency_EmptyCandidates_ReturnsEmpty(t *testing.T) {
    // No candidates → empty result
}

func TestFilterByDocFrequency_AllCommon_ReturnsEmpty(t *testing.T) {
    // All candidates appear in >30% → empty result (valid — memory gets no keywords)
}

func TestFilterByDocFrequency_NewKeyword_AlwaysPasses(t *testing.T) {
    // Keyword not in any existing memory → always passes (0% < 30%)
}
```

- [ ] **Step 5: Run tests + check-full**

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(keyword): add document frequency filter for keyword discrimination (#345)"
```

---

### Task 2: Wire filter into learn pipeline

**Files:**
- Modify: `internal/learn/learn.go`
- Modify: `internal/learn/learn_test.go`

- [ ] **Step 1: Understand the learn pipeline**

Read `internal/learn/learn.go` to find where keywords from the LLM classification are set on the new memory. The filter should be applied AFTER classification returns keywords and BEFORE the memory is written.

Search for where `Keywords` is set: `grep -n "Keywords\|keywords" internal/learn/learn.go`

The filter needs access to all existing memories' keyword lists. The learn pipeline already loads existing memories for dedup — find where that happens and extract the keyword sets.

- [ ] **Step 2: Write failing test**

Add a test in `internal/learn/learn_test.go` that:
- Seeds existing memories with a common keyword (e.g., "test" in 5 of 10 memories)
- Triggers learn with a message that produces a memory with keywords including "test"
- Asserts that "test" was filtered out of the new memory's keywords

**Note:** This may require DI injection for the filter. Check if learn has an injectable interface pattern. If the filter is a pure function, it can be called directly without DI.

- [ ] **Step 3: Implement wiring**

In the learn pipeline, after the LLM classifier returns keywords, call:

```go
existingKeywordSets := make([][]string, 0, len(existingMemories))
for _, mem := range existingMemories {
    existingKeywordSets = append(existingKeywordSets, mem.Keywords)
}

classified.Keywords = keyword.FilterByDocFrequency(
    classified.Keywords, existingKeywordSets, keywordMaxDocFreqRatio,
)
```

Add constant:
```go
const keywordMaxDocFreqRatio = 0.3
```

**Important:** If filtering removes ALL keywords, keep at least the original keywords — a memory with zero keywords will never surface. Add a guard:

```go
filtered := keyword.FilterByDocFrequency(...)
if len(filtered) > 0 {
    classified.Keywords = filtered
}
```

- [ ] **Step 4: Run tests + check-full**

- [ ] **Step 5: Commit + rebuild binary**

```bash
git commit -m "feat(learn): wire keyword IDF filter into learn pipeline (#345)"
go build -o ~/.claude/engram/bin/engram ./cmd/engram/
```
