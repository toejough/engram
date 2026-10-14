//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/toejough/targ"
)

// unexported constants.
const (
	gateAdapter     = "adapter"
	gateBudget      = "budget"
	gateComplexity  = "complexity"
	gateEntry       = "gates/run.js"
	gateLint        = "lint"
	gateModProgram  = "claude"
	gateModTest     = "mod-test"
	gateModWhich    = "green"
	gatePlanMaxSize = 2
	gateSpec        = "spec"
	gateStatus      = "status"
)

// jsGateEnv is what a gate plan reads from the machine: the repository root, the program lookup and the file check.
type jsGateEnv = acceptanceEnv

// lintJSArgs holds the arguments of targ lint-js.
type lintJSArgs struct {
	Part string `targ:"positional,placeholder=PART,desc=standards|complexity|types (empty: all three)"`
}

// testModArgs holds the arguments of targ test-mod.
type testModArgs struct {
	Which  string `targ:"positional,placeholder=WHICH,desc=green (test files) or red (refactor gates); default green"`
	Filter string `targ:"positional,placeholder=FILE-PREFIX,desc=only the files whose name starts with this"`
}

// jsGatePlan returns the commands of one refactor-gate run, or the reason it cannot run (793e8l): `npm ci` when the
// package's dependencies are missing, then the node entry of the gates with its arguments. claude is needed only by the
// gates that run the mod's tests.
func jsGatePlan(gate string, gateArgs []string, needClaude bool, env jsGateEnv) ([]acceptanceCommand, error) {
	programs := []string{nodeProgram, npmProgram}
	if needClaude {
		programs = append(programs, gateModProgram)
	}

	for _, program := range programs {
		_, err := env.LookPath(program)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: `%s` (the refactor gates of %s need it: install it, then run again)",
				errAcceptanceProgram, program, acceptanceDir,
			)
		}
	}

	dir := filepath.Join(env.Root, acceptanceDir)
	plan := make([]acceptanceCommand, 0, gatePlanMaxSize)

	if !env.Exists(filepath.Join(dir, acceptanceModules)) {
		plan = append(plan, acceptanceCommand{Name: npmProgram, Args: []string{"ci", "--prefix", dir}})
	}

	runArgs := append([]string{filepath.Join(dir, gateEntry), gate}, gateArgs...)

	return append(plan, acceptanceCommand{Name: nodeProgram, Args: runArgs}), nil
}

func runCheckModBudget(ctx context.Context) error {
	return runJSGate(ctx, gateBudget, nil, false)
}

func runCheckSpecConformance(ctx context.Context) error {
	return runJSGate(ctx, gateSpec, nil, true)
}

// runJSGate runs the plan of jsGatePlan from the repository root, streaming each command's output.
func runJSGate(ctx context.Context, gate string, gateArgs []string, needClaude bool) error {
	root, err := acceptanceRoot()
	if err != nil {
		return err
	}

	plan, err := jsGatePlan(gate, gateArgs, needClaude, acceptanceEnv{
		Root:     root,
		LookPath: exec.LookPath,
		Exists: func(path string) bool {
			_, statErr := os.Stat(path)

			return statErr == nil
		},
	})
	if err != nil {
		return err
	}

	for _, step := range plan {
		runErr := targ.RunContext(ctx, step.Name, step.Args...)
		if runErr != nil {
			return fmt.Errorf("running %s %v: %w", step.Name, step.Args, runErr)
		}
	}

	return nil
}

func runLintJS(ctx context.Context, args lintJSArgs) error {
	gateArgs := []string{}
	if args.Part != "" {
		gateArgs = append(gateArgs, args.Part)
	}

	return runJSGate(ctx, gateLint, gateArgs, false)
}

func runLintJSAdapter(ctx context.Context) error {
	return runJSGate(ctx, gateLint, []string{gateAdapter}, false)
}

func runLintJSComplexity(ctx context.Context) error {
	return runJSGate(ctx, gateLint, []string{gateComplexity}, false)
}

func runRefactorRed(ctx context.Context) error {
	return runJSGate(ctx, gateStatus, nil, true)
}

func runTestMod(ctx context.Context, args testModArgs) error {
	which := args.Which
	if which == "" {
		which = gateModWhich
	}

	return runJSGate(ctx, gateModTest, []string{which, args.Filter}, true)
}
