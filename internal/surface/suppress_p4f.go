package surface

import (
	"strings"
	"time"

	"engram/internal/memory"
)

// Exported constants.
const (
	SuppressionReasonTranscript = "transcript"
)

// SuppressionEvent records a single suppression decision.
type SuppressionEvent struct {
	MemoryID     string    `json:"memoryId"`
	Timestamp    time.Time `json:"timestamp"`
	Reason       string    `json:"reason"`
	SuppressedBy string    `json:"suppressedBy"`
}

// SuppressionStats holds aggregate suppression metrics for a surface invocation.
type SuppressionStats struct {
	Suppressed int     `json:"suppressed"`
	Surfaced   int     `json:"surfaced"`
	Rate       float64 `json:"rate"`
}

// suppressByTranscript removes memories whose action text appears in the transcript window.
func suppressByTranscript(
	candidates []*memory.Stored,
	transcriptWindow string,
) ([]*memory.Stored, []SuppressionEvent) {
	if transcriptWindow == "" {
		return candidates, nil
	}

	lower := strings.ToLower(transcriptWindow)
	now := time.Now()

	var events []SuppressionEvent

	filtered := make([]*memory.Stored, 0, len(candidates))

	for _, mem := range candidates {
		// Check if the action text appears verbatim in the transcript.
		if mem.Action != "" && strings.Contains(lower, strings.ToLower(mem.Action)) {
			events = append(events, SuppressionEvent{
				MemoryID:     mem.FilePath,
				Timestamp:    now,
				Reason:       SuppressionReasonTranscript,
				SuppressedBy: mem.Action,
			})

			continue
		}

		filtered = append(filtered, mem)
	}

	return filtered, events
}
