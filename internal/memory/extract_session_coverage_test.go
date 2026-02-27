package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// ─── containsErrorSignal tests ────────────────────────────────────────────────

func TestContainsErrorSignal_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("some error occurred")).To(BeTrue())
}

func TestContainsErrorSignal_Exception(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("NullPointerException occurred")).To(BeTrue())
}

func TestContainsErrorSignal_ExitStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("exit status 1")).To(BeTrue())
}

func TestContainsErrorSignal_Fail(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("test fail: expected X got Y")).To(BeTrue())
}

func TestContainsErrorSignal_NoError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("everything worked fine")).To(BeFalse())
}

func TestContainsErrorSignal_Panic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsErrorSignal("panic: runtime error: index out of range")).To(BeTrue())
}

// ─── containsPositiveSignal tests ─────────────────────────────────────────────

func TestContainsPositiveSignal_BuildSucceeded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsPositiveSignal("Build succeeded")).To(BeTrue())
}

func TestContainsPositiveSignal_NoSignal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsPositiveSignal("some output without success signal")).To(BeFalse())
}

func TestContainsPositiveSignal_OkPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsPositiveSignal("ok  github.com/example/pkg\n")).To(BeTrue())
}

func TestContainsPositiveSignal_PASS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsPositiveSignal("PASS\n")).To(BeTrue())
}

func TestContainsPositiveSignal_ZeroErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(containsPositiveSignal("lint: 0 errors found")).To(BeTrue())
}

// ─── extractCommandPrefix tests ───────────────────────────────────────────────

func TestExtractCommandPrefix_CompoundDockerCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractCommandPrefix(map[string]any{"command": "docker build -t myapp ."})
	g.Expect(result).To(Equal("docker build"))
}

func TestExtractCommandPrefix_CompoundGitCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractCommandPrefix(map[string]any{"command": "git commit -m 'fix'"})
	g.Expect(result).To(Equal("git commit"))
}

func TestExtractCommandPrefix_CompoundGoCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractCommandPrefix(map[string]any{"command": "go test ./..."})
	g.Expect(result).To(Equal("go test"))
}

func TestExtractCommandPrefix_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(extractCommandPrefix(map[string]any{})).To(Equal(""))
}

func TestExtractCommandPrefix_Nil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(extractCommandPrefix(nil)).To(Equal(""))
}

func TestExtractCommandPrefix_NonCompoundCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractCommandPrefix(map[string]any{"command": "ls -la /tmp"})
	g.Expect(result).To(Equal("ls"))
}

func TestExtractCommandPrefix_SingleWord(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractCommandPrefix(map[string]any{"command": "make"})
	g.Expect(result).To(Equal("make"))
}

// ─── extractFullCommand tests ─────────────────────────────────────────────────

func TestExtractFullCommand_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(extractFullCommand(map[string]any{})).To(Equal(""))
}

func TestExtractFullCommand_Nil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(extractFullCommand(nil)).To(Equal(""))
}

func TestExtractFullCommand_ValidCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractFullCommand(map[string]any{"command": "go test -tags sqlite_fts5 ./..."})
	g.Expect(result).To(Equal("go test -tags sqlite_fts5 ./..."))
}

// ─── stripSystemReminders tests ───────────────────────────────────────────────

func TestStripSystemReminders_MultipleTagsStripped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := "a <system-reminder>first</system-reminder> b <system-reminder>second</system-reminder> c"
	result := stripSystemReminders(input)
	g.Expect(result).To(Equal("a  b  c"))
}

func TestStripSystemReminders_NoTags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := "plain text without any tags"
	g.Expect(stripSystemReminders(input)).To(Equal(input))
}

func TestStripSystemReminders_UnclosedTag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := "before <system-reminder>unclosed content"
	result := stripSystemReminders(input)
	g.Expect(result).To(Equal("before "), "unclosed tag should strip from tag to end of string")
}

func TestStripSystemReminders_WithTag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := "before <system-reminder>SECRET CONTENT</system-reminder> after"
	result := stripSystemReminders(input)
	g.Expect(result).To(Equal("before  after"))
	g.Expect(result).ToNot(ContainSubstring("SECRET CONTENT"))
}

// ─── truncateForItem tests ─────────────────────────────────────────────────────

func TestTruncateForItem_ExactLength(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	text := "exactly"
	result := truncateForItem(text, 7)
	g.Expect(result).To(Equal(text))
}

func TestTruncateForItem_LongText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	text := "this is a longer text that exceeds the limit here"
	result := truncateForItem(text, 10)
	g.Expect(result).To(HaveSuffix("..."))
	g.Expect(result).To(HaveLen(10))
}

func TestTruncateForItem_ShortText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	text := "short text"
	result := truncateForItem(text, 100)
	g.Expect(result).To(Equal(text))
}
