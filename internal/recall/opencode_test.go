package recall_test

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/recall"

	_ "modernc.org/sqlite"
)

func TestCompositeSessionFinder_MergesFinders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_oc1", Title: "OpenCode session", Updated: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)},
	})

	opencodeFinder := recall.NewOpencodeSessionFinder(dbPath)
	claudeFinder := recall.NewSessionFinder(&fakeDirLister{
		entries: []recall.FileEntry{
			{Path: "/claude/ses_cc1.jsonl", Mtime: time.Date(2026, 5, 2, 11, 0, 0, 0, time.UTC)},
		},
	})

	composite := recall.NewCompositeSessionFinder(claudeFinder, opencodeFinder)

	entries, err := composite.Find("/claude")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Path).To(Equal("opencode://ses_oc1"))
	g.Expect(entries[1].Path).To(Equal("/claude/ses_cc1.jsonl"))
}

func TestCompositeTranscriptReader_TriesReaders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_comp1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_comp1", []testPart{
		{Type: "text", Text: "Composite test content", TimeCreated: 1},
	})

	opencodeReader := recall.NewOpencodeTranscriptReader(dbPath)
	fileReader := recall.NewTranscriptReader(&fakeFileReader{})

	composite := recall.NewCompositeTranscriptReader(fileReader, opencodeReader)

	content, size, err := composite.Read("opencode://ses_comp1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("Composite test content"))
}

func TestOpencodeSessionFinder_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, nil)

	finder := recall.NewOpencodeSessionFinder(dbPath)

	entries, err := finder.Find()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

func TestOpencodeSessionFinder_NonexistentDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	finder := recall.NewOpencodeSessionFinder("/nonexistent/path/opencode.db")

	_, err := finder.Find()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("querying sessions"))
}

func TestOpencodeSessionFinder_ReturnsEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_abc123", Title: "Test session 1", Updated: time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)},
		{ID: "ses_def456", Title: "Test session 2", Updated: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)},
	})

	finder := recall.NewOpencodeSessionFinder(dbPath)

	entries, err := finder.Find()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Path).To(Equal("opencode://ses_def456"))
	g.Expect(entries[1].Path).To(Equal("opencode://ses_abc123"))
}

func TestOpencodeTranscriptReader_EmptySessionID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := recall.NewOpencodeTranscriptReader("")

	_, _, err := reader.Read("opencode://", 1024)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty opencode session ID"))
}

func TestOpencodeTranscriptReader_IgnoresNonTextParts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_skip1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_skip1", []testPart{
		{Type: "step-start", TimeCreated: 1},
		{Type: "reasoning", Text: "thinking...", TimeCreated: 2},
		{Type: "step-finish", TimeCreated: 3},
		{Type: "text", Text: "Actual content", TimeCreated: 4},
	})

	reader := recall.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://ses_skip1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("Actual content"))
	g.Expect(content).NotTo(ContainSubstring("thinking"))
}

func TestOpencodeTranscriptReader_NonexistentSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, nil)
	reader := recall.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://nonexistent", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
	g.Expect(size).To(Equal(0))
}

func TestOpencodeTranscriptReader_ReadsTextParts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_test1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_test1", []testPart{
		{Type: "text", Text: "Hello from user", TimeCreated: 1},
		{Type: "text", Text: "Hello from assistant", TimeCreated: 2},
	})

	reader := recall.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://ses_test1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("Hello from user"))
	g.Expect(content).To(ContainSubstring("Hello from assistant"))
}

func TestOpencodeTranscriptReader_ReadsToolParts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_tool1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_tool1", []testPart{
		{
			Type:        "tool",
			Tool:        "bash",
			State:       `{"status":"completed","input":{"command":"ls"},"output":"file.txt"}`,
			TimeCreated: 1,
		},
	})

	reader := recall.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://ses_tool1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("bash"))
}

type testPart struct {
	Type        string
	Text        string
	Tool        string
	State       string
	TimeCreated int64
}

type testSession struct {
	ID      string
	Title   string
	Updated time.Time
}

func (s testSession) UnixMilli() int64 {
	return s.Updated.UnixMilli()
}

func createTestOpencodeDB(t *testing.T, sessions []testSession) string {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/opencode.db"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	defer db.Close() //nolint:errcheck

	_, err = db.Exec(`
		CREATE TABLE session (
			id text PRIMARY KEY,
			project_id text NOT NULL,
			slug text NOT NULL,
			directory text NOT NULL,
			title text NOT NULL,
			time_created integer NOT NULL,
			time_updated integer NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("creating session table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE part (
			id text PRIMARY KEY,
			message_id text NOT NULL,
			session_id text NOT NULL,
			time_created integer NOT NULL,
			time_updated integer NOT NULL,
			data text NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("creating part table: %v", err)
	}

	for i, s := range sessions {
		updatedAt := s.Updated.UnixMilli()
		slug := "slug-" + strconv.Itoa(i)

		_, err = db.Exec(
			"INSERT INTO session "+
				"(id, project_id, slug, directory, title, time_created, time_updated) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			s.ID, "proj_test", slug, "/test/dir", s.Title, updatedAt, updatedAt,
		)
		if err != nil {
			t.Fatalf("inserting session: %v", err)
		}
	}

	return dbPath
}

func insertParts(t *testing.T, dbPath string, sessionID string, parts []testPart) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	defer db.Close() //nolint:errcheck

	for i, p := range parts {
		data := `{"type":"` + p.Type + `"`
		if p.Text != "" {
			data += `,"text":` + fmt.Sprintf("%q", p.Text)
		}

		if p.Tool != "" {
			data += `,"tool":` + fmt.Sprintf("%q", p.Tool)
		}

		if p.State != "" {
			data += `,"state":` + p.State
		}

		data += "}"

		partID := "prt_" + strconv.Itoa(i)
		msgID := "msg_" + strconv.Itoa(i)

		updatedAt := time.Now().UnixMilli()

		_, err = db.Exec(
			"INSERT INTO part "+
				"(id, message_id, session_id, time_created, time_updated, data) "+
				"VALUES (?, ?, ?, ?, ?, ?)",
			partID, msgID, sessionID, p.TimeCreated, updatedAt, data,
		)
		if err != nil {
			t.Fatalf("inserting part: %v", err)
		}
	}
}
