package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// --- Behavioral consistency (3c): concise observations ---

func TestBehavioralConsistency_ProducesConciseObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 5 assistant messages mentioning "targ" without correction
	blocks := make([]parsedBlock, 0, 5)

	messages := []string{
		"I'll run targ check to verify the code quality",
		"Using targ install-binary to build the binary",
		"Let me run targ test to execute the test suite",
		"Running targ build for the project",
		"I'll use targ coverage to check test coverage",
	}
	for _, msg := range messages {
		blocks = append(blocks, parsedBlock{
			role: "assistant", blockType: "text", text: msg,
		})
	}

	items := extractBehavioralConsistency(blocks)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractBehavioralConsistency")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Type).To(Equal("behavioral-consistency"))
	g.Expect(items[0].Content).To(ContainSubstring("targ"))
	g.Expect(items[0].Content).To(ContainSubstring("5 times"))
	// Should NOT dump raw message text
	g.Expect(items[0].Content).NotTo(ContainSubstring("I'll run targ check"))
}

// --- Behavioral conventions (3e): rich context ---

func TestBehavioralConventions_IncludesMatchingText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Long assistant text blocks (>50 chars) that all match the same memory
	longText := "I'll use targ check to verify code quality before committing the changes to the repository"
	blocks := []parsedBlock{
		{role: "assistant", blockType: "text", text: longText},
		{role: "assistant", blockType: "text", text: longText},
		{role: "assistant", blockType: "text", text: longText},
	}

	matcher := &mockMatcher{
		matches: map[string][]string{
			longText: {"Always use targ for build commands instead of mage"},
		},
	}

	items := extractBehavioralConventions(blocks, matcher)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractBehavioralConventions")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Type).To(Equal("behavioral-convention"))

	// Must include the matching memory AND the behavior context
	g.Expect(items[0].Content).To(ContainSubstring("targ"))
}

// --- Legacy canned extraction purge ---

func TestIsLegacyCannedExtraction_DetectsOldPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	oldPatterns := []string{
		"used ls successfully in session",
		"used go build successfully in session",
		"used git add successfully in session",
		"autonomously fixed Bash error",
		"autonomously fixed error",
		"CLAUDE.md was edited",
		"consistently used targ throughout session",
		"tests passed using go test",
		"session behavior aligns with: Use targ instead of mage",
	}
	for _, p := range oldPatterns {
		g.Expect(IsSessionBoilerplate(p)).To(BeTrue(),
			"should detect legacy canned extraction: %q", p)
	}

	// Concise observations should NOT be detected as boilerplate
	richContent := []string{
		"When 'projctl query --query foo' fails, fix: 'projctl query foo'",
		"Prefer 'go test' for this task (used 3 times, 100% success rate)",
		"CLAUDE.md modifications:\nChanged: \"Use mage\" → \"Use targ\"",
		"Prefer 'targ' — used consistently (5 times) without correction",
	}
	for _, c := range richContent {
		g.Expect(IsSessionBoilerplate(c)).To(BeFalse(),
			"should NOT detect concise observation as boilerplate: %q", c)
	}
}

// --- Positive outcomes (3b): concise observations ---

func TestPositiveOutcomes_ProducesConciseObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "go test -tags sqlite_fts5 -count=1 ./internal/memory/..."},
			toolID:    "t1"},
		{role: "user", blockType: "tool_result",
			text:   "ok  github.com/toejough/projctl/internal/memory 1.234s\nPASS",
			toolID: "t1"},
	}

	items := extractPositiveOutcomes(blocks)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractPositiveOutcomes")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Type).To(Equal("positive-outcome"))
	g.Expect(items[0].Content).To(ContainSubstring("go test"))
	// Should NOT dump the raw output
	g.Expect(items[0].Content).NotTo(ContainSubstring("PASS"))
	g.Expect(items[0].Content).NotTo(ContainSubstring("1.234s"))
}

// --- Self-corrected failures (3d): concise observations ---

