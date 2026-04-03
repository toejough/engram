package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// AtomicWriter writes a TOML-serializable record to a path atomically.
type AtomicWriter interface {
	AtomicWrite(targetPath string, record any) error
}

// Lister reads memory TOML files from a directory.
type Lister struct {
	readDir  func(string) ([]os.DirEntry, error)
	readFile func(string) ([]byte, error)
}

// NewLister creates a Lister with default I/O wired to the real filesystem.
func NewLister(opts ...ListerOption) *Lister {
	lister := &Lister{
		readDir:  os.ReadDir,
		readFile: os.ReadFile,
	}

	for _, opt := range opts {
		opt(lister)
	}

	return lister
}

// ListAll reads all .toml files from a directory, returning parsed records.
func (l *Lister) ListAll(dir string) ([]StoredRecord, error) {
	entries, err := l.readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	records := make([]StoredRecord, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		data, readErr := l.readFile(path)
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

// ListAllMemories reads memories from the new layout (memory/feedback/ + memory/facts/)
// when available, falling back to the legacy memories/ directory.
// It returns all memories sorted by UpdatedAt descending.
func (l *Lister) ListAllMemories(dataDir string) ([]*Stored, error) {
	if l.hasNewLayout(dataDir) {
		return l.listFromNewLayout(dataDir)
	}

	return l.ListStored(MemoriesDir(dataDir))
}

// ListStored reads all .toml files from a directory, converts them to Stored,
// and returns them sorted by UpdatedAt descending.
func (l *Lister) ListStored(dir string) ([]*Stored, error) {
	records, err := l.ListAll(dir)
	if err != nil {
		return nil, err
	}

	stored := make([]*Stored, 0, len(records))

	for idx := range records {
		stored = append(stored, records[idx].Record.ToStored(records[idx].Path))
	}

	sort.Slice(stored, func(i, j int) bool {
		return stored[i].UpdatedAt.After(stored[j].UpdatedAt)
	})

	return stored, nil
}

// hasNewLayout returns true if the feedback directory exists and is non-empty.
func (l *Lister) hasNewLayout(dataDir string) bool {
	entries, err := l.readDir(FeedbackDir(dataDir))
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			return true
		}
	}

	return false
}

// listFromNewLayout reads memories from both feedback/ and facts/ directories,
// merges them, and returns sorted by UpdatedAt descending.
func (l *Lister) listFromNewLayout(dataDir string) ([]*Stored, error) {
	feedbackMemories, feedbackErr := l.ListStored(FeedbackDir(dataDir))
	if feedbackErr != nil && !isNotExist(feedbackErr) {
		return nil, fmt.Errorf("listing feedback: %w", feedbackErr)
	}

	factsMemories, factsErr := l.ListStored(FactsDir(dataDir))
	if factsErr != nil && !isNotExist(factsErr) {
		return nil, fmt.Errorf("listing facts: %w", factsErr)
	}

	combined := make([]*Stored, 0, len(feedbackMemories)+len(factsMemories))
	combined = append(combined, feedbackMemories...)
	combined = append(combined, factsMemories...)

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].UpdatedAt.After(combined[j].UpdatedAt)
	})

	return combined, nil
}

// ListerOption configures a Lister.
type ListerOption func(*Lister)

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

// ModifyFunc atomically reads, mutates, and writes a memory record.
// This is the signature of Modifier.ReadModifyWrite and is used as a DI boundary
// by packages that need to update memory files without importing the full Modifier.
type ModifyFunc func(path string, mutate func(*MemoryRecord)) error

// StoredRecord pairs a file path with its parsed MemoryRecord.
type StoredRecord struct {
	Path   string
	Record MemoryRecord
}

// WithListerReadDir overrides the directory reading function.
func WithListerReadDir(fn func(string) ([]os.DirEntry, error)) ListerOption {
	return func(l *Lister) { l.readDir = fn }
}

// WithListerReadFile overrides the file reading function.
func WithListerReadFile(fn func(string) ([]byte, error)) ListerOption {
	return func(l *Lister) { l.readFile = fn }
}

// WithModifierReadFile overrides the file reading function.
func WithModifierReadFile(fn func(string) ([]byte, error)) ModifierOption {
	return func(m *Modifier) { m.readFile = fn }
}

// WithModifierWriter sets the AtomicWriter for atomic writes.
func WithModifierWriter(w AtomicWriter) ModifierOption {
	return func(m *Modifier) { m.writer = w }
}

// isNotExist checks if an error wraps os.ErrNotExist.
func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
