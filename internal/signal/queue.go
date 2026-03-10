package signal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// QueueOption configures a QueueStore.
type QueueOption func(*QueueStore)

// QueueStore manages the proposal queue JSONL file with atomic writes.
type QueueStore struct {
	readFile   func(string) ([]byte, error)
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
	now        func() time.Time
}

// NewQueueStore creates a QueueStore with optional DI overrides.
func NewQueueStore(opts ...QueueOption) *QueueStore {
	store := &QueueStore{
		readFile:   os.ReadFile,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Append adds signals to the queue file atomically.
func (q *QueueStore) Append(signals []Signal, path string) error {
	if len(signals) == 0 {
		return nil
	}

	existing, err := q.readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading signal queue: %w", err)
	}

	var sb strings.Builder
	if len(existing) > 0 {
		sb.Write(existing)

		if existing[len(existing)-1] != '\n' {
			sb.WriteByte('\n')
		}
	}

	for _, sig := range signals {
		line, marshalErr := json.Marshal(sig)
		if marshalErr != nil {
			return fmt.Errorf("marshaling signal: %w", marshalErr)
		}

		sb.Write(line)
		sb.WriteByte('\n')
	}

	return q.writeAtomic(path, sb.String())
}

// ClearBySourceID removes all entries matching the given sourceID.
func (q *QueueStore) ClearBySourceID(path, sourceID string) error {
	signals, err := q.Read(path)
	if err != nil {
		return err
	}

	kept := make([]Signal, 0, len(signals))

	for _, sig := range signals {
		if sig.SourceID != sourceID {
			kept = append(kept, sig)
		}
	}

	return q.writeSignals(path, kept)
}

// Prune removes stale entries (>30 days), entries for deleted sources, and deduplicates
// by source_id+type. existsCheck returns true if the source file exists.
func (q *QueueStore) Prune(path string, existsCheck func(string) bool) error {
	signals, err := q.Read(path)
	if err != nil {
		return err
	}

	if len(signals) == 0 {
		return nil
	}

	now := q.now()
	seen := make(map[string]struct{})
	kept := make([]Signal, 0, len(signals))

	for _, sig := range signals {
		if now.Sub(sig.DetectedAt) > pruneMaxAge {
			continue
		}

		if sig.SourceID != "" && !existsCheck(sig.SourceID) {
			continue
		}

		dedupKey := sig.SourceID + "|" + sig.Type
		if _, exists := seen[dedupKey]; exists {
			continue
		}

		seen[dedupKey] = struct{}{}

		kept = append(kept, sig)
	}

	return q.writeSignals(path, kept)
}

// Read reads all signals from the queue file.
// Missing file returns empty slice with no error. Malformed lines are skipped.
func (q *QueueStore) Read(path string) ([]Signal, error) {
	data, err := q.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]Signal, 0), nil
		}

		return nil, fmt.Errorf("reading signal queue: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	signals := make([]Signal, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var sig Signal

		jsonErr := json.Unmarshal([]byte(line), &sig)
		if jsonErr != nil {
			continue
		}

		signals = append(signals, sig)
	}

	return signals, nil
}

func (q *QueueStore) writeAtomic(targetPath, content string) error {
	tmpFile, err := q.createTemp(filepath.Dir(targetPath), "engram-signal-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	_, writeErr := tmpFile.WriteString(content)
	if writeErr != nil {
		_ = tmpFile.Close()
		_ = q.remove(tmpPath)

		return fmt.Errorf("writing signal queue: %w", writeErr)
	}

	closeErr := tmpFile.Close()
	if closeErr != nil {
		_ = q.remove(tmpPath)

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	renameErr := q.rename(tmpPath, targetPath)
	if renameErr != nil {
		_ = q.remove(tmpPath)

		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}

func (q *QueueStore) writeSignals(path string, signals []Signal) error {
	var sb strings.Builder

	for _, sig := range signals {
		//nolint:errchkjson // Signal has only string/time fields; cannot fail.
		line, _ := json.Marshal(sig)
		sb.Write(line)
		sb.WriteByte('\n')
	}

	return q.writeAtomic(path, sb.String())
}

// WithQueueCreateTemp injects a temp file creation function.
func WithQueueCreateTemp(fn func(dir, pattern string) (*os.File, error)) QueueOption {
	return func(q *QueueStore) {
		q.createTemp = fn
	}
}

// WithQueueNow injects a clock function.
func WithQueueNow(fn func() time.Time) QueueOption {
	return func(q *QueueStore) {
		q.now = fn
	}
}

// WithQueueReadFile injects a readFile function.
func WithQueueReadFile(fn func(string) ([]byte, error)) QueueOption {
	return func(q *QueueStore) {
		q.readFile = fn
	}
}

// WithQueueRemove injects a remove function.
func WithQueueRemove(fn func(name string) error) QueueOption {
	return func(q *QueueStore) {
		q.remove = fn
	}
}

// WithQueueRename injects a rename function.
func WithQueueRename(fn func(oldpath, newpath string) error) QueueOption {
	return func(q *QueueStore) {
		q.rename = fn
	}
}

// unexported constants.
const (
	pruneMaxAge = 30 * 24 * time.Hour
)
