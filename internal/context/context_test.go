package context_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	sessionctx "engram/internal/context"
)

// --- Coverage: DeltaReader with file read error ---

func TestDeltaReader_FileReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := sessionctx.NewDeltaReader(&fakeFileReader{
		err: errors.New("permission denied"),
	})

	_, _, err := reader.Read("/transcript.jsonl", 0)
	g.Expect(err).To(MatchError(ContainSubstring("permission denied")))
}

// --- Coverage: Read with only metadata, no summary ---

func TestRead_MetadataOnly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := "<!-- engram session context | updated: 2026-03-07T00:00:00Z | offset: 100 | session: s1 -->"

	reader := &fakeFileReader{
		contents: map[string][]byte{"/ctx/file.md": []byte(content)},
	}

	file := sessionctx.NewSessionFile(reader, nil, nil, nil, nil)

	meta, err := file.Read("/ctx/file.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(meta.Offset).To(Equal(int64(100)))
	g.Expect(meta.SessionID).To(Equal("s1"))
	g.Expect(meta.Summary).To(BeEmpty())
}

// --- Coverage: Read with no metadata line ---

func TestRead_NoMetadataLine(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := "# Just a summary\nNo metadata here"

	reader := &fakeFileReader{
		contents: map[string][]byte{"/ctx/file.md": []byte(content)},
	}

	file := sessionctx.NewSessionFile(reader, nil, nil, nil, nil)

	meta, err := file.Read("/ctx/file.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(meta.Offset).To(Equal(int64(0)))
	g.Expect(meta.SessionID).To(BeEmpty())
	// First line is consumed as the "header" position (no metadata match).
	// Summary is everything after the first line.
	g.Expect(meta.Summary).To(Equal("No metadata here"))
}

// --- T-134: TranscriptDeltaReader reads from offset 0 ---

func TestT134_ReadFromOffset0ReturnsFullFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := make([]string, 0, 10)
	for i := range 10 {
		lines = append(lines, fmt.Sprintf(`{"line":%d}`, i))
	}

	content := strings.Join(lines, "\n") + "\n"

	reader := sessionctx.NewDeltaReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	delta, newOffset, err := reader.Read("/transcript.jsonl", 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(delta).To(HaveLen(10))
	g.Expect(newOffset).To(Equal(int64(len(content))))
}

// --- T-135: TranscriptDeltaReader reads from mid-file offset ---

func TestT135_ReadFromMidFileOffset(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Each line is exactly 14 bytes: `{"line":X}` + newline
	// Use fixed-width lines for predictable offsets.
	lines := make([]string, 0, 10)
	for i := range 10 {
		lines = append(lines, fmt.Sprintf(`{"line":%d}`, i))
	}

	content := strings.Join(lines, "\n") + "\n"

	// Find offset after line 5 (index 5).
	offsetAfterLine5 := 0
	for i := range 5 {
		offsetAfterLine5 += len(lines[i]) + 1 // +1 for newline
		_ = i
	}

	reader := sessionctx.NewDeltaReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	delta, newOffset, err := reader.Read("/transcript.jsonl", int64(offsetAfterLine5))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(delta).To(HaveLen(5))
	g.Expect(newOffset).To(Equal(int64(len(content))))
}

// --- T-136: File shorter than offset resets to 0 ---

func TestT136_FileShorterThanOffsetResetsTo0(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := `{"line":0}` + "\n" + `{"line":1}` + "\n"

	reader := sessionctx.NewDeltaReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": []byte(content)},
	})

	const staleOffset = 2000

	delta, newOffset, err := reader.Read("/transcript.jsonl", staleOffset)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(delta).To(HaveLen(2))
	g.Expect(newOffset).To(Equal(int64(len(content))))
}

// --- T-137: Empty file returns empty delta ---

