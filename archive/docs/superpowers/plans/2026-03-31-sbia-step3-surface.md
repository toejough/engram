# SBIA Step 3: Surface Upgrades Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the surface pipeline with Haiku semantic gating, cold-start budget, configurable display format, pending evaluation tracking, and SBIA display in recall.

**Architecture:** The surface pipeline (`internal/surface/surface.go`) currently does BM25 → sort → truncate → budget → suppress → render. This plan inserts two new stages (cold-start budget + Haiku gate) between budget and suppress, adds pending_evaluation writes after surfacing, replaces the hardcoded injection format with a configurable preamble showing all four SBIA fields, and updates recall to use the same SBIA display format. Policy gets six new fields (three params, three prompts). All new logic is injected via existing `SurfacerOption` pattern.

**Tech Stack:** Go, BM25, Anthropic Haiku API, TOML

**Source spec:** `docs/superpowers/specs/2026-03-29-sbia-feedback-model-design.md` — see "Surfacing Pipeline" section.

---

### Task 1: Add Surface Parameters to Policy

**Files:**
- Modify: `internal/policy/policy.go`
- Modify: `internal/policy/policy_test.go`

- [ ] **Step 1: Write failing test for new surface policy fields**

Add to `policy_test.go`:

```go
func TestDefaults_SurfaceFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	pol := policy.Defaults()

	g.Expect(pol.SurfaceCandidateCountMin).To(Equal(3))
	g.Expect(pol.SurfaceCandidateCountMax).To(Equal(8))
	g.Expect(pol.SurfaceBM25Threshold).To(Equal(0.3))
	g.Expect(pol.SurfaceColdStartBudget).To(Equal(2))
	g.Expect(pol.SurfaceIrrelevanceHalfLife).To(Equal(5))
	g.Expect(pol.SurfaceGateHaikuPrompt).NotTo(BeEmpty())
	g.Expect(pol.SurfaceInjectionPreamble).NotTo(BeEmpty())
}
```

Run: `targ test`
Expected: FAIL — fields don't exist yet.

- [ ] **Step 2: Add surface fields to Policy struct, defaults, and TOML mapping**

In `policy.go`, add to `Policy` struct after `ExtractSonnetPrompt`:

```go
	// SurfaceCandidateCountMin is the minimum BM25 candidates to retrieve before gating.
	SurfaceCandidateCountMin int

	// SurfaceCandidateCountMax is the maximum BM25 candidates to retrieve before gating.
	SurfaceCandidateCountMax int

	// SurfaceBM25Threshold is the minimum BM25 score for a candidate to be surfaced.
	SurfaceBM25Threshold float64

	// SurfaceColdStartBudget is the max unproven (never surfaced) memories per invocation.
	SurfaceColdStartBudget int

	// SurfaceIrrelevanceHalfLife is the half-life for irrelevance penalty decay.
	SurfaceIrrelevanceHalfLife int

	// SurfaceGateHaikuPrompt is the system prompt for the Haiku semantic gate.
	SurfaceGateHaikuPrompt string

	// SurfaceInjectionPreamble is the preamble injected with surfaced memories.
	SurfaceInjectionPreamble string
```

Add to `policyFileParams`:

```go
	SurfaceCandidateCountMin   int     `toml:"surface_candidate_count_min"`
	SurfaceCandidateCountMax   int     `toml:"surface_candidate_count_max"`
	SurfaceBM25Threshold       float64 `toml:"surface_bm25_threshold"`
	SurfaceColdStartBudget     int     `toml:"surface_cold_start_budget"`
	SurfaceIrrelevanceHalfLife int     `toml:"surface_irrelevance_half_life"`
```

Add to `policyFilePrompts`:

```go
	SurfaceGateHaiku      string `toml:"surface_gate_haiku"`
	SurfaceInjectionPreamble string `toml:"surface_injection_preamble"`
```

Add default constants:

```go
	defaultSurfaceCandidateCountMin   = 3
	defaultSurfaceCandidateCountMax   = 8
	defaultSurfaceBM25Threshold       = 0.3
	defaultSurfaceColdStartBudget     = 2
	defaultSurfaceIrrelevanceHalfLife = 5
	defaultSurfaceGateHaikuPrompt     = `You are a memory relevance classifier. Given the user's current context and a set of candidate memories, determine which memories are situationally relevant.

For each candidate memory, decide: does the SITUATION described in the memory match what the user is currently doing? Keyword overlap alone is NOT sufficient — the situation must genuinely apply.

Return a JSON array of the memory slugs that are relevant. Return an empty array if none are relevant.

Example: ["memory-slug-1", "memory-slug-3"]`
	defaultSurfaceInjectionPreamble = `[engram] Memories — for any relevant memory, call ` + "`engram show --name <name>`" + ` for full details. After your turn, call ` + "`engram feedback --name <name> --relevant|--irrelevant --used|--notused`" + ` for each:`
```

Add to `Defaults()`:

