# Measurement Backbone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the adaptive policy feedback loop — every active policy proves its value or gets proposed for retirement.

**Architecture:** Snapshot-based attribution with session-counted measurement windows. Corpus-wide metrics are snapshotted at policy activation and compared after N sessions. MaintenanceHistory is populated at action time with deferred "after" measurement.

**Tech Stack:** Go, TOML persistence, DI via interfaces, `targ` build system

**Issues:** #398, #399, #400, #401

---

### Task 1: Expand Policy.Effectiveness to Flat Corpus Snapshot Fields

**Files:**
- Modify: `internal/policy/policy.go:51-56`
- Modify: `internal/policy/policy_test.go`

- [ ] **Step 1: Write failing test for expanded Effectiveness TOML round-trip**

```go
func TestEffectiveness_ExpandedFields_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	pf := &policy.File{
		Policies: []policy.Policy{{
			ID:        "pol-001",
			Dimension: policy.DimensionSurfacing,
			Directive: "test",
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate:        0.45,
				BeforeIrrelevanceRatio:  0.12,
				BeforeMeanEffectiveness: 62.5,
				AfterFollowRate:         0.55,
				AfterIrrelevanceRatio:   0.08,
				AfterMeanEffectiveness:  71.0,
				MeasuredSessions:        10,
				Validated:               true,
			},
		}},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	err := policy.Save(path, pf)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	loaded, err := policy.Load(path)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(loaded.Policies).To(HaveLen(1))

	eff := loaded.Policies[0].Effectiveness
	g.Expect(eff.BeforeFollowRate).To(BeNumerically("~", 0.45, 0.001))
	g.Expect(eff.BeforeIrrelevanceRatio).To(BeNumerically("~", 0.12, 0.001))
	g.Expect(eff.BeforeMeanEffectiveness).To(BeNumerically("~", 62.5, 0.001))
	g.Expect(eff.AfterFollowRate).To(BeNumerically("~", 0.55, 0.001))
	g.Expect(eff.AfterIrrelevanceRatio).To(BeNumerically("~", 0.08, 0.001))
	g.Expect(eff.AfterMeanEffectiveness).To(BeNumerically("~", 71.0, 0.001))
	g.Expect(eff.MeasuredSessions).To(Equal(10))
	g.Expect(eff.Validated).To(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestEffectiveness_ExpandedFields_RoundTrip -v ./internal/policy/...`
Expected: FAIL — `BeforeFollowRate` field does not exist

- [ ] **Step 3: Replace Effectiveness struct with flat corpus snapshot fields**

In `internal/policy/policy.go`, replace lines 51-56:

```go
// Effectiveness tracks before/after corpus-wide metrics for a policy.
// Uses flat fields (not nested struct) for TOML simplicity.
type Effectiveness struct {
	BeforeFollowRate        float64 `toml:"before_follow_rate,omitempty"`
	BeforeIrrelevanceRatio  float64 `toml:"before_irrelevance_ratio,omitempty"`
	BeforeMeanEffectiveness float64 `toml:"before_mean_effectiveness,omitempty"`
	AfterFollowRate         float64 `toml:"after_follow_rate,omitempty"`
	AfterIrrelevanceRatio   float64 `toml:"after_irrelevance_ratio,omitempty"`
	AfterMeanEffectiveness  float64 `toml:"after_mean_effectiveness,omitempty"`
	MeasuredSessions        int     `toml:"measured_sessions"`
	Validated               bool    `toml:"validated,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestEffectiveness_ExpandedFields_RoundTrip -v ./internal/policy/...`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: PASS. If any existing code references the old `Before`/`After` float64 fields, fix the compilation errors.

- [ ] **Step 6: Commit**

```
feat(policy): expand Effectiveness to flat corpus snapshot fields (#401)

Replaces single Before/After float64 with per-metric fields for corpus-wide
follow rate, irrelevance ratio, and mean effectiveness. Adds Validated flag.
```

---

### Task 2: Add FeedbackCountBefore to MaintenanceAction

**Files:**
- Modify: `internal/memory/record.go:19-29`
- Modify: `internal/memory/maintenance_test.go`

- [ ] **Step 1: Write failing test for FeedbackCountBefore round-trip**

Add to `internal/memory/maintenance_test.go`:

```go
func TestMaintenanceAction_FeedbackCountBefore_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := memory.MemoryRecord{
		Title:   "test",
		Content: "test content",
		MaintenanceHistory: []memory.MaintenanceAction{{
			Action:              "rewrite",
			AppliedAt:           "2026-03-27T10:00:00Z",
			EffectivenessBefore: 25.0,
			FeedbackCountBefore: 42,
			Measured:            false,
		}},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	writeErr := os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}

	var loaded memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &loaded)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	if decodeErr != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory).To(HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].FeedbackCountBefore).To(Equal(42))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestMaintenanceAction_FeedbackCountBefore -v ./internal/memory/...`
Expected: FAIL — `FeedbackCountBefore` field does not exist

- [ ] **Step 3: Add FeedbackCountBefore field**

In `internal/memory/record.go`, add to `MaintenanceAction` struct after `SurfacedCountBefore`:

```go
FeedbackCountBefore int     `toml:"feedback_count_before"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestMaintenanceAction_FeedbackCountBefore -v ./internal/memory/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(memory): add FeedbackCountBefore to MaintenanceAction (#398)

Stores total feedback count at action time so MeasureOutcomes can
determine when sufficient new feedback has accumulated.
```

---

### Task 3: Add CorpusSnapshot Computation

**Files:**
- Create: `internal/adapt/snapshot.go`
- Create: `internal/adapt/snapshot_test.go`

- [ ] **Step 1: Write failing test for ComputeCorpusSnapshot**

Create `internal/adapt/snapshot_test.go`:

```go
package adapt_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
)

func TestComputeCorpusSnapshot_AggregatesMetrics(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:          "mem-1.toml",
			FollowedCount:     3,
			ContradictedCount: 1,
			IgnoredCount:      1,
			IrrelevantCount:   5,
			UpdatedAt:         time.Now(),
		},
		{
			FilePath:          "mem-2.toml",
			FollowedCount:     8,
			ContradictedCount: 0,
			IgnoredCount:      0,
			IrrelevantCount:   2,
			UpdatedAt:         time.Now(),
		},
	}

	snap := adapt.ComputeCorpusSnapshot(memories)

	// Total feedback: 10 + 10 = 20
	// Total followed: 3 + 8 = 11
	// Follow rate: 11/20 = 0.55
	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.55, 0.001))
	// Total irrelevant: 5 + 2 = 7
	// Irrelevance ratio: 7/20 = 0.35
	g.Expect(snap.IrrelevanceRatio).To(BeNumerically("~", 0.35, 0.001))
	// Effectiveness per memory: mem1 = 3/10*100 = 30, mem2 = 8/10*100 = 80
	// Mean: (30 + 80) / 2 = 55
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 55.0, 0.001))
}

func TestComputeCorpusSnapshot_EmptyMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	snap := adapt.ComputeCorpusSnapshot(nil)

	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.0, 0.001))
	g.Expect(snap.IrrelevanceRatio).To(BeNumerically("~", 0.0, 0.001))
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 0.0, 0.001))
}

