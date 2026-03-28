package surface

import (
	"strings"
	"time"

	"engram/internal/memory"
)

// Exported constants.
const (
	SuppressionReasonClusterDedup = "cluster_dedup"
	SuppressionReasonCrossSource  = "cross_source"
	SuppressionReasonTranscript   = "transcript"
)

// SuppressionEvent records a single suppression decision (REQ-P4f-4).
type SuppressionEvent struct {
	MemoryID     string    `json:"memoryId"`
	Timestamp    time.Time `json:"timestamp"`
	Reason       string    `json:"reason"`
	SuppressedBy string    `json:"suppressedBy"`
}

// SuppressionStats holds aggregate suppression metrics for a surface invocation (REQ-P4f-5).
type SuppressionStats struct {
	Suppressed int     `json:"suppressed"`
	Surfaced   int     `json:"surfaced"`
	Rate       float64 `json:"rate"`
}

// computeSuppressionStats builds SuppressionStats from event and surfaced counts (REQ-P4f-5).
// Returns nil when both counts are zero.
func computeSuppressionStats(suppressedCount, surfacedCount int) *SuppressionStats {
	if suppressedCount == 0 && surfacedCount == 0 {
		return nil
	}

	rate := 0.0
	total := suppressedCount + surfacedCount

	if total > 0 {
		rate = float64(suppressedCount) / float64(total)
	}

	return &SuppressionStats{
		Suppressed: suppressedCount,
		Surfaced:   surfacedCount,
		Rate:       rate,
	}
}

// matchTranscriptKeyword returns the first keyword found in lowerTranscript, or "".
func matchTranscriptKeyword(keywords []string, lowerTranscript string) string {
	for _, kw := range keywords {
		if strings.Contains(lowerTranscript, strings.ToLower(kw)) {
			return kw
		}
	}

	return ""
}

// suppressByCrossRef removes memories covered by an external source (REQ-P4f-2).
// Fire-and-forget: checker errors leave the memory unsuppressed.
func suppressByCrossRef(
	candidates []*memory.Stored,
	checker CrossRefChecker,
) ([]*memory.Stored, []SuppressionEvent) {
	if checker == nil {
		return candidates, nil
	}

	now := time.Now()

	var events []SuppressionEvent

	filtered := make([]*memory.Stored, 0, len(candidates))

	for _, mem := range candidates {
		covered, source, err := checker.IsCoveredBySource(mem.FilePath)
		if err != nil || !covered {
			filtered = append(filtered, mem)

			continue
		}

		events = append(events, SuppressionEvent{
			MemoryID:     mem.FilePath,
			Timestamp:    now,
			Reason:       SuppressionReasonCrossSource,
			SuppressedBy: source,
		})
	}

	return filtered, events
}

// suppressByTranscript removes memories whose keywords appear in the transcript window (REQ-P4f-3).
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
		matchedKW := matchTranscriptKeyword(mem.Keywords, lower)
		if matchedKW != "" {
			events = append(events, SuppressionEvent{
				MemoryID:     mem.FilePath,
				Timestamp:    now,
				Reason:       SuppressionReasonTranscript,
				SuppressedBy: matchedKW,
			})

			continue
		}

		filtered = append(filtered, mem)
	}

	return filtered, events
}
