# Adaptive Policy System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the feedback loop across extraction, surfacing, and maintenance by introducing a policy system that learns from feedback patterns and adapts behavior with user approval.

**Architecture:** New `internal/policy` and `internal/adapt` packages. Policy types are stored in `policy.toml` alongside existing data. Analysis runs at extract time (session end), proposals surface at triage time (session start). Extraction prompt becomes dynamically composed. Frecency scorer accepts policy overrides via options. Maintenance thresholds become policy-adjustable.

**Tech Stack:** Go 1.25, BurntSushi/toml, Gomega tests, targ CLI

**Spec:** `docs/plan/2026-03-27-adaptive-policy-system.md`

---

### Task 1: Fix effectiveness denominator bug

Include `IrrelevantCount` in the effectiveness denominator. Without this, every downstream adaptation is miscalibrated.

**Files:**
- Modify: `internal/effectiveness/aggregate.go:23`
- Modify: `internal/effectiveness/aggregate_test.go`
- Modify: `internal/frecency/frecency.go:64-70`
- Modify: `internal/frecency/frecency_test.go`

- [ ] **Step 1: Write failing test for effectiveness including IrrelevantCount**

In `internal/effectiveness/aggregate_test.go`, add:

```go
func TestFromMemories_IncludesIrrelevantInDenominator(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "/data/memories/mem-irr.toml",
			FollowedCount:   5,
			IrrelevantCount: 5,
		},
	}

	stats := effectiveness.FromMemories(memories)
	stat := stats["/data/memories/mem-irr.toml"]
	// 5 / (5 + 0 + 0 + 5) = 50%, not 100%
	g.Expect(stat.EffectivenessScore).To(gomega.BeNumerically("~", 50.0, 0.001))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestFromMemories_IncludesIrrelevantInDenominator ./internal/effectiveness/`
Expected: FAIL — currently returns 100.0 because IrrelevantCount is excluded

- [ ] **Step 3: Fix effectiveness denominator**

In `internal/effectiveness/aggregate.go`, change line 23:

```go
// Before:
total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount
// After:
total := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
```

Update the `Stat` struct comment on line 12:

```go
EffectivenessScore float64 // followed / (followed + contradicted + ignored + irrelevant) * 100
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestFromMemories_IncludesIrrelevantInDenominator ./internal/effectiveness/`
Expected: PASS

- [ ] **Step 5: Write failing test for frecency effectiveness including IrrelevantCount**

In `internal/frecency/frecency_test.go`, add:

