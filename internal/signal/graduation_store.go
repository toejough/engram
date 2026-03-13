package signal

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GraduationStore manages graduation-queue.jsonl.
type GraduationStore struct {
	readFile   func(string) ([]byte, error)
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
}

// NewGraduationStore creates a GraduationStore with optional DI overrides.
func NewGraduationStore(opts ...GraduationStoreOption) *GraduationStore {
	store := &GraduationStore{
		readFile:   os.ReadFile,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
	}
	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Append adds an entry to the graduation queue.
func (g *GraduationStore) Append(entry GraduationEntry, path string) error {
	existing, err := g.readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading graduation queue: %w", err)
	}

	var sb strings.Builder
	if len(existing) > 0 {
		sb.Write(existing)

		if existing[len(existing)-1] != '\n' {
			sb.WriteByte('\n')
		}
	}

	line, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		return fmt.Errorf("marshaling entry: %w", marshalErr)
	}

	sb.Write(line)
	sb.WriteByte('\n')

	return g.writeAtomic(path, sb.String())
}

// List reads all entries from the graduation queue.
func (g *GraduationStore) List(path string) ([]GraduationEntry, error) {
	data, err := g.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]GraduationEntry, 0), nil
		}

		return nil, fmt.Errorf("reading graduation queue: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	entries := make([]GraduationEntry, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry GraduationEntry

		jsonErr := json.Unmarshal([]byte(line), &entry)
		if jsonErr != nil {
			continue // skip malformed
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// SetStatus updates an entry's status, resolved_at, and issue_url by ID.
func (g *GraduationStore) SetStatus(path, id, status, resolvedAt, issueURL string) error {
	entries, err := g.List(path)
	if err != nil {
		return err
	}

	found := false

	for i := range entries {
		if entries[i].ID == id {
			entries[i].Status = status
			entries[i].ResolvedAt = resolvedAt
			entries[i].IssueURL = issueURL
			found = true

			break
		}
	}

	if !found {
		return ErrGraduationNotFound
	}

	return g.writeEntries(path, entries)
}

func (g *GraduationStore) writeAtomic(targetPath, content string) error {
	tmpFile, err := g.createTemp(filepath.Dir(targetPath), "engram-grad-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	_, writeErr := tmpFile.WriteString(content)
	if writeErr != nil {
		_ = tmpFile.Close()
		_ = g.remove(tmpPath)

		return fmt.Errorf("writing graduation queue: %w", writeErr)
	}

	closeErr := tmpFile.Close()
	if closeErr != nil {
		_ = g.remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	renameErr := g.rename(tmpPath, targetPath)
	if renameErr != nil {
		_ = g.remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}

func (g *GraduationStore) writeEntries(path string, entries []GraduationEntry) error {
	var sb strings.Builder

	for _, entry := range entries {
		//nolint:errchkjson // GraduationEntry has only string/time fields; cannot fail.
		line, _ := json.Marshal(entry)
		sb.Write(line)
		sb.WriteByte('\n')
	}

	return g.writeAtomic(path, sb.String())
}

// GraduationStoreOption configures a GraduationStore.
type GraduationStoreOption func(*GraduationStore)

// GenerateGraduationID returns a stable 12-char ID from memory path.
func GenerateGraduationID(memoryPath string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(memoryPath)))[:12]
}

// WithGraduationCreateTemp injects a temp file creation function.
func WithGraduationCreateTemp(
	fn func(dir, pattern string) (*os.File, error),
) GraduationStoreOption {
	return func(g *GraduationStore) {
		g.createTemp = fn
	}
}

// WithGraduationReadFile injects a readFile function.
func WithGraduationReadFile(fn func(string) ([]byte, error)) GraduationStoreOption {
	return func(g *GraduationStore) {
		g.readFile = fn
	}
}

// WithGraduationRemove injects a remove function.
func WithGraduationRemove(fn func(name string) error) GraduationStoreOption {
	return func(g *GraduationStore) {
		g.remove = fn
	}
}

// WithGraduationRename injects a rename function.
func WithGraduationRename(fn func(oldpath, newpath string) error) GraduationStoreOption {
	return func(g *GraduationStore) {
		g.rename = fn
	}
}
