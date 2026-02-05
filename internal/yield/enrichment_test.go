package yield_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/yield"
)

// TEST-400 traces: TASK-7 AC-1
// Yield includes [context.gap_analysis] section
func TestValidateContent_GapAnalysisSection(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
subphase = "GATHER"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory", "memory", "context-files"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue(), "Gap analysis section should be valid")
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-401 traces: TASK-7 AC-2
// All required gap_analysis fields are present
func TestValidateContent_GapAnalysisFields(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
subphase = "GATHER"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory", "memory"]
unanswered_critical = ["tech-stack", "scale"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue())

	// Parse and verify fields can be accessed
	yieldFile, parseErr := yield.ParseContent(content)
	g.Expect(parseErr).ToNot(HaveOccurred())
	g.Expect(yieldFile.Context.GapAnalysis).ToNot(BeNil())
	g.Expect(yieldFile.Context.GapAnalysis.TotalKeyQuestions).To(Equal(10))
	g.Expect(yieldFile.Context.GapAnalysis.QuestionsAnswered).To(Equal(7))
	g.Expect(yieldFile.Context.GapAnalysis.CoveragePercent).To(Equal(70.0))
	g.Expect(yieldFile.Context.GapAnalysis.GapSize).To(Equal("medium"))
	g.Expect(yieldFile.Context.GapAnalysis.QuestionCount).To(Equal(4))
	g.Expect(yieldFile.Context.GapAnalysis.Sources).To(Equal([]string{"territory", "memory"}))
	g.Expect(yieldFile.Context.GapAnalysis.UnansweredCritical).To(Equal([]string{"tech-stack", "scale"}))
}

// TEST-402 traces: TASK-7 AC-2
// Missing total_key_questions field fails validation
func TestValidateContent_GapAnalysisMissingTotalKeyQuestions(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].total_key_questions")))
}

// TEST-403 traces: TASK-7 AC-2
// Missing questions_answered field fails validation
func TestValidateContent_GapAnalysisMissingQuestionsAnswered(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].questions_answered")))
}

// TEST-404 traces: TASK-7 AC-2
// Missing coverage_percent field fails validation
func TestValidateContent_GapAnalysisMissingCoveragePercent(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
gap_size = "medium"
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].coverage_percent")))
}

// TEST-405 traces: TASK-7 AC-2
// Missing gap_size field fails validation
func TestValidateContent_GapAnalysisMissingGapSize(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].gap_size")))
}

// TEST-406 traces: TASK-7 AC-2
// Missing question_count field fails validation
func TestValidateContent_GapAnalysisMissingQuestionCount(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].question_count")))
}

// TEST-407 traces: TASK-7 AC-3
// Missing sources field fails validation
func TestValidateContent_GapAnalysisMissingSources(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("missing required field: [context.gap_analysis].sources")))
}

// TEST-408 traces: TASK-7 AC-3
// Sources array correctly lists context gathering mechanisms
func TestValidateContent_GapAnalysisSourcesArray(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory", "memory", "context-files"]
unanswered_critical = []
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue())

	yieldFile, parseErr := yield.ParseContent(content)
	g.Expect(parseErr).ToNot(HaveOccurred())
	g.Expect(yieldFile.Context.GapAnalysis.Sources).To(HaveLen(3))
	g.Expect(yieldFile.Context.GapAnalysis.Sources).To(ContainElements("territory", "memory", "context-files"))
}

// TEST-409 traces: TASK-7 AC-4
// unanswered_critical field is optional
func TestValidateContent_GapAnalysisUnansweredCriticalOptional(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	// Should be valid even without unanswered_critical field
	g.Expect(result.Valid).To(BeTrue())
}

// TEST-410 traces: TASK-7 AC-4
// unanswered_critical lists critical questions not answered
func TestValidateContent_GapAnalysisUnansweredCriticalList(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 8
coverage_percent = 80.0
gap_size = "small"
question_count = 2
sources = ["territory", "memory"]
unanswered_critical = ["tech-stack", "deployment"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue())

	yieldFile, parseErr := yield.ParseContent(content)
	g.Expect(parseErr).ToNot(HaveOccurred())
	g.Expect(yieldFile.Context.GapAnalysis.UnansweredCritical).To(HaveLen(2))
	g.Expect(yieldFile.Context.GapAnalysis.UnansweredCritical).To(ContainElements("tech-stack", "deployment"))
}

