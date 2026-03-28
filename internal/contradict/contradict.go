// Package contradict implements cross-source contradiction detection (UC-P1-1).
// Two-pass detection: keyword heuristic then BM25 similarity, with optional LLM classifier fallback.
package contradict

import (
	"context"
	"strings"

	"engram/internal/memory"
)

// Classifier makes LLM calls to verify whether two memories contradict (REQ-P1-3).
type Classifier interface {
	Classify(ctx context.Context, a, b *memory.Stored) (bool, error)
}

// Detector detects contradicting memory pairs (REQ-P1-1).
type Detector struct {
	classifier  Classifier
	maxLLMCalls int
}

// NewDetector constructs a Detector with optional configuration (DES-P1-1).
func NewDetector(opts ...DetectorOption) *Detector {
	d := &Detector{maxLLMCalls: defaultMaxLLMCalls}
	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Check detects contradicting pairs among candidates (REQ-P1-1).
// Pass 1: keyword heuristic — fires on opposing-verb patterns (REQ-P1-2).
// Pass 2: Jaccard similarity — borderline pairs sent to LLM classifier.
// Input order determines rank: index 0 is highest-ranked.
//
//nolint:cyclop,funlen // two-pass detection with LLM budget: inherent branching
func (d *Detector) Check(ctx context.Context, candidates []*memory.Stored) ([]Pair, error) {
	if len(candidates) < minCandidates {
		return nil, nil
	}

	type pairCandidate struct {
		a, b       *memory.Stored
		ai, bi     int // indices into texts slice
		heuristic  bool
		similarity float64
	}

	// Pre-compute lowered search text for all candidates to avoid O(N²) allocations.
	texts := make([]string, len(candidates))
	for i, c := range candidates {
		texts[i] = strings.ToLower(c.SearchText())
	}

	pairCandidates := make([]pairCandidate, 0, len(candidates)*(len(candidates)-1)/pairDivisor)

	// Pass 1: keyword heuristic over all pairs (REQ-P1-2).
	for i := range candidates {
		for j := i + 1; j < len(candidates); j++ {
			fires := heuristicFires(texts[i], texts[j])
			pairCandidates = append(pairCandidates, pairCandidate{
				a:         candidates[i],
				b:         candidates[j],
				ai:        i,
				bi:        j,
				heuristic: fires,
			})
		}
	}

	// Pass 2: token-overlap similarity for pairs that didn't fire heuristic (REQ-P1-2).
	for idx := range pairCandidates {
		pc := &pairCandidates[idx]
		if pc.heuristic {
			continue // heuristic is sufficient, skip similarity check
		}

		pc.similarity = jaccardSimilarity(texts[pc.ai], texts[pc.bi])
	}

	// Classify pairs and resolve (REQ-P1-2, REQ-P1-3).
	llmBudget := d.maxLLMCalls
	pairs := make([]Pair, 0)

	for _, pc := range pairCandidates {
		switch {
		case pc.heuristic:
			// Heuristic fires: confirmed contradiction, skip LLM (T-P1-2, T-P1-3, T-P1-8).
			pairs = append(pairs, Pair{A: pc.a, B: pc.b, Confidence: heuristicConfidence})

		case pc.similarity > similarityThresh && d.classifier != nil && llmBudget > 0:
			// Borderline similarity: ask LLM (T-P1-5, T-P1-6).
			ok, err := d.classifier.Classify(ctx, pc.a, pc.b)
			llmBudget--

			if err != nil {
				// Fire-and-forget: treat error as non-contradiction (T-P1-7).
				continue
			}

			if ok {
				pairs = append(pairs, Pair{A: pc.a, B: pc.b, Confidence: llmConfidence})
			}
		}
	}

	return pairs, nil
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector)

// Pair identifies two memories that contradict each other (REQ-P1-1).
// A is higher-ranked (kept), B is lower-ranked (suppressed).
type Pair struct {
	A          *memory.Stored
	B          *memory.Stored
	Confidence float64 // 0–1
}

// WithClassifier sets the LLM classifier for borderline pair resolution (REQ-P1-3).
func WithClassifier(c Classifier) DetectorOption {
	return func(d *Detector) { d.classifier = c }
}

// WithMaxLLMCalls sets the maximum number of LLM classifier calls per Check invocation (REQ-P1-3).
func WithMaxLLMCalls(n int) DetectorOption {
	return func(d *Detector) { d.maxLLMCalls = n }
}

// unexported constants.
const (
	defaultMaxLLMCalls  = 3
	heuristicConfidence = 0.9
	llmConfidence       = 0.7
	minCandidates       = 2
	pairDivisor         = 2
	similarityThresh    = 0.3
)

// unexported variables.
var (
	// opposingPairs lists bidirectional opposing-verb patterns (DES-P1-2).
	// Each entry is [positive, negative]. Match fires if one text contains positive
	// and the other contains negative (in either direction).
	opposingPairs = [][2]string{ //nolint:gochecknoglobals // compile-time constant table; no mutable state
		{"use", "avoid"},
		{"always", "never"},
		{"should not", "should"},
		{"do not", "do"},
		{"enable", "disable"},
		{"is not", "is"},
	}
)

// heuristicFires returns true if the two lowered search texts contain opposing verb patterns (REQ-P1-2).
func heuristicFires(textA, textB string) bool {
	for _, pair := range opposingPairs {
		pos, neg := pair[0], pair[1]
		// Bidirectional check.
		if (strings.Contains(textA, pos) && strings.Contains(textB, neg)) ||
			(strings.Contains(textA, neg) && strings.Contains(textB, pos)) {
			return true
		}
	}

	return false
}

// jaccardSimilarity returns the Jaccard token-overlap coefficient for two lowered texts (0–1).
// Used as a corpus-independent similarity metric in the second-pass borderline detection.
// Callers must pass already-lowered text.
func jaccardSimilarity(a, b string) float64 {
	tokensA := strings.Fields(a)
	tokensB := strings.Fields(b)

	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	setA := make(map[string]bool, len(tokensA))
	for _, t := range tokensA {
		setA[t] = true
	}

	intersection := 0

	for _, t := range tokensB {
		if setA[t] {
			intersection++
		}
	}

	union := len(setA)

	for _, t := range tokensB {
		if !setA[t] {
			union++
		}
	}

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
