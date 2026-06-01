# L3 Tier — Binary Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the engram binary the three primitives the L3 synthesis flow needs: a `tier` tag on notes, a `query --tier` cap, and a helper that scores a cluster centroid against existing notes by cosine similarity.

**Architecture:** `tier` is a frontmatter field — derived `episode→L1`, `fact/feedback→L2`, and set to `L3` via an explicit `--tier` flag (so the skill can write ADRs). `query` gains a `--tier` filter that mirrors the existing `--project` filter (post-pipeline item filter, default off). A new `internal/cluster` helper reuses the existing cosine/centroid math to find the best-matching existing note for a cluster centroid.

**Tech Stack:** Go; `targ` build tool (`targ test`, `targ check-full` — never `go test`/`go build`); `gopkg.in/yaml.v3`; packages `internal/cli`, `internal/cluster`, `internal/embed`. Tests are blackbox (`package foo_test`), gomega assertions, with the nilaway guard pattern (after `g.Expect(err).NotTo(HaveOccurred())` add `if err != nil { return }` before using values).

---

### Task 1: `tier` frontmatter field on notes

**Files:**
- Modify: `internal/cli/learn.go` (frontmatter doc structs ~133/172/197; field structs ~115/159/185; render functions ~518/548/572; `assembleLearnContent` ~231)
- Modify: `internal/cli/targets.go` (`CommonLearnArgs` ~17-26 — add `Tier` flag)
- Test: `internal/cli/learn_test.go`

- [ ] **Step 1: Write the failing test.** Episodes get `tier: L1`, facts/feedback `tier: L2`, and `--tier L3` overrides to `tier: L3`.

```go
func TestLearn_TierDerivedAndOverridable(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cases := []struct{ typ, tierFlag, want string }{
		{"episode", "", "L1"},
		{"fact", "", "L2"},
		{"feedback", "", "L2"},
		{"fact", "L3", "L3"},
	}
	for _, c := range cases {
		body, err := assembleLearnContentForTest(c.typ, c.tierFlag) // thin export_test helper around assembleLearnContent
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil { return }
		g.Expect(body).To(ContainSubstring("tier: " + c.want))
	}
}
```

- [ ] **Step 2: Run to verify it fails.** `targ test` — expect FAIL (no `tier` rendered / helper missing).

- [ ] **Step 3: Implement.**
  - Add `Tier string \`yaml:"tier,omitempty"\`` immediately after the `Type` field in `episodeFrontmatterDoc`, `factFrontmatterDoc`, `feedbackFrontmatterDoc`.
  - Add `Tier string` to `episodeFields`, `factFields`, `feedbackFields`.
  - In `assembleLearnContent`, before rendering: episodes set `Tier = "L1"`; fact/feedback set `Tier = args.Tier` if non-empty else `"L2"`. Pass `Tier: f.Tier` into each `*FrontmatterDoc{...}` literal in the render funcs.
  - Add `Tier string \`targ:"flag,name=tier,desc=tier override L1|L2|L3 (optional; default derived from type)"\`` to `CommonLearnArgs`; thread it into `LearnArgs`/`args.Tier`.
  - Add the `assembleLearnContentForTest` helper in `export_test.go`.

- [ ] **Step 4: Run to verify it passes.** `targ test` — expect PASS.

- [ ] **Step 5: Validate input.** Add a sentinel `var errLearnBadTier = errors.New("tier must be L1, L2, or L3")` and reject other `--tier` values in `assembleLearnContent`. Add a test asserting `--tier L9` returns `MatchError(errLearnBadTier)`.

- [ ] **Step 6: Commit.**

```bash
git add internal/cli/learn.go internal/cli/targets.go internal/cli/learn_test.go internal/cli/export_test.go
git commit -m "feat(learn): tier frontmatter field (derived L1/L2, --tier override for L3)

AI-Used: [claude]"
```

---

### Task 2: `engram query --tier` cap

**Files:**
- Modify: `internal/cli/query.go` (`QueryArgs` ~22-28; add `tierLineRE` near `projectLineRE` ~110; add `applyTierFilter` + `itemMatchesTier` mirroring `applyProjectFilter`/`itemMatchesProject` ~331/647; call at ~77)
- Modify: `internal/cli/targets.go` (`QueryArgs` flag wiring)
- Test: `internal/cli/query_test.go` (or `query_pipeline_test.go`)

- [ ] **Step 1: Write the failing test.** `--tier L3` returns only `tier: L3` items.

```go
func TestRunQuery_TierFilterReturnsOnlyThatTier(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	// vault fixture with one L2 fact and one L3 adr-fact, both matching the phrase
	payload, err := runQueryForTest(vaultWithL2andL3, QueryArgs{Phrases: []string{"go cli architecture"}, Tier: "L3"})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	for _, it := range payload.Items {
		g.Expect(it.Content).To(ContainSubstring("tier: L3"))
	}
	g.Expect(payload.Items).NotTo(BeEmpty())
}
```

