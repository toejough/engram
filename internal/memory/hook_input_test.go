package memory_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

func TestDeriveProjectNameEmptyCwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(memory.DeriveProjectName("")).To(BeEmpty())
}

func TestDeriveProjectNameFromCwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(memory.DeriveProjectName("/Users/joe/repos/personal/projctl")).To(Equal("projctl"))
	g.Expect(memory.DeriveProjectName("/tmp/my-project")).To(Equal("my-project"))
	g.Expect(memory.DeriveProjectName("/")).To(Equal("/"))
}

func TestDeriveProjectNameProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		// Generate random directory-like paths
		segments := rapid.SliceOfN(rapid.StringMatching(`[a-zA-Z0-9_-]+`), 1, 5).Draw(rt, "segments")
		path := "/" + strings.Join(segments, "/")
		result := memory.DeriveProjectName(path)
		// Result should be the last segment
		g.Expect(result).To(Equal(segments[len(segments)-1]))
	})
}

func TestExtractToolQueryBash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{"description":"run tests","command":"go test ./..."}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("run tests"))
}

func TestExtractToolQueryBashNoDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Bash",
		ToolInput: json.RawMessage(`{"command":"go test ./..."}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("go test ./..."))
}

func TestExtractToolQueryEdit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Edit",
		ToolInput: json.RawMessage(`{"file_path":"/tmp/baz.go"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("/tmp/baz.go"))
}

func TestExtractToolQueryEmptyToolInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Bash",
		ToolInput: nil,
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("Bash"))
}

func TestExtractToolQueryGlob(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Glob",
		ToolInput: json.RawMessage(`{"pattern":"**/*.go"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("**/*.go"))
}

func TestExtractToolQueryGrep(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Grep",
		ToolInput: json.RawMessage(`{"pattern":"func.*Query"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("func.*Query"))
}

func TestExtractToolQueryRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Read",
		ToolInput: json.RawMessage(`{"file_path":"/tmp/foo.go"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("/tmp/foo.go"))
}

func TestExtractToolQueryTask(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Task",
		ToolInput: json.RawMessage(`{"description":"explore codebase","prompt":"find all test files"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("explore codebase find all test files"))
}

func TestExtractToolQueryUnknownTool(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "SomeNewTool",
		ToolInput: json.RawMessage(`{}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("SomeNewTool"))
}

func TestExtractToolQueryWebFetch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "WebFetch",
		ToolInput: json.RawMessage(`{"prompt":"extract API docs"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("extract API docs"))
}

func TestExtractToolQueryWebSearch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "WebSearch",
		ToolInput: json.RawMessage(`{"query":"golang testing"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("golang testing"))
}

func TestExtractToolQueryWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{
		ToolName:  "Write",
		ToolInput: json.RawMessage(`{"file_path":"/tmp/bar.go"}`),
	}
	g.Expect(hi.ExtractToolQuery()).To(Equal("/tmp/bar.go"))
}

func TestParseHookInputEmptyReader(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi, err := memory.ParseHookInput(strings.NewReader(""))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).To(BeNil())
}

func TestParseHookInputInvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := memory.ParseHookInput(strings.NewReader("{invalid"))
	g.Expect(err).To(HaveOccurred())
}

func TestParseHookInputPartialJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"session_id":"abc","cwd":"/tmp/project"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())

	if hi == nil {
		t.Fatal("ParseHookInput returned nil")
	}

	g.Expect(hi.SessionID).To(Equal("abc"))
	g.Expect(hi.Cwd).To(Equal("/tmp/project"))
	g.Expect(hi.TranscriptPath).To(BeEmpty())
}

func TestParseHookInputPreToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"tool_name":"Bash","tool_input":{"command":"go test","description":"run tests"},"cwd":"/tmp","hook_event_name":"PreToolUse","session_id":"s1"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())

	if hi == nil {
		t.Fatal("ParseHookInput returned nil")
	}

	g.Expect(hi.ToolName).To(Equal("Bash"))
	g.Expect(hi.ToolInput).ToNot(BeNil())

	// Verify ToolInput is valid raw JSON with expected keys
	var toolInput map[string]any
	g.Expect(json.Unmarshal(hi.ToolInput, &toolInput)).To(Succeed())
	g.Expect(toolInput).To(HaveKeyWithValue("command", "go test"))
	g.Expect(toolInput).To(HaveKeyWithValue("description", "run tests"))
}

func TestParseHookInputRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"session_id":"sess-123","transcript_path":"/tmp/transcript.jsonl","cwd":"/Users/joe/repos/projctl","permission_mode":"default","hook_event_name":"PostToolUse"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())

	if hi == nil {
		t.Fatal("ParseHookInput returned nil")
	}

	g.Expect(hi.SessionID).To(Equal("sess-123"))
	g.Expect(hi.TranscriptPath).To(Equal("/tmp/transcript.jsonl"))
	g.Expect(hi.Cwd).To(Equal("/Users/joe/repos/projctl"))
	g.Expect(hi.PermissionMode).To(Equal("default"))
	g.Expect(hi.HookEventName).To(Equal("PostToolUse"))
}

func TestParseHookInputSessionStartStillWorks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"session_id":"abc","cwd":"/tmp/project","hook_event_name":"SessionStart"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())

	if hi == nil {
		t.Fatal("ParseHookInput returned nil")
	}

	g.Expect(hi.SessionID).To(Equal("abc"))
	g.Expect(hi.HookEventName).To(Equal("SessionStart"))
	// New fields should be zero values for backwards compatibility
	g.Expect(hi.Prompt).To(BeEmpty())
	g.Expect(hi.ToolName).To(BeEmpty())
	g.Expect(hi.ToolInput).To(BeNil())
}

func TestParseHookInputUserPromptSubmit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"prompt":"implement auth","cwd":"/tmp/project","hook_event_name":"UserPromptSubmit","session_id":"s1"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())

	if hi == nil {
		t.Fatal("ParseHookInput returned nil")
	}

	g.Expect(hi.Prompt).To(Equal("implement auth"))
	g.Expect(hi.HookEventName).To(Equal("UserPromptSubmit"))
	g.Expect(hi.SessionID).To(Equal("s1"))
}

func TestParseHookInputWhitespaceOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi, err := memory.ParseHookInput(strings.NewReader("   \n  "))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).To(BeNil())
}
