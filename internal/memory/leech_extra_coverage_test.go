//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestApplyLeechAction_DefaultCase verifies ApplyLeechAction returns error for unknown action.
func TestApplyLeechAction_DefaultCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	diagnosis := memory.LeechDiagnosis{
		MemoryID:       1,
		ProposedAction: "unknown_action",
	}

	err = memory.ApplyLeechAction(db, diagnosis, memory.RealFS{})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown action"))
}

// TestApplyLeechAction_MarkActionRecommended_DBError verifies markActionRecommended error path.
func TestApplyLeechAction_MarkActionRecommended_DBError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop and recreate embeddings without the leech_action column so UPDATE fails
	_, err = db.Exec("DROP TABLE embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE TABLE embeddings (id INTEGER PRIMARY KEY, content TEXT, source TEXT)")
	g.Expect(err).ToNot(HaveOccurred())

	diagnosis := memory.LeechDiagnosis{
		MemoryID:       1,
		ProposedAction: "promote_to_claude_md",
	}

	err = memory.ApplyLeechAction(db, diagnosis, memory.RealFS{})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("mark action_recommended"))
}
