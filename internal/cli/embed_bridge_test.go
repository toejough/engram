package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestBridgeEmbedder_DelegatesAfterWire asserts all three methods forward
// to the wired embedder.
func TestBridgeEmbedder_DelegatesAfterWire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, wire := cli.ExportNewBridgeEmbedder()
	wire(bridgeStubEmbedder{})

	g.Expect(bridge.ModelID()).To(Equal("stub@4"))
	g.Expect(bridge.Dims()).To(Equal(4))

	vec, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(Equal([]float32{0, 0, 0, 0}))
}

// TestBridgeEmbedder_UnwiredFallbacks asserts the pre-wiring behavior
// mirrors LazyEmbedder's pre-init semantics.
func TestBridgeEmbedder_UnwiredFallbacks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, _ := cli.ExportNewBridgeEmbedder()

	g.Expect(bridge.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(bridge.Dims()).To(Equal(0))

	_, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("embedder not wired")))
}

type bridgeStubEmbedder struct{}

func (bridgeStubEmbedder) Dims() int { return 4 }

func (bridgeStubEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0, 0, 0, 0}, nil
}

func (bridgeStubEmbedder) ModelID() string { return "stub@4" }
