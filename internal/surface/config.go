package surface

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

// DefaultSurfaceConfig returns a SurfaceConfig with default values.
func DefaultSurfaceConfig() SurfaceConfig {
	return SurfaceConfig{
		CandidateCountMin:   defaultCandidateCountMin,
		CandidateCountMax:   defaultCandidateCountMax,
		BM25Threshold:       defaultBM25Threshold,
		ColdStartBudget:     defaultColdStartBudget,
		IrrelevanceHalfLife: defaultIrrelevanceHalfLife,
		InjectionPreamble:   defaultInjectionPreamble,
	}
}

// WithSurfaceConfig sets the surface configuration.
func WithSurfaceConfig(cfg SurfaceConfig) SurfacerOption {
	return func(s *Surfacer) { s.config = cfg }
}

// unexported constants.
const (
	defaultBM25Threshold     = 0.3
	defaultCandidateCountMax = 8
	defaultCandidateCountMin = 3
	defaultColdStartBudget   = 2
	defaultInjectionPreamble = "[engram] Memories — for any relevant memory, call " +
		"`engram show --name <name>` for full details. " +
		"After your turn, call `engram feedback --name <name> --relevant|--irrelevant " +
		"--used|--notused` for each:"
	defaultIrrelevanceHalfLife = 5
)