```go
func TestEffectiveness_IncludesIrrelevantCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 100)

	input := frecency.Input{
		FollowedCount:   5,
		IrrelevantCount: 5,
	}

	quality := scorer.Quality(input)
	// effectiveness = 5/(5+0+0+5) = 0.5, not 1.0
	// quality = 0.3*0.5 + 1.0*0 + 0.3*0 = 0.15
	g.Expect(quality).To(BeNumerically("~", 0.15, 0.01))
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `targ test -- -run TestEffectiveness_IncludesIrrelevantCount ./internal/frecency/`
Expected: FAIL — Input struct doesn't have IrrelevantCount field yet

- [ ] **Step 7: Add IrrelevantCount to frecency Input and effectiveness calculation**

In `internal/frecency/frecency.go`, add to the `Input` struct (after `IgnoredCount`):

```go
IrrelevantCount int
```

Update the `effectiveness` method:

```go
func (s *Scorer) effectiveness(input Input) float64 {
	total := input.FollowedCount + input.ContradictedCount + input.IgnoredCount + input.IrrelevantCount
	if total == 0 {
		return defaultEffectiveness
	}

	return float64(input.FollowedCount) / float64(total)
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `targ test -- -run TestEffectiveness_IncludesIrrelevantCount ./internal/frecency/`
Expected: PASS

- [ ] **Step 9: Update frecency Input construction in surface.go**

In `internal/surface/surface.go`, find where `frecency.Input` is constructed and add `IrrelevantCount` from the memory's tracking data. Search for `frecency.Input{` — it should be in the `sortPromptMatchesByActivation` function or similar. Add `IrrelevantCount: mem.IrrelevantCount` to the struct literal.

- [ ] **Step 10: Run all tests**

Run: `targ check-full`
Expected: All tests pass, no lint errors

- [ ] **Step 11: Commit**

```bash
git add internal/effectiveness/aggregate.go internal/effectiveness/aggregate_test.go \
       internal/frecency/frecency.go internal/frecency/frecency_test.go \
       internal/surface/surface.go
git commit -m "fix: include IrrelevantCount in effectiveness denominator (#387)

Previously IrrelevantCount was excluded from the effectiveness score
denominator, making memories with high irrelevance appear more effective
than they are. This is a prerequisite for adaptive policy calibration.

AI-Used: [claude]"
```

---

### Task 2: Policy types and TOML persistence

Create the `internal/policy` package with types for policies, the policy file, and load/save functions.

**Files:**
- Create: `internal/policy/policy.go`
- Create: `internal/policy/policy_test.go`

- [ ] **Step 1: Write failing test for Policy types and round-trip TOML**

Create `internal/policy/policy_test.go`:

```go
package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/policy"
)

func TestRoundTrip_SaveAndLoad(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Directive: "Increase effectiveness weight to 0.5",
				Rationale: "High-effectiveness memories followed 3x more",
				Evidence:  policy.Evidence{FollowRate: 0.9, SampleSize: 45},
				Status:    policy.StatusProposed,
				CreatedAt: "2026-03-27T10:00:00Z",
			},
		},
		ApprovalStreak: policy.ApprovalStreak{Surfacing: 2},
	}

	err := policy.Save(path, pf)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, err := policy.Load(path)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(loaded.Policies).To(HaveLen(1))
	g.Expect(loaded.Policies[0].ID).To(Equal("pol-001"))
	g.Expect(loaded.Policies[0].Dimension).To(Equal(policy.DimensionSurfacing))
	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusProposed))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(2))
}

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf, err := policy.Load("/nonexistent/policy.toml")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(pf.Policies).To(BeEmpty())
}

func TestActivePolicies_FiltersByStatus(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusActive, Dimension: policy.DimensionSurfacing},
			{ID: "pol-002", Status: policy.StatusProposed, Dimension: policy.DimensionSurfacing},
			{ID: "pol-003", Status: policy.StatusActive, Dimension: policy.DimensionExtraction},
			{ID: "pol-004", Status: policy.StatusRetired, Dimension: policy.DimensionSurfacing},
		},
	}

	active := pf.Active(policy.DimensionSurfacing)
	g.Expect(active).To(HaveLen(1))
	g.Expect(active[0].ID).To(Equal("pol-001"))
}

func TestPendingPolicies_FiltersByProposed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusActive},
			{ID: "pol-002", Status: policy.StatusProposed},
			{ID: "pol-003", Status: policy.StatusProposed},
		},
	}

	pending := pf.Pending()
	g.Expect(pending).To(HaveLen(2))
}

func TestApprove_TransitionsAndUpdatesStreak(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusProposed, Dimension: policy.DimensionSurfacing},
		},
	}

	err := pf.Approve("pol-001", "2026-03-27T12:00:00Z")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pf.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(pf.Policies[0].ApprovedAt).To(Equal("2026-03-27T12:00:00Z"))
	g.Expect(pf.ApprovalStreak.Surfacing).To(Equal(1))
}

func TestReject_TransitionsAndResetsStreak(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusProposed, Dimension: policy.DimensionSurfacing},
		},
		ApprovalStreak: policy.ApprovalStreak{Surfacing: 3},
	}

	err := pf.Reject("pol-001")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pf.Policies[0].Status).To(Equal(policy.StatusRejected))
	g.Expect(pf.ApprovalStreak.Surfacing).To(Equal(0))
}

func TestNextID_Sequential(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001"},
			{ID: "pol-003"},
		},
	}

	g.Expect(pf.NextID()).To(Equal("pol-004"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRoundTrip ./internal/policy/`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement policy package**

Create `internal/policy/policy.go`:

```go
// Package policy manages adaptive policy storage and lifecycle.
package policy

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Dimension identifies which subsystem a policy targets.
type Dimension string

// Status tracks the lifecycle state of a policy.
type Status string

// Dimension constants.
const (
	DimensionExtraction  Dimension = "extraction"
	DimensionSurfacing   Dimension = "surfacing"
	DimensionMaintenance Dimension = "maintenance"
)

// Status constants.
const (
	StatusProposed Status = "proposed"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusActive   Status = "active"
	StatusRetired  Status = "retired"
)

// Exported errors.
var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrInvalidStatus  = errors.New("policy is not in proposed status")
)

// Evidence holds the statistical basis for a policy proposal.
type Evidence struct {
	IrrelevantRate   float64 `toml:"irrelevant_rate,omitempty"`
	FollowRate       float64 `toml:"follow_rate,omitempty"`
	Correlation      float64 `toml:"correlation,omitempty"`
	SampleSize       int     `toml:"sample_size"`
	SessionsObserved int     `toml:"sessions_observed,omitempty"`
}

// Effectiveness tracks before/after measurement for a policy.
type Effectiveness struct {
	Before           float64 `toml:"before"`
	After            float64 `toml:"after"`
	MeasuredSessions int     `toml:"measured_sessions"`
}

// Policy represents a single learned adaptation directive.
//
//nolint:tagliatelle // TOML field names use snake_case by convention.
type Policy struct {
	ID            string        `toml:"id"`
	Dimension     Dimension     `toml:"dimension"`
	Directive     string        `toml:"directive"`
	Rationale     string        `toml:"rationale"`
	Evidence      Evidence      `toml:"evidence"`
	Status        Status        `toml:"status"`
	CreatedAt     string        `toml:"created_at"`
	ApprovedAt    string        `toml:"approved_at,omitempty"`
	Effectiveness Effectiveness `toml:"effectiveness"`
	Parameter     string        `toml:"parameter,omitempty"`
	Value         float64       `toml:"value,omitempty"`
}

// ApprovalStreak tracks consecutive approvals per dimension.
type ApprovalStreak struct {
	Extraction  int `toml:"extraction"`
	Surfacing   int `toml:"surfacing"`
	Maintenance int `toml:"maintenance"`
}

// File represents the policy.toml file.
type File struct {
	Policies       []Policy       `toml:"policies"`
	ApprovalStreak ApprovalStreak `toml:"approval_streak"`
}

// Load reads a policy file from disk. Returns an empty File if the file does not exist.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{}, nil
		}

		return nil, fmt.Errorf("loading policy file: %w", err)
	}

	var pf File

	err = toml.Unmarshal(data, &pf)
	if err != nil {
		return nil, fmt.Errorf("parsing policy file: %w", err)
	}

	return &pf, nil
}

// Save writes the policy file to disk.
func Save(path string, pf *File) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating policy file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := toml.NewEncoder(f)

	err = enc.Encode(pf)
	if err != nil {
		return fmt.Errorf("encoding policy file: %w", err)
	}

	return nil
}

// Active returns all policies with status "active" for the given dimension.
func (pf *File) Active(dim Dimension) []Policy {
	result := make([]Policy, 0)

	for _, p := range pf.Policies {
		if p.Status == StatusActive && p.Dimension == dim {
			result = append(result, p)
		}
	}

	return result
}

// Pending returns all policies with status "proposed".
func (pf *File) Pending() []Policy {
	result := make([]Policy, 0)

	for _, p := range pf.Policies {
		if p.Status == StatusProposed {
			result = append(result, p)
		}
	}

	return result
}

// Approve transitions a proposed policy to active and increments the approval streak.
func (pf *File) Approve(id, timestamp string) error {
	for i := range pf.Policies {
		if pf.Policies[i].ID == id {
			if pf.Policies[i].Status != StatusProposed {
				return fmt.Errorf("%w: %s is %s", ErrInvalidStatus, id, pf.Policies[i].Status)
			}

			pf.Policies[i].Status = StatusActive
			pf.Policies[i].ApprovedAt = timestamp
			pf.incrementStreak(pf.Policies[i].Dimension)

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrPolicyNotFound, id)
}

// Reject transitions a proposed policy to rejected and resets the streak for its dimension.
func (pf *File) Reject(id string) error {
	for i := range pf.Policies {
		if pf.Policies[i].ID == id {
			if pf.Policies[i].Status != StatusProposed {
				return fmt.Errorf("%w: %s is %s", ErrInvalidStatus, id, pf.Policies[i].Status)
			}

			pf.Policies[i].Status = StatusRejected
			pf.resetStreak(pf.Policies[i].Dimension)

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrPolicyNotFound, id)
}

// Retire transitions an active policy to retired.
func (pf *File) Retire(id string) error {
	for i := range pf.Policies {
		if pf.Policies[i].ID == id {
			pf.Policies[i].Status = StatusRetired

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrPolicyNotFound, id)
}

// NextID returns the next sequential policy ID (e.g., "pol-004").
func (pf *File) NextID() string {
	const idPrefix = "pol-"

	maxNum := 0

	for _, p := range pf.Policies {
		numStr := strings.TrimPrefix(p.ID, idPrefix)

		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}

		if num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("%s%03d", idPrefix, maxNum+1)
}

func (pf *File) incrementStreak(dim Dimension) {
	switch dim {
	case DimensionExtraction:
		pf.ApprovalStreak.Extraction++
	case DimensionSurfacing:
		pf.ApprovalStreak.Surfacing++
	case DimensionMaintenance:
		pf.ApprovalStreak.Maintenance++
	}
}

func (pf *File) resetStreak(dim Dimension) {
	switch dim {
	case DimensionExtraction:
		pf.ApprovalStreak.Extraction = 0
	case DimensionSurfacing:
		pf.ApprovalStreak.Surfacing = 0
	case DimensionMaintenance:
		pf.ApprovalStreak.Maintenance = 0
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/policy/`
Expected: All tests pass

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/policy/
git commit -m "feat: add policy package for adaptive policy storage (#387)

Introduces Policy, File, Evidence, Effectiveness types with TOML
persistence. Supports lifecycle transitions (propose, approve, reject,
retire) and approval streak tracking per dimension.

AI-Used: [claude]"
```

---

### Task 3: Maintenance history on memory records

Add `MaintenanceHistory` to `MemoryRecord` so maintenance action outcomes can be tracked.

**Files:**
- Modify: `internal/memory/record.go`
- Create: `internal/memory/maintenance_test.go`

- [ ] **Step 1: Write failing test for MaintenanceHistory round-trip**

Create `internal/memory/maintenance_test.go`:

```go
package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_MaintenanceHistory_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rec := memory.MemoryRecord{
		Title:     "test memory",
		Content:   "test content",
		CreatedAt: "2026-03-27T10:00:00Z",
		UpdatedAt: "2026-03-27T10:00:00Z",
		MaintenanceHistory: []memory.MaintenanceAction{
			{
				Action:                "rewrite",
				AppliedAt:             "2026-03-20T10:00:00Z",
				EffectivenessBefore:   25.0,
				SurfacedCountBefore:   12,
				EffectivenessAfter:    0.0,
				SurfacedCountAfter:    0,
				Measured:              false,
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	f, err := os.Create(path)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	err = toml.NewEncoder(f).Encode(rec)
	_ = f.Close()
	g.Expect(err).NotTo(HaveOccurred())

	var loaded memory.MemoryRecord
	_, err = toml.DecodeFile(path, &loaded)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory).To(HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].Action).To(Equal("rewrite"))
	g.Expect(loaded.MaintenanceHistory[0].EffectivenessBefore).To(BeNumerically("~", 25.0, 0.001))
	g.Expect(loaded.MaintenanceHistory[0].Measured).To(BeFalse())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestMemoryRecord_MaintenanceHistory ./internal/memory/`
Expected: FAIL — MaintenanceAction type and MaintenanceHistory field don't exist

- [ ] **Step 3: Add MaintenanceAction and MaintenanceHistory to record.go**

In `internal/memory/record.go`, add before `MemoryRecord`:

```go
// MaintenanceAction records a single maintenance action applied to this memory
// and its before/after effectiveness for outcome tracking (#387).
type MaintenanceAction struct {
	Action              string  `toml:"action"`
	AppliedAt           string  `toml:"applied_at"`
	EffectivenessBefore float64 `toml:"effectiveness_before"`
	SurfacedCountBefore int     `toml:"surfaced_count_before"`
	EffectivenessAfter  float64 `toml:"effectiveness_after"`
	SurfacedCountAfter  int     `toml:"surfaced_count_after"`
	Measured            bool    `toml:"measured"`
}
```

Add to `MemoryRecord` struct, after the `Absorbed` field:

```go
	// Maintenance history — tracks action outcomes for adaptive policy (#387).
	MaintenanceHistory []MaintenanceAction `toml:"maintenance_history,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestMemoryRecord_MaintenanceHistory ./internal/memory/`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/memory/record.go internal/memory/maintenance_test.go
git commit -m "feat: add MaintenanceHistory to MemoryRecord (#387)

Tracks before/after effectiveness for each maintenance action applied
to a memory, enabling outcome measurement for adaptive policies.

AI-Used: [claude]"
```

---

### Task 4: Extraction adaptation — dynamic prompt composition

Make `systemPrompt()` accept active extraction policies and inject dynamic guidance sections.

**Files:**
- Modify: `internal/extract/extract.go`
- Modify: `internal/extract/extract_test.go`

- [ ] **Step 1: Write failing test for dynamic prompt composition**

In `internal/extract/extract_test.go`, add (or create if it doesn't exist):

```go
func TestSystemPrompt_WithPolicies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	policies := []extract.ExtractionGuidance{
		{
			Directive: "De-prioritize tool-specific mechanical patterns",
			Rationale: "80% irrelevance rate across 15 memories",
		},
		{
			Directive: "Prioritize design rationale",
			Rationale: "90% follow rate across 12 memories",
		},
	}

	prompt := extract.SystemPromptWithGuidance(policies)
	g.Expect(prompt).To(ContainSubstring("De-prioritize tool-specific"))
	g.Expect(prompt).To(ContainSubstring("Learned Extraction Guidance"))
	g.Expect(prompt).To(ContainSubstring("80% irrelevance rate"))
}

func TestSystemPrompt_NoPolicies_MatchesOriginal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	prompt := extract.SystemPromptWithGuidance(nil)
	original := extract.SystemPromptWithGuidance([]extract.ExtractionGuidance{})
	g.Expect(prompt).To(Equal(original))
	// Should not contain the guidance header
	g.Expect(prompt).NotTo(ContainSubstring("Learned Extraction Guidance"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestSystemPrompt_WithPolicies ./internal/extract/`
Expected: FAIL — ExtractionGuidance type and SystemPromptWithGuidance don't exist

- [ ] **Step 3: Implement dynamic prompt composition**

In `internal/extract/extract.go`, add:

```go
// ExtractionGuidance is a learned directive to inject into the extraction prompt.
type ExtractionGuidance struct {
	Directive string
	Rationale string
}

// SystemPromptWithGuidance returns the system prompt with optional learned guidance sections.
func SystemPromptWithGuidance(guidance []ExtractionGuidance) string {
	base := strings.TrimSpace(extractionSystemPrompt)
	if len(guidance) == 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Learned Extraction Guidance\n\n")
	sb.WriteString("Based on feedback from this user's memory corpus:\n\n")

	for _, g := range guidance {
		sb.WriteString("- ")
		sb.WriteString(g.Directive)
		sb.WriteString(" (")
		sb.WriteString(g.Rationale)
		sb.WriteString(")\n")
	}

	return sb.String()
}
```

Update `systemPrompt()` to call `SystemPromptWithGuidance(nil)`:

```go
func systemPrompt() string {
	return SystemPromptWithGuidance(nil)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/extract/`
Expected: PASS

- [ ] **Step 5: Wire guidance into LLMExtractor**

Add a `guidance` field to `LLMExtractor`:

```go
type LLMExtractor struct {
	token    string
	client   HTTPDoer
	guidance []ExtractionGuidance
}
```

Add an option function:

```go
// Option configures an LLMExtractor.
type Option func(*LLMExtractor)

// WithGuidance sets learned extraction guidance policies.
func WithGuidance(guidance []ExtractionGuidance) Option {
	return func(e *LLMExtractor) {
		e.guidance = guidance
	}
}
```

Update `New` to accept options:

```go
func New(token string, client HTTPDoer, opts ...Option) *LLMExtractor {
	e := &LLMExtractor{
		token:  token,
		client: client,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
```

Update `sendRequest` to use `SystemPromptWithGuidance(e.guidance)` instead of `systemPrompt()`.

- [ ] **Step 6: Fix all callers of extract.New**

Search for `extract.New(` across the codebase — existing callers pass `(token, client)` which still works with variadic options. Verify no compilation errors.

- [ ] **Step 7: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 8: Commit**

```bash
git add internal/extract/extract.go internal/extract/extract_test.go
git commit -m "feat: add dynamic extraction guidance from policies (#387)

SystemPromptWithGuidance composes the extraction prompt from a static
base plus learned guidance directives. LLMExtractor accepts WithGuidance
option to inject active extraction policies.

AI-Used: [claude]"
```

---

### Task 5: Surfacing adaptation — frecency policy overrides

Allow the frecency scorer to accept weight overrides from active surfacing policies.

**Files:**
- Modify: `internal/frecency/frecency.go`
- Modify: `internal/frecency/frecency_test.go`

- [ ] **Step 1: Write failing test for weight overrides**

In `internal/frecency/frecency_test.go`, add:

```go
func TestQuality_WithWeightOverrides(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	const maxSurfaced = 100
	scorer := frecency.New(now, maxSurfaced,
		frecency.WithWEff(0.5),
		frecency.WithWFreq(0.7),
	)

	input := frecency.Input{
		FollowedCount: 8,
		IgnoredCount:  2,
		SurfacedCount: 50,
	}

	quality := scorer.Quality(input)
	// effectiveness = 8/10 = 0.8
	// frequency = log(51)/log(101) ≈ 0.851
	// tierBoost = 0
	// quality = 0.5*0.8 + 0.7*0.851 + 0.3*0 = 0.4 + 0.596 = 0.996
	g.Expect(quality).To(BeNumerically("~", 0.996, 0.02))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestQuality_WithWeightOverrides ./internal/frecency/`
Expected: FAIL — WithWEff and WithWFreq options don't exist

- [ ] **Step 3: Add weight override options**

In `internal/frecency/frecency.go`, add:

```go
// WithWEff overrides the effectiveness weight.
func WithWEff(w float64) Option {
	return func(s *Scorer) { s.wEff = w }
}

// WithWFreq overrides the frequency weight.
func WithWFreq(w float64) Option {
	return func(s *Scorer) { s.wFreq = w }
}

// WithWTier overrides the tier boost weight.
func WithWTier(w float64) Option {
	return func(s *Scorer) { s.wTier = w }
}

// WithTierABoost overrides the Tier A boost multiplier.
func WithTierABoost(b float64) Option {
	return func(s *Scorer) { s.tierABoost = b }
}

// WithTierBBoost overrides the Tier B boost multiplier.
func WithTierBBoost(b float64) Option {
	return func(s *Scorer) { s.tierBBoost = b }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/frecency/`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/frecency/frecency.go internal/frecency/frecency_test.go
git commit -m "feat: add weight override options to frecency scorer (#387)

WithWEff, WithWFreq, WithWTier, WithTierABoost, WithTierBBoost allow
active surfacing policies to override default scoring weights.

AI-Used: [claude]"
```

---

### Task 6: Feedback pattern analysis engine

Create the `internal/adapt` package that analyzes feedback patterns and generates policy proposals.

**Files:**
- Create: `internal/adapt/analyze.go`
- Create: `internal/adapt/analyze_test.go`

- [ ] **Step 1: Write failing tests for content pattern analysis**

Create `internal/adapt/analyze_test.go`:

```go
package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAnalyze_ContentPattern_HighIrrelevanceCluster(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"build-tool", "targ"}, IrrelevantCount: 8, FollowedCount: 1, SurfacedCount: 10},
		{FilePath: "b.toml", Keywords: []string{"build-tool", "compilation"}, IrrelevantCount: 7, FollowedCount: 1, SurfacedCount: 9},
		{FilePath: "c.toml", Keywords: []string{"build-tool", "lint"}, IrrelevantCount: 6, FollowedCount: 2, SurfacedCount: 9},
		{FilePath: "d.toml", Keywords: []string{"build-tool", "test-runner"}, IrrelevantCount: 9, FollowedCount: 0, SurfacedCount: 10},
		{FilePath: "e.toml", Keywords: []string{"build-tool", "coverage"}, IrrelevantCount: 5, FollowedCount: 2, SurfacedCount: 8},
	}

	config := adapt.Config{
		MinClusterSize:    5,
		MinFeedbackEvents: 3,
	}

	proposals := adapt.AnalyzeContentPatterns(memories, config)
	g.Expect(proposals).NotTo(BeEmpty())
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionExtraction))
	g.Expect(proposals[0].Directive).To(ContainSubstring("build-tool"))
}

func TestAnalyze_ContentPattern_BelowMinCluster(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"build-tool"}, IrrelevantCount: 8, SurfacedCount: 10},
		{FilePath: "b.toml", Keywords: []string{"build-tool"}, IrrelevantCount: 7, SurfacedCount: 9},
	}

	config := adapt.Config{MinClusterSize: 5, MinFeedbackEvents: 3}

	proposals := adapt.AnalyzeContentPatterns(memories, config)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyze_SurfacingPattern_TierBoostMismatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "a.toml", Tier: "A", FollowedCount: 2, IgnoredCount: 8, SurfacedCount: 10},
		{FilePath: "b.toml", Tier: "A", FollowedCount: 1, IgnoredCount: 7, SurfacedCount: 10},
		{FilePath: "c.toml", Tier: "A", FollowedCount: 3, IgnoredCount: 7, SurfacedCount: 10},
		{FilePath: "d.toml", Tier: "B", FollowedCount: 8, IgnoredCount: 2, SurfacedCount: 10},
		{FilePath: "e.toml", Tier: "B", FollowedCount: 7, IgnoredCount: 1, SurfacedCount: 10},
		{FilePath: "f.toml", Tier: "B", FollowedCount: 9, IgnoredCount: 1, SurfacedCount: 10},
	}

	config := adapt.Config{MinClusterSize: 3, MinFeedbackEvents: 3}

	proposals := adapt.AnalyzeStructuralPatterns(memories, config)
	// Tier B outperforms Tier A — should propose tier boost adjustment
	g.Expect(proposals).NotTo(BeEmpty())
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionSurfacing))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- ./internal/adapt/`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement analysis engine**

Create `internal/adapt/analyze.go`:

```go
// Package adapt analyzes feedback patterns and generates policy proposals.
package adapt

import (
	"fmt"
	"sort"
	"strings"

	"engram/internal/memory"
	"engram/internal/policy"
)

// Config holds thresholds for the analysis engine.
type Config struct {
	MinClusterSize    int
	MinFeedbackEvents int
}

const (
	highIrrelevanceThreshold = 0.6
	tierDifferenceThreshold  = 0.15
)

// AnalyzeContentPatterns clusters memories by shared keywords and identifies
// clusters with consistently high irrelevance or follow rates.
func AnalyzeContentPatterns(memories []*memory.Stored, cfg Config) []policy.Policy {
	clusters := clusterByKeywords(memories)
	proposals := make([]policy.Policy, 0)

	for keyword, mems := range clusters {
		if len(mems) < cfg.MinClusterSize {
			continue
		}

		var totalIrrelevant, totalFeedback int

		for _, mem := range mems {
			feedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
			if feedback < cfg.MinFeedbackEvents {
				continue
			}

			totalIrrelevant += mem.IrrelevantCount
			totalFeedback += feedback
		}

		if totalFeedback == 0 {
			continue
		}

		irrelevanceRate := float64(totalIrrelevant) / float64(totalFeedback)
		if irrelevanceRate >= highIrrelevanceThreshold {
			proposals = append(proposals, policy.Policy{
				Dimension: policy.DimensionExtraction,
				Directive: fmt.Sprintf(
					"De-prioritize memories about %q — consistently irrelevant",
					keyword,
				),
				Rationale: fmt.Sprintf(
					"%.0f%% irrelevance rate across %d memories",
					irrelevanceRate*100, len(mems), //nolint:mnd
				),
				Evidence: policy.Evidence{
					IrrelevantRate: irrelevanceRate,
					SampleSize:     len(mems),
				},
				Status: policy.StatusProposed,
			})
		}
	}

	return proposals
}

// AnalyzeStructuralPatterns compares effectiveness across tiers and generalizability levels.
func AnalyzeStructuralPatterns(memories []*memory.Stored, cfg Config) []policy.Policy {
	tierStats := make(map[string]struct{ followed, total int })

	for _, mem := range memories {
		if mem.Tier == "" {
			continue
		}

		feedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
		if feedback < cfg.MinFeedbackEvents {
			continue
		}

		stat := tierStats[mem.Tier]
		stat.followed += mem.FollowedCount
		stat.total += feedback
		tierStats[mem.Tier] = stat
	}

	proposals := make([]policy.Policy, 0)

	statA, hasA := tierStats["A"]
	statB, hasB := tierStats["B"]

	if hasA && hasB && statA.total >= cfg.MinClusterSize && statB.total >= cfg.MinClusterSize {
		effA := float64(statA.followed) / float64(statA.total)
		effB := float64(statB.followed) / float64(statB.total)

		if effB > effA+tierDifferenceThreshold {
			proposals = append(proposals, policy.Policy{
				Dimension: policy.DimensionSurfacing,
				Directive: fmt.Sprintf(
					"Tier B outperforms Tier A (%.0f%% vs %.0f%%) — reduce tierABoost or increase tierBBoost",
					effB*100, effA*100, //nolint:mnd
				),
				Rationale: fmt.Sprintf(
					"Tier B follow rate %.0f%% exceeds Tier A %.0f%% across %d+%d memories",
					effB*100, effA*100, statA.total, statB.total, //nolint:mnd
				),
				Evidence: policy.Evidence{
					FollowRate: effB,
					SampleSize: statA.total + statB.total,
				},
				Status: policy.StatusProposed,
			})
		}
	}

	return proposals
}

// clusterByKeywords groups memories by their most common shared keyword.
func clusterByKeywords(memories []*memory.Stored) map[string][]*memory.Stored {
	clusters := make(map[string][]*memory.Stored)

	for _, mem := range memories {
		for _, kw := range mem.Keywords {
			normalized := strings.ToLower(kw)
			clusters[normalized] = append(clusters[normalized], mem)
		}
	}

	return clusters
}

// AnalyzeAll runs all pattern analyses and returns combined proposals.
func AnalyzeAll(memories []*memory.Stored, cfg Config) []policy.Policy {
	proposals := make([]policy.Policy, 0)
	proposals = append(proposals, AnalyzeContentPatterns(memories, cfg)...)
	proposals = append(proposals, AnalyzeStructuralPatterns(memories, cfg)...)

	// Sort by evidence strength (sample size descending)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Evidence.SampleSize > proposals[j].Evidence.SampleSize
	})

	return proposals
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/adapt/`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/adapt/
git commit -m "feat: add feedback pattern analysis engine (#387)

AnalyzeContentPatterns clusters memories by keyword and identifies
high-irrelevance clusters. AnalyzeStructuralPatterns compares tier
effectiveness. Both generate policy proposals with evidence.

AI-Used: [claude]"
```

---

### Task 7: CLI adapt command

Add the `adapt` subcommand for listing, approving, and rejecting policy proposals.

**Files:**
- Create: `internal/cli/adapt.go`
- Create: `internal/cli/adapt_test.go`
- Modify: `internal/cli/cli.go` (add dispatch case)
- Modify: `internal/cli/targets.go` (add targ target)

- [ ] **Step 1: Write failing test for adapt --status**

Create `internal/cli/adapt_test.go`:

```go
package cli_test

import (
	"bytes"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/policy"
)

func TestRunAdapt_Status_ShowsPolicies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Directive: "Increase wEff to 0.5",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionExtraction,
				Directive: "De-prioritize tool patterns",
				Status:    policy.StatusActive,
			},
		},
	}

	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())

	var buf bytes.Buffer
	err = cli.RunAdapt([]string{"--data-dir", dir, "--status"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("pol-001"))
	g.Expect(output).To(ContainSubstring("proposed"))
	g.Expect(output).To(ContainSubstring("pol-002"))
	g.Expect(output).To(ContainSubstring("active"))
}

func TestRunAdapt_Approve(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
		},
	}
	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())

	var buf bytes.Buffer
	err = cli.RunAdapt([]string{"--data-dir", dir, "--approve", "pol-001"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, err := policy.Load(policyPath)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(1))
}

func TestRunAdapt_Reject(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
		},
		ApprovalStreak: policy.ApprovalStreak{Surfacing: 3},
	}
	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())

	var buf bytes.Buffer
	err = cli.RunAdapt([]string{"--data-dir", dir, "--reject", "pol-001"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, err := policy.Load(policyPath)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusRejected))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(0))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunAdapt ./internal/cli/`
Expected: FAIL — RunAdapt doesn't exist

- [ ] **Step 3: Implement RunAdapt**

Create `internal/cli/adapt.go`:

```go
package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"engram/internal/policy"
)

