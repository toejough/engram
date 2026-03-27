package maintain

import (
	"fmt"
	"os"
	"path/filepath"

	"engram/internal/memory"
)

// PurgeTierC deletes all .toml memory files in memoriesDir where confidence == "C".
// Returns the count of deleted files and any error encountered during deletion.
func PurgeTierC(memoriesDir string, remove func(string) error) (int, error) {
	records, err := memory.ListAll(memoriesDir)
	if err != nil {
		return 0, fmt.Errorf("purge tier C: listing memories: %w", err)
	}

	deleted := 0

	for _, stored := range records {
		if stored.Record.Confidence != tierCConfidence {
			continue
		}

		path := filepath.Clean(stored.Path)

		removeErr := remove(path)
		if removeErr != nil && !os.IsNotExist(removeErr) {
			return deleted, fmt.Errorf("purge tier C: removing %s: %w", path, removeErr)
		}

		deleted++
	}

	return deleted, nil
}

// unexported constants.
const (
	tierCConfidence = "C"
)
