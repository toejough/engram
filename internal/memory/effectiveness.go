package memory

// Exported constants.
const (
	QuadrantHiddenGem        = "hidden-gem"
	QuadrantInsufficientData = "insufficient-data"
	QuadrantLeech            = "leech"
	QuadrantNoise            = "noise"
	QuadrantWorking          = "working"
)

// Effectiveness returns the ratio of followed evaluations to total evaluations.
// Returns 0.0 if no evaluations have been recorded.
func (s *Stored) Effectiveness() float64 {
	total := s.TotalEvaluations()
	if total == 0 {
		return 0.0
	}

	return float64(s.FollowedCount) / float64(total)
}

// Quadrant classifies a memory into one of five categories based on
// effectiveness and surfacing frequency relative to the median.
//
// The five quadrants are:
//   - insufficient-data: fewer than minEvaluationsForQuadrant total evaluations
//   - working: effective (>= 50%) and surfaced at or above median
//   - hidden-gem: effective (>= 50%) but surfaced below median
//   - leech: ineffective (< 50%) but surfaced at or above median
//   - noise: ineffective (< 50%) and surfaced below median
func (s *Stored) Quadrant(medianSurfacedCount int) string {
	if s.TotalEvaluations() < minEvaluationsForQuadrant {
		return QuadrantInsufficientData
	}

	effective := s.Effectiveness() >= effectivenessThreshold
	aboveMedian := s.SurfacedCount >= medianSurfacedCount

	switch {
	case effective && aboveMedian:
		return QuadrantWorking
	case effective && !aboveMedian:
		return QuadrantHiddenGem
	case !effective && aboveMedian:
		return QuadrantLeech
	default:
		return QuadrantNoise
	}
}

// unexported constants.
const (
	effectivenessThreshold    = 0.5
	minEvaluationsForQuadrant = 5
)
