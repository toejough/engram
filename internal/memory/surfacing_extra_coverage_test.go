//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestRecordMemoryFeedback_NoRelevantSession verifies error when no haiku_relevant event exists for session.
func TestRecordMemoryFeedback_NoRelevantSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = memory.RecordMemoryFeedback(db, "nonexistent-session-id", "helpful")

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no relevant surfacing event found"))
}

// TestUpdateSurfacingOutcome_DBError verifies UpdateSurfacingOutcome returns error when table is missing.
func TestUpdateSurfacingOutcome_DBError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec("DROP TABLE surfacing_events")
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.UpdateSurfacingOutcome(db, 1, 0.9, "positive")

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("UpdateSurfacingOutcome"))
}
