package cli

import (
	"context"
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

// Chunking parameters for the auto-ingested vector space. Target merges
// small turns into meaningful units; max keeps every chunk inside the
// embedder's input window (MiniLM truncates ~1500 chars).
const (
	chunkTargetChars = 500
	chunkMaxChars    = 1500
	// ingestBudgetBytes bounds a single transcript read; generous because
	// ingestion is offline (no agent context at stake).
	ingestBudgetBytes = 10 * 1024 * 1024
)

// IngestArgs holds parsed flags for `engram ingest`.
type IngestArgs struct {
	Transcripts []string `targ:"flag,name=transcript,desc=session transcript (JSONL) to chunk+embed (repeatable)"`
	Markdowns   []string `targ:"flag,name=markdown,desc=markdown file to chunk+embed (repeatable)"`
	ChunksDir   string   `targ:"flag,name=chunks-dir,required,desc=directory for per-source chunk index (.jsonl) files"`
}

// IngestDeps holds injected dependencies for RunIngest. ReadTranscript
// produces stripped USER:/ASSISTANT: text for a session path (wired to
// transcript.JSONLReader.ReadFrom in production).
type IngestDeps struct {
	ReadFile       func(path string) ([]byte, error)
	WriteFile      func(path string, data []byte) error
	ReadTranscript func(path string, from time.Time, budget int) (transcript.ReadResult, error)
	Embedder       embed.Embedder
}

// RunIngest chunks and embeds the given sources into per-source .jsonl chunk
// indexes under ChunksDir. Re-runs are idempotent: chunks whose content hash
// is already indexed are skipped. This is the zero-LLM write path of the
// auto-chunk memory experiment — no agent involvement.
func RunIngest(ctx context.Context, args IngestArgs, deps IngestDeps, stdout io.Writer) error {
	for _, path := range args.Transcripts {
		result, err := deps.ReadTranscript(path, time.Time{}, ingestBudgetBytes)
		if err != nil {
			return fmt.Errorf("ingest: reading transcript %s: %w", path, err)
		}

		chunks := chunk.Transcript(result.Content, chunkTargetChars, chunkMaxChars)
		if err := indexChunks(ctx, path, chunks, args.ChunksDir, deps, stdout); err != nil {
			return err
		}
	}

	for _, path := range args.Markdowns {
		raw, err := deps.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ingest: reading markdown %s: %w", path, err)
		}

		chunks := chunk.Markdown(string(raw), chunkMaxChars)
		if err := indexChunks(ctx, path, chunks, args.ChunksDir, deps, stdout); err != nil {
			return err
		}
	}

	return nil
}

// indexChunks merges new chunks into the source's index file, embedding only
// chunks whose hash is not already present.
func indexChunks(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	chunksDir string,
	deps IngestDeps,
	stdout io.Writer,
) error {
	indexPath := filepath.Join(chunksDir, sourceSlug(source)+".jsonl")

	existing, seen, err := readIndex(indexPath, deps)
	if err != nil {
		return err
	}

	added := 0

	for _, c := range chunks {
		hash := chunk.HashText(c.Text)
		if _, ok := seen[hash]; ok {
			continue
		}

		vector, err := deps.Embedder.Embed(ctx, c.Text)
		if err != nil {
			return fmt.Errorf("ingest: embedding chunk %s/%s: %w", source, c.Anchor, err)
		}

		existing = append(existing, chunk.Record{
			Source: source, Anchor: c.Anchor, ContentHash: hash, Text: c.Text, Vector: vector,
		})
		seen[hash] = struct{}{}
		added++
	}

	if added == 0 {
		_, _ = fmt.Fprintf(stdout, "ingest %s: up to date (%d chunks)\n", source, len(existing))

		return nil
	}

	data, err := chunk.EncodeRecords(existing)
	if err != nil {
		return fmt.Errorf("ingest: encoding index %s: %w", indexPath, err)
	}

	if err := deps.WriteFile(indexPath, data); err != nil {
		return fmt.Errorf("ingest: writing index %s: %w", indexPath, err)
	}

	_, _ = fmt.Fprintf(stdout, "ingest %s: +%d chunks (%d total)\n", source, added, len(existing))

	return nil
}

// readIndex loads an existing index file, tolerating absence (first ingest).
func readIndex(indexPath string, deps IngestDeps) ([]chunk.Record, map[string]struct{}, error) {
	seen := map[string]struct{}{}

	data, err := deps.ReadFile(indexPath)
	if err != nil {
		// Absent index = first ingest of this source; any read error on a
		// present file surfaces later as a write conflict, not silent loss.
		return nil, seen, nil //nolint:nilerr // absence is the expected first-run case
	}

	records, err := chunk.DecodeRecords(data)
	if err != nil {
		return nil, nil, fmt.Errorf("ingest: reading index %s: %w", indexPath, err)
	}

	for _, r := range records {
		seen[r.ContentHash] = struct{}{}
	}

	return records, seen, nil
}

// sourceSlug derives the index filename from the source path basename.
func sourceSlug(source string) string {
	base := filepath.Base(source)

	return strings.TrimSuffix(base, filepath.Ext(base))
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
			if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
				return fmt.Errorf("ingest: creating chunks dir: %w", err)
			}

			return fs.Write(path, data)
		},
		ReadTranscript: reader.ReadFrom,
		Embedder:       sharedEmbedder,
	}
}
