package recall

import (
	"context"
	"fmt"
)

// HaikuCaller calls the Haiku API for summarization/extraction.
type HaikuCaller interface {
	Call(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// NoopSummarizer satisfies SummarizerI without performing any LLM call.
// Used when no API token is configured: extraction returns empty (no match),
// summarization returns empty (caller short-circuits on empty buffer).
type NoopSummarizer struct{}

// ExtractRelevant returns an empty result.
func (NoopSummarizer) ExtractRelevant(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// SummarizeFindings returns an empty result.
func (NoopSummarizer) SummarizeFindings(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// Summarizer extracts relevant content from session transcripts via LLM.
// The HaikuCaller is required; pass NoopSummarizer at the wiring edge when
// no LLM access is configured.
type Summarizer struct {
	caller HaikuCaller
}

// NewSummarizer creates a Summarizer backed by the given HaikuCaller.
// caller must be non-nil; use NoopSummarizer when LLM access is disabled.
func NewSummarizer(caller HaikuCaller) *Summarizer {
	return &Summarizer{caller: caller}
}

// ExtractRelevant extracts content relevant to a specific query from transcript content.
func (s *Summarizer) ExtractRelevant(ctx context.Context, content, query string) (string, error) {
	userPrompt := "Query: " + query + "\n\nTranscript:\n" + content

	result, err := s.caller.Call(ctx, extractSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("extracting relevant: %w", err)
	}

	return result, nil
}

// SummarizeFindings produces a structured summary from accumulated findings.
func (s *Summarizer) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
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
