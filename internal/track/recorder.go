package track

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// Recorder writes surfacing tracking updates to memory TOML files.
type Recorder struct {
	readFile func(string) ([]byte, error)
	writer   *tomlwriter.Writer
	now      func() time.Time
}

// NewRecorder creates a Recorder with default I/O wired to the real filesystem.
func NewRecorder(opts ...RecorderOption) *Recorder {
	r := &Recorder{
		readFile: os.ReadFile,
		writer:   tomlwriter.New(),
		now:      time.Now,
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
	data, err := r.readFile(mem.FilePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var record memory.MemoryRecord

	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return fmt.Errorf("decoding TOML: %w", err)
	}

	return r.writer.AtomicWrite(mem.FilePath, &record)
}

// RecorderOption configures a Recorder.
type RecorderOption func(*Recorder)

// WithNow sets the time provider function.
func WithNow(fn func() time.Time) RecorderOption {
	return func(r *Recorder) { r.now = fn }
}

// WithReadFile sets the file reading function.
func WithReadFile(fn func(string) ([]byte, error)) RecorderOption {
	return func(r *Recorder) { r.readFile = fn }
}

// WithWriter sets the tomlwriter.Writer for atomic writes.
func WithWriter(w *tomlwriter.Writer) RecorderOption {
	return func(r *Recorder) { r.writer = w }
}
