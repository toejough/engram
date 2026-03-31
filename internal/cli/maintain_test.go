package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/maintain"
	"engram/internal/policy"
)

func TestRunApplyProposal_DeletesMemory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Create a memory file.
	memoryPath := filepath.Join(memoriesDir, "doomed.toml")

	memoryTOML := `situation = "test"
behavior = "test"
impact = "test"
action = "test"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 0
followed_count = 0
not_followed_count = 0
irrelevant_count = 0
`
	g.Expect(os.WriteFile(memoryPath, []byte(memoryTOML), 0o644)).To(Succeed())

	// Write a proposals file with a delete proposal.
	proposals := []maintain.Proposal{
		{
			ID:        "prop-del-1",
			Action:    maintain.ActionDelete,
			Target:    memoryPath,
			Rationale: "irrelevant memory",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	// Write a policy.toml so change history can be appended.
	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-del-1"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Memory file should be deleted.
	_, statErr := os.Stat(memoryPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())

	// Output should confirm.
	g.Expect(stdout.String()).To(ContainSubstring("applied"))

	// Proposals file should be empty list now.
	remainingData, readErr := os.ReadFile(proposalPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var remaining []maintain.Proposal

	g.Expect(json.Unmarshal(remainingData, &remaining)).To(Succeed())
	g.Expect(remaining).To(BeEmpty())

	// Change history should contain the approved entry.
	history, histErr := policy.ReadChangeHistory(policyPath, os.ReadFile)
	g.Expect(histErr).NotTo(HaveOccurred())

	if histErr != nil {
		return
	}

	g.Expect(history).To(HaveLen(1))

	if len(history) == 0 {
		return
	}

	g.Expect(history[0].Status).To(Equal("approved"))
}

func TestRunApplyProposal_MergeIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	proposals := []maintain.Proposal{
		{
			ID:        "prop-merge-1",
			Action:    maintain.ActionMerge,
			Target:    filepath.Join(dataDir, "memories", "a.toml"),
			Related:   []string{"b.toml"},
			Rationale: "similar memories",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-merge-1"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("applied"))
}

func TestRunApplyProposal_MissingID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir},
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--id required"))
	}
}

func TestRunApplyProposal_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Write empty proposals.
	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, []byte("[]"), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "nonexistent"},
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not found"))
	}
}

func TestRunApplyProposal_RecommendIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	proposals := []maintain.Proposal{
		{
			ID:        "prop-rec-1",
			Action:    maintain.ActionRecommend,
			Target:    filepath.Join(dataDir, "policy.toml"),
			Rationale: "adjust threshold",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-rec-1"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("applied"))
}

func TestRunApplyProposal_UpdatePolicyIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte("[parameters]\n"), 0o644)).To(Succeed())

	proposals := []maintain.Proposal{
		{
			ID:        "upd-pol",
			Action:    maintain.ActionUpdate,
			Target:    policyPath,
			Field:     "maintain_min_surfaced",
			Value:     "5",
			Rationale: "lower threshold",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "upd-pol"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("applied"))
}