```go
		SurfaceCandidateCountMin:   defaultSurfaceCandidateCountMin,
		SurfaceCandidateCountMax:   defaultSurfaceCandidateCountMax,
		SurfaceBM25Threshold:       defaultSurfaceBM25Threshold,
		SurfaceColdStartBudget:     defaultSurfaceColdStartBudget,
		SurfaceIrrelevanceHalfLife: defaultSurfaceIrrelevanceHalfLife,
		SurfaceGateHaikuPrompt:     defaultSurfaceGateHaikuPrompt,
		SurfaceInjectionPreamble:   defaultSurfaceInjectionPreamble,
```

Add merge logic to `mergeParams`:

```go
	if params.SurfaceCandidateCountMin != 0 {
		pol.SurfaceCandidateCountMin = params.SurfaceCandidateCountMin
	}

	if params.SurfaceCandidateCountMax != 0 {
		pol.SurfaceCandidateCountMax = params.SurfaceCandidateCountMax
	}

	if params.SurfaceBM25Threshold != 0 {
		pol.SurfaceBM25Threshold = params.SurfaceBM25Threshold
	}

	if params.SurfaceColdStartBudget != 0 {
		pol.SurfaceColdStartBudget = params.SurfaceColdStartBudget
	}

	if params.SurfaceIrrelevanceHalfLife != 0 {
		pol.SurfaceIrrelevanceHalfLife = params.SurfaceIrrelevanceHalfLife
	}
```

Add merge logic to `mergePrompts`:

```go
	if prompts.SurfaceGateHaiku != "" {
		pol.SurfaceGateHaikuPrompt = prompts.SurfaceGateHaiku
	}

	if prompts.SurfaceInjectionPreamble != "" {
		pol.SurfaceInjectionPreamble = prompts.SurfaceInjectionPreamble
	}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 4: Write test for TOML override of surface fields**

Add to `policy_test.go`:

```go
func TestLoad_OverridesSurfaceFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tomlData := `
[parameters]
surface_candidate_count_min = 5
surface_cold_start_budget = 4

[prompts]
surface_gate_haiku = "custom gate prompt"
`

	pol, err := policy.Load(func(string) ([]byte, error) {
		return []byte(tomlData), nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pol.SurfaceCandidateCountMin).To(Equal(5))
	g.Expect(pol.SurfaceColdStartBudget).To(Equal(4))
	g.Expect(pol.SurfaceGateHaikuPrompt).To(Equal("custom gate prompt"))
	// Non-overridden fields keep defaults
	g.Expect(pol.SurfaceCandidateCountMax).To(Equal(8))
	g.Expect(pol.SurfaceInjectionPreamble).NotTo(BeEmpty())
}
```

Run: `targ test`
Expected: PASS (merge logic already handles this)

- [ ] **Step 5: Commit**

```bash
git add internal/policy/
git commit -m "feat(policy): add surface parameters and prompts for SBIA Step 3

Adds SurfaceCandidateCountMin/Max, SurfaceBM25Threshold,
SurfaceColdStartBudget, SurfaceIrrelevanceHalfLife,
SurfaceGateHaikuPrompt, and SurfaceInjectionPreamble to policy.
All configurable via policy.toml [parameters] and [prompts] sections.

AI-Used: [claude]"
```

---

### Task 2: Add SurfaceConfig to Surfacer and Wire Policy

The surface pipeline currently uses hardcoded constants (`promptLimit = 2`, `irrelevancePenaltyHalfLife = 5`). Replace with a `SurfaceConfig` struct injected via a new `WithSurfaceConfig` option, populated from `policy.Policy` at the CLI layer.

**Files:**
- Create: `internal/surface/config.go`
- Create: `internal/surface/config_test.go`
- Modify: `internal/surface/surface.go`

- [ ] **Step 1: Write failing test for SurfaceConfig**

Create `internal/surface/config_test.go`:

```go
package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

func TestDefaultSurfaceConfig(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	cfg := surface.DefaultSurfaceConfig()

	g.Expect(cfg.CandidateCountMax).To(Equal(8))
	g.Expect(cfg.CandidateCountMin).To(Equal(3))
	g.Expect(cfg.BM25Threshold).To(Equal(0.3))
	g.Expect(cfg.ColdStartBudget).To(Equal(2))
	g.Expect(cfg.IrrelevanceHalfLife).To(Equal(5))
	g.Expect(cfg.InjectionPreamble).NotTo(BeEmpty())
}
```

Run: `targ test`
Expected: FAIL — `SurfaceConfig` doesn't exist.

- [ ] **Step 2: Create config.go with SurfaceConfig**

Create `internal/surface/config.go`:

```go
package surface

// SurfaceConfig holds tunable parameters for the surface pipeline.
type SurfaceConfig struct {
	CandidateCountMin   int
	CandidateCountMax   int
	BM25Threshold       float64
	ColdStartBudget     int
	IrrelevanceHalfLife int
	GateHaikuPrompt     string
	InjectionPreamble   string
}

