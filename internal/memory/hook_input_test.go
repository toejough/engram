package memory_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

func TestParseHookInputRoundTrip(t *testing.T) {
	g := NewWithT(t)

	input := `{"session_id":"sess-123","transcript_path":"/tmp/transcript.jsonl","cwd":"/Users/joe/repos/projctl","permission_mode":"default","hook_event_name":"PostToolUse"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())
	g.Expect(hi.SessionID).To(Equal("sess-123"))
	g.Expect(hi.TranscriptPath).To(Equal("/tmp/transcript.jsonl"))
	g.Expect(hi.Cwd).To(Equal("/Users/joe/repos/projctl"))
	g.Expect(hi.PermissionMode).To(Equal("default"))
	g.Expect(hi.HookEventName).To(Equal("PostToolUse"))
}

func TestParseHookInputEmptyReader(t *testing.T) {
	g := NewWithT(t)

	hi, err := memory.ParseHookInput(strings.NewReader(""))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).To(BeNil())
}

func TestParseHookInputWhitespaceOnly(t *testing.T) {
	g := NewWithT(t)

	hi, err := memory.ParseHookInput(strings.NewReader("   \n  "))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).To(BeNil())
}

func TestParseHookInputPartialJSON(t *testing.T) {
	g := NewWithT(t)

	input := `{"session_id":"abc","cwd":"/tmp/project"}`
	hi, err := memory.ParseHookInput(strings.NewReader(input))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(hi).ToNot(BeNil())
	g.Expect(hi.SessionID).To(Equal("abc"))
	g.Expect(hi.Cwd).To(Equal("/tmp/project"))
	g.Expect(hi.TranscriptPath).To(BeEmpty())
}

func TestParseHookInputInvalidJSON(t *testing.T) {
	g := NewWithT(t)

	_, err := memory.ParseHookInput(strings.NewReader("{invalid"))
	g.Expect(err).To(HaveOccurred())
}

func TestDeriveProjectNameFromCwd(t *testing.T) {
	g := NewWithT(t)

	g.Expect(memory.DeriveProjectName("/Users/joe/repos/personal/projctl")).To(Equal("projctl"))
	g.Expect(memory.DeriveProjectName("/tmp/my-project")).To(Equal("my-project"))
	g.Expect(memory.DeriveProjectName("/")).To(Equal("/"))
}

func TestDeriveProjectNameEmptyCwd(t *testing.T) {
	g := NewWithT(t)

	g.Expect(memory.DeriveProjectName("")).To(BeEmpty())
}

func TestDeriveProjectNameProperty(t *testing.T) {
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
