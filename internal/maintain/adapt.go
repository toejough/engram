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

// Adapter analyzes aggregate metrics and proposes parameter/prompt tuning via Sonnet.
type Adapter struct {
	caller         anthropic.CallerFunc
	promptTemplate string
}

// NewAdapter creates an Adapter with the given caller and system prompt template.
func NewAdapter(caller anthropic.CallerFunc, promptTemplate string) *Adapter {
	return &Adapter{
		caller:         caller,
		promptTemplate: promptTemplate,
	}
}

// Analyze examines aggregate metrics across all records and proposes parameter changes.
// Returns nil if there are no records to analyze.
func (a *Adapter) Analyze(
	ctx context.Context,
	records []memory.StoredRecord,
	pol policy.Policy,
	changeHistory []policy.ChangeEntry,
) ([]Proposal, error) {
	if len(records) < minRecordsForAdapt {
		return nil, nil
	}

	userPrompt := buildAdaptPrompt(records, pol, changeHistory)

	response, err := a.caller(ctx, anthropic.SonnetModel, a.promptTemplate, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("calling adapt model: %w", err)
	}

	var rawProposals []adaptProposal

	unmarshalErr := json.Unmarshal([]byte(response), &rawProposals)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("parsing adapt response: %w", unmarshalErr)
	}

	proposals := make([]Proposal, 0, len(rawProposals))

	for index, raw := range rawProposals {
		proposals = append(proposals, Proposal{
			ID:        fmt.Sprintf("adapt-%d", index),
			Action:    ActionUpdate,
			Target:    adaptTarget,
			Field:     raw.Field,
			Value:     raw.Value,
			Rationale: raw.Rationale,
		})
	}

	return proposals, nil
}

// unexported constants.
const (
	adaptTarget        = policy.Filename
	minRecordsForAdapt = 1
)

// adaptProposal is the JSON structure returned by the adapt model.
type adaptProposal struct {
	Field     string `json:"field"`
	Value     string `json:"value"`
	Rationale string `json:"rationale"`
}

// aggregateMetrics sums evaluation counters across all records.
func aggregateMetrics(records []memory.StoredRecord) (
	surfaced, followed, notFollowed, irrelevant int,
) {
	for index := range records {
		record := &records[index]
		surfaced += record.Record.SurfacedCount
		followed += record.Record.FollowedCount
		notFollowed += record.Record.NotFollowedCount
		irrelevant += record.Record.IrrelevantCount
	}

	return surfaced, followed, notFollowed, irrelevant
}

// buildAdaptPrompt formats aggregate metrics, current parameters, and change history
// into a user prompt for the adapt model.
func buildAdaptPrompt(
	records []memory.StoredRecord,
	pol policy.Policy,
	changeHistory []policy.ChangeEntry,
) string {
	var builder strings.Builder

	// Aggregate metrics across all records.
	totalSurfaced, totalFollowed, totalNotFollowed, totalIrrelevant := aggregateMetrics(records)

	builder.WriteString("=== Aggregate Metrics ===\n")
	fmt.Fprintf(&builder, "Total memories: %d\n", len(records))
	fmt.Fprintf(&builder, "Total surfaced: %d\n", totalSurfaced)
	fmt.Fprintf(&builder, "Total followed: %d\n", totalFollowed)
	fmt.Fprintf(&builder, "Total not followed: %d\n", totalNotFollowed)
	fmt.Fprintf(&builder, "Total irrelevant: %d\n\n", totalIrrelevant)

	builder.WriteString("=== Current Parameters ===\n")
	writeCurrentParameters(&builder, pol)

	builder.WriteString("\n=== Change History ===\n")
	writeChangeHistory(&builder, changeHistory)

	return builder.String()
}

// writeChangeHistory formats change history entries into the builder.
func writeChangeHistory(builder *strings.Builder, history []policy.ChangeEntry) {
	if len(history) == 0 {
		builder.WriteString("No recent changes.\n")

		return
	}

	for _, entry := range history {
		fmt.Fprintf(builder, "- %s %s", entry.Action, entry.Target)

		if entry.Field != "" {
			fmt.Fprintf(builder, " field=%s", entry.Field)
		}

		if entry.OldValue != "" {
			fmt.Fprintf(builder, " old=%s", entry.OldValue)
		}

		if entry.NewValue != "" {
			fmt.Fprintf(builder, " new=%s", entry.NewValue)
		}

		fmt.Fprintf(builder, " (%s)\n", entry.Rationale)
	}
}

// writeCurrentParameters formats policy parameters into the builder.
func writeCurrentParameters(builder *strings.Builder, pol policy.Policy) {
	fmt.Fprintf(builder, "SurfaceBM25Threshold: %.2f\n", pol.SurfaceBM25Threshold)
	fmt.Fprintf(builder, "SurfaceCandidateCountMin: %d\n", pol.SurfaceCandidateCountMin)
	fmt.Fprintf(builder, "SurfaceCandidateCountMax: %d\n", pol.SurfaceCandidateCountMax)
	fmt.Fprintf(builder, "SurfaceColdStartBudget: %d\n", pol.SurfaceColdStartBudget)
	fmt.Fprintf(builder, "ExtractBM25Threshold: %.2f\n", pol.ExtractBM25Threshold)
	fmt.Fprintf(builder, "MaintainEffectivenessThreshold: %.2f\n",
		pol.MaintainEffectivenessThreshold)
	fmt.Fprintf(builder, "MaintainIrrelevanceThreshold: %.2f\n",
		pol.MaintainIrrelevanceThreshold)
	fmt.Fprintf(builder, "MaintainNotFollowedThreshold: %.2f\n",
		pol.MaintainNotFollowedThreshold)
}