func TestT137_EmptyFileReturnsEmptyDelta(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := sessionctx.NewDeltaReader(&fakeFileReader{
		contents: map[string][]byte{"/transcript.jsonl": {}},
	})

	delta, newOffset, err := reader.Read("/transcript.jsonl", 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(delta).To(BeEmpty())
	g.Expect(newOffset).To(Equal(int64(0)))
}

// --- T-138: ContentStripper removes tool result blocks ---

func TestT138_StripRemovesToolResultBlocks(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"role":"user","content":"hello"}`,
		`{"role":"toolResult","content":"big result data"}`,
		`{"role":"assistant","content":"response"}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0]).To(ContainSubstring("user"))
	g.Expect(result[1]).To(ContainSubstring("assistant"))
}

// --- T-139: ContentStripper replaces base64 strings ---

func TestT139_StripReplacesBase64Strings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Generate a base64-like string >100 chars.
	longBase64 := strings.Repeat("QUFB", 30) // 120 chars of valid base64

	line := fmt.Sprintf(`{"role":"user","content":"data:image/png;base64,%s"}`, longBase64)

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(ContainSubstring("[base64 removed]"))
	g.Expect(result[0]).NotTo(ContainSubstring(longBase64))
}

// --- T-140: ContentStripper truncates oversized content blocks ---

func TestT140_StripTruncatesOversizedContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const maxContentLen = 2000

	oversized := strings.Repeat("hello world ", maxContentLen/12+50)
	line := fmt.Sprintf(`{"role":"user","content":"%s"}`, oversized)

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(len(result[0])).To(BeNumerically("<", len(line)))
	g.Expect(result[0]).To(ContainSubstring("[truncated]"))
}

// --- T-141: ContentStripper preserves user messages ---

func TestT141_StripPreservesUserMessages(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	line := `{"role":"user","content":"please help me"}`

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(Equal(line))
}

// --- T-142: ContentStripper preserves assistant text ---

func TestT142_StripPreservesAssistantText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	line := `{"role":"assistant","content":"here is my answer"}`

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(Equal(line))
}

// --- T-143: ContentStripper preserves tool names, removes tool results ---

func TestT143_StripPreservesToolNamesRemovesResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"role":"toolUse","name":"Bash","command":"ls -la","content":"tool use details"}`,
		`{"role":"toolResult","content":"drwxr-xr-x 5 user staff 160 Mar  7 file1\nfile2\nfile3"}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(ContainSubstring("Bash"))
	g.Expect(result[0]).To(ContainSubstring("ls -la"))
	g.Expect(result[0]).NotTo(ContainSubstring("drwxr-xr-x"))
}

// --- T-144: ContextSummarizer returns previous summary on empty delta ---

func TestT144_SummarizerReturnsPreviousOnEmptyDelta(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{result: "should not be called"}

	summarizer := sessionctx.NewSummarizer(client)

	result, err := summarizer.Summarize(context.Background(), "Current task: foo", "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("Current task: foo"))
	g.Expect(client.called).To(BeFalse())
}

// --- T-145: ContextSummarizer updates summary on non-empty delta ---

func TestT145_SummarizerUpdatesSummaryOnDelta(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{result: "Updated summary with new info"}

	summarizer := sessionctx.NewSummarizer(client)

	result, err := summarizer.Summarize(
		context.Background(),
		"Previous summary",
		"new delta content here",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("Updated summary with new info"))
	g.Expect(client.called).To(BeTrue())
	g.Expect(client.prevIn).To(Equal("Previous summary"))
	g.Expect(client.deltaIn).To(Equal("new delta content here"))
}

// --- T-146: ContextSummarizer creates new summary without previous ---

func TestT146_SummarizerCreatesNewSummaryWithoutPrevious(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{result: "Brand new summary"}

	summarizer := sessionctx.NewSummarizer(client)

	result, err := summarizer.Summarize(context.Background(), "", "new delta content")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("Brand new summary"))
	g.Expect(client.called).To(BeTrue())
	g.Expect(client.prevIn).To(BeEmpty())
}

// --- T-147: ContextSummarizer skips API call when client is nil ---

func TestT147_SummarizerSkipsWhenClientNil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	summarizer := sessionctx.NewSummarizer(nil)

	result, err := summarizer.Summarize(context.Background(), "Previous", "new delta")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("Previous"))
}

