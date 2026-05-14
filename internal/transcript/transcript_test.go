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

	_, _, err := reader.Read("/missing.jsonl", budget)
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

	result, bytesRead, err := reader.Read("/transcript.jsonl", tinyBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(bytesRead).To(BeNumerically(">", 0))
	g.Expect(len(result)).To(BeNumerically("<", len(content)))

	g.Expect(result).To(ContainSubstring("message 19"), "should contain last message")
	g.Expect(result).NotTo(ContainSubstring("message 0"), "should not contain first message")
}

func TestTranscriptReader_ReturnsTailWhenBudgetLimited(t *testing.T) {
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

	result, _, err := reader.Read("/transcript.jsonl", limitedBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("zeta"), "should contain latest content")
	g.Expect(result).NotTo(ContainSubstring("alpha"), "should not contain earliest content")
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

	result, bytesRead, err := reader.Read("/transcript.jsonl", largeBudget)
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

	result, _, err := reader.Read("/transcript.jsonl", largeBudget)
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
