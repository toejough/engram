package memory

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/onsi/gomega"
)

func TestAutoMaintenance_PrunesLowConfidence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	db := createTestMaintenanceDB(t)

	// Insert low-confidence entry (should be pruned)
	insertTestEntry(t, db, "low-conf content", 0.2, 0, time.Now().Add(-10*24*time.Hour))

	// Insert healthy entry (should NOT be pruned)
	insertTestEntry(t, db, "healthy content", 0.9, 10, time.Now().Add(-5*24*time.Hour))

	pruned, _, err := AutoMaintenance(db)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(pruned).To(gomega.Equal(1))

	// Verify healthy entry still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(count).To(gomega.Equal(1))
}

func TestAutoMaintenance_DecaysStaleEntries(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	db := createTestMaintenanceDB(t)

	// Insert stale entry (>90 days, <5 retrievals, confidence >= 0.3)
	insertTestEntry(t, db, "stale content", 0.8, 2, time.Now().Add(-100*24*time.Hour))

	// Insert recent entry (should NOT decay)
	insertTestEntry(t, db, "recent content", 0.8, 2, time.Now().Add(-10*24*time.Hour))

	_, decayed, err := AutoMaintenance(db)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(decayed).To(gomega.Equal(1))

	// Verify stale entry confidence was halved
	var conf float64
	err = db.QueryRow("SELECT confidence FROM embeddings WHERE content = 'stale content'").Scan(&conf)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conf).To(gomega.BeNumerically("~", 0.4, 0.01))
}

func TestAutoMaintenance_NoOpWhenNothingToDo(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	db := createTestMaintenanceDB(t)

	// Insert only healthy entries
	insertTestEntry(t, db, "healthy 1", 0.9, 10, time.Now().Add(-5*24*time.Hour))
	insertTestEntry(t, db, "healthy 2", 0.7, 8, time.Now().Add(-30*24*time.Hour))

	pruned, decayed, err := AutoMaintenance(db)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(pruned).To(gomega.Equal(0))
	g.Expect(decayed).To(gomega.Equal(0))
}

// createTestMaintenanceDB creates an in-memory SQLite database with the embeddings schema.
func createTestMaintenanceDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		confidence REAL DEFAULT 1.0,
		retrieval_count INTEGER DEFAULT 0,
		created_at TEXT NOT NULL,
		last_retrieved TEXT,
		promoted INTEGER DEFAULT 0,
		embedding_id TEXT
	)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	return db
}

// insertTestEntry inserts a test embedding entry.
func insertTestEntry(t *testing.T, db *sql.DB, content string, confidence float64, retrievals int, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO embeddings (content, confidence, retrieval_count, created_at) VALUES (?, ?, ?, ?)`,
		content, confidence, retrievals, createdAt.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("failed to insert test entry: %v", err)
	}
}
