// Package policy reads and provides SBIA pipeline configuration from policy.toml.
package policy

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ChangeEntry records a single change made by the maintain or adapt pipeline.
type ChangeEntry struct {
	Action    string `toml:"action"`
	Target    string `toml:"target"`
	Field     string `toml:"field,omitempty"`
	OldValue  string `toml:"old_value,omitempty"`
	NewValue  string `toml:"new_value,omitempty"`
	Status    string `toml:"status"`
	Rationale string `toml:"rationale"`
	ChangedAt string `toml:"changed_at"`
}

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

	// SurfaceCandidateCountMin is the minimum number of candidate memories to surface.
	SurfaceCandidateCountMin int

	// SurfaceCandidateCountMax is the maximum number of candidate memories to surface.
	SurfaceCandidateCountMax int

	// SurfaceBM25Threshold is the minimum BM25 score for a memory to pass the relevance gate.
	SurfaceBM25Threshold float64

	// SurfaceColdStartBudget is the number of memories to include when no BM25 candidates qualify.
	SurfaceColdStartBudget int

	// SurfaceIrrelevanceHalfLife is the number of sessions after which an irrelevant memory's score halves.
	SurfaceIrrelevanceHalfLife int

	// MaintainEffectivenessThreshold is the minimum effectiveness score (percent) before a memory is flagged.
	MaintainEffectivenessThreshold float64

	// MaintainMinSurfaced is the minimum number of times a memory must be surfaced before it is eligible
	// for maintenance evaluation.
	MaintainMinSurfaced int

	// MaintainIrrelevanceThreshold is the percentage of IRRELEVANT verdicts that triggers a rewrite.
	MaintainIrrelevanceThreshold float64

	// MaintainNotFollowedThreshold is the percentage of NOT_FOLLOWED verdicts that triggers investigation.
	MaintainNotFollowedThreshold float64

	// GateIrrelevanceThreshold is the aggregate irrelevance rate (percent) across all memories
	// that triggers a prompt re-evaluation recommendation. Default 10%.
	GateIrrelevanceThreshold float64

	// AdaptChangeHistoryLimit is the maximum number of change history entries to retain in policy.toml.
	AdaptChangeHistoryLimit int

	// DetectHaikuPrompt is the system prompt for the Haiku detection call.
	DetectHaikuPrompt string

	// ExtractSonnetPrompt is the system prompt for the Sonnet extraction call.
	ExtractSonnetPrompt string

	// RefineSonnetPrompt is the system prompt for rewriting existing memory SBIA fields.
	RefineSonnetPrompt string

	// SurfaceGateHaikuPrompt is the system prompt for the Haiku gate call that filters surfaced memories.
	SurfaceGateHaikuPrompt string

	// SurfaceInjectionPreamble is the text prepended to the surfaced memories block injected into context.
	SurfaceInjectionPreamble string

	// EvaluateHaikuPrompt is the system prompt for the Haiku evaluation call that scores memory adherence.
	EvaluateHaikuPrompt string

	// MaintainRewritePrompt is the system prompt for Sonnet to rewrite a memory field more precisely.
	MaintainRewritePrompt string

	// MaintainConsolidatePrompt is the system prompt for Sonnet to synthesize similar memories into one.
	MaintainConsolidatePrompt string

	// AdaptSonnetPrompt is the system prompt for Sonnet to propose parameter adjustments.
	AdaptSonnetPrompt string
}

// ReadFileFunc reads a file by path and returns its contents.
type ReadFileFunc func(path string) ([]byte, error)

// WriteFileFunc writes content to a file by path.
type WriteFileFunc func(path string, data []byte) error

