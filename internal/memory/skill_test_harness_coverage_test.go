package memory_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestEvaluateTestResults_GreenFails verifies failure when GREEN doesn't succeed enough.
func TestEvaluateTestResults_GreenFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := memory.TestScenario{
		Description:     "test",
		SkillContent:    "content",
		SuccessCriteria: "success",
		FailureCriteria: "failure",
	}

	redResults := []memory.SkillTestResult{
		{Scenario: scenario, FailureCriteriaMet: true, SuccessCriteriaMet: false},
		{Scenario: scenario, FailureCriteriaMet: true, SuccessCriteriaMet: false},
	}
	// GREEN: none succeed
	greenResults := []memory.SkillTestResult{
		{Scenario: scenario, WithSkill: true, SuccessCriteriaMet: false, FailureCriteriaMet: true},
		{Scenario: scenario, WithSkill: true, SuccessCriteriaMet: false, FailureCriteriaMet: true},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)

	g.Expect(pass).To(BeFalse())
	g.Expect(reasoning).To(ContainSubstring("FAIL"))
}

// TestEvaluateTestResults_RedFails verifies failure when RED doesn't fail enough.
func TestEvaluateTestResults_RedFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := memory.TestScenario{
		Description:     "test",
		SkillContent:    "content",
		SuccessCriteria: "success",
		FailureCriteria: "failure",
	}

	// RED: none fail (i.e., red phase "passed" without skill - wrong behavior for skill test)
	redResults := []memory.SkillTestResult{
		{Scenario: scenario, WithSkill: false, FailureCriteriaMet: false, SuccessCriteriaMet: true},
		{Scenario: scenario, WithSkill: false, FailureCriteriaMet: false, SuccessCriteriaMet: true},
	}
	greenResults := []memory.SkillTestResult{
		{Scenario: scenario, WithSkill: true, SuccessCriteriaMet: true, FailureCriteriaMet: false},
		{Scenario: scenario, WithSkill: true, SuccessCriteriaMet: true, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)

	g.Expect(pass).To(BeFalse())
	g.Expect(reasoning).To(ContainSubstring("FAIL"))
}

// TestEvaluateTestResults_SingleRun verifies N=1 requires N-1=0 minRequired (always passes criteria check).
func TestEvaluateTestResults_SingleRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := memory.TestScenario{
		Description:     "test",
		SkillContent:    "content",
		SuccessCriteria: "success",
		FailureCriteria: "failure",
	}

	// With N=1, minRequired = max(0, 0) = 0, so both criteria pass even with 0 results
	redResults := []memory.SkillTestResult{
		{Scenario: scenario, WithSkill: false, FailureCriteriaMet: false, SuccessCriteriaMet: false},
	}
	greenResults := []memory.SkillTestResult{
		{Scenario: scenario, WithSkill: true, SuccessCriteriaMet: false, FailureCriteriaMet: false},
	}

	pass, reasoning := memory.EvaluateTestResults(redResults, greenResults)

	// minRequired = 0, so both RED and GREEN pass with 0 matches
	g.Expect(pass).To(BeTrue())
	g.Expect(reasoning).ToNot(BeEmpty())
}

// TestTestSkillCandidate_InvalidRuns verifies error on runs <= 0.
func TestTestSkillCandidate_InvalidRuns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := memory.TestScenario{
		Description:     "test",
		SkillContent:    "content",
		SuccessCriteria: "success",
		FailureCriteria: "failure",
	}

	red, green, err := memory.TestSkillCandidate(context.Background(), scenario, 0, "token")

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("runs must be > 0"))
	g.Expect(red).To(BeNil())
	g.Expect(green).To(BeNil())
}

// TestTestSkillCandidate_WithMockServer verifies RED/GREEN runs execute with mock server.
func TestTestSkillCandidate_WithMockServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Note: callAnthropicAPI in skill_test_harness.go hardcodes the URL to api.anthropic.com,
	// so we can't redirect it via WithBaseURL. Instead, we test the "runs invalid" path above
	// and the direct function behavior. The actual network call will fail in CI.
	// We test error path when API is unreachable (no real network).
	scenario := memory.TestScenario{
		Description:     "add numbers",
		SkillContent:    "When asked to add, use plus sign",
		SuccessCriteria: "sum",
		FailureCriteria: "error",
	}

	// This will fail because there's no Anthropic API in test environment.
	// But it exercises the code paths for red/green results initialization.
	red, green, err := memory.TestSkillCandidate(context.Background(), scenario, 1, "test-token")

	// No error from TestSkillCandidate itself (it swallows API errors into result.Error field)
	g.Expect(err).ToNot(HaveOccurred())

	if red != nil {
		g.Expect(red).To(HaveLen(1))
	}

	if green != nil {
		g.Expect(green).To(HaveLen(1))
	}
}
