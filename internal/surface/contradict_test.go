// Contradiction suppression tests for surface package (T-P1-10, T-P1-11).
package surface_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/contradict"
	"engram/internal/memory"
	"engram/internal/signal"
	"engram/internal/surface"
)

// T-P1-10: runSessionStart suppresses lower-ranked memory and emits signal.
func TestSurface_ContradictionSuppressesLowerRanked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := contradictMem("use-targ", "Always use targ for builds")
	memB := contradictMem("avoid-targ", "Avoid using targ directly")

	retriever := &fakeRetriever{memories: []*memory.Stored{memA, memB}}
	emitter := &fakeSignalEmitter{}
	detector := &fakeContradictionDetector{
		pairs: []contradict.Pair{{A: memA, B: memB, Confidence: 0.9}},
	}

	s := surface.New(retriever,
		surface.WithContradictionDetector(detector),
		surface.WithSignalEmitter(emitter),
	)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := buf.String()
	// A is kept, B is suppressed.
	g.Expect(out).To(ContainSubstring("use-targ"))
	g.Expect(out).NotTo(ContainSubstring("avoid-targ"))

	// Signal emitted for suppressed memory.
	g.Expect(emitter.signals).To(HaveLen(1))
	g.Expect(emitter.signals[0].SignalKind).To(Equal(signal.KindContradiction))
	g.Expect(emitter.signals[0].SourceID).To(Equal(memB.FilePath))
}

// T-P1-11: no detector set → no panic, normal output.
func TestSurface_NoDetectorNoPanic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mem1 := contradictMem("use-targ", "Always use targ for builds")
	retriever := &fakeRetriever{memories: []*memory.Stored{mem1}}

	s := surface.New(retriever) // no detector

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("use-targ"))
}

// fakeContradictionDetector returns a fixed set of pairs.
type fakeContradictionDetector struct {
	pairs []contradict.Pair
	err   error
}

func (f *fakeContradictionDetector) Check(_ context.Context, _ []*memory.Stored) ([]contradict.Pair, error) {
	return f.pairs, f.err
}

// fakeSignalEmitter records emitted signals.
type fakeSignalEmitter struct {
	signals []signal.Signal
}

func (f *fakeSignalEmitter) Emit(s signal.Signal) error {
	f.signals = append(f.signals, s)
	return nil
}

// contradictMem creates a test memory for contradiction tests.
func contradictMem(title, principle string) *memory.Stored {
	return &memory.Stored{
		Title:     title,
		Principle: principle,
		FilePath:  title + ".toml",
		UpdatedAt: time.Now(),
	}
}
