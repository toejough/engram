# Step 5: Maintain + Adapt + Triage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the stubbed `engram maintain` with an effectiveness-based decision tree, unified proposal system, and Sonnet-powered adapt/consolidation — completing the SBIA pipeline.

**Architecture:** Per-memory health diagnosis via a decision tree (no LLM needed) produces proposals. Consolidation (similar memories) and adapt (parameter tuning) use Sonnet. All proposals land in a pending file; `/memory-triage` skill walks the user through approve/reject. Change history in policy.toml prevents compounding.

**Tech Stack:** Go, BurntSushi/toml, Anthropic API (Sonnet for consolidation/adapt, no LLM for decision tree), gomega tests

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `internal/maintain/proposal.go` | Proposal type, pending file read/write, change history entry type |
| `internal/maintain/proposal_test.go` | Proposal I/O tests |
| `internal/maintain/decisiontree.go` | Per-memory health diagnosis → proposals (pure logic, no LLM) |
| `internal/maintain/decisiontree_test.go` | Decision tree tests |
| `internal/maintain/consolidation.go` | Similar memory detection via Sonnet → merge proposals |
| `internal/maintain/consolidation_test.go` | Consolidation tests |
| `internal/maintain/adapt.go` | Aggregate metric analysis via Sonnet → parameter proposals |
| `internal/maintain/adapt_test.go` | Adapt tests |
| `internal/maintain/maintain_test.go` | Orchestrator tests |
| `internal/cli/maintain.go` | CLI wiring: runMaintain, runApplyProposal, runRejectProposal |
| `internal/cli/maintain_test.go` | CLI maintain tests |

### Modified Files

| File | Changes |
|------|---------|
| `internal/maintain/maintain.go` | Rewrite: orchestrator Run() that calls decision tree + consolidation + adapt |
| `internal/anthropic/anthropic.go` | Add `SonnetModel` constant |
| `internal/policy/policy.go` | Add maintain/adapt thresholds, new prompts, change_history read/write |
| `internal/policy/policy_test.go` | Tests for new fields |
| `internal/cli/cli.go` | Replace `runMaintainStub` dispatch, add `apply-proposal`/`reject-proposal` commands |
| `internal/cli/targets.go` | Update MaintainArgs, add ApplyProposal/RejectProposal to targets (already partially exists) |
| `internal/cli/export_test.go` | Export new CLI functions for testing |
| `internal/cli/targets_test.go` | Update target list assertions |
| `hooks/session-start.sh` | Update maintain output parsing for new proposal schema |
| `skills/memory-triage/SKILL.md` | Rewrite for new proposal-based flow |
| `skills/adapt/adapt.md` | Delete (merged into memory-triage) |
| `docs/superpowers/plans/2026-03-30-sbia-migration-overview.md` | Check off Step 5 items |

---

## Task 1: Policy — Add Maintain/Adapt Fields + Change History

Add the maintain thresholds, adapt settings, new prompts, and change history support to the policy package.

**Files:**
- Modify: `internal/anthropic/anthropic.go:14-17`
- Modify: `internal/policy/policy.go`
- Test: `internal/policy/policy_test.go`

- [ ] **Step 1: Add SonnetModel constant**

In `internal/anthropic/anthropic.go`, add to the exported constants block:

```go
const (
	HaikuModel  = "claude-haiku-4-5-20251001"
	SonnetModel = "claude-sonnet-4-6-20250514"
)
```

- [ ] **Step 2: Write failing test for new policy fields**

Add a test in `internal/policy/policy_test.go` that verifies `Defaults()` returns the new maintain/adapt fields:

```go
func TestDefaults_MaintainFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pol := policy.Defaults()

	g.Expect(pol.MaintainEffectivenessThreshold).To(Equal(50.0))
	g.Expect(pol.MaintainMinSurfaced).To(Equal(5))
	g.Expect(pol.MaintainIrrelevanceThreshold).To(Equal(60.0))
	g.Expect(pol.MaintainNotFollowedThreshold).To(Equal(50.0))
	g.Expect(pol.AdaptChangeHistoryLimit).To(Equal(50))
	g.Expect(pol.MaintainRewritePrompt).NotTo(BeEmpty())
	g.Expect(pol.MaintainConsolidatePrompt).NotTo(BeEmpty())
	g.Expect(pol.AdaptSonnetPrompt).NotTo(BeEmpty())
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `targ test -- -run TestDefaults_MaintainFields ./internal/policy/...`
Expected: FAIL — fields don't exist on Policy struct

- [ ] **Step 4: Add maintain/adapt fields to Policy struct, defaults, and file parsing**

In `internal/policy/policy.go`:

Add to `Policy` struct:
```go
	// MaintainEffectivenessThreshold is the minimum effectiveness % to consider a memory "working".
	MaintainEffectivenessThreshold float64

	// MaintainMinSurfaced is the minimum surfaced_count before diagnosis applies.
	MaintainMinSurfaced int

	// MaintainIrrelevanceThreshold is the irrelevant_rate % that triggers narrowing.
	MaintainIrrelevanceThreshold float64

	// MaintainNotFollowedThreshold is the not_followed_rate % that triggers action rewriting.
	MaintainNotFollowedThreshold float64

	// AdaptChangeHistoryLimit is the max entries retained in [[change_history]].
	AdaptChangeHistoryLimit int

	// MaintainRewritePrompt is the system prompt for Sonnet when rewriting situation or action fields.
	MaintainRewritePrompt string

	// MaintainConsolidatePrompt is the system prompt for Sonnet when synthesising similar memories.
	MaintainConsolidatePrompt string

	// AdaptSonnetPrompt is the system prompt for Sonnet when analysing aggregate metrics.
	AdaptSonnetPrompt string
```

Add default constants:
```go
	defaultMaintainEffectivenessThreshold = 50.0
	defaultMaintainMinSurfaced            = 5
	defaultMaintainIrrelevanceThreshold   = 60.0
	defaultMaintainNotFollowedThreshold   = 50.0
	defaultAdaptChangeHistoryLimit        = 50
	defaultMaintainRewritePrompt          = `You are rewriting a memory field for an AI assistant's memory system.

Given the original memory and the diagnosis, rewrite the specified field to be more precise.
If rewriting "situation", make it narrower and more specific.
If rewriting "action", make it clearer and more actionable.

Return ONLY the rewritten text. No explanation.`
	defaultMaintainConsolidatePrompt = `You are consolidating similar memories for an AI assistant's memory system.

Given multiple memories with overlapping situations, synthesize them into a single memory
that captures the essential guidance from all of them.

Return a JSON object with fields: situation, behavior, impact, action.
No explanation outside the JSON.`
	defaultAdaptSonnetPrompt = `You are tuning parameters for an AI assistant's memory system.

Given the current parameters, aggregate metrics across all memories, and recent change history,
propose parameter or prompt adjustments that would improve system performance.

Return a JSON array of proposals, each with:
- "field": the parameter or prompt name to change
- "value": the new value (number or string)
- "rationale": why this change would help

Return an empty array if no changes are needed.`
```

Add to `Defaults()`:
```go
		MaintainEffectivenessThreshold: defaultMaintainEffectivenessThreshold,
		MaintainMinSurfaced:            defaultMaintainMinSurfaced,
		MaintainIrrelevanceThreshold:   defaultMaintainIrrelevanceThreshold,
		MaintainNotFollowedThreshold:   defaultMaintainNotFollowedThreshold,
		AdaptChangeHistoryLimit:         defaultAdaptChangeHistoryLimit,
		MaintainRewritePrompt:          defaultMaintainRewritePrompt,
		MaintainConsolidatePrompt:      defaultMaintainConsolidatePrompt,
		AdaptSonnetPrompt:              defaultAdaptSonnetPrompt,
