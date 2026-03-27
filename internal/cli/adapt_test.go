package cli_test

import (
	"bytes"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/cli"
	"engram/internal/policy"
)

func TestAdaptApproveWithSnapshot_StoresBeforeMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{{
			ID:        "pol-001",
			Dimension: policy.DimensionSurfacing,
			Directive: "test policy",
			Status:    policy.StatusProposed,
		}},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	snapshot := adapt.CorpusSnapshot{
		FollowRate:        0.45,
		IrrelevanceRatio:  0.12,
		MeanEffectiveness: 62.5,
	}

	var buf bytes.Buffer

	approveErr := cli.AdaptApproveWithSnapshot(policyPath, "pol-001", snapshot, &buf)
	g.Expect(approveErr).NotTo(HaveOccurred())

	if approveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(policyPath)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeFollowRate).To(BeNumerically("~", 0.45, 0.001))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeIrrelevanceRatio).To(BeNumerically("~", 0.12, 0.001))
	g.Expect(loaded.Policies[0].Effectiveness.BeforeMeanEffectiveness).To(BeNumerically("~", 62.5, 0.001))
}

func TestIncrementPolicySessions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusActive,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 3},
			},
			{
				ID: "pol-002", Dimension: policy.DimensionExtraction, Status: policy.StatusActive,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 7},
			},
			{
				ID: "pol-003", Dimension: policy.DimensionSurfacing, Status: policy.StatusRetired,
				Effectiveness: policy.Effectiveness{MeasuredSessions: 5},
			},
		},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cli.IncrementPolicySessions(policyPath)

	loaded, loadErr := policy.Load(policyPath)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	// Active policies get incremented
	g.Expect(loaded.Policies[0].Effectiveness.MeasuredSessions).To(Equal(4))
	g.Expect(loaded.Policies[1].Effectiveness.MeasuredSessions).To(Equal(8))
	// Retired policy is NOT incremented
	g.Expect(loaded.Policies[2].Effectiveness.MeasuredSessions).To(Equal(5))
}

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

func TestRunAdapt_Approve_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	var buf bytes.Buffer

	err := cli.RunAdapt([]string{"--data-dir", dir, "--approve", "pol-999"}, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("policy not found")))
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

func TestRunAdapt_Reject_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	var buf bytes.Buffer

	err := cli.RunAdapt([]string{"--data-dir", dir, "--reject", "pol-999"}, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("policy not found")))
}

func TestRunAdapt_Retire(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.toml")

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusActive},
		},
	}

	err := policy.Save(policyPath, policyFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	err = cli.RunAdapt([]string{"--data-dir", dir, "--retire", "pol-001"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	loaded, err := policy.Load(policyPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded.Policies[0].Status).To(Equal(policy.StatusRetired))
	g.Expect(buf.String()).To(ContainSubstring("Retired policy pol-001"))
}

func TestRunAdapt_Retire_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	var buf bytes.Buffer

	err := cli.RunAdapt([]string{"--data-dir", dir, "--retire", "pol-999"}, &buf)
	g.Expect(err).To(MatchError(ContainSubstring("policy not found")))
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
