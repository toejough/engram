// Package interview provides functions for interview gap analysis.
package interview

// Exported constants.
const (
	// GapSizeLarge means <50% coverage - need 6+ questions, full interview.
	GapSizeLarge GapSize = "large"
	// GapSizeMedium means 50-79% coverage - need 3-5 clarification questions.
	GapSizeMedium GapSize = "medium"
	// GapSizeSmall means ≥80% coverage - need 1-2 confirmation questions.
	GapSizeSmall GapSize = "small"
	// PriorityCritical means the question must be answered for any project.
	PriorityCritical Priority = "critical"
	// PriorityImportant means the question is usually needed, occasionally skippable.
	PriorityImportant Priority = "important"
	// PriorityOptional means the question is nice-to-have, often inferable.
	PriorityOptional Priority = "optional"
)

// GapAnalysis holds the results of calculating interview depth.
type GapAnalysis struct {
	CoveragePercent     float64  // Coverage percentage (0-100)
	GapSize             GapSize  // Classification: small, medium, or large
	UnansweredQuestions []string // List of question IDs that are unanswered
}

// GapSize represents the classification of the knowledge gap.
type GapSize string

// KeyQuestion represents a question that should be answered to understand a project phase.
type KeyQuestion struct {
	ID       string   // Unique identifier for the question
	Text     string   // The question text
	Priority Priority // Importance level of the question
}

// Priority represents the importance level of a key question.
type Priority string

// CalculateGap determines interview depth based on context coverage.
//
// The function uses a penalty-based formula: coverage starts at 100% and
// subtracts penalties for each unanswered question based on priority.
//
// Priority weights for unanswered questions:
// - Critical: -15% each
// - Important: -10% each
// - Optional: -5% each
//
// Gap size classification:
// - Small (≥80%): 1-2 confirmation questions needed
// - Medium (50-79%): 3-5 clarification questions needed
// - Large (<50%): 6+ questions needed, full interview
//
// Edge case: When coverage is <20%, always returns large gap regardless of
// normal classification, as both issue description and context are too sparse.
func CalculateGap(keyQuestions []KeyQuestion, answeredQuestions []string) GapAnalysis {
	// Handle empty key questions case (vacuous truth: all 0 questions answered)
	if len(keyQuestions) == 0 {
		return GapAnalysis{
			CoveragePercent:     100.0,
			GapSize:             GapSizeSmall,
			UnansweredQuestions: []string{},
		}
	}

	// Build set of answered questions for O(1) lookup
	answeredSet := make(map[string]bool)
	for _, id := range answeredQuestions {
		answeredSet[id] = true
	}

	// Calculate penalty for unanswered questions
	// Coverage = 100% - sum(penalties for unanswered)
	penalty := 0.0
	unanswered := make([]string, 0)

	for _, kq := range keyQuestions {
		if !answeredSet[kq.ID] {
			penalty += questionPenalty(kq.Priority)
			unanswered = append(unanswered, kq.ID)
		}
	}

	// Calculate coverage as 100% minus penalties
	coverage := 100.0 - penalty

	// Ensure coverage stays within bounds [0, 100]
	if coverage < 0.0 {
		coverage = 0.0
	}

	if coverage > 100.0 {
		coverage = 100.0
	}

	// Classify gap size
	gapSize := classifyGapSize(coverage)

	return GapAnalysis{
		CoveragePercent:     coverage,
		GapSize:             gapSize,
		UnansweredQuestions: unanswered,
	}
}

// classifyGapSize determines the gap size based on coverage percentage.
func classifyGapSize(coverage float64) GapSize {
	// Edge case: <20% coverage always returns large
	if coverage < 20.0 {
		return GapSizeLarge
	}

	// Normal classification
	if coverage >= 80.0 {
		return GapSizeSmall
	}

	if coverage >= 50.0 {
		return GapSizeMedium
	}

	return GapSizeLarge
}

// questionPenalty returns the coverage penalty for an unanswered question.
func questionPenalty(p Priority) float64 {
	switch p {
	case PriorityCritical:
		return 15.0
	case PriorityImportant:
		return 10.0
	case PriorityOptional:
		return 5.0
	default:
		return 0.0
	}
}
