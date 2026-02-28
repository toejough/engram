// Package extract enriches, classifies, and reconciles learnings from transcripts.
package extract

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Classifier categorizes raw learnings by observation type.
type Classifier interface {
	Classify(ctx context.Context, learning RawLearning, transcript []byte) (string, error)
}

// Config holds dependencies for the Extractor.
type Config struct {
	Enricher   Enricher
	Classifier Classifier
	Reconciler Reconciler
	Overlaps   SessionOverlaps
}

// Enricher extracts raw learnings from a transcript.
type Enricher interface {
	Enrich(ctx context.Context, transcript []byte) ([]RawLearning, error)
}

// Extractor enriches, classifies, and reconciles learnings from transcripts.
type Extractor struct{}

// NewExtractor creates a new Extractor with the given config, validating required dependencies.
func NewExtractor(cfg Config) (*Extractor, error) {
	var missing []string

	if cfg.Enricher == nil {
		missing = append(missing, "Enricher")
	}

	if cfg.Classifier == nil {
		missing = append(missing, "Classifier")
	}

	if cfg.Reconciler == nil {
		missing = append(missing, "Reconciler")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", errMissingDeps, strings.Join(missing, ", "))
	}

	return &Extractor{}, nil
}

// Learning represents a processed learning extracted from a transcript.
type Learning struct {
	Content  string
	Keywords []string
	Title    string
}

// RawLearning represents an unprocessed learning extracted directly from a transcript.
type RawLearning struct {
	Content         string
	Title           string
	ObservationType string
	Concepts        []string
	Principle       string
	AntiPattern     string
	Rationale       string
	Keywords        []string
}

// ReconcileResult represents the outcome of reconciling a learning.
type ReconcileResult struct {
	Action   string
	MemoryID string
}

// Reconciler reconciles processed learnings into the memory store.
type Reconciler interface {
	Reconcile(ctx context.Context, l Learning) (ReconcileResult, error)
}

// SessionOverlaps checks whether content overlaps with learnings already captured in the current session.
type SessionOverlaps interface {
	HasOverlap(content string) bool
}

// Run extracts learnings from a transcript by enriching, classifying, and reconciling them, returning an audit log.
func Run(
	ctx context.Context,
	enricher Enricher,
	classifier Classifier,
	reconciler Reconciler,
	overlaps SessionOverlaps,
	transcript []byte,
) (string, error) {
	learnings, err := enricher.Enrich(ctx, transcript)
	if err != nil {
		return "", fmt.Errorf("extract: enrich: %w", err)
	}

	if len(learnings) == 0 {
		return "", nil
	}

	var auditLines []string

	for _, rawLearning := range learnings {
		// Quality gate: reject if content < minTokens tokens
		if tokenCount(rawLearning.Content) < minTokens {
			auditLines = append(
				auditLines,
				fmt.Sprintf("rejected: %q (below quality threshold)", rawLearning.Title),
			)

			continue
		}

		// Dedup: skip if already captured mid-session
		if overlaps != nil && overlaps.HasOverlap(rawLearning.Content) {
			auditLines = append(
				auditLines,
				fmt.Sprintf("skipped: %q (mid-session dedup)", rawLearning.Title),
			)

			continue
		}

		// Classify
		tier, err := classifier.Classify(ctx, rawLearning, transcript)
		if err != nil {
			return strings.Join(auditLines, "\n"), fmt.Errorf("extract: classify: %w", err)
		}

		// Reconcile
		learning := Learning{
			Content:  rawLearning.Content,
			Keywords: rawLearning.Keywords,
			Title:    rawLearning.Title,
		}

		result, err := reconciler.Reconcile(ctx, learning)
		if err != nil {
			return strings.Join(auditLines, "\n"), fmt.Errorf("extract: reconcile: %w", err)
		}

		auditLines = append(
			auditLines,
			fmt.Sprintf("%s: %q confidence=%s", result.Action, rawLearning.Title, tier),
		)
	}

	return strings.Join(auditLines, "\n"), nil
}

// unexported constants.
const (
	minTokens = 10
)

// unexported variables.
var (
	errMissingDeps = errors.New("extract: missing dependencies")
)

func tokenCount(s string) int {
	return len(strings.Fields(s))
}
