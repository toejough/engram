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
	"strconv"
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

	targ.Register(testDevTarget, testDevIntegrationTarget, testIntegrationAllTarget)

	targ.Register(targ.Targ(runEval).
		Name("eval").
		Description("Run the memory eval harness for one arm (nothing|skills-only|current-state)"))

	targ.Register(acceptanceTarget, testPkgTarget)
}

// unexported constants.
const (
	// acceptanceMinutes is the time limit of one repetition of the whole set of those tests.
	acceptanceMinutes = 30
	// acceptancePackage holds the tests that drive real Claude Code sessions in isolated homes; they carry
	// the network build tag, so no default target runs them.
	acceptancePackage = "./dev/acceptance"
	// acceptanceParallel is how many of those tests run at once: more than 6 make the timed ones fail.
	acceptanceParallel = "4"
	acceptanceTags     = "network"
	evalOutDirPerm     = 0o750
	// goTestCommand is the go command that runs tests.
	goTestCommand = "test"
	// integrationAllTimeout is the time limit of the whole module's tagged tests: the one of Targ's whole-module legs.
	integrationAllTimeout = "-timeout=10m"
	targSourcePackage     = "github.com/toejough/targ/dev"
	// testPkgTimeout bounds the tests of one package.
	testPkgTimeout = "-timeout=2m"
)

