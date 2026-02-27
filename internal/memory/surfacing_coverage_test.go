package memory_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestFindLatestUnscoredSession_AllScored verifies empty string returned when all sessions scored.
func TestFindLatestUnscoredSession_AllScored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	haikuRelevant := true
	faithfulness := 0.9

	_, err = memory.LogSurfacingEvent(db, memory.SurfacingEvent{
		MemoryID:      1,
		QueryText:     "test query",
		HookEvent:     "Stop",
		Timestamp:     time.Now(),
		SessionID:     "session-scored-001",
		HaikuRelevant: &haikuRelevant,
		Faithfulness:  &faithfulness,
	})
	g.Expect(err).ToNot(HaveOccurred())

	sessionID, err := memory.FindLatestUnscoredSession(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sessionID).To(BeEmpty())
}

// TestFindLatestUnscoredSession_EmptyDB verifies empty string returned for empty DB.
func TestFindLatestUnscoredSession_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	sessionID, err := memory.FindLatestUnscoredSession(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sessionID).To(BeEmpty())
}

// TestFindLatestUnscoredSession_WithUnscoredSession verifies session ID returned when unscored exists.
func TestFindLatestUnscoredSession_WithUnscoredSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := memory.InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert an embedding first
	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, ?)", "test content", "memory")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(res).ToNot(BeNil())

	if res == nil {
		t.Fatal("res is nil")
	}

	memID, err := res.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	// Insert surfacing event with haiku_relevant=true and faithfulness IS NULL
	haikuRelevant := true

	_, err = memory.LogSurfacingEvent(db, memory.SurfacingEvent{
		MemoryID:      memID,
		QueryText:     "test query",
		HookEvent:     "Stop",
		Timestamp:     time.Now(),
		SessionID:     "session-unscored-abc",
		HaikuRelevant: &haikuRelevant,
		Faithfulness:  nil, // unscored
	})
	g.Expect(err).ToNot(HaveOccurred())

	sessionID, err := memory.FindLatestUnscoredSession(db)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sessionID).To(Equal("session-unscored-abc"))
}
