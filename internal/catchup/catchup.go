// Package catchup finds missed corrections in transcripts and records them.
package catchup

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CandidatePattern holds a regex pattern for matching missed corrections.
type CandidatePattern struct {
	Regex string
}

// CapturedEvent records a correction that was already detected mid-session.
type CapturedEvent struct {
	MemoryID string
	Pattern  string
	Message  string
}

// Config holds dependencies for the catchup Processor.
type Config struct {
	Evaluator  Evaluator
	Reconciler Reconciler
}

// Evaluator finds corrections that were missed during a session.
type Evaluator interface {
	FindMissed(
		ctx context.Context,
		transcript []byte,
		captured []CapturedEvent,
	) ([]MissedCorrection, error)
}

// Learning holds the content to reconcile into a memory.
type Learning struct {
	Content  string
	Keywords []string
	Title    string
}

// MissedCorrection represents a correction that was not caught during the session.
type MissedCorrection struct {
	Content string
	Context string
	Phrase  string
}

// Processor finds missed corrections and reconciles them into memory.
type Processor struct{}

// NewProcessor creates a Processor with the given config, validating that required dependencies are present.
func NewProcessor(cfg Config) (*Processor, error) {
	var missing []string

	if cfg.Evaluator == nil {
		missing = append(missing, "Evaluator")
	}

	if cfg.Reconciler == nil {
		missing = append(missing, "Reconciler")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", errMissingDeps, strings.Join(missing, ", "))
	}

	return &Processor{}, nil
}

// ReconcileResult holds the outcome of reconciling a learned correction.
type ReconcileResult struct {
	Action   string
	MemoryID string
}

// Reconciler reconciles a learned correction into memory.
type Reconciler interface {
	Reconcile(ctx context.Context, l Learning) (ReconcileResult, error)
}

// Run evaluates a transcript for missed corrections and reconciles them into memory.
func Run(
	ctx context.Context,
	evaluator Evaluator,
	reconciler Reconciler,
	captured []CapturedEvent,
	transcript []byte,
) ([]CandidatePattern, string, error) {
	missed, err := evaluator.FindMissed(ctx, transcript, captured)
	if err != nil {
		return nil, "", fmt.Errorf("catchup: find missed: %w", err)
	}

	if len(missed) == 0 {
		return nil, "", nil
	}

	var candidates []CandidatePattern

	var auditLines []string

	for _, correction := range missed {
		learning := Learning{
			Content:  correction.Content,
			Keywords: []string{"correction", "missed"},
			Title:    summarize(correction.Content),
		}

		result, reconcileErr := reconciler.Reconcile(ctx, learning)
		if reconcileErr != nil {
			return candidates, strings.Join(
					auditLines,
					"\n",
				), fmt.Errorf(
					"catchup: reconcile: %w",
					reconcileErr,
				)
		}

		candidates = append(candidates, CandidatePattern{Regex: correction.Phrase})
		auditLines = append(
			auditLines,
			fmt.Sprintf("catchup %s: %q phrase=%q", result.Action, learning.Title, correction.Phrase),
		)
	}

	return candidates, strings.Join(auditLines, "\n"), nil
}

// unexported constants.
const (
	maxTitleWords = 8
)

// unexported variables.
var (
	errMissingDeps = errors.New("catchup: missing dependencies")
)

func summarize(s string) string {
	words := strings.Fields(s)
	if len(words) > maxTitleWords {
		words = words[:maxTitleWords]
	}

	return strings.Join(words, " ")
}