```

Add to `policyFileParams`:
```go
	MaintainEffectivenessThreshold float64 `toml:"maintain_effectiveness_threshold"`
	MaintainMinSurfaced            int     `toml:"maintain_min_surfaced"`
	MaintainIrrelevanceThreshold   float64 `toml:"maintain_irrelevance_threshold"`
	MaintainNotFollowedThreshold   float64 `toml:"maintain_not_followed_threshold"`
	AdaptChangeHistoryLimit        int     `toml:"adapt_change_history_limit"`
```

Add to `policyFilePrompts`:
```go
	MaintainRewrite     string `toml:"maintain_rewrite"`
	MaintainConsolidate string `toml:"maintain_consolidate"`
	AdaptSonnet         string `toml:"adapt_sonnet"`
```

Add merge logic in a new `mergeMaintainParams` function (called from `mergeParams`):
```go
func mergeMaintainParams(pol *Policy, params policyFileParams) {
	if params.MaintainEffectivenessThreshold != 0 {
		pol.MaintainEffectivenessThreshold = params.MaintainEffectivenessThreshold
	}
	if params.MaintainMinSurfaced != 0 {
		pol.MaintainMinSurfaced = params.MaintainMinSurfaced
	}
	if params.MaintainIrrelevanceThreshold != 0 {
		pol.MaintainIrrelevanceThreshold = params.MaintainIrrelevanceThreshold
	}
	if params.MaintainNotFollowedThreshold != 0 {
		pol.MaintainNotFollowedThreshold = params.MaintainNotFollowedThreshold
	}
	if params.AdaptChangeHistoryLimit != 0 {
		pol.AdaptChangeHistoryLimit = params.AdaptChangeHistoryLimit
	}
}
```

Add merge logic for new prompts in `mergePrompts`:
```go
	if prompts.MaintainRewrite != "" {
		pol.MaintainRewritePrompt = prompts.MaintainRewrite
	}
	if prompts.MaintainConsolidate != "" {
		pol.MaintainConsolidatePrompt = prompts.MaintainConsolidate
	}
	if prompts.AdaptSonnet != "" {
		pol.AdaptSonnetPrompt = prompts.AdaptSonnet
	}
```

- [ ] **Step 5: Write failing test for change history read/write**

```go
func TestChangeHistory_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	entry := policy.ChangeEntry{
		Action:    "update",
		Target:    "memories/foo.toml",
		Field:     "situation",
		OldValue:  "old text",
		NewValue:  "new text",
		Status:    "approved",
		Rationale: "test rationale",
		ChangedAt: "2026-03-31T10:00:00Z",
	}

	err := policy.AppendChangeHistory(path, entry, os.ReadFile, os.WriteFile)
	g.Expect(err).NotTo(HaveOccurred())

	entries, readErr := policy.ReadChangeHistory(path, os.ReadFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Action).To(Equal("update"))
	g.Expect(entries[0].Target).To(Equal("memories/foo.toml"))
}
```

- [ ] **Step 6: Implement ChangeEntry type and read/write functions**

Add to `internal/policy/policy.go` (or a new `internal/policy/changehistory.go`):

```go
// ChangeEntry records an applied or rejected proposal in policy.toml.
type ChangeEntry struct {
	Action    string `toml:"action"`
	Target    string `toml:"target"`
	Field     string `toml:"field,omitempty"`
	OldValue  string `toml:"old_value,omitempty"`
	NewValue  string `toml:"new_value,omitempty"`
	Status    string `toml:"status"`
	Rationale string `toml:"rationale"`
	ChangedAt string `toml:"changed_at"`
}

// ReadFileFunc reads a file by path and returns its contents.
// (already defined — reuse)

// WriteFileFunc writes data to a file.
type WriteFileFunc func(path string, data []byte, perm os.FileMode) error

// ReadChangeHistory reads [[change_history]] entries from a policy TOML file.
func ReadChangeHistory(path string, readFile func(string) ([]byte, error)) ([]ChangeEntry, error) {
	data, err := readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading change history: %w", err)
	}

	var file struct {
		ChangeHistory []ChangeEntry `toml:"change_history"`
	}
	_, decErr := toml.Decode(string(data), &file)
	if decErr != nil {
		return nil, fmt.Errorf("parsing change history: %w", decErr)
	}
	return file.ChangeHistory, nil
}

