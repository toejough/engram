# Optimize LLM Eval Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Insert Haiku triage and Sonnet behavioral testing between mechanical scanners and human review in the optimize pipeline, move prune/decay to automatic stop hook execution.

**Architecture:** Mechanical scanners generate proposals → Haiku triages judgment-call proposals in parallel (drops false positives) → Sonnet runs behavioral tests on survivors (generates scenarios, simulates before/after context) → human reviews enriched proposals with apply/skip prompt. Prune/decay execute automatically in a stop hook.

**Tech Stack:** Go, Anthropic API (Haiku for triage, Sonnet for behavioral testing), existing DirectAPIExtractor, gomega test assertions, rapid property testing

---

### Task 1: Add LLMEvalResult to MaintenanceProposal

Extend `MaintenanceProposal` to carry LLM evaluation results so the review UI can display them.

**Files:**
- Modify: `internal/memory/optimize.go:76-82`
- Test: `internal/memory/review_ui_test.go`

**Step 1: Write the failing test**

Add to `internal/memory/review_ui_test.go`:

```go
func TestFormatProposal_WithLLMEval(t *testing.T) {
	g := gomega.NewWithT(t)

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  "id1,id2",
		Reason:  "Redundant (similarity 0.92)",
		Preview: "Keep: When managing teams...\nDelete: When using multi-agent teams...",
		LLMEval: &LLMEvalResult{
			HaikuValid:     true,
			HaikuRationale: "Entries share vocabulary but teach different lessons",
			SonnetRecommend: "skip",
			SonnetConfidence: "high",
			SonnetSummary:   "Deleted entry contains actionable advice not in kept entry",
			ScenarioResults: []ScenarioResult{
				{Prompt: "team structure", Preserved: true},
				{Prompt: "idle agents", Preserved: false, Lost: "explicit polling instruction"},
			},
		},
	}

	formatted := formatProposal(proposal)
	g.Expect(formatted).To(gomega.ContainSubstring("Proposed Change:"))
	g.Expect(formatted).To(gomega.ContainSubstring("Haiku:"))
	g.Expect(formatted).To(gomega.ContainSubstring("different lessons"))
	g.Expect(formatted).To(gomega.ContainSubstring("Sonnet recommends: Skip"))
	g.Expect(formatted).To(gomega.ContainSubstring("✓"))
	g.Expect(formatted).To(gomega.ContainSubstring("✗"))
	g.Expect(formatted).To(gomega.ContainSubstring("[a]pply"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestFormatProposal_WithLLMEval -v`
Expected: FAIL — `LLMEvalResult` type undefined

**Step 3: Add types to optimize.go**

Add after `MaintenanceProposal` struct in `internal/memory/optimize.go:82`:

```go
// LLMEvalResult holds the results of LLM evaluation stages.
type LLMEvalResult struct {
	HaikuValid       bool             // Did Haiku consider this a valid concern?
	HaikuRationale   string           // Haiku's one-line explanation
	SonnetRecommend  string           // "apply" or "skip"
	SonnetConfidence string           // "high", "medium", "low"
	SonnetSummary    string           // Human-readable change analysis
	ScenarioResults  []ScenarioResult // Per-scenario preservation checks
}

// ScenarioResult holds one behavioral test scenario result.
type ScenarioResult struct {
	Prompt    string // Simulated user prompt
	Preserved bool   // Did expected guidance surface?
	Lost      string // What was lost (if not preserved)
}
```

Add `LLMEval *LLMEvalResult` field to `MaintenanceProposal`.

**Step 4: Update formatProposal in review_ui.go**

Update `formatProposal()` to render `LLMEval` when present. When `LLMEval` is non-nil:
- Header becomes: `━━━ Proposed Change: {action explanation} ━━━`
- Show "What changes:" section from Preview
- Show "Haiku:" line with rationale
- Show "Sonnet recommends: Apply/Skip this change" with confidence
- Show scenario results with ✓/✗ markers
- Prompt becomes `[a]pply change / [s]kip change`

When `LLMEval` is nil, keep existing format unchanged (backward compatible).

**Step 5: Update reviewProposalCtx to accept a/s**

In `reviewProposalCtx()`, add `"a", "apply"` as aliases for `true` and keep `"y", "yes"`. This way both old and new prompts work.

