package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ShowChunkArgs holds parsed flags for `engram show-chunk`. Ref is a chunk id
// (source#anchor) — the same id `engram query` / `query-chunks` emit per item.
type ShowChunkArgs struct {
	Ref       string `targ:"positional,required,desc=chunk id: source#anchor"`
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// ShowChunkDeps holds injected dependencies for RunShowChunk. The command is
// read-only; it reuses the same index-loading path as the chunk query.
type ShowChunkDeps struct {
	// ListIndexes returns the .jsonl index file paths under a chunks dir.
	ListIndexes func(dir string) ([]string, error)
	ReadFile    func(path string) ([]byte, error)
}

// RunShowChunk fetches a single chunk's text by its source#anchor id. With
// `engram query --lazy-chunks` omitting chunk content, recall uses this to pull
// a specific chunk's evidence on demand. The chunk id is the record's
// Source + "#" + Anchor (the same form ingest writes), so the lookup compares
// that full id rather than splitting the ref — robust even if an anchor were to
// contain '#'. Not found is a loud error. Read-only; no writes.
func RunShowChunk(_ context.Context, args ShowChunkArgs, deps ShowChunkDeps, stdout io.Writer) error {
	ref := strings.TrimSpace(args.Ref)
	if ref == "" {
		return errShowChunkEmptyRef
	}

	records, err := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListIndexes,
		ReadFile:    deps.ReadFile,
	})
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Source+chunkRefSep+record.Anchor == ref {
			return writeChunkText(stdout, record.Text)
		}
	}

	return fmt.Errorf("%w: %s", errShowChunkNotFound, ref)
}

// unexported constants.
const (
	chunkRefSep = "#"
)

// unexported variables.
var (
	errShowChunkEmptyRef = errors.New("show-chunk: empty chunk ref")
	errShowChunkNotFound = errors.New("chunk not found")
)

// newOsShowChunkDeps wires the production filesystem index loader for
// `engram show-chunk`. No embedder is needed — lookup is by id, not similarity.
func newOsShowChunkDeps() ShowChunkDeps {
	fs := &osEmbedFS{}

	return ShowChunkDeps{
		ListIndexes: listJSONLIndexes,
		ReadFile:    fs.Read,
	}
}

// writeChunkText prints the chunk text, ensuring a trailing newline for clean
// terminal output (mirrors `engram show`).
func writeChunkText(stdout io.Writer, text string) error {
	out := text
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	_, err := io.WriteString(stdout, out)
	if err != nil {
		return fmt.Errorf("show-chunk: writing text: %w", err)
	}

	return nil
}
