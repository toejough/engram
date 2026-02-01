package result_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/result"
)

// TEST-001-001 traces: TASK-001
// Test that Parse accepts valid TOML with required sections
func TestParse_AcceptsValidTOML(t *testing.T) {
	g := NewWithT(t)

	toml := `
[status]
success = true

[outputs]
files_modified = ["docs/requirements.md", "docs/design.md"]

[[decisions]]
context = "Analyzing CLI structure"
choice = "Use subcommands instead of flags"
reason = "More intuitive for users"
alternatives = ["Flat flags", "Positional arguments"]

[[learnings]]
content = "The codebase uses targ for CLI parsing"
`

	r, err := result.Parse([]byte(toml))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Status.Success).To(BeTrue())
	g.Expect(r.Outputs.FilesModified).To(ConsistOf("docs/requirements.md", "docs/design.md"))
	g.Expect(r.Decisions).To(HaveLen(1))
	g.Expect(r.Decisions[0].Choice).To(Equal("Use subcommands instead of flags"))
	g.Expect(r.Learnings).To(HaveLen(1))
	g.Expect(r.Learnings[0].Content).To(Equal("The codebase uses targ for CLI parsing"))
}

// TEST-001-002 traces: TASK-001
// Test that Parse rejects TOML missing required status section
func TestParse_RejectsMissingStatus(t *testing.T) {
	g := NewWithT(t)

	toml := `
[outputs]
files_modified = []
`

	_, err := result.Parse([]byte(toml))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("status"))
}

// TEST-001-003 traces: TASK-001
// Test that Parse rejects TOML missing required outputs section
func TestParse_RejectsMissingOutputs(t *testing.T) {
	g := NewWithT(t)

	toml := `
[status]
success = true
`

	_, err := result.Parse([]byte(toml))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("outputs"))
}

// TEST-001-004 traces: TASK-001
// Test that Parse accepts TOML with optional sections omitted
func TestParse_AcceptsOptionalSectionsOmitted(t *testing.T) {
	g := NewWithT(t)

	toml := `
[status]
success = false

[outputs]
files_modified = []
`

	r, err := result.Parse([]byte(toml))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Status.Success).To(BeFalse())
	g.Expect(r.Decisions).To(BeEmpty())
	g.Expect(r.Learnings).To(BeEmpty())
}

// TEST-001-005 traces: TASK-001
// Test that decision requires context, choice, reason
func TestParse_RejectsDecisionMissingFields(t *testing.T) {
	g := NewWithT(t)

	toml := `
[status]
success = true

[outputs]
files_modified = []

[[decisions]]
choice = "Use subcommands"
# missing context and reason
`

	_, err := result.Parse([]byte(toml))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("context"))
}

// TEST-001-006 traces: TASK-001
// Test that learning requires content
func TestParse_RejectsLearningMissingContent(t *testing.T) {
	g := NewWithT(t)

	toml := `
[status]
success = true

[outputs]
files_modified = []

[[learnings]]
# missing content
`

	_, err := result.Parse([]byte(toml))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("content"))
}

// TEST-001-007 traces: TASK-001
// Property test: valid results round-trip correctly
func TestParse_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Generate random valid result
		r := result.Result{
			Status: result.Status{
				Success: rapid.Bool().Draw(t, "success"),
			},
			Outputs: result.Outputs{
				FilesModified: rapid.SliceOf(rapid.StringMatching(`[a-z]+\.md`)).Draw(t, "files"),
			},
		}

		// Add random decisions
		numDecisions := rapid.IntRange(0, 3).Draw(t, "numDecisions")
		for i := 0; i < numDecisions; i++ {
			r.Decisions = append(r.Decisions, result.Decision{
				Context:      rapid.StringMatching(`[A-Za-z ]+`).Draw(t, "context"),
				Choice:       rapid.StringMatching(`[A-Za-z ]+`).Draw(t, "choice"),
				Reason:       rapid.StringMatching(`[A-Za-z ]+`).Draw(t, "reason"),
				Alternatives: rapid.SliceOf(rapid.StringMatching(`[A-Za-z]+`)).Draw(t, "alts"),
			})
		}

		// Add random learnings
		numLearnings := rapid.IntRange(0, 3).Draw(t, "numLearnings")
		for i := 0; i < numLearnings; i++ {
			r.Learnings = append(r.Learnings, result.Learning{
				Content: rapid.StringMatching(`[A-Za-z ]+`).Draw(t, "content"),
			})
		}

		// Round-trip
		tomlBytes, err := result.Marshal(r)
		g.Expect(err).ToNot(HaveOccurred())

		parsed, err := result.Parse(tomlBytes)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(parsed.Status.Success).To(Equal(r.Status.Success))
		g.Expect(parsed.Outputs.FilesModified).To(Equal(r.Outputs.FilesModified))
		g.Expect(len(parsed.Decisions)).To(Equal(len(r.Decisions)))
		g.Expect(len(parsed.Learnings)).To(Equal(len(r.Learnings)))
	})
}
