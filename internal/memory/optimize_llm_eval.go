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
	if err := json.Unmarshal(raw, &result); err != nil {
		// Try to find JSON in response
		s := string(raw)
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(s[start:end+1]), &result); err2 != nil {
				return false, "", fmt.Errorf("parse triage response: %w", err)
			}
		} else {
			return false, "", fmt.Errorf("parse triage response: %w", err)
		}
	}

	return result.Valid, result.Rationale, nil
}