// DefaultSurfaceConfig returns a SurfaceConfig with default values.
func DefaultSurfaceConfig() SurfaceConfig {
	return SurfaceConfig{
		CandidateCountMin:   defaultCandidateCountMin,
		CandidateCountMax:   defaultCandidateCountMax,
		BM25Threshold:       defaultBM25Threshold,
		ColdStartBudget:     defaultColdStartBudget,
		IrrelevanceHalfLife: defaultIrrelevanceHalfLife,
		InjectionPreamble:   defaultInjectionPreamble,
	}
}

// WithSurfaceConfig sets the surface configuration.
func WithSurfaceConfig(cfg SurfaceConfig) SurfacerOption {
	return func(s *Surfacer) { s.config = cfg }
}

// unexported constants.
const (
	defaultCandidateCountMin = 3
	defaultCandidateCountMax = 8
	defaultBM25Threshold     = 0.3
	defaultColdStartBudget   = 2
	defaultIrrelevanceHalfLife = 5
	defaultInjectionPreamble = `[engram] Memories — for any relevant memory, call ` +
		"`engram show --name <name>`" +
		` for full details. After your turn, call ` +
		"`engram feedback --name <name> --relevant|--irrelevant --used|--notused`" +
		` for each:`
)
```

Add `config SurfaceConfig` field to the `Surfacer` struct in `surface.go` and initialize it in `New()`:

```go
type Surfacer struct {
	retriever             MemoryRetriever
	tracker               MemoryTracker
	surfacingLogger       SurfacingEventLogger
	invocationTokenLogger InvocationTokenLogger
	budgetConfig          *BudgetConfig
	recordSurfacing       func(path string) error
	config                SurfaceConfig
}

func New(retriever MemoryRetriever, opts ...SurfacerOption) *Surfacer {
	s := &Surfacer{
		retriever: retriever,
		config:    DefaultSurfaceConfig(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
```

- [ ] **Step 3: Replace hardcoded constants in surface.go with config fields**

In `surface.go`, replace:
- `promptLimit` constant usage (line 168) → `s.config.CandidateCountMax`
- `irrelevancePenaltyHalfLife` constant in `irrelevancePenalty` → accept `halfLife int` parameter

Change `irrelevancePenalty` signature:

```go
func irrelevancePenalty(irrelevantCount, halfLife int) float64 {
	return float64(halfLife) / float64(halfLife+irrelevantCount)
}
```

Update call in `matchPromptMemories` to accept `halfLife` parameter:

```go
func matchPromptMemories(
	message string,
	memories []*memory.Stored,
	halfLife int,
) []promptMatch {
```

And update the penalty line:

```go
		penalizedScore := result.Score * irrelevancePenalty(mem.IrrelevantCount, halfLife)
```

Update call site in `runPrompt`:

```go
	matches := matchPromptMemories(message, memories, s.config.IrrelevanceHalfLife)
```

And:

```go
	if len(matches) > s.config.CandidateCountMax {
		matches = matches[:s.config.CandidateCountMax]
	}
```

Remove the old constants `irrelevancePenaltyHalfLife` and `promptLimit` from the unexported const block.

- [ ] **Step 4: Run tests to verify nothing broke**

Run: `targ test`
Expected: PASS — behavior unchanged, just parameterized.

- [ ] **Step 5: Commit**

```bash
git add internal/surface/
git commit -m "refactor(surface): replace hardcoded constants with SurfaceConfig

Introduces SurfaceConfig struct with WithSurfaceConfig option.
Parameterizes CandidateCountMax and IrrelevanceHalfLife, removing
hardcoded promptLimit=2 and irrelevancePenaltyHalfLife=5.

AI-Used: [claude]"
```

---

### Task 3: Add BM25 Threshold Filter

Apply the `SurfaceBM25Threshold` to filter out low-scoring matches before the candidate count limit.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Write failing test for BM25 threshold**

Add to `surface_test.go`:

```go
func TestPromptMode_BM25Threshold_FiltersLowScores(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code with git",
			Behavior:  "amending pushed commits",
			Impact:    "rewrites shared history",
			Action:    "create new commits instead",
			FilePath:  "commit-safety.toml",
		},
		{
			Situation: "when deploying to production",
			Behavior:  "skipping review",
			Impact:    "bugs reach production",
			Action:    "always get review",
			FilePath:  "deploy-review.toml",
		},
	}

	cfg := surface.DefaultSurfaceConfig()
	cfg.BM25Threshold = 999.0 // impossibly high threshold

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithSurfaceConfig(cfg))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "git commit",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// High threshold filters everything
	g.Expect(buf.String()).To(BeEmpty())
}
```

Run: `targ test`
Expected: FAIL — no threshold filtering yet.

- [ ] **Step 2: Add threshold filtering to runPrompt**

In `runPrompt`, after `matchPromptMemories` and before `sortPromptMatchesByScore`, add:

```go
	// Apply BM25 threshold filter.
	if s.config.BM25Threshold > 0 {
		filtered := make([]promptMatch, 0, len(matches))
		for _, m := range matches {
			if m.bm25Score >= s.config.BM25Threshold {
				filtered = append(filtered, m)
			}
		}

		matches = filtered
	}

	if len(matches) == 0 {
		return Result{}, nil, nil
	}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/surface/
git commit -m "feat(surface): add BM25 threshold filter

Filters candidates below SurfaceConfig.BM25Threshold before
candidate count limit. Default threshold 0.3.

AI-Used: [claude]"
```

---

### Task 4: Add Cold-Start Budget

Limit the number of unproven (never surfaced) memories per invocation to `SurfaceColdStartBudget`.

**Files:**
- Create: `internal/surface/coldstart.go`
- Create: `internal/surface/coldstart_test.go`

- [ ] **Step 1: Write failing test for cold-start budget**

Create `internal/surface/coldstart_test.go`:

```go
package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestApplyColdStartBudget_LimitsUnproven(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "proven-1.toml", SurfacedCount: 5, Action: "proven 1"},
		{FilePath: "unproven-1.toml", SurfacedCount: 0, Action: "unproven 1"},
		{FilePath: "unproven-2.toml", SurfacedCount: 0, Action: "unproven 2"},
		{FilePath: "unproven-3.toml", SurfacedCount: 0, Action: "unproven 3"},
		{FilePath: "proven-2.toml", SurfacedCount: 3, Action: "proven 2"},
	}

	const budget = 2
	result := surface.ApplyColdStartBudget(candidates, budget)

	g.Expect(result).To(HaveLen(4)) // 2 proven + 2 unproven (capped)

	// All proven memories kept
	g.Expect(result[0].FilePath).To(Equal("proven-1.toml"))
	g.Expect(result[3].FilePath).To(Equal("proven-2.toml"))

	// Only first 2 unproven kept, third dropped
	unprovenPaths := []string{}
	for _, m := range result {
		if m.SurfacedCount == 0 {
			unprovenPaths = append(unprovenPaths, m.FilePath)
		}
	}

	g.Expect(unprovenPaths).To(HaveLen(2))
}

