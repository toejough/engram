package maintain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrUserQuit = errors.New("user quit")
)

// ApplyReport summarizes the results of applying proposals.
type ApplyReport struct {
	Applied     int
	Skipped     int
	NotReached  int
	Total       int
	SkipReasons []string
}

// Confirmer asks the user to confirm an action.
type Confirmer interface {
	Confirm(preview string) (bool, error)
}

// Executor applies maintenance proposals to memories.
type Executor struct {
	rewriter  MemoryRewriter
	remover   MemoryRemover
	registry  RegistryUpdater
	llmCaller LLMCaller
	confirmer Confirmer
}

// NewExecutor creates an Executor with the given options.
func NewExecutor(opts ...ExecutorOption) *Executor {
	exec := &Executor{}
	for _, opt := range opts {
		opt(exec)
	}

	return exec
}

// Apply walks proposals, confirming and applying each one.
func (e *Executor) Apply(ctx context.Context, proposals []Proposal) ApplyReport {
	report := ApplyReport{Total: len(proposals)}

	for idx := range proposals {
		applied, skipReason, err := e.applyOne(ctx, proposals[idx])
		if errors.Is(err, ErrUserQuit) {
			report.NotReached = len(proposals) - idx

			break
		}

		if applied {
			report.Applied++
		} else {
			report.Skipped++

			if skipReason != "" {
				report.SkipReasons = append(report.SkipReasons, skipReason)
			}
		}
	}

	return report
}

func (e *Executor) applyBroadenKeywords(
	ctx context.Context,
	proposal Proposal,
) (bool, string, error) {
	// Extract existing keywords from proposal details.
	var details struct {
		AdditionalKeywords []string `json:"additional_keywords"`
	}

	unmarshalErr := json.Unmarshal(proposal.Details, &details)
	if unmarshalErr != nil {
		return false, "invalid details", nil
	}

	newKeywords := details.AdditionalKeywords
	if len(newKeywords) == 0 {
		// Ask LLM for suggestions.
		prompt := fmt.Sprintf(
			"Suggest additional keywords for this memory.\n"+
				"Memory: %s\nDiagnosis: %s\nDetails: %s\n"+
				"Output JSON: {\"additional_keywords\": [...]}",
			proposal.MemoryPath, proposal.Diagnosis, string(proposal.Details),
		)

		response, err := e.llmCaller.Call(ctx, prompt)
		if err != nil {
			return false, fmt.Sprintf("llm error: %s", err), nil
		}

		unmarshalErr := json.Unmarshal([]byte(response), &details)
		if unmarshalErr != nil {
			return false, "invalid llm response", nil
		}

		newKeywords = details.AdditionalKeywords
	}

	updates := map[string]any{
		"keywords": newKeywords,
	}

	confirmed, confirmErr := e.confirm(proposal, updates)
	if confirmErr != nil {
		return false, "", confirmErr
	}

	if !confirmed {
		return false, "", nil
	}

	rewriteErr := e.rewriter.Rewrite(proposal.MemoryPath, updates)
	if rewriteErr != nil {
		return false, fmt.Sprintf("rewrite error: %s", rewriteErr), nil
	}

	return true, "", nil
}

// applyOne applies a single proposal. Returns (applied, skipReason, error).
func (e *Executor) applyOne(
	ctx context.Context,
	proposal Proposal,
) (bool, string, error) {
	needsLLM := proposal.Action != actionRemove
	if needsLLM && e.llmCaller == nil {
		return false, "no token", nil
	}

	switch proposal.Action {
	case actionReviewStaleness:
		return e.applyStaleUpdate(ctx, proposal)
	case actionRewrite:
		return e.applyRewrite(ctx, proposal)
	case actionBroadenKeywords:
		return e.applyBroadenKeywords(ctx, proposal)
	case actionRemove:
		return e.applyRemoval(proposal)
	default:
		return false, "unknown action: " + proposal.Action, nil
	}
}

func (e *Executor) applyRemoval(proposal Proposal) (bool, string, error) {
	confirmed, confirmErr := e.confirm(proposal, nil)
	if confirmErr != nil {
		return false, "", confirmErr
	}

	if !confirmed {
		return false, "", nil
	}

	if e.remover != nil {
		removeErr := e.remover.Remove(proposal.MemoryPath)
		if removeErr != nil {
			return false, fmt.Sprintf("remove error: %s", removeErr), nil
		}
	}

	if e.registry != nil {
		// Best-effort registry cleanup; ignore not-found errors.
		_ = e.registry.RemoveEntry(proposal.MemoryPath)
	}

	return true, "", nil
}

