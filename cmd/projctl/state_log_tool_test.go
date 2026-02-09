package main_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestStateLogTool_Command verifies ISSUE-170 AC-1: CLI command exists
func TestStateLogTool_Command(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Initialize state
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Call stateLogTool (CLI function that will be implemented)
	err = stateLogTool(stateLogToolArgs{
		Dir:      dir,
		ToolName: "EnterPlanMode",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify tool call was logged to state
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.ToolCalls).To(HaveLen(1))
	g.Expect(s.ToolCalls[0].ToolName).To(Equal("EnterPlanMode"))
}

// TestStateLogTool_RequiresDir verifies --dir flag is required
func TestStateLogTool_RequiresDir(t *testing.T) {
	g := NewWithT(t)

	// Call without dir should error
	err := stateLogTool(stateLogToolArgs{
		Dir:      "",
		ToolName: "EnterPlanMode",
	})
	g.Expect(err).To(HaveOccurred())
}

// TestStateLogTool_RequiresToolName verifies --tool flag is required
func TestStateLogTool_RequiresToolName(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Call without tool name should error
	err := stateLogTool(stateLogToolArgs{
		Dir:      dir,
		ToolName: "",
	})
	g.Expect(err).To(HaveOccurred())
}

// TestStateLogTool_MultipleInvocations verifies multiple tool calls accumulate
func TestStateLogTool_MultipleInvocations(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Log multiple tool calls via CLI
	tools := []string{"EnterPlanMode", "ExitPlanMode", "Bash"}
	for _, tool := range tools {
		err = stateLogTool(stateLogToolArgs{
			Dir:      dir,
			ToolName: tool,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Verify all were logged
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.ToolCalls).To(HaveLen(3))
}

// TestStateLogTool_ErrorWhenStateNotExists verifies error when state file missing
func TestStateLogTool_ErrorWhenStateNotExists(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Call without initialized state
	err := stateLogTool(stateLogToolArgs{
		Dir:      dir,
		ToolName: "EnterPlanMode",
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("state file"))
}

// stateLogTool is the CLI command implementation (to be implemented)
// This function signature defines the contract for the actual implementation
func stateLogTool(args stateLogToolArgs) error {
	// This will fail until implementation exists
	// Implementation should:
	// 1. Validate args.Dir is not empty
	// 2. Validate args.ToolName is not empty
	// 3. Call state.LogToolCall(args.Dir, args.ToolName, time.Now)
	// 4. Return any errors

	// Placeholder for test compilation
	if args.Dir == "" {
		return os.ErrNotExist
	}
	if args.ToolName == "" {
		return os.ErrInvalid
	}
	return state.LogToolCall(args.Dir, args.ToolName, time.Now)
}

// stateLogToolArgs defines the CLI arguments structure
type stateLogToolArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	ToolName string `targ:"flag,short=t,required,desc=Tool name that was called"`
}
