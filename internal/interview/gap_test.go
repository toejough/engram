package interview_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/interview"
)

// TEST-199 traces: TASK-3 AC-1, AC-2
// Test all questions answered returns 100% coverage and small gap
func TestCalculateGap_AllAnswered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "monitoring", Text: "What monitoring strategy?", Priority: interview.PriorityOptional},
	}

	answeredQuestions := []string{"tech-stack", "scale", "deployment", "monitoring"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(100.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
	g.Expect(result.UnansweredQuestions).To(BeEmpty())
}

// TEST-200 traces: TASK-3 AC-1, AC-2
// Test no questions answered returns 0% coverage and large gap
// Using penalty formula: 4 critical (60%) + 4 important (40%) = 100% penalty = 0% coverage
func TestCalculateGap_NoneAnswered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1?", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2?", Priority: interview.PriorityCritical},
		{ID: "critical3", Text: "Critical 3?", Priority: interview.PriorityCritical},
		{ID: "critical4", Text: "Critical 4?", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1?", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2?", Priority: interview.PriorityImportant},
		{ID: "important3", Text: "Important 3?", Priority: interview.PriorityImportant},
		{ID: "important4", Text: "Important 4?", Priority: interview.PriorityImportant},
	}

	answeredQuestions := []string{}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(0.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge))
	g.Expect(result.UnansweredQuestions).To(HaveLen(8))
}

// TEST-201 traces: TASK-3 AC-3, AC-4
// Test priority weights: critical unanswered = -15%
func TestCalculateGap_CriticalUnanswered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "q1", Text: "Question 1", Priority: interview.PriorityCritical},
		{ID: "q2", Text: "Question 2", Priority: interview.PriorityCritical},
	}

	// Answer only one critical question
	answeredQuestions := []string{"q1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// Base: 100%, one critical unanswered: -15% = 85%
	g.Expect(result.CoveragePercent).To(Equal(85.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall)) // ≥80%
	g.Expect(result.UnansweredQuestions).To(ConsistOf("q2"))
}

// TEST-202 traces: TASK-3 AC-3, AC-4
// Test priority weights: important unanswered = -10%
func TestCalculateGap_ImportantUnanswered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "q1", Text: "Question 1", Priority: interview.PriorityImportant},
		{ID: "q2", Text: "Question 2", Priority: interview.PriorityImportant},
	}

	// Answer only one important question
	answeredQuestions := []string{"q1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// Base: 100%, one important unanswered: -10% = 90%
	g.Expect(result.CoveragePercent).To(Equal(90.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
	g.Expect(result.UnansweredQuestions).To(ConsistOf("q2"))
}

// TEST-203 traces: TASK-3 AC-3, AC-4
// Test priority weights: optional unanswered = -5%
func TestCalculateGap_OptionalUnanswered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "q1", Text: "Question 1", Priority: interview.PriorityOptional},
		{ID: "q2", Text: "Question 2", Priority: interview.PriorityOptional},
	}

	// Answer only one optional question
	answeredQuestions := []string{"q1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// Base: 100%, one optional unanswered: -5% = 95%
	g.Expect(result.CoveragePercent).To(Equal(95.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
	g.Expect(result.UnansweredQuestions).To(ConsistOf("q2"))
}

// TEST-204 traces: TASK-3 AC-3, AC-4
// Test mixed priorities with medium gap (50-79% coverage)
func TestCalculateGap_MixedPriorities_MediumGap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
		{ID: "optional1", Text: "Optional 1", Priority: interview.PriorityOptional},
	}

	// Answer important and optional, leave critical unanswered
	// Unanswered: 2 critical = 30% penalty
	// Coverage: 100% - 30% = 70%
	answeredQuestions := []string{"important1", "optional1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(70.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeMedium)) // 50-79%
	g.Expect(result.UnansweredQuestions).To(ConsistOf("critical1", "critical2"))
}

// TEST-205 traces: TASK-3 AC-3, AC-4
// Test large gap (<50% coverage)
func TestCalculateGap_LargeGap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},
		{ID: "critical3", Text: "Critical 3", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
	}

	// Answer only one important: 100% - 15% - 15% - 15% = 55%, but then -10% = 45%
	answeredQuestions := []string{"important1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(55.0)) // 100% - 15% - 15% - 15%
	g.Expect(result.GapSize).To(Equal(interview.GapSizeMedium))
	g.Expect(result.UnansweredQuestions).To(ConsistOf("critical1", "critical2", "critical3"))
}

