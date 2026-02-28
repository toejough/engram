// Package reconcile handles merging new learnings with existing memories via overlap detection.
package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"engram/internal/store"
)

// Learning represents a new fact to be reconciled with existing memories.
type Learning struct {
	Content  string
	Keywords []string
	Title    string
}

// OverlapGate checks whether a candidate memory overlaps with a new learning.
type OverlapGate interface {
	Check(ctx context.Context, learning Learning, candidate store.Memory) (bool, string, error)
}

// Result describes the outcome of reconciling a new learning with the store.
type Result struct {
	Action   string
	MemoryID string
	Title    string
	Keywords []string
	Overlap  float64
}

// Store provides persistence and retrieval of memories.
type Store interface {
	FindSimilar(ctx context.Context, query string, k int) ([]store.ScoredMemory, error)
	Create(ctx context.Context, m *store.Memory) error
	Update(ctx context.Context, m *store.Memory) error
}

// Run reconciles a new learning with the store, merging into overlapping memories or creating a new one.
func Run(ctx context.Context, s Store, gate OverlapGate, k int, learning Learning) (Result, error) {
	query := learning.Content + " " + joinKeywords(learning.Keywords)

	candidates, err := s.FindSimilar(ctx, query, k)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile: find similar: %w", err)
	}

	for _, candidate := range candidates {
		overlap, _, err := gate.Check(ctx, learning, candidate.Memory)
		if err != nil {
			return Result{}, fmt.Errorf("reconcile: overlap gate: %w", err)
		}

		if overlap {
			merged := mergeMemory(candidate.Memory, learning)

			updateErr := s.Update(ctx, &merged)
			if updateErr != nil {
				return Result{}, fmt.Errorf("reconcile: update: %w", updateErr)
			}

			return Result{
				Action:   "enriched",
				MemoryID: merged.ID,
				Title:    merged.Title,
				Keywords: merged.Keywords,
				Overlap:  candidate.Score,
			}, nil
		}
	}

	m := newMemory(learning)

	createErr := s.Create(ctx, &m)
	if createErr != nil {
		return Result{}, fmt.Errorf("reconcile: create: %w", createErr)
	}

	return Result{
		Action:   "created",
		MemoryID: m.ID,
		Title:    m.Title,
		Keywords: m.Keywords,
	}, nil
}

// unexported constants.
const (
	idBitmask = 0xFFFFFFFF
)

func joinKeywords(kws []string) string {
	return strings.Join(kws, " ")
}

func mergeMemory(existing store.Memory, learning Learning) store.Memory {
	existing.EnrichmentCount++
	existing.UpdatedAt = time.Now().UTC()
	// Merge keywords (add new ones not already present)
	seen := make(map[string]bool, len(existing.Keywords))
	for _, kw := range existing.Keywords {
		seen[kw] = true
	}

	for _, kw := range learning.Keywords {
		if !seen[kw] {
			existing.Keywords = append(existing.Keywords, kw)
		}
	}

	return existing
}

func newMemory(learning Learning) store.Memory {
	now := time.Now().UTC()

	return store.Memory{
		ID:        fmt.Sprintf("m_%08x", now.UnixNano()&idBitmask),
		Title:     learning.Title,
		Content:   learning.Content,
		Keywords:  learning.Keywords,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
