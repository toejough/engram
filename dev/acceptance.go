//go:build targ

package dev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/toejough/targ"
)

// unexported constants.
const (
	acceptanceDir      = "dev/acceptance"
	acceptanceDryRun   = "--dry-run"
	acceptanceModules  = "node_modules"
	acceptancePlanSize = 3 // npm ci, self-tests, run
	acceptanceRunEntry = "run.js"
	nodeProgram        = "node"
	npmProgram         = "npm"
)

// unexported variables.
var (
	errAcceptanceProgram = errors.New("acceptance needs a program that is not on PATH")
)

// acceptanceArgs holds the arguments of targ acceptance.
type acceptanceArgs struct {
	Filter string `targ:"positional,placeholder=ID-PREFIX,desc=scenario ID prefix (empty: product scenarios; 793d: H1)"`
	DryRun bool   `targ:"flag,name=dry-run,desc=vocabulary check only; no session and no credentials"`
	Check  bool   `targ:"flag,name=check,desc=self-tests and vocabulary check; no session and no credentials"`
}

// acceptanceCommand is one program run of an acceptance plan; paths in Args are absolute.
type acceptanceCommand struct {
	Name string
	Args []string
}

// acceptanceEnv is what a plan reads from the machine: the repository root, the program lookup and the file check.
type acceptanceEnv struct {
	Root     string
	LookPath func(string) (string, error)
	Exists   func(string) bool
}

// acceptancePlan returns the commands of one targ acceptance run, or the missing program that stops it (793e2a).
func acceptancePlan(args acceptanceArgs, env acceptanceEnv) ([]acceptanceCommand, error) {
	live := !args.DryRun && !args.Check

	programs := []string{nodeProgram, npmProgram}
	if live {
		programs = append(programs, "claude")
	}

	for _, program := range programs {
		_, err := env.LookPath(program)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: `%s` (a dev dependency of %s: install it, then run targ acceptance again)",
				errAcceptanceProgram, program, acceptanceDir,
			)
		}
	}

	dir := filepath.Join(env.Root, acceptanceDir)
	plan := make([]acceptanceCommand, 0, acceptancePlanSize)

	if !env.Exists(filepath.Join(dir, acceptanceModules)) {
		plan = append(plan, acceptanceCommand{Name: npmProgram, Args: []string{"ci", "--prefix", dir}})
	}

	entry := filepath.Join(dir, acceptanceRunEntry)

	if args.Check {
		plan = append(plan,
			acceptanceCommand{Name: nodeProgram, Args: []string{"--test", filepath.Join(dir, "test", "self") + "/"}},
			acceptanceCommand{Name: nodeProgram, Args: []string{entry, acceptanceDryRun}},
		)

		return plan, nil
	}

	runArgs := []string{entry}
	if args.DryRun {
		runArgs = append(runArgs, acceptanceDryRun)
	}

	if args.Filter != "" {
		runArgs = append(runArgs, args.Filter)
	}

	return append(plan, acceptanceCommand{Name: nodeProgram, Args: runArgs}), nil
}

// acceptanceRoot finds the repository root from the working directory: the nearest ancestor that holds the package.
func acceptanceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("finding the repository root: %w", err)
	}

	for {
		_, statErr := os.Stat(filepath.Join(dir, acceptanceDir, "package.json"))
		if statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w: no %s above the working directory", errAcceptanceProgram, acceptanceDir)
		}

		dir = parent
	}
}

// runAcceptance runs the plan of acceptancePlan from the repository root, streaming each command's output.
func runAcceptance(ctx context.Context, args acceptanceArgs) error {
	root, err := acceptanceRoot()
	if err != nil {
		return err
	}

	plan, err := acceptancePlan(args, acceptanceEnv{
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
