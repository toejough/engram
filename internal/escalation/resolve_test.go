package escalation_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/escalation"
)

// TEST-209 traces: TASK-007
// Test resolved status applies the decision to artifacts
func TestApplyResolutions_Resolved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolutions := []escalation.Escalation{
		{
			ID:       "ESC-001",
			Category: "requirement",
			Context:  "Analyzing auth",
			Question: "Use OAuth?",
			Status:   "resolved",
			Notes:    "Yes, we need OAuth for enterprise customers.",
		},
	}

	result := escalation.ApplyResolutions(resolutions)

	g.Expect(result.Applied).To(HaveLen(1))
	g.Expect(result.Applied[0].ID).To(Equal("ESC-001"))
	g.Expect(result.Applied[0].Notes).To(ContainSubstring("OAuth"))
	g.Expect(result.Issues).To(BeEmpty())
}

// TEST-210 traces: TASK-007
// Test deferred status creates issue
func TestApplyResolutions_Deferred(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolutions := []escalation.Escalation{
		{
			ID:       "ESC-002",
			Category: "design",
			Context:  "UI layout",
			Question: "Sidebar or top nav?",
			Status:   "deferred",
			Notes:    "",
		},
	}

	result := escalation.ApplyResolutions(resolutions)

	g.Expect(result.Applied).To(BeEmpty())
	g.Expect(result.Issues).To(HaveLen(1))
	g.Expect(result.Issues[0].Source).To(Equal("ESC-002"))
	g.Expect(result.Issues[0].Title).To(ContainSubstring("Sidebar or top nav?"))
	g.Expect(result.Issues[0].Category).To(Equal("deferred-escalation"))
}

// TEST-211 traces: TASK-007
// Test issue status creates issue with user description
func TestApplyResolutions_Issue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolutions := []escalation.Escalation{
		{
			ID:       "ESC-003",
			Category: "architecture",
			Context:  "Database choice",
			Question: "SQL or NoSQL?",
			Status:   "issue",
			Notes:    "Need to evaluate PostgreSQL vs MongoDB performance characteristics.",
		},
	}

	result := escalation.ApplyResolutions(resolutions)

	g.Expect(result.Applied).To(BeEmpty())
	g.Expect(result.Issues).To(HaveLen(1))
	g.Expect(result.Issues[0].Source).To(Equal("ESC-003"))
	g.Expect(result.Issues[0].Title).To(ContainSubstring("SQL or NoSQL?"))
	g.Expect(result.Issues[0].Description).To(ContainSubstring("PostgreSQL"))
	g.Expect(result.Issues[0].Category).To(Equal("user-issue"))
}

// TEST-212 traces: TASK-007
// Test pending status is skipped
func TestApplyResolutions_Pending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolutions := []escalation.Escalation{
		{
			ID:       "ESC-004",
			Category: "requirement",
			Context:  "Testing",
			Question: "Coverage target?",
			Status:   "pending",
			Notes:    "",
		},
	}

	result := escalation.ApplyResolutions(resolutions)

	g.Expect(result.Applied).To(BeEmpty())
	g.Expect(result.Issues).To(BeEmpty())
	g.Expect(result.Pending).To(HaveLen(1))
}

// TEST-213 traces: TASK-007
// Test mixed statuses are handled correctly
func TestApplyResolutions_Mixed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolutions := []escalation.Escalation{
		{ID: "ESC-001", Status: "resolved", Notes: "Done"},
		{ID: "ESC-002", Status: "deferred", Question: "Later?"},
		{ID: "ESC-003", Status: "issue", Question: "Track this?", Notes: "Details here"},
		{ID: "ESC-004", Status: "pending"},
	}

	result := escalation.ApplyResolutions(resolutions)

	g.Expect(result.Applied).To(HaveLen(1))
	g.Expect(result.Issues).To(HaveLen(2))
	g.Expect(result.Pending).To(HaveLen(1))
}

// TEST-214 traces: TASK-007
// Test IssueSpec contains required fields
func TestIssueSpec_Fields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	spec := escalation.IssueSpec{
		Source:      "ESC-001",
		Title:       "Evaluate auth options",
		Description: "Need to decide between OAuth and SAML",
		Category:    "user-issue",
	}

	g.Expect(spec.Source).To(Equal("ESC-001"))
	g.Expect(spec.Title).To(Equal("Evaluate auth options"))
	g.Expect(spec.Description).To(Equal("Need to decide between OAuth and SAML"))
	g.Expect(spec.Category).To(Equal("user-issue"))
}
