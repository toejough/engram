package maintain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// Consolidator detects similar memories and generates merge proposals via Sonnet.
type Consolidator struct {
	caller         anthropic.CallerFunc
	promptTemplate string
}

// NewConsolidator creates a Consolidator with the given caller and system prompt template.
func NewConsolidator(caller anthropic.CallerFunc, promptTemplate string) *Consolidator {
	return &Consolidator{
		caller:         caller,
		promptTemplate: promptTemplate,
	}
}

// FindMerges identifies groups of similar memories that could be merged.
// Returns nil if there are fewer than 2 records.
func (c *Consolidator) FindMerges(
	ctx context.Context, records []memory.StoredRecord,
) ([]Proposal, error) {
	if len(records) < minRecordsForConsolidation {
		return nil, nil
	}

	userPrompt := buildConsolidationPrompt(records)

	response, err := c.caller(ctx, anthropic.SonnetModel, c.promptTemplate, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("calling consolidation model: %w", err)
	}

	var candidates []mergeCandidate

	unmarshalErr := json.Unmarshal([]byte(response), &candidates)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("parsing consolidation response: %w", unmarshalErr)
	}

	proposals := make([]Proposal, 0, len(candidates))

	for index, candidate := range candidates {
		related := filterSurvivor(candidate.Members, candidate.Survivor)

		proposals = append(proposals, Proposal{
			ID:        fmt.Sprintf("consolidate-%d", index),
			Action:    ActionMerge,
			Target:    candidate.Survivor,
			Related:   related,
			Rationale: candidate.Rationale,
		})
	}

	return proposals, nil
}

// unexported constants.
const (
	minRecordsForConsolidation = 2
)

// mergeCandidate is the JSON structure returned by the consolidation model.
type mergeCandidate struct {
	Survivor  string   `json:"survivor"`
	Members   []string `json:"members"`
	Rationale string   `json:"rationale"`
}

// buildConsolidationPrompt formats all memory records into a prompt for the model.
func buildConsolidationPrompt(records []memory.StoredRecord) string {
	var builder strings.Builder

	builder.WriteString("Analyze these memories for consolidation opportunities:\n\n")

	for _, stored := range records {
		fmt.Fprintf(&builder, "- Path: %s\n", stored.Path)
		fmt.Fprintf(&builder, "  Situation: %s\n", stored.Record.Situation)
		fmt.Fprintf(&builder, "  Action: %s\n\n", stored.Record.Action)
	}

	return builder.String()
}

// filterSurvivor returns members with the survivor path removed.
func filterSurvivor(members []string, survivor string) []string {
	filtered := make([]string, 0, len(members))

	for _, member := range members {
		if member != survivor {
			filtered = append(filtered, member)
		}
	}

	return filtered
}
