package signal_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/effectiveness"
	"engram/internal/promote"
	"engram/internal/registry"
	"engram/internal/review"
	"engram/internal/signal"
)

func TestDetector_ClaudeMDSignals(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	scanner := &stubClaudeMDScanner{
		promoCandidates: []promote.Candidate{
			{Entry: registry.InstructionEntry{SourcePath: "skills/promo.md"}},
		},
		demoCandidates: []promote.Candidate{
			{Entry: registry.InstructionEntry{SourcePath: "claudemd/demote.md"}},
		},
	}

	detector := signal.NewDetector(
		signal.WithClaudeMDScanner(scanner),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(2))

	if len(signals) < 2 {
		return
	}

	g.Expect(signals[0].SignalKind).To(gomega.Equal(signal.KindGraduation))
	g.Expect(signals[0].Summary).To(gomega.ContainSubstring("promoting this skill to CLAUDE.md"))
	g.Expect(signals[1].SignalKind).To(gomega.Equal(signal.KindGraduation))
	g.Expect(signals[1].Summary).To(gomega.ContainSubstring("demoting this CLAUDE.md entry"))
}

func TestDetector_HiddenGemSignal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "gem.toml", Quadrant: review.HiddenGem},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))

	if len(signals) == 0 {
		return
	}

	g.Expect(signals[0].SignalKind).To(gomega.Equal(signal.KindHiddenGemBroaden))
}

func TestDetector_InsufficientDataSkipped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "new.toml", Quadrant: review.InsufficientData},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.BeEmpty())
}

func TestDetector_LeechSignal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "leech.toml", Quadrant: review.Leech},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))

	if len(signals) == 0 {
		return
	}

	g.Expect(signals[0].SignalKind).To(gomega.Equal(signal.KindLeechRewrite))
}

func TestDetector_MultipleQuadrants(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "noise.toml", Quadrant: review.Noise},
			{Name: "leech.toml", Quadrant: review.Leech},
			{Name: "gem.toml", Quadrant: review.HiddenGem},
			{Name: "working.toml", Quadrant: review.Working},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(3))
}

func TestDetector_NoiseSignal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "stale-memory.toml", Quadrant: review.Noise},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))

	if len(signals) == 0 {
		return
	}

	g.Expect(signals[0].SignalKind).To(gomega.Equal(signal.KindNoiseRemoval))
	g.Expect(signals[0].Type).To(gomega.Equal(signal.TypeMaintain))
	g.Expect(signals[0].SourceID).To(gomega.Equal("stale-memory.toml"))
	g.Expect(signals[0].Quadrant).To(gomega.Equal(string(review.Noise)))
}

func TestDetector_PromotionSignals(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	promoter := &stubPromotionScanner{
		candidates: []promote.Candidate{
			{Entry: registry.InstructionEntry{SourcePath: "memories/good.toml"}},
		},
	}

	detector := signal.NewDetector(
		signal.WithPromoter(promoter),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))

	if len(signals) == 0 {
		return
	}

	g.Expect(signals[0].SignalKind).To(gomega.Equal(signal.KindMemoryToSkill))
	g.Expect(signals[0].Type).To(gomega.Equal(signal.TypePromote))
}

func TestDetector_WorkingSkipped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classifier := &stubClassifier{
		result: []review.ClassifiedMemory{
			{Name: "working.toml", Quadrant: review.Working},
		},
	}

	detector := signal.NewDetector(
		signal.WithClassifier(classifier),
		signal.WithDetectorNow(func() time.Time { return fixedTime }),
	)

	signals, err := detector.Detect(context.Background(), nil, nil)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.BeEmpty())
}

// unexported variables.
var (
	fixedTime = time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
)

type stubClassifier struct {
	result []review.ClassifiedMemory
}

func (s *stubClassifier) Classify(
	_ map[string]effectiveness.Stat,
	_ map[string]review.TrackingData,
) []review.ClassifiedMemory {
	return s.result
}

type stubClaudeMDScanner struct {
	promoCandidates []promote.Candidate
	promoErr        error
	demoCandidates  []promote.Candidate
	demoErr         error
}

func (s *stubClaudeMDScanner) DemotionCandidates() ([]promote.Candidate, error) {
	return s.demoCandidates, s.demoErr
}

func (s *stubClaudeMDScanner) PromotionCandidates(_ int) ([]promote.Candidate, error) {
	return s.promoCandidates, s.promoErr
}

type stubPromotionScanner struct {
	candidates []promote.Candidate
	err        error
}

func (s *stubPromotionScanner) Candidates(_ int) ([]promote.Candidate, error) {
	return s.candidates, s.err
}
