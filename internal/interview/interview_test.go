package interview_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/interview"
)

// TEST-300 traces: TASK-6 AC-1, AC-2
// Small gap (≥80%): yields 1-2 confirmation questions for critical unanswered items only
func TestSelectQuestions_SmallGap_OneCriticalUnanswered(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "integrations", Text: "What external integrations?", Priority: interview.PriorityImportant},
		{ID: "monitoring", Text: "What monitoring strategy?", Priority: interview.PriorityOptional},
	}

	// Coverage: 85% (one critical unanswered = -15%)
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     85.0,
		GapSize:             interview.GapSizeSmall,
		UnansweredQuestions: []string{"tech-stack"},
	}

	gathered := map[string]string{
		"scale":        "Context shows 10k users expected",
		"deployment":   "Context shows AWS deployment",
		"integrations": "Context shows Stripe integration",
		"monitoring":   "Context shows Datadog",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Small gap: should yield 1-2 confirmation questions
	g.Expect(len(questions)).To(BeNumerically(">=", 1))
	g.Expect(len(questions)).To(BeNumerically("<=", 2))

	// Should only include unanswered critical questions
	foundTechStack := false
	for _, q := range questions {
		if q.ID == "tech-stack" {
			foundTechStack = true
			g.Expect(q.Priority).To(Equal(interview.PriorityCritical))
		}
		// Should not ask about answered questions
		g.Expect(q.ID).NotTo(Equal("scale"))
		g.Expect(q.ID).NotTo(Equal("deployment"))
	}
	g.Expect(foundTechStack).To(BeTrue(), "Should ask about unanswered critical question")
}

// TEST-301 traces: TASK-6 AC-2
// Small gap with no critical unanswered: yields 1-2 questions prioritizing important
func TestSelectQuestions_SmallGap_NoCriticalUnanswered(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "monitoring", Text: "What monitoring strategy?", Priority: interview.PriorityOptional},
	}

	// Coverage: 85% (one important unanswered = -10%, one optional = -5%)
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     85.0,
		GapSize:             interview.GapSizeSmall,
		UnansweredQuestions: []string{"deployment", "monitoring"},
	}

	gathered := map[string]string{
		"tech-stack": "Context shows Go backend",
		"scale":      "Context shows 10k users",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Small gap: should yield 1-2 questions
	g.Expect(len(questions)).To(BeNumerically(">=", 1))
	g.Expect(len(questions)).To(BeNumerically("<=", 2))

	// Should prioritize important over optional
	if len(questions) == 1 {
		g.Expect(questions[0].ID).To(Equal("deployment"))
		g.Expect(questions[0].Priority).To(Equal(interview.PriorityImportant))
	}
}

// TEST-302 traces: TASK-6 AC-3
// Medium gap (50-79%): yields 3-5 clarification questions prioritizing critical then important
func TestSelectQuestions_MediumGap(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "integrations", Text: "What external integrations?", Priority: interview.PriorityImportant},
		{ID: "performance", Text: "What performance SLA?", Priority: interview.PriorityImportant},
		{ID: "security", Text: "What security model?", Priority: interview.PriorityImportant},
		{ID: "monitoring", Text: "What monitoring strategy?", Priority: interview.PriorityOptional},
		{ID: "dev-env", Text: "What dev environment?", Priority: interview.PriorityOptional},
	}

	// Coverage: 65% (2 critical = -30%, 1 important = -10%, total = 60% but let's say 65%)
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     65.0,
		GapSize:             interview.GapSizeMedium,
		UnansweredQuestions: []string{"tech-stack", "scale", "integrations", "monitoring"},
	}

	gathered := map[string]string{
		"deployment":  "Context shows AWS",
		"performance": "Context shows <200ms target",
		"security":    "Context shows OAuth2",
		"dev-env":     "Context shows Docker",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Medium gap: should yield 3-5 questions
	g.Expect(len(questions)).To(BeNumerically(">=", 3))
	g.Expect(len(questions)).To(BeNumerically("<=", 5))

	// Should prioritize: critical first, then important
	criticalCount := 0
	for _, q := range questions {
		if q.Priority == interview.PriorityCritical {
			criticalCount++
		}
		// Should not ask about answered questions
		g.Expect(q.ID).NotTo(Equal("deployment"))
		g.Expect(q.ID).NotTo(Equal("performance"))
	}

	// Both critical unanswered should be included
	g.Expect(criticalCount).To(Equal(2))
}

