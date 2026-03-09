package registry

// Exported constants.
const (
	HiddenGem    Quadrant = "Hidden Gem"
	Insufficient Quadrant = "Insufficient"
	Leech        Quadrant = "Leech"
	Noise        Quadrant = "Noise"
	Working      Quadrant = "Working"
)

// Quadrant represents a position in the effectiveness/surfacing matrix.
type Quadrant string

// Classify assigns a quadrant to an instruction entry.
// Always-loaded sources (claude-md, memory-md) get binary classification:
// Working or Leech only.
// Configurable thresholds control the surfacing/effectiveness boundaries.
//
//nolint:cyclop // classification matrix
func Classify(
	entry *InstructionEntry,
	surfacingThreshold int,
	effectivenessThreshold float64,
) Quadrant {
	eff := Effectiveness(entry)
	if eff == nil {
		return Insufficient
	}

	highEffectiveness := *eff >= effectivenessThreshold
	isAlwaysLoaded := alwaysLoadedSources[entry.SourceType]

	if isAlwaysLoaded {
		if highEffectiveness {
			return Working
		}

		return Leech
	}

	oftenSurfaced := entry.SurfacedCount >= surfacingThreshold

	switch {
	case oftenSurfaced && highEffectiveness:
		return Working
	case oftenSurfaced && !highEffectiveness:
		return Leech
	case !oftenSurfaced && highEffectiveness:
		return HiddenGem
	default:
		return Noise
	}
}

// unexported variables.
var (
	alwaysLoadedSources = map[string]bool{
		"claude-md": true,
		"memory-md": true,
		"rule":      true,
		"skill":     true,
	}
)