func TestApplyColdStartBudget_ZeroBudget_AllowsAll(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "unproven-1.toml", SurfacedCount: 0},
		{FilePath: "unproven-2.toml", SurfacedCount: 0},
	}

	result := surface.ApplyColdStartBudget(candidates, 0)

	g.Expect(result).To(HaveLen(2))
}
```

Run: `targ test`
Expected: FAIL — `ApplyColdStartBudget` doesn't exist.

- [ ] **Step 2: Implement cold-start budget**

Create `internal/surface/coldstart.go`:

```go
package surface

import "engram/internal/memory"

// ApplyColdStartBudget limits the number of unproven (never surfaced) memories.
// Proven memories (SurfacedCount > 0) always pass through.
// Budget of 0 means unlimited.
func ApplyColdStartBudget(candidates []*memory.Stored, budget int) []*memory.Stored {
	if budget <= 0 {
		return candidates
	}

	result := make([]*memory.Stored, 0, len(candidates))
	unprovenCount := 0

	for _, mem := range candidates {
		if mem.SurfacedCount > 0 {
			result = append(result, mem)
			continue
		}

		if unprovenCount < budget {
			result = append(result, mem)
			unprovenCount++
		}
	}

	return result
}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 4: Wire cold-start budget into runPrompt**

In `surface.go` `runPrompt`, after candidate count truncation and token budget, before transcript suppression, add:

```go
	// Apply cold-start budget for unproven memories.
	promptMems = ApplyColdStartBudget(promptMems, s.config.ColdStartBudget)
```

This goes after the existing line that builds `promptMems` from matches and before `suppressByTranscript`.

- [ ] **Step 5: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/surface/
git commit -m "feat(surface): add cold-start budget for unproven memories

Limits unproven (SurfacedCount == 0) memories to ColdStartBudget
per invocation. Proven memories always pass through. Default budget: 2.

AI-Used: [claude]"
```

---

### Task 5: Add Haiku Semantic Gate

Add a Haiku LLM call that filters BM25 candidates by situational relevance.

**Files:**
- Create: `internal/surface/gate.go`
- Create: `internal/surface/gate_test.go`
- Modify: `internal/surface/surface.go`

- [ ] **Step 1: Write failing test for gate function**

Create `internal/surface/gate_test.go`:

```go
package surface_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestGateMemories_FiltersIrrelevant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "commit-safety.toml", Situation: "when committing"},
		{FilePath: "deploy-review.toml", Situation: "when deploying"},
	}

	// Mock caller that says only commit-safety is relevant
	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return `["commit-safety"]`, nil
	}

	result, err := surface.GateMemories(
		context.Background(),
		candidates,
		"I want to commit this change",
		caller,
		"gate prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].FilePath).To(Equal("commit-safety.toml"))
}

