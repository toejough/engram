package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// RecallSurfacer adapts the surface pipeline to the recall.MemorySurfacer interface.
type RecallSurfacer struct {
	runner  SurfaceRunner
	dataDir string
}

// NewRecallSurfacer creates a RecallSurfacer.
func NewRecallSurfacer(runner SurfaceRunner, dataDir string) *RecallSurfacer {
	return &RecallSurfacer{runner: runner, dataDir: dataDir}
}

// Surface finds relevant memories for the given query.
func (s *RecallSurfacer) Surface(query string) (string, error) {
	if query == "" {
		return "", nil
	}

	var buf strings.Builder

	err := s.runner.Run(context.Background(), &buf, SurfaceRunnerOptions{
		Mode:    "prompt",
		DataDir: s.dataDir,
		Message: query,
	})
	if err != nil {
		return "", fmt.Errorf("recall surfacer: %w", err)
	}

	return buf.String(), nil
}

// SurfaceRunner runs the memory surface pipeline.
type SurfaceRunner interface {
	Run(ctx context.Context, w io.Writer, opts SurfaceRunnerOptions) error
}

// SurfaceRunnerOptions holds the options for a surface run.
type SurfaceRunnerOptions struct {
	Mode    string
	DataDir string
	Message string
}
