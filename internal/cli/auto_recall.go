package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
func RunAutoRecall(ctx context.Context, args AutoRecallArgs, deps AutoRecallDeps, _ io.Writer) error {
	var input auto.RecallInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto recall: reading the reply: %w", decodeErr)
	}

	var payload bytes.Buffer

	queryErr := deps.Query(ctx, QueryArgs{
		Phrases: auto.PhrasesOf(input.Reply, input.Carry.Depth), VaultPath: args.VaultPath, ChunksDir: args.ChunksDir,
		LazyChunks: true, RecentFill: -1, Limit: maxCandidates,
	}, &payload)
	if queryErr != nil {
		// a failure is forwarded by the mod and stored: it names none of the prompt (793b107)
		return fmt.Errorf("auto recall: lookup: %w", queryErr)
	}

	return nil
}

// unexported constants.
const (
	// maxCandidates is how many items the lookup is asked for: the most candidates the judgement holds (855d8b).
	maxCandidates = 30
)
