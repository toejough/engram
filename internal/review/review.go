// Package review classifies memories into a 2x2 effectiveness matrix.
package review

import (
	"fmt"
	"io"
	"sort"

	"engram/internal/effectiveness"
)

// Exported constants.
const (
	HiddenGem        Quadrant = "Hidden Gem"
	InsufficientData Quadrant = "Insufficient Data"
	Leech            Quadrant = "Leech"
	Noise            Quadrant = "Noise"
	Working          Quadrant = "Working"
)

// ClassifiedMemory holds a memory's quadrant assignment and stats.
type ClassifiedMemory struct {
	Name               string
	Quadrant           Quadrant
	SurfacedCount      int
	EffectivenessScore float64
	EvaluationCount    int
	Flagged            bool
}

// ClassifyOption is a functional option for Classify.
type ClassifyOption func(*classifyConfig)

// Quadrant represents a position in the 2x2 effectiveness matrix.
type Quadrant string

// TrackingData holds surfacing tracking fields for a memory.
type TrackingData struct {
	SurfacedCount int
}

// Classify takes effectiveness stats and tracking data, returns classified memories
// sorted by Flagged descending then EffectivenessScore ascending.
func Classify(
	stats map[string]effectiveness.Stat,
	tracking map[string]TrackingData,
	opts ...ClassifyOption,
) []ClassifiedMemory {
	cfg := defaultClassifyConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	keys := unionKeys(stats, tracking)
	if len(keys) == 0 {
		return []ClassifiedMemory{}
	}

	median := computeMedian(tracking)
	result := make([]ClassifiedMemory, 0, len(keys))

	for _, name := range keys {
		stat := stats[name]
		trackData := tracking[name]
		total := stat.FollowedCount + stat.ContradictedCount + stat.IgnoredCount

		mem := ClassifiedMemory{
			Name:               name,
			SurfacedCount:      trackData.SurfacedCount,
			EffectivenessScore: stat.EffectivenessScore,
			EvaluationCount:    total,
		}

		if total < cfg.minEvaluations {
			mem.Quadrant = InsufficientData
		} else {
			mem.Quadrant = assignQuadrant(trackData.SurfacedCount, stat.EffectivenessScore, median, cfg.effectivenessThreshold)
			mem.Flagged = stat.EffectivenessScore < cfg.flagThreshold
		}

		result = append(result, mem)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Flagged != result[j].Flagged {
			return result[i].Flagged
		}

		return result[i].EffectivenessScore < result[j].EffectivenessScore
	})

	return result
}

// Render writes a human-readable effectiveness review to w per DES-16.
func Render(classified []ClassifiedMemory, w io.Writer) {
	total, sufficient, flaggedCount, quadrantCounts := summarize(classified)

	_, _ = fmt.Fprintf(w, "[engram] Memory Effectiveness Review\n")
	_, _ = fmt.Fprintf(w, "  Total: %d memories, %d with sufficient data, %d flagged\n",
		total, sufficient, flaggedCount)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Quadrant Summary:\n")
	_, _ = fmt.Fprintf(
		w,
		"    Working:    %d  (often surfaced, high follow-through)\n",
		quadrantCounts[Working],
	)
	_, _ = fmt.Fprintf(
		w,
		"    Hidden Gem: %d  (rarely surfaced, high follow-through)\n",
		quadrantCounts[HiddenGem],
	)
	_, _ = fmt.Fprintf(
		w,
		"    Leech:      %d  (often surfaced, low follow-through)\n",
		quadrantCounts[Leech],
	)
	_, _ = fmt.Fprintf(
		w,
		"    Noise:      %d  (rarely surfaced, low follow-through)\n",
		quadrantCounts[Noise],
	)

	renderFlaggedSection(classified, w)
	renderInsufficientSection(classified, w)
}

// WithEffectivenessThreshold overrides the quadrant boundary (default 50.0).
func WithEffectivenessThreshold(threshold float64) ClassifyOption {
	return func(cfg *classifyConfig) {
		cfg.effectivenessThreshold = threshold
	}
}

// WithFlagThreshold overrides the flagging boundary (default 40.0).
func WithFlagThreshold(threshold float64) ClassifyOption {
	return func(cfg *classifyConfig) {
		cfg.flagThreshold = threshold
	}
}

