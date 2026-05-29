//go:build targ

package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestDetectBehaviors_CleanRun_NoOccurrence(t *testing.T) {
	t.Parallel()

	var calibration eval.Scenario
	for _, s := range eval.Scenarios() {
		if s.Name == "calibration" {
			calibration = s
		}
	}

	out := eval.DetectBehaviors(calibration, []string{"targ test", "targ build"})
	if len(out) != 1 || out[0].Occurred {
		t.Fatalf("expected single non-occurred outcome, got %+v", out)
	}
}

func TestDetectBehaviors_FlagsGoTest(t *testing.T) {
	t.Parallel()

	var calibration eval.Scenario
	for _, s := range eval.Scenarios() {
		if s.Name == "calibration" {
			calibration = s
		}
	}

	out := eval.DetectBehaviors(calibration, []string{"go test ./...", "ls"})
	if len(out) != 1 {
		t.Fatalf("got %d outcomes, want 1: %+v", len(out), out)
	}
	if out[0].Name != "used-go-test-not-targ" || !out[0].Occurred {
		t.Fatalf("unexpected outcome: %+v", out[0])
	}
}
