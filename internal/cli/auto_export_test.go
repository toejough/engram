package cli

import (
	"context"
	"time"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/update"
)

// The adapters of `engram auto run`, for their tests.

func AutoCommandForTest(
	ctx context.Context, cmd update.Commander, argv []string, deadline time.Duration,
) (auto.Output, error) {
	return runAutoCommand(ctx, cmd, argv, deadline)
}

func NewAutoClockForTest(now func() time.Time) auto.Clock { return autoClock{now: now} }

func NewAutoEnvForTest(getenv func(string) string) auto.Env { return autoEnv{getenv: getenv} }

func NewAutoFilesForTest(edge EdgeFS) auto.Files { return autoFiles{edge: edge} }

func RunAutoForTest(ctx context.Context, deps Deps) error { return runAuto(ctx, deps) }
