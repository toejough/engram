// Package signal detects, queues, surfaces, and applies maintenance signals (UC-28).
package signal

import "time"

// Exported constants.
const (
	KindContradiction    = "contradiction"
	KindEscalation       = "escalation"
	KindGraduation       = "graduation"
	KindHiddenGemBroaden = "hidden_gem_broadening"
	KindLeechRewrite     = "leech_rewrite"
	KindMemoryToSkill    = "memory_to_skill"
	KindNoiseRemoval     = "noise_removal"
	KindStalenessReview  = "staleness_review"
	TypeMaintain         = "maintain"
	TypePromote          = "promote"
)

// Signal represents a detected maintenance or promotion signal.
//
//nolint:tagliatelle // DES-43 specifies snake_case JSON field names.
type Signal struct {
	Type       string    `json:"type"`
	SourceID   string    `json:"source_id"`
	SignalKind string    `json:"signal"`
	Quadrant   string    `json:"quadrant,omitempty"`
	Summary    string    `json:"summary"`
	DetectedAt time.Time `json:"detected_at"`
}
