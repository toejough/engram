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

// Summarizer produces session summaries or extracts relevant content.
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

// Summarize produces a concise summary of session transcript content.
func (s *Summarizer) Summarize(ctx context.Context, content string) (string, error) {
	if s.caller == nil {
		return "", ErrNilCaller
	}

	result, err := s.caller.Call(ctx, summarizeSystemPrompt, content)
	if err != nil {
		return "", fmt.Errorf("summarizing: %w", err)
	}

	return result, nil
}

// unexported constants.
const (
	extractSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or tightly paraphrased. Return nothing if irrelevant.`
	summarizeSystemPrompt = `Summarize these session transcripts for someone resuming work on this project.
Prioritize in this order:
1. What was being worked on and current status
2. Open questions and blockers
3. Key decisions made and why
4. What was attempted but didn't work

Use plain text. No emoji. No markdown headers or formatting.
Keep it under 1500 bytes — concise sentences, not a report.`
)
