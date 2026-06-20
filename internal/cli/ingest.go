package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/transcript"
)

// IngestArgs holds parsed flags for `engram ingest`.
type IngestArgs struct {
	Transcripts []string `targ:"flag,name=transcript,desc=session transcript (JSONL) to chunk+embed (repeatable)"`
	Markdowns   []string `targ:"flag,name=markdown,desc=markdown file to chunk+embed (repeatable)"`
	Sweep       []string `targ:"flag,name=sweep,desc=directory to scan for new/changed sources (.md + .jsonl; repeatable)"`
	Auto        bool     `targ:"flag,name=auto,desc=sweep the declarative default roots: repo markdown + ancestor .claude dirs + session logs (see .engram/sweep.json)"` //nolint:lll // single unbreakable struct-tag string
	ChunksDir   string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// IngestDeps holds injected dependencies for RunIngest. ReadTranscript
// produces stripped USER:/ASSISTANT: text for a session path (wired to
// transcript.JSONLReader.ReadFrom in production).
type IngestDeps struct {
	ReadFile       func(path string) ([]byte, error)
	WriteFile      func(path string, data []byte) error
	Stat           func(path string) (SourceStat, error)
	ListSources    func(root SweepRoot) ([]string, error)
	ReadTranscript func(path string, from time.Time, budget int) (transcript.ReadResult, error)
	Embedder       embed.Embedder
	// Now returns the current wall-clock time for IngestedAt stamping. Nil-safe:
	// callers guard with "if deps.Now != nil" before calling. Wire time.Now in
	// newOsIngestDeps.
	Now func() time.Time
	// IsDir, Getwd, and SessionDir feed --auto's declarative root resolution.
	IsDir      func(path string) bool
	Getwd      func() (string, error)
	SessionDir func(cwd string) string
}

// SourceStat is the cheap staleness signature of a source file. A sweep
// skips any file whose mtime+size match the manifest without reading it.
type SourceStat struct {
	MtimeUnixNano int64 `json:"mtime_unix_nano"` //nolint:tagliatelle // manifest schema uses snake_case like .vec.json
	Size          int64 `json:"size"`
}

// ResolveChunksDir resolves the chunk index location with the same precedence
// as the vault: explicit flag, then ENGRAM_CHUNKS_DIR, then the XDG data dir
// default ($XDG_DATA_HOME/engram/chunks).
func ResolveChunksDir(flagValue, home string, getenv func(string) string) string {
	if flagValue != "" {
		return flagValue
	}

	if env := getenv("ENGRAM_CHUNKS_DIR"); env != "" {
		return env
	}

	return filepath.Join(DataDirFromHome(home, getenv), "chunks")
}

// RunIngest trues the chunk index up against the given sources. ONE mechanism
// for every source kind: a manifest (mtime/size/hash) detects change, and a
// changed source's index file is REBUILT wholesale — re-chunked with vectors
// reused by chunk hash, so unchanged sections never re-embed and stale chunks
// from edited content disappear. Zero-LLM; no agent involvement.
func RunIngest(ctx context.Context, args IngestArgs, deps IngestDeps, stdout io.Writer) error {
	sources, err := gatherSources(args, deps)
	if err != nil {
		return err
	}

	manifest, err := readManifest(args.ChunksDir, deps)
	if err != nil {
		return err
	}

	changed := false

	for _, source := range sources {
		didChange, err := ingestSource(ctx, source.path, args.ChunksDir, deps, manifest, stdout)
		if err != nil {
			// A swept source vanishing between walk and read is normal life
			// (cleanup races, deleted sessions) — skip it and keep going so
			// one ghost file can't abort the run or lose the manifest.
			// Explicitly-named sources still error loudly.
			if !source.explicit && errorsIsReadFailure(err) {
				_, _ = fmt.Fprintf(stdout, "skip %s: %v\n", source.path, err)

				continue
			}

			return err
		}

		changed = changed || didChange
	}

	if changed {
		return writeManifestFile(args.ChunksDir, manifest, deps)
	}

	return nil
}

// unexported constants.
const (
	chunkMaxChars    = 1500
	chunkTargetChars = 500
	// ingestBudgetBytes bounds a single transcript read; generous because
	// ingestion is offline (no agent context at stake).
	ingestBudgetBytes = 10 * 1024 * 1024
	jsonlExt          = ".jsonl"
	manifestName      = "manifest.json"
)

// ingestManifest maps source path -> staleness signature.
type ingestManifest map[string]manifestEntry

// manifestEntry records what was last ingested for one source.
type manifestEntry struct {
	SourceStat

	FileHash string `json:"file_hash"` //nolint:tagliatelle // manifest schema uses snake_case like .vec.json
}

// sourceRef tags a source path with how it was selected: explicit flags must
// fail loudly when unreadable; swept files may vanish benignly.
type sourceRef struct {
	path     string
	explicit bool
}

// assembleSweepRoots combines manual --sweep roots (default excludes, hidden
// pruned) with --auto's declaratively-resolved roots.
func assembleSweepRoots(args IngestArgs, deps IngestDeps) ([]SweepRoot, error) {
	defaultExcludes := DefaultSweepSpec().ExcludeDirs
	roots := make([]SweepRoot, 0, len(args.Sweep))

	for _, manual := range args.Sweep {
		roots = append(roots, SweepRoot{Path: manual, ExcludeDirs: defaultExcludes, SkipHidden: true})
	}

	if args.Auto {
		spec, env, err := resolveAutoSpec(deps)
		if err != nil {
			return nil, err
		}

		roots = append(roots, ResolveSweepRoots(spec, env)...)
	}

	return roots, nil
}

// buildChunkIDSet loads all chunk records from chunksDir and returns a
// set of "source#anchor" id strings. This is an O(total chunks) scan;
// callers should build it once and reuse the map for O(1) validation.
// DI-compliant: accepts injected listIndexes and readFile funcs (no os.* calls).
// Returns map[string]bool for consistent membership testing (Component 5 reuses this).
func buildChunkIDSet(
	chunksDir string,
	listIndexes func(dir string) ([]string, error),
	readFile func(path string) ([]byte, error),
) (map[string]bool, error) {
	paths, err := listIndexes(chunksDir)
	if err != nil {
		return nil, fmt.Errorf("ingest: listing chunk indexes for id-set: %w", err)
	}

	idSet := make(map[string]bool)

	for _, path := range paths {
		data, readErr := readFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("ingest: reading chunk index %s for id-set: %w", path, readErr)
		}

		records, decodeErr := chunk.DecodeRecords(data)
		if decodeErr != nil {
			return nil, fmt.Errorf("ingest: decoding chunk index %s for id-set: %w", path, decodeErr)
		}

		for _, r := range records {
			idSet[r.Source+"#"+r.Anchor] = true
		}
	}

	return idSet, nil
}

