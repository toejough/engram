// Package merge implements memory merging for the learn pipeline (UC-33).
package merge

import "context"

// MemoryMerger combines principles from existing and candidate memories.
type MemoryMerger interface {
	MergePrinciples(ctx context.Context, existing, candidate string) (string, error)
}

// LLMMerger uses an LLM to merge principles.
type LLMMerger struct {
	client LLMClient
}

// LLMClient is the interface for LLM calls.
type LLMClient interface {
	Call(ctx context.Context, prompt string) (string, error)
}

// New creates an LLMMerger.
func New(client LLMClient) *LLMMerger {
	return &LLMMerger{client: client}
}

// MergePrinciples calls the LLM to combine two principle texts.
func (m *LLMMerger) MergePrinciples(ctx context.Context, existing, candidate string) (string, error) {
	prompt := "Combine these two memory principles into a single stronger, more specific statement:\n\n" +
		"Existing: " + existing + "\n\n" +
		"Candidate: " + candidate + "\n\n" +
		"Return only the combined principle, no explanation."

	return m.client.Call(ctx, prompt)
}
