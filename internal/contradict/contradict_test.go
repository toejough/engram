// Package contradict_test tests contradiction detection (UC-P1-1).
package contradict_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/contradict"
	"engram/internal/memory"
)

// T-P1-5: high BM25 similarity without verb heuristic → borderline, LLM called.
func TestDetector_BM25BorderlineSentToLLM(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Near-identical content but no opposing verbs — high BM25, no heuristic.
	a := newMem("a", "Store credentials in environment variables for security")
	b := newMem("b", "Store credentials in environment variables carefully")

	clf := &mockClassifier{result: true}
	d := contradict.NewDetector(contradict.WithClassifier(clf))
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clf.calls).To(BeNumerically(">=", 1))
	g.Expect(pairs).To(HaveLen(1))
}

// T-P1-7: classifier error → pair not included.
func TestDetector_ClassifierErrorNonContradiction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := newMem("a", "Store credentials in environment variables for security")
	b := newMem("b", "Store credentials in environment variables carefully")

	clf := &mockClassifier{result: false, err: errors.New("LLM unavailable")}
	d := contradict.NewDetector(contradict.WithClassifier(clf))
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(BeEmpty())
}

// T-P1-3: heuristic fires on always/never pair.
func TestDetector_HeuristicAlwaysNever(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := newMem("a", "Always add t.Parallel() to tests")
	b := newMem("b", "Never use t.Parallel() in benchmark tests")

	d := contradict.NewDetector()
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(HaveLen(1))
}

// T-P1-2: heuristic fires on use/avoid pair.
func TestDetector_HeuristicUseAvoid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := newMem("a", "Always use targ for builds")
	b := newMem("b", "Avoid using targ directly")

	d := contradict.NewDetector()
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(HaveLen(1))

	if len(pairs) == 0 {
		return
	}

	g.Expect(pairs[0].A).To(Equal(a))
	g.Expect(pairs[0].B).To(Equal(b))
}

// T-P1-8: high-confidence pair (heuristic fires) skips LLM.
func TestDetector_HighConfidenceSkipsLLM(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Opposing verbs AND high textual overlap.
	a := newMem("a", "Always use targ to run go tests for the build system")
	b := newMem("b", "Avoid using targ to run go tests for the build system")

	clf := &mockClassifier{result: true}
	d := contradict.NewDetector(contradict.WithClassifier(clf))
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(HaveLen(1))
	g.Expect(clf.calls).To(Equal(0)) // LLM not called (heuristic fired)
}

// T-P1-6: LLM budget enforced — max 3 calls for many borderline pairs.
func TestDetector_LLMBudgetEnforced(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build 4 memories → 6 pairs all similar enough to be borderline.
	base := "Store credentials in environment variables"
	memories := []*memory.Stored{
		newMem("a", base+" for security"),
		newMem("b", base+" carefully for config"),
		newMem("c", base+" using dotenv patterns"),
		newMem("d", base+" with vault integration"),
	}

	clf := &mockClassifier{result: true}
	d := contradict.NewDetector(
		contradict.WithClassifier(clf),
		contradict.WithMaxLLMCalls(3),
	)
	_, err := d.Check(context.Background(), memories)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clf.calls).To(BeNumerically("<=", 3))
}

// T-P1-12: no classifier set → LLM phase skipped, borderline not returned.
func TestDetector_NoClassifierBorderlineSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := newMem("a", "Store credentials in environment variables for security")
	b := newMem("b", "Store credentials in environment variables carefully")

	d := contradict.NewDetector() // no classifier
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Borderline pair not confirmed without LLM → not returned.
	g.Expect(pairs).To(BeEmpty())
}

// T-P1-4: heuristic does not fire on unrelated memories.
func TestDetector_NoContradictionUnrelated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := newMem("a", "Commit messages use conventional commits")
	b := newMem("b", "Add t.Parallel() to every test function")

	d := contradict.NewDetector()
	pairs, err := d.Check(context.Background(), []*memory.Stored{a, b})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(BeEmpty())
}

// T-P1-1: single memory → no pairs.
func TestDetector_SingleMemory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	d := contradict.NewDetector()
	pairs, err := d.Check(context.Background(), []*memory.Stored{newMem("a", "Always use targ")})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pairs).To(BeEmpty())
}

// mockClassifier counts calls and returns a fixed result.
type mockClassifier struct {
	result bool
	err    error
	calls  int
}

func (m *mockClassifier) Classify(_ context.Context, _, _ *memory.Stored) (bool, error) {
	m.calls++
	return m.result, m.err
}

func newMem(title, principle string) *memory.Stored {
	return &memory.Stored{
		Title:     title,
		Principle: principle,
		UpdatedAt: time.Now(),
		FilePath:  title + ".toml",
	}
}
