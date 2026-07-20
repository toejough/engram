package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
)

// ChunkQueryArgs holds parsed flags for `engram query --space chunks`
// (surfaced as the `query-chunks` target during the experiment).
type ChunkQueryArgs struct {
	Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable)"`
	ChunksDir string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
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

	if len(records) == 0 {
		// Empty index: emit the empty payload without waking the embedder.
		return writeChunkPayload(stdout, args.Phrases, []scoredChunk{}, nil, 0)
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
// baseScore is the pre-decay raw cosine; zero in the global scoreChunks path
// (which only computes max-across-phrases and does not apply recency).
// It is populated by scoreChunkForPhrase for the per-phrase recency path.
type scoredChunk struct {
	record    chunk.Record
	score     float32
	baseScore float32 // pre-decay raw cosine; 0 when not set (global path)
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

// listJSONLIndexes returns a lister over fsys for the .jsonl files directly
// under a dir. A missing dir is an empty index (cold start), not an error —
// matched via errors.Is so EdgeFS implementations may wrap the not-exist
// error (os.IsNotExist would not unwrap a %w chain).
func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error) {
	listNames := listEntryNamesMatching(fsys, "list jsonl", func(entry fs.DirEntry) bool {
		return filepath.Ext(entry.Name()) == jsonlExt
	})

	return func(dir string) ([]string, error) {
		names, err := listNames(dir)
		if err != nil {
			return nil, err
		}

		var paths []string

		for _, name := range names {
			paths = append(paths, filepath.Join(dir, name))
		}

		return paths, nil
	}
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

// newChunkQueryDeps wires `engram query-chunks` from the injected CLI
// capabilities — pure composition (#700).
func newChunkQueryDeps(d Deps) ChunkQueryDeps {
	return ChunkQueryDeps{
		ListIndexes: listJSONLIndexes(d.FS),
		ReadFile:    d.FS.ReadFile,
		Embedder:    d.Embed,
	}
}

// scoreChunkForPhrase scores every record against a single pre-embedded phrase
// vector, applying recency bias exactly once. baseScore is the raw cosine
// (pre-decay); score is the recency-biased result used for ranking. The
// caller is responsible for not calling this again on the same phrase
// (no double-apply). Used by the per-phrase unified ranking
// path; the global scoreChunks is used by all other paths.
func scoreChunkForPhrase(
	phraseVec []float32,
	records []chunk.Record,
	now time.Time,
	maxTurnBySrc map[string]int,
	p recencyParams,
) []scoredChunk {
	scored := make([]scoredChunk, 0, len(records))

	for _, record := range records {
		base := embed.Cosine(phraseVec, record.Vector)

		ageDays := 0.0

		if !record.IngestedAt.IsZero() && !now.IsZero() {
			age := now.Sub(record.IngestedAt).Hours() / hoursPerDay
			if age > 0 {
				ageDays = age
			}
		}

		turnFrac := 0.0

		if n, ok := parseTurnN(record.Anchor); ok {
			if maxN := maxTurnBySrc[record.Source]; maxN > 0 {
				turnFrac = float64(n) / float64(maxN)
			}
		}

		biasedScore := base * float32(recencyMultiplier(ageDays, turnFrac, p))

		scored = append(scored, scoredChunk{
			record:    record,
			score:     biasedScore,
			baseScore: base,
		})
	}

	return scored
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

		scored = append(scored, scoredChunk{record: record, score: best, baseScore: best})
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
