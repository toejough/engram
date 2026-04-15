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

// SummarizeFindings produces a structured summary from accumulated findings.
func (s *Summarizer) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	if s.caller == nil {
		return "", ErrNilCaller
	}

	userPrompt := "Query: " + query + "\n\nFindings:\n" + content

	result, err := s.caller.Call(ctx, summarizeFindingsPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return result, nil
}

// unexported constants.
const (
	extractSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or very lightly paraphrased in service of ` +
		`grammatical correctness and consistency. Return nothing if irrelevant.`
	summarizeFindingsPrompt = `Create a structured summary of the following findings ` +
		`relevant to the query. Use markdown headers and bullet points. ` +
		`Preserve specific details, file paths, and code references.`
)
