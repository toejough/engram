# Generalizability Penalty Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Penalize BM25 relevance for low-generalizability memories when surfacing cross-project.

**Architecture:** Wire `Generalizability` and `ProjectSlug` fields from TOML through retrieve → surface pipeline. Add `genFactor` helper. Modify `CombinedScore` to accept and apply the factor to the BM25 component only.

**Tech Stack:** Go, `targ` build system

**Spec:** `docs/superpowers/specs/2026-03-25-generalizability-penalty-design.md`

---

### Task 1: Add fields to `memory.Stored` and wire in `retrieve.go`

**Files:**
- Modify: `internal/memory/memory.go:85-101`
- Modify: `internal/retrieve/retrieve.go:89-105`
- Modify: `internal/retrieve/retrieve_test.go`

- [ ] **Step 1: Write failing test for `Generalizability` parsing**

In `internal/retrieve/retrieve_test.go`, add a test that creates a TOML file with `generalizability = 3` and `project_slug = "my-project"`, calls `ListMemories`, and asserts both fields are populated on the returned `Stored`.

```go
func TestGeneralizabilityAndProjectSlugParsed(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    tomlContent := `title = "test memory"
content = "test content"
generalizability = 3
project_slug = "my-project"
`
    dir := t.TempDir()
    memDir := filepath.Join(dir, "memories")
    err := os.MkdirAll(memDir, 0o755)
    g.Expect(err).NotTo(HaveOccurred())

    err = os.WriteFile(filepath.Join(memDir, "test.toml"), []byte(tomlContent), 0o644)
    g.Expect(err).NotTo(HaveOccurred())

    r := retrieve.New()
    memories, err := r.ListMemories(dir)
    g.Expect(err).NotTo(HaveOccurred())
    if err != nil { return }
    g.Expect(memories).To(HaveLen(1))
    g.Expect(memories[0].Generalizability).To(Equal(3))
    g.Expect(memories[0].ProjectSlug).To(Equal("my-project"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestGeneralizabilityAndProjectSlugParsed ./internal/retrieve/`
Expected: FAIL — `Stored` has no `Generalizability` or `ProjectSlug` fields.

- [ ] **Step 3: Add fields to `Stored` struct**

In `internal/memory/memory.go`, add to the `Stored` struct (after `FilePath`):

```go
Generalizability int
ProjectSlug      string
```

- [ ] **Step 4: Wire fields in `parseMemoryFile`**

In `internal/retrieve/retrieve.go`, add to the `&memory.Stored{...}` construction (after `FilePath: filePath,`):

```go
Generalizability: record.Generalizability,
ProjectSlug:      record.ProjectSlug,
```

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test -- -run TestGeneralizabilityAndProjectSlugParsed ./internal/retrieve/`
Expected: PASS

- [ ] **Step 6: Run full check**

Run: `targ check-full`
Expected: Pass (pre-existing hooks failure OK).

- [ ] **Step 7: Commit**

```bash
git add internal/memory/memory.go internal/retrieve/retrieve.go internal/retrieve/retrieve_test.go
git commit -m "feat: wire Generalizability and ProjectSlug into memory.Stored (#373)

AI-Used: [claude]"
```

### Task 2: Add `genFactor` helper in surface package

**Files:**
- Create: `internal/surface/genfactor.go`
- Create: `internal/surface/genfactor_test.go`

- [ ] **Step 1: Write failing tests for `genFactor`**

Create `internal/surface/genfactor_test.go`:

```go
package surface_test

import (
    "testing"

    . "github.com/onsi/gomega"

    "engram/internal/surface"
)

func TestGenFactor_SameProject(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Same project = no penalty regardless of generalizability
    g.Expect(surface.GenFactor(1, "proj-a", "proj-a")).To(Equal(1.0))
    g.Expect(surface.GenFactor(5, "proj-a", "proj-a")).To(Equal(1.0))
}

func TestGenFactor_EmptySlug(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // Empty slug = no penalty (can't determine cross-project)
    g.Expect(surface.GenFactor(1, "", "proj-a")).To(Equal(1.0))
    g.Expect(surface.GenFactor(1, "proj-a", "")).To(Equal(1.0))
}

func TestGenFactor_CrossProjectPenalty(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    g.Expect(surface.GenFactor(5, "proj-a", "proj-b")).To(Equal(1.0))
    g.Expect(surface.GenFactor(4, "proj-a", "proj-b")).To(Equal(0.8))
    g.Expect(surface.GenFactor(3, "proj-a", "proj-b")).To(Equal(0.5))
    g.Expect(surface.GenFactor(2, "proj-a", "proj-b")).To(Equal(0.2))
    g.Expect(surface.GenFactor(1, "proj-a", "proj-b")).To(Equal(0.05))
}

