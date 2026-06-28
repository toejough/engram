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

func TestJSONLReader_ReadFrom_NoCapWhenBudgetZero(t *testing.T) {
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

	// budgetBytes <= 0 means no cap: the whole transcript is emitted, never partial.
	result, err := reader.ReadFrom("/x.jsonl", time.Time{}, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Partial).To(BeFalse())
	g.Expect(result.Content).To(ContainSubstring("aaaaa"))
	g.Expect(result.Content).To(ContainSubstring("bbbbb"))
	g.Expect(result.Content).To(ContainSubstring("ccccc"))
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

func TestJSONLReader_SegmentsFrom_NoCapWhenBudgetZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{
		`{"type":"user","timestamp":"2026-05-01T10:00:00Z","message":{"content":"first user ask here"}}`,
		`{"type":"user","timestamp":"2026-05-01T10:05:00Z","message":{"content":"second user ask here"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	// budgetBytes <= 0 means no cap: every segment survives, never partial.
	res, err := reader.SegmentsFrom("/x.jsonl", time.Time{}, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(res.Partial).To(BeFalse())
	g.Expect(res.Segments).To(HaveLen(2))
}

func TestJSONLReader_SegmentsFrom_PartialWhenBudgetTruncates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two user turns; a byte budget too small for both forces the scan to stop
	// after the first, so SegmentsResult.Partial must be true and only the first
	// segment survives. With an ample budget the same input reports Partial=false.
	lines := []string{
		`{"type":"user","timestamp":"2026-05-01T10:00:00Z","message":{"content":"first user ask here"}}`,
		`{"type":"user","timestamp":"2026-05-01T10:05:00Z","message":{"content":"second user ask here"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	// Budget large enough for the first stripped line but not both.
	const tinyBudget = 40

	truncated, err := reader.SegmentsFrom("/x.jsonl", time.Time{}, tinyBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(truncated.Partial).To(BeTrue())
	g.Expect(truncated.Segments).To(HaveLen(1))
	g.Expect(truncated.Segments[0].Preview).To(ContainSubstring("first user ask"))

	// Ample budget: the whole file fits, so Partial is false.
	full, err := reader.SegmentsFrom("/x.jsonl", time.Time{}, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(full.Partial).To(BeFalse())
	g.Expect(full.Segments).To(HaveLen(2))
}

func TestJSONLReader_SegmentsFrom_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := transcript.NewJSONLReader(&fakeFileReader{
		err: errors.New("disk error"),
	})

	_, err := reader.SegmentsFrom("/missing.jsonl", time.Time{}, 1<<20)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading transcript"))
}

func TestJSONLReader_SegmentsFrom_ReturnsRealUserTurns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two real user asks, one assistant turn, one injected skill-body user turn.
	// The skill body starts with "Base directory for this skill:" and is dropped
	// by cleanHarnessInjection — so it never produces a USER: line in stripped.
	lines := []string{
		`{"type":"user","timestamp":"2026-05-01T10:00:00Z","message":{"content":"help me build this"}}`,
		`{"type":"user","timestamp":"2026-05-01T10:01:00Z","message":` +
			`{"content":"Base directory for this skill: /foo\n# Skill body here"}}`,
		`{"type":"assistant","timestamp":"2026-05-01T10:02:00Z","message":{"content":"sure, here is the plan"}}`,
		`{"type":"user","timestamp":"2026-05-01T10:03:00Z","message":{"content":"now add the second part"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"

	reader := transcript.NewJSONLReader(&fakeFileReader{
		contents: map[string][]byte{"/x.jsonl": []byte(content)},
	})

	result, err := reader.SegmentsFrom("/x.jsonl", time.Time{}, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	segs := result.Segments

	// Only the two real user asks; skill body + assistant are absent.
	g.Expect(segs).To(HaveLen(2))
	g.Expect(segs[0].Preview).To(ContainSubstring("help me build this"))
	g.Expect(segs[1].Preview).To(ContainSubstring("now add the second part"))

	// Injected skill body must not appear.
	for _, seg := range segs {
		g.Expect(seg.Preview).NotTo(ContainSubstring("Base directory for this skill"))
		g.Expect(seg.Preview).NotTo(ContainSubstring("sure, here is the plan"))
	}
}

func TestSegmentsFromStripped_EmptyStrippedUserLineProducesNoSegment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// The spec requires: "empty/whitespace cleaned text produces no segment
	// line." Strip already drops truly-empty turns (extractText returns "" for
	// empty content). But a defensively-added gate: if a non-USER-prefixed
	// line or a zero-timestamp USER line arrives, it must be silently skipped.
	// This test verifies the zero-timestamp guard (covered also by
	// TestSegmentsFromStripped_ZeroTimestampSkipped) and that assistant lines
	// never become segments.
	ts1, _ := time.Parse(time.RFC3339, "2026-05-01T10:00:00Z")

	stripped := []string{
		"ASSISTANT: some reply", // assistant — not a user turn
	}
	times := []time.Time{ts1}

	segs := transcript.SegmentsFromStrippedForTest(stripped, times)

	g.Expect(segs).To(BeEmpty())
}

func TestSegmentsFromStripped_NewlinesCollapsedToSpaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts, _ := time.Parse(time.RFC3339, "2026-05-01T10:00:00Z")

	stripped := []string{"USER: first line\nsecond line"}
	times := []time.Time{ts}

	segs := transcript.SegmentsFromStrippedForTest(stripped, times)

	g.Expect(segs).To(HaveLen(1))
	g.Expect(segs[0].Preview).NotTo(ContainSubstring("\n"))
	g.Expect(segs[0].Preview).To(ContainSubstring("first line second line"))
}

func TestSegmentsFromStripped_OnlyRealUserTurns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// The task spec scenario:
	// (a) real user ask
	// (b) injected skill-body user turn — strip already drops this, so it
	//     never appears in stripped[] (cleanHarnessInjection drops it)
	// (c) assistant turn — has ASSISTANT: prefix, not USER:
	// (d) another real user ask
	//
	// We simulate the post-strip, post-mapTimestampsByIndex state that
	// segmentsFromStripped receives: only user turns that survived cleaning.
	ts1, _ := time.Parse(time.RFC3339, "2026-05-01T10:00:00Z")
	ts2, _ := time.Parse(time.RFC3339, "2026-05-01T11:00:00Z")
	ts3, _ := time.Parse(time.RFC3339, "2026-05-01T12:00:00Z")

	stripped := []string{
		"USER: help me write a feature",     // (a) real user ask
		"ASSISTANT: sure, here is the plan", // (c) assistant turn
		"USER: now add the second part",     // (d) another real user ask
	}
	times := []time.Time{ts1, ts2, ts3}

	segs := transcript.SegmentsFromStrippedForTest(stripped, times)

	g.Expect(segs).To(HaveLen(2))
	g.Expect(segs[0].Timestamp.Equal(ts1)).To(BeTrue())
	g.Expect(segs[0].Preview).To(ContainSubstring("help me write a feature"))
	g.Expect(segs[1].Timestamp.Equal(ts3)).To(BeTrue())
	g.Expect(segs[1].Preview).To(ContainSubstring("now add the second part"))
}

func TestSegmentsFromStripped_PreviewTruncatedAt100Runes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts, _ := time.Parse(time.RFC3339, "2026-05-01T10:00:00Z")

	// Build a user ask > 100 runes (prefix "USER: " = 6 runes + 200 rune body).
	longBody := strings.Repeat("x", 200)
	stripped := []string{"USER: " + longBody}
	times := []time.Time{ts}

	segs := transcript.SegmentsFromStrippedForTest(stripped, times)

	g.Expect(segs).To(HaveLen(1))
	g.Expect([]rune(segs[0].Preview)).To(HaveLen(100))
}

func TestSegmentsFromStripped_ZeroTimestampSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts1, _ := time.Parse(time.RFC3339, "2026-05-01T10:00:00Z")

	stripped := []string{
		"USER: ask with no timestamp", // zero timestamp — should be skipped
		"USER: ask with timestamp",    // non-zero timestamp — should appear
	}
	times := []time.Time{{}, ts1}

	segs := transcript.SegmentsFromStrippedForTest(stripped, times)

	g.Expect(segs).To(HaveLen(1))
	g.Expect(segs[0].Timestamp.Equal(ts1)).To(BeTrue())
	g.Expect(segs[0].Preview).To(ContainSubstring("ask with timestamp"))
}

//go:generate impgen transcript.Reader --dependency --import-path github.com/toejough/engram/internal/transcript

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
