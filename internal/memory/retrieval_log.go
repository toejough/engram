package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// RetrievalLogEntry represents a single retrieval event logged for measurement.
type RetrievalLogEntry struct {
	Timestamp     string            `json:"timestamp"`
	Hook          string            `json:"hook"`
	Query         string            `json:"query"`
	Results       []RetrievalResult `json:"results"`
	FilteredCount int               `json:"filtered_count,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// RetrievalLogFilter specifies criteria for filtering retrieval log entries.
type RetrievalLogFilter struct {
	Hook      string
	SessionID string
	Since     *time.Time
}

// RetrievalResult represents a single result from a retrieval query.
type RetrievalResult struct {
	ID      int64   `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Tier    string  `json:"tier"`
}

// LogRetrieval appends a retrieval log entry as a JSON line to retrievals.jsonl.
func LogRetrieval(memoryRoot string, entry RetrievalLogEntry) error {
	if err := os.MkdirAll(memoryRoot, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	logPath := filepath.Join(memoryRoot, "retrievals.jsonl")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open retrievals log: %w", err)
	}

	defer func() { _ = f.Close() }()

	return logRetrievalTo(f, entry)
}

// ReadRetrievalLogs reads retrieval log entries from retrievals.jsonl, applying optional filters.
func ReadRetrievalLogs(memoryRoot string, filter RetrievalLogFilter) ([]RetrievalLogEntry, error) {
	logPath := filepath.Join(memoryRoot, "retrievals.jsonl")

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to open retrievals log: %w", err)
	}

	defer func() { _ = f.Close() }()

	var entries []RetrievalLogEntry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry RetrievalLogEntry

		err := json.Unmarshal(line, &entry)
		if err != nil {
			continue
		}

		if matchesRetrievalFilter(entry, filter) {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read retrievals log: %w", err)
	}

	return entries, nil
}

// logRetrievalTo marshals entry as JSON and writes it to w.
// The caller owns the writer lifecycle (open/close).
func logRetrievalTo(w io.WriteCloser, entry RetrievalLogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal retrieval log entry: %w", err)
	}

	data = append(data, '\n')

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write retrieval log entry: %w", err)
	}

	return nil
}

func matchesRetrievalFilter(entry RetrievalLogEntry, filter RetrievalLogFilter) bool {
	if filter.Hook != "" && entry.Hook != filter.Hook {
		return false
	}

	if filter.SessionID != "" && entry.SessionID != filter.SessionID {
		return false
	}

	if filter.Since != nil {
		ts, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil || ts.Before(*filter.Since) {
			return false
		}
	}

	return true
}
