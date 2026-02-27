package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MetricsSnapshot represents a periodic snapshot of system health metrics.
type MetricsSnapshot struct {
	Timestamp                   time.Time         `json:"timestamp"`
	CorrectionRecurrenceRate    float64           `json:"correction_recurrence_rate"`
	RetrievalPrecision          float64           `json:"retrieval_precision"`
	HookViolationTrend          map[string]string `json:"hook_violation_trend,omitempty"`
	EmbeddingCount              int               `json:"embedding_count"`
	SkillCount                  int               `json:"skill_count"`
	ClaudeMDLines               int               `json:"claude_md_lines"`
	AverageCorrectionConfidence float64           `json:"average_correction_confidence"`
	SkillsAwaitingTest          int               `json:"skills_awaiting_test"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

// ReadMetricsSnapshotsOpts holds options for reading metrics snapshots.
type ReadMetricsSnapshotsOpts struct {
	MetricsDir string
	Since      *time.Time
}

// TakeMetricsSnapshotOpts holds options for taking a metrics snapshot.
type TakeMetricsSnapshotOpts struct {
	MetricsDir   string
	ClaudeMDPath string
}

// ReadMetricsSnapshots reads metrics snapshots from metrics.jsonl, applying optional filters.
func ReadMetricsSnapshots(opts ReadMetricsSnapshotsOpts) ([]MetricsSnapshot, error) {
	metricsPath := filepath.Join(opts.MetricsDir, "metrics.jsonl")

	f, err := os.Open(metricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}

	defer func() { _ = f.Close() }()

	var snapshots []MetricsSnapshot

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var snap MetricsSnapshot

		err := json.Unmarshal(line, &snap)
		if err != nil {
			continue
		}

		if opts.Since != nil && snap.Timestamp.Before(*opts.Since) {
			continue
		}

		snapshots = append(snapshots, snap)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	return snapshots, nil
}

// TakeMetricsSnapshot computes and appends a metrics snapshot to metrics.jsonl.
func TakeMetricsSnapshot(opts TakeMetricsSnapshotOpts) error {
	if err := os.MkdirAll(opts.MetricsDir, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %w", err)
	}

	snap := MetricsSnapshot{
		Timestamp: time.Now(),
	}

	// Count CLAUDE.md lines if path provided
	if opts.ClaudeMDPath != "" {
		data, err := os.ReadFile(opts.ClaudeMDPath)
		if err == nil {
			snap.ClaudeMDLines = len(strings.Split(strings.TrimRight(string(data), "\n"), "\n"))
		}
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics snapshot: %w", err)
	}

	data = append(data, '\n')

	metricsPath := filepath.Join(opts.MetricsDir, "metrics.jsonl")

	f, err := os.OpenFile(metricsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}

	defer func() { _ = f.Close() }()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write metrics snapshot: %w", err)
	}

	return nil
}