// chunkSource dispatches by extension: transcripts strip+turn-chunk, markdown
// heading-chunks. Returns (chunks, sourceTimestamp, err). For transcripts,
// sourceTimestamp is ReadResult.LastTimestamp (used as IngestedAt for all chunks
// from this call — a per-session approximation; see Task 2.4 comment). For
// markdown, sourceTimestamp is zero (caller uses deps.Now()).
func chunkSource(source string, raw []byte, deps IngestDeps) ([]chunk.Chunk, time.Time, error) {
	if filepath.Ext(source) == jsonlExt {
		result, err := deps.ReadTranscript(source, time.Time{}, ingestBudgetBytes)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("ingest: stripping transcript %s: %w", source, err)
		}

		return chunk.Transcript(result.Content, chunkTargetChars, chunkMaxChars), result.LastTimestamp, nil
	}

	return chunk.Markdown(string(raw), chunkMaxChars), time.Time{}, nil
}

// defaultSessionDir is the root of ALL recorded session transcripts:
// ENGRAM_TRANSCRIPT_DIR when set (headless/eval cells get only their own
// sessions), else ~/.claude/projects — every project, every conversation.
func defaultSessionDir(_ string) string {
	if dir := os.Getenv("ENGRAM_TRANSCRIPT_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".claude", "projects")
}

// errorsIsReadFailure reports whether ingesting failed at the source-read
// stage (the only stage where a vanished swept file is expected and safe to
// skip — embed/write failures must still surface).
func errorsIsReadFailure(err error) bool {
	return strings.Contains(err.Error(), "ingest: reading ") ||
		strings.Contains(err.Error(), "ingest: stripping transcript ")
}

// gatherSources merges explicit flags, --auto's declarative roots, and manual
// sweep roots into one source list.
func gatherSources(args IngestArgs, deps IngestDeps) ([]sourceRef, error) {
	sources := make([]sourceRef, 0, len(args.Transcripts)+len(args.Markdowns))
	for _, path := range args.Transcripts {
		sources = append(sources, sourceRef{path: path, explicit: true})
	}

	for _, path := range args.Markdowns {
		sources = append(sources, sourceRef{path: path, explicit: true})
	}

	roots, err := assembleSweepRoots(args, deps)
	if err != nil {
		return nil, err
	}

	for _, root := range roots {
		found, err := deps.ListSources(root)
		if err != nil {
			return nil, fmt.Errorf("ingest: sweeping %s: %w", root.Path, err)
		}

		chunksPrefix := filepath.Clean(args.ChunksDir) + string(filepath.Separator)

		for _, path := range found {
			// Never self-ingest the chunk index files or manifest: a sweep
			// root that contains the chunks dir must skip it.
			if strings.HasPrefix(filepath.Clean(path), chunksPrefix) {
				continue
			}

			ext := filepath.Ext(path)
			if ext == ".md" || ext == jsonlExt {
				sources = append(sources, sourceRef{path: path})
			}
		}
	}

	return sources, nil
}

// hashBytes returns the manifest file-hash for raw source bytes.
func hashBytes(raw []byte) string {
	sum := sha256.Sum256(raw)

	return "sha256:" + hex.EncodeToString(sum[:])
}

// ingestSource checks one source against the manifest and rebuilds its index
// file when changed. Returns whether anything was written.
func ingestSource(
	ctx context.Context,
	source, chunksDir string,
	deps IngestDeps,
	manifest ingestManifest,
	stdout io.Writer,
) (bool, error) {
	prior, known := manifest[source]

	if known && deps.Stat != nil {
		stat, err := deps.Stat(source)
		if err == nil && stat == prior.SourceStat {
			return false, nil // cheap skip: mtime+size unchanged, no read
		}
	}

	raw, err := deps.ReadFile(source)
	if err != nil {
		return false, fmt.Errorf("ingest: reading %s: %w", source, err)
	}

	fileHash := hashBytes(raw)
	if known && fileHash == prior.FileHash {
		manifest[source] = manifestEntry{SourceStat: statOrZero(deps, source), FileHash: fileHash}

		return true, nil // touched but identical: refresh stat, keep index
	}

	chunks, sourceTS, err := chunkSource(source, raw, deps)
	if err != nil {
		return false, err
	}

	rebuilt, reused, embedded, err := rebuildIndex(
		ctx, source, chunks, chunksDir, deps, ingestTimeFor(sourceTS, deps), manifestBackfill(manifest))
	if err != nil {
		return false, err
	}

	manifest[source] = manifestEntry{SourceStat: statOrZero(deps, source), FileHash: fileHash}

	_, _ = fmt.Fprintf(stdout, "ingest %s: %d chunks (%d reused, %d embedded)\n",
		source, rebuilt, reused, embedded)

	return true, nil
}

// ingestTimeFor resolves the IngestedAt stamp for a source: the per-session
// sourceTS for transcripts (LastTimestamp), else deps.Now() for markdown. Zero
// when both are absent (test fixtures that omit the clock) — nil-safe.
func ingestTimeFor(sourceTS time.Time, deps IngestDeps) time.Time {
	if !sourceTS.IsZero() {
		return sourceTS
	}

	if deps.Now == nil {
		return time.Time{}
	}

	return deps.Now()
}

// loadPriorRecords maps chunk ContentHash -> full Record from the existing
// index file. An absent or unreadable index returns an empty map (first ingest).
// The full Record is returned so IngestedAt survives the merge (D5).
func loadPriorRecords(indexPath string, deps IngestDeps) map[string]chunk.Record {
	records := map[string]chunk.Record{}

	data, err := deps.ReadFile(indexPath)
	if err != nil {
		return records
	}

	decoded, err := chunk.DecodeRecords(data)
	if err != nil {
		return records
	}

	for _, r := range decoded {
		records[r.ContentHash] = r
	}

	return records
}

// manifestBackfill returns a closure mapping a source path to its manifest
// mtime, used to backfill IngestedAt on legacy (zero-IngestedAt) records during
// merge. Sources absent from the manifest yield the zero time (no backfill).
func manifestBackfill(manifest ingestManifest) func(source string) time.Time {
	return func(src string) time.Time {
		entry, ok := manifest[src]
		if !ok {
			return time.Time{}
		}

		return time.Unix(0, entry.MtimeUnixNano)
	}
}

// mergeChunkRecords builds the append-only merged record set: prior records are
// preserved (never deleted, zero-IngestedAt legacy records backfilled via
// backfillTime), and new-hash chunks are embedded and stamped with ingestTime.
func mergeChunkRecords(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	priorRecords map[string]chunk.Record,
	deps IngestDeps,
	ingestTime time.Time,
	backfillTime func(source string) time.Time,
) (merged []chunk.Record, reused, embedded int, err error) {
	// Preserve prior records first (append-only: never delete).
	existingHashes := make(map[string]bool, len(priorRecords))
	merged = make([]chunk.Record, 0, len(priorRecords)+len(chunks))

	for _, r := range priorRecords {
		existingHashes[r.ContentHash] = true
		if r.IngestedAt.IsZero() && backfillTime != nil {
			r.IngestedAt = backfillTime(r.Source)
		}

		merged = append(merged, r)
	}

	for _, piece := range chunks {
		hash := chunk.HashText(piece.Text)
		if existingHashes[hash] {
			reused++

			continue
		}

		vector, embedErr := deps.Embedder.Embed(ctx, piece.Text)
		if embedErr != nil {
			return nil, 0, 0, fmt.Errorf("ingest: embedding chunk %s/%s: %w", source, piece.Anchor, embedErr)
		}

		embedded++

		merged = append(merged, chunk.Record{
			Source:      source,
			Anchor:      piece.Anchor,
			ContentHash: hash,
			Text:        piece.Text,
			Vector:      vector,
			IngestedAt:  ingestTime,
		})
	}

	return merged, reused, embedded, nil
}

// newOsIngestDeps wires the production filesystem, JSONL transcript reader,
// and bundled embedder for `engram ingest`. WriteFile creates the chunks
// directory on demand so first ingest into a fresh dir succeeds.
func newOsIngestDeps() IngestDeps {
	fs := &osEmbedFS{}
	reader := transcript.NewJSONLReader(&osFileReader{})

	return IngestDeps{
		ReadFile: fs.Read,
		WriteFile: func(path string, data []byte) error {
			const dirPerm = 0o700

			err := os.MkdirAll(filepath.Dir(path), dirPerm)
			if err != nil {
				return fmt.Errorf("ingest: creating chunks dir: %w", err)
			}

			return fs.Write(path, data)
		},
		Stat: func(path string) (SourceStat, error) {
			info, err := os.Stat(path)
			if err != nil {
				return SourceStat{}, fmt.Errorf("ingest: stat %s: %w", path, err)
			}

			return SourceStat{MtimeUnixNano: info.ModTime().UnixNano(), Size: info.Size()}, nil
		},
		ListSources:    walkSourcesExcluding,
		ReadTranscript: reader.ReadFrom,
		Embedder:       sharedEmbedder,
		Now:            time.Now,
		IsDir: func(path string) bool {
			info, err := os.Stat(path)

			return err == nil && info.IsDir()
		},
		Getwd:      os.Getwd,
		SessionDir: defaultSessionDir,
	}
}

// readManifest loads the chunks dir's manifest; absent = empty (first run).
func readManifest(chunksDir string, deps IngestDeps) (ingestManifest, error) {
	manifest := ingestManifest{}

	data, err := deps.ReadFile(filepath.Join(chunksDir, manifestName))
	if err != nil {
		return manifest, nil //nolint:nilerr // absence is the expected first-run case
	}

	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, fmt.Errorf("ingest: reading manifest: %w", err)
	}

	return manifest, nil
}

