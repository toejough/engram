package memory_test

import (
	"strings"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestComputeDigest_EmptyDirectory(t *testing.T) {
	memoryRoot := t.TempDir()

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(digest.RecentLearnings) != 0 {
		t.Errorf("expected no recent learnings, got %d", len(digest.RecentLearnings))
	}
	if len(digest.Flags) != 0 {
		t.Errorf("expected no flags, got %d", len(digest.Flags))
	}
}

func TestComputeDigest_WithChangelogEntries(t *testing.T) {
	memoryRoot := t.TempDir()

	// Write sample changelog entries
	now := time.Now()
	entries := []memory.ChangelogEntry{
		{
			Timestamp:       now.Add(-2 * time.Hour),
			Action:          "store",
			DestinationTier: "embedding",
			ContentSummary:  "Use AI-Used trailer",
			Reason:          "correction",
		},
		{
			Timestamp:       now.Add(-1 * time.Hour),
			Action:          "promote",
			SourceTier:      "embedding",
			DestinationTier: "skill",
			ContentSummary:  "TDD workflow",
			Reason:          "cluster_size=5",
		},
		{
			Timestamp:       now.Add(-30 * time.Minute),
			Action:          "demote",
			SourceTier:      "claude_md",
			DestinationTier: "skill",
			ContentSummary:  "git workflow",
			Reason:          "narrowing scope",
		},
	}

	for _, entry := range entries {
		if err := memory.WriteChangelogEntry(memoryRoot, entry); err != nil {
			t.Fatalf("failed to write changelog entry: %v", err)
		}
	}

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(digest.RecentLearnings) != 3 {
		t.Errorf("expected 3 recent learnings, got %d", len(digest.RecentLearnings))
	}
}

func TestComputeDigest_TierFilter(t *testing.T) {
	memoryRoot := t.TempDir()

	now := time.Now()
	entries := []memory.ChangelogEntry{
		{
			Timestamp:       now.Add(-1 * time.Hour),
			Action:          "promote",
			SourceTier:      "embedding",
			DestinationTier: "skill",
			ContentSummary:  "skill change",
		},
		{
			Timestamp:       now.Add(-30 * time.Minute),
			Action:          "store",
			DestinationTier: "embedding",
			ContentSummary:  "embedding change",
		},
	}

	for _, entry := range entries {
		if err := memory.WriteChangelogEntry(memoryRoot, entry); err != nil {
			t.Fatalf("failed to write changelog entry: %v", err)
		}
	}

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
		Tier:  "skill",
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should only include skill-related entries
	if len(digest.SkillChanges) == 0 {
		t.Error("expected skill changes, got none")
	}
	if len(digest.RecentLearnings) > 1 {
		t.Errorf("expected at most 1 learning when filtering by tier, got %d", len(digest.RecentLearnings))
	}
}

func TestComputeDigest_FlagDetection_RecurringCorrections(t *testing.T) {
	memoryRoot := t.TempDir()

	now := time.Now()
	// Create multiple correction recurrence entries
	entries := []memory.ChangelogEntry{
		{
			Timestamp: now.Add(-1 * time.Hour),
			Action:    "correction_recurrence",
			Metadata: map[string]string{
				"count": "2",
			},
			ContentSummary: "git commit trailer",
		},
		{
			Timestamp: now.Add(-30 * time.Minute),
			Action:    "correction_recurrence",
			Metadata: map[string]string{
				"count": "3",
			},
			ContentSummary: "TDD workflow",
		},
	}

	for _, entry := range entries {
		if err := memory.WriteChangelogEntry(memoryRoot, entry); err != nil {
			t.Fatalf("failed to write changelog entry: %v", err)
		}
	}

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should detect recurring corrections flag
	foundFlag := false
	for _, flag := range digest.Flags {
		lowerFlag := strings.ToLower(flag)
		if strings.Contains(lowerFlag, "recurring") || strings.Contains(lowerFlag, "recurrence") {
			foundFlag = true
			break
		}
	}

	if !foundFlag {
		t.Errorf("expected to find recurring corrections flag, got flags: %v", digest.Flags)
	}
}

func TestComputeDigest_MaxEntries(t *testing.T) {
	memoryRoot := t.TempDir()

	now := time.Now()
	// Create 10 entries
	for i := 0; i < 10; i++ {
		entry := memory.ChangelogEntry{
			Timestamp:      now.Add(-time.Duration(i) * time.Hour),
			Action:         "store",
			ContentSummary: "entry " + string(rune('A'+i)),
		}
		if err := memory.WriteChangelogEntry(memoryRoot, entry); err != nil {
			t.Fatalf("failed to write changelog entry: %v", err)
		}
	}

	opts := memory.DigestOptions{
		Since:      24 * time.Hour,
		MaxEntries: 5,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(digest.RecentLearnings) > 5 {
		t.Errorf("expected at most 5 learnings, got %d", len(digest.RecentLearnings))
	}
}

func TestFormatDigest_BasicStructure(t *testing.T) {
	now := time.Now()
	digest := memory.Digest{
		Since:       24 * time.Hour,
		GeneratedAt: now,
		RecentLearnings: []memory.ChangelogEntry{
			{
				Timestamp:      now.Add(-1 * time.Hour),
				Action:         "store",
				ContentSummary: "Use AI-Used trailer",
			},
		},
		SkillChanges: []memory.ChangelogEntry{
			{
				Timestamp:       now.Add(-30 * time.Minute),
				Action:          "promote",
				DestinationTier: "skill",
				ContentSummary:  "TDD workflow",
			},
		},
		Flags: []string{
			"Recurring corrections detected (2 instances)",
		},
	}

	output := memory.FormatDigest(digest)

	// Should contain key sections
	if !strings.Contains(output, "Learning Digest") {
		t.Error("expected output to contain 'Learning Digest' header")
	}
	if !strings.Contains(output, "Recent Learnings") {
		t.Error("expected output to contain 'Recent Learnings' section")
	}
	if !strings.Contains(output, "Skill Changes") {
		t.Error("expected output to contain 'Skill Changes' section")
	}
	if !strings.Contains(output, "Flags") {
		t.Error("expected output to contain 'Flags' section")
	}
	if !strings.Contains(output, "AI-Used trailer") {
		t.Error("expected output to contain learning content")
	}
}

func TestFormatDigest_FlagsOnly(t *testing.T) {
	now := time.Now()
	digest := memory.Digest{
		Since:       24 * time.Hour,
		GeneratedAt: now,
		RecentLearnings: []memory.ChangelogEntry{
			{
				Timestamp:      now.Add(-1 * time.Hour),
				Action:         "store",
				ContentSummary: "Some learning",
			},
		},
		Flags: []string{
			"Low retrieval precision (0.45)",
		},
	}

	// Test that when we have flags-only mode, we don't show learnings
	// (This will be implemented in the CLI layer, but the format function should support it)
	output := memory.FormatDigest(digest)

	// For now, just verify it formats flags
	if !strings.Contains(output, "Low retrieval precision") {
		t.Error("expected output to contain flag")
	}
}

func TestComputeDigest_WithMetrics(t *testing.T) {
	memoryRoot := t.TempDir()

	// Write a metrics snapshot
	metricsOpts := memory.TakeMetricsSnapshotOpts{
		MetricsDir: memoryRoot,
	}
	if err := memory.TakeMetricsSnapshot(metricsOpts); err != nil {
		t.Fatalf("failed to write metrics snapshot: %v", err)
	}

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if digest.LatestMetrics == nil {
		t.Error("expected latest metrics to be populated")
	}
}

func TestComputeDigest_CLAUDEMDChanges(t *testing.T) {
	memoryRoot := t.TempDir()

	now := time.Now()
	entries := []memory.ChangelogEntry{
		{
			Timestamp:       now.Add(-1 * time.Hour),
			Action:          "promote",
			SourceTier:      "skill",
			DestinationTier: "claude_md",
			ContentSummary:  "promoted to CLAUDE.md",
		},
		{
			Timestamp:       now.Add(-30 * time.Minute),
			Action:          "demote",
			SourceTier:      "claude_md",
			DestinationTier: "skill",
			ContentSummary:  "demoted from CLAUDE.md",
		},
	}

	for _, entry := range entries {
		if err := memory.WriteChangelogEntry(memoryRoot, entry); err != nil {
			t.Fatalf("failed to write changelog entry: %v", err)
		}
	}

	opts := memory.DigestOptions{
		Since: 24 * time.Hour,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(digest.ClaudeMDChanges) != 2 {
		t.Errorf("expected 2 CLAUDE.md changes, got %d", len(digest.ClaudeMDChanges))
	}
}
