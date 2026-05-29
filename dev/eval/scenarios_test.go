//go:build targ

package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestScenarios_BehaviorChecksCompile(t *testing.T) {
	t.Parallel()

	for _, s := range eval.Scenarios() {
		for _, c := range s.Checks {
			if c.Pattern == nil {
				t.Fatalf("scenario %q check %q has nil pattern", s.Name, c.Name)
			}
		}
	}
}

func TestScenarios_IncludesCalibration(t *testing.T) {
	t.Parallel()

	got := map[string]bool{}
	for _, s := range eval.Scenarios() {
		got[s.Name] = true
		if s.Prompt == "" {
			t.Fatalf("scenario %q has empty prompt", s.Name)
		}
	}
	if !got["calibration"] {
		t.Fatal("missing calibration scenario")
	}
}
