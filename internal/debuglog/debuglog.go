// Package debuglog provides a tail-friendly debug logger for engram
// pipelines. New wraps an injected io.Writer sink and returns a *Logger.
// Log calls write one line at a time under a mutex; the production sink
// (internal/cli's composed debug sink over the cmd-injected open primitive)
// syncs to disk after every write so `tail -F` shows progress live. The
// package itself performs no I/O and reads no clock — the sink and the now
// func are injected at the edge (#700).
//
// Loggers are threaded through context (see WithLogger / LoggerFromContext).
// The package-level Log and Timed helpers read the logger from ctx, so call
// sites stay short while production wiring stays explicit.
package debuglog

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Logger writes structured debug lines to an injected sink. Methods are
// safe for concurrent use within one process and safe to call on a nil
// receiver (no-op), which means tests can pass a nil logger without panics.
type Logger struct {
	component string
	out       io.Writer
	now       func() time.Time
	mu        sync.Mutex
}

// New returns a *Logger tagged with prefix that writes to w, stamping each
// line via now. A nil w returns a nil *Logger — every method is a
// nil-receiver-safe no-op, preserving the "unset ENGRAM_DEBUG_LOG disables
// logging" behavior. now must be non-nil when w is non-nil.
func New(w io.Writer, prefix string, now func() time.Time) *Logger {
	if w == nil {
		return nil
	}

	return &Logger{component: prefix, out: w, now: now}
}

// Log writes a single line: <timestamp> [<component>] <stage>: <message>.
// Safe on a nil receiver (no-op) and safe for concurrent use.
//
//nolint:goprintffuncname // "Log" reads more naturally than "Logf" at call sites
func (l *Logger) Log(stage, format string, args ...any) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := l.now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", timestamp, l.component, stage, msg)

	_, _ = io.WriteString(l.out, line)
}

// Timed wraps a stage with .start and .end log entries plus duration.
// Returns a defer-friendly closer:
//
//	defer logger.Timed("Cycle.Run", "projectDir=%s", projectDir)()
//
// Safe on a nil receiver.
func (l *Logger) Timed(stage, format string, args ...any) func() {
	if l == nil {
		return func() {}
	}

	l.Log(stage+".start", format, args...)

	start := l.now()

	return func() {
		l.Log(stage+".end", "took=%s", l.now().Sub(start))
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
