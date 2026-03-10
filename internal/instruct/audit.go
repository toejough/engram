package instruct

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// AuditReport is the full output of the instruction quality audit pipeline.
type AuditReport struct {
	Duplicates []DuplicatePair `json:"duplicates"`
	Diagnoses  json.RawMessage `json:"diagnoses"`
	Proposals  json.RawMessage `json:"proposals"`
	Gaps       []GapCandidate  `json:"gaps"`
}

// Auditor runs the instruction quality audit pipeline.
type Auditor struct {
	Scanner   *Scanner
	LLMCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	EvalData  []EvalRecord
}

// Run executes the full audit pipeline.
func (a *Auditor) Run(ctx context.Context, dataDir, projectDir string) (*AuditReport, error) {
	items, err := a.Scanner.ScanAll(dataDir, projectDir)
	if err != nil {
		return nil, fmt.Errorf("scanning instructions: %w", err)
	}

	report := &AuditReport{}

	// Step 1: Deduplication (always runs)
	report.Duplicates = findDuplicates(items)

	// Step 2: Diagnosis (requires LLM)
	if a.LLMCaller == nil {
		skipped, marshalErr := json.Marshal(SkippedSection{SkippedReason: "no API token"})
		if marshalErr != nil {
			return nil, fmt.Errorf("marshaling skipped section: %w", marshalErr)
		}

		report.Diagnoses = skipped
		report.Proposals = skipped

		return a.finishReport(report, items), nil
	}

	diagnoses, diagErr := a.diagnoseBottom(ctx, items)
	if diagErr != nil {
		return nil, fmt.Errorf("diagnosing instructions: %w", diagErr)
	}

	diagJSON, marshalErr := json.Marshal(diagnoses)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshaling diagnoses: %w", marshalErr)
	}

	report.Diagnoses = diagJSON

	// Step 3: Proposals from diagnoses
	proposals := buildProposals(diagnoses)

	propJSON, propMarshalErr := json.Marshal(proposals)
	if propMarshalErr != nil {
		return nil, fmt.Errorf("marshaling proposals: %w", propMarshalErr)
	}

	report.Proposals = propJSON

	return a.finishReport(report, items), nil
}

// diagnoseBottom sends the bottom 20% (by effectiveness) to the LLM for diagnosis.
func (a *Auditor) diagnoseBottom(
	ctx context.Context,
	items []InstructionItem,
) ([]Diagnosis, error) {
	if len(items) == 0 {
		return []Diagnosis{}, nil
	}

	// Filter to items with effectiveness data (score > 0 means data exists)
	scored := make([]InstructionItem, 0, len(items))

	scored = append(scored, items...)

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].EffectivenessScore < scored[j].EffectivenessScore
	})

	bottomCount := int(math.Ceil(float64(len(scored)) * bottomPercentile))
	bottom := scored[:bottomCount]

	diagnoses := make([]Diagnosis, 0, bottomCount)

	for _, item := range bottom {
		userPrompt := fmt.Sprintf(
			"Instruction source: %s\nPath: %s\nContent:\n%s",
			item.Source, item.Path, item.Content,
		)

		resp, err := a.LLMCaller(ctx, haikuModel, diagnosisPrompt, userPrompt)
		if err != nil {
			return nil, fmt.Errorf("calling LLM for %s: %w", item.Path, err)
		}

		var parsed diagnosisResponse

		unmarshalErr := json.Unmarshal([]byte(resp), &parsed)
		if unmarshalErr != nil {
			return nil, fmt.Errorf("parsing LLM response for %s: %w", item.Path, unmarshalErr)
		}

		diagnoses = append(diagnoses, Diagnosis{
			Path:       item.Path,
			Diagnosis:  parsed.Diagnosis,
			RootCause:  parsed.RootCause,
			Suggestion: parsed.Suggestion,
		})
	}

	return diagnoses, nil
}

// findGaps finds contradicted evaluation patterns not covered by existing instructions.
func (a *Auditor) findGaps(items []InstructionItem) []GapCandidate {
	gaps := make([]GapCandidate, 0)

	if len(a.EvalData) == 0 {
		return gaps
	}

	// Build set of covered memory paths
	covered := make(map[string]bool, len(items))
	for _, item := range items {
		covered[item.Path] = true
	}

	// Group contradictions by pattern
	type patternInfo struct {
		count   int
		example string
	}

	patternMap := make(map[string]*patternInfo)

	for _, rec := range a.EvalData {
		if rec.Outcome != "contradicted" {
			continue
		}

		if covered[rec.MemoryPath] {
			continue
		}

		info, exists := patternMap[rec.Pattern]
		if !exists {
			info = &patternInfo{example: rec.Example}
			patternMap[rec.Pattern] = info
		}

		info.count++
	}

	for pattern, info := range patternMap {
		gaps = append(gaps, GapCandidate{
			Pattern:        pattern,
			ViolationCount: info.count,
			Example:        info.example,
		})
	}

	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].ViolationCount > gaps[j].ViolationCount
	})

	return gaps
}

