//go:build targ

package eval_test

import (
	"context"
	"errors"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestRun_UnknownArm_ReturnsErrUnknownArm(t *testing.T) {
	t.Parallel()

	err := eval.Run(context.Background(), "bogus", eval.RunConfig{}, eval.Deps{})
	if !errors.Is(err, eval.ErrUnknownArm) {
		t.Fatalf("got %v, want ErrUnknownArm", err)
	}
}
