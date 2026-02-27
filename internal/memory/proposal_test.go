package memory

import (
	"errors"
	"testing"

	"github.com/onsi/gomega"
)

func TestMaintenanceProposal(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("creates proposal with all fields", func(t *testing.T) {
		proposal := MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  "entry-123",
			Reason:  "Low confidence (0.15) - 90 days old",
			Preview: "Delete entry: 'Try using rapid for property testing'",
		}

		g.Expect(proposal.Tier).To(gomega.Equal("embeddings"))
		g.Expect(proposal.Action).To(gomega.Equal("prune"))
		g.Expect(proposal.Target).To(gomega.Equal("entry-123"))
		g.Expect(proposal.Reason).To(gomega.Equal("Low confidence (0.15) - 90 days old"))
		g.Expect(proposal.Preview).To(gomega.Equal("Delete entry: 'Try using rapid for property testing'"))
	})

	t.Run("supports different tiers", func(t *testing.T) {
		tiers := []string{"embeddings", "skills", "claude-md"}
		for _, tier := range tiers {
			proposal := MaintenanceProposal{Tier: tier}
			g.Expect(proposal.Tier).To(gomega.Equal(tier))
		}
	})

	t.Run("supports different actions", func(t *testing.T) {
		actions := []string{"prune", "decay", "consolidate", "split", "promote", "demote"}
		for _, action := range actions {
			proposal := MaintenanceProposal{Action: action}
			g.Expect(proposal.Action).To(gomega.Equal(action))
		}
	})
}

func TestMaintenanceReviewFunc(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("review function accepts proposal", func(t *testing.T) {
		var reviewFunc MaintenanceReviewFunc = func(p MaintenanceProposal) bool {
			return true
		}

		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
			Target: "entry-123",
		}

		approved := reviewFunc(proposal)
		g.Expect(approved).To(gomega.BeTrue())
	})

	t.Run("review function rejects proposal", func(t *testing.T) {
		var reviewFunc MaintenanceReviewFunc = func(p MaintenanceProposal) bool {
			return false
		}

		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
			Target: "entry-123",
		}

		approved := reviewFunc(proposal)
		g.Expect(approved).To(gomega.BeFalse())
	})

	t.Run("review function can inspect proposal fields", func(t *testing.T) {
		var (
			capturedProposal MaintenanceProposal
			reviewFunc       MaintenanceReviewFunc = func(p MaintenanceProposal) bool {
				capturedProposal = p
				return p.Tier == "embeddings" && p.Action == "prune"
			}
		)

		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
			Target: "entry-123",
			Reason: "Low confidence",
		}

		approved := reviewFunc(proposal)
		g.Expect(approved).To(gomega.BeTrue())
		g.Expect(capturedProposal.Tier).To(gomega.Equal("embeddings"))
		g.Expect(capturedProposal.Action).To(gomega.Equal("prune"))

		// Reject other actions
		proposal2 := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "decay",
			Target: "entry-456",
		}
		approved2 := reviewFunc(proposal2)
		g.Expect(approved2).To(gomega.BeFalse())
	})
}

func TestProposalApplier(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("applier applies proposal successfully", func(t *testing.T) {
		applier := &mockProposalApplier{}

		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
			Target: "entry-123",
		}

		err := applier.Apply(proposal)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(applier.appliedProposals).To(gomega.HaveLen(1))
		g.Expect(applier.appliedProposals[0].Target).To(gomega.Equal("entry-123"))
	})

	t.Run("applier returns error on failure", func(t *testing.T) {
		applier := &mockProposalApplier{
			shouldError: true,
		}

		proposal := MaintenanceProposal{
			Tier:   "embeddings",
			Action: "prune",
			Target: "entry-123",
		}

		err := applier.Apply(proposal)
		g.Expect(err).To(gomega.HaveOccurred())
	})
}

func TestTierScanner(t *testing.T) {
	g := gomega.NewWithT(t)

	t.Run("scanner returns proposals", func(t *testing.T) {
		scanner := &mockTierScanner{
			proposals: []MaintenanceProposal{
				{
					Tier:   "embeddings",
					Action: "prune",
					Target: "entry-123",
				},
			},
		}

		proposals, err := scanner.Scan()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(proposals).To(gomega.HaveLen(1))
		g.Expect(proposals[0].Tier).To(gomega.Equal("embeddings"))
	})

	t.Run("scanner returns error on failure", func(t *testing.T) {
		scanner := &mockTierScanner{
			shouldError: true,
		}

		_, err := scanner.Scan()
		g.Expect(err).To(gomega.HaveOccurred())
	})
}

type mockProposalApplier struct {
	appliedProposals []MaintenanceProposal
	shouldError      bool
}

func (m *mockProposalApplier) Apply(p MaintenanceProposal) error {
	if m.shouldError {
		return errors.New("apply error")
	}

	m.appliedProposals = append(m.appliedProposals, p)

	return nil
}

// Mock implementations for testing

type mockTierScanner struct {
	proposals   []MaintenanceProposal
	shouldError bool
}

func (m *mockTierScanner) Scan() ([]MaintenanceProposal, error) {
	if m.shouldError {
		return nil, errors.New("scan error")
	}

	return m.proposals, nil
}
