package embed_test

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/embed"
)

const cosineFloatTolerance = 1e-6

func TestCosine_Identical(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	v := []float32{0.1, 0.2, 0.3, 0.4}
	g.Expect(float64(embed.Cosine(v, v))).To(BeNumerically("~", 1.0, cosineFloatTolerance))
}

func TestCosine_Orthogonal(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	a := []float32{1, 0}
	b := []float32{0, 1}
	g.Expect(float64(embed.Cosine(a, b))).To(BeNumerically("~", 0.0, cosineFloatTolerance))
}

func TestCosine_ZeroVector(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	g.Expect(embed.Cosine(a, b)).To(Equal(float32(0)))
}

func TestCosine_MismatchedLengths(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(embed.Cosine([]float32{1}, []float32{1, 2})).To(Equal(float32(0)))
}

// TestCosine_SelfSimilarityProperty asserts cosine(v,v) == 1 for any
// non-zero vector.
func TestCosine_SelfSimilarityProperty(t *testing.T) {
	t.Parallel()

	const propertyTolerance = 1e-4

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 16).Draw(t, "n")
		v := rapid.SliceOfN(rapid.Float32Range(-10, 10), n, n).Draw(t, "v")

		var sumSq float64
		for _, x := range v {
			sumSq += float64(x) * float64(x)
		}

		if sumSq < cosineFloatTolerance {
			t.Skip("zero vector handled by separate test")
		}

		got := float64(embed.Cosine(v, v))
		if math.Abs(got-1.0) > propertyTolerance {
			t.Fatalf("cosine(v,v) = %v, want ~1.0", got)
		}
	})
}
