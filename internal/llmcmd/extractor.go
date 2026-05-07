package llmcmd

import (
	"context"
	"fmt"
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

// SummarizeFindings runs the directive-synthesis prompt over the supplied
// sources. The output is prose advice grounded in the provided evidence.
func (e *Extractor) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	prompt := synthesisPromptHeader + "\n\n"

	if query != "" {
		prompt += "Focus on material relevant to: " + query + "\n\n"
	}

	prompt += "Sources:\n" + content

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return out, nil
}

// unexported constants.
const (
	extractRelevantSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or very lightly paraphrased in service of grammatical ` +
		`correctness and consistency. Return nothing if irrelevant.`
	synthesisPromptHeader = `You are synthesizing engram memory sources into a coherent report for ` +
		`an AI agent.

The sources include facts, behavioral feedback, action records, and outcomes ` +
		`drawn from prior project work. Weave them into a narrative that captures ` +
		`what has been learned and tried.

Then end with directive advice — concrete instructions, warnings, or ` +
		`constraints the reader must apply going forward. Use imperative voice ` +
		`("Do X", "Avoid Y", "Verify Z before W"). cite the specific memory or ` +
		`outcome that grounds each piece of advice. Do not hedge with "consider", ` +
		`"you might", or "think about" — issue clear guidance derived from prior ` +
		`evidence.

Output the report only — no preamble, no list of sources, no JSON.`
)
