package context

import (
	"context"
)

// Summarizer uses a HaikuClient to produce rolling context summaries.
type Summarizer struct {
	client HaikuClient
}

// NewSummarizer creates a Summarizer. Pass nil client to disable API calls.
func NewSummarizer(client HaikuClient) *Summarizer {
	return &Summarizer{client: client}
}

// Summarize produces an updated summary from a previous summary and new delta.
// Returns the previous summary unchanged when:
//   - delta is empty (no new content)
//   - client is nil (no API token configured)
//   - API call fails (graceful degradation)
func (s *Summarizer) Summarize(ctx context.Context, previousSummary, delta string) (string, error) {
	if delta == "" {
		return previousSummary, nil
	}

	if s.client == nil {
		return previousSummary, nil
	}

	result, err := s.client.Summarize(ctx, previousSummary, delta)
	if err != nil {
		return previousSummary, nil //nolint:nilerr // graceful degradation: keep previous summary
	}

	return result, nil
}
