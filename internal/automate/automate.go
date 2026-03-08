// Package automate implements UC-22 mechanical instruction extraction (ARCH-51).
// Identifies memories containing mechanical patterns and proposes automation replacements.
package automate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/memory"
)

// Minimum keyword score to consider a memory a mechanical candidate.
const minKeywordScore = 2

// mechanicalKeywords are words that indicate a memory contains a mechanical instruction.
var mechanicalKeywords = []string{ //nolint:gochecknoglobals // static lookup table
	"always", "never", "before", "after", "format", "convention",
}

// ErrNoDataDir is returned when no data directory is provided.
var ErrNoDataDir = errors.New("automate: missing data directory")

// Memory is a simplified memory representation for the automate pipeline.
type Memory struct {
	Title    string
	Content  string
	Keywords []string
	FilePath string
}

// AutomationProposal represents a proposed automation for a mechanical memory.
type AutomationProposal struct {
	MemoryPath     string `json:"memory_path"`
	MemoryTitle    string `json:"memory_title"`
	KeywordScore   int    `json:"keyword_score"`
	Generated      bool   `json:"generated"`
	SkippedReason  string `json:"skipped_reason,omitempty"`
	AutomationType string `json:"automation_type,omitempty"`
	Code           string `json:"code,omitempty"`
	Description    string `json:"description,omitempty"`
	TestCommand    string `json:"test_command,omitempty"`
	InstallPath    string `json:"install_path,omitempty"`
	Verified       bool   `json:"verified"`
}

// LLMResponse is the expected JSON structure from the LLM generation call.
type LLMResponse struct {
	AutomationType string `json:"automation_type"`
	Code           string `json:"code"`
	Description    string `json:"description"`
	TestCommand    string `json:"test_command"`
	InstallPath    string `json:"install_path"`
}

// Automator orchestrates the mechanical instruction extraction pipeline.
type Automator struct {
	MemoryLoader func(dataDir string) ([]Memory, error)
	LLMCaller    func(ctx context.Context, prompt string) (string, error)
	RunCommand   func(cmd string) (exitCode int, output string, err error)
	MemoryWriter func(path, retiredBy string, retiredAt time.Time) error
}

// Run executes the automation pipeline: scan → generate → verify → propose.
func (a *Automator) Run(ctx context.Context, dataDir string) ([]AutomationProposal, error) {
	if dataDir == "" {
		return nil, ErrNoDataDir
	}

	memories, err := a.MemoryLoader(dataDir)
	if err != nil {
		return nil, fmt.Errorf("automate: loading memories: %w", err)
	}

	candidates := scoreCandidates(memories)
	if len(candidates) == 0 {
		return []AutomationProposal{}, nil
	}

	proposals := make([]AutomationProposal, 0, len(candidates))

	for _, candidate := range candidates {
		proposal := a.processCandidate(ctx, candidate)
		proposals = append(proposals, proposal)
	}

	return proposals, nil
}

// Retire marks a memory as retired by setting retired_by and retired_at fields.
func (a *Automator) Retire(memoryPath, installPath string, retiredAt time.Time) error {
	if a.MemoryWriter == nil {
		return errors.New("automate: no memory writer configured")
	}

	return a.MemoryWriter(memoryPath, installPath, retiredAt)
}

type scoredCandidate struct {
	mem   Memory
	score int
}

func scoreCandidates(memories []Memory) []scoredCandidate {
	candidates := make([]scoredCandidate, 0)

	for _, mem := range memories {
		score := keywordScore(mem)
		if score >= minKeywordScore {
			candidates = append(candidates, scoredCandidate{mem: mem, score: score})
		}
	}

	return candidates
}

func keywordScore(mem Memory) int {
	searchText := strings.ToLower(mem.Title + " " + mem.Content)
	score := 0

	for _, kw := range mechanicalKeywords {
		if strings.Contains(searchText, kw) {
			score++
		}
	}

	return score
}

func (a *Automator) processCandidate(
	ctx context.Context,
	candidate scoredCandidate,
) AutomationProposal {
	proposal := AutomationProposal{
		MemoryPath:   candidate.mem.FilePath,
		MemoryTitle:  candidate.mem.Title,
		KeywordScore: candidate.score,
	}

	if a.LLMCaller == nil {
		proposal.Generated = false
		proposal.SkippedReason = "no API token"

		return proposal
	}

	prompt := buildGenerationPrompt(candidate.mem)

	response, err := a.LLMCaller(ctx, prompt)
	if err != nil {
		proposal.Generated = false
		proposal.SkippedReason = fmt.Sprintf("LLM error: %s", err)

		return proposal
	}

	var llmResp LLMResponse

	if jsonErr := json.Unmarshal([]byte(response), &llmResp); jsonErr != nil {
		proposal.Generated = false
		proposal.SkippedReason = fmt.Sprintf("parse error: %s", jsonErr)

		return proposal
	}

	proposal.Generated = true
	proposal.AutomationType = llmResp.AutomationType
	proposal.Code = llmResp.Code
	proposal.Description = llmResp.Description
	proposal.TestCommand = llmResp.TestCommand
	proposal.InstallPath = llmResp.InstallPath

	// Verify via test command.
	if a.RunCommand != nil && llmResp.TestCommand != "" {
		exitCode, _, runErr := a.RunCommand(llmResp.TestCommand)
		if runErr == nil && exitCode == 0 {
			proposal.Verified = true
		}
	}

	return proposal
}

func buildGenerationPrompt(mem Memory) string {
	return fmt.Sprintf(
		"Generate automation for this mechanical instruction:\n"+
			"Title: %s\nContent: %s\n"+
			"Respond with JSON: {\"automation_type\", \"code\", \"description\", \"test_command\", \"install_path\"}",
		mem.Title, mem.Content,
	)
}

// MemoriesFromStored converts memory.Stored slice to automate.Memory slice.
func MemoriesFromStored(stored []*memory.Stored) []Memory {
	result := make([]Memory, 0, len(stored))

	for _, s := range stored {
		result = append(result, Memory{
			Title:    s.Title,
			Content:  s.Content,
			Keywords: s.Keywords,
			FilePath: s.FilePath,
		})
	}

	return result
}
