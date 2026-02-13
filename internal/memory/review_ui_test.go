package memory

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestFormatProposal(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("formats embeddings prune proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence (0.15) - 90 days old",
			Preview: "Try using rapid for property testing",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("Embeddings"))
		g.Expect(formatted).To(gomega.ContainSubstring("Low confidence (0.15) - 90 days old"))
		g.Expect(formatted).To(gomega.ContainSubstring("Try using rapid for property testing"))
		g.Expect(formatted).To(gomega.ContainSubstring("Prune?"))
	})

	t.Run("formats skills archive proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "skills",
			Action:  "prune",
			Target:  "skills/old-pattern/SKILL.md",
			Reason:  "No retrievals in 45 days, utility 0.2",
			Preview: "Archive skill: old-pattern",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("Skills"))
		g.Expect(formatted).To(gomega.ContainSubstring("No retrievals in 45 days"))
		g.Expect(formatted).To(gomega.ContainSubstring("old-pattern"))
		g.Expect(formatted).To(gomega.ContainSubstring("Archive?"))
	})

	t.Run("formats CLAUDE.md consolidate proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "claude-md",
			Action:  "consolidate",
			Target:  "redundant entries",
			Reason:  "Redundant (similarity 0.92)",
			Preview: "Consolidate to: 'Always use TDD: write failing tests before implementation'",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("CLAUDE.md"))
		g.Expect(formatted).To(gomega.ContainSubstring("Redundant"))
		g.Expect(formatted).To(gomega.ContainSubstring("Consolidate to"))
		g.Expect(formatted).To(gomega.ContainSubstring("Apply?"))
	})

	t.Run("formats split proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "split",
			Target:  "entry-456",
			Reason:  "Split opportunity (850 tokens, 3 topics detected)",
			Preview: "Split into 3 separate entries",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("Embeddings"))
		g.Expect(formatted).To(gomega.ContainSubstring("Split opportunity"))
		g.Expect(formatted).To(gomega.ContainSubstring("Split?"))
	})

	t.Run("formats promote proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "promote",
			Target:  "entry-789",
			Reason:  "High retrieval (15x), confidence 0.92, used in 4 projects",
			Preview: "Promote to skill: 'Always use property-based testing'",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("Embeddings"))
		g.Expect(formatted).To(gomega.ContainSubstring("High retrieval"))
		g.Expect(formatted).To(gomega.ContainSubstring("Promote?"))
	})

	t.Run("formats demote proposal", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "claude-md",
			Action:  "demote",
			Target:  "specific entry",
			Reason:  "Too specific (only applicable to single package)",
			Preview: "Demote to skill: 'projctl-specific pattern'",
		}

		formatted := formatProposal(proposal)
		g.Expect(formatted).To(gomega.ContainSubstring("CLAUDE.md"))
		g.Expect(formatted).To(gomega.ContainSubstring("Too specific"))
		g.Expect(formatted).To(gomega.ContainSubstring("Demote?"))
	})
}

func TestReviewProposal(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("accepts y input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("y\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
		g.Expect(output.String()).To(gomega.ContainSubstring("[y/n/s]"))
	})

	t.Run("accepts Y input (uppercase)", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("Y\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
	})

	t.Run("rejects n input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("n\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeFalse())
	})

	t.Run("accepts N input (uppercase)", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("N\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeFalse())
	})

	t.Run("skips on s input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("s\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeFalse())
	})

	t.Run("skips on S input (uppercase)", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("S\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeFalse())
	})

	t.Run("handles EOF gracefully", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		input := strings.NewReader("") // Empty input simulates EOF
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err).To(gomega.Equal(io.EOF))
		g.Expect(result).To(gomega.BeFalse())
	})

	t.Run("retries on invalid input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		// Provide invalid input first, then valid input
		input := strings.NewReader("x\ny\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
		g.Expect(output.String()).To(gomega.ContainSubstring("Invalid"))
	})

	t.Run("retries on empty input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		// Provide empty line first, then valid input
		input := strings.NewReader("\ny\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
	})

	t.Run("displays formatted proposal before prompting", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence (0.15) - 90 days old",
			Preview: "Try using rapid for property testing",
		}

		input := strings.NewReader("y\n")
		output := &bytes.Buffer{}

		_, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		outputStr := output.String()
		g.Expect(outputStr).To(gomega.ContainSubstring("Embeddings"))
		g.Expect(outputStr).To(gomega.ContainSubstring("Low confidence"))
		g.Expect(outputStr).To(gomega.ContainSubstring("Try using rapid"))
	})

	t.Run("handles whitespace in input", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		// Input with leading/trailing whitespace
		input := strings.NewReader("  y  \n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
	})
}

func TestReviewProposalEdgeCases(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("handles proposal with empty fields", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
		}

		input := strings.NewReader("y\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
	})

	t.Run("handles very long preview text", func(t *testing.T) {
		longText := strings.Repeat("Lorem ipsum dolor sit amet. ", 20)
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: longText,
		}

		input := strings.NewReader("y\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())
		g.Expect(output.String()).To(gomega.ContainSubstring("Lorem ipsum"))
	})

	t.Run("handles multiple invalid inputs before valid", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence",
			Preview: "Delete entry",
		}

		// Multiple invalid inputs, then valid
		input := strings.NewReader("invalid\n123\nmaybe\ny\n")
		output := &bytes.Buffer{}

		result, err := reviewProposal(proposal, input, output)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(result).To(gomega.BeTrue())

		// Should have multiple "Invalid" messages
		invalidCount := strings.Count(output.String(), "Invalid")
		g.Expect(invalidCount).To(gomega.BeNumerically(">=", 2))
	})
}

func TestFormatProposalActionVerbs(t *testing.T) {
	g := gomega.NewWithT(t)

	// Test that action verbs are correctly transformed into questions
	testCases := []struct {
		action       string
		expectedVerb string
	}{
		{"prune", "Prune?"},
		{"decay", "Decay?"},
		{"consolidate", "Apply?"},
		{"split", "Split?"},
		{"promote", "Promote?"},
		{"demote", "Demote?"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("action %s formats as %s", tc.action, tc.expectedVerb), func(t *testing.T) {
			proposal := MaintenanceProposal{
				Tier:    "embeddings",
				Action:  tc.action,
				Target:  "test-target",
				Reason:  "test reason",
				Preview: "test preview",
			}

			formatted := formatProposal(proposal)
			g.Expect(formatted).To(gomega.ContainSubstring(tc.expectedVerb))
		})
	}
}

// ============================================================================
// Additional tests for ISSUE-218: Content Refinement Action Verbs
// ============================================================================

// TEST-1216: formatProposal handles new refinement action verbs
func TestFormatProposalNewRefinementActions(t *testing.T) {
	g := gomega.NewWithT(t)

	testCases := []struct {
		action       string
		expectedVerb string
	}{
		{"rewrite", "Rewrite?"},
		{"add-rationale", "Add rationale?"},
		{"extract-examples", "Extract?"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("action %s formats as %s", tc.action, tc.expectedVerb), func(t *testing.T) {
			proposal := MaintenanceProposal{
				Tier:    "embeddings",
				Action:  tc.action,
				Target:  "test-target",
				Reason:  "test reason",
				Preview: "test preview",
			}

			formatted := formatProposal(proposal)
			g.Expect(formatted).To(gomega.ContainSubstring(tc.expectedVerb))
		})
	}
}
