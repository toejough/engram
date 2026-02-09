package state_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"pgregory.net/rapid"
)

// TestLogToolCall verifies ISSUE-170 AC-1: projctl state log-tool command persists tool calls
func TestLogToolCall(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Initialize state
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log a tool call
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify tool call was persisted
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.ToolCalls).To(HaveLen(1))
	g.Expect(s.ToolCalls[0].ToolName).To(Equal("EnterPlanMode"))
}

// TestLogToolCall_MultipleTools verifies multiple tool calls can be logged
func TestLogToolCall_MultipleTools(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log multiple tool calls
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	err = state.LogToolCall(dir, "ExitPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	err = state.LogToolCall(dir, "Bash", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify all tool calls were persisted
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.ToolCalls).To(HaveLen(3))
	g.Expect(s.ToolCalls[0].ToolName).To(Equal("EnterPlanMode"))
	g.Expect(s.ToolCalls[1].ToolName).To(Equal("ExitPlanMode"))
	g.Expect(s.ToolCalls[2].ToolName).To(Equal("Bash"))
}

// TestLogToolCall_PreservesTimestamp verifies tool call timestamps are persisted
func TestLogToolCall_PreservesTimestamp(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	fixedTime := time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)
	nowFunc := func() time.Time { return fixedTime }

	_, err := state.Init(dir, "test-project", nowFunc)
	g.Expect(err).ToNot(HaveOccurred())

	// Log tool call with fixed timestamp
	err = state.LogToolCall(dir, "EnterPlanMode", nowFunc)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify timestamp was persisted
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.ToolCalls[0].Timestamp).To(Equal(fixedTime))
}

// TestLogToolCall_ErrorsWhenStateNotInitialized verifies error handling
func TestLogToolCall_ErrorsWhenStateNotInitialized(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Try to log tool call without initialized state
	err := state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("state file"))
}

// TestGetToolCalls verifies tool calls can be retrieved
func TestGetToolCalls(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log multiple tool calls
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	err = state.LogToolCall(dir, "ExitPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Get tool calls
	calls, err := state.GetToolCalls(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(calls).To(HaveLen(2))
	g.Expect(calls[0].ToolName).To(Equal("EnterPlanMode"))
	g.Expect(calls[1].ToolName).To(Equal("ExitPlanMode"))
}

// TestClearToolCalls verifies tool calls can be cleared (for phase transitions)
func TestClearToolCalls(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log tool calls
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify calls exist
	calls, err := state.GetToolCalls(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(calls).To(HaveLen(1))

	// Clear tool calls
	err = state.ClearToolCalls(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify calls were cleared
	calls, err = state.GetToolCalls(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(calls).To(HaveLen(0))
}

// TestHasToolCall verifies checking for specific tool calls
func TestHasToolCall(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log EnterPlanMode
	err = state.LogToolCall(dir, "EnterPlanMode", time.Now)
	g.Expect(err).ToNot(HaveOccurred())

	// Check for EnterPlanMode - should exist
	has, err := state.HasToolCall(dir, "EnterPlanMode")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(has).To(BeTrue())

	// Check for ExitPlanMode - should not exist
	has, err = state.HasToolCall(dir, "ExitPlanMode")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(has).To(BeFalse())

	// Check for non-existent tool
	has, err = state.HasToolCall(dir, "NonExistentTool")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(has).To(BeFalse())
}

// TestLogToolCall_Property verifies any valid tool name can be logged
func TestLogToolCall_Property(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
		g.Expect(err).ToNot(HaveOccurred())

		// Generate random tool name (alphanumeric with underscores)
		toolName := rapid.StringMatching(`[A-Za-z][A-Za-z0-9_]*`).Draw(rt, "toolName")

		// Log the tool call
		err = state.LogToolCall(dir, toolName, time.Now)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it was persisted
		has, err := state.HasToolCall(dir, toolName)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(has).To(BeTrue())
	})
}

// TestToolCallsPreservedAcrossReads verifies tool calls survive state round-trips
func TestToolCallsPreservedAcrossReads(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log multiple tool calls
	tools := []string{"EnterPlanMode", "ExitPlanMode", "Bash", "Edit", "Write"}
	for _, tool := range tools {
		err = state.LogToolCall(dir, tool, time.Now)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Read state multiple times and verify tool calls persist
	for i := 0; i < 3; i++ {
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.ToolCalls).To(HaveLen(len(tools)))

		for j, tool := range tools {
			g.Expect(s.ToolCalls[j].ToolName).To(Equal(tool))
		}
	}
}