func TestComputeCorpusSnapshot_SkipsZeroFeedback(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "no-feedback.toml"},
		{
			FilePath:      "has-feedback.toml",
			FollowedCount: 4,
			IgnoredCount:  6,
			UpdatedAt:     time.Now(),
		},
	}

	snap := adapt.ComputeCorpusSnapshot(memories)

	// Only has-feedback counts: follow rate = 4/10 = 0.4
	g.Expect(snap.FollowRate).To(BeNumerically("~", 0.4, 0.001))
	// Mean effectiveness: only 1 memory with feedback, score = 4/10*100 = 40
	g.Expect(snap.MeanEffectiveness).To(BeNumerically("~", 40.0, 0.001))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestComputeCorpusSnapshot -v ./internal/adapt/...`
Expected: FAIL — `ComputeCorpusSnapshot` undefined, `CorpusSnapshot` undefined

- [ ] **Step 3: Implement ComputeCorpusSnapshot**

Create `internal/adapt/snapshot.go`:

```go
package adapt

import "engram/internal/memory"

// CorpusSnapshot captures corpus-wide metrics at a point in time.
type CorpusSnapshot struct {
	FollowRate        float64
	IrrelevanceRatio  float64
	MeanEffectiveness float64
}

// ComputeCorpusSnapshot aggregates follow rate, irrelevance ratio, and mean
// effectiveness across all memories with feedback.
func ComputeCorpusSnapshot(memories []*memory.Stored) CorpusSnapshot {
	var totalFollowed, totalIrrelevant, totalFeedback int

	var effectivenessSum float64

	memoriesWithFeedback := 0

	for _, mem := range memories {
		feedback := mem.FollowedCount + mem.ContradictedCount + mem.IgnoredCount + mem.IrrelevantCount
		if feedback == 0 {
			continue
		}

		totalFollowed += mem.FollowedCount
		totalIrrelevant += mem.IrrelevantCount
		totalFeedback += feedback

		score := float64(mem.FollowedCount) / float64(feedback) * percentMultiplier
		effectivenessSum += score
		memoriesWithFeedback++
	}

	if totalFeedback == 0 {
		return CorpusSnapshot{}
	}

	return CorpusSnapshot{
		FollowRate:        float64(totalFollowed) / float64(totalFeedback),
		IrrelevanceRatio:  float64(totalIrrelevant) / float64(totalFeedback),
		MeanEffectiveness: effectivenessSum / float64(memoriesWithFeedback),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestComputeCorpusSnapshot -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(adapt): add CorpusSnapshot computation (#401)

Shared utility for snapshotting corpus-wide follow rate, irrelevance
ratio, and mean effectiveness. Used at policy activation and evaluation.
```

---

### Task 4: Add HistoryRecorder Interface and Wire into Executor (#398)

**Files:**
- Modify: `internal/maintain/apply.go`
- Modify: `internal/maintain/apply_test.go`

- [ ] **Step 1: Write failing test for MaintenanceHistory population on rewrite**

Add to `internal/maintain/apply_test.go`:

```go
func TestApply_RecordsMaintenanceHistory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var recordedPath string
	var recordedAction memory.MaintenanceAction

	recorder := &fakeHistoryRecorder{
		readFn: func(path string) (*memory.MemoryRecord, error) {
			return &memory.MemoryRecord{
				FollowedCount:     3,
				ContradictedCount: 1,
				IgnoredCount:      1,
				IrrelevantCount:   5,
				SurfacedCount:     12,
			}, nil
		},
		appendFn: func(path string, action memory.MaintenanceAction) error {
			recordedPath = path
			recordedAction = action
			return nil
		},
	}

	rewritten := false
	executor := maintain.NewExecutor(
		maintain.WithRewriter(fakeRewriter(func(_ string, _ map[string]any) error {
			rewritten = true
			return nil
		})),
		maintain.WithLLMCaller2(fakeLLM(`{"principle": "new", "anti_pattern": "old"}`)),
		maintain.WithHistoryRecorder(recorder),
	)

	proposals := []maintain.Proposal{{
		MemoryPath: "/memories/test.toml",
		Quadrant:   "Leech",
		Action:     "rewrite",
		Diagnosis:  "low effectiveness",
		Details:    json.RawMessage(`{}`),
	}}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(Equal(1))
	g.Expect(rewritten).To(BeTrue())
	g.Expect(recordedPath).To(Equal("/memories/test.toml"))
	g.Expect(recordedAction.Action).To(Equal("rewrite"))
	g.Expect(recordedAction.EffectivenessBefore).To(BeNumerically("~", 30.0, 0.001))
	g.Expect(recordedAction.SurfacedCountBefore).To(Equal(12))
	g.Expect(recordedAction.FeedbackCountBefore).To(Equal(10))
	g.Expect(recordedAction.Measured).To(BeFalse())
}
```

And the test helper types:

```go
type fakeHistoryRecorder struct {
	readFn   func(string) (*memory.MemoryRecord, error)
	appendFn func(string, memory.MaintenanceAction) error
}

func (f *fakeHistoryRecorder) ReadRecord(path string) (*memory.MemoryRecord, error) {
	return f.readFn(path)
}

func (f *fakeHistoryRecorder) AppendAction(path string, action memory.MaintenanceAction) error {
	return f.appendFn(path, action)
}
```

Note: the test file already has `fakeRewriter` and `fakeLLM` helpers — reuse them. If `fakeRewriter` is a function type, adapt accordingly. Check the existing test helpers before writing.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestApply_RecordsMaintenanceHistory -v ./internal/maintain/...`
Expected: FAIL — `HistoryRecorder` type does not exist, `WithHistoryRecorder` does not exist

- [ ] **Step 3: Add HistoryRecorder interface and wire into Executor**

In `internal/maintain/apply.go`, add:

```go
// HistoryRecorder reads memory records and appends maintenance action history.
type HistoryRecorder interface {
	ReadRecord(path string) (*memory.MemoryRecord, error)
	AppendAction(path string, action memory.MaintenanceAction) error
}
```

Add `historyRecorder` field to `Executor` struct:

```go
type Executor struct {
	rewriter        MemoryRewriter
	remover         MemoryRemover
	removeFile      func(path string) error
	llmCaller       LLMCaller
	confirmer       Confirmer
	historyRecorder HistoryRecorder
}
```

Add option:

```go
// WithHistoryRecorder sets the maintenance history recorder.
func WithHistoryRecorder(r HistoryRecorder) ExecutorOption {
	return func(e *Executor) { e.historyRecorder = r }
}
```

Add import for `"engram/internal/memory"` and `"time"`.

Add a helper method to record history:

```go
// recordHistory reads the memory, computes before-state, and appends a MaintenanceAction.
// No-op if historyRecorder is nil (backward compatible).
func (e *Executor) recordHistory(path, action string) {
	if e.historyRecorder == nil {
		return
	}

	record, err := e.historyRecorder.ReadRecord(path)
	if err != nil {
		return // best-effort; don't fail the action
	}

	feedbackCount := record.FollowedCount + record.ContradictedCount +
		record.IgnoredCount + record.IrrelevantCount

	var effectivenessBefore float64
	if feedbackCount > 0 {
		effectivenessBefore = float64(record.FollowedCount) / float64(feedbackCount) * percentMultiplier
	}

	entry := memory.MaintenanceAction{
		Action:              action,
		AppliedAt:           time.Now().UTC().Format(time.RFC3339),
		EffectivenessBefore: effectivenessBefore,
		SurfacedCountBefore: record.SurfacedCount,
		FeedbackCountBefore: feedbackCount,
		Measured:            false,
	}

	_ = e.historyRecorder.AppendAction(path, entry) // best-effort
}
```

Add constant: `const percentMultiplier = 100.0`

Wire `recordHistory` into `applyOne` — call it after each successful action (except remove, since the file is deleted). Modify each apply method to call `e.recordHistory(proposal.MemoryPath, proposal.Action)` after the successful rewrite:

In `llmRewrite`, after `e.rewriter.Rewrite(...)` succeeds (line 266), before `return true, "", nil`:
```go
e.recordHistory(proposal.MemoryPath, proposal.Action)
```

In `applyBroadenKeywords`, after `e.rewriter.Rewrite(...)` succeeds (line 125), before `return true, "", nil`:
```go
e.recordHistory(proposal.MemoryPath, proposal.Action)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestApply_RecordsMaintenanceHistory -v ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(maintain): populate MaintenanceHistory on action apply (#398)

Adds HistoryRecorder interface to Executor. After each successful
rewrite/broaden/stale-update, records a MaintenanceAction with
before-state effectiveness, surfaced count, and feedback count.
Skipped for removals (file is deleted). Best-effort — recording
failures don't block the action.
```

---

### Task 5: Implement HistoryRecorder in CLI Wiring

**Files:**
- Create: `internal/maintain/history.go`
- Create: `internal/maintain/history_test.go`
- Modify: `internal/cli/cli.go` (RunMaintain wiring)

- [ ] **Step 1: Write failing test for TOMLHistoryRecorder**

Create `internal/maintain/history_test.go`:

```go
package maintain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestTOMLHistoryRecorder_ReadRecord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	record := memory.MemoryRecord{
		Title:         "test memory",
		Content:       "some content",
		FollowedCount: 5,
		SurfacedCount: 10,
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	writeErr := os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	recorder := maintain.NewTOMLHistoryRecorder()

	loaded, readErr := recorder.ReadRecord(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}

	g.Expect(loaded.FollowedCount).To(Equal(5))
	g.Expect(loaded.SurfacedCount).To(Equal(10))
}

func TestTOMLHistoryRecorder_AppendAction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	record := memory.MemoryRecord{
		Title:   "test memory",
		Content: "some content",
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	writeErr := os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	recorder := maintain.NewTOMLHistoryRecorder()

	action := memory.MaintenanceAction{
		Action:              "rewrite",
		AppliedAt:           "2026-03-27T10:00:00Z",
		EffectivenessBefore: 25.0,
		SurfacedCountBefore: 12,
		FeedbackCountBefore: 10,
		Measured:            false,
	}

	appendErr := recorder.AppendAction(path, action)
	g.Expect(appendErr).NotTo(HaveOccurred())
	if appendErr != nil {
		return
	}

	// Verify by re-reading
	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}

	var loaded memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &loaded)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	if decodeErr != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory).To(HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].Action).To(Equal("rewrite"))
	g.Expect(loaded.MaintenanceHistory[0].FeedbackCountBefore).To(Equal(10))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestTOMLHistoryRecorder -v ./internal/maintain/...`
Expected: FAIL — `NewTOMLHistoryRecorder` undefined

- [ ] **Step 3: Implement TOMLHistoryRecorder**

Create `internal/maintain/history.go`:

```go
package maintain

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// TOMLHistoryRecorder reads and appends maintenance history to memory TOML files.
type TOMLHistoryRecorder struct {
	readFile  func(name string) ([]byte, error)
	writeFile func(name string, data []byte, perm os.FileMode) error
}

// NewTOMLHistoryRecorder creates a TOMLHistoryRecorder with real filesystem operations.
func NewTOMLHistoryRecorder(opts ...HistoryRecorderOption) *TOMLHistoryRecorder {
	return &TOMLHistoryRecorder{
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}
}

// ReadRecord reads a memory TOML file and returns the MemoryRecord.
func (r *TOMLHistoryRecorder) ReadRecord(path string) (*memory.MemoryRecord, error) {
	data, err := r.readFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading record: %w", err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding record: %w", decodeErr)
	}

	return &record, nil
}

// AppendAction reads the memory TOML, appends a MaintenanceAction, and writes it back.
func (r *TOMLHistoryRecorder) AppendAction(path string, action memory.MaintenanceAction) error {
	data, err := r.readFile(path)
	if err != nil {
		return fmt.Errorf("reading record for history: %w", err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return fmt.Errorf("decoding record for history: %w", decodeErr)
	}

	record.MaintenanceHistory = append(record.MaintenanceHistory, action)

	var buf bytes.Buffer

	encodeErr := toml.NewEncoder(&buf).Encode(record)
	if encodeErr != nil {
		return fmt.Errorf("encoding record with history: %w", encodeErr)
	}

	const filePerm = 0o644

	writeErr := r.writeFile(path, buf.Bytes(), filePerm)
	if writeErr != nil {
		return fmt.Errorf("writing record with history: %w", writeErr)
	}

	return nil
}

// HistoryRecorderOption configures a TOMLHistoryRecorder.
type HistoryRecorderOption func(*TOMLHistoryRecorder)

// WithHistoryReadFile overrides the file reading function.
func WithHistoryReadFile(fn func(name string) ([]byte, error)) HistoryRecorderOption {
	return func(r *TOMLHistoryRecorder) { r.readFile = fn }
}

// WithHistoryWriteFile overrides the file writing function.
func WithHistoryWriteFile(fn func(name string, data []byte, perm os.FileMode) error) HistoryRecorderOption {
	return func(r *TOMLHistoryRecorder) { r.writeFile = fn }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestTOMLHistoryRecorder -v ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Wire into CLI**

In `internal/cli/cli.go`, in the `RunMaintain` function where `maintain.NewExecutor` is called, add:

```go
maintain.WithHistoryRecorder(maintain.NewTOMLHistoryRecorder()),
```

- [ ] **Step 6: Run full check**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(maintain): add TOMLHistoryRecorder and wire into CLI (#398)

Implements the HistoryRecorder interface that reads/writes maintenance
action history to memory TOML files. Wired into the CLI maintain path
so all applied actions now record their before-state.
```

---

### Task 6: Add MeasureOutcomes to Adapt Package

**Files:**
- Create: `internal/adapt/measure.go`
- Create: `internal/adapt/measure_test.go`

- [ ] **Step 1: Write failing test for MeasureOutcomes**

Create `internal/adapt/measure_test.go`:

```go
package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
)

func TestMeasureOutcomes_FillsAfterScores(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount:     8,
				ContradictedCount: 0,
				IgnoredCount:      2,
				IrrelevantCount:   5,
				SurfacedCount:     20,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action:              "rewrite",
					AppliedAt:           "2026-03-27T10:00:00Z",
					EffectivenessBefore: 30.0,
					SurfacedCountBefore: 10,
					FeedbackCountBefore: 10,
					Measured:            false,
				}},
			},
		},
	}

	// Total current feedback = 8+0+2+5 = 15
	// FeedbackCountBefore = 10, so 5 new events (>= minNewFeedback of 5)
	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal("mem-1.toml"))
	g.Expect(results[0].ActionIndex).To(Equal(0))
	// Effectiveness after: 8/15 * 100 = 53.33
	g.Expect(results[0].EffectivenessAfter).To(BeNumerically("~", 53.33, 0.01))
	g.Expect(results[0].SurfacedCountAfter).To(Equal(20))
}

func TestMeasureOutcomes_SkipsInsufficientFeedback(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount:  4,
				IgnoredCount:   2,
				SurfacedCount:  8,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action:              "rewrite",
					FeedbackCountBefore: 5,
					Measured:            false,
				}},
			},
		},
	}

	// Current feedback = 6, before = 5, diff = 1 < minNewFeedback of 5
	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)

	g.Expect(results).To(BeEmpty())
}

func TestMeasureOutcomes_SkipsAlreadyMeasured(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				FollowedCount: 20,
				MaintenanceHistory: []memory.MaintenanceAction{{
					Action:              "rewrite",
					FeedbackCountBefore: 5,
					Measured:            true,
				}},
			},
		},
	}

	const minNewFeedback = 5

	results := adapt.MeasureOutcomes(records, minNewFeedback)

	g.Expect(results).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestMeasureOutcomes -v ./internal/adapt/...`
Expected: FAIL — `MeasureOutcomes`, `MeasurableRecord` undefined

- [ ] **Step 3: Implement MeasureOutcomes**

Create `internal/adapt/measure.go`:

```go
package adapt

import "engram/internal/memory"

// MeasurableRecord pairs a memory file path with its loaded record.
type MeasurableRecord struct {
	Path   string
	Record memory.MemoryRecord
}

// MeasuredResult identifies a specific MaintenanceAction that is now measurable.
type MeasuredResult struct {
	Path                string
	ActionIndex         int
	EffectivenessAfter  float64
	SurfacedCountAfter  int
}

// MeasureOutcomes scans records for unmeasured MaintenanceActions that have
// accumulated at least minNewFeedback new feedback events since the action.
// Returns results identifying which actions can now have their "after" scores filled.
func MeasureOutcomes(records []MeasurableRecord, minNewFeedback int) []MeasuredResult {
	results := make([]MeasuredResult, 0)

	for _, rec := range records {
		currentFeedback := rec.Record.FollowedCount + rec.Record.ContradictedCount +
			rec.Record.IgnoredCount + rec.Record.IrrelevantCount

		var effectivenessNow float64
		if currentFeedback > 0 {
			effectivenessNow = float64(rec.Record.FollowedCount) /
				float64(currentFeedback) * percentMultiplier
		}

		for idx, action := range rec.Record.MaintenanceHistory {
			if action.Measured {
				continue
			}

			newFeedback := currentFeedback - action.FeedbackCountBefore
			if newFeedback < minNewFeedback {
				continue
			}

			results = append(results, MeasuredResult{
				Path:                rec.Path,
				ActionIndex:         idx,
				EffectivenessAfter:  effectivenessNow,
				SurfacedCountAfter:  rec.Record.SurfacedCount,
			})
		}
	}

	return results
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestMeasureOutcomes -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(adapt): add MeasureOutcomes for deferred action measurement (#398)

Scans memory records for unmeasured MaintenanceActions, checks if
sufficient new feedback has accumulated, and returns results with
computed after-state effectiveness and surfaced counts.
```

---

### Task 7: Add AnalyzeSurfacingPatterns (#399)

**Files:**
- Modify: `internal/adapt/analyze.go`
- Modify: `internal/adapt/analyze_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/adapt/analyze_test.go`:

```go
func TestAnalyzeSurfacingPatterns_ProposesRetirement(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "mem-1.toml",
			FollowedCount:   2,
			IrrelevantCount: 8,
		},
		{
			FilePath:      "mem-2.toml",
			FollowedCount: 3,
			IgnoredCount:  7,
		},
	}

	activePolicies := []policy.Policy{{
		ID:        "pol-001",
		Dimension: policy.DimensionSurfacing,
		Status:    policy.StatusActive,
		Directive: "adjust tier boosts",
		Effectiveness: policy.Effectiveness{
			BeforeFollowRate:        0.50,
			BeforeIrrelevanceRatio:  0.10,
			BeforeMeanEffectiveness: 60.0,
			MeasuredSessions:        10,
		},
	}}

	const measurementWindow = 10

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, measurementWindow)

	// Current corpus: follow rate = 5/20 = 0.25, worse than before (0.50)
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Directive).To(ContainSubstring("pol-001"))
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionSurfacing))
	g.Expect(proposals[0].Status).To(Equal(policy.StatusProposed))
}

func TestAnalyzeSurfacingPatterns_ValidatesImproved(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:      "mem-1.toml",
			FollowedCount: 8,
			IgnoredCount:  2,
		},
	}

	activePolicies := []policy.Policy{{
		ID:        "pol-002",
		Dimension: policy.DimensionSurfacing,
		Status:    policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			BeforeFollowRate:        0.50,
			BeforeMeanEffectiveness: 40.0,
			MeasuredSessions:        10,
		},
	}}

	const measurementWindow = 10

	proposals := adapt.AnalyzeSurfacingPatterns(memories, activePolicies, measurementWindow)

	// Current follow rate = 0.8, better than before (0.5)
	// No retirement proposal — returns empty
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeSurfacingPatterns_SkipsBelowWindow(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	activePolicies := []policy.Policy{{
		ID:        "pol-003",
		Dimension: policy.DimensionSurfacing,
		Status:    policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			MeasuredSessions: 5,
		},
	}}

	const measurementWindow = 10

	proposals := adapt.AnalyzeSurfacingPatterns(nil, activePolicies, measurementWindow)

	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeSurfacingPatterns_SkipsValidated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	activePolicies := []policy.Policy{{
		ID:        "pol-004",
		Dimension: policy.DimensionSurfacing,
		Status:    policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			MeasuredSessions: 10,
			Validated:        true,
		},
	}}

	const measurementWindow = 10

	proposals := adapt.AnalyzeSurfacingPatterns(nil, activePolicies, measurementWindow)

	g.Expect(proposals).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestAnalyzeSurfacingPatterns -v ./internal/adapt/...`
Expected: FAIL — `AnalyzeSurfacingPatterns` undefined

- [ ] **Step 3: Implement AnalyzeSurfacingPatterns**

Add to `internal/adapt/analyze.go`:

```go
// AnalyzeSurfacingPatterns evaluates active surfacing policies that have
// exceeded their measurement window. Compares current corpus snapshot to
// the before-snapshot stored on each policy. Returns retirement proposals
// for policies that degraded or didn't improve metrics.
func AnalyzeSurfacingPatterns(
	memories []*memory.Stored,
	activePolicies []policy.Policy,
	measurementWindow int,
) []policy.Policy {
	proposals := make([]policy.Policy, 0)

	for _, pol := range activePolicies {
		if pol.Dimension != policy.DimensionSurfacing {
			continue
		}

		if pol.Effectiveness.Validated {
			continue
		}

		if pol.Effectiveness.MeasuredSessions < measurementWindow {
			continue
		}

		current := ComputeCorpusSnapshot(memories)

		improved := current.FollowRate > pol.Effectiveness.BeforeFollowRate ||
			current.MeanEffectiveness > pol.Effectiveness.BeforeMeanEffectiveness

		if improved {
			continue
		}

		proposals = append(proposals, policy.Policy{
			Dimension: policy.DimensionSurfacing,
			Directive: fmt.Sprintf(
				"retire %s: follow rate %.0f%% (was %.0f%%), mean effectiveness %.1f (was %.1f)",
				pol.ID,
				current.FollowRate*percentMultiplier,
				pol.Effectiveness.BeforeFollowRate*percentMultiplier,
				current.MeanEffectiveness,
				pol.Effectiveness.BeforeMeanEffectiveness,
			),
			Rationale: fmt.Sprintf(
				"policy %s did not improve corpus metrics after %d sessions",
				pol.ID, pol.Effectiveness.MeasuredSessions,
			),
			Evidence: policy.Evidence{
				FollowRate:       current.FollowRate,
				SampleSize:       len(memories),
				SessionsObserved: pol.Effectiveness.MeasuredSessions,
			},
			Status: policy.StatusProposed,
		})
	}

	return proposals
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestAnalyzeSurfacingPatterns -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(adapt): add AnalyzeSurfacingPatterns for policy evaluation (#399)

Evaluates active surfacing policies past their measurement window.
Compares current corpus snapshot to before-snapshot. Proposes
retirement for policies that didn't improve follow rate or
mean effectiveness.
```

---

### Task 8: Add AnalyzeMaintenanceOutcomes (#400)

**Files:**
- Modify: `internal/adapt/analyze.go` (or new file `internal/adapt/maintenance.go`)
- Add tests

- [ ] **Step 1: Write failing test**

Create `internal/adapt/maintenance_test.go`:

```go
package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAnalyzeMaintenanceOutcomes_ProposesAlternative(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
					{Action: "rewrite", EffectivenessBefore: 25.0, EffectivenessAfter: 20.0, Measured: true},
				},
			},
		},
		{
			Path: "mem-2.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 30.0, EffectivenessAfter: 28.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: 3,
		MinSuccessRate:      0.4,
	}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)

	// 3 rewrites, all degraded (after < before), success rate = 0/3 = 0.0
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionMaintenance))
	g.Expect(proposals[0].Directive).To(ContainSubstring("rewrite"))
	g.Expect(proposals[0].Status).To(Equal(policy.StatusProposed))
}

func TestAnalyzeMaintenanceOutcomes_SkipsInsufficientData(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: 3,
		MinSuccessRate:      0.4,
	}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)

	// Only 1 measured outcome, need 3 minimum
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeMaintenanceOutcomes_NoProposalWhenSuccessful(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "broaden_keywords", EffectivenessBefore: 20.0, EffectivenessAfter: 45.0, Measured: true},
					{Action: "broaden_keywords", EffectivenessBefore: 30.0, EffectivenessAfter: 50.0, Measured: true},
					{Action: "broaden_keywords", EffectivenessBefore: 25.0, EffectivenessAfter: 40.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: 3,
		MinSuccessRate:      0.4,
	}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)

	// 3/3 improved = 100% success rate > 40% threshold
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeMaintenanceOutcomes_IgnoresUnmeasured(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
					{Action: "rewrite", EffectivenessBefore: 25.0, Measured: false},
					{Action: "rewrite", EffectivenessBefore: 30.0, Measured: false},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: 3,
		MinSuccessRate:      0.4,
	}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)

	// Only 1 measured, need 3
	g.Expect(proposals).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestAnalyzeMaintenanceOutcomes -v ./internal/adapt/...`
Expected: FAIL — `AnalyzeMaintenanceOutcomes`, `MaintenanceAnalysisConfig` undefined

- [ ] **Step 3: Implement AnalyzeMaintenanceOutcomes**

Create `internal/adapt/maintenance.go`:

```go
package adapt

import (
	"fmt"

	"engram/internal/policy"
)

// MaintenanceAnalysisConfig holds thresholds for maintenance outcome analysis.
type MaintenanceAnalysisConfig struct {
	MinMeasuredOutcomes int
	MinSuccessRate      float64
}

// AnalyzeMaintenanceOutcomes groups measured MaintenanceHistory entries by action
// type and generates maintenance policy proposals for action types with low success
// rates (effectiveness didn't improve).
func AnalyzeMaintenanceOutcomes(
	records []MeasurableRecord,
	cfg MaintenanceAnalysisConfig,
) []policy.Policy {
	// Aggregate: action type -> (total measured, total improved)
	type actionStats struct {
		total    int
		improved int
	}

	stats := make(map[string]*actionStats)

	for _, rec := range records {
		for _, action := range rec.Record.MaintenanceHistory {
			if !action.Measured {
				continue
			}

			entry, exists := stats[action.Action]
			if !exists {
				entry = &actionStats{}
				stats[action.Action] = entry
			}

			entry.total++

			if action.EffectivenessAfter > action.EffectivenessBefore {
				entry.improved++
			}
		}
	}

	proposals := make([]policy.Policy, 0)

	for actionType, stat := range stats {
		if stat.total < cfg.MinMeasuredOutcomes {
			continue
		}

		successRate := float64(stat.improved) / float64(stat.total)
		if successRate >= cfg.MinSuccessRate {
			continue
		}

		proposals = append(proposals, policy.Policy{
			Dimension: policy.DimensionMaintenance,
			Directive: fmt.Sprintf(
				"%s has %.0f%% success rate (%d/%d improved); consider alternative actions",
				actionType,
				successRate*percentMultiplier,
				stat.improved, stat.total,
			),
			Rationale: fmt.Sprintf(
				"action type %q improved effectiveness in only %d of %d measured outcomes (%.0f%% < %.0f%% threshold)",
				actionType, stat.improved, stat.total,
				successRate*percentMultiplier, cfg.MinSuccessRate*percentMultiplier,
			),
			Evidence: policy.Evidence{
				FollowRate: successRate,
				SampleSize: stat.total,
			},
			Status: policy.StatusProposed,
		})
	}

	return proposals
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestAnalyzeMaintenanceOutcomes -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(adapt): add AnalyzeMaintenanceOutcomes (#400)

Groups measured maintenance actions by type, computes success rate
(did effectiveness improve?), and proposes alternatives for action
types with low success rates.
```

---

### Task 9: Add EvaluateActivePolicies (#401)

**Files:**
- Create: `internal/adapt/evaluate.go`
- Create: `internal/adapt/evaluate_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/adapt/evaluate_test.go`:

```go
package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestEvaluateActivePolicies_ProposesRetirementForDegraded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "mem-1.toml",
			FollowedCount:   2,
			IrrelevantCount: 8,
		},
	}

	policies := []policy.Policy{
		{
			ID:        "pol-001",
			Dimension: policy.DimensionExtraction,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate:        0.50,
				BeforeMeanEffectiveness: 60.0,
				MeasuredSessions:        10,
			},
		},
		{
			ID:        "pol-002",
			Dimension: policy.DimensionMaintenance,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				BeforeFollowRate:        0.50,
				BeforeMeanEffectiveness: 60.0,
				MeasuredSessions:        10,
			},
		},
	}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(memories, policies, measurementWindow)

	// Current follow rate = 2/10 = 0.2, worse than 0.50
	// Both policies should get retirement proposals
	g.Expect(result.RetirementProposals).To(HaveLen(2))
	g.Expect(result.ValidatedPolicyIDs).To(BeEmpty())
}

func TestEvaluateActivePolicies_ValidatesImproved(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:      "mem-1.toml",
			FollowedCount: 9,
			IgnoredCount:  1,
		},
	}

	policies := []policy.Policy{{
		ID:        "pol-001",
		Dimension: policy.DimensionExtraction,
		Status:    policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			BeforeFollowRate:        0.50,
			BeforeMeanEffectiveness: 40.0,
			MeasuredSessions:        10,
		},
	}}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(memories, policies, measurementWindow)

	// Current follow rate = 0.9, better than 0.5
	g.Expect(result.RetirementProposals).To(BeEmpty())
	g.Expect(result.ValidatedPolicyIDs).To(ConsistOf("pol-001"))
}

func TestEvaluateActivePolicies_SkipsValidatedAndBelowWindow(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policies := []policy.Policy{
		{
			ID:        "pol-001",
			Dimension: policy.DimensionSurfacing,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				MeasuredSessions: 5, // below window
			},
		},
		{
			ID:        "pol-002",
			Dimension: policy.DimensionExtraction,
			Status:    policy.StatusActive,
			Effectiveness: policy.Effectiveness{
				MeasuredSessions: 10,
				Validated:        true, // already validated
			},
		},
	}

	const measurementWindow = 10

	result := adapt.EvaluateActivePolicies(nil, policies, measurementWindow)

	g.Expect(result.RetirementProposals).To(BeEmpty())
	g.Expect(result.ValidatedPolicyIDs).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestEvaluateActivePolicies -v ./internal/adapt/...`
Expected: FAIL — `EvaluateActivePolicies`, `EvaluationResult` undefined

- [ ] **Step 3: Implement EvaluateActivePolicies**

Create `internal/adapt/evaluate.go`:

```go
package adapt

import (
	"fmt"

	"engram/internal/memory"
	"engram/internal/policy"
)

// EvaluationResult holds the outcome of evaluating active policies.
type EvaluationResult struct {
	RetirementProposals []policy.Policy
	ValidatedPolicyIDs  []string
}

// EvaluateActivePolicies checks all active policies past their measurement
// window. Compares current corpus snapshot to before-snapshot.
// Returns retirement proposals for degraded policies and IDs of validated ones.
func EvaluateActivePolicies(
	memories []*memory.Stored,
	activePolicies []policy.Policy,
	measurementWindow int,
) EvaluationResult {
	result := EvaluationResult{
		RetirementProposals: make([]policy.Policy, 0),
		ValidatedPolicyIDs:  make([]string, 0),
	}

	current := ComputeCorpusSnapshot(memories)

	for _, pol := range activePolicies {
		if pol.Status != policy.StatusActive {
			continue
		}

		if pol.Effectiveness.Validated {
			continue
		}

		if pol.Effectiveness.MeasuredSessions < measurementWindow {
			continue
		}

		improved := current.FollowRate > pol.Effectiveness.BeforeFollowRate ||
			current.MeanEffectiveness > pol.Effectiveness.BeforeMeanEffectiveness

		if improved {
			result.ValidatedPolicyIDs = append(result.ValidatedPolicyIDs, pol.ID)

			continue
		}

		result.RetirementProposals = append(result.RetirementProposals, policy.Policy{
			Dimension: pol.Dimension,
			Directive: fmt.Sprintf(
				"retire %s: follow rate %.0f%% (was %.0f%%), mean effectiveness %.1f (was %.1f)",
				pol.ID,
				current.FollowRate*percentMultiplier,
				pol.Effectiveness.BeforeFollowRate*percentMultiplier,
				current.MeanEffectiveness,
				pol.Effectiveness.BeforeMeanEffectiveness,
			),
			Rationale: fmt.Sprintf(
				"policy %s did not improve corpus metrics after %d sessions",
				pol.ID, pol.Effectiveness.MeasuredSessions,
			),
			Evidence: policy.Evidence{
				FollowRate:       current.FollowRate,
				SampleSize:       len(memories),
				SessionsObserved: pol.Effectiveness.MeasuredSessions,
			},
			Status: policy.StatusProposed,
		})
	}

	return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestEvaluateActivePolicies -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(adapt): add EvaluateActivePolicies for measurement loop (#401)

Checks all active policies past their measurement window. Compares
current corpus snapshot to before-snapshot. Proposes retirement for
degraded policies, marks improved ones as validated.
```

---

### Task 10: Snapshot Corpus Metrics at Policy Approval

**Files:**
- Modify: `internal/cli/adapt.go:59-75`
- Modify: `internal/cli/adapt_test.go` (or create if doesn't exist)

- [ ] **Step 1: Write failing test**

The `adaptApprove` function needs to accept a snapshot function so it can populate `Effectiveness.Before*` fields. Add a test:

```go
func TestAdaptApprove_SnapshotsBefore(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{{
			ID:        "pol-001",
			Dimension: policy.DimensionSurfacing,
			Directive: "test policy",
			Status:    policy.StatusProposed,
		}},
	}

	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	snapshot := adapt.CorpusSnapshot{
		FollowRate:        0.45,
		IrrelevanceRatio:  0.12,
		MeanEffectiveness: 62.5,
	}

	var buf bytes.Buffer

	approveErr := cli.AdaptApproveWithSnapshot(policyPath, "pol-001", snapshot, &buf)
	g.Expect(approveErr).NotTo(HaveOccurred())
	if approveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(policyPath)
	g.Expect(loadErr).NotTo(HaveOccurred())
	if loadErr != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeFollowRate).To(
		BeNumerically("~", 0.45, 0.001))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeIrrelevanceRatio).To(
		BeNumerically("~", 0.12, 0.001))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeMeanEffectiveness).To(
		BeNumerically("~", 62.5, 0.001))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestAdaptApprove_SnapshotsBefore -v ./internal/cli/...`
Expected: FAIL — `AdaptApproveWithSnapshot` undefined

- [ ] **Step 3: Implement snapshot at approval time**

Add to `internal/cli/adapt.go`:

```go
// AdaptApproveWithSnapshot approves a policy and stores the corpus snapshot
// as the before-state for later effectiveness measurement.
func AdaptApproveWithSnapshot(
	policyPath, id string,
	snapshot adapt.CorpusSnapshot,
	stdout io.Writer,
) error {
	policyFile, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	approveErr := policyFile.Approve(id, timestamp)
	if approveErr != nil {
		return fmt.Errorf("adapt: %w", approveErr)
	}

	// Find the approved policy and set before-snapshot
	for idx := range policyFile.Policies {
		if policyFile.Policies[idx].ID == id {
			policyFile.Policies[idx].Effectiveness.BeforeFollowRate = snapshot.FollowRate
			policyFile.Policies[idx].Effectiveness.BeforeIrrelevanceRatio = snapshot.IrrelevanceRatio
			policyFile.Policies[idx].Effectiveness.BeforeMeanEffectiveness = snapshot.MeanEffectiveness

			break
		}
	}

	saveErr := policy.Save(policyPath, policyFile)
	if saveErr != nil {
		return fmt.Errorf("adapt: %w", saveErr)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Approved policy %s (snapshot: follow=%.0f%%, eff=%.1f)\n",
		id, snapshot.FollowRate*percentMultiplier, snapshot.MeanEffectiveness)

	return nil
}
```

Add constant `const percentMultiplier = 100.0` and import `"engram/internal/adapt"`.

Update `RunAdapt` to call `AdaptApproveWithSnapshot` instead of `adaptApprove` when approving. The snapshot is computed from the current memory corpus:

```go
case *approve != "":
	allMemories, listErr := retrieve.New().ListMemories(context.Background(), *dataDir)
	if listErr != nil {
		return fmt.Errorf("adapt: listing memories: %w", listErr)
	}

	snapshot := adapt.ComputeCorpusSnapshot(allMemories)

	return AdaptApproveWithSnapshot(policyPath, *approve, snapshot, stdout)
```

Add imports for `"context"`, `"engram/internal/adapt"`, `"engram/internal/retrieve"`.

Remove the old `adaptApprove` function (it's replaced by `AdaptApproveWithSnapshot`).

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestAdaptApprove_SnapshotsBefore -v ./internal/cli/...`
Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```
feat(cli): snapshot corpus metrics at policy approval (#401)

When approving a policy, compute current corpus follow rate,
irrelevance ratio, and mean effectiveness and store as the
before-snapshot for later evaluation.
```

---

### Task 11: Add Session Counting for Active Policies

**Files:**
- Modify: `internal/cli/cli.go` (in the surface/extract path)
- Add test

- [ ] **Step 1: Write failing test for incrementPolicySessions**

```go
func TestIncrementPolicySessions(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	pf := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 3},
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionExtraction,
				Status:    policy.StatusActive,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 7},
			},
			{
				ID:        "pol-003",
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusRetired,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 5},
			},
		},
	}

	err := policy.Save(policyPath, pf)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	cli.IncrementPolicySessions(policyPath)

	loaded, loadErr := policy.Load(policyPath)
	g.Expect(loadErr).NotTo(HaveOccurred())
	if loadErr != nil {
		return
	}

	// Active policies get incremented
	g.Expect(loaded.Policies[0].Effectiveness.MeasuredSessions).To(Equal(4))
	g.Expect(loaded.Policies[1].Effectiveness.MeasuredSessions).To(Equal(8))
	// Retired policy is NOT incremented
	g.Expect(loaded.Policies[2].Effectiveness.MeasuredSessions).To(Equal(5))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestIncrementPolicySessions -v ./internal/cli/...`
Expected: FAIL — `IncrementPolicySessions` undefined

- [ ] **Step 3: Implement IncrementPolicySessions**

Add to `internal/cli/adapt.go`:

```go
// IncrementPolicySessions increments MeasuredSessions on all active policies.
// Called once per session. Errors silently ignored (fire-and-forget, ARCH-6).
func IncrementPolicySessions(policyPath string) {
	pf, err := policy.Load(policyPath)
	if err != nil {
		return
	}

	changed := false

	for idx := range pf.Policies {
		if pf.Policies[idx].Status == policy.StatusActive {
			pf.Policies[idx].Effectiveness.MeasuredSessions++
			changed = true
		}
	}

	if changed {
		_ = policy.Save(policyPath, pf)
	}
}
```

- [ ] **Step 4: Wire into the surface path**

In `internal/cli/cli.go`, in the surface code path (wherever `RunSurface` or the surface subcommand is dispatched), add a call to `IncrementPolicySessions(policyPath)` after surfacing completes. This should be after the `s.Run(...)` call succeeds. The `policyPath` is `filepath.Join(dataDir, "policy.toml")`.

Look for the exact location in RunSurface or the surface dispatch in `Run`. Add the call as a fire-and-forget at the end of the surface path.

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test -- -run TestIncrementPolicySessions -v ./internal/cli/...`
Expected: PASS

