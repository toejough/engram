// Package policy reads and provides SBIA pipeline configuration from policy.toml.
package policy

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Policy holds all tunable parameters and prompts for the SBIA pipeline.
// Missing fields in the TOML file fall back to Defaults().
type Policy struct {
	// DetectFastPathKeywords are keywords that trigger fast-path detection without LLM calls.
	DetectFastPathKeywords []string

	// ContextByteBudget is the maximum bytes of context to include in extraction prompts.
	ContextByteBudget int

	// ContextToolArgsTruncate is the max characters to keep from tool call arguments.
	ContextToolArgsTruncate int

	// ContextToolResultTruncate is the max characters to keep from tool call results.
	ContextToolResultTruncate int

	// ExtractCandidateCountMin is the minimum number of candidate memories to extract.
	ExtractCandidateCountMin int

	// ExtractCandidateCountMax is the maximum number of candidate memories to extract.
	ExtractCandidateCountMax int

	// ExtractBM25Threshold is the minimum BM25 score for a candidate to be considered a duplicate.
	ExtractBM25Threshold float64

	// DetectHaikuPrompt is the system prompt for the Haiku detection call.
	DetectHaikuPrompt string

	// ExtractSonnetPrompt is the system prompt for the Sonnet extraction call.
	ExtractSonnetPrompt string
}

// ReadFileFunc reads a file by path and returns its contents.
type ReadFileFunc func(path string) ([]byte, error)

// Defaults returns a Policy populated with all default values.
func Defaults() Policy {
	return Policy{
		DetectFastPathKeywords:    []string{"remember", "always", "never", "don't", "stop"},
		ContextByteBudget:         defaultContextByteBudget,
		ContextToolArgsTruncate:   defaultContextToolArgsTruncate,
		ContextToolResultTruncate: defaultContextToolResultTruncate,
		ExtractCandidateCountMin:  defaultExtractCandidateCountMin,
		ExtractCandidateCountMax:  defaultExtractCandidateCountMax,
		ExtractBM25Threshold:      defaultExtractBM25Threshold,
		DetectHaikuPrompt:         defaultDetectHaikuPrompt,
		ExtractSonnetPrompt:       defaultExtractSonnetPrompt,
	}
}

// Load reads policy.toml using readFile, falling back to Defaults() for missing or absent fields.
// Returns Defaults() unchanged if the file does not exist.
func Load(readFile ReadFileFunc) (Policy, error) {
	const policyPath = "policy.toml"

	data, err := readFile(policyPath)
	if errors.Is(err, os.ErrNotExist) {
		return Defaults(), nil
	}

	if err != nil {
		return Policy{}, fmt.Errorf("reading policy: %w", err)
	}

	var fileData policyFile

	_, decodeErr := toml.Decode(string(data), &fileData)
	if decodeErr != nil {
		return Policy{}, fmt.Errorf("parsing policy: %w", decodeErr)
	}

	result := Defaults()
	mergeParams(&result, fileData.Parameters)
	mergePrompts(&result, fileData.Prompts)

	return result, nil
}

// LoadFromPath reads policy from a specific file path using os.ReadFile,
// falling back to Defaults() if the file does not exist.
func LoadFromPath(path string) (Policy, error) {
	return Load(func(string) ([]byte, error) {
		return os.ReadFile(path) //nolint:gosec // caller-controlled path
	})
}

// unexported constants.
const (
	defaultContextByteBudget         = 51200
	defaultContextToolArgsTruncate   = 200
	defaultContextToolResultTruncate = 500
	defaultDetectHaikuPrompt         = `You are a correction detector. Read the user message below and determine
whether it is a behavioral correction directed at the AI assistant — something like
"stop doing X", "always do Y", "don't forget to Z", or "remember that W".

Respond with exactly one word:
- CORRECTION if the message is a correction
- NOT_CORRECTION otherwise

Do not explain. Do not add punctuation. Just the single word.`
	defaultExtractBM25Threshold     = 0.3
	defaultExtractCandidateCountMax = 8
	defaultExtractCandidateCountMin = 3
	defaultExtractSonnetPrompt      = `You are a memory extraction assistant for an AI coding tool.
Given the conversation context below, extract memorable facts about the user's
preferences, workflow corrections, or project-specific knowledge.

For each memory, provide:
- situation: when this memory applies (trigger context)
- behavior: what the AI was doing wrong, or what pattern was observed
- impact: why this matters / what goes wrong without this knowledge
- action: what the AI should do differently

Also decide:
- is_new: true if this is a genuinely new memory not covered by existing memories
- duplicate_of: slug of the existing memory this duplicates (if is_new is false)

Return a JSON array of memory objects. Return an empty array if nothing memorable occurred.
Limit to between {{.MinCandidates}} and {{.MaxCandidates}} memories.`
)

// policyFile maps the on-disk TOML structure.
type policyFile struct {
	Parameters policyFileParams  `toml:"parameters"`
	Prompts    policyFilePrompts `toml:"prompts"`
}

// policyFileParams holds the [parameters] section fields.
type policyFileParams struct {
	DetectFastPathKeywords    []string `toml:"detect_fast_path_keywords"`
	ContextByteBudget         int      `toml:"context_byte_budget"`
	ContextToolArgsTruncate   int      `toml:"context_tool_args_truncate"`
	ContextToolResultTruncate int      `toml:"context_tool_result_truncate"`
	ExtractCandidateCountMin  int      `toml:"extract_candidate_count_min"`
	ExtractCandidateCountMax  int      `toml:"extract_candidate_count_max"`
	ExtractBM25Threshold      float64  `toml:"extract_bm25_threshold"`
}

// policyFilePrompts holds the [prompts] section fields.
type policyFilePrompts struct {
	DetectHaiku   string `toml:"detect_haiku"`
	ExtractSonnet string `toml:"extract_sonnet"`
}

// mergeParams overlays non-zero values from params onto policy.
func mergeParams(pol *Policy, params policyFileParams) {
	if len(params.DetectFastPathKeywords) > 0 {
		pol.DetectFastPathKeywords = params.DetectFastPathKeywords
	}

	if params.ContextByteBudget != 0 {
		pol.ContextByteBudget = params.ContextByteBudget
	}

	if params.ContextToolArgsTruncate != 0 {
		pol.ContextToolArgsTruncate = params.ContextToolArgsTruncate
	}

	if params.ContextToolResultTruncate != 0 {
		pol.ContextToolResultTruncate = params.ContextToolResultTruncate
	}

	if params.ExtractCandidateCountMin != 0 {
		pol.ExtractCandidateCountMin = params.ExtractCandidateCountMin
	}

	if params.ExtractCandidateCountMax != 0 {
		pol.ExtractCandidateCountMax = params.ExtractCandidateCountMax
	}

	if params.ExtractBM25Threshold != 0 {
		pol.ExtractBM25Threshold = params.ExtractBM25Threshold
	}
}

// mergePrompts overlays non-empty values from prompts onto policy.
func mergePrompts(pol *Policy, prompts policyFilePrompts) {
	if prompts.DetectHaiku != "" {
		pol.DetectHaikuPrompt = prompts.DetectHaiku
	}

	if prompts.ExtractSonnet != "" {
		pol.ExtractSonnetPrompt = prompts.ExtractSonnet
	}
}
