package tfidf_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/tfidf"
)

func TestClusterConfidence_EmptyStrings_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{"", ""})

	g.Expect(score).To(Equal(0.0))
}

func TestClusterConfidence_EmptyTexts_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{})

	g.Expect(score).To(Equal(0.0))
}

func TestClusterConfidence_PartialOverlap_BetweenZeroAndOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{
		"targ check full lint build",
		"targ check full errors validation",
	})

	g.Expect(score).To(BeNumerically(">", 0.0))
	g.Expect(score).To(BeNumerically("<", 1.0))
}

func TestClusterConfidence_SingleText_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{"only one text"})

	g.Expect(score).To(Equal(0.0))
}

func TestClusterConfidence_ThreeTexts_AveragesPairwise(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{
		"alpha beta gamma",
		"alpha beta delta",
		"epsilon zeta eta",
	})

	// Two pairs share terms, one pair is disjoint. Average should be between 0 and 1.
	g.Expect(score).To(BeNumerically(">", 0.0))
	g.Expect(score).To(BeNumerically("<", 1.0))
}

// T-367: TF-IDF cosine similarity returns 1.0 for identical keyword lists.
func TestT367_IdenticalTexts_ScoreOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{
		"always use targ check-full for builds",
		"always use targ check-full for builds",
	})

	g.Expect(score).To(BeNumerically("~", 1.0, 0.001))
}

// T-368: TF-IDF cosine similarity returns 0.0 for completely disjoint texts.
func TestT368_DisjointTexts_ScoreZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scorer := tfidf.NewScorer()
	score := scorer.ClusterConfidence([]string{
		"alpha beta gamma delta",
		"epsilon zeta eta theta",
	})

	g.Expect(score).To(BeNumerically("~", 0.0, 0.001))
}
