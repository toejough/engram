package memory

import (
	"fmt"
	"io"
	"time"
)

// LearningItem represents a single learning extracted from a session.
type LearningItem struct {
	Type       string
	Content    string
	Confidence float64
}

// SessionSummary contains all data for a session-end summary.
type SessionSummary struct {
	SessionID          string
	ExtractedAt        time.Time
	Learnings          []LearningItem
	RetrievalsCount    int
	RetrievalsRelevant int
	SkillCandidates    []string
	CLAUDEMDDemotions  []string
	SkillRefinements   []string
}

// PrintSessionSummary outputs a formatted session summary to the provided writer.
func PrintSessionSummary(summary SessionSummary, w io.Writer) {
	fmt.Fprintln(w, "── Learning Summary ──────────────────────")

	// Print extracted learnings
	if len(summary.Learnings) == 0 {
		fmt.Fprintln(w, "No new learnings extracted")
	} else {
		fmt.Fprintln(w, "Extracted:")
		for _, learning := range summary.Learnings {
			fmt.Fprintf(w, "  • %s: \"%s\" (confidence: %.1f)\n",
				learning.Type, learning.Content, learning.Confidence)
		}
	}

	// Print retrieval statistics if present
	if summary.RetrievalsCount > 0 {
		fmt.Fprintln(w)
		filtered := summary.RetrievalsCount - summary.RetrievalsRelevant
		fmt.Fprintf(w, "Retrievals: %d this session (%d relevant, %d filtered)\n",
			summary.RetrievalsCount, summary.RetrievalsRelevant, filtered)
	}

	// Print pending optimization items
	hasPendingOptimization := len(summary.SkillCandidates) > 0 ||
		len(summary.CLAUDEMDDemotions) > 0 ||
		len(summary.SkillRefinements) > 0

	if hasPendingOptimization {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Pending optimization:")

		for _, candidate := range summary.SkillCandidates {
			fmt.Fprintf(w, "  • skill candidate: \"%s\" — run `projctl memory optimize` to compile\n", candidate)
		}

		for _, demotion := range summary.CLAUDEMDDemotions {
			fmt.Fprintf(w, "  • CLAUDE.md demotion: \"%s\" — run `projctl memory optimize` to migrate\n", demotion)
		}

		for _, refinement := range summary.SkillRefinements {
			fmt.Fprintf(w, "  • skill refinement: \"%s\" — retrieved but correction followed\n", refinement)
		}
	}

	fmt.Fprintln(w, "──────────────────────────────────────────")
}
