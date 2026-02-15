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

// APIMessageCaller is the interface for making LLM API calls.
// DirectAPIExtractor already implements this via CallAPIWithMessages.
type APIMessageCaller interface {
	CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error)
}

// ContextAssembler assembles the before/after memory context for behavioral testing.
type ContextAssembler interface {
	AssembleContext(proposal MaintenanceProposal, applied bool) string
}

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

// behavioralTestResponse holds the JSON output from Sonnet behavioral testing.
type behavioralTestResponse struct {
	Recommend          string                       `json:"recommend"`
	Confidence         string                       `json:"confidence"`
	ChangeAnalysis     string                       `json:"change_analysis"`
	PreservationReport []behavioralScenarioResult `json:"preservation_report"`
}

// behavioralScenarioResult holds one scenario from the behavioral test.
type behavioralScenarioResult struct {
	Scenario  string `json:"scenario"`
	Preserved bool   `json:"preserved"`
	Lost      string `json:"lost,omitempty"`
}

// TriageProposals sends judgment-call proposals to Haiku in parallel.
// Returns filtered proposals with LLMEval populated for triaged items.
// Proposals that don't need triage (rewrite, add-rationale) pass through unchanged.
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
		index     int
		valid     bool
		rationale string
		err       error
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

const sonnetBehavioralSystem = `You perform behavioral testing for proposed memory system changes. You receive a maintenance proposal with before/after memory contexts.

Your job:
1. Generate 3-5 realistic user scenarios that would trigger the affected memory content
2. For each scenario, simulate retrieval in BOTH contexts
3. Check if the expected guidance surfaces in both
4. Recommend "apply" only if all scenarios preserve critical guidance

A scenario is PRESERVED if the essential information is accessible in both contexts. It's LOST if consolidation/demotion/etc. makes it harder to retrieve or understand.

Output ONLY a JSON object:
{
  "recommend": "apply" or "skip",
  "confidence": "high", "medium", or "low",
  "change_analysis": "one-sentence summary of what changes",
  "preservation_report": [
    {"scenario": "user asks about X", "preserved": true},
    {"scenario": "user asks about Y", "preserved": false, "lost": "specific guidance that disappeared"}
  ]
}`

func triageOneProposal(ctx context.Context, ext APIMessageCaller, p MaintenanceProposal) (bool, string, error) {
	userMsg := fmt.Sprintf("Proposal: %s (%s tier)\nMechanical reason: %s\n\nContent:\n%s",
		p.Action, p.Tier, p.Reason, p.Preview)

	params := APIMessageParams{
		System: haikuTriageSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
		},
		MaxTokens: 256,
	}

	raw, err := ext.CallAPIWithMessages(ctx, params)
	if err != nil {
		return false, "", err
	}

	var result triageResult
	if err := parseJSONResponse(raw, &result); err != nil {
		return false, "", err
	}

	return result.Valid, result.Rationale, nil
}

// BehavioralTest performs Sonnet behavioral testing on a proposal that passed Haiku triage.
// If the proposal doesn't have LLMEval or HaikuValid is false, returns the proposal unchanged.
// Otherwise, assembles before/after context and runs behavioral scenarios through Sonnet.
func BehavioralTest(ctx context.Context, proposal MaintenanceProposal, ext APIMessageCaller, assembler ContextAssembler, progress io.Writer) (MaintenanceProposal, error) {
	// Skip if no LLMEval or not Haiku-validated
	if proposal.LLMEval == nil || !proposal.LLMEval.HaikuValid {
		return proposal, nil
	}

	logf := func(format string, args ...any) {
		if progress != nil {
			fmt.Fprintf(progress, format+"\n", args...)
		}
	}

	logf("- Behavioral test: %s %s...", proposal.Action, proposal.Tier)

	// Assemble before/after contexts
	beforeCtx := assembler.AssembleContext(proposal, false)
	afterCtx := assembler.AssembleContext(proposal, true)

	userMsg := fmt.Sprintf(`Proposal: %s (%s tier)
Reason: %s

BEFORE (current state):
%s

AFTER (if applied):
%s

Generate test scenarios and check if the change preserves critical guidance.`,
		proposal.Action, proposal.Tier, proposal.Reason, beforeCtx, afterCtx)

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
		return proposal, err
	}

	var result behavioralTestResponse
	if err := parseJSONResponse(raw, &result); err != nil {
		return proposal, err
	}

	// Populate Sonnet fields in LLMEval
	proposal.LLMEval.SonnetRecommend = result.Recommend
	proposal.LLMEval.SonnetConfidence = result.Confidence
	proposal.LLMEval.SonnetSummary = result.ChangeAnalysis

	// Convert behavioral scenario results to ScenarioResult
	proposal.LLMEval.ScenarioResults = make([]ScenarioResult, len(result.PreservationReport))
	for i, sr := range result.PreservationReport {
		proposal.LLMEval.ScenarioResults[i] = ScenarioResult{
			Prompt:    sr.Scenario,
			Preserved: sr.Preserved,
			Lost:      sr.Lost,
		}
	}

	logf("  result: %s (%s confidence)", result.Recommend, result.Confidence)
	return proposal, nil
}

// parseJSONResponse attempts to parse JSON from raw bytes.
// If direct unmarshal fails, tries to find JSON object in the text.
func parseJSONResponse[T any](raw []byte, result *T) error {
	if err := json.Unmarshal(raw, result); err != nil {
		// Try to find JSON in response
		s := string(raw)
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(s[start:end+1]), result); err2 != nil {
				return fmt.Errorf("parse JSON response: %w", err)
			}
		} else {
			return fmt.Errorf("parse JSON response: %w", err)
		}
	}
	return nil
}