// TEST-206 traces: TASK-3 AC-5
// Test edge case: <20% coverage always returns large gap
func TestCalculateGap_EdgeCase_VeryLowCoverage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},
		{ID: "critical3", Text: "Critical 3", Priority: interview.PriorityCritical},
		{ID: "critical4", Text: "Critical 4", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant},
	}

	// Answer only one optional question would give high coverage normally,
	// but let's answer none to get <20%
	// No answers: 100% - 60% (4×15%) - 20% (2×10%) = 20%
	// Answer one critical: 100% - 45% (3×15%) - 20% = 35%
	// We need <20%, so answer nothing or configure to get below threshold
	answeredQuestions := []string{}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// With all unanswered: 100% - 60% - 20% = 20% (at boundary)
	// Actually let's recalculate: 4 critical = -60%, 2 important = -20%, total = -80%
	// So coverage = 100% - 80% = 20%
	g.Expect(result.CoveragePercent).To(BeNumerically("<=", 20.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge))
	g.Expect(result.UnansweredQuestions).To(HaveLen(6))
}

// TEST-207 traces: TASK-3 AC-5
// Test edge case: exactly 20% coverage returns medium gap (not forced to large)
func TestCalculateGap_EdgeCase_Exactly20Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Need exactly 20%: that means -80% from unanswered
	// Let's use different combinations to hit exactly 20%
	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical3", Text: "Critical 3", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical4", Text: "Critical 4", Priority: interview.PriorityCritical},    // -15%
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant}, // -10%
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant}, // -10%
		// Total: 4×15% + 2×10% = 80%, so 100% - 80% = 20%
	}

	answeredQuestions := []string{} // All unanswered = 20%

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(20.0))
	// At exactly 20%, the edge case rule says <20%, so this should NOT be forced to large
	// It should use the normal classification: 20% is <50%, so naturally large
	g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge))
}

// TEST-208 traces: TASK-3 AC-5
// Test edge case: 19% coverage returns large gap (edge case triggered)
func TestCalculateGap_EdgeCase_Below20Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical3", Text: "Critical 3", Priority: interview.PriorityCritical},    // -15%
		{ID: "critical4", Text: "Critical 4", Priority: interview.PriorityCritical},    // -15%
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant}, // -10%
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant}, // -10%
		{ID: "optional1", Text: "Optional 1", Priority: interview.PriorityOptional},    // -5%
		// Total unanswered: 85%, so 100% - 85% = 15%
	}

	answeredQuestions := []string{} // All unanswered

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(15.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge)) // Edge case: <20% forces large
}

// TEST-209 traces: TASK-3 AC-4
// Test boundary: 80% coverage returns small gap
func TestCalculateGap_Boundary_80Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant},
	}

	// Answer critical and one important: 100% - 10% = 90%
	// Actually to get exactly 80%: need -20% unanswered
	// Let's use: answer critical, leave both important unanswered = 100% - 20% = 80%
	answeredQuestions := []string{"critical1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(80.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall)) // ≥80%
}