**Step 6: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestFormatProposal -v`
Expected: ALL PASS

**Step 7: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS — existing format tests still pass (nil LLMEval = old format)

**Step 8: Commit**

```bash
git add internal/memory/optimize.go internal/memory/review_ui.go internal/memory/review_ui_test.go
git commit -m "feat(memory): add LLMEvalResult to MaintenanceProposal and review UI"
```

---

### Task 2: Implement Haiku Triage

Add `TriageProposals()` that sends judgment-call proposals to Haiku in parallel and filters out invalid ones.

**Files:**
- Create: `internal/memory/optimize_llm_eval.go`
- Create: `internal/memory/optimize_llm_eval_test.go`

**Step 1: Write the failing test**

Create `internal/memory/optimize_llm_eval_test.go`:

```go
package memory

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

func TestTriageProposals_FiltersInvalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Mock extractor that returns "invalid" for consolidate proposals
	ext := &mockTriageExtractor{
		responses: map[string]string{
			"consolidate": `{"valid": false, "rationale": "different lessons"}`,
			"promote":     `{"valid": true, "rationale": "good candidate"}`,
		},
	}

	proposals := []MaintenanceProposal{
		{Tier: "embeddings", Action: "consolidate", Target: "id1,id2", Reason: "similarity 0.92"},
		{Tier: "embeddings", Action: "promote", Target: "id3", Reason: "high retrieval"},
		{Tier: "embeddings", Action: "rewrite", Target: "id4", Reason: "clarity"},
	}

	result, err := TriageProposals(context.Background(), proposals, ext, nil)
	g.Expect(err).ToNot(HaveOccurred())

	// consolidate filtered out, promote kept with eval, rewrite passed through (no triage)
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].Action).To(Equal("promote"))
	g.Expect(result[0].LLMEval).ToNot(BeNil())
	g.Expect(result[0].LLMEval.HaikuValid).To(BeTrue())
	g.Expect(result[1].Action).To(Equal("rewrite"))
	g.Expect(result[1].LLMEval).To(BeNil()) // no triage for refinements
}

func TestTriageProposals_PassesThroughMechanical(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &mockTriageExtractor{}
	proposals := []MaintenanceProposal{
		{Tier: "embeddings", Action: "prune", Target: "id1", Reason: "low confidence"},
		{Tier: "embeddings", Action: "decay", Target: "id2", Reason: "stale"},
	}

	result, err := TriageProposals(context.Background(), proposals, ext, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(2))
	// prune/decay pass through without LLM eval
	g.Expect(result[0].LLMEval).To(BeNil())
	g.Expect(result[1].LLMEval).To(BeNil())
}

type mockTriageExtractor struct {
	responses map[string]string
}

func (m *mockTriageExtractor) CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error) {
	// Extract action from the user message to return canned response
	for action, resp := range m.responses {
		if containsAction(params.Messages[0].Content, action) {
			return []byte(resp), nil
		}
	}
	return []byte(`{"valid": true, "rationale": "default"}`), nil
}

func containsAction(msg, action string) bool {
	return len(msg) > 0 && len(action) > 0 &&
		(msg[0:1] != "" && action[0:1] != "") // placeholder — real impl checks content
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestTriageProposals -v`
Expected: FAIL — `TriageProposals` undefined

**Step 3: Implement TriageProposals**

Create `internal/memory/optimize_llm_eval.go`:

```go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
)

// needsLLMTriage returns true for proposal actions that require LLM judgment.
func needsLLMTriage(action string) bool {
	switch action {
	case "consolidate", "promote", "demote", "split":
		return true
	default:
		return false
	}
}

// triageResult holds the JSON output from Haiku triage.
type triageResult struct {
	Valid     bool   `json:"valid"`
	Rationale string `json:"rationale"`
}

