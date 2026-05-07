// Package debuglog provides a tail-friendly debug logger for engram pipelines.
// Init opens an append-mode log file. Subsequent Log calls write atomically
// and sync to disk so `tail -F` shows progress live.
package debuglog

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Singleton logger state. The package is a process-singleton observability
// tool intentionally; do not refactor to DI.
//
//nolint:gochecknoglobals // intentional package-level singleton
var (
	mu        sync.Mutex
	file      *os.File
	component string
)

const (
	filePerm    = 0o644
	truncateMax = 200
	ellipsis    = "..."
)

// Init opens the log file at path in append mode. Subsequent calls replace
// the previous file. If path is empty, logging is disabled (no-op).
// Init is idempotent for the same path.
func Init(path, comp string) error {
	mu.Lock()
	defer mu.Unlock()

	if path == "" {
		file = nil
		component = ""

		return nil
	}

	if file != nil {
		_ = file.Close()
		file = nil
	}

	// Path comes from operator-set env var (ENGRAM_DEBUG_LOG), not user input.
	//nolint:gosec // operator-controlled path
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("opening debug log %s: %w", path, err)
	}

	file = f
	component = comp

	return nil
}

// Log writes a single line: <timestamp> [<component>] <stage>: <message>.
// Each call appends and syncs. Safe for concurrent use within one process.
// No-op when Init was called with empty path.
//
//nolint:goprintffuncname // "Log" reads more naturally than "Logf" at call sites
func Log(stage, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if file == nil {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", now, component, stage, msg)

	_, _ = file.WriteString(line)
	_ = file.Sync()
}

// Timed wraps a stage with .start and .end log entries plus duration.
// Returns a defer-friendly closer:
//
//	defer debuglog.Timed("Cycle.Run", "projectDir=%s", projectDir)()
func Timed(stage, format string, args ...any) func() {
	Log(stage+".start", format, args...)

	start := time.Now()

	return func() {
		Log(stage+".end", "took=%s", time.Since(start))
	}
}

// Truncate returns s capped at truncateMax bytes, with newlines collapsed
// to spaces and trailing "..." if truncated. Suitable for prompt/output previews.
func Truncate(s string, maxLen int) string {
	collapsed := strings.NewReplacer("\n", " ", "\r", " ").Replace(s)

	if len(collapsed) <= maxLen {
		return collapsed
	}

	if maxLen <= len(ellipsis) {
		return collapsed[:maxLen]
	}

	return collapsed[:maxLen-len(ellipsis)] + ellipsis
}
