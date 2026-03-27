# Configuration Overrides Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make hardcoded thresholds overridable via maintenance policies (#397) and adaptation config in policy.toml (#402).

**Architecture:** Functional options for review/maintain threshold overrides (same pattern as surfacing overrides). AdaptationConfig struct on policy.File for adaptation settings with zero-value fallback to defaults.

**Tech Stack:** Go, TOML, DI via functional options, `targ` build system

**Issues:** #397, #402

---

### Task 1: Add Functional Options to review.Classify (#397)

**Files:**
- Modify: `internal/review/review.go`
- Modify: `internal/review/review_test.go`

- [ ] **Step 1: Write failing test for threshold overrides**

Add to `internal/review/review_test.go`:

```go
func TestClassify_WithCustomThresholds(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	stats := map[string]effectiveness.Stat{
		"mem-1": {FollowedCount: 3, IgnoredCount: 7, EffectivenessScore: 30.0},
		"mem-2": {FollowedCount: 4, IgnoredCount: 1, EffectivenessScore: 80.0},
	}

	tracking := map[string]review.TrackingData{
		"mem-1": {SurfacedCount: 10},
		"mem-2": {SurfacedCount: 10},
	}

	// With default thresholds: mem-1 is Leech (30% < 50%), flagged (30% < 40%)
	defaultResult := review.Classify(stats, tracking)
	g.Expect(defaultResult[0].Name).To(Equal("mem-1"))
	g.Expect(defaultResult[0].Flagged).To(BeTrue())
	g.Expect(defaultResult[0].Quadrant).To(Equal(review.Leech))

	// With lowered thresholds: effectivenessThreshold=25, flagThreshold=20
	// mem-1 (30%) is now Working (30% >= 25%), not flagged (30% >= 20%)
	customResult := review.Classify(stats, tracking,
		review.WithEffectivenessThreshold(25.0),
		review.WithFlagThreshold(20.0),
	)

	var mem1 review.ClassifiedMemory
	for _, m := range customResult {
		if m.Name == "mem-1" {
			mem1 = m
			break
		}
	}

	g.Expect(mem1.Quadrant).To(Equal(review.Working))
	g.Expect(mem1.Flagged).To(BeFalse())
}

func TestClassify_WithCustomMinEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	stats := map[string]effectiveness.Stat{
		"mem-1": {FollowedCount: 2, IgnoredCount: 1, EffectivenessScore: 66.7},
	}

	tracking := map[string]review.TrackingData{
		"mem-1": {SurfacedCount: 10},
	}

	// Default: 3 evals < 5 min → InsufficientData
	defaultResult := review.Classify(stats, tracking)
	g.Expect(defaultResult[0].Quadrant).To(Equal(review.InsufficientData))

	// With minEvaluations=2: 3 evals >= 2 → gets a real quadrant
	customResult := review.Classify(stats, tracking,
		review.WithMinEvaluations(2),
	)

	g.Expect(customResult[0].Quadrant).NotTo(Equal(review.InsufficientData))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `WithEffectivenessThreshold` undefined

- [ ] **Step 3: Add ClassifyOption type and options to review.go**

Add a config struct, option type, and option functions. Change `Classify` to accept variadic options:

```go
// ClassifyOption configures Classify behavior.
type ClassifyOption func(*classifyConfig)

type classifyConfig struct {
	effectivenessThreshold float64
	flagThreshold          float64
	minEvaluations         int
}

func defaultClassifyConfig() classifyConfig {
	return classifyConfig{
		effectivenessThreshold: effectivenessThreshold,
		flagThreshold:          flagThreshold,
		minEvaluations:         minEvaluations,
	}
}

// WithEffectivenessThreshold overrides the quadrant boundary for high effectiveness.
func WithEffectivenessThreshold(v float64) ClassifyOption {
	return func(c *classifyConfig) { c.effectivenessThreshold = v }
}

// WithFlagThreshold overrides the threshold below which memories are flagged.
func WithFlagThreshold(v float64) ClassifyOption {
	return func(c *classifyConfig) { c.flagThreshold = v }
}

