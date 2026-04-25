//go:build targ

package dev

import (
	"context"
	"os"

	"github.com/toejough/targ"
	_ "github.com/toejough/targ/dev"
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
}

func testDev(ctx context.Context) error {
	targ.Print(ctx, "Running dev-package tests (tag=targ)...\n")
	if err := targ.RunContext(ctx, "go", "clean", "-testcache"); err != nil {
		return err
	}
	return targ.RunContext(ctx, "go", "test", "-timeout=2m", "-race", "-tags=targ", "./dev/...")
}
