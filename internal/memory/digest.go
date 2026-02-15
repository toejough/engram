package memory

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DigestOptions specifies parameters for computing a learning digest.
type DigestOptions struct {
	Since      time.Duration // Only include entries from this duration ago
	Tier       string        // Filter by tier (skill, embedding, claude_md)
	FlagsOnly  bool          // Only show flags, not full digest
	MaxEntries int           // Maximum number of entries to include (0 = unlimited)
}

// Digest represents a summary of recent learning activity and system health.
type Digest struct {
	Since           time.Duration
	GeneratedAt     time.Time
	RecentLearnings []ChangelogEntry
	SkillChanges    []ChangelogEntry
	ClaudeMDChanges []ChangelogEntry
	LatestMetrics   *MetricsSnapshot
	Flags           []string
}

// ComputeDigest computes a learning digest from changelog, retrievals, and metrics.
func ComputeDigest(opts DigestOptions, memoryRoot string) (Digest, error) {
	digest := Digest{
		Since:       opts.Since,
		GeneratedAt: time.Now(),
	}

	// Compute time threshold
	since := time.Now().Add(-opts.Since)

	// Read changelog entries (without tier filtering initially)
	changelogFilter := ChangelogFilter{
		Since: since,
	}

	entries, err := ReadChangelogEntries(memoryRoot, changelogFilter)
	if err != nil {
		return digest, fmt.Errorf("failed to read changelog: %w", err)
	}

	// Filter by tier if specified (match either source or destination)
	if opts.Tier != "" {
		var filteredEntries []ChangelogEntry
		for _, entry := range entries {
			if entry.SourceTier == opts.Tier || entry.DestinationTier == opts.Tier {
				filteredEntries = append(filteredEntries, entry)
			}
		}
		entries = filteredEntries
	}

	// Separate entries by category
	for _, entry := range entries {
		// CLAUDE.md changes (promote to or demote from claude_md)
		if entry.SourceTier == "claude_md" || entry.DestinationTier == "claude_md" {
			digest.ClaudeMDChanges = append(digest.ClaudeMDChanges, entry)
		}

		// Skill changes (promote to or demote from skill)
		if entry.SourceTier == "skill" || entry.DestinationTier == "skill" {
			digest.SkillChanges = append(digest.SkillChanges, entry)
		}

		// All entries go to recent learnings
		digest.RecentLearnings = append(digest.RecentLearnings, entry)
	}

	// Apply max entries limit if set
	if opts.MaxEntries > 0 {
		if len(digest.RecentLearnings) > opts.MaxEntries {
			digest.RecentLearnings = digest.RecentLearnings[:opts.MaxEntries]
		}
		if len(digest.SkillChanges) > opts.MaxEntries {
			digest.SkillChanges = digest.SkillChanges[:opts.MaxEntries]
		}
		if len(digest.ClaudeMDChanges) > opts.MaxEntries {
			digest.ClaudeMDChanges = digest.ClaudeMDChanges[:opts.MaxEntries]
		}
	}

	// Read latest metrics snapshot
	metricsOpts := ReadMetricsSnapshotsOpts{
		MetricsDir: memoryRoot,
		Since:      &since,
	}
	snapshots, err := ReadMetricsSnapshots(metricsOpts)
	if err == nil && len(snapshots) > 0 {
		digest.LatestMetrics = &snapshots[len(snapshots)-1]
	}

	// Detect flags
	digest.Flags = detectFlags(entries, digest.LatestMetrics)

	return digest, nil
}

