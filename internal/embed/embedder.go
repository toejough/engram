package embed

import (
	"context"
	"errors"
	"fmt"
)

// Exported constants.
const (
	SidecarSchemaVersion = 1
)

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

// Exported variables.
var (
	ErrDimsMismatch     = errors.New("sidecar dims mismatch len(vector)")
	ErrSchemaVersion    = errors.New("sidecar schema version unsupported")
	ErrSidecarMalformed = errors.New("sidecar malformed")
)

// Embedder produces fixed-dimension dense vectors from text. Implementations
// are expected to be safe for concurrent use unless documented otherwise.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	ModelID() string
	Dims() int
}

// Sidecar is the on-disk shape of a per-note .vec.json file. Field order
// here is the JSON key order. Snake-case keys match the spike spec's
// sidecar contract verbatim and are part of the on-disk file format:
// MiniLM-L6-v2@384 is the shipped bundled model, and the 2026-05-24 query
// spike froze the snake_case sidecar keys as a file format. Each note
// carries two vectors — one for its situation: frontmatter field and
// one for its body — so retrieval can match by max(situation, body).
//
//nolint:tagliatelle // sidecar JSON keys are spec contract
type Sidecar struct {
	SchemaVersion    int       `json:"schema_version"`
	EmbeddingModelID string    `json:"embedding_model_id"`
	Dims             int       `json:"dims"`
	SituationVector  []float32 `json:"situation_vector"`
	BodyVector       []float32 `json:"body_vector"`
	ContentHash      string    `json:"content_hash"`
	// LastUsed is the date (YYYY-MM-DD) this note last surfaced as a useful
	// (above-cutoff) recall hit. Additive metadata: omitempty, EXCLUDED from
	// ContentHash (hash.go hashes situation+body of the raw note, not this), and
	// it does NOT bump SidecarSchemaVersion — old sidecars decode LastUsed=""
	// ("never used"). Never feed LastUsed into any hash: bumping it must not
	// mark a note stale.
	LastUsed string `json:"last_used,omitempty"` //nolint:tagliatelle // sidecar JSON keys are spec contract
}

// BuildSidecar embeds a note's situation and body and returns a fully
// stamped dual-vector sidecar. When the note has no situation field, the
// body text stands in for the situation embedding so every note still
// carries a meaningful situation vector. Either embed failure is returned
// to the caller, which applies its own warn-or-fail policy.
func BuildSidecar(ctx context.Context, embedder Embedder, raw []byte) (Sidecar, error) {
	situationInput := SituationText(raw)
	if len(situationInput) == 0 {
		situationInput = BodyText(raw)
	}

	situationVector, err := embedder.Embed(ctx, string(situationInput))
	if err != nil {
		return Sidecar{}, fmt.Errorf("embed: situation vector: %w", err)
	}

	bodyVector, err := embedder.Embed(ctx, string(BodyText(raw)))
	if err != nil {
		return Sidecar{}, fmt.Errorf("embed: body vector: %w", err)
	}

	return Sidecar{
		SchemaVersion:    SidecarSchemaVersion,
		EmbeddingModelID: embedder.ModelID(),
		Dims:             embedder.Dims(),
		SituationVector:  situationVector,
		BodyVector:       bodyVector,
		ContentHash:      ContentHash(raw),
	}, nil
}