// AppendChangeHistory appends a ChangeEntry to the policy file's change_history section,
// trimming to the configured limit. Preserves existing file content.
func AppendChangeHistory(
	path string,
	entry ChangeEntry,
	readFile ReadFileFunc,
	writeFile WriteFileFunc,
) error {
	existing, err := ReadChangeHistory(path, readFile)
	if err != nil {
		return fmt.Errorf("appending change history: %w", err)
	}

	entries := append(existing, entry) //nolint:gocritic // intentional append to new slice

	limit := Defaults().AdaptChangeHistoryLimit
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Read original file content to preserve other sections.
	originalData, readErr := readFile(path)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading policy for change history append: %w", readErr)
	}

	// Strip existing [[change_history]] entries from original content.
	cleaned := stripChangeHistory(string(originalData))

	// Encode new change_history entries.
	historyData := changeHistoryFile{ChangeHistory: entries}

	var buf bytes.Buffer

	encoder := toml.NewEncoder(&buf)

	encodeErr := encoder.Encode(historyData)
	if encodeErr != nil {
		return fmt.Errorf("encoding change history: %w", encodeErr)
	}

	// Combine cleaned original content with new change_history.
	var result bytes.Buffer

	if len(cleaned) > 0 {
		result.WriteString(cleaned)

		if cleaned[len(cleaned)-1] != '\n' {
			result.WriteByte('\n')
		}

		result.WriteByte('\n')
	}

	result.Write(buf.Bytes())

	return writeFile(path, result.Bytes())
}

