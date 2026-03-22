package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// unexported variables.
var (
	errFlushMissingDataDir = errors.New("flush: --data-dir is required")
)

func runFlush(args []string, _ io.Writer, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("flush", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	transcriptPath := fs.String("transcript-path", "", "path to session transcript")
	sessionID := fs.String("session-id", "", "session identifier")
	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("flush: %w", parseErr)
	}

	if *dataDir == "" {
		return errFlushMissingDataDir
	}

	// Clean up surfacing log — evaluate no longer consumes it (#348).
	_ = os.Remove(filepath.Join(*dataDir, "surfacing-log.jsonl"))

	if *transcriptPath == "" || *sessionID == "" {
		return nil
	}

	learnArgs := []string{
		"--transcript-path", *transcriptPath,
		"--session-id", *sessionID,
		"--data-dir", *dataDir,
	}

	return RunLearn(learnArgs, resolveToken(context.Background()), stderr, stdin, nil)
}
