//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"

	"github.com/toejough/engram/dev/eval"
)

func init() {
	*targdev.CheckNilsFix = *targ.Targ(checkNilsFix).Name("check-nils-fix").Description("Fix nil issues in Engram packages")
	*targdev.CheckNilsForFail = *targ.Targ(checkNilsForFail).Name("check-nils-for-fail").Description("Check nil issues in Engram packages")

	// Engram's spec-traced tests are named Test<scenario ID>_ (#792, e.g.
	// Test792b9c5a6_) or, until renamed, the older TestT<N>_ (not TestProperty_).
	if os.Getenv("TARG_BASELINE_PATTERN") == "" {
		os.Setenv("TARG_BASELINE_PATTERN", `Test(T[0-9]+|[0-9]+[a-z][a-z0-9]*)_`)
	}

	// Sentinel target so targ discovers this package and loads targdev.
	targ.Register(targ.Targ(func(_ context.Context) error { return nil }).
		Name("engram-dev").
		Description("Engram dev target registration"))

	targ.Register(testDevTarget, testDevIntegrationTarget)

	// The dev package's tests carry build tags the shared test targets do not pass, so the checks run them here.
	targdev.CheckForFail.Deps(testDevTarget, testDevIntegrationTarget, targ.DepModeParallel)
	targdev.CheckFull.Deps(testDevTarget, testDevIntegrationTarget, targ.DepModeParallel, targ.CollectAllErrors)

	targ.Register(targ.Targ(runEval).
		Name("eval").
		Description("Run the memory eval harness for one arm (nothing|skills-only|current-state)"))
}

// unexported variables.
var (
	testDevIntegrationTarget = targ.Targ(testDevIntegration).
					Name("test-dev-integration").
					Description("Run local integration tests for dev tooling")
	testDevTarget = targ.Targ(testDev).
			Name("test-dev").
			Description("Run unit tests for the dev package (targ build tag)")
)

// evalArgs holds the positional arm name and optional trials count for targ eval.
type evalArgs struct {
	Arm    string `targ:"positional,placeholder=ARM,desc=arm name: nothing|skills-only|current-state"`
	Trials int    `targ:"positional,default=1,placeholder=TRIALS,desc=number of trials per scenario (default 1)"`
}

func runEval(ctx context.Context, args evalArgs) error {
	if args.Arm == "" {
		return fmt.Errorf("usage: targ eval <arm> [trials]  (nothing|skills-only|current-state)")
	}

	enginePath, err := exec.LookPath("engram")
	if err != nil {
		return fmt.Errorf("engram binary not found on PATH: %w", err)
	}

	home, _ := os.UserHomeDir()

	cfg := eval.RunConfig{
		Trials:   1,
		Model:    "haiku",
		VaultSrc: filepath.Join(home, ".local", "share", "engram", "vault"),
		OutDir:   "/tmp/engram-eval",
	}

	if args.Trials > 0 {
		cfg.Trials = args.Trials
	}

	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return fmt.Errorf("creating out dir: %w", err)
	}

	deps := eval.Deps{
		Cloner:  eval.NewOSVaultCloner(),
		Config:  eval.NewOSConfigBuilder(enginePath),
		Runner:  eval.NewOSAgentRunner(),
		Results: eval.NewJSONLResultsWriter(filepath.Join(cfg.OutDir, args.Arm+".jsonl")),
	}

	return eval.Run(ctx, args.Arm, cfg, deps)
}

func testDev(ctx context.Context) error {
	targ.Print(ctx, "Running dev-package tests (tag=targ)...\n")
	if err := targ.RunContext(ctx, "go", "clean", "-testcache"); err != nil {
		return err
	}
	return targ.RunContext(ctx, "go", "test", "-timeout=2m", "-race", "-tags=targ", "./dev/...")
}

func testDevIntegration(ctx context.Context) error {
	return targ.RunContext(ctx, "go", "test", "-timeout=2m", "-race", "-tags=targ,integration", "./dev")
}