// TriageProposals sends judgment-call proposals to Haiku in parallel.
// Returns filtered proposals with LLMEval populated for triaged items.
// Proposals that don't need triage (prune, decay, rewrite, add-rationale) pass through unchanged.
func TriageProposals(ctx context.Context, proposals []MaintenanceProposal, ext APIMessageCaller, progress io.Writer) ([]MaintenanceProposal, error) {
	logf := func(format string, args ...any) {
		if progress != nil {
			fmt.Fprintf(progress, format+"\n", args...)
		}
	}

	// Separate proposals by triage need
	var needsTriage []int
	for i, p := range proposals {
		if needsLLMTriage(p.Action) {
			needsTriage = append(needsTriage, i)
		}
	}

	if len(needsTriage) == 0 {
		return proposals, nil
	}

	logf("- LLM triage: evaluating %d proposals with Haiku...", len(needsTriage))

	// Parallel triage with semaphore
	type indexedResult struct {
		index int
		valid bool
		rationale string
		err   error
	}

	results := make(chan indexedResult, len(needsTriage))
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	for _, idx := range needsTriage {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			p := proposals[i]
			valid, rationale, err := triageOneProposal(ctx, ext, p)
			results <- indexedResult{index: i, valid: valid, rationale: rationale, err: err}
		}(idx)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	triageResults := make(map[int]indexedResult)
	for r := range results {
		triageResults[r.index] = r
	}

	// Build filtered list
	var filtered []MaintenanceProposal
	dropped := 0
	for i, p := range proposals {
		if !needsLLMTriage(p.Action) {
			filtered = append(filtered, p)
			continue
		}

		tr, ok := triageResults[i]
		if !ok || tr.err != nil {
			// On error, keep the proposal (fail open)
			filtered = append(filtered, p)
			continue
		}

		if !tr.valid {
			dropped++
			logf("  dropped: %s %s — %s", p.Action, p.Tier, tr.rationale)
			continue
		}

		p.LLMEval = &LLMEvalResult{
			HaikuValid:     true,
			HaikuRationale: tr.rationale,
		}
		filtered = append(filtered, p)
	}

	logf("  triage complete: %d kept, %d dropped", len(filtered), dropped)
	return filtered, nil
}

const haikuTriageSystem = `You evaluate maintenance proposals for a memory system. Each proposal suggests a change (consolidate, promote, demote, split) based on mechanical signals like similarity scores.

Your job: Judge whether the proposal is valid based on the ACTUAL CONTENT, not just the mechanical signal. Two entries can share vocabulary (high similarity) but teach completely different lessons.

Output ONLY a JSON object: {"valid": true/false, "rationale": "one-line explanation"}`

func triageOneProposal(ctx context.Context, ext APIMessageCaller, p MaintenanceProposal) (bool, string, error) {
	userMsg := fmt.Sprintf("Proposal: %s (%s tier)\nMechanical reason: %s\n\nContent:\n%s",
		p.Action, p.Tier, p.Reason, p.Preview)

	params := APIMessageParams{
		System: haikuTriageSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
		},
		MaxTokens: 256,
		// No Model override — uses default Haiku
	}

	raw, err := ext.CallAPIWithMessages(ctx, params)
	if err != nil {
		return false, "", err
	}

	var result triageResult
	if err := json.Unmarshal(raw, &result); err != nil {
		// Try to find JSON in response
		start := strings.Index(string(raw), "{")
		end := strings.LastIndex(string(raw), "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal(raw[start:end+1], &result); err2 != nil {
				return false, "", fmt.Errorf("parse triage response: %w", err)
			}
		} else {
			return false, "", fmt.Errorf("parse triage response: %w", err)
		}
	}

	return result.Valid, result.Rationale, nil
}
```

Define `APIMessageCaller` interface so we can mock it in tests:

```go
// APIMessageCaller is the interface for making LLM API calls.
type APIMessageCaller interface {
	CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestTriageProposals -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/memory/optimize_llm_eval.go internal/memory/optimize_llm_eval_test.go
git commit -m "feat(memory): add Haiku triage for optimize proposals"
```

---

### Task 3: Implement Sonnet Behavioral Testing

Add `BehavioralTest()` that generates test scenarios and simulates before/after context for each triaged proposal.

**Files:**
- Modify: `internal/memory/optimize_llm_eval.go`
- Modify: `internal/memory/optimize_llm_eval_test.go`

**Step 1: Write the failing test**

Add to `internal/memory/optimize_llm_eval_test.go`:

```go
func TestBehavioralTest_PopulatesSonnetFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &mockBehavioralExtractor{
		response: `{
			"recommend": "skip",
			"confidence": "high",
			"change_analysis": "loses polling advice",
			"preservation_report": [
				{"scenario": "team structure", "preserved": true},
				{"scenario": "idle agents", "preserved": false, "lost": "polling instruction"}
			]
		}`,
	}

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  "id1,id2",
		Reason:  "similarity 0.92",
		Preview: "Keep: entry A text\nDelete: entry B text",
		LLMEval: &LLMEvalResult{
			HaikuValid:     true,
			HaikuRationale: "valid concern",
		},
	}

	// contextAssembler provides the simulated context window
	assembler := &mockContextAssembler{
		before: "CLAUDE.md content\n---\nMemory A\nMemory B",
		after:  "CLAUDE.md content\n---\nMemory A",
	}

	result, err := BehavioralTest(context.Background(), proposal, ext, assembler, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LLMEval.SonnetRecommend).To(Equal("skip"))
	g.Expect(result.LLMEval.SonnetConfidence).To(Equal("high"))
	g.Expect(result.LLMEval.SonnetSummary).To(Equal("loses polling advice"))
	g.Expect(result.LLMEval.ScenarioResults).To(HaveLen(2))
	g.Expect(result.LLMEval.ScenarioResults[0].Preserved).To(BeTrue())
	g.Expect(result.LLMEval.ScenarioResults[1].Preserved).To(BeFalse())
	g.Expect(result.LLMEval.ScenarioResults[1].Lost).To(Equal("polling instruction"))
}

