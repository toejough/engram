package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// AtomicWriter writes a TOML-serializable record to a path atomically.
type AtomicWriter interface {
	AtomicWrite(targetPath string, record any) error
}

// Modifier atomically reads a memory TOML, applies a mutation, and writes back.
// All I/O is injected for testability.
type Modifier struct {
	readFile func(string) ([]byte, error)
	writer   AtomicWriter
}

// NewModifier creates a Modifier with real filesystem operations.
// The caller must provide a writer via WithModifierWriter; if none is given,
// the Modifier will panic on first use. This avoids importing tomlwriter
// from the memory package.
func NewModifier(opts ...ModifierOption) *Modifier {
	m := &Modifier{
		readFile: os.ReadFile,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ReadModifyWrite atomically reads a memory TOML, applies a mutation, and writes back.
func (m *Modifier) ReadModifyWrite(path string, mutate func(*MemoryRecord)) error {
	data, err := m.readFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var record MemoryRecord

	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}

	mutate(&record)

	return m.writer.AtomicWrite(path, record)
}

// ModifierOption configures a Modifier.
type ModifierOption func(*Modifier)

// StoredRecord pairs a file path with its parsed MemoryRecord.
type StoredRecord struct {
	Path   string
	Record MemoryRecord
}

// ListAll reads all .toml files from a directory, returning parsed records.
func ListAll(dir string) ([]StoredRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	records := make([]StoredRecord, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		data, readErr := os.ReadFile(path) //nolint:gosec // trusted dir
		if readErr != nil {
			continue
		}

		var record MemoryRecord

		_, decErr := toml.Decode(string(data), &record)
		if decErr != nil {
			continue
		}

		records = append(records, StoredRecord{Path: path, Record: record})
	}

	return records, nil
}

// WithModifierReadFile overrides the file reading function.
func WithModifierReadFile(fn func(string) ([]byte, error)) ModifierOption {
	return func(m *Modifier) { m.readFile = fn }
}

// WithModifierWriter sets the AtomicWriter for atomic writes.
func WithModifierWriter(w AtomicWriter) ModifierOption {
	return func(m *Modifier) { m.writer = w }
}