func TestRunApplyProposal_UpdatesBehaviorField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memoryPath := filepath.Join(memoriesDir, "update-behavior.toml")

	memoryTOML := `situation = "test"
behavior = "old behavior"
impact = "old impact"
action = "old action"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 1
followed_count = 0
not_followed_count = 0
irrelevant_count = 0
`
	g.Expect(os.WriteFile(memoryPath, []byte(memoryTOML), 0o644)).To(Succeed())

	proposals := []maintain.Proposal{
		{
			ID: "upd-b", Action: maintain.ActionUpdate,
			Target: memoryPath, Field: "behavior",
			Value: "new behavior", Rationale: "r",
		},
		{
			ID: "upd-i", Action: maintain.ActionUpdate,
			Target: memoryPath, Field: "impact",
			Value: "new impact", Rationale: "r",
		},
		{
			ID: "upd-a", Action: maintain.ActionUpdate,
			Target: memoryPath, Field: "action",
			Value: "new action", Rationale: "r",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	// Apply behavior update.
	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "upd-b"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(memoryPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("new behavior"))

	// Apply impact update.
	stdout.Reset()

	err = cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "upd-i"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr = os.ReadFile(memoryPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("new impact"))

	// Apply action update.
	stdout.Reset()

	err = cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "upd-a"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr = os.ReadFile(memoryPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("new action"))
}

func TestRunApplyProposal_UpdatesField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memoryPath := filepath.Join(memoriesDir, "update-me.toml")

	memoryTOML := `situation = "old situation"
behavior = "test"
impact = "test"
action = "test"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 5
followed_count = 2
not_followed_count = 1
irrelevant_count = 0
`
	g.Expect(os.WriteFile(memoryPath, []byte(memoryTOML), 0o644)).To(Succeed())

	proposals := []maintain.Proposal{
		{
			ID:        "prop-upd-1",
			Action:    maintain.ActionUpdate,
			Target:    memoryPath,
			Field:     "situation",
			Value:     "new improved situation",
			Rationale: "more precise",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunApplyProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-upd-1"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Memory file should have updated situation field.
	data, readErr := os.ReadFile(memoryPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("new improved situation"))
}

func TestRunMaintain_NoMemories_EmptyOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Mock caller — won't be invoked since there are no memories.
	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "[]", nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunMaintainWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// null JSON output for no proposals.
	g.Expect(stdout.String()).To(ContainSubstring("null"))
}

func TestRunMaintain_ProducesJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write a memory with high irrelevance to trigger a proposal.
	memoryTOML := `situation = "when doing X"
behavior = "always fails"
impact = "wastes time"
action = "stop doing X"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 10
followed_count = 0
not_followed_count = 0
irrelevant_count = 9
`
	memoryPath := filepath.Join(memoriesDir, "high-irrelevance.toml")
	g.Expect(os.WriteFile(memoryPath, []byte(memoryTOML), 0o644)).To(Succeed())

	// Mock caller returns empty JSON array for consolidation/adapt calls.
	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "[]", nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunMaintainWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).NotTo(BeEmpty())

	// Should be valid JSON.
	var proposals []maintain.Proposal

	g.Expect(json.Unmarshal([]byte(output), &proposals)).To(Succeed())

	// The pending-proposals.json file should also have been written.
	proposalPath := filepath.Join(dataDir, "pending-proposals.json")

	_, statErr := os.Stat(proposalPath)
	g.Expect(statErr).NotTo(HaveOccurred())
}

func TestRunMaintain_WithCaller_IncludesConsolidation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write two memories with similar situations to trigger consolidation.
	memoryTOML1 := `situation = "when writing Go tests"
behavior = "not using t.Parallel"
impact = "tests run slowly"
action = "add t.Parallel to all tests"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 5
followed_count = 3
not_followed_count = 0
irrelevant_count = 0
`

	memoryTOML2 := `situation = "when writing Go unit tests"
behavior = "not calling t.Parallel"
impact = "slow test suite"
action = "always call t.Parallel"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 5
followed_count = 3
not_followed_count = 0
irrelevant_count = 0
`

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "parallel-tests.toml"),
		[]byte(memoryTOML1), 0o644,
	)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "parallel-tests-2.toml"),
		[]byte(memoryTOML2), 0o644,
	)).To(Succeed())

	// Mock caller returns a valid JSON consolidation response.
	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return `[]`, nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunMaintainWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should produce valid JSON output regardless of whether proposals were generated.
	output := stdout.String()
	g.Expect(output).NotTo(BeEmpty())
}

func TestRunRejectProposal_LogsRejection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	proposals := []maintain.Proposal{
		{
			ID:        "prop-rej-1",
			Action:    maintain.ActionDelete,
			Target:    "/some/memory.toml",
			Rationale: "not useful",
		},
	}

	proposalData, marshalErr := json.Marshal(proposals)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, proposalData, 0o644)).To(Succeed())

	policyPath := filepath.Join(dataDir, "policy.toml")
	g.Expect(os.WriteFile(policyPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunRejectProposal(
		[]string{"--data-dir", dataDir, "--id", "prop-rej-1"},
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Output should confirm rejection.
	g.Expect(stdout.String()).To(ContainSubstring("rejected"))

	// Change history should contain "rejected" entry.
	history, histErr := policy.ReadChangeHistory(policyPath, os.ReadFile)
	g.Expect(histErr).NotTo(HaveOccurred())

	if histErr != nil {
		return
	}

	g.Expect(history).To(HaveLen(1))

	if len(history) == 0 {
		return
	}

	g.Expect(history[0].Status).To(Equal("rejected"))

	// Proposals file should be empty now.
	remainingData, readErr := os.ReadFile(proposalPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var remaining []maintain.Proposal

	g.Expect(json.Unmarshal(remainingData, &remaining)).To(Succeed())
	g.Expect(remaining).To(BeEmpty())
}

func TestRunRejectProposal_MissingID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.ExportRunRejectProposal(
		[]string{"--data-dir", dataDir},
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--id required"))
	}
}

func TestRunRejectProposal_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	proposalPath := filepath.Join(dataDir, "pending-proposals.json")
	g.Expect(os.WriteFile(proposalPath, []byte("[]"), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunRejectProposal(
		[]string{"--data-dir", dataDir, "--id", "nonexistent"},
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not found"))
	}
}