type mockBehavioralExtractor struct {
	response string
}

func (m *mockBehavioralExtractor) CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error) {
	return []byte(m.response), nil
}

type mockContextAssembler struct {
	before string
	after  string
}

func (m *mockContextAssembler) AssembleContext(proposal MaintenanceProposal, applied bool) string {
	if applied {
		return m.after
	}
	return m.before
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestBehavioralTest -v`
Expected: FAIL — `BehavioralTest` undefined

**Step 3: Implement BehavioralTest**

Add to `internal/memory/optimize_llm_eval.go`:

```go
// ContextAssembler builds the simulated context window for behavioral testing.
type ContextAssembler interface {
	// AssembleContext returns the context that would be assembled if the proposal
	// is applied (applied=true) or not applied (applied=false).
	AssembleContext(proposal MaintenanceProposal, applied bool) string
}

// behavioralTestResponse is the JSON output from Sonnet.
type behavioralTestResponse struct {
	Recommend          string                    `json:"recommend"`
	Confidence         string                    `json:"confidence"`
	ChangeAnalysis     string                    `json:"change_analysis"`
	PreservationReport []behavioralScenarioResult `json:"preservation_report"`
}

type behavioralScenarioResult struct {
	Scenario  string `json:"scenario"`
	Preserved bool   `json:"preserved"`
	Lost      string `json:"lost,omitempty"`
}

const sonnetBehavioralSystem = `You are testing the behavioral impact of a proposed change to a memory system. You will receive:
1. The proposed change and its context
2. The assembled context window BEFORE the change
3. The assembled context window AFTER the change

Your job:
1. Generate 2-3 realistic user prompts that would trigger retrieval of the affected content
2. For each prompt, check whether the expected guidance still surfaces in the AFTER context
3. Recommend "apply" or "skip" based on whether the change preserves important behaviors

Output ONLY a JSON object with: recommend, confidence, change_analysis, preservation_report`

// BehavioralTest runs Sonnet behavioral testing on a single proposal.
// The proposal must have LLMEval.HaikuValid = true (survived triage).
// Returns the proposal with SonnetRecommend/SonnetConfidence/ScenarioResults populated.
func BehavioralTest(ctx context.Context, proposal MaintenanceProposal, ext APIMessageCaller, assembler ContextAssembler, progress io.Writer) (MaintenanceProposal, error) {
	logf := func(format string, args ...any) {
		if progress != nil {
			fmt.Fprintf(progress, format+"\n", args...)
		}
	}

	beforeCtx := assembler.AssembleContext(proposal, false)
	afterCtx := assembler.AssembleContext(proposal, true)

	userMsg := fmt.Sprintf(`Proposed change: %s (%s tier)
Reason: %s

Content affected:
%s

--- BEFORE context ---
%s

--- AFTER context ---
%s`, proposal.Action, proposal.Tier, proposal.Reason, proposal.Preview, beforeCtx, afterCtx)

	params := APIMessageParams{
		System: sonnetBehavioralSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
		},
		MaxTokens: 2048,
		Model:     sonnetModel,
	}

	raw, err := ext.CallAPIWithMessages(ctx, params)
	if err != nil {
		return proposal, fmt.Errorf("behavioral test API call: %w", err)
	}

	var resp behavioralTestResponse
	if err := parseJSONResponse(raw, &resp); err != nil {
		return proposal, fmt.Errorf("parse behavioral test response: %w", err)
	}

	// Populate LLMEval fields
	proposal.LLMEval.SonnetRecommend = resp.Recommend
	proposal.LLMEval.SonnetConfidence = resp.Confidence
	proposal.LLMEval.SonnetSummary = resp.ChangeAnalysis

	for _, sr := range resp.PreservationReport {
		proposal.LLMEval.ScenarioResults = append(proposal.LLMEval.ScenarioResults, ScenarioResult{
			Prompt:    sr.Scenario,
			Preserved: sr.Preserved,
			Lost:      sr.Lost,
		})
	}

	logf("  behavioral test: %s (%s confidence) — %s", resp.Recommend, resp.Confidence, resp.ChangeAnalysis)
	return proposal, nil
}

