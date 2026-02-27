package escalation_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/escalation"
)

// TEST-208 traces: TASK-006
// Test multiple escalations write/parse
func TestEscalationFile_MultipleItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{files: make(map[string]string)}

	items := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "C1", Question: "Q1?", Status: "pending"},
		{ID: "ESC-002", Category: "design", Context: "C2", Question: "Q2?", Status: "pending"},
		{ID: "ESC-003", Category: "architecture", Context: "C3", Question: "Q3?", Status: "pending"},
	}

	err := escalation.WriteEscalationFile("/tmp/escalations.md", items, fs)
	g.Expect(err).ToNot(HaveOccurred())

	parsed, err := escalation.ParseEscalationFile("/tmp/escalations.md", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parsed).To(HaveLen(3))
}

// TEST-204 traces: TASK-006
// Test round-trip: write then parse returns same escalations
func TestEscalationRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{files: make(map[string]string)}

	original := []escalation.Escalation{
		{
			ID:       "ESC-001",
			Category: "requirement",
			Context:  "Analyzing auth",
			Question: "Use OAuth?",
			Status:   "pending",
			Notes:    "",
		},
		{
			ID:       "ESC-002",
			Category: "design",
			Context:  "UI layout",
			Question: "Sidebar or top nav?",
			Status:   "pending",
			Notes:    "",
		},
	}

	err := escalation.WriteEscalationFile("/tmp/escalations.md", original, fs)
	g.Expect(err).ToNot(HaveOccurred())

	parsed, err := escalation.ParseEscalationFile("/tmp/escalations.md", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parsed).To(HaveLen(2))

	if len(parsed) < 2 {
		t.Fatal("expected at least 2 parsed escalations")
	}

	g.Expect(parsed[0].ID).To(Equal("ESC-001"))
	g.Expect(parsed[0].Category).To(Equal("requirement"))
	g.Expect(parsed[0].Question).To(Equal("Use OAuth?"))
	g.Expect(parsed[1].ID).To(Equal("ESC-002"))
}

// TEST-202 traces: TASK-006
// Test Escalation struct has required fields
func TestEscalation_Fields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	e := escalation.Escalation{
		ID:       "ESC-001",
		Category: "requirement",
		Context:  "Analyzing user authentication",
		Question: "Should we support OAuth in addition to password auth?",
		Status:   "pending",
		Notes:    "",
	}

	g.Expect(e.ID).To(Equal("ESC-001"))
	g.Expect(e.Category).To(Equal("requirement"))
	g.Expect(e.Context).To(Equal("Analyzing user authentication"))
	g.Expect(e.Question).To(Equal("Should we support OAuth in addition to password auth?"))
	g.Expect(e.Status).To(Equal("pending"))
	g.Expect(e.Notes).To(Equal(""))
}

// TEST-207 traces: TASK-006
// Test valid status values are accepted
func TestEscalation_ValidStatuses(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	validStatuses := []string{"pending", "resolved", "deferred", "issue"}

	for _, status := range validStatuses {
		g.Expect(escalation.IsValidStatus(status)).To(BeTrue(), "status %q should be valid", status)
	}

	g.Expect(escalation.IsValidStatus("invalid")).To(BeFalse())
	g.Expect(escalation.IsValidStatus("")).To(BeFalse())
}

// TEST-206 traces: TASK-006
// Test invalid status values are rejected
func TestParseEscalationFile_InvalidStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	invalidContent := `# Escalations

## ESC-001

**Category:** requirement
**Context:** Test
**Question:** Test?

**Status:** unknown_status
**Notes:**
`

	fs := &mockFS{files: map[string]string{
		"/tmp/escalations.md": invalidContent,
	}}

	_, err := escalation.ParseEscalationFile("/tmp/escalations.md", fs)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid status"))
}

// TEST-205 traces: TASK-006
// Test user edits to status and notes are captured
func TestParseEscalationFile_UserEdits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Simulate user-edited file content
	editedContent := `# Escalations

## ESC-001

**Category:** requirement
**Context:** Analyzing auth
**Question:** Use OAuth?

**Status:** resolved
**Notes:** Yes, we need OAuth for enterprise customers.
`

	fs := &mockFS{files: map[string]string{
		"/tmp/escalations.md": editedContent,
	}}

	parsed, err := escalation.ParseEscalationFile("/tmp/escalations.md", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parsed).To(HaveLen(1))

	if len(parsed) < 1 {
		t.Fatal("expected at least 1 parsed escalation")
	}

	g.Expect(parsed[0].Status).To(Equal("resolved"))
	g.Expect(parsed[0].Notes).To(Equal("Yes, we need OAuth for enterprise customers."))
}

// TEST-211 traces: TASK-001
// Test Resolve returns error for invalid status
func TestResolve_InvalidStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	escalations := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "C1", Question: "Q1?", Status: "pending"},
	}

	_, err := escalation.Resolve(escalations, "ESC-001", "bad_status", "notes")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid status"))
}

// TEST-210 traces: TASK-001
// Test Resolve returns error for unknown ID
func TestResolve_UnknownID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	escalations := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "C1", Question: "Q1?", Status: "pending"},
	}

	_, err := escalation.Resolve(escalations, "ESC-999", "resolved", "notes")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ESC-999"))
}

// TEST-209 traces: TASK-001
// Test Resolve updates escalation status and notes by ID
func TestResolve_UpdatesEscalation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	escalations := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "C1", Question: "Q1?", Status: "pending", Notes: ""},
		{ID: "ESC-002", Category: "design", Context: "C2", Question: "Q2?", Status: "pending", Notes: ""},
	}

	updated, err := escalation.Resolve(escalations, "ESC-001", "resolved", "Yes, do it this way")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated).To(HaveLen(2))

	if len(updated) < 2 {
		t.Fatal("expected at least 2 updated escalations")
	}

	g.Expect(updated[0].ID).To(Equal("ESC-001"))
	g.Expect(updated[0].Status).To(Equal("resolved"))
	g.Expect(updated[0].Notes).To(Equal("Yes, do it this way"))
	// ESC-002 should be unchanged
	g.Expect(updated[1].Status).To(Equal("pending"))
}

// TEST-203 traces: TASK-006
// Test WriteEscalationFile creates markdown format
func TestWriteEscalationFile_Format(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{files: make(map[string]string)}

	escalations := []escalation.Escalation{
		{
			ID:       "ESC-001",
			Category: "requirement",
			Context:  "Analyzing auth",
			Question: "Use OAuth?",
			Status:   "pending",
			Notes:    "",
		},
	}

	err := escalation.WriteEscalationFile("/tmp/escalations.md", escalations, fs)
	g.Expect(err).ToNot(HaveOccurred())

	content := fs.files["/tmp/escalations.md"]
	g.Expect(content).To(ContainSubstring("# Escalations"))
	g.Expect(content).To(ContainSubstring("ESC-001"))
	g.Expect(content).To(ContainSubstring("requirement"))
	g.Expect(content).To(ContainSubstring("Analyzing auth"))
	g.Expect(content).To(ContainSubstring("Use OAuth?"))
	g.Expect(content).To(ContainSubstring("pending"))
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// Mock file system for escalation tests
type mockFS struct {
	files map[string]string // path -> content
}

func (m *mockFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}

	return content, nil
}

func (m *mockFS) WriteFile(path string, content string) error {
	m.files[path] = content
	return nil
}
