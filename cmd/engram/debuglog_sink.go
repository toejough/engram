package main

import (
	"fmt"
	"io"
	"os"
)

// unexported constants.
const (
	debugLogPerm = 0o644
)

// syncWriter wraps an *os.File so every write is flushed to disk. debuglog
// is documented tail -F friendly; the Logger now sees only an io.Writer, so
// the per-line sync lives here at the edge.
type syncWriter struct {
	file *os.File
}

// Write appends p and syncs to disk.
func (w *syncWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("debug log write: %w", err)
	}

	_ = w.file.Sync()

	return n, nil
}

// openDebugSink opens path in append mode as the debug-log sink. An empty
// path or an open failure yields nil — debuglog.New treats a nil writer as
// "logging disabled", so the CLI still runs (matches the pre-#700 behavior
// where a failed open fell back to a no-op logger).
func openDebugSink(path string) io.Writer {
	if path == "" {
		return nil
	}

	// Path comes from operator-set env var (ENGRAM_DEBUG_LOG), not user input.
	//nolint:gosec // operator-controlled path
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, debugLogPerm)
	if err != nil {
		return nil
	}

	return &syncWriter{file: f}
}
