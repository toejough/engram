// Package surfacinglog records and retrieves memory surfacing events in a JSONL log file.
package surfacinglog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const logFilename = "surfacing-log.jsonl"

// SurfacingEvent represents a single memory surfacing event.
type SurfacingEvent struct {
	MemoryPath string    `json:"memory_path"`
	Mode       string    `json:"mode"`
	SurfacedAt time.Time `json:"surfaced_at"`
}

// Option configures a Logger.
type Option func(*Logger)

// Logger records and retrieves surfacing events.
type Logger struct {
	dataDir    string
	appendFile func(name string, data []byte, perm os.FileMode) error
	readFile   func(name string) ([]byte, error)
	removeFile func(name string) error
}

// New creates a Logger for the given data directory with optional DI overrides.
// Defaults use real os.* functions.
func New(dataDir string, opts ...Option) *Logger {
	logger := &Logger{
		dataDir:    dataDir,
		appendFile: appendFileDefault,
		readFile:   os.ReadFile,
		removeFile: os.Remove,
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

// LogSurfacing appends a surfacing event for the given memory path, mode, and timestamp.
func (l *Logger) LogSurfacing(memoryPath, mode string, timestamp time.Time) error {
	event := SurfacingEvent{
		MemoryPath: memoryPath,
		Mode:       mode,
		SurfacedAt: timestamp.UTC(),
	}

	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling surfacing event: %w", err)
	}

	line = append(line, '\n')

	path := l.dataDir + "/" + logFilename

	if appendErr := l.appendFile(path, line, 0o644); appendErr != nil {
		return fmt.Errorf("appending surfacing log: %w", appendErr)
	}

	return nil
}

// ReadAndClear reads all surfacing events and removes the log file.
// Missing file returns empty slice with no error and does not call removeFile.
// Malformed lines are skipped.
func (l *Logger) ReadAndClear() ([]SurfacingEvent, error) {
	path := l.dataDir + "/" + logFilename

	data, err := l.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]SurfacingEvent, 0), nil
		}

		return nil, fmt.Errorf("reading surfacing log: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	events := make([]SurfacingEvent, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event SurfacingEvent
		if jsonErr := json.Unmarshal([]byte(line), &event); jsonErr != nil {
			continue // skip malformed lines
		}

		events = append(events, event)
	}

	if removeErr := l.removeFile(path); removeErr != nil {
		return nil, fmt.Errorf("removing surfacing log: %w", removeErr)
	}

	return events, nil
}

// WithAppendFile injects a file append function into a Logger.
func WithAppendFile(fn func(name string, data []byte, perm os.FileMode) error) Option {
	return func(l *Logger) { l.appendFile = fn }
}

// WithReadFile injects a read function into a Logger.
func WithReadFile(fn func(name string) ([]byte, error)) Option {
	return func(l *Logger) { l.readFile = fn }
}

// WithRemoveFile injects a remove function into a Logger.
func WithRemoveFile(fn func(name string) error) Option {
	return func(l *Logger) { l.removeFile = fn }
}

// appendFileDefault appends data to the named file, creating it if it doesn't exist.
func appendFileDefault(name string, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(data)
	closeErr := file.Close()

	if writeErr != nil {
		return writeErr
	}

	return closeErr
}
