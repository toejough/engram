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

func TestStrip_DropsSystemReminderContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	sysReminderOnly := `{"type":"user","message":{"role":"user",` +
		`"content":[{"type":"text",` +
		`"text":"<system-reminder>engram hook output</system-reminder>"}]}}`
	mixedContent := `{"type":"user","message":{"role":"user",` +
		`"content":[{"type":"text",` +
		`"text":"<system-reminder>hook noise</system-reminder>"},` +
		`{"type":"text","text":"yes, do that"}]}}`
	assistantLine := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"OK, doing it."}]}}`

	lines := []string{
		sysReminderOnly,
		mixedContent,
		assistantLine,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0]).To(Equal("USER: yes, do that"))
	g.Expect(result[1]).To(Equal("ASSISTANT: OK, doing it."))
}

// --- Strip extracts text content from JSON, drops tool noise ---

func TestStrip_ExtractsTextFromUserMessages(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	toolResultLine := `{"type":"user","message":{"role":"user",` +
		`"content":[{"type":"tool_result","tool_use_id":"toolu_123",` +
		`"content":"package recall\nimport fmt"}]}}`
	assistantWithTool := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"Let me check the code."},` +
		`{"type":"tool_use","id":"toolu_123","name":"Read",` +
		`"input":{"path":"/foo.go"}}]}}`

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"what was I working on?"}}`,
		toolResultLine,
		assistantWithTool,
		`{"type":"assistant","message":{"role":"assistant",` +
			`"content":[{"type":"text","text":"The bug is fixed."}]}}`,
		`{"type":"progress","data":{"type":"hook_progress"}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0]).To(Equal("USER: what was I working on?"))
	g.Expect(result[1]).To(Equal("ASSISTANT: Let me check the code."))
	g.Expect(result[2]).To(Equal("ASSISTANT: The bug is fixed."))
}

// --- Strip filters by JSONL type field ---

func TestStrip_FiltersProgressAndSystemEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"progress","data":{"type":"hook_progress"}}`,
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
		`{"type":"system","data":"init"}`,
		`{"type":"file-history-snapshot","data":{}}`,
		`{"type":"last-prompt","data":{}}`,
		`{"type":"user","message":{"role":"user","content":"bye"}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0]).To(Equal("USER: hello"))
	g.Expect(result[1]).To(Equal("ASSISTANT: hi"))
	g.Expect(result[2]).To(Equal("USER: bye"))
}

func TestStrip_HandlesStringContentInUserMessages(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"plain string message"}}`,
		`{"type":"user","message":{"role":"user","content":"<system-reminder>noise</system-reminder>"}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(Equal("USER: plain string message"))
}

func TestStrip_LegacyRoleFallback(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Legacy format: role at message level, no outer type field.
	lines := []string{
		`{"message":{"role":"user","content":"legacy user msg"}}`,
		`{"message":{"role":"assistant","content":"legacy assistant msg"}}`,
		`{"message":{"role":"toolUse","content":"should be dropped"}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0]).To(Equal("USER: legacy user msg"))
	g.Expect(result[1]).To(Equal("ASSISTANT: legacy assistant msg"))
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
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"role":"toolResult","content":"big result data"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"response"}]}}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0]).To(Equal("USER: hello"))
	g.Expect(result[1]).To(Equal("ASSISTANT: response"))
}

// --- T-139: ContentStripper replaces base64 strings ---

func TestT139_StripReplacesBase64Strings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Generate a base64-like string >100 chars.
	longBase64 := strings.Repeat("QUFB", 30) // 120 chars of valid base64

	line := fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"data:image/png;base64,%s"}}`, longBase64)

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(ContainSubstring("[base64 removed]"))
	g.Expect(result[0]).NotTo(ContainSubstring(longBase64))
	g.Expect(result[0]).To(HavePrefix("USER: "))
}

// --- T-140: ContentStripper truncates oversized content blocks ---

func TestT140_StripTruncatesOversizedContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	const maxContentLen = 2000

	oversized := strings.Repeat("hello world ", maxContentLen/12+50)
	line := fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"%s"}}`, oversized)

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(HavePrefix("USER: "))
	g.Expect(result[0]).To(ContainSubstring("[truncated]"))
	g.Expect(len(result[0])).To(BeNumerically("<=", maxContentLen+len("[truncated]")))
}

// --- T-141: ContentStripper preserves user messages ---

func TestT141_StripPreservesUserMessages(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	line := `{"type":"user","message":{"role":"user","content":"please help me"}}`

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(Equal("USER: please help me"))
}

// --- T-142: ContentStripper preserves assistant text ---

func TestT142_StripPreservesAssistantText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	line := `{"type":"assistant","message":{"role":"assistant","content":"here is my answer"}}`

	result := sessionctx.Strip([]string{line})

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(Equal("ASSISTANT: here is my answer"))
}

// --- T-143: ContentStripper preserves tool names, removes tool results ---

func TestT143_StripDropsToolUseAndToolResultLines(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lines := []string{
		`{"role":"toolUse","name":"Bash","command":"ls -la","content":"tool use details"}`,
		`{"role":"toolResult","content":"drwxr-xr-x 5 user staff 160 Mar  7 file1\nfile2\nfile3"}`,
	}

	result := sessionctx.Strip(lines)

	g.Expect(result).To(BeEmpty())
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
