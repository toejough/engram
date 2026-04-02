package maintain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// Rewriter uses an LLM to generate replacement text for update proposals
// that lack a Value field.
type Rewriter struct {
	caller         anthropic.CallerFunc
	promptTemplate string
}

// NewRewriter creates a Rewriter with the given caller and system prompt template.
func NewRewriter(caller anthropic.CallerFunc, promptTemplate string) *Rewriter {
	return &Rewriter{
		caller:         caller,
		promptTemplate: promptTemplate,
	}
}

// RewriteProposals fills in the Value field for update proposals that have an empty Value.
// Non-update proposals and proposals with an existing Value are returned unchanged.
// On caller error, returns the original proposals alongside the error.
func (r *Rewriter) RewriteProposals(
	ctx context.Context,
	proposals []Proposal,
	records []memory.StoredRecord,
) ([]Proposal, error) {
	recordsByPath := indexRecordsByPath(records)
	result := make([]Proposal, len(proposals))
	copy(result, proposals)

	var errs []error

	for idx := range result {
		proposal := &result[idx]

		if proposal.Action != ActionUpdate || proposal.Value != "" {
			continue
		}

		record, found := recordsByPath[proposal.Target]
		if !found {
			continue
		}

		userPrompt := BuildRewritePrompt(proposal.Field, &record.Record)

		response, err := r.caller(
			ctx, anthropic.SonnetModel, r.promptTemplate, userPrompt,
		)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"rewriting %s field for %s: %w",
				proposal.Field, proposal.Target, err,
			))

			continue
		}

		proposal.Value = strings.TrimSpace(response)
	}

	if len(errs) > 0 {
		return result, fmt.Errorf(
			"rewriting proposals: %w", errors.Join(errs...),
		)
	}

	return result, nil
}

// BuildRewritePrompt formats a user prompt for the rewrite LLM call.
func BuildRewritePrompt(field string, record *memory.MemoryRecord) string {
	var builder strings.Builder

	currentValue := fieldValue(field, record)

	fmt.Fprintf(&builder, "=== Memory ===\n")
	fmt.Fprintf(&builder, "situation = %q\n", record.Situation)
	fmt.Fprintf(&builder, "behavior = %q\n", record.Behavior)
	fmt.Fprintf(&builder, "impact = %q\n", record.Impact)
	fmt.Fprintf(&builder, "action = %q\n\n", record.Action)

	fmt.Fprintf(&builder, "=== Field to rewrite ===\n")
	fmt.Fprintf(&builder, "Field: %s\n", field)
	fmt.Fprintf(&builder, "Current value: %q\n\n", currentValue)

	fmt.Fprintf(&builder, "=== Verdict summary ===\n")
	fmt.Fprintf(&builder, "Surfaced: %d\n", record.SurfacedCount)
	fmt.Fprintf(&builder, "Followed: %d\n", record.FollowedCount)
	fmt.Fprintf(&builder, "Not followed: %d\n", record.NotFollowedCount)
	fmt.Fprintf(&builder, "Irrelevant: %d\n", record.IrrelevantCount)

	return builder.String()
}

// fieldValue returns the value of a named SBIA field from a memory record.
func fieldValue(field string, record *memory.MemoryRecord) string {
	switch field {
	case "situation":
		return record.Situation
	case "behavior":
		return record.Behavior
	case "impact":
		return record.Impact
	case "action":
		return record.Action
	default:
		return ""
	}
}

// indexRecordsByPath creates a lookup map from path to StoredRecord.
func indexRecordsByPath(
	records []memory.StoredRecord,
) map[string]*memory.StoredRecord {
	index := make(map[string]*memory.StoredRecord, len(records))

	for idx := range records {
		index[records[idx].Path] = &records[idx]
	}

	return index
}
