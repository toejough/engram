//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestRecordHookEvent_InsertError verifies RecordHookEvent returns error when schema is corrupt.
func TestRecordHookEvent_InsertError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop and recreate hook_events with minimal schema (missing exit_code column)
	_, err = db.Exec("DROP TABLE hook_events")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE TABLE hook_events (id INTEGER PRIMARY KEY, hook_name TEXT NOT NULL, fired_at TEXT NOT NULL, duration_ms INTEGER)")
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.RecordHookEvent(db, "Stop", 0, 100)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to insert hook event"))
}

// TestRecordHookEvent_PruneError verifies RecordHookEvent returns error when the prune DELETE fails.
func TestRecordHookEvent_PruneError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Pre-insert 1001 rows so the prune DELETE will actually try to delete rows (prune keeps last 1000)
	_, err = db.Exec(`WITH RECURSIVE cnt(x) AS (
		SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 1001
	)
	INSERT INTO hook_events (hook_name, fired_at, exit_code, duration_ms)
	SELECT 'Stop', datetime('now'), 0, 100 FROM cnt`)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a BEFORE DELETE trigger that aborts any DELETE on hook_events
	_, err = db.Exec("CREATE TRIGGER block_delete BEFORE DELETE ON hook_events BEGIN SELECT RAISE(FAIL, 'deletion blocked'); END")
	g.Expect(err).ToNot(HaveOccurred())

	// RecordHookEvent inserts 1 more row (1002 total), then tries to DELETE 2 old rows — trigger fires and fails
	err = memory.RecordHookEvent(db, "Stop", 0, 100)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to prune hook events"))
}
