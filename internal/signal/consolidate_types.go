package signal

import (
	"context"

	"engram/internal/memory"
)

// ActionType describes the consolidator's decision at an intervention point.
type ActionType int

// ActionType values.
const (
	// StoreAsIs means no cluster was found; store the memory normally.
	StoreAsIs ActionType = iota
	// Consolidated means a cluster was found and merged into a generalized memory.
	Consolidated
	// RefineKeywords means no cluster was found; keyword refinement is suggested.
	RefineKeywords
	// ProceedWithRemoval means no cluster was found; removal can proceed.
	ProceedWithRemoval
)

// Action is the result of a consolidation intervention point.
type Action struct {
	Type              ActionType
	ConsolidatedMem   *memory.MemoryRecord
	Archived          []string
	RefinementContext *RefinementContext
}

// Archiver moves memory files to an archive directory.
type Archiver interface {
	Archive(sourcePath string) error
}

// ConfirmedCluster is a group of memories confirmed by LLM to share a principle.
type ConfirmedCluster struct {
	Members   []*memory.MemoryRecord
	Principle string
}

// Confirmer asks an LLM whether candidate memories share a principle.
type Confirmer interface {
	ConfirmClusters(ctx context.Context, query *memory.MemoryRecord, candidates []ScoredCandidate) ([]ConfirmedCluster, error)
}

// Extractor creates a generalized memory from a confirmed cluster.
type Extractor interface {
	ExtractPrinciple(ctx context.Context, cluster ConfirmedCluster) (*memory.MemoryRecord, error)
}

// OnIrrelevantInput carries the memory and surfacing context for irrelevant feedback.
type OnIrrelevantInput struct {
	Memory         *memory.MemoryRecord
	SurfacingQuery string
	ToolName       string
	ToolInput      string
}

// RefinementContext carries surfacing context for keyword refinement (#346).
type RefinementContext struct {
	Memory          *memory.MemoryRecord
	SurfacingQuery  string
	MatchedKeywords []string
	ToolName        string
	ToolInput       string
}

// ScoredCandidate is a memory with its BM25 similarity score.
type ScoredCandidate struct {
	Memory *memory.MemoryRecord
	Score  float64
}

// Scorer retrieves candidate memories similar to a query memory.
type Scorer interface {
	FindSimilar(ctx context.Context, query *memory.MemoryRecord, exclude []string) ([]ScoredCandidate, error)
}
