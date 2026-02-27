//go:build sqlite_fts5

package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestResetLastNSessions_QueryError verifies ResetLastNSessions returns error when schema is corrupt.
func TestResetLastNSessions_QueryError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop and recreate processed_sessions with minimal schema (missing session_id, processed_at columns)
	_, err = db.Exec("DROP TABLE processed_sessions")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("CREATE TABLE processed_sessions (id INTEGER PRIMARY KEY)")
	g.Expect(err).ToNot(HaveOccurred())

	count, err := memory.ResetLastNSessions(db, 5, tmpDir)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to query sessions to reset"))
	g.Expect(count).To(Equal(int64(0)))
}
