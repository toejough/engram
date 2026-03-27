package adapt_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAnalyzeMaintenanceOutcomes_ProposesAlternative(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
					{Action: "rewrite", EffectivenessBefore: 25.0, EffectivenessAfter: 20.0, Measured: true},
				},
			},
		},
		{
			Path: "mem-2.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 30.0, EffectivenessAfter: 28.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{
		MinMeasuredOutcomes: 3,
		MinSuccessRate:      0.4,
	}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)

	// 3 rewrites, all degraded, success rate = 0/3 = 0.0
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Dimension).To(Equal(policy.DimensionMaintenance))
	g.Expect(proposals[0].Directive).To(ContainSubstring("rewrite"))
	g.Expect(proposals[0].Status).To(Equal(policy.StatusProposed))
}

func TestAnalyzeMaintenanceOutcomes_SkipsInsufficientData(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{MinMeasuredOutcomes: 3, MinSuccessRate: 0.4}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeMaintenanceOutcomes_NoProposalWhenSuccessful(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "broaden_keywords", EffectivenessBefore: 20.0, EffectivenessAfter: 45.0, Measured: true},
					{Action: "broaden_keywords", EffectivenessBefore: 30.0, EffectivenessAfter: 50.0, Measured: true},
					{Action: "broaden_keywords", EffectivenessBefore: 25.0, EffectivenessAfter: 40.0, Measured: true},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{MinMeasuredOutcomes: 3, MinSuccessRate: 0.4}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)
	g.Expect(proposals).To(BeEmpty())
}

func TestAnalyzeMaintenanceOutcomes_IgnoresUnmeasured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	records := []adapt.MeasurableRecord{
		{
			Path: "mem-1.toml",
			Record: memory.MemoryRecord{
				MaintenanceHistory: []memory.MaintenanceAction{
					{Action: "rewrite", EffectivenessBefore: 20.0, EffectivenessAfter: 15.0, Measured: true},
					{Action: "rewrite", EffectivenessBefore: 25.0, Measured: false},
					{Action: "rewrite", EffectivenessBefore: 30.0, Measured: false},
				},
			},
		},
	}

	cfg := adapt.MaintenanceAnalysisConfig{MinMeasuredOutcomes: 3, MinSuccessRate: 0.4}

	proposals := adapt.AnalyzeMaintenanceOutcomes(records, cfg)
	g.Expect(proposals).To(BeEmpty())
}
