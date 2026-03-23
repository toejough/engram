// Package maintain generates maintenance proposals for memories based on
// effectiveness quadrant classification.
package maintain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"engram/internal/memory"
	"engram/internal/review"
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

// Generator produces maintenance proposals for classified memories.
type Generator struct {
	llmCaller    func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	now          func() time.Time
	consolidator interface {
		BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (ConsolidateResult, error)
	}
	memLoader func(path string) (*memory.MemoryRecord, error)
}

// New creates a Generator with the given options.
func New(opts ...Option) *Generator {
	gen := &Generator{
		now: time.Now,
	}
	for _, opt := range opts {
		opt(gen)
	}

	return gen
}

// Generate produces maintenance proposals for the given classified memories.
func (g *Generator) Generate(
	ctx context.Context,
	classified []review.ClassifiedMemory,
	memories map[string]*memory.Stored,
) []Proposal {
	proposals := make([]Proposal, 0, len(classified))

	for _, classifiedMem := range classified {
		stored := memories[classifiedMem.Name]

		proposal, ok := g.generateOne(ctx, classifiedMem, stored)
		if ok {
			proposals = append(proposals, proposal)
		}
	}

	return proposals
}

// checkIrrelevance proposes refine_keywords when a memory's irrelevance ratio exceeds
// the threshold (60%) with sufficient total feedback (>=5). Returns (proposal, true)
// if the check triggers, or (zero, false) to fall through to quadrant handling.
func (g *Generator) checkIrrelevance(
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
	if stored == nil {
		return Proposal{}, false
	}

	totalFeedback := stored.FollowedCount + stored.ContradictedCount +
		stored.IgnoredCount + stored.IrrelevantCount
	if totalFeedback < refineKeywordsMinFeedback {
		return Proposal{}, false
	}

	ratio := float64(stored.IrrelevantCount) / float64(totalFeedback)
	if ratio <= refineKeywordsIrrelevanceThreshold {
		return Proposal{}, false
	}

	const percentMultiplier = 100

	return Proposal{
		MemoryPath: classifiedMem.Name,
		Quadrant:   string(classifiedMem.Quadrant),
		Action:     actionRefineKeywords,
		Diagnosis: fmt.Sprintf(
			"%d%% of feedback is irrelevant — keywords may be too generic",
			int(ratio*percentMultiplier),
		),
	}, true
}

// generateOne produces a proposal for a single classified memory.
// Returns (proposal, true) if a proposal was generated, or (zero, false) to skip.
func (g *Generator) generateOne(
	ctx context.Context,
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
	// Check for high-irrelevance memories — propose keyword refinement (#343).
	if proposal, ok := g.checkIrrelevance(classifiedMem, stored); ok {
		return proposal, true
	}

	switch classifiedMem.Quadrant {
	case review.Working:
		return g.handleWorking(classifiedMem, stored)
	case review.Leech:
		return g.handleLeech(ctx, classifiedMem, stored)
	case review.HiddenGem:
		return g.handleHiddenGem(ctx, classifiedMem, stored)
	case review.Noise:
		if g.shouldSkipRemoval(ctx, classifiedMem) {
			return Proposal{}, false
		}

		return g.handleNoise(classifiedMem)
	case review.InsufficientData:
		return Proposal{}, false
	}

	return Proposal{}, false
}

func (g *Generator) handleHiddenGem(
	ctx context.Context,
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
	if g.llmCaller == nil {
		return Proposal{}, false
	}

	systemPrompt := hiddenGemSystemPrompt
	userPrompt := buildMemoryDescription(classifiedMem, stored)

	response, err := g.llmCaller(
		ctx, maintainModel, systemPrompt, userPrompt,
	)
	if err != nil {
		// Fire-and-forget: skip this proposal on LLM failure (ARCH-6).
		return Proposal{}, false
	}

	return Proposal{
		MemoryPath: classifiedMem.Name,
		Quadrant:   string(classifiedMem.Quadrant),
		Diagnosis:  "Memory has high follow-through but is rarely surfaced",
		Action:     actionBroadenKeywords,
		Details:    json.RawMessage(response),
	}, true
}

func (g *Generator) handleLeech(
	ctx context.Context,
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
	if g.llmCaller == nil {
		return Proposal{}, false
	}

	systemPrompt := leechSystemPrompt
	userPrompt := buildMemoryDescription(classifiedMem, stored)

	response, err := g.llmCaller(
		ctx, maintainModel, systemPrompt, userPrompt,
	)
	if err != nil {
		// Fire-and-forget: skip this proposal on LLM failure (ARCH-6).
		return Proposal{}, false
	}

	return Proposal{
		MemoryPath: classifiedMem.Name,
		Quadrant:   string(classifiedMem.Quadrant),
		Diagnosis:  "Memory is frequently surfaced but rarely followed",
		Action:     actionRewrite,
		Details:    json.RawMessage(response),
	}, true
}

func (g *Generator) handleNoise(
	classifiedMem review.ClassifiedMemory,
) (Proposal, bool) {
	return Proposal{
		MemoryPath: classifiedMem.Name,
		Quadrant:   string(classifiedMem.Quadrant),
		Diagnosis:  "Memory is rarely surfaced and ineffective when it is",
		Action:     actionRemove,
		Details:    marshalEvidence(classifiedMem),
	}, true
}

