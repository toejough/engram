package llmcmd

import (
	"context"
	"fmt"
)

const (
	extractRelevantSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or very lightly paraphrased in service of grammatical ` +
		`correctness and consistency. Return nothing if irrelevant.`
)

// Extractor implements recall.Extractor and recall.FindingSummarizer
// by composing a single-prompt call through the underlying Runner.
type Extractor struct {
	runner *Runner
}

// NewExtractor wires a Runner into the Extractor adapter.
func NewExtractor(runner *Runner) *Extractor {
	return &Extractor{runner: runner}
}

// ExtractRelevant composes the existing extract prompt and calls the runner.
func (e *Extractor) ExtractRelevant(ctx context.Context, content, query string) (string, error) {
	prompt := fmt.Sprintf(
		"%s\n\nQuery: %s\n\nContent:\n%s",
		extractRelevantSystemPrompt, query, content,
	)

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("extracting relevant: %w", err)
	}

	return out, nil
}

// SummarizeFindings is wired to the new synthesis prompt added in Phase C.
// For now it uses a temporary shape so phase A is testable in isolation.
func (e *Extractor) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	prompt := fmt.Sprintf(
		"Synthesize the following findings into a coherent report.\n\nQuery: %s\n\nFindings:\n%s",
		query, content,
	)

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return out, nil
}