// parseJSONResponse tries to unmarshal JSON, with fallback to finding JSON in response text.
func parseJSONResponse(raw []byte, target interface{}) error {
	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}
	s := string(raw)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return json.Unmarshal([]byte(s[start:end+1]), target)
	}
	return fmt.Errorf("no JSON object found in response")
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestBehavioralTest -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/memory/optimize_llm_eval.go internal/memory/optimize_llm_eval_test.go
git commit -m "feat(memory): add Sonnet behavioral testing for optimize proposals"
```

---

### Task 4: Wire LLM Eval into OptimizeInteractive

Insert the triage and behavioral test stages into the optimize_interactive.go flow, between proposal collection and human review.

**Files:**
- Modify: `internal/memory/optimize_interactive.go:196-222`
- Modify: `internal/memory/optimize_interactive.go:12-23` (OptimizeInteractiveOpts)
- Modify: `cmd/projctl/memory_optimize.go` (add `--no-llm-eval` flag)

**Step 1: Add NoLLMEval and ContextAssembler to OptimizeInteractiveOpts**

In `internal/memory/optimize_interactive.go`, add to `OptimizeInteractiveOpts`:

```go
NoLLMEval       bool              // Skip LLM triage and behavioral testing
ContextAssembler ContextAssembler // For behavioral testing (nil = skip behavioral)
```

**Step 2: Insert LLM eval pipeline after tier filter, before review loop**

After the tier filter block (line ~210) and before `result.Total = len(allProposals)` (line ~212), add:

```go
// LLM evaluation pipeline (skip if --no-llm-eval or no extractor)
if !opts.NoLLMEval && opts.Extractor != nil {
	// Cast extractor to APIMessageCaller
	caller, ok := opts.Extractor.(APIMessageCaller)
	if ok {
		// Stage 1: Haiku triage
		allProposals, err = TriageProposals(opts.Context, allProposals, caller, opts.Output)
		if err != nil {
			fmt.Fprintf(opts.Output, "Warning: LLM triage failed: %v (proceeding without)\n", err)
		}

		// Stage 2: Sonnet behavioral testing (sequential, on triaged proposals only)
		if opts.ContextAssembler != nil {
			triaged := 0
			for i, p := range allProposals {
				if p.LLMEval != nil && p.LLMEval.HaikuValid {
					triaged++
					fmt.Fprintf(opts.Output, "- Behavioral test %d: %s %s...\n", triaged, p.Action, p.Tier)
					tested, testErr := BehavioralTest(opts.Context, p, caller, opts.ContextAssembler, opts.Output)
					if testErr != nil {
						fmt.Fprintf(opts.Output, "  Warning: behavioral test failed: %v\n", testErr)
					} else {
						allProposals[i] = tested
					}
				}
			}
		}
	}
}
```

**Step 3: Add --no-llm-eval flag to CLI**

In `cmd/projctl/memory_optimize.go`, the existing `NoLLM` flag covers "no LLM at all." Add a separate `NoLLMEval` flag:

```go
NoLLMEval bool `targ:"flag,name=no-llm-eval,desc=Skip LLM triage and behavioral testing (mechanical + human only)"`
```

Wire it into `OptimizeInteractiveOpts`:

```go
NoLLMEval: args.NoLLMEval,
```

**Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/memory/optimize_interactive.go cmd/projctl/memory_optimize.go
git commit -m "feat(memory): wire LLM eval pipeline into optimize --review"
```

---

### Task 5: Implement ContextAssembler

