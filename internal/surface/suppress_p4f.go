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
			Timestamp:    time.Now(),
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

	var events []SuppressionEvent

	filtered := make([]*memory.Stored, 0, len(candidates))

	for _, mem := range candidates {
		matchedKW := matchTranscriptKeyword(mem.Keywords, lower)
		if matchedKW != "" {
			events = append(events, SuppressionEvent{
				MemoryID:     mem.FilePath,
				Timestamp:    time.Now(),
				Reason:       SuppressionReasonTranscript,
				SuppressedBy: matchedKW,
			})

			continue
		}

		filtered = append(filtered, mem)
	}

	return filtered, events
}

// suppressClusterDuplicates removes lower-effectiveness members of linked pairs (REQ-P4f-1).
// When two memories linked via the graph would both surface, keeps the higher-effectiveness one.
//
//nolint:cyclop,funlen // nested link traversal with skip/suppress logic: inherent branching
func suppressClusterDuplicates(
	candidates []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
	linkReader LinkReader,
) ([]*memory.Stored, []SuppressionEvent) {
	if len(candidates) < 2 || linkReader == nil {
		return candidates, nil
	}

	// Build lookup by file path.
	inSet := make(map[string]*memory.Stored, len(candidates))
	for _, mem := range candidates {
		inSet[mem.FilePath] = mem
	}

	var events []SuppressionEvent

	suppressed := make(map[string]bool)

	for _, memA := range candidates {
		if suppressed[memA.FilePath] {
			continue
		}

		links, err := linkReader.GetEntryLinks(memA.FilePath)
		if err != nil {
			continue
		}

		for _, link := range links {
			memB, ok := inSet[link.Target]
			if !ok || suppressed[link.Target] {
				continue
			}

			scoreA := effectivenessScoreFor(memA.FilePath, effectiveness)
			scoreB := effectivenessScoreFor(memB.FilePath, effectiveness)

			// Only dedup when one is clearly better — equal scores keep both.
			if scoreA == scoreB {
				continue
			}

			toSuppress, winner := memB, memA
			if scoreB > scoreA {
				toSuppress, winner = memA, memB
			}

			suppressed[toSuppress.FilePath] = true
			events = append(events, SuppressionEvent{
				MemoryID:     toSuppress.FilePath,
				Timestamp:    time.Now(),
				Reason:       SuppressionReasonClusterDedup,
				SuppressedBy: winner.FilePath,
			})
		}
	}

	filtered := make([]*memory.Stored, 0, len(candidates)-len(suppressed))
	for _, mem := range candidates {
		if !suppressed[mem.FilePath] {
			filtered = append(filtered, mem)
		}
	}

	return filtered, events
}
