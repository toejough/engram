package cli

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/memory"
)

// readRecord reads a memory TOML file into a *memory.MemoryRecord.
// Always overwrites SourcePath with the given path (on-disk value may be stale).
func readRecord(path string) (*memory.MemoryRecord, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted flag/internal source
	if err != nil {
		return nil, fmt.Errorf("reading record %s: %w", path, err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding record TOML: %w", decodeErr)
	}

	record.SourcePath = path

	return &record, nil
}
