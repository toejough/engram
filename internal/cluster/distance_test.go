package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cluster"
)

func TestCosineDistance_IdenticalVectorsIsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	a := []float32{1, 2, 3, 4}
	b := []float32{1, 2, 3, 4}

	g.Expect(cluster.CosineDistance(a, b)).To(BeNumerically("~", 0, 1e-6))
}

func TestCosineDistance_OppositeVectorsIsTwo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	a := []float32{1, 0}
	b := []float32{-1, 0}

	g.Expect(cluster.CosineDistance(a, b)).To(BeNumerically("~", 2, 1e-6))
}

func TestCosineDistance_OrthogonalVectorsIsOne(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	a := []float32{1, 0}
	b := []float32{0, 1}

	g.Expect(cluster.CosineDistance(a, b)).To(BeNumerically("~", 1, 1e-6))
}

func TestCosineDistance_ZeroVectorReturnsOne(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	a := []float32{0, 0, 0, 0}
	b := []float32{1, 2, 3, 4}

	// Zero vector has no direction; treat as max distance.
	g.Expect(cluster.CosineDistance(a, b)).To(BeNumerically("~", 1, 1e-6))
}