- [ ] **Step 6: Run full check**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 7: Commit**

```
feat(cli): increment policy session counter per surface invocation (#401)

Each surface invocation increments MeasuredSessions on all active
policies. Used to determine when a policy's measurement window has
been reached for evaluation.
```

---

### Task 12: Wire Everything into the Analysis Pipeline

**Files:**
- Modify: `internal/adapt/analyze.go` (expand AnalyzeAll)
- Modify: `internal/cli/cli.go:1149-1184` (runAdaptationAnalysis)

- [ ] **Step 1: Write failing test for expanded AnalyzeAll**

Add to `internal/adapt/analyze_test.go`:

```go
func TestAnalyzeAll_IncludesSurfacingAndEvaluation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "mem-1.toml",
			FollowedCount:   2,
			IrrelevantCount: 8,
			Keywords:        []string{"test"},
			Tier:            "A",
		},
	}

	activePolicies := []policy.Policy{{
		ID:        "pol-001",
		Dimension: policy.DimensionSurfacing,
		Status:    policy.StatusActive,
		Effectiveness: policy.Effectiveness{
			BeforeFollowRate:        0.80,
			BeforeMeanEffectiveness: 70.0,
			MeasuredSessions:        10,
		},
	}}

	cfg := adapt.Config{
		MinClusterSize:      5,
		MinFeedbackEvents:   3,
		MeasurementWindow:   10,
		MaintenanceMinOutcomes: 3,
		MaintenanceMinSuccess:  0.4,
	}

	proposals := adapt.AnalyzeAll(memories, cfg, activePolicies, nil)

	// Should include at least the retirement proposal for pol-001
	// (current follow rate 0.2 < before 0.80)
	hasRetirement := false
	for _, p := range proposals {
		if p.Dimension == policy.DimensionSurfacing {
			hasRetirement = true
		}
	}
	g.Expect(hasRetirement).To(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestAnalyzeAll_IncludesSurfacingAndEvaluation -v ./internal/adapt/...`
