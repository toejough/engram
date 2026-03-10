package signal

import (
	"context"
	"time"

	"engram/internal/effectiveness"
	"engram/internal/promote"
	"engram/internal/review"
)

// Classifier classifies memories into effectiveness quadrants.
type Classifier interface {
	Classify(
		stats map[string]effectiveness.Stat,
		tracking map[string]review.TrackingData,
	) []review.ClassifiedMemory
}

// ClaudeMDScanner finds skill→CLAUDE.md promotion and demotion candidates.
type ClaudeMDScanner interface {
	PromotionCandidates(threshold int) ([]promote.Candidate, error)
	DemotionCandidates() ([]promote.Candidate, error)
}

// Detector aggregates maintenance and promotion signals.
type Detector struct {
	classifier Classifier
	promoter   PromotionScanner
	claudeMD   ClaudeMDScanner
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

	if d.promoter != nil {
		promoSignals, err := d.promotionSignals()
		if err != nil {
			return nil, err
		}

		signals = append(signals, promoSignals...)
	}

	if d.claudeMD != nil {
		cmdSignals, err := d.claudeMDSignals()
		if err != nil {
			return nil, err
		}

		signals = append(signals, cmdSignals...)
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

func (d *Detector) claudeMDSignals() ([]Signal, error) {
	now := d.now()

	var signals []Signal

	promoCandidates, err := d.claudeMD.PromotionCandidates(defaultClaudeMDThreshold)
	if err != nil {
		return nil, err
	}

	for _, c := range promoCandidates {
		signals = append(signals, Signal{
			Type:       TypePromote,
			SourceID:   c.Entry.SourcePath,
			SignalKind: KindSkillToClaudeMD,
			Summary:    "Skill eligible for CLAUDE.md promotion",
			DetectedAt: now,
		})
	}

	demoCandidates, err := d.claudeMD.DemotionCandidates()
	if err != nil {
		return nil, err
	}

	for _, c := range demoCandidates {
		signals = append(signals, Signal{
			Type:       TypePromote,
			SourceID:   c.Entry.SourcePath,
			SignalKind: KindClaudeMDDemotion,
			Summary:    "CLAUDE.md entry eligible for demotion to skill",
			DetectedAt: now,
		})
	}

	return signals, nil
}

func (d *Detector) promotionSignals() ([]Signal, error) {
	candidates, err := d.promoter.Candidates(defaultPromotionThreshold)
	if err != nil {
		return nil, err
	}

	now := d.now()
	signals := make([]Signal, 0, len(candidates))

	for _, c := range candidates {
		signals = append(signals, Signal{
			Type:       TypePromote,
			SourceID:   c.Entry.SourcePath,
			SignalKind: KindMemoryToSkill,
			Summary:    "Memory eligible for skill promotion",
			DetectedAt: now,
		})
	}

	return signals, nil
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector)

// PromotionScanner finds memory→skill promotion candidates.
type PromotionScanner interface {
	Candidates(threshold int) ([]promote.Candidate, error)
}

// WithClassifier sets the classifier for the Detector.
func WithClassifier(c Classifier) DetectorOption {
	return func(d *Detector) {
		d.classifier = c
	}
}

// WithClaudeMDScanner sets the CLAUDE.md scanner for the Detector.
func WithClaudeMDScanner(s ClaudeMDScanner) DetectorOption {
	return func(d *Detector) {
		d.claudeMD = s
	}
}

// WithDetectorNow sets the time source for the Detector.
func WithDetectorNow(fn func() time.Time) DetectorOption {
	return func(d *Detector) {
		d.now = fn
	}
}

// WithPromoter sets the promotion scanner for the Detector.
func WithPromoter(p PromotionScanner) DetectorOption {
	return func(d *Detector) {
		d.promoter = p
	}
}

// unexported constants.
const (
	defaultClaudeMDThreshold  = 10
	defaultPromotionThreshold = 5
)