// Defaults returns a Policy populated with all default values.
func Defaults() Policy {
	return Policy{
		DetectFastPathKeywords:         []string{"remember", "always", "never", "don't", "stop"},
		ContextByteBudget:              defaultContextByteBudget,
		ContextToolArgsTruncate:        defaultContextToolArgsTruncate,
		ContextToolResultTruncate:      defaultContextToolResultTruncate,
		ExtractCandidateCountMin:       defaultExtractCandidateCountMin,
		ExtractCandidateCountMax:       defaultExtractCandidateCountMax,
		ExtractBM25Threshold:           defaultExtractBM25Threshold,
		SurfaceCandidateCountMin:       defaultSurfaceCandidateCountMin,
		SurfaceCandidateCountMax:       defaultSurfaceCandidateCountMax,
		SurfaceBM25Threshold:           defaultSurfaceBM25Threshold,
		SurfaceColdStartBudget:         defaultSurfaceColdStartBudget,
		SurfaceIrrelevanceHalfLife:     defaultSurfaceIrrelevanceHalfLife,
		MaintainEffectivenessThreshold: defaultMaintainEffectivenessThreshold,
		MaintainMinSurfaced:            defaultMaintainMinSurfaced,
		MaintainIrrelevanceThreshold:   defaultMaintainIrrelevanceThreshold,
		MaintainNotFollowedThreshold:   defaultMaintainNotFollowedThreshold,
		GateIrrelevanceThreshold:       defaultGateIrrelevanceThreshold,
		AdaptChangeHistoryLimit:        defaultAdaptChangeHistoryLimit,
		DetectHaikuPrompt:              defaultDetectHaikuPrompt,
		ExtractSonnetPrompt:            defaultExtractSonnetPrompt,
		RefineSonnetPrompt:             defaultRefineSonnetPrompt,
		SurfaceGateHaikuPrompt:         defaultSurfaceGateHaikuPrompt,
		SurfaceInjectionPreamble:       defaultSurfaceInjectionPreamble,
		EvaluateHaikuPrompt:            defaultEvaluateHaikuPrompt,
		MaintainRewritePrompt:          defaultMaintainRewritePrompt,
		MaintainConsolidatePrompt:      defaultMaintainConsolidatePrompt,
		AdaptSonnetPrompt:              defaultAdaptSonnetPrompt,
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

// ReadChangeHistory reads the change_history entries from a policy TOML file.
func ReadChangeHistory(path string, readFile ReadFileFunc) ([]ChangeEntry, error) {
	data, err := readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading change history: %w", err)
	}

	var fileData policyFile

	_, decodeErr := toml.Decode(string(data), &fileData)
	if decodeErr != nil {
		return nil, fmt.Errorf("parsing change history: %w", decodeErr)
	}

	return fileData.ChangeHistory, nil
}

// unexported constants.
const (
	defaultAdaptChangeHistoryLimit = 50
	defaultAdaptSonnetPrompt       = "You are analyzing engram's memory system performance " +
		"to propose parameter adjustments.\n\n" +
		"Current parameters:\n{{.CurrentParams}}\n\n" +
		"Recent change history:\n{{.ChangeHistory}}\n\n" +
		"Performance summary:\n{{.PerformanceSummary}}\n\n" +
		"Propose parameter adjustments to improve memory effectiveness.\n" +
		"Return a JSON array of objects, each with: field, value, rationale.\n" +
		"Return an empty array if no changes are needed.\n" +
		"Do not explain. Return only the JSON array."
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
	defaultEvaluateHaikuPrompt = "You are evaluating whether a memory was relevant and followed" +
		" during a conversation.\n\n" +
		"Memory:\n" +
		"- Situation: {situation}\n" +
		"- Behavior to avoid: {behavior}\n" +
		"- Action: {action}\n\n" +
		"Transcript (agent's response after memory was surfaced):\n" +
		"{transcript}\n\n" +
		"Assess:\n" +
		"1. Was the situation relevant to what was happening? (yes/no)\n" +
		"2. If relevant, was the action taken by the agent? (yes/no)\n\n" +
		"Return exactly one of: FOLLOWED, NOT_FOLLOWED, IRRELEVANT\n" +
		"Do not explain. Return only the verdict."
	defaultExtractBM25Threshold     = 0.3
	defaultExtractCandidateCountMax = 8
	defaultExtractCandidateCountMin = 3
	defaultExtractSonnetPrompt      = `You are a structured data extraction tool. You output ONLY valid JSON, never prose.

Given a correction message and conversation context, extract SBIA memory fields:
- situation: when this memory applies (observable activity context, not topic tags)
- behavior: what the AI was doing wrong (the default/faulty decision, not just the tool call)
- impact: what goes wrong as a result (concrete negative outcome)
- action: what the AI should do differently
- filename_slug: a kebab-case slug for the memory filename
- project_scoped: true if this only applies to a specific project

If existing candidate memories are provided, evaluate EACH candidate against this SBIA decision tree:

1. How similar is the Situation?
   - Same situation + Same behavior:
     - Same impact + same action → DUPLICATE (don't create new memory)
     - Same impact + different action → CONTRADICTION (user changed mind or tech changed)
     - Different impact + same action → IMPACT_UPDATE (richer impact description)
     - Different impact + different action → REFINEMENT (unusual — flag for review)
   - Same situation + Different behavior → STORE_BOTH (two different mistakes in same context)
   - Similar situation (related but not identical):
     - Same behavior + same impact → POTENTIAL_GENERALIZATION (merge into broader situation)
     - Same behavior + different impact → LEGITIMATE_SEPARATE (situation nuance matters)
     - Different behavior → STORE_BOTH (independent lessons)
   - Different situation → STORE (no relationship)

For each candidate, output a disposition object:
{"name": "<candidate-slug>", "disposition": "<DISPOSITION>", "reason": "<why>"}

Return a single JSON object (not an array):
{
  "situation": "...",
  "behavior": "...",
  "impact": "...",
  "action": "...",
  "filename_slug": "...",
  "project_scoped": false,
  "candidates": [{"name": "...", "disposition": "...", "reason": "..."}]
}

If no candidates provided, return an empty candidates array.

CRITICAL: Do NOT respond to the conversation context. Do NOT continue any conversation.
Output ONLY the JSON object. No explanation. No prose. No markdown outside code fences.`
	defaultGateIrrelevanceThreshold  = 10.0
	defaultMaintainConsolidatePrompt = "You are consolidating similar memories into one.\n\n" +
		"Memories to consolidate:\n" +
		"{{.Memories}}\n\n" +
		"Synthesize these into a single memory that captures the essential pattern.\n" +
		"Return a JSON object with fields: situation, behavior, impact, action.\n" +
		"Do not explain. Return only the JSON object."
	defaultMaintainEffectivenessThreshold = 50.0
	defaultMaintainIrrelevanceThreshold   = 60.0
	defaultMaintainMinSurfaced            = 5
	defaultMaintainNotFollowedThreshold   = 50.0
	defaultMaintainRewritePrompt          = "You are rewriting a memory field to be more precise.\n\n" +
		"The memory's {{.Field}} field currently reads:\n" +
		"\"{{.CurrentValue}}\"\n\n" +
		"It has been surfaced {{.SurfacedCount}} times with these verdicts:\n" +
		"{{.VerdictSummary}}\n\n" +
		"Rewrite the {{.Field}} field to be more specific and actionable.\n" +
		"Return only the rewritten text, no explanation."
	defaultRefineSonnetPrompt = `You are a structured data rewriting tool. You output ONLY valid JSON, never prose.

You are given an existing memory with SBIA fields and the original session transcript where
the memory was created. Your task: rewrite the SBIA fields to be clearer, more specific, and more
actionable while preserving the existing memory's core meaning.

For each field, improve clarity:
- situation: describe the observable activity context when this applies (not topic tags)
- behavior: describe the specific faulty decision or default action (not just the tool call)
- impact: describe the concrete negative outcome
- action: describe the specific corrective action to take

Do NOT change the memory's identity — same lesson, better wording. Strip any "Keywords: ..."
suffixes from field values. Do not add new information not supported by the transcript.

Return a single JSON object with the rewritten fields:
{"situation": "...", "behavior": "...", "impact": "...", "action": "..."}

CRITICAL: Do NOT respond to the transcript. Do NOT continue any conversation.
Output ONLY the JSON object. No explanation. No prose. No markdown outside code fences.`
	defaultSurfaceBM25Threshold     = 0.3
	defaultSurfaceCandidateCountMax = 8
	defaultSurfaceCandidateCountMin = 3
	defaultSurfaceColdStartBudget   = 2
	defaultSurfaceGateHaikuPrompt   = `You are a memory relevance classifier for an AI coding assistant.
Given the user's current context below and a list of memories (each with a slug and situation description),
classify which memories' situations match the user's current context.

Return a JSON array of slugs for matching memories. Return an empty array if none match.
Do not explain. Return only the JSON array.`
	defaultSurfaceInjectionPreamble = "[engram] Memories — use `engram show --name <name>`" +
		" for tracking data (effectiveness, relevance):"
	defaultSurfaceIrrelevanceHalfLife = 5
)

// changeHistoryFile is used for encoding just the change_history section.
type changeHistoryFile struct {
	ChangeHistory []ChangeEntry `toml:"change_history"`
}

// policyFile maps the on-disk TOML structure.
type policyFile struct {
	Parameters    policyFileParams  `toml:"parameters"`
	Prompts       policyFilePrompts `toml:"prompts"`
	ChangeHistory []ChangeEntry     `toml:"change_history"`
}

// policyFileParams holds the [parameters] section fields.
type policyFileParams struct {
	DetectFastPathKeywords         []string `toml:"detect_fast_path_keywords"`
	ContextByteBudget              int      `toml:"context_byte_budget"`
	ContextToolArgsTruncate        int      `toml:"context_tool_args_truncate"`
	ContextToolResultTruncate      int      `toml:"context_tool_result_truncate"`
	ExtractCandidateCountMin       int      `toml:"extract_candidate_count_min"`
	ExtractCandidateCountMax       int      `toml:"extract_candidate_count_max"`
	ExtractBM25Threshold           float64  `toml:"extract_bm25_threshold"`
	SurfaceCandidateCountMin       int      `toml:"surface_candidate_count_min"`
	SurfaceCandidateCountMax       int      `toml:"surface_candidate_count_max"`
	SurfaceBM25Threshold           float64  `toml:"surface_bm25_threshold"`
	SurfaceColdStartBudget         int      `toml:"surface_cold_start_budget"`
	SurfaceIrrelevanceHalfLife     int      `toml:"surface_irrelevance_half_life"`
	MaintainEffectivenessThreshold float64  `toml:"maintain_effectiveness_threshold"`
	MaintainMinSurfaced            int      `toml:"maintain_min_surfaced"`
	MaintainIrrelevanceThreshold   float64  `toml:"maintain_irrelevance_threshold"`
	MaintainNotFollowedThreshold   float64  `toml:"maintain_not_followed_threshold"`
	GateIrrelevanceThreshold       float64  `toml:"gate_irrelevance_threshold"`
	AdaptChangeHistoryLimit        int      `toml:"adapt_change_history_limit"`
}

// policyFilePrompts holds the [prompts] section fields.
type policyFilePrompts struct {
	DetectHaiku              string `toml:"detect_haiku"`
	ExtractSonnet            string `toml:"extract_sonnet"`
	RefineSonnet             string `toml:"refine_sonnet"`
	SurfaceGateHaiku         string `toml:"surface_gate_haiku"`
	SurfaceInjectionPreamble string `toml:"surface_injection_preamble"`
	EvaluateHaiku            string `toml:"evaluate_haiku"`
	MaintainRewrite          string `toml:"maintain_rewrite"`
	MaintainConsolidate      string `toml:"maintain_consolidate"`
	AdaptSonnet              string `toml:"adapt_sonnet"`
}

// mergeMaintainParams overlays non-zero maintain/adapt pipeline values from params onto policy.
func mergeMaintainParams(pol *Policy, params policyFileParams) {
	if params.MaintainEffectivenessThreshold != 0 {
		pol.MaintainEffectivenessThreshold = params.MaintainEffectivenessThreshold
	}

	if params.MaintainMinSurfaced != 0 {
		pol.MaintainMinSurfaced = params.MaintainMinSurfaced
	}

	if params.MaintainIrrelevanceThreshold != 0 {
		pol.MaintainIrrelevanceThreshold = params.MaintainIrrelevanceThreshold
	}

	if params.MaintainNotFollowedThreshold != 0 {
		pol.MaintainNotFollowedThreshold = params.MaintainNotFollowedThreshold
	}

	if params.GateIrrelevanceThreshold != 0 {
		pol.GateIrrelevanceThreshold = params.GateIrrelevanceThreshold
	}

	if params.AdaptChangeHistoryLimit != 0 {
		pol.AdaptChangeHistoryLimit = params.AdaptChangeHistoryLimit
	}
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

	mergeSurfaceParams(pol, params)
	mergeMaintainParams(pol, params)
}

// mergePrompts overlays non-empty values from prompts onto policy.
func mergePrompts(pol *Policy, prompts policyFilePrompts) {
	if prompts.DetectHaiku != "" {
		pol.DetectHaikuPrompt = prompts.DetectHaiku
	}

	if prompts.ExtractSonnet != "" {
		pol.ExtractSonnetPrompt = prompts.ExtractSonnet
	}

	if prompts.RefineSonnet != "" {
		pol.RefineSonnetPrompt = prompts.RefineSonnet
	}

	if prompts.SurfaceGateHaiku != "" {
		pol.SurfaceGateHaikuPrompt = prompts.SurfaceGateHaiku
	}

	if prompts.SurfaceInjectionPreamble != "" {
		pol.SurfaceInjectionPreamble = prompts.SurfaceInjectionPreamble
	}

	if prompts.EvaluateHaiku != "" {
		pol.EvaluateHaikuPrompt = prompts.EvaluateHaiku
	}

	if prompts.MaintainRewrite != "" {
		pol.MaintainRewritePrompt = prompts.MaintainRewrite
	}

	if prompts.MaintainConsolidate != "" {
		pol.MaintainConsolidatePrompt = prompts.MaintainConsolidate
	}

	if prompts.AdaptSonnet != "" {
		pol.AdaptSonnetPrompt = prompts.AdaptSonnet
	}
}

// mergeSurfaceParams overlays non-zero surface pipeline values from params onto policy.
func mergeSurfaceParams(pol *Policy, params policyFileParams) {
	if params.SurfaceCandidateCountMin != 0 {
		pol.SurfaceCandidateCountMin = params.SurfaceCandidateCountMin
	}

	if params.SurfaceCandidateCountMax != 0 {
		pol.SurfaceCandidateCountMax = params.SurfaceCandidateCountMax
	}

	if params.SurfaceBM25Threshold != 0 {
		pol.SurfaceBM25Threshold = params.SurfaceBM25Threshold
	}

	if params.SurfaceColdStartBudget != 0 {
		pol.SurfaceColdStartBudget = params.SurfaceColdStartBudget
	}

	if params.SurfaceIrrelevanceHalfLife != 0 {
		pol.SurfaceIrrelevanceHalfLife = params.SurfaceIrrelevanceHalfLife
	}
}

// stripChangeHistory removes all [[change_history]] table array entries from TOML content.
func stripChangeHistory(content string) string {
	if content == "" {
		return ""
	}

	var result bytes.Buffer

	lines := bytes.Split([]byte(content), []byte("\n"))
	inChangeHistory := false

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if bytes.Equal(trimmed, []byte("[[change_history]]")) {
			inChangeHistory = true

			continue
		}

		// A new section header ends the change_history block.
		if inChangeHistory && len(trimmed) > 0 && trimmed[0] == '[' {
			inChangeHistory = false
		}

		if !inChangeHistory {
			result.Write(line)
			result.WriteByte('\n')
		}
	}

	// Trim trailing whitespace but keep one newline.
	output := bytes.TrimRight(result.Bytes(), "\n")
	if len(output) > 0 {
		return string(output) + "\n"
	}

	return ""
}