Expected: FAIL — `AnalyzeAll` signature mismatch (too many arguments) or missing Config fields

- [ ] **Step 3: Expand Config and AnalyzeAll**

Update `Config` in `internal/adapt/analyze.go`:

```go
// Config holds thresholds for the analysis engine.
type Config struct {
	MinClusterSize         int
	MinFeedbackEvents      int
	MeasurementWindow      int
	MaintenanceMinOutcomes int
	MaintenanceMinSuccess  float64
	MinNewFeedback         int
}
```

Update `AnalyzeAll` signature and implementation:

```go
// AnalyzeAll runs all analysis and evaluation functions. Combines proposals sorted by sample size.
func AnalyzeAll(
	memories []*memory.Stored,
	cfg Config,
	activePolicies []policy.Policy,
	measurableRecords []MeasurableRecord,
) []policy.Policy {
	contentProposals := AnalyzeContentPatterns(memories, cfg)
	structuralProposals := AnalyzeStructuralPatterns(memories, cfg)
	surfacingProposals := AnalyzeSurfacingPatterns(memories, activePolicies, cfg.MeasurementWindow)

	maintenanceCfg := MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: cfg.MaintenanceMinOutcomes,
		MinSuccessRate:      cfg.MaintenanceMinSuccess,
	}
	maintenanceProposals := AnalyzeMaintenanceOutcomes(measurableRecords, maintenanceCfg)

	evalResult := EvaluateActivePolicies(memories, activePolicies, cfg.MeasurementWindow)

	proposals := make([]policy.Policy, 0,
		len(contentProposals)+len(structuralProposals)+
			len(surfacingProposals)+len(maintenanceProposals)+
			len(evalResult.RetirementProposals))
	proposals = append(proposals, contentProposals...)
	proposals = append(proposals, structuralProposals...)
	proposals = append(proposals, surfacingProposals...)
	proposals = append(proposals, maintenanceProposals...)
	proposals = append(proposals, evalResult.RetirementProposals...)

	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Evidence.SampleSize > proposals[j].Evidence.SampleSize
	})

	return proposals, evalResult.ValidatedPolicyIDs
}
```

