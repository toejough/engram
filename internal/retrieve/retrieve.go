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
type Retriever struct{}

// New creates a Retriever.
func New() *Retriever {
	return &Retriever{}
}

// ListMemories reads all .toml files from <dataDir>/memories/,
// parses them into Stored structs, and returns them sorted by UpdatedAt descending.
// Unparseable files are skipped (logged to stderr).
func (r *Retriever) ListMemories(_ context.Context, dataDir string) ([]*memory.Stored, error) {
	memoriesDir := filepath.Join(dataDir, "memories")

	entries, err := os.ReadDir(memoriesDir)
	if err != nil {
		return nil, fmt.Errorf("retrieve: read memories dir: %w", err)
	}

	memories := make([]*memory.Stored, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		filePath := filepath.Join(memoriesDir, entry.Name())

		mem, parseErr := parseMemoryFile(filePath)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "[engram] Warning: skipping %s: %v\n", entry.Name(), parseErr)
			continue
		}

		memories = append(memories, mem)
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})

	return memories, nil
}

// tomlRecord mirrors the on-disk TOML format for reading.
//

type tomlRecord struct {
	Title       string   `toml:"title"`
	Content     string   `toml:"content"`
	Concepts    []string `toml:"concepts"`
	Keywords    []string `toml:"keywords"`
	AntiPattern string   `toml:"anti_pattern"`
	Principle   string   `toml:"principle"`
	UpdatedAt   string   `toml:"updated_at"`
}

func parseMemoryFile(filePath string) (*memory.Stored, error) {
	var record tomlRecord

	_, err := toml.DecodeFile(filePath, &record)
	if err != nil {
		return nil, fmt.Errorf("decoding TOML: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, record.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}

	return &memory.Stored{
		Title:       record.Title,
		Content:     record.Content,
		Concepts:    record.Concepts,
		Keywords:    record.Keywords,
		AntiPattern: record.AntiPattern,
		Principle:   record.Principle,
		UpdatedAt:   updatedAt,
		FilePath:    filePath,
	}, nil
}