func TestSelfCorrectedFailures_IncludesErrorAndFixContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "projctl query --query foo"},
			toolID:    "tool1"},
		{role: "user", blockType: "tool_result",
			text: "Error: flag provided but not defined: --query", isError: true,
			toolID: "tool1"},
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "projctl query foo"},
			toolID:    "tool2"},
		{role: "user", blockType: "tool_result",
			text:   "## Recent Context from Memory\n1. some result",
			toolID: "tool2"},
	}

	items := extractSelfCorrectedFailures(blocks)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractSelfCorrectedFailures")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Type).To(Equal("self-corrected-failure"))
	g.Expect(items[0].Confidence).To(Equal(0.5))

	// Must mention both the failed and fixed commands
	g.Expect(items[0].Content).To(ContainSubstring("projctl query --query foo"))
	g.Expect(items[0].Content).To(ContainSubstring("projctl query foo"))
}

func TestSelfCorrectedFailures_UnknownToolStillIncludesContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Error without matching toolID (legacy format)
	blocks := []parsedBlock{
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "make build"}, toolID: ""},
		{role: "user", blockType: "tool_result",
			text: "make: *** No rule to make target 'build'. Stop.", isError: true,
			toolID: ""},
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "targ build"}, toolID: ""},
		{role: "user", blockType: "tool_result",
			text: "Build succeeded", toolID: ""},
	}

	items := extractSelfCorrectedFailures(blocks)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractSelfCorrectedFailures")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Content).To(ContainSubstring("make build"))
	g.Expect(items[0].Content).To(ContainSubstring("targ build"))
}

// --- CLAUDE.md edit (Tier A): captures actual changes ---

func TestTierA_CLAUDEMDEdit_CapturesEditContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		// Edit operation targeting CLAUDE.md
		{role: "assistant", blockType: "tool_use", toolName: "Edit",
			toolInput: map[string]any{
				"file_path":  "/Users/joe/repos/personal/projctl/CLAUDE.md",
				"old_string": "Use mage for builds",
				"new_string": "Use targ for builds",
			},
			toolID: "t1"},
		{role: "user", blockType: "tool_result", text: "Edit applied", toolID: "t1"},
	}

	rawMessages := []map[string]any{
		{
			"type": "file-history-snapshot",
			"snapshot": map[string]any{
				"trackedFileBackups": map[string]any{
					"/Users/joe/repos/personal/projctl/CLAUDE.md": "backup content",
				},
			},
		},
	}

	items := extractTierA(blocks, rawMessages)

	// Should find the CLAUDE.md edit item
	var claudeItem *SessionExtractedItem

	for i := range items {
		if items[i].Type == "claude-md-edit" {
			claudeItem = &items[i]
			break
		}
	}

	g.Expect(claudeItem).NotTo(BeNil(), "should extract claude-md-edit item")

	if claudeItem == nil {
		t.Fatal("claudeItem is nil")
	}

	// Must include actual edit content
	g.Expect(claudeItem.Content).To(ContainSubstring("Use mage for builds"))
	g.Expect(claudeItem.Content).To(ContainSubstring("Use targ for builds"))
	g.Expect(claudeItem.Content).NotTo(Equal("CLAUDE.md was edited"))
}

func TestTierA_CLAUDEMDEdit_FallsBackWhenNoEditsInBlocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No Edit/Write blocks, but file-history-snapshot shows CLAUDE.md
	blocks := []parsedBlock{}
	rawMessages := []map[string]any{
		{
			"type": "file-history-snapshot",
			"snapshot": map[string]any{
				"trackedFileBackups": map[string]any{
					"/Users/joe/.claude/CLAUDE.md": "backup",
				},
			},
		},
	}

	items := extractTierA(blocks, rawMessages)

	var claudeItem *SessionExtractedItem

	for i := range items {
		if items[i].Type == "claude-md-edit" {
			claudeItem = &items[i]
			break
		}
	}

	g.Expect(claudeItem).NotTo(BeNil(), "should still detect CLAUDE.md edit from snapshot")

	if claudeItem == nil {
		t.Fatal("claudeItem is nil")
	}
	// Fallback content should note that changes weren't captured
	g.Expect(claudeItem.Content).To(ContainSubstring("CLAUDE.md"))
	g.Expect(claudeItem.Content).NotTo(Equal("CLAUDE.md was edited"))
}

