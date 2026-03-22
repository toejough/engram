package recall

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	ErrNilCaller = errors.New("haiku caller is nil")
)

// HaikuCaller calls the Haiku API for summarization/extraction.
type HaikuCaller interface {
	Call(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// Summarizer extracts relevant content from session transcripts via LLM.
type Summarizer struct {
	caller HaikuCaller
}

// NewSummarizer creates a Summarizer with the given HaikuCaller.
func NewSummarizer(caller HaikuCaller) *Summarizer {
	return &Summarizer{caller: caller}
}

// ExtractRelevant extracts content relevant to a specific query from transcript content.
func (s *Summarizer) ExtractRelevant(ctx context.Context, content, query string) (string, error) {
	if s.caller == nil {
		return "", ErrNilCaller
	}

	userPrompt := "Query: " + query + "\n\nTranscript:\n" + content

	result, err := s.caller.Call(ctx, extractSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("extracting relevant: %w", err)
	}

	return result, nil
}

// unexported constants.
const (
	extractSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or tightly paraphrased. Return nothing if irrelevant.`
)
