package maintain

import (
	"fmt"
	"path/filepath"
	"strings"

	"engram/internal/memory"
)

// DiagnosisConfig holds thresholds for the per-memory health decision tree.
type DiagnosisConfig struct {
	MinSurfaced            int
	EffectivenessThreshold float64
	IrrelevanceThreshold   float64
	NotFollowedThreshold   float64
}

// Diagnose evaluates a single memory record and returns a Proposal if maintenance
// is needed, or nil if the memory should be left alone (insufficient data, working,
// or ambiguous).
func Diagnose(path string, record *memory.MemoryRecord, cfg DiagnosisConfig) *Proposal {
	if record.SurfacedCount < cfg.MinSurfaced {
		return nil
	}

	surfaced := float64(record.SurfacedCount)
	effectiveness := float64(record.FollowedCount) / surfaced * percentScale
	irrelevantRate := float64(record.IrrelevantCount) / surfaced * percentScale
	notFollowedRate := float64(record.NotFollowedCount) / surfaced * percentScale
	name := memoryNameFromPath(path)

	// Priority 2: low effectiveness + high irrelevance
	if effectiveness < cfg.EffectivenessThreshold && irrelevantRate >= cfg.IrrelevanceThreshold {
		return diagnoseRemove(path, name, effectiveness, irrelevantRate)
	}

	// Priority 3: high irrelevance only
	if irrelevantRate >= cfg.IrrelevanceThreshold {
		return diagnoseNarrowSituation(path, name, irrelevantRate)
	}

	// Priority 4: high not-followed rate
	if notFollowedRate >= cfg.NotFollowedThreshold {
		return diagnoseNotFollowed(path, name, record.SurfacedCount, cfg.MinSurfaced, notFollowedRate)
	}

	// Priority 5+6: working or ambiguous — no action needed
	return nil
}

// DiagnoseAll runs the decision tree on each record and returns only non-nil proposals.
func DiagnoseAll(records []memory.StoredRecord, cfg DiagnosisConfig) []Proposal {
	proposals := make([]Proposal, 0, len(records))

	for idx := range records {
		proposal := Diagnose(records[idx].Path, &records[idx].Record, cfg)
		if proposal != nil {
			proposals = append(proposals, *proposal)
		}
	}

	return proposals
}

// unexported constants.
const (
	escalationMultiplier = 2
	percentScale         = 100.0
)

func diagnoseNarrowSituation(path, name string, irrelevantRate float64) *Proposal {
	return &Proposal{
		ID:     fmt.Sprintf("diag-%s-narrow", name),
		Action: ActionUpdate,
		Target: path,
		Field:  "situation",
		Rationale: fmt.Sprintf(
			"irrelevant rate %.0f%% — situation field is too broad",
			irrelevantRate,
		),
	}
}

func diagnoseNotFollowed(
	path, name string,
	surfacedCount, minSurfaced int,
	notFollowedRate float64,
) *Proposal {
	// 4a: persistent not-followed with sufficient data
	if surfacedCount >= escalationMultiplier*minSurfaced {
		return &Proposal{
			ID:     fmt.Sprintf("diag-%s-escalate", name),
			Action: ActionRecommend,
			Target: path,
			Rationale: fmt.Sprintf(
				"not-followed rate %.0f%% over %d surfacings — persistent non-compliance",
				notFollowedRate, surfacedCount,
			),
		}
	}

	// 4b: less data
	return &Proposal{
		ID:     fmt.Sprintf("diag-%s-rewrite", name),
		Action: ActionUpdate,
		Target: path,
		Field:  "action",
		Rationale: fmt.Sprintf(
			"not-followed rate %.0f%% — action field may need clearer guidance",
			notFollowedRate,
		),
	}
}

func diagnoseRemove(path, name string, effectiveness, irrelevantRate float64) *Proposal {
	return &Proposal{
		ID:     fmt.Sprintf("diag-%s-remove", name),
		Action: ActionDelete,
		Target: path,
		Rationale: fmt.Sprintf(
			"effectiveness %.0f%% below threshold, irrelevant rate %.0f%% — memory is not useful",
			effectiveness, irrelevantRate,
		),
	}
}

// memoryNameFromPath extracts the base filename without extension.
func memoryNameFromPath(path string) string {
	base := filepath.Base(path)

	return strings.TrimSuffix(base, filepath.Ext(base))
}
