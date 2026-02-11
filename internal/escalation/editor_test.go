package escalation_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/escalation"
)

// MockExecutor implements escalation.CommandExecutor for testing
type MockExecutor struct {
	Called      bool
	Command     string
	Args        []string
	ShouldError bool
	Err         error
	// OnRun is called when Run is invoked, allowing test to modify state
	OnRun func(name string, args []string)
}

func (m *MockExecutor) Run(name string, args ...string) error {
	m.Called = true
	m.Command = name
	m.Args = args
	if m.OnRun != nil {
		m.OnRun(name, args)
	}
	if m.ShouldError {
		return m.Err
	}
	return nil
}

// TEST-215 traces: TASK-008
// Test editor selection uses $EDITOR first
func TestSelectEditor_EnvVar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := func(key string) string {
		if key == "EDITOR" {
			return "code"
		}
		return ""
	}

	editor := escalation.SelectEditor(env)
	g.Expect(editor).To(Equal("code"))
}

// TEST-216 traces: TASK-008
// Test editor fallback to vim when $EDITOR not set
func TestSelectEditor_Fallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := func(key string) string {
		return ""
	}

	editor := escalation.SelectEditor(env)
	g.Expect(editor).To(Equal("vim"))
}

// TEST-217 traces: TASK-008
// Test OpenInEditor invokes editor command
func TestOpenInEditor_InvokesCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	exec := &MockExecutor{}

	err := escalation.OpenInEditor("/tmp/escalations.md", "vim", exec)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exec.Called).To(BeTrue())
	g.Expect(exec.Command).To(Equal("vim"))
	g.Expect(exec.Args).To(ContainElement("/tmp/escalations.md"))
}

// TEST-218 traces: TASK-008
// Test ReviewEscalations full workflow
func TestReviewEscalations_Workflow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Mock FS that simulates user editing the file
	fs := &mockFS{files: make(map[string]string)}

	// Content after user edits
	editedContent := `# Escalations

Review each escalation and update the **Status** field:
- ` + "`pending`" + ` - Not yet reviewed
- ` + "`resolved`" + ` - Add your answer in **Notes**
- ` + "`deferred`" + ` - Create an issue for later
- ` + "`issue`" + ` - Create an issue with your description in **Notes**

---

## ESC-001

**Category:** requirement
**Context:** Auth
**Question:** Use OAuth?

**Status:** resolved
**Notes:** Yes, use OAuth 2.0.

---

`

	// Mock executor that simulates editor modifying file
	exec := &MockExecutor{
		OnRun: func(name string, args []string) {
			// Simulate user editing the file when editor is invoked
			fs.files["/tmp/escalations.md"] = editedContent
		},
	}

	// Original escalations
	escalations := []escalation.Escalation{
		{
			ID:       "ESC-001",
			Category: "requirement",
			Context:  "Auth",
			Question: "Use OAuth?",
			Status:   "pending",
			Notes:    "",
		},
	}

	env := func(key string) string { return "vim" }

	result, err := escalation.ReviewEscalations(escalations, "/tmp/escalations.md", env, exec, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Status).To(Equal("resolved"))
	g.Expect(result[0].Notes).To(Equal("Yes, use OAuth 2.0."))
}

// TEST-219 traces: TASK-008
// Test ReviewEscalations handles editor error
func TestReviewEscalations_EditorError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{files: make(map[string]string)}
	exec := &MockExecutor{ShouldError: true, Err: &editorError{}}

	escalations := []escalation.Escalation{
		{ID: "ESC-001", Status: "pending"},
	}

	env := func(key string) string { return "vim" }

	_, err := escalation.ReviewEscalations(escalations, "/tmp/escalations.md", env, exec, fs)
	g.Expect(err).To(HaveOccurred())
}

type editorError struct{}

func (e *editorError) Error() string { return "editor exited with error" }
