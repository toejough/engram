// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/toejough/engram/internal/transcript"
	"github.com/toejough/engram/internal/vaultgraph"
)

// unexported constants.
const (
	defaultRecallRecentLimit = 20
	luhmannLockFile          = ".luhmann.lock"
)

// unexported variables.
var (
	errNotADirectory       = errors.New("not a directory")
	errRecallVaultRequired = errors.New(
		"recall: --vault required (or set ENGRAM_VAULT_PATH)",
	)
)

// osDirLister lists .jsonl files in a directory using os.ReadDir.
type osDirLister struct{}

func (l *osDirLister) ListJSONL(dir string) ([]transcript.FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("listing directory: %w", err)
	}

	results := make([]transcript.FileEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		results = append(results, transcript.FileEntry{
			Path:  filepath.Join(dir, name),
			Mtime: info.ModTime(),
		})
	}

	return results, nil
}

// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

// osLearnFS is the production filesystem adapter for the learn subcommand.
type osLearnFS struct{}

// ListIDs returns Luhmann IDs from filenames in vault/Permanent and vault/MOCs.
func (*osLearnFS) ListIDs(vault string) ([]string, error) {
	out := []string{}

	for _, sub := range []string{vaultgraph.PermanentSubdir, vaultgraph.MOCsSubdir} {
		entries, err := os.ReadDir(filepath.Join(vault, sub))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("read %s: %w", sub, err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			id, ok := extractLuhmannFromFilename(e.Name())
			if !ok {
				continue
			}

			out = append(out, id)
		}
	}

	return out, nil
}

// Lock acquires an exclusive flock on vault/.luhmann.lock; returns a release func.
func (*osLearnFS) Lock(vault string) (func(), error) {
	path := filepath.Join(vault, luhmannLockFile)

	const perm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(f.Fd()) //nolint:gosec // fd fits in int on supported platforms

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	release := func() {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		_ = f.Close()
	}

	return release, nil
}

// StatDir returns an error if the directory does not exist or isn't accessible.
func (*osLearnFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}

	return nil
}

// WriteNew creates the file with O_EXCL — errors with fs.ErrExist if it already exists.
func (*osLearnFS) WriteNew(path string, data []byte) error {
	const perm = 0o600

	f, err := os.OpenFile( //nolint:gosec // path from caller
		path,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		perm,
	)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
	}

	return nil
}

func emitBasenames(stdout io.Writer, names []string) error {
	for _, name := range names {
		_, err := fmt.Fprintln(stdout, name)
		if err != nil {
			return fmt.Errorf("recall: writing output: %w", err)
		}
	}

	return nil
}

func runRecall(_ context.Context, args RecallArgs, stdout io.Writer) error {
	if args.VaultPath == "" {
		return errRecallVaultRequired
	}

	fs := &osVaultFS{}

	switch {
	case len(args.Follow) > 0:
		return runRecallFollow(fs, args, stdout)
	case args.Recent:
		return runRecallRecent(fs, args, stdout)
	default:
		return runRecallAnchors(fs, args, stdout)
	}
}

func runRecallAnchors(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	points, err := vaultgraph.StartingPoints(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return emitBasenames(stdout, points)
}

func runRecallFollow(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	graph := vaultgraph.BuildGraph(notes)

	return emitBasenames(stdout, vaultgraph.Follow(graph, args.Follow, args.AlreadyRead))
}

func runRecallRecent(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	limit := args.Limit
	if limit == 0 {
		limit = defaultRecallRecentLimit
	}

	return emitBasenames(stdout, vaultgraph.Recent(notes, limit))
}