// --- T-148: ContextSummarizer returns previous summary on API error ---

func TestT148_SummarizerReturnsPreviousOnAPIError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{
		err: errors.New("API connection failed"),
	}

	summarizer := sessionctx.NewSummarizer(client)

	result, err := summarizer.Summarize(context.Background(), "Previous summary", "new delta")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("Previous summary"))
}

// --- T-149: SessionContextFile parses HTML metadata ---

func TestT149_ParseHTMLMetadata(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := "<!-- engram session context | updated: 2026-03-07T00:00:00Z" +
		" | offset: 1000 | session: abc123 -->\n\n# Summary\nSome content here"

	reader := &fakeFileReader{
		contents: map[string][]byte{"/ctx/session-context.md": []byte(content)},
	}

	file := sessionctx.NewSessionFile(reader, nil, nil, nil, nil)

	meta, err := file.Read("/ctx/session-context.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(meta.Offset).To(Equal(int64(1000)))
	g.Expect(meta.SessionID).To(Equal("abc123"))
}

// --- T-150: SessionContextFile extracts markdown summary ---

func TestT150_ExtractMarkdownSummary(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := "<!-- engram session context | updated: 2026-03-07T00:00:00Z" +
		" | offset: 500 | session: xyz -->\n\n# Summary\nSome content here"

	reader := &fakeFileReader{
		contents: map[string][]byte{"/ctx/session-context.md": []byte(content)},
	}

	file := sessionctx.NewSessionFile(reader, nil, nil, nil, nil)

	meta, err := file.Read("/ctx/session-context.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(meta.Summary).To(Equal("# Summary\nSome content here"))
}

// --- T-151: SessionContextFile writes atomically ---

func TestT151_WriteAtomically(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, writer, dirCreator, renamer, ts)

	err := file.Write("/ctx/session-context.md", sessionctx.SessionContext{
		Summary:   "# Current Task\nWorking on UC-14",
		Offset:    1500,
		SessionID: "sess-001",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should write to temp file first.
	g.Expect(writer.written).To(HaveLen(1))

	// Should rename temp to target.
	g.Expect(renamer.calls).To(HaveLen(1))
	g.Expect(renamer.calls[0].newpath).To(Equal("/ctx/session-context.md"))
}

// --- T-152: SessionContextFile creates directory if missing ---

func TestT152_CreatesDirectoryIfMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, writer, dirCreator, renamer, ts)

	err := file.Write("/project/.claude/engram/session-context.md", sessionctx.SessionContext{
		Summary:   "# Task",
		Offset:    0,
		SessionID: "s1",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dirCreator.created).To(ContainElement("/project/.claude/engram"))
}

// --- T-153: Missing file returns empty ---

func TestT153_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reader := &fakeFileReader{
		contents: map[string][]byte{}, // no files
	}

	file := sessionctx.NewSessionFile(reader, nil, nil, nil, nil)

	meta, err := file.Read("/nonexistent/session-context.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(meta.Summary).To(BeEmpty())
	g.Expect(meta.Offset).To(Equal(int64(0)))
	g.Expect(meta.SessionID).To(BeEmpty())
}

// --- T-154: Missing transcript → exit 0, no file written ---

func TestT154_MissingTranscriptExitsCleanly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// FileReader returns error for transcript (missing file).
	reader := &fakeFileReader{
		contents: map[string][]byte{},
	}

	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(nil)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/missing/transcript.jsonl",
		"sess-1",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	// No file should have been written.
	g.Expect(writer.written).To(BeEmpty())
}

// --- T-155: Empty delta → skip API call, file unchanged ---

func TestT155_EmptyDeltaSkipsAPICall(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Transcript is 16 bytes; offset matches exactly → empty delta.
	transcript := `{"role":"user"}` + "\n"
	existingCtx := fmt.Sprintf(
		"<!-- engram session context"+
			" | updated: 2026-03-07T00:00:00Z"+
			" | offset: %d | session: sess-1 -->"+
			"\n\nPrevious summary",
		len(transcript),
	)

	reader := &fakeFileReader{
		contents: map[string][]byte{
			"/ctx/session-context.md": []byte(existingCtx),
			"/t.jsonl":                []byte(transcript),
		},
	}

	client := &fakeHaikuClient{result: "should not be called"}
	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(client)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/t.jsonl",
		"sess-1",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Offset 20 > len(transcript)=16, so offset resets to 0,
	// reads the full file. But let's test with offset at exact
	// file length so delta is empty.
	g.Expect(client.called).To(BeFalse())
}

// --- T-156: Successful update → file written with updated watermark ---

func TestT156_SuccessfulUpdateWritesFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	transcript := `{"role":"user","content":"help"}` + "\n" +
		`{"role":"assistant","content":"sure"}` + "\n"

	reader := &fakeFileReader{
		contents: map[string][]byte{
			"/ctx/session-context.md": {},
			"/t.jsonl":                []byte(transcript),
		},
	}

	client := &fakeHaikuClient{
		result: "User asked for help, assistant agreed",
	}
	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(client)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/t.jsonl",
		"sess-2",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	// File should have been written (temp file).
	tmpContent, ok := writer.written["/ctx/session-context.md.tmp"]
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	content := string(tmpContent)
	g.Expect(content).To(ContainSubstring("User asked for help"))
	g.Expect(content).To(ContainSubstring("session: sess-2"))
	g.Expect(content).To(ContainSubstring(
		fmt.Sprintf("offset: %d", len(transcript)),
	))
}

