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

func executeNilaway(ctx context.Context, args []string, settings map[string]string) error {
	command := targ.Cmd("nilaway", args...)
	for key, value := range settings {
		command.Env(key, value)
	}
	return command.Run(ctx)
}

func nilawayArgs(fix bool) []string {
	args := make([]string, 0, 3)
	args = append(args, "-include-pkgs=github.com/toejough/engram")
	if fix {
		args = append(args, "-fix")
	}
	return append(args, "./...")
}

func nilawayEnvironment(lookup func(string) (string, bool)) map[string]string {
	settings := map[string]string{
		"GOMEMLIMIT": "2GiB",
		"GOGC":       "50",
		"GOMAXPROCS": "2",
	}
	for key := range settings {
		if value, present := lookup(key); present {
			settings[key] = value
		}
	}
	return settings
}

func runConfiguredNilaway(ctx context.Context, fix bool, lookup func(string) (string, bool), execute func(context.Context, []string, map[string]string) error) error {
	return execute(ctx, nilawayArgs(fix), nilawayEnvironment(lookup))
}

func runNilaway(ctx context.Context, fix bool) error {
	return runConfiguredNilaway(ctx, fix, os.LookupEnv, executeNilaway)
}
