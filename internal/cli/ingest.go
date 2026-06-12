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
	Sweep       []string `targ:"flag,name=sweep,desc=directory to scan for new/changed sources (.md + .jsonl, repeatable)"`
	ChunksDir   string   `targ:"flag,name=chunks-dir,required,desc=directory for per-source chunk index (.jsonl) files"`
}

// IngestDeps holds injected dependencies for RunIngest. ReadTranscript
// produces stripped USER:/ASSISTANT: text for a session path (wired to
// transcript.JSONLReader.ReadFrom in production).
type IngestDeps struct {
	ReadFile       func(path string) ([]byte, error)
	WriteFile      func(path string, data []byte) error
	Stat           func(path string) (SourceStat, error)
	ListSources    func(root string) ([]string, error)
	ReadTranscript func(path string, from time.Time, budget int) (transcript.ReadResult, error)
	Embedder       embed.Embedder
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
		didChange, err := ingestSource(ctx, source, args.ChunksDir, deps, manifest, stdout)
		if err != nil {
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

// gatherSources merges explicit flags with sweep results into one list.
func gatherSources(args IngestArgs, deps IngestDeps) ([]string, error) {
	sources := make([]string, 0, len(args.Transcripts)+len(args.Markdowns))
	sources = append(sources, args.Transcripts...)
	sources = append(sources, args.Markdowns...)

	for _, root := range args.Sweep {
		found, err := deps.ListSources(root)
		if err != nil {
			return nil, fmt.Errorf("ingest: sweeping %s: %w", root, err)
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
				sources = append(sources, path)
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
		ListSources: func(root string) ([]string, error) {
			var paths []string

			err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}

				paths = append(paths, path)

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("ingest: walking %s: %w", root, err)
			}

			return paths, nil
		},
		ReadTranscript: reader.ReadFrom,
		Embedder:       sharedEmbedder,
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

// sourceSlug derives the index filename from the source path basename.
func sourceSlug(source string) string {
	base := filepath.Base(source)

	return strings.TrimSuffix(base, filepath.Ext(base))
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
