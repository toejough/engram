package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
)

// PruneArgs holds parsed flags for `engram prune`.
type PruneArgs struct {
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
	Empty     bool   `targ:"flag,name=empty,desc=remove 0-byte chunk-index files (regenerable; ranking-neutral)"`
	// Duplicates retroactively collapses every exact-content-hash group in
	// the live manifest down to one canonical member (Unit 3's
	// selectCanonical), removing the index files + manifest entries of the
	// rest — cleanup for duplicates indexed before Unit 3's ingest-time
	// dedup existed. Safe by construction: a duplicate is only removed once
	// its retained twin's index file is verified present.
	Duplicates bool `targ:"flag,name=duplicates,desc=remove non-canonical duplicate index files+manifest entries; keeps one per content hash"` //nolint:lll // single unbreakable struct-tag string
	DryRun     bool `targ:"flag,name=dry-run,desc=report what would be removed without deleting"`
}

// PruneDeps holds injected dependencies for RunPrune.
type PruneDeps struct {
	// Lock acquires an exclusive flock on chunksDir/.manifest.lock and returns a
	// release func. Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newPruneDeps.
	// Guards the manifest read-modify-write against concurrent ingest/prune (#660).
	Lock        func(chunksDir string) (func(), error)
	ReadFile    func(path string) ([]byte, error)
	WriteFile   func(path string, data []byte) error
	Exists      func(path string) bool
	ListIndexes func(dir string) ([]string, error)
	Remove      func(path string) error
	// LogWarning reports a non-fatal failure without stopping the run — used
	// by --duplicates for a per-item removal error ("[FAIL] <path>: <err>").
	// Nil-safe: callers guard with "if deps.LogWarning != nil". Wired to
	// logWarningTo(d.Stderr) in newPruneDeps, matching every other RunX
	// command's warning-reporting convention (activate.go, amend.go).
	LogWarning func(format string, args ...any)
}

// RunPrune detaches dead sources from the chunk index: every manifest source
// whose file no longer exists has its manifest entry dropped, but its
// per-source .jsonl index file (the embedded chunk vectors) is left on disk.
// Chunk search discovers .jsonl files by directory scan and never consults
// the manifest, so detached chunks remain fully searchable — this lets a
// user delete source files without losing the recovered memory. Zero-LLM.
// With --dry-run, the manifest is left unwritten and stdout is prefixed
// "[dry-run] ". With --empty, RunPrune instead delegates to
// pruneEmptyLocked, which DOES remove 0-byte .jsonl index files
// (regenerable; ranking-neutral) — see that helper's doc comment. With
// --duplicates, RunPrune delegates to pruneDuplicatesLocked, which DOES
// remove non-canonical duplicate .jsonl index files (safe by construction —
// see that helper's doc comment); per-item removal failures are reported via
// deps.LogWarning and do not stop the run.
func RunPrune(_ context.Context, args PruneArgs, deps PruneDeps, stdout io.Writer) error {
	// Acquire the manifest lock before any read-modify-write on manifest.json
	// so concurrent ingest/prune runs cannot produce lost updates (#660).
	release, lockErr := acquireOptionalLock(deps.Lock, args.ChunksDir)
	if lockErr != nil {
		return fmt.Errorf("prune: acquiring manifest lock: %w", lockErr)
	}

	defer release()

	if args.Empty {
		return pruneEmptyLocked(args, deps, stdout)
	}

	if args.Duplicates {
		return pruneDuplicatesLocked(args, deps, stdout)
	}

	manifest := ingestManifest{}

	data, err := deps.ReadFile(filepath.Join(args.ChunksDir, manifestName))
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "prune: no manifest, nothing to prune")

		return nil // absent manifest = empty index, not an error
	}

	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return fmt.Errorf("prune: reading manifest: %w", err)
	}

	detached := 0

	for source := range manifest {
		if deps.Exists(source) {
			continue
		}

		delete(manifest, source)

		detached++
	}

	if detached == 0 {
		_, _ = fmt.Fprintln(stdout, "prune: no dead sources")

		return nil
	}

	writeErr := writePrunedManifest(args, deps, manifest)
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintf(stdout, "%sprune: detached %d source(s) — embedded chunks preserved (still searchable)\n",
		dryRunPrefix(args.DryRun), detached)

	return nil
}

// unexported constants.
const (
	// dryRunLinePrefix marks a stdout line as reporting what a --dry-run WOULD
	// do rather than what it did. Shared across prune and update's dry-run
	// report lines (goconst: 3+ occurrences of the same literal).
	dryRunLinePrefix = "[dry-run] "
)