// TEST-411 traces: TASK-7 AC-2
// Invalid gap_size value fails validation
func TestValidateContent_GapAnalysisInvalidGapSize(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "huge"
question_count = 4
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("invalid gap_size")))
}

// TEST-412 traces: TASK-7 AC-2
// Valid gap_size values: small, medium, large
func TestValidateContent_GapAnalysisValidGapSizes(t *testing.T) {
	validSizes := []string{"small", "medium", "large"}

	for _, size := range validSizes {
		t.Run("gap_size="+size, func(t *testing.T) {
			g := NewWithT(t)

			content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "` + size + `"
question_count = 4
sources = ["territory"]
`

			result, err := yield.ValidateContent(content)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(result.Valid).To(BeTrue(), "gap_size=%s should be valid", size)
		})
	}
}

// TEST-413 traces: TASK-7 AC-2
// coverage_percent must be between 0 and 100
func TestValidateContent_GapAnalysisCoverageRange(t *testing.T) {
	testCases := []struct {
		name     string
		coverage string
		valid    bool
	}{
		{"zero coverage", "0.0", true},
		{"mid coverage", "50.0", true},
		{"full coverage", "100.0", true},
		{"negative coverage", "-10.0", false},
		{"over 100 coverage", "150.0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = ` + tc.coverage + `
gap_size = "medium"
question_count = 4
sources = ["territory"]
`

			result, err := yield.ValidateContent(content)
			g.Expect(err).ToNot(HaveOccurred())

			if tc.valid {
				g.Expect(result.Valid).To(BeTrue(), "coverage_percent=%s should be valid", tc.coverage)
			} else {
				g.Expect(result.Valid).To(BeFalse(), "coverage_percent=%s should be invalid", tc.coverage)
				g.Expect(result.Errors).To(ContainElement(ContainSubstring("coverage_percent must be between 0 and 100")))
			}
		})
	}
}

// TEST-414 traces: TASK-7 AC-2
// question_count must be non-negative
func TestValidateContent_GapAnalysisQuestionCountNonNegative(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "need-user-input"
timestamp = 2026-02-05T10:00:00Z

[payload]
question = "What technology stack?"

[context]
phase = "arch"
awaiting = "user-response"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70.0
gap_size = "medium"
question_count = -1
sources = ["territory"]
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("question_count must be non-negative")))
}

// TEST-415 traces: TASK-7 AC-5
// Yield validates against schema before output
func TestValidateContent_GapAnalysisCompleteValidation(t *testing.T) {
	g := NewWithT(t)

	// Test a fully valid yield with gap_analysis
	content := `
[yield]
type = "complete"
timestamp = 2026-02-05T10:00:00Z

[payload]
artifact = "docs/architecture.md"
ids_created = ["ARCH-1", "ARCH-2"]
files_modified = ["docs/architecture.md"]

[context]
phase = "arch"
subphase = "complete"

[context.gap_analysis]
total_key_questions = 10
questions_answered = 10
coverage_percent = 100.0
gap_size = "small"
question_count = 0
sources = ["territory", "memory", "context-files", "user-interview"]
unanswered_critical = []
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Valid).To(BeTrue(), "Complete yield with gap_analysis should be valid")
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-416 traces: TASK-7 AC-1
// Gap analysis section is not required for non-interview yields
func TestValidateContent_GapAnalysisOptionalForNonInterview(t *testing.T) {
	g := NewWithT(t)

	content := `
[yield]
type = "complete"
timestamp = 2026-02-05T10:00:00Z

[payload]
artifact = "docs/architecture.md"

[context]
phase = "arch"
subphase = "complete"
`

	result, err := yield.ValidateContent(content)
	g.Expect(err).ToNot(HaveOccurred())
	// Should be valid even without gap_analysis for non-interview yields
	g.Expect(result.Valid).To(BeTrue())
}
