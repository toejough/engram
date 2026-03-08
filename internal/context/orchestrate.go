package context

import (
	"context"
	"strings"
)

// MaxSummaryBytes is the maximum size of session context summary content
// (excluding the HTML metadata header). Summaries exceeding this are truncated.
const MaxSummaryBytes = 1024

// Orchestrator composes the session continuity pipeline:
// read watermark → extract delta → strip content → summarize → write file.
// All errors are swallowed (fire-and-forget, ARCH-6).
type Orchestrator struct {
	delta      *DeltaReader
	summarizer *Summarizer
	file       *SessionFile
}

// NewOrchestrator creates an Orchestrator with composed components.
func NewOrchestrator(
	delta *DeltaReader,
	summarizer *Summarizer,
	file *SessionFile,
) *Orchestrator {
	return &Orchestrator{
		delta:      delta,
		summarizer: summarizer,
		file:       file,
	}
}

// Update runs the full session continuity pipeline.
// Returns nil always (fire-and-forget per ARCH-6).
func (o *Orchestrator) Update(
	ctx context.Context,
	transcriptPath string,
	sessionID string,
	contextFilePath string,
) error {
	// Read existing context file for watermark.
	existing, err := o.file.Read(contextFilePath)
	if err != nil {
		return nil //nolint:nilerr // fire-and-forget
	}

	offset := existing.Offset

	// Reset offset if session ID changed (new session).
	if existing.SessionID != "" && existing.SessionID != sessionID {
		offset = 0
	}

	// Extract delta from transcript.
	lines, newOffset, err := o.delta.Read(transcriptPath, offset)
	if err != nil {
		return nil //nolint:nilerr // fire-and-forget
	}

	if len(lines) == 0 {
		return nil
	}

	// Strip noisy content.
	stripped := Strip(lines)

	if len(stripped) == 0 {
		return nil
	}

	deltaText := strings.Join(stripped, "\n")

	// Summarize via Haiku.
	summary, err := o.summarizer.Summarize(
		ctx, existing.Summary, deltaText,
	)
	if err != nil {
		return nil //nolint:nilerr // fire-and-forget
	}

	// Cap summary size to avoid bloating CLAUDE.md context.
	if len(summary) > MaxSummaryBytes {
		summary = summary[:MaxSummaryBytes]
	}

	// Write updated context file.
	_ = o.file.Write(contextFilePath, SessionContext{
		Summary:   summary,
		Offset:    newOffset,
		SessionID: sessionID,
	})

	return nil
}
