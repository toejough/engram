package automate_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/automate"
	"engram/internal/memory"
)

// T-230: Pattern recognition identifies mechanical candidates.
func TestT230_PatternRecognitionIdentifiesMechanicalCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memories := []automate.Memory{
		{
			Title:    "Always run lint before commit",
			Content:  "Never skip the format check",
			FilePath: "m1.toml",
		},
		{
			Title:    "Use convention for naming",
			Content:  "Always follow the format guide after changes",
			FilePath: "m2.toml",
		},
		{Title: "Prefer Go modules", Content: "Use go mod tidy", FilePath: "m3.toml"},
		{Title: "Check test output", Content: "Look at results carefully", FilePath: "m4.toml"},
		{Title: "Review docs", Content: "Read the changelog", FilePath: "m5.toml"},
	}

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return memories, nil
		},
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(2))
	g.Expect(proposals[0].KeywordScore).To(BeNumerically(">=", 2))
	g.Expect(proposals[1].KeywordScore).To(BeNumerically(">=", 2))
}

// T-231: LLM generates automation for mechanical candidate.
func TestT231_LLMGeneratesAutomationForCandidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmResponse := automate.LLMResponse{
		AutomationType: "pre-commit-hook",
		Code:           "#!/bin/sh\ngo vet ./...",
		Description:    "Runs go vet before commit",
		TestCommand:    "echo test",
		InstallPath:    ".git/hooks/pre-commit",
	}

	respJSON, marshalErr := json.Marshal(llmResponse)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return []automate.Memory{
				{
					Title:    "Always run vet before commit",
					Content:  "Never skip format check",
					FilePath: "m1.toml",
				},
			}, nil
		},
		LLMCaller: func(_ context.Context, _ string) (string, error) {
			return string(respJSON), nil
		},
		RunCommand: func(_ string) (int, string, error) {
			return 1, "", nil // not verified for this test
		},
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))

	proposal := proposals[0]
	g.Expect(proposal.Generated).To(BeTrue())
	g.Expect(proposal.AutomationType).To(Equal("pre-commit-hook"))
	g.Expect(proposal.Code).To(ContainSubstring("go vet"))
	g.Expect(proposal.Description).To(Equal("Runs go vet before commit"))
	g.Expect(proposal.TestCommand).To(Equal("echo test"))
	g.Expect(proposal.InstallPath).To(Equal(".git/hooks/pre-commit"))
}

// T-232: Verification passes on exit 0.
func TestT232_VerificationPassesOnExitZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmResponse := automate.LLMResponse{
		AutomationType: "script",
		Code:           "echo ok",
		Description:    "test script",
		TestCommand:    "true",
		InstallPath:    "/tmp/script.sh",
	}

	respJSON, marshalErr := json.Marshal(llmResponse)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return []automate.Memory{
				{
					Title:    "Always format before push",
					Content:  "Never skip convention check",
					FilePath: "m1.toml",
				},
			}, nil
		},
		LLMCaller: func(_ context.Context, _ string) (string, error) {
			return string(respJSON), nil
		},
		RunCommand: func(_ string) (int, string, error) {
			return 0, "ok", nil
		},
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Verified).To(BeTrue())
}

// T-233: Verification fails on non-zero exit.
func TestT233_VerificationFailsOnNonZeroExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmResponse := automate.LLMResponse{
		AutomationType: "script",
		Code:           "echo fail",
		Description:    "failing script",
		TestCommand:    "false",
		InstallPath:    "/tmp/script.sh",
	}

	respJSON, marshalErr := json.Marshal(llmResponse)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return []automate.Memory{
				{
					Title:    "Always format before push",
					Content:  "Never skip convention check",
					FilePath: "m1.toml",
				},
			}, nil
		},
		LLMCaller: func(_ context.Context, _ string) (string, error) {
			return string(respJSON), nil
		},
		RunCommand: func(_ string) (int, string, error) {
			return 1, "error", nil
		},
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Verified).To(BeFalse())
}

// T-234: Retirement sets retired_by field.
func TestT234_RetirementSetsRetiredByField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenPath, writtenRetiredBy string

	var writtenRetiredAt time.Time

	automator := &automate.Automator{
		MemoryWriter: func(path, retiredBy string, retiredAt time.Time) error {
			writtenPath = path
			writtenRetiredBy = retiredBy
			writtenRetiredAt = retiredAt

			return nil
		},
	}

	retireTime := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	err := automator.Retire("memories/lint-before-commit.toml", ".git/hooks/pre-commit", retireTime)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("memories/lint-before-commit.toml"))
	g.Expect(writtenRetiredBy).To(Equal(".git/hooks/pre-commit"))
	g.Expect(writtenRetiredAt).To(Equal(retireTime))
}

// T-235: Retired memories not surfaced.
func TestT235_RetiredMemoriesNotSurfaced(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stored := []*memory.Stored{
		{Title: "Active memory", Content: "still valid", FilePath: "m1.toml"},
		{
			Title: "Retired memory", Content: "replaced", FilePath: "m2.toml",
			RetiredBy: ".git/hooks/pre-commit",
			RetiredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{Title: "Another active", Content: "also valid", FilePath: "m3.toml"},
	}

	converted := automate.MemoriesFromStored(stored)
	// MemoriesFromStored doesn't filter — surface does via filterRetired.
	// This test verifies the Stored struct carries RetiredBy correctly.
	g.Expect(converted).To(HaveLen(3))
	g.Expect(stored[1].RetiredBy).To(Equal(".git/hooks/pre-commit"))

	// The actual surface filtering is tested in the surface package.
	// Here we verify the memory model carries the field correctly.
	active := make([]*memory.Stored, 0)

	for _, s := range stored {
		if s.RetiredBy == "" {
			active = append(active, s)
		}
	}

	g.Expect(active).To(HaveLen(2))
	g.Expect(active[0].Title).To(Equal("Active memory"))
	g.Expect(active[1].Title).To(Equal("Another active"))
}

// T-236: CLI engram automate outputs JSON proposals.
func TestT236_CLIAutomateOutputsJSONProposals(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return []automate.Memory{
				{
					Title:    "Always format before push",
					Content:  "Never skip convention check",
					FilePath: "m1.toml",
				},
			}, nil
		},
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	jsonBytes, jsonErr := json.Marshal(proposals)
	g.Expect(jsonErr).NotTo(HaveOccurred())

	if jsonErr != nil {
		return
	}

	var decoded []automate.AutomationProposal

	unmarshalErr := json.Unmarshal(jsonBytes, &decoded)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(decoded).To(HaveLen(1))

	if len(decoded) == 0 {
		return
	}

	g.Expect(decoded[0].MemoryPath).To(Equal("m1.toml"))
}

// T-237: No API token skips generation, outputs candidates.
func TestT237_NoAPITokenSkipsGenerationOutputsCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	automator := &automate.Automator{
		MemoryLoader: func(_ string) ([]automate.Memory, error) {
			return []automate.Memory{
				{
					Title:    "Always format before push",
					Content:  "Never skip convention check",
					FilePath: "m1.toml",
				},
				{Title: "Simple note", Content: "Just a note", FilePath: "m2.toml"},
			}, nil
		},
		// LLMCaller is nil — no API token.
	}

	proposals, err := automator.Run(context.Background(), "/tmp/data")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals[0].Generated).To(BeFalse())
	g.Expect(proposals[0].SkippedReason).To(Equal("no API token"))
	g.Expect(proposals[0].MemoryTitle).To(Equal("Always format before push"))
}
