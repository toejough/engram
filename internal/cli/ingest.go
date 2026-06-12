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
	ChunksDir   string   `targ:"flag,name=chunks-dir,required,desc=directory for per-source chunk index (.jsonl) files"`
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

// chunkSource dispatches by extension: transcripts strip+turn-chunk, markdown
// heading-chunks. Same mechanism either way; only the chunker differs.
func chunkSource(source string, raw []byte, deps IngestDeps) ([]chunk.Chunk, error) {
	if filepath.Ext(source) == jsonlExt {
		result, err := deps.ReadTranscript(source, time.Time{}, ingestBudgetBytes)
		if err != nil {
			return nil, fmt.Errorf("ingest: stripping transcript %s: %w", source, err)
		}

		return chunk.Transcript(result.Content, chunkTargetChars, chunkMaxChars), nil
	}

	return chunk.Markdown(string(raw), chunkMaxChars), nil
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

	chunks, err := chunkSource(source, raw, deps)
	if err != nil {
		return false, err
	}

	rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps)
	if err != nil {
		return false, err
	}

	manifest[source] = manifestEntry{SourceStat: statOrZero(deps, source), FileHash: fileHash}

	_, _ = fmt.Fprintf(stdout, "ingest %s: %d chunks (%d reused, %d embedded)\n",
		source, rebuilt, reused, embedded)

	return true, nil
}

// loadPriorVectors maps chunk hash -> vector from the existing index file.
// An absent or unreadable index is an empty map (first ingest).
func loadPriorVectors(indexPath string, deps IngestDeps) map[string][]float32 {
	vectors := map[string][]float32{}

	data, err := deps.ReadFile(indexPath)
	if err != nil {
		return vectors
	}

	records, err := chunk.DecodeRecords(data)
	if err != nil {
		return vectors
	}

	for _, r := range records {
		vectors[r.ContentHash] = r.Vector
	}

	return vectors
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

// rebuildIndex writes the source's index file from scratch, reusing vectors
// from the previous index by chunk hash so unchanged content never re-embeds.
// Stale chunks vanish because the file is replaced, not appended to.
func rebuildIndex(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	chunksDir string,
	deps IngestDeps,
) (total, reused, embedded int, err error) {
	indexPath := filepath.Join(chunksDir, sourceSlug(source)+jsonlExt)
	priorVectors := loadPriorVectors(indexPath, deps)

	records := make([]chunk.Record, 0, len(chunks))

	for _, piece := range chunks {
		hash := chunk.HashText(piece.Text)

		vector, ok := priorVectors[hash]
		if ok {
			reused++
		} else {
			vector, err = deps.Embedder.Embed(ctx, piece.Text)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("ingest: embedding chunk %s/%s: %w", source, piece.Anchor, err)
			}

			embedded++
		}

		records = append(records, chunk.Record{
			Source: source, Anchor: piece.Anchor, ContentHash: hash, Text: piece.Text, Vector: vector,
		})
	}

	data, err := chunk.EncodeRecords(records)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: encoding index %s: %w", indexPath, err)
	}

	err = deps.WriteFile(indexPath, data)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: writing index %s: %w", indexPath, err)
	}

	return len(records), reused, embedded, nil
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

			_, named := excluded[entry.Name()]
			hidden := root.SkipHidden && strings.HasPrefix(entry.Name(), ".")

			if named || hidden {
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