- [ ] **Step 2: Run to verify it fails.** `targ test` — expect FAIL (Tier field/flag absent).

- [ ] **Step 3: Implement** by mirroring the `--project` path exactly:
  - `QueryArgs`: add `Tier string`.
  - `tierLineRE = regexp.MustCompile(\`(?m)^tier:\s*(L[0-9]+)\s*$\`)`.
  - `applyTierFilter(items []resolvedItem, tier string) []resolvedItem` — returns items unchanged if `tier == ""`; else keeps only items whose content frontmatter `tier:` equals `tier` (drop items with empty content, same as project).
  - Call it right after the `applyProjectFilter` line (~77): `merged.resolvedItems = applyTierFilter(merged.resolvedItems, args.Tier)`.
  - Add the `--tier` flag to the query `QueryArgs` in `targets.go`.

- [ ] **Step 4: Run to verify it passes.** `targ test` — expect PASS.

- [ ] **Step 5: Coverage — empty/unset.** Add a test that `Tier: ""` returns items of all tiers (filter is a no-op), guarding the default-unchanged behavior.

- [ ] **Step 6: Commit.**

```bash
git add internal/cli/query.go internal/cli/targets.go internal/cli/query_test.go
git commit -m "feat(query): --tier filter caps results to one tier (mirrors --project)

AI-Used: [claude]"
```

---

### Task 3: cluster-centroid → existing-note cosine match helper

**Files:**
- Create: `internal/cluster/match.go`
- Test: `internal/cluster/match_test.go`

- [ ] **Step 1: Write the failing test.**

```go
func TestBestMatch_PicksHighestSimilarityAboveThreshold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	centroid := []float32{1, 0, 0}
	candidates := [][]float32{{0, 1, 0}, {0.95, 0.05, 0}, {1, 0, 0}}
	idx, sim := cluster.BestMatch(centroid, candidates, 0.9)
	g.Expect(idx).To(Equal(2))          // exact match wins
	g.Expect(sim).To(BeNumerically(">=", float32(0.9)))
}

func TestBestMatch_NoneAboveThresholdReturnsMinusOne(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	idx, _ := cluster.BestMatch([]float32{1, 0, 0}, [][]float32{{0, 1, 0}}, 0.9)
	g.Expect(idx).To(Equal(-1))
}
```

- [ ] **Step 2: Run to verify it fails.** `targ test` — expect FAIL (`BestMatch` undefined).

- [ ] **Step 3: Implement** in `internal/cluster/match.go`, reusing the existing `CosineDistance` (`internal/cluster/distance.go`):

```go
package cluster

// BestMatch returns the index of the candidate most similar (cosine) to centroid,
// and that similarity, but only if it meets threshold; otherwise (-1, best-seen).
// Similarity = 1 - CosineDistance. Used to decide whether a fresh L2 cluster maps
// onto an existing L3 (update) or is a new topic (create).
func BestMatch(centroid []float32, candidates [][]float32, threshold float32) (int, float32) {
	bestIdx, bestSim := -1, float32(-1)
	for i, c := range candidates {
		sim := 1 - CosineDistance(centroid, c)
		if sim > bestSim {
			bestSim = sim
			if sim >= threshold {
				bestIdx = i
			}
		}
	}
	return bestIdx, bestSim
}
```

- [ ] **Step 4: Run to verify it passes.** `targ test` — expect PASS.

- [ ] **Step 5: Property test (rapid).** Add a rapid test: for random centroid + candidates, the returned index (when >= 0) always has similarity `>= threshold` and is the argmax similarity.

- [ ] **Step 6: Commit.**

```bash
git add internal/cluster/match.go internal/cluster/match_test.go
git commit -m "feat(cluster): BestMatch centroid->note cosine matcher for L3 update-vs-create

AI-Used: [claude]"
```

---

### Final: full check

- [ ] Run `targ check-full` (all 8 checks, all errors at once) and fix anything (coverage, nilaway, lint). Commit any fixups separately.

## Self-review notes
- **Spec coverage:** tier tag (Task 1), `--tier` cap default-off (Task 2, decision 3), semantic match primitive for the 90% rule (Task 3, decision 2 = centroid cosine). L3-as-`fact`-with-`tier:L3` keeps the binary's type system unchanged; the ADR *shape* (tested variable, decision 4) lives in the note body authored by the skill — no binary cost.
- **Out of scope here (Plan 2):** scenario-seed search, L2 discoverability-tweak, ADR authoring, the update-vs-create orchestration — all skill-side, consuming these three primitives.