func TestTierA_ExplicitLearning_SkipsSystemReminderOnlyMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Message that's entirely system-reminder content (injected by hooks)
	blocks := []parsedBlock{
		{role: "user", blockType: "text",
			text: "<system-reminder>remember this skill content...</system-reminder>"},
	}

	items := extractTierA(blocks, nil)

	// Should not produce any explicit-learning items
	for _, item := range items {
		g.Expect(item.Type).NotTo(Equal("explicit-learning"),
			"should not treat system-reminder as explicit learning")
	}
}

// --- Explicit learning: system-reminder stripping ---

func TestTierA_ExplicitLearning_StripsSystemReminders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		{role: "user", blockType: "text",
			text: "<system-reminder>Skill file content here...</system-reminder>remember this: always use targ"},
	}

	items := extractTierA(blocks, nil)

	var explicit *SessionExtractedItem

	for i := range items {
		if items[i].Type == "explicit-learning" {
			explicit = &items[i]
			break
		}
	}

	g.Expect(explicit).NotTo(BeNil())

	if explicit == nil {
		t.Fatal("explicit is nil")
	}

	g.Expect(explicit.Content).NotTo(ContainSubstring("Skill file content"))
	g.Expect(explicit.Content).To(ContainSubstring("always use targ"))
}

// --- Tier B: repeated 4-word patterns removed ---

func TestTierB_NoRepeatedPatternItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create blocks with a 4-word phrase repeated 5 times
	blocks := make([]parsedBlock, 0, 5)
	for range 5 {
		blocks = append(blocks, parsedBlock{
			role:      "assistant",
			blockType: "text",
			text:      "I will now proceed to check the test results carefully for issues",
		})
	}

	items := extractTierB(blocks)

	// No repeated-pattern items should be produced
	for _, item := range items {
		g.Expect(item.Type).NotTo(Equal("repeated-pattern"),
			"should not produce repeated-pattern items, got: %s", item.Content)
	}
}

// --- Tool usage patterns (3a): concise observations ---

func TestToolUsagePatterns_ProducesConciseObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	blocks := []parsedBlock{
		// 3 successful "go test" invocations
		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "go test -tags sqlite_fts5 ./internal/memory/..."},
			toolID:    "t1"},
		{role: "user", blockType: "tool_result", text: "ok  github.com/toejough/projctl/internal/memory", toolID: "t1"},

		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "go test -tags sqlite_fts5 -run TestFoo ./..."},
			toolID:    "t2"},
		{role: "user", blockType: "tool_result", text: "ok  github.com/toejough/projctl", toolID: "t2"},

		{role: "assistant", blockType: "tool_use", toolName: "Bash",
			toolInput: map[string]any{"command": "go test -tags sqlite_fts5 ./cmd/..."},
			toolID:    "t3"},
		{role: "user", blockType: "tool_result", text: "ok  github.com/toejough/projctl/cmd", toolID: "t3"},
	}

	items := extractToolUsagePatterns(blocks)
	if len(items) < 1 {
		t.Fatal("expected at least 1 item from extractToolUsagePatterns")
	}

	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Type).To(Equal("tool-usage-pattern"))
	g.Expect(items[0].Content).To(ContainSubstring("go test"))
	g.Expect(items[0].Content).To(ContainSubstring("100%"))
}

// mockMatcher implements SemanticMatcher for testing behavioral conventions.
type mockMatcher struct {
	matches map[string][]string // text → matching memories
}

func (m *mockMatcher) FindSimilarMemories(text string, threshold float64, limit int) ([]string, error) {
	if matches, ok := m.matches[text]; ok {
		return matches, nil
	}

	return nil, nil
}

func (m *mockMatcher) FindSimilarMemoriesBatch(texts []string, threshold float64, limit int) ([][]string, error) {
	results := make([][]string, len(texts))
	for i, text := range texts {
		results[i], _ = m.FindSimilarMemories(text, threshold, limit)
	}

	return results, nil
}
