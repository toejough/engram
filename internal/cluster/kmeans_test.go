package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cluster"
)

func TestKMeans_DeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.1, 0, 0},
		{0.9, 0.05, 0, 0},
		{0, 1, 0, 0},
		{0.1, 0.95, 0, 0},
		{0.05, 0.9, 0, 0},
	}

	const seed = uint64(42)

	first, err1 := cluster.KMeans(vectors, 2, seed)
	g.Expect(err1).NotTo(HaveOccurred())

	if err1 != nil {
		return
	}

	second, err2 := cluster.KMeans(vectors, 2, seed)
	g.Expect(err2).NotTo(HaveOccurred())

	if err2 != nil {
		return
	}

	g.Expect(second.Assignments).To(Equal(first.Assignments))
	g.Expect(second.Centroids).To(Equal(first.Centroids))
}

func TestKMeans_EmptyInputReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cluster.KMeans(nil, 2, 1)
	g.Expect(err).To(HaveOccurred())
}

func TestKMeans_KGreaterThanNReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectors := [][]float32{{1, 0}, {0, 1}}

	_, err := cluster.KMeans(vectors, 5, 1)
	g.Expect(err).To(HaveOccurred())
}

func TestKMeans_KOneReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectors := [][]float32{{1, 0}, {0, 1}}

	_, err := cluster.KMeans(vectors, 1, 1)
	g.Expect(err).To(HaveOccurred())
}

func TestKMeans_TwoWellSeparatedClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Two groups of vectors: one clustered around (1,0,0,0), one around (0,1,0,0).
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.1, 0, 0},
		{0.9, 0.05, 0, 0},
		{0, 1, 0, 0},
		{0.1, 0.95, 0, 0},
		{0.05, 0.9, 0, 0},
	}

	result, err := cluster.KMeans(vectors, 2, 1)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Assignments).To(HaveLen(len(vectors)))
	g.Expect(result.Centroids).To(HaveLen(2))

	// First 3 share a cluster; last 3 share a cluster.
	g.Expect(result.Assignments[0]).To(Equal(result.Assignments[1]))
	g.Expect(result.Assignments[0]).To(Equal(result.Assignments[2]))
	g.Expect(result.Assignments[3]).To(Equal(result.Assignments[4]))
	g.Expect(result.Assignments[3]).To(Equal(result.Assignments[5]))
	g.Expect(result.Assignments[0]).NotTo(Equal(result.Assignments[3]))
}
