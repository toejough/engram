package memory

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestLogChangelogMutation_AllActions verifies all action types can be logged.
func TestLogChangelogMutation_AllActions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	actions := []string{"promote", "demote", "prune", "decay", "consolidate", "rewrite"}

	for _, action := range actions {
		logChangelogMutation(tmpDir, action, "embeddings", "skills", "test")
	}

	// All 6 actions should result in 6 entries
	entries, err := ReadChangelogEntries(tmpDir, ChangelogFilter{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(len(actions)))
}

// TestLogChangelogMutation_SilentOnBadPath verifies logChangelogMutation ignores write errors.
func TestLogChangelogMutation_SilentOnBadPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Calling with a path we can't write to should not panic
	logChangelogMutation("/dev/null/cannot/write", "prune", "embeddings", "", "test")

	g.Expect(true).To(BeTrue()) // just verify no panic
}

// TestLogChangelogMutation_WritesEntry verifies logChangelogMutation writes to changelog.
func TestLogChangelogMutation_WritesEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Should write an entry without error (it silently ignores errors)
	logChangelogMutation(tmpDir, "promote", "embeddings", "skills", "cluster_size >= 3")

	logPath := filepath.Join(tmpDir, "changelog.jsonl")
	_, err := os.Stat(logPath)

	g.Expect(err).ToNot(HaveOccurred(), "changelog.jsonl should be created")
}
