// Package debuglog provides a tail-friendly debug logger for engram pipelines.
// New opens an append-mode log file and returns a *Logger. Log calls write
// atomically and sync to disk so `tail -F` shows progress live.
//
// Loggers are threaded through context (see WithLogger / LoggerFromContext).
// The package-level Log and Timed helpers read the logger from ctx, so call
// sites stay short while production wiring stays explicit.
package debuglog

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger writes structured debug lines to an append-mode file. Methods are
// safe for concurrent use within one process and safe to call on a nil
// receiver (no-op), which means tests can pass a nil logger without panics.
type Logger struct {
	component string
	file      *os.File
	mu        sync.Mutex
}

// New opens path in append mode and returns a *Logger tagged with comp.
// If path is empty, returns a no-op *Logger that ignores all writes.
// Errors only surface for non-empty paths that fail to open.
func New(path, comp string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil
	}

	// Path comes from operator-set env var (ENGRAM_DEBUG_LOG), not user input.
	//nolint:gosec // operator-controlled path
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return nil, fmt.Errorf("opening debug log %s: %w", path, err)
	}

	return &Logger{file: f, component: comp}, nil
}

// Log writes a single line: <timestamp> [<component>] <stage>: <message>.
// Each call appends and syncs. Safe on a nil receiver (no-op) and safe for
// concurrent use.
//
//nolint:goprintffuncname // "Log" reads more naturally than "Logf" at call sites
func (l *Logger) Log(stage, format string, args ...any) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", now, l.component, stage, msg)

	_, _ = l.file.WriteString(line)
	_ = l.file.Sync()
}

// Timed wraps a stage with .start and .end log entries plus duration.
// Returns a defer-friendly closer:
//
//	defer logger.Timed("Cycle.Run", "projectDir=%s", projectDir)()
//
// Safe on a nil receiver.
func (l *Logger) Timed(stage, format string, args ...any) func() {
	l.Log(stage+".start", format, args...)

	start := time.Now()

	return func() {
		l.Log(stage+".end", "took=%s", time.Since(start))
	}
}

// Log reads a *Logger from ctx and writes a line. No-op when ctx carries
// no logger.
//
//nolint:goprintffuncname // mirrors Logger.Log naming
func Log(ctx context.Context, stage, format string, args ...any) {
	LoggerFromContext(ctx).Log(stage, format, args...)
}

// Timed reads a *Logger from ctx and starts a timed entry. No-op closer
// when ctx carries no logger.
func Timed(ctx context.Context, stage, format string, args ...any) func() {
	return LoggerFromContext(ctx).Timed(stage, format, args...)
}

// unexported constants.
const (
	filePerm = 0o644
)
