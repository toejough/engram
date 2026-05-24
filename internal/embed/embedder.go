package embed

import (
	"context"
	"errors"
)

// Embedder produces fixed-dimension dense vectors from text. Implementations
// are expected to be safe for concurrent use unless documented otherwise.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	ModelID() string
	Dims() int
}

// Sidecar is the on-disk shape of a per-note .vec.json file. Field order
// here is the JSON key order.
type Sidecar struct {
	EmbeddingModelID string    `json:"embedding_model_id"`
	Dims             int       `json:"dims"`
	Vector           []float32 `json:"vector"`
	ContentHash      string    `json:"content_hash"`
}

// State is the relationship between a note and its sidecar relative to
// the current binary's embedder.
type State int

// State values.
const (
	StateOK State = iota
	StateMissing
	StateStale
	StateIncompatible
	StateBroken
)

// String returns the lowercase label for s used by `engram embed status`.
func (s State) String() string {
	switch s {
	case StateOK:
		return "ok"
	case StateMissing:
		return "missing"
	case StateStale:
		return "stale"
	case StateIncompatible:
		return "incompatible"
	case StateBroken:
		return "broken"
	default:
		return "unknown"
	}
}

// Sentinel errors.
var (
	ErrSidecarMalformed = errors.New("sidecar malformed")
	ErrDimsMismatch     = errors.New("sidecar dims mismatch len(vector)")
)