// AppendChangeHistory appends a ChangeEntry to [[change_history]] in policy.toml.
// Creates the file if it doesn't exist. Trims oldest entries beyond the default limit.
func AppendChangeHistory(
	path string,
	entry ChangeEntry,
	readFile func(string) ([]byte, error),
	writeFile WriteFileFunc,
) error {
	// Read existing file content (preserve [parameters] and [prompts] sections).
	data, err := readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading policy for change history: %w", err)
	}

	var file policyFile
	if len(data) > 0 {
		_, decErr := toml.Decode(string(data), &file)
		if decErr != nil {
			return fmt.Errorf("parsing policy for change history: %w", decErr)
		}
	}

	file.ChangeHistory = append(file.ChangeHistory, entry)

	// Trim to limit (use default if not set in params).
	limit := defaultAdaptChangeHistoryLimit
	if file.Parameters.AdaptChangeHistoryLimit > 0 {
		limit = file.Parameters.AdaptChangeHistoryLimit
	}
	if len(file.ChangeHistory) > limit {
		file.ChangeHistory = file.ChangeHistory[len(file.ChangeHistory)-limit:]
	}

	var buf bytes.Buffer
	encErr := toml.NewEncoder(&buf).Encode(file)
	if encErr != nil {
		return fmt.Errorf("encoding policy: %w", encErr)
	}

	const filePerms = 0o644
	return writeFile(path, buf.Bytes(), filePerms)
}
```

Update `policyFile` struct to include:
```go
type policyFile struct {
	Parameters    policyFileParams    `toml:"parameters"`
	Prompts       policyFilePrompts   `toml:"prompts"`
	ChangeHistory []ChangeEntry       `toml:"change_history"`
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `targ test -- ./internal/policy/...`
Expected: All PASS

- [ ] **Step 8: Run full check**

Run: `targ check-full`
Fix any lint/coverage issues.

- [ ] **Step 9: Commit**

```
feat(policy): add maintain/adapt thresholds, prompts, and change history (Step 5 S1)
```

---

## Task 2: Proposal Type + Pending File I/O

Define the unified proposal schema and file I/O for the pending proposals file.

**Files:**
- Create: `internal/maintain/proposal.go`
- Create: `internal/maintain/proposal_test.go`

- [ ] **Step 1: Write failing test for Proposal struct and file I/O**

```go
package maintain_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
)

func TestWriteAndReadProposals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "pending-proposals.json")

	proposals := []maintain.Proposal{
		{
			ID:        "prop-001",
			Action:    maintain.ActionDelete,
			Target:    "memories/foo.toml",
			Rationale: "Failing both thresholds",
		},
		{
			ID:        "prop-002",
			Action:    maintain.ActionUpdate,
			Target:    "memories/bar.toml",
			Field:     "situation",
			Value:     "Narrower situation text",
			Rationale: "High irrelevant rate",
		},
	}

	err := maintain.WriteProposals(path, proposals, os.WriteFile)
	g.Expect(err).NotTo(HaveOccurred())

	read, readErr := maintain.ReadProposals(path, os.ReadFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}
	g.Expect(read).To(HaveLen(2))
	g.Expect(read[0].ID).To(Equal("prop-001"))
	g.Expect(read[0].Action).To(Equal(maintain.ActionDelete))
	g.Expect(read[1].Field).To(Equal("situation"))
	g.Expect(read[1].Value).To(Equal("Narrower situation text"))
}

func TestReadProposals_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	proposals, err := maintain.ReadProposals("/nonexistent/path.json", os.ReadFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestWriteAndReadProposals ./internal/maintain/...`
Expected: FAIL — types/functions don't exist

- [ ] **Step 3: Implement Proposal type and I/O**

Create `internal/maintain/proposal.go`:

```go
package maintain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Action constants for proposal types.
const (
	ActionUpdate    = "update"
	ActionDelete    = "delete"
	ActionMerge     = "merge"
	ActionRecommend = "recommend"
)

// Proposal represents a unified maintenance or tuning action.
type Proposal struct {
	ID        string   `json:"id"`
	Action    string   `json:"action"`
	Target    string   `json:"target"`
	Field     string   `json:"field,omitempty"`
	Value     string   `json:"value,omitempty"`
	Related   []string `json:"related,omitempty"`
	Rationale string   `json:"rationale"`
}

// ReadFileFunc reads a file by path.
type ReadFileFunc func(string) ([]byte, error)

// WriteFileFunc writes data to a file.
type WriteFileFunc func(string, []byte, os.FileMode) error

// ReadProposals reads proposals from a JSON file. Returns empty slice if file doesn't exist.
func ReadProposals(path string, readFile ReadFileFunc) ([]Proposal, error) {
	data, err := readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading proposals: %w", err)
	}

	var proposals []Proposal
	if decErr := json.Unmarshal(data, &proposals); decErr != nil {
		return nil, fmt.Errorf("parsing proposals: %w", decErr)
	}
	return proposals, nil
}

// WriteProposals writes proposals to a JSON file atomically.
func WriteProposals(path string, proposals []Proposal, writeFile WriteFileFunc) error {
	data, err := json.MarshalIndent(proposals, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding proposals: %w", err)
	}

	const filePerms = 0o644
	if writeErr := writeFile(path, data, filePerms); writeErr != nil {
		return fmt.Errorf("writing proposals: %w", writeErr)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(maintain): add Proposal type and pending file I/O (Step 5 S2)
```

---

## Task 3: Decision Tree — Per-Memory Health Diagnosis

Pure logic that computes derived metrics and maps each memory to a proposal (or skip).

**Files:**
- Create: `internal/maintain/decisiontree.go`
- Create: `internal/maintain/decisiontree_test.go`

- [ ] **Step 1: Write failing tests for decision tree**

```go
package maintain_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestDiagnose_InsufficientData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := memory.MemoryRecord{SurfacedCount: 3}
	cfg := maintain.DiagnosisConfig{MinSurfaced: 5}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).To(BeNil()) // skip — insufficient data
}

func TestDiagnose_Remove(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    1,
		NotFollowedCount: 2,
		IrrelevantCount:  7,
	}
	cfg := maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 50.0,
		IrrelevanceThreshold:   60.0,
		NotFollowedThreshold:   50.0,
	}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).NotTo(BeNil())
	if result == nil {
		return
	}
	g.Expect(result.Action).To(Equal(maintain.ActionDelete))
	g.Expect(result.Rationale).To(ContainSubstring("effectiveness"))
}

func TestDiagnose_NarrowSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// High irrelevance but effectiveness is OK
	record := memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    3,
		NotFollowedCount: 0,
		IrrelevantCount:  7,
	}
	cfg := maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 50.0,
		IrrelevanceThreshold:   60.0,
		NotFollowedThreshold:   50.0,
	}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).NotTo(BeNil())
	if result == nil {
		return
	}
	g.Expect(result.Action).To(Equal(maintain.ActionUpdate))
	g.Expect(result.Field).To(Equal("situation"))
}

func TestDiagnose_RewriteAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    3,
		NotFollowedCount: 6,
		IrrelevantCount:  1,
	}
	cfg := maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 50.0,
		IrrelevanceThreshold:   60.0,
		NotFollowedThreshold:   50.0,
	}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).NotTo(BeNil())
	if result == nil {
		return
	}
	g.Expect(result.Action).To(Equal(maintain.ActionUpdate))
	g.Expect(result.Field).To(Equal("action"))
}

func TestDiagnose_Working(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := memory.MemoryRecord{
		SurfacedCount:    10,
		FollowedCount:    8,
		NotFollowedCount: 1,
		IrrelevantCount:  1,
	}
	cfg := maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 50.0,
		IrrelevanceThreshold:   60.0,
		NotFollowedThreshold:   50.0,
	}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).To(BeNil()) // working — no action
}

func TestDiagnose_Recommend_PersistentNotFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// surfaced_count >= 2*MinSurfaced and not_followed_rate >= threshold
	record := memory.MemoryRecord{
		SurfacedCount:    12,
		FollowedCount:    2,
		NotFollowedCount: 8,
		IrrelevantCount:  2,
	}
	cfg := maintain.DiagnosisConfig{
		MinSurfaced:            5,
		EffectivenessThreshold: 50.0,
		IrrelevanceThreshold:   60.0,
		NotFollowedThreshold:   50.0,
	}

	result := maintain.Diagnose("memories/foo.toml", &record, cfg)
	g.Expect(result).NotTo(BeNil())
	if result == nil {
		return
	}
	// Persistent not-followed with enough data → recommend escalation
	g.Expect(result.Action).To(Equal(maintain.ActionRecommend))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestDiagnose ./internal/maintain/...`
Expected: FAIL — Diagnose function doesn't exist

- [ ] **Step 3: Implement decision tree**

Create `internal/maintain/decisiontree.go`:

```go
package maintain

import (
	"fmt"
	"path/filepath"
	"strings"

	"engram/internal/memory"
)

// DiagnosisConfig holds thresholds for the decision tree.
type DiagnosisConfig struct {
	MinSurfaced            int
	EffectivenessThreshold float64
	IrrelevanceThreshold   float64
	NotFollowedThreshold   float64
}

// persistentNotFollowedMultiplier is the surfaced count multiplier
// above which not-followed triggers a recommend (escalation) instead of rewrite.
const persistentNotFollowedMultiplier = 2

// Diagnose evaluates a single memory against the decision tree and returns
// a proposal, or nil if no action is needed (insufficient data or working).
func Diagnose(path string, record *memory.MemoryRecord, cfg DiagnosisConfig) *Proposal {
	// Priority 1: Insufficient data → skip.
	if record.SurfacedCount < cfg.MinSurfaced {
		return nil
	}

	surfaced := float64(record.SurfacedCount)
	effectiveness := float64(record.FollowedCount) / surfaced * 100
	irrelevantRate := float64(record.IrrelevantCount) / surfaced * 100
	notFollowedRate := float64(record.NotFollowedCount) / surfaced * 100
	name := memoryNameFromPath(path)

	// Priority 2: Both failing → remove.
	if effectiveness < cfg.EffectivenessThreshold && irrelevantRate >= cfg.IrrelevanceThreshold {
		return &Proposal{
			ID:     fmt.Sprintf("diag-%s-remove", name),
			Action: ActionDelete,
			Target: path,
			Rationale: fmt.Sprintf(
				"Low effectiveness (%.0f%%) and high irrelevance (%.0f%%)",
				effectiveness, irrelevantRate,
			),
		}
	}

	// Priority 3: High irrelevance → narrow situation.
	if irrelevantRate >= cfg.IrrelevanceThreshold {
		return &Proposal{
			ID:     fmt.Sprintf("diag-%s-narrow", name),
			Action: ActionUpdate,
			Target: path,
			Field:  "situation",
			Rationale: fmt.Sprintf(
				"High irrelevant rate (%.0f%%) — situation too broad",
				irrelevantRate,
			),
		}
	}

	// Priority 4: High not-followed rate.
	if notFollowedRate >= cfg.NotFollowedThreshold {
		// Persistent not-followed (enough data to be confident) → recommend escalation.
		if record.SurfacedCount >= cfg.MinSurfaced*persistentNotFollowedMultiplier {
			return &Proposal{
				ID:     fmt.Sprintf("diag-%s-recommend", name),
				Action: ActionRecommend,
				Target: path,
				Rationale: fmt.Sprintf(
					"Persistent not-followed (%.0f%% over %d surfacings) — consider converting to rule/hook",
					notFollowedRate, record.SurfacedCount,
				),
			}
		}
		// Otherwise → rewrite action.
		return &Proposal{
			ID:     fmt.Sprintf("diag-%s-rewrite-action", name),
			Action: ActionUpdate,
			Target: path,
			Field:  "action",
			Rationale: fmt.Sprintf(
				"High not-followed rate (%.0f%%) — action not compelling or clear",
				notFollowedRate,
			),
		}
	}

	// Priority 5: Working → no action.
	if effectiveness >= cfg.EffectivenessThreshold {
		return nil
	}

	// Priority 6: Ambiguous signal → no action (monitor).
	return nil
}

// DiagnoseAll runs the decision tree on a batch of memory records.
func DiagnoseAll(records []memory.StoredRecord, cfg DiagnosisConfig) []Proposal {
	proposals := make([]Proposal, 0, len(records))
	for _, sr := range records {
		record := sr.Record // copy to get addressable value
		proposal := Diagnose(sr.Path, &record, cfg)
		if proposal != nil {
			proposals = append(proposals, *proposal)
		}
	}
	return proposals
}

// memoryNameFromPath extracts the base filename without extension.
func memoryNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(maintain): add effectiveness decision tree for per-memory diagnosis (Step 5 S3)
```

---

## Task 4: Consolidation Analysis (Sonnet)

Detect similar memories and generate merge proposals via Sonnet.

**Files:**
- Create: `internal/maintain/consolidation.go`
- Create: `internal/maintain/consolidation_test.go`

- [ ] **Step 1: Write failing test for consolidation**

```go
package maintain_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestFindConsolidationCandidates_SimilarMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/use-targ-for-tests.toml",
			Record: memory.MemoryRecord{
				Situation: "When running tests in a Go project",
				Action:    "Use targ test instead of go test",
			},
		},
		{
			Path: "memories/use-targ-for-builds.toml",
			Record: memory.MemoryRecord{
				Situation: "When building a Go project",
				Action:    "Use targ build instead of go build",
			},
		},
		{
			Path: "memories/unrelated.toml",
			Record: memory.MemoryRecord{
				Situation: "When writing commit messages",
				Action:    "Use conventional commits format",
			},
		},
	}

	mockCaller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		return `[{"survivor": "memories/use-targ-for-tests.toml", "members": ["memories/use-targ-for-tests.toml", "memories/use-targ-for-builds.toml"], "rationale": "Both about using targ"}]`, nil
	}

	consolidator := maintain.NewConsolidator(mockCaller, "test prompt")
	proposals, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionMerge))
	g.Expect(proposals[0].Related).To(ContainElement("memories/use-targ-for-builds.toml"))
}

func TestFindConsolidationCandidates_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	consolidator := maintain.NewConsolidator(nil, "unused")
	proposals, err := consolidator.FindMerges(context.Background(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestFindConsolidation ./internal/maintain/...`
Expected: FAIL

- [ ] **Step 3: Implement Consolidator**

Create `internal/maintain/consolidation.go`:

```go
package maintain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// minMemoriesForConsolidation is the minimum memory count to attempt consolidation.
const minMemoriesForConsolidation = 2

// Consolidator finds similar memories and proposes merges via Sonnet.
type Consolidator struct {
	caller         anthropic.CallerFunc
	promptTemplate string
}

// NewConsolidator creates a Consolidator with the given caller and prompt.
func NewConsolidator(caller anthropic.CallerFunc, promptTemplate string) *Consolidator {
	return &Consolidator{caller: caller, promptTemplate: promptTemplate}
}

// mergeCandidate is the JSON structure returned by Sonnet.
type mergeCandidate struct {
	Survivor  string   `json:"survivor"`
	Members   []string `json:"members"`
	Rationale string   `json:"rationale"`
}

// FindMerges asks Sonnet to identify groups of similar memories that could be merged.
func (c *Consolidator) FindMerges(
	ctx context.Context,
	records []memory.StoredRecord,
) ([]Proposal, error) {
	if len(records) < minMemoriesForConsolidation {
		return nil, nil
	}

	userPrompt := buildConsolidationPrompt(records)

	response, err := c.caller(ctx, anthropic.SonnetModel, c.promptTemplate, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("consolidation sonnet call: %w", err)
	}

	var candidates []mergeCandidate
	if decErr := json.Unmarshal([]byte(response), &candidates); decErr != nil {
		return nil, fmt.Errorf("parsing consolidation response: %w", decErr)
	}

	proposals := make([]Proposal, 0, len(candidates))
	for i, cand := range candidates {
		// Related = all members except the survivor.
		related := make([]string, 0, len(cand.Members))
		for _, m := range cand.Members {
			if m != cand.Survivor {
				related = append(related, m)
			}
		}

		proposals = append(proposals, Proposal{
			ID:        fmt.Sprintf("consolidate-%d", i+1),
			Action:    ActionMerge,
			Target:    cand.Survivor,
			Related:   related,
			Rationale: cand.Rationale,
		})
	}

	return proposals, nil
}

// buildConsolidationPrompt formats memory records for the Sonnet prompt.
func buildConsolidationPrompt(records []memory.StoredRecord) string {
	var sb strings.Builder
	sb.WriteString("Memories to analyze for consolidation:\n\n")

	for _, sr := range records {
		fmt.Fprintf(&sb, "Path: %s\nSituation: %s\nAction: %s\n\n",
			sr.Path, sr.Record.Situation, sr.Record.Action)
	}

	sb.WriteString("Return a JSON array of merge groups. Each group has:\n")
	sb.WriteString("- survivor: path of the memory to keep\n")
	sb.WriteString("- members: all paths in the group (including survivor)\n")
	sb.WriteString("- rationale: why they should be merged\n")
	sb.WriteString("Return an empty array if no memories are similar enough to merge.")

	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(maintain): add Sonnet-powered consolidation analysis (Step 5 S4)
```

---

## Task 5: Adapt Analysis (Sonnet)

Analyse aggregate metrics and propose parameter/prompt tuning via Sonnet.

**Files:**
- Create: `internal/maintain/adapt.go`
- Create: `internal/maintain/adapt_test.go`

- [ ] **Step 1: Write failing test for adapt analysis**

```go
package maintain_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAdaptAnalyze_ProposesParameterChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{SurfacedCount: 20, FollowedCount: 5, IrrelevantCount: 12}},
		{Path: "b.toml", Record: memory.MemoryRecord{SurfacedCount: 15, FollowedCount: 3, IrrelevantCount: 10}},
	}

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return `[{"field": "surface_bm25_threshold", "value": "0.25", "rationale": "High irrelevant rate suggests BM25 is surfacing weak candidates"}]`, nil
	}

	adapter := maintain.NewAdapter(mockCaller, "test prompt")
	proposals, err := adapter.Analyze(context.Background(), records, policy.Defaults(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposals[0].Target).To(Equal("policy.toml"))
	g.Expect(proposals[0].Field).To(Equal("surface_bm25_threshold"))
}

func TestAdaptAnalyze_EmptyRecords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := maintain.NewAdapter(nil, "unused")
	proposals, err := adapter.Analyze(context.Background(), nil, policy.Defaults(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestAdaptAnalyze ./internal/maintain/...`
Expected: FAIL

- [ ] **Step 3: Implement Adapter**

Create `internal/maintain/adapt.go`:

```go
package maintain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
	"engram/internal/policy"
)

// minMemoriesForAdapt is the minimum memory count to attempt parameter tuning.
const minMemoriesForAdapt = 1

// Adapter analyses aggregate metrics and proposes parameter tuning via Sonnet.
type Adapter struct {
	caller         anthropic.CallerFunc
	promptTemplate string
}

// NewAdapter creates an Adapter with the given caller and prompt.
func NewAdapter(caller anthropic.CallerFunc, promptTemplate string) *Adapter {
	return &Adapter{caller: caller, promptTemplate: promptTemplate}
}

// adaptProposal is the JSON structure returned by Sonnet for parameter changes.
type adaptProposal struct {
	Field     string `json:"field"`
	Value     string `json:"value"`
	Rationale string `json:"rationale"`
}

// Analyze asks Sonnet to propose parameter/prompt adjustments based on aggregate metrics.
func (a *Adapter) Analyze(
	ctx context.Context,
	records []memory.StoredRecord,
	pol policy.Policy,
	changeHistory []policy.ChangeEntry,
) ([]Proposal, error) {
	if len(records) < minMemoriesForAdapt {
		return nil, nil
	}

	userPrompt := buildAdaptPrompt(records, pol, changeHistory)

	response, err := a.caller(ctx, anthropic.SonnetModel, a.promptTemplate, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("adapt sonnet call: %w", err)
	}

	var raw []adaptProposal
	if decErr := json.Unmarshal([]byte(response), &raw); decErr != nil {
		return nil, fmt.Errorf("parsing adapt response: %w", decErr)
	}

	proposals := make([]Proposal, 0, len(raw))
	for i, ap := range raw {
		proposals = append(proposals, Proposal{
			ID:        fmt.Sprintf("adapt-%d", i+1),
			Action:    ActionUpdate,
			Target:    "policy.toml",
			Field:     ap.Field,
			Value:     ap.Value,
			Rationale: ap.Rationale,
		})
	}

	return proposals, nil
}

// buildAdaptPrompt formats aggregate metrics for the Sonnet prompt.
func buildAdaptPrompt(
	records []memory.StoredRecord,
	pol policy.Policy,
	changeHistory []policy.ChangeEntry,
) string {
	var sb strings.Builder

	// Aggregate metrics.
	var totalSurfaced, totalFollowed, totalNotFollowed, totalIrrelevant int
	for _, sr := range records {
		totalSurfaced += sr.Record.SurfacedCount
		totalFollowed += sr.Record.FollowedCount
		totalNotFollowed += sr.Record.NotFollowedCount
		totalIrrelevant += sr.Record.IrrelevantCount
	}

	fmt.Fprintf(&sb, "Aggregate metrics across %d memories:\n", len(records))
	fmt.Fprintf(&sb, "- Total surfaced: %d\n", totalSurfaced)
	fmt.Fprintf(&sb, "- Total followed: %d\n", totalFollowed)
	fmt.Fprintf(&sb, "- Total not followed: %d\n", totalNotFollowed)
	fmt.Fprintf(&sb, "- Total irrelevant: %d\n\n", totalIrrelevant)

	fmt.Fprintf(&sb, "Current parameters:\n")
	fmt.Fprintf(&sb, "- surface_bm25_threshold: %.2f\n", pol.SurfaceBM25Threshold)
	fmt.Fprintf(&sb, "- surface_candidate_count_min: %d\n", pol.SurfaceCandidateCountMin)
	fmt.Fprintf(&sb, "- surface_candidate_count_max: %d\n", pol.SurfaceCandidateCountMax)
	fmt.Fprintf(&sb, "- surface_cold_start_budget: %d\n", pol.SurfaceColdStartBudget)
	fmt.Fprintf(&sb, "- extract_bm25_threshold: %.2f\n", pol.ExtractBM25Threshold)
	fmt.Fprintf(&sb, "- maintain_effectiveness_threshold: %.1f\n", pol.MaintainEffectivenessThreshold)
	fmt.Fprintf(&sb, "- maintain_irrelevance_threshold: %.1f\n", pol.MaintainIrrelevanceThreshold)
	fmt.Fprintf(&sb, "- maintain_not_followed_threshold: %.1f\n\n", pol.MaintainNotFollowedThreshold)

	if len(changeHistory) > 0 {
		fmt.Fprintf(&sb, "Recent change history (%d entries):\n", len(changeHistory))
		for _, entry := range changeHistory {
			fmt.Fprintf(&sb, "- %s %s.%s: %s → %s (%s) at %s\n",
				entry.Status, entry.Target, entry.Field,
				entry.OldValue, entry.NewValue, entry.Rationale, entry.ChangedAt)
		}
	}

	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(maintain): add Sonnet-powered adapt analysis for parameter tuning (Step 5 S5)
```

---

## Task 6: Maintain Orchestrator

Wire decision tree, consolidation, and adapt into a single `Run()` function.

**Files:**
- Modify: `internal/maintain/maintain.go`
- Create: `internal/maintain/maintain_test.go`

- [ ] **Step 1: Write failing test for orchestrator**

```go
package maintain_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestRun_ProducesProposals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Set up memory files in a temp dir.
	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	// A memory with high irrelevance → should get narrow-situation proposal.
	highIrrelevance := `situation = "When doing anything"
behavior = "Too broad"
impact = "Surfaces everywhere"
action = "Narrow this down"
surfaced_count = 10
followed_count = 3
not_followed_count = 0
irrelevant_count = 7
`
	g.Expect(os.WriteFile(filepath.Join(memDir, "broad.toml"), []byte(highIrrelevance), 0o644)).To(Succeed())

	// A working memory → should not produce a proposal.
	working := `situation = "When writing Go tests"
behavior = "Missing t.Parallel"
impact = "Slow tests"
action = "Add t.Parallel to every test"
surfaced_count = 10
followed_count = 9
not_followed_count = 1
irrelevant_count = 0
`
	g.Expect(os.WriteFile(filepath.Join(memDir, "working.toml"), []byte(working), 0o644)).To(Succeed())

	// No Sonnet caller → consolidation and adapt are skipped.
	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	// Should have exactly 1 proposal (narrow situation on "broad").
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposals[0].Field).To(Equal("situation"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRun_ProducesProposals ./internal/maintain/...`
Expected: FAIL

- [ ] **Step 3: Rewrite maintain orchestrator**

Replace `internal/maintain/maintain.go` contents:

```go
// Package maintain provides memory health diagnosis, consolidation, and tuning.
package maintain

import (
	"context"
	"errors"
	"fmt"

	"engram/internal/anthropic"
	"engram/internal/memory"
	"engram/internal/policy"
)

// Exported variables.
var (
	ErrUserQuit = errors.New("user quit")
)

// Confirmer prompts the user for confirmation during maintenance operations.
type Confirmer interface {
	Confirm(prompt string) (bool, error)
}

// Config holds the dependencies for a maintain run.
type Config struct {
	Policy        policy.Policy
	DataDir       string
	Caller        anthropic.CallerFunc // nil = skip Sonnet-dependent analyses
	ChangeHistory []policy.ChangeEntry
}

// Run executes the full maintain pipeline:
// 1. Read all memories
// 2. Decision tree → per-memory proposals
// 3. Consolidation analysis via Sonnet (if caller provided)
// 4. Adapt analysis via Sonnet (if caller provided)
// Returns the combined proposals.
func Run(ctx context.Context, cfg Config) ([]Proposal, error) {
	memDir := cfg.DataDir + "/memories"

	records, err := memory.ListAll(memDir)
	if err != nil {
		return nil, fmt.Errorf("maintain: listing memories: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	diagCfg := DiagnosisConfig{
		MinSurfaced:            cfg.Policy.MaintainMinSurfaced,
		EffectivenessThreshold: cfg.Policy.MaintainEffectivenessThreshold,
		IrrelevanceThreshold:   cfg.Policy.MaintainIrrelevanceThreshold,
		NotFollowedThreshold:   cfg.Policy.MaintainNotFollowedThreshold,
	}

	proposals := DiagnoseAll(records, diagCfg)

	// Sonnet-dependent analyses require a caller.
	if cfg.Caller != nil {
		consolidator := NewConsolidator(cfg.Caller, cfg.Policy.MaintainConsolidatePrompt)

		mergeProposals, consErr := consolidator.FindMerges(ctx, records)
		if consErr != nil {
			return nil, fmt.Errorf("maintain: consolidation: %w", consErr)
		}

		proposals = append(proposals, mergeProposals...)

		adapter := NewAdapter(cfg.Caller, cfg.Policy.AdaptSonnetPrompt)

		adaptProposals, adaptErr := adapter.Analyze(ctx, records, cfg.Policy, cfg.ChangeHistory)
		if adaptErr != nil {
			return nil, fmt.Errorf("maintain: adapt: %w", adaptErr)
		}

		proposals = append(proposals, adaptProposals...)
	}

	return proposals, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(maintain): wire orchestrator with decision tree, consolidation, and adapt (Step 5 S6)
```

---

## Task 7: CLI — Maintain, Apply-Proposal, Reject-Proposal Commands

Replace the maintain stub and add apply-proposal/reject-proposal commands.

**Files:**
- Create: `internal/cli/maintain.go`
- Create: `internal/cli/maintain_test.go`
- Modify: `internal/cli/cli.go:57-83` (dispatch switch)
- Modify: `internal/cli/cli.go:557-572` (remove runMaintainStub)
- Modify: `internal/cli/export_test.go`
- Modify: `internal/cli/targets.go` (update MaintainArgs, add RejectProposalArgs)
- Modify: `internal/cli/targets_test.go`

- [ ] **Step 1: Write failing test for runMaintain**

In `internal/cli/maintain_test.go`:

```go
package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/maintain"
)

func TestRunMaintain_ProducesJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	// Memory with high irrelevance.
	mem := `situation = "When doing anything"
behavior = "Too broad"
impact = "Surfaces everywhere"
action = "Narrow this down"
surfaced_count = 10
followed_count = 3
not_followed_count = 0
irrelevant_count = 7
`
	g.Expect(os.WriteFile(filepath.Join(memDir, "broad.toml"), []byte(mem), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunMaintainWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		nil, // no caller → skip Sonnet
	)
	g.Expect(err).NotTo(HaveOccurred())

	var proposals []maintain.Proposal
	g.Expect(json.Unmarshal(stdout.Bytes(), &proposals)).To(Succeed())
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionUpdate))
}
```

- [ ] **Step 2: Write failing test for runApplyProposal**

```go
func TestRunApplyProposal_DeletesMemory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	memPath := filepath.Join(memDir, "doomed.toml")
	g.Expect(os.WriteFile(memPath, []byte(`situation = "test"
behavior = "test"
impact = "test"
action = "test"
`), 0o644)).To(Succeed())

	// Write a pending proposal for delete.
	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	proposals := []maintain.Proposal{
		{ID: "prop-001", Action: "delete", Target: memPath, Rationale: "test"},
	}
	data, _ := json.Marshal(proposals)
	g.Expect(os.WriteFile(proposalPath, data, 0o644)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-001"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Memory file should be gone.
	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestRunApplyProposal_UpdatesField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	memPath := filepath.Join(memDir, "broad.toml")
	g.Expect(os.WriteFile(memPath, []byte(`situation = "When doing anything"
behavior = "test"
impact = "test"
action = "test"
`), 0o644)).To(Succeed())

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	proposals := []maintain.Proposal{
		{
			ID: "prop-002", Action: "update", Target: memPath,
			Field: "situation", Value: "When writing Go tests",
			Rationale: "narrowing",
		},
	}
	data, _ := json.Marshal(proposals)
	g.Expect(os.WriteFile(proposalPath, data, 0o644)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-002"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Memory should have updated situation.
	content, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("When writing Go tests"))
}
```

- [ ] **Step 3: Write failing test for runRejectProposal**

```go
func TestRunRejectProposal_LogsRejection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	proposals := []maintain.Proposal{
		{ID: "prop-003", Action: "delete", Target: "memories/x.toml", Rationale: "test"},
	}
	data, _ := json.Marshal(proposals)
	g.Expect(os.WriteFile(proposalPath, data, 0o644)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.ExportRunRejectProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-003"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Change history should contain rejection.
	policyPath := filepath.Join(dataDir, "policy.toml")
	content, readErr := os.ReadFile(policyPath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("rejected"))
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `targ test -- -run "TestRunMaintain|TestRunApplyProposal|TestRunRejectProposal" ./internal/cli/...`
Expected: FAIL

- [ ] **Step 5: Implement CLI maintain commands**

Create `internal/cli/maintain.go`:

```go
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/tomlwriter"
)

