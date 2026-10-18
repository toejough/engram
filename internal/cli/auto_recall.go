package cli

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

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
	// ReadChunks reads the records of the index file of a source in a directory of chunk indexes.
	ReadChunks func(dir, source string) ([]chunk.Record, error)
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
		LazyChunks: true, RecentFill: -1, Limit: auto.MaxCandidates,
	}, &payload)
	if queryErr != nil {
		// a failure is forwarded by the mod and stored: it names none of the prompt (793b107)
		return fmt.Errorf("auto recall: lookup: %w", queryErr)
	}

	matched, candidatesErr := candidatesOf(payload.Bytes(), input.Session, args.ChunksDir, deps.ReadChunks)
	if candidatesErr != nil {
		return fmt.Errorf("auto recall: %w", candidatesErr)
	}

	step, found := auto.Recall(input, matched)
	if !found {
		return nil
	}

	return printStep(stdout, "auto recall", step)
}

// candidatesOf reads the notes and the passages that matched out of an `engram query` payload, in the order of the
// payload: not a note that only the explore half picked, and not a chunk that only the recency channel gave (855d4b).
// Nor is a chunk of the session's own transcript. The text of a chunk is read from the index file of its source, as
// `engram show-chunk` does, once for each source that matched.
func candidatesOf(
	payload []byte, session, chunksDir string, readChunks func(dir, source string) ([]chunk.Record, error),
) ([]auto.Candidate, error) {
	var parsed queryPayload

	parseErr := yaml.Unmarshal(payload, &parsed)
	if parseErr != nil {
		return nil, fmt.Errorf("reading the payload: %w", parseErr)
	}

	// the candidates are in the order of their scores, and the payload lists the notes before the chunks (855b59)
	slices.SortStableFunc(parsed.Items, func(left, right queryItem) int { return cmp.Compare(right.Score, left.Score) })

	bySource := make(map[string]map[string]chunk.Record)
	candidates := make([]auto.Candidate, 0, len(parsed.Items))

	for _, item := range parsed.Items {
		if slices.Equal(item.Provenances, []string{provenanceExplore}) ||
			slices.Equal(item.Provenances, []string{provenanceRecent}) {
			continue
		}

		if item.Kind != chunkItemKind {
			frontmatter, body, _ := splitFrontmatterAndBody(item.Content)
			fields := noteFieldsOf([]byte(frontmatter))

			candidates = append(candidates, auto.Candidate{
				ID: item.Path, Kind: auto.KindNote, Type: item.Kind, Date: fields.Created, Situation: fields.Situation,
				Text:   strings.TrimSpace(body),
				Parent: item.FromParent != nil && *item.FromParent, Pending: fields.Pending,
			})

			continue
		}

		source := item.Path[:max(strings.LastIndex(item.Path, "#"), 0)]
		if filepath.Base(source) == session+".jsonl" {
			continue
		}

		records, readErr := chunkRecordsOf(bySource, chunksDir, source, readChunks)
		if readErr != nil {
			return nil, fmt.Errorf("reading the chunks: %w", readErr)
		}

		record := records[item.Path]
		candidates = append(candidates, auto.Candidate{
			ID: item.Path, Kind: auto.KindPassage, Date: dayOf(record.IngestedAt), Text: record.Text,
		})
	}

	return candidates, nil
}

// chunkRecordsOf reads the records of the index file of a source by their IDs, once.
func chunkRecordsOf(
	bySource map[string]map[string]chunk.Record, dir, source string,
	read func(dir, source string) ([]chunk.Record, error),
) (map[string]chunk.Record, error) {
	if records, found := bySource[source]; found {
		return records, nil
	}

	records, err := read(dir, source)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]chunk.Record, len(records))

	for _, record := range records {
		byID[chunkNotePath(record)] = record
	}

	bySource[source] = byID

	return byID, nil
}

// dayOf is the day a chunk was indexed, or nothing for a chunk indexed before the index kept the time.
func dayOf(indexed time.Time) string {
	if indexed.IsZero() {
		return ""
	}

	return indexed.UTC().Format(noteDateFormat)
}

// readIndexFile reads the records of one index file of the chunk index.
func readIndexFile(path string, readFile func(string) ([]byte, error)) ([]chunk.Record, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	records, decodeErr := chunk.DecodeRecords(data)
	if decodeErr != nil {
		return nil, fmt.Errorf("%s: %w", path, decodeErr)
	}

	return records, nil
}

// newAutoRecallDeps wires RunAutoRecall from the injected CLI capabilities - pure composition (#700). The lookup is the
// one of the delivery, whose warnings go to the debug log.
func newAutoRecallDeps(d Deps) AutoRecallDeps {
	return AutoRecallDeps{
		Stdin: d.Stdin,
		Query: newAutoDeliverDeps(d).Query,
		ReadChunks: func(dir, source string) ([]chunk.Record, error) {
			return readIndexFile(indexPathFor(dir, source), d.FS.ReadFile)
		},
	}
}
