//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"

	"github.com/toejough/targ"
	_ "github.com/toejough/targ/dev"

	"github.com/toejough/engram/dev/eval"
)

func init() {
	// Engram's spec-traced tests use TestT<N>_ naming (not TestProperty_).
	if os.Getenv("TARG_BASELINE_PATTERN") == "" {
		os.Setenv("TARG_BASELINE_PATTERN", `TestT[0-9]+_`)
	}

	// Sentinel target so targ discovers this package and loads targdev.
	targ.Register(targ.Targ(func(_ context.Context) error { return nil }).
		Name("engram-dev").
		Description("Engram dev target registration"))

	targ.Register(targ.Targ(testDev).
		Name("test-dev").
		Description("Run unit tests for the dev package (targ build tag)"))

	targ.Register(targ.Targ(runEval).
		Name("eval").
		Description("Run the memory eval harness for one arm (nothing|skills-only|current-state)"))
}

// evalArgs holds the positional arm name for targ eval.
type evalArgs struct {
	Arm string `targ:"positional,placeholder=ARM,desc=arm name: nothing|skills-only|current-state"`
}

func runEval(ctx context.Context, args evalArgs) error {
	if args.Arm == "" {
		return fmt.Errorf("usage: targ eval <arm>  (nothing|skills-only|current-state)")
	}
	return eval.Run(ctx, args.Arm, eval.RunConfig{}, eval.Deps{})
}

func testDev(ctx context.Context) error {
	targ.Print(ctx, "Running dev-package tests (tag=targ)...\n")
	if err := targ.RunContext(ctx, "go", "clean", "-testcache"); err != nil {
		return err
	}
	return targ.RunContext(ctx, "go", "test", "-timeout=2m", "-race", "-tags=targ", "./dev/...")
}