func TestGenFactor_UnsetGeneralizability(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    // gen 0 (unset) cross-project = conservative 0.5
    g.Expect(surface.GenFactor(0, "proj-a", "proj-b")).To(Equal(0.5))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestGenFactor ./internal/surface/`
Expected: FAIL — `surface.GenFactor` undefined.

- [ ] **Step 3: Implement `genFactor`**

Create `internal/surface/genfactor.go`:

```go
package surface

// genFactorTable maps generalizability score to BM25 penalty factor for cross-project surfacing.
var genFactorTable = [6]float64{
    0.5,  // 0: unset — conservative default
    0.05, // 1: this-project-only
    0.2,  // 2: narrow
    0.5,  // 3: moderate
    0.8,  // 4: similar projects
    1.0,  // 5: universal
}

// GenFactor returns the BM25 relevance penalty factor for a memory based on its
// generalizability and whether it belongs to the current project.
// Same-project or missing slug = 1.0 (no penalty).
func GenFactor(generalizability int, memProject, currentProject string) float64 {
    if memProject == "" || currentProject == "" || memProject == currentProject {
        return 1.0
    }

    if generalizability < 0 || generalizability > 5 {
        return genFactorTable[0] // treat out-of-range as unset
    }

    return genFactorTable[generalizability]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestGenFactor ./internal/surface/`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 6: Commit**

```bash
git add internal/surface/genfactor.go internal/surface/genfactor_test.go
git commit -m "feat: add GenFactor helper for cross-project penalty (#373)

AI-Used: [claude]"
```

### Task 3: Modify `CombinedScore` to accept `genFactor`

**Files:**
- Modify: `internal/frecency/frecency.go:59-61`
- Modify: `internal/frecency/frecency_test.go`

- [ ] **Step 1: Write failing test for genFactor in CombinedScore**

In `internal/frecency/frecency_test.go`, add:

```go
func TestCombinedScore_GenFactorReducesRelevance(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    scorer := frecency.New()
    input := frecency.Input{} // zero values = no quality contribution

    full := scorer.CombinedScore(1.0, 0.0, 1.0, input)    // genFactor=1.0
    halved := scorer.CombinedScore(1.0, 0.0, 0.5, input)   // genFactor=0.5

    g.Expect(halved).To(BeNumerically("<", full))
    g.Expect(halved).To(BeNumerically("~", full*0.5, 0.01))
}

func TestCombinedScore_GenFactorDoesNotAffectSpreading(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    scorer := frecency.New()
    input := frecency.Input{}

    // With zero relevance, genFactor shouldn't matter — only spreading counts
    a := scorer.CombinedScore(0.0, 1.0, 0.05, input)
    b := scorer.CombinedScore(0.0, 1.0, 1.0, input)

    g.Expect(a).To(Equal(b))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestCombinedScore_GenFactor ./internal/frecency/`
Expected: FAIL — `CombinedScore` takes 3 args, not 4.

- [ ] **Step 3: Update `CombinedScore` signature**

In `internal/frecency/frecency.go`, change:

```go
func (s *Scorer) CombinedScore(relevance, spreading float64, input Input) float64 {
	return (relevance + s.alpha*spreading) * (1.0 + s.Quality(input))
}
```

To:

```go
func (s *Scorer) CombinedScore(relevance, spreading, genFactor float64, input Input) float64 {
	return (relevance*genFactor + s.alpha*spreading) * (1.0 + s.Quality(input))
}
```

- [ ] **Step 4: Fix existing test callers**

Update all existing `CombinedScore` calls in `frecency_test.go` to pass `1.0` as the third argument (genFactor=1.0 preserves existing behavior):

- `TestCombinedScore_Basic`: add `1.0` arg
- `TestCombinedScore_SpreadingOnly`: add `1.0` arg
- `TestCombinedScore_ZeroRelevanceZeroSpreading`: add `1.0` arg

- [ ] **Step 5: Fix surface.go — update sort function signatures**

Both sort functions need `currentProjectSlug` to compute `GenFactor` per memory. Change their signatures:

```go
// line ~1129
func sortPromptMatchesByActivation(matches []promptMatch, scorer *frecency.Scorer, currentProjectSlug string) {
    sort.SliceStable(matches, func(i, j int) bool {
        gi := GenFactor(matches[i].mem.Generalizability, matches[i].mem.ProjectSlug, currentProjectSlug)
        gj := GenFactor(matches[j].mem.Generalizability, matches[j].mem.ProjectSlug, currentProjectSlug)
        si := scorer.CombinedScore(matches[i].bm25Score, matches[i].spreadingScore, gi, toFrecencyInput(matches[i].mem))
        sj := scorer.CombinedScore(matches[j].bm25Score, matches[j].spreadingScore, gj, toFrecencyInput(matches[j].mem))
        return si > sj
    })
}

// line ~1139
func sortToolMatchesByActivation(matches []toolMatch, scorer *frecency.Scorer, currentProjectSlug string) {
    // same pattern as above
}
```

- [ ] **Step 6: Fix surface.go — update `runPrompt` to receive and pass `currentProjectSlug`**

`runPrompt` (line 301) currently takes individual fields, not `opts`. Add `currentProjectSlug string` parameter:

```go
func (s *Surfacer) runPrompt(
    ctx context.Context, dataDir, message, transcriptWindow, currentProjectSlug string,
    effectiveness map[string]EffectivenessStat, scorer *frecency.Scorer,
) ...
```

Update its call to `sortPromptMatchesByActivation` (line 363):
```go
sortPromptMatchesByActivation(matches, scorer, currentProjectSlug)
```

Update the caller in `Run()` switch (line 186-188):
```go
result, matched, suppressionEvents, err = s.runPrompt(
    ctx, opts.DataDir, opts.Message, opts.TranscriptWindow, opts.CurrentProjectSlug, effectiveness, scorer,
)
```

- [ ] **Step 7: Fix surface.go — update `runTool` sort call**

`runTool` already receives full `opts`. Update its call to `sortToolMatchesByActivation` (line 568):
```go
sortToolMatchesByActivation(candidates, scorer, opts.CurrentProjectSlug)
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `targ test -- ./internal/frecency/ ./internal/surface/`
Expected: PASS

- [ ] **Step 9: Run full check**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 10: Commit**

```bash
git add internal/frecency/frecency.go internal/frecency/frecency_test.go internal/surface/surface.go
git commit -m "feat: apply genFactor penalty to BM25 in CombinedScore (#373)

AI-Used: [claude]"
```

### Task 4: Wire `CurrentProjectSlug` through CLI

**Files:**
- Modify: `internal/surface/surface.go:92-103` (Options struct)
- Modify: `internal/cli/cli.go:1457-1498` (runSurface function)
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add `CurrentProjectSlug` to `Options`**

In `internal/surface/surface.go`, add to the `Options` struct:

```go
CurrentProjectSlug string // derived from data-dir for cross-project penalty
```

- [ ] **Step 2: Derive project slug from data-dir in CLI**

In `internal/cli/cli.go` `runSurface()`, after `applyDataDirDefault`, derive the project slug. The standard data dir path is `~/.claude/engram/data/<project-slug>/memories/`. The project slug is the leaf directory of `--data-dir` (the parent of the `memories/` subdirectory):

```go
currentProjectSlug := filepath.Base(*dataDir)
```

Pass it to Options:

```go
opts := surface.Options{
    // ... existing fields ...
    CurrentProjectSlug: currentProjectSlug,
}
```

- [ ] **Step 3: Write test for project slug derivation**

Add a test in `internal/cli/cli_test.go` that verifies the surface command passes the project slug through. This may be an integration-style test — check existing patterns in cli_test.go.

Alternatively, if cli_test.go tests are too heavy, verify via the surface package directly: create an Options with `CurrentProjectSlug` set, and verify it reaches the sort functions.

- [ ] **Step 4: Run full check**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: wire CurrentProjectSlug from data-dir into surface Options (#373)

AI-Used: [claude]"
```

### Task 5: End-to-end verification and cleanup

**Files:**
- Modify: `internal/surface/surface_test.go` (add integration-level test)

- [ ] **Step 1: Write end-to-end test**

Add a test in `surface_test.go` that verifies the full penalty flow:

1. Create two memories: one with `ProjectSlug="proj-a", Generalizability=1` (narrow) and one with `ProjectSlug="proj-a", Generalizability=5` (universal)
2. Run surfacing with `CurrentProjectSlug="proj-b"` (different project)
3. Assert the universal memory ranks higher than the narrow one (despite equal BM25 scores)
4. Run again with `CurrentProjectSlug="proj-a"` (same project)
5. Assert both memories rank equally (no penalty applied)

- [ ] **Step 2: Run test**

Run: `targ test -- -run TestGenFactor_EndToEnd ./internal/surface/`
Expected: PASS

- [ ] **Step 3: Run full check**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 4: Commit**

```bash
git add internal/surface/surface_test.go
git commit -m "test: add end-to-end test for cross-project generalizability penalty (#373)

AI-Used: [claude]"
```