// CallerFunc is an alias for anthropic.CallerFunc used by test injection.
type CallerFunc = func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// unexported error variables.
var (
	errApplyProposalMissingID  = errors.New("apply-proposal: --id required")
	errRejectProposalMissingID = errors.New("reject-proposal: --id required")
	errProposalNotFound        = errors.New("proposal not found")
)

// runMaintain is the public entry point for the maintain command.
func runMaintain(args []string, stdout io.Writer) error {
	return runMaintainWith(args, stdout, nil)
}

// runMaintainWith runs maintain with an optional caller override for testing.
func runMaintainWith(args []string, stdout io.Writer, callerOverride CallerFunc) error {
	fs := flag.NewFlagSet("maintain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("maintain: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("maintain: %w", defaultErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		pol = policy.Defaults()
	}

	var caller CallerFunc
	if callerOverride != nil {
		caller = callerOverride
	} else {
		ctx := context.Background()
		token := resolveToken(ctx)
		if token != "" {
			caller = makeAnthropicCaller(token)
		}
	}

	changeHistory, _ := policy.ReadChangeHistory(policyPath, os.ReadFile)

	cfg := maintain.Config{
		Policy:        pol,
		DataDir:       *dataDir,
		Caller:        caller,
		ChangeHistory: changeHistory,
	}

	ctx := context.Background()

	proposals, err := maintain.Run(ctx, cfg)
	if err != nil {
		return fmt.Errorf("maintain: %w", err)
	}

	// Write proposals to pending file.
	pendingPath := filepath.Join(*dataDir, "..", "pending-proposals.json")

	writeErr := maintain.WriteProposals(pendingPath, proposals, os.WriteFile)
	if writeErr != nil {
		return fmt.Errorf("maintain: %w", writeErr)
	}

	// Output JSON to stdout (consumed by session-start.sh).
	encoded, encErr := json.Marshal(proposals)
	if encErr != nil {
		return fmt.Errorf("maintain: encoding: %w", encErr)
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", encoded)

	return nil
}

// runApplyProposal executes a proposal by ID.
func runApplyProposal(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("apply-proposal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	proposalID := fs.String("id", "", "proposal ID to apply")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("apply-proposal: %w", parseErr)
	}

	if *proposalID == "" {
		return errApplyProposalMissingID
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("apply-proposal: %w", defaultErr)
	}

	pendingPath := filepath.Join(*dataDir, "..", "pending-proposals.json")

	proposals, readErr := maintain.ReadProposals(pendingPath, os.ReadFile)
	if readErr != nil {
		return fmt.Errorf("apply-proposal: %w", readErr)
	}

	proposal, remaining := findAndRemoveProposal(proposals, *proposalID)
	if proposal == nil {
		return fmt.Errorf("apply-proposal: %w: %s", errProposalNotFound, *proposalID)
	}

	if applyErr := executeProposal(*proposal, *dataDir); applyErr != nil {
		return fmt.Errorf("apply-proposal: %w", applyErr)
	}

	// Log to change history.
	policyPath := filepath.Join(*dataDir, "policy.toml")
	entry := policy.ChangeEntry{
		Action:    proposal.Action,
		Target:    proposal.Target,
		Field:     proposal.Field,
		NewValue:  proposal.Value,
		Status:    "approved",
		Rationale: proposal.Rationale,
		ChangedAt: time.Now().UTC().Format(time.RFC3339),
	}

	_ = policy.AppendChangeHistory(policyPath, entry, os.ReadFile, os.WriteFile)

	// Remove from pending.
	_ = maintain.WriteProposals(pendingPath, remaining, os.WriteFile)

	_, _ = fmt.Fprintf(stdout, "Applied proposal %s: %s %s\n",
		proposal.ID, proposal.Action, proposal.Target)

	return nil
}

// runRejectProposal records a rejection and removes the proposal.
func runRejectProposal(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("reject-proposal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	proposalID := fs.String("id", "", "proposal ID to reject")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("reject-proposal: %w", parseErr)
	}

	if *proposalID == "" {
		return errRejectProposalMissingID
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("reject-proposal: %w", defaultErr)
	}

	pendingPath := filepath.Join(*dataDir, "..", "pending-proposals.json")

	proposals, readErr := maintain.ReadProposals(pendingPath, os.ReadFile)
	if readErr != nil {
		return fmt.Errorf("reject-proposal: %w", readErr)
	}

	proposal, remaining := findAndRemoveProposal(proposals, *proposalID)
	if proposal == nil {
		return fmt.Errorf("reject-proposal: %w: %s", errProposalNotFound, *proposalID)
	}

	// Log rejection to change history.
	policyPath := filepath.Join(*dataDir, "policy.toml")
	entry := policy.ChangeEntry{
		Action:    proposal.Action,
		Target:    proposal.Target,
		Field:     proposal.Field,
		Status:    "rejected",
		Rationale: proposal.Rationale,
		ChangedAt: time.Now().UTC().Format(time.RFC3339),
	}

	_ = policy.AppendChangeHistory(policyPath, entry, os.ReadFile, os.WriteFile)

	// Remove from pending.
	_ = maintain.WriteProposals(pendingPath, remaining, os.WriteFile)

	_, _ = fmt.Fprintf(stdout, "Rejected proposal %s: %s %s\n",
		proposal.ID, proposal.Action, proposal.Target)

	return nil
}

// executeProposal carries out the action described by a proposal.
func executeProposal(p maintain.Proposal, dataDir string) error {
	switch p.Action {
	case maintain.ActionDelete:
		if rmErr := os.Remove(p.Target); rmErr != nil {
			return fmt.Errorf("deleting %s: %w", p.Target, rmErr)
		}
		return nil

	case maintain.ActionUpdate:
		if p.Target == "policy.toml" {
			return applyPolicyUpdate(dataDir, p.Field, p.Value)
		}
		modifier := memory.NewModifier(memory.WithModifierWriter(tomlwriter.New()))
		return modifier.ReadModifyWrite(p.Target, func(record *memory.MemoryRecord) {
			applyFieldUpdate(record, p.Field, p.Value)
		})

	case maintain.ActionMerge:
		// Merge: overwrite survivor with synthesised content, archive related.
		// For now, merge is a recommend-only action handled via the skill.
		return nil

	case maintain.ActionRecommend:
		// Recommend is informational — no file changes.
		return nil

	default:
		return fmt.Errorf("unknown action: %s", p.Action)
	}
}

// applyFieldUpdate sets a field on a MemoryRecord by name.
func applyFieldUpdate(record *memory.MemoryRecord, field, value string) {
	switch field {
	case "situation":
		record.Situation = value
	case "behavior":
		record.Behavior = value
	case "impact":
		record.Impact = value
	case "action":
		record.Action = value
	}

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

// applyPolicyUpdate updates a single field in policy.toml.
func applyPolicyUpdate(dataDir, field, value string) error {
	// Policy updates are handled by writing the field directly.
	// For now, this is a no-op placeholder — the adapt flow is informational.
	_ = dataDir
	_ = field
	_ = value

	return nil
}

// findAndRemoveProposal finds a proposal by ID and returns it plus the remaining list.
func findAndRemoveProposal(proposals []maintain.Proposal, id string) (*maintain.Proposal, []maintain.Proposal) {
	remaining := make([]maintain.Proposal, 0, len(proposals))

	var found *maintain.Proposal

	for i, p := range proposals {
		if p.ID == id {
			found = &proposals[i]

			continue
		}

		remaining = append(remaining, p)
	}

	return found, remaining
}
```

- [ ] **Step 6: Update cli.go dispatch**

In `internal/cli/cli.go`, replace the `"maintain"` case and `runMaintainStub`:

Replace:
```go
	case "maintain":
		return runMaintainStub(subArgs, stdout)
```
With:
```go
	case "maintain":
		return runMaintain(subArgs, stdout)
	case "apply-proposal":
		return runApplyProposal(subArgs, stdout)
	case "reject-proposal":
		return runRejectProposal(subArgs, stdout)
```

Delete the `runMaintainStub` function entirely.

Update the `errUsage` message to include `apply-proposal` and `reject-proposal`.

- [ ] **Step 7: Add exports for testing**

In `internal/cli/export_test.go`, add:

```go
	ExportRunMaintainWith   = runMaintainWith
	ExportRunApplyProposal  = runApplyProposal
	ExportRunRejectProposal = runRejectProposal
```

- [ ] **Step 8: Update targets.go**

Update `MaintainArgs` to the simplified form (remove `Apply`, `Proposals`, `Yes`, `PurgeTierC` — those are old):

```go
type MaintainArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}
```

Update `MaintainFlags`:
```go
func MaintainFlags(a MaintainArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}
```

Add `RejectProposalArgs` and flags:
```go
type RejectProposalArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ID      string `targ:"flag,name=id,desc=proposal ID to reject"`
}

func RejectProposalFlags(a RejectProposalArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--id", a.ID)
}
```

Update `ApplyProposalArgs` to the new form:
```go
type ApplyProposalArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ID      string `targ:"flag,name=id,desc=proposal ID to apply"`
}