func TestGateMemories_EmptyResponse_ReturnsNone(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "test.toml", Situation: "when testing"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return `[]`, nil
	}

	result, err := surface.GateMemories(
		context.Background(),
		candidates,
		"unrelated query",
		caller,
		"gate prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestGateMemories_CallerError_ReturnsAllCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "test.toml", Situation: "when testing"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", context.DeadlineExceeded
	}

	result, err := surface.GateMemories(
		context.Background(),
		candidates,
		"query",
		caller,
		"gate prompt",
	)

	// On error, return all candidates (fail open)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(1))
}
```

Run: `targ test`
Expected: FAIL — `GateMemories` doesn't exist.

- [ ] **Step 2: Implement gate function**

Create `internal/surface/gate.go`:

```go
package surface

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"engram/internal/memory"
)

// HaikuCallerFunc calls the Haiku API with model, system, and user prompts.
type HaikuCallerFunc func(ctx context.Context, model, system, user string) (string, error)

// GateMemories uses Haiku to filter candidates by situational relevance.
// On API error, returns all candidates (fail-open).
func GateMemories(
	ctx context.Context,
	candidates []*memory.Stored,
	userMessage string,
	caller HaikuCallerFunc,
	systemPrompt string,
) ([]*memory.Stored, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	userPrompt := buildGateUserPrompt(candidates, userMessage)

	response, err := caller(ctx, haikuModel, systemPrompt, userPrompt)
	if err != nil {
		// Fail open: return all candidates if Haiku is unavailable.
		return candidates, nil
	}

	slugs, parseErr := parseGateResponse(response)
	if parseErr != nil {
		return candidates, nil
	}

	return filterBySlug(candidates, slugs), nil
}

// WithHaikuGate sets the Haiku gate caller on the Surfacer.
func WithHaikuGate(caller HaikuCallerFunc) SurfacerOption {
	return func(s *Surfacer) { s.haikuGate = caller }
}

func buildGateUserPrompt(candidates []*memory.Stored, userMessage string) string {
	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "User context:\n%s\n\nCandidate memories:\n", userMessage)

	for _, mem := range candidates {
		slug := strings.TrimSuffix(filepath.Base(mem.FilePath), ".toml")
		_, _ = fmt.Fprintf(&buf, "\n- slug: %s\n  situation: %s\n  behavior: %s\n  impact: %s\n  action: %s\n",
			slug, mem.Situation, mem.Behavior, mem.Impact, mem.Action)
	}

	return buf.String()
}

func parseGateResponse(response string) ([]string, error) {
	var slugs []string

	err := json.Unmarshal([]byte(strings.TrimSpace(response)), &slugs)
	if err != nil {
		return nil, fmt.Errorf("parsing gate response: %w", err)
	}

	return slugs, nil
}

func filterBySlug(candidates []*memory.Stored, slugs []string) []*memory.Stored {
	slugSet := make(map[string]bool, len(slugs))
	for _, slug := range slugs {
		slugSet[slug] = true
	}

	result := make([]*memory.Stored, 0, len(slugs))

	for _, mem := range candidates {
		slug := strings.TrimSuffix(filepath.Base(mem.FilePath), ".toml")
		if slugSet[slug] {
			result = append(result, mem)
		}
	}

	return result
}

// unexported constants.
const (
	haikuModel = "claude-haiku-4-5-20251001"
)
```

Add `haikuGate HaikuCallerFunc` field to `Surfacer` struct in `surface.go`:

```go
type Surfacer struct {
	retriever             MemoryRetriever
	tracker               MemoryTracker
	surfacingLogger       SurfacingEventLogger
	invocationTokenLogger InvocationTokenLogger
	budgetConfig          *BudgetConfig
	recordSurfacing       func(path string) error
	config                SurfaceConfig
	haikuGate             HaikuCallerFunc
}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 4: Wire Haiku gate into runPrompt**

In `runPrompt`, after cold-start budget and before transcript suppression, add:

```go
	// Apply Haiku semantic gate if configured.
	if s.haikuGate != nil && s.config.GateHaikuPrompt != "" {
		promptMems, _ = GateMemories(
			ctx, promptMems, message, s.haikuGate, s.config.GateHaikuPrompt,
		)
	}
```

- [ ] **Step 5: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/surface/
git commit -m "feat(surface): add Haiku semantic gate for situational filtering

GateMemories calls Haiku to filter BM25 candidates by situation
relevance. Fails open on API error. Injected via WithHaikuGate option.

AI-Used: [claude]"
```

---

### Task 6: Add Pending Evaluation Tracking

Write `pending_evaluations` entries into memory TOML files when memories are surfaced.

**Files:**
- Create: `internal/surface/pendingeval.go`
- Create: `internal/surface/pendingeval_test.go`
- Modify: `internal/surface/surface.go`

- [ ] **Step 1: Add Options fields for pending evaluation context**

In `surface.go`, add to the `Options` struct:

```go
	SessionID  string // session ID for pending evaluation tracking
	UserPrompt string // original user prompt for pending evaluation
