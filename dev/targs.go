//go:build targ

package dev

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"

	"github.com/toejough/engram/dev/eval"
)

func init() {
	// Engram customizes Targ's shared development targets by deregistering them and registering its own nil targets
	// and aggregates, plus every other shared target unchanged (794b4a). Targ's aggregates hold direct references to
	// its own nil targets, so replacing the nil targets alone would leave the aggregates running Targ's versions.
	err := targ.DeregisterFrom(targSourcePackage)
	if err != nil {
		panic(fmt.Sprintf("deregistering Targ's shared development targets: %v", err))
	}

	targ.Register(checkNilsFixTarget, checkNilsForFailTarget, checkNilsTarget,
		checkTarget, checkForFailTarget, checkFullTarget, mutateTarget)
	targ.Register(sharedTargetList()...)

	// Engram's spec-traced tests are named Test<scenario ID>_ (#792, e.g.
	// Test792b9c5a6_) or, until renamed, the older TestT<N>_ (not TestProperty_).
	if os.Getenv("TARG_BASELINE_PATTERN") == "" {
		err = os.Setenv("TARG_BASELINE_PATTERN", `Test(T[0-9]+|[0-9]+[a-z][a-z0-9]*)_`)
		if err != nil {
			panic(fmt.Sprintf("setting TARG_BASELINE_PATTERN: %v", err))
		}
	}

	// Sentinel target so targ discovers this package and loads targdev.
	targ.Register(targ.Targ(func(_ context.Context) error { return nil }).
		Name("engram-dev").
		Description("Engram dev target registration"))

	targ.Register(testDevTarget, testDevIntegrationTarget)

	targ.Register(targ.Targ(runEval).
		Name("eval").
		Description("Run the memory eval harness for one arm (nothing|skills-only|current-state)"))
}

// unexported constants.
const (
	evalOutDirPerm    = 0o750
	targSourcePackage = "github.com/toejough/targ/dev"
)

