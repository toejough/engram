package embed_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestHugotEmbedder_Close_ErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	emb := embed.NewHugotEmbedderWithPipelineForTest("fake@4", 4,
		func(string) ([][]float32, error) { return [][]float32{{0}}, nil },
		func() error { return errors.New("destroy failed") },
	)
	g.Expect(emb.Close()).To(MatchError(ContainSubstring("destroy failed")))
}

func TestHugotEmbedder_Embed_EmptyResultReportsErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	emb := embed.NewHugotEmbedderWithPipelineForTest("fake@4", 4,
		func(string) ([][]float32, error) { return [][]float32{}, nil },
		func() error { return nil },
	)

	_, err := emb.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(embed.ErrHugotEmbedEmpty))
}

func TestHugotEmbedder_Embed_HappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	closeCalls := 0
	emb := embed.NewHugotEmbedderWithPipelineForTest("fake@4", 4,
		func(string) ([][]float32, error) {
			return [][]float32{{0.1, 0.2, 0.3, 0.4}}, nil
		},
		func() error { closeCalls++; return nil },
	)

	vec, err := emb.Embed(context.Background(), "hello")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(Equal([]float32{0.1, 0.2, 0.3, 0.4}))
	g.Expect(emb.Dims()).To(Equal(4))
	g.Expect(emb.ModelID()).To(Equal("fake@4"))
	g.Expect(emb.Close()).NotTo(HaveOccurred())
	g.Expect(closeCalls).To(Equal(1))

	// Second Close is a no-op (already cleared).
	g.Expect(emb.Close()).NotTo(HaveOccurred())
	g.Expect(closeCalls).To(Equal(1))
}

func TestHugotEmbedder_Embed_RunErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	emb := embed.NewHugotEmbedderWithPipelineForTest("fake@4", 4,
		func(string) ([][]float32, error) { return nil, errors.New("backend down") },
		func() error { return nil },
	)

	_, err := emb.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("backend down")))
}

// TestLazyEmbedder_DimsAndModelIDBeforeInit asserts the pre-init shortcut
// paths that return BundledModelID / Dims without unpacking the model.
func TestLazyEmbedder_DimsAndModelIDBeforeInit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lazy := embed.NewLazyEmbedder()
	// ModelID before init returns the bundled constant.
	g.Expect(lazy.ModelID()).To(Equal(embed.BundledModelID))
}

// TestLazyEmbedder_InitFailurePathsAllUnreachable drives every init-
// failure branch via an injected factory: Embed wraps the error, Dims
// returns 0, ModelID returns the bundled constant fallback.
func TestLazyEmbedder_InitFailurePaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bootErr := errors.New("model load failed")
	lazy := embed.NewLazyEmbedderWithFactoryForTest(func() (*embed.HugotEmbedder, error) {
		return nil, bootErr
	})

	_, err := lazy.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("embedder unavailable")))
	g.Expect(err).To(MatchError(bootErr))
	g.Expect(lazy.Dims()).To(Equal(0))
	g.Expect(lazy.ModelID()).To(Equal(embed.BundledModelID))
}

// TestLazyEmbedder_InitSuccessPaths drives the happy paths via an
// injected factory returning a HugotEmbedder built from the fake
// pipeline constructor.
func TestLazyEmbedder_InitSuccessPaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeEmb := embed.NewHugotEmbedderWithPipelineForTest("fake@4", 4,
		func(string) ([][]float32, error) {
			return [][]float32{{0.1, 0.2, 0.3, 0.4}}, nil
		},
		func() error { return nil },
	)

	lazy := embed.NewLazyEmbedderWithFactoryForTest(func() (*embed.HugotEmbedder, error) {
		return fakeEmb, nil
	})

	g.Expect(lazy.Dims()).To(Equal(4))
	g.Expect(lazy.ModelID()).To(Equal("fake@4"))

	vec, err := lazy.Embed(context.Background(), "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(4))
}
