// Package maintain provides maintenance operations for memories.
package maintain

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"engram/internal/anthropic"
	"engram/internal/memory"
	"engram/internal/policy"
)

// Exported variables.
var (
	ErrUserQuit = errors.New("user quit")
)

// Config holds all dependencies for the maintain orchestrator.
type Config struct {
	Policy        policy.Policy
	DataDir       string
	Caller        anthropic.CallerFunc // nil = skip Sonnet-dependent analyses
	ChangeHistory []policy.ChangeEntry
}

// Confirmer prompts the user for confirmation during maintenance operations.
type Confirmer interface {
	Confirm(prompt string) (bool, error)
}

// Run executes all maintenance analyses: decision tree (always), consolidation
// and adapt (only when Caller is non-nil). Returns combined proposals from all sources.
// When Sonnet-dependent analyses fail, returns decision tree proposals alongside the error.
func Run(ctx context.Context, cfg Config) ([]Proposal, error) {
	memDir := filepath.Join(cfg.DataDir, "memories")

	records, err := memory.ListAll(memDir)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
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

	if cfg.Caller == nil {
		return proposals, nil
	}

	sonnetProposals, sonnetErr := runSonnetAnalyses(ctx, cfg, records)
	proposals = append(proposals, sonnetProposals...)

	return proposals, sonnetErr
}

// runSonnetAnalyses runs consolidation and adapt analyses, collecting all errors.
func runSonnetAnalyses(
	ctx context.Context, cfg Config, records []memory.StoredRecord,
) ([]Proposal, error) {
	var proposals []Proposal

	var errs []error

	consolidator := NewConsolidator(cfg.Caller, cfg.Policy.MaintainConsolidatePrompt)

	mergeProposals, mergeErr := consolidator.FindMerges(ctx, records)
	if mergeErr != nil {
		errs = append(errs, fmt.Errorf("finding merges: %w", mergeErr))
	} else {
		proposals = append(proposals, mergeProposals...)
	}

	adapter := NewAdapter(cfg.Caller, cfg.Policy.AdaptSonnetPrompt)

	adaptProposals, adaptErr := adapter.Analyze(ctx, records, cfg.Policy, cfg.ChangeHistory)
	if adaptErr != nil {
		errs = append(errs, fmt.Errorf("running adapt analysis: %w", adaptErr))
	} else {
		proposals = append(proposals, adaptProposals...)
	}

	return proposals, errors.Join(errs...)
}
