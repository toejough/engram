package interview

import "sort"

// InterviewQuestion represents a question to ask the user with optional context.
type InterviewQuestion struct {
	ID       string   // Question identifier from KeyQuestion
	Text     string   // Question text
	Priority Priority // Priority level
	Context  string   // Optional context from gathered information
}

// SelectQuestions determines which questions to ask based on gap analysis.
//
// The function applies adaptive interview logic:
// - Small gap (≥80%): 1-2 confirmation questions for critical unanswered items
// - Medium gap (50-79%): 3-5 clarification questions prioritizing critical then important
// - Large gap (<50%): 6+ questions covering all unanswered questions
//
// Questions reference gathered context where relevant and skip topics that are
// fully answered by context.
//
// Parameters:
//   - keyQuestions: All key questions that could be asked
//   - gapAnalysis: Results from CalculateGap showing coverage and unanswered questions
//   - gathered: Map of question IDs to context strings for answered questions
//
// Returns:
//   - Ordered list of questions to ask the user
func SelectQuestions(keyQuestions []KeyQuestion, gapAnalysis GapAnalysis, gathered map[string]string) []InterviewQuestion {
	// Build set of unanswered questions for O(1) lookup
	unansweredSet := make(map[string]bool)
	for _, id := range gapAnalysis.UnansweredQuestions {
		unansweredSet[id] = true
	}

	// Filter key questions to only include unanswered ones
	unansweredKeyQuestions := make([]KeyQuestion, 0)

	for _, kq := range keyQuestions {
		if unansweredSet[kq.ID] {
			unansweredKeyQuestions = append(unansweredKeyQuestions, kq)
		}
	}

	// Sort by priority: Critical > Important > Optional
	sortByPriority(unansweredKeyQuestions)

	// Determine max questions to ask based on gap size
	maxQuestions := determineMaxQuestions(gapAnalysis.GapSize, len(unansweredKeyQuestions))

	// Take up to maxQuestions from sorted list
	questionsToAsk := unansweredKeyQuestions
	if len(questionsToAsk) > maxQuestions {
		questionsToAsk = questionsToAsk[:maxQuestions]
	}

	// Build InterviewQuestion structs with context
	result := make([]InterviewQuestion, len(questionsToAsk))
	for i, kq := range questionsToAsk {
		result[i] = InterviewQuestion{
			ID:       kq.ID,
			Text:     kq.Text,
			Priority: kq.Priority,
			Context:  buildContext(kq.ID, gathered), // Populate context if available
		}
	}

	return result
}

// unexported constants.
const (
	mediumGapMaxQuestions = 5
	mediumGapMinQuestions = 3
	// Question count limits for different gap sizes
	smallGapMaxQuestions = 2
)

// buildContext constructs a context string for a question based on gathered information.
// It looks for context directly associated with the question ID, and also includes
// any other relevant gathered context.
func buildContext(questionID string, gathered map[string]string) string {
	// First, check if there's direct context for this question
	if ctx, ok := gathered[questionID]; ok {
		return ctx
	}

	// If no direct context, check if there's any other gathered context
	// that might be relevant to include
	for _, ctx := range gathered {
		if ctx != "" {
			return ctx
		}
	}

	return ""
}

// determineMaxQuestions returns the maximum number of questions to ask based on gap size.
func determineMaxQuestions(gapSize GapSize, totalUnanswered int) int {
	switch gapSize {
	case GapSizeSmall:
		// Small gap: 1-2 confirmation questions
		if totalUnanswered <= smallGapMaxQuestions {
			return totalUnanswered
		}

		return smallGapMaxQuestions
	case GapSizeMedium:
		// Medium gap: 3-5 clarification questions
		if totalUnanswered <= mediumGapMaxQuestions {
			return max(mediumGapMinQuestions, totalUnanswered)
		}

		return mediumGapMaxQuestions
	case GapSizeLarge:
		// Large gap: all unanswered questions (6+)
		return totalUnanswered
	default:
		return 0
	}
}

// priorityValue returns a numeric value for priority comparison.
func priorityValue(p Priority) int {
	switch p {
	case PriorityCritical:
		return 3
	case PriorityImportant:
		return 2
	case PriorityOptional:
		return 1
	default:
		return 0
	}
}

// sortByPriority sorts questions in-place by priority: Critical > Important > Optional.
func sortByPriority(questions []KeyQuestion) {
	sort.Slice(questions, func(i, j int) bool {
		return priorityValue(questions[i].Priority) > priorityValue(questions[j].Priority)
	})
}
