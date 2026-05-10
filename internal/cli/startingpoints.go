package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"engram/internal/vaultgraph"
)

// errVaultPathRequired is returned when neither --vault nor ENGRAM_VAULT_PATH is set.
var errVaultPathRequired = errors.New(
	"starting-points: vault path required (pass --vault or set ENGRAM_VAULT_PATH)",
)

// runStartingPoints prints one wikilink per starting point of the vault graph.
// Starting points = every MOC plus the per-component winner of every MOC-less
// connected component. Output is globally sorted by Luhmann tree order.
func runStartingPoints(_ context.Context, args StartingPointsArgs, stdout io.Writer) error {
	if args.VaultPath == "" {
		return errVaultPathRequired
	}

	points, err := vaultgraph.StartingPoints(&osVaultFS{}, args.VaultPath)
	if err != nil {
		return fmt.Errorf("starting-points: %w", err)
	}

	for _, name := range points {
		_, writeErr := fmt.Fprintln(stdout, "[["+name+"]]")
		if writeErr != nil {
			return fmt.Errorf("starting-points: writing output: %w", writeErr)
		}
	}

	return nil
}

// osVaultFS is the production adapter satisfying vaultgraph.VaultFS. Listing a
// non-existent directory returns an empty slice (not an error) — the scanner
// uses this to skip missing subdirs like an absent Fleeting/.
type osVaultFS struct{}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (*osVaultFS) ListMD(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		out = append(out, entry.Name())
	}

	return out, nil
}

// ReadFile reads the file at path.
func (*osVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}