// rebuildIndex merge-appends: prior records are preserved (never deleted), new
// hashes are embedded and stamped with ingestTime, and zero-IngestedAt legacy
// records are backfilled via backfillTime (nil = no backfill).
func rebuildIndex(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	chunksDir string,
	deps IngestDeps,
	ingestTime time.Time,
	backfillTime func(source string) time.Time,
) (total, reused, embedded int, err error) {
	indexPath := filepath.Join(chunksDir, sourceSlug(source)+jsonlExt)
	priorRecords := loadPriorRecords(indexPath, deps)

	merged, reused, embedded, err := mergeChunkRecords(
		ctx, source, chunks, priorRecords, deps, ingestTime, backfillTime)
	if err != nil {
		return 0, 0, 0, err
	}

	data, err := chunk.EncodeRecords(merged)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: encoding index %s: %w", indexPath, err)
	}

	err = deps.WriteFile(indexPath, data)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: writing index %s: %w", indexPath, err)
	}

	return len(merged), reused, embedded, nil
}

// resolveAutoSpec assembles the sweep environment and loads the repo's
// .engram/sweep.json override (defaults when absent).
func resolveAutoSpec(deps IngestDeps) (SweepSpec, SweepEnv, error) {
	cwd, err := deps.Getwd()
	if err != nil {
		return SweepSpec{}, SweepEnv{}, fmt.Errorf("ingest: getwd: %w", err)
	}

	env := SweepEnv{Cwd: cwd, SessionDir: deps.SessionDir(cwd), IsDir: deps.IsDir}
	spec := DefaultSweepSpec()

	override := filepath.Join(repoRootFor(env), ".engram", "sweep.json")

	raw, readErr := deps.ReadFile(override)
	if readErr == nil {
		spec, err = LoadSweepSpec(raw)
		if err != nil {
			return SweepSpec{}, SweepEnv{}, fmt.Errorf("ingest: %s: %w", override, err)
		}
	}

	return spec, env, nil
}

