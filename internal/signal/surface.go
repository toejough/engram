package signal

import (
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/memory"
)

// EnrichedSignal combines a signal with its memory content for surfacing.
//
//nolint:tagliatelle // DES-44 specifies snake_case JSON field names.
type EnrichedSignal struct {
	Type       string `json:"type"`
	SourceID   string `json:"source_id"`
	SignalKind string `json:"signal"`
	Quadrant   string `json:"quadrant,omitempty"`
	Summary    string `json:"summary"`
	Title      string `json:"title,omitempty"`
	Principle  string `json:"principle,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
}

// MemoryLoader loads a memory TOML file by path.
type MemoryLoader interface {
	Load(path string) (*memory.Stored, error)
}

// Surfacer loads and enriches signals for model-facing output.
type Surfacer struct {
	loader MemoryLoader
}

// NewSurfacer creates a Surfacer with the given options.
func NewSurfacer(opts ...SurfacerOption) *Surfacer {
	s := &Surfacer{}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Surface enriches signals with memory content and formats output.
func (s *Surfacer) Surface(signals []Signal) ([]EnrichedSignal, error) {
	enriched := make([]EnrichedSignal, 0, len(signals))

	for _, sig := range signals {
		e := EnrichedSignal{
			Type:       sig.Type,
			SourceID:   sig.SourceID,
			SignalKind: sig.SignalKind,
			Quadrant:   sig.Quadrant,
			Summary:    sig.Summary,
		}

		if s.loader != nil && sig.SourceID != "" {
			stored, err := s.loader.Load(sig.SourceID)
			if err == nil && stored != nil {
				e.Title = stored.Title
				e.Principle = stored.Principle
				e.Keywords = strings.Join(stored.Keywords, ", ")
			}
		}

		enriched = append(enriched, e)
	}

	return enriched, nil
}

// SurfacerOption configures a Surfacer.
type SurfacerOption func(*Surfacer)

// FormatContext formats enriched signals as model-facing context.
func FormatContext(enriched []EnrichedSignal) (string, error) {
	if len(enriched) == 0 {
		return "", nil
	}

	type contextOutput struct {
		Signals      []EnrichedSignal `json:"signals"`
		Instructions string           `json:"instructions"`
	}

	output := contextOutput{
		Signals:      enriched,
		Instructions: formatInstructions(enriched),
	}

	data, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshaling signal context: %w", err)
	}

	return string(data), nil
}

// WithLoader sets the memory loader for the Surfacer.
func WithLoader(l MemoryLoader) SurfacerOption {
	return func(s *Surfacer) {
		s.loader = l
	}
}

func formatInstructions(enriched []EnrichedSignal) string {
	var sb strings.Builder

	sb.WriteString(
		"Present these maintenance signals to the user for their decision. " +
			"Do not act on them without user approval:\n",
	)

	for _, e := range enriched {
		switch e.SignalKind {
		case KindNoiseRemoval:
			fmt.Fprintf(
				&sb,
				"- %s: Recommend removal (rarely surfaced, low effectiveness)\n",
				e.SourceID,
			)
		case KindLeechRewrite:
			fmt.Fprintf(
				&sb,
				"- %s: Recommend rewrite (frequently surfaced, rarely followed)\n",
				e.SourceID,
			)
		case KindHiddenGemBroaden:
			fmt.Fprintf(
				&sb,
				"- %s: Recommend broadening keywords (high effectiveness, rarely surfaced)\n",
				e.SourceID,
			)
		case KindMemoryToSkill:
			fmt.Fprintf(&sb, "- %s: Eligible for promotion to skill\n", e.SourceID)
		default:
			fmt.Fprintf(&sb, "- %s: %s\n", e.SourceID, e.Summary)
		}
	}

	return sb.String()
}
