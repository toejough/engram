// Package retrieve reads stored memories from TOML files on disk (ARCH-9).
package retrieve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// Retriever reads memory TOML files from a data directory.
type Retriever struct {
	readDir  func(string) ([]os.DirEntry, error)
	readFile func(string) ([]byte, error)
}

// New creates a Retriever with default I/O wired to the real filesystem.
func New() *Retriever {
	return &Retriever{
		readDir:  os.ReadDir,
		readFile: os.ReadFile,
	}
}

// ListMemories reads all .toml files from <dataDir>/memories/,
// parses them into Stored structs, and returns them sorted by UpdatedAt descending.
// Unparseable files are skipped silently.
func (r *Retriever) ListMemories(_ context.Context, dataDir string) ([]*memory.Stored, error) {
	memoriesDir := filepath.Join(dataDir, "memories")

	entries, err := r.readDir(memoriesDir)
	if err != nil {
		return nil, fmt.Errorf("retrieve: read memories dir: %w", err)
	}

	memories := make([]*memory.Stored, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		filePath := filepath.Join(memoriesDir, entry.Name())

		mem, parseErr := r.parseMemoryFile(filePath)
		if parseErr != nil {
			continue
		}

		memories = append(memories, mem)
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})

	return memories, nil
}

func (r *Retriever) parseMemoryFile(filePath string) (*memory.Stored, error) {
	data, err := r.readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var record memory.MemoryRecord

	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return nil, fmt.Errorf("decoding TOML: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, record.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}

	return &memory.Stored{
		Title:             record.Title,
		Content:           record.Content,
		Concepts:          record.Concepts,
		Keywords:          record.Keywords,
		AntiPattern:       record.AntiPattern,
		Principle:         record.Principle,
		SurfacedCount:     record.SurfacedCount,
		FollowedCount:     record.FollowedCount,
		ContradictedCount: record.ContradictedCount,
		IgnoredCount:      record.IgnoredCount,
		IrrelevantCount:   record.IrrelevantCount,
		IrrelevantQueries: record.IrrelevantQueries,
		UpdatedAt:         updatedAt,
		FilePath:          filePath,
	}, nil
}
