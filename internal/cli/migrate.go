package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/toejough/engram/internal/vaultgraph"
)

// MigrateArgs holds parsed flags for `engram migrate-links`.
type MigrateArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Apply     bool   `targ:"flag,name=apply,desc=write changes (default: dry-run report only)"`
}

// MigrateDeps holds injected dependencies for RunMigrateLinks.
type MigrateDeps struct {
	Scan  func(vault string) ([]vaultgraph.Note, error)
	Read  func(path string) ([]byte, error)
	Write func(path string, data []byte) error
}

// RunMigrateLinks rewrites bare-id relation links to full basenames (D1/G0) in
// every note's "Related to:" section. Without --apply it reports what would
// change without writing. Idempotent: an already-migrated link no longer matches
// a bare id, so re-running is a no-op.
func RunMigrateLinks(
	_ context.Context,
	args MigrateArgs,
	deps MigrateDeps,
	stdout io.Writer,
) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("migrate-links: scan: %w", err)
	}

	basenames := make([]string, len(notes))
	for i, note := range notes {
		basenames[i] = note.Basename
	}

	idToBasename := indexBasenamesByID(basenames)

	notesChanged, linksChanged := 0, 0

	for _, note := range notes {
		relPath := pathOf(note.Basename)
		full := filepath.Join(args.VaultPath, relPath)

		body, readErr := deps.Read(full)
		if readErr != nil {
			return fmt.Errorf("migrate-links: read %s: %w", relPath, readErr)
		}

		newBody, count := migrateRelationLinks(string(body), idToBasename)
		if count == 0 {
			continue
		}

		notesChanged++
		linksChanged += count

		verb := "would-rewrite"

		if args.Apply {
			writeErr := deps.Write(full, []byte(newBody))
			if writeErr != nil {
				return fmt.Errorf("migrate-links: write %s: %w", relPath, writeErr)
			}

			verb = "rewrote"
		}

		_, _ = fmt.Fprintf(stdout, "%s %s (%d links)\n", verb, relPath, count)
	}

	mode := "dry-run (use --apply to write)"
	if args.Apply {
		mode = "applied"
	}

	_, _ = fmt.Fprintf(stdout, "%s: %d notes, %d links\n", mode, notesChanged, linksChanged)

	return nil
}

// newOsMigrateDeps wires RunMigrateLinks to the real filesystem.
func newOsMigrateDeps() MigrateDeps {
	const perm = 0o600

	return MigrateDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
		Read: (&osVaultFS{}).ReadFile,
		Write: func(path string, data []byte) error {
			err := atomicWriteFile(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
	}
}
