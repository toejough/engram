//go:build targ

package dev

import (
	"context"
	"fmt"

	"github.com/toejough/targ"
)

// unexported constants.
const (
	nilawayArgCapacity = 3
)

func checkNilsFix(ctx context.Context) error {
	targ.Print(ctx, "Fixing nil issues in Engram packages...\n")
	return runNilaway(ctx, true)
}

func checkNilsForFail(ctx context.Context) error {
	targ.Print(ctx, "Checking for nil issues in Engram packages...\n")
	return runNilaway(ctx, false)
}

// executeNilaway starts NilAway with the arguments and the parent environment unchanged.
func executeNilaway(ctx context.Context, args []string) error {
	err := targ.Cmd("nilaway", args...).Run(ctx)
	if err != nil {
		return fmt.Errorf("running nilaway: %w", err)
	}

	return nil
}

// nilawayArgs is the invocation policy: Engram's module scope, plus -fix in fix mode.
func nilawayArgs(fix bool) []string {
	args := make([]string, 0, nilawayArgCapacity)

	args = append(args, "-include-pkgs=github.com/toejough/engram")
	if fix {
		args = append(args, "-fix")
	}

	return append(args, "./...")
}

// runConfiguredNilaway sends one request with the context and the policy's arguments, returning the failure as is.
func runConfiguredNilaway(
	ctx context.Context,
	fix bool,
	policy func(bool) []string,
	execute func(context.Context, []string) error,
) error {
	return execute(ctx, policy(fix))
}

func runNilaway(ctx context.Context, fix bool) error {
	return runConfiguredNilaway(ctx, fix, nilawayArgs, executeNilaway)
}