```

- [ ] **Step 2: Write failing test for pending evaluation writer**

Create `internal/surface/pendingeval_test.go`:

```go
package surface_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestWritePendingEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var mutations []pendingMutation
	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		var record memory.MemoryRecord
		mutate(&record)
		mutations = append(mutations, pendingMutation{path: path, record: record})

		return nil
	}

	memories := []*memory.Stored{
		{FilePath: "/data/commit-safety.toml"},
		{FilePath: "/data/test-tdd.toml"},
	}

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	err := surface.WritePendingEvaluations(memories, modifier, "sess-123", "proj-slug", "user prompt", now)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mutations).To(HaveLen(2))
	g.Expect(mutations[0].record.PendingEvaluations).To(HaveLen(1))
	g.Expect(mutations[0].record.PendingEvaluations[0].SessionID).To(Equal("sess-123"))
	g.Expect(mutations[0].record.PendingEvaluations[0].UserPrompt).To(Equal("user prompt"))
}

type pendingMutation struct {
	path   string
	record memory.MemoryRecord
}
```

Run: `targ test`
Expected: FAIL — `WritePendingEvaluations` doesn't exist.

- [ ] **Step 3: Implement pending evaluation writer**

Create `internal/surface/pendingeval.go`:

```go
package surface

import (
	"fmt"
	"time"

	"engram/internal/memory"
)

// ModifyFunc reads a memory TOML, applies a mutation, and writes it back.
type ModifyFunc func(path string, mutate func(*memory.MemoryRecord)) error