// detectFlags analyzes changelog entries and metrics to detect system health issues.
func detectFlags(entries []ChangelogEntry, metrics *MetricsSnapshot) []string {
	var flags []string

	// Flag 1: Corrections recurring
	recurrenceCount := 0
	for _, entry := range entries {
		if entry.Action == "correction_recurrence" {
			if countStr, ok := entry.Metadata["count"]; ok {
				if count, err := strconv.Atoi(countStr); err == nil && count > 1 {
					recurrenceCount++
				}
			}
		}
	}
	if recurrenceCount > 0 {
		flags = append(flags, fmt.Sprintf("Recurring corrections detected (%d instances)", recurrenceCount))
	}

	// Flag 2: Low retrieval precision
	if metrics != nil && metrics.RetrievalPrecision > 0 && metrics.RetrievalPrecision < 0.5 {
		flags = append(flags, fmt.Sprintf("Low retrieval precision (%.2f)", metrics.RetrievalPrecision))
	}

	// Flag 3: Violations increasing
	if metrics != nil && len(metrics.HookViolationTrend) > 0 {
		for rule, trend := range metrics.HookViolationTrend {
			if trend == "degrading" || trend == "increasing" {
				flags = append(flags, fmt.Sprintf("Hook violations increasing: %s", rule))
			}
		}
	}

	// Flag 4: Skills awaiting test
	if metrics != nil && metrics.SkillsAwaitingTest > 0 {
		flags = append(flags, fmt.Sprintf("Skills awaiting test: %d", metrics.SkillsAwaitingTest))
	}

	return flags
}

// FormatDigest formats a digest into human-readable output.
func FormatDigest(d Digest) string {
	var out strings.Builder

	// Header
	out.WriteString("─── Learning Digest ───────────────────────\n")
	out.WriteString(fmt.Sprintf("Generated: %s\n", d.GeneratedAt.Format("2006-01-02 15:04:05")))
	out.WriteString(fmt.Sprintf("Period: Last %s\n", d.Since))
	out.WriteString("\n")

	// Recent Learnings
	if len(d.RecentLearnings) > 0 {
		out.WriteString("Recent Learnings:\n")
		for _, entry := range d.RecentLearnings {
			out.WriteString(fmt.Sprintf("  • [%s] %s", entry.Action, entry.ContentSummary))
			if entry.SourceTier != "" {
				out.WriteString(fmt.Sprintf(" (%s", entry.SourceTier))
				if entry.DestinationTier != "" {
					out.WriteString(fmt.Sprintf(" → %s", entry.DestinationTier))
				}
				out.WriteString(")")
			}
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}

	// Skill Changes
	if len(d.SkillChanges) > 0 {
		out.WriteString("Skill Changes:\n")
		for _, entry := range d.SkillChanges {
			out.WriteString(fmt.Sprintf("  • [%s] %s\n", entry.Action, entry.ContentSummary))
		}
		out.WriteString("\n")
	}

	// CLAUDE.md Changes
	if len(d.ClaudeMDChanges) > 0 {
		out.WriteString("CLAUDE.md Changes:\n")
		for _, entry := range d.ClaudeMDChanges {
			out.WriteString(fmt.Sprintf("  • [%s] %s\n", entry.Action, entry.ContentSummary))
		}
		out.WriteString("\n")
	}

	// Metrics
	if d.LatestMetrics != nil {
		out.WriteString("Latest Metrics:\n")
		out.WriteString(fmt.Sprintf("  • Correction recurrence rate: %.2f\n", d.LatestMetrics.CorrectionRecurrenceRate))
		out.WriteString(fmt.Sprintf("  • Retrieval precision: %.2f\n", d.LatestMetrics.RetrievalPrecision))
		out.WriteString(fmt.Sprintf("  • Embeddings: %d\n", d.LatestMetrics.EmbeddingCount))
		out.WriteString(fmt.Sprintf("  • Skills: %d\n", d.LatestMetrics.SkillCount))
		out.WriteString(fmt.Sprintf("  • CLAUDE.md lines: %d\n", d.LatestMetrics.ClaudeMDLines))
		out.WriteString("\n")
	}

	// Flags
	if len(d.Flags) > 0 {
		out.WriteString("Flags:\n")
		for _, flag := range d.Flags {
			out.WriteString(fmt.Sprintf("  ⚠ %s\n", flag))
		}
		out.WriteString("\n")
	}

	out.WriteString("───────────────────────────────────────────\n")

	return out.String()
}
