package recall_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/recall"
)

func TestSessionFinder_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &fakeDirLister{entries: []recall.FileEntry{}}

	finder := recall.NewSessionFinder(lister)

	paths, err := finder.Find("/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paths).To(BeEmpty())
}

func TestSessionFinder_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &fakeDirLister{err: errors.New("access denied")}

	finder := recall.NewSessionFinder(lister)

	_, err := finder.Find("/project")
	g.Expect(err).To(MatchError(ContainSubstring("access denied")))
	g.Expect(err).To(MatchError(ContainSubstring("listing sessions")))
}

// --- SessionFinder tests ---

func TestSessionFinder_SortsByMtimeDescending(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Now()
	lister := &fakeDirLister{
		entries: []recall.FileEntry{
			{Path: "/sessions/old.jsonl", Mtime: now.Add(-2 * time.Hour)},
			{Path: "/sessions/newest.jsonl", Mtime: now},
			{Path: "/sessions/middle.jsonl", Mtime: now.Add(-1 * time.Hour)},
		},
	}

	finder := recall.NewSessionFinder(lister)

	paths, err := finder.Find("/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paths).To(Equal([]string{
		"/sessions/newest.jsonl",
		"/sessions/middle.jsonl",
		"/sessions/old.jsonl",
	}))
}

func TestTranscriptReader_ReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := recall.NewTranscriptReader(&fakeFileReader{
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

	// Create content that exceeds a small budget.
	lines := make([]string, 0, 20)
	for i := range 20 {
		lines = append(lines, fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"message %d"}}`, i))
	}

	content := strings.Join(lines, "\n") + "\n"

	reader := recall.NewTranscriptReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const tinyBudget = 50

	result, bytesRead, err := reader.Read("/transcript.jsonl", tinyBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should have stopped accumulating after exceeding budget.
	g.Expect(bytesRead).To(BeNumerically(">=", tinyBudget))
	g.Expect(len(result)).To(BeNumerically("<", len(content)))
}

// --- TranscriptReader tests ---

func TestTranscriptReader_StripsToolResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"role":"toolResult","content":"big result data"}`,
		`{"type":"assistant","message":{"role":"assistant","content":"response"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := recall.NewTranscriptReader(&fakeFileReader{
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

// --- Fake implementations ---

type fakeDirLister struct {
	entries []recall.FileEntry
	err     error
}

func (f *fakeDirLister) ListJSONL(_ string) ([]recall.FileEntry, error) {
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
