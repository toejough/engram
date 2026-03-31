// Package maintain generates maintenance proposals for memories based on
// effectiveness quadrant classification.
// NOTE: Generator is stubbed during SBIA migration (Step 1). Will be rebuilt in Step 5.
package maintain

import (
	"context"
	"encoding/json"

	"engram/internal/memory"
)

// Exported constants.
const (
	// ActionConsolidate is the action for consolidation proposals (#373).
	ActionConsolidate = "consolidate"
)

// ConsolidateResultType describes the outcome of a BeforeRemove check.
type ConsolidateResultType int

// ConsolidateResultType values.
const (
	// ConsolidateSkip means a cluster was found and merged; skip removal.
	ConsolidateSkip ConsolidateResultType = iota
	// ConsolidateProceed means no cluster found; proceed with removal.
	ConsolidateProceed
)

// ConsolidateResult is the outcome of a BeforeRemove consolidation check.
type ConsolidateResult struct {
	Type ConsolidateResultType
}

// Generator produces maintenance proposals. Stubbed during SBIA migration.
type Generator struct{}

// New creates a Generator with the given options.
func New(_ ...Option) *Generator {
	return &Generator{}
}

// Option configures a Generator.
type Option func(*Generator)

// Proposal represents a recommended maintenance action for a memory.
//
//nolint:tagliatelle // DES-23 specifies snake_case JSON field names.
type Proposal struct {
	MemoryPath string          `json:"memory_path"`
	Quadrant   string          `json:"quadrant"`
	Diagnosis  string          `json:"diagnosis"`
	Action     string          `json:"action"`
	Details    json.RawMessage `json:"details"`
}

// WithConsolidator sets the consolidation check (stubbed).
func WithConsolidator(
	_ interface {
		BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (ConsolidateResult, error)
	},
	_ func(string) (*memory.MemoryRecord, error),
) Option {
	return func(_ *Generator) {}
}

// WithLLMCaller sets the LLM calling function (stubbed).
func WithLLMCaller(
	_ func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) Option {
	return func(_ *Generator) {}
}

// WithRefineKeywordsIrrelevanceThreshold is stubbed.
func WithRefineKeywordsIrrelevanceThreshold(_ float64) Option {
	return func(_ *Generator) {}
}

// WithStalenessThresholdDays is stubbed.
func WithStalenessThresholdDays(_ int) Option {
	return func(_ *Generator) {}
}

// unexported constants.
const (
	actionBroadenKeywords = "broaden_keywords"
	actionRemove          = "remove"
	actionReviewStaleness = "review_staleness"
	actionRewrite         = "rewrite"
)
