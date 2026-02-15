package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestMetricsSnapshot_MarshalRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	snap := memory.MetricsSnapshot{
		Timestamp:                   now,
		CorrectionRecurrenceRate:    0.15,
		RetrievalPrecision:          0.82,
		HookViolationTrend:          map[string]string{"no-force-push": "improving", "ai-trailer": "stable"},
		EmbeddingCount:              42,
		SkillCount:                  7,
		ClaudeMDLines:               95,
		AverageCorrectionConfidence: 0.91,
		SkillsAwaitingTest:          3,
		Metadata:                    map[string]string{"trigger": "optimize"},
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got memory.MetricsSnapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !got.Timestamp.Equal(snap.Timestamp) {
		t.Errorf("timestamp: got %v, want %v", got.Timestamp, snap.Timestamp)
	}
	if got.CorrectionRecurrenceRate != snap.CorrectionRecurrenceRate {
		t.Errorf("correction_recurrence_rate: got %v, want %v", got.CorrectionRecurrenceRate, snap.CorrectionRecurrenceRate)
	}
	if got.RetrievalPrecision != snap.RetrievalPrecision {
		t.Errorf("retrieval_precision: got %v, want %v", got.RetrievalPrecision, snap.RetrievalPrecision)
	}
	if got.EmbeddingCount != snap.EmbeddingCount {
		t.Errorf("embedding_count: got %v, want %v", got.EmbeddingCount, snap.EmbeddingCount)
	}
	if got.SkillCount != snap.SkillCount {
		t.Errorf("skill_count: got %v, want %v", got.SkillCount, snap.SkillCount)
	}
	if got.ClaudeMDLines != snap.ClaudeMDLines {
		t.Errorf("claude_md_lines: got %v, want %v", got.ClaudeMDLines, snap.ClaudeMDLines)
	}
	if got.AverageCorrectionConfidence != snap.AverageCorrectionConfidence {
		t.Errorf("average_correction_confidence: got %v, want %v", got.AverageCorrectionConfidence, snap.AverageCorrectionConfidence)
	}
	if got.SkillsAwaitingTest != snap.SkillsAwaitingTest {
		t.Errorf("skills_awaiting_test: got %v, want %v", got.SkillsAwaitingTest, snap.SkillsAwaitingTest)
	}
	if len(got.HookViolationTrend) != 2 {
		t.Errorf("hook_violation_trend: got %d entries, want 2", len(got.HookViolationTrend))
	}
	if got.Metadata["trigger"] != "optimize" {
		t.Errorf("metadata[trigger]: got %q, want %q", got.Metadata["trigger"], "optimize")
	}
}

func TestTakeMetricsSnapshot_WritesValidJSONL(t *testing.T) {
	dir := t.TempDir()
	metricsPath := filepath.Join(dir, "metrics.jsonl")

	opts := memory.TakeMetricsSnapshotOpts{
		MetricsDir: dir,
		// nil DB — stub mode: tier sizes default to 0
	}
	if err := memory.TakeMetricsSnapshot(opts); err != nil {
		t.Fatalf("TakeMetricsSnapshot: %v", err)
	}

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("read metrics file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d", len(lines))
	}

	var snap memory.MetricsSnapshot
	if err := json.Unmarshal([]byte(lines[0]), &snap); err != nil {
		t.Fatalf("unmarshal JSONL line: %v", err)
	}

	if snap.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
	// Stub mode — counters should be 0
	if snap.EmbeddingCount != 0 {
		t.Errorf("embedding_count: got %d, want 0", snap.EmbeddingCount)
	}
	if snap.SkillCount != 0 {
		t.Errorf("skill_count: got %d, want 0", snap.SkillCount)
	}
}