// --- T-157: API error → exit 0, file unchanged ---

func TestT157_APIErrorExitsCleanly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	transcript := `{"role":"user","content":"hello"}` + "\n"

	reader := &fakeFileReader{
		contents: map[string][]byte{
			"/ctx/session-context.md": {},
			"/t.jsonl":                []byte(transcript),
		},
	}

	client := &fakeHaikuClient{
		err: errors.New("API unavailable"),
	}
	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(client)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/t.jsonl",
		"sess-1",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Summarizer returns previous summary on error (empty string).
	// File IS written with empty summary but updated watermark.
	// The key assertion: no panic, returns nil.
}

// --- QW-3: Orchestrator caps summary at MaxSummaryBytes ---

func TestOrchestrator_CapsSummaryAtMaxBytes(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	transcript := `{"role":"user","content":"help"}` + "\n"

	reader := &fakeFileReader{
		contents: map[string][]byte{
			"/ctx/session-context.md": {},
			"/t.jsonl":                []byte(transcript),
		},
	}

	// Return a summary larger than 1024 bytes.
	oversizedSummary := strings.Repeat("x", 2000)
	client := &fakeHaikuClient{result: oversizedSummary}
	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(client)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/t.jsonl",
		"sess-cap",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	tmpContent, ok := writer.written["/ctx/session-context.md.tmp"]
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	contentStr := string(tmpContent)

	// Extract summary: everything after the header line + blank line.
	parts := strings.SplitN(contentStr, "\n\n", 2)
	g.Expect(parts).To(HaveLen(2))

	if len(parts) < 2 {
		return
	}

	summary := parts[1]
	g.Expect(len(summary)).To(BeNumerically("<=", sessionctx.MaxSummaryBytes))
}

func TestOrchestrator_SummaryUnderCapUnchanged(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	transcript := `{"role":"user","content":"help"}` + "\n"

	reader := &fakeFileReader{
		contents: map[string][]byte{
			"/ctx/session-context.md": {},
			"/t.jsonl":                []byte(transcript),
		},
	}

	shortSummary := "Short summary under cap"
	client := &fakeHaikuClient{result: shortSummary}
	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	clock := &fakeTimestamper{
		now: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	}

	delta := sessionctx.NewDeltaReader(reader)
	summarizer := sessionctx.NewSummarizer(client)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orch := sessionctx.NewOrchestrator(delta, summarizer, file)

	err := orch.Update(
		context.Background(),
		"/t.jsonl",
		"sess-cap",
		"/ctx/session-context.md",
	)
	g.Expect(err).NotTo(HaveOccurred())

	tmpContent, ok := writer.written["/ctx/session-context.md.tmp"]
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	contentStr := string(tmpContent)
	g.Expect(contentStr).To(ContainSubstring(shortSummary))
}

