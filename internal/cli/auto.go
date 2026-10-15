package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/auto"
)

// unexported constants.
const (
	autoDirPerm fs.FileMode = 0o755
	// the backlog marks name the user's conversations: the user alone reads them
	autoFilePerm fs.FileMode = 0o600
)

// autoClock is the wall clock of a job.
type autoClock struct{ now func() time.Time }

// Now is the time.
func (c autoClock) Now() time.Time { return c.now() }

// autoFiles is the file system of a job.
type autoFiles struct{ edge EdgeFS }

// Exists says whether the path is there.
func (f autoFiles) Exists(path string) bool {
	_, err := f.edge.Stat(path)

	return err == nil
}

// Read reads a file.
func (f autoFiles) Read(path string) (string, error) {
	data, err := f.edge.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading: %w", err)
	}

	return string(data), nil
}

// Write writes a file, and the directory it is in.
func (f autoFiles) Write(path, text string) error {
	err := f.edge.MkdirAll(filepath.Dir(path), autoDirPerm)
	if err == nil {
		err = f.edge.WriteFileAtomic(path, []byte(text), autoFilePerm)
	}

	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}

// autoOffers lists the pending offers of the vault as the pending-offer bookkeeping tells them.
type autoOffers struct{ edge EdgeFS }

// Pending lists the file names of the pending offers of the vault, in name order.
func (o autoOffers) Pending(vault string) ([]string, error) {
	return pendingOfferNames(vault, o.edge.ReadDir, o.edge.ReadFile)
}

// autoDirectories are where the vault and the chunk index are, resolved as the engram commands resolve them.
func autoDirectories(deps Deps) (vault, chunks string) {
	home := homeOrEmpty(deps)

	return resolveVault("", home, deps.Getenv), ResolveChunksDir("", home, deps.Getenv)
}

// autoTargets returns `engram auto run`.
func autoTargets(deps Deps, errHandler func(error)) any {
	return targ.Group("auto",
		targ.Targ(func(ctx context.Context) {
			errHandler(runAuto(ctx, deps))
		}).Name("run").Description(
			"Advance one job of the memory mod: one JSON job on standard input, one JSON response on standard output"),
	)
}

// runAuto advances one job of the memory mod (793b368, 793e8m): the job is the JSON on standard input, the response the
// JSON on standard output.
func runAuto(ctx context.Context, deps Deps) error {
	input, err := io.ReadAll(deps.Stdin)
	if err != nil {
		return fmt.Errorf("auto run: reading the job: %w", err)
	}

	vault, chunks := autoDirectories(deps)

	output, err := auto.Run(auto.Deps{
		Vault: newAutoVault(ctx, deps),
		Files: autoFiles{edge: deps.FS}, Offers: autoOffers{edge: deps.FS}, Clock: autoClock{now: deps.Now},
		VaultPath: vault, ChunksDir: chunks,
	}, input)
	if err != nil {
		return fmt.Errorf("auto run: %w", err)
	}

	_, err = deps.Stdout.Write(output)
	if err != nil {
		return fmt.Errorf("auto run: writing the response: %w", err)
	}

	return nil
}
