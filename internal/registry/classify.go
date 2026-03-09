package registry

// Quadrant represents a position in the effectiveness/surfacing matrix.
type Quadrant string

// Quadrant values.
const (
	Working      Quadrant = "Working"
	Leech        Quadrant = "Leech"
	HiddenGem    Quadrant = "Hidden Gem"
	Noise        Quadrant = "Noise"
	Insufficient Quadrant = "Insufficient"
)

// alwaysLoadedSources are source types that are always loaded into context
// (not gated by surfacing). They can only be Working or Leech.
var alwaysLoadedSources = map[string]bool{
	"claude-md": true,
	"memory-md": true,
}

// defaultSurfacingThreshold is the minimum surfaced_count to be
// considered "often surfaced" when no explicit threshold is provided.
const defaultSurfacingThreshold = 3

// defaultEffectivenessThreshold is the minimum effectiveness percentage
// to be considered "high follow-through".
const defaultEffectivenessThreshold = 50.0

// Classify assigns a quadrant to an instruction entry.
// Always-loaded sources (claude-md, memory-md) get binary classification:
// Working or Leech only.
// Configurable thresholds control the surfacing/effectiveness boundaries.
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
