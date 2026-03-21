package context_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

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

// --- Strip filters by JSONL type field ---

func TestStrip_FiltersProgressAndSystemEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"progress","data":{"type":"hook_progress"}}`,
		`{"type":"user","message":{"content":"hello"}}`,
		`{"type":"assistant","message":{"content":"hi"}}`,
		`{"type":"system","data":"init"}`,
		`{"type":"file-history-snapshot","data":{}}`,
		`{"type":"last-prompt","data":{}}`,
		`{"type":"user","message":{"content":"bye"}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0]).To(ContainSubstring(`"type":"user"`))
	g.Expect(result[1]).To(ContainSubstring(`"type":"assistant"`))
	g.Expect(result[2]).To(ContainSubstring(`"type":"user"`))
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