// WritePendingEvaluations appends a pending evaluation entry to each surfaced memory.
func WritePendingEvaluations(
	memories []*memory.Stored,
	modify ModifyFunc,
	sessionID, projectSlug, userPrompt string,
	now time.Time,
) error {
	for _, mem := range memories {
		writeErr := modify(mem.FilePath, func(record *memory.MemoryRecord) {
			record.PendingEvaluations = append(record.PendingEvaluations, memory.PendingEvaluation{
				SurfacedAt:  now.Format(time.RFC3339),
				UserPrompt:  userPrompt,
				SessionID:   sessionID,
				ProjectSlug: projectSlug,
			})
		})
		if writeErr != nil {
			return fmt.Errorf("writing pending evaluation for %s: %w", mem.FilePath, writeErr)
		}
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Wire pending evaluation writing into Run()**

In `surface.go`, add a new field to `Surfacer`:

```go
	pendingEvalModifier ModifyFunc
```

Add a new option:

```go
// WithPendingEvalModifier sets the modifier for writing pending evaluations.
func WithPendingEvalModifier(fn ModifyFunc) SurfacerOption {
	return func(s *Surfacer) { s.pendingEvalModifier = fn }
}
```

In `Run()`, after `recordSurfacing` and before `writeResult`, add:

```go
	if s.pendingEvalModifier != nil && len(matched) > 0 {
		_ = WritePendingEvaluations(
			matched, s.pendingEvalModifier,
			opts.SessionID, opts.CurrentProjectSlug, opts.UserPrompt,
			time.Now(),
		)
	}
```

- [ ] **Step 6: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/surface/
git commit -m "feat(surface): write pending evaluations at surface time

Appends PendingEvaluation entries to surfaced memory TOMLs for
consumption by engram evaluate at stop hook (Step 4).

AI-Used: [claude]"
```

---

### Task 7: Update Display Format to Full SBIA Fields

Replace the `- slug: action` format with full SBIA fields and configurable preamble.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Write test for new SBIA display format**

Add to `surface_test.go`:

```go
func TestPromptMode_SBIADisplayFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Behavior:  "amending pushed commits",
			Impact:    "rewrites shared history",
			Action:    "create new commits instead",
			FilePath:  "commit-safety.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "committing code changes",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("Situation:"))
	g.Expect(output).To(ContainSubstring("when committing code"))
	g.Expect(output).To(ContainSubstring("Behavior to avoid:"))
	g.Expect(output).To(ContainSubstring("amending pushed commits"))
	g.Expect(output).To(ContainSubstring("Impact if ignored:"))
	g.Expect(output).To(ContainSubstring("rewrites shared history"))
	g.Expect(output).To(ContainSubstring("Action:"))
	g.Expect(output).To(ContainSubstring("create new commits instead"))
}
```

Run: `targ test`
Expected: FAIL — current format only shows `- slug: action`.

- [ ] **Step 2: Update rendering in runPrompt**

Replace the rendering block in `runPrompt` (the `var buf strings.Builder` section through the Summary builder) with:

```go
	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "%s\n", s.config.InjectionPreamble)

	for i, match := range matches {
		slug := filenameSlug(match.mem.FilePath)
		_, _ = fmt.Fprintf(&buf, "  %d. %s\n", i+1, slug)
		_, _ = fmt.Fprintf(&buf, "     Situation: %s\n", match.mem.Situation)
		_, _ = fmt.Fprintf(&buf, "     Behavior to avoid: %s\n", match.mem.Behavior)
		_, _ = fmt.Fprintf(&buf, "     Impact if ignored: %s\n", match.mem.Impact)
		_, _ = fmt.Fprintf(&buf, "     Action: %s\n", match.mem.Action)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	var summaryBuf strings.Builder

	_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d relevant memories:\n", len(matches))

	for _, match := range matches {
		_, _ = fmt.Fprintf(&summaryBuf, "  - %s: %s\n",
			filenameSlug(match.mem.FilePath), match.mem.Action)
	}
```

The summary stays compact (slug + action) since it appears in the status line. Only the Context (injected into additionalContext) gets the full SBIA format.

- [ ] **Step 3: Update existing tests that assert on old format**

Update `TestPromptMode_JSONFormat` and `TestPromptMode_KeywordMatch_SurfacesRelevant` to expect the new format (check for `Situation:` instead of `- slug: action` in `Context` field).

- [ ] **Step 4: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/surface/
git commit -m "feat(surface): display full SBIA fields in injection format

Injection context now shows numbered memories with Situation,
Behavior to avoid, Impact if ignored, and Action fields.
Preamble is configurable via SurfaceConfig.InjectionPreamble.

AI-Used: [claude]"
```

---

### Task 8: Update Recall to Use SBIA Display Format

Recall uses the same surface pipeline via `RecallSurfacer`, so it automatically gets the new format. However, the recall output for mode A (raw transcript) also surfaces memories — verify this works and update the recall skill's instructions if needed.

**Files:**
- Modify: `internal/cli/recallsurfacer.go` (if needed)
- Verify: `internal/recall/orchestrate.go`

- [ ] **Step 1: Verify recall surfacer passes through new format**

The `RecallSurfacer` calls `surface.Run()` with `ModePrompt` and captures the output. Since it writes plain text (not JSON), it gets the new SBIA format automatically. Read through the code to confirm no format assumptions are violated.

Check that `surfaceRunnerAdapter.Run()` doesn't strip or reformat the output.

- [ ] **Step 2: Write test confirming SBIA format in recall output**

Add to `internal/cli/recallsurfacer_test.go`:

```go
func TestRecallSurfacer_SBIAFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	runner := &fakeSurfaceRunner{
		output: "<system-reminder source=\"engram\">\n  1. test-mem\n     Situation: when testing\n</system-reminder>\n",
	}

	surfacer := cli.NewRecallSurfacer(runner, "/tmp/data")
	result, err := surfacer.Surface("testing")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("Situation:"))
}
```

Run: `targ test`
Expected: PASS (passthrough works)

- [ ] **Step 3: Commit**

```bash
git add internal/cli/
git commit -m "test(recall): verify SBIA display format in recall surfacer

RecallSurfacer passes through the new SBIA injection format
from the surface pipeline without modification.

AI-Used: [claude]"
```

---

### Task 9: Wire Policy and Haiku Gate in CLI

Connect policy loading and Haiku caller to the surface pipeline in `runSurface`.

**Files:**
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Wire SurfaceConfig from policy in runSurface**

In `runSurface()`, after `applyDataDirDefault`, load policy and build config:

```go
	policyPath := filepath.Join(*dataDir, "policy.toml")
	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		pol = policy.Defaults()
	}

	cfg := surface.SurfaceConfig{
		CandidateCountMin:   pol.SurfaceCandidateCountMin,
		CandidateCountMax:   pol.SurfaceCandidateCountMax,
		BM25Threshold:       pol.SurfaceBM25Threshold,
		ColdStartBudget:     pol.SurfaceColdStartBudget,
		IrrelevanceHalfLife: pol.SurfaceIrrelevanceHalfLife,
		InjectionPreamble:   pol.SurfaceInjectionPreamble,
		GateHaikuPrompt:     pol.SurfaceGateHaikuPrompt,
	}
```

Add `surface.WithSurfaceConfig(cfg)` to `surfacerOpts`.

- [ ] **Step 2: Wire Haiku gate caller**

In `runSurface()`, after building `surfacerOpts`, add:

```go
	token := os.Getenv("ANTHROPIC_API_KEY")
	if token != "" {
		caller := makeAnthropicCaller(token)
		surfacerOpts = append(surfacerOpts, surface.WithHaikuGate(caller))
	}
```

This uses the existing `makeAnthropicCaller` function.

- [ ] **Step 3: Wire pending evaluation modifier**

In `runSurface()`, add to `surfacerOpts`:

```go
	surfacerOpts = append(surfacerOpts, surface.WithPendingEvalModifier(
		func(path string, mutate func(*memory.MemoryRecord)) error {
			return defaultModifier.ReadModifyWrite(path, mutate)
		},
	))
```

- [ ] **Step 4: Pass SessionID and UserPrompt through Options**

In the `opts` construction, add the new fields:

```go
	opts := surface.Options{
		Mode:               *mode,
		DataDir:            *dataDir,
		Message:            *message,
		Format:             *format,
		TranscriptWindow:   *transcriptWindow,
		CurrentProjectSlug: currentProjectSlug,
		SessionID:          *sessionID,
		UserPrompt:         *message,
	}
```

Note: `sessionID` flag already exists in `runSurface`. `UserPrompt` is set to the message.

- [ ] **Step 5: Add policy import if not already present**

Add `"engram/internal/policy"` to imports in `cli.go`.

- [ ] **Step 6: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/
git commit -m "feat(cli): wire policy, Haiku gate, and pending evals into surface pipeline

runSurface now loads SurfaceConfig from policy.toml, wires
Haiku gate caller when API key is available, and enables
pending evaluation writes. SessionID and UserPrompt passed
through for evaluation tracking.

AI-Used: [claude]"
```

---

### Task 10: Pass Session ID Through Hooks

The hooks need to pass the session ID to `engram surface` for pending evaluation tracking.

**Files:**
- Modify: `hooks/user-prompt-submit.sh`
- Modify: `hooks/stop-surface.sh`

- [ ] **Step 1: Add session ID to user-prompt-submit hook**

In `hooks/user-prompt-submit.sh`, after the `TRANSCRIPT_PATH` extraction, add:

```bash
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"
```

Update the surface call to include session ID:

```bash
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode prompt \
        --message "$USER_MESSAGE" --session-id "$SESSION_ID" --format json) || true
