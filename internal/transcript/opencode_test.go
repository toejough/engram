package transcript_test

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"github.com/toejough/engram/internal/transcript"
)

func TestCompositeSessionFinder_MergesFinders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{
			ID:      "ses_oc1",
			Title:   "OpenCode session",
			Updated: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
	})

	opencodeFinder := transcript.NewOpencodeSessionFinder(dbPath, "")
	claudeFinder := transcript.NewSessionFinder(&fakeDirLister{
		entries: []transcript.FileEntry{
			{Path: "/claude/ses_cc1.jsonl", Mtime: time.Date(2026, 5, 2, 11, 0, 0, 0, time.UTC)},
		},
	})

	composite := transcript.NewCompositeSessionFinder(claudeFinder, opencodeFinder)

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

	opencodeReader := transcript.NewOpencodeTranscriptReader(dbPath)
	fileReader := transcript.NewJSONLReader(&fakeFileReader{})

	composite := transcript.NewCompositeTranscriptReader(fileReader, opencodeReader)

	content, size, err := composite.Read("opencode://ses_comp1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("Composite test content"))
}

func TestDefaultOpencodeDBPath_ErrorsWhenNoHome(t *testing.T) {
	// No t.Parallel() — t.Setenv modifies a shared environment variable.
	g := NewWithT(t)

	t.Setenv("HOME", "")

	got := transcript.DefaultOpencodeDBPath()
	g.Expect(got).To(BeEmpty())
}

func TestDefaultOpencodeDBPath_UsesHome(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got := transcript.DefaultOpencodeDBPath()
	g.Expect(got).To(HaveSuffix("/.local/share/opencode/opencode.db"))
}

func TestMustMarshalJSON_MarshalableValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got := transcript.MustMarshalJSON("hello world")
	g.Expect(got).To(Equal(`"hello world"`))
}

func TestMustMarshalJSON_UnmarshalableValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Channels cannot be marshalled to JSON; json.Marshal returns an error.
	got := transcript.MustMarshalJSON(make(chan int))
	g.Expect(got).To(Equal(`""`))
}

func TestOpencodeSessionFinder_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, nil)

	finder := transcript.NewOpencodeSessionFinder(dbPath, "")

	entries, err := finder.Find()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

func TestOpencodeSessionFinder_FiltersByCwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{
			ID:        "ses_proj_root",
			Directory: "/projects/foo",
			Updated:   time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        "ses_proj_sub",
			Directory: "/projects/foo/sub",
			Updated:   time.Date(2026, 5, 2, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:        "ses_other_proj",
			Directory: "/projects/bar",
			Updated:   time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:        "ses_prefix_collide",
			Directory: "/projects/foobar",
			Updated:   time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC),
		},
	})

	finder := transcript.NewOpencodeSessionFinder(dbPath, "/projects/foo")

	entries, err := finder.Find()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}

	g.Expect(paths).To(ConsistOf("opencode://ses_proj_root", "opencode://ses_proj_sub"))
}

func TestOpencodeSessionFinder_NonexistentDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	finder := transcript.NewOpencodeSessionFinder("/nonexistent/path/opencode.db", "")

	_, err := finder.Find()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("querying sessions"))
}

func TestOpencodeSessionFinder_ReturnsEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{
			ID:      "ses_abc123",
			Title:   "Test session 1",
			Updated: time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:      "ses_def456",
			Title:   "Test session 2",
			Updated: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
	})

	finder := transcript.NewOpencodeSessionFinder(dbPath, "")

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

	reader := transcript.NewOpencodeTranscriptReader("")

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

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://ses_skip1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("Actual content"))
	g.Expect(content).NotTo(ContainSubstring("thinking"))
}

func TestOpencodeTranscriptReader_InvalidToolState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_bad1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_bad1", []testPart{
		{Type: "tool", Tool: "bash", State: `"plain string, not an object"`, TimeCreated: 1},
	})

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, _, err := reader.Read("opencode://ses_bad1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
}

