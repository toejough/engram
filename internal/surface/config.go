package surface

import "engram/internal/policy"

// SurfaceConfig holds tunable parameters for the surface pipeline.
//
//nolint:revive // "surface.SurfaceConfig" stutter is intentional for clarity at call sites.
type SurfaceConfig struct {
	CandidateCountMin   int
	CandidateCountMax   int
	BM25Threshold       float64
	ColdStartBudget     int
	IrrelevanceHalfLife int
	GateHaikuPrompt     string
	InjectionPreamble   string
}

// ConfigFromPolicy builds a SurfaceConfig from a Policy.
func ConfigFromPolicy(pol policy.Policy) SurfaceConfig {
	return SurfaceConfig{
		CandidateCountMin:   pol.SurfaceCandidateCountMin,
		CandidateCountMax:   pol.SurfaceCandidateCountMax,
		BM25Threshold:       pol.SurfaceBM25Threshold,
		ColdStartBudget:     pol.SurfaceColdStartBudget,
		IrrelevanceHalfLife: pol.SurfaceIrrelevanceHalfLife,
		GateHaikuPrompt:     pol.SurfaceGateHaikuPrompt,
		InjectionPreamble:   pol.SurfaceInjectionPreamble,
	}
}

// DefaultSurfaceConfig returns a SurfaceConfig with default values from policy.Defaults().
func DefaultSurfaceConfig() SurfaceConfig {
	return ConfigFromPolicy(policy.Defaults())
}

// WithSurfaceConfig sets the surface configuration.
func WithSurfaceConfig(cfg SurfaceConfig) SurfacerOption {
	return func(s *Surfacer) { s.config = cfg }
}