```

- [ ] **Step 2: Verify stop-surface.sh already passes session ID**

Check that `stop-surface.sh` already passes `--session-id "$SESSION_ID"`. It does (from the code read earlier).

- [ ] **Step 3: Run hooks manually to verify**

```bash
echo '{"prompt": "test", "session_id": "sess-1", "transcript_path": ""}' | bash hooks/user-prompt-submit.sh
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add hooks/
git commit -m "feat(hooks): pass session ID to surface for pending evaluation tracking

user-prompt-submit.sh now extracts session_id from hook JSON
and passes it to engram surface via --session-id flag.

AI-Used: [claude]"
```

---

### Task 11: Clean Up Dead Code

Remove dead code and stubs that are no longer needed after the surface upgrades.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/suppress_p4f.go`
- Modify: `internal/surface/budget.go`

- [ ] **Step 1: Remove EffectivenessComputer stub**

Delete the `EffectivenessComputer` interface, `EffectivenessStat` struct, and `WithEffectiveness` no-op stub from `surface.go`. These were marked "stubbed during SBIA migration" and are not used.

- [ ] **Step 2: Remove CrossRefChecker dead code**

Delete the `CrossRefChecker` interface and `SuppressionReasonCrossSource` constant from `suppress_p4f.go`. These are defined but never used.

- [ ] **Step 3: Remove unused budget constants**

`DefaultSessionStartBudget` and `DefaultStopBudget` are defined but never used (only `DefaultUserPromptSubmitBudget` is referenced via `ForMode`). Remove the unused constants. Remove the `SessionStart` and `Stop` fields from `BudgetConfig` and `DefaultBudgetConfig()` if no other code references them.

Check first: `grep -r "SessionStart\|DefaultSessionStartBudget\|DefaultStopBudget\|\.Stop " internal/` to confirm they're unused.

- [ ] **Step 4: Run all tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full quality check**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 6: Commit**

```bash
git add internal/surface/
git commit -m "chore(surface): remove dead code stubs and unused budget constants

Removes EffectivenessComputer stub, CrossRefChecker dead interface,
and unused SessionStart/Stop budget constants. All were remnants of
pre-SBIA migration code.

AI-Used: [claude]"
```

---

### Task 12: Final Integration Verification

- [ ] **Step 1: Run targ check-full**

Run: `targ check-full`
Expected: All 8 checks pass (coverage, lint, nils, reorder, deadcode, thin-api, uncommitted)

- [ ] **Step 2: Verify surface output format manually**

Build and test with a real memory:

```bash
targ build
~/.claude/engram/bin/engram surface --mode prompt --message "I want to commit code" --format json --data-dir ~/.claude/engram
```

Expected: JSON output with full SBIA fields in the `context` field.

- [ ] **Step 3: Commit any remaining fixes**

If the integration test reveals issues, fix and commit.

---

## File Map Summary

| Action | Files | Count |
|--------|-------|-------|
| **Modify** | `internal/policy/policy.go`, test | 2 |
| **Create** | `internal/surface/config.go`, test | 2 |
| **Create** | `internal/surface/coldstart.go`, test | 2 |
| **Create** | `internal/surface/gate.go`, test | 2 |
| **Create** | `internal/surface/pendingeval.go`, test | 2 |
| **Modify** | `internal/surface/surface.go`, test | 2 |
| **Modify** | `internal/cli/cli.go` | 1 |
| **Modify** | `internal/cli/recallsurfacer_test.go` | 1 |
| **Modify** | `hooks/user-prompt-submit.sh` | 1 |
| **Modify** | `internal/surface/suppress_p4f.go`, `budget.go` | 2 |

## What Works After Step 3

| Feature | Status |
|---------|--------|
| BM25 + irrelevance penalty | Works (parameterized via policy) |
| BM25 threshold filter | New — filters low scores |
| Cold-start budget | New — limits unproven memories to 2/invocation |
| Haiku semantic gate | New — situational relevance filter |
| Pending evaluations | New — written at surface time for Step 4 |
| Full SBIA display | New — all 4 fields in injection context |
| Configurable preamble | New — via policy.toml |
| Recall SBIA display | Works — passthrough from surface |
| `engram feedback` shim | Still works (removed in Step 4) |