// unexported variables.
var (
	errChunkManifestMalformed = errors.New("malformed chunk manifest")
)

// dryRunPrefix returns the stdout line prefix RunPrune's two modes — the
// default dead-source detach and pruneEmptyLocked's empty-file removal —
// both use to mark a --dry-run report: nothing was written or removed.
func dryRunPrefix(dryRun bool) string {
	if dryRun {
		return dryRunLinePrefix
	}

	return ""
}

// newPruneDeps composes production PruneDeps from the CLI-edge Deps.
func newPruneDeps(d Deps) PruneDeps {
	return PruneDeps{
		Lock:     manifestLockFrom(d),
		ReadFile: d.FS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, indexFilePerm)
			if err != nil {
				return fmt.Errorf("prune: writing %s: %w", path, err)
			}

			return nil
		},
		Exists:      func(path string) bool { return statExists(d.FS.Stat, path) },
		ListIndexes: listJSONLIndexes(d.FS),
		Remove:      d.FS.Remove,
		LogWarning:  logWarningTo(d.Stderr),
	}
}

// pruneEmptyLocked removes 0-byte .jsonl chunk-index files under the chunks
// dir. Empty files hold zero records (a source that yielded no chunks — see the
// rebuildIndex guard), so removing them is ranking-neutral: the loaded record
// set is byte-identical. It re-reads each file at delete time and removes only
// what is genuinely empty NOW, never from a stale enumeration. Runs under the
// manifest lock already held by RunPrune. --dry-run reports counts, deletes
// nothing.
func pruneEmptyLocked(args PruneArgs, deps PruneDeps, stdout io.Writer) error {
	paths, err := deps.ListIndexes(args.ChunksDir)
	if err != nil {
		return fmt.Errorf("prune: listing chunk indexes: %w", err)
	}

	prefix := dryRunPrefix(args.DryRun)

	removed := 0

	for _, path := range paths {
		data, readErr := deps.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("prune: reading %s: %w", path, readErr)
		}

		if len(data) != 0 {
			continue
		}

		if !args.DryRun {
			rmErr := deps.Remove(path)
			if rmErr != nil {
				return fmt.Errorf("prune: removing %s: %w", path, rmErr)
			}
		}

		removed++
	}

	_, _ = fmt.Fprintf(stdout,
		"%sprune: removed %d empty chunk-index file(s) of %d scanned\n",
		prefix, removed, len(paths))

	return nil
}

// readChunkManifest reads and decodes chunksDir's manifest.json via the
// injected readFile func, returning the raw (ingestManifest, error) with no
// imposed error policy — chunkIndexHasPrunableDuplicates treats any error as
// silent-false (a detection failure must never fail `engram update`), while
// pruneDuplicatesLocked distinguishes "no manifest yet" (nilerr, first-run)
// from a malformed manifest (wrapped error) — each caller decides (#714).
func readChunkManifest(chunksDir string, readFile func(string) ([]byte, error)) (ingestManifest, error) {
	data, readErr := readFile(filepath.Join(chunksDir, manifestName))
	if readErr != nil {
		return nil, readErr
	}

	manifest := ingestManifest{}

	unmarshalErr := json.Unmarshal(data, &manifest)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", errChunkManifestMalformed, unmarshalErr)
	}

	return manifest, nil
}

// statExists reports whether path exists by calling the injected stat func
// and treating any error (not found, permission, or otherwise) as absence —
// the shared Exists-via-Stat pattern used by update's dry-run duplicate
// detection and prune's production deps. Generic over the stat func's info
// type (update.FileInfo, fs.FileInfo, ...) so callers don't need to adapt
// return types just to check existence (#714).
func statExists[T any](stat func(string) (T, error), path string) bool {
	_, statErr := stat(path)

	return statErr == nil
}

// writePrunedManifest marshals and writes the detached manifest, unless
// args.DryRun is set — a dry run reports what would change without
// mutating the manifest on disk.
func writePrunedManifest(args PruneArgs, deps PruneDeps, manifest ingestManifest) error {
	if args.DryRun {
		return nil
	}

	out, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("prune: encoding manifest: %w", err)
	}

	err = deps.WriteFile(filepath.Join(args.ChunksDir, manifestName), out)
	if err != nil {
		return fmt.Errorf("prune: writing manifest: %w", err)
	}

	return nil
}
