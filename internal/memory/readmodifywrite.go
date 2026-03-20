package memory

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

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

// ReadModifyWrite atomically reads a memory TOML, applies a mutation, and writes back.
func ReadModifyWrite(path string, mutate func(*MemoryRecord)) error {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted internal source
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var record MemoryRecord

	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}

	mutate(&record)

	var buf bytes.Buffer

	err = toml.NewEncoder(&buf).Encode(record)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, ".tmp-rmw")

	const filePerm = 0o644

	err = os.WriteFile(tmpPath, buf.Bytes(), filePerm)
	if err != nil {
		return fmt.Errorf("writing temp: %w", err)
	}

	err = os.Rename(tmpPath, path)
	if err != nil {
		return fmt.Errorf("renaming temp to %s: %w", path, err)
	}

	return nil
}
