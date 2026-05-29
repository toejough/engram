//go:build targ

package eval_test

import (
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestLookupArm_CurrentState_SkillsAndBinary(t *testing.T) {
	t.Parallel()

	arm, ok := eval.LookupArm("current-state")
	if !ok {
		t.Fatal("current-state arm not found")
	}
	if !arm.BinaryOnPATH {
		t.Fatal("current-state must have binary on PATH")
	}
	if len(arm.Skills) == 0 {
		t.Fatal("current-state must install skills")
	}
}

func TestLookupArm_Nothing_NoSkillsNoBinary(t *testing.T) {
	t.Parallel()

	arm, ok := eval.LookupArm("nothing")
	if !ok {
		t.Fatal("nothing arm not found")
	}
	if len(arm.Skills) != 0 || arm.BinaryOnPATH {
		t.Fatalf("nothing arm should have no skills and no binary, got %+v", arm)
	}
}

func TestLookupArm_Unknown_False(t *testing.T) {
	t.Parallel()

	if _, ok := eval.LookupArm("nope"); ok {
		t.Fatal("unknown arm should return false")
	}
}
