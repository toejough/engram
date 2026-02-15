package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

type mockTriageExtractor struct {
	responses map[string]string // action -> JSON response
}

func (m *mockTriageExtractor) CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error) {
	// Look for action keyword in user message
	userMsg := ""
	for _, msg := range params.Messages {
		if msg.Role == "user" {
			userMsg = msg.Content
			break
		}
	}

	for action, resp := range m.responses {
		if strings.Contains(userMsg, action) {
			return []byte(resp), nil
		}
	}
	return []byte(`{"valid": true, "rationale": "default pass"}`), nil
}

func TestTriageProposals_FiltersInvalid(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ext := &mockTriageExtractor{
		responses: map[string]string{
			"consolidate": `{"valid": false, "rationale": "different lessons"}`,
			"promote":     `{"valid": true, "rationale": "good candidate"}`,
		},
	}

	proposals := []MaintenanceProposal{
		{Tier: "embeddings", Action: "consolidate", Target: "id1,id2", Reason: "similarity 0.92", Preview: "consolidate content"},
		{Tier: "embeddings", Action: "promote", Target: "id3", Reason: "high retrieval", Preview: "promote content"},
		{Tier: "embeddings", Action: "rewrite", Target: "id4", Reason: "clarity", Preview: "rewrite content"},
	}

	result, err := TriageProposals(context.Background(), proposals, ext, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// consolidate filtered out, promote kept with eval, rewrite passed through (no triage)
	g.Expect(result).To(gomega.HaveLen(2))
	g.Expect(result[0].Action).To(gomega.Equal("promote"))
	g.Expect(result[0].LLMEval).ToNot(gomega.BeNil())
	g.Expect(result[0].LLMEval.HaikuValid).To(gomega.BeTrue())
	g.Expect(result[0].LLMEval.HaikuRationale).To(gomega.Equal("good candidate"))
	g.Expect(result[1].Action).To(gomega.Equal("rewrite"))
	g.Expect(result[1].LLMEval).To(gomega.BeNil()) // no triage for refinements
}

func TestTriageProposals_PassesThroughNonJudgment(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ext := &mockTriageExtractor{}
	proposals := []MaintenanceProposal{
		{Tier: "embeddings", Action: "rewrite", Target: "id1", Reason: "clarity"},
		{Tier: "embeddings", Action: "add-rationale", Target: "id2", Reason: "missing rationale"},
	}

	result, err := TriageProposals(context.Background(), proposals, ext, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result).To(gomega.HaveLen(2))
	g.Expect(result[0].LLMEval).To(gomega.BeNil())
	g.Expect(result[1].LLMEval).To(gomega.BeNil())
}

func TestTriageProposals_AllValid(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ext := &mockTriageExtractor{
		responses: map[string]string{},
	}

	proposals := []MaintenanceProposal{
		{Tier: "embeddings", Action: "consolidate", Target: "id1,id2", Reason: "similarity 0.95", Preview: "consolidate content"},
		{Tier: "skills", Action: "demote", Target: "skill-1", Reason: "too specific", Preview: "demote content"},
	}

	result, err := TriageProposals(context.Background(), proposals, ext, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result).To(gomega.HaveLen(2))
	g.Expect(result[0].LLMEval).ToNot(gomega.BeNil())
	g.Expect(result[0].LLMEval.HaikuValid).To(gomega.BeTrue())
	g.Expect(result[1].LLMEval).ToNot(gomega.BeNil())
}

func TestNeedsLLMTriage(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(needsLLMTriage("consolidate")).To(gomega.BeTrue())
	g.Expect(needsLLMTriage("promote")).To(gomega.BeTrue())
	g.Expect(needsLLMTriage("demote")).To(gomega.BeTrue())
	g.Expect(needsLLMTriage("split")).To(gomega.BeTrue())
	g.Expect(needsLLMTriage("rewrite")).To(gomega.BeFalse())
	g.Expect(needsLLMTriage("add-rationale")).To(gomega.BeFalse())
	g.Expect(needsLLMTriage("prune")).To(gomega.BeFalse())
	g.Expect(needsLLMTriage("decay")).To(gomega.BeFalse())
}

type mockBehavioralExtractor struct {
	response string
}

func (m *mockBehavioralExtractor) CallAPIWithMessages(ctx context.Context, params APIMessageParams) ([]byte, error) {
	return []byte(m.response), nil
}

type mockContextAssembler struct {
	before string
	after  string
}

func (m *mockContextAssembler) AssembleContext(proposal MaintenanceProposal, applied bool) string {
	if applied {
		return m.after
	}
	return m.before
}

func TestBehavioralTest_PopulatesSonnetFields(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ext := &mockBehavioralExtractor{
		response: `{
			"recommend": "skip",
			"confidence": "high",
			"change_analysis": "loses polling advice",
			"preservation_report": [
				{"scenario": "user asks about polling", "preserved": true},
				{"scenario": "user asks about rate limits", "preserved": false, "lost": "polling instruction"}
			]
		}`,
	}

	assembler := &mockContextAssembler{
		before: "before context",
		after:  "after context",
	}

	proposal := MaintenanceProposal{
		Action: "consolidate",
		Tier:   "embeddings",
		Target: "id1,id2",
		LLMEval: &LLMEvalResult{
			HaikuValid:     true,
			HaikuRationale: "valid consolidation",
		},
	}

	result, err := BehavioralTest(context.Background(), proposal, ext, assembler, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(result.LLMEval).ToNot(gomega.BeNil())
	g.Expect(result.LLMEval.SonnetRecommend).To(gomega.Equal("skip"))
	g.Expect(result.LLMEval.SonnetConfidence).To(gomega.Equal("high"))
	g.Expect(result.LLMEval.SonnetSummary).To(gomega.Equal("loses polling advice"))
	g.Expect(result.LLMEval.ScenarioResults).To(gomega.HaveLen(2))
	g.Expect(result.LLMEval.ScenarioResults[0].Prompt).To(gomega.Equal("user asks about polling"))
	g.Expect(result.LLMEval.ScenarioResults[0].Preserved).To(gomega.BeTrue())
	g.Expect(result.LLMEval.ScenarioResults[0].Lost).To(gomega.BeEmpty())
	g.Expect(result.LLMEval.ScenarioResults[1].Prompt).To(gomega.Equal("user asks about rate limits"))
	g.Expect(result.LLMEval.ScenarioResults[1].Preserved).To(gomega.BeFalse())
	g.Expect(result.LLMEval.ScenarioResults[1].Lost).To(gomega.Equal("polling instruction"))
}

func TestBehavioralTest_SkipsWithoutHaikuValid(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ext := &mockBehavioralExtractor{
		response: `{"recommend": "apply", "confidence": "high", "change_analysis": "good", "preservation_report": []}`,
	}

	assembler := &mockContextAssembler{}

	// Proposal without LLMEval
	proposal := MaintenanceProposal{
		Action: "consolidate",
		Tier:   "embeddings",
		Target: "id1,id2",
	}

	result, err := BehavioralTest(context.Background(), proposal, ext, assembler, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result).To(gomega.Equal(proposal)) // Unchanged
	g.Expect(result.LLMEval).To(gomega.BeNil())

	// Proposal with LLMEval but HaikuValid=false
	proposal2 := MaintenanceProposal{
		Action: "consolidate",
		Tier:   "embeddings",
		Target: "id1,id2",
		LLMEval: &LLMEvalResult{
			HaikuValid:     false,
			HaikuRationale: "not valid",
		},
	}

	result2, err := BehavioralTest(context.Background(), proposal2, ext, assembler, nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result2).To(gomega.Equal(proposal2)) // Unchanged
	g.Expect(result2.LLMEval.SonnetRecommend).To(gomega.BeEmpty())
}
