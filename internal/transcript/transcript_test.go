package transcript_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/transcript"
)

func TestJSONLReader_ReadFrom_EmitsRowsStrictlyAfterMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"a"}}`,
		`{"type":"user","timestamp":"2026-01-01T00:01:00Z","message":{"role":"user","content":"b"}}`,
		`{"type":"user","timestamp":"2026-01-01T00:02:00Z","message":{"role":"user","content":"c"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})
	from, err := time.Parse(time.RFC3339, "2026-01-01T00:01:00Z")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := reader.ReadFrom("/x.jsonl", from, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Content).To(ContainSubstring("USER: c"))
	g.Expect(result.Content).NotTo(ContainSubstring("USER: a"))
	g.Expect(result.Content).NotTo(ContainSubstring("USER: b"))

	expected, _ := time.Parse(time.RFC3339, "2026-01-01T00:02:00Z")
	g.Expect(result.LastTimestamp.Equal(expected)).To(BeTrue())
	g.Expect(result.Partial).To(BeFalse())
}

func TestJSONLReader_ReadFrom_HandlesUnparseableTimestampAndNonJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Row 1: non-JSON (should still pass through extractRowTimestamps without
	// crashing — its timestamp resolves to the carry value, zero).
	// Row 2: JSON but missing "timestamp" field (resolves to zero).
	// Row 3: JSON with malformed timestamp (parse error → resolves to zero).
	// Row 4: well-formed.
	lines := []string{
		`not even json`,
		`{"type":"user","message":{"role":"user","content":"a"}}`,
		`{"type":"user","timestamp":"not-a-date","message":{"role":"user","content":"b"}}`,
		`{"type":"user","timestamp":"2026-01-01T00:03:00Z","message":{"role":"user","content":"c"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	result, err := reader.ReadFrom("/x.jsonl", time.Time{}, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())

	// Row 4's timestamp wins (others all carry zero).
	expected, _ := time.Parse(time.RFC3339, "2026-01-01T00:03:00Z")
	g.Expect(result.LastTimestamp.Equal(expected)).To(BeTrue())
}

func TestJSONLReader_ReadFrom_NullTimestampRowsInheritPrior(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Row 1 has timestamp null (e.g. file-history-snapshot). Row 2 has a real
	// timestamp. Marker = zero. Expect both rows pass filter (null inherits
	// zero, zero <= ... wait actually null inherits the PRIOR row's time;
	// row 1 is first so it inherits zero; with marker=zero we DON'T filter,
	// so both rows are emitted). LastTimestamp comes from row 2.
	lines := []string{
		`{"type":"snapshot","timestamp":null}`,
		`{"type":"user","timestamp":"2026-01-01T00:01:00Z","message":{"role":"user","content":"a"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	result, err := reader.ReadFrom("/x.jsonl", time.Time{}, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())

	expected, _ := time.Parse(time.RFC3339, "2026-01-01T00:01:00Z")
	g.Expect(result.LastTimestamp.Equal(expected)).To(BeTrue())
	g.Expect(result.Partial).To(BeFalse())
}

func TestJSONLReader_ReadFrom_PartialWhenBudgetExceeded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"aaaaa"}}`,
		`{"type":"user","timestamp":"2026-01-01T00:01:00Z","message":{"role":"user","content":"bbbbb"}}`,
		`{"type":"user","timestamp":"2026-01-01T00:02:00Z","message":{"role":"user","content":"ccccc"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	// Budget tight enough to fit one stripped line ("USER: aaaaa" = 11 + \n)
	// but not a second.
	result, err := reader.ReadFrom("/x.jsonl", time.Time{}, 20)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Partial).To(BeTrue())
	g.Expect(result.Content).To(ContainSubstring("aaaaa"))
	g.Expect(result.Content).NotTo(ContainSubstring("bbbbb"))
	g.Expect(result.Content).NotTo(ContainSubstring("ccccc"))

	expected, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	g.Expect(result.LastTimestamp.Equal(expected)).To(BeTrue())
}

//go:generate impgen transcript.Finder --dependency --import-path github.com/toejough/engram/internal/transcript
//go:generate impgen transcript.Reader --dependency --import-path github.com/toejough/engram/internal/transcript

func TestSessionFinder_DeduplicatesAcrossDirectories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()
	lister := &fakeDirLister{
		entries: []transcript.FileEntry{
			{Path: "/shared/session.jsonl", Mtime: now},
		},
	}

	finder := transcript.NewSessionFinder(lister)

	entries, err := finder.Find("/claude", "/opencode")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Path).To(Equal("/shared/session.jsonl"))
}

func TestSessionFinder_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &fakeDirLister{entries: []transcript.FileEntry{}}

	finder := transcript.NewSessionFinder(lister)

	entries, err := finder.Find("/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(BeEmpty())
}

func TestSessionFinder_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &fakeDirLister{err: errors.New("access denied")}

	finder := transcript.NewSessionFinder(lister)

	_, err := finder.Find("/project")
	g.Expect(err).To(MatchError(ContainSubstring("access denied")))
	g.Expect(err).To(MatchError(ContainSubstring("listing sessions in /project")))
}