// WithMinEvaluations overrides the minimum feedback events required (default 5).
func WithMinEvaluations(minEvals int) ClassifyOption {
	return func(cfg *classifyConfig) {
		cfg.minEvaluations = minEvals
	}
}

// unexported constants.
const (
	effectivenessThreshold = 50.0 // >= 50% → high follow-through
	flagThreshold          = 40.0 // < 40% → flagged for action
	minEvaluations         = 5
)

// classifyConfig holds configuration for Classify.
type classifyConfig struct {
	effectivenessThreshold float64
	flagThreshold          float64
	minEvaluations         int
}

// assignQuadrant returns the quadrant for a memory with sufficient data.
func assignQuadrant(surfaced int, score, median, effThreshold float64) Quadrant {
	aboveMedian := float64(surfaced) > median
	highFollowThrough := score >= effThreshold

	switch {
	case aboveMedian && highFollowThrough:
		return Working
	case !aboveMedian && highFollowThrough:
		return HiddenGem
	case aboveMedian && !highFollowThrough:
		return Leech
	default:
		return Noise
	}
}

// computeMedian computes the median surfaced count from all tracking entries.
// Returns 0 if there are no tracking entries.
func computeMedian(tracking map[string]TrackingData) float64 {
	if len(tracking) == 0 {
		return 0
	}

	counts := make([]int, 0, len(tracking))
	for _, trackData := range tracking {
		counts = append(counts, trackData.SurfacedCount)
	}

	sort.Ints(counts)

	n := len(counts)
	if n%2 == 1 {
		return float64(counts[n/2])
	}

	return float64(counts[n/2-1]+counts[n/2]) / 2.0 //nolint:mnd // standard even-n median formula
}

// defaultClassifyConfig returns the default configuration for Classify.
func defaultClassifyConfig() classifyConfig {
	return classifyConfig{
		effectivenessThreshold: effectivenessThreshold,
		flagThreshold:          flagThreshold,
		minEvaluations:         minEvaluations,
	}
}

// renderFlaggedSection writes the flagged-memories section if any exist.
func renderFlaggedSection(classified []ClassifiedMemory, w io.Writer) {
	var flaggedMems []ClassifiedMemory

	for _, mem := range classified {
		if mem.Flagged {
			flaggedMems = append(flaggedMems, mem)
		}
	}

	if len(flaggedMems) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Flagged for action (effectiveness < 40%%, 5+ evaluations):\n")

	for _, mem := range flaggedMems {
		_, _ = fmt.Fprintf(w,
			"    %-30s %-12s surfaced: %-4d effectiveness: %.0f%%  evaluations: %d\n",
			mem.Name, string(mem.Quadrant), mem.SurfacedCount,
			mem.EffectivenessScore, mem.EvaluationCount)
	}
}

// renderInsufficientSection writes the insufficient-data section if any exist.
func renderInsufficientSection(classified []ClassifiedMemory, w io.Writer) {
	var insufficientMems []ClassifiedMemory

	for _, mem := range classified {
		if mem.Quadrant == InsufficientData {
			insufficientMems = append(insufficientMems, mem)
		}
	}

	if len(insufficientMems) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Insufficient data (< 5 evaluations):\n")

	for _, mem := range insufficientMems {
		_, _ = fmt.Fprintf(w, "    %-30s surfaced: %-4d evaluations: %d\n",
			mem.Name, mem.SurfacedCount, mem.EvaluationCount)
	}
}

// summarize computes aggregate counts for the review header.
func summarize(
	classified []ClassifiedMemory,
) (total, sufficient, flaggedCount int, quadrantCounts map[Quadrant]int) {
	total = len(classified)
	quadrantCounts = make(map[Quadrant]int, len(classified))

	for _, mem := range classified {
		if mem.Quadrant != InsufficientData {
			sufficient++
		}

		if mem.Flagged {
			flaggedCount++
		}

		quadrantCounts[mem.Quadrant]++
	}

	return total, sufficient, flaggedCount, quadrantCounts
}

// unionKeys returns the union of keys from both maps, in sorted order.
func unionKeys(stats map[string]effectiveness.Stat, tracking map[string]TrackingData) []string {
	seen := make(map[string]struct{}, len(stats)+len(tracking))

	for key := range stats {
		seen[key] = struct{}{}
	}

	for key := range tracking {
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