// AdaptArgs holds parsed flags for the adapt subcommand.
type AdaptArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Status  bool   `targ:"flag,name=status,desc=show all policies"`
	Approve string `targ:"flag,name=approve,desc=approve a policy by ID"`
	Reject  string `targ:"flag,name=reject,desc=reject a policy by ID"`
	Retire  string `targ:"flag,name=retire,desc=retire a policy by ID"`
}

// RunAdapt implements the adapt subcommand.
func RunAdapt(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("adapt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	status := fs.Bool("status", false, "show all policies")
	approve := fs.String("approve", "", "approve a policy by ID")
	reject := fs.String("reject", "", "reject a policy by ID")
	retire := fs.String("retire", "", "retire a policy by ID")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("adapt: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("adapt: %w", defaultErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pf, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	switch {
	case *approve != "":
		return adaptApprove(pf, policyPath, *approve, stdout)
	case *reject != "":
		return adaptReject(pf, policyPath, *reject, stdout)
	case *retire != "":
		return adaptRetire(pf, policyPath, *retire, stdout)
	default:
		*status = true
	}

	if *status {
		return adaptStatus(pf, stdout)
	}

	return nil
}

func adaptApprove(pf *policy.File, path, id string, stdout io.Writer) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	err := pf.Approve(id, timestamp)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, pf)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Approved policy %s\n", id)

	return nil
}

