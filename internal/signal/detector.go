package signal

import (
	"context"
	"time"

	"engram/internal/effectiveness"
	"engram/internal/review"
)

// Classifier classifies memories into effectiveness quadrants.
type Classifier interface {
	Classify(
		stats map[string]effectiveness.Stat,
		tracking map[string]review.TrackingData,
	) []review.ClassifiedMemory
}

// Detector aggregates maintenance signals.
type Detector struct {
	classifier Classifier
	now        func() time.Time
}

// NewDetector creates a Detector with the given options.
func NewDetector(opts ...DetectorOption) *Detector {
	d := &Detector{
		now: time.Now,
	}
	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Detect runs all signal detection and returns discovered signals.
func (d *Detector) Detect(
	_ context.Context,
	stats map[string]effectiveness.Stat,
	tracking map[string]review.TrackingData,
) ([]Signal, error) {
	var signals []Signal

	if d.classifier != nil {
		classified := d.classifier.Classify(stats, tracking)
		signals = append(signals, d.classifiedToSignals(classified)...)
	}

	return signals, nil
}

func (d *Detector) classifiedToSignals(classified []review.ClassifiedMemory) []Signal {
	now := d.now()
	signals := make([]Signal, 0, len(classified))

	for _, mem := range classified {
		var kind, summary string

		switch mem.Quadrant {
		case review.Noise:
			kind = KindNoiseRemoval
			summary = "Rarely surfaced and ineffective — candidate for removal"
		case review.Leech:
			kind = KindLeechRewrite
			summary = "Frequently surfaced but rarely followed — needs rewrite"
		case review.HiddenGem:
			kind = KindHiddenGemBroaden
			summary = "High follow-through but rarely surfaced — broaden keywords"
		case review.Working, review.InsufficientData:
			continue
		}

		signals = append(signals, Signal{
			Type:       TypeMaintain,
			SourceID:   mem.Name,
			SignalKind: kind,
			Quadrant:   string(mem.Quadrant),
			Summary:    summary,
			DetectedAt: now,
		})
	}

	return signals
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector)

// WithClassifier sets the classifier for the Detector.
func WithClassifier(c Classifier) DetectorOption {
	return func(d *Detector) {
		d.classifier = c
	}
}

// WithDetectorNow sets the time source for the Detector.
func WithDetectorNow(fn func() time.Time) DetectorOption {
	return func(d *Detector) {
		d.now = fn
	}
}
