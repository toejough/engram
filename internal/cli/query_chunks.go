package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
)

// ChunkQueryArgs holds parsed flags for `engram query --space chunks`
// (surfaced as the `query-chunks` target during the experiment).
type ChunkQueryArgs struct {
	Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable)"`
	ChunksDir string   `targ:"flag,name=chunks-dir,required,desc=directory of chunk index (.jsonl) files"`
	Limit     int      `targ:"flag,name=limit,desc=max chunks to return (default 20)"`
}

// ChunkQueryDeps holds injected dependencies for RunChunkQuery.
type ChunkQueryDeps struct {
	// ListIndexes returns the .jsonl index file paths under a chunks dir.
	ListIndexes func(dir string) ([]string, error)
	ReadFile    func(path string) ([]byte, error)
	Embedder    embed.Embedder
}

// RunChunkQuery embeds each phrase, scores every indexed chunk by max cosine
// across phrases, and emits the top Limit chunks plus vector-neighborhood
// clusters as YAML. No graph walk or hubs — chunks carry no authored links;
// clustering substitutes for the wikilink structure the note query relies on.
func RunChunkQuery(ctx context.Context, args ChunkQueryArgs, deps ChunkQueryDeps, stdout io.Writer) error {
	if len(args.Phrases) == 0 {
		return errChunkQueryNoPhrase
	}

	limit := args.Limit
	if limit == 0 {
		limit = defaultQueryLimit
	}

	records, err := loadChunkRecords(args.ChunksDir, deps)
	if err != nil {
		return err
	}

	scored, err := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
	if err != nil {
		return err
	}

	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })

	top := scored
	if len(top) > limit {
		top = top[:limit]
	}

	return writeChunkPayload(stdout, args.Phrases, top, clusterChunks(top), len(records))
}

// unexported constants.
const (
	chunkClusterMaxK       = 7
	chunkClusterMinK       = 2
	chunkClusterMinMembers = 6
	chunkClusterSeed       = 1
	chunkClusterThreshold  = 0.10
)

// unexported variables.
var (
	errChunkQueryNoPhrase = errors.New("query chunks: at least one --phrase is required")
)

// chunkCluster is one vector-neighborhood cluster over the returned chunks.
type chunkCluster struct {
	id      int
	members []int // indexes into the top slice
}

// scoredChunk pairs a record with its best per-phrase cosine score.
type scoredChunk struct {
	record chunk.Record
	score  float32
}

// clusterChunks runs auto-k k-means over the returned chunks' vectors.
// Too-few chunks → no clusters (same floor as the note query).
func clusterChunks(top []scoredChunk) []chunkCluster {
	if len(top) < chunkClusterMinMembers {
		return nil
	}

	vectors := make([][]float32, 0, len(top))
	for _, s := range top {
		vectors = append(vectors, s.record.Vector)
	}

	result, err := cluster.AutoK(vectors, chunkClusterMinK, chunkClusterMaxK, chunkClusterThreshold, chunkClusterSeed)
	if err != nil || result.K == 0 {
		return nil
	}

	byID := map[int][]int{}
	for memberIdx, clusterID := range result.Assignments {
		byID[clusterID] = append(byID[clusterID], memberIdx)
	}

	clusters := make([]chunkCluster, 0, len(byID))
	for id := range result.K {
		if members := byID[id]; len(members) > 0 {
			clusters = append(clusters, chunkCluster{id: id, members: members})
		}
	}

	return clusters
}

// listJSONLIndexes returns the .jsonl files directly under dir. A missing
// dir is an empty index (cold start), not an error.
func listJSONLIndexes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("listing chunk indexes: %w", err)
	}

	var paths []string

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}

	return paths, nil
}