func adaptReject(pf *policy.File, path, id string, stdout io.Writer) error {
	err := pf.Reject(id)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, pf)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Rejected policy %s\n", id)

	return nil
}

func adaptRetire(pf *policy.File, path, id string, stdout io.Writer) error {
	err := pf.Retire(id)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, pf)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Retired policy %s\n", id)

	return nil
}

func adaptStatus(pf *policy.File, stdout io.Writer) error {
	if len(pf.Policies) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No policies.")

		return nil
	}

	for _, p := range pf.Policies {
		_, _ = fmt.Fprintf(stdout, "  %s [%s] %s — %s\n",
			p.ID, p.Dimension, string(p.Status), p.Directive)
	}

	return nil
}
```

- [ ] **Step 4: Wire adapt into CLI dispatch**

In `internal/cli/cli.go`, add a case in the `Run` switch:

```go
	case "adapt":
		return RunAdapt(subArgs, stdout)
```

In `internal/cli/targets.go`, add `AdaptFlags` and add to `BuildTargets`:

```go
// AdaptFlags returns the CLI flag args for the adapt subcommand.
func AdaptFlags(a AdaptArgs) []string {
	flags := BuildFlags("--data-dir", a.DataDir, "--approve", a.Approve, "--reject", a.Reject, "--retire", a.Retire)
	flags = AddBoolFlag(flags, "--status", a.Status)

	return flags
}
```

Add to the `BuildTargets` slice:

```go
		targ.Targ(func(a AdaptArgs) { run("adapt", AdaptFlags(a)) }).
			Name("adapt").Description("Manage adaptive policies"),
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test -- -run TestRunAdapt ./internal/cli/`
Expected: PASS

- [ ] **Step 6: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 7: Commit**

```bash
git add internal/cli/adapt.go internal/cli/adapt_test.go \
       internal/cli/cli.go internal/cli/targets.go
