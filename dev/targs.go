//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
