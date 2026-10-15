package cli

import (
	"context"
	"time"

	"github.com/toejough/engram/internal/auto"
)

// The adapters of `engram auto run`, for their tests.

func AutoDirectoriesForTest(home string, env map[string]string) (vault, chunks string) {
	return autoDirectories(Deps{
		UserHomeDir: func() (string, error) { return home, nil },
		Getenv:      func(key string) string { return env[key] },
	})
}

func NewAutoClockForTest(now func() time.Time) auto.Clock { return autoClock{now: now} }

func NewAutoFilesForTest(edge EdgeFS) auto.Files { return autoFiles{edge: edge} }

func NewAutoOffersForTest(edge EdgeFS) auto.Offers { return autoOffers{edge: edge} }

// NewAutoVaultForTest is the vault of a job, with the harness fault the dependencies carry.
func NewAutoVaultForTest(ctx context.Context, deps Deps) auto.Vault { return newAutoVault(ctx, deps) }

func RunAutoForTest(ctx context.Context, deps Deps) error { return runAuto(ctx, deps) }
