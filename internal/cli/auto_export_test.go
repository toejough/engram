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

func AutoDirectoriesForTest(home string, env map[string]string) (vault, chunks string) {
	return autoDirectories(Deps{
		UserHomeDir: func() (string, error) { return home, nil },
		Getenv:      func(key string) string { return env[key] },
	})
}

func NewAutoClockForTest(now func() time.Time) auto.Clock { return autoClock{now: now} }

func NewAutoFilesForTest(edge EdgeFS) auto.Files { return autoFiles{edge: edge} }

func NewAutoOffersForTest(edge EdgeFS) auto.Offers { return autoOffers{edge: edge} }

func RunAutoForTest(ctx context.Context, deps Deps) error { return runAuto(ctx, deps) }
