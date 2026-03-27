package cli_test

import (
	"bytes"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/policy"
)

func TestRunAdapt_Approve(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
		},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	err = cli.RunAdapt([]string{"--data-dir", dir, "--approve", "pol-001"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, err := policy.Load(policyPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(1))
}

func TestRunAdapt_NoPolicies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	var buf bytes.Buffer

	err := cli.RunAdapt([]string{"--data-dir", dir}, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("No policies"))
}

func TestRunAdapt_Reject(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
		},
		ApprovalStreak: policy.ApprovalStreak{Surfacing: 3},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	err = cli.RunAdapt([]string{"--data-dir", dir, "--reject", "pol-001"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, err := policy.Load(policyPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusRejected))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(0))
}

func TestRunAdapt_Status_ShowsPolicies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Directive: "Increase wEff to 0.5",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionExtraction,
				Directive: "De-prioritize tool patterns",
				Status:    policy.StatusActive,
			},
		},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	err = cli.RunAdapt([]string{"--data-dir", dir, "--status"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("pol-001"))
	g.Expect(output).To(ContainSubstring("proposed"))
	g.Expect(output).To(ContainSubstring("pol-002"))
	g.Expect(output).To(ContainSubstring("active"))
}