git commit -m "feat: add adapt CLI command for policy management (#387)

Supports --status, --approve, --reject, and --retire flags for
interactive policy lifecycle management.

AI-Used: [claude]"
```

---

### Task 8: Wire policy loading into surfacing pipeline

Connect active surfacing policies to the frecency scorer when surfacing memories.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go` (or create if needed)

- [ ] **Step 1: Write failing test for policy-aware surfacing**

In `internal/surface/surface_test.go`, add a test that verifies the surfacer accepts a policy path option and uses it to override frecency weights. The key assertion: with a policy that changes wEff to 1.0 and wFreq to 0.0, a memory with high effectiveness but low frequency should rank higher than one with low effectiveness but high frequency.

```go
func TestSurfacer_WithPolicyOverrides(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Create policy file with surfacing weight override
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")
	pf := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				Parameter: "wEff",
				Value:     1.0,
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				Parameter: "wFreq",
				Value:     0.0,
			},
		},
	}
	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())

	// ... (test that scorer uses overridden weights)
}
```

Note: The exact test shape depends on the existing surface test patterns. Follow the existing DI patterns — inject a mock retriever that returns memories with known effectiveness/frequency values, then verify the ranking order changes with vs without policy overrides.

- [ ] **Step 2: Add PolicyPath option to Surfacer**