Create a real `ContextAssembler` that builds the simulated context window from CLAUDE.md + skills + embeddings.

**Files:**
- Create: `internal/memory/context_assembler.go`
- Create: `internal/memory/context_assembler_test.go`

**Step 1: Write the failing test**

Create `internal/memory/context_assembler_test.go`:

```go
package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestMemoryContextAssembler_BeforeAfter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	assembler := &MemoryContextAssembler{
		ClaudeMDContent: "# CLAUDE.md\nAlways use TDD.\n",
		SkillDescriptions: []string{"commit: stages and commits code"},
		Embeddings: []string{
			"When managing teams, delegate authority",
			"When using multi-agent teams, use active polling",
		},
	}

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  "0,1", // indices into Embeddings slice
		Preview: "Keep: When managing teams, delegate authority\nDelete: When using multi-agent teams, use active polling",
	}

	before := assembler.AssembleContext(proposal, false)
	g.Expect(before).To(ContainSubstring("Always use TDD"))
	g.Expect(before).To(ContainSubstring("delegate authority"))
	g.Expect(before).To(ContainSubstring("active polling"))

	after := assembler.AssembleContext(proposal, true)
	g.Expect(after).To(ContainSubstring("Always use TDD"))
	g.Expect(after).To(ContainSubstring("delegate authority"))
	g.Expect(after).ToNot(ContainSubstring("active polling"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestMemoryContextAssembler -v`
Expected: FAIL — `MemoryContextAssembler` undefined

**Step 3: Implement MemoryContextAssembler**

Create `internal/memory/context_assembler.go`:

```go
package memory

import (
	"fmt"
	"strconv"
	"strings"
)

// MemoryContextAssembler builds simulated context windows for behavioral testing.
// It holds the current state of each tier and can simulate before/after for a proposal.
type MemoryContextAssembler struct {
	ClaudeMDContent   string
	SkillDescriptions []string
	Embeddings        []string
}

// AssembleContext returns the context window as it would appear before or after the proposal.
func (a *MemoryContextAssembler) AssembleContext(proposal MaintenanceProposal, applied bool) string {
	var sb strings.Builder

	// CLAUDE.md section
	sb.WriteString("## CLAUDE.md (always loaded)\n")
	claudemd := a.ClaudeMDContent
	if applied && proposal.Tier == "claude-md" {
		claudemd = a.applyToClaudeMD(proposal)
	}
	sb.WriteString(claudemd)
	sb.WriteString("\n")

	// Skills section
	sb.WriteString("## Skills (matched by context)\n")
	skills := a.SkillDescriptions
	if applied && proposal.Tier == "skills" {
		skills = a.applyToSkills(proposal)
	}
	for _, s := range skills {
		sb.WriteString("- ")
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Embeddings section
	sb.WriteString("## Memories (retrieved by similarity)\n")
	embeddings := a.Embeddings
	if applied && proposal.Tier == "embeddings" {
		embeddings = a.applyToEmbeddings(proposal)
	}
	for _, e := range embeddings {
		sb.WriteString("- ")
		sb.WriteString(e)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (a *MemoryContextAssembler) applyToEmbeddings(p MaintenanceProposal) []string {
	switch p.Action {
	case "consolidate":
		// Target format: "idx1,idx2" — remove the second
		parts := strings.SplitN(p.Target, ",", 2)
		if len(parts) == 2 {
			removeIdx, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && removeIdx < len(a.Embeddings) {
				result := make([]string, 0, len(a.Embeddings)-1)
				for i, e := range a.Embeddings {
					if i != removeIdx {
						result = append(result, e)
					}
				}
				return result
			}
		}
	case "promote":
		// Remove from embeddings (moved to skills)
		idx, err := strconv.Atoi(strings.TrimSpace(p.Target))
		if err == nil && idx < len(a.Embeddings) {
			result := make([]string, 0, len(a.Embeddings)-1)
			for i, e := range a.Embeddings {
				if i != idx {
					result = append(result, e)
				}
			}
			return result
		}
	}
	return a.Embeddings
}

func (a *MemoryContextAssembler) applyToSkills(p MaintenanceProposal) []string {
	// For promote: add the promoted content
	if p.Action == "promote" {
		return append(a.SkillDescriptions, p.Preview)
	}
	return a.SkillDescriptions
}

func (a *MemoryContextAssembler) applyToClaudeMD(p MaintenanceProposal) string {
	switch p.Action {
	case "demote":
		// Remove the target line from CLAUDE.md
		lines := strings.Split(a.ClaudeMDContent, "\n")
		var result []string
		for _, line := range lines {
			if !strings.Contains(line, p.Target) {
				result = append(result, line)
			}
		}
		return strings.Join(result, "\n")
	}
	return a.ClaudeMDContent
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestMemoryContextAssembler -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/memory/context_assembler.go internal/memory/context_assembler_test.go
git commit -m "feat(memory): add MemoryContextAssembler for behavioral testing"
```

