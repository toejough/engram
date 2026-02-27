//go:build sqlite_fts5

package memory

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

// TestTriageOneProposal_CallerError verifies triageOneProposal propagates API call errors.
func TestTriageOneProposal_CallerError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &mockAPIMessageCaller{result: nil, err: errors.New("api unavailable")}
	proposal := MaintenanceProposal{
		Action:  "prune",
		Tier:    "claude-md",
		Reason:  "too specific",
		Preview: "use targ for builds in projctl",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("api unavailable"))
	g.Expect(valid).To(BeFalse())
	g.Expect(rationale).To(BeEmpty())
}

// TestTriageOneProposal_ParseError verifies triageOneProposal returns error on invalid JSON response.
func TestTriageOneProposal_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Return bytes that are not valid JSON
	caller := &mockAPIMessageCaller{result: []byte("not json at all here"), err: nil}
	proposal := MaintenanceProposal{
		Action:  "prune",
		Tier:    "claude-md",
		Reason:  "too specific",
		Preview: "use targ for builds in projctl",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)

	g.Expect(err).To(HaveOccurred())
	g.Expect(valid).To(BeFalse())
	g.Expect(rationale).To(BeEmpty())
}

// TestTriageOneProposal_ValidResponse verifies triageOneProposal parses a valid triage result.
func TestTriageOneProposal_ValidResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &mockAPIMessageCaller{
		result: []byte(`{"valid":true,"rationale":"This is a universal learning"}`),
		err:    nil,
	}
	proposal := MaintenanceProposal{
		Action:  "prune",
		Tier:    "claude-md",
		Reason:  "too specific",
		Preview: "always use TDD",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(valid).To(BeTrue())
	g.Expect(rationale).To(Equal("This is a universal learning"))
}

// mockAPIMessageCaller implements APIMessageCaller with configurable results.
type mockAPIMessageCaller struct {
	result []byte
	err    error
}

func (m *mockAPIMessageCaller) CallAPIWithMessages(_ context.Context, _ APIMessageParams) ([]byte, error) {
	return m.result, m.err
}
