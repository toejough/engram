// Package correct detects user corrections in messages and records them as memories.
package correct

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/corpus"
)

// Config contains dependencies for the Detector.
type Config struct {
	Reconciler Reconciler
	Corpus     *corpus.Corpus
}

// Detector detects user corrections in messages.
type Detector struct{}

// NewDetector creates a new Detector with the provided config.
func NewDetector(cfg Config) (*Detector, error) {
	var missing []string
	if cfg.Reconciler == nil {
		missing = append(missing, "Recon")
	}

	if cfg.Corpus == nil {
		missing = append(missing, "Corpus")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", errMissingDeps, strings.Join(missing, ", "))
	}

	return &Detector{}, nil
}

// Learning represents a correction to be learned from.
type Learning struct {
	Content  string
	Keywords []string
	Title    string
}

// Reclassifier reclassifies surfaced memories on correction detection (ARCH-13).
type Reclassifier interface {
	Reclassify(ctx context.Context) (int, error)
}

// ReconcileResult contains the outcome of a reconciliation.
type ReconcileResult struct {
	Action   string
	MemoryID string
	Title    string
}

// Reconciler reconciles corrections into memory.
type Reconciler interface {
	Reconcile(ctx context.Context, l Learning) (ReconcileResult, error)
}

// Recording records a correction as a memory.
type Recording struct {
	MemoryID string
	Content  string
}

// DetectCorrection detects and records user corrections in a message.
// If reclassifier is non-nil, it is invoked after a successful reconciliation
// to decrease impact scores for memories surfaced before this correction (ARCH-13).
func DetectCorrection(
	ctx context.Context,
	reconciler Reconciler,
	patterns *corpus.Corpus,
	reclassifier Reclassifier,
	message string,
) (reminder string, recordings []Recording, auditStr string, err error) {
	if patterns == nil {
		return "", nil, "", nil
	}

	m := patterns.Match(message)
	if m == nil {
		return "", nil, "", nil
	}

	learning := Learning{
		Content:  message,
		Keywords: []string{m.Pattern.Label},
		Title:    summarize(message),
	}

	result, err := reconciler.Reconcile(ctx, learning)
	if err != nil {
		return "", nil, "", fmt.Errorf("correct: reconcile: %w", err)
	}

	if reclassifier != nil {
		_, err = reclassifier.Reclassify(ctx)
		if err != nil {
			return "", nil, "", fmt.Errorf("correct: reclassify: %w", err)
		}
	}

	rec := Recording{MemoryID: result.MemoryID, Content: message}

	var audit strings.Builder

	fmt.Fprintf(&audit, "%s: %q pattern=%s", result.Action, learning.Title, m.Pattern.Label)

	var rem strings.Builder

	if result.Action == "enriched" {
		fmt.Fprintf(&rem, "Correction captured. Enriched: %s", result.Title)
	} else {
		fmt.Fprintf(&rem, "Correction captured. Created: %s", learning.Title)
	}

	return rem.String(), []Recording{rec}, audit.String(), nil
}

// unexported constants.
const (
	maxTitleWords = 8
)

// unexported variables.
var (
	errMissingDeps = errors.New("correct: missing dependencies")
)

func summarize(message string) string {
	words := strings.Fields(message)
	if len(words) > maxTitleWords {
		words = words[:maxTitleWords]
	}

	return strings.Join(words, " ")
}
