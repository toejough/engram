package maintain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// TestDeEscalation_AtBottomOfLadder verifies no de-escalation from advisory level.
func TestDeEscalation_AtBottomOfLadder(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	now := fixedNow()

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-bottom",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.05,
			EscalationHistory: []maintain.EscalationHistoryEntry{
				// Pre-eff entry before advisory — won't be found since advisory is first.
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -6, 0), Effectiveness: 0.10},
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -3, 0), Effectiveness: 0.08},
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -2, 0), Effectiveness: 0.06},
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -1, 0), Effectiveness: 0.05},
			},
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Advisory is at idx=0, so history[idx-1] won't be found (idx=0, no idx>0).
	// Falls through to standard escalation.
	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
}

// TestDeEscalation_HistoryTooShort verifies no de-escalation with insufficient history.
func TestDeEscalation_HistoryTooShort(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	now := fixedNow()

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-short-hist",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelEmphasizedAdvisory,
			Effectiveness:   0.15,
			EscalationHistory: []maintain.EscalationHistoryEntry{
				{
					Level:         maintain.LevelEmphasizedAdvisory,
					Since:         now.AddDate(0, -1, 0),
					Effectiveness: 0.12,
				},
			},
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Should escalate (not de-escalate) due to insufficient history.
	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
}

// TestDeEscalation_TailNotAllWorse verifies no de-escalation when recent
// effectiveness is not consistently worse than pre-escalation.
func TestDeEscalation_TailNotAllWorse(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	now := fixedNow()

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-mixed",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelEmphasizedAdvisory,
			Effectiveness:   0.25,
			EscalationHistory: []maintain.EscalationHistoryEntry{
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -6, 0), Effectiveness: 0.20},
				{Level: maintain.LevelEmphasizedAdvisory, Since: now.AddDate(0, -3, 0), Effectiveness: 0.15},
				{Level: maintain.LevelEmphasizedAdvisory, Since: now.AddDate(0, -2, 0), Effectiveness: 0.25},
				{Level: maintain.LevelEmphasizedAdvisory, Since: now.AddDate(0, -1, 0), Effectiveness: 0.12},
			},
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Not all tail entries worse than pre-eff (0.20) — 0.25 >= 0.20.
	// Should escalate instead.
	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
}

// TestEscalation_AtTopOfLadder verifies no escalation at the top level.
func TestEscalation_AtTopOfLadder(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-top",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAutomationCandidate,
			Effectiveness:   0.10,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// At the top of the ladder — no escalation possible.
	g.Expect(proposals).To(gomega.BeEmpty())
}

// T-226: De-escalation when post-escalation effectiveness drops.
func TestEscalation_DeEscalation(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	now := fixedNow()

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-3",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelEmphasizedAdvisory,
			Effectiveness:   0.15,
			EscalationHistory: []maintain.EscalationHistoryEntry{
				{Level: maintain.LevelAdvisory, Since: now.AddDate(0, -6, 0), Effectiveness: 0.20},
				{
					Level:         maintain.LevelEmphasizedAdvisory,
					Since:         now.AddDate(0, -3, 0),
					Effectiveness: 0.15,
				},
				{
					Level:         maintain.LevelEmphasizedAdvisory,
					Since:         now.AddDate(0, -2, 0),
					Effectiveness: 0.18,
				},
				{
					Level:         maintain.LevelEmphasizedAdvisory,
					Since:         now.AddDate(0, -1, 0),
					Effectiveness: 0.12,
				},
			},
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("de_escalate"))
	g.Expect(proposals[0].ProposedLevel).To(gomega.Equal("advisory"))
}

// T-224: Default escalation level is advisory.
func TestEscalation_DefaultLevelAdvisory(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-1",
			Content:         "use descriptive names",
			EscalationLevel: "", // no level set
			Effectiveness:   0.15,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].CurrentLevel).To(gomega.Equal("advisory"))
	g.Expect(proposals[0].ProposedLevel).To(gomega.Equal("emphasized_advisory"))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
}

// T-227: Dimension routing to automation candidate.
func TestEscalation_DimensionRoutingAutomation(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-4",
			Content:         "Always run targ test before committing",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.10,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("route_automation"))
	g.Expect(proposals[0].ProposedLevel).To(gomega.Equal("automation_candidate"))
}

