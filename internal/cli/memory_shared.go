package cli

import (
	"context"

	"engram/internal/memory"
)

// memorySchemaVersion is the current version of the memory record schema.
const memorySchemaVersion = 2

// typeFact is the memory type identifier for factual statements.
const typeFact = "fact"

// typeFeedback is the memory type identifier for behavioral feedback.
const typeFeedback = "feedback"

// llmCaller calls an LLM with a model, system prompt, and user prompt.
type llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// memoryLister lists all stored memories.
type memoryLister interface {
	ListAllMemories(dataDir string) ([]*memory.Stored, error)
}
