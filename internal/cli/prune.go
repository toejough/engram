package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)

// PruneArgs holds parsed flags for `engram prune`.
type PruneArgs struct {
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
	Empty     bool   `targ:"flag,name=empty,desc=remove 0-byte chunk-index files (regenerable; ranking-neutral)"`
	DryRun    bool   `targ:"flag,name=dry-run,desc=report what would be removed without deleting"`
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
}

// RunPrune detaches dead sources from the chunk index: every manifest source
// whose file no longer exists has its manifest entry dropped, but its
// per-source .jsonl index file (the embedded chunk vectors) is left on disk.
// Chunk search discovers .jsonl files by directory scan and never consults
// the manifest, so detached chunks remain fully searchable — this lets a
// user delete source files without losing the recovered memory. Zero-LLM.
// With --empty, RunPrune instead delegates to pruneEmptyLocked, which DOES
// remove 0-byte .jsonl index files (regenerable; ranking-neutral) — see that
// helper's doc comment.
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

	out, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("prune: encoding manifest: %w", err)
	}

	err = deps.WriteFile(filepath.Join(args.ChunksDir, manifestName), out)
	if err != nil {
		return fmt.Errorf("prune: writing manifest: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "prune: detached %d source(s) — embedded chunks preserved (still searchable)\n", detached)

	return nil
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
		Exists: func(path string) bool {
			_, statErr := d.FS.Stat(path)

			return statErr == nil
		},
		ListIndexes: listJSONLIndexes(d.FS),
		Remove:      d.FS.Remove,
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

	prefix := ""
	if args.DryRun {
		prefix = "[dry-run] "
	}

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