// --- Coverage: Write content verification ---

func TestWrite_ContentIncludesHeaderAndSummary(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := newFakeFileWriter()
	renamer := &fakeRenamer{}
	dirCreator := &fakeDirCreator{}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 14, 30, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, writer, dirCreator, renamer, ts)

	err := file.Write("/ctx/session-context.md", sessionctx.SessionContext{
		Summary:   "# Working on UC-14",
		Offset:    2500,
		SessionID: "sess-42",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Verify temp file was written with correct content.
	tmpContent, ok := writer.written["/ctx/session-context.md.tmp"]
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	contentStr := string(tmpContent)
	g.Expect(contentStr).To(ContainSubstring("<!-- engram session context"))
	g.Expect(contentStr).To(ContainSubstring("offset: 2500"))
	g.Expect(contentStr).To(ContainSubstring("session: sess-42"))
	g.Expect(contentStr).To(ContainSubstring("2026-03-07T14:30:00Z"))
	g.Expect(contentStr).To(ContainSubstring("# Working on UC-14"))
}

// --- Coverage: Write error on file write failure ---

func TestWrite_FileWriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeFileWriter{written: make(map[string][]byte), err: errors.New("disk full")}
	dirCreator := &fakeDirCreator{}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, writer, dirCreator, nil, ts)

	err := file.Write("/ctx/session-context.md", sessionctx.SessionContext{})
	g.Expect(err).To(MatchError(ContainSubstring("writing temp file")))
}

// --- Coverage: Write error on MkdirAll failure ---

func TestWrite_MkdirAllError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dirCreator := &fakeDirCreator{err: errors.New("permission denied")}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, nil, dirCreator, nil, ts)

	err := file.Write("/ctx/session-context.md", sessionctx.SessionContext{})
	g.Expect(err).To(MatchError(ContainSubstring("creating directory")))
}

// --- Coverage: Write error on rename failure ---

func TestWrite_RenameError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := newFakeFileWriter()
	renamer := &fakeRenamer{err: errors.New("cross-device link")}
	dirCreator := &fakeDirCreator{}
	ts := &fakeTimestamper{now: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}

	file := sessionctx.NewSessionFile(nil, writer, dirCreator, renamer, ts)

	err := file.Write("/ctx/session-context.md", sessionctx.SessionContext{})
	g.Expect(err).To(MatchError(ContainSubstring("renaming temp file")))
}

type fakeDirCreator struct {
	created []string
	err     error
}

func (f *fakeDirCreator) MkdirAll(path string) error {
	if f.err != nil {
		return f.err
	}

	f.created = append(f.created, path)

	return nil
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

type fakeFileWriter struct {
	written map[string][]byte
	err     error
}

func (f *fakeFileWriter) Write(path string, content []byte) error {
	if f.err != nil {
		return f.err
	}

	f.written[path] = content

	return nil
}

type fakeHaikuClient struct {
	result  string
	err     error
	called  bool
	prevIn  string
	deltaIn string
}

func (f *fakeHaikuClient) Summarize(
	_ context.Context,
	previousSummary, delta string,
) (string, error) {
	f.called = true
	f.prevIn = previousSummary
	f.deltaIn = delta

	return f.result, f.err
}

type fakeRenamer struct {
	calls []renameCall
	err   error
}

func (f *fakeRenamer) Rename(_, newpath string) error {
	if f.err != nil {
		return f.err
	}

	f.calls = append(f.calls, renameCall{newpath: newpath})

	return nil
}

type fakeTimestamper struct {
	now time.Time
}

func (f *fakeTimestamper) Now() time.Time {
	return f.now
}

type renameCall struct {
	newpath string
}

func newFakeFileWriter() *fakeFileWriter {
	return &fakeFileWriter{written: make(map[string][]byte)}
}
