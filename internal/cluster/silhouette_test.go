package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cluster"
)

func TestPointSilhouette_OutOfRangeOwnClusterReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectors := [][]float32{{1, 0}, {0, 1}}
	members := [][]int{{0}, {1}}

	g.Expect(cluster.PointSilhouette(vectors[0], vectors, members, -1, 0)).To(BeZero())
	g.Expect(cluster.PointSilhouette(vectors[0], vectors, members, 5, 0)).To(BeZero())
}

func TestPointSilhouette_SingletonOwnClusterReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectors := [][]float32{{1, 0}, {0, 1}}
	members := [][]int{{0}, {1}}

	g.Expect(cluster.PointSilhouette(vectors[0], vectors, members, 0, 0)).To(BeZero())
}

func TestPointSilhouette_TwoMemberCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Two clusters of 2 each; well-separated.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.99, 0.01, 0, 0},
		{0, 1, 0, 0},
		{0, 0.99, 0.01, 0},
	}
	members := [][]int{{0, 1}, {2, 3}}

	score := cluster.PointSilhouette(vectors[0], vectors, members, 0, 0)
	g.Expect(score).To(BeNumerically(">", 0.5))
}

func TestPointSilhouette_ZeroDenominator(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// All identical vectors → both intra and inter distances are 0.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
		{1, 0, 0, 0},
	}
	members := [][]int{{0, 1}, {2, 3}}

	score := cluster.PointSilhouette(vectors[0], vectors, members, 0, 0)
	g.Expect(score).To(BeZero())
}

func TestSilhouette_RandomAssignmentIsLow(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Vectors form a real two-cluster structure but assignments split clusters across labels.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.1, 0, 0},
		{0.9, 0.05, 0, 0},
		{0, 1, 0, 0},
		{0.1, 0.95, 0, 0},
		{0.05, 0.9, 0, 0},
	}
	assignments := []int{0, 1, 0, 1, 0, 1}

	score := cluster.Silhouette(vectors, assignments, 2)
	g.Expect(score).To(BeNumerically("<", 0.1))
}

func TestSilhouette_SinglePointPerClusterIsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// k clusters of one point each → silhouette undefined per-point; the
	// helper convention returns 0 for unrepresentative inputs.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
	}
	assignments := []int{0, 1}

	score := cluster.Silhouette(vectors, assignments, 2)
	g.Expect(score).To(BeNumerically("~", 0, 1e-6))
}

func TestSilhouette_WellSeparatedClustersIsHigh(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Two well-separated groups. Silhouette should be close to 1.
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0.95, 0.1, 0, 0},
		{0.9, 0.05, 0, 0},
		{0, 1, 0, 0},
		{0.1, 0.95, 0, 0},
		{0.05, 0.9, 0, 0},
	}
	assignments := []int{0, 0, 0, 1, 1, 1}

	score := cluster.Silhouette(vectors, assignments, 2)
	g.Expect(score).To(BeNumerically(">", 0.5))
}
