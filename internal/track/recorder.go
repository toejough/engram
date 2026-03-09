package track

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// Recorder writes surfacing tracking updates to memory TOML files.
type Recorder struct {
	readFile   func(string) ([]byte, error)
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
	now        func() time.Time
}

// NewRecorder creates a Recorder with default I/O wired to the real filesystem.
func NewRecorder(opts ...RecorderOption) *Recorder {
	r := &Recorder{
		readFile:   os.ReadFile,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
		now:        time.Now,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// RecordSurfacing updates tracking fields in memory TOML files.
// It processes all memories, collecting errors per-memory without stopping.
func (r *Recorder) RecordSurfacing(
	_ context.Context,
	memories []*memory.Stored,
	mode string,
) error {
	now := r.now()

	var errs []error

	for _, mem := range memories {
		if mem.FilePath == "" {
			continue
		}

		err := r.updateMemoryFile(mem, mode, now)
		if err != nil {
			errs = append(errs, fmt.Errorf("recording %s: %w", mem.FilePath, err))
		}
	}

	return errors.Join(errs...)
}

func (r *Recorder) updateMemoryFile(mem *memory.Stored, _ string, _ time.Time) error {
	// Tracking fields are now managed by the instruction registry (UC-23).
	// The Recorder reads and re-writes the TOML to strip old tracking fields
	// (surfaced_count, last_surfaced, surfacing_contexts) from disk.
	data, err := r.readFile(mem.FilePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var record tomlRecord

	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return fmt.Errorf("decoding TOML: %w", err)
	}

	return r.writeAtomic(mem.FilePath, &record)
}

func (r *Recorder) writeAtomic(targetPath string, record *tomlRecord) error {
	dir := filepath.Dir(targetPath)

	tmpFile, err := r.createTemp(dir, "engram-track-*.toml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	encoder := toml.NewEncoder(tmpFile)

	encodeErr := encoder.Encode(record)
	if encodeErr != nil {
		_ = tmpFile.Close()
		_ = r.remove(tmpPath)

		return fmt.Errorf("encoding TOML: %w", encodeErr)
	}

	closeErr := tmpFile.Close()
	if closeErr != nil {
		_ = r.remove(tmpPath)

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	renameErr := r.rename(tmpPath, targetPath)
	if renameErr != nil {
		_ = r.remove(tmpPath)

		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}

// RecorderOption configures a Recorder.
type RecorderOption func(*Recorder)

// WithCreateTemp sets the temp file creation function.
func WithCreateTemp(fn func(dir, pattern string) (*os.File, error)) RecorderOption {
	return func(r *Recorder) { r.createTemp = fn }
}

// WithNow sets the time provider function.
func WithNow(fn func() time.Time) RecorderOption {
	return func(r *Recorder) { r.now = fn }
}

// WithReadFile sets the file reading function.
func WithReadFile(fn func(string) ([]byte, error)) RecorderOption {
	return func(r *Recorder) { r.readFile = fn }
}

// WithRemove sets the file remove function.
func WithRemove(fn func(name string) error) RecorderOption {
	return func(r *Recorder) { r.remove = fn }
}

// WithRename sets the file rename function.
func WithRename(fn func(oldpath, newpath string) error) RecorderOption {
	return func(r *Recorder) { r.rename = fn }
}

// tomlRecord mirrors content TOML fields to preserve round-trip fidelity.
// Tracking fields (surfaced_count, last_surfaced, surfacing_contexts) are no longer
// included — they are managed by the instruction registry (UC-23). Re-writing a TOML
// through this struct strips any old tracking fields from disk.
type tomlRecord struct {
	Title           string   `toml:"title"`
	Content         string   `toml:"content"`
	ObservationType string   `toml:"observation_type"`
	Concepts        []string `toml:"concepts"`
	Keywords        []string `toml:"keywords"`
	Principle       string   `toml:"principle"`
	AntiPattern     string   `toml:"anti_pattern"`
	Rationale       string   `toml:"rationale"`
	Confidence      string   `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`
}