func TestTakeMetricsSnapshot_Appends(t *testing.T) {
	dir := t.TempDir()
	metricsPath := filepath.Join(dir, "metrics.jsonl")

	opts := memory.TakeMetricsSnapshotOpts{MetricsDir: dir}

	// Write two snapshots
	if err := memory.TakeMetricsSnapshot(opts); err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if err := memory.TakeMetricsSnapshot(opts); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}

	// Both should be valid JSON
	for i, line := range lines {
		var snap memory.MetricsSnapshot
		if err := json.Unmarshal([]byte(line), &snap); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
		}
	}
}

func TestReadMetricsSnapshots_All(t *testing.T) {
	dir := t.TempDir()

	// Write 3 snapshots with known timestamps
	now := time.Now()
	snaps := []memory.MetricsSnapshot{
		{Timestamp: now.Add(-48 * time.Hour), EmbeddingCount: 10},
		{Timestamp: now.Add(-24 * time.Hour), EmbeddingCount: 20},
		{Timestamp: now, EmbeddingCount: 30},
	}
	writeMetricsJSONL(t, dir, snaps)

	got, err := memory.ReadMetricsSnapshots(memory.ReadMetricsSnapshotsOpts{
		MetricsDir: dir,
	})
	if err != nil {
		t.Fatalf("ReadMetricsSnapshots: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(got))
	}
}

func TestReadMetricsSnapshots_FiltersSince(t *testing.T) {
	dir := t.TempDir()

	now := time.Now()
	snaps := []memory.MetricsSnapshot{
		{Timestamp: now.Add(-72 * time.Hour), EmbeddingCount: 10},
		{Timestamp: now.Add(-24 * time.Hour), EmbeddingCount: 20},
		{Timestamp: now.Add(-1 * time.Hour), EmbeddingCount: 30},
	}
	writeMetricsJSONL(t, dir, snaps)

	since := now.Add(-36 * time.Hour)
	got, err := memory.ReadMetricsSnapshots(memory.ReadMetricsSnapshotsOpts{
		MetricsDir: dir,
		Since:      &since,
	})
	if err != nil {
		t.Fatalf("ReadMetricsSnapshots: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots (since 36h ago), got %d", len(got))
	}
	if got[0].EmbeddingCount != 20 {
		t.Errorf("first result embedding_count: got %d, want 20", got[0].EmbeddingCount)
	}
}

func TestReadMetricsSnapshots_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	// No file exists yet
	got, err := memory.ReadMetricsSnapshots(memory.ReadMetricsSnapshotsOpts{
		MetricsDir: dir,
	})
	if err != nil {
		t.Fatalf("ReadMetricsSnapshots: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 snapshots for missing file, got %d", len(got))
	}
}

func TestTakeMetricsSnapshot_WithClaudeMDPath(t *testing.T) {
	dir := t.TempDir()

	// Create a fake CLAUDE.md with known line count
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	content := "# Title\n\nLine 1\nLine 2\nLine 3\n"
	if err := os.WriteFile(claudeMD, []byte(content), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}

	opts := memory.TakeMetricsSnapshotOpts{
		MetricsDir:  dir,
		ClaudeMDPath: claudeMD,
	}
	if err := memory.TakeMetricsSnapshot(opts); err != nil {
		t.Fatalf("TakeMetricsSnapshot: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metrics.jsonl"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var snap memory.MetricsSnapshot
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &snap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if snap.ClaudeMDLines != 5 {
		t.Errorf("claude_md_lines: got %d, want 5", snap.ClaudeMDLines)
	}
}

// writeMetricsJSONL writes a slice of MetricsSnapshot to metrics.jsonl in dir.
func writeMetricsJSONL(t *testing.T, dir string, snaps []memory.MetricsSnapshot) {
	t.Helper()
	path := filepath.Join(dir, "metrics.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create metrics file: %v", err)
	}
	defer func() { _ = f.Close() }()

	for _, snap := range snaps {
		data, err := json.Marshal(snap)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := f.WriteString("\n"); err != nil {
			t.Fatalf("write newline: %v", err)
		}
	}
}
