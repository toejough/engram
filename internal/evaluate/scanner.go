// Package evaluate implements memory evaluation scanning and scoring (UC-28).
package evaluate

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// DirListerFunc lists directory entries.
type DirListerFunc func(name string) ([]fs.DirEntry, error)

// PendingMemory pairs a file path and record with a specific pending evaluation entry.
type PendingMemory struct {
	Path   string
	Record *memory.MemoryRecord
	Eval   memory.PendingEvaluation
}

// ReadFileFunc reads a file's contents.
type ReadFileFunc func(name string) ([]byte, error)

// NewFileScanner returns a function that scans TOML files in dataDir/memories/
// for pending evaluations matching the given session ID.
func NewFileScanner(
	dataDir string,
	readFile ReadFileFunc,
	listDir DirListerFunc,
) func(sessionID string) ([]PendingMemory, error) {
	return func(sessionID string) ([]PendingMemory, error) {
		memoriesDir := memory.MemoriesDir(dataDir)

		entries, err := listDir(memoriesDir)
		if err != nil {
			return nil, fmt.Errorf("listing memories directory %s: %w", memoriesDir, err)
		}

		results := make([]PendingMemory, 0)

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
				continue
			}

			filePath := filepath.Join(memoriesDir, entry.Name())

			data, readErr := readFile(filePath)
			if readErr != nil {
				continue
			}

			var record memory.MemoryRecord

			_, decErr := toml.Decode(string(data), &record)
			if decErr != nil {
				continue
			}

			for _, eval := range record.PendingEvaluations {
				if eval.SessionID != sessionID {
					continue
				}

				recordCopy := record

				results = append(results, PendingMemory{
					Path:   filePath,
					Record: &recordCopy,
					Eval:   eval,
				})
			}
		}

		return results, nil
	}
}