// unexported variables.
var (
	acceptanceTarget = targ.Targ(runAcceptance).Name("acceptance").
				Description("Run the tests that drive real Claude Code sessions in isolated homes (they cost money)")
	// The aggregates below mirror Targ's dependencies, order and mode with Engram's nil targets in place of Targ's
	// (794b4a); Test794b4a1c fails when Targ's aggregates change. The dev package's tests carry build tags the
	// shared test targets do not pass, so check-for-fail and check-full also run them here. Targ's integration leg
	// selects tests by name and runs none, so they also run test-integration-all, which selects them by tag (862b3a).
	checkForFailTarget = targ.Targ(targdev.CheckForFail.Fn()).Name("check-for-fail").
				Description(targdev.CheckForFail.GetDescription()).
				Deps(
			targdev.CheckCoverageForFail, targdev.CheckUncommitted, targdev.ReorderDeclsCheck, targdev.LintFast,
			targdev.LintForFail, targdev.Deadcode, targdev.CheckThinAPI, checkNilsForFailTarget,
			targdev.TestIntegration, testDevTarget, testDevIntegrationTarget, lintDevTarget, testIntegrationAllTarget,
			targ.DepModeParallel,
		)
	checkFullTarget = targ.Targ(targdev.CheckFull.Fn()).Name("check-full").
			Description(targdev.CheckFull.GetDescription()).
			Deps(
			targdev.CheckCoverageForFail, targdev.CheckUncommitted, targdev.ReorderDeclsCheck, targdev.LintFast,
			targdev.LintFull, targdev.Deadcode, targdev.CheckThinAPI, checkNilsForFailTarget,
			targdev.TestIntegration, testDevTarget, testDevIntegrationTarget, lintDevTarget, testIntegrationAllTarget,
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
	errEvalUsage = errors.New("usage: targ eval <arm> [trials]  (nothing|skills-only|current-state)")
	// errTestPkgNoPackage is what test-pkg reports when it is given no package path (861b2a1a).
	errTestPkgNoPackage = errors.New("usage: targ test-pkg <package> [--run REGEXP]  (for example ./internal/cli)")
	lintDevTarget       = targ.Targ(lintDev).Name("lint-dev").
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
	testIntegrationAllTarget = targ.Targ(testIntegrationAll).
					Name("test-integration-all").
					Description("Run every integration-tagged test of the module, selected by build tag")
	testPkgTarget = targ.Targ(testPkg).
			Name("test-pkg").
			Description("Run the tests of one package with -race, only those matching --run when it is given")
	// watchCheck is the check watch runs on a change: Engram's, never Targ's (794b4a1e).
	watchCheck  = checkTarget
	watchTarget = targ.Targ(watch).Name("watch").Description(targdev.Watch.GetDescription())
)

// acceptanceArgs holds the optional test filter and the count of targ acceptance.
type acceptanceArgs struct {
	Count int    `targ:"flag,name=count,default=1,desc=run each test this many times (default 1)"`
	Run   string `targ:"flag,name=run,desc=run only the tests whose names match this regular expression"`
}

// commandRunner runs a command to completion and returns its failure; targ.RunContext is the production one.
type commandRunner func(ctx context.Context, name string, args ...string) error

// evalArgs holds the positional arm name and optional trials count for targ eval.
type evalArgs struct {
	Arm    string `targ:"positional,placeholder=ARM,desc=arm name: nothing|skills-only|current-state"`
	Trials int    `targ:"positional,default=1,placeholder=TRIALS,desc=number of trials per scenario (default 1)"`
}

// testPkgArgs holds the package path and the optional test filter of targ test-pkg.
type testPkgArgs struct {
	Package string `targ:"positional,placeholder=PACKAGE,desc=path of the package to test (for example ./internal/cli)"`
	Run     string `targ:"flag,name=run,desc=run only the tests whose names match this regular expression"`
}

// acceptanceTestArgs is the go test command line of targ acceptance.
func acceptanceTestArgs(args acceptanceArgs) []string {
	count := max(args.Count, 1)
	argv := []string{
		goTestCommand, "-count=" + strconv.Itoa(count), "-timeout=" + strconv.Itoa(acceptanceMinutes*count) + "m", "-v",
		"-parallel=" + acceptanceParallel, "-tags=" + acceptanceTags,
	}
	if args.Run != "" {
		argv = append(argv, "-run", args.Run)
	}

	return append(argv, acceptancePackage+"/...")
}

// hasRelevantChanges mirrors Targ's filter (dev/targets.go hasRelevantChanges): generated files and coverage output
// are ignored so a check's own output cannot retrigger it.
func hasRelevantChanges(changes targ.ChangeSet) bool {
	allFiles := slices.Concat(changes.Added, changes.Removed, changes.Modified)

	return slices.ContainsFunc(allFiles, func(file string) bool {
		return !strings.Contains(file, "generated_") && !strings.HasSuffix(file, "coverage.out")
	})
}

// integrationAllCommandLine is the go test command line of targ test-integration-all: every package of the module,
// with the tags of the dev tree and the integration tag, and no test filter, so that a test is selected by its build
// tag and never by its name (862b2a).
func integrationAllCommandLine() []string {
	return []string{goTestCommand, "-count=1", integrationAllTimeout, "-race", "-tags=targ,integration", "./..."}
}

// lintDev lints the dev package with the build tags its files need. Targ's lint targets use no tags, so they skip
// every file in this package; this is the same lint configuration with the tags added.
func lintDev(ctx context.Context) error {
	targ.Print(ctx, "Linting the dev package and the acceptance tests (tags=targ,integration,network)...\n")

	err := targ.RunContext(ctx,
		"golangci-lint", "run",
		"-c", "dev/golangci-lint.toml",
		"--build-tags", "targ,integration,"+acceptanceTags,
		"--fix=false",
		"--max-issues-per-linter=0",
		"--max-same-issues=0",
		"--allow-parallel-runners",
		"./dev", acceptancePackage,
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

// runAcceptance runs the product-scenario tests and the technical tests that start a real Claude Code process. They
// are not part of any check: they cost money, need the network and a login, and start many processes.
func runAcceptance(ctx context.Context, args acceptanceArgs) error {
	err := targ.RunContext(ctx, "go", acceptanceTestArgs(args)...)
	if err != nil {
		return fmt.Errorf("running the acceptance tests: %w", err)
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

// runIntegrationAll runs go test on the whole module with the integration tag through run, and fails with the failure
// of go test.
func runIntegrationAll(ctx context.Context, run commandRunner) error {
	err := run(ctx, "go", integrationAllCommandLine()...)
	if err != nil {
		return fmt.Errorf("running the integration-tagged tests of the module: %w", err)
	}

	return nil
}

// runTestPkg runs go test on the package of args through run, and fails with the failure of go test. It starts no
// command when args hold no package path.
func runTestPkg(ctx context.Context, args testPkgArgs, run commandRunner) error {
	argv, err := testPkgCommandLine(args)
	if err != nil {
		return err
	}

	err = run(ctx, "go", argv...)
	if err != nil {
		return fmt.Errorf("running the tests of %s: %w", args.Package, err)
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

	err := targ.RunContext(ctx, "go", goTestCommand, "-count=1", "-timeout=2m", "-race", "-tags=targ", "./dev/...")
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
		"go", goTestCommand, "-count=1", "-timeout=2m", "-race", "-tags=targ,integration",
		"./dev/...",
	)
	if err != nil {
		return fmt.Errorf("running dev integration tests: %w", err)
	}

	return nil
}

// testIntegrationAll runs every test the integration tag selects, in every package of the module. Targ's shared
// integration leg selects by test name and runs none today, so this is the command that runs them (862b2a). The tag
// only adds files, so the untagged tests run a second time, as in testDevIntegration; the cost is accepted (862d3).
func testIntegrationAll(ctx context.Context) error {
	return runIntegrationAll(ctx, targ.RunContext)
}

// testPkg runs the tests of one package with the race detector and no build tag (861b2a), so that a builder runs the
// package it touched and not the whole module.
func testPkg(ctx context.Context, args testPkgArgs) error {
	return runTestPkg(ctx, args, targ.RunContext)
}

// testPkgCommandLine is the go test command line of targ test-pkg: the count, the timeout and the race detector, the
// filter when one is given, and the package path last.
func testPkgCommandLine(args testPkgArgs) ([]string, error) {
	if args.Package == "" {
		return nil, errTestPkgNoPackage
	}

	argv := []string{goTestCommand, "-count=1", testPkgTimeout, "-race"}
	if args.Run != "" {
		argv = append(argv, "-run", args.Run)
	}

	return append(argv, args.Package), nil
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
