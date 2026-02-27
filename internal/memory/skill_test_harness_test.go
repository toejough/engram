package memory_test

import (
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestDeriveScenarioFromEmbeddings_CreatesValidScenario(t *testing.T) {
	embeddings := []memory.Embedding{
		{
			ID:      1,
			Content: "Always use TDD for all code changes",
			Metadata: map[string]any{
				"type": "correction",
			},
		},
		{
			ID:      2,
			Content: "Skipped TDD and wrote buggy code",
			Metadata: map[string]any{
				"type": "anti_pattern",
			},
		},
	}

	scenario := memory.DeriveScenarioFromEmbeddings(embeddings)

	if scenario.Description == "" {
		t.Error("Expected non-empty description")
	}

	if scenario.SkillName == "" {
		t.Error("Expected non-empty skill name")
	}

	if scenario.SkillContent == "" {
		t.Error("Expected non-empty skill content")
	}

	if scenario.SuccessCriteria == "" {
		t.Error("Expected non-empty success criteria")
	}

	if scenario.FailureCriteria == "" {
		t.Error("Expected non-empty failure criteria")
	}
}

func TestDeriveScenarioFromEmbeddings_EmptyInput(t *testing.T) {
	scenario := memory.DeriveScenarioFromEmbeddings([]memory.Embedding{})

	// Should still return a valid (though possibly generic) scenario
	if scenario.SkillName == "" {
		t.Error("Expected non-empty skill name even for empty input")
	}
}

func TestEvaluateTestResults_AllPass(t *testing.T) {
	// RED: 3/3 failures, GREEN: 3/3 successes
	redResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
	}
	greenResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)
	if !pass {
		t.Errorf("Expected pass=true, got false. Reasoning: %s", reasoning)
	}

	if reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}
}

func TestEvaluateTestResults_BothFail(t *testing.T) {
	// RED: 1/3 failures, GREEN: 1/3 successes
	redResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
	}
	greenResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)
	if pass {
		t.Errorf("Expected pass=false (both RED and GREEN failed criteria), got true. Reasoning: %s", reasoning)
	}
}

func TestEvaluateTestResults_EdgeCase_N_Minus_1(t *testing.T) {
	// RED: 2/3 failures (meets >=N-1), GREEN: 2/3 successes (meets >=N-1)
	redResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false}, // 1 success
	}
	greenResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true}, // 1 failure
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)
	if !pass {
		t.Errorf("Expected pass=true with >=N-1 criterion, got false. Reasoning: %s", reasoning)
	}
}

func TestEvaluateTestResults_EmptyResults(t *testing.T) {
	// Edge case: no test runs
	pass, reasoning := memory.EvaluateTestResults([]memory.SkillTestResult{}, []memory.SkillTestResult{})
	if pass {
		t.Errorf("Expected pass=false with empty results, got true. Reasoning: %s", reasoning)
	}
}

func TestEvaluateTestResults_GreenFails_TooManyFailures(t *testing.T) {
	// RED: 3/3 failures, GREEN: 1/3 successes (does NOT meet >=N-1)
	redResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
	}
	greenResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)
	if pass {
		t.Errorf("Expected pass=false (GREEN had too many failures), got true. Reasoning: %s", reasoning)
	}

	if reasoning == "" {
		t.Error("Expected non-empty reasoning explaining GREEN failure")
	}
}

func TestEvaluateTestResults_RedFails_TooManySuccesses(t *testing.T) {
	// RED: 1/3 failures (does NOT meet >=N-1), GREEN: 3/3 successes
	redResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: false, FailureCriteriaMet: true},
	}
	greenResults := []memory.SkillTestResult{
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{SuccessCriteriaMet: true, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)
	if pass {
		t.Errorf("Expected pass=false (RED had too many successes), got true. Reasoning: %s", reasoning)
	}

	if reasoning == "" {
		t.Error("Expected non-empty reasoning explaining RED failure")
	}
}
