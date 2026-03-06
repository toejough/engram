// Package creationlog records and retrieves memory creation events in a JSONL log file.
package creationlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// LogEntry represents a single memory creation event.
type LogEntry struct {
	Timestamp string `json:"timestamp"` // RFC 3339
	Title     string `json:"title"`
	Tier      string `json:"tier"`     // A/B/C
	Filename  string `json:"filename"` // e.g. "use-targ-test.toml"
}

// LogReader reads and clears the creation log.
type LogReader struct {
	readFile   func(string) ([]byte, error)
	removeFile func(string) error
}

// NewLogReader creates a LogReader with optional DI overrides.
// Defaults use real os.* functions.
func NewLogReader(opts ...ReaderOption) *LogReader {
	r := &LogReader{
		readFile:   os.ReadFile,
		removeFile: os.Remove,
	}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReadAndClear reads all log entries from dataDir and deletes the log file.
// Missing file returns empty slice with no error and does not call removeFile.
// Malformed lines are skipped.
// Non-ErrNotExist read errors are returned without calling removeFile.
func (r *LogReader) ReadAndClear(dataDir string) ([]LogEntry, error) {
	path := dataDir + "/" + logFilename

	data, err := r.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]LogEntry, 0), nil
		}

		return nil, fmt.Errorf("reading creation log: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry LogEntry

		jsonErr := json.Unmarshal([]byte(line), &entry)
		if jsonErr != nil {
			continue // skip malformed lines
		}

		entries = append(entries, entry)
	}

	err = r.removeFile(path)
	if err != nil {
		return nil, fmt.Errorf("removing creation log: %w", err)
	}

	return entries, nil
}

// LogWriter appends creation events to a JSONL log file.
type LogWriter struct {
	readFile  func(string) ([]byte, error)
	writeFile func(string, []byte, os.FileMode) error
	now       func() time.Time
}

// NewLogWriter creates a LogWriter with optional DI overrides.
// Defaults use real os.* functions and time.Now.
func NewLogWriter(opts ...WriterOption) *LogWriter {
	w := &LogWriter{
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		now:       time.Now,
	}
	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Append adds a LogEntry to the creation log in dataDir.
// If entry.Timestamp is empty, it is set from the injected clock.
// Missing log file is not an error.
func (w *LogWriter) Append(entry LogEntry, dataDir string) error {
	if entry.Timestamp == "" {
		entry.Timestamp = w.now().UTC().Format(time.RFC3339)
	}

	path := dataDir + "/" + logFilename

	existing, err := w.readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading creation log: %w", err)
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	var sb strings.Builder
	if len(existing) > 0 {
		sb.Write(existing)

		if existing[len(existing)-1] != '\n' {
			sb.WriteByte('\n')
		}
	}

	sb.Write(line)
	sb.WriteByte('\n')

	err = w.writeFile(path, []byte(sb.String()), logFileMode)
	if err != nil {
		return fmt.Errorf("writing creation log: %w", err)
	}

	return nil
}

// ReaderOption configures a LogReader.
type ReaderOption func(*LogReader)

// WriterOption configures a LogWriter.
type WriterOption func(*LogWriter)

// WithNow injects a clock function into a LogWriter.
func WithNow(fn func() time.Time) WriterOption {
	return func(w *LogWriter) {
		w.now = fn
	}
}

// WithReadFile injects a readFile function into a LogWriter.
func WithReadFile(fn func(string) ([]byte, error)) WriterOption {
	return func(w *LogWriter) {
		w.readFile = fn
	}
}

// WithReaderReadFile injects a readFile function into a LogReader.
func WithReaderReadFile(fn func(string) ([]byte, error)) ReaderOption {
	return func(r *LogReader) {
		r.readFile = fn
	}
}

// WithRemoveFile injects a removeFile function into a LogReader.
func WithRemoveFile(fn func(string) error) ReaderOption {
	return func(r *LogReader) {
		r.removeFile = fn
	}
}

// WithWriteFile injects a writeFile function into a LogWriter.
func WithWriteFile(fn func(string, []byte, os.FileMode) error) WriterOption {
	return func(w *LogWriter) {
		w.writeFile = fn
	}
}

// unexported constants.
const (
	logFileMode = os.FileMode(0o644)
	logFilename = "creation-log.jsonl"
)
