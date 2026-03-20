package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// FlushRunner executes the end-of-turn memory management pipeline:
// learn → context-update (#309, #348).
type FlushRunner struct {
	learn         func() error
	contextUpdate func() error
}

// NewFlushRunner creates a FlushRunner with the given step functions.
func NewFlushRunner(learn, contextUpdate func() error) *FlushRunner {
	return &FlushRunner{
		learn:         learn,
		contextUpdate: contextUpdate,
	}
}

// Run executes the flush pipeline in order. Stops on first error.
func (f *FlushRunner) Run() error {
	learnErr := f.learn()
	if learnErr != nil {
		return fmt.Errorf("flush: learn: %w", learnErr)
	}

	ctxErr := f.contextUpdate()
	if ctxErr != nil {
		return fmt.Errorf("flush: context-update: %w", ctxErr)
	}

	return nil
}

// unexported variables.
var (
	errFlushMissingDataDir = errors.New("flush: --data-dir is required")
)

//nolint:funlen // CLI wiring function
func runFlush(args []string, _ io.Writer, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("flush", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	transcriptPath := fs.String("transcript-path", "", "path to session transcript")
	sessionID := fs.String("session-id", "", "session identifier")
	dataDir := fs.String("data-dir", "", "path to data directory")
	contextPath := fs.String("context-path", "", "path to session-context.md")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("flush: %w", parseErr)
	}

	if *dataDir == "" {
		return errFlushMissingDataDir
	}

	token := os.Getenv("ENGRAM_API_TOKEN")

	runner := NewFlushRunner(
		func() error {
			if *transcriptPath == "" || *sessionID == "" {
				return nil
			}

			learnArgs := []string{
				"--transcript-path", *transcriptPath,
				"--session-id", *sessionID,
				"--data-dir", *dataDir,
			}

			return RunLearn(learnArgs, token, stderr, stdin, nil)
		},
		func() error {
			if *transcriptPath == "" || *sessionID == "" {
				return nil
			}

			ctxArgs := []string{
				"--transcript-path", *transcriptPath,
				"--session-id", *sessionID,
				"--data-dir", *dataDir,
			}
			if *contextPath != "" {
				ctxArgs = append(ctxArgs, "--context-path", *contextPath)
			}

			return runContextUpdate(ctxArgs)
		},
	)

	return runner.Run()
}