Note: the return type is `([]policy.Policy, []string)`. Update the test to capture both return values:

```go
proposals, _ := adapt.AnalyzeAll(memories, cfg, activePolicies, nil)
```

- [ ] **Step 4: Update runAdaptationAnalysis in cli.go**

Modify `runAdaptationAnalysis` in `internal/cli/cli.go`:

```go
func runAdaptationAnalysis(ctx context.Context, dataDir, policyPath string) {
	const (
		minClusterSize         = 5
		minFeedbackEvents      = 3
		measurementWindow      = 10
		maintenanceMinOutcomes = 3
		minNewFeedback         = 5
	)

	const maintenanceMinSuccess = 0.4

	analysisConfig := adapt.Config{
		MinClusterSize:         minClusterSize,
		MinFeedbackEvents:      minFeedbackEvents,
		MeasurementWindow:      measurementWindow,
		MaintenanceMinOutcomes: maintenanceMinOutcomes,
		MaintenanceMinSuccess:  maintenanceMinSuccess,
		MinNewFeedback:         minNewFeedback,
	}

	allMemories, listErr := retrieve.New().ListMemories(ctx, dataDir)
	if listErr != nil || len(allMemories) == 0 {
		return
	}

	adaptPF, loadErr := policy.Load(policyPath)
	if loadErr != nil {
		return
	}

	// Collect active policies across all dimensions
	activePolicies := make([]policy.Policy, 0)
	for _, pol := range adaptPF.Policies {
		if pol.Status == policy.StatusActive {
			activePolicies = append(activePolicies, pol)
		}
	}

	// Load measurable records (memories with maintenance history)
	measurableRecords := loadMeasurableRecords(allMemories)

	// Run deferred outcome measurement first
	measureResults := adapt.MeasureOutcomes(measurableRecords, minNewFeedback)
	applyMeasureResults(measurableRecords, measureResults, dataDir)

	// Reload measurable records after measurement updates
	allMemories, listErr = retrieve.New().ListMemories(ctx, dataDir)
	if listErr != nil {
		return
	}
	measurableRecords = loadMeasurableRecords(allMemories)

	newProposals, validatedIDs := adapt.AnalyzeAll(
		allMemories, analysisConfig, activePolicies, measurableRecords,
	)

	// Mark validated policies
	for _, validID := range validatedIDs {
		for idx := range adaptPF.Policies {
			if adaptPF.Policies[idx].ID == validID {
				adaptPF.Policies[idx].Effectiveness.Validated = true
				// Fill in after-snapshot
				snap := adapt.ComputeCorpusSnapshot(allMemories)
				adaptPF.Policies[idx].Effectiveness.AfterFollowRate = snap.FollowRate
				adaptPF.Policies[idx].Effectiveness.AfterIrrelevanceRatio = snap.IrrelevanceRatio
				adaptPF.Policies[idx].Effectiveness.AfterMeanEffectiveness = snap.MeanEffectiveness
			}
		}
	}

	for i := range newProposals {
		newProposals[i].ID = adaptPF.NextID()
		newProposals[i].CreatedAt = time.Now().UTC().Format(time.RFC3339)
		adaptPF.Policies = append(adaptPF.Policies, newProposals[i])
	}

	_ = policy.Save(policyPath, adaptPF)
}
```