---

### Task 6: Wire ContextAssembler into CLI

Build the `MemoryContextAssembler` from real data in the CLI and pass it to `OptimizeInteractiveOpts`.

**Files:**
- Modify: `cmd/projctl/memory_optimize.go`

**Step 1: Build assembler from real data**

In `runInteractiveOptimize()`, before creating `OptimizeInteractiveOpts`, build the assembler:

```go
// Build context assembler for behavioral testing
var contextAssembler memory.ContextAssembler
if !args.NoLLMEval {
	claudeMDContent, _ := os.ReadFile(claudeMDPath)

	// Load skill descriptions
	var skillDescs []string
	if skillsDir != "" {
		entries, _ := os.ReadDir(skillsDir)
		for _, e := range entries {
			if e.IsDir() {
				skillPath := filepath.Join(skillsDir, e.Name(), "SKILL.md")
				if data, err := os.ReadFile(skillPath); err == nil {
					// Use first line as description
					lines := strings.SplitN(string(data), "\n", 2)
					if len(lines) > 0 {
						skillDescs = append(skillDescs, e.Name()+": "+lines[0])
					}
				}
			}
		}
	}

	// Load recent embeddings
	db, err := InitEmbeddingsDB(memoryRoot)
	if err == nil {
		embeddings := loadRecentEmbeddings(db, 50) // top 50 by confidence
		db.Close()
		contextAssembler = &memory.MemoryContextAssembler{
			ClaudeMDContent:   string(claudeMDContent),
			SkillDescriptions: skillDescs,
			Embeddings:        embeddings,
		}
	}
}
```

