package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PruneArgs holds parsed flags for `engram prune`.
type PruneArgs struct {
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// PruneDeps holds injected dependencies for RunPrune.
type PruneDeps struct {
	// Lock acquires an exclusive flock on chunksDir/.manifest.lock and returns a
	// release func. Wired to flockPath(chunksDir/.manifest.lock) in newOsPruneDeps.
	// Guards the manifest read-modify-write against concurrent ingest/prune (#660).
	Lock      func(chunksDir string) (func(), error)
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte) error
	Exists    func(path string) bool
	Remove    func(path string) error
}

// RunPrune garbage-collects the chunk index: every manifest source whose file
// no longer exists has its per-source index file deleted and its manifest entry
// dropped. Append-only history is preserved for live sources. Zero-LLM.
func RunPrune(_ context.Context, args PruneArgs, deps PruneDeps, stdout io.Writer) error {
	// Acquire the manifest lock before any read-modify-write on manifest.json
	// so concurrent ingest/prune runs cannot produce lost updates (#660).
	release, lockErr := acquireOptionalLock(deps.Lock, args.ChunksDir)
	if lockErr != nil {
		return fmt.Errorf("prune: acquiring manifest lock: %w", lockErr)
	}

	defer release()

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

	pruned := 0

	for source := range manifest {
		if deps.Exists(source) {
			continue
		}

		indexPath := filepath.Join(args.ChunksDir, sourceSlug(source)+jsonlExt)

		err = deps.Remove(indexPath)
		if err != nil {
			return fmt.Errorf("prune: removing index %s: %w", indexPath, err)
		}

		delete(manifest, source)

		pruned++
	}

	if pruned == 0 {
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

	_, _ = fmt.Fprintf(stdout, "prune: removed %d dead source(s)\n", pruned)

	return nil
}

// newOsPruneDeps wires the production filesystem for `engram prune`.
func newOsPruneDeps() PruneDeps {
	fs := &osEmbedFS{}

	return PruneDeps{
		Lock:      osManifestLock,
		ReadFile:  fs.Read,
		WriteFile: fs.Write,
		Exists: func(path string) bool {
			_, statErr := os.Stat(path)

			return statErr == nil
		},
		Remove: os.Remove,
	}
}