Add a `WithPolicyPath` option to the Surfacer. In the `Run` method, load active surfacing policies and pass them as frecency options when creating the scorer:

```go
// In surface.go, in the Run method, replace:
scorer := frecency.New(time.Now(), maxSurfaced)

// With:
var frecencyOpts []frecency.Option
if s.policyPath != "" {
    pf, policyErr := policy.Load(s.policyPath)
    if policyErr == nil {
        frecencyOpts = surfacingPolicyToFrecencyOpts(pf)
    }
}
scorer := frecency.New(time.Now(), maxSurfaced, frecencyOpts...)
```

Add the helper:

```go
func surfacingPolicyToFrecencyOpts(pf *policy.File) []frecency.Option {
    opts := make([]frecency.Option, 0)
    for _, p := range pf.Active(policy.DimensionSurfacing) {
        switch p.Parameter {
        case "wEff":
            opts = append(opts, frecency.WithWEff(p.Value))
        case "wFreq":
            opts = append(opts, frecency.WithWFreq(p.Value))
        case "wTier":
            opts = append(opts, frecency.WithWTier(p.Value))
        case "tierABoost":
            opts = append(opts, frecency.WithTierABoost(p.Value))
        case "tierBBoost":
            opts = append(opts, frecency.WithTierBBoost(p.Value))
        }
    }
    return opts
}

// For irrelevancePenaltyHalfLife and coldStartBudget overrides:
func surfacingPolicyOverrides(pf *policy.File) (halfLife int, coldStart int, hasHL, hasCS bool) {
    for _, p := range pf.Active(policy.DimensionSurfacing) {
        switch p.Parameter {
        case "irrelevancePenaltyHalfLife":
            halfLife = int(p.Value)
            hasHL = true
        case "coldStartBudget":
            coldStart = int(p.Value)
            hasCS = true
        }
    }
    return
}
```

