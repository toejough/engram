package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cluster"
)

func TestAutoK_AbsorbsSingletons(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// 6 vectors: 5 cluster nicely, one is an outlier. With k=2 we expect
	// the outlier might form its own singleton cluster; AutoK must absorb
	// it.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.1, 0, 0},
		{0.9, 0.05, 0, 0},
		{0.97, 0.03, 0, 0},
		{0.98, 0.02, 0, 0},
		{-1, 0, 0, 0},
	}

	result, err := cluster.AutoK(vectors, 2, 2, 0.0, 7)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Every cluster must have size >= 2.
	counts := make(map[int]int)
	for _, c := range result.Assignments {
		counts[c]++
	}

	for clusterID, count := range counts {
		g.Expect(count).To(BeNumerically(">=", 2),
			"cluster %d has size %d (must be >= 2 after singleton absorption)", clusterID, count)
	}
}

func TestAutoK_KRangeBounded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Only 3 vectors; max k = 3 but we ask for up to 7. Must cap at 3.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}

	result, err := cluster.AutoK(vectors, 2, 7, 0.0, 42)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.K).To(BeNumerically("<=", 3))
}

func TestAutoK_LowSilhouetteReturnsZeroClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Identical vectors → silhouette is exactly 0 for every k (all distances
	// are zero, no separable structure). AutoK with threshold 0.10 must
	// return K: 0.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
	}

	result, err := cluster.AutoK(vectors, 2, 7, 0.10, 42)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.K).To(BeZero())
	g.Expect(result.Assignments).To(BeNil())
}

func TestAutoK_PicksThreeForKnownThreeClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Three well-separated groups in 4D.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.05, 0, 0},
		{0.9, 0.02, 0, 0},
		{0.97, 0.03, 0, 0},
		{0, 1, 0, 0},
		{0.05, 0.95, 0, 0},
		{0.02, 0.9, 0, 0},
		{0.03, 0.97, 0, 0},
		{0, 0, 1, 0},
		{0, 0.05, 0.95, 0},
		{0, 0.02, 0.9, 0},
		{0, 0.03, 0.97, 0},
	}

	result, err := cluster.AutoK(vectors, 2, 7, 0.10, 42)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.K).To(Equal(3))
	g.Expect(result.Silhouette).To(BeNumerically(">", 0.5))
}
