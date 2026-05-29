//go:build targ

package eval

import (
	"errors"
	"fmt"
)

// Exported variables.
var (
	// ErrCalibrationFlat means the floor arm was not measurably worse than the
	// baseline on the calibration scenario — the harness can't detect a known
	// win, so subtle deltas are untrustworthy.
	ErrCalibrationFlat = errors.New("calibration scenario shows no floor-vs-baseline delta")
)

// Cell returns the stats for one (arm × scenario) cell (zero value if absent).
func (s Summary) Cell(arm, scenario string) CellStats {
	return s.cells[cellKey(arm, scenario)]
}

// Aggregate folds run results into per-cell stats.
func Aggregate(results []RunResult) Summary {
	type acc struct {
		trials     int
		turns      float64
		cost       float64
		violations map[string]int
	}
	accs := map[string]*acc{}
	meta := map[string][2]string{}

	for _, r := range results {
		k := cellKey(r.Arm, r.Scenario)
		a := accs[k]
		if a == nil {
			a = &acc{violations: map[string]int{}}
			accs[k] = a
			meta[k] = [2]string{r.Arm, r.Scenario}
		}
		a.trials++
		a.turns += float64(r.Layer1.Turns)
		a.cost += r.Layer1.CostUSD
		for _, b := range r.Behaviors {
			if b.Occurred {
				a.violations[b.Name]++
			}
		}
	}

	cells := map[string]CellStats{}
	for k, a := range accs {
		cells[k] = CellStats{
			Arm:        meta[k][0],
			Scenario:   meta[k][1],
			Trials:     a.trials,
			MeanTurns:  a.turns / float64(a.trials),
			MeanCost:   a.cost / float64(a.trials),
			violations: a.violations,
		}
	}
	return Summary{cells: cells}
}

// CalibrationGate passes only if the `nothing` arm is measurably worse than
// `current-state` on the calibration scenario — by convention violation rate
// or by mean turns. Otherwise the harness can't detect a known win.
func CalibrationGate(s Summary) error {
	const (
		armFloor    = "nothing"
		armBaseline = "current-state"
		scenCalib   = "calibration"
	)
	floor := s.Cell(armFloor, scenCalib)
	base := s.Cell(armBaseline, scenCalib)
	if floor.Trials == 0 || base.Trials == 0 {
		return fmt.Errorf("%w: missing calibration cells", ErrCalibrationFlat)
	}
	worseOnViolations := floor.ViolationRate(checkGoTest) > base.ViolationRate(checkGoTest)
	worseOnTurns := floor.MeanTurns > base.MeanTurns
	if !worseOnViolations && !worseOnTurns {
		return ErrCalibrationFlat
	}
	return nil
}

func cellKey(arm, scenario string) string { return arm + "\x00" + scenario }