// unexported variables.
var (
	// The aggregates below mirror Targ's dependencies, order and mode with Engram's nil targets in place of Targ's
	// (794b4a); Test794b4a1c fails when Targ's aggregates change. The dev package's tests carry build tags the
	// shared test targets do not pass, so check-for-fail and check-full also run them here.
	checkForFailTarget = targ.Targ(checkForFailAggregate).Name("check-for-fail").
				Description(targdev.CheckForFail.GetDescription()).
				Deps(
			targdev.CheckCoverageForFail, targdev.CheckUncommitted, targdev.ReorderDeclsCheck, targdev.LintFast,
			targdev.LintForFail, targdev.Deadcode, targdev.CheckThinAPI, checkNilsForFailTarget,
			targdev.TestIntegration, testDevTarget, testDevIntegrationTarget, lintDevTarget, targ.DepModeParallel,
		)
	checkFullTarget = targ.Targ(checkFullAggregate).Name("check-full").
			Description(targdev.CheckFull.GetDescription()).
			Deps(
			targdev.CheckCoverageForFail, targdev.CheckUncommitted, targdev.ReorderDeclsCheck, targdev.LintFast,
			targdev.LintFull, targdev.Deadcode, targdev.CheckThinAPI, checkNilsForFailTarget,
			targdev.TestIntegration, testDevTarget, testDevIntegrationTarget, lintDevTarget,
			targ.DepModeParallel, targ.CollectAllErrors,
		)
	checkNilsFixTarget = targ.Targ(checkNilsFix).Name("check-nils-fix").
				Description("Fix nil issues in Engram packages")
	checkNilsForFailTarget = targ.Targ(checkNilsForFail).Name("check-nils-for-fail").
				Description("Check nil issues in Engram packages")
	checkNilsTarget = targ.Targ(checkNilsAggregate).Name("check-nils").
			Description(targdev.CheckNils.GetDescription()).
			Deps(checkNilsFixTarget, checkNilsForFailTarget)
	checkTarget = targ.Targ(checkAggregate).Name("check").
			Description(targdev.Check.GetDescription()).
			Deps(
			targdev.DeleteDeadcode, targdev.Fmt, targdev.Tidy, targdev.Modernize, checkNilsTarget,
			targdev.CheckCoverage, targdev.ReorderDecls, targdev.Lint, targdev.CheckThinAPI, lintDevTarget,
		)
	errEvalUsage  = errors.New("usage: targ eval <arm> [trials]  (nothing|skills-only|current-state)")
	lintDevTarget = targ.Targ(lintDev).Name("lint-dev").
			Description("Lint the dev package's build-tagged files (targ, integration)")
	// sharedTargets are Targ's shared targets other than the nil targets and aggregates Engram replaces, keyed by
	// Targ's variable name. Test794b4a1d fails when Targ adds or removes one.
	sharedTargets = map[string]*targ.Target{
		"CheckCoverage":        targdev.CheckCoverage,
		"CheckCoverageForFail": targdev.CheckCoverageForFail,
		"CheckThinAPI":         targdev.CheckThinAPI,
		"CheckUncommitted":     targdev.CheckUncommitted,
		"Clean":                targdev.Clean,
		"Coverage":             targdev.Coverage,
		"Deadcode":             targdev.Deadcode,
		"DeleteDeadcode":       targdev.DeleteDeadcode,
		"FindRedundantTests":   targdev.FindRedundantTests,
		"Fmt":                  targdev.Fmt,
		"Fuzz":                 targdev.Fuzz,
		"Generate":             targdev.Generate,
		"InstallTools":         targdev.InstallTools,
		"Lint":                 targdev.Lint,
		"LintFast":             targdev.LintFast,
		"LintForFail":          targdev.LintForFail,
		"LintFull":             targdev.LintFull,
		"Modernize":            targdev.Modernize,
		"ReorderDecls":         targdev.ReorderDecls,
		"ReorderDeclsCheck":    targdev.ReorderDeclsCheck,
		"Status":               targdev.Status,
		"Test":                 targdev.Test,
		"TestForFail":          targdev.TestForFail,
		"TestIntegration":      targdev.TestIntegration,
		"Tidy":                 targdev.Tidy,
		"TodoCheck":            targdev.TodoCheck,
		"Watch":                targdev.Watch,
	}
	// mutateTarget reuses Targ's mutation run behind Engram's check-for-fail, as Targ's runs behind its own (794b4a).
	mutateTarget = targ.Targ(targdev.Mutate.Fn()).Name("mutate").
			Description(targdev.Mutate.GetDescription()).
			Deps(checkForFailTarget)
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

func checkAggregate(ctx context.Context) error {
	targ.Print(ctx, "Checking...\n")
	return nil // dependencies run before this function
}

func checkForFailAggregate(ctx context.Context) {
	targ.Print(ctx, "All checks passed!\n")
}

func checkFullAggregate(ctx context.Context) {
	targ.Print(ctx, "All checks passed!\n")
}

func checkNilsAggregate(_ context.Context) error {
	return nil // dependencies run before this function
}

// lintDev lints the dev package with the build tags its files need. Targ's lint targets use no tags, so they skip
// every file in this package; this is the same lint configuration with the tags added.
func lintDev(ctx context.Context) error {
	targ.Print(ctx, "Linting the dev package (tags=targ,integration)...\n")

	err := targ.RunContext(ctx,
		"golangci-lint", "run",
		"-c", "dev/golangci-lint.toml",
		"--build-tags", "targ,integration",
		"--fix=false",
		"--max-issues-per-linter=0",
		"--max-same-issues=0",
		"--allow-parallel-runners",
		"./dev",
	)
	if err != nil {
		return fmt.Errorf("linting the dev package: %w", err)
	}

	return nil
}

func runEval(ctx context.Context, args evalArgs) error {
	if args.Arm == "" {
		return errEvalUsage
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

	err = os.MkdirAll(cfg.OutDir, evalOutDirPerm)
	if err != nil {
		return fmt.Errorf("creating out dir: %w", err)
	}

	deps := eval.Deps{
		Cloner:  eval.NewOSVaultCloner(),
		Config:  eval.NewOSConfigBuilder(enginePath),
		Runner:  eval.NewOSAgentRunner(),
		Results: eval.NewJSONLResultsWriter(filepath.Join(cfg.OutDir, args.Arm+".jsonl")),
	}

	err = eval.Run(ctx, args.Arm, cfg, deps)
	if err != nil {
		return fmt.Errorf("running eval arm %q: %w", args.Arm, err)
	}

	return nil
}

// sharedTargetList returns Targ's retained shared targets in name order.
func sharedTargetList() []any {
	names := slices.Sorted(maps.Keys(sharedTargets))
	targets := make([]any, 0, len(names))

	for _, name := range names {
		targets = append(targets, sharedTargets[name])
	}

	return targets
}

func testDev(ctx context.Context) error {
	targ.Print(ctx, "Running dev-package tests (tag=targ)...\n")

	err := targ.RunContext(ctx, "go", "test", "-count=1", "-timeout=2m", "-race", "-tags=targ", "./dev/...")
	if err != nil {
		return fmt.Errorf("running dev-package tests: %w", err)
	}

	return nil
}

// testDevIntegration runs the dev tree with the integration tag added. Integration tests are selected by that build
// tag (files named *_integration_internal_test.go), never by test name, so a new one cannot be skipped by its name.
// They stay in package dev because they drive its unexported entry points and targets. The tag only adds files, so
// the pure tests compile in this run too and run a second time; that duplication is deliberate and cheap.
func testDevIntegration(ctx context.Context) error {
	err := targ.RunContext(ctx,
		"go", "test", "-count=1", "-timeout=2m", "-race", "-tags=targ,integration",
		"./dev/...",
	)
	if err != nil {
		return fmt.Errorf("running dev integration tests: %w", err)
	}

	return nil
}