- [ ] **Step 3: Wire PolicyPath in CLI**

In the CLI's `runSurface` function, pass the policy path to the surfacer via the new option. The policy file lives at `filepath.Join(dataDir, "policy.toml")`.

- [ ] **Step 4: Run tests**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go \
       internal/cli/cli.go
git commit -m "feat: wire surfacing policy overrides into frecency scorer (#387)

Active surfacing policies (wEff, wFreq, wTier, tierABoost, tierBBoost)
are read from policy.toml and passed as frecency.Option overrides.

AI-Used: [claude]"
```

---

### Task 9: Wire policy loading into extraction pipeline

Connect active extraction policies to the extraction prompt when extracting memories.

**Files:**
- Modify: `internal/cli/cli.go` (in RunLearn / extraction wiring)

- [ ] **Step 1: Load active extraction policies in RunLearn**

In `internal/cli/cli.go`, in the `RunLearn` function (or wherever the extractor is created), load the policy file and convert active extraction policies to `extract.ExtractionGuidance`:

```go
// After creating the extractor, before calling extract:
policyPath := filepath.Join(*dataDir, "policy.toml")
pf, policyErr := policy.Load(policyPath)
var guidance []extract.ExtractionGuidance
if policyErr == nil {
    for _, p := range pf.Active(policy.DimensionExtraction) {
        guidance = append(guidance, extract.ExtractionGuidance{
            Directive: p.Directive,
            Rationale: p.Rationale,
        })
    }
}

extractor := extract.New(token, httpClient, extract.WithGuidance(guidance))
```

- [ ] **Step 2: Run tests**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 3: Commit**

```bash
git add internal/cli/cli.go
git commit -m "feat: wire extraction policies into LLM prompt (#387)

Active extraction policies are loaded from policy.toml and passed as
ExtractionGuidance to compose the dynamic extraction prompt.

AI-Used: [claude]"
```

---

### Task 10: Wire analysis into extract pipeline and proposals into triage

Run feedback analysis at extract time (session end) and surface proposals at triage time (session start).

**Files:**
- Modify: `internal/cli/cli.go` (add analysis after extraction)
- Modify: `hooks/session-start.sh` (add adaptation proposals to triage output)

- [ ] **Step 1: Run analysis after extraction in RunLearn**

In `internal/cli/cli.go`, after the learn/extract step completes successfully, add analysis:

```go
// After extraction is done, run feedback analysis
analysisConfig := adapt.Config{
    MinClusterSize:    5,
    MinFeedbackEvents: 3,
}
allMemories, listErr := retrieve.New().ListMemories(ctx, *dataDir)
if listErr == nil && len(allMemories) > 0 {
    policyPath := filepath.Join(*dataDir, "policy.toml")
    pf, _ := policy.Load(policyPath)

    newProposals := adapt.AnalyzeAll(allMemories, analysisConfig)
    if len(newProposals) > 0 {
        for i := range newProposals {
            newProposals[i].ID = pf.NextID()
            newProposals[i].CreatedAt = time.Now().UTC().Format(time.RFC3339)
            pf.Policies = append(pf.Policies, newProposals[i])
        }
        _ = policy.Save(policyPath, pf)
    }
}
```

Note: This is fire-and-forget per ARCH-6. Errors don't fail extraction.

- [ ] **Step 2: Add adaptation proposals to session-start.sh triage output**

In `hooks/session-start.sh`, after the existing maintain proposal parsing, add a section that reads `policy.toml` for pending proposals:

```bash
    # Adaptation proposals from policy.toml
    POLICY_FILE="${ENGRAM_HOME}/data/policy.toml"
    ADAPT_COUNT=0
    ADAPT_DETAIL=""
    if [[ -f "$POLICY_FILE" ]]; then
        # Count proposed policies (simple grep — policy.toml is small)
        ADAPT_COUNT=$(grep -c 'status = "proposed"' "$POLICY_FILE" 2>/dev/null) || ADAPT_COUNT=0

        if [[ "$ADAPT_COUNT" -gt 0 ]]; then
            ADAPT_DETAIL="## Adaptation Proposals (${ADAPT_COUNT} pending)
Feedback patterns suggest system improvements.
$(grep -B3 'status = "proposed"' "$POLICY_FILE" | grep 'directive' | sed 's/.*= "//;s/"$//' | nl -ba | sed 's/^/  /')

