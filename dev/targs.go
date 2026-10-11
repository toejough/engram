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
	"strings"
	"sync"

	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"

	"github.com/toejough/engram/dev/eval"
)

func init() {
	err := registerTargets(targ.DeregisterFrom, targ.Register)
	if err != nil {
		panic(err.Error())
	}

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
	checkForFailTarget = targ.Targ(targdev.CheckForFail.Fn()).Name("check-for-fail").
				Description(targdev.CheckForFail.GetDescription()).
				Deps(
			targdev.CheckCoverageForFail, targdev.CheckUncommitted, targdev.ReorderDeclsCheck, targdev.LintFast,
			targdev.LintForFail, targdev.Deadcode, targdev.CheckThinAPI, checkNilsForFailTarget,
			targdev.TestIntegration, testDevTarget, testDevIntegrationTarget, lintDevTarget, targ.DepModeParallel,
		)
	checkFullTarget = targ.Targ(targdev.CheckFull.Fn()).Name("check-full").
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
	checkNilsTarget = targ.Targ(targdev.CheckNils.Fn()).Name("check-nils").
			Description(targdev.CheckNils.GetDescription()).
			Deps(checkNilsFixTarget, checkNilsForFailTarget)
	checkTarget = targ.Targ(targdev.Check.Fn()).Name("check").
			Description(targdev.Check.GetDescription()).
			Deps(
			targdev.DeleteDeadcode, targdev.Fmt, targdev.Tidy, targdev.Modernize, checkNilsTarget,
			targdev.CheckCoverage, targdev.ReorderDecls, targdev.Lint, targdev.CheckThinAPI, lintDevTarget,
		)
	errEvalUsage  = errors.New("usage: targ eval <arm> [trials]  (nothing|skills-only|current-state)")
	lintDevTarget = targ.Targ(lintDev).Name("lint-dev").
			Description("Lint the dev package's build-tagged files (targ, integration)")
	// mutateTarget reuses Targ's mutation run behind Engram's check-for-fail, as Targ's runs behind its own (794b4a).
	mutateTarget = targ.Targ(targdev.Mutate.Fn()).Name("mutate").
			Description(targdev.Mutate.GetDescription()).
			Deps(checkForFailTarget)
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
	}
	testDevIntegrationTarget = targ.Targ(testDevIntegration).
					Name("test-dev-integration").
					Description("Run local integration tests for dev tooling")
	testDevTarget = targ.Targ(testDev).
			Name("test-dev").
			Description("Run unit tests for the dev package (targ build tag)")
	// watchCheck is the check watch runs on a change: Engram's, never Targ's (794b4a1e).
	watchCheck  = checkTarget
	watchTarget = targ.Targ(watch).Name("watch").Description(targdev.Watch.GetDescription())
)

// evalArgs holds the positional arm name and optional trials count for targ eval.
type evalArgs struct {
	Arm    string `targ:"positional,placeholder=ARM,desc=arm name: nothing|skills-only|current-state"`
	Trials int    `targ:"positional,default=1,placeholder=TRIALS,desc=number of trials per scenario (default 1)"`
}

// hasRelevantChanges mirrors Targ's filter (dev/targets.go hasRelevantChanges): generated files and coverage output
// are ignored so a check's own output cannot retrigger it.
func hasRelevantChanges(changes targ.ChangeSet) bool {
	allFiles := slices.Concat(changes.Added, changes.Removed, changes.Modified)

	return slices.ContainsFunc(allFiles, func(file string) bool {
		return !strings.Contains(file, "generated_") && !strings.HasSuffix(file, "coverage.out")
	})
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

// registerTargets customizes Targ's shared development targets (794b4a): it deregisters them, then registers Engram's
// nil targets, aggregates, watch and lint-dev, and every other shared target unchanged. Targ's aggregates hold direct
// references to its own nil targets, so replacing the nil targets alone would leave them running Targ's versions.
func registerTargets(deregister func(string) error, register func(...any)) error {
	err := deregister(targSourcePackage)
	if err != nil {
		return fmt.Errorf("deregistering Targ's shared development targets: %w", err)
	}

	register(checkNilsFixTarget, checkNilsForFailTarget, checkNilsTarget,
		checkTarget, checkForFailTarget, checkFullTarget, mutateTarget, watchTarget, lintDevTarget)
	register(sharedTargetList()...)

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

// watch is Targ's watch (dev/targets.go watch) running Engram's check instead of Targ's (794b4a). Targ's cannot be
// parameterised: it calls its own Check from inside the function, so this mirrors it. Drift risk: a change to Targ's
// watch patterns, filter or cancellation is not picked up here; the watch handler below is the part that is tested.
func watch(ctx context.Context) error {
	targ.Print(ctx, "Watching...\n")

	err := targ.Watch(ctx, []string{"**/*.go", "**/*.fish", "**/*.toml"}, targ.WatchOptions{},
		watchOnChange(ctx, func(checkCtx context.Context) error { return watchCheck.Run(checkCtx) }))
	if err != nil {
		return fmt.Errorf("watching: %w", err)
	}

	return nil
}

// watchOnChange returns the change handler: on a relevant change it cancels any running check and starts a new one
// through check. It never stops the watch on a failed check.
func watchOnChange(ctx context.Context, check func(context.Context) error) func(targ.ChangeSet) error {
	var (
		cancelCheck context.CancelFunc
		checkMu     sync.Mutex
	)

	return func(changes targ.ChangeSet) error {
		if !hasRelevantChanges(changes) {
			return nil
		}

		targ.Print(ctx, "Change detected in:\n")

		for _, file := range changes.Added {
			targ.Printf(ctx, "  + %s\n", file)
		}

		for _, file := range changes.Modified {
			targ.Printf(ctx, "  ~ %s\n", file)
		}

		for _, file := range changes.Removed {
			targ.Printf(ctx, "  - %s\n", file)
		}

		checkMu.Lock()
		defer checkMu.Unlock()

		if cancelCheck != nil {
			targ.Print(ctx, "Cancelling previous check...\n")
			cancelCheck()
		}

		checkCtx, cancel := context.WithCancel(ctx)
		cancelCheck = cancel

		err := check(checkCtx)

		switch {
		case errors.Is(err, context.Canceled):
			targ.Print(ctx, "Check cancelled, restarting...\n")
		case err != nil:
			targ.Print(ctx, "continuing to watch after check failure (see errors above)\n")
		default:
			targ.Print(ctx, "continuing to watch after all checks passed!\n")
		}

		return nil
	}
}
