// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	errNotADirectory    = errors.New("not a directory")
	errRecallPathFormat = errors.New(
		"must be a full relative path of the form MOCs/<basename>.md or Permanent/<basename>.md",
	)
	errRecallVaultMissing = errors.New(
		"recall: vault not found (set --vault or ENGRAM_VAULT_PATH, or run `engram learn` to bootstrap the default)",
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

	fileDescriptor := int(f.Fd())

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

// MkdirAll creates path with any missing parents; no-op when path exists.
func (*osLearnFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

// StatDir returns fs.ErrNotExist if the directory is missing, errNotADirectory
// if the path exists but is a file, or a wrapped error otherwise.
func (*osLearnFS) StatDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fs.ErrNotExist
		}

		return fmt.Errorf("stat: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", errNotADirectory, path)
	}

	return nil
}

// WriteFileIfMissing writes data with O_EXCL so existing files are left
// untouched; ErrExist is swallowed so initializeVault is idempotent.
func (*osLearnFS) WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile( //nolint:gosec // path from caller
		path,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		perm,
	)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("open: %w", err)
	}

	defer func() { _ = f.Close() }()

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write: %w", writeErr)
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

// WriteSidecar writes a .vec.json sidecar to path with 0o600 perms. Used
// by autoEmbedNote after a successful note write; lives on osLearnFS so
// the production wiring uses a named method (visible to coverage) instead
// of an anonymous closure.
func (*osLearnFS) WriteSidecar(path string, data []byte) error {
	const perm = 0o600

	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}

	return nil
}

// buildSubdirMap returns a basename→subdir-name lookup for every note in the vault.
// "MOCs" or "Permanent" — used to format recall output as full relative paths.
func buildSubdirMap(notes []vaultgraph.Note) map[string]bool {
	out := make(map[string]bool, len(notes))

	for _, note := range notes {
		out[note.Basename] = note.IsMOC
	}

	return out
}

func emitRelPaths(stdout io.Writer, basenames []string, isMOCByBasename map[string]bool) error {
	for _, basename := range basenames {
		_, err := fmt.Fprintln(stdout, pathOf(basename, isMOCByBasename[basename]))
		if err != nil {
			return fmt.Errorf("recall: writing output: %w", err)
		}
	}

	return nil
}

// newTranscriptDeps constructs production transcript finder and reader,
// combining Claude Code (.jsonl) and OpenCode (SQLite) sources. The cwd
// parameter filters OpenCode sessions to those whose stored directory
// matches cwd or is a subdirectory of cwd.
func newTranscriptDeps(cwd string) (transcript.Finder, transcript.Reader) {
	dbPath := transcript.DefaultOpencodeDBPath()

	claudeFinder := transcript.NewSessionFinder(&osDirLister{})
	ocFinder := transcript.NewOpencodeSessionFinder(dbPath, cwd)
	finder := transcript.NewCompositeSessionFinder(claudeFinder, ocFinder)

	claudeReader := transcript.NewJSONLReader(&osFileReader{})
	ocReader := transcript.NewOpencodeTranscriptReader(dbPath)
	reader := transcript.NewCompositeTranscriptReader(claudeReader, ocReader)

	return finder, reader
}

// parseRecallPath validates a single --follow / --already-read argument and
// returns the basename for graph lookup. Inputs MUST be the exact format that
// recall stdout emits: "<Subdir>/<basename>.md". Anything else is a hard error;
// the previous silent-miss behavior was a footgun.
func parseRecallPath(flag, raw string) (string, error) {
	subdir, rest, ok := strings.Cut(raw, "/")
	if !ok || (subdir != vaultgraph.MOCsSubdir && subdir != vaultgraph.PermanentSubdir) {
		return "", fmt.Errorf("%s: %q: %w", flag, raw, errRecallPathFormat)
	}

	basename, ok := strings.CutSuffix(rest, ".md")
	if !ok || basename == "" {
		return "", fmt.Errorf("%s: %q: %w", flag, raw, errRecallPathFormat)
	}

	return basename, nil
}

func parseRecallPaths(flag string, raws []string) ([]string, error) {
	out := make([]string, 0, len(raws))

	for _, raw := range raws {
		basename, err := parseRecallPath(flag, raw)
		if err != nil {
			return nil, err
		}

		out = append(out, basename)
	}

	return out, nil
}

// pathOf returns the vault-relative path for a note, e.g. "Permanent/foo.md"
// or "MOCs/bar.md". Callers can pass the result directly to Read tools.
func pathOf(basename string, isMOC bool) string {
	subdir := vaultgraph.PermanentSubdir
	if isMOC {
		subdir = vaultgraph.MOCsSubdir
	}

	return subdir + "/" + basename + ".md"
}

func runRecall(_ context.Context, args RecallArgs, stdout io.Writer) error {
	if args.VaultPath == "" {
		return errRecallVaultMissing
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
	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	points, err := vaultgraph.StartingPoints(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return emitRelPaths(stdout, points, buildSubdirMap(notes))
}

func runRecallFollow(fs vaultgraph.VaultFS, args RecallArgs, stdout io.Writer) error {
	follow, err := parseRecallPaths("--follow", args.Follow)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	alreadyRead, err := parseRecallPaths("--already-read", args.AlreadyRead)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	notes, err := vaultgraph.ScanVault(fs, args.VaultPath)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	graph := vaultgraph.BuildGraph(notes)

	return emitRelPaths(
		stdout,
		vaultgraph.Follow(graph, follow, alreadyRead),
		buildSubdirMap(notes),
	)
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

	return emitRelPaths(stdout, vaultgraph.Recent(notes, limit), buildSubdirMap(notes))
}