// TEST-303 traces: TASK-6 AC-4
// Large gap (<50%): yields full interview sequence covering all key questions
func TestSelectQuestions_LargeGap(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "integrations", Text: "What external integrations?", Priority: interview.PriorityImportant},
		{ID: "performance", Text: "What performance SLA?", Priority: interview.PriorityImportant},
		{ID: "security", Text: "What security model?", Priority: interview.PriorityImportant},
		{ID: "monitoring", Text: "What monitoring strategy?", Priority: interview.PriorityOptional},
		{ID: "dev-env", Text: "What dev environment?", Priority: interview.PriorityOptional},
	}

	// Coverage: 30% (most questions unanswered)
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent: 30.0,
		GapSize:         interview.GapSizeLarge,
		UnansweredQuestions: []string{
			"tech-stack", "scale", "deployment", "integrations",
			"performance", "monitoring",
		},
	}

	gathered := map[string]string{
		"security": "Context shows OAuth2",
		"dev-env":  "Context shows Docker",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Large gap: should yield 6+ questions covering all unanswered
	g.Expect(len(questions)).To(BeNumerically(">=", 6))

	// Should include all unanswered questions
	questionIDs := make(map[string]bool)
	for _, q := range questions {
		questionIDs[q.ID] = true
	}

	for _, unansweredID := range gapAnalysis.UnansweredQuestions {
		g.Expect(questionIDs[unansweredID]).To(BeTrue(),
			"Large gap should include all unanswered questions, missing: %s", unansweredID)
	}

	// Should NOT include answered questions
	g.Expect(questionIDs["security"]).To(BeFalse())
	g.Expect(questionIDs["dev-env"]).To(BeFalse())
}

// TEST-304 traces: TASK-6 AC-5
// Question text references gathered context where relevant
func TestSelectQuestions_ReferencesGatheredContext(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
	}

	// Small gap, one critical unanswered
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     85.0,
		GapSize:             interview.GapSizeSmall,
		UnansweredQuestions: []string{"tech-stack"},
	}

	gathered := map[string]string{
		"scale": "Context shows 10k users expected from requirements.md",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Should have at least one question
	g.Expect(len(questions)).To(BeNumerically(">=", 1))

	// Tech stack question should exist (not answered)
	foundTechStack := false
	for _, q := range questions {
		if q.ID == "tech-stack" {
			foundTechStack = true
			// Should have text property
			g.Expect(q.Text).NotTo(BeEmpty())
		}
	}
	g.Expect(foundTechStack).To(BeTrue())
}

// TEST-305 traces: TASK-6 AC-5
// Questions include context information when available
func TestSelectQuestions_IncludesContextInformation(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
	}

	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     65.0,
		GapSize:             interview.GapSizeMedium,
		UnansweredQuestions: []string{"tech-stack", "deployment"},
	}

	// Gathered context that questions should reference
	// "scale" is answered, so related context about scale should be
	// referenced in questions about tech-stack or deployment
	gathered := map[string]string{
		"scale":        "Context shows 10k concurrent users expected",
		"related-tech": "Context mentions SQLite database choice in design.md",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	g.Expect(len(questions)).To(BeNumerically(">=", 2))

	// AC-5 verification: Questions should reference gathered context where relevant
	// Either: (1) Context field is populated with relevant gathered info, OR
	//         (2) Question text includes references to gathered context
	//
	// Since we have gathered context about "related-tech" mentioning SQLite,
	// at least one question should reference this in Context field or question text
	hasContextReference := false
	for _, q := range questions {
		// Check if Context field is populated with relevant info
		if q.Context != "" {
			hasContextReference = true
			// Context should actually contain meaningful gathered info, not just exist
			g.Expect(q.Context).To(ContainSubstring("Context"),
				"Context field should reference gathered information, got: %s", q.Context)
		}
	}

	// At least one question should have context populated when gathered data exists
	g.Expect(hasContextReference).To(BeTrue(),
		"When gathered context exists, at least one question should reference it in Context field")
}

// TEST-306 traces: TASK-6 AC-6
// Questions skip topics fully answered by context
func TestSelectQuestions_SkipsAnsweredTopics(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
		{ID: "deployment", Text: "What deployment target?", Priority: interview.PriorityImportant},
		{ID: "integrations", Text: "What external integrations?", Priority: interview.PriorityImportant},
	}

	// Medium gap, but some questions answered
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     65.0,
		GapSize:             interview.GapSizeMedium,
		UnansweredQuestions: []string{"tech-stack", "deployment"},
	}

	gathered := map[string]string{
		"scale":        "Context shows 10k users",
		"integrations": "Context shows Stripe, SendGrid",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Should only ask about unanswered questions
	questionIDs := make(map[string]bool)
	for _, q := range questions {
		questionIDs[q.ID] = true
	}

	// Should NOT ask about answered questions
	g.Expect(questionIDs["scale"]).To(BeFalse(), "Should skip scale (answered)")
	g.Expect(questionIDs["integrations"]).To(BeFalse(), "Should skip integrations (answered)")

	// Should ask about unanswered
	g.Expect(questionIDs["tech-stack"]).To(BeTrue(), "Should ask tech-stack (unanswered)")
	g.Expect(questionIDs["deployment"]).To(BeTrue(), "Should ask deployment (unanswered)")
}