// WithMinEvaluations overrides the minimum feedback events for quadrant assignment.
func WithMinEvaluations(v int) ClassifyOption {
	return func(c *classifyConfig) { c.minEvaluations = v }
}
```

Change the `Classify` signature to:
```go
func Classify(
	stats map[string]effectiveness.Stat,
	tracking map[string]TrackingData,
	opts ...ClassifyOption,
) []ClassifiedMemory {
	cfg := defaultClassifyConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
```

Replace uses of package-level constants with `cfg.effectivenessThreshold`, `cfg.flagThreshold`, `cfg.minEvaluations` inside Classify. The `assignQuadrant` function needs to receive `effectivenessThreshold` as a parameter:

```go
func assignQuadrant(surfaced int, score, median, effThreshold float64) Quadrant {
	aboveMedian := float64(surfaced) > median
	highFollowThrough := score >= effThreshold
```

Update the call in Classify:
```go
mem.Quadrant = assignQuadrant(trackData.SurfacedCount, stat.EffectivenessScore, median, cfg.effectivenessThreshold)
mem.Flagged = stat.EffectivenessScore < cfg.flagThreshold
```

And:
```go
if total < cfg.minEvaluations {
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Only check-uncommitted fails

- [ ] **Step 6: Commit**

```
feat(review): add functional options for threshold overrides (#397)

Classify now accepts WithEffectivenessThreshold, WithFlagThreshold,
and WithMinEvaluations options. Defaults unchanged.

AI-Used: [claude]
```

---

### Task 2: Add Functional Options to maintain.Generator (#397)

**Files:**
- Modify: `internal/maintain/maintain.go`
- Modify: `internal/maintain/maintain_test.go`

- [ ] **Step 1: Write failing test for staleness threshold override**

Add to `internal/maintain/maintain_test.go`:

```go
func TestGenerate_WithCustomStalenessThreshold(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-60 * 24 * time.Hour) // 60 days ago

	classified := []review.ClassifiedMemory{
		{Name: "mem-1", Quadrant: review.Working, SurfacedCount: 10, EffectivenessScore: 70.0},
	}

	memories := map[string]*memory.Stored{
		"mem-1": {UpdatedAt: updatedAt},
	}

	// Default staleness = 90 days: 60 days < 90 → no proposal
	genDefault := maintain.New(maintain.WithNow(func() time.Time { return now }))

	proposalsDefault := genDefault.Generate(context.Background(), classified, memories)
	g.Expect(proposalsDefault).To(BeEmpty())

	// Custom staleness = 30 days: 60 days > 30 → staleness proposal
	genCustom := maintain.New(
		maintain.WithNow(func() time.Time { return now }),
		maintain.WithStalenessThresholdDays(30),
	)

	proposalsCustom := genCustom.Generate(context.Background(), classified, memories)
	g.Expect(proposalsCustom).To(HaveLen(1))
	g.Expect(proposalsCustom[0].Action).To(Equal("review_staleness"))
}

func TestGenerate_WithCustomIrrelevanceThreshold(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	classified := []review.ClassifiedMemory{
		{Name: "mem-1", Quadrant: review.Working, SurfacedCount: 10, EffectivenessScore: 70.0},
	}

	memories := map[string]*memory.Stored{
		"mem-1": {
			FollowedCount:   5,
			IrrelevantCount: 4,
			IgnoredCount:    1,
			UpdatedAt:       time.Now(),
		},
	}

	// Default threshold=0.6: irrelevance ratio 4/10=0.4, below → no proposal
	genDefault := maintain.New()

	proposalsDefault := genDefault.Generate(context.Background(), classified, memories)
	g.Expect(proposalsDefault).To(BeEmpty())

	// Custom threshold=0.3: 0.4 > 0.3 → refine_keywords proposal
	genCustom := maintain.New(maintain.WithRefineKeywordsIrrelevanceThreshold(0.3))

	proposalsCustom := genCustom.Generate(context.Background(), classified, memories)
	g.Expect(proposalsCustom).To(HaveLen(1))
	g.Expect(proposalsCustom[0].Action).To(Equal("refine_keywords"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `WithStalenessThresholdDays` undefined

- [ ] **Step 3: Add threshold fields to Generator and option functions**

Add fields to the `Generator` struct:

```go
type Generator struct {
	llmCaller                         func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	now                               func() time.Time
	consolidator                      interface {
		BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (ConsolidateResult, error)
	}
	memLoader                         func(path string) (*memory.MemoryRecord, error)
	stalenessThresholdDays            int
	refineKeywordsIrrelevanceThreshold float64
}
```

Update `New` to set defaults:

```go
func New(opts ...Option) *Generator {
	gen := &Generator{
		now:                                time.Now,
		stalenessThresholdDays:             stalenessThresholdDays,
		refineKeywordsIrrelevanceThreshold: refineKeywordsIrrelevanceThreshold,
	}
	for _, opt := range opts {
		opt(gen)
	}

	return gen
}
```

Add option functions:

```go
// WithStalenessThresholdDays overrides the days before a working memory is considered stale.
func WithStalenessThresholdDays(days int) Option {
	return func(g *Generator) { g.stalenessThresholdDays = days }
}

// WithRefineKeywordsIrrelevanceThreshold overrides the irrelevance ratio triggering keyword refinement.
func WithRefineKeywordsIrrelevanceThreshold(threshold float64) Option {
	return func(g *Generator) { g.refineKeywordsIrrelevanceThreshold = threshold }
}
```

In `handleWorking`, replace the constant with the field:
```go
if ageDays <= g.stalenessThresholdDays {
```

In `checkIrrelevance`, replace the constant with the field:
```go
if ratio <= g.refineKeywordsIrrelevanceThreshold {
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Only check-uncommitted fails

- [ ] **Step 6: Commit**

```
feat(maintain): add functional options for threshold overrides (#397)

Generator now accepts WithStalenessThresholdDays and
WithRefineKeywordsIrrelevanceThreshold options. Defaults unchanged.

AI-Used: [claude]
```

---

### Task 3: Wire Maintenance Policy Overrides in CLI (#397)

**Files:**
- Modify: `internal/cli/cli.go` (RunMaintain and RunReview paths)

- [ ] **Step 1: Add maintenancePolicyToReviewOpts helper**

In `internal/cli/cli.go`, add:

```go
// maintenancePolicyToReviewOpts converts active maintenance policies to review.ClassifyOption values.
func maintenancePolicyToReviewOpts(pf *policy.File) []review.ClassifyOption {
	opts := make([]review.ClassifyOption, 0)

	for _, p := range pf.Active(policy.DimensionMaintenance) {
		switch p.Parameter {
		case "effectivenessThreshold":
			opts = append(opts, review.WithEffectivenessThreshold(p.Value))
		case "flagThreshold":
			opts = append(opts, review.WithFlagThreshold(p.Value))
		case "minEvaluations":
			opts = append(opts, review.WithMinEvaluations(int(p.Value)))
		}
	}

	return opts
}

// maintenancePolicyToGeneratorOpts converts active maintenance policies to maintain.Option values.
func maintenancePolicyToGeneratorOpts(pf *policy.File) []maintain.Option {
	opts := make([]maintain.Option, 0)

	for _, p := range pf.Active(policy.DimensionMaintenance) {
		switch p.Parameter {
		case "stalenessThresholdDays":
			opts = append(opts, maintain.WithStalenessThresholdDays(int(p.Value)))
		case "refineKeywordsIrrelevanceThreshold":
			opts = append(opts, maintain.WithRefineKeywordsIrrelevanceThreshold(p.Value))
		}
	}

	return opts
}
```

- [ ] **Step 2: Wire into RunReview and RunMaintain**

Find where `review.Classify` is called in cli.go. Pass the policy overrides:

```go
policyPath := filepath.Join(dataDir, "policy.toml")
pf, _ := policy.Load(policyPath)
reviewOpts := maintenancePolicyToReviewOpts(pf)
classified := review.Classify(stats, tracking, reviewOpts...)
```

Find where `maintain.New` is called. Pass the policy overrides:

```go
generatorOpts := maintenancePolicyToGeneratorOpts(pf)
// append to existing opts if any
gen := maintain.New(append(existingOpts, generatorOpts...)...)
```

Look for the exact locations in cli.go — search for `review.Classify(` and `maintain.New(`.

- [ ] **Step 3: Write test for the helper functions**

Add to a test file in `internal/cli/`:

```go
func TestMaintenancePolicyToReviewOpts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "effectivenessThreshold",
				Value:     45.0,
			},
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "flagThreshold",
				Value:     30.0,
			},
			{
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				Parameter: "wEff",
				Value:     0.5,
			},
		},
	}

	opts := maintenancePolicyToReviewOpts(pf)

	// Only maintenance policies converted (not surfacing)
	g.Expect(opts).To(HaveLen(2))
}

func TestMaintenancePolicyToGeneratorOpts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "stalenessThresholdDays",
				Value:     60.0,
			},
		},
	}

	opts := maintenancePolicyToGeneratorOpts(pf)
	g.Expect(opts).To(HaveLen(1))
}
```

- [ ] **Step 4: Run full check**

Run: `targ check-full`
Expected: Only check-uncommitted fails

- [ ] **Step 5: Commit**

```
feat(cli): wire maintenance policy overrides into review and maintain (#397)

Active DimensionMaintenance policies now override effectivenessThreshold,
flagThreshold, minEvaluations, stalenessThresholdDays, and
refineKeywordsIrrelevanceThreshold via the same pattern as surfacing overrides.

AI-Used: [claude]
```

---

### Task 4: Add AdaptationConfig to policy.File (#402)

**Files:**
- Modify: `internal/policy/policy.go`
- Modify: `internal/policy/policy_test.go`

- [ ] **Step 1: Write failing test for AdaptationConfig TOML round-trip**

Add to `internal/policy/policy_test.go`:

```go
func TestAdaptationConfig_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pf := &policy.File{
		Adaptation: policy.AdaptationConfig{
			MinClusterSize:         7,
			MinFeedbackEvents:      4,
			MeasurementWindow:      15,
			MaintenanceMinOutcomes: 5,
			MaintenanceMinSuccess:  0.5,
			MinNewFeedback:         8,
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	err := policy.Save(path, pf)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())
	if loadErr != nil {
		return
	}

	g.Expect(loaded.Adaptation.MinClusterSize).To(Equal(7))
	g.Expect(loaded.Adaptation.MinFeedbackEvents).To(Equal(4))
	g.Expect(loaded.Adaptation.MeasurementWindow).To(Equal(15))
	g.Expect(loaded.Adaptation.MaintenanceMinOutcomes).To(Equal(5))
	g.Expect(loaded.Adaptation.MaintenanceMinSuccess).To(BeNumerically("~", 0.5, 0.001))
	g.Expect(loaded.Adaptation.MinNewFeedback).To(Equal(8))
}

func TestAdaptationConfig_EmptyOmitted(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pf := &policy.File{}

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	err := policy.Save(path, pf)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())
	if loadErr != nil {
		return
	}

	g.Expect(loaded.Adaptation.MinClusterSize).To(Equal(0))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `AdaptationConfig` undefined

- [ ] **Step 3: Add AdaptationConfig struct and field**

Add to `internal/policy/policy.go`:

```go
// AdaptationConfig holds configurable thresholds for the adaptation analysis engine.
// Zero values mean "use default".
type AdaptationConfig struct {
	MinClusterSize         int     `toml:"min_cluster_size,omitempty"`
	MinFeedbackEvents      int     `toml:"min_feedback_events,omitempty"`
	MeasurementWindow      int     `toml:"measurement_window,omitempty"`
	MaintenanceMinOutcomes int     `toml:"maintenance_min_outcomes,omitempty"`
	MaintenanceMinSuccess  float64 `toml:"maintenance_min_success,omitempty"`
	MinNewFeedback         int     `toml:"min_new_feedback,omitempty"`
}
```

Add to the `File` struct:

```go
type File struct {
	Policies       []Policy         `toml:"policies"`
	ApprovalStreak ApprovalStreak   `toml:"approval_streak"`
	Adaptation     AdaptationConfig `toml:"adaptation"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(policy): add AdaptationConfig to policy.File (#402)

TOML-persisted adaptation thresholds in [adaptation] section of
policy.toml. Zero values mean "use default".

AI-Used: [claude]
```

---

### Task 5: Add adaptationConfigToAdaptConfig Helper and Wire into CLI (#402)

**Files:**
- Modify: `internal/cli/cli.go` (runAdaptationAnalysis)
- Add test to: `internal/cli/measurement_helpers_test.go`

Note: `ToAdaptConfig` lives in the CLI layer (not policy package) to avoid a circular dependency (`adapt` imports `policy`, so `policy` cannot import `adapt`).

- [ ] **Step 1: Write failing test**

Add to `internal/cli/measurement_helpers_test.go`:

```go
func TestAdaptationConfigToAdaptConfig_PartialOverride(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	defaults := adapt.Config{
		MinClusterSize:         5,
		MinFeedbackEvents:      3,
		MeasurementWindow:      10,
		MaintenanceMinOutcomes: 3,
		MaintenanceMinSuccess:  0.4,
		MinNewFeedback:         5,
	}

	ac := policy.AdaptationConfig{
		MinClusterSize:    7,
		MeasurementWindow: 15,
	}

	result := adaptationConfigToAdaptConfig(ac, defaults)

	g.Expect(result.MinClusterSize).To(Equal(7))
	g.Expect(result.MinFeedbackEvents).To(Equal(3))
	g.Expect(result.MeasurementWindow).To(Equal(15))
	g.Expect(result.MaintenanceMinOutcomes).To(Equal(3))
	g.Expect(result.MaintenanceMinSuccess).To(BeNumerically("~", 0.4, 0.001))
	g.Expect(result.MinNewFeedback).To(Equal(5))
}

func TestAdaptationConfigToAdaptConfig_AllZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	defaults := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}
	ac := policy.AdaptationConfig{}

	result := adaptationConfigToAdaptConfig(ac, defaults)

	g.Expect(result.MinClusterSize).To(Equal(5))
	g.Expect(result.MinFeedbackEvents).To(Equal(3))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `adaptationConfigToAdaptConfig` undefined

- [ ] **Step 3: Implement the helper in cli.go**

Add to `internal/cli/cli.go`:

```go
// adaptationConfigToAdaptConfig applies non-zero overrides from AdaptationConfig on top of defaults.
func adaptationConfigToAdaptConfig(ac policy.AdaptationConfig, defaults adapt.Config) adapt.Config {
	result := defaults

	if ac.MinClusterSize != 0 {
		result.MinClusterSize = ac.MinClusterSize
	}

	if ac.MinFeedbackEvents != 0 {
		result.MinFeedbackEvents = ac.MinFeedbackEvents
	}

	if ac.MeasurementWindow != 0 {
		result.MeasurementWindow = ac.MeasurementWindow
	}

	if ac.MaintenanceMinOutcomes != 0 {
		result.MaintenanceMinOutcomes = ac.MaintenanceMinOutcomes
	}

	if ac.MaintenanceMinSuccess != 0 {
		result.MaintenanceMinSuccess = ac.MaintenanceMinSuccess
	}

	if ac.MinNewFeedback != 0 {
		result.MinNewFeedback = ac.MinNewFeedback
	}

	return result
}
```

- [ ] **Step 4: Wire into runAdaptationAnalysis**

In `runAdaptationAnalysis`, after loading `adaptPF`, replace the hardcoded config construction with:

```go
const (
	defaultMinClusterSize         = 5
	defaultMinFeedbackEvents      = 3
	defaultMeasurementWindow      = 10
	defaultMaintenanceMinOutcomes = 3
	defaultMinNewFeedback         = 5
)

const defaultMaintenanceMinSuccess = 0.4

defaultConfig := adapt.Config{
	MinClusterSize:         defaultMinClusterSize,
	MinFeedbackEvents:      defaultMinFeedbackEvents,
	MeasurementWindow:      defaultMeasurementWindow,
	MaintenanceMinOutcomes: defaultMaintenanceMinOutcomes,
	MaintenanceMinSuccess:  defaultMaintenanceMinSuccess,
	MinNewFeedback:         defaultMinNewFeedback,
}

analysisConfig := adaptationConfigToAdaptConfig(adaptPF.Adaptation, defaultConfig)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full check**

Run: `targ check-full`
Expected: Only check-uncommitted fails

- [ ] **Step 7: Commit**

```
feat: wire adaptation config from policy.toml into analysis pipeline (#402)

adaptationConfigToAdaptConfig applies non-zero overrides on top of
defaults. runAdaptationAnalysis reads from policy.toml [adaptation]
section with fallback to hardcoded defaults.

AI-Used: [claude]
```

---

### Task 6: Final Integration Smoke Test

**Files:**
- No new files

- [ ] **Step 1: Run full check**

Run: `targ check-full`
Expected: ALL PASS

- [ ] **Step 2: Build binary**

Run: `go build -o /tmp/engram-test ./cmd/engram/`
Expected: Builds successfully

- [ ] **Step 3: Smoke test adapt command**

Run: `/tmp/engram-test adapt --status`
Expected: Shows policies or "No policies" without error