Add helpers:

```go
// loadMeasurableRecords reads MemoryRecords for memories that have maintenance history.
func loadMeasurableRecords(memories []*memory.Stored) []adapt.MeasurableRecord {
	records := make([]adapt.MeasurableRecord, 0)

	for _, mem := range memories {
		record, err := readRecord(mem.FilePath)
		if err != nil {
			continue
		}

		if len(record.MaintenanceHistory) == 0 {
			continue
		}

		records = append(records, adapt.MeasurableRecord{
			Path:   mem.FilePath,
			Record: *record,
		})
	}

	return records
}

// applyMeasureResults writes measured outcomes back to memory TOML files.
func applyMeasureResults(records []adapt.MeasurableRecord, results []adapt.MeasuredResult, dataDir string) {
	recorder := maintain.NewTOMLHistoryRecorder()

	// Group results by path
	byPath := make(map[string][]adapt.MeasuredResult)
	for _, result := range results {
		byPath[result.Path] = append(byPath[result.Path], result)
	}

	for _, rec := range records {
		pathResults, exists := byPath[rec.Path]
		if !exists {
			continue
		}

		record, err := recorder.ReadRecord(rec.Path)
		if err != nil {
			continue
		}

		for _, result := range pathResults {
			if result.ActionIndex < len(record.MaintenanceHistory) {
				record.MaintenanceHistory[result.ActionIndex].EffectivenessAfter = result.EffectivenessAfter
				record.MaintenanceHistory[result.ActionIndex].SurfacedCountAfter = result.SurfacedCountAfter
				record.MaintenanceHistory[result.ActionIndex].Measured = true
			}
		}

		// Write back using rewriter pattern
		rewriter := maintain.NewTOMLRewriter()
		updates := map[string]any{
			"maintenance_history": record.MaintenanceHistory,
		}
		_ = rewriter.Rewrite(rec.Path, updates)
	}
}
```