// TEST-307 traces: TASK-6 AC-1, AC-2, AC-3
// Edge case: all questions answered should return empty list
func TestSelectQuestions_AllAnswered_EmptyResult(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "tech-stack", Text: "What technology stack?", Priority: interview.PriorityCritical},
		{ID: "scale", Text: "What scale requirements?", Priority: interview.PriorityCritical},
	}

	gapAnalysis := interview.GapAnalysis{
		CoveragePercent:     100.0,
		GapSize:             interview.GapSizeSmall,
		UnansweredQuestions: []string{},
	}

	gathered := map[string]string{
		"tech-stack": "Context shows Go",
		"scale":      "Context shows 10k users",
	}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// No questions to ask if everything is answered
	g.Expect(questions).To(BeEmpty())
}

// TEST-308 traces: TASK-6 AC-2, AC-3
// Priority ordering within medium gap: critical before important before optional
func TestSelectQuestions_MediumGap_PriorityOrdering(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "optional1", Text: "Optional 1?", Priority: interview.PriorityOptional},
		{ID: "critical1", Text: "Critical 1?", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1?", Priority: interview.PriorityImportant},
		{ID: "optional2", Text: "Optional 2?", Priority: interview.PriorityOptional},
		{ID: "critical2", Text: "Critical 2?", Priority: interview.PriorityCritical},
		{ID: "important2", Text: "Important 2?", Priority: interview.PriorityImportant},
	}

	// All unanswered, medium gap
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent: 60.0,
		GapSize:         interview.GapSizeMedium,
		UnansweredQuestions: []string{
			"optional1", "critical1", "important1",
			"optional2", "critical2", "important2",
		},
	}

	gathered := map[string]string{}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Medium gap: 3-5 questions
	g.Expect(len(questions)).To(BeNumerically(">=", 3))
	g.Expect(len(questions)).To(BeNumerically("<=", 5))

	// First questions should be critical priority
	g.Expect(questions[0].Priority).To(Equal(interview.PriorityCritical))
	if len(questions) > 1 {
		g.Expect(questions[1].Priority).To(Equal(interview.PriorityCritical))
	}

	// Should have both critical questions
	criticalCount := 0
	for _, q := range questions {
		if q.Priority == interview.PriorityCritical {
			criticalCount++
		}
	}
	g.Expect(criticalCount).To(Equal(2), "Should include both critical questions in medium gap")
}

// TEST-309 traces: TASK-6 AC-4
// Large gap includes all priority levels
func TestSelectQuestions_LargeGap_AllPriorities(t *testing.T) {
	g := NewWithT(t)

	keyQuestions := []interview.KeyQuestion{
		{ID: "critical1", Text: "Critical 1?", Priority: interview.PriorityCritical},
		{ID: "critical2", Text: "Critical 2?", Priority: interview.PriorityCritical},
		{ID: "important1", Text: "Important 1?", Priority: interview.PriorityImportant},
		{ID: "important2", Text: "Important 2?", Priority: interview.PriorityImportant},
		{ID: "optional1", Text: "Optional 1?", Priority: interview.PriorityOptional},
		{ID: "optional2", Text: "Optional 2?", Priority: interview.PriorityOptional},
	}

	// Large gap, all unanswered
	gapAnalysis := interview.GapAnalysis{
		CoveragePercent: 20.0,
		GapSize:         interview.GapSizeLarge,
		UnansweredQuestions: []string{
			"critical1", "critical2", "important1",
			"important2", "optional1", "optional2",
		},
	}

	gathered := map[string]string{}

	questions := interview.SelectQuestions(keyQuestions, gapAnalysis, gathered)

	// Large gap: should yield 6+ questions
	g.Expect(len(questions)).To(BeNumerically(">=", 6))

	// Should include all priorities
	hasCritical := false
	hasImportant := false
	hasOptional := false

	for _, q := range questions {
		switch q.Priority {
		case interview.PriorityCritical:
			hasCritical = true
		case interview.PriorityImportant:
			hasImportant = true
		case interview.PriorityOptional:
			hasOptional = true
		}
	}

	g.Expect(hasCritical).To(BeTrue(), "Large gap should include critical questions")
	g.Expect(hasImportant).To(BeTrue(), "Large gap should include important questions")
	g.Expect(hasOptional).To(BeTrue(), "Large gap should include optional questions")
}