func TestSessionFinder_MultipleDirectoriesMerged(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()
	lister := &fakeDirLister{
		entries: []transcript.FileEntry{
			{Path: "/claude/a.jsonl", Mtime: now.Add(-1 * time.Hour)},
			{Path: "/claude/c.jsonl", Mtime: now.Add(-3 * time.Hour)},
			{Path: "/opencode/b.jsonl", Mtime: now.Add(-2 * time.Hour)},
		},
	}

	finder := transcript.NewSessionFinder(lister)

	entries, err := finder.Find("/claude", "/opencode")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(3))
	g.Expect(entries[0].Path).To(Equal("/claude/a.jsonl"))
	g.Expect(entries[1].Path).To(Equal("/opencode/b.jsonl"))
	g.Expect(entries[2].Path).To(Equal("/claude/c.jsonl"))
}

func TestSessionFinder_NoDirectories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &fakeDirLister{entries: []transcript.FileEntry{}}

	finder := transcript.NewSessionFinder(lister)

	entries, err := finder.Find()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

func TestSessionFinder_SortsByMtimeDescending(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()
	lister := &fakeDirLister{
		entries: []transcript.FileEntry{
			{Path: "/sessions/old.jsonl", Mtime: now.Add(-2 * time.Hour)},
			{Path: "/sessions/newest.jsonl", Mtime: now},
			{Path: "/sessions/middle.jsonl", Mtime: now.Add(-1 * time.Hour)},
		},
	}

	finder := transcript.NewSessionFinder(lister)

	entries, err := finder.Find("/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(3))
	g.Expect(entries[0].Path).To(Equal("/sessions/newest.jsonl"))
	g.Expect(entries[1].Path).To(Equal("/sessions/middle.jsonl"))
	g.Expect(entries[2].Path).To(Equal("/sessions/old.jsonl"))
}

func TestTranscriptReader_ReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := transcript.NewJSONLReader(&fakeFileReader{
		err: errors.New("file not found"),
	})

	const budget = 10000

	_, _, err := readAll(reader, "/missing.jsonl", budget)
	g.Expect(err).To(MatchError(ContainSubstring("file not found")))
	g.Expect(err).To(MatchError(ContainSubstring("reading transcript")))
}

func TestTranscriptReader_RespectsBudget(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := make([]string, 0, 20)
	for i := range 20 {
		lines = append(
			lines,
			fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"message %d"}}`, i),
		)
	}

	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const tinyBudget = 50

	result, bytesRead, err := readAll(reader, "/transcript.jsonl", tinyBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(bytesRead).To(BeNumerically(">", 0))
	g.Expect(len(result)).To(BeNumerically("<", len(content)))

	// ReadFrom emits chronologically from the oldest rows forward, halting
	// when the budget would be exceeded. With marker advancing forward,
	// the early messages are the ones that fit; later messages are picked
	// up on subsequent runs.
	g.Expect(result).To(ContainSubstring("message 0"), "should contain earliest message")
	g.Expect(result).NotTo(ContainSubstring("message 19"), "should not contain latest message")
}

func TestTranscriptReader_ReturnsLeadingRowsWhenBudgetLimited(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"early message alpha"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"early response beta"}}`,
		`{"type":"user","message":{"role":"user","content":"middle message gamma"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"middle response delta"}}`,
		`{"type":"user","message":{"role":"user","content":"late message epsilon"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"late response zeta"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const limitedBudget = 80

	result, _, err := readAll(reader, "/transcript.jsonl", limitedBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("alpha"), "should contain earliest content")
	g.Expect(result).NotTo(ContainSubstring("zeta"), "should not contain latest content")
}

func TestTranscriptReader_StripsToolResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"role":"toolResult","content":"big result data"}`,
		`{"type":"assistant","message":{"role":"assistant","content":"response"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const largeBudget = 10000

	result, bytesRead, err := readAll(reader, "/transcript.jsonl", largeBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("USER: hello"))
	g.Expect(result).To(ContainSubstring("ASSISTANT: response"))
	g.Expect(result).NotTo(ContainSubstring("toolResult"))
	g.Expect(bytesRead).To(BeNumerically(">", 0))
}

func TestTranscriptReader_ToolSummaryMode(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[` +
			`{"type":"text","text":"Let me check"},` +
			`{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}` +
			`]}}`,
		`{"type":"user","message":{"role":"user","content":[` +
			`{"type":"tool_result","tool_use_id":"tu1","content":"file1.go\nfile2.go","is_error":false}` +
			`]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const largeBudget = 10000

	result, _, err := readAll(reader, "/transcript.jsonl", largeBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("[tool]"), "should contain tool summary line")
	g.Expect(result).To(ContainSubstring("Bash"), "should contain tool name")
	g.Expect(result).To(ContainSubstring("Let me check"), "should contain assistant text")
}

// --- Fake implementations ---

type fakeDirLister struct {
	entries []transcript.FileEntry
	err     error
}

func (f *fakeDirLister) ListJSONL(_ string) ([]transcript.FileEntry, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.entries, nil
}

type fakeFileReader struct {
	contents map[string][]byte
	err      error
}

func (f *fakeFileReader) Read(path string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}

	content, ok := f.contents[path]
	if !ok {
		return nil, fmt.Errorf("open %s: no such file or directory", path)
	}

	return content, nil
}

// readAll wraps Reader.ReadFrom with zero fromTime to match the legacy
// Read contract for existing tests.
func readAll(
	r interface {
		ReadFrom(path string, fromTime time.Time, budgetBytes int) (transcript.ReadResult, error)
	},
	path string, budget int,
) (string, int, error) {
	res, err := r.ReadFrom(path, time.Time{}, budget)
	return res.Content, res.BytesUsed, err
}