func (g *Generator) handleWorking(
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) (Proposal, bool) {
	if stored == nil {
		return Proposal{}, false
	}

	ageDays := int(g.now().Sub(stored.UpdatedAt).Hours() / hoursPerDay)
	if ageDays <= stalenessThresholdDays {
		return Proposal{}, false
	}

	//nolint:errchkjson // stalenessDetails has only int fields; cannot fail.
	details, _ := json.Marshal(stalenessDetails{AgeDays: ageDays})

	return Proposal{
		MemoryPath: classifiedMem.Name,
		Quadrant:   string(classifiedMem.Quadrant),
		Diagnosis: fmt.Sprintf(
			"Working memory not updated in %d days", ageDays,
		),
		Action:  actionReviewStaleness,
		Details: details,
	}, true
}

// shouldSkipRemoval checks if a noise-quadrant memory should be kept because
// it belongs to a semantic cluster that was consolidated.
func (g *Generator) shouldSkipRemoval(
	ctx context.Context,
	classifiedMem review.ClassifiedMemory,
) bool {
	if g.consolidator == nil || g.memLoader == nil {
		return false
	}

	mem, loadErr := g.memLoader(classifiedMem.Name)
	if loadErr != nil {
		return false
	}

	action, consErr := g.consolidator.BeforeRemove(ctx, mem)

	return consErr == nil && action.Type == ConsolidateSkip
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

// WithConsolidator sets the consolidation check for noise-quadrant memories.
func WithConsolidator(
	consolidator interface {
		BeforeRemove(ctx context.Context, mem *memory.MemoryRecord) (ConsolidateResult, error)
	},
	loader func(string) (*memory.MemoryRecord, error),
) Option {
	return func(g *Generator) {
		g.consolidator = consolidator
		g.memLoader = loader
	}
}

// WithLLMCaller sets the LLM calling function for diagnosis.
func WithLLMCaller(
	caller func(
		ctx context.Context, model, systemPrompt, userPrompt string,
	) (string, error),
) Option {
	return func(g *Generator) {
		g.llmCaller = caller
	}
}

// WithNow sets the time source for staleness checks.
func WithNow(nowFn func() time.Time) Option {
	return func(g *Generator) {
		g.now = nowFn
	}
}

// unexported constants.
const (
	actionBroadenKeywords = "broaden_keywords"
	actionRefineKeywords  = "refine_keywords"
	actionRemove          = "remove"
	actionReviewStaleness = "review_staleness"
	actionRewrite         = "rewrite"
	hiddenGemSystemPrompt = "You are a memory maintenance assistant. " +
		"Analyze what contexts this memory could be relevant in. " +
		"Propose additional keywords to broaden its triggers. " +
		"Output: " +
		`{"additional_keywords":[...],"rationale":"..."}`
	hoursPerDay       = 24
	leechSystemPrompt = "You are a memory maintenance assistant. " +
		"Analyze why this memory is being ignored by agents " +
		"(content quality, keyword mismatch, wrong tier). " +
		"Propose specific field-level changes as JSON. " +
		"Output: " +
		`{"proposed_keywords":[...],"proposed_principle":"...","rationale":"..."}`
	maintainModel                      = "claude-haiku-4-5-20251001"
	refineKeywordsIrrelevanceThreshold = 0.6
	refineKeywordsMinFeedback          = 5
	stalenessThresholdDays             = 90
)

//nolint:tagliatelle // DES-23 specifies snake_case JSON field names.
type noiseEvidence struct {
	SurfacedCount      int     `json:"surfaced_count"`
	EffectivenessScore float64 `json:"effectiveness_score"`
	EvaluationCount    int     `json:"evaluation_count"`
}

//nolint:tagliatelle // DES-23 specifies snake_case JSON field names.
type stalenessDetails struct {
	AgeDays int `json:"age_days"`
}

func buildMemoryDescription(
	classifiedMem review.ClassifiedMemory,
	stored *memory.Stored,
) string {
	title := ""
	keywords := ""
	principle := ""

	if stored != nil {
		title = stored.Title
		keywords = fmt.Sprintf("%v", stored.Keywords)
		principle = stored.Principle
	}

	return fmt.Sprintf(
		"Memory: %s\nTitle: %s\nKeywords: %s\nPrinciple: %s\n"+
			"Surfaced: %d times\nEffectiveness: %.1f%%\n"+
			"Evaluations: %d",
		classifiedMem.Name, title, keywords, principle,
		classifiedMem.SurfacedCount,
		classifiedMem.EffectivenessScore,
		classifiedMem.EvaluationCount,
	)
}

// marshalEvidence marshals noise evidence stats.
//
//nolint:errchkjson // noiseEvidence has only int/float fields; cannot fail.
func marshalEvidence(
	classifiedMem review.ClassifiedMemory,
) json.RawMessage {
	data, _ := json.Marshal(noiseEvidence{
		SurfacedCount:      classifiedMem.SurfacedCount,
		EffectivenessScore: classifiedMem.EffectivenessScore,
		EvaluationCount:    classifiedMem.EvaluationCount,
	})

	return data
}
