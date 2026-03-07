package learn

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// DeltaReader reads new content from a file since a byte offset.
type DeltaReader interface {
	Read(path string, offset int64) ([]string, int64, error)
}

// IncrementalLearner wraps a Learner to support incremental transcript extraction.
type IncrementalLearner struct {
	learner     *Learner
	delta       DeltaReader
	strip       StripFunc
	offsetStore OffsetStore
	stderr      io.Writer
}

// NewIncrementalLearner creates an IncrementalLearner with all dependencies injected.
func NewIncrementalLearner(
	learner *Learner,
	delta DeltaReader,
	strip StripFunc,
	offsetStore OffsetStore,
	stderr io.Writer,
) *IncrementalLearner {
	return &IncrementalLearner{
		learner:     learner,
		delta:       delta,
		strip:       strip,
		offsetStore: offsetStore,
		stderr:      stderr,
	}
}

// RunIncremental reads new transcript content since the last offset and extracts learnings.
// Errors are logged to stderr and return nil (fire-and-forget semantics).
func (il *IncrementalLearner) RunIncremental(
	ctx context.Context,
	transcriptPath, sessionID, offsetPath string,
) (*Result, error) {
	stored, readErr := il.offsetStore.Read(offsetPath)
	if readErr != nil {
		// Treat as fresh start.
		stored = Offset{}
	}

	offset := stored.Offset
	if sessionID != stored.SessionID {
		offset = 0
	}

	lines, newOffset, deltaErr := il.delta.Read(transcriptPath, offset)
	if deltaErr != nil {
		_, _ = fmt.Fprintf(il.stderr, "[engram] incremental learn: delta read: %v\n", deltaErr)

		return nil, nil //nolint:nilnil // fire-and-forget
	}

	if len(lines) == 0 {
		return &Result{}, nil
	}

	stripped := il.strip(lines)

	if len(stripped) == 0 {
		il.writeOffset(offsetPath, newOffset, sessionID)

		return &Result{}, nil
	}

	text := strings.Join(stripped, "\n")

	result, runErr := il.learner.Run(ctx, text)
	if runErr != nil {
		_, _ = fmt.Fprintf(il.stderr, "[engram] incremental learn: %v\n", runErr)

		return nil, nil //nolint:nilnil // fire-and-forget
	}

	il.writeOffset(offsetPath, newOffset, sessionID)

	return result, nil
}

// writeOffset persists the new offset, logging errors to stderr.
func (il *IncrementalLearner) writeOffset(
	offsetPath string, newOffset int64, sessionID string,
) {
	writeErr := il.offsetStore.Write(offsetPath, Offset{
		Offset:    newOffset,
		SessionID: sessionID,
	})
	if writeErr != nil {
		_, _ = fmt.Fprintf(
			il.stderr, "[engram] incremental learn: write offset: %v\n", writeErr,
		)
	}
}

// StripFunc filters and cleans transcript lines.
type StripFunc func(lines []string) []string
