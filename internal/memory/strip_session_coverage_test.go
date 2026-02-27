package memory

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// ─── computeEditDiff tests ────────────────────────────────────────────────────

func TestComputeEditDiff_BothStringsShort(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := computeEditDiff("old value", "new value")

	g.Expect(result).To(ContainSubstring("  - old value"))
	g.Expect(result).To(ContainSubstring("  + new value"))
}

func TestComputeEditDiff_LongStringsNoCommonPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Strings that are >= 200 chars but completely different
	old := strings.Repeat("a", 250)
	updated := strings.Repeat("b", 250)

	result := computeEditDiff(old, updated)

	// No common prefix → no "..." context line
	g.Expect(result).ToNot(HavePrefix("  ..."))
	g.Expect(result).To(ContainSubstring("- "))
	g.Expect(result).To(ContainSubstring("+ "))
}

func TestComputeEditDiff_LongStringsWithCommonPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Both strings must be >= 200 chars to trigger the long-string branch.
	// Use distinct endings to avoid common-suffix stripping.
	prefix := strings.Repeat("x", 200)
	old := prefix + "OLDVALUE"
	updated := prefix + "NEWSTUFF"

	result := computeEditDiff(old, updated)

	g.Expect(result).To(ContainSubstring("..."))
	g.Expect(result).To(ContainSubstring("- OLDVALUE"))
	g.Expect(result).To(ContainSubstring("+ NEWSTUFF"))
}

func TestComputeEditDiff_LongStringsWithCommonSuffix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	suffix := strings.Repeat("y", 150)
	old := "OLD_" + suffix
	updated := "NEW_" + suffix

	result := computeEditDiff(old, updated)

	g.Expect(result).To(ContainSubstring("- OLD_"))
	g.Expect(result).To(ContainSubstring("+ NEW_"))
}

// ─── formatToolUse tests ──────────────────────────────────────────────────────

func TestFormatToolUse_BashNoHeredoc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Bash", map[string]any{"command": "go test ./..."})

	g.Expect(result).To(Equal("TOOL:Bash $ go test ./..."))
}

func TestFormatToolUse_BashWithHeredoc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := "git commit -m \"$(cat <<'EOF'\nCommit message\nEOF\n)\""
	result := formatToolUse("Bash", map[string]any{"command": cmd})

	g.Expect(result).To(HavePrefix("TOOL:Bash $ git commit -m"))
	g.Expect(result).To(ContainSubstring("... ["))
}

func TestFormatToolUse_DefaultTool(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Task", map[string]any{"description": "do something"})

	g.Expect(result).To(HavePrefix("TOOL:Task "))
}

func TestFormatToolUse_DefaultToolLongInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	longDesc := strings.Repeat("z", 200)
	result := formatToolUse("Task", map[string]any{"description": longDesc})

	g.Expect(result).To(HaveSuffix("..."))
}

func TestFormatToolUse_EditEmptyOldString(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Edit", map[string]any{
		"file_path":  "foo.go",
		"old_string": "",
		"new_string": "new content",
	})

	g.Expect(result).To(Equal("TOOL:Edit foo.go"))
}

func TestFormatToolUse_EditWithNonEmptyOldString(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Edit", map[string]any{
		"file_path":  "bar.go",
		"old_string": "old line",
		"new_string": "new line",
	})

	g.Expect(result).To(HavePrefix("TOOL:Edit bar.go"))
	g.Expect(result).To(ContainSubstring("- old line"))
	g.Expect(result).To(ContainSubstring("+ new line"))
}

func TestFormatToolUse_Glob(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Glob", map[string]any{"pattern": "**/*.go"})

	g.Expect(result).To(HavePrefix("TOOL:Glob "))
}

func TestFormatToolUse_Grep(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Grep", map[string]any{"pattern": "func Test"})

	g.Expect(result).To(HavePrefix("TOOL:Grep "))
}

func TestFormatToolUse_Read(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Read", map[string]any{"file_path": "foo.go"})

	g.Expect(result).To(HavePrefix("TOOL:Read "))
	g.Expect(result).To(ContainSubstring("foo.go"))
}

func TestFormatToolUse_ReadLongInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// input that serializes to > 150 chars
	longPath := strings.Repeat("a", 200)
	result := formatToolUse("Read", map[string]any{"file_path": longPath})

	g.Expect(result).To(HaveSuffix("..."))
}

func TestFormatToolUse_Write(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatToolUse("Write", map[string]any{"file_path": "output.go"})

	g.Expect(result).To(Equal("TOOL:Write output.go"))
}

// ─── isSkillContent tests ─────────────────────────────────────────────────────

func TestIsSkillContent_BaseDirectoryPhrase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isSkillContent("Base directory for this skill: /path/to/skill")).To(BeTrue())
}

func TestIsSkillContent_LaunchingSkillPhrase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isSkillContent("Launching skill: my-skill")).To(BeTrue())
}

func TestIsSkillContent_PlainText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isSkillContent("normal assistant response")).To(BeFalse())
}

func TestStripNoise_EmptyString(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := stripNoise("")

	g.Expect(result).To(Equal(""))
}

// ─── stripNoise tests ─────────────────────────────────────────────────────────

func TestStripNoise_PlainText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := stripNoise("hello world")

	g.Expect(result).To(Equal("hello world"))
}

func TestStripNoise_SkillContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := stripNoise("Launching skill: my-skill\nsome extra content")

	g.Expect(result).To(Equal("(skill loaded)"))
}

func TestStripNoise_SystemReminder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := "before <system-reminder>SECRET</system-reminder> after"
	result := stripNoise(input)

	g.Expect(result).To(Equal("before  after"))
	g.Expect(result).ToNot(ContainSubstring("SECRET"))
}

func TestStripNoise_TeammateMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `<teammate-message teammate_id="agent-123">Hello teammate!</teammate-message>`
	result := stripNoise(input)

	g.Expect(result).To(ContainSubstring("[teammate agent-123]"))
	g.Expect(result).To(ContainSubstring("Hello teammate!"))
}

func TestStripNoise_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := stripNoise("   \n   ")

	g.Expect(result).To(Equal(""))
}
