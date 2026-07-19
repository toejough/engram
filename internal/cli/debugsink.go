package cli

import (
	"fmt"
	"io"
	"io/fs"
)

// unexported constants.
const (
	debugLogEnvVar             = "ENGRAM_DEBUG_LOG"
	debugLogPerm   fs.FileMode = 0o644
)

// syncWriter flushes after every write. debuglog is documented tail -F
// friendly; the Logger sees only an io.Writer, so the per-line sync lives
// here in the composed sink.
type syncWriter struct {
	file WriteSyncer
}

// Write appends p and syncs the underlying sink.
func (w *syncWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("debug log write: %w", err)
	}

	_ = w.file.Sync()

	return n, nil
}

// openDebugSink builds the debug-log sink: nil for an empty path, an
// unwired open capability, or a failed open — debuglog treats a nil writer
// as "logging disabled", so the CLI still runs (pre-#700 behavior
// preserved). Otherwise every write is followed by Sync so `tail -F` shows
// progress live.
func openDebugSink(path string, open func(string, fs.FileMode) (WriteSyncer, error)) io.Writer {
	if path == "" || open == nil {
		return nil
	}

	file, err := open(path, debugLogPerm)
	if err != nil {
		return nil
	}

	return &syncWriter{file: file}
}