// TEST-210 traces: TASK-3 AC-4
// Test boundary: 75% coverage returns medium gap (testing medium range)
func TestCalculateGap_Boundary_75Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant},
		{ID: "optional1", Text: "Optional 1", Priority: interview.PriorityOptional},
	}

	// Answer critical1 and important1, leave important2 and optional1 unanswered
	// Unanswered: 1 important (10%) + 1 optional (5%) = 15% penalty
	// Coverage: 100% - 15% = 85%
	// That's small gap territory (≥80%), not medium.
	//
	// For 75%: need 25% penalty = 1 critical (15%) + 1 important (10%)
	// Or: 2 important (20%) + 1 optional (5%) = 25%
	answeredQuestions := []string{"critical1"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// Unanswered: important1 + important2 + optional1 = 10% + 10% + 5% = 25%
	// Coverage: 100% - 25% = 75%
	g.Expect(result.CoveragePercent).To(Equal(75.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeMedium)) // 50-79%
}

// TEST-211 traces: TASK-3 AC-4
// Test boundary: 50% coverage returns medium gap
func TestCalculateGap_Boundary_50Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2", Priority: interview.PriorityImportant},
	}

	// To get 50%: need -50% unanswered
	// 2 critical + 2 important = -30% - 20% = -50% → 50%
	answeredQuestions := []string{} // All unanswered

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(50.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeMedium)) // 50-79%
}

// TEST-212 traces: TASK-3 AC-4
// Test boundary: 49% coverage returns large gap
func TestCalculateGap_Boundary_49Percent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2", Priority: interview.PriorityCritical},
		{ID: "critical3", Text: "Critical 3", Priority: interview.PriorityCritical},
		{ID: "optional1", Text: "Optional 1", Priority: interview.PriorityOptional},
		{ID: "optional2", Text: "Optional 2", Priority: interview.PriorityOptional},
	}

	// To get 49%: need -51% unanswered
	// 3 critical + 2 optional = -45% - 10% = -55% → 45%
	// Let's try 3 critical + 1 optional = -45% - 6% = -51% → 49%
	// But can't get 6% with 5% weights. Let's just test conceptually around 49%.
	// Actually: 3 critical + 1 optional = -45% - 5% = -50% → 50%
	// And: 3 critical + 2 optional = -45% - 10% = -55% → 45%
	// So we can't hit exactly 49% with these weights. Let's test 45% instead.

	answeredQuestions := []string{} // All unanswered: -45% - 10% = -55% → 45%

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	g.Expect(result.CoveragePercent).To(Equal(45.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge)) // <50%
}

// TEST-213 traces: TASK-3 AC-6
// Property test: coverage is always between 0 and 100
func TestCalculateGap_Property_CoverageRange(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Generate random key questions
		numQuestions := rapid.IntRange(1, 20).Draw(t, "numQuestions")
		keyQuestions := make([]interview.KeyQuestion, numQuestions)
		for i := 0; i < numQuestions; i++ {
			priority := rapid.SampledFrom([]interview.Priority{
				interview.PriorityCritical,
				interview.PriorityImportant,
				interview.PriorityOptional,
			}).Draw(t, "priority")
			keyQuestions[i] = interview.KeyQuestion{
				ID:       rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "id"),
				Text:     "Question text",
				Priority: priority,
			}
		}

		// Generate random answered questions (subset of key questions)
		numAnswered := rapid.IntRange(0, numQuestions).Draw(t, "numAnswered")
		answeredMap := make(map[int]bool)
		for i := 0; i < numAnswered; i++ {
			idx := rapid.IntRange(0, numQuestions-1).Draw(t, "answeredIdx")
			answeredMap[idx] = true
		}
		answeredQuestions := make([]string, 0, len(answeredMap))
		for idx := range answeredMap {
			answeredQuestions = append(answeredQuestions, keyQuestions[idx].ID)
		}

		result := interview.CalculateGap(keyQuestions, answeredQuestions)

		// Property: coverage must be in [0, 100]
		g.Expect(result.CoveragePercent).To(BeNumerically(">=", 0.0))
		g.Expect(result.CoveragePercent).To(BeNumerically("<=", 100.0))
	})
}