// shouldPruneDir reports whether a swept subdirectory should be skipped: its
// name is an excluded build/dependency name, or it starts with a
// non-persistent-workspace prefix (a slugified throwaway cwd).
func shouldPruneDir(name string, excludeNames map[string]struct{}, excludePrefixes []string) bool {
	if _, named := excludeNames[name]; named {
		return true
	}

	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// sourceSlug derives a unique index filename for a source: the basename for
// readability plus a short path hash for uniqueness (two README.md files in
// different directories must not share an index file).
func sourceSlug(source string) string {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	sum := sha256.Sum256([]byte(source))

	const hashLen = 8

	return base + "-" + hex.EncodeToString(sum[:])[:hashLen]
}

// statOrZero fetches the current stat signature, tolerating a nil Stat dep.
func statOrZero(deps IngestDeps, path string) SourceStat {
	if deps.Stat == nil {
		return SourceStat{}
	}

	stat, err := deps.Stat(path)
	if err != nil {
		return SourceStat{}
	}

	return stat
}

// walkSourcesExcluding lists files under root, pruning excluded directory
// names (build/dependency trees) and, when SkipHidden, every dot-directory.
func walkSourcesExcluding(root SweepRoot) ([]string, error) {
	excluded := make(map[string]struct{}, len(root.ExcludeDirs))
	for _, name := range root.ExcludeDirs {
		excluded[name] = struct{}{}
	}

	var paths []string

	err := filepath.WalkDir(root.Path, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path == root.Path {
				return nil
			}

			hidden := root.SkipHidden && strings.HasPrefix(entry.Name(), ".")
			if hidden || shouldPruneDir(entry.Name(), excluded, root.ExcludePrefixes) {
				return filepath.SkipDir
			}

			return nil
		}

		paths = append(paths, path)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ingest: walking %s: %w", root.Path, err)
	}

	return paths, nil
}

// writeManifestFile persists the manifest next to the index files it covers.
func writeManifestFile(chunksDir string, manifest ingestManifest, deps IngestDeps) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("ingest: encoding manifest: %w", err)
	}

	err = deps.WriteFile(filepath.Join(chunksDir, manifestName), data)
	if err != nil {
		return fmt.Errorf("ingest: writing manifest: %w", err)
	}

	return nil
}