Run /adapt to review proposals or adjust adaptation settings."
        fi
    fi
```

Add `ADAPT_DETAIL` to the `TRIAGE_DETAILS` aggregation and include in the counts line:

```bash
        [[ "$ADAPT_COUNT" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${ADAPT_COUNT} adaptation"
```

And add to the details loop:

```bash
    for detail in "$NOISE_DETAIL" "$HIDDEN_GEM_DETAIL" "$LEECH_DETAIL" "$REFINE_DETAIL" "$CONSOLIDATE_DETAIL" "$ADAPT_DETAIL"; do
```

Also add to the proposal count threshold check — the pending file should be written even if there are only adaptation proposals:

```bash
    TOTAL_PROPOSALS=$((PROPOSAL_COUNT + ADAPT_COUNT))
    if [[ "$TOTAL_PROPOSALS" -gt 0 ]]; then
```

- [ ] **Step 3: Add auto-promotion hint to triage output**

In the triage details section, add streak detection:

```bash
    # Auto-promotion hint (one dimension at a time)
    STREAK_HINT=""
    if [[ -f "$POLICY_FILE" ]]; then
        for dim in surfacing extraction maintenance; do
            streak=$(grep -A1 "\\[approval_streak\\]" "$POLICY_FILE" | grep "$dim" | grep -oE '[0-9]+' | head -1) || streak=0
            if [[ "$streak" -ge 3 ]]; then
                STREAK_HINT="You've approved ${streak} ${dim} proposals in a row. Run /adapt to toggle auto-apply."
                break
            fi
        done
    fi
```

- [ ] **Step 4: Test the integration manually**

Run: `targ build && ~/.claude/engram/bin/engram adapt --status`
Expected: Shows policy status (empty if no policies yet)

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go hooks/session-start.sh
git commit -m "feat: wire analysis into extract and proposals into triage (#387)

Feedback pattern analysis runs at extract time (session end). Resulting
proposals are written to policy.toml and surfaced alongside quadrant
suggestions at session start via the existing triage mechanism.

AI-Used: [claude]"
```

---

### Task 11: Create /adapt skill

Create the `/adapt` skill for interactive policy management within Claude Code.

**Files:**
- Create: `skills/adapt/adapt.md`

- [ ] **Step 1: Create the adapt skill**

Create `skills/adapt/adapt.md`:

```markdown
---
name: adapt
description: >
  Use when the user says "/adapt", "review adaptation proposals", "adjust
  adaptation settings", "toggle auto-apply", or wants to manage engram's
  adaptive policies. Also triggered by triage output suggesting "Run /adapt".
---

# Adapt — Manage Adaptive Policies

Review and manage engram's learned adaptation policies.

## Commands

| Action | Command |
|--------|---------|
| List all policies | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR"` |
| Approve a proposal | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --approve <id>` |
| Reject a proposal | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --reject <id>` |
| Retire an active policy | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --retire <id>` |

## Presentation

When the user asks to review proposals:

1. Run the status command to get all policies
2. Present pending proposals first, grouped by dimension, with rationale
3. Ask the user to approve or reject each pending proposal
4. After all pending proposals are handled, show active policy effectiveness if any have measured results
5. If any dimension has 3+ consecutive approvals, offer to toggle auto-apply

## Auto-Apply

Per-dimension auto-apply is configured in the engram config. When a user wants to toggle it, edit `$ENGRAM_DATA_DIR/../config.toml` to add or update:

```toml
[adaptation]
extraction_auto = false
surfacing_auto = true
maintenance_auto = false
```

Inform the user which dimensions are currently automatic and which are manual.
```

- [ ] **Step 2: Commit**

```bash
git add skills/adapt/adapt.md
git commit -m "feat: add /adapt skill for interactive policy management (#387)

Skill provides commands for reviewing proposals, approving/rejecting,
retiring policies, and toggling per-dimension auto-apply.

AI-Used: [claude]"
```

---

### Task 12: Final integration test and cleanup

Run the full test suite, verify the build, and do a final commit with any fixes.

**Files:**
- All files from previous tasks

- [ ] **Step 1: Run full check**

Run: `targ check-full`
Expected: All tests pass, all lint clean

- [ ] **Step 2: Build and smoke test**

Run: `targ build && ~/.claude/engram/bin/engram adapt --status`
Expected: Prints empty policy list or "No policies."

- [ ] **Step 3: Verify extraction prompt composition**

Manually verify that `extract.SystemPromptWithGuidance` with sample policies produces a well-formed prompt by adding a quick test if not already covered.

- [ ] **Step 4: Fix any issues found**

If any tests or lint issues were found, fix them and commit.

- [ ] **Step 5: Final commit (if needed)**

```bash
git add -A
git commit -m "fix: integration fixes for adaptive policy system (#387)

AI-Used: [claude]"
```

---

## Deferred to Follow-Up Issues

This plan builds the core adaptive policy infrastructure. The following extensions should be filed as separate issues after the foundation is working:

1. **Maintenance threshold policy overrides** — Wire active maintenance policies into `maintain.go` and `review.go` to override `effectivenessThreshold`, `flagThreshold`, `minEvaluations`, etc. Same pattern as surfacing overrides: read policy file, apply overrides.

2. **MaintenanceHistory population** — Modify `maintain/apply.go` to record `MaintenanceAction` entries on memory records when actions are applied, capturing the pre-action effectiveness score. Required for maintenance outcome analysis.

3. **Surfacing pattern analysis** — Add `AnalyzeSurfacingPatterns` to `internal/adapt`. Requires recording which weight configuration was active at surfacing time (new telemetry field), then correlating weight regimes with follow/irrelevance rates.

4. **Maintenance outcome analysis** — Add `AnalyzeMaintenanceOutcomes` to `internal/adapt`. Reads `MaintenanceHistory` from memory records, computes per-action-type success rates, generates maintenance policies.

5. **Policy effectiveness measurement** — Track session count since policy activation. After measurement window, compute before/after corpus-wide metrics (follow rate, irrelevance ratio) and auto-propose retirement for ineffective policies.

6. **Adaptation config in config.toml** — Read `[adaptation]` section from existing config TOML for `measurement_window`, `min_cluster_size`, `min_feedback_events`, and per-dimension `_auto` flags. Currently hardcoded in `adapt.Config`.