func TestOpencodeTranscriptReader_NonexistentSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, nil)
	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://nonexistent", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
	g.Expect(size).To(Equal(0))
}

func TestOpencodeTranscriptReader_NullPartType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_null1", Title: "Test", Updated: time.Now()},
	})

	db, err := sql.Open("sqlite", dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer db.Close() //nolint:errcheck

	now := time.Now().UnixMilli()
	_, err = db.Exec(
		"INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		"prt_nulltype",
		"msg_nulltype",
		"ses_null1",
		now,
		now,
		`{"text":"orphan text without type"}`,
	)
	g.Expect(err).NotTo(HaveOccurred())

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, _, err := reader.Read("opencode://ses_null1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
}

func TestOpencodeTranscriptReader_PreservesMessageRole(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_role1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_role1", []testPart{
		{Type: "text", Text: "user said this", Role: "user", TimeCreated: 1},
		{Type: "text", Text: "assistant replied", Role: "assistant", TimeCreated: 2},
		{Type: "text", Text: "user said that", Role: "user", TimeCreated: 3},
	})

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, _, err := reader.Read("opencode://ses_role1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(ContainSubstring("USER: user said this"))
	g.Expect(content).To(ContainSubstring("ASSISTANT: assistant replied"))
	g.Expect(content).To(ContainSubstring("USER: user said that"))
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

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

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

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, size, err := reader.Read("opencode://ses_tool1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(content).To(ContainSubstring("bash"))
}

func TestOpencodeTranscriptReader_ToolMissingStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_nostatus1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_nostatus1", []testPart{
		{Type: "tool", Tool: "bash", State: `{"input":{"command":"ls"}}`, TimeCreated: 1},
	})

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, _, err := reader.Read("opencode://ses_nostatus1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
}

func TestOpencodeTranscriptReader_UnknownPartType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dbPath := createTestOpencodeDB(t, []testSession{
		{ID: "ses_unk1", Title: "Test", Updated: time.Now()},
	})
	insertParts(t, dbPath, "ses_unk1", []testPart{
		{Type: "unknown-part-kind", Text: "should be ignored", TimeCreated: 1},
	})

	reader := transcript.NewOpencodeTranscriptReader(dbPath)

	content, _, err := reader.Read("opencode://ses_unk1", 1024*50)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).To(BeEmpty())
}

type testPart struct {
	Type        string
	Text        string
	Tool        string
	State       string
	Role        string
	TimeCreated int64
}

type testSession struct {
	ID        string
	Title     string
	Directory string
	Updated   time.Time
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

	_, err = db.Exec(`
		CREATE TABLE message (
			id text PRIMARY KEY,
			session_id text NOT NULL,
			time_created integer NOT NULL,
			time_updated integer NOT NULL,
			data text NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("creating message table: %v", err)
	}

	for i, s := range sessions {
		updatedAt := s.Updated.UnixMilli()
		slug := "slug-" + strconv.Itoa(i)
		directory := s.Directory

		if directory == "" {
			directory = "/test/dir"
		}

		_, err = db.Exec(
			"INSERT INTO session "+
				"(id, project_id, slug, directory, title, time_created, time_updated) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			s.ID, "proj_test", slug, directory, s.Title, updatedAt, updatedAt,
		)
		if err != nil {
			t.Fatalf("inserting session: %v", err)
		}
	}

	return dbPath
}

func insertParts(t *testing.T, dbPath, sessionID string, parts []testPart) {
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

		role := p.Role
		if role == "" {
			role = "user"
		}

		updatedAt := time.Now().UnixMilli()
		msgData := fmt.Sprintf(`{"role":%q}`, role)

		_, err = db.Exec(
			"INSERT INTO message "+
				"(id, session_id, time_created, time_updated, data) "+
				"VALUES (?, ?, ?, ?, ?)",
			msgID, sessionID, p.TimeCreated, updatedAt, msgData,
		)
		if err != nil {
			t.Fatalf("inserting message: %v", err)
		}

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