Wire into opts:
```go
ContextAssembler: contextAssembler,
NoLLMEval:        args.NoLLMEval,
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 3: Commit**

```bash
git add cmd/projctl/memory_optimize.go
git commit -m "feat(memory): wire ContextAssembler into optimize CLI"
```

---

### Task 7: Move Prune/Decay to Stop Hook

Add automatic prune/decay execution to the session stop hook.

**Files:**
- Modify: `cmd/projctl/memory_extract_session.go`
- Modify: `internal/memory/embeddings_maintenance.go` (add `AutoMaintenance()`)

**Step 1: Write the failing test**

Add to an appropriate test file (or create `internal/memory/auto_maintenance_test.go`):

```go
func TestAutoMaintenance_PrunesAndDecays(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := setupTestDB(t) // helper that creates in-memory DB with schema

	// Insert low-confidence entry (should be pruned)
	insertTestEmbedding(t, db, "low-conf", 0.2, 100) // 100 days old

	// Insert stale entry (should be decayed)
	insertTestEmbedding(t, db, "stale", 0.8, 100) // 100 days, will have <5 retrievals

	// Insert healthy entry (should be untouched)
	insertTestEmbedding(t, db, "healthy", 0.9, 5)

	pruned, decayed, err := AutoMaintenance(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(BeNumerically(">=", 1))
	g.Expect(decayed).To(BeNumerically(">=", 1))
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestAutoMaintenance -v`
Expected: FAIL — `AutoMaintenance` undefined

**Step 3: Implement AutoMaintenance**

Add to `internal/memory/embeddings_maintenance.go`:

```go
// AutoMaintenance runs automatic prune and decay operations.
// Returns counts of pruned and decayed entries.
func AutoMaintenance(db *sql.DB) (pruned int, decayed int, err error) {
	// Prune: confidence < 0.3
	res, err := db.Exec(`DELETE FROM embeddings WHERE confidence < 0.3`)
	if err != nil {
		return 0, 0, fmt.Errorf("auto-prune: %w", err)
	}
	rows, _ := res.RowsAffected()
	pruned = int(rows)

	// Decay: >90 days old, <5 retrievals → confidence × 0.5
	res, err = db.Exec(`UPDATE embeddings SET confidence = confidence * 0.5
		WHERE julianday('now') - julianday(created_at) > 90
		AND retrieval_count < 5
		AND confidence >= 0.3`)
	if err != nil {
		return pruned, 0, fmt.Errorf("auto-decay: %w", err)
	}
	rows, _ = res.RowsAffected()
	decayed = int(rows)

	return pruned, decayed, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestAutoMaintenance -v`
Expected: PASS

**Step 5: Wire into stop hook**

In `cmd/projctl/memory_extract_session.go`, after principle storage, add:

```go
// Auto-maintenance: prune and decay
if db != nil {
	pruned, decayed, err := memory.AutoMaintenance(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auto-maintenance failed: %v\n", err)
	} else if pruned > 0 || decayed > 0 {
		fmt.Printf("Memory maintenance: pruned %d, decayed %d\n", pruned, decayed)
	}
}
```

**Step 6: Remove prune/decay from optimize scanners**

In `internal/memory/embeddings_maintenance.go`, modify `scanEmbeddings()` to skip `scanLowConfidenceEmbeddings()` and `scanStaleEmbeddings()` — these now happen automatically.

**Step 7: Build and verify**

Run: `go build ./...`
Expected: Clean build

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 8: Commit**

```bash
git add internal/memory/embeddings_maintenance.go internal/memory/auto_maintenance_test.go cmd/projctl/memory_extract_session.go
git commit -m "feat(memory): move prune/decay to automatic stop hook execution"
```

---

### Task 8: Update Existing Review UI Tests

Update existing tests to handle the new format and ensure backward compatibility.

**Files:**
- Modify: `internal/memory/review_ui_test.go`

**Step 1: Update existing format tests**

Verify that existing tests still pass with nil `LLMEval` (backward compatible). Add a test that the old `[y]es / [n]o / [s]kip` prompt still appears for proposals without LLMEval.

```go
func TestFormatProposal_BackwardCompatible(t *testing.T) {
	g := gomega.NewWithT(t)

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "rewrite",
		Target:  "id1",
		Reason:  "Clarity improvement",
		Preview: "Rewritten content here",
		// No LLMEval — old format
	}

	formatted := formatProposal(proposal)
	g.Expect(formatted).To(gomega.ContainSubstring("[y]es / [n]o / [s]kip"))
	g.Expect(formatted).ToNot(gomega.ContainSubstring("Haiku:"))
	g.Expect(formatted).ToNot(gomega.ContainSubstring("Sonnet"))
}

func TestReviewProposal_AcceptsApplyInput(t *testing.T) {
	g := gomega.NewWithT(t)

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  "id1,id2",
		Reason:  "Redundant",
		Preview: "content",
		LLMEval: &LLMEvalResult{HaikuValid: true},
	}

	input := strings.NewReader("a\n")
	output := &bytes.Buffer{}

	result, err := reviewProposal(proposal, input, output)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result).To(gomega.BeTrue())
}
```

**Step 2: Run all review UI tests**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestReviewProposal -v`
Expected: ALL PASS

Run: `go test -tags sqlite_fts5 ./internal/memory/... -run TestFormatProposal -v`
Expected: ALL PASS

**Step 3: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/... -count=1`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/memory/review_ui_test.go
git commit -m "test(memory): add backward compatibility and apply/skip input tests"
```

---

## Verification

After all tasks complete:

1. `go build ./...` — clean build
2. `go test -tags sqlite_fts5 ./internal/memory/... -count=1` — all tests pass
3. `go test -tags sqlite_fts5 ./cmd/projctl/... -count=1` — CLI tests pass
4. `projctl memory optimize --review --no-llm-eval` — works like today (backward compatible)
5. `projctl memory optimize --review` — runs full pipeline with LLM eval

## Task Dependency Graph

```
Task 1 (types + UI)
    ↓
Task 2 (Haiku triage) ──→ Task 4 (wire into optimize)
    ↓                          ↓
Task 3 (Sonnet behavioral) ──→ Task 5 (context assembler) → Task 6 (CLI wiring)
                                                                ↓
Task 7 (stop hook prune/decay) ←── independent ──────────→ Task 8 (test updates)
```

Tasks 1-3 are sequential (each builds on the prior). Task 4 depends on 2+3. Task 5 depends on 3. Task 6 depends on 4+5. Task 7 is independent. Task 8 depends on 1.