- [ ] **Step 5: Fix any compilation issues from AnalyzeAll signature change**

The old call site `adapt.AnalyzeAll(allMemories, analysisConfig)` no longer compiles. The new signature requires 4 args and returns 2 values. Update all call sites.

- [ ] **Step 6: Run tests to verify passes**

Run: `targ test -- -run TestAnalyzeAll -v ./internal/adapt/...`
Expected: PASS

- [ ] **Step 7: Run full check**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 8: Commit**

```
feat: wire measurement backbone into analysis pipeline (#398 #399 #400 #401)

Expands AnalyzeAll to include surfacing pattern analysis, maintenance
outcome analysis, and active policy evaluation. Adds MeasureOutcomes
to deferred measurement pass. Marks validated policies and fills
after-snapshots. Session counting and corpus snapshots close the loop.
```

---

### Task 13: Final Integration — Smoke Test

**Files:**
- No new files

- [ ] **Step 1: Run full build and check**

Run: `targ check-full`
Expected: ALL PASS

- [ ] **Step 2: Build binary**

Run: `targ build`
Expected: Binary builds successfully

- [ ] **Step 3: Smoke test the adapt command**

Run: `./engram adapt -status`
Expected: Shows policies (or "No policies") without error

- [ ] **Step 4: Verify the binary runs surface without error**

Run: `echo "test" | ./engram surface -data-dir /tmp/engram-test -message "test"`
Expected: Runs without panic. Policy session counting is fire-and-forget so no visible output change.

- [ ] **Step 5: Run full check one final time**

Run: `targ check-full`
Expected: ALL PASS
