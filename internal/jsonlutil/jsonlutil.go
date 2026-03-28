// Package jsonlutil provides generic JSONL parsing.
package jsonlutil

import (
	"encoding/json"
	"strings"
)

// ParseLines parses JSONL data into a slice of T, skipping empty/malformed lines.
func ParseLines[T any](data []byte) []T {
	if len(data) == 0 {
		return nil
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	result := make([]T, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		var entry T

		err := json.Unmarshal([]byte(trimmed), &entry)
		if err != nil {
			continue
		}

		result = append(result, entry)
	}

	return result
}
