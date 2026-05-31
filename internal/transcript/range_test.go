package transcript_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/transcript"
)

func TestCompositeRangeReader_AllFailReturnsLastError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	last := errors.New("last reader error")
	composite := transcript.NewCompositeRangeReader(
		fakeRangeReader{err: errors.New("first reader error")},
		fakeRangeReader{err: last},
	)

	_, err := composite.ReadRange("p", time.Time{}, time.Now())
	g.Expect(err).To(MatchError(last))
}

func TestCompositeRangeReader_NoReadersReturnsErrNoReader(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	composite := transcript.NewCompositeRangeReader()

	_, err := composite.ReadRange("p", time.Time{}, time.Now())
	g.Expect(err).To(MatchError(transcript.ErrNoReader))
}

func TestCompositeRangeReader_ReturnsFirstSuccess(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	composite := transcript.NewCompositeRangeReader(
		fakeRangeReader{err: errors.New("jsonl miss on opencode path")},
		fakeRangeReader{chunk: "second wins"},
	)

	got, err := composite.ReadRange("opencode://x", time.Time{}, time.Now())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal("second wins"))
}

func TestJSONLRangeReader_AppliesToolSummaryStrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"assistant","timestamp":"2026-05-25T22:30:00Z","message":{"role":"assistant","content":[` +
			`{"type":"text","text":"Let me check"},` +
			`{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}` +
			`]}}`,
		`{"type":"user","timestamp":"2026-05-25T22:31:00Z","message":{"role":"user","content":[` +
			`{"type":"tool_result","tool_use_id":"tu1","content":"file1.go","is_error":false}` +
			`]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		contents: map[string][]byte{"/sess.jsonl": []byte(content)},
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	result, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("Let me check"))
	g.Expect(result).To(ContainSubstring("[tool]"))
	g.Expect(result).To(ContainSubstring("Bash"))
}

func TestJSONLRangeReader_DropsLinesWithoutTimestamp(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-05-25T22:30:00Z","message":{"role":"user","content":"hello"}}`,
		// No timestamp — must be dropped even if it falls notionally inside.
		`{"type":"user","message":{"role":"user","content":"untimed"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		contents: map[string][]byte{"/sess.jsonl": []byte(content)},
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	result, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("hello"))
	g.Expect(result).NotTo(ContainSubstring("untimed"))
}

func TestJSONLRangeReader_EmptyRangeYieldsEmptyContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-05-25T22:00:00Z","message":{"role":"user","content":"too-early"}}`,
		`{"type":"user","timestamp":"2026-05-25T23:30:00Z","message":{"role":"user","content":"too-late"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		contents: map[string][]byte{"/sess.jsonl": []byte(content)},
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:15:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	result, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestJSONLRangeReader_IncludesEndpoints(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-05-25T22:00:00Z","message":{"role":"user","content":"at-start"}}`,
		`{"type":"user","timestamp":"2026-05-25T23:00:00Z","message":{"role":"user","content":"at-end"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		contents: map[string][]byte{"/sess.jsonl": []byte(content)},
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	result, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("at-start"))
	g.Expect(result).To(ContainSubstring("at-end"))
}

func TestJSONLRangeReader_ReaderErrorWraps(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		err: errors.New("permission denied"),
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	_, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).To(MatchError(ContainSubstring("permission denied")))
	g.Expect(err).To(MatchError(ContainSubstring("reading transcript")))
}

func TestJSONLRangeReader_ReadsLinesWithinRange(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-05-25T22:00:00Z","message":{"role":"user","content":"before"}}`,
		`{"type":"user","timestamp":"2026-05-25T22:30:00Z","message":{"role":"user","content":"inside-one"}}`,
		`{"type":"assistant","timestamp":"2026-05-25T22:45:00Z","message":{"role":"assistant","content":"inside-two"}}`,
		`{"type":"user","timestamp":"2026-05-25T23:30:00Z","message":{"role":"user","content":"after"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLRangeReader(&fakeFileReader{
		contents: map[string][]byte{"/sess.jsonl": []byte(content)},
	})

	start, _ := time.Parse(time.RFC3339, "2026-05-25T22:15:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-05-25T23:00:00Z")

	result, err := reader.ReadRange("/sess.jsonl", start, end)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("inside-one"))
	g.Expect(result).To(ContainSubstring("inside-two"))
	g.Expect(result).NotTo(ContainSubstring("before"))
	g.Expect(result).NotTo(ContainSubstring("after"))
}

// fakeRangeReader is a RangeReader returning a fixed chunk or error, used to
// exercise CompositeRangeReader dispatch without real Claude/OpenCode sources.
type fakeRangeReader struct {
	chunk string
	err   error
}

func (f fakeRangeReader) ReadRange(string, time.Time, time.Time) (string, error) {
	return f.chunk, f.err
}