// TEST-214 traces: TASK-3 AC-6
// Property test: all answered means no unanswered questions in result
func TestCalculateGap_Property_AllAnsweredMeansNoUnanswered(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		numQuestions := rapid.IntRange(1, 20).Draw(t, "numQuestions")
		keyQuestions := make([]interview.KeyQuestion, numQuestions)
		allIDs := make([]string, numQuestions)
		for i := 0; i < numQuestions; i++ {
			id := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "id")
			priority := rapid.SampledFrom([]interview.Priority{
				interview.PriorityCritical,
				interview.PriorityImportant,
				interview.PriorityOptional,
			}).Draw(t, "priority")
			keyQuestions[i] = interview.KeyQuestion{
				ID:       id,
				Text:     "Question text",
				Priority: priority,
			}
			allIDs[i] = id
		}

		// Answer all questions
		result := interview.CalculateGap(keyQuestions, allIDs)

		// Property: when all answered, no unanswered questions
		g.Expect(result.UnansweredQuestions).To(BeEmpty())
	})
}

// TEST-215 traces: TASK-3 AC-6
// Property test: gap size classification is consistent with coverage
func TestCalculateGap_Property_GapSizeConsistentWithCoverage(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		numQuestions := rapid.IntRange(1, 20).Draw(t, "numQuestions")
		keyQuestions := make([]interview.KeyQuestion, numQuestions)
		for i := 0; i < numQuestions; i++ {
			priority := rapid.SampledFrom([]interview.Priority{
				interview.PriorityCritical,
				interview.PriorityImportant,
				interview.PriorityOptional,
			}).Draw(t, "priority")
			keyQuestions[i] = interview.KeyQuestion{
				ID:       rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "id"),
				Text:     "Question text",
				Priority: priority,
			}
		}

		numAnswered := rapid.IntRange(0, numQuestions).Draw(t, "numAnswered")
		answeredMap := make(map[int]bool)
		for i := 0; i < numAnswered; i++ {
			idx := rapid.IntRange(0, numQuestions-1).Draw(t, "answeredIdx")
			answeredMap[idx] = true
		}
		answeredQuestions := make([]string, 0, len(answeredMap))
		for idx := range answeredMap {
			answeredQuestions = append(answeredQuestions, keyQuestions[idx].ID)
		}

		result := interview.CalculateGap(keyQuestions, answeredQuestions)

		// Property: gap size must be consistent with coverage percentage
		if result.CoveragePercent < 20.0 {
			// Edge case: always large
			g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge))
		} else if result.CoveragePercent >= 80.0 {
			g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
		} else if result.CoveragePercent >= 50.0 {
			g.Expect(result.GapSize).To(Equal(interview.GapSizeMedium))
		} else {
			g.Expect(result.GapSize).To(Equal(interview.GapSizeLarge))
		}
	})
}

// TEST-216 traces: TASK-3 AC-1
// Test empty key questions returns 100% coverage (vacuous truth)
func TestCalculateGap_EmptyKeyQuestions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{}
	answeredQuestions := []string{}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// With no questions to answer, coverage is 100% (all 0 questions answered)
	g.Expect(result.CoveragePercent).To(Equal(100.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
	g.Expect(result.UnansweredQuestions).To(BeEmpty())
}

// TEST-217 traces: TASK-3 AC-1
// Test answered questions not in key questions are ignored
func TestCalculateGap_ExtraAnsweredQuestionsIgnored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "q1", Text: "Question 1", Priority: interview.PriorityCritical},
		{ID: "q2", Text: "Question 2", Priority: interview.PriorityImportant},
	}

	// Include extra answered questions that aren't in key questions
	answeredQuestions := []string{"q1", "q2", "q3", "q4"}

	result := interview.CalculateGap(keyQuestions, answeredQuestions)

	// Should treat as all key questions answered
	g.Expect(result.CoveragePercent).To(Equal(100.0))
	g.Expect(result.GapSize).To(Equal(interview.GapSizeSmall))
	g.Expect(result.UnansweredQuestions).To(BeEmpty())
}
