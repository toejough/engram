//go:build targ

package dev

import (
	"context"
	"os"

	"github.com/toejough/targ"
)

func checkNilsFix(ctx context.Context) error {
	targ.Print(ctx, "Fixing nil issues in Engram packages...\n")
	return runNilaway(ctx, true)
}

func checkNilsForFail(ctx context.Context) error {
	targ.Print(ctx, "Checking for nil issues in Engram packages...\n")
	return runNilaway(ctx, false)
}

func nilawayArgs(fix bool) []string {
	args := make([]string, 0, 3)
	if fix {
		args = append(args, "-fix")
	}
	return append(args, "./...")
}

func nilawayEnvironment(_ func(string) (string, bool)) map[string]string {
	return nil
}

func runNilaway(ctx context.Context, fix bool) error {
	command := targ.Cmd("nilaway", nilawayArgs(fix)...)
	for key, value := range nilawayEnvironment(os.LookupEnv) {
		command.Env(key, value)
	}
	return command.Run(ctx)
}
