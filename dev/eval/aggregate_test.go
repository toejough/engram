//go:build targ

package eval_test

import (
	"errors"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestAggregate_ComputesPerCellRates(t *testing.T) {
	t.Parallel()

	sum := eval.Aggregate(results())
	cell := sum.Cell("nothing", "calibration")
	if cell.Trials != 2 || cell.ViolationRate("used-go-test-not-targ") != 1.0 {
		t.Fatalf("nothing cell wrong: %+v", cell)
	}
	clean := sum.Cell("current-state", "calibration")
	if clean.ViolationRate("used-go-test-not-targ") != 0.0 {
		t.Fatalf("current-state cell wrong: %+v", clean)
	}
}

func TestCalibrationGate_FailsWhenNoDelta(t *testing.T) {
	t.Parallel()

	flat := []eval.RunResult{
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
	}
	if err := eval.CalibrationGate(eval.Aggregate(flat)); !errors.Is(err, eval.ErrCalibrationFlat) {
		t.Fatalf("expected ErrCalibrationFlat, got %v", err)
	}
}

func TestCalibrationGate_PassesWhenNothingWorse(t *testing.T) {
	t.Parallel()

	if err := eval.CalibrationGate(eval.Aggregate(results())); err != nil {
		t.Fatalf("gate should pass: %v", err)
	}
}

func results() []eval.RunResult {
	return []eval.RunResult{
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 30, CostUSD: 0.5}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: true}}},
		{Arm: "nothing", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 28, CostUSD: 0.4}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: true}}},
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 12, CostUSD: 0.2}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
		{Arm: "current-state", Scenario: "calibration", Layer1: eval.Layer1Metrics{Turns: 10, CostUSD: 0.2}, Behaviors: []eval.BehaviorOutcome{{Name: "used-go-test-not-targ", Occurred: false}}},
	}
}
