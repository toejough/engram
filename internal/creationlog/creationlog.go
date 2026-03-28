// Package creationlog records and retrieves memory creation events in a JSONL log file.
package creationlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"engram/internal/jsonlutil"
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
	path := filepath.Join(dataDir, logFilename)

	data, err := r.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]LogEntry, 0), nil
		}

		return nil, fmt.Errorf("reading creation log: %w", err)
	}

	entries := jsonlutil.ParseLines[LogEntry](data)

	err = r.removeFile(path)
	if err != nil {
		return nil, fmt.Errorf("removing creation log: %w", err)
	}

	return entries, nil
}

// LogWriter appends creation events to a JSONL log file.
type LogWriter struct {
	readFile   func(string) ([]byte, error)
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
	now        func() time.Time
}

// NewLogWriter creates a LogWriter with optional DI overrides.
// Defaults use real os.* functions and time.Now.
func NewLogWriter(opts ...WriterOption) *LogWriter {
	w := &LogWriter{
		readFile:   os.ReadFile,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Append adds a LogEntry to the creation log in dataDir.
// If entry.Timestamp is empty, it is set from the injected clock.
// Missing log file is not an error. Write is atomic via temp+rename.
func (w *LogWriter) Append(entry LogEntry, dataDir string) error {
	if entry.Timestamp == "" {
		entry.Timestamp = w.now().UTC().Format(time.RFC3339)
	}

	path := filepath.Join(dataDir, logFilename)

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

	return w.writeAtomic(path, sb.String())
}

func (w *LogWriter) writeAtomic(targetPath, content string) error {
	tmpFile, err := w.createTemp(filepath.Dir(targetPath), "engram-creation-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	_, writeErr := tmpFile.WriteString(content)
	if writeErr != nil {
		_ = tmpFile.Close()
		_ = w.remove(tmpPath)

		return fmt.Errorf("writing creation log: %w", writeErr)
	}

	closeErr := tmpFile.Close()
	if closeErr != nil {
		_ = w.remove(tmpPath)

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	renameErr := w.rename(tmpPath, targetPath)
	if renameErr != nil {
		_ = w.remove(tmpPath)

		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}

// ReaderOption configures a LogReader.
type ReaderOption func(*LogReader)

// WriterOption configures a LogWriter.
type WriterOption func(*LogWriter)

// WithCreateTemp injects a temp file creation function into a LogWriter.
func WithCreateTemp(fn func(dir, pattern string) (*os.File, error)) WriterOption {
	return func(w *LogWriter) {
		w.createTemp = fn
	}
}

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

// WithRemove injects a remove function into a LogWriter.
func WithRemove(fn func(name string) error) WriterOption {
	return func(w *LogWriter) {
		w.remove = fn
	}
}

// WithRemoveFile injects a removeFile function into a LogReader.
func WithRemoveFile(fn func(string) error) ReaderOption {
	return func(r *LogReader) {
		r.removeFile = fn
	}
}

// WithRename injects a rename function into a LogWriter.
func WithRename(fn func(oldpath, newpath string) error) WriterOption {
	return func(w *LogWriter) {
		w.rename = fn
	}
}

// unexported constants.
const (
	logFilename = "creation-log.jsonl"
)