func (a *Auditor) finishReport(report *AuditReport, items []InstructionItem) *AuditReport {
	report.Gaps = a.findGaps(items)

	return report
}

// Diagnosis holds an LLM-generated diagnosis for a low-performing instruction.
//
//nolint:tagliatelle // external JSON API contract
type Diagnosis struct {
	Path       string `json:"path"`
	Diagnosis  string `json:"diagnosis"`
	RootCause  string `json:"root_cause"`
	Suggestion string `json:"suggestion"`
}

// DuplicatePair identifies two memory instructions with overlapping content.
//
//nolint:tagliatelle // external JSON API contract
type DuplicatePair struct {
	PathA   string  `json:"path_a"`
	PathB   string  `json:"path_b"`
	Overlap float64 `json:"overlap"`
}

// EvalRecord captures the outcome of evaluating a memory against a transcript.
//
//nolint:tagliatelle // external JSON API contract
type EvalRecord struct {
	MemoryPath string `json:"memory_path"`
	Outcome    string `json:"outcome"`
	Pattern    string `json:"pattern"`
	Example    string `json:"example"`
}

// GapCandidate represents a recurring violation pattern not yet covered by a memory.
//
//nolint:tagliatelle // external JSON API contract
type GapCandidate struct {
	Pattern        string `json:"pattern"`
	ViolationCount int    `json:"violation_count"`
	Example        string `json:"example"`
}

// PerLineEffectiveness maps "path:line" to follow rate percentage.
type PerLineEffectiveness map[string]float64

// RefinementProposal suggests an action to improve a low-performing instruction.
//
//nolint:tagliatelle // external JSON API contract
type RefinementProposal struct {
	Path       string `json:"path"`
	Action     string `json:"action"`
	RootCause  string `json:"root_cause"`
	Suggestion string `json:"suggestion"`
}

// SkippedSection records a section that was excluded from auditing.
//
//nolint:tagliatelle // external JSON API contract
type SkippedSection struct {
	SkippedReason string `json:"skipped_reason"`
}

// unexported constants.
const (
	bottomPercentile = 0.20
	diagnosisPrompt  = "You are diagnosing why an instruction is ineffective. Common root causes:\n" +
		"- Too abstract, framing mismatch, missing trigger, too narrow, too verbose\n" +
		"Output JSON: {\"diagnosis\": \"...\", \"root_cause\": \"...\", \"suggestion\": \"...\"}"
	dupThreshold = 0.80
	haikuModel   = "claude-haiku-4-5-20251001"
)

// diagnosisResponse is the expected JSON from the LLM.
type diagnosisResponse struct {
	Diagnosis  string `json:"diagnosis"`
	RootCause  string `json:"root_cause"` //nolint:tagliatelle // LLM JSON contract
	Suggestion string `json:"suggestion"`
}

// buildProposals converts diagnoses into maintain-compatible proposals.
func buildProposals(diagnoses []Diagnosis) []RefinementProposal {
	proposals := make([]RefinementProposal, 0, len(diagnoses))

	for _, diag := range diagnoses {
		proposals = append(proposals, RefinementProposal{
			Path:       diag.Path,
			Action:     "rewrite",
			RootCause:  diag.RootCause,
			Suggestion: diag.Suggestion,
		})
	}

	return proposals
}

// extractKeywords splits content into lowercase word tokens for overlap calculation.
func extractKeywords(content string) map[string]bool {
	words := strings.Fields(strings.ToLower(content))
	keywords := make(map[string]bool, len(words))

	for _, word := range words {
		// Strip punctuation
		word = strings.Trim(word, ".,;:!?\"'`()[]{}#*-_=+<>/\\|~@$%^&")
		if len(word) > 1 { // skip single chars
			keywords[word] = true
		}
	}

	return keywords
}

// findDuplicates detects memory pairs with >80% keyword overlap.
func findDuplicates(items []InstructionItem) []DuplicatePair {
	pairs := make([]DuplicatePair, 0)

	for i := range items {
		kwA := extractKeywords(items[i].Content)

		for j := i + 1; j < len(items); j++ {
			kwB := extractKeywords(items[j].Content)
			overlap := keywordOverlap(kwA, kwB)

			if overlap > dupThreshold {
				pairs = append(pairs, DuplicatePair{
					PathA:   items[i].Path,
					PathB:   items[j].Path,
					Overlap: overlap,
				})
			}
		}
	}

	return pairs
}

// keywordOverlap computes Jaccard similarity between two keyword sets.
func keywordOverlap(setA, setB map[string]bool) float64 {
	if len(setA) == 0 && len(setB) == 0 {
		return 0
	}

	intersection := 0

	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
