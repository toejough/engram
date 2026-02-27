//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestRecordFeedback_InsertError verifies RecordFeedback returns error when feedback table is missing.
func TestRecordFeedback_InsertError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec("DROP TABLE feedback")
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.RecordFeedback(db, 1, memory.FeedbackHelpful)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to insert feedback"))
}
