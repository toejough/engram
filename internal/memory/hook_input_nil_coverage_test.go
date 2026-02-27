package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestHookInput_ExtractToolQuery_BashWithDescription verifies Bash uses description.
func TestHookInput_ExtractToolQuery_BashWithDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	h := &memory.HookInput{
		ToolName:      "Bash",
		HookEventName: "PreToolUse",
		ToolInput:     []byte(`{"description":"Run tests","command":"go test ./..."}`),
	}

	query := h.ExtractToolQuery()

	g.Expect(query).To(Equal("Run tests"))
}

// TestHookInput_ExtractToolQuery_DefaultCase verifies fallback to ToolName for unknown tools.
func TestHookInput_ExtractToolQuery_DefaultCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	h := &memory.HookInput{
		ToolName:      "UnknownTool",
		HookEventName: "PreToolUse",
	}

	g.Expect(h.ExtractToolQuery()).To(Equal("UnknownTool"))
}

// TestHookInput_ExtractToolQuery_EmptyToolName verifies empty string returned for no tool.
func TestHookInput_ExtractToolQuery_EmptyToolName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	h := &memory.HookInput{
		HookEventName: "Stop",
	}

	g.Expect(h.ExtractToolQuery()).To(BeEmpty())
}

// TestHookInput_ExtractToolQuery_NilReceiver verifies nil HookInput returns empty string.
func TestHookInput_ExtractToolQuery_NilReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var h *memory.HookInput

	g.Expect(h.ExtractToolQuery()).To(BeEmpty())
}

// TestHookInput_ExtractToolQuery_TaskTool verifies Task tool extracts description and prompt.
func TestHookInput_ExtractToolQuery_TaskTool(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	h := &memory.HookInput{
		ToolName:      "Task",
		HookEventName: "PreToolUse",
		ToolInput:     []byte(`{"description":"explore codebase","prompt":"find auth patterns"}`),
	}

	query := h.ExtractToolQuery()

	g.Expect(query).To(ContainSubstring("explore codebase"))
	g.Expect(query).To(ContainSubstring("find auth patterns"))
}

// TestHookInput_IsPreToolUse_NilReceiver verifies nil HookInput returns false.
func TestHookInput_IsPreToolUse_NilReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var h *memory.HookInput

	g.Expect(h.IsPreToolUse()).To(BeFalse())
}
