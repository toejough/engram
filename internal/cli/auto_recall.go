package cli

import (
	"context"
	"io"

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
func RunAutoRecall(_ context.Context, _ AutoRecallArgs, _ AutoRecallDeps, _ io.Writer) error {
	return nil
}
