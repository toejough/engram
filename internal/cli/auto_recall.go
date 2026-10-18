package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/chunk"
)

// AutoRecallArgs holds parsed flags for `engram auto recall`.
type AutoRecallArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// AutoRecallDeps holds injected dependencies for RunAutoRecall.
type AutoRecallDeps struct {
	// Stdin carries the reply of the phrases judgement and its carry: one JSON object.
	Stdin io.Reader
	// Query runs the lookup of `engram query` and prints its payload.
	Query QueryRun
	// ReadChunks reads the records of the chunk index in a directory.
	ReadChunks func(dir string) ([]chunk.Record, error)
}

// RunAutoRecall reads the reply of the phrases judgement from standard input, looks the phrases up and prints the
// request for the model that judges the candidates.
func RunAutoRecall(ctx context.Context, args AutoRecallArgs, deps AutoRecallDeps, stdout io.Writer) error {
	var input auto.RecallInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto recall: reading the reply: %w", decodeErr)
	}

	phrases, phrasesErr := auto.PhrasesOf(input.Reply, input.Carry.Depth)
	if phrasesErr != nil {
		return fmt.Errorf("auto recall: %w", phrasesErr)
	}

	var payload bytes.Buffer

	queryErr := deps.Query(ctx, QueryArgs{
		Phrases: phrases, VaultPath: args.VaultPath, ChunksDir: args.ChunksDir,
		LazyChunks: true, RecentFill: -1, Limit: maxCandidates,
	}, &payload)
	if queryErr != nil {
		// a failure is forwarded by the mod and stored: it names none of the prompt (793b107)
		return fmt.Errorf("auto recall: lookup: %w", queryErr)
	}

	matched, candidatesErr := candidatesOf(payload.Bytes(), input.Session, args.ChunksDir, deps.ReadChunks)
	if candidatesErr != nil {
		return fmt.Errorf("auto recall: %w", candidatesErr)
	}

	if len(matched) == 0 {
		return nil
	}

	return printStep(stdout, "auto recall", auto.Recall(input, matched))
}

// candidatesOf reads the notes and the passages that matched out of an `engram query` payload, in the order of the
// payload: not a note that only the explore half picked, and not a chunk that only the recency channel gave (855d4b).
// Nor is a chunk of the session's own transcript. The text of a chunk is read from the chunk index, as `engram
// show-chunk` does.
func candidatesOf(
	payload []byte, session, chunksDir string, readChunks func(dir string) ([]chunk.Record, error),
) ([]auto.Candidate, error) {
	var parsed queryPayload

	parseErr := yaml.Unmarshal(payload, &parsed)
	if parseErr != nil {
		return nil, fmt.Errorf("reading the payload: %w", parseErr)
	}

	var records map[string]chunk.Record

	candidates := make([]auto.Candidate, 0, len(parsed.Items))

	for _, item := range parsed.Items {
		if slices.Equal(item.Provenances, []string{provenanceExplore}) ||
			slices.Equal(item.Provenances, []string{provenanceRecent}) {
			continue
		}

		if item.Kind != chunkItemKind {
			frontmatter, body, _ := splitFrontmatterAndBody(item.Content)
			situation, _ := situationAndSourceOf([]byte(frontmatter))

			candidates = append(candidates, auto.Candidate{
				ID: item.Path, Kind: auto.KindNote, Situation: situation, Text: body,
			})

			continue
		}

		if records == nil {
			var readErr error

			records, readErr = chunkRecordsByID(chunksDir, readChunks)
			if readErr != nil {
				return nil, fmt.Errorf("reading the chunks: %w", readErr)
			}
		}

		record := records[item.Path]
		if filepath.Base(record.Source) == session+".jsonl" {
			continue
		}

		candidates = append(candidates, auto.Candidate{ID: item.Path, Kind: auto.KindPassage, Text: record.Text})
	}

	return candidates, nil
}

// chunkRecordsByID reads the records of the chunk index by their IDs.
func chunkRecordsByID(dir string, read func(dir string) ([]chunk.Record, error)) (map[string]chunk.Record, error) {
	records, err := read(dir)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]chunk.Record, len(records))

	for _, record := range records {
		byID[chunkNotePath(record)] = record
	}

	return byID, nil
}

// unexported constants.
const (
	// maxCandidates is how many items the lookup is asked for: the most candidates the judgement holds (855d8b).
	maxCandidates = 30
)