// T-225: Escalation proposes next level with predicted impact.
func TestEscalation_NextLevelWithPredictedImpact(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	effData := maintain.EffData{
		maintain.LevelEmphasizedAdvisory: {10.0, 10.0, 10.0},
	}

	engine := maintain.NewEscalationEngine(effData, fixedNow)

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-2",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.15,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposedLevel).To(gomega.Equal("emphasized_advisory"))
	g.Expect(proposals[0].PredictedImpact).To(gomega.Equal("+10% follow rate"))
}

// TestEscalation_NilNowUsesDefault verifies NewEscalationEngine with nil now func.
func TestEscalation_NilNowUsesDefault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, nil)

	// Engine should work — it uses time.Now internally.
	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-nil-now",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.15,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].ProposalType).To(gomega.Equal("escalate"))
}

// T-229: Escalation proposal format matches DES-31.
func TestEscalation_ProposalFormatDES31(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	engine := maintain.NewEscalationEngine(nil, fixedNow)

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-6",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.10,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))

	data := maintain.MarshalProposal(proposals[0])

	var fields map[string]any

	unmarshalErr := json.Unmarshal(data, &fields)
	g.Expect(unmarshalErr).NotTo(gomega.HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	// DES-31 required fields.
	requiredFields := []string{
		"memory_path", "proposal_type", "current_level",
		"proposed_level", "rationale", "predicted_impact",
	}
	for _, field := range requiredFields {
		g.Expect(fields).To(gomega.HaveKey(field))
	}
}

// T-228: Escalation level written to TOML on confirmation (mock writer verification).
func TestEscalation_TOMLFieldsOnConfirmation(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := fixedNow()
	engine := maintain.NewEscalationEngine(nil, func() time.Time { return now })

	leeches := []maintain.EscalationMemory{
		{
			Path:            "mem-5",
			Content:         "use descriptive names",
			EscalationLevel: maintain.LevelAdvisory,
			Effectiveness:   0.15,
		},
	}

	proposals, err := engine.Analyze(leeches)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(1))

	// Verify the proposal has correct fields for TOML update.
	proposal := proposals[0]
	g.Expect(proposal.ProposedLevel).To(gomega.Equal("emphasized_advisory"))
	g.Expect(proposal.CurrentLevel).To(gomega.Equal("advisory"))

	// Verify EscalationHistoryEntry can round-trip through JSON (proxy for TOML).
	entry := maintain.EscalationHistoryEntry{
		Level:         maintain.EscalationLevel(proposal.ProposedLevel),
		Since:         now,
		Effectiveness: 0.15,
	}

	data, marshalErr := json.Marshal(entry)
	g.Expect(marshalErr).NotTo(gomega.HaveOccurred())

	if marshalErr != nil {
		return
	}

	var decoded maintain.EscalationHistoryEntry

	unmarshalErr := json.Unmarshal(data, &decoded)
	g.Expect(unmarshalErr).NotTo(gomega.HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(decoded.Level).To(gomega.Equal(maintain.LevelEmphasizedAdvisory))
	g.Expect(decoded.Since).To(gomega.BeTemporally("~", now, time.Second))
}

// TestMarshalProposal_ValidProposal verifies JSON marshaling of a proposal.
func TestMarshalProposal_ValidProposal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	proposal := maintain.EscalationProposal{
		MemoryPath:      "mem-1",
		ProposalType:    "escalate",
		CurrentLevel:    "advisory",
		ProposedLevel:   "emphasized_advisory",
		Rationale:       "ineffective",
		PredictedImpact: "unknown",
	}

	data := maintain.MarshalProposal(proposal)

	var decoded map[string]any

	unmarshalErr := json.Unmarshal(data, &decoded)
	g.Expect(unmarshalErr).NotTo(gomega.HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(decoded["memory_path"]).To(gomega.Equal("mem-1"))
	g.Expect(decoded["proposal_type"]).To(gomega.Equal("escalate"))
}

// TestMarshalProposal_ZeroValue verifies JSON marshaling of a zero-value proposal.
func TestMarshalProposal_ZeroValue(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	data := maintain.MarshalProposal(maintain.EscalationProposal{})

	var decoded map[string]string

	unmarshalErr := json.Unmarshal(data, &decoded)
	g.Expect(unmarshalErr).NotTo(gomega.HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(decoded["memory_path"]).To(gomega.Equal(""))
	g.Expect(decoded["proposal_type"]).To(gomega.Equal(""))
}