// loadChunkRecords reads every index file under the chunks dir.
func loadChunkRecords(dir string, deps ChunkQueryDeps) ([]chunk.Record, error) {
	paths, err := deps.ListIndexes(dir)
	if err != nil {
		return nil, fmt.Errorf("query chunks: listing %s: %w", dir, err)
	}

	var records []chunk.Record

	for _, path := range paths {
		data, err := deps.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("query chunks: reading %s: %w", path, err)
		}

		decoded, err := chunk.DecodeRecords(data)
		if err != nil {
			return nil, fmt.Errorf("query chunks: %s: %w", path, err)
		}

		records = append(records, decoded...)
	}

	return records, nil
}

// newOsChunkQueryDeps wires the production filesystem + bundled embedder for
// `engram query-chunks`.
func newOsChunkQueryDeps() ChunkQueryDeps {
	fs := &osEmbedFS{}

	return ChunkQueryDeps{
		ListIndexes: listJSONLIndexes,
		ReadFile:    fs.Read,
		Embedder:    sharedEmbedder,
	}
}

// scoreChunks embeds the phrases and scores each record by its best cosine
// across all phrase vectors.
func scoreChunks(
	ctx context.Context,
	phrases []string,
	records []chunk.Record,
	embedder embed.Embedder,
) ([]scoredChunk, error) {
	vectors := make([][]float32, 0, len(phrases))

	for _, phrase := range phrases {
		vec, err := embedder.Embed(ctx, phrase)
		if err != nil {
			return nil, fmt.Errorf("query chunks: embedding phrase: %w", err)
		}

		vectors = append(vectors, vec)
	}

	scored := make([]scoredChunk, 0, len(records))

	for _, record := range records {
		best := float32(-1)

		for _, vec := range vectors {
			if c := embed.Cosine(vec, record.Vector); c > best {
				best = c
			}
		}

		scored = append(scored, scoredChunk{record: record, score: best})
	}

	return scored, nil
}

func writeChunkClusters(buf *strings.Builder, clusters []chunkCluster, top []scoredChunk) {
	if len(clusters) == 0 {
		return
	}

	buf.WriteString("clusters:\n")

	for _, c := range clusters {
		fmt.Fprintf(buf, "  - id: %d\n    size: %d\n    members:\n", c.id, len(c.members))

		for _, memberIdx := range c.members {
			record := top[memberIdx].record
			fmt.Fprintf(buf, "      - { source: %q, anchor: %q, score: %.4f }\n",
				record.Source, record.Anchor, top[memberIdx].score)
		}
	}
}

func writeChunkItems(buf *strings.Builder, top []scoredChunk) {
	buf.WriteString("items:\n")

	for _, s := range top {
		fmt.Fprintf(buf, "  - source: %q\n    anchor: %q\n    score: %.4f\n    content: |\n",
			s.record.Source, s.record.Anchor, s.score)

		for line := range strings.Lines(s.record.Text) {
			fmt.Fprintf(buf, "      %s\n", strings.TrimRight(line, "\n"))
		}
	}
}

// writeChunkPayload emits the YAML payload. Shape mirrors the note query's
// (items + clusters + budget) so the recall skill variant parses familiarly.
func writeChunkPayload(
	stdout io.Writer,
	phrases []string,
	top []scoredChunk,
	clusters []chunkCluster,
	totalChunks int,
) error {
	var buf strings.Builder

	buf.WriteString("version: 1\n")
	buf.WriteString("space: chunks\n")
	writePhrases(&buf, phrases)
	writeChunkItems(&buf, top)
	writeChunkClusters(&buf, clusters, top)
	fmt.Fprintf(&buf, "budget:\n  total_chunks: %d\n  returned: %d\n", totalChunks, len(top))

	_, err := io.WriteString(stdout, buf.String())
	if err != nil {
		return fmt.Errorf("query chunks: writing payload: %w", err)
	}

	return nil
}

func writePhrases(buf *strings.Builder, phrases []string) {
	buf.WriteString("phrases:\n")

	for _, p := range phrases {
		fmt.Fprintf(buf, "  - %q\n", p)
	}
}