func ApplyProposalFlags(a ApplyProposalArgs) []string {
	return BuildFlags("--data-dir", a.DataDir, "--id", a.ID)
}
```

Update `BuildTargets` to register `apply-proposal` and `reject-proposal` with new arg types, and update the maintain target.

- [ ] **Step 9: Update targets_test.go**

Add `"apply-proposal"` and `"reject-proposal"` to the expected command list (and remove any stale entries).

- [ ] **Step 10: Run tests**

Run: `targ test -- ./internal/cli/...`
Expected: PASS

- [ ] **Step 11: Run full check**

Run: `targ check-full`
Fix any lint/coverage issues.

- [ ] **Step 12: Commit**

```
feat(cli): wire maintain, apply-proposal, reject-proposal commands (Step 5 S7)
```

---

## Task 8: Session-Start Hook Update

Update the background hook to parse the new proposal format.

**Files:**
- Modify: `hooks/session-start.sh:72-210`

- [ ] **Step 1: Replace maintain output parsing**

The new `engram maintain` outputs a JSON array of `{id, action, target, field, value, related, rationale}` proposals. Replace the old quadrant-based jq parsing with proposal-action-based parsing.

Replace the jq parsing block (lines ~77-199) with:

```bash
    # UC-28: Run engram maintain — single source of truth for proposals
    echo "Running maintain..."
    SIGNAL_OUTPUT=$("$ENGRAM_BIN" maintain --data-dir "${ENGRAM_HOME}/data") || true
    echo "Maintain output length: ${#SIGNAL_OUTPUT}"

    # Parse maintain proposals (JSON array with id, action, target, rationale)
    PROPOSAL_COUNT=0
    TRIAGE_DETAILS=""
    if [[ -n "$SIGNAL_OUTPUT" ]] && echo "$SIGNAL_OUTPUT" | jq -e 'type == "array" and length > 0' >/dev/null 2>&1; then
        PROPOSAL_COUNT=$(echo "$SIGNAL_OUTPUT" | jq 'length' 2>/dev/null) || PROPOSAL_COUNT=0

        DELETE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "delete")] | length' 2>/dev/null) || DELETE_COUNT=0
        UPDATE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "update" and .target != "policy.toml")] | length' 2>/dev/null) || UPDATE_COUNT=0
        MERGE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "merge")] | length' 2>/dev/null) || MERGE_COUNT=0
        RECOMMEND_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "recommend")] | length' 2>/dev/null) || RECOMMEND_COUNT=0
        ADAPT_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "update" and .target == "policy.toml")] | length' 2>/dev/null) || ADAPT_COUNT=0

        # Build triage details
        DELETE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "delete")] |
            if length == 0 then empty else
                "## Delete (\(length) memories)\nFailing both effectiveness and irrelevance thresholds.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        UPDATE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "update" and .target != "policy.toml")] |
            if length == 0 then empty else
                "## Rewrite (\(length) memories)\nFields need narrowing or clarification.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) [\(.value.field)] — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        MERGE_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "merge")] |
            if length == 0 then empty else
                "## Consolidate (\(length) groups)\nSimilar memories that could be merged.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        RECOMMEND_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "recommend")] |
            if length == 0 then empty else
                "## Escalation (\(length) memories)\nConsider converting to rules/hooks/CLAUDE.md.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.target | split("/") | last | rtrimstr(".toml")) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        ADAPT_DETAIL=$(echo "$SIGNAL_OUTPUT" | jq -r '
            [.[] | select(.action == "update" and .target == "policy.toml")] |
            if length == 0 then empty else
                "## Parameter Tuning (\(length) proposals)\nSystem parameter adjustments suggested.\n" +
                (to_entries | map(
                    "  \(.key + 1). \(.value.field): \(.value.value) — \(.value.rationale)"
                ) | join("\n"))
            end
        ' 2>/dev/null) || true

        for detail in "$DELETE_DETAIL" "$UPDATE_DETAIL" "$MERGE_DETAIL" "$RECOMMEND_DETAIL" "$ADAPT_DETAIL"; do
            if [[ -n "$detail" ]]; then
                TRIAGE_DETAILS="${TRIAGE_DETAILS}