func (e *Executor) applyRewrite(
	ctx context.Context,
	proposal Proposal,
) (bool, string, error) {
	prompt := fmt.Sprintf(
		"Rewrite this underperforming memory to improve its actionability.\n"+
			"Memory: %s\nDiagnosis: %s\nDetails: %s\n"+
			"Output JSON: {\"principle\": \"...\", \"anti_pattern\": \"...\"}",
		proposal.MemoryPath, proposal.Diagnosis, string(proposal.Details),
	)

	return e.llmRewrite(ctx, proposal, prompt)
}

func (e *Executor) applyStaleUpdate(
	ctx context.Context,
	proposal Proposal,
) (bool, string, error) {
	prompt := fmt.Sprintf(
		"Rewrite this stale memory to be current and actionable.\n"+
			"Memory: %s\nDiagnosis: %s\nDetails: %s\n"+
			"Output JSON: {\"content\": \"...\", \"principle\": \"...\"}",
		proposal.MemoryPath, proposal.Diagnosis, string(proposal.Details),
	)

	return e.llmRewrite(ctx, proposal, prompt)
}

func (e *Executor) confirm(
	proposal Proposal,
	updates map[string]any,
) (bool, error) {
	if e.confirmer == nil {
		// No confirmer means auto-approve (--yes mode).
		return true, nil
	}

	preview := fmt.Sprintf(
		"[%s] %s: %s\nAction: %s",
		proposal.Quadrant, proposal.MemoryPath,
		proposal.Diagnosis, proposal.Action,
	)

	if updates != nil {
		//nolint:errchkjson // preview formatting; non-critical
		updateJSON, _ := json.MarshalIndent(updates, "  ", "  ")
		preview += "\nChanges:\n  " + string(updateJSON)
	}

	confirmed, err := e.confirmer.Confirm(preview)
	if err != nil {
		return false, fmt.Errorf("confirming proposal: %w", err)
	}

	return confirmed, nil
}

// llmRewrite calls the LLM with the given prompt, confirms changes, and applies them.
func (e *Executor) llmRewrite(
	ctx context.Context,
	proposal Proposal,
	prompt string,
) (bool, string, error) {
	response, err := e.llmCaller.Call(ctx, prompt)
	if err != nil {
		return false, fmt.Sprintf("llm error: %s", err), nil
	}

	var updates map[string]any
	unmarshalErr := json.Unmarshal([]byte(response), &updates)
	if unmarshalErr != nil {
		return false, "invalid llm response", nil
	}

	confirmed, confirmErr := e.confirm(proposal, updates)
	if confirmErr != nil {
		return false, "", confirmErr
	}

	if !confirmed {
		return false, "", nil
	}

	rewriteErr := e.rewriter.Rewrite(proposal.MemoryPath, updates)
	if rewriteErr != nil {
		return false, fmt.Sprintf("rewrite error: %s", rewriteErr), nil
	}

	return true, "", nil
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// LLMCaller generates rewrites via an LLM.
type LLMCaller interface {
	Call(ctx context.Context, prompt string) (string, error)
}

// MemoryRemover deletes a memory file.
type MemoryRemover interface {
	Remove(path string) error
}

// MemoryRewriter updates specific fields of a memory TOML file.
type MemoryRewriter interface {
	Rewrite(path string, updates map[string]any) error
}

// RegistryUpdater removes entries from the instruction registry.
type RegistryUpdater interface {
	RemoveEntry(id string) error
}

// IngestProposals parses a JSON array of proposals, skipping invalid entries.
func IngestProposals(data []byte) ([]Proposal, error) {
	var raw []Proposal

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("parsing proposals: %w", err)
	}

	valid := make([]Proposal, 0, len(raw))

	for idx := range raw {
		if raw[idx].MemoryPath == "" || raw[idx].Quadrant == "" || raw[idx].Action == "" {
			continue
		}

		valid = append(valid, raw[idx])
	}

	return valid, nil
}

// WithConfirmer sets the user confirmation handler.
func WithConfirmer(c Confirmer) ExecutorOption {
	return func(e *Executor) { e.confirmer = c }
}

// WithLLMCaller2 sets the LLM caller for rewrites.
func WithLLMCaller2(c LLMCaller) ExecutorOption {
	return func(e *Executor) { e.llmCaller = c }
}

// WithRegistry sets the registry updater.
func WithRegistry(r RegistryUpdater) ExecutorOption {
	return func(e *Executor) { e.registry = r }
}

// WithRemover sets the memory remover.
func WithRemover(r MemoryRemover) ExecutorOption {
	return func(e *Executor) { e.remover = r }
}

// WithRewriter sets the memory rewriter.
func WithRewriter(r MemoryRewriter) ExecutorOption {
	return func(e *Executor) { e.rewriter = r }
}