${detail}
"
            fi
        done
    fi

    # Only write pending file if there are proposals
    if [[ "$PROPOSAL_COUNT" -gt 0 ]]; then
        # Build compact counts line
        COUNTS=""
        [[ "${DELETE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${DELETE_COUNT} delete"
        [[ "${UPDATE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${UPDATE_COUNT} rewrite"
        [[ "${MERGE_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${MERGE_COUNT} consolidate"
        [[ "${RECOMMEND_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${RECOMMEND_COUNT} escalation"
        [[ "${ADAPT_COUNT:-0}" -gt 0 ]] && COUNTS="${COUNTS}${COUNTS:+, }${ADAPT_COUNT} parameter tuning"
        DIRECTIVE="[engram] Memory triage: ${COUNTS} pending. Say \"triage\" to review, or ignore to proceed."

        TRIAGE_CTX="[engram] Memory triage details (present interactively if user says 'triage'):
${TRIAGE_DETAILS}
Use the engram:memory-triage skill for commands and presentation format.
Present one category at a time. Ask what the user wants to do with each before moving to the next."
```

- [ ] **Step 2: Verify hook runs without errors**

Run: `bash -n hooks/session-start.sh` (syntax check)
Expected: No errors

- [ ] **Step 3: Commit**

```
refactor(hooks): update session-start.sh for unified proposal format (Step 5 S8)
```

---

## Task 9: Memory-Triage Skill Rewrite + Delete Adapt Skill

Replace the old quadrant-based skill with the new proposal-based flow.

**Files:**
- Modify: `skills/memory-triage/SKILL.md`
- Delete: `skills/adapt/adapt.md`

- [ ] **Step 1: Rewrite memory-triage skill**

Replace `skills/memory-triage/SKILL.md` with:

```markdown
---
name: memory-triage
description: |
  Use when the user asks about "memory management", "engram signals",
  "memory maintenance", "pending recommendations", "triage memories",
  or "clean up memories". Walks through maintenance proposals from
  engram maintain, presenting each for approval or rejection.
---

# Engram Memory Triage

Review and act on maintenance proposals generated by `engram maintain`.

## Proposal Categories (presentation order)

1. **Delete** — memories failing both effectiveness and irrelevance thresholds. "Nothing is working. Remove?"
2. **Narrow situation** — memories exceeding irrelevance threshold. "Surfacing in wrong contexts. Narrow the situation?"
3. **Rewrite action / Recommend escalation** — memories exceeding not-followed threshold. "Surfaced when relevant but not followed. Rewrite the action, or consider converting to a rule/hook/CLAUDE.md entry?"
4. **Consolidate** — memories with similar situations that could be merged.
5. **Parameter tuning** — system parameter adjustments from aggregate analysis.

## How to Present

Group proposals by category. For each category:
- State count and give 2-3 examples with rationale
- Ask what the user wants to do before proceeding to the next category
- For rewrite proposals, show the current field value and the proposed change (if available)

## Executing Decisions

> **WARNING:** NEVER use `rm` to delete memory files or directly edit `.toml` files. Always use the proposal commands below.

| Decision | Command |
|----------|---------|
| Approve a proposal | `engram apply-proposal --id <proposal-id>` |
| Reject a proposal | `engram reject-proposal --id <proposal-id>` |
| Approve all in category | Run `apply-proposal` for each ID |
| Skip category | Run `reject-proposal` for each ID, or skip entirely |

## After All Categories

Summarize what was approved, rejected, and skipped. Note how many proposals remain if any were deferred.
```

- [ ] **Step 2: Delete adapt skill**

Delete `skills/adapt/adapt.md` (and the `skills/adapt/` directory).

- [ ] **Step 3: Commit**

```
refactor(skills): rewrite memory-triage for proposals, delete adapt skill (Step 5 S9)
```

---

## Task 10: Cleanup + Migration Overview

Remove stale code, update docs, and verify everything works.

**Files:**
- Modify: `internal/cli/cli.go` (remove stale imports if any)
- Modify: `internal/cli/targets.go` (remove old AdaptArgs/AdaptFlags if not used by active code)
- Modify: `docs/superpowers/plans/2026-03-30-sbia-migration-overview.md`

- [ ] **Step 1: Remove old adapt command references**

Check if `engram adapt` is still dispatched in `cli.go`. If so, remove it. Remove `AdaptArgs` and `AdaptFlags` from `targets.go` and the adapt target from `BuildTargets`. Update `targets_test.go` accordingly.

- [ ] **Step 2: Run full check**

Run: `targ check-full`
Fix all lint/coverage issues in one pass.

- [ ] **Step 3: Update migration overview**

In `docs/superpowers/plans/2026-03-30-sbia-migration-overview.md`, check off all Step 5 items:

```markdown
### Step 5: Maintain + Adapt + Triage

- [x] Rewrite `engram maintain` to: effectiveness decision tree + consolidation analysis + Sonnet adapt analysis → unified proposals
- [x] Implement `engram apply-proposal <id>` and `engram reject-proposal <id>`
- [x] Add change_history to policy.toml
- [x] Update /memory-triage skill for new proposal flow
- [x] Update session-start.sh background maintain to use new proposal format
- [x] Remove old quadrant analysis, signal packages, policy lifecycle, approval streaks
- [x] **After:** Complete SBIA pipeline operational. All old code removed. All hooks updated.
```

- [ ] **Step 4: Verify binary builds and smoke test**

Run:
```bash
targ build
engram maintain --help
engram apply-proposal --help
engram reject-proposal --help
```

- [ ] **Step 5: Commit**

```
refactor(cleanup): remove stale adapt command, mark Step 5 complete (Step 5 S10)
```

---

## Summary

| Task | What | Sonnet Required |
|------|------|----------------|
| 1 | Policy fields + change history | No |
| 2 | Proposal type + file I/O | No |
| 3 | Decision tree (pure logic) | No |
| 4 | Consolidation analysis | Yes |
| 5 | Adapt analysis | Yes |
| 6 | Maintain orchestrator | No (wires together) |
| 7 | CLI commands | No |
| 8 | Session-start hook | No |
| 9 | Memory-triage skill + delete adapt | No |
| 10 | Cleanup + docs | No |

Tasks 1-3 are independent and can be parallelized. Tasks 4-5 depend on 2 (Proposal type). Task 6 depends on 3-5. Task 7 depends on 6. Tasks 8-10 depend on 7.
